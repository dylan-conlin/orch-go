package dupdetect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestDupPairTitle_Deterministic(t *testing.T) {
	pair1 := DupPair{
		FuncA:      FuncInfo{Name: "processItems", File: "a.go"},
		FuncB:      FuncInfo{Name: "handleEntries", File: "b.go"},
		Similarity: 0.92,
	}
	pair2 := DupPair{
		FuncA:      FuncInfo{Name: "handleEntries", File: "b.go"},
		FuncB:      FuncInfo{Name: "processItems", File: "a.go"},
		Similarity: 0.92,
	}

	title1 := DupPairTitle(pair1)
	title2 := DupPairTitle(pair2)

	if title1 != title2 {
		t.Errorf("titles should be identical regardless of order:\n  %q\n  %q", title1, title2)
	}

	if title1 != "Extract shared logic: handleEntries / processItems (92% similar)" {
		t.Errorf("unexpected title: %q", title1)
	}
}

func TestDupPairDescription(t *testing.T) {
	pair := DupPair{
		FuncA:      FuncInfo{Name: "foo", File: "pkg/a.go", StartLine: 10, Lines: 20},
		FuncB:      FuncInfo{Name: "bar", File: "pkg/b.go", StartLine: 5, Lines: 15},
		Similarity: 0.85,
	}
	desc := dupPairDescription(pair, ReportConfig{})

	if desc == "" {
		t.Fatal("description should not be empty")
	}
	// Verify it contains key info
	for _, want := range []string{"85%", "foo", "bar", "pkg/a.go", "pkg/b.go", "line 10", "line 5"} {
		if !contains(desc, want) {
			t.Errorf("description missing %q", want)
		}
	}
}

func TestReportToBeads_CreatesIssues(t *testing.T) {
	mock := beads.NewMockClient()

	pairs := []DupPair{
		{
			FuncA:      FuncInfo{Name: "processItems", File: "a.go", StartLine: 1, Lines: 20},
			FuncB:      FuncInfo{Name: "handleEntries", File: "b.go", StartLine: 1, Lines: 18},
			Similarity: 0.90,
		},
	}

	result, err := ReportToBeads(mock, pairs, ReportConfig{})
	if err != nil {
		t.Fatalf("ReportToBeads failed: %v", err)
	}

	if result.Created != 1 {
		t.Errorf("expected 1 issue created, got %d", result.Created)
	}
	if len(result.IssueIDs) != 1 {
		t.Fatalf("expected 1 issue ID, got %d", len(result.IssueIDs))
	}

	// Verify issue properties
	issue, err := mock.Show(result.IssueIDs[0])
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}
	if issue.IssueType != "task" {
		t.Errorf("issue type = %q, want %q", issue.IssueType, "task")
	}
	if issue.Priority != 3 {
		t.Errorf("priority = %d, want 3", issue.Priority)
	}

	// Check labels
	hasLabel := func(label string) bool {
		for _, l := range issue.Labels {
			if l == label {
				return true
			}
		}
		return false
	}
	if !hasLabel("dupdetect") {
		t.Error("missing 'dupdetect' label")
	}
	if !hasLabel("triage:review") {
		t.Error("missing 'triage:review' label")
	}
}

func TestReportToBeads_MultiplePairs(t *testing.T) {
	mock := beads.NewMockClient()

	pairs := []DupPair{
		{
			FuncA:      FuncInfo{Name: "funcA1", File: "a.go"},
			FuncB:      FuncInfo{Name: "funcA2", File: "b.go"},
			Similarity: 0.95,
		},
		{
			FuncA:      FuncInfo{Name: "funcB1", File: "c.go"},
			FuncB:      FuncInfo{Name: "funcB2", File: "d.go"},
			Similarity: 0.88,
		},
	}

	result, err := ReportToBeads(mock, pairs, ReportConfig{})
	if err != nil {
		t.Fatalf("ReportToBeads failed: %v", err)
	}

	if result.Created != 2 {
		t.Errorf("expected 2 issues created, got %d", result.Created)
	}
}

func TestReportToBeads_EmptyPairs(t *testing.T) {
	mock := beads.NewMockClient()

	result, err := ReportToBeads(mock, nil, ReportConfig{})
	if err != nil {
		t.Fatalf("ReportToBeads failed: %v", err)
	}

	if result.Created != 0 {
		t.Errorf("expected 0 issues, got %d", result.Created)
	}
}

func TestScanProject(t *testing.T) {
	// Create a temp project with duplicate functions across packages
	dir := t.TempDir()

	// Create two packages with similar functions
	pkg1Dir := filepath.Join(dir, "pkg1")
	pkg2Dir := filepath.Join(dir, "pkg2")
	os.MkdirAll(pkg1Dir, 0755)
	os.MkdirAll(pkg2Dir, 0755)

	file1 := `package pkg1

func processItems(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
	}
}
`
	file2 := `package pkg2

func handleEntries(entries []string) {
	for _, entry := range entries {
		cleaned := strings.TrimSpace(entry)
		if cleaned == "" {
			continue
		}
		fmt.Println(cleaned)
	}
}
`
	os.WriteFile(filepath.Join(pkg1Dir, "process.go"), []byte(file1), 0644)
	os.WriteFile(filepath.Join(pkg2Dir, "handle.go"), []byte(file2), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.8

	pairs, err := d.ScanProject(dir)
	if err != nil {
		t.Fatalf("ScanProject failed: %v", err)
	}

	if len(pairs) == 0 {
		t.Fatal("expected cross-package duplicate detection")
	}

	// Verify relative paths are used
	for _, pair := range pairs {
		if filepath.IsAbs(pair.FuncA.File) {
			t.Errorf("expected relative path, got absolute: %s", pair.FuncA.File)
		}
	}
}

func TestScanProject_SkipsVendorAndTestFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a vendor directory with Go files
	vendorDir := filepath.Join(dir, "vendor", "lib")
	os.MkdirAll(vendorDir, 0755)

	src := `package lib
func process() {
	x := 1
	y := x + 2
	z := y * 3
	w := z + 4
	println(w)
}
`
	// Same function in vendor and src - should NOT be detected
	os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte(src), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(strings.Replace(src, "package lib", "package main", 1)), 0644)

	d := NewDetector()
	d.MinBodyLines = 3

	pairs, err := d.ScanProject(dir)
	if err != nil {
		t.Fatalf("ScanProject failed: %v", err)
	}

	if len(pairs) != 0 {
		t.Errorf("expected vendor to be skipped, got %d pairs", len(pairs))
	}
}

func TestScanProject_SkipsGitDir(t *testing.T) {
	dir := t.TempDir()

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	src := `package hooks
func process() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
`
	os.WriteFile(filepath.Join(gitDir, "hook.go"), []byte(src), 0644)

	d := NewDetector()
	d.MinBodyLines = 3

	pairs, err := d.ScanProject(dir)
	if err != nil {
		t.Fatalf("ScanProject failed: %v", err)
	}

	// No pairs should be found (only one file outside .git)
	if len(pairs) != 0 {
		t.Errorf("expected .git to be skipped, got %d pairs", len(pairs))
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstr(s, substr)
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
