# Decision: Reduce Investigation Overhead During Firefighting

**Date:** 2026-01-23
**Status:** accepted
**Deciders:** Dylan, Claude

## Context

After adding the docker backend, we experienced cascading issues across daemon, beads, opencode, and registry. In ~36 hours, 17 investigations were created across interconnected subsystems:

- 6 daemon capacity investigations
- 5 docker/backend investigations
- 5 opencode investigations
- 1 beads corruption investigation

These investigations were created while actively fixing bugs in real-time. The investigation artifact overhead added friction without proportional value - we needed fixes NOW, not documented understanding.

## Decision

**Triage investigation requirements based on situation:**

| Situation | Artifact | Rationale |
|-----------|----------|-----------|
| Bug blocking me right now | Commit message only | Knowledge is cheap, speed matters |
| Firefighting mode (multiple issues surfacing) | Commit messages, maybe 1 summary | Don't stop to document each fire |
| 3+ fixes in same area in 24h | STOP - step back, look at architecture | Signal of systemic issue, not more bugs |
| "How does X work?" | Investigation | Genuine question worth answering |
| Architecture/design decision | Decision record | Expensive to regenerate |
| Recurring pattern worth capturing | Guide or model | Reusable knowledge |

**The signal to watch for:** When investigations are being created faster than they're being read, the system is over-producing artifacts.

## Consequences

**Positive:**
- Less friction during active development
- Investigations reserved for actual questions
- Commit history becomes primary bug-fix documentation
- 3+ fixes signal triggers architectural review instead of more tactical work

**Negative:**
- Some bug context may be lost (acceptable - commit messages capture enough)
- Requires judgment about when to investigate vs just fix
- May need to retrofit understanding later if pattern recurs

**Risks:**
- Could swing too far toward "just ship it" without capturing anything
- Need discipline to recognize the 3+ fixes signal

## Alternatives Considered

1. **Keep current approach** - All significant work gets investigation
   - Rejected: Evidence shows this creates artifact overhead during firefighting

2. **Gastown approach** - No persistent knowledge artifacts, seance predecessors
   - Rejected: Goes too far, loses valuable architectural knowledge

3. **Time-boxed investigations** - Cap at 30 mins during firefighting
   - Considered: Might work but doesn't address root cause (creating investigations for bugs)

## References

- Evidence: 17 investigations in 36 hours (2026-01-23)
- Gastown comparison: `.kb/investigations/2026-01-23-inv-gastown-gap-analysis.md`
- Principle: "Knowledge is expensive" needs nuance - some knowledge IS cheap
