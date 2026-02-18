package verify

import (
	"bufio"
	"encoding/json"
	"os"
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
func CalculateOverrideTrend(windowDays int) (*OverrideTrend, error) {
	if windowDays <= 0 {
		windowDays = 7
	}

	path := events.DefaultLogPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &OverrideTrend{WindowDays: windowDays, Direction: "flat"}, nil
		}
		return nil, err
	}
	defer f.Close()

	now := time.Now()
	windowStart := now.AddDate(0, 0, -windowDays)
	previousStart := windowStart.AddDate(0, 0, -windowDays)

	current := 0
	previous := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Type != events.EventTypeVerificationBypassed {
			continue
		}

		eventTime := time.Unix(event.Timestamp, 0)
		if eventTime.After(windowStart) || eventTime.Equal(windowStart) {
			current++
			continue
		}
		if eventTime.After(previousStart) || eventTime.Equal(previousStart) {
			previous++
		}
	}
	if err := scanner.Err(); err != nil {
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
