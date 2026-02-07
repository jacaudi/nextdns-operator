package configimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

func TestMergeBoolPtr(t *testing.T) {
	t.Run("spec field wins over import", func(t *testing.T) {
		specVal := ptrBool(false)
		mergeBoolPtr(&specVal, ptrBool(true))
		require.NotNil(t, specVal)
		assert.Equal(t, false, *specVal)
	})

	t.Run("import fills nil spec field", func(t *testing.T) {
		var specVal *bool
		mergeBoolPtr(&specVal, ptrBool(true))
		require.NotNil(t, specVal)
		assert.Equal(t, true, *specVal)
	})

	t.Run("both nil is no-op", func(t *testing.T) {
		var specVal *bool
		mergeBoolPtr(&specVal, nil)
		assert.Nil(t, specVal)
	})
}

func TestMergeIntoSpec_NilImport(t *testing.T) {
	spec := &nextdnsv1alpha1.NextDNSProfileSpec{
		Name: "test",
		Security: &nextdnsv1alpha1.SecuritySpec{
			Cryptojacking: ptrBool(true),
		},
	}
	MergeIntoSpec(spec, nil)
	// No panic, spec unchanged
	assert.Equal(t, ptrBool(true), spec.Security.Cryptojacking)
}

func TestMergeSecurity(t *testing.T) {
	t.Run("nil spec security populated from import", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{}
		imported := &ProfileConfigJSON{
			Security: &SecurityJSON{
				AIThreatDetection:  ptrBool(true),
				GoogleSafeBrowsing: ptrBool(false),
				Cryptojacking:      ptrBool(true),
				DNSRebinding:       ptrBool(true),
				IDNHomographs:      ptrBool(false),
				Typosquatting:      ptrBool(true),
				DGA:                ptrBool(false),
				NRD:                ptrBool(true),
				DDNS:               ptrBool(false),
				Parking:            ptrBool(true),
				CSAM:               ptrBool(true),
			},
		}
		MergeIntoSpec(spec, imported)

		require.NotNil(t, spec.Security)
		assert.Equal(t, ptrBool(true), spec.Security.AIThreatDetection)
		assert.Equal(t, ptrBool(false), spec.Security.GoogleSafeBrowsing)
		assert.Equal(t, ptrBool(true), spec.Security.Cryptojacking)
		assert.Equal(t, ptrBool(true), spec.Security.DNSRebinding)
		assert.Equal(t, ptrBool(false), spec.Security.IDNHomographs)
		assert.Equal(t, ptrBool(true), spec.Security.Typosquatting)
		assert.Equal(t, ptrBool(false), spec.Security.DGA)
		assert.Equal(t, ptrBool(true), spec.Security.NRD)
		assert.Equal(t, ptrBool(false), spec.Security.DDNS)
		assert.Equal(t, ptrBool(true), spec.Security.Parking)
		assert.Equal(t, ptrBool(true), spec.Security.CSAM)
	})

	t.Run("spec bool fields win over import", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Security: &nextdnsv1alpha1.SecuritySpec{
				AIThreatDetection: ptrBool(false),
				Cryptojacking:     ptrBool(false),
			},
		}
		imported := &ProfileConfigJSON{
			Security: &SecurityJSON{
				AIThreatDetection:  ptrBool(true),
				GoogleSafeBrowsing: ptrBool(true),
				Cryptojacking:      ptrBool(true),
			},
		}
		MergeIntoSpec(spec, imported)

		// Spec values preserved
		assert.Equal(t, ptrBool(false), spec.Security.AIThreatDetection)
		assert.Equal(t, ptrBool(false), spec.Security.Cryptojacking)
		// Import fills in nil field
		assert.Equal(t, ptrBool(true), spec.Security.GoogleSafeBrowsing)
	})

	t.Run("nil import security is no-op", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{}
		imported := &ProfileConfigJSON{}
		MergeIntoSpec(spec, imported)
		assert.Nil(t, spec.Security)
	})
}

