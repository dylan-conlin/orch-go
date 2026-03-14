package dupdetect

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFindDuplicatesAgainst_MatchesFindDuplicatesFiltered(t *testing.T) {
	// Verify that scoped comparison produces the same results as
	// all-vs-all comparison followed by filtering.
	src := `package foo

func processItems(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
	}
}

func handleEntries(entries []string) {
	for _, entry := range entries {
		cleaned := strings.TrimSpace(entry)
		if cleaned == "" {
			continue
		}
		fmt.Println(cleaned)
	}
}

func computeSum(a, b int) int {
	result := a + b
	if result < 0 {
		return 0
	}
	for i := 0; i < result; i++ {
		fmt.Println(i)
	}
	return result
}
`
	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	funcs, err := d.ParseSource("test.go", src)
	if err != nil {
		t.Fatal(err)
	}

	// "Modified" = first function only
	modified := funcs[:1]
	corpus := funcs[1:]

	scopedPairs := d.FindDuplicatesAgainst(modified, corpus)

	// Reference: all-vs-all then filter
	allPairs := d.FindDuplicates(funcs)
	var filteredPairs []DupPair
	for _, p := range allPairs {
		if p.FuncA.Name == modified[0].Name || p.FuncB.Name == modified[0].Name {
			filteredPairs = append(filteredPairs, p)
		}
	}

	if len(scopedPairs) != len(filteredPairs) {
		t.Errorf("scoped found %d pairs, filtered found %d", len(scopedPairs), len(filteredPairs))
	}

	// Verify same similarities
	for i := range scopedPairs {
		if i >= len(filteredPairs) {
			break
		}
		if scopedPairs[i].Similarity != filteredPairs[i].Similarity {
			t.Errorf("pair %d: scoped sim=%f, filtered sim=%f",
				i, scopedPairs[i].Similarity, filteredPairs[i].Similarity)
		}
	}
}

func TestFindDuplicatesAgainst_ModifiedVsModified(t *testing.T) {
	// Two modified functions that are clones of each other
	src := `package foo

func cloneA(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
	}
}

func cloneB(entries []string) {
	for _, entry := range entries {
		cleaned := strings.TrimSpace(entry)
		if cleaned == "" {
			continue
		}
		fmt.Println(cleaned)
	}
}
`
	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	funcs, err := d.ParseSource("modified.go", src)
	if err != nil {
		t.Fatal(err)
	}

	// Both are "modified", corpus is empty
	pairs := d.FindDuplicatesAgainst(funcs, nil)
	if len(pairs) == 0 {
		t.Fatal("expected modified-vs-modified duplicate to be detected")
	}
}

func TestFindDuplicatesAgainst_ExcludesCorpusVsCorpus(t *testing.T) {
	// Corpus has duplicates, but since no modified file is involved,
	// they should NOT appear in results.
	src1 := `package foo

func corpusCloneA(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
	}
}

func corpusCloneB(entries []string) {
	for _, entry := range entries {
		cleaned := strings.TrimSpace(entry)
		if cleaned == "" {
			continue
		}
		fmt.Println(cleaned)
	}
}
`
	src2 := `package foo

func unrelatedModified(x int) int {
	y := x * 2
	z := y + 1
	w := z - 3
	if w > 100 {
		return w
	}
	return 0
}
`
	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	corpusFuncs, _ := d.ParseSource("corpus.go", src1)
	modifiedFuncs, _ := d.ParseSource("modified.go", src2)

	pairs := d.FindDuplicatesAgainst(modifiedFuncs, corpusFuncs)

	// Should NOT find corpus-vs-corpus pairs
	for _, p := range pairs {
		if p.FuncA.Name == "corpusCloneA" && p.FuncB.Name == "corpusCloneB" {
			t.Error("corpus-vs-corpus pair should not appear in scoped results")
		}
		if p.FuncA.Name == "corpusCloneB" && p.FuncB.Name == "corpusCloneA" {
			t.Error("corpus-vs-corpus pair should not appear in scoped results")
		}
	}
}

func TestCanMeetThreshold(t *testing.T) {
	tests := []struct {
		aLen, bLen int
		threshold  float64
		want       bool
	}{
		{100, 100, 0.80, true},  // same length
		{100, 80, 0.80, true},   // ratio = 0.80, exactly at threshold
		{100, 50, 0.80, false},  // ratio = 0.50, below threshold
		{100, 79, 0.80, false},  // ratio = 0.79, below threshold
		{0, 0, 0.80, true},     // both empty
		{100, 0, 0.80, false},  // one empty
		{10, 12, 0.80, true},   // ratio = 0.83
		{10, 13, 0.80, false},  // ratio = 0.77
	}

	for _, tt := range tests {
		a := make([]string, tt.aLen)
		b := make([]string, tt.bLen)
		got := canMeetThreshold(a, b, tt.threshold)
		if got != tt.want {
			t.Errorf("canMeetThreshold(len=%d, len=%d, %.2f) = %v, want %v",
				tt.aLen, tt.bLen, tt.threshold, got, tt.want)
		}
	}
}

