package internal

import (
	"github.com/ing-bank/quota-scaler/pkg/resources"
	"testing"
	"time"
)

var resizeApiCalledExampleDev = 0
var exampleDevCpu int64 = 0

var resizeApiCalledFooDev = 0
var fooDevCpu int64 = 0

func FakeResizeApiCall(ns NamespaceResizeEvent) error {
	<-time.After(100 * time.Millisecond)
	if ns.Namespace == "example-dev" {
		resizeApiCalledExampleDev++
		exampleDevCpu = ns.New.Cpu
	} else if ns.Namespace == "foo-dev" {
		resizeApiCalledFooDev++
		fooDevCpu = ns.New.Cpu
	}

	return nil
}

func TestRunEventHandler(t *testing.T) {
	ResizeApiFunc = FakeResizeApiCall

	resizeResultsReceived := 0
	go RunEventHandler()
	go func() {
		<-ResizeResultChan
		resizeResultsReceived++
		<-ResizeResultChan
		resizeResultsReceived++
	}()

	for i := 1; i <= 4000; i++ {
		InvokeResizeApiAsync(
			"example-dev",
			resources.Resources{Cpu: int64(399 + i), Memory: int64(999 + i)},
			resources.Resources{Cpu: int64(400 + i), Memory: int64(1000 + i)},
		)
		InvokeResizeApiAsync(
			"foo-dev",
			resources.Resources{Cpu: int64(399 + i), Memory: int64(999 + i)},
			resources.Resources{Cpu: int64(400 + i), Memory: int64(1000 + i)},
		)
	}

	time.Sleep(250 * time.Millisecond)
	if resizeApiCalledExampleDev != 2 {
		t.Errorf("expected resize API to be called 2 times for example-dev but got: %d\n", resizeApiCalledExampleDev)
	}
	if exampleDevCpu != 4400 {
		t.Errorf("expected example-dev CPU to be 1400 but got: %d\n", exampleDevCpu)
	}

	if resizeApiCalledFooDev != 2 {
		t.Errorf("expected resize API to be called 2 for foo-dev times but got: %d\n", resizeApiCalledExampleDev)
	}
	if fooDevCpu != 4400 {
		t.Errorf("expected foo-dev Cpu CPU to be 1400 but got: %d\n", fooDevCpu)
	}

	if resizeResultsReceived != 2 {
		t.Errorf("expected resizeResultsReceived to be 2 but got: %d\n", resizeResultsReceived)
	}
}
