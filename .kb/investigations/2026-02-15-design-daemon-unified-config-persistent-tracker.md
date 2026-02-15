# Design: Daemon Unified Config Construction & Persistent VerificationTracker

**Date:** 2026-02-15
**Phase:** Complete
**Type:** Design Investigation
**Beads:** orch-go-xlvm

## Design Question

How should we eliminate Config construction divergence across daemon code paths and make the VerificationTracker counter survive daemon restarts?

## Problem Framing

### Success Criteria
1. New Config fields only need to be set in ONE place (DefaultConfig or flag-override layer)
2. All daemon paths (run, once, dry-run, preview) get consistent Config with same defaults
3. VerificationTracker counter reflects actual unverified backlog on startup
4. Recovery and rate limiting work in the production daemon loop (currently broken)

### Constraints
- Must be backward-compatible (no behavior changes for correct paths)
- Must work with current CLI flag structure (Cobra flags)
- Must work when daemon runs outside the main project (cross-project)
- Checkpoint file format must not change

### Scope
- **In:** Config construction, VerificationTracker persistence, RecoveryEnabled fix
- **Out:** New verification features, checkpoint file schema changes, daemon architecture changes

---

## Exploration: Fork Navigation

### Fork 1: Config Construction Pattern

**Options:**
- **A: Start from DefaultConfig(), override with flags** — All paths call `daemonConfigFromFlags()` which starts from `DefaultConfig()` and overrides only the fields that have corresponding CLI flags
- **B: Builder pattern** — `NewConfigBuilder().WithLabel(x).WithPollInterval(y).Build()`
- **C: Keep current pattern, add tests** — Test that all paths set required fields

**Substrate says:**
- Principle "Coherence over patches": Option A addresses the root cause (scattered construction), C is another patch
- Principle "Avoid over-engineering": Builder pattern (B) adds abstraction for 4 call sites that could be 1 function
- Decision "Pain as signal": The repeated dead-code bugs (commits 77c0cf9b, 40b00774) signal structural problem needing structural fix

**RECOMMENDATION:** Option A — `daemonConfigFromFlags()` function

**Trade-off accepted:** Paths that don't need all fields (preview, dry-run) will get unnecessary defaults, but these are harmless (zero cost when features are unused, e.g. CleanupEnabled is irrelevant for preview).

**When this would change:** If daemon paths diverge intentionally (e.g., preview needs fundamentally different Config), extract a `MinimalConfig()` variant.

---

### Fork 2: Persistent Tracker State

**Options:**
- **A: Count `daemon:ready-review` open issues without checkpoint entries** — On startup, query beads for open issues with `daemon:ready-review` label, cross-reference with checkpoint file, seed counter
- **B: Persist VerificationTracker state to a file** — Write `~/.orch/daemon-verification-state.json` on each state change, read on startup
- **C: Use events log** — Parse `~/.orch/events.jsonl` for `daemon.complete` events since last `daemon.verification` event

**Substrate says:**
- Decision "verifiability-first-hard-constraint": The checkpoint file IS the source of truth for human verification. Using it for seeding is coherent.
- Principle "Evidence hierarchy": `daemon:ready-review` label IS the source of truth for unverified work. Beads database is authoritative.
- Constraint: Checkpoint file exists and works (`~/.orch/verification-checkpoints.jsonl`, 2 entries currently)

**RECOMMENDATION:** Option A — Count `daemon:ready-review` open issues without checkpoint entries

**Rationale:**
- No new files to manage (reuses existing beads and checkpoint infrastructure)
- Beads is already the source of truth for issue state
- `daemon:ready-review` label is the canonical marker for "daemon-completed, not human-verified"
- Open issues with this label are, by definition, unverified completions (verified ones get closed by `orch complete`)
- Simple: `count(issues with daemon:ready-review AND status=open/in_progress) - count(checkpoint entries for those IDs)`

**Trade-off accepted:** Small startup latency (~100ms for beads query + checkpoint file read). Acceptable for daemon startup which happens rarely.

**When this would change:** If checkpoint file grows very large (>10K entries), an index or state file (Option B) would be faster. Current file has 2 entries; even at 1000 this is negligible.

---

### Fork 3: Where to seed the tracker

**Options:**
- **A: In NewWithConfig()** — Constructor auto-seeds by querying beads/checkpoints
- **B: In cmd layer, after construction** — Separate `SeedFromBacklog()` call in cmd/orch/daemon.go
- **C: New constructor `NewWithBacklogSeed()`** — Encapsulates seeding

**Substrate says:**
- Principle "Evolve by distinction": pkg/daemon should not depend on beads CLI. Keep I/O in cmd layer.
- Architecture: `pkg/daemon/` currently has no direct beads dependency (shells out via functions like `ListReadyIssues()`). Adding checkpoint reading would create a new dependency.

