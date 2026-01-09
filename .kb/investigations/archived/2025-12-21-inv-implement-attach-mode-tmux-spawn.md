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

# Investigation: Implement OpenCode Attach Mode for Tmux Spawn

**Question:** How to implement OpenCode "attach" mode for `orch spawn --tmux` to enable dual TUI and API access?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Current Tmux Spawn Behavior
**Evidence:** `runSpawnTmux` in `cmd/orch/main.go` used `tmux.BuildStandaloneCommand` which runs `opencode {project_dir} --model {model}`. This starts an ephemeral server, making the session invisible to the main OpenCode server.

**Source:** `cmd/orch/main.go` and `pkg/tmux/tmux.go`

**Significance:** Tmux spawns were "human-friendly" (TUI visible) but NOT "AI-friendly" (no API access).

---

### Finding 2: OpenCode Attach Mode
**Evidence:** OpenCode supports an `attach` command: `opencode attach {server_url} --dir {project_dir} --model {model}`. This connects the TUI to a shared server, making sessions visible via API.

**Source:** Issue `orch-go-559o` description.

**Significance:** Using `attach` mode allows `orch-go` to capture the `session_id` and interact with the agent via HTTP API while still providing the interactive TUI in tmux.

---

### Finding 3: Session ID Capture
**Evidence:** Sessions can be discovered via the `/session` API by filtering with the `x-opencode-directory` header. The most recent session for the project directory is the one just spawned.

**Source:** `pkg/opencode/client.go` implementation and tests.

**Significance:** We can automatically capture the `session_id` after spawning in tmux and store it in the registry.

---

## Synthesis

**Key Insights:**

1. **Dual Access** - By switching to `opencode attach`, we achieve both interactive TUI access (via tmux) and programmatic API access (via `session_id`).

2. **Registry Integration** - Capturing the `session_id` allows subsequent commands like `orch tail`, `orch send`, and `orch question` to use the HTTP API instead of falling back to tmux scraping.

**Answer to Investigation Question:**
OpenCode attach mode was implemented by adding `OpencodeAttachConfig` and `BuildOpencodeAttachCommand` to `pkg/tmux`. `runSpawnTmux` was updated to use this command and then call `client.FindRecentSession` to capture the `session_id`, which is then stored in the agent registry.

---

## Implementation Recommendations

### Recommended Approach ⭐
**Use opencode attach and capture session ID** - This approach was implemented.

**Why this approach:**
- Provides full API access for tmux-spawned agents.
- Maintains the interactive TUI experience.
- Enables better monitoring and Q&A.

**Implementation sequence:**
1. Added `OpencodeAttachConfig` and `BuildOpencodeAttachCommand` to `pkg/tmux/tmux.go`.
2. Added `FindRecentSession` to `pkg/opencode/client.go`.
3. Updated `runSpawnTmux` in `cmd/orch/main.go` to use the new command and capture the session ID.
4. Updated the registry and event logging to include the session ID.

---

## Summary (D.E.K.N.)

**Delta:** Implemented OpenCode `attach` mode for tmux spawns and automated `session_id` capture.

**Evidence:** Unit tests for command building and session discovery passed; build succeeded.

**Knowledge:** `opencode attach` enables dual TUI/API access by connecting to a shared server; `x-opencode-directory` header is required for session discovery.

**Next:** None - implementation complete.

**Confidence:** Very High (95%) - verified with unit tests and code review.



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
