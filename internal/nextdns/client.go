package nextdns

import (
	"context"
	"fmt"
	"time"

	"github.com/jacaudi/nextdns-go/nextdns"
	"github.com/jacaudi/nextdns-operator/internal/metrics"
)

// Client wraps the NextDNS API client
type Client struct {
	client *nextdns.Client
}

// NewClient creates a new NextDNS API client
func NewClient(apiKey string) (*Client, error) {
	client, err := nextdns.New(
		nextdns.WithAPIKey(apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NextDNS client: %w", err)
	}

	return &Client{client: client}, nil
}

// ProfileConfig represents the configuration for a NextDNS profile
type ProfileConfig struct {
	Name            string
	Security        *SecurityConfig
	Privacy         *PrivacyConfig
	ParentalControl *ParentalControlConfig
	Denylist        []string
	Allowlist       []string
	BlockedTLDs     []string
	Settings        *SettingsConfig
}

// SecurityConfig represents security settings
type SecurityConfig struct {
	ThreatIntelligenceFeeds bool
	AIThreatDetection       bool
	GoogleSafeBrowsing      bool
	Cryptojacking           bool
	DNSRebinding            bool
	IDNHomographs           bool
	Typosquatting           bool
	DGA                     bool
	NRD                     bool
	DDNS                    bool
	Parking                 bool
	CSAM                    bool
}

// PrivacyConfig represents privacy settings
type PrivacyConfig struct {
	Blocklists        []string
	Natives           []string
	DisguisedTrackers bool
	AllowAffiliate    bool
}

// ParentalControlConfig represents parental control settings
type ParentalControlConfig struct {
	Categories            []string
	Services              []string
	SafeSearch            bool
	YouTubeRestrictedMode bool
	BlockBypass           bool
}

// SettingsConfig represents general settings
type SettingsConfig struct {
	LogsEnabled     bool
	LogClientsIPs   bool
	LogDomains      bool
	LogRetention    int
	Location        string
	BlockPageEnable bool
	Web3            bool
	BAV             bool
	// Performance settings
	Ecs             bool
	CacheBoost      bool
	CnameFlattening bool
}

// DomainEntry represents a domain with its active state for syncing to NextDNS
type DomainEntry struct {
	Domain string
	Active bool
}

// RewriteEntry represents a DNS rewrite for syncing to NextDNS
type RewriteEntry struct {
	Name    string
	Content string
}

// CreateProfile creates a new NextDNS profile and returns the profile ID
func (c *Client) CreateProfile(ctx context.Context, name string) (string, error) {
	start := time.Now()
	request := &nextdns.CreateProfileRequest{
		Name: name,
	}

	profileID, err := c.client.Profiles.Create(ctx, request)
	duration := time.Since(start).Seconds()
	metrics.RecordAPIRequest("CreateProfile", duration, err == nil)

	if err != nil {
		return "", fmt.Errorf("failed to create profile: %w", err)
	}

	return profileID, nil
}

// GetProfile retrieves a NextDNS profile by ID
func (c *Client) GetProfile(ctx context.Context, profileID string) (*nextdns.Profile, error) {
	start := time.Now()
	request := &nextdns.GetProfileRequest{
		ProfileID: profileID,
	}

	profile, err := c.client.Profiles.Get(ctx, request)
	metrics.RecordAPIRequest("GetProfile", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

// UpdateProfile updates a NextDNS profile name
func (c *Client) UpdateProfile(ctx context.Context, profileID, name string) error {
	start := time.Now()
	request := &nextdns.UpdateProfileRequest{
		ProfileID: profileID,
		Profile: &nextdns.Profile{
			Name: name,
		},
	}

	err := c.client.Profiles.Update(ctx, request)
	metrics.RecordAPIRequest("UpdateProfile", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// DeleteProfile deletes a NextDNS profile
func (c *Client) DeleteProfile(ctx context.Context, profileID string) error {
	start := time.Now()
	request := &nextdns.DeleteProfileRequest{
		ProfileID: profileID,
	}

	err := c.client.Profiles.Delete(ctx, request)
	metrics.RecordAPIRequest("DeleteProfile", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	return nil
}

// UpdateSecurity updates security settings for a profile
func (c *Client) UpdateSecurity(ctx context.Context, profileID string, config *SecurityConfig) error {
	if config == nil {
		return nil
	}

	start := time.Now()
	request := &nextdns.UpdateSecurityRequest{
		ProfileID: profileID,
		Security: &nextdns.Security{
			ThreatIntelligenceFeeds: config.ThreatIntelligenceFeeds,
			AiThreatDetection:       config.AIThreatDetection,
			GoogleSafeBrowsing:      config.GoogleSafeBrowsing,
			Cryptojacking:           config.Cryptojacking,
			DNSRebinding:            config.DNSRebinding,
			IdnHomographs:           config.IDNHomographs,
			Typosquatting:           config.Typosquatting,
			Dga:                     config.DGA,
			Nrd:                     config.NRD,
			DDNS:                    config.DDNS,
			Parking:                 config.Parking,
			Csam:                    config.CSAM,
		},
	}

	err := c.client.Security.Update(ctx, request)
	metrics.RecordAPIRequest("UpdateSecurity", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return fmt.Errorf("failed to update security settings: %w", err)
	}

	return nil
}

// UpdatePrivacy updates privacy settings for a profile
func (c *Client) UpdatePrivacy(ctx context.Context, profileID string, config *PrivacyConfig) error {
	if config == nil {
		return nil
	}

	start := time.Now()
	request := &nextdns.UpdatePrivacyRequest{
		ProfileID: profileID,
		Privacy: &nextdns.Privacy{
			DisguisedTrackers: config.DisguisedTrackers,
			AllowAffiliate:    config.AllowAffiliate,
		},
	}

	err := c.client.Privacy.Update(ctx, request)
	metrics.RecordAPIRequest("UpdatePrivacy", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return fmt.Errorf("failed to update privacy settings: %w", err)
	}

	return nil
}

// SyncRewrites synchronizes DNS rewrites for a profile using diff-based create/delete.
// The NextDNS API does not support update for rewrites, so we delete removed entries
// and create new ones.
func (c *Client) SyncRewrites(ctx context.Context, profileID string, entries []RewriteEntry) error {
	start := time.Now()

	// Get current rewrites
	listRequest := &nextdns.ListRewritesRequest{ProfileID: profileID}
	current, err := c.client.Rewrites.List(ctx, listRequest)
	if err != nil {
		metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to list rewrites: %w", err)
	}

	// Build desired set keyed by name+content
	type rewriteKey struct{ Name, Content string }
	desired := make(map[rewriteKey]bool, len(entries))
	for _, e := range entries {
		desired[rewriteKey(e)] = true
	}

	// Find entries to delete (in current but not in desired)
	currentSet := make(map[rewriteKey]bool, len(current))
	for _, rw := range current {
		key := rewriteKey{rw.Name, rw.Content}
		currentSet[key] = true
		if !desired[key] {
			deleteReq := &nextdns.DeleteRewritesRequest{ProfileID: profileID, ID: rw.ID}
			if err := c.client.Rewrites.Delete(ctx, deleteReq); err != nil {
				metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), false)
				return fmt.Errorf("failed to delete rewrite %s: %w", rw.Name, err)
			}
		}
	}

	// Create entries not in current state
	for _, e := range entries {
		key := rewriteKey(e)
		if !currentSet[key] {
			createReq := &nextdns.CreateRewritesRequest{
				ProfileID: profileID,
				Rewrites:  &nextdns.Rewrites{Name: e.Name, Content: e.Content},
			}
			if _, err := c.client.Rewrites.Create(ctx, createReq); err != nil {
				metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), false)
				return fmt.Errorf("failed to create rewrite %s: %w", e.Name, err)
			}
		}
	}

	metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), true)
	return nil
}

