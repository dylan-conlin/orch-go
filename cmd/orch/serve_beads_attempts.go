package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// AttemptHistoryEntry represents a single attempt for an issue.
type AttemptHistoryEntry struct {
	AttemptNumber int      `json:"attempt_number"`
	Timestamp     string   `json:"timestamp"`      // ISO 8601 timestamp
	Outcome       string   `json:"outcome"`        // success, failed, died, closed→reopened, in_progress
	Phase         string   `json:"phase"`          // last reported phase (e.g., Complete, Implementing, Planning)
	Artifacts     []string `json:"artifacts"`      // list of artifact paths/names
	WorkspaceName string   `json:"workspace_name"` // workspace directory name for reference
}

// AttemptHistoryAPIResponse is the JSON structure returned by /api/beads/{id}/attempts.
type AttemptHistoryAPIResponse struct {
	BeadsID  string                `json:"beads_id"`
	Attempts []AttemptHistoryEntry `json:"attempts"`
	Count    int                   `json:"count"`
	Error    string                `json:"error,omitempty"`
}

// handleBeadsAttempts returns attempt history for a specific beads issue.
func (s *Server) handleBeadsAttempts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathPrefix := "/api/beads/"
	pathSuffix := "/attempts"
	if !strings.HasPrefix(r.URL.Path, pathPrefix) || !strings.HasSuffix(r.URL.Path, pathSuffix) {
		resp := AttemptHistoryAPIResponse{Error: "Invalid URL format"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	beadsID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, pathPrefix), pathSuffix)
	if beadsID == "" {
		resp := AttemptHistoryAPIResponse{Error: "Missing beads ID"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	attemptsResult, attemptsErr, _ := s.bdLimitedAttempts(beadsID, func() (interface{}, error) {
		return s.collectAttemptHistory(beadsID)
	})
	var attempts []AttemptHistoryEntry
	var err error
	if attemptsErr != nil {
		err = attemptsErr
	} else {
		attempts = attemptsResult.([]AttemptHistoryEntry)
	}
	if err != nil {
		resp := AttemptHistoryAPIResponse{
			BeadsID: beadsID,
			Error:   fmt.Sprintf("Failed to collect attempt history: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := AttemptHistoryAPIResponse{BeadsID: beadsID, Attempts: attempts, Count: len(attempts)}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode attempt history: %v", err), http.StatusInternalServerError)
		return
	}
}

// collectAttemptHistory scans all workspaces (including archived) for a given beads ID
// and builds the attempt history.
func (s *Server) collectAttemptHistory(beadsID string) ([]AttemptHistoryEntry, error) {
	workspaceDir := filepath.Join(s.SourceDir, ".orch", "workspace")
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return []AttemptHistoryEntry{}, nil
	}

	type workspaceInfo struct {
		path      string
		spawnTime time.Time
	}
	var workspaces []workspaceInfo

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		beadsIDData, err := os.ReadFile(filepath.Join(dirPath, ".beads_id"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(beadsIDData)) != beadsID {
			continue
		}

		spawnTimeData, err := os.ReadFile(filepath.Join(dirPath, ".spawn_time"))
		if err != nil {
			continue
		}

		var spawnTimeNs int64
		if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
			continue
		}

		workspaces = append(workspaces, workspaceInfo{path: dirPath, spawnTime: time.Unix(0, spawnTimeNs)})
	}

	// Scan archived workspaces
	archivedDir := filepath.Join(workspaceDir, "archived")
	if archivedEntries, err := os.ReadDir(archivedDir); err == nil {
		for _, entry := range archivedEntries {
			if !entry.IsDir() {
				continue
			}

			dirPath := filepath.Join(archivedDir, entry.Name())

			beadsIDData, err := os.ReadFile(filepath.Join(dirPath, ".beads_id"))
			if err != nil {
				continue
			}
			if strings.TrimSpace(string(beadsIDData)) != beadsID {
				continue
			}

			spawnTimeData, err := os.ReadFile(filepath.Join(dirPath, ".spawn_time"))
			if err != nil {
				continue
			}

			var spawnTimeNs int64
			if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
				continue
			}

			workspaces = append(workspaces, workspaceInfo{path: dirPath, spawnTime: time.Unix(0, spawnTimeNs)})
		}
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].spawnTime.Before(workspaces[j].spawnTime)
	})

	attempts := make([]AttemptHistoryEntry, 0, len(workspaces))
	for attemptNum, ws := range workspaces {
		entry := AttemptHistoryEntry{
			AttemptNumber: attemptNum + 1,
			Timestamp:     ws.spawnTime.Format(time.RFC3339),
			WorkspaceName: filepath.Base(ws.path),
		}
		entry.Outcome = determineOutcome(ws.path)
		entry.Artifacts = s.findArtifacts(ws.path)
		attempts = append(attempts, entry)
	}

	if len(attempts) > 0 {
		var comments []beads.Comment
		err := beads.Do(s.SourceDir, func(client *beads.Client) error {
			var rpcErr error
			comments, rpcErr = client.Comments(beadsID)
			return rpcErr
		},
			beads.WithAutoReconnect(3),
			beads.WithTimeout(5*time.Second),
		)
		if err != nil {
			cmd := exec.Command(getBdPath(), "--sandbox", "comments", beadsID, "--json")
			cmd.Dir = s.SourceDir
			cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
			if output, cmdErr := cmd.Output(); cmdErr == nil {
				json.Unmarshal(output, &comments)
			}
		}

		for i := range attempts {
			phase := findPhaseForAttempt(attempts[i].Timestamp, comments)
			if phase != "" {
				attempts[i].Phase = phase
				if strings.EqualFold(phase, "Complete") {
					attempts[i].Outcome = "success"
				}
			}
		}
	}

	return attempts, nil
}

