package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/ing-bank/quota-scaler/pkg/logging"
	"github.com/ing-bank/quota-scaler/pkg/resources"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetResourcesFromPodEvents(client kubernetes.Interface, events []v12.Event) (*resources.Resources, error) {
	sum := &resources.Resources{}
	involvedObjects := map[string]bool{} // Make sure we only handle each InvolvedObject once

	for _, ev := range events {
		if isDaemonSet(ev) {
			continue // Skip these for now
		}

		name := ev.InvolvedObject.Kind + ev.InvolvedObject.Name
		if _, ok := involvedObjects[name]; !ok {
			logging.LogInfo("[%s] Processing event %s %s", ev.Namespace, ev.InvolvedObject.Kind, ev.InvolvedObject.Name)
			involvedObjects[name] = true

			spec, missingReplicas, err := getPodTemplateSpecFromEv(client, ev)
			if err != nil {
				logging.LogError("[%s] Cannot get template spec from event: %s %s: %v. Ignoring it", ev.Namespace, ev.InvolvedObject.Kind, ev.InvolvedObject.Name, err)
				continue // We process those we do know
			}

			res := CalculatePodResources(spec, int64(missingReplicas))
			sum.Add(&res)
		}
	}

	return sum, nil
}

// CalculatePodResources sums the container resources of a Pod and multiplies them by the missing replicas.
// Zero or negative replicas will result in an empty Resource.
func CalculatePodResources(podTemplate v12.PodTemplateSpec, missingReplicas int64) resources.Resources {
	neededCpu := resource.NewQuantity(0, resource.DecimalSI)
	neededMemory := resource.NewQuantity(0, resource.DecimalSI)
	for _, container := range podTemplate.Spec.Containers {
		if container.Resources.Requests != nil {
			expectedNeededCpu := *container.Resources.Requests.Cpu()
			expectedNeededMemory := *container.Resources.Requests.Memory()

			// Take limits into accounts, especially the ratio between CPU requests and limits
			if container.Resources.Limits != nil {
				expectedNeededMemory = *container.Resources.Limits.Memory()
				expectedNeededCpu = GetNormalizedUsedCpu(container.Resources.Requests.Cpu(), container.Resources.Limits.Cpu(), podTemplate.Namespace)
			}

			neededCpu.Add(expectedNeededCpu)
			neededMemory.Add(expectedNeededMemory)
		}
	}

	if missingReplicas <= 0 {
		return resources.Resources{}
	}
	return resources.Resources{
		Cpu:    neededCpu.ScaledValue(resource.Milli) * missingReplicas,
		Memory: neededMemory.ScaledValue(resource.Mega) * missingReplicas,
	}
}

func isDaemonSet(ev v12.Event) bool {
	return ev.InvolvedObject.Kind == "DaemonSet"
}

func getPodTemplateSpecFromEv(client kubernetes.Interface, ev v12.Event) (v12.PodTemplateSpec, int32, error) {
	var pod v12.PodTemplateSpec
	var replicas int32 = 1

	namespace := ev.InvolvedObject.Namespace
	name := ev.InvolvedObject.Name
	switch ev.InvolvedObject.Kind {
	case "ReplicaSet":
		target, err := client.AppsV1().ReplicaSets(namespace).Get(context.TODO(), name, v13.GetOptions{})
		if err != nil {
			return pod, replicas, err
		}
		pod = target.Spec.Template
		replicas = *target.Spec.Replicas - target.Status.Replicas
	case "Job":
		target, err := client.BatchV1().Jobs(namespace).Get(context.TODO(), name, v13.GetOptions{})
		if err != nil {
			return pod, replicas, err
		}
		pod = target.Spec.Template
	case "StatefulSet":
		target, err := client.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, v13.GetOptions{})
		if err != nil {
			return pod, replicas, err
		}
		pod = target.Spec.Template
		replicas = *target.Spec.Replicas - target.Status.Replicas
	case "ReplicationController":
		target, err := client.CoreV1().ReplicationControllers(namespace).Get(context.TODO(), name, v13.GetOptions{})
		if err != nil {
			return pod, replicas, err
		}
		if target.Spec.Template == nil {
			return pod, replicas, fmt.Errorf("ReplicationController %s Pod Template missing", name)
		}
		pod = *target.Spec.Template
		replicas = *target.Spec.Replicas - target.Status.Replicas
	case "Challenge":
		// The challenge manifest doesn't contain the pod specs,
		// so we create a generic spec with the defaults from cert-manager pod.
		pod = v12.PodTemplateSpec{
			Spec: v12.PodSpec{
				Containers: []v12.Container{
					{Name: "dummy-container",
						Resources: v12.ResourceRequirements{
							Requests: v12.ResourceList{v12.ResourceCPU: resource.MustParse("10m"), v12.ResourceMemory: resource.MustParse("64Mi")},
							Limits:   v12.ResourceList{v12.ResourceCPU: resource.MustParse("100m"), v12.ResourceMemory: resource.MustParse("64Mi")},
						}}}}}
		// Just one ephemeral pod needed for a challenge
		replicas = 1
	default:
		return v12.PodTemplateSpec{}, 0, errors.New("unsupported event")
	}

	pod.Namespace = namespace
	return pod, replicas, nil
}

// GetNormalizedUsedCpu calculates if the CPU limit / 10 is bigger than the CPU requests, if so we should scale
// based on the CPU limit in order not to breach the namespace quota. We then "fake" the CPU request to be higher so that
// future calculations only have to worry about CPU requests. If the ratio is not exceeded the requested values are
// returned. Ns variable only used for logging.
func GetNormalizedUsedCpu(request, limit *resource.Quantity, ns string) resource.Quantity {
	normalizedUsedCpuLimit := limit.ScaledValue(resource.Milli) / REQ_LIM_RATIO
	if normalizedUsedCpuLimit > request.ScaledValue(resource.Milli) {
		normalizedRes := *resource.NewScaledQuantity(normalizedUsedCpuLimit, resource.Milli)
		logging.LogInfo("[%s] Using CPU limit instead of request due to ratio (x%d) %d -> %d\n", ns, REQ_LIM_RATIO, request.ScaledValue(resource.Milli), normalizedRes.ScaledValue(resource.Milli))
		return normalizedRes
	}
	return *request
}
