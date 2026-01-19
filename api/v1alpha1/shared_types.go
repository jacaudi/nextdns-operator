package v1alpha1

// ResourceReference identifies a Kubernetes resource
type ResourceReference struct {
	// Name of the resource
	Name string `json:"name"`

	// Namespace of the resource (optional, defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ListReference references a list CRD (allowlist, denylist, or TLD list)
type ListReference struct {
	// Name of the list resource
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the list resource (defaults to profile's namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// SecretKeySelector references a key in a Secret
type SecretKeySelector struct {
	// Name is the name of the Secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key within the Secret
	// +kubebuilder:default=api-key
	// +optional
	Key string `json:"key,omitempty"`
}

// DomainEntry represents a domain in allow/deny lists
type DomainEntry struct {
	// Domain is the domain name (supports wildcards like *.example.com)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Domain string `json:"domain"`

	// Active indicates if this entry is enabled
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`

	// Reason documents why this domain is in the list
	// +optional
	Reason string `json:"reason,omitempty"`
}

// RewriteEntry defines a DNS rewrite rule
type RewriteEntry struct {
	// From is the source domain
	// +kubebuilder:validation:Required
	From string `json:"from"`

	// To is the target (IP or domain)
	// +kubebuilder:validation:Required
	To string `json:"to"`

	// Active indicates if this rewrite is enabled
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`
}

// ReferencedResourceStatus tracks the status of a referenced resource
type ReferencedResourceStatus struct {
	// Name of the resource
	Name string `json:"name"`

	// Namespace of the resource
	Namespace string `json:"namespace"`

	// Ready indicates if the resource is ready
	Ready bool `json:"ready"`

	// Count of items (domains or TLDs)
	// +optional
	Count int `json:"count,omitempty"`
}
