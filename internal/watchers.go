package internal

// This file contains functions that take namespaced ResourceQuota and QuotaAutoscaler events and turn
// them into behaviour which can invoke the resize API. Event watching is the responsibility of the
// calling function, this also includes stream resets.
//
// Example usage:
//  quotas, _ := core.ResourceQuotas("").Watch(context.TODO(), v1.ListOptions{})
//  scalers, _ := ichp.QuotaAutoscalers("").Watch(context.TODO(), v1.ListOptions{})
//  events, _ := client.CoreV1().Events("").Watch(context.TODO(), v1.ListOptions{})
//
//  // This is a blocking call, until either watcher channel terminates
//  internal.WatchQuotas(client, quotas.ResultChan(), scalers.ResultChan(), events.ResultChan())
//
//	quotas.Stop()
//  scalers.Stop()
//  events.Stop()

import (
	"context"
	v14 "github.com/ing-bank/quota-scaler/pkg/scalerclient/apis/quotaautoscaler/v1"
	"errors"
	_ "net/http/pprof"
	"time"

	"github.com/ing-bank/quota-scaler/pkg/logging"
	"github.com/ing-bank/quota-scaler/pkg/resources"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// REQ_LIM_RATIO is the ratio between namespace ResourceQuota CPU requests and CPU limits.
// E.g. with ratio=10 when a consumer requests 400m CPU, they will have a 4 core CPU limit.
const REQ_LIM_RATIO = 10

// QuotaWatcher internally manages a list of QuotaAutoscalers and ResourceQuotas.
type QuotaWatcher struct {
	Scalers map[string]v14.QuotaAutoscaler
	Quotas  map[string]v12.ResourceQuota
	Events  map[string][]v12.Event

	Client *kubernetes.Clientset
}

// WatchQuotas listens to namespaced ResourceQuotas and QuotaAutoscalers. When both are known for a namespace
// the required behaviour is calculated. If scaling is required, following the behavior, the resize API is
// invoked. This is a blocking call until either channel terminates.
func WatchQuotas(client *kubernetes.Clientset, startScalers []v14.QuotaAutoscaler, quotas, scalers, events <-chan watch.Event, cmEvents <-chan watch.Event) {
	watcher := &QuotaWatcher{
		Scalers: map[string]v14.QuotaAutoscaler{},
		Quotas:  map[string]v12.ResourceQuota{},
		Events:  map[string][]v12.Event{},
		Client:  client,
	}

	// Init scaler state so that we know which ResourceQuotas to couple. The Scaler has a field with the
	// resource name of the ResourceQuota object, so we cannot store ResourceQuota objects until we know
	// the scaler spec. After registering the Scalers we will get the ResourceQuotas in the event stream.
	for _, nsScaler := range startScalers {
		watcher.Scalers[nsScaler.Namespace] = nsScaler
	}
	startQuotas, _ := client.CoreV1().ResourceQuotas("").List(context.TODO(), v13.ListOptions{})
	for _, startQuota := range startQuotas.Items {
		scaler, ok := watcher.Scalers[startQuota.Namespace]
		if ok && scaler.Spec.ResourceQuota == startQuota.Name {
			watcher.Quotas[startQuota.Namespace] = startQuota
		}
	}

	// Ticker aggregates Namespace and ResourceQuota events
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-quotas:
			// Quota Changes are very frequent, every Pod "modifies" the Quota status twice.
			// To be a bit more friendly during a scale down we let ticker aggregate Quota
			// events. For most cases this will suffice. The ideal implementation would be to
			// aggregate a few seconds after the first ResourceQuota event per unique namespace.
			if !ok {
				return
			}
			ns := watcher.RegisterQuotaEvent(event)
			if ns != "" {
				logging.LogDebug("[%s] QuotaEvent", ns)
				if _, ok := watcher.Events[ns]; !ok {
					watcher.Events[ns] = []v12.Event{}
					// We let the ticker aggregate events
				}
			}
		case event, ok := <-scalers:
			if !ok {
				return
			}
			ns := watcher.RegisterScalerEvent(event)
			if ns != "" {
				logging.LogDebug("[%s] ScalerEvent", ns)
				watcher.UpdateNs(ns, false)
			}

		case event, ok := <-events:
			if !ok {
				return
			}
			ns := watcher.RegisterNamespacedEvent(event)
			if ns != "" {
				logging.LogDebug("[%s] PodEvent", ns)
			}
			// We let the ticker aggregate events

		// cert-manager events will also be handled by the function RegisterNamespacedEvent
		case cmEvent, ok := <-cmEvents:
			if !ok {
				return
			}
			ns := watcher.RegisterNamespacedEvent(cmEvent)
			if ns != "" {
				logging.LogDebug("[%s] CertManagerEvent", ns)
			}
			// We let the ticker aggregate events

		case <-ticker.C:
			for ns, _ := range watcher.Events {
				logging.LogDebug("[%s] Ticker", ns)
				watcher.UpdateNs(ns, true) // New expensive, as reading events will read ReplicaSets, etc
				delete(watcher.Events, ns)
			}
		case event := <-ResizeResultChan: // This channel is managed by resize_api.go, should never close
			scalerObj := watcher.Scalers[event.Namespace]
			ref := v12.ObjectReference{
				Kind:            "quotaautoscaler",
				Namespace:       scalerObj.Namespace,
				Name:            scalerObj.Name,
				UID:             scalerObj.UID,
				APIVersion:      scalerObj.APIVersion,
				ResourceVersion: scalerObj.ResourceVersion,
			}
			go func() {
				if err := PublishNamespaceEvent(client, ref, event); err != nil {
					logging.LogError("[%s] Cannot publish namespace event: %s", event.Namespace, err.Error())
				}
			}()
		}
	}
}

