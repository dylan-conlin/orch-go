package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHotspot(t *testing.T) {
	// Create a request to the hotspot endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/hotspot", nil)
	rec := httptest.NewRecorder()

	// Call the handler
	handleHotspot(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check Content-Type header
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Parse the response body
	var report HotspotReport
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify required fields are present
	if report.GeneratedAt == "" {
		t.Error("expected GeneratedAt to be set")
	}

	if report.AnalysisPeriod == "" {
		t.Error("expected AnalysisPeriod to be set")
	}

	// FixThreshold should be the default (5)
	if report.FixThreshold != 5 {
		t.Errorf("expected FixThreshold 5, got %d", report.FixThreshold)
	}

	// InvThreshold should be the default (3)
	if report.InvThreshold != 3 {
		t.Errorf("expected InvThreshold 3, got %d", report.InvThreshold)
	}

	// Hotspots slice should not be nil
	if report.Hotspots == nil {
		t.Error("expected Hotspots to be non-nil slice")
	}
}

func TestHandleHotspotMethodNotAllowed(t *testing.T) {
	// Test that POST is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/hotspot", nil)
	rec := httptest.NewRecorder()

	handleHotspot(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
