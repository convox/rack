package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Stack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec StackSpec `json:"spec,omitempty"`

	Outputs map[string]string `json:"outputs,omitempty"`
	Status  StackStatus       `json:"status,omitempty"`
}

type StackSpec struct {
	Parameters map[string]string `json:"parameters"`
	Template   string            `json:"template"`
}

type StackStatus string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type StackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Stack `json:"items"`
}
