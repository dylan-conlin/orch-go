# Probe: Does CLI fallback auto-import JSONL before stale retries on create-then-read drift?

**Model:** /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
**Date:** 2026-02-09
**Status:** Complete

---

## Question

When create-then-read operations hit `Database out of sync with JSONL`, does orch's `runBDCommand` recover by running `bd sync --import-only` before falling back to `--allow-stale`?

---

## What I Tested

**Command/Code:**

```bash
# Reproduced baseline symptom in repo with direct bd commands
bd create "repro stale db loop 1 $(date +%s)" --type task -l triage:review
bd show <new-id>

# Validated new recovery behavior through beads fallback test surface
go test ./pkg/beads -run 'TestRunBDCommand_AutoSyncsBeforeAllowStaleRetry|TestRunBDCommand_RetriesAllowStaleWhenImportRecent|TestRunBDCommand_RetriesAllowStaleWhenJSONLRecentlyUpdated|TestRunBDCommand_RetriesAllowStaleWhenOutOfSyncJSONErrorPayload' -count=1 -v
go test ./pkg/beads -run 'TestRunBDCommand_CreateThenReadRecoversWithoutManualSync' -count=1 -v
go test ./pkg/beads -count=1
```

**Environment:**

- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Changed code: `pkg/beads/client.go`, `pkg/beads/client_stale_retry_test.go`
- New regression test covers successful recovery path where import-only heals staleness and avoids `--allow-stale`

---

## What I Observed

**Output (key lines):**

```text
show failed for orch-go-pcwuw
Error: Database out of sync with JSONL. Run 'bd sync --import-only' to fix.

=== RUN   TestRunBDCommand_AutoSyncsBeforeAllowStaleRetry
--- PASS: TestRunBDCommand_AutoSyncsBeforeAllowStaleRetry (0.01s)
=== RUN   TestRunBDCommand_CreateThenReadRecoversWithoutManualSync
--- PASS: TestRunBDCommand_CreateThenReadRecoversWithoutManualSync (0.01s)
=== RUN   TestRunBDCommand_RetriesAllowStaleWhenImportRecent
event=bd_stale_import_retry_failed component=beads error="exit status 1"
event=bd_stale_grace_retry component=beads source=last_import_time grace=30s
--- PASS: TestRunBDCommand_RetriesAllowStaleWhenImportRecent (0.01s)
PASS
ok   github.com/dylan-conlin/orch-go/pkg/beads  0.067s

ok   github.com/dylan-conlin/orch-go/pkg/beads  11.169s
```

---

## Model Impact

**Verdict:** extends — CLI fallback now attempts repair before stale-read bypass

**Details:**
This extends the model's staleness handling invariant: fallback does not jump straight to `--allow-stale`. It now first attempts `bd sync --import-only` and retries the original operation. If repair succeeds, stale bypass is avoided; if repair fails, existing grace-window `--allow-stale` logic still applies.

**Confidence:** High — baseline failure reproduced and fix behavior validated by targeted + full package tests.
