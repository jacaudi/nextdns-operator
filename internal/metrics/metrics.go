package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ProfilesTotal tracks the total number of NextDNSProfile resources
	ProfilesTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nextdns_profiles_total",
		Help: "Total number of NextDNSProfile resources",
	})

	// ProfilesSyncedTotal tracks successful profile syncs
	ProfilesSyncedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nextdns_profiles_synced_total",
		Help: "Total number of successful profile syncs",
	}, []string{"profile", "namespace"})

	// ProfilesSyncErrorsTotal tracks failed profile syncs
	ProfilesSyncErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nextdns_profiles_sync_errors_total",
		Help: "Total number of failed profile syncs",
	}, []string{"profile", "namespace", "reason"})

	// APIRequestDuration tracks NextDNS API call latency
	APIRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nextdns_api_request_duration_seconds",
		Help:    "Duration of NextDNS API requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	// APIRequestsTotal tracks total NextDNS API calls
	APIRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nextdns_api_requests_total",
		Help: "Total number of NextDNS API requests",
	}, []string{"operation", "status"})

	// AllowlistsTotal tracks the total number of NextDNSAllowlist resources
	AllowlistsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nextdns_allowlists_total",
		Help: "Total number of NextDNSAllowlist resources",
	})

	// DenylistsTotal tracks the total number of NextDNSDenylist resources
	DenylistsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nextdns_denylists_total",
		Help: "Total number of NextDNSDenylist resources",
	})

	// TLDListsTotal tracks the total number of NextDNSTLDList resources
	TLDListsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nextdns_tldlists_total",
		Help: "Total number of NextDNSTLDList resources",
	})
)

func init() {
	// Register custom metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		ProfilesTotal,
		ProfilesSyncedTotal,
		ProfilesSyncErrorsTotal,
		APIRequestDuration,
		APIRequestsTotal,
		AllowlistsTotal,
		DenylistsTotal,
		TLDListsTotal,
	)
}

// RecordAPIRequest records an API request with its duration and status
func RecordAPIRequest(operation string, duration float64, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	APIRequestDuration.WithLabelValues(operation).Observe(duration)
	APIRequestsTotal.WithLabelValues(operation, status).Inc()
}

// RecordProfileSync records a successful profile sync
func RecordProfileSync(profile, namespace string) {
	ProfilesSyncedTotal.WithLabelValues(profile, namespace).Inc()
}

// RecordProfileSyncError records a failed profile sync
func RecordProfileSyncError(profile, namespace, reason string) {
	ProfilesSyncErrorsTotal.WithLabelValues(profile, namespace, reason).Inc()
}
