package stats

import (
	"context"
	"testing"
	"time"
)

func TestTracker_Stats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker := NewTracker(ctx)

	now := time.Now()

	// Add some timestamps manually for testing
	tracker.mu.Lock()
	tracker.timestamps = []time.Time{
		now.Add(-30 * time.Minute), // Last hour
		now.Add(-2 * time.Hour),    // Last 24h
		now.Add(-25 * time.Hour),   // Older (should be ignored by 24h)
	}

	// Ensure "today" includes something from earlier today if possible
	y, m, d := now.Local().Date()
	todayStart := time.Date(y, m, d, 0, 0, 1, 0, now.Local().Location())
	if now.After(todayStart.Add(time.Hour)) {
		tracker.timestamps = append(tracker.timestamps, todayStart)
	}
	tracker.mu.Unlock()

	stats := tracker.Stats()

	if stats.LastHour < 1 {
		t.Errorf("Expected at least 1 request in last hour, got %d", stats.LastHour)
	}

	if stats.Last24Hours < 2 {
		t.Errorf("Expected at least 2 requests in last 24 hours, got %d", stats.Last24Hours)
	}
}

func TestTracker_Cleanup(t *testing.T) {
	ctx := t.Context()

	tracker := NewTracker(ctx)

	tracker.mu.Lock()
	tracker.timestamps = []time.Time{
		time.Now().Add(-26 * time.Hour),
		time.Now().Add(-25 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	}
	tracker.mu.Unlock()

	tracker.cleanup()

	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	if len(tracker.timestamps) != 1 {
		t.Errorf("Expected 1 timestamp after cleanup, got %d", len(tracker.timestamps))
	}
}