func TestMergePrivacy(t *testing.T) {
	t.Run("nil spec privacy populated from import", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{}
		imported := &ProfileConfigJSON{
			Privacy: &PrivacyJSON{
				DisguisedTrackers: ptrBool(true),
				AllowAffiliate:    ptrBool(false),
				Blocklists: []BlocklistEntryJSON{
					{ID: "nextdns-recommended", Active: ptrBool(true)},
				},
				Natives: []NativeEntryJSON{
					{ID: "apple", Active: ptrBool(true)},
				},
			},
		}
		MergeIntoSpec(spec, imported)

		require.NotNil(t, spec.Privacy)
		assert.Equal(t, ptrBool(true), spec.Privacy.DisguisedTrackers)
		assert.Equal(t, ptrBool(false), spec.Privacy.AllowAffiliate)
		require.Len(t, spec.Privacy.Blocklists, 1)
		assert.Equal(t, "nextdns-recommended", spec.Privacy.Blocklists[0].ID)
		require.Len(t, spec.Privacy.Natives, 1)
		assert.Equal(t, "apple", spec.Privacy.Natives[0].ID)
	})

	t.Run("blocklists merged with dedup by ID", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Privacy: &nextdnsv1alpha1.PrivacySpec{
				Blocklists: []nextdnsv1alpha1.BlocklistEntry{
					{ID: "nextdns-recommended", Active: ptrBool(true)},
					{ID: "oisd", Active: ptrBool(false)},
				},
			},
		}
		imported := &ProfileConfigJSON{
			Privacy: &PrivacyJSON{
				Blocklists: []BlocklistEntryJSON{
					{ID: "nextdns-recommended", Active: ptrBool(false)}, // dup - should be skipped
					{ID: "adguard", Active: ptrBool(true)},             // new - should be added
				},
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Privacy.Blocklists, 3)
		// Original entries preserved
		assert.Equal(t, "nextdns-recommended", spec.Privacy.Blocklists[0].ID)
		assert.Equal(t, ptrBool(true), spec.Privacy.Blocklists[0].Active) // spec value kept
		assert.Equal(t, "oisd", spec.Privacy.Blocklists[1].ID)
		// New entry appended
		assert.Equal(t, "adguard", spec.Privacy.Blocklists[2].ID)
		assert.Equal(t, ptrBool(true), spec.Privacy.Blocklists[2].Active)
	})

	t.Run("natives merged with dedup by ID", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Privacy: &nextdnsv1alpha1.PrivacySpec{
				Natives: []nextdnsv1alpha1.NativeEntry{
					{ID: "apple", Active: ptrBool(true)},
				},
			},
		}
		imported := &ProfileConfigJSON{
			Privacy: &PrivacyJSON{
				Natives: []NativeEntryJSON{
					{ID: "apple", Active: ptrBool(false)},   // dup
					{ID: "windows", Active: ptrBool(true)},  // new
					{ID: "samsung", Active: ptrBool(false)}, // new
				},
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Privacy.Natives, 3)
		assert.Equal(t, "apple", spec.Privacy.Natives[0].ID)
		assert.Equal(t, ptrBool(true), spec.Privacy.Natives[0].Active) // spec value kept
		assert.Equal(t, "windows", spec.Privacy.Natives[1].ID)
		assert.Equal(t, "samsung", spec.Privacy.Natives[2].ID)
	})

	t.Run("spec privacy bools win", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Privacy: &nextdnsv1alpha1.PrivacySpec{
				DisguisedTrackers: ptrBool(false),
			},
		}
		imported := &ProfileConfigJSON{
			Privacy: &PrivacyJSON{
				DisguisedTrackers: ptrBool(true),
				AllowAffiliate:    ptrBool(true),
			},
		}
		MergeIntoSpec(spec, imported)

		assert.Equal(t, ptrBool(false), spec.Privacy.DisguisedTrackers) // spec wins
		assert.Equal(t, ptrBool(true), spec.Privacy.AllowAffiliate)     // import fills
	})
}

