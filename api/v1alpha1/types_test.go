package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResourceReference(t *testing.T) {
	ref := ResourceReference{
		Name:      "test-resource",
		Namespace: "test-namespace",
	}

	assert.Equal(t, "test-resource", ref.Name)
	assert.Equal(t, "test-namespace", ref.Namespace)

	// Test with empty namespace
	ref2 := ResourceReference{
		Name: "test-resource",
	}
	assert.Equal(t, "", ref2.Namespace)
}

func TestListReference(t *testing.T) {
	ref := ListReference{
		Name:      "test-list",
		Namespace: "test-namespace",
	}

	assert.Equal(t, "test-list", ref.Name)
	assert.Equal(t, "test-namespace", ref.Namespace)

	// Test with empty namespace (should use profile's namespace)
	ref2 := ListReference{
		Name: "test-list",
	}
	assert.Equal(t, "", ref2.Namespace)
}

func TestSecretKeySelector(t *testing.T) {
	selector := SecretKeySelector{
		Name: "my-secret",
		Key:  "custom-key",
	}

	assert.Equal(t, "my-secret", selector.Name)
	assert.Equal(t, "custom-key", selector.Key)

	// Test with default key
	selector2 := SecretKeySelector{
		Name: "my-secret",
	}
	assert.Equal(t, "", selector2.Key) // Empty, but kubebuilder default is "api-key"
}

func TestDomainEntry(t *testing.T) {
	active := true
	entry := DomainEntry{
		Domain: "example.com",
		Active: &active,
		Reason: "Test domain",
	}

	assert.Equal(t, "example.com", entry.Domain)
	assert.NotNil(t, entry.Active)
	assert.True(t, *entry.Active)
	assert.Equal(t, "Test domain", entry.Reason)

	// Test with nil Active (defaults to true in logic)
	entry2 := DomainEntry{
		Domain: "example.org",
	}
	assert.Nil(t, entry2.Active)
}

func TestDomainEntry_Wildcard(t *testing.T) {
	entry := DomainEntry{
		Domain: "*.example.com",
		Reason: "Wildcard domain",
	}

	assert.Equal(t, "*.example.com", entry.Domain)
}

func TestRewriteEntry(t *testing.T) {
	active := true
	entry := RewriteEntry{
		From:   "old.example.com",
		To:     "new.example.com",
		Active: &active,
	}

	assert.Equal(t, "old.example.com", entry.From)
	assert.Equal(t, "new.example.com", entry.To)
	assert.NotNil(t, entry.Active)
	assert.True(t, *entry.Active)
}

func TestRewriteEntry_ToIP(t *testing.T) {
	entry := RewriteEntry{
		From: "custom.local",
		To:   "192.168.1.100",
	}

	assert.Equal(t, "custom.local", entry.From)
	assert.Equal(t, "192.168.1.100", entry.To)
}

func TestReferencedResourceStatus(t *testing.T) {
	status := ReferencedResourceStatus{
		Name:      "test-allowlist",
		Namespace: "default",
		Ready:     true,
		Count:     10,
	}

	assert.Equal(t, "test-allowlist", status.Name)
	assert.Equal(t, "default", status.Namespace)
	assert.True(t, status.Ready)
	assert.Equal(t, 10, status.Count)
}

func TestNextDNSAllowlist(t *testing.T) {
	allowlist := NextDNSAllowlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-allowlist",
			Namespace: "default",
		},
		Spec: NextDNSAllowlistSpec{
			Description: "Test allowlist",
			Domains: []DomainEntry{
				{Domain: "allowed1.com"},
				{Domain: "allowed2.com"},
			},
		},
	}

	assert.Equal(t, "test-allowlist", allowlist.Name)
	assert.Equal(t, "default", allowlist.Namespace)
	assert.Equal(t, "Test allowlist", allowlist.Spec.Description)
	assert.Equal(t, 2, len(allowlist.Spec.Domains))
}

func TestNextDNSDenylist(t *testing.T) {
	denylist := NextDNSDenylist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-denylist",
			Namespace: "default",
		},
		Spec: NextDNSDenylistSpec{
			Description: "Test denylist",
			Domains: []DomainEntry{
				{Domain: "blocked1.com"},
				{Domain: "blocked2.com"},
			},
		},
	}

	assert.Equal(t, "test-denylist", denylist.Name)
	assert.Equal(t, "default", denylist.Namespace)
	assert.Equal(t, "Test denylist", denylist.Spec.Description)
	assert.Equal(t, 2, len(denylist.Spec.Domains))
}

