# Session Synthesis

**Agent:** `og-inv-agent-status-states-26mar-a5e4`
**Issue:** `orch-go-8mw0y`
**Duration:** 2026-03-26 09:31:27 -> 2026-03-26 09:32
**Outcome:** success

---

## TLDR

I traced the liveness classifier in `pkg/verify/liveness.go` and documented its exact state machine: three statuses (`active`, `completed`, `dead`) selected by four mutually exclusive conditions. I also verified the behavior with targeted unit tests and clarified that caller-specific freshness policy lives outside the core classifier.

---

## Plain-Language Summary

This session answered a very specific question: when orch says an agent is still running, finished, or dead, what exact rules lead to that answer? The result is that the liveness system is intentionally simple. It only tracks whether an agent has reported a phase recently enough to count as running, whether it explicitly reported `Phase: Complete`, or whether it stayed silent long enough to be treated as dead.

That matters because some downstream commands add their own policy on top. `orch complete` uses the liveness result as a warning before destructive cleanup, while `orch abandon` adds a separate 30-minute recency check for non-complete phases. The investigation now captures that boundary clearly so future changes can tell whether they are altering the shared state machine or just caller behavior.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-agent-status-states-transitions-liveness.md` - Investigation artifact documenting the liveness states, transitions, evidence, and recommendations.
- `.orch/workspace/og-inv-agent-status-states-26mar-a5e4/VERIFICATION_SPEC.yaml` - Verification contract listing commands run and expected liveness semantics.
- `.orch/workspace/og-inv-agent-status-states-26mar-a5e4/SYNTHESIS.md` - Session synthesis for orchestrator review.

### Files Modified
- None.

### Commits
- None yet.

---

## Evidence (What Was Observed)

- `pkg/verify/liveness.go:102` contains the full liveness classifier with exactly four branches.
- `pkg/verify/beads_api.go:84` shows that only the latest matching `Phase:` comment is parsed, so phase history collapses to the last valid signal.
- `pkg/verify/liveness_test.go:193` proves the grace period is strict `< 5 min`, meaning the exact 5-minute boundary returns `dead`.
- `cmd/orch/abandon_cmd.go:281` shows abandon adds separate 30-minute recency logic instead of extending the shared liveness state machine.

### Tests Run
```bash
# Targeted liveness verification
go test ./pkg/verify -run 'TestVerifyLiveness|TestLivenessResult_Warning|TestVerifyLivenessGracePeriod'
# PASS: ok github.com/dylan-conlin/orch-go/pkg/verify 0.307s
```

---

## Verification Contract

See `.orch/workspace/og-inv-agent-status-states-26mar-a5e4/VERIFICATION_SPEC.yaml`.

Key outcomes:
- All four liveness conditions are covered by passing tests.
- The investigation distinguishes core liveness state from caller-specific recency policy.
- The 5-minute grace-period cutoff is explicitly documented and verified.

---

## Architectural Choices

No architectural choices - task was within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-agent-status-states-transitions-liveness.md` - Durable explanation of the liveness state machine and its transition rules.

### Decisions Made
- No code change recommendation because the observed behavior matches the accepted phase-based liveness decision.

### Constraints Discovered
- `VerifyLiveness` is intentionally coarse-grained; detailed freshness semantics for non-complete phases are delegated to callers.

### Externalized via `kb quick`
- No new learnings.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-8mw0y`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch abandon` always pass spawn time into `VerifyLiveness` so grace-period handling is consistent everywhere?
- Should reason codes be surfaced more directly in dashboards so `recently_spawned` false positives are easier to distinguish from phase-reported activity?

**Areas worth exploring further:**
- Caller-specific recency policies across the CLI.

**What remains unclear:**
- Whether the hotspot area now has enough layered policy to justify an architectural consolidation pass.

---

## Friction

No friction - smooth session.

---

## Session Metadata

**Skill:** `investigation`
**Model:** `openai/gpt-5.4`
**Workspace:** `.orch/workspace/og-inv-agent-status-states-26mar-a5e4/`
**Investigation:** `.kb/investigations/2026-03-26-inv-agent-status-states-transitions-liveness.md`
**Beads:** `bd show orch-go-8mw0y`
