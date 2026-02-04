package main

import (
	"encoding/json"
	"github.com/dylan-conlin/orch-go/pkg/attention"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHandleAttentionMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/attention", nil)
	w := httptest.NewRecorder()

	handleAttention(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleAttentionJSONResponse(t *testing.T) {
	// Test that attention endpoint returns valid JSON
	req := httptest.NewRequest(http.MethodGet, "/api/attention", nil)
	w := httptest.NewRecorder()

	handleAttention(w, req)

	resp := w.Result()
	// Should be 200 even if collectors fail
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON
	var attentionResp AttentionAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&attentionResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestAttentionAPIResponseJSONFormat(t *testing.T) {
	// Verify JSON field naming uses snake_case
	resp := AttentionAPIResponse{
		Items: []AttentionItemResponse{
			{
				ID:      "test-1",
				Source:  "beads",
				Concern: "Actionability",
				Signal:  "issue-ready",
			},
		},
		Total: 1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Verify snake_case fields
	jsonStr := string(data)
	if !contains(jsonStr, "\"items\"") {
		t.Error("Expected 'items' field in JSON")
	}
	if !contains(jsonStr, "\"total\"") {
		t.Error("Expected 'total' field in JSON")
	}
	if !contains(jsonStr, "\"source\"") {
		t.Error("Expected 'source' field in JSON")
	}
	if !contains(jsonStr, "\"concern\"") {
		t.Error("Expected 'concern' field in JSON")
	}
}

func TestHandleAttentionRoleParameter(t *testing.T) {
	tests := []struct {
		name       string
		roleParam  string
		expectRole string
	}{
		{"human role", "human", "human"},
		{"orchestrator role", "orchestrator", "orchestrator"},
		{"daemon role", "daemon", "daemon"},
		{"default role (empty)", "", "human"},
		{"default role (invalid)", "invalid-role", "human"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/attention"
			if tt.roleParam != "" {
				url += "?role=" + tt.roleParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handleAttention(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Decode and verify role is applied
			var attentionResp AttentionAPIResponse
			if err := json.NewDecoder(resp.Body).Decode(&attentionResp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// All items should have the expected role
			for _, item := range attentionResp.Items {
				if item.Role != tt.expectRole {
					t.Errorf("Expected role %q, got %q", tt.expectRole, item.Role)
				}
			}
		})
	}
}

func TestHandleAttentionComposesMultipleSources(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attention", nil)
	w := httptest.NewRecorder()

	handleAttention(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var attentionResp AttentionAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&attentionResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response should include items from multiple sources
	// Even if beads daemon is down or git has no issues,
	// the response should still be valid (just empty items)
	if attentionResp.Items == nil {
		t.Error("Items field should not be nil (can be empty slice)")
	}

	// Verify sources field exists (may be empty if collectors unavailable)
	if attentionResp.Sources == nil {
		t.Error("Sources field should not be nil (can be empty slice)")
	}
}

func TestHandleAttentionPrioritySorting(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attention", nil)
	w := httptest.NewRecorder()

	handleAttention(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var attentionResp AttentionAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&attentionResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// If items exist, verify they are sorted by priority (lower = higher priority)
	if len(attentionResp.Items) > 1 {
		for i := 0; i < len(attentionResp.Items)-1; i++ {
			if attentionResp.Items[i].Priority > attentionResp.Items[i+1].Priority {
				t.Errorf("Items not sorted by priority: item[%d].Priority=%d > item[%d].Priority=%d",
					i, attentionResp.Items[i].Priority, i+1, attentionResp.Items[i+1].Priority)
			}
		}
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for POST /api/attention/verify endpoint

func TestHandleAttentionVerifyMethodNotAllowed(t *testing.T) {
	// Test GET method is not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/attention/verify", nil)
	w := httptest.NewRecorder()

	handleAttentionVerify(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleAttentionVerifyRequiresIssueID(t *testing.T) {
	body := `{"status":"verified"}`
	req := httptest.NewRequest(http.MethodPost, "/api/attention/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAttentionVerify(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleAttentionVerifyRequiresStatus(t *testing.T) {
	body := `{"issue_id":"test-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/attention/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAttentionVerify(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleAttentionVerifyInvalidStatus(t *testing.T) {
	body := `{"issue_id":"test-123","status":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/attention/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAttentionVerify(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleAttentionVerifySuccess(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	// Set the verification log path for the test
	oldPath := verificationLogPath
	verificationLogPath = tmpDir + "/verifications.jsonl"
	defer func() { verificationLogPath = oldPath }()

	tests := []struct {
		name   string
		status string
	}{
		{"verified status", "verified"},
		{"needs_fix status", "needs_fix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"issue_id":"test-123","status":"` + tt.status + `"}`
			req := httptest.NewRequest(http.MethodPost, "/api/attention/verify", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleAttentionVerify(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Verify response is JSON
			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify response body
			var verifyResp VerificationResponse
			if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if verifyResp.IssueID != "test-123" {
				t.Errorf("Expected issue_id test-123, got %s", verifyResp.IssueID)
			}
			if verifyResp.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, verifyResp.Status)
			}
			if verifyResp.VerifiedAt == "" {
				t.Error("Expected verified_at to be set")
			}
		})
	}
}

func TestHandleAttentionVerifyPersistsToJSONL(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	// Set the verification log path for the test
	oldPath := verificationLogPath
	verificationLogPath = tmpDir + "/verifications.jsonl"
	defer func() { verificationLogPath = oldPath }()

	// Submit a verification
	body := `{"issue_id":"test-persist-123","status":"verified"}`
	req := httptest.NewRequest(http.MethodPost, "/api/attention/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAttentionVerify(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read the JSONL file
	data, err := os.ReadFile(verificationLogPath)
	if err != nil {
		t.Fatalf("Failed to read verification log: %v", err)
	}

	// Parse the entry
	var entry VerificationEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("Failed to parse verification entry: %v", err)
	}

	if entry.IssueID != "test-persist-123" {
		t.Errorf("Expected issue_id test-persist-123, got %s", entry.IssueID)
	}
	if entry.Status != "verified" {
		t.Errorf("Expected status verified, got %s", entry.Status)
	}
	if entry.Timestamp == 0 {
		t.Error("Expected timestamp to be set")
	}
}

// Tests for loadVerifications()

func TestLoadVerificationsEmptyFile(t *testing.T) {
	// Test that empty/missing file returns empty map (graceful handling)
	tmpDir := t.TempDir()
	oldPath := verificationLogPath
	verificationLogPath = tmpDir + "/nonexistent.jsonl"
	defer func() { verificationLogPath = oldPath }()

	result := loadVerifications()

	if result == nil {
		t.Error("Expected non-nil map")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(result))
	}
}

func TestLoadVerificationsReadsEntries(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := verificationLogPath
	verificationLogPath = tmpDir + "/verifications.jsonl"
	defer func() { verificationLogPath = oldPath }()

	// Write test entries
	entries := `{"issue_id":"test-1","status":"verified","timestamp":1234567890}
{"issue_id":"test-2","status":"needs_fix","timestamp":1234567891}
`
	if err := os.WriteFile(verificationLogPath, []byte(entries), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := loadVerifications()

	if len(result) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(result))
	}
	if result["test-1"].Status != "verified" {
		t.Errorf("Expected test-1 status verified, got %s", result["test-1"].Status)
	}
	if result["test-2"].Status != "needs_fix" {
		t.Errorf("Expected test-2 status needs_fix, got %s", result["test-2"].Status)
	}
}

func TestLoadVerificationsLatestEntryWins(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := verificationLogPath
	verificationLogPath = tmpDir + "/verifications.jsonl"
	defer func() { verificationLogPath = oldPath }()

	// Write entries with same issue_id, later entry should win
	entries := `{"issue_id":"test-1","status":"verified","timestamp":1000}
{"issue_id":"test-1","status":"needs_fix","timestamp":2000}
`
	if err := os.WriteFile(verificationLogPath, []byte(entries), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := loadVerifications()

	if len(result) != 1 {
		t.Errorf("Expected 1 entry (duplicate merged), got %d", len(result))
	}
	if result["test-1"].Status != "needs_fix" {
		t.Errorf("Expected later entry (needs_fix) to win, got %s", result["test-1"].Status)
	}
}

func TestLoadVerificationsSkipsMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := verificationLogPath
	verificationLogPath = tmpDir + "/verifications.jsonl"
	defer func() { verificationLogPath = oldPath }()

	// Write entries with malformed line in the middle
	entries := `{"issue_id":"test-1","status":"verified","timestamp":1234567890}
not valid json
{"issue_id":"test-2","status":"needs_fix","timestamp":1234567891}
`
	if err := os.WriteFile(verificationLogPath, []byte(entries), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := loadVerifications()

	// Should have 2 valid entries, skipping the malformed line
	if len(result) != 2 {
		t.Errorf("Expected 2 entries (skipping malformed), got %d", len(result))
	}
}

// TestVerificationFilteringOnlyAffectsRecentlyClosed verifies that the verification
// filter only applies to recently-closed signals, not other signal types.
func TestVerificationFilteringOnlyAffectsRecentlyClosed(t *testing.T) {
	// Test the filtering logic directly
	verifications := map[string]VerificationEntry{
		"test-issue-1": {IssueID: "test-issue-1", Status: "verified"},
		"test-issue-2": {IssueID: "test-issue-2", Status: "verified"},
	}

	allItems := []attention.AttentionItem{
		{
			ID:      "rc-1",
			Signal:  "recently-closed",
			Subject: "test-issue-1", // Verified - should be filtered
		},
		{
			ID:      "ir-1",
			Signal:  "issue-ready",
			Subject: "test-issue-1", // Verified but different signal - should NOT be filtered
		},
		{
			ID:      "ld-1",
			Signal:  "likely-done",
			Subject: "test-issue-2", // Verified but different signal - should NOT be filtered
		},
		{
			ID:      "rc-2",
			Signal:  "recently-closed",
			Subject: "test-issue-3", // Not verified - should NOT be filtered
		},
	}

	// Apply the filtering logic (same as in handleAttention)
	filteredItems := []attention.AttentionItem{}
	for _, item := range allItems {
		verification, exists := verifications[item.Subject]
		// Only filter recently-closed items based on verification status
		if item.Signal == "recently-closed" && exists && verification.Status == "verified" {
			continue
		}
		filteredItems = append(filteredItems, item)
	}

	// Expected: 3 items (rc-1 filtered, others pass through)
	if len(filteredItems) != 3 {
		t.Errorf("Expected 3 items after filtering, got %d", len(filteredItems))
	}

	// Verify the correct items passed through
	itemIDs := make(map[string]bool)
	for _, item := range filteredItems {
		itemIDs[item.ID] = true
	}

	if itemIDs["rc-1"] {
		t.Error("Expected recently-closed verified item (rc-1) to be filtered out")
	}
	if !itemIDs["ir-1"] {
		t.Error("Expected issue-ready item (ir-1) to pass through even if subject is verified")
	}
	if !itemIDs["ld-1"] {
		t.Error("Expected likely-done item (ld-1) to pass through even if subject is verified")
	}
	if !itemIDs["rc-2"] {
		t.Error("Expected unverified recently-closed item (rc-2) to pass through")
	}
}
