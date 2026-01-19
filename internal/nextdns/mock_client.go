package nextdns

import (
	"context"
	"fmt"
	"sync"

	"github.com/amalucelli/nextdns-go/nextdns"
)

// MockClient is a mock implementation of ClientInterface for testing
type MockClient struct {
	mu sync.RWMutex

	// Profiles stores mock profiles
	Profiles map[string]*nextdns.Profile

	// Security stores mock security settings per profile
	Security map[string]*nextdns.Security

	// Privacy stores mock privacy settings per profile
	Privacy map[string]*nextdns.Privacy

	// ParentalControl stores mock parental control settings per profile
	ParentalControl map[string]*nextdns.ParentalControl

	// Denylists stores mock denylists per profile
	Denylists map[string][]*nextdns.Denylist

	// Allowlists stores mock allowlists per profile
	Allowlists map[string][]*nextdns.Allowlist

	// SecurityTLDs stores mock security TLDs per profile
	SecurityTLDs map[string][]*nextdns.SecurityTlds

	// PrivacyBlocklists stores mock privacy blocklists per profile
	PrivacyBlocklists map[string][]*nextdns.PrivacyBlocklists

	// PrivacyNatives stores mock privacy natives per profile
	PrivacyNatives map[string][]*nextdns.PrivacyNatives

	// SettingsLogs stores mock settings logs per profile
	SettingsLogs map[string]*nextdns.SettingsLogs

	// SettingsBlockPage stores mock block page settings per profile
	SettingsBlockPage map[string]*nextdns.SettingsBlockPage

	// ParentalControlCategories stores categories per profile
	ParentalControlCategories map[string][]*nextdns.ParentalControlCategories

	// ParentalControlServices stores services per profile
	ParentalControlServices map[string][]*nextdns.ParentalControlServices

	// Error injection for testing error paths
	CreateProfileError           error
	GetProfileError              error
	UpdateProfileError           error
	DeleteProfileError           error
	UpdateSecurityError          error
	GetSecurityError             error
	UpdatePrivacyError           error
	GetPrivacyError              error
	SyncPrivacyBlocklistsError   error
	SyncPrivacyNativesError      error
	UpdateParentalControlError   error
	GetParentalControlError      error
	SyncDenylistError            error
	SyncAllowlistError           error
	SyncSecurityTLDsError        error
	GetDenylistError             error
	GetAllowlistError            error
	GetSecurityTLDsError         error
	UpdateSettingsError          error

	// Call tracking
	Calls []MockCall

	// NextProfileID for auto-generating profile IDs
	NextProfileID int
}

// MockCall records a method call for verification
type MockCall struct {
	Method string
	Args   []interface{}
}

// NewMockClient creates a new mock client with initialized maps
func NewMockClient() *MockClient {
	return &MockClient{
		Profiles:                  make(map[string]*nextdns.Profile),
		Security:                  make(map[string]*nextdns.Security),
		Privacy:                   make(map[string]*nextdns.Privacy),
		ParentalControl:           make(map[string]*nextdns.ParentalControl),
		Denylists:                 make(map[string][]*nextdns.Denylist),
		Allowlists:                make(map[string][]*nextdns.Allowlist),
		SecurityTLDs:              make(map[string][]*nextdns.SecurityTlds),
		PrivacyBlocklists:         make(map[string][]*nextdns.PrivacyBlocklists),
		PrivacyNatives:            make(map[string][]*nextdns.PrivacyNatives),
		SettingsLogs:              make(map[string]*nextdns.SettingsLogs),
		SettingsBlockPage:         make(map[string]*nextdns.SettingsBlockPage),
		ParentalControlCategories: make(map[string][]*nextdns.ParentalControlCategories),
		ParentalControlServices:   make(map[string][]*nextdns.ParentalControlServices),
		Calls:                     make([]MockCall, 0),
		NextProfileID:             1,
	}
}

func (m *MockClient) recordCall(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, MockCall{Method: method, Args: args})
}

