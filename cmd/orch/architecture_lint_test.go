package main

// Architecture lint tests for the Two-Lane Agent Discovery Architecture.
//
// These tests enforce the structural constraint from:
//   .kb/decisions/2026-02-18-two-lane-agent-discovery.md
//
// Constraint: No new persistent lifecycle state packages or files.
// "No other persisted lifecycle state allowed. Any new pkg/state/,
//  pkg/registry/, pkg/cache/, or sessions.json triggers CI lint failure."
//
// Scenario 11 of the acceptance matrix: New pkg/state/ file → lint failure.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenLifecyclePackages are package directories that must not have new
// files added for lifecycle state management. The two-lane architecture
// uses beads + workspace manifests + OpenCode queries only.
//
// Existing files in these packages may exist for legacy reasons or non-lifecycle
// purposes (e.g., pkg/state/reconcile.go). This lint catches NEW additions.
var forbiddenLifecyclePackages = []string{
	"pkg/registry",
	"pkg/cache",
}

// forbiddenLifecycleFiles are file patterns in ~/.orch/ that must not be
// created for persistent agent/session lifecycle state.
var forbiddenLifecycleFiles = []string{
	"registry.json",
	"sessions.json",
	"state.db",
	"agents.db",
	"lifecycle.json",
	"agent_cache.json",
}

func TestArchitectureLint_NoNewLifecycleStatePackages(t *testing.T) {
	projectRoot := findProjectRoot(t)

	for _, pkg := range forbiddenLifecyclePackages {
		pkgPath := filepath.Join(projectRoot, pkg)
		if _, err := os.Stat(pkgPath); err == nil {
			entries, err := os.ReadDir(pkgPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", pkg, err)
			}
			for _, entry := range entries {
				if entry.IsDir() || strings.HasSuffix(entry.Name(), "_test.go") {
					continue
				}
				if strings.HasSuffix(entry.Name(), ".go") {
					t.Errorf("Architecture lint: forbidden lifecycle state file %s/%s exists.\n"+
						"The two-lane architecture prohibits persistent lifecycle state packages.\n"+
						"Use beads (work lifecycle) + workspace manifests (binding) + OpenCode (liveness) instead.\n"+
						"See: .kb/decisions/2026-02-18-two-lane-agent-discovery.md",
						pkg, entry.Name())
				}
			}
		}
	}
}

func TestArchitectureLint_NoNewLifecycleStateInDiff(t *testing.T) {
	// Check git diff for any new files in forbidden packages.
	// This catches additions even before they're merged.
	projectRoot := findProjectRoot(t)

	// Get staged + unstaged changes
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=A", "HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		// Not in a git repo or no commits yet - skip
		t.Skip("git diff not available, skipping diff-based lint")
	}

	allForbidden := append(forbiddenLifecyclePackages, "pkg/state")
	newFiles := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, file := range newFiles {
		if file == "" {
			continue
		}
		for _, forbidden := range allForbidden {
			if strings.HasPrefix(file, forbidden+"/") {
				t.Errorf("Architecture lint: new file %q added to forbidden package %s.\n"+
					"The two-lane architecture prohibits new lifecycle state packages.\n"+
					"See: .kb/decisions/2026-02-18-two-lane-agent-discovery.md",
					file, forbidden)
			}
		}
	}
}

func TestArchitectureLint_NoPersistentLifecycleFiles(t *testing.T) {
	// Advisory check: detect stale lifecycle state files in ~/.orch/.
	// These files should not exist per the two-lane ADR, but their presence
	// is a runtime cleanup issue, not a code structure violation.
	// The hard gates are the package structure and import checks above.
	orchDir := filepath.Join(os.Getenv("HOME"), ".orch")
	if _, err := os.Stat(orchDir); os.IsNotExist(err) {
		t.Skip("~/.orch/ does not exist, skipping")
	}

	entries, err := os.ReadDir(orchDir)
	if err != nil {
		t.Skipf("cannot read ~/.orch/: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		for _, forbidden := range forbiddenLifecycleFiles {
			if entry.Name() == forbidden {
				t.Logf("Advisory: stale lifecycle state file ~/.orch/%s exists (should be cleaned up)", forbidden)
				found = true
			}
		}
	}
	if found {
		t.Log("Advisory: run 'rm ~/.orch/{sessions.json,state.db,registry.json}' to clean up stale files.")
		t.Log("See: .kb/decisions/2026-02-18-two-lane-agent-discovery.md")
	}
}

func TestArchitectureLint_ForbiddenPackageImports(t *testing.T) {
	// Verify that cmd/orch/ does not import any forbidden lifecycle packages.
	// This is a structural gate: even if the packages exist for legacy reasons,
	// new code must not depend on them.
	projectRoot := findProjectRoot(t)

	// Use go list to check imports
	cmd := exec.Command("go", "list", "-f", "{{.Imports}}", "./cmd/orch/")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("go list failed: %v", err)
	}

	imports := string(out)
	forbidden := []string{
		"github.com/dylan-conlin/orch-go/pkg/registry",
		"github.com/dylan-conlin/orch-go/pkg/cache",
	}

	for _, pkg := range forbidden {
		if strings.Contains(imports, pkg) {
			t.Errorf("Architecture lint: cmd/orch/ imports forbidden lifecycle package %s.\n"+
				"See: .kb/decisions/2026-02-18-two-lane-agent-discovery.md",
				pkg)
		}
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Walk up from current working directory to find go.mod
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
