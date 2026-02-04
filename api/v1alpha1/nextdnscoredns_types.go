package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DNSProtocol specifies the DNS protocol to use for upstream queries
// +kubebuilder:validation:Enum=DoT;DoH;DNS
type DNSProtocol string

const (
	// DNSProtocolDoT uses DNS over TLS (port 853)
	// Recommended for privacy - encrypts DNS queries
	DNSProtocolDoT DNSProtocol = "DoT"
	// DNSProtocolDoH uses DNS over HTTPS
	// Recommended for privacy - encrypts DNS queries
	DNSProtocolDoH DNSProtocol = "DoH"
	// DNSProtocolDNS uses plain DNS (port 53)
	// WARNING: Plain DNS traffic is unencrypted. Your NextDNS profile ID
	// and DNS queries may be visible to network observers. Use DoT or DoH
	// for privacy in untrusted networks.
	DNSProtocolDNS DNSProtocol = "DNS"
)

// DeploymentMode specifies how CoreDNS instances are deployed
// +kubebuilder:validation:Enum=Deployment;DaemonSet
type DeploymentMode string

const (
	// DeploymentModeDeployment uses a Kubernetes Deployment
	DeploymentModeDeployment DeploymentMode = "Deployment"
	// DeploymentModeDaemonSet uses a Kubernetes DaemonSet
	DeploymentModeDaemonSet DeploymentMode = "DaemonSet"
)

// CoreDNSServiceType specifies the type of Kubernetes Service
// +kubebuilder:validation:Enum=ClusterIP;LoadBalancer;NodePort
type CoreDNSServiceType string

const (
	// ServiceTypeClusterIP exposes the service on a cluster-internal IP
	ServiceTypeClusterIP CoreDNSServiceType = "ClusterIP"
	// ServiceTypeLoadBalancer exposes the service via a cloud load balancer
	ServiceTypeLoadBalancer CoreDNSServiceType = "LoadBalancer"
	// ServiceTypeNodePort exposes the service on each node's IP at a static port
	ServiceTypeNodePort CoreDNSServiceType = "NodePort"
)

// UpstreamConfig specifies how to connect to NextDNS upstream servers
type UpstreamConfig struct {
	// Primary specifies the primary protocol for DNS queries
	// +kubebuilder:validation:Required
	// +kubebuilder:default=DoT
	Primary DNSProtocol `json:"primary"`

	// Fallback specifies an optional fallback protocol if primary fails
	// +optional
	Fallback *DNSProtocol `json:"fallback,omitempty"`
}

