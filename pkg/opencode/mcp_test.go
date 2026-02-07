package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" || r.Method != "GET" {
			t.Errorf("Expected GET /mcp, got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"glass":{"status":"connected"},"playwright":{"status":"disabled"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	status, err := client.MCPStatus()
	if err != nil {
		t.Fatalf("MCPStatus() error = %v", err)
	}

	if len(status) != 2 {
		t.Fatalf("Expected 2 servers, got %d", len(status))
	}
	if status["glass"].Status != "connected" {
		t.Errorf("glass status = %v, want connected", status["glass"].Status)
	}
	if status["playwright"].Status != "disabled" {
		t.Errorf("playwright status = %v, want disabled", status["playwright"].Status)
	}
}

func TestMCPConnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp/playwright/connect" || r.Method != "POST" {
			t.Errorf("Expected POST /mcp/playwright/connect, got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`true`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.MCPConnect("playwright")
	if err != nil {
		t.Fatalf("MCPConnect() error = %v", err)
	}
}

func TestMCPConnectError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server not configured"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.MCPConnect("nonexistent")
	if err == nil {
		t.Fatal("MCPConnect() expected error for server error response")
	}
}

func TestMCPDisconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp/glass/disconnect" || r.Method != "POST" {
			t.Errorf("Expected POST /mcp/glass/disconnect, got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`true`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.MCPDisconnect("glass")
	if err != nil {
		t.Fatalf("MCPDisconnect() error = %v", err)
	}
}
