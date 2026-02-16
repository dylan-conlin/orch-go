# Session Synthesis

**Agent:** og-work-knowledge-maintenance-automation-15feb-263c
**Issue:** orch-go-ymjz
**Duration:** 2026-02-15T18:15:00Z → 2026-02-15T18:45:00Z
**Outcome:** success

---

## Plain-Language Summary

The system detects stale knowledge models at spawn time (working since Feb 14) but has no mechanism to actually *fix* them — agents see warnings about outdated models, then nothing happens. This design session produced a complete three-layer automation loop: (1) detection at spawn time and via periodic daemon reflection, (2) throttled issue creation with backpressure to prevent flooding the verification queue, and (3) human-in-loop remediation through reflection sessions where the orchestrator updates models directly or delegates to architect workers for severe drift. The key tension — automation creating work faster than a human can verify — is resolved through spawn-count thresholds (3 stale spawns before creating an issue), backpressure (max 3 open model-update issues), and batching (group related stale models).

## Verification Contract

See: `VERIFICATION_SPEC.yaml` in this workspace. Key outcomes: decision record with 5 navigated forks, explicit verifiability-first compliance, and 5-phase implementation plan.

---

## TLDR

Designed complete knowledge maintenance automation loop closing the gap between "detect stale model" and "update stale model", throttled to human verification bandwidth. Decision record produced at `.kb/decisions/2026-02-15-knowledge-maintenance-automation-loop.md`.

---

## Delta (What Changed)

### Files Created
- `.kb/decisions/2026-02-15-knowledge-maintenance-automation-loop.md` - Full decision record for three-layer automation loop
- `.orch/workspace/og-work-knowledge-maintenance-automation-15feb-263c/SYNTHESIS.md` - This file
- `.orch/workspace/og-work-knowledge-maintenance-automation-15feb-263c/VERIFICATION_SPEC.yaml` - Verification specification

### Commits
- (pending final commit)

---

## Evidence (What Was Observed)

- Spawn-time staleness detection is **fully implemented and behaviorally verified** in `pkg/spawn/kbcontext.go` (lines 979-1136). Probe passed Feb 15: 4 models detected with staleness warnings.
- Daemon periodic reflection in `pkg/daemon/reflect.go` only handles `synthesis` type with issue creation. Other types are surface-only.
- Open issue `orch-go-fq5` (P3) exists for `kb reflect --type model-drift` but is blocked by code_refs backfill (`orch-go-bm9`).
- Verifiability-first decision (Feb 14) established: `C × V ≤ 1` where C = change rate, V = verification time. Any automation must respect this constraint.
- Jan 6 investigation recommended two-tier automation (synthesis + open auto-create issues). Only synthesis was implemented.
- 12+ stale file references across 24 models (Feb 14 investigation). ~50% of models have at least one stale reference.
- Reflection sessions guide already defines lanes, ATS scoring, and session protocol. Model-drift is not yet a lane.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/decisions/2026-02-15-knowledge-maintenance-automation-loop.md` - The full automation loop design

### Decisions Made
- Spawn-count threshold (3) over time-based for model-drift issues: measures actual agent impact
- Backpressure at 3 open issues: matches verification cadence (1-3 per session)
- 4-hour model-drift cadence: prevents burst-spawning from immediately creating issues
- Orchestrator-primary remediation: models are orchestrator understanding, not auto-fixable
- Batching for same-domain staleness: reduces issue count without losing coverage

### Constraints Discovered
- Model-update issue creation requires deduplication by model path (same model shouldn't spawn multiple issues)
- Circuit breaker at 5 open model-update issues (daemon halts creation, logs warning)
- Model updates are Tier 2 verification (explain-back only, no behavioral gate)

### Externalized via `kn`
- `kb quick decide "Model-drift issues use spawn-count threshold (3 stale spawns) not time-based"` → kb-f3e775
- `kb quick constrain "Model-update issue backpressure: max 3 open model-maintenance issues before daemon halts creation"` → kb-b1799f

---

## Next (What Should Happen)

**Recommendation:** close

The design session is complete. The decision record is ready for orchestrator review and acceptance.

### Implementation Readiness

The decision record contains a 5-phase implementation plan:

1. **Phase 1: Staleness Event Recording** (orch-go, `pkg/spawn/staleness_events.go`) — Record spawn-time staleness events to `~/.orch/model-staleness-events.jsonl`
2. **Phase 2: Daemon Model-Drift Reflection** (orch-go, `pkg/daemon/`) — Daemon reads events, creates issues with throttling
3. **Phase 3: Daemon Open Reflection** (orch-go + kb-cli) — Add `open` type auto-issue creation
4. **Phase 4: Completion Reverse Linkage** (orch-go, `cmd/orch/`) — Inform orchestrator about model impact at completion
5. **Phase 5: Reflection Guide Update** (`.kb/guides/`) — Add model-drift lane to reflection session protocol

**Depends on:** `orch-go-bm9` (backfill code_refs in 24 models) must complete before Phase 2 can create meaningful model-drift issues.

### Suggested Follow-up Issues

After decision is accepted:
- Epic: "Implement knowledge maintenance automation loop" with children for each phase
- Update `orch-go-fq5` (kb reflect --type model-drift) to reference this decision

---

## Unexplored Questions

- **Perpetually stale models:** If same model triggers >5 issues in 30 days, its code_refs may be too granular. Need heuristic to surface this for code_ref refinement.
- **Cross-repo model staleness:** Models referencing files in other repos (e.g., opencode fork). Deferred — current models are primarily intra-repo.
- **Agent correction feedback loop:** When agents verify stale model claims and find corrections, how does that flow back into model updates? Currently via SYNTHESIS.md → orchestrator review → manual update. Could be more automated.
- **kb reflect archived/synthesized directory false positives:** Known bug (Failure Mode 5 in kb-reflect-cluster-hygiene model). Affects synthesis type accuracy.

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-work-knowledge-maintenance-automation-15feb-263c/`
**Beads:** `bd show orch-go-ymjz`
