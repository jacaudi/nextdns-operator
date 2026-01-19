package nextdns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid API key",
			apiKey:  "test-api-key-12345",
			wantErr: false,
		},
		{
			name:    "empty API key returns error",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.client)
			}
		})
	}
}

func TestSecurityConfig(t *testing.T) {
	config := &SecurityConfig{
		ThreatIntelligenceFeeds: true,
		AIThreatDetection:       true,
		GoogleSafeBrowsing:      true,
		Cryptojacking:           true,
		DNSRebinding:            true,
		IDNHomographs:           true,
		Typosquatting:           true,
		DGA:                     true,
		NRD:                     false,
		DDNS:                    false,
		Parking:                 true,
		CSAM:                    true,
	}

	assert.True(t, config.AIThreatDetection)
	assert.True(t, config.GoogleSafeBrowsing)
	assert.True(t, config.Cryptojacking)
	assert.True(t, config.DNSRebinding)
	assert.True(t, config.IDNHomographs)
	assert.True(t, config.Typosquatting)
	assert.True(t, config.DGA)
	assert.False(t, config.NRD)
	assert.False(t, config.DDNS)
	assert.True(t, config.Parking)
	assert.True(t, config.CSAM)
}

func TestPrivacyConfig(t *testing.T) {
	config := &PrivacyConfig{
		Blocklists:        []string{"blocklist1", "blocklist2"},
		Natives:           []string{"apple", "windows"},
		DisguisedTrackers: true,
		AllowAffiliate:    false,
	}

	assert.Equal(t, 2, len(config.Blocklists))
	assert.Equal(t, 2, len(config.Natives))
	assert.True(t, config.DisguisedTrackers)
	assert.False(t, config.AllowAffiliate)
}

func TestParentalControlConfig(t *testing.T) {
	config := &ParentalControlConfig{
		Categories:            []string{"adult", "gambling"},
		Services:              []string{"tiktok", "instagram"},
		SafeSearch:            true,
		YouTubeRestrictedMode: true,
	}

	assert.Equal(t, 2, len(config.Categories))
	assert.Equal(t, 2, len(config.Services))
	assert.True(t, config.SafeSearch)
	assert.True(t, config.YouTubeRestrictedMode)
}

func TestSettingsConfig(t *testing.T) {
	config := &SettingsConfig{
		LogsEnabled:     true,
		LogClientsIPs:   false,
		LogDomains:      true,
		LogRetention:    30,
		BlockPageEnable: true,
		Web3:            false,
	}

	assert.True(t, config.LogsEnabled)
	assert.False(t, config.LogClientsIPs)
	assert.True(t, config.LogDomains)
	assert.Equal(t, 30, config.LogRetention)
	assert.True(t, config.BlockPageEnable)
	assert.False(t, config.Web3)
}

func TestProfileConfig(t *testing.T) {
	config := &ProfileConfig{
		Name: "Test Profile",
		Security: &SecurityConfig{
			AIThreatDetection: true,
		},
		Privacy: &PrivacyConfig{
			DisguisedTrackers: true,
		},
		ParentalControl: &ParentalControlConfig{
			SafeSearch: true,
		},
		Denylist:    []string{"bad.com"},
		Allowlist:   []string{"good.com"},
		BlockedTLDs: []string{"xyz"},
		Settings: &SettingsConfig{
			LogsEnabled: true,
		},
	}

	assert.Equal(t, "Test Profile", config.Name)
	assert.NotNil(t, config.Security)
	assert.NotNil(t, config.Privacy)
	assert.NotNil(t, config.ParentalControl)
	assert.Equal(t, 1, len(config.Denylist))
	assert.Equal(t, 1, len(config.Allowlist))
	assert.Equal(t, 1, len(config.BlockedTLDs))
	assert.NotNil(t, config.Settings)
}

func TestMockClient_CreateProfile(t *testing.T) {
	mock := NewMockClient()

	profileID, err := mock.CreateProfile(nil, "Test Profile")
	require.NoError(t, err)
	assert.Equal(t, "mock-1", profileID)
	assert.True(t, mock.WasMethodCalled("CreateProfile"))

	// Create another profile
	profileID2, err := mock.CreateProfile(nil, "Test Profile 2")
	require.NoError(t, err)
	assert.Equal(t, "mock-2", profileID2)
	assert.Equal(t, 2, mock.GetCallCount("CreateProfile"))
}

