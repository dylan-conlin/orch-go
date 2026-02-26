<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless spawn HTTP API works correctly - the issue is an outdated orch-go binary still using tmux mode.

**Evidence:** Direct HTTP API test created session, sent prompt, and received agent response successfully; orch-go binary (Dec 20 17:08) predates commits e096aad and a9ecfa1 that implement headless mode.

**Knowledge:** The server-side prompt flow (CreateSession → SendPrompt → loop) functions correctly; sessions showing updated times confirm prompts reach the server; binary rebuild needed to use HTTP API instead of tmux.

**Next:** Rebuild orch-go binary from latest source code to enable headless mode by default.

**Confidence:** High (90%) - HTTP API test confirms functionality; binary age/commits confirm root cause.

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

# Investigation: Headless Spawn Not Sending Prompts

**Question:** Why do headless spawns create sessions with updated timestamps but investigation files remain template-only (prompts not being processed)?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** og-inv-headless-spawn-not-21dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%) - HTTP API tested, root cause identified

---

## Findings

### Finding 1: HTTP API integration exists and is being called

**Evidence:**

- `runSpawnHeadless` creates session via `/session` POST endpoint (main.go:679)
- Sends prompt via `/session/{sessionID}/prompt_async` POST (main.go:685)
- Payload structure: `{"parts": [{"type": "text", "text": prompt}], "agent": "build"}`

**Source:**

- cmd/orch/main.go:672-749 (runSpawnHeadless function)
- pkg/opencode/client.go:257-261 (SendPrompt function)
- pkg/opencode/client.go:150-171 (SendMessageAsync implementation)

**Significance:** The Go side is correctly calling the HTTP API endpoints. Problem must be on server-side processing or in how the prompt is being handled.

---

### Finding 2: Server endpoint exists and processes prompts

**Evidence:**

- Endpoint `/session/:sessionID/prompt_async` exists in OpenCode server (server.ts)
- Handler calls `SessionPrompt.prompt({ ...body, sessionID })`
- Prompt function creates user message, touches session, and calls `loop(sessionID)`

**Source:**

- /opencode/packages/opencode/src/server/server.ts (prompt_async endpoint)
- /opencode/packages/opencode/src/session/prompt.ts:199-211 (prompt function)
- /opencode/packages/opencode/src/session/prompt.ts:204 (Session.touch call)

**Significance:** Sessions show updated times because `Session.touch()` is called, confirming the prompt reaches the server. The agent loop should be starting.

---

### Finding 3: Agent loop should process messages but may be failing silently

**Evidence:**

- `loop(sessionID)` function filters messages, finds last user/assistant messages
- Exits loop if last assistant message has finish status and is newer than last user
- Creates assistant response and processes tool calls in a while loop

**Source:**

- /opencode/packages/opencode/src/session/prompt.ts:238-349 (loop function)
- /opencode/packages/opencode/src/session/prompt.ts:273-281 (exit condition)

**Significance:** If agents aren't responding, the loop may be exiting early or the assistant message generation is failing. Need to test actual spawn and check message state.

---

### Finding 4: HTTP API endpoints work correctly - binary is outdated

**Evidence:**

- Direct HTTP API test: Created session, sent prompt via /prompt_async, agent responded correctly
- Session ses_4c0016a59ffetMu6pci894Fy6Q received "Say hello and then exit" and responded appropriately
- Git commits show e096aad (remove tmux) and a9ecfa1 (headless default) exist in source
- Binary orch-go (Dec 20 17:08) predates these commits
- Running orch-go spawn still creates tmux windows ("Window: workers-orch-go:3")

**Source:**

- Test commands: curl POST /session, curl POST /session/{id}/prompt_async, curl GET /session/{id}/message
- git log cmd/orch/main.go showing recent refactors
- File modification times: ~/bin/orch-go vs commit dates

**Significance:** The HTTP API implementation is correct and functional. The problem is NOT with the server-side code - it's that the deployed binary hasn't been rebuilt with the latest changes that switch from tmux to HTTP API spawning.

---

## Synthesis

**Key Insights:**

1. **Server-side implementation is correct** - The OpenCode server properly handles /session and /prompt_async endpoints, creates user messages, touches sessions (explaining updated timestamps), and starts the agent loop. Direct HTTP API testing confirms end-to-end functionality.

2. **Binary-code mismatch** - The orch-go binary in ~/bin/ was compiled on Dec 20 17:08, before commits e096aad (remove tmux) and a9ecfa1 (headless default) were merged. The binary still uses tmux-based spawning, which is why agents appear in tmux windows instead of running headlessly.

3. **Investigation files remaining template-only is a symptom** - If the binary were actually calling runSpawnHeadless, sessions would be created via HTTP API and prompts would be sent. But since the old binary uses tmux mode, it's creating windows rather than HTTP API sessions, leading to different behavior.

**Answer to Investigation Question:**

Headless spawns are NOT failing to send prompts - the HTTP API works correctly when tested directly. The issue is that the deployed `orch-go` binary predates the code changes that implement headless mode, so it's still using tmux-based spawning instead of HTTP API spawning. Sessions show updated times because Session.touch() is called regardless of spawn mode. The solution is to rebuild the orch-go binary from the latest source code.

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

**Rebuild orch-go binary from latest source** - Compile from current main branch to enable headless HTTP API spawning.

**Why this approach:**

- Directly addresses root cause (outdated binary)
- No code changes needed - implementation already exists in source
- Restores intended behavior from commits e096aad and a9ecfa1

**Trade-offs accepted:**

- None - this is a straightforward rebuild

**Implementation sequence:**

1. Navigate to orch-go repository
2. Run build command: `go build -o ~/bin/orch-go cmd/orch/*.go`
3. Verify with test spawn: `orch-go spawn investigation "test" --no-track`
4. Confirm output shows Session ID instead of Window ID

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
