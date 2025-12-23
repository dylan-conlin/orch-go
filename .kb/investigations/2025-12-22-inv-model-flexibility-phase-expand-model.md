<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Confidence:** [Level] ([Percentage]) - [Key limitation in one phrase]

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

# Investigation: Model Flexibility Phase Expand Model

**Question:** How can we extend model selection support to headless spawn mode (currently only works for inline and tmux modes)?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** feature-impl agent
**Phase:** Investigating
**Next Step:** Implement model parameter support in CreateSession API
**Status:** In Progress
**Confidence:** High (85%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: CreateSessionRequest missing Model field

**Evidence:** The `CreateSessionRequest` struct at pkg/opencode/client.go:260-263 only has `Title` and `Directory` fields, no `Model` field.

**Source:** pkg/opencode/client.go:260-263

**Significance:** Headless spawns cannot specify model because the HTTP API request doesn't include a model parameter, even though the resolved model is available in cfg.Model.

---

### Finding 2: Inline and tmux modes already support model selection

**Evidence:** BuildSpawnCommand (client.go:128-142) accepts model parameter and adds `--model` flag to opencode CLI command. BuildOpencodeAttachCommand (tmux package) also accepts model parameter.

**Source:** pkg/opencode/client.go:128-142, pkg/tmux/tmux.go (BuildOpencodeAttachCommand)

**Significance:** The CLI-based spawn modes (inline, tmux) already have working model selection - only headless mode (HTTP API) is missing this capability.

---

### Finding 3: runSpawnHeadless has model available but doesn't use it

**Evidence:** runSpawnHeadless at cmd/orch/main.go:1115 has cfg.Model available (line 1143) and prints it (line 1165), but CreateSession call at line 1119 doesn't pass the model parameter.

**Source:** cmd/orch/main.go:1115-1175

**Significance:** The fix is straightforward - we just need to thread the model parameter through CreateSession API, the data is already available in the spawn config.

---

## Synthesis

**Key Insights:**

1. **Model selection inconsistency across spawn modes** - Inline and tmux modes support model selection via CLI flags, but headless mode (HTTP API) lacks this capability due to missing field in CreateSessionRequest struct.

2. **Simple fix with minimal API surface changes** - The model parameter just needs to be added to CreateSessionRequest struct and threaded through CreateSession function - no complex refactoring needed.

3. **Data already available** - All spawn modes already resolve the model via model.Resolve() and store it in cfg.Model, so headless mode just needs to pass it to the API.

**Answer to Investigation Question:**

To extend model selection to headless spawn mode, we need to: (1) Add a `Model` field to `CreateSessionRequest` struct, (2) Update `CreateSession` function signature to accept model parameter and include it in the HTTP request, (3) Update `runSpawnHeadless` to pass cfg.Model to CreateSession. This will achieve parity with inline and tmux modes.

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

**Add Model field to CreateSessionRequest and thread through CreateSession** - Extend the HTTP API request to include model parameter, matching the CLI-based spawn modes.

**Why this approach:**
- Minimal API surface change - just one new optional field
- Matches existing pattern in BuildSpawnCommand (uses optional --model flag)
- No breaking changes - model field is optional, backward compatible
- Achieves parity across all spawn modes (inline, tmux, headless)

**Trade-offs accepted:**
- OpenCode server must support model parameter in POST /session (assumption based on CLI support)
- No validation that model string is valid - relies on OpenCode server validation

**Implementation sequence:**
1. Add `Model string` field to CreateSessionRequest struct - enables API request
2. Update CreateSession function to accept model parameter - threads it through
3. Update runSpawnHeadless to pass cfg.Model to CreateSession - connects spawn config to API

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
- Add Model field to CreateSessionRequest struct (pkg/opencode/client.go:260-263)
- Update CreateSession function signature to accept model parameter
- Update runSpawnHeadless to pass cfg.Model to CreateSession

**Things to watch out for:**
- ⚠️ Model field should be optional (omitempty tag) for backward compatibility
- ⚠️ Empty model string should not be sent in request (OpenCode might error or use unexpected default)
- ⚠️ Need to verify OpenCode server actually accepts model parameter in POST /session

**Areas needing further investigation:**
- Whether OpenCode server supports model parameter in HTTP API (assumption based on CLI support)
- Default model behavior when model parameter is empty vs omitted
- Whether model validation happens server-side (likely yes, but untested)

**Success criteria:**
- ✅ Headless spawn with --model flag sets correct model in OpenCode session
- ✅ Headless spawn without --model flag uses default (opus) like other modes
- ✅ Model parameter appears in logged events for headless spawns
- ✅ No regressions in inline or tmux spawn modes

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
