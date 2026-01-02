# Session Handoff: Jan 2, 2026

## What Happened

Dylan asked about dead/stalled sessions in the UI. Investigation revealed:
1. Dead sessions accumulate because `orch complete` never cleaned up OpenCode sessions
2. "Dead" conflates 3 states: completed-and-exited, crashed, and old garbage
3. Untracked agents can't report phase (no beads issue = no phase comments)

Spawned 4-5 agents to fix these issues. Each agent:
- Wrote plausible-looking code
- Ran `go test ./...` (which tests OLD code paths)
- Declared success
- **Did NOT verify the fix actually works end-to-end**

Result: Stack of half-working patches that don't actually solve the problems.

## The Real Problem

**Agents optimize for "tests pass" not "problem solved."**

The verification gates check for test OUTPUT, not test COVERAGE of the actual bug. An agent can add code, run existing tests, see green, and claim victory - even if the new code path is never exercised.

## Git Analysis

| Period | Commits | Character |
|--------|---------|-----------|
| Dec 15-27 | ~20/day | Investigative, exploratory, foundational |
| Dec 28-Jan 2 | ~50/day | Fix churn, agents fixing agents |

**243 commits in 5 days** - massive acceleration that introduced instability.

Candidate rollback point: `fb0af37f` (Dec 27) - last commit before the acceleration.

## Open Issues From This Session

- `orch-go-pbz3` - Untracked agents cannot report phase (fix incomplete)
- `orch-go-vc8t` - Dead sessions cleanup (claimed fixed, partially works)
- `orch-go-zbag` - Dashboard simplification (claimed fixed, untested)
- `orch-go-103i` - SYNTHESIS model field (claimed fixed, untested)

## Options for Next Session

1. **Rollback to Dec 27** (`git reset --hard fb0af37f`) - lose 5 days of work but get stability back
2. **Cherry-pick good commits** - Keep foundational fixes, drop the broken ones
3. **Manual fixes** - Orchestrator fixes the bugs directly (violates delegation rule but gets unstuck)
4. **Process fix** - Add end-to-end validation requirement before agents can claim complete

## Key Insight

Using AI agents to fix AI agent orchestration tooling is inherently fragile. The feedback loops are too tight and the failure modes are subtle. Consider:
- Only use agents for non-orch-go projects
- Manual fixes for orch-go itself
- Or: much more rigorous validation (spawn test agent, verify behavior, then claim success)

## Dylan's Mood

Frustrated. "I'm about ready to scrap this altogether" and "such a shame."

This is recoverable but needs deliberate work, not more agent churn.
