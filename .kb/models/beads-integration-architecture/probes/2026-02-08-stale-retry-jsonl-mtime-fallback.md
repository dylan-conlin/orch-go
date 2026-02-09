# Probe: Does stale retry trigger when import metadata is unavailable but JSONL is actively changing?

**Model:** .kb/models/beads-integration-architecture.md
**Date:** 2026-02-08
**Status:** Complete

---

## Question

When `bd` returns "Database out of sync with JSONL", does orch's CLI fallback retry with `--allow-stale` if `last_import_time` cannot be read but `.beads/issues.jsonl` was updated within the 30s grace window?

---

## What I Tested

**Command/Code:**
```bash
go test ./pkg/beads -run 'TestRunBDCommand_(RetriesAllowStaleWhenJSONLRecentlyUpdated|DoesNotRetryAllowStaleWhenJSONLNotRecent|RetriesAllowStaleWhenImportRecent)' -count=1 -v
```

**Environment:**
- Repo: `orch-go`
- Package under test: `pkg/beads`
- New path tested: JSONL mtime fallback in `shouldRetryWithAllowStale`

---

## What I Observed

**Output:**
```text
=== RUN   TestRunBDCommand_RetriesAllowStaleWhenImportRecent
event=bd_stale_grace_retry component=beads source=last_import_time grace=30s
--- PASS: TestRunBDCommand_RetriesAllowStaleWhenImportRecent
=== RUN   TestRunBDCommand_RetriesAllowStaleWhenJSONLRecentlyUpdated
event=bd_stale_grace_retry component=beads source=jsonl_mtime grace=30s
--- PASS: TestRunBDCommand_RetriesAllowStaleWhenJSONLRecentlyUpdated
=== RUN   TestRunBDCommand_DoesNotRetryAllowStaleWhenJSONLNotRecent
--- PASS: TestRunBDCommand_DoesNotRetryAllowStaleWhenJSONLNotRecent
PASS
```

**Key observations:**
- Retry now triggers from two independent freshness signals: `last_import_time` OR hot JSONL mtime.
- Behavior still fails closed when neither signal is fresh (no blind `--allow-stale` retries).

---

## Model Impact

**Verdict:** extends — RPC-first + CLI fallback resilience under concurrent write contention

**Details:**
The model already describes silent fallback and out-of-sync friction under concurrency. This probe extends it with a second grace signal: active JSONL writes can now authorize a safe `--allow-stale` retry even when import metadata is unreadable/missing. This reduces false-negative failures in concurrent agent sessions without making stale bypass unconditional.

**Confidence:** High — targeted tests cover both positive and negative branches with concrete command output.