// GetRewrites retrieves the current rewrites for a profile
func (c *Client) GetRewrites(ctx context.Context, profileID string) ([]*nextdns.Rewrites, error) {
	start := time.Now()
	request := &nextdns.ListRewritesRequest{
		ProfileID: profileID,
	}

	list, err := c.client.Rewrites.List(ctx, request)
	metrics.RecordAPIRequest("GetRewrites", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get rewrites: %w", err)
	}

	return list, nil
}

// SyncDenylist synchronizes the denylist for a profile
func (c *Client) SyncDenylist(ctx context.Context, profileID string, entries []DomainEntry) error {
	start := time.Now()

	// Build the desired denylist
	var denylist []*nextdns.Denylist
	for _, entry := range entries {
		denylist = append(denylist, &nextdns.Denylist{
			ID:     entry.Domain,
			Active: entry.Active,
		})
	}

	// PUT replaces the entire list
	createRequest := &nextdns.CreateDenylistRequest{
		ProfileID: profileID,
		Denylist:  denylist,
	}
	if err := c.client.Denylist.Create(ctx, createRequest); err != nil {
		metrics.RecordAPIRequest("SyncDenylist", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to sync denylist: %w", err)
	}

	metrics.RecordAPIRequest("SyncDenylist", time.Since(start).Seconds(), true)
	return nil
}

// SyncAllowlist synchronizes the allowlist for a profile
func (c *Client) SyncAllowlist(ctx context.Context, profileID string, entries []DomainEntry) error {
	start := time.Now()

	// Build the desired allowlist
	var allowlist []*nextdns.Allowlist
	for _, entry := range entries {
		allowlist = append(allowlist, &nextdns.Allowlist{
			ID:     entry.Domain,
			Active: entry.Active,
		})
	}

	// Create/update the allowlist (PUT replaces the entire list)
	createRequest := &nextdns.CreateAllowlistRequest{
		ProfileID: profileID,
		Allowlist: allowlist,
	}
	if err := c.client.Allowlist.Create(ctx, createRequest); err != nil {
		metrics.RecordAPIRequest("SyncAllowlist", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to sync allowlist: %w", err)
	}

	metrics.RecordAPIRequest("SyncAllowlist", time.Since(start).Seconds(), true)
	return nil
}

// AddAllowlistEntry adds a single entry to the allowlist.
func (c *Client) AddAllowlistEntry(ctx context.Context, profileID string, domain string, active bool) error {
	start := time.Now()

	request := &nextdns.AddAllowlistRequest{
		ProfileID: profileID,
		ID:        domain,
		Active:    &active,
	}

	err := c.client.Allowlist.Add(ctx, request)
	metrics.RecordAPIRequest("AddAllowlistEntry", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to add allowlist entry %s: %w", domain, err)
	}

	return nil
}

// DeleteAllowlistEntry removes a single entry from the allowlist.
func (c *Client) DeleteAllowlistEntry(ctx context.Context, profileID string, domain string) error {
	start := time.Now()

	request := &nextdns.DeleteAllowlistRequest{
		ProfileID: profileID,
		ID:        domain,
	}

	err := c.client.Allowlist.Delete(ctx, request)
	metrics.RecordAPIRequest("DeleteAllowlistEntry", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to delete allowlist entry %s: %w", domain, err)
	}

	return nil
}

// AddDenylistEntry adds a single entry to the denylist.
func (c *Client) AddDenylistEntry(ctx context.Context, profileID string, domain string, active bool) error {
	start := time.Now()

	request := &nextdns.AddDenylistRequest{
		ProfileID: profileID,
		ID:        domain,
		Active:    &active,
	}

	err := c.client.Denylist.Add(ctx, request)
	metrics.RecordAPIRequest("AddDenylistEntry", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to add denylist entry %s: %w", domain, err)
	}

	return nil
}

// DeleteDenylistEntry removes a single entry from the denylist.
func (c *Client) DeleteDenylistEntry(ctx context.Context, profileID string, domain string) error {
	start := time.Now()

	request := &nextdns.DeleteDenylistRequest{
		ProfileID: profileID,
		ID:        domain,
	}

	err := c.client.Denylist.Delete(ctx, request)
	metrics.RecordAPIRequest("DeleteDenylistEntry", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to delete denylist entry %s: %w", domain, err)
	}

	return nil
}

// AddSecurityTLD adds a single TLD to the blocked list.
func (c *Client) AddSecurityTLD(ctx context.Context, profileID string, tld string) error {
	start := time.Now()

	request := &nextdns.AddSecurityTldsRequest{
		ProfileID: profileID,
		ID:        tld,
	}

	err := c.client.SecurityTlds.Add(ctx, request)
	metrics.RecordAPIRequest("AddSecurityTLD", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to add security TLD %s: %w", tld, err)
	}

	return nil
}

// DeleteSecurityTLD removes a single TLD from the blocked list.
func (c *Client) DeleteSecurityTLD(ctx context.Context, profileID string, tld string) error {
	start := time.Now()

	request := &nextdns.DeleteSecurityTldsRequest{
		ProfileID: profileID,
		TldID:     tld,
	}

	err := c.client.SecurityTlds.Delete(ctx, request)
	metrics.RecordAPIRequest("DeleteSecurityTLD", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to delete security TLD %s: %w", tld, err)
	}

	return nil
}

// AddPrivacyNative adds a single native tracker protection.
func (c *Client) AddPrivacyNative(ctx context.Context, profileID string, nativeID string) error {
	start := time.Now()

	request := &nextdns.AddPrivacyNativesRequest{
		ProfileID: profileID,
		ID:        nativeID,
	}

	err := c.client.PrivacyNatives.Add(ctx, request)
	metrics.RecordAPIRequest("AddPrivacyNative", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to add privacy native %s: %w", nativeID, err)
	}

	return nil
}

// DeletePrivacyNative removes a single native tracker protection.
func (c *Client) DeletePrivacyNative(ctx context.Context, profileID string, nativeID string) error {
	start := time.Now()

	request := &nextdns.DeletePrivacyNativesRequest{
		ProfileID: profileID,
		NativeID:  nativeID,
	}

	err := c.client.PrivacyNatives.Delete(ctx, request)
	metrics.RecordAPIRequest("DeletePrivacyNative", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to delete privacy native %s: %w", nativeID, err)
	}

	return nil
}

// UpdateSettings updates general settings for a profile
func (c *Client) UpdateSettings(ctx context.Context, profileID string, config *SettingsConfig) error {
	if config == nil {
		return nil
	}

	start := time.Now()

	// Build the full settings object for a single PATCH call.
	// Note: LogClientsIPs and LogDomains use positive logic in the operator spec
	// (true = log them), but the NextDNS API uses inverted logic via the Drop struct
	// (true = don't log them). We invert here at the client boundary.
	settings := &nextdns.Settings{
		Logs: &nextdns.SettingsLogs{
			Enabled:   config.LogsEnabled,
			Retention: config.LogRetention,
			Location:  config.Location,
			Drop: &nextdns.SettingsLogsDrop{
				IP:     !config.LogClientsIPs,
				Domain: !config.LogDomains,
			},
		},
		BlockPage: &nextdns.SettingsBlockPage{
			Enabled: config.BlockPageEnable,
		},
		Performance: &nextdns.SettingsPerformance{
			Ecs:             config.Ecs,
			CacheBoost:      config.CacheBoost,
			CnameFlattening: config.CnameFlattening,
		},
		Web3: config.Web3,
		BAV:  config.BAV,
	}

	request := &nextdns.UpdateSettingsRequest{
		ProfileID: profileID,
		Settings:  settings,
	}

	err := c.client.Settings.Update(ctx, request)
	metrics.RecordAPIRequest("UpdateSettings", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}

// SyncSecurityTLDs synchronizes blocked TLDs for a profile
func (c *Client) SyncSecurityTLDs(ctx context.Context, profileID string, tlds []string) error {
	start := time.Now()

	// Build the desired TLD list
	var securityTlds []*nextdns.SecurityTlds
	for _, tld := range tlds {
		securityTlds = append(securityTlds, &nextdns.SecurityTlds{
			ID: tld,
		})
	}

	// Create/update the TLD list (PUT replaces the entire list)
	createRequest := &nextdns.CreateSecurityTldsRequest{
		ProfileID:    profileID,
		SecurityTlds: securityTlds,
	}
	if err := c.client.SecurityTlds.Create(ctx, createRequest); err != nil {
		metrics.RecordAPIRequest("SyncSecurityTLDs", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to sync security TLDs: %w", err)
	}

	metrics.RecordAPIRequest("SyncSecurityTLDs", time.Since(start).Seconds(), true)
	return nil
}

// UpdateParentalControl updates parental control settings for a profile
func (c *Client) UpdateParentalControl(ctx context.Context, profileID string, config *ParentalControlConfig) error {
	if config == nil {
		return nil
	}

	start := time.Now()
	request := &nextdns.UpdateParentalControlRequest{
		ProfileID: profileID,
		ParentalControl: &nextdns.ParentalControl{
			SafeSearch:            config.SafeSearch,
			YoutubeRestrictedMode: config.YouTubeRestrictedMode,
			BlockBypass:           config.BlockBypass,
		},
	}

	err := c.client.ParentalControl.Update(ctx, request)
	if err != nil {
		metrics.RecordAPIRequest("UpdateParentalControl", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to update parental control settings: %w", err)
	}

	// Sync blocked categories
	if len(config.Categories) > 0 {
		var categories []*nextdns.ParentalControlCategories
		for _, category := range config.Categories {
			categories = append(categories, &nextdns.ParentalControlCategories{
				ID:     category,
				Active: true,
			})
		}
		catRequest := &nextdns.CreateParentalControlCategoriesRequest{
			ProfileID:                 profileID,
			ParentalControlCategories: categories,
		}
		if err := c.client.ParentalControlCategories.Create(ctx, catRequest); err != nil {
			metrics.RecordAPIRequest("UpdateParentalControl", time.Since(start).Seconds(), false)
			return fmt.Errorf("failed to sync parental control categories: %w", err)
		}
	}

	// Sync blocked services
	if len(config.Services) > 0 {
		var services []*nextdns.ParentalControlServices
		for _, service := range config.Services {
			services = append(services, &nextdns.ParentalControlServices{
				ID:     service,
				Active: true,
			})
		}
		svcRequest := &nextdns.CreateParentalControlServicesRequest{
			ProfileID:               profileID,
			ParentalControlServices: services,
		}
		if err := c.client.ParentalControlServices.Create(ctx, svcRequest); err != nil {
			metrics.RecordAPIRequest("UpdateParentalControl", time.Since(start).Seconds(), false)
			return fmt.Errorf("failed to sync parental control services: %w", err)
		}
	}

	metrics.RecordAPIRequest("UpdateParentalControl", time.Since(start).Seconds(), true)
	return nil
}

// SyncPrivacyBlocklists synchronizes privacy blocklists for a profile
func (c *Client) SyncPrivacyBlocklists(ctx context.Context, profileID string, blocklists []string) error {
	start := time.Now()
	var privacyBlocklists []*nextdns.PrivacyBlocklists
	for _, blocklist := range blocklists {
		privacyBlocklists = append(privacyBlocklists, &nextdns.PrivacyBlocklists{
			ID: blocklist,
		})
	}

	request := &nextdns.CreatePrivacyBlocklistsRequest{
		ProfileID:         profileID,
		PrivacyBlocklists: privacyBlocklists,
	}
	if err := c.client.PrivacyBlocklists.Create(ctx, request); err != nil {
		metrics.RecordAPIRequest("SyncPrivacyBlocklists", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to sync privacy blocklists: %w", err)
	}

	metrics.RecordAPIRequest("SyncPrivacyBlocklists", time.Since(start).Seconds(), true)
	return nil
}

// SyncPrivacyNatives synchronizes native tracker blocking for a profile
func (c *Client) SyncPrivacyNatives(ctx context.Context, profileID string, natives []string) error {
	start := time.Now()
	var privacyNatives []*nextdns.PrivacyNatives
	for _, native := range natives {
		privacyNatives = append(privacyNatives, &nextdns.PrivacyNatives{
			ID: native,
		})
	}

	request := &nextdns.CreatePrivacyNativesRequest{
		ProfileID:      profileID,
		PrivacyNatives: privacyNatives,
	}
	if err := c.client.PrivacyNatives.Create(ctx, request); err != nil {
		metrics.RecordAPIRequest("SyncPrivacyNatives", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to sync privacy natives: %w", err)
	}

	metrics.RecordAPIRequest("SyncPrivacyNatives", time.Since(start).Seconds(), true)
	return nil
}

// GetDenylist retrieves the current denylist for a profile
func (c *Client) GetDenylist(ctx context.Context, profileID string) ([]*nextdns.Denylist, error) {
	start := time.Now()
	request := &nextdns.ListDenylistRequest{
		ProfileID: profileID,
	}

	list, err := c.client.Denylist.List(ctx, request)
	metrics.RecordAPIRequest("GetDenylist", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get denylist: %w", err)
	}

	return list, nil
}

// GetAllowlist retrieves the current allowlist for a profile
func (c *Client) GetAllowlist(ctx context.Context, profileID string) ([]*nextdns.Allowlist, error) {
	start := time.Now()
	request := &nextdns.ListAllowlistRequest{
		ProfileID: profileID,
	}

	list, err := c.client.Allowlist.List(ctx, request)
	metrics.RecordAPIRequest("GetAllowlist", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get allowlist: %w", err)
	}

	return list, nil
}

// GetSecurityTLDs retrieves the current blocked TLDs for a profile
func (c *Client) GetSecurityTLDs(ctx context.Context, profileID string) ([]*nextdns.SecurityTlds, error) {
	start := time.Now()
	request := &nextdns.ListSecurityTldsRequest{
		ProfileID: profileID,
	}

	list, err := c.client.SecurityTlds.List(ctx, request)
	metrics.RecordAPIRequest("GetSecurityTLDs", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get security TLDs: %w", err)
	}

	return list, nil
}

// GetSecurity retrieves the current security settings for a profile
func (c *Client) GetSecurity(ctx context.Context, profileID string) (*nextdns.Security, error) {
	start := time.Now()
	request := &nextdns.GetSecurityRequest{
		ProfileID: profileID,
	}

	security, err := c.client.Security.Get(ctx, request)
	metrics.RecordAPIRequest("GetSecurity", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get security settings: %w", err)
	}

	return security, nil
}

// GetPrivacy retrieves the current privacy settings for a profile
func (c *Client) GetPrivacy(ctx context.Context, profileID string) (*nextdns.Privacy, error) {
	start := time.Now()
	request := &nextdns.GetPrivacyRequest{
		ProfileID: profileID,
	}

	privacy, err := c.client.Privacy.Get(ctx, request)
	metrics.RecordAPIRequest("GetPrivacy", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get privacy settings: %w", err)
	}

	return privacy, nil
}

// GetParentalControl retrieves the current parental control settings for a profile
func (c *Client) GetParentalControl(ctx context.Context, profileID string) (*nextdns.ParentalControl, error) {
	start := time.Now()
	request := &nextdns.GetParentalControlRequest{
		ProfileID: profileID,
	}

	pc, err := c.client.ParentalControl.Get(ctx, request)
	metrics.RecordAPIRequest("GetParentalControl", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get parental control settings: %w", err)
	}

	return pc, nil
}

// GetSetup retrieves the current setup/endpoint data for a profile
func (c *Client) GetSetup(ctx context.Context, profileID string) (*nextdns.Setup, error) {
	start := time.Now()
	request := &nextdns.GetSetupRequest{
		ProfileID: profileID,
	}

	setup, err := c.client.Setup.Get(ctx, request)
	metrics.RecordAPIRequest("GetSetup", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get setup: %w", err)
	}

	return setup, nil
}

// GetSettings retrieves the current settings for a profile
func (c *Client) GetSettings(ctx context.Context, profileID string) (*nextdns.Settings, error) {
	start := time.Now()
	request := &nextdns.GetSettingsRequest{
		ProfileID: profileID,
	}

	settings, err := c.client.Settings.Get(ctx, request)
	metrics.RecordAPIRequest("GetSettings", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

// GetPrivacyBlocklists retrieves the current privacy blocklists for a profile
func (c *Client) GetPrivacyBlocklists(ctx context.Context, profileID string) ([]*nextdns.PrivacyBlocklists, error) {
	start := time.Now()
	request := &nextdns.ListPrivacyBlocklistsRequest{
		ProfileID: profileID,
	}

	list, err := c.client.PrivacyBlocklists.List(ctx, request)
	metrics.RecordAPIRequest("GetPrivacyBlocklists", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get privacy blocklists: %w", err)
	}

	return list, nil
}

// GetPrivacyNatives retrieves the current privacy natives for a profile
func (c *Client) GetPrivacyNatives(ctx context.Context, profileID string) ([]*nextdns.PrivacyNatives, error) {
	start := time.Now()
	request := &nextdns.ListPrivacyNativesRequest{
		ProfileID: profileID,
	}

	list, err := c.client.PrivacyNatives.List(ctx, request)
	metrics.RecordAPIRequest("GetPrivacyNatives", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get privacy natives: %w", err)
	}

	return list, nil
}

// GetParentalControlCategories retrieves the current parental control categories for a profile
func (c *Client) GetParentalControlCategories(ctx context.Context, profileID string) ([]*nextdns.ParentalControlCategories, error) {
	start := time.Now()
	request := &nextdns.ListParentalControlCategoriesRequest{
		ProfileID: profileID,
	}

	list, err := c.client.ParentalControlCategories.List(ctx, request)
	metrics.RecordAPIRequest("GetParentalControlCategories", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get parental control categories: %w", err)
	}

	return list, nil
}

// GetParentalControlServices retrieves the current parental control services for a profile
func (c *Client) GetParentalControlServices(ctx context.Context, profileID string) ([]*nextdns.ParentalControlServices, error) {
	start := time.Now()
	request := &nextdns.ListParentalControlServicesRequest{
		ProfileID: profileID,
	}

	list, err := c.client.ParentalControlServices.List(ctx, request)
	metrics.RecordAPIRequest("GetParentalControlServices", time.Since(start).Seconds(), err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get parental control services: %w", err)
	}

	return list, nil
}