func TestMergeParentalControl(t *testing.T) {
	t.Run("nil spec parental control populated from import", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{}
		imported := &ProfileConfigJSON{
			ParentalControl: &ParentalControlJSON{
				SafeSearch:            ptrBool(true),
				YouTubeRestrictedMode: ptrBool(true),
				Categories: []CategoryEntryJSON{
					{ID: "gambling", Active: ptrBool(true)},
				},
				Services: []ServiceEntryJSON{
					{ID: "tiktok", Active: ptrBool(true)},
				},
			},
		}
		MergeIntoSpec(spec, imported)

		require.NotNil(t, spec.ParentalControl)
		assert.Equal(t, ptrBool(true), spec.ParentalControl.SafeSearch)
		assert.Equal(t, ptrBool(true), spec.ParentalControl.YouTubeRestrictedMode)
		require.Len(t, spec.ParentalControl.Categories, 1)
		assert.Equal(t, "gambling", spec.ParentalControl.Categories[0].ID)
		require.Len(t, spec.ParentalControl.Services, 1)
		assert.Equal(t, "tiktok", spec.ParentalControl.Services[0].ID)
	})

	t.Run("categories merged with dedup by ID", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			ParentalControl: &nextdnsv1alpha1.ParentalControlSpec{
				Categories: []nextdnsv1alpha1.CategoryEntry{
					{ID: "gambling", Active: ptrBool(true)},
				},
			},
		}
		imported := &ProfileConfigJSON{
			ParentalControl: &ParentalControlJSON{
				Categories: []CategoryEntryJSON{
					{ID: "gambling", Active: ptrBool(false)}, // dup
					{ID: "adult", Active: ptrBool(true)},     // new
				},
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.ParentalControl.Categories, 2)
		assert.Equal(t, "gambling", spec.ParentalControl.Categories[0].ID)
		assert.Equal(t, ptrBool(true), spec.ParentalControl.Categories[0].Active) // spec kept
		assert.Equal(t, "adult", spec.ParentalControl.Categories[1].ID)
	})

	t.Run("services merged with dedup by ID", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			ParentalControl: &nextdnsv1alpha1.ParentalControlSpec{
				Services: []nextdnsv1alpha1.ServiceEntry{
					{ID: "tiktok", Active: ptrBool(true)},
				},
			},
		}
		imported := &ProfileConfigJSON{
			ParentalControl: &ParentalControlJSON{
				Services: []ServiceEntryJSON{
					{ID: "tiktok", Active: ptrBool(false)},    // dup
					{ID: "youtube", Active: ptrBool(true)},    // new
					{ID: "instagram", Active: ptrBool(false)}, // new
				},
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.ParentalControl.Services, 3)
		assert.Equal(t, "tiktok", spec.ParentalControl.Services[0].ID)
		assert.Equal(t, ptrBool(true), spec.ParentalControl.Services[0].Active) // spec kept
		assert.Equal(t, "youtube", spec.ParentalControl.Services[1].ID)
		assert.Equal(t, "instagram", spec.ParentalControl.Services[2].ID)
	})

	t.Run("spec bools win over import", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			ParentalControl: &nextdnsv1alpha1.ParentalControlSpec{
				SafeSearch: ptrBool(false),
			},
		}
		imported := &ProfileConfigJSON{
			ParentalControl: &ParentalControlJSON{
				SafeSearch:            ptrBool(true),
				YouTubeRestrictedMode: ptrBool(true),
			},
		}
		MergeIntoSpec(spec, imported)

		assert.Equal(t, ptrBool(false), spec.ParentalControl.SafeSearch)           // spec wins
		assert.Equal(t, ptrBool(true), spec.ParentalControl.YouTubeRestrictedMode) // import fills
	})
}

