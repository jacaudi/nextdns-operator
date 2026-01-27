package controller

import (
	"testing"
	"time"
)

func TestCalculateSyncInterval(t *testing.T) {
	tests := []struct {
		name       string
		syncPeriod time.Duration
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "zero period returns zero",
			syncPeriod: 0,
			wantMin:    0,
			wantMax:    0,
		},
		{
			name:       "one hour period with jitter",
			syncPeriod: 1 * time.Hour,
			wantMin:    54 * time.Minute, // 1h - 10% = 54min
			wantMax:    66 * time.Minute, // 1h + 10% = 66min
		},
		{
			name:       "five minute period with jitter",
			syncPeriod: 5 * time.Minute,
			wantMin:    270 * time.Second, // 5m - 10% = 4.5m
			wantMax:    330 * time.Second, // 5m + 10% = 5.5m
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to verify randomness is within bounds
			for i := 0; i < 100; i++ {
				got := CalculateSyncInterval(tt.syncPeriod)

				if got < tt.wantMin || got > tt.wantMax {
					t.Errorf("CalculateSyncInterval(%v) = %v, want between %v and %v",
						tt.syncPeriod, got, tt.wantMin, tt.wantMax)
				}
			}
		})
	}
}

func TestCalculateSyncIntervalDistribution(t *testing.T) {
	// Verify jitter actually produces different values (not always the same)
	syncPeriod := 1 * time.Hour
	results := make(map[time.Duration]bool)

	for i := 0; i < 100; i++ {
		result := CalculateSyncInterval(syncPeriod)
		results[result] = true
	}

	// Should have at least 10 different values from 100 runs
	if len(results) < 10 {
		t.Errorf("CalculateSyncInterval produced only %d unique values from 100 runs, expected variety due to jitter", len(results))
	}
}
