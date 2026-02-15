# Probe: Daemon Config Construction Divergence Audit

**Date:** 2026-02-15
**Status:** Complete
**Model:** daemon-autonomous-operation
**Question:** Are all daemon Config construction sites consistent? Does VerificationTracker state survive restarts?

## What I Tested

Audited all Config construction sites in `cmd/orch/daemon.go` and `pkg/daemon/daemon.go` by reading the source code and comparing field-by-field.

### Config Construction Sites Found

**4 construction sites in cmd/orch/daemon.go:**

1. **`runDaemonLoop()` (line 188-205)** — Primary daemon loop
2. **`runDaemonDryRun()` (line 688-694)** — Dry-run mode
3. **`runDaemonOnce()` (line 746-752)** — Single-issue mode
4. **`runDaemonPreview()` (line 800-804)** — Preview mode

**3 constructor sites in pkg/daemon/daemon.go:**
5. **`New()` (line 225-227)** — Calls `NewWithConfig(DefaultConfig())`
6. **`NewWithConfig(config)` (line 230-252)** — Main constructor
7. **`NewWithPool(config, pool)` (line 256-274)** — Testing constructor

## What I Observed

### Field-by-Field Divergence Matrix

| Field | DefaultConfig | runDaemonLoop | runDaemonDryRun | runDaemonOnce | runDaemonPreview |
|-------|:------------:|:-------------:|:---------------:|:-------------:|:----------------:|
| PollInterval | 15s | ✅ flag | ❌ 0 | ❌ 0 | ❌ 0 |
| MaxAgents | 3 | ✅ flag | ❌ 0 | ❌ 0 | ❌ 0 |
| **MaxSpawnsPerHour** | **20** | **❌ 0** | ❌ 0 | ❌ 0 | ❌ 0 |
| Label | triage:ready | ✅ flag | ✅ flag | ✅ flag | ✅ flag |
| SpawnDelay | 3s | ✅ flag | ❌ 0 | ❌ 0 | ❌ 0 |
| DryRun | false | ✅ flag | ❌ false | ❌ false | ❌ false |
| Verbose | false | ✅ flag | ❌ false | ❌ false | ❌ false |
| ReflectEnabled | true | ✅ flag | ❌ false | ❌ false | ❌ false |
| ReflectInterval | 1h | ✅ flag | ❌ 0 | ❌ 0 | ❌ 0 |
| ReflectCreateIssues | true | ✅ flag | ❌ false | ❌ false | ❌ false |
| CleanupEnabled | true | ✅ flag | ❌ false | ❌ false | ❌ false |
| CleanupInterval | 6h | ✅ flag | ❌ 0 | ❌ 0 | ❌ 0 |
| CleanupAgeDays | 7 | ✅ flag | ❌ 0 | ❌ 0 | ❌ 0 |
| CleanupPreserveOrch | true | ✅ flag | ❌ false | ❌ false | ❌ false |
| CleanupServerURL | localhost:4096 | ✅ global | ❌ "" | ❌ "" | ❌ "" |
| **RecoveryEnabled** | **true** | **❌ false** | ❌ false | ❌ false | ❌ false |
| **RecoveryInterval** | **5m** | **❌ 0** | ❌ 0 | ❌ 0 | ❌ 0 |
| **RecoveryIdleThreshold** | **10m** | **❌ 0** | ❌ 0 | ❌ 0 | ❌ 0 |
| **RecoveryRateLimit** | **1h** | **❌ 0** | ❌ 0 | ❌ 0 | ❌ 0 |
| VerificationPauseThreshold | 3 | ✅ defaults | ✅ defaults | ✅ defaults | **❌ 0** |

### Critical Findings

**Bug 1: Recovery NEVER runs in production daemon.**
`runDaemonLoop()` constructs Config from scratch and never sets Recovery* fields. They default to `false`/0. Despite `daemon.go:260-266` checking `config.RecoveryEnabled` and the loop at line 393 calling `d.RunPeriodicRecovery()`, the check returns early because `RecoveryEnabled=false`. DefaultConfig() sets `RecoveryEnabled=true` but runDaemonLoop() doesn't use DefaultConfig as a base.

**Bug 2: MaxSpawnsPerHour never set in production daemon.**
`DefaultConfig()` sets `MaxSpawnsPerHour=20` but `runDaemonLoop()` doesn't set it. Rate limiter is initialized as nil in `NewWithConfig()` because `config.MaxSpawnsPerHour == 0`.

**Bug 3: runDaemonPreview() omits VerificationPauseThreshold.**
All other paths use `defaults.VerificationPauseThreshold` but preview creates `Config{Label: daemonLabel}` only. The verification tracker is created with threshold=0 (disabled).

**Bug 4: VerificationTracker always starts at 0.**
`NewVerificationTracker(threshold)` initializes `completionsSinceVerification=0`. On daemon restart, the 63 unverified completions in the backlog don't contribute. The daemon immediately starts spawning more work without pausing.

### Persistence Gap

`~/.orch/verification-checkpoints.jsonl` has 2 entries (only from `orch complete --explain`).
Daemon `daemon:ready-review` labels are NOT cross-referenced with checkpoint entries.
On restart, counter resets to 0 regardless of backlog.

## Model Impact

**Confirms:**
- Model claim "Capacity Starvation" pattern - Config divergence is a specific instance: fields like MaxSpawnsPerHour that should protect against runaway spawning are silently disabled
- Model claim about "Spawn failures don't release slots" - rate limiter being nil means no spawn rate protection at all

**Extends:**
- The model describes "duplicate spawns" and "skill inference mismatch" as failure modes but **doesn't identify scattered Config construction as a root cause**. This is a new systemic failure mode: **Config Divergence Drift** — every new Config field requires updating 4 independent construction sites, and the compiler won't catch omissions because Go zero-values are valid.
- Recovery being silently disabled is a new finding not covered by any model claim

**Contradicts:**
- Model implies daemon has rate limiting. In production, it does NOT due to `MaxSpawnsPerHour=0` in the loop path. Only the `New()` convenience constructor (which nobody calls in production) gets the default.
