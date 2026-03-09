# Probe: 30-Day Accretion Trajectory — Do Harness Gates Bend the Line Count Curve?

**Model:** harness-engineering
**Date:** 2026-03-08
**Status:** Complete

---

## Question

The harness engineering model claims gates (pre-commit growth gate, spawn hotspot gate, completion accretion gate) prevent accretion. Does the empirical line count trajectory of key files show an inflection point after gate deployment? Specifically:

1. Does daemon.go's line count trend reverse or stabilize after gate deployment?
2. Does the count of bloated files (>800 lines) decrease over time?
3. Does the fix:feat ratio shift sustainably after gates ship?
4. Does spawn_cmd.go demonstrate that attractors (pkg/spawn/backends/) permanently prevent re-accretion?

Testing model invariants #2 ("every convention without a gate will eventually be violated"), #4 ("extraction without routing is a pump"), and the "Attractors Without Gates" failure mode.

---

## What I Tested

Weekly line count snapshots extracted from git history for key cmd/orch/ files over a 12-week window (Dec 22, 2025 → Mar 8, 2026). Measured:

1. **Individual file trajectories** via `git show <commit>:<file> | wc -l` at weekly commit boundaries
2. **Aggregate cmd/orch/ metrics**: total lines, file count, files >800 lines
3. **Fix:feat ratio** via `git log --format="%s" | grep "^feat\|^fix"` per week
4. **Gate code analysis**: read `pkg/verify/accretion.go` (completion gate), `pkg/verify/accretion_precommit.go` (pre-commit gate), and `scripts/pre-commit-exec-start-cleanup.sh` (hook wiring)

### Gate Deployment Timeline (verified from git log)

| Date | Gate | Type |
|------|------|------|
| Feb 13 | `pkg/spawn/backends/` created (attractor for spawn_cmd.go) | Structural |
| Feb 15 | Completion accretion gate (`pkg/verify/accretion.go`) | Hard (with exemption) |
| Feb 19 | Spawn hotspot gate blocking for CRITICAL files | Hard |
| Feb 24 | Accretion gate pre-existing bloat skip added | Policy (weakening) |
| Mar 1 | Pre-commit hook: compilation + architecture lint | Hard (not accretion) |
| Mar 8 | `CheckStagedAccretion` code written | Code exists, NOT WIRED |

### Data: Individual File Line Counts (Weekly)

| Date | daemon.go | main.go | spawn_cmd.go | status_cmd.go | complete_cmd.go |
|------|-----------|---------|--------------|---------------|-----------------|
| Dec 22 | 507 | 3,217 | — | — | — |
| Dec 29 | 546 | 5,551 | — | — | — |
| Jan 5 | 588 | 195 | 1,619 | 1,059 | 1,021 |
| Jan 12 | 723 | 197 | 2,260 | 1,415 | 1,132 |
| Jan 19 | 768 | 197 | 2,432 | 1,656 | 1,640 |
| Feb 16 | 983 | 199 | 677 | 1,639 | 1,992 |
| Feb 23 | 1,038 | 198 | 833 | 1,366 | 2,146 |
| Mar 2 | 1,214 | 349 | 935 | 1,415 | 293 |
| Mar 8 | 1,559 | 354 | 1,160 | 1,361 | 340 |

### Data: Aggregate cmd/orch/ Metrics

| Date | Files | Total Lines | Files >800 | % Bloated |
|------|-------|-------------|------------|-----------|
| Dec 22 | 10 | 7,315 | 1 | 10% |
| Dec 29 | 30 | 22,357 | 6 | 20% |
| Jan 5 | 46 | 20,931 | 5 | 11% |
| Jan 12 | 60 | 29,840 | 10 | 17% |
| Jan 19 | 62 | 34,607 | 13 | 21% |
| Feb 16 | 70 | 34,977 | 13 | 19% |
| Feb 23 | 82 | 36,450 | 14 | 17% |
| Mar 2 | 111 | 42,703 | 12 | 11% |
| Mar 8 | 125 | 47,605 | 12 | 10% |

### Data: Fix:Feat Ratio by Week

