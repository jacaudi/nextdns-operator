package main

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetupLogger_DefaultLevel(t *testing.T) {
	logger := setupLogger("info", "json")
	require.NotNil(t, logger)

	handler := logger.Handler()
	_, ok := handler.(*slog.JSONHandler)
	assert.True(t, ok, "expected JSONHandler for format=json")
}

func TestSetupLogger_DebugLevel(t *testing.T) {
	logger := setupLogger("debug", "json")
	require.NotNil(t, logger)

	assert.True(t, logger.Handler().Enabled(t.Context(), slog.LevelDebug))
}

func TestSetupLogger_WarnLevel(t *testing.T) {
	logger := setupLogger("warn", "json")
	require.NotNil(t, logger)

	assert.False(t, logger.Handler().Enabled(t.Context(), slog.LevelInfo))
	assert.True(t, logger.Handler().Enabled(t.Context(), slog.LevelWarn))
}

func TestSetupLogger_ErrorLevel(t *testing.T) {
	logger := setupLogger("error", "json")
	require.NotNil(t, logger)

	assert.False(t, logger.Handler().Enabled(t.Context(), slog.LevelWarn))
	assert.True(t, logger.Handler().Enabled(t.Context(), slog.LevelError))
}

func TestSetupLogger_UnknownLevelDefaultsToInfo(t *testing.T) {
	logger := setupLogger("garbage", "json")
	require.NotNil(t, logger)

	assert.True(t, logger.Handler().Enabled(t.Context(), slog.LevelInfo))
	assert.False(t, logger.Handler().Enabled(t.Context(), slog.LevelDebug))
}

func TestSetupLogger_TextFormat(t *testing.T) {
	logger := setupLogger("info", "text")
	require.NotNil(t, logger)

	handler := logger.Handler()
	_, ok := handler.(*slog.TextHandler)
	assert.True(t, ok, "expected TextHandler for format=text")
}

func TestSetupLogger_UnknownFormatDefaultsToJSON(t *testing.T) {
	logger := setupLogger("info", "garbage")
	require.NotNil(t, logger)

	handler := logger.Handler()
	_, ok := handler.(*slog.JSONHandler)
	assert.True(t, ok, "expected JSONHandler for unknown format")
}

func TestSetupLogger_CaseInsensitiveLevel(t *testing.T) {
	logger := setupLogger("DEBUG", "json")
	require.NotNil(t, logger)

	assert.True(t, logger.Handler().Enabled(t.Context(), slog.LevelDebug))
}

func TestSetupLogger_CaseInsensitiveFormat(t *testing.T) {
	logger := setupLogger("info", "TEXT")
	require.NotNil(t, logger)

	handler := logger.Handler()
	_, ok := handler.(*slog.TextHandler)
	assert.True(t, ok, "expected TextHandler for format=TEXT (uppercase)")
}

func gatewayList(groupVersion string, kinds ...string) *metav1.APIResourceList {
	list := &metav1.APIResourceList{GroupVersion: groupVersion}
	for _, kind := range kinds {
		list.APIResources = append(list.APIResources, metav1.APIResource{Kind: kind})
	}
	return list
}

func TestDetectGatewayAPI(t *testing.T) {
	tests := []struct {
		name        string
		lists       []*metav1.APIResourceList
		wantOK      bool
		wantMissing []string
	}{
		{
			name: "all three kinds served in v1",
			lists: []*metav1.APIResourceList{
				gatewayList("gateway.networking.k8s.io/v1", "GatewayClass", "Gateway", "TCPRoute", "UDPRoute"),
			},
			wantOK: true,
		},
		{
			name: "only GatewayClass served in v1",
			lists: []*metav1.APIResourceList{
				gatewayList("gateway.networking.k8s.io/v1", "GatewayClass", "Gateway"),
			},
			wantMissing: []string{"TCPRoute", "UDPRoute"},
		},
		{
			name: "routes served only in v1alpha2",
			lists: []*metav1.APIResourceList{
				gatewayList("gateway.networking.k8s.io/v1", "GatewayClass", "Gateway"),
				gatewayList("gateway.networking.k8s.io/v1alpha2", "TCPRoute", "UDPRoute"),
			},
			wantMissing: []string{"TCPRoute", "UDPRoute"},
		},
		{
			name:        "empty discovery",
			wantMissing: []string{"GatewayClass", "TCPRoute", "UDPRoute"},
		},
		{
			name: "partial discovery keeps a complete gateway group",
			lists: []*metav1.APIResourceList{
				gatewayList("apps/v1", "Deployment"),
				gatewayList("gateway.networking.k8s.io/v1", "GatewayClass", "TCPRoute", "UDPRoute"),
			},
			wantOK: true,
		},
		{
			name: "partial discovery lost the gateway group",
			lists: []*metav1.APIResourceList{
				gatewayList("apps/v1", "Deployment"),
			},
			wantMissing: []string{"GatewayClass", "TCPRoute", "UDPRoute"},
		},
		{
			name: "kinds split across the group are not combined from other versions",
			lists: []*metav1.APIResourceList{
				gatewayList("gateway.networking.k8s.io/v1", "GatewayClass", "TCPRoute"),
				gatewayList("gateway.networking.k8s.io/v1alpha2", "UDPRoute"),
			},
			wantMissing: []string{"UDPRoute"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOK, gotMissing := detectGatewayAPI(tt.lists)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantMissing, gotMissing)
		})
	}
}
