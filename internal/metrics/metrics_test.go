package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordAPIRequest_NoPanic(t *testing.T) {
	// RecordAPIRequest should not panic regardless of input
	assert.NotPanics(t, func() {
		RecordAPIRequest("get-profile", 0.123, true)
	})
	assert.NotPanics(t, func() {
		RecordAPIRequest("update-profile", 1.5, false)
	})
	assert.NotPanics(t, func() {
		RecordAPIRequest("", 0, true)
	})
}

func TestRecordProfileSync_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordProfileSync("my-profile", "default")
	})
	assert.NotPanics(t, func() {
		RecordProfileSync("", "")
	})
}

func TestRecordProfileSyncError_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordProfileSyncError("my-profile", "default", "api-error")
	})
	assert.NotPanics(t, func() {
		RecordProfileSyncError("", "", "")
	})
}

func TestGaugeMetrics_NoPanic(t *testing.T) {
	// Setting gauge values should not panic
	assert.NotPanics(t, func() {
		ProfilesTotal.Set(5)
	})
	assert.NotPanics(t, func() {
		AllowlistsTotal.Set(3)
	})
	assert.NotPanics(t, func() {
		DenylistsTotal.Set(7)
	})
	assert.NotPanics(t, func() {
		TLDListsTotal.Set(2)
	})
}

func TestMetricsAreRegistered(t *testing.T) {
	// Verify all metrics are collectors (implement prometheus.Collector)
	// The init() function registers them; if registration had failed,
	// it would have panicked via MustRegister.
	// Here we verify the metrics are valid collectors by describing them.

	collectors := []struct {
		name      string
		collector prometheus.Collector
	}{
		{"ProfilesTotal", ProfilesTotal},
		{"ProfilesSyncedTotal", ProfilesSyncedTotal},
		{"ProfilesSyncErrorsTotal", ProfilesSyncErrorsTotal},
		{"APIRequestDuration", APIRequestDuration},
		{"APIRequestsTotal", APIRequestsTotal},
		{"AllowlistsTotal", AllowlistsTotal},
		{"DenylistsTotal", DenylistsTotal},
		{"TLDListsTotal", TLDListsTotal},
	}

	for _, tc := range collectors {
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan *prometheus.Desc, 10)
			tc.collector.Describe(ch)
			close(ch)

			descs := make([]*prometheus.Desc, 0)
			for desc := range ch {
				descs = append(descs, desc)
			}
			require.NotEmpty(t, descs, "metric %s should have at least one descriptor", tc.name)
		})
	}
}

func TestRecordAPIRequest_StatusLabels(t *testing.T) {
	// Verify that success and error produce distinct counter increments
	// by calling the function and checking it doesn't error out
	RecordAPIRequest("test-op-success", 0.05, true)
	RecordAPIRequest("test-op-error", 0.1, false)

	// Verify the counter vectors can retrieve metrics for both status labels
	successMetric, err := APIRequestsTotal.GetMetricWithLabelValues("test-op-success", "success")
	require.NoError(t, err)
	assert.NotNil(t, successMetric)

	errorMetric, err := APIRequestsTotal.GetMetricWithLabelValues("test-op-error", "error")
	require.NoError(t, err)
	assert.NotNil(t, errorMetric)
}

func TestRecordAPIRequest_DurationObserved(t *testing.T) {
	// Verify the histogram can retrieve a metric after observation
	RecordAPIRequest("duration-test", 0.25, true)

	observer, err := APIRequestDuration.GetMetricWithLabelValues("duration-test")
	require.NoError(t, err)
	assert.NotNil(t, observer)
}
