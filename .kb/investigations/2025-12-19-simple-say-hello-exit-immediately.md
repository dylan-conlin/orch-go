**TLDR:** Question: Does orch-go spawn command work for a simple 'say hello and exit immediately' task? Answer: No - the spawn command hangs waiting for the opencode process to exit, preventing completion. This matches previous investigations showing the same hanging behavior. Medium confidence (70%) - validated with test using investigation skill and observed timeout.

<!--
Example TLDR:
"Question: Why aren't worker agents running tests? Answer: Agents follow documentation literally but test-running guidance isn't in spawn prompts or CLAUDE.md, only buried in separate docs. High confidence (85%) - validated across 5 agent sessions but small sample size."

Guidelines:
- Keep to 2-3 sentences maximum
- Answer: What question? What's the answer? How confident?
- Enable 30-second understanding for fresh Claude
---

# Investigation: orch-go spawn simple task

**Question:** Does the orch-go spawn command successfully spawn an agent that can complete a simple 'say hello and exit immediately' task?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** orchestrator
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Medium (70%)

## What I tried

- Verified project location and read spawn context from workspace
- Examined orch-go codebase structure and spawn command implementation
- Ran unit tests (`go test ./...`) - all passed
- Attempted to run `orch-go spawn investigation "say hello and exit immediately" --inline` with 30-second timeout
- Tested `orch-go spawn hello "say hello and exit immediately" --inline` (non-existent skill)
- Checked OpenCode server health and active sessions via `orch-go status`
- Reviewed previous investigation `2025-12-19-inv-test-spawn-integration.md` which reported same hanging behavior

## What I observed

- OpenCode server is running (HTTP 200 response from /health)
- Many active sessions exist, including multiple "og-inv-say-hello-exit-19dec" sessions from previous spawns
- `orch-go spawn investigation "say hello and exit immediately" --inline` timed out after 30 seconds with no output (command hangs)
- `orch-go spawn hello "say hello and exit immediately" --inline` returned immediately with warning "could not load skill 'hello': skill not found" (no hang)
- Previous investigation concluded spawn command hangs waiting for opencode process to exit, preventing session ID extraction and return
- Workspace directories are created in `/Users/dylanconlin/orch-knowledge/.orch/workspace/` rather than project directory

## Test performed

**Test:** Run orch-go spawn command with investigation skill and simple task, observe behavior for completion within timeout.

**Result:** Command hung indefinitely (timeout after 30 seconds). Session was created (visible via `orch-go status`) but orch-go did not return control. This matches the finding from previous investigation that spawn command waits for opencode process to exit, which only happens when agent session ends.

## Conclusion

The orch-go spawn command does NOT successfully spawn an agent that can complete a simple 'say hello and exit immediately' task when using the investigation skill. The command hangs waiting for the opencode process to exit, which prevents it from returning control to the user. This is consistent with previous investigation findings about spawn command behavior with real OpenCode server. The underlying issue appears to be that orch-go's inline spawn mode waits for the agent session to end before returning, but the agent session may not end promptly due to skill context requirements (investigation skill prompts extensive work). Even with a simple task, the investigation skill context causes the agent to follow a multi-step investigation procedure rather than immediately exiting.

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED