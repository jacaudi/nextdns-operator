package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
// +kubebuilder:validation:Enum=ClusterIP;LoadBalancer
type CoreDNSServiceType string

const (
	// ServiceTypeClusterIP exposes the service on a cluster-internal IP
	ServiceTypeClusterIP CoreDNSServiceType = "ClusterIP"
	// ServiceTypeLoadBalancer exposes the service via a cloud load balancer
	ServiceTypeLoadBalancer CoreDNSServiceType = "LoadBalancer"
)

// UpstreamConfig specifies how to connect to NextDNS upstream servers
type UpstreamConfig struct {
	// Primary specifies the primary protocol for DNS queries
	// +kubebuilder:validation:Required
	// +kubebuilder:default=DoT
	Primary DNSProtocol `json:"primary"`

	// DeviceName identifies this CoreDNS instance in NextDNS Analytics and Logs.
	// The name is embedded in the upstream endpoint: prepended to the DoT SNI
	// hostname or appended to the DoH URL path.
	// Only alphanumeric characters, hyphens, and spaces are allowed.
	// Spaces are converted to -- for DoT and URL-encoded for DoH.
	// Ignored when using plain DNS protocol.
	// +optional
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[-a-zA-Z0-9 ]+$`
	DeviceName string `json:"deviceName,omitempty"`
}

// CoreDNSDeploymentConfig configures the CoreDNS deployment
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
	// +kubebuilder:default="mirror.gcr.io/coredns/coredns:1.13.1"
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

	// PodAnnotations specifies additional annotations for the CoreDNS pods
	// Useful for Multus CNI network attachments, Istio sidecar injection control, etc.
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// PodDisruptionBudget configures disruption budget for HA deployments
	// +optional
	PodDisruptionBudget *CoreDNSPDBConfig `json:"podDisruptionBudget,omitempty"`
}

// CoreDNSPDBConfig configures PodDisruptionBudget for CoreDNS HA deployments
type CoreDNSPDBConfig struct {
	// MinAvailable is the minimum number of pods that must be available.
	// Mutually exclusive with MaxUnavailable.
	// +optional
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`

	// MaxUnavailable is the maximum number of pods that can be unavailable.
	// Mutually exclusive with MinAvailable. Defaults to 1 if neither is set.
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// CoreDNSServiceConfig configures the CoreDNS Kubernetes Service
type CoreDNSServiceConfig struct {
	// Type specifies the type of Service
	// +kubebuilder:default=ClusterIP
	// +optional
	Type CoreDNSServiceType `json:"type,omitempty"`

	// LoadBalancerIP specifies the IP address for LoadBalancer type services.
	// Accepts any valid IPv4 or IPv6 address.
	// Deprecated: This field is deprecated since Kubernetes v1.24 but is still
	// honored by most cloud providers. Future versions may migrate to
	// Service annotations or a gateway API mechanism.
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`

	// Annotations specifies additional annotations for the Service
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// NameOverride overrides the generated service name
	// +optional
	NameOverride string `json:"nameOverride,omitempty"`
}

// CoreDNSMetricsConfig configures metrics and monitoring
type CoreDNSMetricsConfig struct {
	// Enabled enables the metrics endpoint on CoreDNS
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
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

// DomainOverride specifies a domain-specific DNS upstream configuration
type DomainOverride struct {
	// Domain is the DNS domain to override (e.g., "corp.example.com")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Domain string `json:"domain"`

	// Upstreams is the list of upstream DNS server IPs for this domain
	// Each entry must be a valid IPv4 address or IPv4:port
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Upstreams []string `json:"upstreams"`

	// CacheTTL specifies the cache TTL for this domain in seconds (optional)
	// +kubebuilder:validation:Minimum=0
	// +optional
	CacheTTL *int32 `json:"cacheTTL,omitempty"`
}

// MultusConfig configures secondary network attachment via Multus CNI
type MultusConfig struct {
	// NetworkAttachmentDefinition is the name of the existing
	// NetworkAttachmentDefinition CR to attach to CoreDNS pods
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	NetworkAttachmentDefinition string `json:"networkAttachmentDefinition"`

	// Namespace is the namespace of the NetworkAttachmentDefinition
	// Defaults to the namespace of the NextDNSCoreDNS resource
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// IPs is an optional list of static IPs to request from IPAM
	// When specified, the IPAM plugin assigns one per pod from this list
	// The number of IPs should be >= the number of replicas
	// +optional
	IPs []string `json:"ips,omitempty"`
}

// GatewayConfig configures Gateway API resources for DNS traffic exposure
type GatewayConfig struct {
	// Addresses specifies the IP addresses for the Gateway.
	// These are requested from the Gateway implementation (e.g., Envoy Gateway + Cilium LB IPAM).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Addresses []GatewayAddress `json:"addresses"`

	// Annotations specifies additional annotations for the Gateway resource
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GatewayAddress specifies an address for the Gateway
type GatewayAddress struct {
	// Type specifies the address type
	// +kubebuilder:validation:Enum=IPAddress;Hostname
	// +kubebuilder:default=IPAddress
	// +optional
	Type *string `json:"type,omitempty"`

	// Value is the address value (e.g., "192.168.1.53")
	// +kubebuilder:validation:Required
	Value string `json:"value"`
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

	// DomainOverrides configures domain-specific DNS upstream servers
	// Queries for these domains will be forwarded to the specified upstreams
	// instead of NextDNS
	// +optional
	DomainOverrides []DomainOverride `json:"domainOverrides,omitempty"`

	// Multus configures a secondary network interface via Multus CNI
	// +optional
	Multus *MultusConfig `json:"multus,omitempty"`

	// Gateway configures Gateway API resources for DNS traffic exposure.
	// When set, the operator creates a Gateway with TCPRoute and UDPRoute
	// instead of a LoadBalancer Service. A ClusterIP Service is always
	// created as the route backend target.
	// Mutually exclusive with Service.Type=LoadBalancer.
	// +optional
	Gateway *GatewayConfig `json:"gateway,omitempty"`
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

	// MultusIPs lists the IPs assigned to pods via Multus
	// +optional
	MultusIPs []string `json:"multusIPs,omitempty"`

	// Upstream is the status of the NextDNS upstream connection
	// +optional
	Upstream *UpstreamStatus `json:"upstream,omitempty"`

	// Replicas is the status of the deployment replicas
	// +optional
	Replicas *ReplicaStatus `json:"replicas,omitempty"`

	// Ready indicates if the CoreDNS deployment is fully ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// GatewayReady indicates if the Gateway is programmed and accepting traffic
	// +optional
	GatewayReady bool `json:"gatewayReady,omitempty"`

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
