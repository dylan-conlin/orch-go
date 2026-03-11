package dupdetect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckModifiedFiles_DetectsNewDuplication(t *testing.T) {
	// Setup: two files in a directory, the second introduces a near-clone
	dir := t.TempDir()

	existing := `package foo

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
	newFile := `package foo

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
	os.WriteFile(filepath.Join(dir, "existing.go"), []byte(existing), 0644)
	os.WriteFile(filepath.Join(dir, "new_file.go"), []byte(newFile), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	result, err := d.CheckModifiedFiles(dir, []string{"new_file.go"})
	if err != nil {
		t.Fatalf("CheckModifiedFiles failed: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("expected duplication detected between new and existing file")
	}

	// The new function should be one side of the pair
	found := false
	for _, pair := range result {
		if pair.FuncA.Name == "handleEntries" || pair.FuncB.Name == "handleEntries" {
			found = true
			if pair.Similarity < 0.80 {
				t.Errorf("expected similarity >= 0.80, got %f", pair.Similarity)
			}
		}
	}
	if !found {
		t.Error("expected handleEntries to be detected as a duplicate")
	}
}

func TestCheckModifiedFiles_NoDuplication(t *testing.T) {
	dir := t.TempDir()

	existing := `package foo

func computeSum(a, b int) int {
	result := a + b
	if result < 0 {
		return 0
	}
	return result
}
`
	newFile := `package foo

func formatOutput(items []string) string {
	var sb strings.Builder
	for i, item := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(item)
	}
	return sb.String()
}
`
	os.WriteFile(filepath.Join(dir, "math.go"), []byte(existing), 0644)
	os.WriteFile(filepath.Join(dir, "format.go"), []byte(newFile), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	result, err := d.CheckModifiedFiles(dir, []string{"format.go"})
	if err != nil {
		t.Fatalf("CheckModifiedFiles failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected no duplicates for different functions, got %d", len(result))
	}
}

func TestCheckModifiedFiles_OnlyReportsModifiedSide(t *testing.T) {
	// If two existing files are duplicates but neither was modified,
	// they should NOT be reported.
	dir := t.TempDir()

	dup1 := `package foo

func processA(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
	}
}
`
	dup2 := `package foo

func processB(entries []string) {
	for _, entry := range entries {
		cleaned := strings.TrimSpace(entry)
		if cleaned == "" {
			continue
		}
		fmt.Println(cleaned)
	}
}
`
	unrelated := `package foo

func unrelated() {
	x := 42
	y := x * 2
	z := y + 1
	fmt.Println(z)
}
`
	os.WriteFile(filepath.Join(dir, "dup1.go"), []byte(dup1), 0644)
	os.WriteFile(filepath.Join(dir, "dup2.go"), []byte(dup2), 0644)
	os.WriteFile(filepath.Join(dir, "unrelated.go"), []byte(unrelated), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	// Only unrelated.go was modified — existing duplicates should not be reported
	result, err := d.CheckModifiedFiles(dir, []string{"unrelated.go"})
	if err != nil {
		t.Fatalf("CheckModifiedFiles failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected no duplicates when modified file is unique, got %d", len(result))
	}
}

func TestCheckModifiedFiles_CrossDirectoryProject(t *testing.T) {
	dir := t.TempDir()

	pkg1Dir := filepath.Join(dir, "pkg1")
	pkg2Dir := filepath.Join(dir, "pkg2")
	os.MkdirAll(pkg1Dir, 0755)
	os.MkdirAll(pkg2Dir, 0755)

	existing := `package pkg1

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
	newFile := `package pkg2

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
	os.WriteFile(filepath.Join(pkg1Dir, "process.go"), []byte(existing), 0644)
	os.WriteFile(filepath.Join(pkg2Dir, "handle.go"), []byte(newFile), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	result, err := d.CheckModifiedFilesProject(dir, []string{"pkg2/handle.go"})
	if err != nil {
		t.Fatalf("CheckModifiedFilesProject failed: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("expected cross-directory duplicate detection")
	}
}

func TestCheckModifiedFiles_EmptyModifiedList(t *testing.T) {
	dir := t.TempDir()

	d := NewDetector()
	result, err := d.CheckModifiedFiles(dir, nil)
	if err != nil {
		t.Fatalf("CheckModifiedFiles failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected no results for empty modified list, got %d", len(result))
	}
}

func TestFormatDuplicationAdvisory_NoPairs(t *testing.T) {
	result := FormatDuplicationAdvisory(nil)
	if result != "" {
		t.Errorf("expected empty string for nil pairs, got %q", result)
	}
}

func TestFormatDuplicationAdvisory_WithPairs(t *testing.T) {
	pairs := []DupPair{
		{
			FuncA:      FuncInfo{Name: "processItems", File: "a.go", StartLine: 3, Lines: 10},
			FuncB:      FuncInfo{Name: "handleEntries", File: "b.go", StartLine: 3, Lines: 10},
			Similarity: 0.95,
		},
	}

	result := FormatDuplicationAdvisory(pairs)
	if result == "" {
		t.Fatal("expected non-empty advisory")
	}

	for _, want := range []string{"DUPLICATION", "processItems", "handleEntries", "95%"} {
		if !strings.Contains(result, want) {
			t.Errorf("advisory missing %q", want)
		}
	}
}
