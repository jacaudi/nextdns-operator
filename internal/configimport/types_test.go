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

func ptrBool(b bool) *bool {
	return &b
}