func TestNextDNSTLDList(t *testing.T) {
	tldList := NextDNSTLDList{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tldlist",
			Namespace: "default",
		},
		Spec: NextDNSTLDListSpec{
			Description: "Test TLD list",
			TLDs: []TLDEntry{
				{TLD: "xyz"},
				{TLD: "tk"},
			},
		},
	}

	assert.Equal(t, "test-tldlist", tldList.Name)
	assert.Equal(t, "default", tldList.Namespace)
	assert.Equal(t, "Test TLD list", tldList.Spec.Description)
	assert.Equal(t, 2, len(tldList.Spec.TLDs))
}

func TestTLDEntry(t *testing.T) {
	active := true
	entry := TLDEntry{
		TLD:    "xyz",
		Active: &active,
		Reason: "High abuse rate",
	}

	assert.Equal(t, "xyz", entry.TLD)
	assert.NotNil(t, entry.Active)
	assert.True(t, *entry.Active)
	assert.Equal(t, "High abuse rate", entry.Reason)
}

func TestNextDNSProfile(t *testing.T) {
	profile := NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	assert.Equal(t, "test-profile", profile.Name)
	assert.Equal(t, "default", profile.Namespace)
	assert.Equal(t, "Test Profile", profile.Spec.Name)
	assert.Equal(t, "nextdns-secret", profile.Spec.CredentialsRef.Name)
}

func TestNextDNSProfileSpec_WithRefs(t *testing.T) {
	spec := NextDNSProfileSpec{
		Name: "Test Profile",
		CredentialsRef: SecretKeySelector{
			Name: "nextdns-secret",
		},
		AllowlistRefs: []ListReference{
			{Name: "allowlist-1"},
			{Name: "allowlist-2", Namespace: "shared"},
		},
		DenylistRefs: []ListReference{
			{Name: "denylist-1"},
		},
		TLDListRefs: []ListReference{
			{Name: "tldlist-1"},
		},
	}

	assert.Equal(t, 2, len(spec.AllowlistRefs))
	assert.Equal(t, 1, len(spec.DenylistRefs))
	assert.Equal(t, 1, len(spec.TLDListRefs))
	assert.Equal(t, "shared", spec.AllowlistRefs[1].Namespace)
}

func TestSecuritySpec(t *testing.T) {
	trueVal := true
	falseVal := false

	spec := SecuritySpec{
		AIThreatDetection:  &trueVal,
		GoogleSafeBrowsing: &trueVal,
		Cryptojacking:      &trueVal,
		DNSRebinding:       &trueVal,
		IDNHomographs:      &trueVal,
		Typosquatting:      &trueVal,
		DGA:                &trueVal,
		NRD:                &falseVal,
		DDNS:               &falseVal,
		Parking:            &trueVal,
		CSAM:               &trueVal,
	}

	assert.True(t, *spec.AIThreatDetection)
	assert.True(t, *spec.GoogleSafeBrowsing)
	assert.True(t, *spec.Cryptojacking)
	assert.True(t, *spec.DNSRebinding)
	assert.True(t, *spec.IDNHomographs)
	assert.True(t, *spec.Typosquatting)
	assert.True(t, *spec.DGA)
	assert.False(t, *spec.NRD)
	assert.False(t, *spec.DDNS)
	assert.True(t, *spec.Parking)
	assert.True(t, *spec.CSAM)
}

func TestPrivacySpec(t *testing.T) {
	trueVal := true
	falseVal := false

	spec := PrivacySpec{
		Blocklists: []BlocklistEntry{
			{ID: "nextdns-recommended"},
			{ID: "oisd"},
		},
		Natives: []NativeEntry{
			{ID: "apple"},
			{ID: "windows"},
		},
		DisguisedTrackers: &trueVal,
		AllowAffiliate:    &falseVal,
	}

	assert.Equal(t, 2, len(spec.Blocklists))
	assert.Equal(t, 2, len(spec.Natives))
	assert.True(t, *spec.DisguisedTrackers)
	assert.False(t, *spec.AllowAffiliate)
}

