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

// ForwardPolicy controls the failover policy for upstream selection
// in the CoreDNS forward plugin.
// +kubebuilder:validation:Enum=random;round_robin;sequential
type ForwardPolicy string

const (
	// ForwardPolicyRandom selects upstreams at random (CoreDNS default).
	ForwardPolicyRandom ForwardPolicy = "random"
	// ForwardPolicyRoundRobin distributes queries across upstreams in order.
	ForwardPolicyRoundRobin ForwardPolicy = "round_robin"
	// ForwardPolicySequential always tries upstreams in declared order,
	// only failing over on error.
	ForwardPolicySequential ForwardPolicy = "sequential"
)

// ForwardTuningConfig exposes performance and reliability knobs for the
// CoreDNS forward plugin used to send queries upstream to NextDNS.
// Maps to https://coredns.io/plugins/forward/.
type ForwardTuningConfig struct {
	// Policy controls failover between upstream addresses.
	// One of: random (default), round_robin, sequential.
	// +optional
	Policy ForwardPolicy `json:"policy,omitempty"`

	// MaxConcurrent caps the number of concurrent queries forwarded
	// upstream. CoreDNS default is unlimited.
	// +kubebuilder:validation:Minimum=1
	// +optional
	MaxConcurrent *int32 `json:"maxConcurrent,omitempty"`

	// HealthCheck is the interval between upstream health checks
	// (e.g., "5s", "500ms"). Must be a Go duration string.
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	HealthCheck string `json:"healthCheck,omitempty"`

	// Expire is how long to keep idle upstream connections before
	// closing them (e.g., "30s"). Must be a Go duration string.
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	Expire string `json:"expire,omitempty"`

	// MaxFails is the number of failed health checks before an
	// upstream is marked down. CoreDNS default is 2.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFails *int32 `json:"maxFails,omitempty"`
}

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

	// Forward exposes tuning options for the CoreDNS forward plugin
	// used to send queries upstream to NextDNS. All fields optional;
	// CoreDNS defaults are used when omitted.
	// +optional
	Forward *ForwardTuningConfig `json:"forward,omitempty"`
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

	// Port is the TCP port the prometheus plugin listens on.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=9153
	// +optional
	Port *int32 `json:"port,omitempty"`
}

// CoreDNSHealthConfig configures the CoreDNS health plugin used for
// liveness probing. Maps to https://coredns.io/plugins/health/.
type CoreDNSHealthConfig struct {
	// Enabled enables the health plugin. Defaults to true.
	// Disabling this also removes the deployment's liveness probe.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Port is the TCP port the health plugin listens on.
	// Must match the deployment's liveness probe port (the operator
	// keeps these in sync automatically).
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8080
	// +optional
	Port *int32 `json:"port,omitempty"`

	// Lameduck delays health endpoint failure during shutdown so load
	// balancers can drain traffic cleanly. Must be a Go duration string
	// (e.g., "10s"). When unset, the lameduck directive is omitted.
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	Lameduck string `json:"lameduck,omitempty"`
}

// CoreDNSReadyConfig configures the CoreDNS ready plugin used for
// readiness probing. Maps to https://coredns.io/plugins/ready/.
type CoreDNSReadyConfig struct {
	// Enabled enables the ready plugin. Defaults to true.
	// Disabling this also removes the deployment's readiness probe.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Port is the TCP port the ready plugin listens on.
	// Must match the deployment's readiness probe port (the operator
	// keeps these in sync automatically).
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8181
	// +optional
	Port *int32 `json:"port,omitempty"`
}

// ConsolidateRule defines a single CoreDNS errors plugin consolidate
// directive used to reduce log spam from repeated errors.
type ConsolidateRule struct {
	// Interval is the consolidation window (Go duration string,
	// e.g., "5m").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	Interval string `json:"interval"`

	// Pattern is the regular expression matched against error log lines.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Pattern string `json:"pattern"`
}

