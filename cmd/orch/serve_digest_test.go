package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

func TestHandleDigest_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, ".orch", "digest"), 0755)

	req := httptest.NewRequest(http.MethodGet, "/api/digest", nil)
	w := httptest.NewRecorder()

	handleDigest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp DigestAPIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Products) != 0 {
		t.Errorf("expected 0 products, got %d", len(resp.Products))
	}
	if resp.UnreadCount != 0 {
		t.Errorf("expected 0 unread, got %d", resp.UnreadCount)
	}
}

func TestHandleDigest_WithProducts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	digestPath := filepath.Join(tmpDir, ".orch", "digest")
	os.MkdirAll(digestPath, 0755)

	product := `{
		"id": "test-product-1",
		"type": "thread_progression",
		"title": "Test Thread Update",
		"summary": "A test thread was updated",
		"significance": "high",
		"source": {"artifact_type": "thread", "path": ".kb/threads/test.md"},
		"state": "new",
		"created_at": "2026-03-14T10:00:00Z"
	}`
	os.WriteFile(filepath.Join(digestPath, "test-product-1.json"), []byte(product), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/digest", nil)
	w := httptest.NewRecorder()

	handleDigest(w, req)

	var resp DigestAPIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(resp.Products))
	}
	if resp.Products[0].ID != "test-product-1" {
		t.Errorf("expected id test-product-1, got %s", resp.Products[0].ID)
	}
	if resp.UnreadCount != 1 {
		t.Errorf("expected 1 unread, got %d", resp.UnreadCount)
	}
}

func TestHandleDigest_FilterByState(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	digestPath := filepath.Join(tmpDir, ".orch", "digest")
	os.MkdirAll(digestPath, 0755)

	newProduct := `{"id": "new-1", "type": "thread_progression", "state": "new", "created_at": "2026-03-14T10:00:00Z", "source": {"artifact_type": "thread", "path": "a"}}`
	readProduct := `{"id": "read-1", "type": "model_update", "state": "read", "created_at": "2026-03-14T09:00:00Z", "source": {"artifact_type": "model", "path": "b"}}`
	os.WriteFile(filepath.Join(digestPath, "new-1.json"), []byte(newProduct), 0644)
	os.WriteFile(filepath.Join(digestPath, "read-1.json"), []byte(readProduct), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/digest?state=new", nil)
	w := httptest.NewRecorder()

	handleDigest(w, req)

	var resp DigestAPIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Products) != 1 {
		t.Fatalf("expected 1 product with state=new, got %d", len(resp.Products))
	}
	if resp.Products[0].ID != "new-1" {
		t.Errorf("expected new-1, got %s", resp.Products[0].ID)
	}
}

func TestHandleDigestStats(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	digestPath := filepath.Join(tmpDir, ".orch", "digest")
	os.MkdirAll(digestPath, 0755)

	products := map[string]string{
		"p1.json": `{"id": "1", "type": "thread_progression", "state": "new", "created_at": "2026-03-14T10:00:00Z", "source": {"artifact_type": "thread", "path": "a"}}`,
		"p2.json": `{"id": "2", "type": "thread_progression", "state": "read", "created_at": "2026-03-14T09:00:00Z", "source": {"artifact_type": "thread", "path": "b"}}`,
		"p3.json": `{"id": "3", "type": "model_update", "state": "starred", "created_at": "2026-03-14T08:00:00Z", "source": {"artifact_type": "model", "path": "c"}}`,
	}
	for name, p := range products {
		os.WriteFile(filepath.Join(digestPath, name), []byte(p), 0644)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/digest/stats", nil)
	w := httptest.NewRecorder()

	handleDigestStats(w, req)

	var resp daemon.DigestStatsResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Unread != 1 {
		t.Errorf("expected 1 unread, got %d", resp.Unread)
	}
	if resp.Read != 1 {
		t.Errorf("expected 1 read, got %d", resp.Read)
	}
	if resp.Starred != 1 {
		t.Errorf("expected 1 starred, got %d", resp.Starred)
	}
}

func TestHandleDigestUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	digestPath := filepath.Join(tmpDir, ".orch", "digest")
	os.MkdirAll(digestPath, 0755)

	product := `{"id": "update-test", "type": "thread_progression", "state": "new", "created_at": "2026-03-14T10:00:00Z", "source": {"artifact_type": "thread", "path": "a"}}`
	os.WriteFile(filepath.Join(digestPath, "update-test.json"), []byte(product), 0644)

	body := `{"state": "read"}`
	req := httptest.NewRequest(http.MethodPost, "/api/digest/update?id=update-test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleDigestUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	data, _ := os.ReadFile(filepath.Join(digestPath, "update-test.json"))
	var p daemon.DigestProduct
	json.Unmarshal(data, &p)

	if p.State != "read" {
		t.Errorf("expected state=read, got %s", p.State)
	}
	if p.ReadAt.IsZero() {
		t.Error("expected read_at to be set")
	}
}

func TestHandleDigest_NoDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	req := httptest.NewRequest(http.MethodGet, "/api/digest", nil)
	w := httptest.NewRecorder()

	handleDigest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even with no directory, got %d", w.Code)
	}

	var resp DigestAPIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Products) != 0 {
		t.Errorf("expected 0 products, got %d", len(resp.Products))
	}
}