func TestBlocklistEntry(t *testing.T) {
	active := true
	entry := BlocklistEntry{
		ID:     "nextdns-recommended",
		Active: &active,
	}

	assert.Equal(t, "nextdns-recommended", entry.ID)
	assert.True(t, *entry.Active)
}

func TestNativeEntry(t *testing.T) {
	active := true
	entry := NativeEntry{
		ID:     "apple",
		Active: &active,
	}

	assert.Equal(t, "apple", entry.ID)
	assert.True(t, *entry.Active)
}

func TestParentalControlSpec(t *testing.T) {
	trueVal := true
	falseVal := false

	spec := ParentalControlSpec{
		Categories: []CategoryEntry{
			{ID: "adult"},
			{ID: "gambling"},
		},
		Services: []ServiceEntry{
			{ID: "tiktok"},
			{ID: "instagram"},
		},
		SafeSearch:            &trueVal,
		YouTubeRestrictedMode: &falseVal,
	}

	assert.Equal(t, 2, len(spec.Categories))
	assert.Equal(t, 2, len(spec.Services))
	assert.True(t, *spec.SafeSearch)
	assert.False(t, *spec.YouTubeRestrictedMode)
}

func TestCategoryEntry(t *testing.T) {
	active := true
	entry := CategoryEntry{
		ID:     "adult",
		Active: &active,
	}

	assert.Equal(t, "adult", entry.ID)
	assert.True(t, *entry.Active)
}

func TestServiceEntry(t *testing.T) {
	active := true
	entry := ServiceEntry{
		ID:     "tiktok",
		Active: &active,
	}

	assert.Equal(t, "tiktok", entry.ID)
	assert.True(t, *entry.Active)
}

func TestSettingsSpec(t *testing.T) {
	trueVal := true
	falseVal := false

	spec := SettingsSpec{
		Logs: &LogsSpec{
			Enabled:       &trueVal,
			LogClientsIPs: &falseVal,
			LogDomains:    &trueVal,
			Retention:     "30d",
		},
		BlockPage: &BlockPageSpec{
			Enabled: &trueVal,
		},
		Performance: &PerformanceSpec{
			ECS:             &trueVal,
			CacheBoost:      &trueVal,
			CNAMEFlattening: &trueVal,
		},
		Web3: &falseVal,
	}

	assert.NotNil(t, spec.Logs)
	assert.True(t, *spec.Logs.Enabled)
	assert.False(t, *spec.Logs.LogClientsIPs)
	assert.True(t, *spec.Logs.LogDomains)
	assert.Equal(t, "30d", spec.Logs.Retention)

	assert.NotNil(t, spec.BlockPage)
	assert.True(t, *spec.BlockPage.Enabled)

	assert.NotNil(t, spec.Performance)
	assert.True(t, *spec.Performance.ECS)
	assert.True(t, *spec.Performance.CacheBoost)
	assert.True(t, *spec.Performance.CNAMEFlattening)

	assert.False(t, *spec.Web3)
}

func TestLogsSpec(t *testing.T) {
	trueVal := true

	spec := LogsSpec{
		Enabled:       &trueVal,
		LogClientsIPs: &trueVal,
		LogDomains:    &trueVal,
		Retention:     "7d",
	}

	assert.True(t, *spec.Enabled)
	assert.True(t, *spec.LogClientsIPs)
	assert.True(t, *spec.LogDomains)
	assert.Equal(t, "7d", spec.Retention)
}

func TestBlockPageSpec(t *testing.T) {
	trueVal := true

	spec := BlockPageSpec{
		Enabled: &trueVal,
	}

	assert.True(t, *spec.Enabled)
}

func TestPerformanceSpec(t *testing.T) {
	trueVal := true

	spec := PerformanceSpec{
		ECS:             &trueVal,
		CacheBoost:      &trueVal,
		CNAMEFlattening: &trueVal,
	}

	assert.True(t, *spec.ECS)
	assert.True(t, *spec.CacheBoost)
	assert.True(t, *spec.CNAMEFlattening)
}

