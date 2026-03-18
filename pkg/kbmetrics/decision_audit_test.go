package kbmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDecisionStatus(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "frontmatter status accepted",
			content: "---\nstatus: accepted\n---\n# Decision\n",
			want:    "accepted",
		},
		{
			name:    "body status accepted",
			content: "# Decision\n\n**Status:** Accepted\n",
			want:    "accepted",
		},
		{
			name:    "body status with extra text",
			content: "# Decision\n\n**Status:** Partially Implemented (reviewed 2026-03-12)\n",
			want:    "partially implemented",
		},
		{
			name:    "body status superseded",
			content: "# Decision\n\n**Status:** Superseded (partially)\n",
			want:    "superseded",
		},
		{
			name:    "frontmatter status proposed",
			content: "---\nstatus: proposed\n---\n",
			want:    "proposed",
		},
		{
			name:    "no status found",
			content: "# Decision\n\nSome content\n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDecisionStatus(tt.content)
			if got != tt.want {
				t.Errorf("parseDecisionStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFileReferences(t *testing.T) {
	content := `# Decision

**Source Investigation:** ` + "`" + `.kb/investigations/2026-01-11-inv-foo.md` + "`" + `
**Code:** ` + "`" + `cmd/orch/clean_cmd.go:archiveStaleWorkspaces()` + "`" + `
**Guide:** ` + "`" + `.kb/guides/workspace-lifecycle.md` + "`" + `

- **Source:** ` + "`" + `.kb/investigations/2026-01-13-design-foo.md` + "`" + `
- **Code:** ` + "`" + `pkg/registry/registry.go` + "`" + `

## Consequences

Updated ` + "`" + `cmd/orch/main.go` + "`" + ` and ` + "`" + `pkg/model/resolve.go` + "`" + `
`

	refs := extractFileReferences(content)

	// Should find multiple references
	if len(refs) == 0 {
		t.Fatal("expected file references, got none")
	}

	// Check some expected references
	found := make(map[string]bool)
	for _, r := range refs {
		found[r] = true
	}

	expected := []string{
		".kb/investigations/2026-01-11-inv-foo.md",
		"cmd/orch/clean_cmd.go",
		".kb/guides/workspace-lifecycle.md",
		".kb/investigations/2026-01-13-design-foo.md",
		"pkg/registry/registry.go",
		"cmd/orch/main.go",
		"pkg/model/resolve.go",
	}

	for _, e := range expected {
		if !found[e] {
			t.Errorf("missing expected reference: %s", e)
		}
	}
}

func TestAuditDecisions(t *testing.T) {
	// Set up temp dir with decision files
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, ".kb")
	decisionsDir := filepath.Join(kbDir, "decisions")
	os.MkdirAll(decisionsDir, 0o755)

	// Create a referenced file so we can test existence checking
	guidesDir := filepath.Join(kbDir, "guides")
	os.MkdirAll(guidesDir, 0o755)
	os.WriteFile(filepath.Join(guidesDir, "existing-guide.md"), []byte("# Guide"), 0o644)

	// Create source code file
	cmdDir := filepath.Join(tmpDir, "cmd", "orch")
	os.MkdirAll(cmdDir, 0o755)
	os.WriteFile(filepath.Join(cmdDir, "foo_cmd.go"), []byte("package main"), 0o644)

	// Decision 1: Accepted, references exist
	os.WriteFile(filepath.Join(decisionsDir, "2026-01-01-good-decision.md"), []byte(`# Decision: Good

**Status:** Accepted

**Guide:** `+"`"+`.kb/guides/existing-guide.md`+"`"+`
**Code:** `+"`"+`cmd/orch/foo_cmd.go`+"`"+`
`), 0o644)

	// Decision 2: Accepted, references missing
	os.WriteFile(filepath.Join(decisionsDir, "2026-01-02-stale-decision.md"), []byte(`# Decision: Stale

**Status:** Accepted

**Code:** `+"`"+`pkg/deleted/gone.go`+"`"+`
**Guide:** `+"`"+`.kb/guides/nonexistent.md`+"`"+`
`), 0o644)

	// Decision 3: Proposed - should be skipped
	os.WriteFile(filepath.Join(decisionsDir, "2026-01-03-proposed.md"), []byte(`# Decision: Proposed

**Status:** Proposed

**Code:** `+"`"+`pkg/deleted/gone.go`+"`"+`
`), 0o644)

	// Decision 4: Superseded - should be skipped
	os.WriteFile(filepath.Join(decisionsDir, "2026-01-04-superseded.md"), []byte(`# Decision: Superseded

**Status:** Superseded
`), 0o644)

	reports, err := AuditDecisions(tmpDir)
	if err != nil {
		t.Fatalf("AuditDecisions() error: %v", err)
	}

	// Should only audit Accepted decisions
	if reports.TotalDecisions != 4 {
		t.Errorf("TotalDecisions = %d, want 4", reports.TotalDecisions)
	}
	if reports.AcceptedDecisions != 2 {
		t.Errorf("AcceptedDecisions = %d, want 2", reports.AcceptedDecisions)
	}

	// Find the stale decision
	var staleReport *DecisionAuditEntry
	for i, r := range reports.Entries {
		if r.Name == "2026-01-02-stale-decision.md" {
			staleReport = &reports.Entries[i]
			break
		}
	}
	if staleReport == nil {
		t.Fatal("expected to find stale decision entry")
	}
	if len(staleReport.MissingFiles) == 0 {
		t.Error("expected missing files for stale decision")
	}
	if len(staleReport.MissingFiles) != 2 {
		t.Errorf("expected 2 missing files, got %d", len(staleReport.MissingFiles))
	}

	// Find the good decision — should have 0 missing files
	var goodReport *DecisionAuditEntry
	for i, r := range reports.Entries {
		if r.Name == "2026-01-01-good-decision.md" {
			goodReport = &reports.Entries[i]
			break
		}
	}
	if goodReport == nil {
		t.Fatal("expected to find good decision entry")
	}
	if len(goodReport.MissingFiles) != 0 {
		t.Errorf("expected 0 missing files for good decision, got %d: %v", len(goodReport.MissingFiles), goodReport.MissingFiles)
	}
}

func TestAuditDecisionsGlobalDir(t *testing.T) {
	// Verify that global decisions are also scanned
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, ".kb")
	decisionsDir := filepath.Join(kbDir, "decisions")
	globalDecisionsDir := filepath.Join(kbDir, "global", "decisions")
	os.MkdirAll(decisionsDir, 0o755)
	os.MkdirAll(globalDecisionsDir, 0o755)

	os.WriteFile(filepath.Join(decisionsDir, "2026-01-01-local.md"), []byte(`# Decision
**Status:** Accepted
`), 0o644)

	os.WriteFile(filepath.Join(globalDecisionsDir, "2026-01-01-global.md"), []byte(`# Decision
**Status:** Accepted
`), 0o644)

	reports, err := AuditDecisions(tmpDir)
	if err != nil {
		t.Fatalf("AuditDecisions() error: %v", err)
	}
	if reports.TotalDecisions != 2 {
		t.Errorf("TotalDecisions = %d, want 2", reports.TotalDecisions)
	}
}

func TestDecisionWithFrontmatterPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, ".kb")
	decisionsDir := filepath.Join(kbDir, "decisions")
	os.MkdirAll(decisionsDir, 0o755)

	// Create the files referenced in patterns
	gatesDir := filepath.Join(tmpDir, "pkg", "spawn", "gates")
	os.MkdirAll(gatesDir, 0o755)
	os.WriteFile(filepath.Join(gatesDir, "hotspot.go"), []byte("package gates"), 0o644)

	os.WriteFile(filepath.Join(decisionsDir, "2026-01-01-with-patterns.md"), []byte(`---
status: accepted
blocks:
  - patterns:
      - "**/spawn/gates/hotspot*"
---
# Decision

**Status:** Accepted
`), 0o644)

	reports, err := AuditDecisions(tmpDir)
	if err != nil {
		t.Fatalf("AuditDecisions() error: %v", err)
	}
	if reports.AcceptedDecisions != 1 {
		t.Errorf("AcceptedDecisions = %d, want 1", reports.AcceptedDecisions)
	}
}