func TestCheckModifiedFilesProject_Scoped(t *testing.T) {
	// Integration test: verify scoped project scan works end-to-end.
	dir := t.TempDir()

	pkg1 := filepath.Join(dir, "pkg1")
	pkg2 := filepath.Join(dir, "pkg2")
	pkg3 := filepath.Join(dir, "pkg3")
	os.MkdirAll(pkg1, 0755)
	os.MkdirAll(pkg2, 0755)
	os.MkdirAll(pkg3, 0755)

	// pkg1: existing code with a function
	os.WriteFile(filepath.Join(pkg1, "existing.go"), []byte(`package pkg1

func processItems(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
	}
}
`), 0644)

	// pkg2: agent-modified file with a clone
	os.WriteFile(filepath.Join(pkg2, "handle.go"), []byte(`package pkg2

func handleEntries(entries []string) {
	for _, entry := range entries {
		cleaned := strings.TrimSpace(entry)
		if cleaned == "" {
			continue
		}
		fmt.Println(cleaned)
	}
}
`), 0644)

	// pkg3: existing duplicates that should NOT be reported
	os.WriteFile(filepath.Join(pkg3, "corpusdup.go"), []byte(`package pkg3

func corpusDupA(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
	}
}

func corpusDupB(entries []string) {
	for _, entry := range entries {
		cleaned := strings.TrimSpace(entry)
		if cleaned == "" {
			continue
		}
		fmt.Println(cleaned)
	}
}
`), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	pairs, err := d.CheckModifiedFilesProject(dir, []string{"pkg2/handle.go"})
	if err != nil {
		t.Fatal(err)
	}

	// Should find handleEntries matched against existing/corpus clones
	if len(pairs) == 0 {
		t.Fatal("expected at least one pair involving modified file")
	}

	// Every pair must involve the modified file
	for _, p := range pairs {
		if p.FuncA.File != "pkg2/handle.go" && p.FuncB.File != "pkg2/handle.go" {
			t.Errorf("pair %s <-> %s doesn't involve modified file",
				p.FuncA.Name, p.FuncB.Name)
		}
	}
}

func TestFindDuplicatesAgainst_SkipsSelfMatch(t *testing.T) {
	// Bug: when the same function (identical name + fingerprint) exists in both
	// a modified file and a corpus file, the detector flags it as a duplicate.
	// This is a pre-existing literal copy, not agent-introduced duplication.
	//
	// Example: inferSkillFromBeadsIssue exists in both cmd/orch/work_cmd.go and
	// pkg/orch/spawn_inference.go. Modifying work_cmd.go puts the function in
	// the modified partition while the corpus copy triggers a 100% self-match.
	identicalBody := `package foo

func inferSkillFromIssue(issue string) string {
	if strings.Contains(issue, "bug") {
		return "debugging"
	}
	if strings.Contains(issue, "feature") {
		return "feature-impl"
	}
	return "investigation"
}
`
	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	modifiedFuncs, err := d.ParseSource("cmd/work_cmd.go", identicalBody)
	if err != nil {
		t.Fatal(err)
	}
	corpusFuncs, err := d.ParseSource("pkg/spawn_inference.go", identicalBody)
	if err != nil {
		t.Fatal(err)
	}

	pairs := d.FindDuplicatesAgainst(modifiedFuncs, corpusFuncs)

	// Self-match (same name, identical fingerprint) should be suppressed
	for _, p := range pairs {
		if p.FuncA.Name == p.FuncB.Name && p.Similarity >= 1.0-1e-9 {
			t.Errorf("self-match not suppressed: %s (%s) ↔ %s (%s) at %.1f%%",
				p.FuncA.Name, p.FuncA.File, p.FuncB.Name, p.FuncB.File, p.Similarity*100)
		}
	}
}

func TestFindDuplicatesAgainst_KeepsRenamedClone(t *testing.T) {
	// Ensure the fix doesn't suppress legitimate renamed clones.
	// processItems ↔ handleEntries should still be detected.
	src1 := `package foo

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
	src2 := `package foo

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
	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	modifiedFuncs, _ := d.ParseSource("modified.go", src2)
	corpusFuncs, _ := d.ParseSource("existing.go", src1)

	pairs := d.FindDuplicatesAgainst(modifiedFuncs, corpusFuncs)
	if len(pairs) == 0 {
		t.Fatal("renamed clone should still be detected")
	}
}

func TestFindDuplicatesAgainst_KeepsSameNameDiverged(t *testing.T) {
	// Same-named function in two files but bodies have diverged.
	// This should still be flagged — the bodies are different enough
	// to be interesting but similar enough to exceed threshold.
	src1 := `package foo

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
	src2 := `package bar

func processItems(items []string) {
	for _, item := range items {
		result := strings.TrimSpace(item)
		if result == "" {
			continue
		}
		fmt.Println(result)
		fmt.Println("extra line")
	}
}
`
	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	modifiedFuncs, _ := d.ParseSource("modified.go", src2)
	corpusFuncs, _ := d.ParseSource("corpus.go", src1)

	pairs := d.FindDuplicatesAgainst(modifiedFuncs, corpusFuncs)
	// Same name but different bodies — should still be flagged if above threshold
	if len(pairs) == 0 {
		t.Fatal("diverged same-name functions should still be detected")
	}
}

func TestCheckModifiedFilesProject_SelfMatchSuppressed(t *testing.T) {
	// Integration test: same function exists as literal copy in two packages.
	// Modifying one file should NOT flag the pre-existing copy.
	dir := t.TempDir()

	pkg1 := filepath.Join(dir, "pkg1")
	pkg2 := filepath.Join(dir, "pkg2")
	os.MkdirAll(pkg1, 0755)
	os.MkdirAll(pkg2, 0755)

	identicalFunc := `func inferSkill(issue string) string {
	if strings.Contains(issue, "bug") {
		return "debugging"
	}
	if strings.Contains(issue, "feature") {
		return "feature-impl"
	}
	return "investigation"
}
`
	// Both files have the EXACT same function
	os.WriteFile(filepath.Join(pkg1, "work.go"), []byte("package pkg1\n\n"+identicalFunc), 0644)
	os.WriteFile(filepath.Join(pkg2, "spawn.go"), []byte("package pkg2\n\n"+identicalFunc), 0644)

	d := NewDetector()
	d.MinBodyLines = 3
	d.Threshold = 0.80

	// Only pkg1/work.go was modified — the copy in pkg2 is pre-existing
	pairs, err := d.CheckModifiedFilesProject(dir, []string{"pkg1/work.go"})
	if err != nil {
		t.Fatal(err)
	}

	// Self-match should be suppressed
	for _, p := range pairs {
		if p.FuncA.Name == p.FuncB.Name && p.Similarity >= 1.0-1e-9 {
			t.Errorf("self-match not suppressed in project scan: %s (%s) ↔ %s (%s)",
				p.FuncA.Name, p.FuncA.File, p.FuncB.Name, p.FuncB.File)
		}
	}
}

// BenchmarkFindDuplicates_AllVsAll benchmarks the old O(N²) approach.
func BenchmarkFindDuplicates_AllVsAll(b *testing.B) {
	funcs := generateSyntheticFuncs(500, 50)
	d := NewDetector()
	d.Threshold = 0.80

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.FindDuplicates(funcs)
	}
}

// BenchmarkFindDuplicatesAgainst_Scoped benchmarks the new O(M×N) approach
// with M=10 modified functions against N=500 total.
func BenchmarkFindDuplicatesAgainst_Scoped(b *testing.B) {
	allFuncs := generateSyntheticFuncs(500, 50)
	modified := allFuncs[:10]
	corpus := allFuncs[10:]
	d := NewDetector()
	d.Threshold = 0.80

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.FindDuplicatesAgainst(modified, corpus)
	}
}

// generateSyntheticFuncs creates N synthetic functions with fingerprints of
// the given length, varying slightly to create realistic comparison work.
func generateSyntheticFuncs(n, fpLen int) []FuncInfo {
	funcs := make([]FuncInfo, n)
	tokens := []string{"CALL", "RETURN", "IF", "FOR", "RANGE", "ASSIGN:=", "BINARY:+", "IDENT:$0", "IDENT:$1", "LIT:INT:1"}

	for i := 0; i < n; i++ {
		fp := make([]string, fpLen)
		for j := 0; j < fpLen; j++ {
			fp[j] = tokens[(i+j)%len(tokens)]
		}
		funcs[i] = FuncInfo{
			File:        fmt.Sprintf("file%d.go", i/10),
			Name:        fmt.Sprintf("func%d", i),
			Lines:       15,
			StartLine:   1,
			Fingerprint: fp,
		}
	}
	return funcs
}
