package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDaemonResume_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/daemon/resume", nil)
	w := httptest.NewRecorder()

	handleDaemonResume(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleDaemonResume_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/daemon/resume", nil)
	w := httptest.NewRecorder()

	handleDaemonResume(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp DaemonResumeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success=true, got false: %s", resp.Error)
	}
}

func TestHandleCloseIssue_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/issues/close", nil)
	w := httptest.NewRecorder()

	handleCloseIssue(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleCloseIssue_MissingBeadsID(t *testing.T) {
	body, _ := json.Marshal(CloseIssueRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/issues/close", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCloseIssue(w, req)

	var resp CloseIssueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success=false for missing beads_id")
	}
	if resp.Error == "" {
		t.Error("Expected error message for missing beads_id")
	}
}

func TestHandleCloseIssue_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/issues/close", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	handleCloseIssue(w, req)

	var resp CloseIssueResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success=false for invalid JSON")
	}
}
