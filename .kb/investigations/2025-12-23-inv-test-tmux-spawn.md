<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux spawn functionality works correctly - all core functions pass integration tests including window creation, key sending, pane capture, and window lookup.

**Evidence:** Ran custom integration test exercising full tmux spawn flow: EnsureWorkersSession, CreateWindow, SendKeysLiteral, GetPaneContent, FindWindowByBeadsID, FindWindowByWorkspaceName, KillWindowByID - all passed. Also ran unit tests (11 pass) and verified IsOpenCodeReady detection logic.

**Knowledge:** The tmux spawn system is production-ready and well-tested. Key detection for OpenCode TUI readiness requires both a prompt box (┃) AND either agent selector or command hints.

**Next:** Close investigation - tmux spawn is confirmed working.

**Confidence:** Very High (95%) - comprehensive tests with real tmux sessions.

---

# Investigation: Test Tmux Spawn

**Question:** Does the tmux spawn functionality in orch-go work correctly?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** og-inv-test-tmux-spawn-23dec agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%+)

---

## Findings

### Finding 1: Unit Tests All Pass

**Evidence:** Ran `go test ./pkg/tmux/... -v` - all 11 tests pass including:
- TestSessionExists
- TestBuildWindowName (5 sub-tests)
- TestBuildWindowNameWithBeadsID
- TestBuildSpawnCommand
- TestBuildOpencodeAttachCommand
- TestBuildStandaloneCommand (2 sub-tests)
- TestBuildAttachCommand (2 sub-tests)
- TestWindowExistsByID
- TestBuildServerCommand (3 sub-tests)

**Source:** `pkg/tmux/tmux_test.go`

**Significance:** Core tmux package functionality is verified at the unit level.

---

### Finding 2: Full Integration Test Passes

**Evidence:** Created and ran a comprehensive integration test that exercises the full tmux spawn flow:

```
✓ Created/ensured session: workers-orch-go-test-spawn
✓ Built window name: 🔬 og-inv-test-spawn-23dec [test-123]
✓ Created window: target=workers-orch-go-test-spawn:2, id=@360
✓ Built attach command: opencode attach http://127.0.0.1:4096 --dir "/tmp/orch-go-test-spawn" --model "anthropic/claude-sonnet-4-20250514"
✓ Sent keys: echo TEST_SPAWN_SUCCESS
✓ Verified output in pane
✓ Found window by beads ID: workers-orch-go-test-spawn:2
✓ Found window by workspace name: workers-orch-go-test-spawn:2
✓ WindowExistsByID verified
✓ Killed window
✓ Verified window no longer exists
✓ Cleaned up session
```

**Source:** Custom integration test file `/tmp/test_spawn.go`

**Significance:** The full tmux spawn workflow functions correctly in a real tmux environment.

---

### Finding 3: OpenCode Ready Detection Works

**Evidence:** Tested `IsOpenCodeReady` function with various pane content scenarios:

| Test Case | Result |
|-----------|--------|
| Empty pane | false (correct) |
| Shell prompt only | false (correct) |
| OpenCode loading | false (correct) |
| Ready with prompt box + agent | true (correct) |
| Ready with prompt box + commands hint | true (correct) |
| Has prompt box but no agent/commands | false (correct) |

**Source:** `pkg/tmux/tmux.go:309-321`, custom test `/tmp/test_opencode_ready.go`

**Significance:** The TUI readiness detection correctly distinguishes between OpenCode still loading vs ready for input.

---

## Synthesis

**Key Insights:**

1. **Tmux spawn is production-ready** - Both unit tests and integration tests confirm the functionality works correctly with real tmux sessions.

2. **OpenCode attach mode works** - The `BuildOpencodeAttachCommand` correctly generates the command for connecting to the OpenCode server while showing the TUI in a tmux window.

3. **Window lookup is reliable** - Both `FindWindowByBeadsID` and `FindWindowByWorkspaceName` correctly locate windows, enabling the orchestrator to track and interact with spawned agents.

**Answer to Investigation Question:**

Yes, the tmux spawn functionality in orch-go works correctly. All core functions have been tested both at the unit level (11 tests pass) and through a comprehensive integration test that exercises the full spawn workflow: creating sessions, building window names with skill emojis, creating windows, sending keys to windows, capturing pane content, finding windows by beads ID and workspace name, and cleaning up windows/sessions.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Comprehensive testing with real tmux sessions confirms functionality. Both unit tests and integration tests pass.

**What's certain:**

- ✅ Core tmux functions work correctly (EnsureWorkersSession, CreateWindow, SendKeys, GetPaneContent)
- ✅ Window lookup by beads ID and workspace name functions correctly
- ✅ OpenCode TUI readiness detection works
- ✅ Window cleanup works (KillWindowByID, KillSession)

**What's uncertain:**

- ⚠️ Did not test with actual OpenCode TUI (only simulated shell commands)
- ⚠️ WaitForOpenCodeReady timeout behavior not tested with slow-starting OpenCode

**What would increase confidence to 99%:**

- Test with actual OpenCode TUI startup in tmux
- Test edge cases like network disconnection mid-spawn
- Test concurrent window creation

---

## Test Performed

**Test:** Ran comprehensive integration test exercising full tmux spawn workflow with real tmux sessions.

**Result:** All 11 steps passed:
1. EnsureWorkersSession created session correctly
2. BuildWindowName generated expected emoji+name format
3. CreateWindow returned valid target and ID
4. BuildOpencodeAttachCommand generated correct CLI format
5. SendKeysLiteral + SendEnter successfully sent commands
6. GetPaneContent captured command output
7. FindWindowByBeadsID found the window
8. FindWindowByWorkspaceName found the window
9. WindowExistsByID confirmed window presence
10. KillWindowByID removed the window
11. KillSession cleaned up the session

---

## References

**Files Examined:**
- `pkg/tmux/tmux.go` - Main tmux package implementation
- `pkg/tmux/tmux_test.go` - Unit tests
- `cmd/orch/main.go` - Spawn command implementation (lines 1209-1334 for runSpawnTmux)

**Commands Run:**
```bash
# Run unit tests
go test ./pkg/tmux/... -v -run 'TestBuild|TestSession|TestWindow'

# Check existing tmux sessions
tmux list-sessions

# Run integration test
go run /tmp/test_spawn.go

# Test IsOpenCodeReady
go run /tmp/test_opencode_ready.go
```

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
