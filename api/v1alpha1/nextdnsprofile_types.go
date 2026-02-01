package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapRef configures the optional ConfigMap containing connection details
type ConfigMapRef struct {
	// Enabled enables creation of the ConfigMap
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Name is the name of the ConfigMap to create
	// If not specified, defaults to "<profile-name>-nextdns"
	// +optional
	Name string `json:"name,omitempty"`
}

// NextDNSProfileSpec defines the desired state of NextDNSProfile
type NextDNSProfileSpec struct {
	// Name is the human-readable name shown in NextDNS dashboard
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// CredentialsRef references a Secret containing the NextDNS API key
	// +kubebuilder:validation:Required
	CredentialsRef SecretKeySelector `json:"credentialsRef"`

	// ProfileID optionally specifies an existing NextDNS profile to manage
	// If not set, a new profile will be created
	// +optional
	ProfileID string `json:"profileID,omitempty"`

	// ===========================================
	// List References (Multi-CRD Architecture)
	// ===========================================

	// AllowlistRefs references NextDNSAllowlist resources
	// Domains from all referenced allowlists are merged
	// +optional
	AllowlistRefs []ListReference `json:"allowlistRefs,omitempty"`

	// DenylistRefs references NextDNSDenylist resources
	// Domains from all referenced denylists are merged
	// +optional
	DenylistRefs []ListReference `json:"denylistRefs,omitempty"`

	// TLDListRefs references NextDNSTLDList resources
	// TLDs from all referenced lists are merged
	// +optional
	TLDListRefs []ListReference `json:"tldListRefs,omitempty"`

	// ===========================================
	// Inline Lists (for simple cases)
	// ===========================================

	// Denylist specifies inline domains to block (merged with DenylistRefs)
	// +optional
	Denylist []DomainEntry `json:"denylist,omitempty"`

	// Allowlist specifies inline domains to allow (merged with AllowlistRefs)
	// +optional
	Allowlist []DomainEntry `json:"allowlist,omitempty"`

	// ===========================================
	// Other Settings
	// ===========================================

	// Security configures threat protection settings
	// +optional
	Security *SecuritySpec `json:"security,omitempty"`

	// Privacy configures tracker and ad blocking
	// +optional
	Privacy *PrivacySpec `json:"privacy,omitempty"`

	// ParentalControl configures content filtering
	// +optional
	ParentalControl *ParentalControlSpec `json:"parentalControl,omitempty"`

	// Rewrites specifies DNS rewrites
	// +optional
	Rewrites []RewriteEntry `json:"rewrites,omitempty"`

	// Settings configures logging, performance, and other options
	// +optional
	Settings *SettingsSpec `json:"settings,omitempty"`

	// ConfigMapRef configures optional ConfigMap creation with connection details
	// +optional
	ConfigMapRef *ConfigMapRef `json:"configMapRef,omitempty"`
}

// SecuritySpec defines security/threat protection settings
type SecuritySpec struct {
	// AIThreatDetection enables AI-based threat detection
	// +kubebuilder:default=true
	// +optional
	AIThreatDetection *bool `json:"aiThreatDetection,omitempty"`

	// ThreatIntelligenceFeeds specifies which threat feeds to use
	// +optional
	ThreatIntelligenceFeeds []string `json:"threatIntelligenceFeeds,omitempty"`

	// GoogleSafeBrowsing enables Google Safe Browsing protection
	// +kubebuilder:default=true
	// +optional
	GoogleSafeBrowsing *bool `json:"googleSafeBrowsing,omitempty"`

	// Cryptojacking blocks cryptomining scripts
	// +kubebuilder:default=true
	// +optional
	Cryptojacking *bool `json:"cryptojacking,omitempty"`

	// DNSRebinding protects against DNS rebinding attacks
	// +kubebuilder:default=true
	// +optional
	DNSRebinding *bool `json:"dnsRebinding,omitempty"`

	// IDNHomographs blocks IDN homograph attacks
	// +kubebuilder:default=true
	// +optional
	IDNHomographs *bool `json:"idnHomographs,omitempty"`

	// Typosquatting blocks typosquatting domains
	// +kubebuilder:default=true
	// +optional
	Typosquatting *bool `json:"typosquatting,omitempty"`

	// DGA blocks algorithmically-generated domains
	// +kubebuilder:default=true
	// +optional
	DGA *bool `json:"dga,omitempty"`

	// NRD blocks newly registered domains
	// +kubebuilder:default=false
	// +optional
	NRD *bool `json:"nrd,omitempty"`

	// DDNS blocks dynamic DNS hostnames
	// +kubebuilder:default=false
	// +optional
	DDNS *bool `json:"ddns,omitempty"`

	// Parking blocks parked domains
	// +kubebuilder:default=true
	// +optional
	Parking *bool `json:"parking,omitempty"`

	// CSAM blocks child sexual abuse material
	// +kubebuilder:default=true
	// +optional
	CSAM *bool `json:"csam,omitempty"`
}

// PrivacySpec defines privacy and ad-blocking settings
type PrivacySpec struct {
	// Blocklists specifies which ad/tracker blocklists to enable
	// +optional
	Blocklists []BlocklistEntry `json:"blocklists,omitempty"`

	// Natives specifies native tracking protection (per-vendor)
	// +optional
	Natives []NativeEntry `json:"natives,omitempty"`

	// DisguisedTrackers blocks trackers using CNAME cloaking
	// +kubebuilder:default=true
	// +optional
	DisguisedTrackers *bool `json:"disguisedTrackers,omitempty"`

	// AllowAffiliate allows affiliate & tracking links
	// +kubebuilder:default=false
	// +optional
	AllowAffiliate *bool `json:"allowAffiliate,omitempty"`
}

