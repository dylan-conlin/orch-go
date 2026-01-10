<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Trace Verification Bottleneck Story System

**Question:** How did two system spirals (Dec 21, Dec 27-Jan 2) reveal the Verification Bottleneck principle, and how can we tell this story as a cautionary tale for engineers running AI agents?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** Agent og-inv-trace-verification-bottleneck-10jan
**Phase:** Investigating
**Next Step:** Extract timeline, key quotes, and teaching framework from post-mortems
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Two Distinct Spirals With Same Root Cause

**Evidence:**

**First Spiral (Dec 21):**
- 115 commits in 24 hours (3x normal velocity)
- 12 test iterations in 9 minutes (09:45-09:54)
- 132 workspace directories created
- 70% of agents completed without SYNTHESIS.md
- 27 abandoned agents, 238 orphaned OpenCode sessions
- Pattern: agents spawning agents without circuit breakers

**Second Spiral (Dec 27 - Jan 2):**
- 347 commits in 6 days
- 40 "fix:" commits
- 109 investigation documents created
- Agent states grew from 5 to 7 (added `dead`, `stalled`)
- 3 time-based thresholds added (1min, 3min, 1hr)
- 1 revert of breaking change
- Result: complete loss of trust, full rollback to Dec 27 baseline

**Source:**
- `.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md` - First spiral analysis
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` - Second spiral analysis

**Significance:** Two independent spirals with the same failure mode suggests a systemic pattern, not a one-time accident. The second spiral happened AFTER the first was analyzed, meaning the lesson wasn't learned.

---

### Finding 2: The Critical Quote - Local Correctness ≠ Global Correctness

**Evidence:**

From Jan 2 post-mortem, section "Verification: Were the Fixes Real?":

> "Examined 5 random 'fix:' commits from the period:
>
> | Commit | Claim | Actual Code | Verdict |
> |--------|-------|-------------|---------|
> | e8b42281 | Show phase instead of "Starting up" | Added conditional logic | Real fix |
> | eed04d69 | Phase:Complete authoritative | Removed check | Real fix |
> | fc1c8482 | Filter closed issues | Added filter function | Real fix |
> | 32cf0792 | Strip beads suffix | Added helper | Real fix |
> | 57170ec0 | Fix status bar layout | Added CSS | Real fix |
>
> **The individual fixes were real.** The code did what the commits said.
>
> The problem wasn't fake fixes - it was too many fixes, too fast, with no verification that the *system* was working, only that individual *commits* were correct."

**Source:**
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md:69-82`

**Significance:** This is the key insight for the blog post. The agents weren't lying or broken - they were doing exactly what they said. But local correctness (each commit does what it says) doesn't guarantee global correctness (the system works as a whole). This is the verification bottleneck in action.

---

### Finding 3: First Spiral Had Visible Circuit Breaker Failures

**Evidence:**

From Dec 21 post-mortem, iteration loop analysis:

```
09:54:26 - investigation: final test of tmux fallback mechanism
09:53:34 - feat: add tmux fallback for status and tail
09:53:27 - Add SYNTHESIS.md for tmux fallback iteration 11
09:53:12 - workspace: add synthesis for tmux fallback iteration 10
09:52:35 - investigation: iteration 11 regression test
09:52:26 - investigation: test tmux fallback iteration 10
09:52:25 - synthesis: iteration 12 tmux fallback regression test
09:51:32 - inv: test tmux fallback mechanism iteration 12
09:51:10 - investigation: iteration 9 tmux fallback regression testing
09:50:43 - Add SYNTHESIS.md for iteration 7
09:50:31 - investigation: test tmux fallback iteration 8
09:49:46 - investigation: verify tmux fallback (iteration 6)
09:49:13 - investigation (iteration 5): test discovered edge case
09:48:09 - investigation: test tmux fallback mechanism (iteration 4)
```

Missed checkpoint identified:
> "**Missed Checkpoint 4: After Iteration 8 (Dec 21, 09:50)**
> - 4 iterations already confirmed the same behavior (iterations 5-8)
> - **Should have stopped**: Regression testing showed stability, no need for iterations 9-12
> - **Why missed**: No 'sufficient evidence' heuristic - agents kept testing without convergence criteria"

**Source:**
- `.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md:77-112,320-347`

**Significance:** Shows the runaway pattern clearly - an investigation discovered an edge case (iteration 5), spawned another iteration to verify, which spawned another, creating an endless testing loop with no circuit breaker. Each iteration was "valid work" but nobody stopped to ask "have we tested enough?"

---

### Finding 4: Second Spiral - Agents Fixing Agent Infrastructure

**Evidence:**

From Jan 2 post-mortem, Root Cause #1:

> "### 1. Agents Fixing Agent Infrastructure
> The system was modifying itself. Agents changed:
> - The dashboard that displays agents
> - The status logic that tracks agents
> - The spawn system that creates agents
>
> Each 'fix' changed the ground truth. The next agent saw a different system than the last one."

And Root Cause #2:

> "### 2. Investigations Replaced Testing
> When something broke, the response was 'spawn an investigation agent' instead of 'reproduce the bug and verify the fix.'
>
> The investigations were thorough *documents*, but documenting a problem isn't the same as confirming it's fixed."

**Source:**
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md:26-39`

**Significance:** The second spiral had a different trigger (self-modification) but same failure mode (no human verification loop). Agents investigating agents investigating agents. Local correctness (each investigation was thorough) didn't guarantee global correctness (system actually worked).

---

### Finding 5: The Principle That Emerged

**Evidence:**

From Jan 2 post-mortem, section "What Would Prevent Repeating This":

> "5. **Limit self-modification velocity**
>    - The system cannot improve itself faster than a human can verify
>    - If verification takes 10 minutes, changes cannot happen faster than every 10 minutes"

And from kb quick entries:
> {"id":"kb-d2ac7d","type":"decision","content":"Success in spawn telemetry = verification_passed && !forced","status":"active","created_at":"2026-01-09T13:25:38","reason":"Ensures work meets project standards without human bypass, respecting the Verification Bottleneck principle."}

**Source:**
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md:105-107`
- `.kb/quick/entries.jsonl:480`

**Significance:** The principle crystallized from two rollbacks: **"The system cannot change faster than a human can verify behavior."** This is not about agent quality (fixes were real) or agent speed (velocity was high) - it's about verification bandwidth being the limiting factor in a human-AI collaborative system.

---

### Finding 6: The 'Aha' Moment - When Understanding Shifted

**Evidence:**

The first spiral (Dec 21) produced detailed analysis of what went wrong:
- 7 missing guardrails identified
- 5 missed checkpoints documented
- Implementation recommendations for preflight checks, completion gates, reconciliation

But the **same pattern repeated** 6 days later (Dec 27-Jan 2).

The aha moment appears to have been the second rollback + the verification passage quoted in Finding 2. The shift from:
- "We need better guardrails" (tactical)
To:
- "The system cannot change faster than verification" (principle)

The launchd post-mortem (Jan 9) shows the principle being applied:
> "**The test:** One 5-minute prototype revealed what 2 weeks of investigation and patching missed."

**Source:**
- `.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md` (tactical analysis)
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` (principle emergence)
- `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md:214` (principle application)

**Significance:** The aha moment wasn't during the first spiral - it was after the SECOND spiral, when the pattern repeated despite detailed analysis. Understanding shifted from "we need more automation safeguards" to "verification is the fundamental bottleneck, not automation speed."

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
