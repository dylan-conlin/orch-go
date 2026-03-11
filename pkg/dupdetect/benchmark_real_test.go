package dupdetect

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRealProject_ScopedVsFullComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real project comparison in short mode")
	}
	projectRoot := findRoot(t)

	d := NewDetector()
	d.MinBodyLines = 10
	d.Threshold = 0.85 // matches completion advisory threshold

	// Simulate a typical agent: modified 3 files
	modifiedFiles := []string{
		"pkg/dupdetect/dupdetect.go",
		"pkg/dupdetect/staged.go",
		"pkg/dupdetect/report.go",
	}

	// Time the new scoped approach
	start := time.Now()
	scopedPairs, err := d.CheckModifiedFilesProject(projectRoot, modifiedFiles)
	scopedDuration := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}

	// Time the old full-scan approach for comparison
	start = time.Now()
	allPairs, err := d.ScanProject(projectRoot)
	fullDuration := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}

	// Count how many of the full pairs involve our modified files
	modSet := make(map[string]bool)
	for _, f := range modifiedFiles {
		modSet[f] = true
	}
	var fullFiltered int
	for _, p := range allPairs {
		if modSet[p.FuncA.File] || modSet[p.FuncB.File] {
			fullFiltered++
		}
	}

	t.Logf("Full project scan (N²): %d pairs in %v", len(allPairs), fullDuration)
	t.Logf("Scoped scan (M×N):      %d pairs in %v", len(scopedPairs), scopedDuration)
	t.Logf("Full scan filtered:      %d pairs involving modified files", fullFiltered)
	t.Logf("Speedup:                 %.1fx", float64(fullDuration)/float64(scopedDuration))

	// Parse functions to log scale
	allFuncs, _ := d.ScanProjectFuncs(projectRoot)
	var modCount int
	for _, fn := range allFuncs {
		if modSet[fn.File] {
			modCount++
		}
	}
	t.Logf("Total functions (N):     %d", len(allFuncs))
	t.Logf("Modified functions (M):  %d", modCount)
	t.Logf("N² comparisons:          %d", len(allFuncs)*(len(allFuncs)-1)/2)
	t.Logf("M×N comparisons:         %d", modCount*len(allFuncs))

	// Scoped results should match the filtered full results
	if len(scopedPairs) != fullFiltered {
		// Allow mismatch to be logged but investigate
		t.Logf("NOTE: scoped=%d vs filtered=%d — checking pair equivalence", len(scopedPairs), fullFiltered)

		// Build lookup for comparison
		scopedSet := make(map[string]float64)
		for _, p := range scopedPairs {
			key := pairKey(p)
			scopedSet[key] = p.Similarity
		}

		for _, p := range allPairs {
			if !modSet[p.FuncA.File] && !modSet[p.FuncB.File] {
				continue
			}
			key := pairKey(p)
			if _, ok := scopedSet[key]; !ok {
				t.Errorf("full-scan pair missing from scoped: %s <-> %s (%.2f)",
					p.FuncA.Name, p.FuncB.Name, p.Similarity)
			}
		}
	}

	// The scoped approach should be significantly faster
	if scopedDuration > fullDuration {
		t.Errorf("scoped approach (%v) should be faster than full scan (%v)", scopedDuration, fullDuration)
	}
}

func pairKey(p DupPair) string {
	a := p.FuncA.File + ":" + p.FuncA.Name
	b := p.FuncB.File + ":" + p.FuncB.Name
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

func BenchmarkRealProject_FullScan(b *testing.B) {
	projectRoot := findRootB(b)

	d := NewDetector()
	d.MinBodyLines = 10
	d.Threshold = 0.85

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.ScanProject(projectRoot)
	}
}

func BenchmarkRealProject_ScopedScan(b *testing.B) {
	projectRoot := findRootB(b)

	d := NewDetector()
	d.MinBodyLines = 10
	d.Threshold = 0.85

	modifiedFiles := []string{
		"pkg/dupdetect/dupdetect.go",
		"pkg/dupdetect/staged.go",
		"pkg/dupdetect/report.go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.CheckModifiedFilesProject(projectRoot, modifiedFiles)
	}
}

func findRootB(b *testing.B) string {
	b.Helper()
	dir := filepath.Join(".", "..", "..")
	abs, err := filepath.Abs(dir)
	if err != nil {
		b.Fatal(err)
	}
	return abs
}
