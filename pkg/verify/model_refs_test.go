package verify

import "testing"

func TestExtractCodeRefsBlock(t *testing.T) {
	content := "# Model Title\n\n" +
		"<!-- code_refs: machine-parseable file references for staleness detection -->\n" +
		"- `cmd/orch/complete_cmd.go`\n" +
		"- `pkg/verify/check.go:123`\n" +
		"- `pkg/verify/git_diff.go#L42`\n" +
		"<!-- /code_refs -->\n" +
		"Outside block: `should/not/be/included.go`\n"

	got := extractCodeRefsBlock(content)
	want := []string{
		"cmd/orch/complete_cmd.go",
		"pkg/verify/check.go",
		"pkg/verify/git_diff.go",
	}

	if len(got) != len(want) {
		t.Fatalf("extractCodeRefsBlock() length = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("extractCodeRefsBlock()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMatchModifiedFilesToModelRefs(t *testing.T) {
	refsByFile := map[string][]string{
		"cmd/orch/complete_cmd.go": {".kb/models/completion-verification.md"},
		"pkg/verify/check.go":      {".kb/models/completion-verification.md", ".kb/models/agent-lifecycle.md"},
	}
	modified := []string{
		"./pkg/verify/check.go",
		"cmd/orch/complete_cmd.go",
		"pkg/verify/other.go",
	}

	matches := matchModifiedFilesToModelRefs(modified, refsByFile)
	if len(matches) != 2 {
		t.Fatalf("matchModifiedFilesToModelRefs() length = %d, want 2", len(matches))
	}

	if matches[0].File != "cmd/orch/complete_cmd.go" {
		t.Fatalf("matches[0].File = %q, want cmd/orch/complete_cmd.go", matches[0].File)
	}
	if matches[1].File != "pkg/verify/check.go" {
		t.Fatalf("matches[1].File = %q, want pkg/verify/check.go", matches[1].File)
	}

	if len(matches[1].Models) != 2 {
		t.Fatalf("matches[1].Models length = %d, want 2", len(matches[1].Models))
	}
}

func TestFormatModelReferenceNote(t *testing.T) {
	matches := []ModelReferenceMatch{
		{
			File:   "cmd/orch/complete_cmd.go",
			Models: []string{".kb/models/completion-verification.md"},
		},
	}

	got := FormatModelReferenceNote(matches)
	want := "NOTE: Modified files referenced by models: cmd/orch/complete_cmd.go -> .kb/models/completion-verification.md. Consider updating affected models."
	if got != want {
		t.Fatalf("FormatModelReferenceNote() = %q, want %q", got, want)
	}
}
