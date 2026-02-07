package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// FocusAPIResponse is the JSON structure returned by /api/focus.
type FocusAPIResponse struct {
	Goal       string `json:"goal,omitempty"`
	BeadsID    string `json:"beads_id,omitempty"`
	SetAt      string `json:"set_at,omitempty"`
	IsDrifting bool   `json:"is_drifting"`
	HasFocus   bool   `json:"has_focus"`
}

// SetFocusRequest is the JSON request body for POST /api/focus.
type SetFocusRequest struct {
	Goal    string `json:"goal"`
	BeadsID string `json:"beads_id,omitempty"`
}

// SetFocusResponse is the JSON response for POST /api/focus.
type SetFocusResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleFocus handles GET, POST, and DELETE for /api/focus.
// GET: returns current focus and drift status
// POST: sets a new focus
// DELETE: clears the current focus
func (s *Server) handleFocus(w http.ResponseWriter, r *http.Request) {
	methodRouter(w, r, map[string]http.HandlerFunc{
		http.MethodGet:    s.handleFocusGet,
		http.MethodPost:   s.handleFocusSet,
		http.MethodDelete: s.handleFocusClear,
	})
}

// handleFocusGet returns current focus and drift status.
func (s *Server) handleFocusGet(w http.ResponseWriter, r *http.Request) {
	store, err := focus.New("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load focus: %v", err), http.StatusInternalServerError)
		return
	}

	resp := FocusAPIResponse{}

	f := store.Get()
	if f != nil {
		resp.HasFocus = true
		resp.Goal = f.Goal
		resp.BeadsID = f.BeadsID
		resp.SetAt = f.SetAt

		// Check drift by getting active agents from current sessions
		client := opencode.NewClient(s.ServerURL)
		sessions, _ := client.ListSessions("")

		var activeIssues []string
		for _, sess := range sessions {
			if beadsID := extractBeadsIDFromTitle(sess.Title); beadsID != "" {
				activeIssues = append(activeIssues, beadsID)
			}
		}

		drift := store.CheckDrift(activeIssues)
		resp.IsDrifting = drift.IsDrifting
	}

	if err := jsonOK(w, resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode focus: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleFocusSet sets a new focus.
func (s *Server) handleFocusSet(w http.ResponseWriter, r *http.Request) {
	var req SetFocusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := SetFocusResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		_ = jsonWithStatus(w, http.StatusBadRequest, resp)
		return
	}

	// Validate: either goal or beads_id must be provided
	if req.Goal == "" && req.BeadsID == "" {
		resp := SetFocusResponse{Success: false, Error: "Either goal or beads_id must be provided"}
		_ = jsonWithStatus(w, http.StatusBadRequest, resp)
		return
	}

	store, err := focus.New("")
	if err != nil {
		resp := SetFocusResponse{Success: false, Error: fmt.Sprintf("Failed to load focus store: %v", err)}
		_ = jsonWithStatus(w, http.StatusInternalServerError, resp)
		return
	}

	// If only beads_id provided, try to get the title for the goal
	goal := req.Goal
	if goal == "" && req.BeadsID != "" {
		// Fetch issue title from beads
		cliClient := beads.NewCLIClient()
		if issue, err := cliClient.Show(req.BeadsID); err == nil && issue != nil {
			goal = issue.Title
		} else {
			goal = req.BeadsID // Fall back to using the ID as the goal
		}
	}

	f := &focus.Focus{
		Goal:    goal,
		BeadsID: req.BeadsID,
	}

	if err := store.Set(f); err != nil {
		resp := SetFocusResponse{Success: false, Error: fmt.Sprintf("Failed to set focus: %v", err)}
		_ = jsonWithStatus(w, http.StatusInternalServerError, resp)
		return
	}

	resp := SetFocusResponse{Success: true}
	_ = jsonOK(w, resp)
}

// handleFocusClear clears the current focus.
func (s *Server) handleFocusClear(w http.ResponseWriter, r *http.Request) {
	store, err := focus.New("")
	if err != nil {
		resp := SetFocusResponse{Success: false, Error: fmt.Sprintf("Failed to load focus store: %v", err)}
		_ = jsonWithStatus(w, http.StatusInternalServerError, resp)
		return
	}

	if err := store.Clear(); err != nil {
		resp := SetFocusResponse{Success: false, Error: fmt.Sprintf("Failed to clear focus: %v", err)}
		_ = jsonWithStatus(w, http.StatusInternalServerError, resp)
		return
	}

	resp := SetFocusResponse{Success: true}
	_ = jsonOK(w, resp)
}
