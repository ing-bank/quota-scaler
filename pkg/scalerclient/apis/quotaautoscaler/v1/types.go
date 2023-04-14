package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type QuotaAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec QuotaAutoscalerSpec `json:"spec"`
}

type QuotaAutoscalerSpec struct {
	ResourceQuota string `json:"resourceQuota"`

	MinCpu     string `json:"minCpu,omitempty"`
	MaxCpu     string `json:"maxCpu,omitempty"`
	MinCpuStep string `json:"minCpuStep,omitempty"`
	MaxCpuStep string `json:"maxCpuStep,omitempty"`

	MinMemory     string `json:"minMemory,omitempty"`
	MaxMemory     string `json:"maxMemory,omitempty"`
	MinMemoryStep string `json:"minMemoryStep,omitempty"`
	MaxMemoryStep string `json:"maxMemoryStep,omitempty"`

	Behavior QuotaAutoscalerSpecBehavior `json:"behavior"`
}

type QuotaAutoscalerSpecBehavior struct {
	ScaleUp   QuotaScaleBehavior `json:"scaleUp,omitempty"`
	ScaleDown QuotaScaleBehavior `json:"scaleDown,omitempty"`
}

type QuotaScaleBehavior struct {
	Policies []QuotaScalePolicy `json:"policies"`
}

type QuotaScalePolicy struct {
	Method        string `json:"method"`
	Value         int    `json:"value"`
	PeriodMinutes int    `json:"periodMinutes,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type QuotaAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []QuotaAutoscaler `json:"items"`
}
