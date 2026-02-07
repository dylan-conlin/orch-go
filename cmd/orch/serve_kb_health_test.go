package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleKBHealthMethodNotAllowed verifies only GET is allowed
func TestHandleKBHealthMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/kb-health", nil)
	rec := httptest.NewRecorder()

	newTestServer().handleKBHealth(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

// TestKBHealthResponseJSONFormat verifies response structure
func TestKBHealthResponseJSONFormat(t *testing.T) {
	resp := KBHealthResponse{
		Synthesis: KBHealthCategory{
			Count: 3,
			Items: []map[string]interface{}{
				{"topic": "test", "count": 3},
			},
		},
		Promote: KBHealthCategory{
			Count: 1,
			Items: []map[string]interface{}{
				{"id": "kn-123", "type": "decision"},
			},
		},
		Stale: KBHealthCategory{
			Count: 0,
			Items: []map[string]interface{}{},
		},
		InvestigationPromotion: KBHealthCategory{
			Count: 0,
			Items: []map[string]interface{}{},
		},
		Total:       4,
		LastUpdated: "2026-01-28T10:30:00Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// Verify snake_case JSON keys
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	requiredKeys := []string{"synthesis", "promote", "stale", "investigation_promotion", "total", "last_updated"}
	for _, key := range requiredKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("missing required key: %s", key)
		}
	}
}

// TestKBHealthCategoryStructure verifies category structure
func TestKBHealthCategoryStructure(t *testing.T) {
	cat := KBHealthCategory{
		Count: 2,
		Items: []map[string]interface{}{
			{"field": "value1"},
			{"field": "value2"},
		},
	}

	data, err := json.Marshal(cat)
	if err != nil {
		t.Fatalf("failed to marshal category: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if count, ok := result["count"].(float64); !ok || int(count) != 2 {
		t.Errorf("expected count=2, got %v", result["count"])
	}

	if items, ok := result["items"].([]interface{}); !ok || len(items) != 2 {
		t.Errorf("expected 2 items, got %v", result["items"])
	}
}

// TestHandleKBHealthGracefulDegradation verifies empty response when kb unavailable
func TestHandleKBHealthGracefulDegradation(t *testing.T) {
	// This test verifies the handler doesn't crash when kb CLI is unavailable
	// The actual graceful degradation is tested by fetchKBReflect returning empty slices

	req := httptest.NewRequest("GET", "/api/kb-health?project=/nonexistent", nil)
	rec := httptest.NewRecorder()

	newTestServer().handleKBHealth(rec, req)

	// Should return 200 with valid JSON even if kb CLI fails
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp KBHealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify structure is valid even with empty data
	if resp.Total < 0 {
		t.Errorf("expected non-negative total, got %d", resp.Total)
	}
}

// TestKBHealthCacheTTL verifies cache behavior
func TestKBHealthCacheTTL(t *testing.T) {
	cache := newKBHealthCache()

	if cache.ttl.Minutes() != 5 {
		t.Errorf("expected 5 minute TTL, got %v", cache.ttl)
	}
}

// TestFetchKBReflectEmptyOnError verifies graceful degradation
func TestFetchKBReflectEmptyOnError(t *testing.T) {
	// Test with invalid project dir - should return empty slice, not error
	items, err := fetchKBReflect("/nonexistent/path/that/does/not/exist", "synthesis")

	// Should gracefully degrade to empty slice
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if items == nil {
		t.Error("expected non-nil items slice")
	}

	if len(items) != 0 {
		t.Errorf("expected empty slice, got %d items", len(items))
	}
}
