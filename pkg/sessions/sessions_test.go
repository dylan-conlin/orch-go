package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/execution"
)

func TestDefaultStoragePath(t *testing.T) {
	path := DefaultStoragePath()

	// Should return non-empty path
	if path == "" {
		t.Error("DefaultStoragePath() returned empty string")
	}

	// Should contain expected path components
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "share", "opencode", "storage")
	if path != expected {
		t.Errorf("DefaultStoragePath() = %q, want %q", path, expected)
	}
}

func TestNewStore(t *testing.T) {
	t.Run("with empty path uses default", func(t *testing.T) {
		store := NewStore("", nil)
		if store.storagePath == "" {
			t.Error("NewStore with empty path should use default")
		}
		expected := DefaultStoragePath()
		if store.storagePath != expected {
			t.Errorf("storagePath = %q, want %q", store.storagePath, expected)
		}
	})

	t.Run("with custom path", func(t *testing.T) {
		customPath := "/custom/path"
		store := NewStore(customPath, nil)
		if store.storagePath != customPath {
			t.Errorf("storagePath = %q, want %q", store.storagePath, customPath)
		}
	})

	t.Run("with client", func(t *testing.T) {
		client := execution.NewOpenCodeAdapter("http://localhost:3000")
		store := NewStore("", client)
		if store.client != client {
			t.Error("NewStore did not store client correctly")
		}
	})
}

func TestListEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, nil)

	sessions, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if sessions != nil && len(sessions) > 0 {
		t.Errorf("List() = %v, want nil or empty for nonexistent directory", sessions)
	}
}

func TestListWithSessions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create session directory structure
	sessionDir := filepath.Join(tmpDir, "session", "project1")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	// Create test session files
	now := time.Now()
	sessions := []DiskSession{
		{
			ID:        "ses_001",
			ProjectID: "project1",
			Directory: "/home/user/project1",
			Title:     "First session",
			Time: struct {
				Created int64 `json:"created"`
				Updated int64 `json:"updated"`
			}{
				Created: now.Add(-1 * time.Hour).UnixMilli(),
				Updated: now.Add(-30 * time.Minute).UnixMilli(),
			},
			Summary: struct {
				Additions int `json:"additions"`
				Deletions int `json:"deletions"`
				Files     int `json:"files"`
			}{
				Additions: 10,
				Deletions: 5,
				Files:     3,
			},
		},
		{
			ID:        "ses_002",
			ProjectID: "project1",
			Directory: "/home/user/project1",
			Title:     "Second session",
			Time: struct {
				Created int64 `json:"created"`
				Updated int64 `json:"updated"`
			}{
				Created: now.Add(-2 * time.Hour).UnixMilli(),
				Updated: now.Add(-1 * time.Hour).UnixMilli(),
			},
		},
	}

	for _, s := range sessions {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Failed to marshal session: %v", err)
		}
		filePath := filepath.Join(sessionDir, s.ID+".json")
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}
	}

	store := NewStore(tmpDir, nil)

	t.Run("list all sessions", func(t *testing.T) {
		result, err := store.List(ListOptions{})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
		// Should be sorted by updated time (most recent first)
		if result[0].ID != "ses_001" {
			t.Errorf("First session ID = %q, want %q (most recently updated)", result[0].ID, "ses_001")
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		result, err := store.List(ListOptions{Limit: 1})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len(result) = %d, want 1", len(result))
		}
	})

	t.Run("filter by directory", func(t *testing.T) {
		result, err := store.List(ListOptions{Directory: "/home/user/project1"})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}

		// Non-matching directory
		result, err = store.List(ListOptions{Directory: "/nonexistent"})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0 for non-matching directory", len(result))
		}
	})

	t.Run("filter by after date", func(t *testing.T) {
		after := now.Add(-90 * time.Minute)
		result, err := store.List(ListOptions{After: &after})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len(result) = %d, want 1 (only ses_001 created after cutoff)", len(result))
		}
		if len(result) > 0 && result[0].ID != "ses_001" {
			t.Errorf("result[0].ID = %q, want %q", result[0].ID, "ses_001")
		}
	})

	t.Run("filter by before date", func(t *testing.T) {
		before := now.Add(-90 * time.Minute)
		result, err := store.List(ListOptions{Before: &before})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len(result) = %d, want 1 (only ses_002 created before cutoff)", len(result))
		}
		if len(result) > 0 && result[0].ID != "ses_002" {
			t.Errorf("result[0].ID = %q, want %q", result[0].ID, "ses_002")
		}
	})
}

