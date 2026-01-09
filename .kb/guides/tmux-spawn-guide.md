# Tmux Spawn Guide

**Purpose:** Authoritative reference for tmux-based agent spawning in orch-go.

**Synthesized from:** 11 investigations (Dec 20-23, 2025) and 6 follow-up investigations (Jan 2026) covering architecture, concurrent spawning, session resolution, attach mode, fallback mechanisms, orchestrator session types, and debugging.

---

## Quick Reference

```bash
# Spawn with tmux (interactive TUI in tmux window)
orch spawn --tmux investigation "task"

# Spawn headless (default - no TUI, HTTP API only)
orch spawn investigation "task"

# Spawn inline (blocking TUI in current terminal)
orch spawn --inline investigation "task"
```

---

## Architecture Overview

### Three Spawn Modes

| Mode | Flag | TUI | API Access | Use Case |
|------|------|-----|------------|----------|
| **Headless** | (default) | No | Yes (SSE) | Daemon, automation, batch processing |
| **Tmux** | `--tmux` | Yes (in window) | Yes (attach mode) | Interactive monitoring |
| **Inline** | `--inline` | Yes (blocking) | No | Debugging, quick tasks |

**Key decision:** Headless is the default because it's optimal for automation. Tmux is opt-in for when you want visual monitoring.

**Orchestrator-type skills exception:** Skills with `skill-type: policy` or `skill-type: orchestrator` default to tmux mode for interactive visibility. This is controlled by `cfg.IsOrchestrator` in spawn logic (spawn_cmd.go:789).

### How Tmux Mode Works

1. **Session Management:** `tmux.EnsureWorkersSession()` creates/verifies `workers-{project}` session
2. **Window Creation:** `tmux.CreateWindow()` creates window with auto-incrementing index
3. **Attach Command:** Uses `opencode attach {server_url}` (not standalone) for dual TUI/API access
4. **TUI Detection:** `WaitForOpenCodeReady()` polls for prompt box + agent selector
5. **Prompt Sending:** `SendPromptAfterReady()` types the prompt once TUI is ready
6. **Registration:** Agent registered with window ID (e.g., `@360`) for tracking

### Session Types

| Session | Constant | Purpose |
|---------|----------|---------|
| `workers-{project}` | `workers-orch-go`, etc. | Worker agents for specific project |
| `orchestrator` | `OrchestratorSessionName` | Regular orchestrators |
| `meta-orchestrator` | `MetaOrchestratorSessionName` | Meta-orchestrators (cross-project) |

**Routing logic:** Spawn code detects skill type and routes to appropriate session. Meta-orchestrators use `EnsureMetaOrchestratorSession()`, regular orchestrators use `EnsureOrchestratorSession()`.

**Why separated:** Viewing `tmux ls` immediately shows the hierarchy. No need to read workspace names to distinguish orchestrator types.

### Window Naming Convention

```
🔬 og-inv-{task-name}-{date} [beads-id]
```

- Skill emoji prefix (🔬 = investigation, ⚙️ = feature-impl, etc.)
- Workspace name
- Beads ID in brackets (for lookup)

---

## Session ID Resolution

**Problem:** Session ID capture from tmux spawns is unreliable. Only ~1% of workspaces have `.session_id` files.

**Solution:** Fallback chain in `resolveSessionID()`:

```
1. If starts with "ses_" → validate format, verify via GetSession
2. Search workspace files for matching .session_id
3. Search OpenCode API sessions by title match
4. Search tmux windows by beads ID in window name
```

**Key insight:** Never assume session ID is available. Always use the fallback chain.

---

## Concurrent Spawning

**Validated capacity:** 6+ concurrent agents (alpha/beta/gamma/delta/epsilon/zeta tests)

**Observed maximum:** 18+ concurrent sessions, 53 workspaces, 29 processes

**Why it works:**
- Fire-and-forget pattern (no blocking on spawn confirmation)
- Workspace isolation via unique naming (task name + date)
- Each agent gets dedicated tmux window with auto-incrementing index

**Key knowledge entry:** `kn-34d52f` - "orch-go tmux spawn is fire-and-forget - no session ID capture"

---

## Fallback Mechanisms

### Status Command

`orch status` uses tmux as source of truth for active agents:
1. Lists tmux `workers-*` sessions and windows
2. Matches windows with OpenCode sessions by title
3. Enriches with registry metadata (Beads ID, Skill)
4. Shows agents even if missing from API/registry

### Tail Command

`orch tail` fallback chain:
1. Try OpenCode API (if session ID available)
2. Try tmux window lookup by beads ID
3. Fall back to `tmux capture-pane` for output

### Send/Question Commands

Same fallback chain as tail:
1. Resolve identifier via fallback chain
2. Send via OpenCode API if session found
3. Send via `tmux send-keys` if only window found

---

## TUI Readiness Detection

