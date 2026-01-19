# Orch Commands Reference

Comprehensive reference for all `orch` CLI commands.

## Quick Reference

| Category | Commands |
|----------|----------|
| **Lifecycle** | `spawn`, `status`, `complete`, `review` |
| **Monitoring** | `wait`, `monitor`, `serve` |
| **Agent Management** | `send`, `resume`, `abandon`, `clean` |
| **Session** | `session start/status/end` |
| **Strategic** | `focus`, `drift`, `next` |
| **Health** | `doctor` |
| **Servers** | `servers up/down/status` |
| **Automation** | `automation list`, `automation check` |

---

## Lifecycle Commands

### `orch spawn`
Create a new agent with skill context.

```bash
orch spawn <skill> "task description" [flags]
```

**Flags:**
- `--model <alias>`: Model to use (opus, sonnet, flash, pro)
- `--issue <id>`: Link to beads issue
- `--tmux`: Create tmux window for monitoring
- `--inline`: Run in current terminal (blocking)
- `--backend claude`: Use Claude CLI escape hatch
- `--no-track`: Skip beads tracking
- `--tier <light|full>`: Spawn tier (affects artifact requirements)

### `orch status`
List active agents.

```bash
orch status [flags]
```

**Flags:**
- `--json`: Output in JSON format

### `orch complete`
Verify and close agent work.

```bash
orch complete <agent-id>
```

### `orch review`
Batch review completed agents.

```bash
orch review
```

---

## Monitoring Commands

### `orch wait`
Block until agent reaches specified phase.

```bash
orch wait <agent-id> [flags]
```

**Flags:**
- `--timeout <duration>`: Maximum wait time (e.g., "30m")

### `orch monitor`
Real-time SSE event watching with desktop notifications.

```bash
orch monitor
```

### `orch serve`
Start HTTP API server for web UI dashboard.

```bash
orch serve
```

Dashboard available at `http://localhost:5188`.

---

## Agent Management Commands

### `orch send`
Send a message to an existing agent session.

```bash
orch send <session-id> "message"
```

### `orch resume`
Continue a paused agent session.

```bash
orch resume <agent-id>
```

### `orch abandon`
Mark a stuck agent as abandoned.

```bash
orch abandon <agent-id>
```

### `orch clean`
Remove completed/abandoned agents from registry.

```bash
orch clean [flags]
```

**Flags:**
- `--stale`: Remove stale agents only
- `--all`: Remove all agents

---

## Session Commands

### `orch session start`
Begin a focused orchestration session.

```bash
orch session start "goal description"
```

### `orch session status`
Check current session duration, spawns, and focus.

```bash
orch session status
```

### `orch session end`
End session with reflection and handoff creation.

```bash
orch session end
```

---

## Strategic Commands

### `orch focus`
Set or adjust the current session focus.

```bash
orch focus "new goal"
```

### `orch drift`
Check alignment between current work and session focus.

```bash
orch drift
```

### `orch next`
Get recommendation for next action based on backlog state.

```bash
orch next
```

---

## Health Commands

### `orch doctor`
Run health checks on the orchestration system.

```bash
orch doctor [flags]
```

**Flags:**
- `--fix`: Attempt to fix detected issues

**Checks:**
- OpenCode server connectivity
- Dashboard service status
- Daemon running status
- Registry consistency

---

## Server Commands

### `orch servers up`
Start development servers for a project.

```bash
orch servers up <project>
```

### `orch servers down`
Stop development servers for a project.

```bash
orch servers down <project>
```

### `orch servers status`
Show status of all project servers.

```bash
orch servers status
```

---

## Automation Commands

Commands for managing custom launchd agents (background services).

### `orch automation list`
Show all custom launchd agents with their status.

```bash
orch automation list [flags]
```

**Flags:**
- `--json`: Output in JSON format

**Output includes:**
- Agent label (name)
- Loaded/running status
- Last exit code
- Schedule (cron-like, interval, or on-load)

**Scanned agents:** Matches `com.dylan.*`, `com.user.*`, `com.orch.*`, `com.cdd.*` in `~/Library/LaunchAgents/`.

**Example output:**
```
com.orch.daemon          running    exit:0    interval:60s
com.dylan.backup         loaded     exit:0    calendar:daily@3am
com.cdd.sync             stopped    exit:1    on-load
```

### `orch automation check`
Run health checks on custom launchd agents.

```bash
orch automation check [flags]
```

**Flags:**
- `--json`: Output in JSON format

**Detects issues:**
- **Failures:** Agents with non-zero exit codes
- **Not loaded:** Agents with plist files but not loaded into launchd
- **Never run:** Agents configured to run at load that haven't executed

**Exit codes:**
- `0`: All agents healthy
- `1`: Issues detected (useful for scripting/monitoring)

**Example usage in scripts:**
```bash
# Health check in cron or monitoring
if ! orch automation check > /dev/null 2>&1; then
    echo "Launchd agent issues detected"
    orch automation check  # Show details
fi
```

---

## Error Recovery

### Decision Tree

```
Error occurred
    ├── Run `orch status` (is agent running?)
    │   ├── Yes → `orch wait` or `orch send` to guide
    │   └── No → `orch abandon` + respawn
    │
    └── System issues
        ├── Dashboard down → `orch doctor --fix`
        ├── Stale agents → `orch clean --stale`
        └── Registry locked → `rm ~/.orch/registry.lock`
```

### Stuck Agent Protocol

| Condition | Action |
|-----------|--------|
| Idle > 10 minutes | System attempts auto-resume |
| Idle > 15 minutes | Check logs via `orch attach` |
| Idle > 30 minutes | Consider `orch abandon` + respawn |

### Nuclear Options

```bash
orch clean --stale        # Remove stuck agents
orch clean --all          # Full registry reset
rm ~/.orch/registry.lock  # Force unlock registry
```

---

## Checkpoint Management

Set session scope at spawn time:
- **Small:** 1-2 hours
- **Medium:** 2-4 hours
- **Large:** 4-6+ hours

Use `orch wait <id>` to monitor progress, `orch resume <id>` to continue paused sessions.