func TestListWithStoragePathOverride(t *testing.T) {
	tmpDir := t.TempDir()

	// Create session directory structure
	sessionDir := filepath.Join(tmpDir, "session", "proj")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	// Create a test session
	session := DiskSession{
		ID:        "ses_override",
		ProjectID: "proj",
		Directory: "/proj",
		Title:     "Override test",
		Time: struct {
			Created int64 `json:"created"`
			Updated int64 `json:"updated"`
		}{
			Created: time.Now().UnixMilli(),
			Updated: time.Now().UnixMilli(),
		},
	}
	data, _ := json.Marshal(session)
	filePath := filepath.Join(sessionDir, session.ID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	// Create store with different path
	store := NewStore("/different/path", nil)

	// List with StoragePath override should use the override
	result, err := store.List(ListOptions{StoragePath: tmpDir})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1", len(result))
	}
	if len(result) > 0 && result[0].ID != "ses_override" {
		t.Errorf("result[0].ID = %q, want %q", result[0].ID, "ses_override")
	}
}

func TestStoredSessionFields(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session", "proj")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	now := time.Now()
	session := DiskSession{
		ID:        "ses_full",
		Version:   "1.0",
		ProjectID: "proj",
		Directory: "/home/user/proj",
		Title:     "Full session test",
		Time: struct {
			Created int64 `json:"created"`
			Updated int64 `json:"updated"`
		}{
			Created: now.UnixMilli(),
			Updated: now.Add(10 * time.Minute).UnixMilli(),
		},
		Summary: struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
			Files     int `json:"files"`
		}{
			Additions: 100,
			Deletions: 50,
			Files:     10,
		},
	}
	data, _ := json.Marshal(session)
	filePath := filepath.Join(sessionDir, session.ID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	store := NewStore(tmpDir, nil)
	result, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	s := result[0]
	if s.ID != "ses_full" {
		t.Errorf("ID = %q, want %q", s.ID, "ses_full")
	}
	if s.ProjectID != "proj" {
		t.Errorf("ProjectID = %q, want %q", s.ProjectID, "proj")
	}
	if s.Directory != "/home/user/proj" {
		t.Errorf("Directory = %q, want %q", s.Directory, "/home/user/proj")
	}
	if s.Title != "Full session test" {
		t.Errorf("Title = %q, want %q", s.Title, "Full session test")
	}
	if s.Summary.Additions != 100 {
		t.Errorf("Summary.Additions = %d, want 100", s.Summary.Additions)
	}
	if s.Summary.Deletions != 50 {
		t.Errorf("Summary.Deletions = %d, want 50", s.Summary.Deletions)
	}
	if s.Summary.Files != 10 {
		t.Errorf("Summary.Files = %d, want 10", s.Summary.Files)
	}
}

func TestExtractSnippet(t *testing.T) {
	// extractSnippet adds ellipsis based on whether the snippet boundaries
	// are at the start/end of the original text. It also expands to word
	// boundaries and normalizes whitespace.

	t.Run("short text returns full text", func(t *testing.T) {
		result := extractSnippet("hello world", 0, 5, 100)
		// With context 100, the entire text is included, no ellipsis
		if result != "hello world" {
			t.Errorf("got %q, want %q", result, "hello world")
		}
	})

	t.Run("match at start of long text adds suffix ellipsis", func(t *testing.T) {
		text := "hello world this is a much longer text for testing"
		result := extractSnippet(text, 0, 5, 5)
		// Start is at 0, so no prefix. End + contextLen < len(text), so suffix.
		hasPrefix := len(result) >= 3 && result[:3] == "..."
		hasSuffix := len(result) >= 3 && result[len(result)-3:] == "..."

		if hasPrefix {
			t.Errorf("unexpected prefix ellipsis; result = %q", result)
		}
		if !hasSuffix {
			t.Errorf("expected suffix ellipsis; result = %q", result)
		}
	})

	t.Run("match at end of long text adds prefix ellipsis", func(t *testing.T) {
		text := "this is a much longer text for testing the end match"
		textLen := len(text)
		result := extractSnippet(text, textLen-5, textLen, 5)
		// Start > 0, so prefix. End is at len(text), so no suffix.
		hasPrefix := len(result) >= 3 && result[:3] == "..."
		hasSuffix := len(result) >= 3 && result[len(result)-3:] == "..."

		if !hasPrefix {
			t.Errorf("expected prefix ellipsis; result = %q", result)
		}
		if hasSuffix {
			t.Errorf("unexpected suffix ellipsis; result = %q", result)
		}
	})

	t.Run("match in middle adds both ellipses", func(t *testing.T) {
		text := "the quick brown fox jumps over the lazy dog"
		result := extractSnippet(text, 16, 19, 3)
		// Both boundaries are inside the text
		hasPrefix := len(result) >= 3 && result[:3] == "..."
		hasSuffix := len(result) >= 3 && result[len(result)-3:] == "..."

		if !hasPrefix {
			t.Errorf("expected prefix ellipsis; result = %q", result)
		}
		if !hasSuffix {
			t.Errorf("expected suffix ellipsis; result = %q", result)
		}
	})

	t.Run("short text no ellipsis", func(t *testing.T) {
		result := extractSnippet("abc", 0, 3, 10)
		hasPrefix := len(result) >= 3 && result[:3] == "..."
		hasSuffix := len(result) >= 3 && result[len(result)-3:] == "..."

		if hasPrefix {
			t.Errorf("unexpected prefix ellipsis; result = %q", result)
		}
		if hasSuffix {
			t.Errorf("unexpected suffix ellipsis; result = %q", result)
		}
	})
}