| Week Starting | feat | fix | refactor | fix:feat |
|---------------|------|-----|----------|----------|
| Dec 15 | 85 | 27 | 4 | 0.31 |
| Dec 22 | 168 | 105 | 5 | 0.62 |
| Dec 29 | 70 | 61 | 21 | 0.87 |
| Jan 5 | 118 | 60 | 1 | 0.50 |
| Jan 12 | 79 | 39 | 0 | 0.49 |
| Feb 9 | 54 | 25 | 10 | 0.46 |
| Feb 16 | 68 | 50 | 2 | 0.73 |
| Feb 23 | 87 | 106 | 17 | **1.21** |
| Mar 2 | 121 | 44 | 3 | 0.36 |

---

## What I Observed

### 1. daemon.go: Line count is ACCELERATING, not decelerating

daemon.go grew from 768 (Jan 19, the last pre-gate snapshot) to 1,559 (Mar 8) — **+791 lines in 7 weeks**. The growth velocity is increasing:

- Jan 19 → Feb 16: +215 lines (4 weeks, ~54/week)
- Feb 16 → Feb 23: +55 lines (1 week)
- Feb 23 → Mar 2: +176 lines (1 week)
- Mar 2 → Mar 8: +345 lines (1 week)

**No inflection point is visible.** The curve bends upward, not downward, after gate deployment.

Between Feb and Mar, daemon.go received 30 commits including: agreement checks, orphan recovery, phase timeout detection, stuck detection, self-check invariants, focus-aware priority boost, beads health checks. Each individually correct, collectively accelerating accretion.

### 2. The completion accretion gate EXEMPTS pre-existing bloat

`pkg/verify/accretion.go` lines 128-136: if a file was ALREADY over 1,500 lines before the agent's changes, the gate downgrades from ERROR (blocking) to WARNING (non-blocking). This means:

- daemon.go at 1,559 lines: the gate will now WARN on additions but not BLOCK them
- Every agent adding 50+ lines gets a warning, not a rejection
- The gate can only block files that CROSS the 1,500 threshold during a single commit

**The gate is structurally unable to enforce on the files that need enforcement most.**

### 3. The pre-commit accretion gate is NOT WIRED

`scripts/pre-commit-exec-start-cleanup.sh` contains only:
1. `go build ./cmd/orch/` (compilation gate)
2. `go test -run TestArchitectureLint` (architecture lint)

`CheckStagedAccretion` was committed today (Mar 8) in `pkg/verify/accretion_precommit.go` but the pre-commit hook was NOT updated to call it. The VERIFICATION_SPEC.yaml for that commit says "Shell integration provided in VERIFICATION_SPEC.yaml (governance-protected)" — meaning the agent documented how to wire it but didn't modify the governance-protected hook.

**Result: the pre-commit accretion gate is dead code from an enforcement perspective.**

### 4. spawn_cmd.go: Attractor works for extraction, fails for prevention

spawn_cmd.go trajectory demonstrates the attractor pattern:
- Pre-attractor peak: 2,432 lines (Jan 19)
- Post-attractor extraction: 677 lines (Feb 16) — **-1,755 lines** (model claimed -840, actual was nearly double)
- Re-accretion: 1,160 lines (Mar 8) — **+483 lines in 3 weeks**

Re-accretion velocity: ~160 lines/week. At this rate, spawn_cmd.go will re-cross 1,500 lines by late March.

The attractor (`pkg/spawn/backends/`) pulled existing code out but doesn't prevent new code from landing in spawn_cmd.go. The Cobra command definition lives in spawn_cmd.go, creating the same "feature gravity" the model describes for daemon.go.

### 5. Bloated file percentage improved, but through proliferation not shrinkage

Files >800 lines: peaked at 14 (Feb 23), now at 12 (Mar 8). The percentage dropped from 21% to 10%.

**But this is because the total file count grew from 62 to 125 (2x), not because bloated files shrank.** The system produces more small files (extraction creates new files) while existing bloated files remain large. Current files >800 lines:

| File | Lines |
|------|-------|
| daemon.go | 1,559 |
| status_cmd.go | 1,361 |
| review.go | 1,353 |
| stats_cmd.go | 1,351 |
| clean_cmd.go | 1,270 |
| spawn_cmd.go | 1,160 |
| serve_beads.go | 1,124 |
| serve_system.go | 1,084 |
| hotspot.go | 1,056 |
| session.go | 1,055 |
| kb.go | 919 |
| handoff.go | 898 |

6 of 12 are over 1,000 lines. The problem is spreading, not concentrating.

### 6. Fix:feat ratio showed transient spike, not regime change

The fix:feat ratio spiked to 1.21 during the week of Feb 23 — coinciding with the intensive gate deployment period. This means more bugs were fixed than features added, which is expected during an infrastructure-heavy week. But it immediately reverted to 0.36 the following week. There is no sustained shift in the ratio.

