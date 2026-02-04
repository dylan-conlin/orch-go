package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/attention"
	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// handleLikelyDone returns LIKELY_DONE attention signals for the dashboard.
// These are issues with recent commits but no active workspace, suggesting
// they may be complete but not yet closed.
func handleLikelyDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get project directory from query parameter (default to sourceDir)
	projectDir := r.URL.Query().Get("project")
	if projectDir == "" {
		projectDir = sourceDir
	}

	// Get beads client (RPC or CLI fallback)
	// Note: Must check beadsClient before assigning to interface to avoid
	// Go's nil interface gotcha (interface with nil data is not == nil)
	beadsClientMu.RLock()
	rpcClient := beadsClient
	beadsClientMu.RUnlock()

	var client beads.BeadsClient
	if rpcClient != nil {
		client = rpcClient
	} else {
		client = beads.NewCLIClient(beads.WithWorkDir(projectDir))
	}

	// Check if cache is initialized
	if globalLikelyDoneCache == nil {
		// Fallback: fetch without cache if not initialized
		data, err := attention.CollectLikelyDoneSignals(projectDir, client)
		if err != nil {
			resp := &attention.LikelyDoneResponse{
				Error: fmt.Sprintf("Failed to collect likely done signals: %v", err),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	// Get cached or fresh data
	data, err := globalLikelyDoneCache.Get(projectDir, client)
	if err != nil {
		resp := &attention.LikelyDoneResponse{
			Error: fmt.Sprintf("Failed to collect likely done signals: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode likely done signals: %v", err), http.StatusInternalServerError)
		return
	}
}

// AttentionItemResponse represents an attention item in the API response.
type AttentionItemResponse struct {
	ID          string         `json:"id"`
	Source      string         `json:"source"`
	Concern     string         `json:"concern"`
	Signal      string         `json:"signal"`
	Subject     string         `json:"subject"`
	Summary     string         `json:"summary"`
	Priority    int            `json:"priority"`
	Role        string         `json:"role"`
	ActionHint  string         `json:"action_hint,omitempty"`
	CollectedAt string         `json:"collected_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// AttentionAPIResponse is the response structure for /api/attention endpoint.
type AttentionAPIResponse struct {
	Items       []AttentionItemResponse `json:"items"`
	Total       int                     `json:"total"`
	Sources     []string                `json:"sources"`
	Role        string                  `json:"role"`
	CollectedAt string                  `json:"collected_at"`
}

// handleAttention returns unified attention signals from multiple collectors.
// Query parameters:
//   - role: Role for priority scoring (human, orchestrator, daemon) - default: human
//   - recently_closed_hours: Hours to look back for closed issues - default: 24
func handleAttention(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse role parameter
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "human"
	}

	// Validate role parameter
	validRoles := map[string]bool{
		"human":        true,
		"orchestrator": true,
		"daemon":       true,
	}
	if !validRoles[role] {
		role = "human" // Default to human for invalid roles
	}

	// Parse recently_closed_hours parameter
	recentlyClosedHours := 24 // Default: 24 hours
	if hoursStr := r.URL.Query().Get("recently_closed_hours"); hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil && hours > 0 {
			recentlyClosedHours = hours
		}
	}

	// Get project directory from query parameter (default to sourceDir)
	projectDir := r.URL.Query().Get("project")
	if projectDir == "" {
		projectDir = sourceDir
	}

	// Get beads client (RPC or CLI fallback)
	// Note: Must check beadsClient before assigning to interface to avoid
	// Go's nil interface gotcha (interface with nil data is not == nil)
	beadsClientMu.RLock()
	rpcClient := beadsClient
	beadsClientMu.RUnlock()

	var client beads.BeadsClient
	if rpcClient != nil {
		client = rpcClient
	} else {
		client = beads.NewCLIClient(beads.WithWorkDir(projectDir))
	}

	// Initialize collectors
	collectors := []attention.Collector{}
	sources := []string{}

	// BeadsCollector - ready issues
	beadsCollector := attention.NewBeadsCollector(client)
	collectors = append(collectors, beadsCollector)
	sources = append(sources, "beads")

	// GitCollector - likely-done signals
	if projectDir != "" {
		gitCollector := attention.NewGitCollector(projectDir, client)
		collectors = append(collectors, gitCollector)
		sources = append(sources, "git")
	}

	// RecentlyClosedCollector - recently closed issues for verification
	recentlyClosedCollector := attention.NewRecentlyClosedCollector(client, recentlyClosedHours)
	collectors = append(collectors, recentlyClosedCollector)
	sources = append(sources, "beads-recently-closed")

	// AgentCollector - awaiting-cleanup agents as verify signals
	// Note: Uses HTTPS to call own /api/agents endpoint (loose coupling)
	agentHTTPClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfigSkipVerify(),
		},
	}
	agentAPIURL := fmt.Sprintf("https://localhost:%d", DefaultServePort)
	agentCollector := attention.NewAgentCollector(agentHTTPClient, agentAPIURL)
	collectors = append(collectors, agentCollector)
	sources = append(sources, "agent")

	// EpicOrphanCollector - epics force-closed with open children
	epicOrphanCollector := attention.NewEpicOrphanCollector()
	collectors = append(collectors, epicOrphanCollector)
	sources = append(sources, "epic-orphan")

	// VerifyFailedCollector - issues where auto-completion verification failed
	verifyFailedCollector := attention.NewVerifyFailedCollector("", 72) // Default path, 72h lookback
	collectors = append(collectors, verifyFailedCollector)
	sources = append(sources, "verify-failed")

	// Collect from all sources
	allItems := []attention.AttentionItem{}
	for _, collector := range collectors {
		items, err := collector.Collect(role)
		if err != nil {
			// Log error but continue with other collectors
			// This ensures partial results if one collector fails
			continue
		}
		allItems = append(allItems, items...)
	}

	// Load verifications and filter/annotate items
	verifications := loadVerifications()
	filteredItems := []attention.AttentionItem{}
	for _, item := range allItems {
		verification, exists := verifications[item.Subject]
		if exists && verification.Status == "verified" {
			// Filter out verified issues from recently-closed
			continue
		}
		if exists && verification.Status == "needs_fix" {
			// Add verification_status to metadata for needs_fix items
			if item.Metadata == nil {
				item.Metadata = make(map[string]any)
			}
			item.Metadata["verification_status"] = "needs_fix"
		}
		filteredItems = append(filteredItems, item)
	}
	allItems = filteredItems

	// Sort by priority (lower = higher priority)
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Priority < allItems[j].Priority
	})

	// Transform to response format
	responseItems := make([]AttentionItemResponse, 0, len(allItems))
	for _, item := range allItems {
		responseItems = append(responseItems, AttentionItemResponse{
			ID:          item.ID,
			Source:      item.Source,
			Concern:     item.Concern.String(),
			Signal:      item.Signal,
			Subject:     item.Subject,
			Summary:     item.Summary,
			Priority:    item.Priority,
			Role:        item.Role,
			ActionHint:  item.ActionHint,
			CollectedAt: item.CollectedAt.Format(time.RFC3339),
			Metadata:    item.Metadata,
		})
	}

	// Build response
	response := AttentionAPIResponse{
		Items:       responseItems,
		Total:       len(responseItems),
		Sources:     sources,
		Role:        role,
		CollectedAt: time.Now().Format(time.RFC3339),
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ============================================================================
// Verification API - POST /api/attention/verify
// ============================================================================

// verificationLogPath is the path to the verification JSONL file.
// Can be overridden in tests.
var verificationLogPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/verifications.jsonl"
	}
	return filepath.Join(home, ".orch", "verifications.jsonl")
}()

// VerificationRequest is the request body for POST /api/attention/verify.
type VerificationRequest struct {
	IssueID string `json:"issue_id"`
	Status  string `json:"status"` // "verified" or "needs_fix"
}

// VerificationResponse is the response for POST /api/attention/verify.
type VerificationResponse struct {
	IssueID    string `json:"issue_id"`
	Status     string `json:"status"`
	VerifiedAt string `json:"verified_at"`
}

// VerificationEntry is the JSONL entry for persisted verifications.
type VerificationEntry struct {
	IssueID   string `json:"issue_id"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// handleAttentionVerify handles POST /api/attention/verify requests.
// It marks an issue as verified or needs_fix and persists to JSONL.
func handleAttentionVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req VerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.IssueID == "" {
		http.Error(w, "issue_id is required", http.StatusBadRequest)
		return
	}
	if req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	// Validate status value
	validStatuses := map[string]bool{
		"verified":  true,
		"needs_fix": true,
	}
	if !validStatuses[req.Status] {
		http.Error(w, "status must be 'verified' or 'needs_fix'", http.StatusBadRequest)
		return
	}

	// Create verification entry
	now := time.Now()
	entry := VerificationEntry{
		IssueID:   req.IssueID,
		Status:    req.Status,
		Timestamp: now.Unix(),
	}

	// Persist to JSONL file
	if err := persistVerification(entry); err != nil {
		http.Error(w, fmt.Sprintf("Failed to persist verification: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	response := VerificationResponse{
		IssueID:    req.IssueID,
		Status:     req.Status,
		VerifiedAt: now.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// loadVerifications reads the JSONL file and returns a map of issue_id -> VerificationEntry.
// Returns an empty map if the file doesn't exist or is empty (graceful handling).
func loadVerifications() map[string]VerificationEntry {
	verifications := make(map[string]VerificationEntry)

	data, err := os.ReadFile(verificationLogPath)
	if err != nil {
		// File doesn't exist or can't be read - return empty map (graceful handling)
		return verifications
	}

	// Parse JSONL (newline-delimited JSON) using existing splitLines from guarded.go
	for _, line := range splitLines(string(data)) {
		if len(line) == 0 {
			continue
		}

		var entry VerificationEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}

		// Keep the latest entry for each issue (later entries override earlier ones)
		verifications[entry.IssueID] = entry
	}

	return verifications
}

// persistVerification appends a verification entry to the JSONL file.
func persistVerification(entry VerificationEntry) error {
	// Ensure directory exists
	dir := filepath.Dir(verificationLogPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for appending
	f, err := os.OpenFile(verificationLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Encode and write
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// ============================================================================
// Verify-Failed Actions API - POST /api/attention/verify-failed/{action}
// ============================================================================

// VerifyFailedActionRequest is the request body for verify-failed action endpoints.
type VerifyFailedActionRequest struct {
	IssueID string `json:"issue_id"`
	Reason  string `json:"reason,omitempty"` // For skip-gate and reset-status
	Gate    string `json:"gate,omitempty"`   // For skip-gate: which gate to skip
}

// VerifyFailedActionResponse is the response for verify-failed action endpoints.
type VerifyFailedActionResponse struct {
	IssueID string `json:"issue_id"`
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleVerifyFailedClear handles POST /api/attention/verify-failed/clear requests.
// It removes a verification failure entry, marking it as resolved.
func handleVerifyFailedClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyFailedActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.IssueID == "" {
		http.Error(w, "issue_id is required", http.StatusBadRequest)
		return
	}

	// Clear the verification failure entry
	if err := attention.ClearVerifyFailed(req.IssueID); err != nil {
		resp := VerifyFailedActionResponse{
			IssueID: req.IssueID,
			Action:  "clear",
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := VerifyFailedActionResponse{
		IssueID: req.IssueID,
		Action:  "clear",
		Success: true,
		Message: fmt.Sprintf("Cleared verification failure for %s", req.IssueID),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleVerifyFailedResetStatus handles POST /api/attention/verify-failed/reset-status requests.
// It resets the issue status to 'open' for re-spawning with new instructions.
func handleVerifyFailedResetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyFailedActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.IssueID == "" {
		http.Error(w, "issue_id is required", http.StatusBadRequest)
		return
	}

	// Get beads client
	beadsClientMu.RLock()
	rpcClient := beadsClient
	beadsClientMu.RUnlock()

	var client beads.BeadsClient
	if rpcClient != nil {
		client = rpcClient
	} else {
		client = beads.NewCLIClient()
	}

	// Reset status to 'open'
	openStatus := "open"
	_, err := client.Update(&beads.UpdateArgs{
		ID:     req.IssueID,
		Status: &openStatus,
	})
	if err != nil {
		resp := VerifyFailedActionResponse{
			IssueID: req.IssueID,
			Action:  "reset-status",
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Clear the verification failure entry
	attention.ClearVerifyFailed(req.IssueID)

	// Add a comment explaining the reset
	reason := req.Reason
	if reason == "" {
		reason = "Verification failed, reset for re-spawn"
	}
	client.AddComment(req.IssueID, "system", fmt.Sprintf("Status reset to open: %s", reason))

	resp := VerifyFailedActionResponse{
		IssueID: req.IssueID,
		Action:  "reset-status",
		Success: true,
		Message: fmt.Sprintf("Reset %s to open status", req.IssueID),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