func TestAggregatedCounts(t *testing.T) {
	counts := AggregatedCounts{
		AllowlistDomains: 10,
		DenylistDomains:  20,
		BlockedTLDs:      5,
	}

	assert.Equal(t, 10, counts.AllowlistDomains)
	assert.Equal(t, 20, counts.DenylistDomains)
	assert.Equal(t, 5, counts.BlockedTLDs)
}

func TestReferencedResources(t *testing.T) {
	resources := ReferencedResources{
		Allowlists: []ReferencedResourceStatus{
			{Name: "allowlist-1", Namespace: "default", Ready: true, Count: 5},
		},
		Denylists: []ReferencedResourceStatus{
			{Name: "denylist-1", Namespace: "default", Ready: true, Count: 10},
		},
		TLDLists: []ReferencedResourceStatus{
			{Name: "tldlist-1", Namespace: "default", Ready: true, Count: 3},
		},
	}

	assert.Equal(t, 1, len(resources.Allowlists))
	assert.Equal(t, 1, len(resources.Denylists))
	assert.Equal(t, 1, len(resources.TLDLists))
}

func TestNextDNSProfileStatus(t *testing.T) {
	now := metav1.Now()

	status := NextDNSProfileStatus{
		ProfileID:   "abc123",
		Fingerprint: "abc123.dns.nextdns.io",
		AggregatedCounts: &AggregatedCounts{
			AllowlistDomains: 10,
			DenylistDomains:  20,
			BlockedTLDs:      5,
		},
		LastSyncTime:       &now,
		ObservedGeneration: 1,
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
			},
		},
	}

	assert.Equal(t, "abc123", status.ProfileID)
	assert.Equal(t, "abc123.dns.nextdns.io", status.Fingerprint)
	assert.NotNil(t, status.AggregatedCounts)
	assert.NotNil(t, status.LastSyncTime)
	assert.Equal(t, int64(1), status.ObservedGeneration)
	assert.Equal(t, 1, len(status.Conditions))
}

func TestNextDNSAllowlistList(t *testing.T) {
	list := NextDNSAllowlistList{
		Items: []NextDNSAllowlist{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "allowlist-1"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "allowlist-2"},
			},
		},
	}

	assert.Equal(t, 2, len(list.Items))
}

func TestNextDNSDenylistList(t *testing.T) {
	list := NextDNSDenylistList{
		Items: []NextDNSDenylist{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "denylist-1"},
			},
		},
	}

	assert.Equal(t, 1, len(list.Items))
}

func TestNextDNSTLDListList(t *testing.T) {
	list := NextDNSTLDListList{
		Items: []NextDNSTLDList{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "tldlist-1"},
			},
		},
	}

	assert.Equal(t, 1, len(list.Items))
}

func TestNextDNSProfileList(t *testing.T) {
	list := NextDNSProfileList{
		Items: []NextDNSProfile{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "profile-1"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "profile-2"},
			},
		},
	}

	assert.Equal(t, 2, len(list.Items))
}

func TestNextDNSCoreDNSStatus_MultusIPs(t *testing.T) {
	status := NextDNSCoreDNSStatus{
		MultusIPs: []string{"10.10.30.100", "10.10.30.101"},
	}
	assert.Equal(t, []string{"10.10.30.100", "10.10.30.101"}, status.MultusIPs)

	empty := NextDNSCoreDNSStatus{}
	assert.Nil(t, empty.MultusIPs)
}

func TestMultusConfig(t *testing.T) {
	config := MultusConfig{
		NetworkAttachmentDefinition: "vlan30-macvlan",
		Namespace:                   "networking",
		IPs:                         []string{"10.10.30.100", "10.10.30.101"},
	}
	assert.Equal(t, "vlan30-macvlan", config.NetworkAttachmentDefinition)
	assert.Equal(t, "networking", config.Namespace)
	assert.Equal(t, []string{"10.10.30.100", "10.10.30.101"}, config.IPs)

	// Zero value — no namespace, no IPs
	empty := MultusConfig{NetworkAttachmentDefinition: "net1"}
	assert.Equal(t, "", empty.Namespace)
	assert.Nil(t, empty.IPs)
}

