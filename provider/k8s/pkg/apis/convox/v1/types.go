package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BuildSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Build `json:"items"`
}

type BuildSpec struct {
	Description string `json:"description"`
	Ended       string `json:"ended"`
	Entrypoint  string `json:"entrypoint"`
	Logs        string `json:"logs"`
	Manifest    string `json:"manifest"`
	Process     string `json:"process"`
	Release     string `json:"release"`
	Started     string `json:"started"`
	Status      string `json:"status"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ReleaseSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Release `json:"items"`
}

type ReleaseSpec struct {
	Build    string `json:"build"`
	Created  string `json:"created"`
	Env      string `json:"env"`
	Manifest string `json:"manifest"`
}
