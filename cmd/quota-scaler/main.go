package main

import (
	"context"
	ichp "github.com/ing-bank/quota-scaler/pkg/scalerclient/client/clientset/versioned"
	"net/http"
	_ "net/http/pprof"

	"github.com/ing-bank/quota-scaler/internal"
	"github.com/ing-bank/quota-scaler/pkg/kubeconfig"
	"github.com/ing-bank/quota-scaler/pkg/logging"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	config, err := kubeconfig.GetKubeConfig()
	if err != nil {
		panic(err)
	}

	ichpClient, err := ichp.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	client, err := kubeconfig.GetKubernetesClient()
	if err != nil {
		panic(err)
	}

	go func() {
		// Profiling
		panic(http.ListenAndServe(":8080", nil))
	}()

	// Runs forever, handles Resize events async by calling the Resize API
	go internal.RunEventHandler()

	for {
		logging.LogInfo("(re-)starting stream")
		var watchTimeoutSec int64 = 3600 // Hourly

		startScalerState, err := ichpClient.IchpV1().QuotaAutoscalers("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			panic(err)
		}

		scalerWatch, err := ichpClient.IchpV1().QuotaAutoscalers("").Watch(context.TODO(), v1.ListOptions{TimeoutSeconds: &watchTimeoutSec})
		if err != nil {
			panic(err)
		}

		quotaWatch, err := client.CoreV1().ResourceQuotas("").Watch(context.TODO(), v1.ListOptions{TimeoutSeconds: &watchTimeoutSec})
		if err != nil {
			panic(err)
		}

		// We catch "FailedCreate" Pod events (and calculate extra resources based on that)
		eventWatch, err := client.CoreV1().Events("").Watch(context.TODO(), v1.ListOptions{TimeoutSeconds: &watchTimeoutSec, FieldSelector: "reason=FailedCreate"})
		if err != nil {
			panic(err)
		}

		// cert-manager has an specific type of event, when trying to create solver pods it will fail under the reason=PresentError
		// FieldSelector should be unique, that's why we triger 2 different watch calls
		cmEventWatch, err := client.CoreV1().Events("").Watch(context.TODO(), v1.ListOptions{TimeoutSeconds: &watchTimeoutSec, FieldSelector: "reason=PresentError"})
		if err != nil {
			panic(err)
		}

		// Blocking call until stream watch timeout
		internal.WatchQuotas(client, startScalerState.Items, quotaWatch.ResultChan(), scalerWatch.ResultChan(), eventWatch.ResultChan(), cmEventWatch.ResultChan())

		scalerWatch.Stop()
		quotaWatch.Stop()
		eventWatch.Stop()
	}
}
