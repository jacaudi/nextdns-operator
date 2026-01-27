package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NextDNSAllowlistSpec defines the desired state of NextDNSAllowlist
type NextDNSAllowlistSpec struct {
	// Description provides context for this allowlist
	// +optional
	Description string `json:"description,omitempty"`

	// Domains is the list of domains to allow
	// +kubebuilder:validation:MinItems=1
	Domains []DomainEntry `json:"domains"`
}

// NextDNSAllowlistStatus defines the observed state of NextDNSAllowlist
type NextDNSAllowlistStatus struct {
	// DomainCount is the number of active domains
	// +optional
	DomainCount int `json:"domainCount,omitempty"`

	// ProfileRefs lists profiles using this allowlist
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

// NextDNSAllowlist is the Schema for the nextdnsallowlists API
type NextDNSAllowlist struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NextDNSAllowlistSpec   `json:"spec,omitempty"`
	Status NextDNSAllowlistStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSAllowlistList contains a list of NextDNSAllowlist
type NextDNSAllowlistList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NextDNSAllowlist `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NextDNSAllowlist{}, &NextDNSAllowlistList{})
}