// BlocklistEntry references a privacy blocklist
type BlocklistEntry struct {
	// ID is the blocklist identifier (e.g., "nextdns-recommended", "oisd")
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// Active indicates if this blocklist is enabled
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`
}

// NativeEntry configures native tracker blocking for a vendor
type NativeEntry struct {
	// ID is the vendor identifier (e.g., "apple", "windows", "samsung")
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// Active indicates if blocking is enabled for this vendor
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`
}

// ParentalControlSpec defines parental control settings
type ParentalControlSpec struct {
	// Categories specifies content categories to block
	// +optional
	Categories []CategoryEntry `json:"categories,omitempty"`

	// Services specifies specific services to block
	// +optional
	Services []ServiceEntry `json:"services,omitempty"`

	// SafeSearch enforces safe search on search engines
	// +kubebuilder:default=false
	// +optional
	SafeSearch *bool `json:"safeSearch,omitempty"`

	// YouTubeRestrictedMode enforces YouTube restricted mode
	// +kubebuilder:default=false
	// +optional
	YouTubeRestrictedMode *bool `json:"youtubeRestrictedMode,omitempty"`
}

// CategoryEntry references a content category
type CategoryEntry struct {
	// ID is the category identifier (e.g., "gambling", "adult", "violence")
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// Active indicates if this category is blocked
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`
}

// ServiceEntry references a specific service
type ServiceEntry struct {
	// ID is the service identifier (e.g., "tiktok", "youtube", "facebook")
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// Active indicates if this service is blocked
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`
}

// SettingsSpec defines general profile settings
type SettingsSpec struct {
	// Logs configures query logging
	// +optional
	Logs *LogsSpec `json:"logs,omitempty"`

	// BlockPage configures the block page
	// +optional
	BlockPage *BlockPageSpec `json:"blockPage,omitempty"`

	// Performance configures performance optimizations
	// +optional
	Performance *PerformanceSpec `json:"performance,omitempty"`

	// Web3 enables Web3 domain resolution
	// +kubebuilder:default=false
	// +optional
	Web3 *bool `json:"web3,omitempty"`
}

// LogsSpec configures logging settings
type LogsSpec struct {
	// Enabled turns logging on/off
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// LogClientsIPs logs client IP addresses
	// +kubebuilder:default=false
	// +optional
	LogClientsIPs *bool `json:"logClientsIPs,omitempty"`

	// LogDomains logs queried domains
	// +kubebuilder:default=true
	// +optional
	LogDomains *bool `json:"logDomains,omitempty"`

	// Retention specifies log retention period (1h, 6h, 1d, 7d, 30d, 90d, 1y, 2y)
	// +kubebuilder:default="7d"
	// +optional
	Retention string `json:"retention,omitempty"`
}

// BlockPageSpec configures the block page
type BlockPageSpec struct {
	// Enabled shows a block page instead of failing silently
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// PerformanceSpec configures performance settings
type PerformanceSpec struct {
	// ECS enables EDNS Client Subnet
	// +kubebuilder:default=true
	// +optional
	ECS *bool `json:"ecs,omitempty"`

	// CacheBoost enables extended caching
	// +kubebuilder:default=true
	// +optional
	CacheBoost *bool `json:"cacheBoost,omitempty"`

	// CNAMEFlattening enables CNAME flattening
	// +kubebuilder:default=true
	// +optional
	CNAMEFlattening *bool `json:"cnameFlattening,omitempty"`
}

// AggregatedCounts tracks total counts from all sources
type AggregatedCounts struct {
	// AllowlistDomains is the total count of allowlisted domains
	AllowlistDomains int `json:"allowlistDomains,omitempty"`

	// DenylistDomains is the total count of denylisted domains
	DenylistDomains int `json:"denylistDomains,omitempty"`

	// BlockedTLDs is the total count of blocked TLDs
	BlockedTLDs int `json:"blockedTLDs,omitempty"`
}

// ReferencedResources tracks the status of all referenced resources
type ReferencedResources struct {
	// Allowlists lists the status of referenced allowlists
	// +optional
	Allowlists []ReferencedResourceStatus `json:"allowlists,omitempty"`

	// Denylists lists the status of referenced denylists
	// +optional
	Denylists []ReferencedResourceStatus `json:"denylists,omitempty"`

	// TLDLists lists the status of referenced TLD lists
	// +optional
	TLDLists []ReferencedResourceStatus `json:"tldLists,omitempty"`
}

// NextDNSProfileStatus defines the observed state of NextDNSProfile
type NextDNSProfileStatus struct {
	// ProfileID is the NextDNS-assigned profile identifier
	// +optional
	ProfileID string `json:"profileID,omitempty"`

	// Fingerprint is the DNS endpoint (e.g., "abc123.dns.nextdns.io")
	// +optional
	Fingerprint string `json:"fingerprint,omitempty"`

	// AggregatedCounts tracks totals from all sources
	// +optional
	AggregatedCounts *AggregatedCounts `json:"aggregatedCounts,omitempty"`

	// ReferencedResources tracks the status of referenced resources
	// +optional
	ReferencedResources *ReferencedResources `json:"referencedResources,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime is the last time the profile was synced with NextDNS
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// ObservedGeneration is the generation last processed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Profile ID",type=string,JSONPath=`.status.profileID`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSProfile is the Schema for the nextdnsprofiles API
type NextDNSProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NextDNSProfileSpec   `json:"spec,omitempty"`
	Status NextDNSProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSProfileList contains a list of NextDNSProfile
type NextDNSProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NextDNSProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NextDNSProfile{}, &NextDNSProfileList{})
}
