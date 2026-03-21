# Decision: Unified Daemon Config Construction & Persistent VerificationTracker

**Date:** 2026-02-15
**Status:** Proposed
**Deciders:** Agent (architect)
**Investigation:** `.kb/investigations/2026-02-15-design-daemon-unified-config-persistent-tracker.md`
**blocks:** daemon config, verification tracker, daemon spawn, daemon pause

## Context

Daemon Config construction is scattered across 4 independent sites in `cmd/orch/daemon.go`. Each path (`runDaemonLoop`, `runDaemonDryRun`, `runDaemonOnce`, `runDaemonPreview`) constructs `daemon.Config` from scratch, leading to:

1. **Repeated dead-code incidents**: New fields (VerificationPauseThreshold) get set in one path but not others. Already caused two fix rounds (commits 77c0cf9b, 40b00774).
2. **Silent feature disablement**: RecoveryEnabled is `true` in `DefaultConfig()` but never set in `runDaemonLoop()` — recovery is silently disabled in production. MaxSpawnsPerHour=20 in defaults but 0 in production loop — no rate limiting.
3. **Persistence gap**: VerificationTracker counter resets to 0 on daemon restart. The 63 unverified completions in the backlog don't trigger pause.

## Decision

### Part 1: Single `daemonConfigFromFlags()` function

All daemon paths MUST use a shared `daemonConfigFromFlags()` function that starts from `DefaultConfig()` and overrides only the fields with CLI flag values. No path may construct `daemon.Config{}` directly.

**Rationale:** New fields added to `DefaultConfig()` automatically propagate. The compiler can't catch zero-value omissions in Go structs, so the structural fix is to never construct from scratch.

### Part 2: Backlog seeding via `SeedFromBacklog()`

On startup, count open/in_progress issues with `daemon:ready-review` label that lack verification checkpoint entries. Seed the VerificationTracker counter with this count.

**Rationale:** Uses existing infrastructure (beads labels + checkpoint file). No new persistence files. Beads is the source of truth for issue state; checkpoint file is the source of truth for human verification.

## Consequences

### Positive
- New Config fields need ONE change (in `DefaultConfig()`)
- Recovery and rate limiting enabled in production immediately
- Daemon correctly reflects unverified backlog on restart
- Preview mode gets verification threshold

### Negative
- Paths that don't need all Config fields (preview) still get them — harmless overhead
- Startup has ~100ms additional latency for beads query (negligible)
- `SeedFromBacklog` depends on `daemon:ready-review` label being consistently applied

### Risks
- If `daemon:ready-review` label is manually removed from unverified issues, counter will be wrong (mitigation: label removal requires `orch complete` which resets counter anyway)
- If checkpoint file is corrupted, all daemon:ready-review issues count as unverified (mitigation: safe default — triggers pause sooner, which is the conservative behavior)

## Alternatives Rejected

1. **Builder pattern**: Unnecessary abstraction for 4 call sites
2. **Persistent state file**: Adds new file when existing infrastructure (beads + checkpoints) suffices
3. **Per-path fix with tests**: Doesn't address root cause, same failure recurs with next field
