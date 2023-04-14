package internal

// This file contains the InvokeResizeApi function which calls the ICHP API namespace
// PATCH operation. The ICHP API is discovered via environment variable ICHP_API_ENDPOINT.
// The authentication is done via a bearer token stored in environment variable TOKEN.

import (
	"context"
	"github.com/ing-bank/quota-scaler/pkg/kubeconfig"
	"github.com/ing-bank/quota-scaler/pkg/logging"
	"github.com/ing-bank/quota-scaler/pkg/resources"
	"github.com/ing-bank/quota-scaler/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"os"
	"time"
)

type NamespaceResizeEvent struct {
	Namespace     string
	ResourceQuota string
	Old           resources.Resources
	New           resources.Resources
}

type ResizeResult struct {
	NamespaceResizeEvent
	Err error
}

var ResizeNsChan = make(chan NamespaceResizeEvent)
var ResizeResultChan = make(chan ResizeResult, 1024)
var eventDoneChan = make(chan NamespaceResizeEvent)

type IchpApiResponse struct {
	Clusters []struct {
		Message string `json:"message"`
	} `json:"clusters"`
	Status    string `json:"status"`
	RequestId string `json:"requestID"`
}

// NamespacePatch is the expected schema for an ICHP-API namespace patch operation
type NamespacePatch struct {
	Name     string             `json:"name"`
	Workload string             `json:"workload,omitempty"`
	Spec     NamespacePatchSpec `json:"spec"`
}

type NamespacePatchSpec struct {
	Quota NamespacePatchSpecQuota `json:"quota"`
}

type NamespacePatchSpecQuota struct {
	Cpu     int64 `json:"cpu"`
	Memory  int64 `json:"memory"`
	Storage int64 `json:"storage"`
}

type ResizeCache struct {
	Timestamp time.Time
	Event     NamespaceResizeEvent
}

var ResizeApiFunc = InvokeResizeApiStub // TODO: replace this with your own stack resize!

func publishResizeResult(ns NamespaceResizeEvent, err error) {
	select {
	case ResizeResultChan <- ResizeResult{NamespaceResizeEvent: ns, Err: err}:
	default: //NoBlock
	}
}

func resizeAsync(event NamespaceResizeEvent) {
	defer func() {
		eventDoneChan <- event
	}()

	if err := ResizeApiFunc(event); err != nil {
		logging.LogError("[%s] Failed to resize ns (%+v): %v\n", event.Namespace, event, err)
		publishResizeResult(event, err)
		return
	}

	logging.LogInfo("[%s] Namespace resized: %+v\n", event.Namespace, event)
	publishResizeResult(event, nil)
}

// RunEventHandler listens to Async Resize API requests. Replies are published on ResizeResultChan and must be
// read.
func RunEventHandler() { // Blocks, forever
	inProgress := map[string]bool{}              // Namespace resizes that the resize API is currently executing
	pending := map[string]NamespaceResizeEvent{} // Resize that is waiting for previous resize API to finish
	cache := map[string]ResizeCache{}

	resize := func(event NamespaceResizeEvent) bool {
		if previous, ok := cache[event.Namespace]; ok {
			if (event.New.Cpu < previous.Event.New.Cpu || event.New.Memory < previous.Event.New.Memory) && previous.Timestamp.Add(time.Minute).After(time.Now()) {
				// Scale down is allowed max once per hour, so ignore this request
				logging.LogInfo("[%s] We updated this object recently (%v) and this is a scaleDown (%v) which will fail. Ignoring this resize event.", event.Namespace, previous, event)
				return false
			}
		}
		go resizeAsync(event)
		return true
	}

	for {
		select {
		case event := <-ResizeNsChan:
			if inProgress[event.Namespace] {
				// Resize API is already handling this namespace, keep event (newest) to execute in the future
				pending[event.Namespace] = event
			} else {
				// Resize API was not handling this namespace
				if resize(event) {
					inProgress[event.Namespace] = true
				}
			}

		case ns := <-eventDoneChan:
			cache[ns.Namespace] = ResizeCache{Timestamp: time.Now(), Event: ns}

			// inProgress is still set, see if there are any Pending
			if event, ok := pending[ns.Namespace]; ok {
				delete(pending, ns.Namespace) // Consume latest event
				if !resize(event) {
					inProgress[ns.Namespace] = false // All events done
				}
			} else {
				inProgress[ns.Namespace] = false // All events done
			}
		}
	}
}

