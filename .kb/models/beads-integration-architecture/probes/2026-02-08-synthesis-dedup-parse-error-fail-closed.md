# Probe: Does synthesis dedup fail closed on JSON parse errors?

**Model:** `.kb/models/beads-integration-architecture/model.md`
**Date:** 2026-02-08
**Status:** Complete

---

## Question

When `kb reflect --create-issue` performs synthesis deduplication and `bd list --json` returns non-JSON output, does `synthesisIssueExists()` block issue creation (fail-closed) instead of allowing duplicates?

---

## What I Tested

**Command/Code:**
```bash
go test ./cmd/kb -run 'TestSynthesisIssueExists|TestSynthesisIssueExists_JSONParseErrorFailClosed|TestOpenIssueExists'
```

Added a targeted regression test in `kb-cli/cmd/kb/reflect_test.go` that:
- injects a fake `bd` binary via `PATH`
- makes `bd list --json` output `not-json`
- calls `synthesisIssueExists("auth", projectDir)`
- asserts `exists == true` and `err == nil`

**Environment:**
- Repo under test: `/Users/dylanconlin/Documents/personal/kb-cli`
- Branch state: dirty working tree with unrelated in-flight changes

---

## What I Observed

**Output:**
```text
ok  github.com/dylanconlin/kb-cli/cmd/kb  0.032s
```

**Key observations:**
- Parse-error path in `synthesisIssueExists()` now returns `true` (assume issue exists), preventing duplicate creation.
- Existing fail-safe behavior for command errors (`bd` unavailable) still passes.

---

## Model Impact

**Verdict:** confirms — Auto-tracking duplicate risk is mitigated when dedup checks fail.

**Details:**
This probe confirms dedup now fails closed under malformed JSON, which directly addresses a known duplicate-issue failure mode. Error handling now prefers false positives (skip creation) over false negatives (create duplicate synthesis issues).

**Confidence:** High — direct executable test exercises the exact parse-error branch.