func TestNextDNSProfileSpec_FullConfiguration(t *testing.T) {
	trueVal := true
	falseVal := false

	spec := NextDNSProfileSpec{
		Name: "Corporate DNS Policy",
		CredentialsRef: SecretKeySelector{
			Name: "nextdns-credentials",
			Key:  "api-key",
		},
		ProfileID: "existing-profile-123",
		AllowlistRefs: []ListReference{
			{Name: "corporate-allowlist"},
		},
		DenylistRefs: []ListReference{
			{Name: "security-denylist"},
		},
		TLDListRefs: []ListReference{
			{Name: "high-risk-tlds"},
		},
		Denylist: []DomainEntry{
			{Domain: "malware.example.com", Reason: "Known malware"},
		},
		Allowlist: []DomainEntry{
			{Domain: "internal.company.com", Reason: "Internal portal"},
		},
		Security: &SecuritySpec{
			AIThreatDetection:  &trueVal,
			GoogleSafeBrowsing: &trueVal,
			Cryptojacking:      &trueVal,
			DNSRebinding:       &trueVal,
			IDNHomographs:      &trueVal,
			Typosquatting:      &trueVal,
			DGA:                &trueVal,
			NRD:                &falseVal,
			DDNS:               &falseVal,
			Parking:            &trueVal,
			CSAM:               &trueVal,
		},
		Privacy: &PrivacySpec{
			Blocklists: []BlocklistEntry{
				{ID: "nextdns-recommended"},
			},
			Natives: []NativeEntry{
				{ID: "apple"},
			},
			DisguisedTrackers: &trueVal,
			AllowAffiliate:    &falseVal,
		},
		ParentalControl: &ParentalControlSpec{
			Categories: []CategoryEntry{
				{ID: "adult"},
			},
			Services: []ServiceEntry{
				{ID: "tiktok"},
			},
			SafeSearch:            &trueVal,
			YouTubeRestrictedMode: &falseVal,
		},
		Rewrites: []RewriteEntry{
			{From: "old.company.com", To: "new.company.com"},
		},
		Settings: &SettingsSpec{
			Logs: &LogsSpec{
				Enabled:   &trueVal,
				Retention: "30d",
			},
			BlockPage: &BlockPageSpec{
				Enabled: &trueVal,
			},
			Performance: &PerformanceSpec{
				ECS:             &trueVal,
				CacheBoost:      &trueVal,
				CNAMEFlattening: &trueVal,
			},
			Web3: &falseVal,
		},
	}

	assert.Equal(t, "Corporate DNS Policy", spec.Name)
	assert.Equal(t, "existing-profile-123", spec.ProfileID)
	assert.NotNil(t, spec.Security)
	assert.NotNil(t, spec.Privacy)
	assert.NotNil(t, spec.ParentalControl)
	assert.NotNil(t, spec.Settings)
	assert.Equal(t, 1, len(spec.Rewrites))
}

func TestUpstreamConfig_DeviceName(t *testing.T) {
	config := UpstreamConfig{
		Primary:    DNSProtocolDoT,
		DeviceName: "Home Router",
	}
	assert.Equal(t, DNSProtocolDoT, config.Primary)
	assert.Equal(t, "Home Router", config.DeviceName)

	// Zero value — no device name
	empty := UpstreamConfig{Primary: DNSProtocolDoH}
	assert.Equal(t, "", empty.DeviceName)
}

// =============================================================================
// Behavioral tests — DeepCopy independence
// =============================================================================

func TestSecuritySpec_DeepCopy_Independent(t *testing.T) {
	trueVal := true
	falseVal := false

	original := &SecuritySpec{
		AIThreatDetection:       &trueVal,
		GoogleSafeBrowsing:      &trueVal,
		Cryptojacking:           &trueVal,
		DNSRebinding:            &trueVal,
		IDNHomographs:           &trueVal,
		Typosquatting:           &trueVal,
		DGA:                     &trueVal,
		NRD:                     &falseVal,
		DDNS:                    &falseVal,
		Parking:                 &trueVal,
		CSAM:                    &trueVal,
		ThreatIntelligenceFeeds: []string{"feed-a", "feed-b"},
	}

	copied := original.DeepCopy()

	// Verify values match
	assert.Equal(t, *original.AIThreatDetection, *copied.AIThreatDetection)
	assert.Equal(t, *original.NRD, *copied.NRD)
	assert.Equal(t, original.ThreatIntelligenceFeeds, copied.ThreatIntelligenceFeeds)

	// Mutate the copy — original must be unaffected
	*copied.AIThreatDetection = false
	*copied.NRD = true
	copied.ThreatIntelligenceFeeds[0] = "mutated"

	assert.True(t, *original.AIThreatDetection, "original AIThreatDetection should still be true after mutating copy")
	assert.False(t, *original.NRD, "original NRD should still be false after mutating copy")
	assert.Equal(t, "feed-a", original.ThreatIntelligenceFeeds[0], "original feed should be unchanged after mutating copy")
}

