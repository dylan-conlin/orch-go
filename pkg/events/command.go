package events

import "time"

// CommandInvokedData contains the data for a command.invoked event.
type CommandInvokedData struct {
	Command string `json:"command"`         // Full command path (e.g., "harness audit", "stats", "doctor")
	Caller  string `json:"caller"`          // human, daemon, orchestrator, worker
	Flags   string `json:"flags,omitempty"` // Notable flags used (e.g., "--json", "--days 7")
}

// LogCommandInvoked logs a command invocation event for usage tracking.
func (l *Logger) LogCommandInvoked(data CommandInvokedData) error {
	eventData := map[string]interface{}{
		"command": data.Command,
		"caller":  data.Caller,
	}
	if data.Flags != "" {
		eventData["flags"] = data.Flags
	}

	return l.Log(Event{
		Type:      EventTypeCommandInvoked,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
