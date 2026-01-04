# Daemon

**Purpose:** Single authoritative reference for how the orch daemon works for autonomous agent spawning. Read this before debugging daemon issues.

**Last verified:** Jan 4, 2026

---

## What the Daemon Does

The daemon automatically spawns agents for issues labeled `triage:ready`:

```
┌─────────────────────────────────────────────────────────────────┐
│  orch daemon run                                                │
│                                                                 │
│  Loop:                                                          │
│    1. Poll beads: bd list --labels triage:ready                │
│    2. For each ready issue:                                     │
│       - Infer skill from issue type                            │
│       - Spawn agent: orch spawn {skill} --issue {id}           │
│    3. Sleep (default 30s)                                       │
│    4. Repeat                                                    │
└─────────────────────────────────────────────────────────────────┘
```

**Key insight:** Daemon is for batch/overnight work. Orchestrator labels issues, daemon spawns them. Orchestrator stays available for triage and synthesis.

---

## Skill Inference

Daemon infers skill from issue type (NOT from labels):

| Issue Type | Skill |
|------------|-------|
| `bug` | `systematic-debugging` |
| `investigation` | `investigation` |
| `feature` | `feature-impl` |
| `task` | `feature-impl` |

**To control skill selection:** Set the correct issue type when creating:
```bash
bd create "fix login bug" --type bug          # → systematic-debugging
bd create "add dark mode" --type feature      # → feature-impl
bd create "how does auth work" --type investigation  # → investigation
```

---

## Triage Labels

| Label | Meaning |
|-------|---------|
| `triage:ready` | High confidence, daemon can auto-spawn |
| `triage:review` | Needs orchestrator review before spawning |

**Workflow:**
1. Create issue with correct type
2. If confident: `bd label <id> triage:ready`
3. If unsure: Leave as `triage:review`, review later
4. Daemon picks up `triage:ready` issues

---

## Running the Daemon

**Foreground (interactive):**
```bash
orch daemon run
```

**Preview (dry run):**
```bash
orch daemon preview    # Show what would spawn
orch daemon run --dry-run  # Same as preview
```

**Background (launchd):**

The daemon can run via launchd for persistent operation:
- Plist: `~/Library/LaunchAgents/com.orch.daemon.plist`
- Logs: `~/.orch/daemon.log`

```bash
# Check status
launchctl list | grep orch

# Restart
launchctl kickstart -k gui/$(id -u)/com.orch.daemon

# View logs
tail -f ~/.orch/daemon.log
```

---

## Capacity Management

Daemon respects agent capacity:

```bash
orch status  # Shows "Active: X/Y" where Y is max capacity
```

**Behavior:**
- If at capacity, daemon waits for agents to complete
- Default max agents: 5 (configurable)
- Check `~/.orch/config.yaml` for settings

---

## Common Problems

### "Daemon not spawning my issue"

**Checklist:**
1. Issue has `triage:ready` label? `bd show <id>`
2. Issue type is set? (bug/feature/investigation/task)
3. Daemon is running? `launchctl list | grep orch`
4. At capacity? `orch status`

### "Daemon spawning wrong skill"

**Cause:** Issue type doesn't match intended skill.

**Fix:** Update issue type:
```bash
bd update <id> --type bug  # For debugging work
```

### "Daemon not picking up issues"

**Possible causes:**

1. **Wrong directory** - Daemon runs in one repo, issues in another
   - Check daemon's working directory in plist

2. **Beads daemon not running** - `bd` commands fail
   - Check: `orch doctor`

3. **No `triage:ready` label** - Only labeled issues spawn
   - Fix: `bd label <id> triage:ready`

### "Too many agents spawning"

**Cause:** Capacity limit too high or not set.

**Fix:** Configure in `~/.orch/config.yaml`:
```yaml
max_agents: 3
```

---

## Daemon vs Manual Spawn

| Use | Approach |
|-----|----------|
| Batch of 3+ issues | Daemon (label `triage:ready`) |
| Overnight processing | Daemon |
| Single urgent item | Manual `orch spawn` |
| Complex/ambiguous | Manual `orch spawn` |
| Needs custom context | Manual `orch spawn` |

**Daemon is preferred for batch work** because:
- Orchestrator stays available for other work
- Automatic capacity management
- Overnight processing without human presence

---

## Completion Detection

Daemon uses **beads polling**, not SSE:

**Why not SSE (busy→idle)?**
- Agents go idle during loading, thinking, tool execution
- SSE idle triggers false positives
- Only `Phase: Complete` in beads comments is reliable

**Daemon checks:** Polls beads for Phase: Complete comments to know when agents finish.

---

## Key Decisions (from kn)

- **Daemon-first for batch work** - manual spawn for urgent/complex only
- **Skill inference from issue type** - not from labels
- **Beads polling over SSE** - false positive avoidance
- **launchd for persistence** - auto-restart, background operation
- **RPC-first for beads** - with CLI fallback

---

## Debugging Checklist

Before spawning an investigation about daemon issues:

1. **Check kb:** `kb context "daemon"`
2. **Check this doc:** You're reading it
3. **Check daemon running:** `launchctl list | grep orch`
4. **Check services:** `orch doctor`
5. **Check issues:** `bd list --labels triage:ready`
6. **Check capacity:** `orch status`

If those don't answer your question, then investigate. But update this doc with what you learn.