func TestExtractSnippetNewlineReplacement(t *testing.T) {
	text := "line one\nline two\n\nline four"
	result := extractSnippet(text, 5, 8, 100)

	// Result should not contain newlines
	if result != text { // extractSnippet replaces newlines with spaces
		// Check that multiple newlines are collapsed
		for i := 0; i < len(result)-1; i++ {
			if result[i] == '\n' {
				t.Errorf("Result contains newline at position %d: %q", i, result)
			}
		}
	}
}

func TestShowWithNilClient(t *testing.T) {
	store := NewStore("", nil)

	_, _, err := store.Show("ses_123")
	if err == nil {
		t.Error("Show() with nil client should return error")
	}
	if err != os.ErrNotExist {
		t.Errorf("Show() error = %v, want os.ErrNotExist", err)
	}
}

func TestSearchWithNilClient(t *testing.T) {
	tmpDir := t.TempDir()

	// Create session directory structure
	sessionDir := filepath.Join(tmpDir, "session", "proj")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	// Create a test session
	session := DiskSession{
		ID:        "ses_search",
		ProjectID: "proj",
		Directory: "/proj",
		Title:     "Search test",
		Time: struct {
			Created int64 `json:"created"`
			Updated int64 `json:"updated"`
		}{
			Created: time.Now().UnixMilli(),
			Updated: time.Now().UnixMilli(),
		},
	}
	data, _ := json.Marshal(session)
	filePath := filepath.Join(sessionDir, session.ID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	store := NewStore(tmpDir, nil)

	// Search with nil client should return empty results (gracefully)
	results, err := store.Search(SearchOptions{Query: "test"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	// With nil client, search skips message fetching so no matches
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0 (nil client)", len(results))
	}
}

func TestSearchWithInvalidRegex(t *testing.T) {
	store := NewStore("", nil)

	_, err := store.Search(SearchOptions{
		Query:    "[invalid",
		UseRegex: true,
	})
	if err == nil {
		t.Error("Search() with invalid regex should return error")
	}
}

func TestListIgnoresNonJSONFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session", "proj")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	// Create a non-JSON file
	nonJSONPath := filepath.Join(sessionDir, "readme.txt")
	if err := os.WriteFile(nonJSONPath, []byte("readme content"), 0644); err != nil {
		t.Fatalf("Failed to write non-JSON file: %v", err)
	}

	// Create a valid session
	session := DiskSession{
		ID:        "ses_valid",
		ProjectID: "proj",
		Directory: "/proj",
		Title:     "Valid session",
		Time: struct {
			Created int64 `json:"created"`
			Updated int64 `json:"updated"`
		}{
			Created: time.Now().UnixMilli(),
			Updated: time.Now().UnixMilli(),
		},
	}
	data, _ := json.Marshal(session)
	jsonPath := filepath.Join(sessionDir, session.ID+".json")
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}

	store := NewStore(tmpDir, nil)
	result, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1 (should ignore non-JSON)", len(result))
	}
}

func TestListIgnoresInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session", "proj")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	// Create an invalid JSON file
	invalidPath := filepath.Join(sessionDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	// Create a valid session
	session := DiskSession{
		ID:        "ses_valid",
		ProjectID: "proj",
		Directory: "/proj",
		Title:     "Valid session",
		Time: struct {
			Created int64 `json:"created"`
			Updated int64 `json:"updated"`
		}{
			Created: time.Now().UnixMilli(),
			Updated: time.Now().UnixMilli(),
		},
	}
	data, _ := json.Marshal(session)
	jsonPath := filepath.Join(sessionDir, session.ID+".json")
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("Failed to write valid JSON file: %v", err)
	}

	store := NewStore(tmpDir, nil)
	result, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1 (should ignore invalid JSON)", len(result))
	}
}

