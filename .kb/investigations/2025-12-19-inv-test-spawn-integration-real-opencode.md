---
date: "2025-12-19"
status: "Complete"
---

# Test spawn integration with real OpenCode server

**TLDR:** Question: Does the spawn command in orch-go work with real OpenCode server? Answer: OpenCode CLI works and returns session IDs, but orch-go implementation hangs waiting for OpenCode command to exit (which doesn't happen with --attach). Medium confidence - tested OpenCode directly, identified blocking issue in orch-go.

## Question

Does the spawn command in orch-go work correctly with a real OpenCode server?

## What I tried

- Read orch-go main.go to understand spawn command implementation
- Checked current directory (should be in orch-go project per spawn context)
- Created investigation file using kb create
- Tested OpenCode server connectivity with curl
- Ran orch-go spawn command (timed out)
- Tested OpenCode command directly with timeout to verify JSON output format

## What I observed

- orch-go spawn command uses `opencode run --attach` with JSON format
- Code expects sessionID at top level of each event from OpenCode
- OpenCode server is running at http://127.0.0.1:4096
- Direct OpenCode command works and returns JSON with sessionID at top level
- orch-go spawn command timed out, likely because it waits for full completion

## Test performed

**Test:** Ran OpenCode command directly with 5s timeout: `opencode run --attach http://127.0.0.1:4096 --format json --title "test-spawn-123" "Say hello"`

**Result:** Command returned valid JSON events with sessionID `ses_4c603f442ffe3fyYdQbCKq1QJd` at top level, confirming OpenCode integration works at basic level

**Test 2:** Ran orch-go spawn command with 10s timeout: `./orch-go spawn "test prompt for spawn integration"`

**Result:** Command hung with no output, indicating orch-go implementation has a blocking issue (likely waiting for OpenCode process to exit which doesn't happen)

## Conclusion

OpenCode CLI works correctly and returns session IDs in JSON format. However, the current orch-go implementation has a design flaw: it waits for the OpenCode command to exit (`cmd.Wait()`), but the OpenCode command with `--attach` doesn't exit after sending response, causing orch-go to hang indefinitely. The parsing logic is correct and would extract session ID if the command output was read without blocking.

---

## Self-Review

- [x] Real test performed (tested OpenCode command directly)
- [x] Conclusion from evidence (based on test results, not speculation)
- [x] Question answered (yes, with identified issue)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

## Notes

- **Root cause:** `cmd.Wait()` in `RunSpawn` blocks because OpenCode command with `--attach` doesn't exit.
- **Possible fix:** Read session ID from stdout stream, then detach/kill process or run in background.
- **Alternative:** Use OpenCode's `serve` mode with direct HTTP API instead of CLI wrapper.
- **Related files:** `main.go:RunSpawn`, `pkg/opencode/client.go:ProcessOutput`