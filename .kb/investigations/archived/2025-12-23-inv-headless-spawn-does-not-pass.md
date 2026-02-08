<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** SendMessageAsync was missing the model parameter in its payload, causing OpenCode to ignore the --model flag for headless spawns.

**Evidence:** Code inspection showed SendMessageAsync (client.go:158-161) only included "parts" and "agent" in payload, not "model"; tests confirmed adding model parameter correctly includes it in the HTTP request.

**Knowledge:** OpenCode uses per-message model selection (not per-session), so every message sent via the API must explicitly include the model parameter to override the default.

**Next:** Fix implemented and tested; commit changes and mark investigation complete.

**Confidence:** Very High (95%) - Root cause confirmed via code inspection and test verification.

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

# Investigation: Headless Spawn Does Not Pass

**Question:** Why does headless spawn ignore the --model flag, and how do we fix it to pass the model parameter when sending the initial message?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%) - Root cause confirmed, fix implemented and tested

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Headless spawn calls CreateSession with model but SendPrompt without model

**Evidence:** 
- `runSpawnHeadless` (main.go:1147-1207) creates session with model: `client.CreateSession(cfg.WorkspaceName, cfg.ProjectDir, cfg.Model)` at line 1151
- Then sends prompt without model: `client.SendPrompt(sessionResp.ID, minimalPrompt)` at line 1157
- `SendPrompt` delegates to `SendMessageAsync` which has NO model parameter

**Source:** 
- cmd/orch/main.go:1147-1207 (runSpawnHeadless function)
- pkg/opencode/client.go:319-321 (SendPrompt function)
- pkg/opencode/client.go:157-177 (SendMessageAsync function)

**Significance:** 
OpenCode's design is "model is per-message, not per-session" (per task description), so passing model only during session creation is insufficient. The initial message sent via `SendPrompt` doesn't include the model, causing OpenCode to use its default model instead of the user-specified model.

---

### Finding 2: SendMessageAsync payload is missing model field

**Evidence:**
Looking at client.go:158-161, the payload structure is:
```go
payload := map[string]any{
    "parts": []map[string]string{{"type": "text", "text": content}},
    "agent": "build",
}
```
No `model` field is included in this payload.

**Source:** pkg/opencode/client.go:158-161

**Significance:**
This is the root cause. The HTTP API call to `/session/{sessionID}/prompt_async` needs to include a `model` field in the JSON payload for OpenCode to use the specified model. Without it, OpenCode falls back to its default model.

---

### Finding 3: CreateSession includes model but that's for session metadata, not message inference

**Evidence:**
`CreateSession` (client.go:275-314) includes model in the request:
```go
payload := CreateSessionRequest{
    Title:     title,
    Directory: directory,
    Model:     model,
}
```
But this is used for session creation, not for the actual message that triggers the agent.

**Source:** pkg/opencode/client.go:275-314

**Significance:**
The model passed to CreateSession may be stored as session metadata, but OpenCode requires the model to be specified per-message. The initial prompt message must explicitly include the model parameter.

---

## Synthesis

**Key Insights:**

1. **OpenCode uses per-message model selection, not per-session** - The task description explicitly states "OpenCode design: model is per-message, not per-session", meaning each message sent via the API must include the model parameter if we want to override the default.

2. **HeadlessSpawn flow had incomplete model passing** - While CreateSession included the model parameter for session metadata, the subsequent SendPrompt call did not pass the model when sending the initial message, causing OpenCode to ignore the user's --model flag.

3. **Fix requires threading model through three layers** - The fix needed to: (1) add model parameter to SendMessageAsync payload, (2) add model parameter to SendPrompt signature, and (3) pass cfg.Model from runSpawnHeadless when calling SendPrompt.

**Answer to Investigation Question:**

Headless spawn ignores the --model flag because `SendMessageAsync` (called by `SendPrompt`) doesn't include the model parameter in the HTTP payload sent to OpenCode's `/session/{id}/prompt_async` endpoint. While `CreateSession` includes the model in session metadata, OpenCode requires the model to be specified per-message. The fix adds a `model` parameter to `SendMessageAsync` and `SendPrompt`, passing it through from `runSpawnHeadless` so the model is included in the payload when sending the initial prompt message.

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