func TestNextDNSAllowlist_DeepCopy_Independent(t *testing.T) {
	trueVal := true

	original := &NextDNSAllowlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-allowlist",
			Namespace: "default",
		},
		Spec: NextDNSAllowlistSpec{
			Description: "Test allowlist",
			Domains: []DomainEntry{
				{Domain: "example.com", Active: &trueVal, Reason: "Testing"},
				{Domain: "example.org", Active: &trueVal},
			},
		},
	}

	copied := original.DeepCopy()

	// Verify values match
	assert.Equal(t, original.Name, copied.Name)
	assert.Equal(t, len(original.Spec.Domains), len(copied.Spec.Domains))
	assert.Equal(t, original.Spec.Domains[0].Domain, copied.Spec.Domains[0].Domain)

	// Mutate the copy
	copied.Name = "mutated-allowlist"
	copied.Spec.Description = "Mutated"
	copied.Spec.Domains[0].Domain = "mutated.com"
	*copied.Spec.Domains[0].Active = false

	// Original must be unaffected
	assert.Equal(t, "my-allowlist", original.Name)
	assert.Equal(t, "Test allowlist", original.Spec.Description)
	assert.Equal(t, "example.com", original.Spec.Domains[0].Domain)
	assert.True(t, *original.Spec.Domains[0].Active, "original Active should still be true after mutating copy")
}

func TestNextDNSProfileSpec_DeepCopy_Independent(t *testing.T) {
	trueVal := true
	falseVal := false

	original := &NextDNSProfileSpec{
		Name: "Original Profile",
		CredentialsRef: SecretKeySelector{
			Name: "my-secret",
			Key:  "api-key",
		},
		AllowlistRefs: []ListReference{
			{Name: "allowlist-1", Namespace: "ns1"},
		},
		DenylistRefs: []ListReference{
			{Name: "denylist-1"},
		},
		Security: &SecuritySpec{
			AIThreatDetection: &trueVal,
			NRD:               &falseVal,
		},
		Privacy: &PrivacySpec{
			Blocklists: []BlocklistEntry{
				{ID: "nextdns-recommended", Active: &trueVal},
			},
			DisguisedTrackers: &trueVal,
		},
	}

	copied := original.DeepCopy()

	// Mutate the copy deeply
	copied.Name = "Mutated Profile"
	copied.AllowlistRefs[0].Name = "mutated-allowlist"
	*copied.Security.AIThreatDetection = false
	copied.Privacy.Blocklists[0].ID = "mutated-blocklist"
	*copied.Privacy.DisguisedTrackers = false

	// Original must be unaffected
	assert.Equal(t, "Original Profile", original.Name)
	assert.Equal(t, "allowlist-1", original.AllowlistRefs[0].Name)
	assert.True(t, *original.Security.AIThreatDetection)
	assert.Equal(t, "nextdns-recommended", original.Privacy.Blocklists[0].ID)
	assert.True(t, *original.Privacy.DisguisedTrackers)
}

func TestDeepCopy_NilReceiver(t *testing.T) {
	// DeepCopy on nil receivers should return nil, not panic
	var nilSecurity *SecuritySpec
	assert.Nil(t, nilSecurity.DeepCopy())

	var nilAllowlist *NextDNSAllowlist
	assert.Nil(t, nilAllowlist.DeepCopy())

	var nilProfile *NextDNSProfile
	assert.Nil(t, nilProfile.DeepCopy())

	var nilSettings *SettingsSpec
	assert.Nil(t, nilSettings.DeepCopy())

	var nilLogs *LogsSpec
	assert.Nil(t, nilLogs.DeepCopy())
}
