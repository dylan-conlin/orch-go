package dupdetect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllowlist_SuppressesBothMatch(t *testing.T) {
	// When both functions in a pair match the same allowlist pattern,
	// the pair should be suppressed.
	src := `package foo
func LogSpawn() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func LogCompleted() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
`
	d := NewDetector()
	d.MinBodyLines = 2
	d.Allowlist = []string{"Log*"}

	funcs, err := d.ParseSource("logger.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	pairs := d.FindDuplicates(funcs)
	if len(pairs) != 0 {
		t.Errorf("expected allowlisted pair to be suppressed, got %d pairs", len(pairs))
	}
}

func TestAllowlist_DoesNotSuppressOneSideMatch(t *testing.T) {
	// When only one function matches the allowlist, the pair is still reported.
	src := `package foo
func LogSpawn() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func processItems() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
`
	d := NewDetector()
	d.MinBodyLines = 2
	d.Allowlist = []string{"Log*"}

	funcs, err := d.ParseSource("test.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	pairs := d.FindDuplicates(funcs)
	if len(pairs) == 0 {
		t.Fatal("expected pair to be reported when only one side matches allowlist")
	}
}

func TestAllowlist_MethodReceiverPattern(t *testing.T) {
	// Patterns should match against full function names including receiver.
	src := `package foo

type Logger struct{}

func (l *Logger) LogSpawn() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func (l *Logger) LogCompleted() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
`
	d := NewDetector()
	d.MinBodyLines = 2
	d.Allowlist = []string{"(Logger).Log*"}

	funcs, err := d.ParseSource("logger.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	pairs := d.FindDuplicates(funcs)
	if len(pairs) != 0 {
		t.Errorf("expected allowlisted method pair to be suppressed, got %d pairs", len(pairs))
	}
}

func TestAllowlist_MultiplePatterns(t *testing.T) {
	// Multiple allowlist patterns, each creates its own suppression group.
	src := `package foo

type Logger struct{}
type Adapter struct{}

func (l *Logger) LogA() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func (l *Logger) LogB() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func (a *Adapter) HandleA() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func (a *Adapter) HandleB() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
`
	d := NewDetector()
	d.MinBodyLines = 2
	d.Allowlist = []string{"(Logger).Log*", "(Adapter).Handle*"}

	funcs, err := d.ParseSource("test.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	pairs := d.FindDuplicates(funcs)

	// Logger.LogA vs Logger.LogB — suppressed (both match pattern 1)
	// Adapter.HandleA vs Adapter.HandleB — suppressed (both match pattern 2)
	// Logger.LogA vs Adapter.HandleA — NOT suppressed (match different patterns)
	if len(pairs) != 4 {
		// Cross-group pairs: LogA-HandleA, LogA-HandleB, LogB-HandleA, LogB-HandleB
		t.Errorf("expected 4 cross-group pairs, got %d", len(pairs))
		for _, p := range pairs {
			t.Logf("  %s vs %s (%.0f%%)", p.FuncA.Name, p.FuncB.Name, p.Similarity*100)
		}
	}
}

func TestAllowlist_EmptyAllowlist(t *testing.T) {
	// Empty allowlist means nothing is suppressed.
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
	z := y * 3
	println(z)
}
`
	d := NewDetector()
	d.MinBodyLines = 2

	funcs, err := d.ParseSource("test.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	pairs := d.FindDuplicates(funcs)
	if len(pairs) == 0 {
		t.Fatal("expected pair to be detected with empty allowlist")
	}
}

func TestLoadAllowlistFile(t *testing.T) {
	dir := t.TempDir()
	content := `# Logger methods are intentionally parallel
(Logger).Log*

# Adapter methods too
(EventLoggerAdapter).Log*

# blank lines and comments are ignored
`
	os.WriteFile(filepath.Join(dir, ".dupdetectignore"), []byte(content), 0644)

	patterns, err := LoadAllowlistFile(dir)
	if err != nil {
		t.Fatalf("LoadAllowlistFile failed: %v", err)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d: %v", len(patterns), patterns)
	}
	if patterns[0] != "(Logger).Log*" {
		t.Errorf("expected first pattern '(Logger).Log*', got %q", patterns[0])
	}
	if patterns[1] != "(EventLoggerAdapter).Log*" {
		t.Errorf("expected second pattern '(EventLoggerAdapter).Log*', got %q", patterns[1])
	}
}

func TestLoadAllowlistFile_NoFile(t *testing.T) {
	dir := t.TempDir()
	patterns, err := LoadAllowlistFile(dir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("expected empty patterns for missing file, got %v", patterns)
	}
}

func TestAllowlist_IntegrationWithScanProject(t *testing.T) {
	// Full integration: ScanProject with allowlist should suppress matching pairs.
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	os.MkdirAll(pkgDir, 0755)

	src := `package pkg

type Logger struct{}

func (l *Logger) LogA() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func (l *Logger) LogB() {
	x := 1
	y := x + 2
	z := y * 3
	println(z)
}
func standalone() {
	a := 10
	b := a + 20
	c := b * 30
	println(c)
}
`
	os.WriteFile(filepath.Join(pkgDir, "logger.go"), []byte(src), 0644)

	// Write allowlist file
	os.WriteFile(filepath.Join(dir, ".dupdetectignore"), []byte("(Logger).Log*\n"), 0644)

	d := NewDetector()
	d.MinBodyLines = 2

	// Load allowlist
	patterns, err := LoadAllowlistFile(dir)
	if err != nil {
		t.Fatalf("LoadAllowlistFile failed: %v", err)
	}
	d.Allowlist = patterns

	pairs, err := d.ScanProject(dir)
	if err != nil {
		t.Fatalf("ScanProject failed: %v", err)
	}

	// Logger.LogA vs Logger.LogB should be suppressed
	// Logger.LogA vs standalone and Logger.LogB vs standalone should remain
	for _, p := range pairs {
		if (p.FuncA.Name == "(Logger).LogA" && p.FuncB.Name == "(Logger).LogB") ||
			(p.FuncA.Name == "(Logger).LogB" && p.FuncB.Name == "(Logger).LogA") {
			t.Errorf("expected Logger pair to be suppressed, found %s vs %s", p.FuncA.Name, p.FuncB.Name)
		}
	}
}
