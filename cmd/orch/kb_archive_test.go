package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestParseInvestigationDate tests parsing YYYY-MM-DD prefix from investigation filenames
func TestParseInvestigationDate(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectDate  time.Time
		expectError bool
	}{
		{
			name:        "valid investigation filename",
			filename:    "2025-12-19-inv-test-investigation.md",
			expectDate:  time.Date(2025, 12, 19, 0, 0, 0, 0, time.Local),
			expectError: false,
		},
		{
			name:        "valid design filename",
			filename:    "2026-01-15-design-feature-planning.md",
			expectDate:  time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local),
			expectError: false,
		},
		{
			name:        "filename without date prefix",
			filename:    "invalid-filename.md",
			expectDate:  time.Time{},
			expectError: true,
		},
		{
			name:        "filename with invalid date",
			filename:    "2025-13-45-inv-test.md",
			expectDate:  time.Time{},
			expectError: true,
		},
		{
			name:        "full path should work",
			filename:    "/path/to/.kb/investigations/2025-12-20-inv-test.md",
			expectDate:  time.Date(2025, 12, 20, 0, 0, 0, 0, time.Local),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := parseInvestigationDate(tt.filename)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %q, got nil", tt.filename)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %q: %v", tt.filename, err)
				}
				if !date.Equal(tt.expectDate) {
					t.Errorf("Expected date %v, got %v", tt.expectDate, date)
				}
			}
		})
	}
}

// TestCalculateInvestigationAge tests age calculation from filename date
func TestCalculateInvestigationAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		filename  string
		expectAge time.Duration
		tolerance time.Duration // Allow some tolerance for test execution time
	}{
		{
			name:      "60 days old",
			filename:  now.AddDate(0, 0, -60).Format("2006-01-02") + "-inv-test.md",
			expectAge: 60 * 24 * time.Hour,
			tolerance: 24 * time.Hour, // Day-precision dates can be off by up to a day
		},
		{
			name:      "30 days old",
			filename:  now.AddDate(0, 0, -30).Format("2006-01-02") + "-inv-test.md",
			expectAge: 30 * 24 * time.Hour,
			tolerance: 24 * time.Hour,
		},
		{
			name:      "today (0 days old)",
			filename:  now.Format("2006-01-02") + "-inv-test.md",
			expectAge: 0,
			tolerance: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age, err := calculateInvestigationAge(tt.filename)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			diff := age - tt.expectAge
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Expected age ~%v, got %v (diff: %v)", tt.expectAge, age, diff)
			}
		})
	}
}