func TestMergeDenylist(t *testing.T) {
	t.Run("appended with deduplication by domain", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Denylist: []nextdnsv1alpha1.DomainEntry{
				{Domain: "ads.example.com", Active: ptrBool(true)},
				{Domain: "tracker.example.com", Active: ptrBool(false)},
			},
		}
		imported := &ProfileConfigJSON{
			Denylist: []DomainEntryJSON{
				{Domain: "ads.example.com", Active: ptrBool(false)},    // dup - skipped
				{Domain: "malware.example.com", Active: ptrBool(true)}, // new - added
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Denylist, 3)
		assert.Equal(t, "ads.example.com", spec.Denylist[0].Domain)
		assert.Equal(t, ptrBool(true), spec.Denylist[0].Active) // spec value kept
		assert.Equal(t, "tracker.example.com", spec.Denylist[1].Domain)
		assert.Equal(t, "malware.example.com", spec.Denylist[2].Domain)
	})

	t.Run("nil spec denylist gets import entries", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{}
		imported := &ProfileConfigJSON{
			Denylist: []DomainEntryJSON{
				{Domain: "ads.example.com", Active: ptrBool(true)},
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Denylist, 1)
		assert.Equal(t, "ads.example.com", spec.Denylist[0].Domain)
	})

	t.Run("empty import denylist is no-op", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Denylist: []nextdnsv1alpha1.DomainEntry{
				{Domain: "ads.example.com", Active: ptrBool(true)},
			},
		}
		imported := &ProfileConfigJSON{}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Denylist, 1)
	})
}

func TestMergeAllowlist(t *testing.T) {
	t.Run("appended with deduplication by domain", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Allowlist: []nextdnsv1alpha1.DomainEntry{
				{Domain: "safe.example.com", Active: ptrBool(true)},
			},
		}
		imported := &ProfileConfigJSON{
			Allowlist: []DomainEntryJSON{
				{Domain: "safe.example.com", Active: ptrBool(false)},   // dup
				{Domain: "trusted.example.com", Active: ptrBool(true)}, // new
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Allowlist, 2)
		assert.Equal(t, "safe.example.com", spec.Allowlist[0].Domain)
		assert.Equal(t, ptrBool(true), spec.Allowlist[0].Active) // spec kept
		assert.Equal(t, "trusted.example.com", spec.Allowlist[1].Domain)
	})
}

func TestMergeRewrites(t *testing.T) {
	t.Run("appended with deduplication by From", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Rewrites: []nextdnsv1alpha1.RewriteEntry{
				{From: "app.local", To: "10.0.0.1", Active: ptrBool(true)},
			},
		}
		imported := &ProfileConfigJSON{
			Rewrites: []RewriteEntryJSON{
				{From: "app.local", To: "10.0.0.2", Active: ptrBool(true)},  // dup by From
				{From: "db.local", To: "10.0.0.3", Active: ptrBool(true)},   // new
				{From: "api.local", To: "10.0.0.4", Active: ptrBool(false)}, // new
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Rewrites, 3)
		assert.Equal(t, "app.local", spec.Rewrites[0].From)
		assert.Equal(t, "10.0.0.1", spec.Rewrites[0].To) // spec value kept
		assert.Equal(t, "db.local", spec.Rewrites[1].From)
		assert.Equal(t, "10.0.0.3", spec.Rewrites[1].To)
		assert.Equal(t, "api.local", spec.Rewrites[2].From)
	})

	t.Run("nil spec rewrites gets import entries", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{}
		imported := &ProfileConfigJSON{
			Rewrites: []RewriteEntryJSON{
				{From: "app.local", To: "10.0.0.1", Active: ptrBool(true)},
			},
		}
		MergeIntoSpec(spec, imported)

		require.Len(t, spec.Rewrites, 1)
		assert.Equal(t, "app.local", spec.Rewrites[0].From)
		assert.Equal(t, "10.0.0.1", spec.Rewrites[0].To)
	})
}