func (watcher *QuotaWatcher) UpdateNs(namespace string, readEvents bool) {
	scaler, scalerOk := watcher.Scalers[namespace]
	quota, quotaOk := watcher.Quotas[namespace]
	var events []v12.Event
	if readEvents {
		events = watcher.Events[namespace]
	}
	logging.LogDebug("[%s] Checking Quota Updates (Found Scaler: %t Quota: %t Events %d (%t) ", namespace, scalerOk, quotaOk, len(events), readEvents)

	if scalerOk && quotaOk {
		go func() {
			err := watcher.UpdateQuotaIfRequired(quota, scaler, events)
			if err != nil {
				logging.LogError("[%s] Failed to update quota for: %s", namespace, err.Error())
			}
		}()
	}
}

// RegisterScalerEvent stores a QuotaAutoscaler in watcher, or deletes it.
func (watcher *QuotaWatcher) RegisterScalerEvent(event watch.Event) string {
	scaler := event.Object.(*v14.QuotaAutoscaler)

	if event.Type == watch.Deleted {
		delete(watcher.Scalers, scaler.Namespace)
		return ""
	}

	watcher.Scalers[scaler.Namespace] = *scaler
	if _, ok := watcher.Quotas[scaler.Namespace]; !ok {
		_ = watcher.RegisterMissingResourceQuota(scaler.Namespace, scaler.Spec.ResourceQuota) // A bit slow, but needed
	}
	return scaler.Namespace
}

// RegisterQuotaEvent stores a ResourceQuota in watcher, or deletes it.
func (watcher *QuotaWatcher) RegisterQuotaEvent(event watch.Event) string {
	quota := event.Object.(*v12.ResourceQuota)

	// Get QuotaAutoscaler to see event Quota is a target
	scaler, ok := watcher.Scalers[quota.Namespace]
	if ok && scaler.Spec.ResourceQuota == quota.Name {

		if event.Type == watch.Deleted {
			delete(watcher.Quotas, quota.Namespace)
			return ""
		}

		watcher.Quotas[quota.Namespace] = *quota
		return quota.Namespace
	}

	return ""
}

// RegisterMissingResourceQuota fetches the given quota and stores it in the watcher. Does not consume Lock, so
// it must be called when holding the (Mutex) Lock.
func (watcher *QuotaWatcher) RegisterMissingResourceQuota(namespace, quotaName string) error {
	logging.LogInfo("[%s] Registering missing ResourceQuota: %s", namespace, quotaName)
	quota, err := watcher.Client.CoreV1().ResourceQuotas(namespace).Get(context.TODO(), quotaName, v13.GetOptions{})
	if err == nil {
		watcher.Quotas[namespace] = *quota // Register
	} else {
		logging.LogError("[%s] Failed to register ResourceQuota %s: %v", namespace, quotaName, err)
	}

	return err
}

