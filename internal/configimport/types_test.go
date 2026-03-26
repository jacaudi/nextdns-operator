package configimport

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileConfigJSON_Unmarshal(t *testing.T) {
	input := `{
		"security": {
			"aiThreatDetection": true,
			"googleSafeBrowsing": false,
			"cryptojacking": true,
			"threatIntelligenceFeeds": ["feed-a", "feed-b"]
		},
		"privacy": {
			"blocklists": [
				{"id": "nextdns-recommended"},
				{"id": "oisd"}
			],
			"disguisedTrackers": true,
			"allowAffiliate": false
		},
		"denylist": [
			{"domain": "ads.example.com", "active": true},
			{"domain": "tracker.example.com", "active": false}
		],
		"allowlist": [
			{"domain": "safe.example.com", "active": true}
		],
		"settings": {
			"logs": {
				"enabled": true,
				"retention": "30d"
			},
			"blockPage": {
				"enabled": true
			}
		}
	}`

	var cfg ProfileConfigJSON
	err := json.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)

	require.NotNil(t, cfg.Security)
	assert.Equal(t, ptrBool(true), cfg.Security.AIThreatDetection)
	assert.Equal(t, ptrBool(false), cfg.Security.GoogleSafeBrowsing)
	assert.Equal(t, []string{"feed-a", "feed-b"}, cfg.Security.ThreatIntelligenceFeeds)

	require.NotNil(t, cfg.Privacy)
	assert.Len(t, cfg.Privacy.Blocklists, 2)
	assert.Equal(t, "nextdns-recommended", cfg.Privacy.Blocklists[0].ID)

	assert.Len(t, cfg.Denylist, 2)
	assert.Equal(t, "ads.example.com", cfg.Denylist[0].Domain)

	assert.Len(t, cfg.Allowlist, 1)

	require.NotNil(t, cfg.Settings)
	require.NotNil(t, cfg.Settings.Logs)
	assert.Equal(t, ptrBool(true), cfg.Settings.Logs.Enabled)
	assert.Equal(t, "30d", cfg.Settings.Logs.Retention)
}

func TestProfileConfigJSON_EmptyJSON(t *testing.T) {
	var cfg ProfileConfigJSON
	err := json.Unmarshal([]byte(`{}`), &cfg)
	require.NoError(t, err)
	assert.Nil(t, cfg.Security)
	assert.Nil(t, cfg.Privacy)
	assert.Nil(t, cfg.Denylist)
	assert.Nil(t, cfg.Allowlist)
	assert.Nil(t, cfg.Settings)
}

func TestProfileConfigJSON_NilNestedFields(t *testing.T) {
	// Security present but all inner fields omitted
	input := `{
		"security": {},
		"privacy": {},
		"settings": {}
	}`

	var cfg ProfileConfigJSON
	err := json.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)

	require.NotNil(t, cfg.Security)
	assert.Nil(t, cfg.Security.AIThreatDetection)
	assert.Nil(t, cfg.Security.GoogleSafeBrowsing)
	assert.Nil(t, cfg.Security.Cryptojacking)
	assert.Nil(t, cfg.Security.ThreatIntelligenceFeeds)

	require.NotNil(t, cfg.Privacy)
	assert.Nil(t, cfg.Privacy.Blocklists)
	assert.Nil(t, cfg.Privacy.Natives)
	assert.Nil(t, cfg.Privacy.DisguisedTrackers)
	assert.Nil(t, cfg.Privacy.AllowAffiliate)

	require.NotNil(t, cfg.Settings)
	assert.Nil(t, cfg.Settings.Logs)
	assert.Nil(t, cfg.Settings.BlockPage)
	assert.Nil(t, cfg.Settings.Performance)
	assert.Nil(t, cfg.Settings.Web3)
}