// CoreDNSDeploymentConfig configures the CoreDNS deployment
// TODO: Consider adding PodDisruptionBudget support for HA deployments.
type CoreDNSDeploymentConfig struct {
	// Mode specifies whether to deploy as Deployment or DaemonSet
	// +kubebuilder:default=Deployment
	// +optional
	Mode DeploymentMode `json:"mode,omitempty"`

	// Replicas specifies the number of CoreDNS replicas (only used when Mode is Deployment)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=2
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Image specifies the CoreDNS container image
	// +kubebuilder:default="coredns/coredns:1.11.1"
	// +optional
	Image string `json:"image,omitempty"`

	// NodeSelector constrains pods to nodes with matching labels
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity specifies pod scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations specifies pod tolerations
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Resources specifies compute resource requirements
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// CoreDNSServiceConfig configures the CoreDNS Kubernetes Service
type CoreDNSServiceConfig struct {
	// Type specifies the type of Service
	// +kubebuilder:default=ClusterIP
	// +optional
	Type CoreDNSServiceType `json:"type,omitempty"`

	// LoadBalancerIP specifies the IP address for LoadBalancer type services.
	// Must be a valid IPv4 address if specified.
	// +kubebuilder:validation:Pattern=`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`

	// Annotations specifies additional annotations for the Service
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// NameOverride overrides the generated service name
	// +optional
	NameOverride string `json:"nameOverride,omitempty"`
}

// ServiceMonitorConfig configures Prometheus ServiceMonitor creation
// TODO: Implement ServiceMonitor reconciliation in the controller.
// Currently this struct is defined but not used.
type ServiceMonitorConfig struct {
	// Enabled creates a ServiceMonitor for Prometheus Operator
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Namespace specifies the namespace for the ServiceMonitor
	// Defaults to the namespace of the NextDNSCoreDNS resource
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Interval specifies the scrape interval
	// +kubebuilder:default="30s"
	// +optional
	Interval string `json:"interval,omitempty"`

	// Labels specifies additional labels for the ServiceMonitor
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// CoreDNSMetricsConfig configures metrics and monitoring
type CoreDNSMetricsConfig struct {
	// Enabled enables the metrics endpoint on CoreDNS
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// ServiceMonitor configures Prometheus ServiceMonitor creation
	// +optional
	ServiceMonitor *ServiceMonitorConfig `json:"serviceMonitor,omitempty"`
}

// CoreDNSCacheConfig configures DNS response caching
type CoreDNSCacheConfig struct {
	// Enabled enables DNS response caching
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// SuccessTTL specifies the TTL for successful responses (in seconds)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=3600
	// +optional
	SuccessTTL *int32 `json:"successTTL,omitempty"`
}

// CoreDNSLoggingConfig configures DNS query logging
type CoreDNSLoggingConfig struct {
	// Enabled enables DNS query logging
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// NextDNSCoreDNSSpec defines the desired state of NextDNSCoreDNS
type NextDNSCoreDNSSpec struct {
	// ProfileRef references the NextDNSProfile to use for DNS resolution
	// +kubebuilder:validation:Required
	ProfileRef ResourceReference `json:"profileRef"`

	// Upstream configures the upstream DNS connection to NextDNS
	// +optional
	Upstream *UpstreamConfig `json:"upstream,omitempty"`

	// Deployment configures the CoreDNS deployment
	// +optional
	Deployment *CoreDNSDeploymentConfig `json:"deployment,omitempty"`

	// Service configures the Kubernetes Service
	// +optional
	Service *CoreDNSServiceConfig `json:"service,omitempty"`

	// Metrics configures metrics and monitoring
	// +optional
	Metrics *CoreDNSMetricsConfig `json:"metrics,omitempty"`

	// Cache configures DNS response caching
	// +optional
	Cache *CoreDNSCacheConfig `json:"cache,omitempty"`

	// Logging configures DNS query logging
	// +optional
	Logging *CoreDNSLoggingConfig `json:"logging,omitempty"`
}

// DNSEndpoint represents a DNS endpoint exposed by the service
type DNSEndpoint struct {
	// IP is the IP address of the DNS endpoint
	IP string `json:"ip"`

	// Port is the port number of the DNS endpoint
	Port int32 `json:"port"`

	// Protocol is the protocol (TCP or UDP)
	Protocol string `json:"protocol"`
}

// UpstreamStatus represents the status of upstream DNS configuration
type UpstreamStatus struct {
	// URL is the NextDNS upstream URL being used
	URL string `json:"url"`
}

// ReplicaStatus represents the status of deployment replicas
type ReplicaStatus struct {
	// Desired is the number of desired replicas
	Desired int32 `json:"desired"`

	// Ready is the number of ready replicas
	Ready int32 `json:"ready"`

	// Available is the number of available replicas
	Available int32 `json:"available"`
}

// NextDNSCoreDNSStatus defines the observed state of NextDNSCoreDNS
type NextDNSCoreDNSStatus struct {
	// ProfileID is the NextDNS profile ID from the referenced profile
	// +optional
	ProfileID string `json:"profileID,omitempty"`

	// Fingerprint is the DNS fingerprint from the referenced profile
	// +optional
	Fingerprint string `json:"fingerprint,omitempty"`

	// Endpoints lists the DNS endpoints exposed by the service
	// +optional
	Endpoints []DNSEndpoint `json:"endpoints,omitempty"`

	// DNSIP is the primary DNS IP address for easy reference
	// +optional
	DNSIP string `json:"dnsIP,omitempty"`

	// Upstream is the status of the NextDNS upstream connection
	// +optional
	Upstream *UpstreamStatus `json:"upstream,omitempty"`

	// Replicas is the status of the deployment replicas
	// +optional
	Replicas *ReplicaStatus `json:"replicas,omitempty"`

	// Ready indicates if the CoreDNS deployment is fully ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastUpdated is the time the status was last updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// ObservedGeneration is the generation last processed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Profile ID",type=string,JSONPath=`.status.profileID`
// +kubebuilder:printcolumn:name="DNS IP",type=string,JSONPath=`.status.dnsIP`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSCoreDNS is the Schema for the nextdnscoredns API
// It deploys a CoreDNS instance configured to forward DNS queries to NextDNS
type NextDNSCoreDNS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NextDNSCoreDNSSpec   `json:"spec,omitempty"`
	Status NextDNSCoreDNSStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSCoreDNSList contains a list of NextDNSCoreDNS
type NextDNSCoreDNSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NextDNSCoreDNS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NextDNSCoreDNS{}, &NextDNSCoreDNSList{})
}
