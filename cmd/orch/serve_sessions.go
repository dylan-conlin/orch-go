package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/dylan-conlin/orch-go/pkg/execution"
)

// SessionAPIResponse is the JSON structure returned by /api/sessions.
type SessionAPIResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title,omitempty"`
	Category      string `json:"category"`
	Role          string `json:"role,omitempty"`
	BeadsID       string `json:"beads_id,omitempty"`
	Tier          string `json:"tier,omitempty"`
	SpawnMode     string `json:"spawn_mode,omitempty"`
	Skill         string `json:"skill,omitempty"`
	Model         string `json:"model,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	ProjectDir    string `json:"project_dir,omitempty"`
	Status        string `json:"status,omitempty"`
	IsProcessing  bool   `json:"is_processing,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

// handleSessions returns JSON list of untracked OpenCode sessions.
// Query parameters:
//   - since: Time filter (12h, 24h, 48h, 7d, all). Default: 12h
//   - project: Project filter (full path or project name). Default: none (all projects)
func handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Default to no time filtering for sessions (matching CLI behavior).
	// The CLI's orch sessions has no time filter; the API should behave the same.
	// Callers can still pass ?since=12h to opt into time filtering.
	sinceDuration := parseSinceParamWithDefault(r, 0)
	projectFilterParam := parseProjectFilter(r)

	projectDir := sourceDir
	if projectDir == "" || projectDir == "unknown" {
		projectDir, _ = os.Getwd()
	}

	client := execution.NewOpenCodeAdapter(serverURL)

	untracked, err := listUntrackedSessions(client, projectDir)
	if err != nil {
		log.Printf("Warning: failed to list sessions: %v", err)
	}

	sessionStatusMap := make(map[string]execution.SessionStatusInfo)
	if len(untracked) > 0 {
		seenIDs := make(map[string]struct{}, len(untracked))
		sessionIDs := make([]string, 0, len(untracked))
		for _, entry := range untracked {
			if entry.Session.ID == "" {
				continue
			}
			if _, exists := seenIDs[entry.Session.ID]; exists {
				continue
			}
			seenIDs[entry.Session.ID] = struct{}{}
			sessionIDs = append(sessionIDs, entry.Session.ID)
		}
		if len(sessionIDs) > 0 {
			if status, err := client.GetSessionStatusByIDs(context.Background(), sessionIDs); err != nil {
				log.Printf("Warning: failed to fetch session status: %v", err)
			} else {
				sessionStatusMap = status
			}
		}
	}

	responses := make([]SessionAPIResponse, 0, len(untracked))
	for _, entry := range untracked {
		updatedAt := entry.Session.Updated
		createdAt := entry.Session.Created
		sessionTime := updatedAt
		if sessionTime.IsZero() {
			sessionTime = createdAt
		}

		if !filterByTime(sessionTime, sinceDuration) {
			continue
		}
		if len(projectFilterParam) > 0 && !filterByProject(entry.Session.Directory, projectFilterParam) {
			continue
		}

		status := "idle"
		isProcessing := false
		if statusInfo, ok := sessionStatusMap[entry.Session.ID]; ok {
			isProcessing = statusInfo.IsBusy() || statusInfo.Type == "retry"
			if isProcessing {
				status = "active"
			}
		}

		responses = append(responses, SessionAPIResponse{
			ID:            entry.Session.ID,
			Title:         entry.Session.Title,
			Category:      entry.Category,
			Role:          entry.Role,
			BeadsID:       entry.BeadsID,
			Tier:          entry.Tier,
			SpawnMode:     entry.SpawnMode,
			Skill:         entry.Skill,
			Model:         entry.Model,
			WorkspacePath: entry.WorkspacePath,
			ProjectDir:    entry.Session.Directory,
			Status:        status,
			IsProcessing:  isProcessing,
			CreatedAt:     formatSessionTime(entry.Session.Created.UnixMilli()),
			UpdatedAt:     formatSessionTime(entry.Session.Updated.UnixMilli()),
		})
	}

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].UpdatedAt > responses[j].UpdatedAt
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode sessions: %v", err), http.StatusInternalServerError)
		return
	}
}
