package configimport

// ProfileConfigJSON represents the JSON structure for importing a NextDNS
// profile configuration from a ConfigMap. Field names match the CRD spec
// for consistency.
type ProfileConfigJSON struct {
	Security        *SecurityJSON        `json:"security,omitempty"`
	Privacy         *PrivacyJSON         `json:"privacy,omitempty"`
	ParentalControl *ParentalControlJSON `json:"parentalControl,omitempty"`
	Denylist        []DomainEntryJSON    `json:"denylist,omitempty"`
	Allowlist       []DomainEntryJSON    `json:"allowlist,omitempty"`
	Settings        *SettingsJSON        `json:"settings,omitempty"`
	Rewrites        []RewriteEntryJSON   `json:"rewrites,omitempty"`
}

type SecurityJSON struct {
	AIThreatDetection       *bool    `json:"aiThreatDetection,omitempty"`
	GoogleSafeBrowsing      *bool    `json:"googleSafeBrowsing,omitempty"`
	Cryptojacking           *bool    `json:"cryptojacking,omitempty"`
	DNSRebinding            *bool    `json:"dnsRebinding,omitempty"`
	IDNHomographs           *bool    `json:"idnHomographs,omitempty"`
	Typosquatting           *bool    `json:"typosquatting,omitempty"`
	DGA                     *bool    `json:"dga,omitempty"`
	NRD                     *bool    `json:"nrd,omitempty"`
	DDNS                    *bool    `json:"ddns,omitempty"`
	Parking                 *bool    `json:"parking,omitempty"`
	CSAM                    *bool    `json:"csam,omitempty"`
	ThreatIntelligenceFeeds []string `json:"threatIntelligenceFeeds,omitempty"`
}

type PrivacyJSON struct {
	Blocklists        []BlocklistEntryJSON `json:"blocklists,omitempty"`
	Natives           []NativeEntryJSON    `json:"natives,omitempty"`
	DisguisedTrackers *bool                `json:"disguisedTrackers,omitempty"`
	AllowAffiliate    *bool                `json:"allowAffiliate,omitempty"`
}

type BlocklistEntryJSON struct {
	ID     string `json:"id"`
	Active *bool  `json:"active,omitempty"`
}

type NativeEntryJSON struct {
	ID     string `json:"id"`
	Active *bool  `json:"active,omitempty"`
}

type ParentalControlJSON struct {
	Categories            []CategoryEntryJSON `json:"categories,omitempty"`
	Services              []ServiceEntryJSON  `json:"services,omitempty"`
	SafeSearch            *bool               `json:"safeSearch,omitempty"`
	YouTubeRestrictedMode *bool               `json:"youtubeRestrictedMode,omitempty"`
}

type CategoryEntryJSON struct {
	ID     string `json:"id"`
	Active *bool  `json:"active,omitempty"`
}

type ServiceEntryJSON struct {
	ID     string `json:"id"`
	Active *bool  `json:"active,omitempty"`
}

type DomainEntryJSON struct {
	Domain string `json:"domain"`
	Active *bool  `json:"active,omitempty"`
}

type RewriteEntryJSON struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Active *bool  `json:"active,omitempty"`
}

type SettingsJSON struct {
	Logs        *LogsJSON        `json:"logs,omitempty"`
	BlockPage   *BlockPageJSON   `json:"blockPage,omitempty"`
	Performance *PerformanceJSON `json:"performance,omitempty"`
	Web3        *bool            `json:"web3,omitempty"`
}

type LogsJSON struct {
	Enabled       *bool  `json:"enabled,omitempty"`
	LogClientsIPs *bool  `json:"logClientsIPs,omitempty"`
	LogDomains    *bool  `json:"logDomains,omitempty"`
	Retention     string `json:"retention,omitempty"`
}

type BlockPageJSON struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type PerformanceJSON struct {
	ECS             *bool `json:"ecs,omitempty"`
	CacheBoost      *bool `json:"cacheBoost,omitempty"`
	CNAMEFlattening *bool `json:"cnameFlattening,omitempty"`
}