func TestMockClient_GetProfile(t *testing.T) {
	mock := NewMockClient()

	// First create a profile
	profileID, err := mock.CreateProfile(nil, "Test Profile")
	require.NoError(t, err)

	// Then get it
	profile, err := mock.GetProfile(nil, profileID)
	require.NoError(t, err)
	assert.Equal(t, "Test Profile", profile.Name)

	// Get non-existent profile
	_, err = mock.GetProfile(nil, "non-existent")
	assert.Error(t, err)
}

func TestMockClient_UpdateProfile(t *testing.T) {
	mock := NewMockClient()

	// Create and update a profile
	profileID, err := mock.CreateProfile(nil, "Original Name")
	require.NoError(t, err)

	err = mock.UpdateProfile(nil, profileID, "Updated Name")
	require.NoError(t, err)

	profile, err := mock.GetProfile(nil, profileID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", profile.Name)
}

func TestMockClient_DeleteProfile(t *testing.T) {
	mock := NewMockClient()

	// Create a profile
	profileID, err := mock.CreateProfile(nil, "To Delete")
	require.NoError(t, err)

	// Delete it
	err = mock.DeleteProfile(nil, profileID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = mock.GetProfile(nil, profileID)
	assert.Error(t, err)
}

func TestMockClient_UpdateSecurity(t *testing.T) {
	mock := NewMockClient()

	config := &SecurityConfig{
		AIThreatDetection:  true,
		GoogleSafeBrowsing: true,
		Cryptojacking:      false,
	}

	err := mock.UpdateSecurity(nil, "profile-1", config)
	require.NoError(t, err)

	security, err := mock.GetSecurity(nil, "profile-1")
	require.NoError(t, err)
	assert.True(t, security.AiThreatDetection)
	assert.True(t, security.GoogleSafeBrowsing)
	assert.False(t, security.Cryptojacking)
}

func TestMockClient_UpdateSecurity_NilConfig(t *testing.T) {
	mock := NewMockClient()

	err := mock.UpdateSecurity(nil, "profile-1", nil)
	require.NoError(t, err)
}

func TestMockClient_UpdatePrivacy(t *testing.T) {
	mock := NewMockClient()

	config := &PrivacyConfig{
		DisguisedTrackers: true,
		AllowAffiliate:    false,
	}

	err := mock.UpdatePrivacy(nil, "profile-1", config)
	require.NoError(t, err)

	privacy, err := mock.GetPrivacy(nil, "profile-1")
	require.NoError(t, err)
	assert.True(t, privacy.DisguisedTrackers)
	assert.False(t, privacy.AllowAffiliate)
}

func TestMockClient_SyncDenylist(t *testing.T) {
	mock := NewMockClient()

	domains := []string{"bad1.com", "bad2.com", "bad3.com"}
	err := mock.SyncDenylist(nil, "profile-1", domains)
	require.NoError(t, err)

	denylist, err := mock.GetDenylist(nil, "profile-1")
	require.NoError(t, err)
	assert.Equal(t, 3, len(denylist))
}

func TestMockClient_SyncAllowlist(t *testing.T) {
	mock := NewMockClient()

	domains := []string{"good1.com", "good2.com"}
	err := mock.SyncAllowlist(nil, "profile-1", domains)
	require.NoError(t, err)

	allowlist, err := mock.GetAllowlist(nil, "profile-1")
	require.NoError(t, err)
	assert.Equal(t, 2, len(allowlist))
}

func TestMockClient_SyncSecurityTLDs(t *testing.T) {
	mock := NewMockClient()

	tlds := []string{"xyz", "tk", "ml"}
	err := mock.SyncSecurityTLDs(nil, "profile-1", tlds)
	require.NoError(t, err)

	securityTLDs, err := mock.GetSecurityTLDs(nil, "profile-1")
	require.NoError(t, err)
	assert.Equal(t, 3, len(securityTLDs))
}

func TestMockClient_UpdateParentalControl(t *testing.T) {
	mock := NewMockClient()

	config := &ParentalControlConfig{
		Categories:            []string{"adult", "gambling"},
		Services:              []string{"tiktok"},
		SafeSearch:            true,
		YouTubeRestrictedMode: false,
	}

	err := mock.UpdateParentalControl(nil, "profile-1", config)
	require.NoError(t, err)

	pc, err := mock.GetParentalControl(nil, "profile-1")
	require.NoError(t, err)
	assert.True(t, pc.SafeSearch)
	assert.False(t, pc.YoutubeRestrictedMode)

	// Check categories were stored
	assert.Equal(t, 2, len(mock.ParentalControlCategories["profile-1"]))
	assert.Equal(t, 1, len(mock.ParentalControlServices["profile-1"]))
}

func TestMockClient_SyncPrivacyBlocklists(t *testing.T) {
	mock := NewMockClient()

	blocklists := []string{"nextdns-recommended", "oisd"}
	err := mock.SyncPrivacyBlocklists(nil, "profile-1", blocklists)
	require.NoError(t, err)

	assert.Equal(t, 2, len(mock.PrivacyBlocklists["profile-1"]))
}

func TestMockClient_SyncPrivacyNatives(t *testing.T) {
	mock := NewMockClient()

	natives := []string{"apple", "windows", "samsung"}
	err := mock.SyncPrivacyNatives(nil, "profile-1", natives)
	require.NoError(t, err)

	assert.Equal(t, 3, len(mock.PrivacyNatives["profile-1"]))
}

func TestMockClient_UpdateSettings(t *testing.T) {
	mock := NewMockClient()

	config := &SettingsConfig{
		LogsEnabled:     true,
		LogRetention:    30,
		BlockPageEnable: true,
	}

	err := mock.UpdateSettings(nil, "profile-1", config)
	require.NoError(t, err)

	assert.True(t, mock.SettingsLogs["profile-1"].Enabled)
	assert.Equal(t, 30, mock.SettingsLogs["profile-1"].Retention)
	assert.True(t, mock.SettingsBlockPage["profile-1"].Enabled)
}

func TestMockClient_ErrorInjection(t *testing.T) {
	mock := NewMockClient()

	// Test error injection for CreateProfile
	mock.CreateProfileError = assert.AnError
	_, err := mock.CreateProfile(nil, "Test")
	assert.Error(t, err)

	// Test error injection for GetProfile
	mock.GetProfileError = assert.AnError
	_, err = mock.GetProfile(nil, "profile-1")
	assert.Error(t, err)

	// Test error injection for UpdateSecurity
	mock.UpdateSecurityError = assert.AnError
	err = mock.UpdateSecurity(nil, "profile-1", &SecurityConfig{})
	assert.Error(t, err)

	// Test error injection for SyncDenylist
	mock.SyncDenylistError = assert.AnError
	err = mock.SyncDenylist(nil, "profile-1", []string{"bad.com"})
	assert.Error(t, err)
}

func TestMockClient_Reset(t *testing.T) {
	mock := NewMockClient()

	// Create some data
	mock.CreateProfile(nil, "Test")
	mock.SyncDenylist(nil, "profile-1", []string{"bad.com"})
	mock.CreateProfileError = assert.AnError

	// Reset
	mock.Reset()

	// Verify everything is cleared
	assert.Equal(t, 0, len(mock.Profiles))
	assert.Equal(t, 0, len(mock.Denylists))
	assert.Equal(t, 0, len(mock.Calls))
	assert.Nil(t, mock.CreateProfileError)
	assert.Equal(t, 1, mock.NextProfileID)
}

func TestMockClient_CallTracking(t *testing.T) {
	mock := NewMockClient()

	mock.CreateProfile(nil, "Test1")
	mock.CreateProfile(nil, "Test2")
	mock.GetProfile(nil, "mock-1")
	mock.UpdateProfile(nil, "mock-1", "Updated")

	assert.Equal(t, 2, mock.GetCallCount("CreateProfile"))
	assert.Equal(t, 1, mock.GetCallCount("GetProfile"))
	assert.Equal(t, 1, mock.GetCallCount("UpdateProfile"))
	assert.Equal(t, 0, mock.GetCallCount("DeleteProfile"))

	assert.True(t, mock.WasMethodCalled("CreateProfile"))
	assert.True(t, mock.WasMethodCalled("GetProfile"))
	assert.False(t, mock.WasMethodCalled("DeleteProfile"))
}

func TestMockClient_ThreadSafety(t *testing.T) {
	mock := NewMockClient()

	// Run concurrent operations
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			mock.CreateProfile(nil, "Test")
			mock.SyncDenylist(nil, "profile-1", []string{"bad.com"})
			mock.GetCallCount("CreateProfile")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify it didn't panic
	assert.True(t, mock.GetCallCount("CreateProfile") >= 10)
}

func TestMockClient_GetSecurity(t *testing.T) {
	mock := NewMockClient()

	// Test getting security when none exists
	security, err := mock.GetSecurity(nil, "profile-1")
	require.NoError(t, err)
	assert.NotNil(t, security)

	// Store security config first
	config := &SecurityConfig{
		AIThreatDetection:  true,
		GoogleSafeBrowsing: true,
	}
	err = mock.UpdateSecurity(nil, "profile-1", config)
	require.NoError(t, err)

	// Now get it
	security, err = mock.GetSecurity(nil, "profile-1")
	require.NoError(t, err)
	assert.True(t, security.AiThreatDetection)
	assert.True(t, security.GoogleSafeBrowsing)

	// Test error injection
	mock.GetSecurityError = assert.AnError
	_, err = mock.GetSecurity(nil, "profile-1")
	assert.Error(t, err)
}

func TestMockClient_GetPrivacy(t *testing.T) {
	mock := NewMockClient()

	// Test getting privacy when none exists
	privacy, err := mock.GetPrivacy(nil, "profile-1")
	require.NoError(t, err)
	assert.NotNil(t, privacy)

	// Store privacy config first
	config := &PrivacyConfig{
		DisguisedTrackers: true,
		AllowAffiliate:    false,
	}
	err = mock.UpdatePrivacy(nil, "profile-1", config)
	require.NoError(t, err)

	// Now get it
	privacy, err = mock.GetPrivacy(nil, "profile-1")
	require.NoError(t, err)
	assert.True(t, privacy.DisguisedTrackers)
	assert.False(t, privacy.AllowAffiliate)

	// Test error injection
	mock.GetPrivacyError = assert.AnError
	_, err = mock.GetPrivacy(nil, "profile-1")
	assert.Error(t, err)
}

func TestMockClient_GetParentalControl(t *testing.T) {
	mock := NewMockClient()

	// Test getting parental control when none exists
	pc, err := mock.GetParentalControl(nil, "profile-1")
	require.NoError(t, err)
	assert.NotNil(t, pc)

	// Store parental control config first
	config := &ParentalControlConfig{
		SafeSearch:            true,
		YouTubeRestrictedMode: true,
		Categories:            []string{"adult"},
		Services:              []string{"tiktok"},
	}
	err = mock.UpdateParentalControl(nil, "profile-1", config)
	require.NoError(t, err)

	// Now get it
	pc, err = mock.GetParentalControl(nil, "profile-1")
	require.NoError(t, err)
	assert.True(t, pc.SafeSearch)
	assert.True(t, pc.YoutubeRestrictedMode)

	// Test error injection
	mock.GetParentalControlError = assert.AnError
	_, err = mock.GetParentalControl(nil, "profile-1")
	assert.Error(t, err)
}

func TestMockClient_GetDenylist(t *testing.T) {
	mock := NewMockClient()

	// Test getting denylist when empty
	denylist, err := mock.GetDenylist(nil, "profile-1")
	require.NoError(t, err)
	assert.Nil(t, denylist)

	// Sync some domains
	err = mock.SyncDenylist(nil, "profile-1", []string{"bad.com", "evil.com"})
	require.NoError(t, err)

	// Now get it
	denylist, err = mock.GetDenylist(nil, "profile-1")
	require.NoError(t, err)
	assert.Equal(t, 2, len(denylist))

	// Test error injection
	mock.GetDenylistError = assert.AnError
	_, err = mock.GetDenylist(nil, "profile-1")
	assert.Error(t, err)
}

func TestMockClient_GetAllowlist(t *testing.T) {
	mock := NewMockClient()

	// Test getting allowlist when empty
	allowlist, err := mock.GetAllowlist(nil, "profile-1")
	require.NoError(t, err)
	assert.Nil(t, allowlist)

	// Sync some domains
	err = mock.SyncAllowlist(nil, "profile-1", []string{"good.com", "trusted.com"})
	require.NoError(t, err)

	// Now get it
	allowlist, err = mock.GetAllowlist(nil, "profile-1")
	require.NoError(t, err)
	assert.Equal(t, 2, len(allowlist))

	// Test error injection
	mock.GetAllowlistError = assert.AnError
	_, err = mock.GetAllowlist(nil, "profile-1")
	assert.Error(t, err)
}

func TestMockClient_GetSecurityTLDs(t *testing.T) {
	mock := NewMockClient()

	// Test getting security TLDs when empty
	tlds, err := mock.GetSecurityTLDs(nil, "profile-1")
	require.NoError(t, err)
	assert.Nil(t, tlds)

	// Sync some TLDs
	err = mock.SyncSecurityTLDs(nil, "profile-1", []string{"xyz", "tk"})
	require.NoError(t, err)

	// Now get them
	tlds, err = mock.GetSecurityTLDs(nil, "profile-1")
	require.NoError(t, err)
	assert.Equal(t, 2, len(tlds))

	// Test error injection
	mock.GetSecurityTLDsError = assert.AnError
	_, err = mock.GetSecurityTLDs(nil, "profile-1")
	assert.Error(t, err)
}

func TestMockClient_UpdateProfile_NonExistent(t *testing.T) {
	mock := NewMockClient()

	// Update a profile that doesn't exist - should create it
	err := mock.UpdateProfile(nil, "non-existent", "New Name")
	require.NoError(t, err)

	// Verify it was created
	profile, err := mock.GetProfile(nil, "non-existent")
	require.NoError(t, err)
	assert.Equal(t, "New Name", profile.Name)
}

func TestMockClient_UpdateProfile_Existing(t *testing.T) {
	mock := NewMockClient()

	// Create a profile first
	profileID, err := mock.CreateProfile(nil, "Original Name")
	require.NoError(t, err)

	// Update it
	err = mock.UpdateProfile(nil, profileID, "Updated Name")
	require.NoError(t, err)

	// Verify it was updated
	profile, err := mock.GetProfile(nil, profileID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", profile.Name)
}

func TestMockClient_UpdateProfile_Error(t *testing.T) {
	mock := NewMockClient()
	mock.UpdateProfileError = assert.AnError

	err := mock.UpdateProfile(nil, "profile-1", "Name")
	assert.Error(t, err)
}

func TestMockClient_DeleteProfile_Error(t *testing.T) {
	mock := NewMockClient()
	mock.DeleteProfileError = assert.AnError

	err := mock.DeleteProfile(nil, "profile-1")
	assert.Error(t, err)
}

func TestMockClient_UpdatePrivacy_NilConfig(t *testing.T) {
	mock := NewMockClient()

	err := mock.UpdatePrivacy(nil, "profile-1", nil)
	require.NoError(t, err)
}

func TestMockClient_UpdatePrivacy_Error(t *testing.T) {
	mock := NewMockClient()
	mock.UpdatePrivacyError = assert.AnError

	err := mock.UpdatePrivacy(nil, "profile-1", &PrivacyConfig{})
	assert.Error(t, err)
}

func TestMockClient_UpdateParentalControl_NilConfig(t *testing.T) {
	mock := NewMockClient()

	err := mock.UpdateParentalControl(nil, "profile-1", nil)
	require.NoError(t, err)
}

func TestMockClient_UpdateParentalControl_Error(t *testing.T) {
	mock := NewMockClient()
	mock.UpdateParentalControlError = assert.AnError

	err := mock.UpdateParentalControl(nil, "profile-1", &ParentalControlConfig{})
	assert.Error(t, err)
}

func TestMockClient_UpdateSettings_NilConfig(t *testing.T) {
	mock := NewMockClient()

	err := mock.UpdateSettings(nil, "profile-1", nil)
	require.NoError(t, err)
}

func TestMockClient_UpdateSettings_Error(t *testing.T) {
	mock := NewMockClient()
	mock.UpdateSettingsError = assert.AnError

	err := mock.UpdateSettings(nil, "profile-1", &SettingsConfig{})
	assert.Error(t, err)
}

func TestMockClient_SyncAllowlist_Error(t *testing.T) {
	mock := NewMockClient()
	mock.SyncAllowlistError = assert.AnError

	err := mock.SyncAllowlist(nil, "profile-1", []string{"good.com"})
	assert.Error(t, err)
}

func TestMockClient_SyncSecurityTLDs_Error(t *testing.T) {
	mock := NewMockClient()
	mock.SyncSecurityTLDsError = assert.AnError

	err := mock.SyncSecurityTLDs(nil, "profile-1", []string{"xyz"})
	assert.Error(t, err)
}

func TestMockClient_SyncPrivacyBlocklists_Error(t *testing.T) {
	mock := NewMockClient()
	mock.SyncPrivacyBlocklistsError = assert.AnError

	err := mock.SyncPrivacyBlocklists(nil, "profile-1", []string{"blocklist"})
	assert.Error(t, err)
}

func TestMockClient_SyncPrivacyNatives_Error(t *testing.T) {
	mock := NewMockClient()
	mock.SyncPrivacyNativesError = assert.AnError

	err := mock.SyncPrivacyNatives(nil, "profile-1", []string{"apple"})
	assert.Error(t, err)
}

func TestMockClient_EmptyListsSync(t *testing.T) {
	mock := NewMockClient()

	// Sync empty lists - should not panic
	err := mock.SyncDenylist(nil, "profile-1", []string{})
	require.NoError(t, err)

	err = mock.SyncAllowlist(nil, "profile-1", []string{})
	require.NoError(t, err)

	err = mock.SyncSecurityTLDs(nil, "profile-1", []string{})
	require.NoError(t, err)

	err = mock.SyncPrivacyBlocklists(nil, "profile-1", []string{})
	require.NoError(t, err)

	err = mock.SyncPrivacyNatives(nil, "profile-1", []string{})
	require.NoError(t, err)
}