// TestFindOldInvestigations tests finding investigations older than threshold
func TestFindOldInvestigations(t *testing.T) {
	// Create temp directory with test investigation files
	tmpDir, err := os.MkdirTemp("", "test-kb-investigations-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invDir := filepath.Join(tmpDir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatalf("Failed to create investigations dir: %v", err)
	}

	now := time.Now()

	// Create test files with different ages
	files := []struct {
		filename string
		age      int // days ago
	}{
		{now.AddDate(0, 0, -70).Format("2006-01-02") + "-inv-old.md", 70},
		{now.AddDate(0, 0, -65).Format("2006-01-02") + "-inv-also-old.md", 65},
		{now.AddDate(0, 0, -30).Format("2006-01-02") + "-inv-recent.md", 30},
		{now.Format("2006-01-02") + "-inv-today.md", 0},
	}

	for _, f := range files {
		path := filepath.Join(invDir, f.filename)
		if err := os.WriteFile(path, []byte("# Test Investigation\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f.filename, err)
		}
	}

	// Find investigations older than 60 days
	threshold := 60 * 24 * time.Hour
	oldFiles, err := findOldInvestigations(tmpDir, threshold)
	if err != nil {
		t.Fatalf("findOldInvestigations failed: %v", err)
	}

	// Should find exactly 2 files (70 and 65 days old)
	if len(oldFiles) != 2 {
		t.Errorf("Expected 2 old files, got %d", len(oldFiles))
	}

	// Verify the correct files were found
	foundOld := false
	foundAlsoOld := false
	for _, path := range oldFiles {
		basename := filepath.Base(path)
		if basename == files[0].filename {
			foundOld = true
		}
		if basename == files[1].filename {
			foundAlsoOld = true
		}
	}

	if !foundOld || !foundAlsoOld {
		t.Errorf("Expected to find specific old files, got: %v", oldFiles)
	}
}

// TestArchiveOldInvestigations tests the full archival workflow
func TestArchiveOldInvestigations(t *testing.T) {
	// Create temp directory with test investigation files
	tmpDir, err := os.MkdirTemp("", "test-kb-archive-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invDir := filepath.Join(tmpDir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatalf("Failed to create investigations dir: %v", err)
	}

	now := time.Now()
	oldFilename := now.AddDate(0, 0, -70).Format("2006-01-02") + "-inv-old.md"
	recentFilename := now.AddDate(0, 0, -30).Format("2006-01-02") + "-inv-recent.md"

	oldPath := filepath.Join(invDir, oldFilename)
	recentPath := filepath.Join(invDir, recentFilename)

	testContent := "# Test Investigation\n\n## Findings\n\nSome findings here.\n"
	if err := os.WriteFile(oldPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create old file: %v", err)
	}
	if err := os.WriteFile(recentPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create recent file: %v", err)
	}

	// Archive investigations older than 60 days
	threshold := 60 * 24 * time.Hour
	result, err := archiveOldInvestigations(tmpDir, threshold, false)
	if err != nil {
		t.Fatalf("archiveOldInvestigations failed: %v", err)
	}

	// Verify result
	if len(result.Moved) != 1 {
		t.Errorf("Expected 1 file moved, got %d", len(result.Moved))
	}

	// Verify old file was moved to archive directory
	archiveDir := filepath.Join(invDir, "archive")
	archivedPath := filepath.Join(archiveDir, oldFilename)

	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Errorf("Expected archived file at %s, but it doesn't exist", archivedPath)
	}

	// Verify old file was removed from original location
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("Expected old file to be removed from %s", oldPath)
	}

	// Verify recent file was NOT moved
	if _, err := os.Stat(recentPath); os.IsNotExist(err) {
		t.Errorf("Expected recent file to remain at %s", recentPath)
	}

	// Verify archived file content is preserved
	content, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("Failed to read archived file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("Archived file content mismatch.\nExpected: %q\nGot: %q", testContent, string(content))
	}
}

// TestArchiveOldInvestigationsDryRun tests dry-run mode
func TestArchiveOldInvestigationsDryRun(t *testing.T) {
	// Create temp directory with test investigation files
	tmpDir, err := os.MkdirTemp("", "test-kb-dry-run-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invDir := filepath.Join(tmpDir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatalf("Failed to create investigations dir: %v", err)
	}

	now := time.Now()
	oldFilename := now.AddDate(0, 0, -70).Format("2006-01-02") + "-inv-old.md"
	oldPath := filepath.Join(invDir, oldFilename)

	if err := os.WriteFile(oldPath, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run in dry-run mode
	threshold := 60 * 24 * time.Hour
	result, err := archiveOldInvestigations(tmpDir, threshold, true)
	if err != nil {
		t.Fatalf("archiveOldInvestigations failed: %v", err)
	}

	// Verify matched but not moved
	if len(result.Matched) != 1 {
		t.Errorf("Expected 1 matched file, got %d", len(result.Matched))
	}
	if len(result.Moved) != 0 {
		t.Errorf("Expected 0 moved files in dry-run, got %d", len(result.Moved))
	}

	// Verify file was NOT moved
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		t.Errorf("File should not be moved in dry-run mode")
	}
}

// TestParseArchiveDuration tests parsing duration strings like "60d", "30d", "90d"
func TestParseArchiveDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{
			name:        "60 days",
			input:       "60d",
			expected:    60 * 24 * time.Hour,
			expectError: false,
		},
		{
			name:        "30 days",
			input:       "30d",
			expected:    30 * 24 * time.Hour,
			expectError: false,
		},
		{
			name:        "1 day",
			input:       "1d",
			expected:    24 * time.Hour,
			expectError: false,
		},
		{
			name:        "invalid format",
			input:       "60",
			expected:    0,
			expectError: true,
		},
		{
			name:        "invalid unit",
			input:       "60h",
			expected:    0,
			expectError: true,
		},
		{
			name:        "negative duration",
			input:       "-30d",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := parseArchiveDuration(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %q, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %q: %v", tt.input, err)
				}
				if duration != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, duration)
				}
			}
		})
	}
}
