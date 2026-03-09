package dupdetect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFingerprint_IdenticalFunctions(t *testing.T) {
	src := `package foo
func a() { x := 1; y := x + 2; println(y) }
func b() { x := 1; y := x + 2; println(y) }
`
	pairs := detectFromSource(t, src)
	if len(pairs) == 0 {
		t.Fatal("expected identical functions to be detected as duplicates")
	}
	if pairs[0].Similarity < 1.0 {
		t.Errorf("expected similarity 1.0 for identical bodies, got %f", pairs[0].Similarity)
	}
}

func TestFingerprint_RenamedVariables(t *testing.T) {
	src := `package foo
func a() { x := 1; y := x + 2; println(y) }
func b() { m := 1; n := m + 2; println(n) }
`
	pairs := detectFromSource(t, src)
	if len(pairs) == 0 {
		t.Fatal("expected renamed-variable functions to be detected as duplicates")
	}
	if pairs[0].Similarity < 0.9 {
		t.Errorf("expected high similarity for renamed variables, got %f", pairs[0].Similarity)
	}
}

func TestFingerprint_DifferentStructure(t *testing.T) {
	src := `package foo
func a() { x := 1; println(x) }
func b() { for i := 0; i < 10; i++ { println(i) } }
`
	pairs := detectFromSource(t, src)
	if len(pairs) != 0 {
		t.Errorf("expected no duplicates for structurally different functions, got %d pairs", len(pairs))
	}
}

func TestFingerprint_SkipSmallFunctions(t *testing.T) {
	// Functions below MinBodyLines should be skipped
	src := `package foo
func a() { return }
func b() { return }
`
	d := NewDetector()
	d.MinBodyLines = 5
	pairs := detectFromSourceWithDetector(t, src, d)
	if len(pairs) != 0 {
		t.Errorf("expected small functions to be skipped, got %d pairs", len(pairs))
	}
}

func TestFingerprint_SimilarButNotIdentical(t *testing.T) {
	src := `package foo
func a() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func b() {
	x := 1
	y := x + 2
	z := y * 4
	println(z)
}
`
	d := NewDetector()
	d.MinBodyLines = 2
	d.Threshold = 0.7
	pairs := detectFromSourceWithDetector(t, src, d)
	if len(pairs) == 0 {
		t.Fatal("expected similar functions to be detected above 0.7 threshold")
	}
}

func TestDetector_ScanDir(t *testing.T) {
	// Create temp directory with Go files
	dir := t.TempDir()

	file1 := `package foo
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
	file2 := `package foo
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
	os.WriteFile(filepath.Join(dir, "file1.go"), []byte(file1), 0644)
	os.WriteFile(filepath.Join(dir, "file2.go"), []byte(file2), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.8
	results, err := d.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected cross-file duplicate detection")
	}
	if !strings.Contains(results[0].FuncA.File, "file1.go") || !strings.Contains(results[0].FuncB.File, "file2.go") {
		t.Errorf("expected cross-file match, got %s vs %s", results[0].FuncA.File, results[0].FuncB.File)
	}
}

func TestDetector_ScanDir_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()

	src := `package foo
func process() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
`
	testSrc := `package foo
func TestProcess() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
`
	os.WriteFile(filepath.Join(dir, "prod.go"), []byte(src), 0644)
	os.WriteFile(filepath.Join(dir, "prod_test.go"), []byte(testSrc), 0644)

	d := NewDetector()
	d.MinBodyLines = 2
	results, err := d.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected test files to be skipped, got %d pairs", len(results))
	}
}

func TestDetector_RealCmdOrch(t *testing.T) {
	// Integration test: run against actual cmd/orch/ files.
	// This is advisory — it reports duplicates but doesn't fail.
	projectRoot := findRoot(t)
	cmdDir := filepath.Join(projectRoot, "cmd", "orch")

	d := NewDetector()
	d.MinBodyLines = 10
	d.Threshold = 0.80
	results, err := d.ScanDir(cmdDir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	if len(results) > 0 {
		t.Logf("Found %d duplicate function pairs in cmd/orch/:", len(results))
		for i, pair := range results {
			t.Logf("  %d. %.0f%% similar: %s:%s (%d lines) <-> %s:%s (%d lines)",
				i+1,
				pair.Similarity*100,
				pair.FuncA.File, pair.FuncA.Name, pair.FuncA.Lines,
				pair.FuncB.File, pair.FuncB.Name, pair.FuncB.Lines,
			)
		}
	} else {
		t.Log("No function duplicates found above threshold")
	}
}

// helpers

func detectFromSource(t *testing.T, src string) []DupPair {
	t.Helper()
	d := NewDetector()
	d.MinBodyLines = 1
	return detectFromSourceWithDetector(t, src, d)
}

func detectFromSourceWithDetector(t *testing.T, src string, d *Detector) []DupPair {
	t.Helper()
	funcs, err := d.ParseSource("test.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	return d.FindDuplicates(funcs)
}

func findRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find project root (no go.mod found)")
		}
		dir = parent
	}
}
