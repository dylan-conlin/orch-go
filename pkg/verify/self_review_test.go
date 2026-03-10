package verify

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestFilterProductionFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  int // number of production files
	}{
		{
			name:  "filters test files",
			files: []string{"main.go", "main_test.go", "handler.go", "handler_test.go"},
			want:  2,
		},
		{
			name:  "filters JS test files",
			files: []string{"App.tsx", "App.test.tsx", "utils.ts", "utils.spec.ts"},
			want:  2,
		},
		{
			name:  "filters test directories",
			files: []string{"src/handler.go", "tests/handler_test.go", "__tests__/App.test.tsx"},
			want:  1,
		},
		{
			name:  "filters non-code files",
			files: []string{"main.go", "README.md", "config.yaml", "go.sum"},
			want:  1,
		},
		{
			name:  "filters kb and beads files",
			files: []string{"main.go", ".kb/investigations/inv.md", ".beads/issues.jsonl"},
			want:  1,
		},
		{
			name:  "empty input",
			files: []string{},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterProductionFiles(tt.files)
			if len(got) != tt.want {
				t.Errorf("filterProductionFiles() returned %d files, want %d; got: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestIsProductionFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"cmd/orch/main.go", true},
		{"cmd/orch/main_test.go", false},
		{"web/src/App.svelte", true},
		{"web/src/App.test.ts", false},
		{"__tests__/App.test.tsx", false},
		{"testdata/fixture.json", false},
		{"README.md", false},
		{".kb/models/foo/model.md", false},
		{".beads/hooks/on_close", false},
		{"pkg/verify/check.go", true},
		{"config.yaml", false},
		{"go.sum", false},
		{"web/src/lib/store.ts", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isProductionFile(tt.path)
			if got != tt.want {
				t.Errorf("isProductionFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchesExtension(t *testing.T) {
	tests := []struct {
		ext        string
		extensions []string
		want       bool
	}{
		{".go", []string{".go"}, true},
		{".ts", []string{".go"}, false},
		{".tsx", []string{".ts", ".tsx", ".js", ".jsx"}, true},
		{".go", []string{}, true}, // empty means match all
		{".py", []string{".go", ".rs"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := matchesExtension(tt.ext, tt.extensions)
			if got != tt.want {
				t.Errorf("matchesExtension(%q, %v) = %v, want %v", tt.ext, tt.extensions, got, tt.want)
			}
		})
	}
}

func TestDebugPatterns(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantHit  bool
		matchExt string
	}{
		{"console.log", `console.log("debug")`, true, ".ts"},
		{"console.debug", `console.debug(x)`, true, ".js"},
		{"console.error", `console.error("fail")`, true, ".tsx"},
		{"debugger", `  debugger`, true, ".ts"},
		{"fmt.Println", `fmt.Println("debug")`, true, ".go"},
		{"fmt.Printf", `fmt.Printf("val: %v\n", x)`, true, ".go"},
		{"fmt.Print", `fmt.Print(x)`, true, ".go"},
		{"python print", `print("hello")`, true, ".py"},
		{"pdb", `pdb.set_trace()`, true, ".py"},
		// Non-matches
		{"log.Printf (not debug)", `log.Printf("info")`, false, ".go"},
		{"normal code", `x := 42`, false, ".go"},
		{"comment about console", `// We use console for output`, false, ".ts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit := false
			for _, dp := range debugPatterns {
				if matchesExtension(tt.matchExt, dp.Extensions) && dp.Pattern.MatchString(tt.line) {
					hit = true
					break
				}
			}
			if hit != tt.wantHit {
				t.Errorf("debugPatterns on %q (ext %s): got hit=%v, want %v", tt.line, tt.matchExt, hit, tt.wantHit)
			}
		})
	}
}

func TestConventionalCommitPattern(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"feat: add new feature", true},
		{"fix: resolve crash on startup", true},
		{"refactor: extract helper function", true},
		{"test: add unit tests for parser", true},
		{"docs: update README", true},
		{"chore: bump dependencies", true},
		{"feat(auth): add OAuth support", true},
		{"fix!: breaking change to API", true},
		// Non-matches
		{"Update README", false},
		{"WIP save progress", false},
		{"fixup! previous commit", false},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := conventionalCommitPattern.MatchString(tt.msg)
			if got != tt.want {
				t.Errorf("conventionalCommitPattern.MatchString(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestWipCommitPattern(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"WIP save progress", true},
		{"wip: working on it", true},
		{"temp fix for tests", true},
		{"fixup! previous commit", true},
		{"squash! merge changes", true},
		{"TODO finish this", true},
		// Non-matches
		{"feat: add new feature", false},
		{"fix: resolve bug", false},
		{"Temporary file handling improvement", false}, // "temp" must be at start
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := wipCommitPattern.MatchString(tt.msg)
			if got != tt.want {
				t.Errorf("wipCommitPattern.MatchString(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestPlaceholderPatterns(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantHit bool
	}{
		{"john doe", `name: "John Doe"`, true},
		{"jane smith", `user = "Jane Smith"`, true},
		{"test user", `const user = "Test User"`, true},
		{"lorem ipsum", `text: "Lorem ipsum dolor sit amet"`, true},
		{"test email", `email: "test@example.com"`, true},
		{"phone 555", `phone: "555-1234"`, true},
		// Non-matches
		{"real name", `name: "Dylan Conlin"`, false},
		{"real domain", `url: "https://example.org/api"`, false},
		{"normal code", `x := calculateTotal(items)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit := false
			for _, pp := range placeholderPatterns {
				if pp.Pattern.MatchString(tt.line) {
					hit = true
					break
				}
			}
			if hit != tt.wantHit {
				t.Errorf("placeholderPatterns on %q: got hit=%v, want %v", tt.line, hit, tt.wantHit)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is longer than the limit", 10, "this is lo..."},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestVerifySelfReviewForCompletion_NilOnEmptyProjectDir(t *testing.T) {
	result := VerifySelfReviewForCompletion("", "")
	if result != nil {
		t.Errorf("expected nil for empty projectDir, got %+v", result)
	}
}

// TestSelfReview_PreExistingDebugNotFlagged is the key reproduction test for the bug:
// Pre-existing fmt.Println in a file touched by the agent should NOT be flagged.
// The self-review gate must compare against the agent's baseline commit, not HEAD.
func TestSelfReview_PreExistingDebugNotFlagged(t *testing.T) {
	repoDir := t.TempDir()
	workspaceDir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	// Initialize repo with pre-existing fmt.Println (committed before agent spawned)
	run("git", "init")
	os.WriteFile(filepath.Join(repoDir, "daemon.go"), []byte(
		"package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"startup\")\n}\n",
	), 0644)
	run("git", "add", "-A")
	run("git", "commit", "-m", "feat: initial daemon with fmt.Println")

	// Record baseline (this is when the agent was spawned)
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	baselineOut, _ := cmd.Output()
	baseline := strings.TrimSpace(string(baselineOut))

	// Write manifest with baseline
	manifest := spawn.AgentManifest{
		GitBaseline:   baseline,
		WorkspaceName: "test-agent",
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(workspaceDir, "AGENT_MANIFEST.json"), data, 0644)

	// Agent adds a new file (no debug statements) — touching the same repo
	os.WriteFile(filepath.Join(repoDir, "handler.go"), []byte(
		"package main\n\nfunc handleRequest() error {\n\treturn nil\n}\n",
	), 0644)
	run("git", "add", "-A")
	run("git", "commit", "-m", "feat: add handler")

	// Run self-review — should pass because agent didn't add any debug statements
	result := VerifySelfReviewForCompletion(workspaceDir, repoDir)
	if result != nil && !result.Passed {
		t.Errorf("self-review should pass (agent didn't add debug statements), but got errors: %v", result.Errors)
	}
}

// TestSelfReview_AgentAddedDebugIsFlagged verifies that debug statements
// added by the agent ARE still detected.
func TestSelfReview_AgentAddedDebugIsFlagged(t *testing.T) {
	repoDir := t.TempDir()
	workspaceDir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	// Initialize repo
	run("git", "init")
	os.WriteFile(filepath.Join(repoDir, "main.go"), []byte(
		"package main\n\nfunc main() {}\n",
	), 0644)
	run("git", "add", "-A")
	run("git", "commit", "-m", "feat: initial commit")

	// Record baseline
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	baselineOut, _ := cmd.Output()
	baseline := strings.TrimSpace(string(baselineOut))

	// Write manifest with baseline
	manifest := spawn.AgentManifest{
		GitBaseline:   baseline,
		WorkspaceName: "test-agent",
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(workspaceDir, "AGENT_MANIFEST.json"), data, 0644)

	// Agent adds a file WITH a debug statement
	os.WriteFile(filepath.Join(repoDir, "handler.go"), []byte(
		"package main\n\nimport \"fmt\"\n\nfunc handle() {\n\tfmt.Println(\"debug\")\n}\n",
	), 0644)
	run("git", "add", "-A")
	run("git", "commit", "-m", "feat: add handler with debug")

	// Run self-review — should fail because agent added fmt.Println
	result := VerifySelfReviewForCompletion(workspaceDir, repoDir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Passed {
		t.Error("self-review should fail (agent added fmt.Println), but it passed")
	}

	// Verify the error mentions debug statement
	found := false
	for _, err := range result.Errors {
		if strings.Contains(err, "fmt.Print") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about fmt.Print, got: %v", result.Errors)
	}
}

func TestParseHunkNewStart(t *testing.T) {
	tests := []struct {
		header string
		want   int
	}{
		{"@@ -10,5 +20,7 @@ func foo()", 20},
		{"@@ -0,0 +1,15 @@", 1},
		{"@@ -5 +5 @@", 5},
		{"@@ -1,3 +100,3 @@", 100},
		{"not a hunk", 0},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := parseHunkNewStart(tt.header)
			if got != tt.want {
				t.Errorf("parseHunkNewStart(%q) = %d, want %d", tt.header, got, tt.want)
			}
		})
	}
}

func TestParseAddedLinesFromDiff(t *testing.T) {
	fmtPrintPattern := debugPatterns[2].Pattern // \bfmt\.Print(ln|f)?\b

	tests := []struct {
		name     string
		diff     string
		pattern  *regexp.Regexp
		wantNums []int
	}{
		{
			name: "detects fmt.Println in added line",
			diff: `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -10,3 +10,5 @@ func main() {
 	existing := true
+	fmt.Println("debug")
+	x := 42
 	return
`,
			pattern:  fmtPrintPattern,
			wantNums: []int{11},
		},
		{
			name: "ignores fmt.Println in removed line",
			diff: `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -10,4 +10,3 @@ func main() {
 	existing := true
-	fmt.Println("old debug")
 	return
`,
			pattern:  fmtPrintPattern,
			wantNums: nil,
		},
		{
			name: "ignores fmt.Println in context line",
			diff: `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -10,3 +10,4 @@ func main() {
 	fmt.Println("pre-existing")
+	x := 42
 	return
`,
			pattern:  fmtPrintPattern,
			wantNums: nil,
		},
		{
			name: "multiple hunks with added debug",
			diff: `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -5,2 +5,3 @@ package main
 import "fmt"
+var debug = true
@@ -20,2 +21,3 @@ func run() {
 	result := compute()
+	fmt.Printf("result: %v\n", result)
`,
			pattern:  fmtPrintPattern,
			wantNums: []int{22},
		},
		{
			name: "no added lines matching pattern",
			diff: `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -10,2 +10,3 @@ func main() {
 	x := 1
+	y := 2
`,
			pattern:  fmtPrintPattern,
			wantNums: nil,
		},
		{
			name:     "empty diff",
			diff:     "",
			pattern:  fmtPrintPattern,
			wantNums: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAddedLinesFromDiff(tt.diff, tt.pattern)
			if len(got) != len(tt.wantNums) {
				t.Fatalf("parseAddedLinesFromDiff() returned %d matches, want %d; got: %v", len(got), len(tt.wantNums), got)
			}
			for i, num := range got {
				if num != tt.wantNums[i] {
					t.Errorf("match[%d] = %d, want %d", i, num, tt.wantNums[i])
				}
			}
		})
	}
}

func TestParseAddedLinesFromDiff_ConsoleLog(t *testing.T) {
	consolePattern := debugPatterns[0].Pattern // console.log/debug/etc.

	diff := `diff --git a/app.ts b/app.ts
--- a/app.ts
+++ b/app.ts
@@ -1,3 +1,5 @@
 const x = 1
+console.log("debug")
 const y = 2
+const z = 3
`
	got := parseAddedLinesFromDiff(diff, consolePattern)
	if len(got) != 1 || got[0] != 2 {
		t.Errorf("expected [2], got %v", got)
	}
}

func TestParseAddedLinesFromDiff_PreExistingNotFlagged(t *testing.T) {
	// This is the key scenario: file has pre-existing fmt.Println,
	// agent only adds a new function — pre-existing debug should NOT be flagged
	fmtPrintPattern := debugPatterns[2].Pattern

	diff := `diff --git a/daemon.go b/daemon.go
--- a/daemon.go
+++ b/daemon.go
@@ -50,2 +50,6 @@ func (d *Daemon) Run() {
 	d.start()
+	// New feature added by agent
+	if d.config.Verbose {
+		log.Printf("daemon started")
+	}
`
	got := parseAddedLinesFromDiff(diff, fmtPrintPattern)
	if len(got) != 0 {
		t.Errorf("expected no matches (pre-existing fmt.Print not in diff), got %v", got)
	}
}