**RECOMMENDATION:** Option B — Seed in cmd layer

**Rationale:**
- Keeps `pkg/daemon` free of I/O concerns
- `SeedFromBacklog(count int)` on VerificationTracker is a pure method (just sets counter)
- Seeding logic lives in `cmd/orch/daemon.go` where beads and checkpoint access already exist
- Easy to test: mock the count, verify tracker state

**Trade-off accepted:** Seeding logic lives in cmd layer, not encapsulated in daemon package. But this matches existing patterns (e.g., HotspotChecker is set after construction).

---

## Synthesis: Implementation Design

### Part 1: Unified Config Construction

**Location:** `cmd/orch/daemon.go`

Add a single function that all daemon paths call:

```go
// daemonConfigFromFlags builds a Config starting from DefaultConfig(),
// overriding with CLI flag values. All daemon paths (run, once, dry-run,
// preview) MUST use this function instead of constructing Config directly.
func daemonConfigFromFlags() daemon.Config {
    config := daemon.DefaultConfig()

    // Override with CLI flags
    config.PollInterval = time.Duration(daemonPollInterval) * time.Second
    config.MaxAgents = daemonMaxAgents
    config.Label = daemonLabel
    config.SpawnDelay = time.Duration(daemonDelay) * time.Second
    config.DryRun = daemonDryRun
    config.Verbose = daemonVerbose
    config.ReflectEnabled = daemonReflectInterval > 0
    config.ReflectInterval = time.Duration(daemonReflectInterval) * time.Minute
    config.ReflectCreateIssues = daemonReflectIssues
    config.CleanupEnabled = daemonCleanupEnabled && daemonCleanupInterval > 0
    config.CleanupInterval = time.Duration(daemonCleanupInterval) * time.Minute
    config.CleanupAgeDays = daemonCleanupAge
    config.CleanupPreserveOrchestrator = daemonCleanupPreserveOrch
    config.CleanupServerURL = serverURL

    return config
}
```

Then update all call sites:

```go
// runDaemonLoop:
config := daemonConfigFromFlags()
d := daemon.NewWithConfig(config)

// runDaemonDryRun:
config := daemonConfigFromFlags()
d := daemon.NewWithConfig(config)

// runDaemonOnce:
config := daemonConfigFromFlags()
d := daemon.NewWithConfig(config)

// runDaemonPreview:
config := daemonConfigFromFlags()
d := daemon.NewWithConfig(config)
```

**Immediate bug fixes this delivers:**
1. RecoveryEnabled=true in production daemon (was false)
2. MaxSpawnsPerHour=20 in production daemon (was 0, no rate limiting)
3. VerificationPauseThreshold=3 in preview mode (was 0)

### Part 2: Persistent VerificationTracker

**Location:** `pkg/daemon/verification_tracker.go` (add method)

```go
// SeedFromBacklog sets the completion counter to reflect existing
// unverified backlog. Call after construction, before entering the
// main loop, to make the tracker aware of work completed before
// this daemon session started.
func (vt *VerificationTracker) SeedFromBacklog(unverifiedCount int) {
    vt.mu.Lock()
    defer vt.mu.Unlock()

    vt.completionsSinceVerification = unverifiedCount
    if vt.threshold > 0 && unverifiedCount >= vt.threshold {
        vt.isPaused = true
    }
}
```

**Location:** `pkg/daemon/verification_tracker.go` (add helper)

```go
// CountUnverifiedCompletions counts open/in_progress issues with
// the daemon:ready-review label that don't have verification checkpoint entries.
// This represents the backlog of daemon-completed work awaiting human review.
func CountUnverifiedCompletions() (int, error) {
    // Get all issues with daemon:ready-review label
    readyForReview, err := ListIssuesWithLabel("daemon:ready-review")
    if err != nil {
        return 0, fmt.Errorf("failed to list ready-for-review issues: %w", err)
    }

    if len(readyForReview) == 0 {
        return 0, nil
    }

    // Read checkpoint file
    checkpoints, err := checkpoint.ReadCheckpoints()
    if err != nil {
        // Checkpoint file missing/corrupt - count all as unverified
        return len(readyForReview), nil
    }

    // Build set of checkpoint beads IDs
    checkpointIDs := make(map[string]bool)
    for _, cp := range checkpoints {
        if cp.Gate1Complete {
            checkpointIDs[cp.BeadsID] = true
        }
    }

    // Count issues without checkpoints
    unverified := 0
    for _, issue := range readyForReview {
        if !checkpointIDs[issue.ID] {
            unverified++
        }
    }

    return unverified, nil
}
```

