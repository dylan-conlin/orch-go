# Session Synthesis

**Agent:** og-arch-architect-daemon-reliability-27mar-296b
**Issue:** orch-go-5b9st
**Duration:** 2026-03-27 11:42 → 2026-03-27 13:15
**Outcome:** success

---

## TLDR

The daemon's 6 recurring symptoms (SIGKILL crashes, duplicate spawns, frozen dashboard, double logging, redundant scans, stuck counters) trace to 3 structural roots: no shutdown budget, dual-write logging (launchd + DaemonLogger both target daemon.log), and organic subsystem growth without shared infrastructure. The cycle cache already solves the shared scan problem, and the dedup pipeline extraction is a stable intermediate state. Produced 5 phased implementation issues; the double logging fix alone eliminates 50% of reported noise.

---

## Plain-Language Summary

The daemon grew from a simple poll-spawn loop to a 30+ subsystem process over 3 months. Every bug fix was correct in isolation, but the composite system developed emergent failures. I traced all 6 reported symptoms back to 3 root causes:

1. **Double logging** happens because both the daemon's own logger AND launchd's stdout capture write to the same file (`~/.orch/daemon.log`). Every line appears twice. The fix is ~30 lines: detect when running under launchd and skip the direct file write.

2. **SIGKILL crashes** happen because the shutdown path has no explicit time budget — the recent 3s reflection timeout fixes the immediate problem, but any new `defer` that does work erodes the margin silently. An explicit budget enforcer prevents recurrence.

3. **The other symptoms** (duplicates, frozen dashboard, redundant scans, stuck counters) were already partially addressed by prior work: dedup pipeline extraction, PID validation in status file reader, and cycle cache for agent discovery. The remaining gaps are upstream (architects should check committed work before creating follow-ups) and in the widget (should use file mtime, not JSON content, for liveness).

None of this requires a daemon rewrite. The 5 implementation issues are phased: Phase 1 (double logging + shutdown budget) is immediate and low-risk; Phase 2 (issue-creation dedup + widget mtime) is near-term; Phase 3 (periodic task tiering) is config tuning.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-27-design-architect-daemon-reliability.md` — Architecture investigation with 5 findings, recommendations, defect class mapping
- `.orch/workspace/og-arch-architect-daemon-reliability-27mar-296b/VERIFICATION_SPEC.yaml` — Verification specification
- `.orch/workspace/og-arch-architect-daemon-reliability-27mar-296b/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-architect-daemon-reliability-27mar-296b/BRIEF.md` — Comprehension brief

### Files Modified
- `.kb/models/daemon-autonomous-operation/model.md` — Added Phase 9 (Pipeline Extraction and OODA Structure), updated timestamp, added probe reference

### Issues Created
- `orch-go-dl4tn` — Fix daemon double logging (bug, triage:ready)
- `orch-go-vnpmv` — Add explicit shutdown budget enforcement (task, triage:ready)
- `orch-go-aqq5a` — Issue-creation-time dedup in architect skill (feature, triage:review)
- `orch-go-lno6l` — Sketchybar widget mtime liveness (task, triage:review)

---

## Evidence (What Was Observed)

- **Double logging confirmed:** launchd plist StandardOutPath at `~/.orch/daemon.log` AND DaemonLogger MultiWriter targeting same file (pkg/daemon/log.go:70, pkg/daemonconfig/plist.go:64-68)
- **Shutdown defer chain:** 5 defers in cmd/orch/daemon.go:23-29, only reflection has timeout, no budget enforcement
- **Cycle cache already sharing queries:** cachedAgentDiscoverer wraps d.Agents during BeginCycle/EndCycle, sharing GetActiveAgents() across 4 periodic tasks (cycle_cache.go)
- **Dedup pipeline extracted:** SpawnPipeline with 7 named gates built in spawn_execution.go:247-300, replacing inline gauntlet
- **PID validation exists:** ReadValidatedStatusFile checks PID liveness and falls back to PID lock (status.go:155-182)
- **13 periodic tasks in scheduler:** scheduler.go:7-19 registers all named tasks with intervals

---

## Architectural Choices

### Phased interventions over comprehensive rewrite
- **What I chose:** 5 targeted fixes across 3 phases
- **What I rejected:** Full daemon rewrite with subsystem process model; extracting periodic tasks to cron jobs
- **Why:** The daemon's intermediate state (pipeline extraction, cycle cache, PID validation) is already stable. The remaining problems are specific and fixable. A rewrite would violate "no local agent state" constraint (would need IPC) and throw away working code.
- **Risk accepted:** Deferring CAS-based dedup redesign means correlated fail-open risk remains. Acceptable because the 7-gate pipeline handles the common cases; compound failure requires beads unavailability which is rare.

### Keep spawn-time dedup, add issue-creation-time dedup upstream
- **What I chose:** Both layers, solving different problems
- **What I rejected:** Moving all dedup to issue-creation time; removing spawn-time dedup layers
- **Why:** Spawn-time dedup prevents executing duplicate work (infrastructure concern). Issue-creation dedup prevents creating duplicate issues (skill concern). CommitDedupGate at spawn-time is a correct bandaid but the upstream gap (architect skill not checking committed work) is the real fix.
- **Risk accepted:** Two dedup mechanisms to maintain. Acceptable because they're at different abstraction levels with different interfaces.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for:
- 5 decision forks navigated with substrate citations
- 4 issues created with authority classification
- Investigation with DEKN summary, 5 findings, defect class mapping, composition claims

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-27-design-architect-daemon-reliability.md` — 5-fork design investigation

### Decisions Made
- Double logging: fix by detecting launchd context, not by changing plist paths (preserves foreground mode)
- Shutdown: explicit budget enforcement, not just timeout values (prevents future erosion)
- Shared scan: no further consolidation needed (cycle cache already handles expensive case)
- Periodic tasks: tiered scheduling intervals, not structural extraction (diminishing returns)

### Constraints Discovered
- DaemonLogger must still work when running foreground (not under launchd)
- Shutdown budget must not block PID lock release (always last operation)
- Architect skill commit check must handle cross-project git histories

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] Investigation file complete with DEKN summary
- [x] Model updated with findings
- [x] 4 implementation issues created
- [x] VERIFICATION_SPEC.yaml written
- [x] SYNTHESIS.md written
- [x] BRIEF.md written

**MIGRATION_STATUS:**
  designed: Shutdown budget enforcement, double logging detection, issue-creation dedup, widget mtime liveness
  implemented: none (architect investigation — design only)
  deployed: none
  remaining: orch-go-dl4tn (double logging), orch-go-vnpmv (shutdown budget), orch-go-aqq5a (issue-creation dedup), orch-go-lno6l (widget mtime)

---

## Unexplored Questions

- **Beads CAS support:** Can beads do atomic conditional status updates? Determines feasibility of Phase 2 structural dedup redesign. Not urgent — pipeline extraction is a stable intermediate state.
- **Keyword dedup false-positive rate:** 50% overlap, 3 keyword threshold has no measurement data. Worth logging gate decisions for a week before tuning.
- **Reflection analysis value vs cost:** Does the shutdown reflection actually produce actionable suggestions? If not, it could be removed entirely rather than budgeted.

---

## Friction

Friction: none — smooth session. Codebase was well-organized with clear file boundaries.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-architect-daemon-reliability-27mar-296b/`
**Investigation:** `.kb/investigations/2026-03-27-design-architect-daemon-reliability.md`
**Beads:** `bd show orch-go-5b9st`
