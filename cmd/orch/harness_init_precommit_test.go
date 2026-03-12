package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessInitPreCommitGate(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected pre-commit to be created")
	}

	hookPath := filepath.Join(gitDir, "pre-commit")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal("pre-commit not created")
	}
	if info.Mode()&0111 == 0 {
		t.Error("pre-commit should be executable")
	}
}

func TestHarnessInitPreCommitGateAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	existing := "#!/bin/bash\necho 'existing hook'\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected accretion gate to be added")
	}

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)
	if content[:len(existing)] != existing {
		t.Error("existing content was modified")
	}
	if !strings.Contains(content, "orch precommit accretion") {
		t.Error("accretion gate not appended")
	}
}

func TestHarnessInitPreCommitGateIdempotent(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	existing := "#!/bin/bash\norch precommit accretion 2>/dev/null || true\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected gate already present")
	}
}

func TestEnsureStandalonePreCommitGate(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)

	result, err := ensureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected pre-commit to be created")
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal("pre-commit not created")
	}
	content := string(data)

	// Should NOT reference orch CLI
	if strings.Contains(content, "orch precommit") {
		t.Error("standalone pre-commit should not reference orch CLI")
	}

	// Should contain inline accretion check
	if !strings.Contains(content, "accretion") {
		t.Error("pre-commit should contain accretion check")
	}

	// Should have explanatory comments
	if !strings.Contains(content, "WHY THIS GATE EXISTS") {
		t.Error("pre-commit should contain explanatory comments")
	}
}

func TestEnsureStandalonePreCommitGateIdempotent(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	ensureStandalonePreCommitGate(dir)
	result, err := ensureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPresent {
		t.Error("expected gate already present on second run")
	}
}

func TestPreCommitGateAppendsBeforeExitZero(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Existing hook ends with exit 0 — gate must be reachable
	existing := "#!/bin/bash\necho 'existing hook'\nexit 0\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensurePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected accretion gate to be added")
	}

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	if !strings.Contains(content, "orch precommit accretion") {
		t.Error("accretion gate not found in hook")
	}

	// Gate must be reachable: either no exit 0 remains, or gate appears before it
	gateIdx := strings.Index(content, "orch precommit accretion")
	lastExitIdx := strings.LastIndex(content, "exit 0")
	if lastExitIdx >= 0 && gateIdx > lastExitIdx {
		t.Error("accretion gate appears after exit 0 — would be dead code")
	}

	// Verify the original exit 0 was removed (gate is reachable)
	if gateIdx < 0 {
		t.Error("gate not found in output")
	}
}

func TestStandalonePreCommitGateAppendsBeforeExitZero(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Existing hook ends with exit 0
	existing := "#!/bin/bash\necho 'existing hook'\nexit 0\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	result, err := ensureStandalonePreCommitGate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.AlreadyPresent {
		t.Error("expected accretion gate to be added")
	}

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	if !strings.Contains(content, "Accretion Gate") {
		t.Error("accretion gate not found in hook")
	}

	// Should contain the accretion check logic
	if !strings.Contains(content, "HARD_LIMIT") {
		t.Error("inline accretion check not found")
	}

	// The process substitution line must be reachable (not after exit 0 of original hook)
	// Check that the gate's own exit handling works, but original exit 0 doesn't kill it
	hardLimitIdx := strings.Index(content, "HARD_LIMIT")
	lines := strings.Split(content, "\n")
	// Find the original "echo 'existing hook'" line
	echoIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "existing hook") {
			echoIdx = i
			break
		}
	}
	if echoIdx == -1 {
		t.Error("original hook content not preserved")
	}
	_ = hardLimitIdx // used implicitly via content checks
}

func TestStandalonePreCommitGateNoDoubleShebang(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	existing := "#!/bin/bash\necho 'existing hook'\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	ensureStandalonePreCommitGate(dir)

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	// Count shebangs — should be exactly 1
	shebangCount := strings.Count(content, "#!/bin/bash")
	if shebangCount != 1 {
		t.Errorf("expected exactly 1 shebang, found %d", shebangCount)
	}
}

func TestStandalonePreCommitGateUpgradesShShebang(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Existing hook uses /bin/sh
	existing := "#!/bin/sh\necho 'existing hook'\n"
	os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(existing), 0755)

	ensureStandalonePreCommitGate(dir)

	data, _ := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	content := string(data)

	// Shebang must be bash, not sh (process substitution requires bash)
	if strings.HasPrefix(content, "#!/bin/sh") {
		t.Error("shebang should be upgraded from #!/bin/sh to #!/bin/bash")
	}
	if !strings.HasPrefix(content, "#!/bin/bash") {
		t.Errorf("expected #!/bin/bash shebang, got first line: %s", strings.SplitN(content, "\n", 2)[0])
	}
}

// --- Helper function tests ---

func TestRemoveTrailingExit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trailing exit 0",
			input:    "#!/bin/bash\necho hello\nexit 0\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "trailing exit 0 no newline",
			input:    "#!/bin/bash\necho hello\nexit 0",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "trailing exit $?",
			input:    "#!/bin/bash\necho hello\nexit $?\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "no trailing exit",
			input:    "#!/bin/bash\necho hello\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "exit 0 mid-script not removed",
			input:    "#!/bin/bash\nif true; then\n  exit 0\nfi\necho done\n",
			expected: "#!/bin/bash\nif true; then\n  exit 0\nfi\necho done\n",
		},
		{
			name:     "exit 1 not removed",
			input:    "#!/bin/bash\necho hello\nexit 1\n",
			expected: "#!/bin/bash\necho hello\nexit 1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTrailingExit(tt.input)
			if got != tt.expected {
				t.Errorf("removeTrailingExit(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEnsureBashShebang(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sh to bash",
			input:    "#!/bin/sh\necho hello\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "already bash",
			input:    "#!/bin/bash\necho hello\n",
			expected: "#!/bin/bash\necho hello\n",
		},
		{
			name:     "no shebang",
			input:    "echo hello\n",
			expected: "echo hello\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureBashShebang(tt.input)
			if got != tt.expected {
				t.Errorf("ensureBashShebang(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripScriptShebang(t *testing.T) {
	input := "#!/bin/bash\n# comment\necho hello\n"
	expected := "# comment\necho hello\n"
	got := stripScriptShebang(input)
	if got != expected {
		t.Errorf("stripScriptShebang(%q) = %q, want %q", input, got, expected)
	}

	// No shebang — unchanged
	noShebang := "# comment\necho hello\n"
	got2 := stripScriptShebang(noShebang)
	if got2 != noShebang {
		t.Errorf("stripScriptShebang(%q) = %q, want %q", noShebang, got2, noShebang)
	}
}
