<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode API requires directory via x-opencode-directory HTTP header, not JSON body parameter; added --workdir flag to spawn command.

**Evidence:** Direct curl test confirmed header works (returns correct directory in response); JSON body parameter is ignored by OpenCode server.

**Knowledge:** OpenCode API parameter passing is inconsistent (title/model in JSON, directory in header); --workdir enables cross-project spawning without cd.

**Next:** Close - fix implemented, tested, and committed in 9568983.

**Confidence:** High (90%) - Fix verified via manual testing and API inspection; unclear why OpenCode uses header vs JSON inconsistently.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Headless Spawn Registers Wrong Project

**Question:** Why do headless spawns register with the wrong project directory, and how can we add support for cross-project spawning?

**Started:** 2025-12-22 22:31
**Updated:** 2025-12-22 23:05
**Owner:** og-debug-headless-spawn-registers-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Missing --workdir flag in spawn command

**Evidence:** The beads issue (orch-go-ig16) describes expected usage: `orch spawn feature-impl 'task' --workdir /path/to/other-project`, but this flag doesn't exist. Checked spawn command flags in cmd/orch/main.go lines 240-257 - no workdir flag defined.

**Source:** 
- cmd/orch/main.go:240-257 (spawn command flag definitions)
- bd show orch-go-ig16 (beads issue description)

**Significance:** This is a missing feature, not a bug in existing code. Users cannot currently spawn agents for a different project without changing directories first.

---

### Finding 2: projectDir always defaults to current working directory

**Evidence:** In runSpawnWithSkill(), projectDir is set via `os.Getwd()` with no override mechanism. All sessions in OpenCode show directory as /Users/dylanconlin/Documents/personal/orch-go regardless of intended target.

**Source:**
- cmd/orch/main.go:958 (`projectDir, err := os.Getwd()`)
- curl http://127.0.0.1:4096/session (all sessions show same directory)

**Significance:** Without a way to override this, all spawns register with orch-go directory even if the workspace/SPAWN_CONTEXT are created elsewhere.

---

### Finding 3: CreateSession correctly passes directory to OpenCode API

**Evidence:** The CreateSession function in pkg/opencode/client.go properly includes Directory field in the request JSON payload and sends it to the OpenCode server. The infrastructure is already in place.

**Source:**
- pkg/opencode/client.go:273-303 (CreateSession implementation)
- Line 278: `Directory: directory` in request struct

**Significance:** The OpenCode API integration is correct - we just need to pass the right directory value. No changes needed to the API client code.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

Headless spawns register with the wrong directory because there's no --workdir flag to override the default os.Getwd() behavior (Finding 1, 2). The CreateSession API correctly supports directory specification (Finding 3), so the fix is straightforward: add a --workdir flag that sets projectDir when provided, defaulting to os.Getwd() for backward compatibility.

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

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
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
