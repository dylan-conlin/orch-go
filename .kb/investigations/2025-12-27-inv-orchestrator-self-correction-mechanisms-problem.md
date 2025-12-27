<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Existing pattern detection mechanisms (`kn tried`, `orch learn`, Session Reflection) are designed for knowledge gaps, not for detecting behavioral patterns in the AI orchestrator itself.

**Evidence:** Analyzed learning.go (tracks KB context gaps), attempts.go (tracks spawn/abandon patterns), and Session Reflection (manual end-of-session checkpoint) - all focus on external artifacts, none can observe in-session tool failures or repeated unsuccessful actions.

**Knowledge:** Self-correction requires observing action outcomes, not just knowledge state. Tool failures are ephemeral and untracked, making behavioral pattern detection impossible with current architecture.

**Next:** Implement action outcome tracking via PostToolUse hook or new `orch action-log` subsystem that can surface "repeated futile actions" to the AI.

---

# Investigation: Orchestrator Self-Correction Mechanisms

**Question:** Why doesn't the orchestrator self-correct when repeatedly making the same mistake (e.g., checking SYNTHESIS.md on light-tier agents)? What mechanisms could enable automatic pattern detection?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent (orch-go-eq8k)
**Phase:** Complete
**Next Step:** None - findings complete, recommendations ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Current Mechanisms Focus on Knowledge, Not Behavior

**Evidence:** 
- `orch learn` (learning.go:232-250) tracks `GapEvent` objects that record when KB context queries return insufficient results
- `kn tried` captures explicit declarations of "I tried X, it failed" but requires the AI to recognize and report the pattern
- Session Reflection (orchestrator SKILL.md:1316-1346) is a manual checkpoint asking "what was harder than it should have been" - dependent on AI self-awareness

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/learning.go:26-55` (GapEvent struct)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/gap.go:20-35` (GapType definitions)
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md:1316-1346` (Session Reflection)

**Significance:** All mechanisms track "did we have the right knowledge?" not "did the action we took work?" The failure mode (checking SYNTHESIS.md on light-tier) is a behavioral pattern, not a knowledge gap - the knowledge EXISTS (tier system documented) but the orchestrator doesn't apply it correctly.

---

### Finding 2: Tool Failures Are Ephemeral and Untracked

**Evidence:**
- Examined hooks system at `~/.claude/hooks/`
- PostToolUse hook (`post-tool-use.sh`) only triggers on specific commands (`orch spawn|complete|abandon|clean`)
- No mechanism exists to log when Read tool returns empty, or when a fallback is used
- The failure sequence "Read SYNTHESIS.md → empty → Read bd show" leaves no trace

**Source:**
- `/Users/dylanconlin/.claude/hooks/post-tool-use.sh` (only tracks orch commands)
- Claude Code API documentation shows tool results are transient

**Significance:** Without observability into action outcomes, pattern detection is impossible. The AI's "working memory" of "I tried X, it didn't work" is lost between tool calls and especially between sessions.

---

### Finding 3: Retry Pattern Detection Exists But Only for Spawn/Complete Cycles

**Evidence:**
- `pkg/verify/attempts.go` implements `FixAttemptStats` that tracks spawn/abandon/complete patterns
- `IsRetryPattern()` returns true when `SpawnCount > 1 && AbandonedCount > 0`
- This detects repeated agent failures but not within-session orchestrator behavioral patterns
- Events logged to `~/.orch/events.jsonl` only for spawn/complete lifecycle events

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/attempts.go:29-36` (IsRetryPattern)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/attempts.go:68-154` (GetFixAttemptStats)

**Significance:** The infrastructure for event logging and pattern detection exists, but it's scoped to agent lifecycle, not orchestrator behavior. The pattern could be extended.

---

### Finding 4: Tier System Is Correctly Implemented But Not Surfaced to Orchestrator AI

**Evidence:**
- `VerifyCompletionWithTier` (check.go:493-494) correctly skips SYNTHESIS.md check when `tier != "light"`
- Workspace `.tier` file exists and is read
- But orchestrator AI (Claude) doesn't get pre-completion context about tier - it learns tier exists only AFTER trying to read SYNTHESIS.md

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go:488-503`

**Significance:** This specific failure mode could be prevented by surfacing tier information before completion. But this is a band-aid - the systemic issue is that behavioral pattern detection doesn't exist.

---

## Synthesis

**Key Insights:**

1. **Knowledge vs Behavior Gap** - Current mechanisms assume "if we know the right thing, we'll do the right thing." But the orchestrator knew about tiers - it just didn't apply that knowledge consistently. Behavioral patterns require observing outcomes, not just state.

2. **Ephemeral Tool Results** - Without persistence of action outcomes, pattern detection is architecturally impossible. Each tool invocation is isolated - there's no "session action log" that could surface "you've tried X 3 times with no success."

3. **Existing Infrastructure Can Be Extended** - The events.jsonl + pattern detection pattern from attempts.go could be adapted for behavioral logging. The PostToolUse hook mechanism exists but isn't used for comprehensive action tracking.

**Answer to Investigation Question:**

