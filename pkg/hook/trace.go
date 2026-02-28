package hook

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TraceEntry represents a single hook trace log entry.
type TraceEntry struct {
	Timestamp     float64 `json:"ts"`
	Hook          string  `json:"hook"`
	Event         string  `json:"event"`
	Tool          string  `json:"tool"`
	Decision      string  `json:"decision"`
	DurationMs    float64 `json:"duration_ms"`
	Context       string  `json:"context"`
	Session       string  `json:"session"`
	OutputPreview string  `json:"output_preview,omitempty"`
}

// TraceOptions configures trace reading.
type TraceOptions struct {
	// Limit is the maximum number of entries to return (0 = all).
	Limit int
	// SessionFilter filters by session ID.
	SessionFilter string
	// HookFilter filters by hook name (substring match).
	HookFilter string
	// EventFilter filters by event type.
	EventFilter string
}

// DefaultTracePath returns the default path to the hook trace file.
func DefaultTracePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".orch", "hooks", "trace.jsonl")
	}
	return filepath.Join(home, ".orch", "hooks", "trace.jsonl")
}

// ReadTrace reads and filters trace entries from the trace file.
func ReadTrace(path string, opts TraceOptions) ([]TraceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no trace file found at %s — enable tracing with HOOK_TRACE=1", path)
		}
		return nil, err
	}
	defer f.Close()

	var entries []TraceEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry TraceEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		if opts.SessionFilter != "" && entry.Session != opts.SessionFilter {
			continue
		}
		if opts.HookFilter != "" && !strings.Contains(entry.Hook, opts.HookFilter) {
			continue
		}
		if opts.EventFilter != "" && entry.Event != opts.EventFilter {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("error reading trace file: %w", err)
	}

	// Apply limit (take last N entries)
	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[len(entries)-opts.Limit:]
	}

	return entries, nil
}

// FormatTraceEntry formats a single trace entry for display.
func FormatTraceEntry(entry TraceEntry) string {
	ts := time.Unix(int64(entry.Timestamp), 0).Format("15:04:05")
	return fmt.Sprintf("[%s] %-20s %-14s %-8s %-6s %6.1fms",
		ts, entry.Hook, entry.Event, entry.Tool, entry.Decision, entry.DurationMs)
}

// FormatTraceEntries formats multiple trace entries for display.
func FormatTraceEntries(entries []TraceEntry) string {
	if len(entries) == 0 {
		return "No trace entries found"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-10s %-20s %-14s %-8s %-6s %8s\n",
		"TIME", "HOOK", "EVENT", "TOOL", "RESULT", "DURATION")
	b.WriteString(strings.Repeat("-", 75) + "\n")

	for _, entry := range entries {
		b.WriteString(FormatTraceEntry(entry) + "\n")
	}

	return b.String()
}
