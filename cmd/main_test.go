package main

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