func InvokeResizeApiAsync(namespace, resourcequota string, old, new resources.Resources) {
	// Storage scaling is not yet supported, old and new are always the same
	ResizeNsChan <- NamespaceResizeEvent{namespace, resourcequota, old, new}
}

// TODO: this is left as an example as a resize endpoint, this code is not used
// InvokeResizeApi issues a namespace patch operation to the ICHP-API. The provided `cpu` must in
// Milli Cores and the `memory` must be in Mega Bytes. The ICHP-API is discovered via environment
// variable `ICHP_API_ENDPOINT`, with bearer token auth using environment variable `TOKEN`.
func InvokeResizeApi(ns NamespaceResizeEvent) error {
	body, err := json.Marshal(&NamespacePatch{
		Name:     ns.Namespace,
		Workload: os.Getenv("WORKLOAD"),
		Spec: NamespacePatchSpec{
			Quota: NamespacePatchSpecQuota{
				Cpu:     ns.New.Cpu,
				Memory:  ns.New.Memory,
				Storage: ns.New.Storage,
			},
		},
	})
	if err != nil {
		return err
	}

	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return errors.New("cannot read serviceaccount token: " + err.Error())
	}

	endpoint := os.Getenv("ICHP_API_ENDPOINT")
	logging.LogInfo("[%s] Calling %s with CPU: %d Memory: %d Storage %d\n", ns.Namespace, endpoint, ns.New.Cpu, ns.New.Memory, ns.New.Storage)
	endpoint += "/api/v1/namespace"
	response, err := utils.HttpPatch(
		endpoint,
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + string(token),
		},
		body,
	)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// TODO: We should wrap this via https://haisum.github.io/2021/08/14/2021-golang-http-errors/
	parsedResp := &IchpApiResponse{}
	if err := json.Unmarshal(respBody, parsedResp); err == nil {
		if response.StatusCode == http.StatusOK {
			logging.LogInfo("[%s] Resize API (%dm, %dM) reply: [%s] %s", ns.Namespace, ns.New.Cpu, ns.New.Memory, parsedResp.RequestId, parsedResp.Status)
		} else {
			errorMsg := []string{}
			for _, cluster := range parsedResp.Clusters {
				errorMsg = append(errorMsg, cluster.Message)
			}
			logging.LogError("[%s] Resize API (%dm, %dM) reply: [%s] %s: %v", ns.Namespace, ns.New.Cpu, ns.New.Memory, parsedResp.RequestId, parsedResp.Status, errorMsg)
		}
	}

	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("resize API status NOK: %s\n", response.Status))
	}
	return nil
}

func InvokeResizeApiStub(ns NamespaceResizeEvent) error {
	client, err := kubeconfig.GetKubernetesClient()
	if err != nil {
		return err
	}

	// Example of resize, e.g. Patch ResourceQuota. But, you should replace this with your own stack!
	fastMergeExample := []byte(fmt.Sprintf("{\"spec\": {\"hard\": {\"cpu\": \"%dm\", \"limits.cpu\": \"%dm\", \"memory\": \"%dM\", \"limits.memory\": \"%dm\"}}}",
		ns.New.Cpu, ns.New.Cpu*REQ_LIM_RATIO, // CPU, CPU LIMIT
		ns.New.Memory, ns.New.Memory, // MEM, MEM LIMIT
	))
	_, err = client.CoreV1().ResourceQuotas(ns.Namespace).Patch(context.TODO(), ns.ResourceQuota, types.MergePatchType, fastMergeExample, v1.PatchOptions{})
	// TODO: Do charging events
	return err
}
