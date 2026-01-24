<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Backend selection in orch spawn uses 5-level priority chain: flags > project config > global config > default opencode, with advisory-only infrastructure warnings.

**Evidence:** Code analysis shows `resolveBackend()` function implements clear priority chain; `addInfrastructureWarning()` only adds warnings, never overrides.

**Knowledge:** User intent (flags) respected over config; infrastructure safety is advisory; default opencode optimizes cost; `--opus` flag implies claude backend.

**Next:** Close investigation - backend selection priority is clearly documented and working as designed.

**Promote to Decision:** recommend-yes - establishes architectural pattern for config precedence and advisory safety mechanisms

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

# Investigation: Backend Selection Priority Work Orch

**Question:** How does the backend selection priority work in orch spawn? Find the code that determines which backend (claude, opencode, docker) is used and explain the priority order.

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Backend selection uses a 5-level priority chain

**Evidence:** The `resolveBackend()` function in `cmd/orch/backend.go` implements a clear priority chain: 1) `--backend` flag, 2) `--opus` flag, 3) project config, 4) global config, 5) default opencode.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/backend.go:23-83` - `resolveBackend()` function with documented priority chain

**Significance:** This establishes a clear, testable precedence order where explicit user flags override configuration, and configuration overrides defaults.

---

### Finding 2: Infrastructure detection warns but doesn't override

**Evidence:** The `addInfrastructureWarning()` function checks for critical infrastructure work (OpenCode server files) and adds advisory warnings but never changes the backend selection.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/backend.go:85-100` - `addInfrastructureWarning()` function that only adds warnings

**Significance:** Infrastructure safety is advisory-only - users get warnings about potential server restarts but their backend choice is respected.

---

### Finding 3: Default backend is opencode for cost optimization

**Evidence:** When no flags or config specify a backend, the system defaults to "opencode" backend (line 80 in backend.go) with reason "default (opencode for cost optimization)".

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/backend.go:79-82` - Default fallback to opencode backend

**Significance:** The system prioritizes cost efficiency by default, using OpenCode with DeepSeek model instead of Claude CLI with Opus (Max subscription).

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
