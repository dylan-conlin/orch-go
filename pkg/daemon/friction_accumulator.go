package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FrictionEntry represents a single friction item from an agent's session.
type FrictionEntry struct {
	BeadsID     string    `json:"beads_id"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// FrictionAccumulationResult contains the result of friction scanning.
type FrictionAccumulationResult struct {
	// NewItems is the number of new friction items found.
	NewItems int

	// ByCategoryCount maps category to count of new items.
	ByCategoryCount map[string]int

	// Error is set if the scan failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// FrictionAccumulationSnapshot is a point-in-time snapshot for the daemon status file.
type FrictionAccumulationSnapshot struct {
	NewItems        int            `json:"new_items"`
	ByCategoryCount map[string]int `json:"by_category_count,omitempty"`
	LastCheck       time.Time      `json:"last_check"`
}

// Snapshot converts to a dashboard-ready snapshot.
func (r *FrictionAccumulationResult) Snapshot() FrictionAccumulationSnapshot {
	return FrictionAccumulationSnapshot{
		NewItems:        r.NewItems,
		ByCategoryCount: r.ByCategoryCount,
		LastCheck:       time.Now(),
	}
}

// FrictionAccumulatorService scans completed agents for friction and stores results.
type FrictionAccumulatorService interface {
	Scan() ([]FrictionEntry, error)
	Store(entries []FrictionEntry) error
}

// defaultFrictionAccumulatorService scans recently-closed beads issues for friction comments.
type defaultFrictionAccumulatorService struct {
	storePath string
}

// Scan finds recently-closed issues and extracts friction comments from them.
func (s *defaultFrictionAccumulatorService) Scan() ([]FrictionEntry, error) {
	// List recently-closed issues (last 24 hours)
	cmd := exec.Command("bd", "list", "--status=closed", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list closed issues: %w", err)
	}

	var issues []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse closed issues: %w", err)
	}

	// Load already-processed issue IDs to avoid duplicate scanning
	processed := s.loadProcessed()

	var entries []FrictionEntry
	for _, issue := range issues {
		if processed[issue.ID] {
			continue
		}

		// Get comments for this issue
		commentsCmd := exec.Command("bd", "comments", "list", issue.ID, "--json")
		commentOutput, err := commentsCmd.Output()
		if err != nil {
			continue // Skip issues whose comments can't be fetched
		}

		var comments []struct {
			Text      string    `json:"text"`
			CreatedAt time.Time `json:"created_at"`
		}
		if err := json.Unmarshal(commentOutput, &comments); err != nil {
			continue
		}

		// Extract friction items
		for _, c := range comments {
			text := strings.TrimSpace(c.Text)
			if !strings.HasPrefix(text, "Friction:") {
				continue
			}

			rest := strings.TrimSpace(strings.TrimPrefix(text, "Friction:"))
			if rest == "" || strings.EqualFold(rest, "none") {
				continue
			}

			parts := strings.SplitN(rest, ":", 2)
			category := strings.TrimSpace(parts[0])
			description := ""
			if len(parts) > 1 {
				description = strings.TrimSpace(parts[1])
			}

			entries = append(entries, FrictionEntry{
				BeadsID:     issue.ID,
				Category:    category,
				Description: description,
				Timestamp:   c.CreatedAt,
			})
		}
	}

	return entries, nil
}

// Store appends friction entries to the JSONL store.
func (s *defaultFrictionAccumulatorService) Store(entries []FrictionEntry) error {
	if len(entries) == 0 {
		return nil
	}

	path := s.storePath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home dir: %w", err)
		}
		path = filepath.Join(home, ".orch", "friction.jsonl")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open friction store: %w", err)
	}
	defer f.Close()

	// Track processed issue IDs
	processedIDs := make(map[string]bool)

	for _, entry := range entries {
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write friction entry: %w", err)
		}
		processedIDs[entry.BeadsID] = true
	}

	// Mark these issues as processed
	s.saveProcessed(processedIDs)

	return nil
}

// loadProcessed reads the set of already-processed issue IDs.
func (s *defaultFrictionAccumulatorService) loadProcessed() map[string]bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return make(map[string]bool)
	}

	path := filepath.Join(home, ".orch", "friction-processed.jsonl")
	f, err := os.Open(path)
	if err != nil {
		return make(map[string]bool)
	}
	defer f.Close()

	processed := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		id := strings.TrimSpace(scanner.Text())
		if id != "" {
			processed[id] = true
		}
	}

	return processed
}

// saveProcessed appends newly-processed issue IDs to the tracking file.
func (s *defaultFrictionAccumulatorService) saveProcessed(ids map[string]bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	path := filepath.Join(home, ".orch", "friction-processed.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	for id := range ids {
		f.WriteString(id + "\n")
	}
}

// ShouldRunFrictionAccumulation returns true if periodic friction scanning should run.
func (d *Daemon) ShouldRunFrictionAccumulation() bool {
	return d.Scheduler.IsDue(TaskFrictionAccumulation)
}

// RunPeriodicFrictionAccumulation scans completed agents for friction and accumulates results.
// Returns the result if the scan was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicFrictionAccumulation() *FrictionAccumulationResult {
	if !d.ShouldRunFrictionAccumulation() {
		return nil
	}

	svc := d.FrictionAccumulator
	if svc == nil {
		svc = &defaultFrictionAccumulatorService{}
	}

	entries, err := svc.Scan()
	if err != nil {
		return &FrictionAccumulationResult{
			Error:   err,
			Message: fmt.Sprintf("Friction scan failed: %v", err),
		}
	}

	// Compute category counts
	byCat := make(map[string]int)
	for _, e := range entries {
		byCat[e.Category]++
	}

	result := &FrictionAccumulationResult{
		NewItems:        len(entries),
		ByCategoryCount: byCat,
	}

	if len(entries) > 0 {
		// Store the entries
		if err := svc.Store(entries); err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("Friction: found %d items but store failed: %v", len(entries), err)
		} else {
			parts := make([]string, 0, len(byCat))
			for cat, count := range byCat {
				parts = append(parts, fmt.Sprintf("%s=%d", cat, count))
			}
			result.Message = fmt.Sprintf("Friction: %d new items (%s)", len(entries), strings.Join(parts, " "))
		}
	} else {
		result.Message = "Friction: no new items"
	}

	d.Scheduler.MarkRun(TaskFrictionAccumulation)

	return result
}

// LastFrictionAccumulationTime returns when friction accumulation was last run.
func (d *Daemon) LastFrictionAccumulationTime() time.Time {
	return d.Scheduler.LastRunTime(TaskFrictionAccumulation)
}
