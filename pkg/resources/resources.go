package resources

import (
	"github.com/ing-bank/quota-scaler/pkg/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// This package combines Cpu and Memory resources into a single struct, combined with some actions upon them.
// For ease of use each function updates a struct pointer, as well as returns a reference to that struct. This
// allows chaining of function calls. For example:
//
//  example := &Resources{Cpu: 1}
//  example.Add(&Resources{Cpu: 1}).Add(&Resources{Cpu: 1}) // example.Cpu = 3
//  example.Limit(&Resources{Cpu: 2})                       // example.Cpu = 2

// Resources combines Cpu and Memory into a single container
type Resources struct {
	Cpu     int64 // In millicores
	Memory  int64 // In megabytes
	Storage int64 // Store is included here for logging purposes, but not intended to be scaled (yet)
}

// Add adds the new resources to the existing resources. Result is updated and also returned.
func (res *Resources) Add(new *Resources) *Resources {
	res.Cpu += new.Cpu
	res.Memory += new.Memory
	res.Storage += new.Storage
	return res
}

// Replace replaces fields in `res` if they are non-default in `new`. Result is updated and also returned.
func (res *Resources) Replace(new *Resources) *Resources {
	if new == nil {
		return res
	}

	if new.Cpu != 0 {
		res.Cpu = new.Cpu
	}
	if new.Memory != 0 {
		res.Memory = new.Memory
	}

	if new.Storage != 0 {
		res.Storage = new.Storage
	}

	return res
}

// Limit limits the resources with a maximum of the specified limit. Result is updated and also returned.
func (res *Resources) Limit(limit *Resources) *Resources {
	if res.Cpu > limit.Cpu {
		res.Cpu = limit.Cpu
	}
	if res.Memory > limit.Memory {
		res.Memory = limit.Memory
	}
	return res
}

// Max updates res with the maximum values of res and new. Result is updated and also returned.
func (res *Resources) Max(new *Resources) *Resources {
	if new.Cpu > res.Cpu {
		res.Cpu = new.Cpu
	}
	if new.Memory > res.Memory {
		res.Memory = new.Memory
	}
	return res
}

// IsEmpty returns true when Cpu and Memory are both zero.
func (res *Resources) IsEmpty() bool {
	return res.Cpu == 0 && res.Memory == 0
}

func (res *Resources) DiffersFrom(quota *v1.ResourceQuota) bool {
	if res.Cpu != quota.Spec.Hard.Cpu().ScaledValue(resource.Milli) {
		return true
	}
	if res.Memory != quota.Spec.Hard.Memory().ScaledValue(resource.Mega) {
		return true
	}
	return false
}

func (res *Resources) IsScaleDown(quota *v1.ResourceQuota) bool {
	return res.Cpu < quota.Spec.Hard.Cpu().ScaledValue(resource.Milli) || res.Memory < quota.Spec.Hard.Memory().ScaledValue(resource.Mega)
}

func (res *Resources) ForceNoScaleDownWhenScaleUp(quota *v1.ResourceQuota) {
	if res.Cpu > quota.Spec.Hard.Cpu().ScaledValue(resource.Milli) { // We are scaling up CPU
		if res.Memory < quota.Spec.Hard.Memory().ScaledValue(resource.Mega) { // But Memory is a scale down
			res.Memory = quota.Spec.Hard.Memory().ScaledValue(resource.Mega) // So keep Memory like quota, no scale down, perhaps it is not allowed (max once per hour)
			logging.LogInfo("[%s] Raising Memory to %d to make sure we do a scaleUp", quota.Namespace, res.Memory)
		}

	} else if res.Memory > quota.Spec.Hard.Memory().ScaledValue(resource.Mega) { // We are scaling up memory
		if res.Cpu < quota.Spec.Hard.Cpu().ScaledValue(resource.Milli) { // But CPU is a scale down
			res.Cpu = quota.Spec.Hard.Cpu().ScaledValue(resource.Milli) // So keep CPU like quota, no scale down, perhaps it is not allowed (max once per hour)
			logging.LogInfo("[%s] Raising CPU to %d to make sure we do a scaleUp", quota.Namespace, res.Cpu)
		}
	}
}
