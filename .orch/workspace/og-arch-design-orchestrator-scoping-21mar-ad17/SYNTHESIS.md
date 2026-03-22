# Session Synthesis

**Agent:** og-arch-design-orchestrator-scoping-21mar-ad17
**Issue:** orch-go-hlklh
**Duration:** 2026-03-21
**Outcome:** success

---

## Plain-Language Summary

The orchestrator currently has two ways to get work done: release issues to the daemon, or spawn agents directly. This creates a competing-executor problem where the orchestrator maintains execution concerns that duplicate daemon logic. Meanwhile, 69% of daemon routing uses the coarsest possible signal (issue type), and zero quality feedback enters the system (0 reworks across 1,102 completions).

This design separates the roles cleanly: the orchestrator becomes judgment-only (scope, type, label, describe issues), the daemon becomes the sole executor, and a comprehension queue prevents the pipeline from outrunning Dylan's understanding. The orchestrator's new superpower is enrichment -- adding `skill:*` labels and structured descriptions that lift daemon routing from 69% type-fallback to label-based precision. Completion becomes two-phase: daemon reclaims slots fast, orchestrator reviews for understanding later, with the queue throttling new spawns when comprehension falls behind.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- 7 design forks identified and analyzed
- 5 forks navigated with substrate-informed recommendations
- 2 forks surfaced as blocking questions (orch-go-24ysj, orch-go-n66ic)
- 4-phase implementation sequence proposed
- Investigation file complete with structured uncertainty

---

## TLDR

Designed orchestrator-as-scoping-agent architecture that separates judgment (orchestrator enriches issues) from execution (daemon spawns agents). Key mechanisms: remove orch spawn from orchestrator tool space, comprehension queue with daemon throttle, two-phase completion, bypass measurement. 7 forks analyzed, 5 navigated, 2 blocking questions surfaced.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-21-inv-design-orchestrator-scoping-agent-architecture.md` - Full architect investigation with 7 forks, 3 findings, recommendations
- `.orch/workspace/og-arch-design-orchestrator-scoping-21mar-ad17/SYNTHESIS.md` - This file
- `.orch/workspace/og-arch-design-orchestrator-scoping-21mar-ad17/VERIFICATION_SPEC.yaml` - Verification spec

### Files Modified
- None (design session, no code changes)

---

## Evidence (What Was Observed)

- 69% of daemon skill inference uses type-based fallback (475/682 unique issues) — verified via events.jsonl probe orch-go-j4ej7
- Only 12% of issues have skill:* labels (83/682) — the highest-confidence routing signal is starved of data
- 0 reworks across 1,102 completions — no quality signal enters the learning loop
- 37% of completions are daemon auto-completed with --force — zero quality check
- `orch work` already uses issue description as ORIENTATION_FRAME (work_cmd.go:196-213)
- Daemon already checks skill:* labels before type inference (skill_inference.go:233)
- Orchestrator SKILL.md currently lists orch spawn in its tool space (line 40)

---

## Architectural Choices

### Choice 1: Remove spawn from orchestrator, not add friction
- **What I chose:** Remove orch spawn from orchestrator tool space entirely
- **What I rejected:** Adding friction to spawn (like triage bypass requirement)
- **Why:** Advisory accretion gates had 100% bypass rate (decision 2026-03-17). Friction-based enforcement doesn't work. Clean removal is the only reliable mechanism.
- **Risk accepted:** Orchestrator loses ability to do urgent direct spawns. Mitigated by Dylan retaining bypass.

### Choice 2: Label-based comprehension queue, not status-based
- **What I chose:** `comprehension:pending` label on closed issues
- **What I rejected:** Separate beads status or new status field
- **Why:** Comprehension state is orthogonal to issue lifecycle. Work is done (closed), but understanding hasn't happened. A label captures this cleanly without changing the beads status model.
- **Risk accepted:** Labels on closed issues may have different persistence behavior. Needs verification.

### Choice 3: Targeted skill labeling, not mandatory
- **What I chose:** Orchestrator adds skill:* only when type-based inference would be wrong
- **What I rejected:** Orchestrator always specifies skill for every issue
- **Why:** Most issues have correct type-based inference (bug -> debugging, feature -> impl). Mandatory labeling adds ceremony without value for the majority case. Targeted labeling focuses orchestrator judgment where it matters.
- **Risk accepted:** Orchestrator must know what daemon would infer. Surfaced as blocking question orch-go-24ysj.

---

## Knowledge (What Was Learned)

### Decisions Made
- Orchestrator tool space should not include execution commands (orch spawn)
- Comprehension-gating should be structural (queue + throttle), not advisory
- Two-phase completion separates mechanical (fast) from comprehension (slow)
- Dylan retains bypass, orchestrator does not

### Constraints Discovered
- Advisory-only gates have 100% bypass rate (accretion gates decision) -- comprehension must be structural
- 777 orphaned investigations prove deferred synthesis becomes permanent deferral -- comprehension queue needs a ceiling
- Labels on closed issues need verification for persistence through bd close

---

## Next (What Should Happen)

**Recommendation:** escalate (strategic decision for Dylan)

### If Accepted (Promote to Decision)
1. Promote investigation to decision: `kb promote` with decision record
2. Create 4 implementation issues (one per phase):
   - Phase 1: Orchestrator skill rewrite (behavioral, no code)
   - Phase 2: Comprehension queue infrastructure
   - Phase 3: Two-phase completion modification
   - Phase 4: Bypass measurement
3. Create integration issue: behavioral verification that pipeline throttles correctly

### Blocking Questions (Need Dylan's Input)
- orch-go-24ysj: Should orchestrator specify skill:* labels, or daemon always infers?
- orch-go-n66ic: Should --inline be human-only?

---

## Unexplored Questions

- What is the right comprehension queue threshold? 5 is a starting guess.
- Does orchestrator enrichment measurably improve agent outcomes? No A/B test exists.
- Should the comprehension queue have an overnight exception (daemon runs, queue grows, orchestrator catches up next morning)?
- How does this interact with the existing escalation model (None/Info/Review/Block/Failed)?

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-orchestrator-scoping-21mar-ad17/`
**Investigation:** `.kb/investigations/2026-03-21-inv-design-orchestrator-scoping-agent-architecture.md`
**Beads:** `bd show orch-go-hlklh`
