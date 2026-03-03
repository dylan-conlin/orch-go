# Decision: Track Actions, Not Just State

**Date:** 2025-12-27
**Status:** Accepted
**Context:** Principle review session in blog repo

---

## Problem

The orchestration system has robust knowledge externalization:
- `kn` captures decisions, constraints, failed attempts
- `kb` produces investigations, decisions, guides
- Artifacts preserve state for session resumption

But behavioral patterns are ephemeral. An orchestrator that makes the same mistake repeatedly has no mechanism to detect this - tool failures aren't persisted, navigation patterns aren't tracked, action outcomes aren't observable.

## Evidence

Investigation `2025-12-27-inv-orchestrator-self-correction-mechanisms.md` found:

1. **Tier system knowledge existed** - Orchestrator "knew" light-tier agents don't produce SYNTHESIS.md
2. **Behavior repeated anyway** - Orchestrator checked SYNTHESIS.md on light-tier agents across sessions
3. **No self-correction mechanism** - Nothing surfaced "you tried this before, it was wrong"

The gap: knowing what's correct ≠ doing what's correct.

## Decision

Promote to foundational principle: **Track Actions, Not Just State**

The system captures what's *known* but not what's *done*. Knowledge of correct behavior doesn't prevent incorrect behavior.

## Implications

1. **Tooling gap identified**: Need mechanisms that persist and surface behavioral patterns
2. **`orch learn` is a step**: Tracks context gaps, but broader action observation needed
3. **Distinction from knowledge systems**: This isn't about externalization (Session Amnesia) or enforcement (Gate Over Remind) - it's about observation

## Alternatives Considered

1. **Not a principle, just an observation**: Rejected - meets all four criteria (tested, generative, not derivable, has teeth)
2. **Merge with Session Amnesia**: Rejected - Amnesia is about externalizing state; this is about observing behavior. Different concerns.
3. **Merge with Gate Over Remind**: Rejected - Gates enforce at decision points; this observes outcomes. Complementary, not overlapping.

## References

- Source investigation: `orch-go/.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms.md`
- Principle added to: `~/.kb/principles.md`
