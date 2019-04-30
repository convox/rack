package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Atom struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Started metav1.Time `json:"started"`
	Status  AtomStatus  `json:"status"`
	Spec    AtomSpec    `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AtomList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Atom `json:"items"`
}

type AtomCondition struct {
	ApiVersion string                        `json:"apiVersion"`
	Conditions map[string]AtomConditionMatch `json:"conditions"`
	Kind       string                        `json:"kind"`
	Name       string                        `json:"name"`
	Namespace  string                        `json:"namespace"`
}

type AtomConditionMatch struct {
	Reason string `json:"reason"`
	Status string `json:"status"`
}

type AtomSpec struct {
	Conditions              []AtomCondition `json:"conditions"`
	Current                 AtomState       `json:"current"`
	Previous                AtomState       `json:"previous"`
	ProgressDeadlineSeconds int32           `json:"progressDeadlineSeconds"`
}

type AtomStatus string

type AtomState struct {
	Template []byte `json:"template"`
	Version  string `json:"version"`
}
