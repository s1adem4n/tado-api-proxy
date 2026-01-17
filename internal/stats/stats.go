package stats

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// StatsResponse represents the JSON response format
type StatsResponse struct {
	Today       int `json:"today"`
	LastHour    int `json:"last_hour"`
	Last24Hours int `json:"last_24_hours"`
}

// Tracker manages the collection and reporting of request statistics
type Tracker struct {
	mu         sync.RWMutex
	timestamps []time.Time
}

// NewTracker creates a new stats tracker and starts the cleanup process
func NewTracker(ctx context.Context) *Tracker {
	t := &Tracker{
		timestamps: make([]time.Time, 0),
	}
	go t.startCleanup(ctx)
	return t
}

// Record logs a new request timestamp
func (t *Tracker) Record() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timestamps = append(t.timestamps, time.Now())
}

// Stats returns the current statistics
func (t *Tracker) Stats() StatsResponse {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	lastHour := now.Add(-time.Hour)
	last24Hours := now.Add(-24 * time.Hour)

	// Calculate start of today in local time
	y, m, d := now.Local().Date()
	todayStart := time.Date(y, m, d, 0, 0, 0, 0, now.Local().Location())

	stats := StatsResponse{}
	for _, ts := range t.timestamps {
		if ts.After(todayStart) {
			stats.Today++
		}
		if ts.After(lastHour) {
			stats.LastHour++
		}
		if ts.After(last24Hours) {
			stats.Last24Hours++
		}
	}

	return stats
}

// startCleanup periodically removes timestamps older than 24 hours
func (t *Tracker) startCleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.cleanup()
		}
	}
}

func (t *Tracker) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	newIndex := 0
	for i, ts := range t.timestamps {
		if ts.After(cutoff) {
			newIndex = i
			break
		}
		// If we reach the end and all are old
		if i == len(t.timestamps)-1 {
			newIndex = len(t.timestamps)
		}
	}

	if newIndex > 0 {
		t.timestamps = t.timestamps[newIndex:]
	}
}

// ServeHTTP implements the http.Handler interface
func (t *Tracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t.Stats())
}