The orchestrator doesn't self-correct because:
1. No mechanism tracks action outcomes within a session
2. `kn tried` requires explicit recognition of failure (which the AI doesn't perceive - it just falls back)
3. Session Reflection is manual and occurs too late (end of session, when details are forgotten)
4. Cross-session memory (via kn/kb) works, but within-session behavioral patterns are invisible

Self-correction requires a feedback loop: `action → outcome → pattern detection → adjustment`. Currently only the first step (action) is observable to the AI.

---

## Structured Uncertainty

**What's tested:**

- ✅ learning.go tracks knowledge gaps, not behavioral patterns (code review + grep verification)
- ✅ attempts.go tracks spawn/abandon patterns at agent lifecycle level (code review)
- ✅ PostToolUse hook only triggers on specific orch commands (read hook script)
- ✅ Tier system is correctly implemented in verification code (check.go review)

**What's untested:**

- ⚠️ Whether PostToolUse hooks can access tool output for failure detection (would need to test with Claude Code API)
- ⚠️ Whether action logging would cause performance issues at scale (would need benchmarking)
- ⚠️ Whether AI can effectively use surfaced patterns for self-correction (would need live testing)

**What would change this:**

- Finding would be wrong if there's an existing action tracking mechanism I didn't discover
- Finding would be wrong if Claude Code has built-in retry detection that isn't exposed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Action Outcome Logging + Pattern Surfacing** - Log tool actions and outcomes to enable behavioral pattern detection, surface patterns to orchestrator via SessionStart or periodic injection.

**Why this approach:**
- Matches existing architecture (events.jsonl pattern, hook system)
- Provides observability without modifying core Claude Code behavior
- Can start simple (log failures) and extend (pattern analysis)
- Addresses root cause (ephemeral outcomes) not symptoms

**Trade-offs accepted:**
- Adds I/O overhead for action logging
- Requires orchestrator to read/act on surfaced patterns (not automatic correction)
- Patterns only detectable after N occurrences (not immediate)

**Implementation sequence:**
1. **Action logging subsystem** - Create `~/.orch/action-log.jsonl` that tracks tool invocations with outcomes (success/empty/error)
2. **Pattern analyzer** - Extend learning.go to detect "repeated action with same outcome" patterns
3. **Surfacing mechanism** - Inject patterns via SessionStart hook or on-demand `orch patterns` command

### Alternative Approaches Considered

**Option B: Tier-specific fix (surface tier in completion flow)**
- **Pros:** Solves the immediate problem quickly
- **Cons:** Doesn't address systemic issue - next behavioral mistake will also not self-correct
- **When to use instead:** If quick fix is needed while systemic solution is built

**Option C: LLM-based pattern detection in post-session reflection**
- **Pros:** Could detect more nuanced patterns
- **Cons:** Expensive (LLM call), still happens too late (end of session), depends on transcript availability
- **When to use instead:** If action logging proves insufficient

**Option D: Real-time PostToolUse pattern detection**
- **Pros:** Immediate feedback
- **Cons:** Complex to implement, may be too noisy, requires hook to access tool output
- **When to use instead:** If periodic surfacing is too slow

**Rationale for recommendation:** Option A provides the best balance of feasibility, systemic impact, and alignment with existing architecture. The events.jsonl + pattern detection pattern from attempts.go proves the approach works.

---

### Implementation Details

**What to implement first:**
1. Define `ActionEvent` struct (tool name, target, outcome, timestamp, workspace context)
2. Create PostToolUse hook extension that logs to action-log.jsonl
3. Implement `orch patterns` command to show detected behavioral patterns

**Things to watch out for:**
- ⚠️ Hook performance - keep logging lightweight
- ⚠️ Privacy - action logs may contain sensitive paths/data, should be local only
- ⚠️ False positives - "empty read" may be intentional (checking if file exists), need outcome context

**Areas needing further investigation:**
- What tool outputs are accessible to PostToolUse hooks?
- How to distinguish "expected empty" from "unexpected empty"?
- Should patterns trigger automatic kn tried entries?

**Success criteria:**
- ✅ Action outcomes are logged to persistent file
- ✅ `orch patterns` shows "you've tried X 3+ times with empty result"
- ✅ Orchestrator can act on surfaced patterns to avoid repeated futile actions

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/learning.go` - GapTracker and learning loop
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/gap.go` - Gap detection and analysis
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/attempts.go` - Retry pattern detection
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go` - Tier system verification
- `/Users/dylanconlin/.claude/hooks/post-tool-use.sh` - PostToolUse hook implementation
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` - Session Reflection section

**Commands Run:**
```bash
# Find existing pattern detection
grep "pattern.*detect|detect.*pattern" /Users/dylanconlin/Documents/personal/orch-go --include=*.go

# Find hooks
ls ~/.claude/hooks/

# Check kn capabilities
kn --help
```

**Related Artifacts:**
- **Decision:** Tiered spawn protocol - `.kb/decisions/` (uses .tier file in workspace)
- **Investigation:** This investigation provides design for behavioral pattern detection

---

## Investigation History

**2025-12-27 16:00:** Investigation started
- Initial question: Why doesn't orchestrator self-correct on repeated mistakes?
- Context: Observed pattern of checking SYNTHESIS.md on light-tier agents repeatedly

**2025-12-27 16:30:** Key finding - knowledge vs behavior gap
- Discovered that all existing mechanisms track knowledge state, not action outcomes
- Identified ephemeral nature of tool results as root cause

**2025-12-27 17:00:** Investigation completed
- Status: Complete
- Key outcome: Self-correction requires action outcome logging - current architecture lacks this capability

---

## Self-Review

- [x] Real test performed (code review, grep searches, file reads - not "analyzed logic")
- [x] Conclusion from evidence (based on actual code examination)
- [x] Question answered (why no self-correction, what would enable it)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED
