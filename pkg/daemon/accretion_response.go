// Package daemon provides autonomous overnight processing capabilities.
// Accretion response reacts to accretion.delta events by creating architect
// extraction issues when files grow rapidly across multiple agent completions.
// This replaces the periodic file-scan approach (proactive extraction) with
// event-driven detection that only reacts to actual agent-caused growth.
package daemon

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

const (
	// AccretionResponseNetDeltaThreshold is the minimum net line growth to trigger an issue.
	AccretionResponseNetDeltaThreshold = 200

	// AccretionResponseMinEvents is the minimum number of completion events
	// contributing to a file's growth before triggering an issue.
	AccretionResponseMinEvents = 3

	// AccretionResponseLabel is the beads label for dedup of accretion response issues.
	AccretionResponseLabel = "daemon:accretion-response"

	// accretionResponseLookbackDays is how far back to scan events.
	accretionResponseLookbackDays = 7
)

// fileAccretion tracks aggregated growth for a single file across events.
type fileAccretion struct {
	NetDelta   int
	EventCount int
}

// AccretionResponseResult contains the result of an accretion response scan.
type AccretionResponseResult struct {
	FilesAnalyzed int
	Created       int
	Skipped       int
	CreatedIssues []string
	Message       string
	Error         error
}

// AccretionResponseService provides I/O for event-driven accretion response.
type AccretionResponseService interface {
	// ReadRecentAccretionDeltas reads accretion.delta events from the last 7 days.
	ReadRecentAccretionDeltas() ([]events.AccretionDeltaData, error)
	// HasOpenExtractionIssue checks if an open architect issue already exists for this file.
	HasOpenExtractionIssue(filePath string) (bool, error)
	// CreateAccretionIssue creates a beads issue for architect review of rapid growth.
	CreateAccretionIssue(filePath string, netDelta, eventCount int) (string, error)
}

// RunPeriodicAccretionResponse reads recent accretion.delta events, aggregates
// per-file growth, and creates architect extraction issues when a file gained
// >200 net lines across >=3 completions.
func (d *Daemon) RunPeriodicAccretionResponse() *AccretionResponseResult {
	if !d.Scheduler.IsDue(TaskAccretionResponse) {
		return nil
	}

	svc := d.AccretionResponse
	if svc == nil {
		return &AccretionResponseResult{
			Error:   fmt.Errorf("accretion response service not configured"),
			Message: "Accretion response: service not configured",
		}
	}

	deltas, err := svc.ReadRecentAccretionDeltas()
	if err != nil {
		return &AccretionResponseResult{
			Error:   err,
			Message: fmt.Sprintf("Accretion response: failed to read events: %v", err),
		}
	}

	// Aggregate per-file growth across all events
	files := make(map[string]*fileAccretion)
	for _, delta := range deltas {
		for _, fd := range delta.FileDeltas {
			fa, ok := files[fd.Path]
			if !ok {
				fa = &fileAccretion{}
				files[fd.Path] = fa
			}
			fa.NetDelta += fd.NetDelta
			fa.EventCount++
		}
	}

	result := &AccretionResponseResult{
		FilesAnalyzed: len(files),
	}

	for path, fa := range files {
		if fa.NetDelta < AccretionResponseNetDeltaThreshold || fa.EventCount < AccretionResponseMinEvents {
			continue
		}

		hasOpen, err := svc.HasOpenExtractionIssue(path)
		if err != nil {
			result.Skipped++
			continue
		}
		if hasOpen {
			result.Skipped++
			continue
		}

		issueID, err := svc.CreateAccretionIssue(path, fa.NetDelta, fa.EventCount)
		if err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("Accretion response: failed to create issue for %s: %v", path, err)
			continue
		}

		result.Created++
		result.CreatedIssues = append(result.CreatedIssues, issueID)
	}

	if result.Created > 0 {
		result.Message = fmt.Sprintf("Accretion response: created %d architect issue(s) for rapidly growing files", result.Created)
	} else if result.FilesAnalyzed > 0 {
		result.Message = fmt.Sprintf("Accretion response: %d file(s) analyzed, %d skipped (dedup or below threshold)",
			result.FilesAnalyzed, result.Skipped)
	} else if result.Error == nil {
		result.Message = "Accretion response: no accretion.delta events in last 7 days"
	}

	d.Scheduler.MarkRun(TaskAccretionResponse)
	return result
}

// --- Default production implementation ---

type defaultAccretionResponseService struct{}

// NewDefaultAccretionResponseService creates a production AccretionResponseService.
func NewDefaultAccretionResponseService() AccretionResponseService {
	return &defaultAccretionResponseService{}
}

func (s *defaultAccretionResponseService) ReadRecentAccretionDeltas() ([]events.AccretionDeltaData, error) {
	eventsPath := events.DefaultLogPath()
	after := time.Now().Add(-accretionResponseLookbackDays * 24 * time.Hour)
	var deltas []events.AccretionDeltaData

	err := events.ScanEventsFromPath(eventsPath, after, time.Time{}, func(event events.Event) {
		if event.Type != events.EventTypeAccretionDelta {
			return
		}

		// Parse file_deltas from event data
		fdRaw, ok := event.Data["file_deltas"]
		if !ok {
			return
		}
		fdBytes, err := json.Marshal(fdRaw)
		if err != nil {
			return
		}
		var fileDeltas []events.FileDelta
		if err := json.Unmarshal(fdBytes, &fileDeltas); err != nil {
			return
		}

		deltas = append(deltas, events.AccretionDeltaData{
			FileDeltas: fileDeltas,
		})
	})
	if err != nil {
		return nil, err
	}

	return deltas, nil
}

func (s *defaultAccretionResponseService) HasOpenExtractionIssue(filePath string) (bool, error) {
	// Check both the new accretion-response label and the legacy proactive-extraction label
	for _, label := range []string{AccretionResponseLabel, ProactiveExtractionLabel} {
		issues, err := ListIssuesWithLabel(label)
		if err != nil {
			return false, err
		}
		baseName := filepath.Base(filePath)
		for _, issue := range issues {
			titleLower := strings.ToLower(issue.Title)
			if strings.Contains(titleLower, strings.ToLower(filePath)) ||
				strings.Contains(titleLower, strings.ToLower(baseName)) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *defaultAccretionResponseService) CreateAccretionIssue(filePath string, netDelta, eventCount int) (string, error) {
	title := fmt.Sprintf("Architect: plan extraction for %s (+%d lines across %d completions)",
		filePath, netDelta, eventCount)

	desc := fmt.Sprintf("%s has grown by %d net lines across %d recent agent completions. "+
		"This rapid growth pattern suggests the file needs extraction or decomposition. "+
		"See .kb/guides/code-extraction-patterns.md for extraction workflow.",
		filePath, netDelta, eventCount)

	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       title,
				Description: desc,
				IssueType:   "task",
				Priority:    3,
				Labels:      []string{AccretionResponseLabel, "triage:ready"},
			})
			if err == nil {
				return issue.ID, nil
			}
		}
	}

	issue, err := beads.FallbackCreate(title, desc, "task", 3, []string{AccretionResponseLabel, "triage:ready"}, "")
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}
