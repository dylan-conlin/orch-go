package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// CreateIssueRequest is the JSON request body for POST /api/issues.
type CreateIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	IssueType   string   `json:"issue_type,omitempty"` // task, bug, etc.
	Priority    int      `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"` // Optional parent issue for follow-ups
	CausedBy    string   `json:"caused_by,omitempty"` // Optional source issue/commit for regressions
}

// CreateIssueResponse is the JSON response for POST /api/issues.
type CreateIssueResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleIssues handles POST /api/issues - creates a new beads issue.
// This is used by the dashboard to create follow-up issues from synthesis recommendations.
func (s *Server) handleIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if req.Title == "" {
		resp := CreateIssueResponse{Success: false, Error: "Title is required"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	createResult, err := s.bdLimitedCreate(func() (interface{}, error) {
		s.BeadsClientMu.RLock()
		currentClient := s.BeadsClient
		s.BeadsClientMu.RUnlock()

		if currentClient != nil {
			issue, createErr := currentClient.Create(&beads.CreateArgs{
				Title:       req.Title,
				Description: req.Description,
				IssueType:   req.IssueType,
				Priority:    req.Priority,
				Labels:      req.Labels,
				Parent:      req.ParentID,
				CausedBy:    req.CausedBy,
			})
			if createErr != nil {
				return beads.FallbackCreateWithParentAndCause(req.Title, req.Description, req.IssueType, req.Priority, req.Labels, req.ParentID, req.CausedBy)
			}
			return issue, nil
		}
		return beads.FallbackCreateWithParentAndCause(req.Title, req.Description, req.IssueType, req.Priority, req.Labels, req.ParentID, req.CausedBy)
	})
	var issue *beads.Issue
	if err == nil {
		issue = createResult.(*beads.Issue)
	}

	if err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Failed to create issue: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := CreateIssueResponse{ID: issue.ID, Title: issue.Title, Success: true}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// QuestionResponse represents a question for the dashboard.
type QuestionResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	ClosedAt    string   `json:"closed_at,omitempty"`
	CloseReason string   `json:"close_reason,omitempty"`
	Blocking    []string `json:"blocking,omitempty"`
}

// questionsResult is the internal result type for singleflight dedup of questions fetch.
type questionsResult struct {
	open          []QuestionResponse
	investigating []QuestionResponse
	answered      []QuestionResponse
}

// QuestionsAPIResponse is the JSON structure returned by /api/questions.
type QuestionsAPIResponse struct {
	Open          []QuestionResponse `json:"open"`
	Investigating []QuestionResponse `json:"investigating"`
	Answered      []QuestionResponse `json:"answered"`
	TotalCount    int                `json:"total_count"`
	Error         string             `json:"error,omitempty"`
}

// handleQuestions returns questions grouped by status for the dashboard.
func (s *Server) handleQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err, _ := s.bdLimitedQuestions(func() (interface{}, error) {
		cliClient := beads.NewCLIClient()

		allQuestions, listErr := cliClient.List(&beads.ListArgs{
			IssueType: "question",
			Limit:     100,
		})
		if listErr != nil {
			return nil, listErr
		}

		var open, investigating, answered []QuestionResponse
		sevenDaysAgo := time.Now().AddDate(0, 0, -7)

		for _, q := range allQuestions {
			qr := QuestionResponse{
				ID: q.ID, Title: q.Title, Status: q.Status,
				Priority: q.Priority, Labels: q.Labels,
				CreatedAt: q.CreatedAt, ClosedAt: q.ClosedAt, CloseReason: q.CloseReason,
			}

			if fullIssue, showErr := cliClient.Show(q.ID); showErr == nil {
				var dependents []struct {
					ID string `json:"id"`
				}
				if fullIssue.Dependencies != nil {
					json.Unmarshal(fullIssue.Dependencies, &dependents)
					for _, dep := range dependents {
						qr.Blocking = append(qr.Blocking, dep.ID)
					}
				}
			}

			switch q.Status {
			case "open":
				open = append(open, qr)
			case "in_progress", "investigating":
				investigating = append(investigating, qr)
			case "closed", "answered":
				if q.ClosedAt != "" {
					closedTime, parseErr := time.Parse(time.RFC3339, q.ClosedAt)
					if parseErr == nil && closedTime.After(sevenDaysAgo) {
						answered = append(answered, qr)
					}
				} else {
					answered = append(answered, qr)
				}
			}
		}

		return &questionsResult{open: open, investigating: investigating, answered: answered}, nil
	})

	if err != nil {
		resp := QuestionsAPIResponse{
			Open: []QuestionResponse{}, Investigating: []QuestionResponse{},
			Answered: []QuestionResponse{},
			Error:    fmt.Sprintf("Failed to list questions: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	qResult := result.(*questionsResult)
	open := qResult.open
	investigating := qResult.investigating
	answered := qResult.answered

	if open == nil {
		open = []QuestionResponse{}
	}
	if investigating == nil {
		investigating = []QuestionResponse{}
	}
	if answered == nil {
		answered = []QuestionResponse{}
	}

	resp := QuestionsAPIResponse{
		Open: open, Investigating: investigating, Answered: answered,
		TotalCount: len(open) + len(investigating) + len(answered),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode questions: %v", err), http.StatusInternalServerError)
		return
	}
}

// CloseIssueRequest is the JSON request body for POST /api/beads/close.
type CloseIssueRequest struct {
	ID         string `json:"id"`
	Reason     string `json:"reason,omitempty"`
	ProjectDir string `json:"project_dir,omitempty"`
}

// CloseIssueResponse is the JSON response for POST /api/beads/close.
type CloseIssueResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleBeadsClose handles POST /api/beads/close - closes a beads issue.
func (s *Server) handleBeadsClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CloseIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := CloseIssueResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if req.ID == "" {
		resp := CloseIssueResponse{Success: false, Error: "Issue ID is required"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	workDir := req.ProjectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	_, closeErr := s.bdLimitedCreate(func() (interface{}, error) {
		cliClient := beads.NewCLIClient(
			beads.WithWorkDir(workDir),
			beads.WithEnv(append(os.Environ(), "BEADS_NO_DAEMON=1")),
		)
		return nil, cliClient.CloseIssue(req.ID, req.Reason)
	})
	if err := closeErr; err != nil {
		resp := CloseIssueResponse{ID: req.ID, Success: false, Error: fmt.Sprintf("Failed to close issue: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if s.BeadsStatsCache != nil {
		s.BeadsStatsCache.invalidate(req.ProjectDir)
	}

	resp := CloseIssueResponse{ID: req.ID, Success: true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateIssueRequest is the JSON request body for POST /api/beads/update.
type UpdateIssueRequest struct {
	ID           string   `json:"id"`
	Priority     *int     `json:"priority,omitempty"`
	AddLabels    []string `json:"add_labels,omitempty"`
	RemoveLabels []string `json:"remove_labels,omitempty"`
	ProjectDir   string   `json:"project_dir,omitempty"`
}

// UpdateIssueResponse is the JSON response for POST /api/beads/update.
type UpdateIssueResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleBeadsUpdate handles POST /api/beads/update - updates priority and labels for a beads issue.
func (s *Server) handleBeadsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := UpdateIssueResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if req.ID == "" {
		resp := UpdateIssueResponse{Success: false, Error: "Issue ID is required"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	workDir := req.ProjectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	_, updateErr := s.bdLimitedCreate(func() (interface{}, error) {
		cliClient := beads.NewCLIClient(
			beads.WithWorkDir(workDir),
			beads.WithEnv(append(os.Environ(), "BEADS_NO_DAEMON=1")),
		)
		return cliClient.Update(&beads.UpdateArgs{
			ID:           req.ID,
			Priority:     req.Priority,
			AddLabels:    req.AddLabels,
			RemoveLabels: req.RemoveLabels,
		})
	})
	if err := updateErr; err != nil {
		resp := UpdateIssueResponse{ID: req.ID, Success: false, Error: fmt.Sprintf("Failed to update issue: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if s.BeadsStatsCache != nil {
		s.BeadsStatsCache.invalidate(req.ProjectDir)
	}

	resp := UpdateIssueResponse{ID: req.ID, Success: true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
