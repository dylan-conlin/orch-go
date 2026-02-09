# Probe: Does stale retry trigger when bd exits 0 but returns JSON error payload?

**Model:** .kb/models/beads-integration-architecture.md
**Date:** 2026-02-09
**Status:** Complete

---

## Question

When `bd` reports staleness as a JSON payload (`{"error":"Database out of sync with JSONL"}`) with exit code 0, does orch still detect out-of-sync state and retry with `--allow-stale`?

---

## What I Tested

**Command/Code:**

```bash
go test ./pkg/beads -run 'TestRunBDCommand_RetriesAllowStaleWhenOutOfSyncJSONErrorPayload|TestRunBDCommand_RetriesAllowStaleWhenImportRecent|TestRunBDCommand_RetriesAllowStaleWhenJSONLRecentlyUpdated' -count=1 -v
go test ./pkg/beads -count=1
```

**Environment:**

- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Code under test: `pkg/beads/runBDCommand` retry gate in `shouldRetryWithAllowStale`
- New coverage: success-path JSON error payload parsing in `outputErrorMessage` + `isOutOfSyncFailure`

---

## What I Observed

**Output:**

```text
=== RUN   TestRunBDCommand_RetriesAllowStaleWhenOutOfSyncJSONErrorPayload
event=bd_stale_grace_retry component=beads source=last_import_time grace=30s
--- PASS: TestRunBDCommand_RetriesAllowStaleWhenOutOfSyncJSONErrorPayload (0.01s)
PASS
ok   github.com/dylan-conlin/orch-go/pkg/beads  0.066s

ok   github.com/dylan-conlin/orch-go/pkg/beads  11.145s
```

**Key observations:**

- Retry now happens even when the first `bd` call returns no process error but the JSON payload contains out-of-sync text.
- Existing recent-import and JSONL-mtime retry paths still pass.

---

## Model Impact

**Verdict:** extends — stale detection must inspect both process errors and successful JSON payloads

**Details:**
This extends the model's CLI fallback resilience invariant: out-of-sync detection is now transport-agnostic (non-zero exit OR structured JSON error). That closes a silent failure mode where `bd sync --import-only` looked successful but subsequent reads still surfaced staleness without triggering retry.

**Confidence:** High — validated by targeted regression plus full package tests.
