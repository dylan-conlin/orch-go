# Probe: Spawn Prerequisites Hard Gates vs Soft Warnings [DAO-02]

**Date:** 2026-03-20
**Model:** Daemon Autonomous Operation
**Claim:** DAO-02 — "Spawn prerequisites must be hard gates (fail-fast), not soft warnings — warn-and-continue on spawn prerequisites causes duplicate spawns"
**Falsification Condition:** Warn-and-continue spawn prerequisites produce fewer duplicate spawns than fail-fast over 100+ daemon cycles

---

## Method

1. Traced the history of warn-and-continue → fail-fast conversion through git commits and prior probes
2. Analyzed the current spawn gate pipeline architecture (`pkg/daemon/spawn_gate.go`, `pkg/daemon/spawn_execution.go`)
3. Counted daemon spawn events in `events.jsonl` for post-fix period (Mar 8 – Mar 20)
4. Checked for actual duplicate spawns (same beads ID spawned more than once)
5. Cross-referenced the accretion-gates-advisory decision for nuance on where advisory IS appropriate

---

## Evidence

### The Incident (Feb 14, 2026)

**Commit `dad6426fb`:** `UpdateBeadsStatus` was logging a warning and continuing when it failed. When the SpawnedIssueTracker TTL (6h) expired or daemon restarted, the issue appeared "ready" again because beads still showed `open`. Result: 10 duplicate spawns for the same issue in 20 minutes (orch-go-w50).

**Root cause:** The in-memory spawn tracker (L1) is ephemeral — it has a 6h TTL and is lost on restart. Without persistent state marking the issue as `in_progress` in beads (L6), there was nothing preventing re-spawn.

### The Audit (Feb 15, 2026)

Probe `2026-02-15-daemon-warn-continue-anti-pattern-audit.md` found 5 additional warn-and-continue patterns in spawn prerequisites:

1. **Dependency check** — could spawn blocked work
2. **Epic expansion** — silently drops spawnable work
3. **Extraction gate** — spawns on critical hotspots without extraction
4. **Rollback after spawn failure** — leaves issues in inconsistent state
5. **Completion processing errors** — completed agents never reviewed

All 5 were converted to fail-fast between Feb 15-17 (commits `bb055f499`, `066a495b3`, `59981d9b8`, `80f8c56e9`, `17390b263`).

### Current Architecture (Post-Fix)

The spawn pipeline now has a clear two-tier architecture:

| Tier | Components | Behavior on Error | Rationale |
|------|-----------|-------------------|-----------|
| **Hard gates** | L6 (beads status update), dependency check, rollback | **Fail-fast** — abort spawn | These affect spawn correctness |
| **Heuristic gates** | L1-L5 (spawn tracker, session dedup, title dedup, fresh status) | **Fail-open** — allow spawn if backing service unavailable | Better to risk a duplicate than block all work |
| **Advisory** | SpawnCountAdvisory, hotspot warning, governance warning | **Warn only** — never block | These are monitoring/signaling, not correctness |

Key design: `SpawnGate` interface has explicit `FailMode()` method (lines 32-42 of `spawn_gate.go`), making the distinction between fail-fast and fail-open a first-class architectural concern, not an ad-hoc choice per call site.

### Post-Fix Duplicate Rate (574 daemon spawns, Mar 8-20)

- **574 daemon.spawn events** across 571 unique issues
- **3 issues spawned twice** — all legitimate respawns:
  - `orch-go-6o6fz`: spawned Mar 10, abandoned, respawned Mar 11 (8h gap)
  - `orch-go-9adro`: spawned Mar 14, respawned Mar 15 (16h gap)
  - `orch-go-0ljv0`: spawned Mar 16, respawned Mar 19 (3 day gap)
- **0 actual dedup failures** — zero cases of same issue spawned concurrently or within minutes
- **7 spawn_tracker catches** — L1 correctly blocked 7 would-be duplicates

### Nuance: Advisory Is Appropriate for Non-Spawn-Correctness Gates

The accretion-gates-advisory decision (`2026-03-17`) provides important context: hotspot/accretion gates were converted FROM blocking TO advisory because measurement showed 100% bypass rate and zero behavioral effect from blocking. These gates work through event emission (triggering daemon extraction), not through blocking.

This does NOT contradict DAO-02. The distinction is:
- **Spawn correctness** (will this cause a duplicate?) → hard gate
- **Code quality signal** (is this a hotspot?) → advisory is fine

---

## Verdict

**CONFIRMS claim DAO-02.**

The falsification condition requires warn-and-continue to produce *fewer* duplicates than fail-fast. The evidence shows the opposite:

1. **Before fail-fast** (pre-Feb 14): 10 duplicate spawns in 20 minutes from a single warn-and-continue pattern
2. **After fail-fast** (574 spawns, Mar 8-20): 0 dedup failures, 7 would-be duplicates caught by pipeline

The claim holds because spawn prerequisites affect **structural correctness** — whether the persistent state (beads) reflects that an issue is being worked on. Without fail-fast on structural gates, the ephemeral dedup layers (L1-L5) are the only defense, and they all have TTL/restart/fail-open gaps.

**Important refinement:** The claim should be scoped to spawn *correctness* prerequisites, not all spawn-adjacent checks. Advisory gates (hotspot, governance, accretion) are correctly non-blocking because they don't affect whether a duplicate will occur.

### Confidence: High

- Direct causal evidence (Feb 14 incident → fix → zero duplicates in 574 spawns)
- Architectural analysis confirms why (ephemeral layers alone are insufficient)
- 574 daemon spawns well exceeds the 100+ cycle threshold

---

## Impact on Model

Model section "Spawn Prerequisites: Fail-Fast Gates" (lines 77-91) is accurate and current. No changes needed — the existing text correctly distinguishes between fail-fast prerequisites and acceptable warn-and-continue for monitoring/observability.

Minor suggestion: the model could note the `FailMode()` interface in `spawn_gate.go` as the architectural mechanism that enforces this constraint, making it structural rather than just a coding convention.
