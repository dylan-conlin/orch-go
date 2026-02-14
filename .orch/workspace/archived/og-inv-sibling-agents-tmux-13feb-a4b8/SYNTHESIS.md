# Synthesis: Sibling Agents Tmux Die (WRONG TASK - ABORTED)

**Note:** This investigation was spawned for the wrong task. The real task was investigating headless OpenCode spawns dying.

## What Was Learned (Still Valuable)

### 1. orch complete is safe for sibling windows
Every step in `runComplete()` was traced and tested. The only tmux kill targets the specific agent by `[beadsID]` pattern. Individual and combined tests confirmed siblings survive.

### 2. cleanPhantomWindows bug (P2)
`orch clean --phantoms` (clean_cmd.go:650-745) kills tmux windows whose beads ID has no OpenCode session. Claude-mode agents (the DEFAULT backend) don't use OpenCode sessions. If someone runs `orch clean --phantoms`, ALL Claude-mode agent windows get killed as false positives.

**Fix needed:** `cleanPhantomWindows` should also check the agent registry for Claude-mode agents before classifying a window as phantom.

### 3. Stale com.orch.reap launchd agent
`~/Library/LaunchAgents/com.orch.reap.plist` tries to run `orch reap` every 5 minutes, but the command was removed. Should be unloaded: `launchctl unload ~/Library/LaunchAgents/com.orch.reap.plist`.

### 4. Architecture: 4-layer state fragmentation
Agent state lives in OpenCode sessions, tmux windows, beads issues, and the agent registry. Cleanup functions that check only one layer produce false positives. This is a systemic issue — any new cleanup code must cross-reference all layers.

## Discovered Issues
- `cleanPhantomWindows` Claude-mode false positive (P2 bug)
- Stale `com.orch.reap` launchd agent (cleanup)