**Function:** `IsOpenCodeReady(paneContent string) bool`

**Detection criteria:**
- Must have prompt box (`┃`)
- AND must have either agent selector OR command hints

**Test results:**
| Pane Content | Result |
|--------------|--------|
| Empty | false |
| Shell prompt only | false |
| OpenCode loading | false |
| Ready with prompt box + agent | true |
| Ready with prompt box + commands hint | true |
| Has prompt box but no agent/commands | false |

**Timing:** `WaitForOpenCodeReady()` polls with configurable timeout (default 30s)

---

## Troubleshooting

### Exit Code 137 (SIGKILL)

**Two known causes:**

1. **Stale binary** - `~/bin/orch` doesn't have latest code
   - Fix: Run `make install`
   - Prevention: Git post-commit hook auto-rebuilds

2. **launchd KeepAlive conflict** - Daemon using same binary path
   - Fix: Use different path for daemon (`build/orch` vs `~/bin/orch`)
   - Check: `launchctl list | grep orch`

**Diagnosis:**
```bash
# Compare binaries
md5 ~/bin/orch build/orch

# Check version info
orch version --source
```

### Session ID Not Found

**Symptom:** `orch send` or `orch tail` fails with "no session ID"

**Causes:**
1. Session ID capture failed during spawn (timing issue)
2. Agent completed and session closed
3. Wrong identifier format

**Solutions:**
1. Use beads ID instead of session ID: `orch send orch-go-abc123 "message"`
2. Use workspace name: `orch send og-inv-task-name-06jan "message"`
3. Check if session still active: `orch status`

### Message Not Arriving

**Symptom:** `orch send` reports success but agent doesn't respond

**Causes:**
1. OpenCode API returns 204 for non-existent sessions (false positive)
2. Session ID truncated (e.g., just `ses_`)

**Solutions:**
1. Use full session ID with validation
2. Let fallback chain find the session
3. Check session exists first: `orch status --json | jq`

### Dashboard Showing Wrong Project

**Symptom:** Dashboard shows beads/agents from wrong project

**Cause:** `GetTmuxCwd()` was returning first window's cwd instead of active window's cwd.

**Fix (2026-01-08):** Two-step approach in `pkg/tmux/follower.go`:
1. Get active window index: `tmux display-message -t session -p "#{window_index}"`
2. Get that window's cwd: `tmux display-message -t session:index -p "#{pane_current_path}"`

**Verification:** `go test ./pkg/tmux/... -run TestGetTmuxCwd`

---

## Best Practices

1. **Use headless for automation** - Tmux is for interactive monitoring only
2. **Never assume session ID exists** - Always use fallback chain
3. **Use beads ID for commands** - More reliable than session ID
4. **Rebuild after code changes** - Stale binaries cause mysterious failures
5. **Separate daemon binary path** - Avoid KeepAlive conflicts

---

## Related Files

- `pkg/tmux/tmux.go` - Core tmux functions
- `pkg/tmux/tmux_test.go` - Unit tests
- `cmd/orch/main.go:runSpawnTmux()` - Spawn implementation
- `cmd/orch/main.go:resolveSessionID()` - Session ID fallback chain

---

## References

**Superseded investigations (archived):**
- archived/2025-12-20-inv-migrate-orch-go-tmux-http.md (architecture decision)
- archived/2025-12-20-inv-tmux-concurrent-{delta,epsilon,zeta}.md (concurrent testing)
- archived/2025-12-21-debug-orch-send-fails-silently-tmux.md (session resolution fix)
- archived/2025-12-21-inv-add-tmux-fallback-orch-status.md (fallback mechanisms)
- archived/2025-12-21-inv-add-tmux-flag-orch-spawn.md (--tmux flag implementation)
- archived/2025-12-21-inv-implement-attach-mode-tmux-spawn.md (attach mode)
- archived/2025-12-21-inv-tmux-spawn-killed.md (SIGKILL debugging)
- archived/2025-12-22-debug-orch-send-fails-silently-tmux.md (session validation)
- archived/2025-12-23-inv-test-tmux-spawn.md (integration testing)
- archived/2026-01-06-inv-tmux-session-naming-confusing-hard.md (meta-orchestrator session separation)
- archived/2026-01-06-inv-synthesize-tmux-investigations-11-synthesis.md (created this guide)
- archived/2026-01-08-inv-synthesize-tmux-investigations-12-synthesis.md (triage for archival)

**Related investigations (not archived):**
- 2026-01-04-inv-test-spawnable-orchestrator-tmux-default.md (orchestrator tmux default)
- 2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md (attach mode --dir fix)
- 2026-01-08-inv-fix-gettmuxcwd-active-window-instead.md (GetTmuxCwd fix)

**Knowledge entries:**
- kn-34d52f - "orch-go tmux spawn is fire-and-forget - no session ID capture"
