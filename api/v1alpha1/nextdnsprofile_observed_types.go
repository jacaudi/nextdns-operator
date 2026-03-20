package v1alpha1

// ObservedConfig represents the full observed state of a remote NextDNS profile.
// This is populated by the controller in observe mode and is read-only for users.
type ObservedConfig struct {
	// Name is the profile name as shown in NextDNS dashboard
	// +optional
	Name string `json:"name,omitempty"`

	// Security contains observed security settings
	// +optional
	Security *ObservedSecurity `json:"security,omitempty"`

	// Privacy contains observed privacy settings
	// +optional
	Privacy *ObservedPrivacy `json:"privacy,omitempty"`

	// ParentalControl contains observed parental control settings
	// +optional
	ParentalControl *ObservedParentalControl `json:"parentalControl,omitempty"`

	// Denylist contains observed denied domains
	// +optional
	Denylist []ObservedDomainEntry `json:"denylist,omitempty"`

	// Allowlist contains observed allowed domains
	// +optional
	Allowlist []ObservedDomainEntry `json:"allowlist,omitempty"`

	// Settings contains observed general settings
	// +optional
	Settings *ObservedSettings `json:"settings,omitempty"`

	// Rewrites contains observed DNS rewrites
	// +optional
	Rewrites []ObservedRewriteEntry `json:"rewrites,omitempty"`

	// BlockedTLDs contains observed blocked TLDs
	// +optional
	BlockedTLDs []string `json:"blockedTLDs,omitempty"`
}

// ObservedSecurity represents observed security settings
type ObservedSecurity struct {
	AIThreatDetection       bool `json:"aiThreatDetection"`
	ThreatIntelligenceFeeds bool `json:"threatIntelligenceFeeds"`
	GoogleSafeBrowsing      bool `json:"googleSafeBrowsing"`
	Cryptojacking           bool `json:"cryptojacking"`
	DNSRebinding            bool `json:"dnsRebinding"`
	IDNHomographs           bool `json:"idnHomographs"`
	Typosquatting           bool `json:"typosquatting"`
	DGA                     bool `json:"dga"`
	NRD                     bool `json:"nrd"`
	DDNS                    bool `json:"ddns"`
	Parking                 bool `json:"parking"`
	CSAM                    bool `json:"csam"`
}

// ObservedPrivacy represents observed privacy settings
type ObservedPrivacy struct {
	DisguisedTrackers bool                     `json:"disguisedTrackers"`
	AllowAffiliate    bool                     `json:"allowAffiliate"`
	Blocklists        []ObservedBlocklistEntry `json:"blocklists,omitempty"`
	Natives           []ObservedNativeEntry    `json:"natives,omitempty"`
}

// ObservedBlocklistEntry represents an observed privacy blocklist
type ObservedBlocklistEntry struct {
	ID string `json:"id"`
}

// ObservedNativeEntry represents an observed native tracker protection
type ObservedNativeEntry struct {
	ID string `json:"id"`
}

// ObservedParentalControl represents observed parental control settings
type ObservedParentalControl struct {
	SafeSearch            bool                    `json:"safeSearch"`
	YouTubeRestrictedMode bool                    `json:"youtubeRestrictedMode"`
	Categories            []ObservedCategoryEntry `json:"categories,omitempty"`
	Services              []ObservedServiceEntry  `json:"services,omitempty"`
}

// ObservedCategoryEntry represents an observed content category
type ObservedCategoryEntry struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
}

// ObservedServiceEntry represents an observed blocked service
type ObservedServiceEntry struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
}

// ObservedDomainEntry represents an observed domain with active state
type ObservedDomainEntry struct {
	Domain string `json:"domain"`
	Active bool   `json:"active"`
}

// ObservedSettings represents observed general settings
type ObservedSettings struct {
	Logs        *ObservedLogs        `json:"logs,omitempty"`
	BlockPage   *ObservedBlockPage   `json:"blockPage,omitempty"`
	Performance *ObservedPerformance `json:"performance,omitempty"`
	Web3        bool                 `json:"web3"`
}

// ObservedLogs represents observed logging settings
type ObservedLogs struct {
	Enabled   bool `json:"enabled"`
	Retention int  `json:"retention,omitempty"`
}

// ObservedBlockPage represents observed block page settings
type ObservedBlockPage struct {
	Enabled bool `json:"enabled"`
}

// ObservedPerformance represents observed performance settings
type ObservedPerformance struct {
	ECS             bool `json:"ecs"`
	CacheBoost      bool `json:"cacheBoost"`
	CNAMEFlattening bool `json:"cnameFlattening"`
}

// ObservedRewriteEntry represents an observed DNS rewrite
type ObservedRewriteEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
