# Decision: Verification Bottleneck Principle

**Date:** 2026-01-04
**Status:** Accepted
**Context:** Two system spirals demonstrated that local correctness doesn't ensure global correctness

## Decision

Add "Verification Bottleneck" as an LLM-First principle: The system cannot change faster than a human can verify behavior.

## Context

Between Dec 21 and Jan 2, the orchestration system experienced two major spirals:

**Dec 21 spiral:**
- 115 commits in 24 hours (3x normal rate)
- 12 test iterations in 9 minutes
- 70% of agents completed without synthesis
- Result: Complete rollback

**Dec 27 - Jan 2 spiral:**
- 347 commits in 6 days
- 40 "fix:" commits
- Agents modifying agent infrastructure (dashboard, status logic, spawn system)
- Result: Complete rollback to Dec 27

**The critical finding:** Individual fixes were real. The code did what the commits said. But the system got worse because:
1. Each fix changed the ground truth for the next agent
2. No human verified behavior between changes
3. Artifacts (synthesis, commit messages) reported "fixed" while system degraded
4. Provenance chains terminated in other agents' output, not human observation

## Rationale

**Why existing principles didn't prevent this:**

| Principle | What It Enforced | Why It Wasn't Enough |
|-----------|------------------|---------------------|
| Provenance | Claims trace to evidence | Evidence was other agents' artifacts |
| Evidence Hierarchy | Code > artifacts | Agents verified their own code changes |
| Gate Over Remind | Can't skip steps | Gates passed - each agent was "correct" |
| Session Amnesia | Externalize state | State was externalized prolifically |

The gap: All principles optimize for **individual agent correctness**. None address **system-level coherence** or **human verification as a bottleneck**.

**The key insight from post-mortems:**

> "The problem wasn't fake fixes - it was too many fixes, too fast, with no verification that the *system* was working, only that individual *commits* were correct."

> "agents followed valid individual logic but no cross-agent coordination detected iteration loops or duplicate work"

**Why this is an LLM-First principle (not just system design):**

This isn't about our specific orchestration architecture - it's a fundamental constraint when multiple LLM agents can modify shared state. Any multi-agent system faces this:
- Agents can be individually correct while collectively destructive
- Self-reported completion is not verification
- Provenance that terminates in other agents' output is a closed loop

## Implications

**For orchestrators:**
- Verify behavior (look at the dashboard) not just output (read the synthesis)
- One change at a time, with pause to confirm it worked
- "I don't know if this is working" halts progress

**For tooling:**
- Build velocity limits into spawn/complete commands
- Surface when changes outpace verification capacity
- Flag when agents are modifying agent infrastructure

**For the principle hierarchy:**
- Provenance tells you *what* counts as evidence
- Evidence Hierarchy tells you *which* evidence to trust  
- Verification Bottleneck tells you *who* must observe the evidence

## Alternatives Considered

**Option A: Treat this as a process failure, not a principle**
- Rejected because: The same pattern occurred twice despite process improvements after the first spiral
- Process changes are reminders; principles are load-bearing

**Option B: Subsume under Provenance ("provenance must include human observation")**
- Rejected because: Provenance is about evidence chains, not verification rates
- The math of velocity vs. verification capacity is distinct from evidence quality

**Option C: Add tooling (velocity limits) without a principle**
- Rejected because: Tooling without principle is a gate without understanding
- Future decisions need the principle to guide them

## Evidence

- `orch-go/.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` - The smoking gun: "The individual fixes were real... The problem wasn't fake fixes"
- `orch-go/.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md` - First spiral analysis: "agents followed valid individual logic but no cross-agent coordination"
- Addy Osmani's "21 Lessons" comparison - surfaced the gap: your principles address *how* to do things correctly, not *whether* to stop
