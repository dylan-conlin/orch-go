# Probe: Control Plane Circuit Breaker Heuristic Calibration

**Model:** daemon-autonomous-operation
**Date:** 2026-02-14
**Status:** Complete
**Beads:** orch-go-6un

## Question

The current circuit breaker uses single-day commit count (MAX_COMMITS_PER_DAY) as its primary signal. The entropy spiral deep analysis shows the dangerous signal was *sustained unverified velocity* (45 commits/day for 26 days without human verification), not single-day spikes. Does the current heuristic correctly distinguish between:
- A burst day with human supervision (59 commits, normal batch review) — should NOT halt
- Sustained autonomous velocity without human verification — SHOULD halt

## What I Tested

### Test 1: Current heuristic behavior on Feb 14 (day-one false positive)
- Git log shows 61 commits on Feb 14 — a normal high-output day with orchestrator actively reviewing
- MAX_COMMITS_PER_DAY was originally 20, tripped immediately. Raised to 100.
- daily-commits.log shows escalating counts: 56, 57, 57, 58, 59, 60, 61 (appended per commit)

### Test 2: Commit type analysis on Feb 14
- 11 docs, 7 feat, 6 architect, 5 investigation, 4 refactor, 4 fix, 3 test, + misc
- Many are knowledge-producing commits (docs, architect, investigation) that don't carry entropy risk
- Only 4 fix commits — well below the 0.96:1 fix:feat ratio from the entropy spiral

### Test 3: Entropy spiral signature analysis
- Third spiral: 45 commits/day average for 26 days
- 0 human commits in entire 26-day period
- fix:feat ratio during spiral: 0.96:1
- Pattern: moderate sustained velocity + zero human interaction + high fix:feat ratio

### Test 4: Rolling window feasibility in shell
- daily-commits.log stores running counts per day (multiple entries per day)
- Can extract last entry per day using `tac | awk '!seen[$1]++'`
- macOS `stat -f %m` gives mtime in epoch seconds for heartbeat staleness
- All calculations feasible in pure shell

## What I Observed

1. **Single-day count is wrong signal.** Feb 14 had 61 commits during active human supervision — would have halted at 20 (original) or passed at 100. Neither threshold correctly captures the state "human is present and verifying."

2. **The entropy spiral's signature is multi-dimensional:** sustained velocity + zero human interaction + degrading quality (fix:feat ratio). No single metric captures it. The circuit breaker needs at least two signals: velocity over time AND human interaction recency.

3. **Knowledge commits inflate raw count.** 26 of 61 commits on Feb 14 were docs/investigation/architect/probe — knowledge-producing work that doesn't create the fix→bug→fix cycle that characterized the entropy spiral.

4. **Shell feasibility confirmed.** Rolling average calculation and heartbeat file staleness checking are both implementable in pure shell with standard macOS utilities.

## Model Impact

**Extends** the daemon-autonomous-operation model's understanding of circuit breaker signals:

- **Current model claim:** "Daemon halts when threshold exceeded" (single threshold)
- **Extension:** Single-threshold approach is insufficient. The circuit breaker needs a composite signal: rolling velocity window + human interaction recency. This is consistent with Implication #7 from the entropy spiral analysis: "Verification bandwidth is a real constraint — but it's the control plane's job to enforce it."

**Extends** with new invariant: Circuit breaker heuristics must distinguish between *supervised velocity* (human is present and acknowledging) and *autonomous drift* (agents running without human verification for multiple days).

## Design Recommendation

See investigation: `.kb/investigations/2026-02-14-design-control-plane-heuristics.md`
