## Summary (D.E.K.N.)

**Delta:** The spawn agent with tmux flow works through three spawn modes: default tmux (visual window), inline (blocking TUI), and headless (HTTP API) - each using different mechanisms to start and track agents.

**Evidence:** Code analysis of cmd/orch/main.go:1075-1200 (runSpawnTmux), pkg/tmux/tmux.go, and live tmux session showing 25 active agent windows in workers-orch-go.

**Knowledge:** Tmux spawn creates visual windows in workers-{project} sessions, uses opencode attach mode to connect to shared server, and waits for TUI ready before sending prompt via send-keys.

**Next:** None - investigation complete. The spawn flow is well-documented and working as designed.

**Confidence:** Very High (95%) - verified against live tmux sessions AND tested actual spawn execution (4.98s, window @623 created).

---

# Investigation: Spawn Agent with Tmux Flow

**Question:** How does `orch spawn` work with tmux to create and manage agent windows?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-inv-spawn-agent-tmux-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Three Spawn Modes with Different Mechanisms

**Evidence:** The spawn command (`cmd/orch/main.go:848-942`) supports three modes:
1. **Tmux (default)** - Creates window in workers-{project} session, runs `opencode attach`
2. **Inline (--inline)** - Runs `opencode run` in current terminal, blocking with TUI
3. **Headless (--headless)** - Uses HTTP API to create session, no TUI

**Source:** `cmd/orch/main.go:929-941`
```go
if inline {
    return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}
if headless {
    return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}
// Default: Tmux mode
return runSpawnTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task, attach)
```

**Significance:** Each mode has distinct tradeoffs - tmux provides visual monitoring + interruptibility, inline is blocking but shows TUI, headless is fire-and-forget for automation.

---

### Finding 2: Tmux Spawn Creates Workers Sessions Per-Project

**Evidence:** The `runSpawnTmux` function (`cmd/orch/main.go:1075-1200`) follows this flow:
1. `EnsureWorkersSession(project, projectDir)` - Creates/verifies `workers-{project}` session
2. `BuildWindowName(workspaceName, skillName, beadsID)` - Creates window name with emoji
3. `CreateWindow(sessionName, windowName, workDir)` - Creates detached window
4. `SendKeys` + `SendEnter` - Sends opencode attach command
5. `WaitForOpenCodeReady` - Polls pane content for TUI indicators
6. `SendKeysLiteral` + `SendEnter` - Sends the prompt

**Source:** 
- `pkg/tmux/tmux.go:197-235` - `EnsureWorkersSession`
- `pkg/tmux/tmux.go:238-266` - `CreateWindow`
- `pkg/tmux/tmux.go:309-343` - `WaitForOpenCodeReady`

**Significance:** The per-project session model (`workers-orch-go`, `workers-beads`, etc.) keeps agent windows organized and enables easy navigation between projects.

---

### Finding 3: OpenCode Attach Mode Enables API + TUI Dual Access

**Evidence:** Tmux spawn uses "attach mode" (`opencode attach {server_url}`) rather than standalone mode:
```go
// pkg/tmux/tmux.go:92-106
func BuildOpencodeAttachCommand(cfg *OpencodeAttachConfig) string {
    cmd := fmt.Sprintf("%s attach %s --dir %q", opencodeBin, cfg.ServerURL, cfg.ProjectDir)
    if cfg.Model != "" {
        cmd += fmt.Sprintf(" --model %q", cfg.Model)
    }
    if cfg.SessionID != "" {
        cmd += fmt.Sprintf(" --session %q", cfg.SessionID)
    }
    return cmd
}
```

**Source:** `pkg/tmux/tmux.go:80-106`, `cmd/orch/main.go:1093-1098`