func TestMergeSettings(t *testing.T) {
	t.Run("nil spec settings populated from import", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{}
		imported := &ProfileConfigJSON{
			Settings: &SettingsJSON{
				Logs: &LogsJSON{
					Enabled:       ptrBool(true),
					LogClientsIPs: ptrBool(false),
					LogDomains:    ptrBool(true),
					Retention:     "30d",
				},
				BlockPage: &BlockPageJSON{
					Enabled: ptrBool(true),
				},
				Performance: &PerformanceJSON{
					ECS:             ptrBool(true),
					CacheBoost:      ptrBool(true),
					CNAMEFlattening: ptrBool(false),
				},
				Web3: ptrBool(true),
			},
		}
		MergeIntoSpec(spec, imported)

		require.NotNil(t, spec.Settings)
		require.NotNil(t, spec.Settings.Logs)
		assert.Equal(t, ptrBool(true), spec.Settings.Logs.Enabled)
		assert.Equal(t, ptrBool(false), spec.Settings.Logs.LogClientsIPs)
		assert.Equal(t, ptrBool(true), spec.Settings.Logs.LogDomains)
		assert.Equal(t, "30d", spec.Settings.Logs.Retention)

		require.NotNil(t, spec.Settings.BlockPage)
		assert.Equal(t, ptrBool(true), spec.Settings.BlockPage.Enabled)

		require.NotNil(t, spec.Settings.Performance)
		assert.Equal(t, ptrBool(true), spec.Settings.Performance.ECS)
		assert.Equal(t, ptrBool(true), spec.Settings.Performance.CacheBoost)
		assert.Equal(t, ptrBool(false), spec.Settings.Performance.CNAMEFlattening)

		assert.Equal(t, ptrBool(true), spec.Settings.Web3)
	})

	t.Run("spec settings fields win over import", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Settings: &nextdnsv1alpha1.SettingsSpec{
				Logs: &nextdnsv1alpha1.LogsSpec{
					Enabled:   ptrBool(false),
					Retention: "7d",
				},
				Web3: ptrBool(false),
			},
		}
		imported := &ProfileConfigJSON{
			Settings: &SettingsJSON{
				Logs: &LogsJSON{
					Enabled:       ptrBool(true),
					LogClientsIPs: ptrBool(true),
					Retention:     "30d",
				},
				BlockPage: &BlockPageJSON{
					Enabled: ptrBool(true),
				},
				Performance: &PerformanceJSON{
					ECS: ptrBool(true),
				},
				Web3: ptrBool(true),
			},
		}
		MergeIntoSpec(spec, imported)

		// Spec values preserved
		assert.Equal(t, ptrBool(false), spec.Settings.Logs.Enabled) // spec wins
		assert.Equal(t, "7d", spec.Settings.Logs.Retention)         // spec wins
		assert.Equal(t, ptrBool(false), spec.Settings.Web3)         // spec wins

		// Import fills in nil fields
		assert.Equal(t, ptrBool(true), spec.Settings.Logs.LogClientsIPs) // import fills

		// Import creates nil sub-structs
		require.NotNil(t, spec.Settings.BlockPage)
		assert.Equal(t, ptrBool(true), spec.Settings.BlockPage.Enabled)
		require.NotNil(t, spec.Settings.Performance)
		assert.Equal(t, ptrBool(true), spec.Settings.Performance.ECS)
	})

	t.Run("nil import settings is no-op", func(t *testing.T) {
		spec := &nextdnsv1alpha1.NextDNSProfileSpec{
			Settings: &nextdnsv1alpha1.SettingsSpec{
				Web3: ptrBool(true),
			},
		}
		imported := &ProfileConfigJSON{}
		MergeIntoSpec(spec, imported)

		assert.Equal(t, ptrBool(true), spec.Settings.Web3)
		assert.Nil(t, spec.Settings.Logs)
	})
}

