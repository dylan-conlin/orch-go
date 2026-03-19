---
stability: foundational
---
# Decision: Verification Bottleneck Principle

**Date:** 2026-01-14
**Status:** Accepted
**Deciders:** Dylan (via 462 lost commits)

## Context

Two complete system rollbacks (Dec 21: 115 commits, Dec 27-Jan 2: 347 commits) revealed a fundamental constraint: every individual commit was correct, but the system spiraled into incoherence because changes happened faster than humans could verify behavior.

## Decision

**The system cannot change faster than a human can verify behavior.**

This is a hard constraint, not a guideline. When violated, the result is rollback.

## Rationale

### Evidence from Rollbacks

| Commit | Claim | Actual Code | Verdict |
|--------|-------|-------------|---------|
| e8b42281 | Show phase instead of "Starting up" | Added conditional logic | Real fix |
| eed04d69 | Phase:Complete authoritative | Removed check | Real fix |
| fc1c8482 | Filter closed issues | Added filter function | Real fix |
| 32cf0792 | Strip beads suffix | Added helper | Real fix |
| 57170ec0 | Fix status bar layout | Added CSS | Real fix |

**All fixes were real.** The failure was compositional: correct pieces that don't compose into a working system.

### Key Insight

Local correctness (each commit works) doesn't guarantee global correctness (system works) when changes outpace verification. This is counterintuitive - most engineers assume "if each commit is good, the system is good."

## Consequences

### Constraints

1. **One human verification per 3 changes** - Actually run the system, not just read synthesis files
2. **Iteration budgets** - Max 3 iterations before human review; explicit convergence criteria
3. **Meta-work limits** - If >50% of agents are fixing orchestration, pause and verify foundation
4. **Pacing** - If verification takes 10 minutes, changes cannot happen faster than every 10 minutes

### What This Changes

**Before:** Trust synthesis files, trust commit messages, trust agent reports. High velocity = success.

**After:** Verify behavior, cap changes per verification, human in loop ≤ every 3 changes. Sustainable velocity = success.

### The Test

When tempted to skip verification: "Am I building on verified foundations, or am I assuming local correctness implies global correctness?"

## Alternatives Considered

1. **More automation safeguards** - Add guardrails, circuit breakers, preflight checks
   - Rejected: Treats symptoms, not cause. After first spiral we added 7 guardrails; same failure happened 6 days later.

2. **Better agent quality** - Improve synthesis, add more tests per agent
   - Rejected: Agents weren't broken. Every fix was real. Quality wasn't the bottleneck.

## Related

- **Source:** `.kb/investigations/2026-01-10-inv-trace-verification-bottleneck-story-system.md`
- **Post-mortems:** `2025-12-21-inv-deep-post-mortem-last-24.md`, `2026-01-02-system-spiral-dec27-jan02.md`
- **Pattern:** Understanding Lag (observability improves faster than understanding)

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-15-inv-fix-tmux-socket-path-orch.md
- .kb/investigations/2026-03-17-design-ground-truth-metric-injection-daemon-orient.md
- .kb/investigations/archived/2026-01-10-inv-trace-verification-bottleneck-story-system.md