// determineOutcome infers the outcome from workspace state.
func determineOutcome(workspacePath string) string {
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	hasSynthesis := false
	if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
		hasSynthesis = true
	}

	tierFile := filepath.Join(workspacePath, ".tier")
	tier := "full"
	if tierData, err := os.ReadFile(tierFile); err == nil {
		tier = strings.TrimSpace(string(tierData))
	}

	if tier == "full" && hasSynthesis {
		return "success"
	}

	isArchived := strings.Contains(workspacePath, "/archived/")
	if isArchived && !hasSynthesis && tier == "full" {
		return "died"
	}

	return "in_progress"
}

// findArtifacts scans the workspace for produced artifacts.
func (s *Server) findArtifacts(workspacePath string) []string {
	artifacts := []string{}

	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if _, err := os.Stat(synthesisPath); err == nil {
		artifacts = append(artifacts, "SYNTHESIS.md")
	}

	kbDir := filepath.Join(s.SourceDir, ".kb", "investigations")
	if entries, err := os.ReadDir(kbDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filePath := filepath.Join(kbDir, entry.Name())
			if content, err := os.ReadFile(filePath); err == nil {
				workspaceName := filepath.Base(workspacePath)
				if strings.Contains(string(content), workspaceName) {
					artifacts = append(artifacts, filepath.Join(".kb/investigations", entry.Name()))
				}
			}
		}
	}

	return artifacts
}

// findPhaseForAttempt finds the phase reported closest to the attempt timestamp.
func findPhaseForAttempt(attemptTimestamp string, comments []beads.Comment) string {
	attemptTime, err := time.Parse(time.RFC3339, attemptTimestamp)
	if err != nil {
		return ""
	}

	var latestPhase string
	var latestPhaseTime time.Time

	phaseRegex := regexp.MustCompile(`(?i)Phase:\s*(\w+)`)

	for _, comment := range comments {
		matches := phaseRegex.FindStringSubmatch(comment.Text)
		if len(matches) < 2 {
			continue
		}

		commentTime, err := time.Parse(time.RFC3339, comment.CreatedAt)
		if err != nil {
			continue
		}

		if commentTime.After(attemptTime) && commentTime.Before(attemptTime.Add(2*time.Hour)) {
			if latestPhase == "" || commentTime.After(latestPhaseTime) {
				latestPhase = matches[1]
				latestPhaseTime = commentTime
			}
		}
	}

	return latestPhase
}