func TestMergeIntoSpec_FullIntegration(t *testing.T) {
	spec := &nextdnsv1alpha1.NextDNSProfileSpec{
		Name: "test-profile",
		Security: &nextdnsv1alpha1.SecuritySpec{
			AIThreatDetection: ptrBool(false), // spec wins
		},
		Denylist: []nextdnsv1alpha1.DomainEntry{
			{Domain: "ads.example.com", Active: ptrBool(true)},
		},
		Rewrites: []nextdnsv1alpha1.RewriteEntry{
			{From: "app.local", To: "10.0.0.1", Active: ptrBool(true)},
		},
	}

	imported := &ProfileConfigJSON{
		Security: &SecurityJSON{
			AIThreatDetection:  ptrBool(true),  // should be ignored
			GoogleSafeBrowsing: ptrBool(true),  // should fill in
			Cryptojacking:      ptrBool(false), // should fill in
		},
		Privacy: &PrivacyJSON{
			DisguisedTrackers: ptrBool(true),
			Blocklists: []BlocklistEntryJSON{
				{ID: "nextdns-recommended", Active: ptrBool(true)},
			},
		},
		ParentalControl: &ParentalControlJSON{
			SafeSearch: ptrBool(true),
			Categories: []CategoryEntryJSON{
				{ID: "gambling", Active: ptrBool(true)},
			},
		},
		Denylist: []DomainEntryJSON{
			{Domain: "ads.example.com", Active: ptrBool(false)},    // dup
			{Domain: "malware.example.com", Active: ptrBool(true)}, // new
		},
		Allowlist: []DomainEntryJSON{
			{Domain: "safe.example.com", Active: ptrBool(true)},
		},
		Settings: &SettingsJSON{
			Logs: &LogsJSON{
				Enabled:   ptrBool(true),
				Retention: "30d",
			},
			Web3: ptrBool(true),
		},
		Rewrites: []RewriteEntryJSON{
			{From: "app.local", To: "10.0.0.2", Active: ptrBool(true)}, // dup
			{From: "db.local", To: "10.0.0.3", Active: ptrBool(true)},  // new
		},
	}

	MergeIntoSpec(spec, imported)

	// Security: spec field wins, import fills nil
	assert.Equal(t, ptrBool(false), spec.Security.AIThreatDetection)
	assert.Equal(t, ptrBool(true), spec.Security.GoogleSafeBrowsing)
	assert.Equal(t, ptrBool(false), spec.Security.Cryptojacking)

	// Privacy: created from import
	require.NotNil(t, spec.Privacy)
	assert.Equal(t, ptrBool(true), spec.Privacy.DisguisedTrackers)
	require.Len(t, spec.Privacy.Blocklists, 1)

	// ParentalControl: created from import
	require.NotNil(t, spec.ParentalControl)
	assert.Equal(t, ptrBool(true), spec.ParentalControl.SafeSearch)
	require.Len(t, spec.ParentalControl.Categories, 1)

	// Denylist: deduped
	require.Len(t, spec.Denylist, 2)
	assert.Equal(t, "ads.example.com", spec.Denylist[0].Domain)
	assert.Equal(t, ptrBool(true), spec.Denylist[0].Active) // spec kept
	assert.Equal(t, "malware.example.com", spec.Denylist[1].Domain)

	// Allowlist: from import
	require.Len(t, spec.Allowlist, 1)

	// Settings: from import
	require.NotNil(t, spec.Settings)
	assert.Equal(t, ptrBool(true), spec.Settings.Logs.Enabled)
	assert.Equal(t, "30d", spec.Settings.Logs.Retention)
	assert.Equal(t, ptrBool(true), spec.Settings.Web3)

	// Rewrites: deduped
	require.Len(t, spec.Rewrites, 2)
	assert.Equal(t, "app.local", spec.Rewrites[0].From)
	assert.Equal(t, "10.0.0.1", spec.Rewrites[0].To) // spec kept
	assert.Equal(t, "db.local", spec.Rewrites[1].From)

	// Name unchanged
	assert.Equal(t, "test-profile", spec.Name)
}
