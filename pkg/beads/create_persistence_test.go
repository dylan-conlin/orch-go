package beads

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeBDCreateShowScript(t *testing.T, createOutput, showOutput string) string {
	t.Helper()

	scriptPath := filepath.Join(t.TempDir(), "fake-bd-create-show.sh")
	script := fmt.Sprintf(`#!/bin/sh
while [ "$1" = "--sandbox" ] || [ "$1" = "--quiet" ] || [ "$1" = "-q" ]; do
  shift
done
cmd="$1"
case "$cmd" in
  create)
    cat <<'EOF_CREATE'
%s
EOF_CREATE
    ;;
  show)
    cat <<'EOF_SHOW'
%s
EOF_SHOW
    ;;
  *)
    echo "unexpected command: $cmd" >&2
    exit 1
    ;;
esac
`, createOutput, showOutput)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	return scriptPath
}

func writeFakeBDCreateShowScriptWithCausedBy(t *testing.T, expectedCausedBy, createOutput, showOutput string) string {
	t.Helper()

	scriptPath := filepath.Join(t.TempDir(), "fake-bd-create-show-caused-by.sh")
	script := fmt.Sprintf(`#!/bin/sh
while [ "$1" = "--sandbox" ] || [ "$1" = "--quiet" ] || [ "$1" = "-q" ]; do
  shift
done
cmd="$1"
shift
case "$cmd" in
  create)
    found=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--caused-by" ]; then
        shift
        if [ "$1" != "%s" ]; then
          echo "unexpected --caused-by value: $1" >&2
          exit 2
        fi
        found="1"
        break
      fi
      shift
    done
    if [ -z "$found" ]; then
      echo "missing --caused-by flag" >&2
      exit 2
    fi
    cat <<'EOF_CREATE'
%s
EOF_CREATE
    ;;
  show)
    cat <<'EOF_SHOW'
%s
EOF_SHOW
    ;;
  *)
    echo "unexpected command: $cmd" >&2
    exit 1
    ;;
esac
`, expectedCausedBy, createOutput, showOutput)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	return scriptPath
}

func TestFallbackCreateWithParent_FailsWhenIssueNotPersisted(t *testing.T) {
	createOutput := `{"id":"orch-go-test123","title":"Test issue","status":"open","priority":2,"issue_type":"task"}`
	showOutput := `[]`
	scriptPath := writeFakeBDCreateShowScript(t, createOutput, showOutput)

	originalBdPath := BdPath
	originalDefaultDir := DefaultDir
	BdPath = scriptPath
	DefaultDir = t.TempDir()
	t.Cleanup(func() {
		BdPath = originalBdPath
		DefaultDir = originalDefaultDir
	})

	issue, err := FallbackCreateWithParent("Test issue", "desc", "task", 2, []string{"triage:review"}, "")
	if err == nil {
		t.Fatalf("expected error when created issue is missing from read-back, got issue=%+v", issue)
	}
	if !strings.Contains(err.Error(), "not persisted") {
		t.Fatalf("expected persistence error, got: %v", err)
	}
}

func TestFallbackCreateWithParent_SucceedsWhenIssuePersisted(t *testing.T) {
	createOutput := `{"id":"orch-go-test456","title":"Persisted issue","status":"open","priority":2,"issue_type":"task"}`
	showOutput := `[{"id":"orch-go-test456","title":"Persisted issue","status":"open","priority":2,"issue_type":"task"}]`
	scriptPath := writeFakeBDCreateShowScript(t, createOutput, showOutput)

	originalBdPath := BdPath
	originalDefaultDir := DefaultDir
	BdPath = scriptPath
	DefaultDir = t.TempDir()
	t.Cleanup(func() {
		BdPath = originalBdPath
		DefaultDir = originalDefaultDir
	})

	issue, err := FallbackCreateWithParent("Persisted issue", "desc", "task", 2, []string{"triage:review"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil || issue.ID != "orch-go-test456" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
}

func TestCLIClientCreate_FailsWhenIssueNotPersisted(t *testing.T) {
	createOutput := `{"id":"orch-go-test789","title":"CLI test issue","status":"open","priority":2,"issue_type":"task"}`
	showOutput := `[]`
	scriptPath := writeFakeBDCreateShowScript(t, createOutput, showOutput)

	client := NewCLIClient(
		WithBdPath(scriptPath),
		WithWorkDir(t.TempDir()),
	)

	issue, err := client.Create(&CreateArgs{Title: "CLI test issue", IssueType: "task", Priority: 2})
	if err == nil {
		t.Fatalf("expected error when created issue is missing from read-back, got issue=%+v", issue)
	}
	if !strings.Contains(err.Error(), "not persisted") {
		t.Fatalf("expected persistence error, got: %v", err)
	}
}

func TestFallbackCreateWithParentAndCause_PassesCausedByFlag(t *testing.T) {
	createOutput := `{"id":"orch-go-regression1","title":"Regression bug","status":"open","priority":1,"issue_type":"bug","caused_by":"orch-go-21149"}`
	showOutput := `[{"id":"orch-go-regression1","title":"Regression bug","status":"open","priority":1,"issue_type":"bug","caused_by":"orch-go-21149"}]`
	scriptPath := writeFakeBDCreateShowScriptWithCausedBy(t, "orch-go-21149", createOutput, showOutput)

	originalBdPath := BdPath
	originalDefaultDir := DefaultDir
	BdPath = scriptPath
	DefaultDir = t.TempDir()
	t.Cleanup(func() {
		BdPath = originalBdPath
		DefaultDir = originalDefaultDir
	})

	issue, err := FallbackCreateWithParentAndCause("Regression bug", "desc", "bug", 1, []string{"triage:ready"}, "", "orch-go-21149")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil || issue.ID != "orch-go-regression1" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
	if issue.CausedBy != "orch-go-21149" {
		t.Fatalf("expected caused_by to round-trip, got %q", issue.CausedBy)
	}
}

func TestCLIClientCreate_PassesCausedByFlag(t *testing.T) {
	createOutput := `{"id":"orch-go-regression2","title":"Regression bug","status":"open","priority":1,"issue_type":"bug","caused_by":"orch-go-21149"}`
	showOutput := `[{"id":"orch-go-regression2","title":"Regression bug","status":"open","priority":1,"issue_type":"bug","caused_by":"orch-go-21149"}]`
	scriptPath := writeFakeBDCreateShowScriptWithCausedBy(t, "orch-go-21149", createOutput, showOutput)

	client := NewCLIClient(
		WithBdPath(scriptPath),
		WithWorkDir(t.TempDir()),
	)

	issue, err := client.Create(&CreateArgs{
		Title:     "Regression bug",
		IssueType: "bug",
		Priority:  1,
		CausedBy:  "orch-go-21149",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil || issue.ID != "orch-go-regression2" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
	if issue.CausedBy != "orch-go-21149" {
		t.Fatalf("expected caused_by to round-trip, got %q", issue.CausedBy)
	}
}
