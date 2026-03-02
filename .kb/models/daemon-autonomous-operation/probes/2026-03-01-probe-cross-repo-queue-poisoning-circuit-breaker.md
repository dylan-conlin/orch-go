# Probe: Cross-Repo Queue Poisoning — Per-Issue Circuit Breaker

**Model:** daemon-autonomous-operation
**Date:** 2026-03-01
**Status:** Complete
**Issue:** orch-go-jcyl

---

## Question

The model documents "Capacity Starvation" as a failure mode (slots not released from failed spawns). Is there a different failure axis: **queue poisoning** from persistently-failing issues that retry every poll cycle forever?

---

## What I Tested

### 1. Spawn Failure Retry Behavior

Traced the data flow: `ListReadyIssuesMultiProject()` → `NextIssueExcluding()` → `spawnIssue()` → failure → rollback → retry.

On spawn failure, `spawnIssue()` rolls status back to "open" and unmarks from the spawn tracker. This makes the issue eligible again on the next 15-second poll cycle. The `skippedThisCycle` map resets every cycle — no persistent memory.

### 2. Global vs Per-Issue Tracking

`SpawnFailureTracker` tracked only global metrics (consecutive failures, total failures). `CompletionFailureTracker` had a per-threshold circuit breaker (threshold=3), but `SpawnFailureTracker` did not. No per-issue memory existed anywhere.

### 3. Cross-Repo Trigger

`ListReadyIssuesMultiProject()` queries all 19 registered projects. Issues from other repos get `ProjectDir` set and spawn via `orch work --workdir`. When the workdir doesn't exist or the issue can't be resolved locally, spawn fails every time — creating an infinite 15-second retry loop.

---

## Findings

**Confirmed: queue poisoning is a distinct failure axis from capacity starvation.**

- Capacity starvation = slots consumed but never released (global resource exhaustion)
- Queue poisoning = specific issues fail every cycle, consuming CPU/IO but not slots (per-issue infinite loop)

The fix adds per-issue spawn failure tracking to `SpawnFailureTracker`. After `MaxIssueFailures` (default: 3) consecutive failures for a single issue, that issue is circuit-broken and skipped in `NextIssueExcluding()`. Successful spawn clears the counter, allowing retry after transient failures resolve.

---

## Model Update

**Extends** the "Capacity Starvation" failure mode section. The model should document two failure axes:

1. **Slot starvation** (existing): Failed spawns don't release pool slots → capacity exhaustion
2. **Queue poisoning** (new): Persistently-failing issues retry every poll cycle → infinite loop, log spam, wasted IO

Both now have circuit breakers: global (`CompletionFailureTracker`) and per-issue (`SpawnFailureTracker.issueFailures`).

---

## Test Evidence

```
$ go test ./pkg/daemon/ -run TestSpawnFailureTracker -v
=== RUN   TestSpawnFailureTracker_PerIssueCircuitBreaker
--- PASS
=== RUN   TestSpawnFailureTracker_ClearIssueFailures
--- PASS
=== RUN   TestSpawnFailureTracker_PerIssueIndependent
--- PASS
=== RUN   TestSpawnFailureTracker_CustomThreshold
--- PASS
=== RUN   TestSpawnFailureTracker_CircuitBrokenIssues
--- PASS
=== RUN   TestSpawnFailureTracker_SnapshotIncludesCircuitBroken
--- PASS
=== RUN   TestSpawnFailureTracker_IssueFailureCount
--- PASS
=== RUN   TestDaemon_NextIssueExcluding_SkipsCircuitBrokenIssues
--- PASS
=== RUN   TestDaemon_NextIssueExcluding_NoIssuesWhenAllCircuitBroken
--- PASS
=== RUN   TestDaemon_spawnIssue_RecordsPerIssueFailure
--- PASS
=== RUN   TestDaemon_spawnIssue_ClearsFailuresOnSuccess
--- PASS
PASS (11/11)
```

## Discovered Issue

24 daemon tests fail due to `~/.orch/halt` file (control plane circuit breaker). `OnceExcluding()` checks `control.HaltStatus()` which reads from `~/.orch/halt` — a real file, making tests environment-dependent. Predates this change.
