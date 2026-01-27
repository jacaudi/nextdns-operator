package controller

import (
	"math/rand"
	"time"
)

// CalculateSyncInterval calculates the next sync interval with ±10% jitter
// to prevent thundering herd when multiple resources sync simultaneously.
// Returns 0 if syncPeriod is 0 (periodic sync disabled).
func CalculateSyncInterval(syncPeriod time.Duration) time.Duration {
	if syncPeriod == 0 {
		return 0
	}

	// Calculate 10% jitter (±10%)
	jitterRange := float64(syncPeriod) * 0.1
	jitter := time.Duration(rand.Float64()*2*jitterRange - jitterRange)

	return syncPeriod + jitter
}
