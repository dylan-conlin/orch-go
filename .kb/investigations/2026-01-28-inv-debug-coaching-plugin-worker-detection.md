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

# Investigation: Debug Coaching Plugin Worker Detection

**Question:** Is the coaching plugin worker detection functioning correctly, and what is the current state of worker vs orchestrator detection?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** og-inv-debug-coaching-plugin-28jan-3d11
**Phase:** Investigating
**Next Step:** Examine why plugin is disabled and verify detection logic
**Status:** Active

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Coaching Plugin is Currently Disabled

**Evidence:** The coaching plugin file is named `coaching.ts.disabled` rather than `coaching.ts`. OpenCode plugins must have `.ts` extension to be loaded - the `.disabled` suffix prevents plugin loading.

**Source:** 
- File listing: `/Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/coaching.ts.disabled` exists
- Attempted to read `coaching.ts` failed with "file not found"
- Directory listing shows only `coaching.ts.disabled` (no active `coaching.ts`)

**Significance:** This means the coaching plugin is not currently running at all, so worker detection issues cannot be actively occurring. This investigation needs to determine whether the plugin was disabled due to worker detection problems, or for another reason.

---

### Finding 2: Plugin Was Disabled After Metadata-Based Detection Failed

**Evidence:** Investigation `2026-01-28-inv-debug-coaching-plugin-still-fires.md` (issue 21001, completed 13:43) found that the coaching plugin was "upgraded" from title-based to metadata-based worker detection, but session.created events do NOT include the metadata field. This caused worker detection to fail completely. The plugin was disabled at 13:39, during or shortly after this investigation.

**Source:**
- `.orch/workspace/og-inv-debug-coaching-plugin-28jan-b245/SYNTHESIS.md` - Details the root cause
- `stat` output showing coaching.ts.disabled modified at 13:39
- Timeline: audit investigation completed 13:35, plugin disabled 13:39, verify investigation completed 13:43

**Significance:** The plugin is disabled because the metadata-based detection approach doesn't work (metadata not available in session.created events). The recommended fix is to revert to title-based detection, which was proven working in prior investigations.

---

### Finding 3: Title-Based Detection Pattern Is Proven to Work

**Evidence:** Investigation `2026-01-28-inv-verify-coaching-plugin-worker-detection.md` tested two separate worker sessions and confirmed zero coaching alerts were fired despite 10+ tool calls each. The title-based pattern used was: `hasBeadsId && !isOrchestratorTitle` where beads ID matches `/\[[\w-]+-\d+\]/` and orchestrator pattern is `/-orch-/`.

**Source:**
- `.kb/investigations/2026-01-28-inv-verify-coaching-plugin-worker-detection.md` - Verification testing
- Tested sessions: `ses_3f9d325bbffetxp88HZ2YFlWhq` and `ses_3f9d0c828ffeGIx3oua2PzXlnx`
- Both had titles like `og-inv-verify-coaching-plugin-28jan-5e08 [orch-go-20993]`

**Significance:** This provides strong evidence that title-based detection is reliable for standard worker spawns with beads tracking. This is the fallback approach that should be used since metadata-based detection is not available.

---

### Finding 4: Current Code Uses Broken Metadata-Based Detection

**Evidence:** The disabled coaching plugin at line 2028 uses `sessionMetadata.role === "worker"` for worker detection. However, investigation 21001 confirmed that `info.metadata` is always an empty object (`{}`) in session.created events, so `sessionMetadata.role` is always undefined, making this check always false.

**Source:**
- `.opencode/plugins/coaching.ts.disabled:2028` - `const isWorker = sessionMetadata.role === "worker"`
- `.opencode/plugins/coaching.ts.disabled:2017` - `const sessionMetadata = info.metadata || {}`
- Investigation finding: session.created events have `properties.info.{id, title, directory, ...}` but NO metadata field

**Significance:** This confirms the plugin cannot work in its current state. It needs to be modified to use title-based detection (pattern: `hasBeadsId && !isOrchestrator`) before being re-enabled.

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
