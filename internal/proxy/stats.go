package proxy

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// StatsResponse represents the JSON response format for the legacy stats endpoint
type StatsResponse struct {
	Today       int `json:"today"`
	LastHour    int `json:"last_hour"`
	Last24Hours int `json:"last_24_hours"`
}

// HandleStatsRequest handles GET /api/stats without authentication
func (h *Handler) HandleStatsRequest(e *core.RequestEvent) error {
	now := time.Now()

	// Calculate start of today in local time
	y, m, d := now.Local().Date()
	todayStart := time.Date(y, m, d, 0, 0, 0, 0, now.Local().Location())

	lastHour := now.Add(-time.Hour)
	last24Hours := now.Add(-24 * time.Hour)

	var today, hourCount, dayCount int

	// Count requests from today
	err := h.app.DB().NewQuery(
		"SELECT count(*) FROM requests WHERE created > {:cutoff}",
	).Bind(dbx.Params{
		"cutoff": todayStart.UTC(),
	}).Row(&today)
	if err != nil {
		return err
	}

	// Count requests from the last hour
	err = h.app.DB().NewQuery(
		"SELECT count(*) FROM requests WHERE created > {:cutoff}",
	).Bind(dbx.Params{
		"cutoff": lastHour.UTC(),
	}).Row(&hourCount)
	if err != nil {
		return err
	}

	// Count requests from the last 24 hours
	err = h.app.DB().NewQuery(
		"SELECT count(*) FROM requests WHERE created > {:cutoff}",
	).Bind(dbx.Params{
		"cutoff": last24Hours.UTC(),
	}).Row(&dayCount)
	if err != nil {
		return err
	}

	return e.JSON(200, StatsResponse{
		Today:       today,
		LastHour:    hourCount,
		Last24Hours: dayCount,
	})
}
