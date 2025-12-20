# Test Fire-and-Forget Spawn Behavior

**TLDR:** Question: Does orch-go's tmux spawn work in fire-and-forget mode without blocking? Answer: Confirmed via timing test - spawn returns in 0.130 seconds while agent continues running in background tmux window. High confidence (95%) - validated with end-to-end timing and process verification.

**Date:** 2025-12-20
**Status:** Complete

## Question

Does `orch spawn` with tmux integration work in fire-and-forget mode (returns immediately without waiting for the agent to complete)?

## What I tried

1. Read existing knowledge: `kn context "fire-and-forget"`
   - Found: `kn-34d52f` - "orch-go tmux spawn is fire-and-forget - no session ID capture"
   - Reason: "opencode run --attach is TUI-based; --format json gives session ID but loses TUI"

2. Examined tmux spawn implementation in codebase:
   - `cmd/orch/main.go:251-318` - `runSpawnInTmux()` function
   - `pkg/tmux/tmux.go:78` - `BuildSpawnCommand()` - explicitly does NOT use `--format json`
   
3. Compared tmux vs inline spawn modes:
   - **Tmux mode**: Sends command to tmux window, returns immediately (no `cmd.Wait()`)
   - **Inline mode**: Uses `--format json`, pipes stdout, calls `cmd.Wait()` (blocking)

## What I observed

### Code Analysis

**Tmux spawn is fire-and-forget by design:**

From `cmd/orch/main.go:251-318`:
- Line 262-265: Creates tmux window
- Line 275-276: Builds opencode command WITHOUT `--format json`
- Line 279-286: Sends command to tmux window and presses Enter
- Line 317: Returns immediately (no waiting for process to finish)

**Key difference from inline spawn:**

Tmux mode (`runSpawnInTmux`):
```go
// Send the command to the tmux window
tmux.SendKeysLiteral(windowTarget, opencodeCmd)
tmux.SendEnter(windowTarget)
// Returns here - process continues in tmux window
return nil
```

Inline mode (`runSpawnInline`):
```go
cmd.Start()
result, err := opencode.ProcessOutput(stdout)  // Blocks reading
cmd.Wait()  // Blocks until completion
```

**Trade-off accepted:**
- Tmux mode: No session ID captured (can't use SSE immediately)
- Inline mode: Blocks orchestrator, but gets session ID for SSE monitoring
- Decision: Use title-matching via `orch status` for tmux-spawned agents

## Test performed

**Test:** Run `orch spawn investigation "quick test task"` and measure:
1. Time to return
2. Whether tmux window exists after return
3. Whether agent is running in background

**Expected result (if fire-and-forget):**
- Spawn command returns in <2 seconds
- Tmux window exists and shows opencode TUI running
- Agent continues running after spawn returns

**Actual result:**

```bash
$ time ./orch spawn investigation "test fire-and-forget timing" 2>&1
Spawned agent:
  Workspace:  og-inv-test-fire-forget-20dec
  Window:     workers-orch-go:11
  Beads ID:   open
  Context:    /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-test-fire-forget-20dec/SPAWN_CONTEXT.md
./orch spawn investigation "test fire-and-forget timing" 2>&1  0.05s user 0.05s system 72% cpu 0.130 total
```

**Verification:**

1. **Timing**: 0.130 seconds total - confirms immediate return
2. **Window exists**: 
   ```bash
   $ tmux list-windows -t workers-orch-go -F "#{window_index}:#{window_name}" | grep "^11:"
   11:🔬 og-inv-test-fire-forget-20dec [open]
   ```
3. **Agent running in background**:
   ```bash
   $ tmux capture-pane -t workers-orch-go:11 -p | head -10
   opencode run --attach http://127.0.0.1:4096 --title og-inv-test-fire-forget-20dec...
   |  Read     .orch/workspace/og-inv-test-fire-forget-20dec/SPAWN_CONTEXT.md
   ```
   Agent is actively running and has started reading its spawn context.

✅ **All expectations met** - Fire-and-forget behavior confirmed.

## Conclusion

**Answer:** Yes, `orch spawn` with tmux integration works in fire-and-forget mode.

**Evidence:**
- Spawn command returned in 0.130 seconds
- Tmux window (workers-orch-go:11) was created and exists
- Agent continues running in background (observed via `tmux capture-pane`)
- No blocking on agent completion

**Implementation pattern:**
- `runSpawnInTmux()` sends command to tmux window via `tmux send-keys`
- Returns immediately after sending Enter (no `cmd.Wait()`)
- Trade-off: No session ID captured (uses `--attach` for TUI, not `--format json`)
- Monitoring: Use `orch status` with title-matching instead of session ID

**Confidence:** High (95%) - Tested end-to-end with timing measurements and process verification.

## Self-Review

- [x] Real test performed (not code review) - Timed spawn execution, verified window creation and agent running
- [x] Conclusion from evidence (not speculation) - Based on timing measurements and tmux window verification
- [x] Question answered - Confirmed fire-and-forget behavior works as designed
- [x] File complete - All sections filled with concrete evidence

**Self-Review Status:** PASSED

