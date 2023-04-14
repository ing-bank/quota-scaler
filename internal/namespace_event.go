package internal

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

func PublishNamespaceEvent(client *kubernetes.Clientset, ref v1.ObjectReference, ev ResizeResult) error {
	msg := fmt.Sprintf("Namespace ResourceQuota resized from CPU: %dm Memory: %dM to CPU: %dm Memory: %dM", ev.Old.Cpu, ev.Old.Memory, ev.New.Cpu, ev.New.Memory)
	evType := "Normal"

	if ev.Err != nil {
		msg = fmt.Sprintf("Failed to resize ResourceQuota from CPU: %dm Memory: %dM to CPU: %dm Memory: %dM: %s", ev.Old.Cpu, ev.Old.Memory, ev.New.Cpu, ev.New.Memory, ev.Err.Error())
		evType = "Warning"
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	_, err := client.CoreV1().Events(ref.Namespace).Create(ctx, &v1.Event{
		ObjectMeta:          v13.ObjectMeta{GenerateName: "ichp-quota-scaler-"},
		FirstTimestamp:      v13.Now(),
		LastTimestamp:       v13.Now(),
		InvolvedObject:      ref,
		Reason:              "QuotaResize",
		Message:             msg,
		Type:                evType,
		ReportingController: "ichp-quota-scaler/scaler",
	}, v13.CreateOptions{})
	return err
}