func TestProfileConfigJSON_EmptyStringIDs(t *testing.T) {
	input := `{
		"privacy": {
			"blocklists": [{"id": ""}, {"id": "valid-id"}],
			"natives": [{"id": ""}]
		},
		"denylist": [{"domain": ""}],
		"allowlist": [{"domain": ""}],
		"rewrites": [{"from": "", "to": ""}]
	}`

	var cfg ProfileConfigJSON
	err := json.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)

	require.NotNil(t, cfg.Privacy)
	require.Len(t, cfg.Privacy.Blocklists, 2)
	assert.Equal(t, "", cfg.Privacy.Blocklists[0].ID)
	assert.Equal(t, "valid-id", cfg.Privacy.Blocklists[1].ID)

	require.Len(t, cfg.Privacy.Natives, 1)
	assert.Equal(t, "", cfg.Privacy.Natives[0].ID)

	require.Len(t, cfg.Denylist, 1)
	assert.Equal(t, "", cfg.Denylist[0].Domain)

	require.Len(t, cfg.Allowlist, 1)
	assert.Equal(t, "", cfg.Allowlist[0].Domain)

	require.Len(t, cfg.Rewrites, 1)
	assert.Equal(t, "", cfg.Rewrites[0].From)
	assert.Equal(t, "", cfg.Rewrites[0].To)
}

func TestProfileConfigJSON_UnknownFieldsIgnored(t *testing.T) {
	input := `{
		"security": {
			"aiThreatDetection": true,
			"unknownSecurityField": "should-be-ignored",
			"anotherUnknown": 42
		},
		"privacy": {
			"disguisedTrackers": false,
			"futureField": [1, 2, 3]
		},
		"completelyUnknownTopLevel": {"nested": true},
		"settings": {
			"logs": {
				"enabled": true,
				"unknownLogField": "ignored"
			}
		}
	}`

	var cfg ProfileConfigJSON
	err := json.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)

	// Known fields are parsed correctly
	require.NotNil(t, cfg.Security)
	assert.Equal(t, ptrBool(true), cfg.Security.AIThreatDetection)

	require.NotNil(t, cfg.Privacy)
	assert.Equal(t, ptrBool(false), cfg.Privacy.DisguisedTrackers)

	require.NotNil(t, cfg.Settings)
	require.NotNil(t, cfg.Settings.Logs)
	assert.Equal(t, ptrBool(true), cfg.Settings.Logs.Enabled)
}

func TestProfileConfigJSON_SettingsNestedNils(t *testing.T) {
	// Settings present with only logs, other sub-structs nil
	input := `{
		"settings": {
			"logs": {
				"enabled": false,
				"retention": "7d"
			}
		}
	}`

	var cfg ProfileConfigJSON
	err := json.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)

	require.NotNil(t, cfg.Settings)
	require.NotNil(t, cfg.Settings.Logs)
	assert.Equal(t, ptrBool(false), cfg.Settings.Logs.Enabled)
	assert.Equal(t, "7d", cfg.Settings.Logs.Retention)
	assert.Nil(t, cfg.Settings.BlockPage)
	assert.Nil(t, cfg.Settings.Performance)
	assert.Nil(t, cfg.Settings.Web3)

	// All other top-level sections should be nil
	assert.Nil(t, cfg.Security)
	assert.Nil(t, cfg.Privacy)
	assert.Nil(t, cfg.ParentalControl)
	assert.Nil(t, cfg.Denylist)
	assert.Nil(t, cfg.Allowlist)
}

func TestProfileConfigJSON_ParentalControlNested(t *testing.T) {
	input := `{
		"parentalControl": {
			"categories": [{"id": "adult", "active": false}],
			"services": [],
			"safeSearch": true
		}
	}`

	var cfg ProfileConfigJSON
	err := json.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)

	require.NotNil(t, cfg.ParentalControl)
	require.Len(t, cfg.ParentalControl.Categories, 1)
	assert.Equal(t, "adult", cfg.ParentalControl.Categories[0].ID)
	assert.Equal(t, ptrBool(false), cfg.ParentalControl.Categories[0].Active)
	assert.Empty(t, cfg.ParentalControl.Services)
	assert.Equal(t, ptrBool(true), cfg.ParentalControl.SafeSearch)
	assert.Nil(t, cfg.ParentalControl.YouTubeRestrictedMode)
}

func ptrBool(b bool) *bool {
	return &b
}
