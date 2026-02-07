package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// CompletionCommit represents a git commit referencing the beads issue.
type CompletionCommit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"` // ISO 8601
}

// CompletionDetails holds gathered completion information for a beads issue.
type CompletionDetails struct {
	BeadsID           string             `json:"beads_id"`
	CompletionMessage string             `json:"completion_message,omitempty"` // Phase: Complete comment text
	CompletionDate    string             `json:"completion_date,omitempty"`    // When Phase: Complete was reported
	Commits           []CompletionCommit `json:"commits"`                      // Git commits referencing issue
	Artifacts         []string           `json:"artifacts"`                    // Files created in workspace
	WorkspaceName     string             `json:"workspace_name,omitempty"`     // Active/latest workspace name
}

// CompletionDetailsAPIResponse is the JSON structure returned by /api/beads/{id}/completion.
type CompletionDetailsAPIResponse struct {
	CompletionDetails
	Error string `json:"error,omitempty"`
}

// handleBeadsCompletion returns completion details for a specific beads issue.
func (s *Server) handleBeadsCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathPrefix := "/api/beads/"
	pathSuffix := "/completion"
	if !strings.HasPrefix(r.URL.Path, pathPrefix) || !strings.HasSuffix(r.URL.Path, pathSuffix) {
		resp := CompletionDetailsAPIResponse{
			CompletionDetails: CompletionDetails{Commits: []CompletionCommit{}, Artifacts: []string{}},
			Error:             "Invalid URL format",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	beadsID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, pathPrefix), pathSuffix)
	if beadsID == "" {
		resp := CompletionDetailsAPIResponse{
			CompletionDetails: CompletionDetails{Commits: []CompletionCommit{}, Artifacts: []string{}},
			Error:             "Missing beads ID",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	completionResult, completionErr, _ := s.bdLimitedCompletion(beadsID, func() (interface{}, error) {
		return s.collectCompletionDetails(beadsID)
	})

	if completionErr != nil {
		resp := CompletionDetailsAPIResponse{
			CompletionDetails: CompletionDetails{
				BeadsID:   beadsID,
				Commits:   []CompletionCommit{},
				Artifacts: []string{},
			},
			Error: fmt.Sprintf("Failed to collect completion details: %v", completionErr),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	details := completionResult.(*CompletionDetails)
	resp := CompletionDetailsAPIResponse{CompletionDetails: *details}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode completion details: %v", err), http.StatusInternalServerError)
		return
	}
}

// collectCompletionDetails gathers all completion information for a beads issue.
func (s *Server) collectCompletionDetails(beadsID string) (*CompletionDetails, error) {
	details := &CompletionDetails{
		BeadsID:   beadsID,
		Commits:   []CompletionCommit{},
		Artifacts: []string{},
	}

	// 1. Get Phase: Complete comment from beads
	s.collectCompletionComment(beadsID, details)

	// 2. Find git commits referencing this issue ID
	s.collectCompletionCommits(beadsID, details)

	// 3. Find workspace artifacts
	s.collectWorkspaceArtifacts(beadsID, details)

	return details, nil
}

// collectCompletionComment finds the Phase: Complete comment for the issue.
func (s *Server) collectCompletionComment(beadsID string, details *CompletionDetails) {
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
		// Fallback to CLI
		cmd := exec.Command(getBdPath(), "comments", beadsID, "--json")
		cmd.Dir = s.SourceDir
		cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
		if output, cmdErr := cmd.Output(); cmdErr == nil {
			json.Unmarshal(output, &comments)
		}
	}

	// Find Phase: Complete comment (search from newest to oldest)
	phaseCompleteRegex := regexp.MustCompile(`(?i)Phase:\s*Complete\b`)
	for i := len(comments) - 1; i >= 0; i-- {
		if phaseCompleteRegex.MatchString(comments[i].Text) {
			details.CompletionMessage = comments[i].Text
			details.CompletionDate = comments[i].CreatedAt
			break
		}
	}
}

// collectCompletionCommits finds git commits that reference the beads issue ID.
func (s *Server) collectCompletionCommits(beadsID string, details *CompletionDetails) {
	// Use git log with --grep to find commits mentioning this issue ID
	cmd := exec.Command("git", "log", "--all",
		"--grep="+beadsID,
		"--format=%H|%s|%an|%aI",
		"--max-count=20",
	)
	cmd.Dir = s.SourceDir

	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}
		details.Commits = append(details.Commits, CompletionCommit{
			Hash:    parts[0][:min(12, len(parts[0]))], // Short hash
			Subject: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
}

// collectWorkspaceArtifacts finds artifacts from the workspace associated with this issue.
func (s *Server) collectWorkspaceArtifacts(beadsID string, details *CompletionDetails) {
	workspaceDir := filepath.Join(s.SourceDir, ".orch", "workspace")
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return
	}

	// Check active workspaces first, then archived
	dirs := []string{workspaceDir, filepath.Join(workspaceDir, "archived")}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "archived" {
				continue
			}

			dirPath := filepath.Join(dir, entry.Name())

			// Check if this workspace belongs to our issue
			beadsIDData, err := os.ReadFile(filepath.Join(dirPath, ".beads_id"))
			if err != nil {
				continue
			}
			if strings.TrimSpace(string(beadsIDData)) != beadsID {
				continue
			}

			// Found a matching workspace
			if details.WorkspaceName == "" {
				details.WorkspaceName = entry.Name()
			}

			// Collect artifacts from this workspace
			s.collectArtifactsFromWorkspace(dirPath, details)
		}
	}

	// Also check .kb for artifacts that reference this issue
	s.collectKBArtifacts(beadsID, details)
}

// collectArtifactsFromWorkspace scans a single workspace for notable artifacts.
func (s *Server) collectArtifactsFromWorkspace(workspacePath string, details *CompletionDetails) {
	// Check for SYNTHESIS.md
	if _, err := os.Stat(filepath.Join(workspacePath, "SYNTHESIS.md")); err == nil {
		details.Artifacts = appendUnique(details.Artifacts, "SYNTHESIS.md")
	}

	// Check for SPAWN_CONTEXT.md
	if _, err := os.Stat(filepath.Join(workspacePath, "SPAWN_CONTEXT.md")); err == nil {
		details.Artifacts = appendUnique(details.Artifacts, "SPAWN_CONTEXT.md")
	}
}

// collectKBArtifacts finds .kb artifacts that reference this beads issue.
func (s *Server) collectKBArtifacts(beadsID string, details *CompletionDetails) {
	kbDirs := []string{
		filepath.Join(s.SourceDir, ".kb", "investigations"),
		filepath.Join(s.SourceDir, ".kb", "decisions"),
	}

	for _, kbDir := range kbDirs {
		entries, err := os.ReadDir(kbDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filePath := filepath.Join(kbDir, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			if strings.Contains(string(content), beadsID) {
				relPath, _ := filepath.Rel(s.SourceDir, filePath)
				if relPath != "" {
					details.Artifacts = appendUnique(details.Artifacts, relPath)
				}
			}
		}
	}
}

// appendUnique appends a value to a slice only if it's not already present.
func appendUnique(slice []string, val string) []string {
	for _, existing := range slice {
		if existing == val {
			return slice
		}
	}
	return append(slice, val)
}
