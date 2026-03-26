package verify

import (
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// OverrideTrend summarizes verification bypass activity over time.
type OverrideTrend struct {
	WindowDays    int    `json:"window_days"`
	CurrentCount  int    `json:"current_count"`
	PreviousCount int    `json:"previous_count"`
	Delta         int    `json:"delta"`
	Direction     string `json:"direction"` // "up", "down", or "flat"
}

// CalculateOverrideTrend counts verification bypass events in the last N days
// and compares to the previous N days to determine trend direction.
// Uses ScanEventsFromPath to read only event files covering the relevant time
// window, skipping the legacy events.jsonl when it predates the query range.
func CalculateOverrideTrend(windowDays int) (*OverrideTrend, error) {
	if windowDays <= 0 {
		windowDays = 7
	}

	now := time.Now()
	windowStart := now.AddDate(0, 0, -windowDays)
	previousStart := windowStart.AddDate(0, 0, -windowDays)

	current := 0
	previous := 0

	err := events.ScanEventsFromPath(events.DefaultLogPath(), previousStart, now, func(event events.Event) {
		if event.Type != events.EventTypeVerificationBypassed {
			return
		}
		eventTime := time.Unix(event.Timestamp, 0)
		if !eventTime.Before(windowStart) {
			current++
		} else {
			previous++
		}
	})
	if err != nil {
		return nil, err
	}

	delta := current - previous
	direction := "flat"
	if delta > 0 {
		direction = "up"
	} else if delta < 0 {
		direction = "down"
	}

	return &OverrideTrend{
		WindowDays:    windowDays,
		CurrentCount:  current,
		PreviousCount: previous,
		Delta:         delta,
		Direction:     direction,
	}, nil
}
