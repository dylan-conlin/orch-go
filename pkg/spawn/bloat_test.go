package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single go file path",
			input:    "Modify pkg/spawn/context.go to add feature",
			expected: []string{"pkg/spawn/context.go"},
		},
		{
			name:     "multiple file paths",
			input:    "Update pkg/spawn/context.go and pkg/spawn/config.go",
			expected: []string{"pkg/spawn/context.go", "pkg/spawn/config.go"},
		},
		{
			name:     "path with leading dot slash",
			input:    "Edit ./internal/handler.go file",
			expected: []string{"internal/handler.go"},
		},
		{
			name:     "typescript file",
			input:    "Fix bug in src/components/Button.tsx",
			expected: []string{"src/components/Button.tsx"},
		},
		{
			name:     "path in backticks",
			input:    "Modify `pkg/spawn/context.go` to add feature",
			expected: []string{"pkg/spawn/context.go"},
		},
		{
			name:     "no file paths",
			input:    "Add a new feature to the application",
			expected: nil,
		},
		{
			name:     "filter out URLs",
			input:    "Check https://example.com/path.html and edit pkg/main.go",
			expected: []string{"pkg/main.go"},
		},
		{
			name:     "filter out version strings",
			input:    "Use version v1.0.0 and edit pkg/main.go",
			expected: []string{"pkg/main.go"},
		},
		{
			name:     "deduplicate paths",
			input:    "Edit pkg/main.go and then edit pkg/main.go again",
			expected: []string{"pkg/main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFilePaths(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("extractFilePaths(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}

			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("extractFilePaths(%q)[%d] = %q, want %q", tt.input, i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"pkg/spawn/context_test.go", true},
		{"pkg/spawn/context.go", false},
		{"src/component.test.ts", true},
		{"src/component.test.tsx", true},
		{"src/component.spec.ts", true},
		{"src/__tests__/component.tsx", true},
		{"src/component.tsx", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isTestFile(tt.path)
			if result != tt.expected {
				t.Errorf("isTestFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"pkg/spawn/context.go", true},
		{"main.go", true},
		{"src/index.ts", true},
		{"http://example.com/path.html", false},
		{"https://example.com/path.html", false},
		{"v1.0.0", false},
		{"1.2.3", false},
		{"noextension", false},
		{"file.unknown", false},
		{"pkg/config.yaml", true},
		{"data.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := looksLikeFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeFilePath(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCountFileLines(t *testing.T) {
	// Create a temporary file with known line count
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	// Create file with 10 lines
	content := strings.Repeat("line\n", 10)
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	count, err := countFileLines(testFile)
	if err != nil {
		t.Fatalf("countFileLines() error = %v", err)
	}

	if count != 10 {
		t.Errorf("countFileLines() = %d, want 10", count)
	}
}

func TestCheckBloatedFiles(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create a "bloated" file (over 800 lines)
	bloatedFile := filepath.Join(tmpDir, "pkg", "spawn", "bloated.go")
	if err := os.MkdirAll(filepath.Dir(bloatedFile), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	bloatedContent := strings.Repeat("line\n", 850)
	if err := os.WriteFile(bloatedFile, []byte(bloatedContent), 0644); err != nil {
		t.Fatalf("failed to create bloated file: %v", err)
	}

	// Create a normal file (under 800 lines)
	normalFile := filepath.Join(tmpDir, "pkg", "spawn", "normal.go")
	normalContent := strings.Repeat("line\n", 100)
	if err := os.WriteFile(normalFile, []byte(normalContent), 0644); err != nil {
		t.Fatalf("failed to create normal file: %v", err)
	}

	// Create a test file (should be exempt)
	testFile := filepath.Join(tmpDir, "pkg", "spawn", "bloated_test.go")
	if err := os.WriteFile(testFile, []byte(bloatedContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name          string
		task          string
		expectedCount int
	}{
		{
			name:          "task mentioning bloated file",
			task:          "Modify pkg/spawn/bloated.go to add feature",
			expectedCount: 1,
		},
		{
			name:          "task mentioning normal file",
			task:          "Modify pkg/spawn/normal.go to add feature",
			expectedCount: 0,
		},
		{
			name:          "task mentioning test file (exempt)",
			task:          "Modify pkg/spawn/bloated_test.go to add tests",
			expectedCount: 0,
		},
		{
			name:          "task mentioning nonexistent file",
			task:          "Modify pkg/spawn/nonexistent.go",
			expectedCount: 0,
		},
		{
			name:          "task with no file paths",
			task:          "Add a new feature",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := CheckBloatedFiles(tt.task, tmpDir)
			if len(warnings) != tt.expectedCount {
				t.Errorf("CheckBloatedFiles() returned %d warnings, want %d", len(warnings), tt.expectedCount)
			}
		})
	}
}

func TestGenerateBloatWarningSection(t *testing.T) {
	tests := []struct {
		name     string
		warnings []BloatWarning
		contains []string
		empty    bool
	}{
		{
			name:     "no warnings returns empty",
			warnings: nil,
			empty:    true,
		},
		{
			name: "single warning",
			warnings: []BloatWarning{
				{Path: "pkg/spawn/context.go", LineCount: 850, Recommendation: "WARNING (850 lines)"},
			},
			contains: []string{"BLOAT WARNING", "pkg/spawn/context.go", "850 lines"},
		},
		{
			name: "critical warning",
			warnings: []BloatWarning{
				{Path: "pkg/spawn/large.go", LineCount: 1600, Recommendation: "CRITICAL (1600 lines)"},
			},
			contains: []string{"BLOAT WARNING", "pkg/spawn/large.go", "CRITICAL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateBloatWarningSection(tt.warnings)

			if tt.empty && result != "" {
				t.Errorf("GenerateBloatWarningSection() = %q, want empty string", result)
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("GenerateBloatWarningSection() result should contain %q", expected)
				}
			}
		})
	}
}

func TestGenerateBloatRecommendation(t *testing.T) {
	tests := []struct {
		lines    int
		contains string
	}{
		{850, "WARNING"},
		{1600, "CRITICAL"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := generateBloatRecommendation("test.go", tt.lines)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("generateBloatRecommendation() = %q, should contain %q", result, tt.contains)
			}
		})
	}
}
