# Probe: Does ProcessedIssueCache mark issues only after confirmed successful spawn?

**Model:** `.kb/models/daemon-autonomous-operation.md`
**Date:** 2026-02-08
**Status:** Complete

---

## Question

When daemon spawn evaluation selects an issue, is `ProcessedIssueCache` updated only after a successful spawn, so rejected/failed spawns remain retryable?

---

## What I Tested

**Command/Code:**
```bash
# Added regression tests asserting cache state during spawn and after failure/success
go test ./pkg/daemon -run 'ProcessedCacheMarkedAfterSuccessfulSpawn|ProcessedCacheNotMarkedOnSpawnFailure'

# Repro behavior for original report: rejected issue not cached, becomes spawnable after label fix
go test ./pkg/daemon -run 'RejectedIssueNotCached'

# Safety sweep for daemon package
go test ./pkg/daemon
```

**Environment:**
- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Branch workspace: `og-debug-daemon-processedissuecache-marks-08feb-35ee`

---

## What I Observed

**Output:**
```text
Pre-fix behavior (new tests failed):
--- FAIL: TestDaemon_OnceExcluding_ProcessedCacheMarkedAfterSuccessfulSpawn
--- FAIL: TestDaemon_OnceExcluding_ProcessedCacheNotMarkedOnSpawnFailure
--- FAIL: TestDaemon_CrossProjectOnceExcluding_ProcessedCacheMarkedAfterSuccessfulSpawn
--- FAIL: TestDaemon_CrossProjectOnceExcluding_ProcessedCacheNotMarkedOnSpawnFailure

Post-fix behavior:
ok   github.com/dylan-conlin/orch-go/pkg/daemon 0.066s
ok   github.com/dylan-conlin/orch-go/pkg/daemon 5.544s
```

**Key observations:**
- Before the fix, `ProcessedIssueCache` was already blocking the issue while spawn execution was still in-flight, proving mark-before-success ordering.
- After moving `MarkProcessed` to post-success in both single-project and cross-project paths, cache-ordering regression tests and full daemon tests passed.
- Rejected issues (missing required label) were not cached and became spawnable immediately after label correction.

---

## Model Impact

**Verdict:** extends — dedup invariants for daemon spawn cycle

**Details:**
The model's dedup narrative is correct at a high level, but this adds a stricter invariant: persistent dedup (`ProcessedIssueCache`) must be written only after confirmed spawn success, while pre-spawn race protection should remain in transient tracking (`SpawnedIssues`). This prevents stale 30-day cache blocks from rejected or failed attempts.

**Confidence:** High — behavior was demonstrated by failing regression tests pre-fix and passing tests post-fix, including full daemon package coverage.
