# Post-Mortem: System Spiral (Dec 27 - Jan 2)

## What Happened

Between Dec 27 and Jan 2, the orch-go system spiraled into a state where nothing worked correctly despite continuous "fix" commits. The dashboard showed dead/stale/stalled agents (internal states that confused the user), agents couldn't fix bugs, and the system reported "everything is working" while getting worse.

**By the numbers:**
- 347 commits in 6 days
- 40 "fix:" commits
- 109 investigation documents created
- Agent states grew from 5 to 7 (added `dead`, `stalled`)
- 3 time-based thresholds added (1min, 3min, 1hr)
- 1 revert of a breaking change
- Result: complete loss of trust in the system

## Timeline

- **Dec 27 (fb0af37f)**: System stable. 5 agent states: active, idle, completed, abandoned, deleted
- **Dec 28**: First crisis. Status unification feature broke dashboard (showed 0 agents). Reverted. "Crisis response" commit.
- **Dec 30**: Second crisis. "Session chaos" - agents completing without Phase comments, respawning already-fixed bugs, beads sync issues.
- **Dec 30**: Added `dead` and `stalled` states to represent failure modes
- **Jan 1-2**: Churn explosion. 12 changes to agents.ts. Multiple fixes trying to filter dead/stalled correctly.
- **Jan 2**: Rolled back to Dec 27.

## Root Causes

### 1. Agents Fixing Agent Infrastructure
The system was modifying itself. Agents changed:
- The dashboard that displays agents
- The status logic that tracks agents  
- The spawn system that creates agents

Each "fix" changed the ground truth. The next agent saw a different system than the last one.

### 2. Investigations Replaced Testing
When something broke, the response was "spawn an investigation agent" instead of "reproduce the bug and verify the fix." 

The investigations were thorough *documents*, but documenting a problem isn't the same as confirming it's fixed.

### 3. No Human Verification Loop
The human saw agent output (commit messages, synthesis files, status reports) but not actual behavior:
- Dashboard said "5 active agents" - was that true?
- Agent said "fix: filter closed issues" - were closed issues actually filtered?
- Synthesis said "outcome: success" - did the code work?

### 4. Velocity Over Correctness
- 347 commits in 6 days = one commit every 25 minutes
- ~58 agents spawned per day
- Nobody can verify quality at that rate
- The system rewarded shipping, not working

### 5. Complexity as Solution to Complexity
- Agents showed wrong status → "add more status types"
- That got confusing → "add time thresholds to categorize statuses"
- Each layer made the next bug harder to diagnose

## The Core Mistake

The system had no way to answer "is this actually working?" that wasn't self-reported by the system itself.

- Agents said they fixed things
- The dashboard said agents were healthy
- The commits said "fix"

But no one outside the loop was checking.

## Verification: Were the Fixes Real?

Examined 5 random "fix:" commits from the period:

| Commit | Claim | Actual Code | Verdict |
|--------|-------|-------------|---------|
| e8b42281 | Show phase instead of "Starting up" | Added conditional logic for phase/working/waiting | Real fix |
| eed04d69 | Phase:Complete authoritative for status | Removed hasActiveSession check | Real fix |
| fc1c8482 | Filter closed issues in pending-reviews | Added filterPendingReviewsByClosedIssues | Real fix |
| 32cf0792 | Strip beads suffix in artifact viewer | Added extractWorkspaceName helper | Real fix |
| 57170ec0 | Fix status bar layout at narrow widths | Added whitespace-nowrap, reduced gaps | Real fix |

**The individual fixes were real.** The code did what the commits said.

The problem wasn't fake fixes - it was too many fixes, too fast, with no verification that the *system* was working, only that individual *commits* were correct.

## What Would Prevent Repeating This

1. **Human verifies behavior, not just output, before the next change**
   - Actually look at the dashboard after a dashboard fix
   - Actually run the command after a CLI fix
   - Don't trust synthesis files or commit messages

2. **Agents don't modify agent infrastructure without manual review**
   - Dashboard code
   - Status determination logic
   - Spawn/completion flow
   - Anything that changes how agents are observed or controlled

3. **One change at a time with a pause to confirm it worked**
   - No parallel agents fixing related things
   - Wait for verification before next change

4. **"I don't know if this is working" halts progress**
   - Uncertainty is a valid state
   - Don't spawn more agents to investigate broken agents
   - Stop and verify manually

5. **Limit self-modification velocity**
   - The system cannot improve itself faster than a human can verify
   - If verification takes 10 minutes, changes cannot happen faster than every 10 minutes

## Recovery

Rolled back to Dec 27 (fb0af37f). At this state:
- Build passes
- Tests pass
- orch doctor shows services healthy
- orch status shows agents correctly
- orch spawn works
- Dashboard displays (5 simple states)

The foundation is sound. The execution pattern was the problem.