**Significance:** Attach mode connects to the shared OpenCode server (default http://127.0.0.1:4096), which means:
- Sessions are visible via `orch status` (API access)
- SSE events stream for monitoring
- TUI is still displayed for visual interaction

---

### Finding 4: TUI Readiness Detection Uses Pane Content Polling

**Evidence:** The system waits for OpenCode TUI to be ready by polling tmux pane content:
```go
// pkg/tmux/tmux.go:309-321
func IsOpenCodeReady(content string) bool {
    contentLower := strings.ToLower(content)
    hasPromptBox := strings.Contains(content, "┃")  // Thick vertical bar
    hasAgentSelector := strings.Contains(contentLower, "build") || strings.Contains(contentLower, "agent")
    hasCommandHint := strings.Contains(contentLower, "alt+x") || strings.Contains(contentLower, "commands")
    return hasPromptBox && (hasAgentSelector || hasCommandHint)
}
```

Default config: 15s timeout, 200ms poll interval, 1s post-ready delay before typing.

**Source:** `pkg/tmux/tmux.go:109-134`, `pkg/tmux/tmux.go:325-343`

**Significance:** This prevents sending the prompt before the TUI is ready to receive input, avoiding race conditions.

---

### Finding 5: Window Names Include Emoji, Workspace, and Beads ID

**Evidence:** Window names follow format: `{emoji} {workspace-name} [{beads-id}]`

```go
// pkg/tmux/tmux.go:12-20
var SKILL_EMOJIS = map[string]string{
    "investigation":        "🔬",
    "feature-impl":         "🏗️",
    "systematic-debugging": "🐛",
    "architect":            "📐",
    // ...
}
```

Example: `🔬 og-inv-spawn-agent-tmux-22dec [orch-go-untracked-1766417772]`

**Source:** `pkg/tmux/tmux.go:164-180`, confirmed by `tmux list-windows -t workers-orch-go`

**Significance:** Makes it easy to identify agents at a glance - skill type via emoji, task via workspace name, tracking via beads ID.

---

## Synthesis

**Key Insights:**

1. **Separation of concerns** - The spawn system cleanly separates workspace creation (`spawn/config.go`, `spawn/context.go`), tmux management (`pkg/tmux/tmux.go`), and spawn orchestration (`cmd/orch/main.go`).

2. **Attach mode is the key innovation** - By using `opencode attach` rather than standalone mode, agents get both visual TUI and API accessibility. This enables `orch status`, `orch send`, and SSE monitoring to work with tmux-spawned agents.

3. **Reliable prompt delivery** - The TUI readiness detection + post-ready delay ensures prompts are sent when OpenCode is actually ready to receive them, preventing lost input.

**Answer to Investigation Question:**

`orch spawn` with tmux works by:
1. Creating/verifying a `workers-{project}` tmux session
2. Creating a new detached window with skill emoji + workspace name
3. Running `opencode attach {server}` in that window to connect to shared OpenCode server
4. Polling the pane until TUI indicators appear (prompt box + agent selector)
5. Waiting 1s for input focus, then sending the prompt via `send-keys -l`
6. Capturing session ID from API for later lookups

The flow is designed for visual monitoring while maintaining API accessibility for automation.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Code analysis is comprehensive and verified against live tmux sessions. The system is working as designed in production.

**What's certain:**

- ✅ Three spawn modes exist: tmux (default), inline, headless
- ✅ Tmux spawn uses `opencode attach` for dual TUI+API access
- ✅ TUI readiness is detected via pane content polling
- ✅ 25 agent windows are currently active in workers-orch-go session

**What's uncertain:**

- ⚠️ Error handling paths not explored in detail
- ⚠️ Tmuxinator integration not fully traced

**What would increase confidence to Very High (95%+):**

- Test error cases (no tmux, no opencode, timeout)
- Verify session ID capture works reliably across more agents

**Update 2025-12-22 15:39:** Executed actual spawn test with:
```bash
orch spawn investigation "test tmux spawn" --no-track --skip-artifact-check
```
Result: Completed in 4.98s, created window workers-orch-go:26 with ID @623, workspace og-inv-test-tmux-spawn-22dec created with SPAWN_CONTEXT.md, agent immediately began reading context and executing. This confirms spawn execution works correctly.

---

## Implementation Recommendations

**Purpose:** N/A - this was an exploratory investigation, not a problem-solving investigation.

### Current State Assessment

The spawn-with-tmux system is well-designed and working. Key design decisions:

1. **Per-project sessions** - Good isolation, easy navigation
2. **Attach mode** - Enables monitoring without losing TUI
3. **Emoji + beads ID in names** - Good discoverability

### Potential Improvements (Not Urgent)

1. **Session ID capture reliability** - Currently silently ignores errors (`sessionID, _ := client.FindRecentSessionWithRetry(...)`). Could add retry logging or metrics.

2. **TUI readiness detection** - Current heuristics (looking for "┃", "build", "agent") could become fragile if OpenCode TUI changes. Consider versioned detection patterns.

3. **Timeout handling** - The 15s timeout is hardcoded. Could be configurable for slow machines.

---

## References

**Files Examined:**
- `cmd/orch/main.go:160-220` - Spawn command flags and definition
- `cmd/orch/main.go:848-1200` - Spawn execution (all three modes)
- `pkg/tmux/tmux.go` - Full file (571 lines) - Tmux session/window management
- `pkg/spawn/config.go` - Spawn configuration
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md generation
- `pkg/spawn/session.go` - Session ID file management

**Commands Run:**
```bash
# List tmux sessions
tmux list-sessions
# Output: 11 sessions including workers-orch-go with 25 windows

# List workers-orch-go windows
tmux list-windows -t workers-orch-go -F "#{window_index} #{window_name}"
# Output: 25 windows with emoji prefixes and beads IDs

# Check opencode availability
which opencode && opencode --version
# Output: /Users/dylanconlin/claude-npm-global/bin/opencode, version 1.0.182
```

**Related Artifacts:**
- **Investigation:** This is an exploratory investigation of the spawn system

---

## Investigation History

**2025-12-22 15:00:** Investigation started
- Initial question: How does orch spawn work with tmux?
- Context: Spawned via orchestrator to understand spawn flow

**2025-12-22 15:30:** Code analysis complete
- Traced spawn flow through main.go → tmux.go → context.go
- Verified against live tmux sessions (25 windows in workers-orch-go)

**2025-12-22 15:45:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Spawn flow is well-designed with three modes (tmux/inline/headless) and uses opencode attach for dual TUI+API access

---

## Self-Review

- [x] Real test performed (verified against live tmux sessions)
- [x] Conclusion from evidence (based on code + live state)
- [x] Question answered (spawn flow fully traced)
- [x] File complete

**Self-Review Status:** PASSED
