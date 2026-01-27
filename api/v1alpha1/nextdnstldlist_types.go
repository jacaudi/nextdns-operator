package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TLDEntry represents a TLD in the block list
type TLDEntry struct {
	// TLD is the top-level domain (without the dot)
	// Examples: "com", "net", "co.uk"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`
	TLD string `json:"tld"`

	// Active indicates if this TLD is blocked
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`

	// Reason documents why this TLD is blocked
	// +optional
	Reason string `json:"reason,omitempty"`
}

// NextDNSTLDListSpec defines the desired state of NextDNSTLDList
type NextDNSTLDListSpec struct {
	// Description provides context for this TLD list
	// +optional
	Description string `json:"description,omitempty"`

	// TLDs is the list of top-level domains to block
	// +kubebuilder:validation:MinItems=1
	TLDs []TLDEntry `json:"tlds"`
}

// NextDNSTLDListStatus defines the observed state of NextDNSTLDList
type NextDNSTLDListStatus struct {
	// TLDCount is the number of active TLDs
	// +optional
	TLDCount int `json:"tldCount,omitempty"`

	// ProfileRefs lists profiles using this TLD list
	// +optional
	ProfileRefs []ResourceReference `json:"profileRefs,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="TLDs",type=integer,JSONPath=`.status.tldCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSTLDList is the Schema for the nextdnstldlists API
type NextDNSTLDList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NextDNSTLDListSpec   `json:"spec,omitempty"`
	Status NextDNSTLDListStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSTLDListList contains a list of NextDNSTLDList
type NextDNSTLDListList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NextDNSTLDList `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NextDNSTLDList{}, &NextDNSTLDListList{})
}