Note: `ListIssuesWithLabel` needs to be added to `pkg/daemon/issue_adapter.go` — a thin wrapper around `bd list -l <label> --status=open --status=in_progress`.

**Location:** `cmd/orch/daemon.go` (add seeding call)

```go
// seedVerificationTracker seeds the tracker with the backlog count.
// Called after daemon construction, before entering the main loop.
func seedVerificationTracker(d *daemon.Daemon) {
    if d.VerificationTracker == nil {
        return
    }

    count, err := daemon.CountUnverifiedCompletions()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not seed verification tracker: %v\n", err)
        return
    }

    if count > 0 {
        d.VerificationTracker.SeedFromBacklog(count)
        fmt.Printf("  Verification backlog: %d unverified completions from previous sessions\n", count)

        if d.VerificationTracker.IsPaused() {
            fmt.Printf("  ⚠️  Verification pause: backlog exceeds threshold (%d/%d)\n",
                count, d.VerificationTracker.Status().Threshold)
            fmt.Println("  Run 'orch daemon resume' after reviewing completed work")
        }
    }
}
```

Call `seedVerificationTracker(d)` in `runDaemonLoop()`, `runDaemonOnce()`, and `runDaemonDryRun()` after `daemon.NewWithConfig(config)`.

### Part 3: Issue Adapter for Label Query

**Location:** `pkg/daemon/issue_adapter.go` (add function)

```go
// ListIssuesWithLabel lists open/in_progress issues with a specific label.
func ListIssuesWithLabel(label string) ([]Issue, error) {
    // Shell out to bd: bd list -l <label> --status=open --status=in_progress --format json
    // Parse JSON output into []Issue
    // Filter for open/in_progress status
}
```

This follows the existing pattern where `ListReadyIssues()` shells out to `bd list`.

---

## File Targets

| File | Action | Description |
|------|--------|-------------|
| `cmd/orch/daemon.go` | Modify | Add `daemonConfigFromFlags()`, update all 4 construction sites, add `seedVerificationTracker()` |
| `pkg/daemon/verification_tracker.go` | Modify | Add `SeedFromBacklog()` method |
| `pkg/daemon/verification_tracker_test.go` | Modify | Add test for `SeedFromBacklog()` |
| `pkg/daemon/issue_adapter.go` | Modify | Add `ListIssuesWithLabel()` function |
| `pkg/daemon/daemon.go` (optional) | No change | `CountUnverifiedCompletions()` could go here or in a new file |

---

## Acceptance Criteria

1. `go build ./cmd/orch/` passes
2. `go vet ./cmd/orch/` passes
3. All 4 daemon paths use `daemonConfigFromFlags()` — no direct `daemon.Config{}` construction in daemon commands
4. `daemon.DefaultConfig().RecoveryEnabled == true` AND production daemon loop gets `RecoveryEnabled=true`
5. `daemon.DefaultConfig().MaxSpawnsPerHour == 20` AND production daemon loop gets rate limiting
6. Preview mode has VerificationPauseThreshold > 0
7. `SeedFromBacklog(5)` with threshold=3 results in `IsPaused()=true`
8. `SeedFromBacklog(2)` with threshold=3 results in `IsPaused()=false` and counter=2
9. Existing tests pass (verification_tracker_test.go, daemon_test.go)

---

## Out of Scope

- Changing the checkpoint file format
- Adding a new persistent state file for the tracker
- Changing the daemon's polling architecture
- Cross-project daemon (single daemon managing multiple projects)

---

## Recommendations

⭐ **RECOMMENDED:** Unified `daemonConfigFromFlags()` + `SeedFromBacklog()` backlog seeding

**Why:**
1. Addresses root cause (scattered construction) rather than symptoms (individual field fixes)
2. Compiler-enforced: new fields added to `DefaultConfig()` automatically propagate to all paths
3. Backlog seeding uses existing infrastructure (beads labels + checkpoint file) — no new files
4. Three real bugs fixed immediately (Recovery, rate limiting, preview threshold)

**Expected outcome:** Future Config field additions require ONE change (in `DefaultConfig()`) instead of 4. Daemon restarts correctly reflect unverified backlog.

**Alternative: Per-path fix with tests**
- Pros: Smaller change, no refactoring
- Cons: Same failure mode recurs with next field addition. Doesn't fix Recovery or rate limiting bugs.
- When to choose: Never — this is strictly worse.

## Decision Gate Guidance

**Add `blocks:` frontmatter when promoting to decision:**
This decision resolves recurring issues (2+ rounds of dead code from Config divergence).

**Suggested blocks keywords:**
- daemon config
- verification tracker
- daemon spawn
- daemon pause
