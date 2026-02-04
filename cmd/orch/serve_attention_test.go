package main

import (
	"encoding/json"
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