// CreateProfile creates a mock profile
func (m *MockClient) CreateProfile(ctx context.Context, name string) (string, error) {
	m.recordCall("CreateProfile", name)
	if m.CreateProfileError != nil {
		return "", m.CreateProfileError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	profileID := fmt.Sprintf("mock-%d", m.NextProfileID)
	m.NextProfileID++

	m.Profiles[profileID] = &nextdns.Profile{
		Name: name,
	}

	return profileID, nil
}

// GetProfile retrieves a mock profile
func (m *MockClient) GetProfile(ctx context.Context, profileID string) (*nextdns.Profile, error) {
	m.recordCall("GetProfile", profileID)
	if m.GetProfileError != nil {
		return nil, m.GetProfileError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	profile, exists := m.Profiles[profileID]
	if !exists {
		return nil, fmt.Errorf("profile not found: %s", profileID)
	}

	return profile, nil
}

// UpdateProfile updates a mock profile
func (m *MockClient) UpdateProfile(ctx context.Context, profileID, name string) error {
	m.recordCall("UpdateProfile", profileID, name)
	if m.UpdateProfileError != nil {
		return m.UpdateProfileError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	profile, exists := m.Profiles[profileID]
	if !exists {
		m.Profiles[profileID] = &nextdns.Profile{Name: name}
	} else {
		profile.Name = name
	}

	return nil
}

// DeleteProfile deletes a mock profile
func (m *MockClient) DeleteProfile(ctx context.Context, profileID string) error {
	m.recordCall("DeleteProfile", profileID)
	if m.DeleteProfileError != nil {
		return m.DeleteProfileError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.Profiles, profileID)
	return nil
}

// UpdateSecurity updates mock security settings
func (m *MockClient) UpdateSecurity(ctx context.Context, profileID string, config *SecurityConfig) error {
	m.recordCall("UpdateSecurity", profileID, config)
	if m.UpdateSecurityError != nil {
		return m.UpdateSecurityError
	}
	if config == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Security[profileID] = &nextdns.Security{
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
	}

	return nil
}

// GetSecurity retrieves mock security settings
func (m *MockClient) GetSecurity(ctx context.Context, profileID string) (*nextdns.Security, error) {
	m.recordCall("GetSecurity", profileID)
	if m.GetSecurityError != nil {
		return nil, m.GetSecurityError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	security, exists := m.Security[profileID]
	if !exists {
		return &nextdns.Security{}, nil
	}

	return security, nil
}

// UpdatePrivacy updates mock privacy settings
func (m *MockClient) UpdatePrivacy(ctx context.Context, profileID string, config *PrivacyConfig) error {
	m.recordCall("UpdatePrivacy", profileID, config)
	if m.UpdatePrivacyError != nil {
		return m.UpdatePrivacyError
	}
	if config == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Privacy[profileID] = &nextdns.Privacy{
		DisguisedTrackers: config.DisguisedTrackers,
		AllowAffiliate:    config.AllowAffiliate,
	}

	return nil
}

// GetPrivacy retrieves mock privacy settings
func (m *MockClient) GetPrivacy(ctx context.Context, profileID string) (*nextdns.Privacy, error) {
	m.recordCall("GetPrivacy", profileID)
	if m.GetPrivacyError != nil {
		return nil, m.GetPrivacyError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	privacy, exists := m.Privacy[profileID]
	if !exists {
		return &nextdns.Privacy{}, nil
	}

	return privacy, nil
}

// SyncPrivacyBlocklists syncs mock privacy blocklists
func (m *MockClient) SyncPrivacyBlocklists(ctx context.Context, profileID string, blocklists []string) error {
	m.recordCall("SyncPrivacyBlocklists", profileID, blocklists)
	if m.SyncPrivacyBlocklistsError != nil {
		return m.SyncPrivacyBlocklistsError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var pbls []*nextdns.PrivacyBlocklists
	for _, bl := range blocklists {
		pbls = append(pbls, &nextdns.PrivacyBlocklists{ID: bl})
	}
	m.PrivacyBlocklists[profileID] = pbls

	return nil
}

// SyncPrivacyNatives syncs mock privacy natives
func (m *MockClient) SyncPrivacyNatives(ctx context.Context, profileID string, natives []string) error {
	m.recordCall("SyncPrivacyNatives", profileID, natives)
	if m.SyncPrivacyNativesError != nil {
		return m.SyncPrivacyNativesError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var pns []*nextdns.PrivacyNatives
	for _, n := range natives {
		pns = append(pns, &nextdns.PrivacyNatives{ID: n})
	}
	m.PrivacyNatives[profileID] = pns

	return nil
}

// UpdateParentalControl updates mock parental control settings
func (m *MockClient) UpdateParentalControl(ctx context.Context, profileID string, config *ParentalControlConfig) error {
	m.recordCall("UpdateParentalControl", profileID, config)
	if m.UpdateParentalControlError != nil {
		return m.UpdateParentalControlError
	}
	if config == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.ParentalControl[profileID] = &nextdns.ParentalControl{
		SafeSearch:            config.SafeSearch,
		YoutubeRestrictedMode: config.YouTubeRestrictedMode,
	}

	// Store categories
	var cats []*nextdns.ParentalControlCategories
	for _, c := range config.Categories {
		cats = append(cats, &nextdns.ParentalControlCategories{ID: c, Active: true})
	}
	m.ParentalControlCategories[profileID] = cats

	// Store services
	var svcs []*nextdns.ParentalControlServices
	for _, s := range config.Services {
		svcs = append(svcs, &nextdns.ParentalControlServices{ID: s, Active: true})
	}
	m.ParentalControlServices[profileID] = svcs

	return nil
}

// GetParentalControl retrieves mock parental control settings
func (m *MockClient) GetParentalControl(ctx context.Context, profileID string) (*nextdns.ParentalControl, error) {
	m.recordCall("GetParentalControl", profileID)
	if m.GetParentalControlError != nil {
		return nil, m.GetParentalControlError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	pc, exists := m.ParentalControl[profileID]
	if !exists {
		return &nextdns.ParentalControl{}, nil
	}

	return pc, nil
}

// SyncDenylist syncs mock denylist
func (m *MockClient) SyncDenylist(ctx context.Context, profileID string, domains []string) error {
	m.recordCall("SyncDenylist", profileID, domains)
	if m.SyncDenylistError != nil {
		return m.SyncDenylistError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var denylist []*nextdns.Denylist
	for _, d := range domains {
		denylist = append(denylist, &nextdns.Denylist{ID: d, Active: true})
	}
	m.Denylists[profileID] = denylist

	return nil
}

// SyncAllowlist syncs mock allowlist
func (m *MockClient) SyncAllowlist(ctx context.Context, profileID string, domains []string) error {
	m.recordCall("SyncAllowlist", profileID, domains)
	if m.SyncAllowlistError != nil {
		return m.SyncAllowlistError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var allowlist []*nextdns.Allowlist
	for _, d := range domains {
		allowlist = append(allowlist, &nextdns.Allowlist{ID: d, Active: true})
	}
	m.Allowlists[profileID] = allowlist

	return nil
}

// SyncSecurityTLDs syncs mock security TLDs
func (m *MockClient) SyncSecurityTLDs(ctx context.Context, profileID string, tlds []string) error {
	m.recordCall("SyncSecurityTLDs", profileID, tlds)
	if m.SyncSecurityTLDsError != nil {
		return m.SyncSecurityTLDsError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var securityTLDs []*nextdns.SecurityTlds
	for _, t := range tlds {
		securityTLDs = append(securityTLDs, &nextdns.SecurityTlds{ID: t})
	}
	m.SecurityTLDs[profileID] = securityTLDs

	return nil
}

// GetDenylist retrieves mock denylist
func (m *MockClient) GetDenylist(ctx context.Context, profileID string) ([]*nextdns.Denylist, error) {
	m.recordCall("GetDenylist", profileID)
	if m.GetDenylistError != nil {
		return nil, m.GetDenylistError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.Denylists[profileID], nil
}

// GetAllowlist retrieves mock allowlist
func (m *MockClient) GetAllowlist(ctx context.Context, profileID string) ([]*nextdns.Allowlist, error) {
	m.recordCall("GetAllowlist", profileID)
	if m.GetAllowlistError != nil {
		return nil, m.GetAllowlistError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.Allowlists[profileID], nil
}

// GetSecurityTLDs retrieves mock security TLDs
func (m *MockClient) GetSecurityTLDs(ctx context.Context, profileID string) ([]*nextdns.SecurityTlds, error) {
	m.recordCall("GetSecurityTLDs", profileID)
	if m.GetSecurityTLDsError != nil {
		return nil, m.GetSecurityTLDsError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.SecurityTLDs[profileID], nil
}

// UpdateSettings updates mock settings
func (m *MockClient) UpdateSettings(ctx context.Context, profileID string, config *SettingsConfig) error {
	m.recordCall("UpdateSettings", profileID, config)
	if m.UpdateSettingsError != nil {
		return m.UpdateSettingsError
	}
	if config == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.SettingsLogs[profileID] = &nextdns.SettingsLogs{
		Enabled:   config.LogsEnabled,
		Retention: config.LogRetention,
	}
	m.SettingsBlockPage[profileID] = &nextdns.SettingsBlockPage{
		Enabled: config.BlockPageEnable,
	}

	return nil
}

// GetCallCount returns the number of calls to a specific method
func (m *MockClient) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, call := range m.Calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// WasMethodCalled checks if a method was called
func (m *MockClient) WasMethodCalled(method string) bool {
	return m.GetCallCount(method) > 0
}

// Reset clears all mock data and calls
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Profiles = make(map[string]*nextdns.Profile)
	m.Security = make(map[string]*nextdns.Security)
	m.Privacy = make(map[string]*nextdns.Privacy)
	m.ParentalControl = make(map[string]*nextdns.ParentalControl)
	m.Denylists = make(map[string][]*nextdns.Denylist)
	m.Allowlists = make(map[string][]*nextdns.Allowlist)
	m.SecurityTLDs = make(map[string][]*nextdns.SecurityTlds)
	m.PrivacyBlocklists = make(map[string][]*nextdns.PrivacyBlocklists)
	m.PrivacyNatives = make(map[string][]*nextdns.PrivacyNatives)
	m.SettingsLogs = make(map[string]*nextdns.SettingsLogs)
	m.SettingsBlockPage = make(map[string]*nextdns.SettingsBlockPage)
	m.ParentalControlCategories = make(map[string][]*nextdns.ParentalControlCategories)
	m.ParentalControlServices = make(map[string][]*nextdns.ParentalControlServices)
	m.Calls = make([]MockCall, 0)
	m.NextProfileID = 1

	// Reset errors
	m.CreateProfileError = nil
	m.GetProfileError = nil
	m.UpdateProfileError = nil
	m.DeleteProfileError = nil
	m.UpdateSecurityError = nil
	m.GetSecurityError = nil
	m.UpdatePrivacyError = nil
	m.GetPrivacyError = nil
	m.SyncPrivacyBlocklistsError = nil
	m.SyncPrivacyNativesError = nil
	m.UpdateParentalControlError = nil
	m.GetParentalControlError = nil
	m.SyncDenylistError = nil
	m.SyncAllowlistError = nil
	m.SyncSecurityTLDsError = nil
	m.GetDenylistError = nil
	m.GetAllowlistError = nil
	m.GetSecurityTLDsError = nil
	m.UpdateSettingsError = nil
}

// Ensure MockClient implements ClientInterface
var _ ClientInterface = (*MockClient)(nil)
