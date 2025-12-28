package sessions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultStoragePath(t *testing.T) {
	path := DefaultStoragePath()
	if path == "" {
		t.Error("DefaultStoragePath returned empty string")
	}

	// Should end with opencode/storage
	if !contains(path, "opencode") || !contains(path, "storage") {
		t.Errorf("unexpected storage path: %s", path)
	}
}

func TestExtractSnippet(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		start      int
		end        int
		contextLen int
		wantPrefix string
		wantSuffix string
	}{
		{
			name:       "basic snippet",
			text:       "the quick brown fox jumps over the lazy dog",
			start:      16, // "fox"
			end:        19,
			contextLen: 10,
			wantPrefix: "",
			wantSuffix: "",
		},
		{
			name:       "snippet at start",
			text:       "hello world this is a test",
			start:      0,
			end:        5,
			contextLen: 10,
			wantPrefix: "",
			wantSuffix: "",
		},
		{
			name:       "snippet at end",
			text:       "this is a test hello",
			start:      15,
			end:        20,
			contextLen: 10,
			wantPrefix: "...",
			wantSuffix: "",
		},
		{
			name:       "snippet in middle",
			text:       "start middle text here end of string",
			start:      6,  // "middle"
			end:        12,
			contextLen: 5,
			wantPrefix: "", // Context of 5 reaches back to "start" which is word-aligned
			wantSuffix: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSnippet(tt.text, tt.start, tt.end, tt.contextLen)
			
			if tt.wantPrefix != "" && !hasPrefix(result, tt.wantPrefix) {
				t.Errorf("expected prefix %q in %q", tt.wantPrefix, result)
			}
			if tt.wantSuffix != "" && !hasSuffix(result, tt.wantSuffix) {
				t.Errorf("expected suffix %q in %q", tt.wantSuffix, result)
			}
		})
	}
}

func TestNewStore(t *testing.T) {
	// Test with default path
	store := NewStore("", nil)
	if store.storagePath != DefaultStoragePath() {
		t.Errorf("expected default path, got %s", store.storagePath)
	}

	// Test with custom path
	customPath := "/custom/path"
	store = NewStore(customPath, nil)
	if store.storagePath != customPath {
		t.Errorf("expected %s, got %s", customPath, store.storagePath)
	}
}

func TestStoreList_NoStorage(t *testing.T) {
	// Use a path that doesn't exist
	store := NewStore("/nonexistent/path", nil)
	sessions, err := store.List(ListOptions{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected empty list, got %d sessions", len(sessions))
	}
}

func TestStoreList_WithMockData(t *testing.T) {
	// Create temporary storage structure
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session", "project123")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock session files
	sessions := []struct {
		id      string
		title   string
		created int64
		updated int64
	}{
		{"ses_001", "First session", time.Now().Add(-2 * time.Hour).UnixMilli(), time.Now().Add(-1 * time.Hour).UnixMilli()},
		{"ses_002", "Second session", time.Now().Add(-1 * time.Hour).UnixMilli(), time.Now().UnixMilli()},
	}

	for _, s := range sessions {
		data := []byte(`{
			"id": "` + s.id + `",
			"version": "local",
			"projectID": "project123",
			"directory": "/test/project",
			"title": "` + s.title + `",
			"time": {"created": ` + itoa(s.created) + `, "updated": ` + itoa(s.updated) + `},
			"summary": {"additions": 10, "deletions": 5, "files": 2}
		}`)
		if err := os.WriteFile(filepath.Join(sessionDir, s.id+".json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	store := NewStore(tmpDir, nil)
	result, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(result))
	}

	// Should be sorted by updated time (most recent first)
	if len(result) >= 2 && result[0].ID != "ses_002" {
		t.Errorf("expected ses_002 first (most recent), got %s", result[0].ID)
	}
}

func TestStoreList_WithFilters(t *testing.T) {
	// Create temporary storage structure
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session", "project123")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	sessions := []struct {
		id        string
		directory string
		created   int64
	}{
		{"ses_001", "/project/a", now.Add(-48 * time.Hour).UnixMilli()},
		{"ses_002", "/project/b", now.Add(-24 * time.Hour).UnixMilli()},
		{"ses_003", "/project/a", now.UnixMilli()},
	}

	for _, s := range sessions {
		data := []byte(`{
			"id": "` + s.id + `",
			"version": "local",
			"projectID": "project123",
			"directory": "` + s.directory + `",
			"title": "Test",
			"time": {"created": ` + itoa(s.created) + `, "updated": ` + itoa(s.created) + `},
			"summary": {}
		}`)
		if err := os.WriteFile(filepath.Join(sessionDir, s.id+".json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	store := NewStore(tmpDir, nil)

	// Test directory filter
	t.Run("filter by directory", func(t *testing.T) {
		result, err := store.List(ListOptions{Directory: "/project/a"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 sessions for /project/a, got %d", len(result))
		}
	})

	// Test limit
	t.Run("limit results", func(t *testing.T) {
		result, err := store.List(ListOptions{Limit: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 session, got %d", len(result))
		}
	})

	// Test date filter
	t.Run("filter by date", func(t *testing.T) {
		after := now.Add(-30 * time.Hour)
		result, err := store.List(ListOptions{After: &after})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 sessions after cutoff, got %d", len(result))
		}
	})
}

func TestDiskSessionParsing(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session", "project123")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test parsing various session formats
	data := []byte(`{
		"id": "ses_test123",
		"version": "local",
		"projectID": "project123",
		"directory": "/Users/test/project",
		"title": "Test session with special chars: éàü",
		"time": {"created": 1735300000000, "updated": 1735310000000},
		"summary": {"additions": 100, "deletions": 50, "files": 5}
	}`)
	
	if err := os.WriteFile(filepath.Join(sessionDir, "ses_test123.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tmpDir, nil)
	result, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 session, got %d", len(result))
	}

	s := result[0]
	if s.ID != "ses_test123" {
		t.Errorf("wrong ID: %s", s.ID)
	}
	if s.Directory != "/Users/test/project" {
		t.Errorf("wrong directory: %s", s.Directory)
	}
	if s.Summary.Additions != 100 || s.Summary.Deletions != 50 || s.Summary.Files != 5 {
		t.Errorf("wrong summary: %+v", s.Summary)
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	
	var buf [20]byte
	i := len(buf)
	negative := n < 0
	if negative {
		n = -n
	}
	
	for n > 0 {
		i--
		buf[i] = byte(n%10) + '0'
		n /= 10
	}
	
	if negative {
		i--
		buf[i] = '-'
	}
	
	return string(buf[i:])
}
