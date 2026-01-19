package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NextDNSDenylistSpec defines the desired state of NextDNSDenylist
type NextDNSDenylistSpec struct {
	// Description provides context for this denylist
	// +optional
	Description string `json:"description,omitempty"`

	// Domains is the list of domains to block
	// +kubebuilder:validation:MinItems=1
	Domains []DomainEntry `json:"domains"`
}

// NextDNSDenylistStatus defines the observed state of NextDNSDenylist
type NextDNSDenylistStatus struct {
	// DomainCount is the number of active domains
	// +optional
	DomainCount int `json:"domainCount,omitempty"`

	// ProfileRefs lists profiles using this denylist
	// +optional
	ProfileRefs []ResourceReference `json:"profileRefs,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Domains",type=integer,JSONPath=`.status.domainCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSDenylist is the Schema for the nextdnsdenylists API
type NextDNSDenylist struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NextDNSDenylistSpec   `json:"spec,omitempty"`
	Status NextDNSDenylistStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSDenylistList contains a list of NextDNSDenylist
type NextDNSDenylistList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NextDNSDenylist `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NextDNSDenylist{}, &NextDNSDenylistList{})
}