### 7. Total cmd/orch/ velocity is accelerating

Total lines in cmd/orch/ (non-test): 47,605, up from 34,977 three weeks ago (+12,628 in 3 weeks, ~4,200/week). The aggregate growth rate hasn't slowed since gate deployment. Gates haven't bent the total line count curve.

---

## Model Impact

- [x] **Confirms** invariant #2: "Every convention without a gate will eventually be violated." daemon.go grew to 1,559 while the >1,500-line convention existed in CLAUDE.md. The convention was soft harness; no blocking gate existed for pre-existing bloated files.

- [x] **Confirms** invariant #4: "Extraction without routing is a pump." spawn_cmd.go shrank -1,755 lines after attractor creation then regrew +483 in 3 weeks. daemon.go has pkg/daemon/ (896 lines) as attractor but the cmd/ file continues growing because no gate prevents the old path.

- [x] **Confirms** "Why This Fails" #3: "Attractors without gates." pkg/daemon/ exists (896 lines of daemon code extracted) but daemon.go (cmd/orch/) still grew to 1,559 because no gate prevents the cmd/ path.

- [x] **Extends** model with: **Gate exemption creates a structural enforcement ceiling.** The completion accretion gate's pre-existing bloat skip (lines 128-136 of accretion.go) means files that cross the CRITICAL threshold can never be blocked again — they only receive warnings. This creates a ratchet: once bloated, always exempted. The model's "Why This Fails" section should add a 6th failure mode: "Gate Exemptions as Permanent Bypasses."

- [x] **Extends** model with: **Dead code enforcement gap.** The pre-commit accretion gate (`CheckStagedAccretion`) exists in `pkg/verify/accretion_precommit.go` but is not wired into the pre-commit hook (`scripts/pre-commit-exec-start-cleanup.sh`). Governance protection of the hook prevented the implementing agent from completing the integration. This is an instance of the "mutable control plane" problem in reverse — immutability of the control plane prevented a needed update.

- [x] **Extends** model with: **Bloated file proliferation.** The model focuses on individual file accretion (daemon.go). The systemic picture is worse: cmd/orch/ grew from 10 to 125 files and 7,315 to 47,605 lines in 12 weeks. 12 files exceed 800 lines. The problem is no longer single-file accretion — it's system-wide growth without structural bounds. Total cmd/orch/ velocity shows no deceleration after any gate deployment.

- [x] **Contradicts** model's spawn_cmd.go shrinkage claim: Model states "-840 lines after pkg/spawn/backends/ was created." Actual shrinkage was **-1,755 lines** (2,432→677). The attractor was more effective than the model claims for initial extraction. However, re-accretion (+483 in 3 weeks) confirms the model's broader point about pump dynamics.

---

## Notes

### Baseline for 30-Day Forward Measurement

This probe establishes the baseline. The true 30-day test starts from Mar 8, 2026:

**Baseline metrics (Mar 8):**
- daemon.go: 1,559 lines
- cmd/orch/ total: 47,605 lines across 125 files
- Files >800 lines: 12 (10% of total files)
- Weekly accretion velocity: ~4,200 lines/week
- Fix:feat ratio: 0.36 (latest week)

**What would constitute "bending the curve":**
- daemon.go stabilizes or decreases (target: <1,500 by Apr 8)
- Files >800 count decreases to <10
- Weekly velocity drops below 2,000 lines/week
- Fix:feat ratio sustains >0.8 for 3+ consecutive weeks

### Critical Gap: No Blocking Gate Exists for Pre-Existing Bloated Files

The completion gate exempts pre-existing bloat. The pre-commit gate isn't wired. The spawn gate blocks skills (feature-impl, debugging) from targeting CRITICAL files, but doesn't prevent the daemon subsystem's own agents from modifying daemon.go.

For the gates to bend the curve, they need:
1. Wire `CheckStagedAccretion` into the pre-commit hook
2. Remove or tighten the pre-existing bloat exemption
3. Create structural attractors WITH gates for the 12 currently-bloated files

### The Real Finding

The harness engineering model is **correct in theory** but the gates as deployed are **too late, too narrow, and self-exempting**. The model predicts exactly what's happening (accretion despite gates) but claims gates are "shipped" when they're partially dead code. The honest assessment: gates haven't been given a fair test yet because the blocking gates for pre-existing bloat literally don't exist.