// RegisterNamespacedEvent stores namespaced Events in watcher. Should be cleaned up by aggregate loop
// every iteration (events should be consumed once). This function does not delete any stored events.
func (watcher *QuotaWatcher) RegisterNamespacedEvent(event watch.Event) string {
	ev := event.Object.(*v12.Event)
	target := ev.InvolvedObject

	if _, ok := watcher.Events[target.Namespace]; !ok {
		watcher.Events[target.Namespace] = []v12.Event{*ev}
	} else {
		watcher.Events[target.Namespace] = append(watcher.Events[target.Namespace], *ev)
	}

	return ev.Namespace
}

func ResourceQuotaUsedMemoryLimit(quota *v12.ResourceQuota) *resource.Quantity {
	memLimit, ok := quota.Status.Used["limits.memory"]
	if !ok {
		return quota.Status.Used.Memory()
	}
	return &memLimit
}

func ResourceQuotaUsedCpuLimit(quota *v12.ResourceQuota) *resource.Quantity {
	cpuLimit, ok := quota.Status.Used["limits.cpu"]
	if !ok {
		return quota.Status.Used.Cpu()
	}
	return &cpuLimit
}

func (watcher *QuotaWatcher) UpdateQuotaIfRequired(quota v12.ResourceQuota, scaler v14.QuotaAutoscaler, events []v12.Event) error {
	validatedScaler := ValidateQuotaScaler(&scaler)
	desired := &resources.Resources{
		Cpu:    quota.Spec.Hard.Cpu().ScaledValue(resource.Milli),
		Memory: quota.Spec.Hard.Memory().ScaledValue(resource.Mega),
	}

	if quota.Status.Used == nil || quota.Status.Hard == nil {
		return errors.New("quota status is nil")
	}

	// Take limits into accounts, especially the ratio between CPU requests and limits. Fake Req CPU if limits are high
	quota.Status.Used[v12.ResourceCPU] = GetNormalizedUsedCpu(quota.Status.Used.Cpu(), ResourceQuotaUsedCpuLimit(&quota), scaler.Namespace)

	for _, policy := range scaler.Spec.Behavior.ScaleDown.Policies {
		desired.Replace(validatedScaler.ActivateScalerPolicy(policy, &quota, false))
	}
	logging.LogDebug("[%s] Desired resources after ScaleDown: %+v\n", scaler.Namespace, desired)
	for _, policy := range scaler.Spec.Behavior.ScaleUp.Policies {
		desired.Replace(validatedScaler.ActivateScalerPolicy(policy, &quota, true))
	}
	logging.LogDebug("[%s] Desired resources after ScaleUp: %+v\n", scaler.Namespace, desired)

	if events != nil {
		if sum, _ := GetResourcesFromPodEvents(watcher.Client, events); sum != nil && !sum.IsEmpty() { // This is a slow call!
			logging.LogInfo("[%s] Namespace events require an extra %+v resources\n", scaler.Namespace, sum)
			desired = (&resources.Resources{
				Cpu:    quota.Status.Used.Cpu().ScaledValue(resource.Milli),
				Memory: ResourceQuotaUsedMemoryLimit(&quota).ScaledValue(resource.Mega),
			}).Add(sum).Max(desired)
		}
	}

	// We don't do anything with Storage
	storage := quota.Spec.Hard["requests.storage"]

	// Make sure desired quota is within bounds
	validatedScaler.ForceLimitToDefaultMax()
	desired.Max(&resources.Resources{Cpu: validatedScaler.MinCpu, Memory: validatedScaler.MinMemory})
	desired.Limit(&resources.Resources{Cpu: validatedScaler.MaxCpu, Memory: validatedScaler.MaxMemory})
	desired.Storage = storage.ScaledValue(resource.Giga)

	current := resources.Resources{
		Cpu:     quota.Spec.Hard.Cpu().ScaledValue(resource.Milli),
		Memory:  quota.Spec.Hard.Memory().ScaledValue(resource.Mega),
		Storage: storage.ScaledValue(resource.Giga),
	}
	logging.LogInfo("[%s] Calculated desired resources (%+v -> %+v) for namespace %s\n", quota.Namespace, current, desired, scaler.Namespace)
	desired.ForceNoScaleDownWhenScaleUp(&quota)
	if desired.DiffersFrom(&quota) {
		logging.LogDebug("[%s] InvokeResizeApiAsync", quota.Namespace)
		InvokeResizeApiAsync(quota.Namespace, scaler.Spec.ResourceQuota, current, *desired)
	}

	return nil
}