// CoreDNSErrorsConfig configures the CoreDNS errors plugin.
// Maps to https://coredns.io/plugins/errors/.
type CoreDNSErrorsConfig struct {
	// Enabled enables the errors plugin. Defaults to true.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Consolidate optionally reduces log spam by consolidating
	// repeated error messages matching a pattern within an interval.
	// +optional
	Consolidate []ConsolidateRule `json:"consolidate,omitempty"`
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
	// GatewayClassName specifies which GatewayClass to use for the Gateway.
	// This must reference a GatewayClass managed by an external controller
	// (e.g., Envoy Gateway, Cilium, Istio).
	// If omitted, uses the operator's default gateway class name.
	// +optional
	GatewayClassName *string `json:"gatewayClassName,omitempty"`

	// Addresses specifies the IP addresses for the Gateway.
	// These are requested from the Gateway implementation (e.g., Envoy Gateway + Cilium LB IPAM).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Addresses []GatewayAddress `json:"addresses"`

	// Replicas sets the desired replica count for the gateway proxy pods.
	// How this is applied depends on the gateway implementation. For Envoy
	// Gateway, the operator generates an EnvoyProxy CR and wires it via
	// infrastructure.parametersRef automatically. For unsupported
	// implementations, this field is ignored with a warning condition.
	// Mutually exclusive with infrastructure.parametersRef.
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Annotations specifies additional annotations for the Gateway resource
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Infrastructure defines metadata propagated to resources created by the
	// Gateway implementation (e.g., LoadBalancer Service, Deployment).
	// +optional
	Infrastructure *GatewayInfrastructure `json:"infrastructure,omitempty"`
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

// GatewayInfrastructure defines infrastructure-level metadata to propagate
// to resources created by the Gateway implementation (e.g., LoadBalancer Service).
type GatewayInfrastructure struct {
	// Annotations to propagate to generated infrastructure resources.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to propagate to generated infrastructure resources.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// ParametersRef is a reference to an implementation-specific resource
	// that contains additional configuration for the Gateway infrastructure.
	// +optional
	ParametersRef *GatewayParametersReference `json:"parametersRef,omitempty"`
}

// GatewayParametersReference identifies a resource that contains
// implementation-specific Gateway infrastructure parameters.
type GatewayParametersReference struct {
	// Group is the API group of the referent.
	// +kubebuilder:validation:Required
	Group string `json:"group"`

	// Kind is the kind of the referent.
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name is the name of the referent.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// RewriteRule defines a single CoreDNS rewrite plugin rule.
// Maps to the CoreDNS rewrite plugin: https://coredns.io/plugins/rewrite/
type RewriteRule struct {
	// Type is the rewrite type. Maps to the first argument of the
	// CoreDNS rewrite directive. The most common value is "name", which
	// rewrites query names. Other values like "class", "type", "ttl",
	// and "edns0" are also supported by CoreDNS.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=name;class;type;ttl;edns0
	Type string `json:"type"`

	// Match is the input pattern (or attribute) the rule matches against.
	// For type=name this is the query name pattern (interpreted according
	// to the optional matcher: exact, prefix, suffix, substring, regex).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Match string `json:"match"`

	// Replacement is the value the matched query is rewritten to.
	// For type=name this is the replacement query name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Replacement string `json:"replacement"`

	// Matcher is an optional sub-type for type=name rewrites.
	// One of: exact, prefix, suffix, substring, regex.
	// When omitted, CoreDNS uses its default (exact match).
	// +optional
	// +kubebuilder:validation:Enum=exact;prefix;suffix;substring;regex
	Matcher string `json:"matcher,omitempty"`
}

// HostsEntry is a single static IP-to-hostname mapping for the
// CoreDNS hosts plugin. One entry can map a single IP to multiple
// hostnames (matching the /etc/hosts file format).
type HostsEntry struct {
	// IP is the IP address (IPv4 or IPv6) to return for the listed hostnames.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	IP string `json:"ip"`

	// Hostnames is the list of hostnames that should resolve to IP.
	// At least one hostname is required.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Hostnames []string `json:"hostnames"`
}

// HostsConfig configures the CoreDNS hosts plugin for inline static
// hostname-to-IP overrides. Maps to https://coredns.io/plugins/hosts/.
type HostsConfig struct {
	// Entries is the list of static IP-to-hostname mappings. Emitted
	// inside the hosts plugin block in the generated Corefile.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Entries []HostsEntry `json:"entries"`

	// Fallthrough controls whether queries that do not match a hosts
	// entry are passed to the next plugin in the chain (forward to
	// NextDNS). Defaults to true so unmatched names continue to resolve.
	// +kubebuilder:default=true
	// +optional
	Fallthrough *bool `json:"fallthrough,omitempty"`

	// TTL is the TTL (in seconds) returned for static entries.
	// CoreDNS default is 3600 if unset.
	// +kubebuilder:validation:Minimum=0
	// +optional
	TTL *int32 `json:"ttl,omitempty"`
}

// CorefileSpec groups CoreDNS plugin-level configuration.
// This is the configuration that ends up in the generated Corefile,
// separate from Kubernetes-level deployment concerns (Deployment, Service,
// Multus, Gateway).
type CorefileSpec struct {
	// Upstream configures the upstream DNS connection to NextDNS
	// +optional
	Upstream *UpstreamConfig `json:"upstream,omitempty"`

	// Cache configures DNS response caching
	// +optional
	Cache *CoreDNSCacheConfig `json:"cache,omitempty"`

	// Metrics configures metrics and monitoring
	// +optional
	Metrics *CoreDNSMetricsConfig `json:"metrics,omitempty"`

	// Logging configures DNS query logging
	// +optional
	Logging *CoreDNSLoggingConfig `json:"logging,omitempty"`

	// DomainOverrides configures domain-specific DNS upstream servers.
	// Queries for these domains will be forwarded to the specified upstreams
	// instead of NextDNS.
	// +optional
	DomainOverrides []DomainOverride `json:"domainOverrides,omitempty"`

	// Rewrite configures the CoreDNS rewrite plugin for query rewriting
	// (CNAME flattening, domain remapping, etc.). Rules are emitted in
	// order and fire before the forward directive.
	// +optional
	Rewrite []RewriteRule `json:"rewrite,omitempty"`

	// Hosts configures the CoreDNS hosts plugin for inline static
	// hostname-to-IP overrides without running a separate upstream
	// DNS server.
	// +optional
	Hosts *HostsConfig `json:"hosts,omitempty"`

	// Health configures the CoreDNS health plugin (liveness endpoint).
	// +optional
	Health *CoreDNSHealthConfig `json:"health,omitempty"`

	// Ready configures the CoreDNS ready plugin (readiness endpoint).
	// +optional
	Ready *CoreDNSReadyConfig `json:"ready,omitempty"`

	// Errors configures the CoreDNS errors plugin (error logging).
	// +optional
	Errors *CoreDNSErrorsConfig `json:"errors,omitempty"`
}

// NextDNSCoreDNSSpec defines the desired state of NextDNSCoreDNS
type NextDNSCoreDNSSpec struct {
	// ProfileRef references the NextDNSProfile to use for DNS resolution
	// +kubebuilder:validation:Required
	ProfileRef ResourceReference `json:"profileRef"`

	// Deployment configures the CoreDNS deployment
	// +optional
	Deployment *CoreDNSDeploymentConfig `json:"deployment,omitempty"`

	// Service configures the Kubernetes Service
	// +optional
	Service *CoreDNSServiceConfig `json:"service,omitempty"`

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

	// Corefile groups CoreDNS plugin-level configuration (upstream, cache,
	// metrics, logging, domain overrides).
	// +optional
	Corefile *CorefileSpec `json:"corefile,omitempty"`
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