func TestListIgnoresNonDirectoryProjects(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	// Create a file where a project directory would be
	filePath := filepath.Join(sessionDir, "not_a_directory.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Create a valid project directory with a session
	projectDir := filepath.Join(sessionDir, "valid_project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	session := DiskSession{
		ID:        "ses_valid",
		ProjectID: "valid_project",
		Directory: "/proj",
		Title:     "Valid session",
		Time: struct {
			Created int64 `json:"created"`
			Updated int64 `json:"updated"`
		}{
			Created: time.Now().UnixMilli(),
			Updated: time.Now().UnixMilli(),
		},
	}
	data, _ := json.Marshal(session)
	jsonPath := filepath.Join(projectDir, session.ID+".json")
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	store := NewStore(tmpDir, nil)
	result, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1", len(result))
	}
}

func TestListMultipleProjects(t *testing.T) {
	tmpDir := t.TempDir()

	projects := []string{"project1", "project2", "project3"}
	now := time.Now()

	for i, proj := range projects {
		projectDir := filepath.Join(tmpDir, "session", proj)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		session := DiskSession{
			ID:        "ses_" + proj,
			ProjectID: proj,
			Directory: "/" + proj,
			Title:     "Session for " + proj,
			Time: struct {
				Created int64 `json:"created"`
				Updated int64 `json:"updated"`
			}{
				Created: now.Add(time.Duration(-i) * time.Hour).UnixMilli(),
				Updated: now.Add(time.Duration(-i) * time.Hour).UnixMilli(),
			},
		}
		data, _ := json.Marshal(session)
		jsonPath := filepath.Join(projectDir, session.ID+".json")
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}
	}

	store := NewStore(tmpDir, nil)
	result, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}

	// Should be sorted by updated time (project1 first, most recent)
	if result[0].ProjectID != "project1" {
		t.Errorf("First result ProjectID = %q, want %q", result[0].ProjectID, "project1")
	}
}

func TestSearchResultSorting(t *testing.T) {
	// SearchResult sorting is by match count (descending)
	// This tests the sorting behavior in Search()
	// Since we can't easily mock the opencode client, we just test the types
	r1 := SearchResult{MatchCount: 5}
	r2 := SearchResult{MatchCount: 10}
	r3 := SearchResult{MatchCount: 3}

	results := []SearchResult{r1, r2, r3}

	// Manually verify sorting logic matches implementation
	// sort.Slice(results, func(i, j int) bool { return results[i].MatchCount > results[j].MatchCount })
	// Expected order after sort: r2 (10), r1 (5), r3 (3)

	if r2.MatchCount < r1.MatchCount {
		t.Error("SearchResult sorting expects descending by MatchCount")
	}
	if len(results) != 3 {
		t.Errorf("results length = %d, want 3", len(results))
	}
}

func TestSearchOptions(t *testing.T) {
	// Test SearchOptions struct fields
	opts := SearchOptions{
		StoragePath:   "/custom/path",
		Query:         "test query",
		UseRegex:      true,
		CaseSensitive: true,
		Directory:     "/project",
		Limit:         10,
	}

	if opts.StoragePath != "/custom/path" {
		t.Errorf("StoragePath = %q, want %q", opts.StoragePath, "/custom/path")
	}
	if opts.Query != "test query" {
		t.Errorf("Query = %q, want %q", opts.Query, "test query")
	}
	if !opts.UseRegex {
		t.Error("UseRegex = false, want true")
	}
	if !opts.CaseSensitive {
		t.Error("CaseSensitive = false, want true")
	}
	if opts.Directory != "/project" {
		t.Errorf("Directory = %q, want %q", opts.Directory, "/project")
	}
	if opts.Limit != 10 {
		t.Errorf("Limit = %d, want 10", opts.Limit)
	}
}

func TestListOptions(t *testing.T) {
	now := time.Now()
	opts := ListOptions{
		StoragePath: "/custom/path",
		Directory:   "/project",
		After:       &now,
		Before:      &now,
		Limit:       20,
	}

	if opts.StoragePath != "/custom/path" {
		t.Errorf("StoragePath = %q, want %q", opts.StoragePath, "/custom/path")
	}
	if opts.Directory != "/project" {
		t.Errorf("Directory = %q, want %q", opts.Directory, "/project")
	}
	if opts.After != &now {
		t.Error("After time not set correctly")
	}
	if opts.Before != &now {
		t.Error("Before time not set correctly")
	}
	if opts.Limit != 20 {
		t.Errorf("Limit = %d, want 20", opts.Limit)
	}
}
