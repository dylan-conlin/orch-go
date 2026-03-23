# Headless Spawn Mode Guide

**Purpose:** Authoritative reference for headless spawn mode in orch-go. Synthesized from 15 investigations (Dec 20, 2025 - Jan 6, 2026).

**Last updated:** Jan 6, 2026

---

## Overview

Headless mode spawns agents via OpenCode's HTTP API without TUI overhead. It's the **default spawn mode** for automation, daemon operations, and batch processing.

```
orch spawn <skill> "task"   # Headless by default
orch spawn --tmux ...       # Opt-in TUI mode
orch spawn --inline ...     # Blocking TUI in current terminal
```

---

## How It Works

```
orch spawn feature-impl "task"
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. CREATE SESSION via HTTP API                                 │
│     POST /session { title, directory, model }                   │
│     Returns: sessionID                                          │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. SEND PROMPT via HTTP API                                    │
│     POST /session/{id}/prompt_async                             │
│     Body: { parts: [{type: "text", text: ...}], agent, model }  │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. RETURN IMMEDIATELY (fire-and-forget)                        │
│     Agent runs asynchronously in background                     │
│     Orchestrator continues without blocking                     │
└─────────────────────────────────────────────────────────────────┘
```

**Key files:**
- `cmd/orch/spawn_cmd.go:runSpawnHeadless()` - Implementation
- `pkg/opencode/client.go:CreateSession()`, `SendPrompt()` - HTTP API calls

---

## When to Use Headless vs Tmux

| Scenario | Use | Why |
|----------|-----|-----|
| Daemon auto-spawning | Headless | No human present, automation |
| Batch processing (3+ agents) | Headless | Parallel without TUI overhead |
| Overnight work | Headless | Fire-and-forget |
| Debugging spawn issues | Tmux | Visual output, interactive |
| Single agent needing attention | Tmux | Can watch progress live |
| Quick ad-hoc test | Inline | Blocking, sees output directly |

**Default is headless** because it optimizes for the primary use case: automated orchestration.

---

## Monitoring Headless Agents

Since headless agents don't have TUI windows, use these tools:

| Command | Purpose |
|---------|---------|
| `orch status` | Show all active agents with phase, runtime, session ID |
| `orch monitor` | Real-time SSE event stream (all sessions) |
| `orch wait <id>` | Block until agent reaches Phase: Complete |
| `orch send <id> "msg"` | Send message to running agent |
| `orch tail <id>` | View agent's recent output |

**Dashboard:** `orch serve` runs at http://localhost:5188 for visual monitoring.

---

## Completion Detection

Headless agents complete via two mechanisms:

1. **Beads comments** - Agent reports `bd comment <id> "Phase: Complete"` 
2. **SSE events** - OpenCode emits `session.status: idle` when agent stops

The `orch wait` command polls beads comments. The `orch monitor` and daemon use SSE events.

---

## Common Issues and Solutions

### Issue: Model parameter ignored
**Symptom:** Agent uses wrong model despite `--model flash`.

**Cause:** OpenCode's prompt_async API requires model as object, not string.

**Solution:** Fixed in Dec 2023. Model passed as `{"providerID": "google", "modelID": "gemini-2.5-flash"}`.

**Source:** `.kb/investigations/2025-12-23-debug-headless-spawn-model-format.md`

---

### Issue: Agent not discoverable by beads ID
**Symptom:** `orch tail <beads-id>` returns "not found".

**Cause:** Workspace names don't contain beads ID. Old code used naive `strings.Contains(dir, beadsID)`.

**Solution:** Use `findWorkspaceByBeadsID()` which scans SPAWN_CONTEXT.md for authoritative beads reference.

**Source:** `.kb/investigations/2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md`

---

### Issue: bd comment fails with "issue not found"
**Symptom:** Agent spawned with legacy `--no-track` can't report phases.

**Cause:** Old `--no-track` generated placeholder IDs that didn't exist in beads. `--no-track` is now deprecated and creates a real lightweight issue instead (use `--light`).

**Source:** `.kb/investigations/archived/2025-12-22-inv-test-headless-mode.md`

---

### Issue: Headless agents shown as "phantom"
**Symptom:** `orch status` shows headless agents as phantom, not running.

**Cause:** Status command incorrectly checked beads issue status instead of recognizing that OpenCode sessions = running.

**Solution:** Fixed. OpenCode agents always `isPhantom = false` because they have active sessions.

**Source:** `.kb/investigations/2025-12-23-inv-orch-status-shows-headless-agents.md`

---

### Issue: Wrong project directory registered
**Symptom:** Agent runs in wrong directory despite `--workdir`.

**Cause:** OpenCode API requires directory via `x-opencode-directory` HTTP header, not JSON body.

**Solution:** Added `--workdir` flag that sets the header correctly.

**Source:** `.kb/investigations/2025-12-22-inv-headless-spawn-registers-wrong-project.md`

---

### Issue: Token limit explosion (207k > 200k)
**Symptom:** Spawn fails with "token limit exceeded".

**Causes:**
1. **KB context explosion** - Broad queries match 2000+ lines of cross-repo content
2. **Double skill loading** - Both orchestrator (1251 lines) and worker skill loaded

**Solutions:**
1. Set `ORCH_WORKER=1` for worker spawns (saves ~37k tokens)
2. Use `--skip-artifact-check` to bypass KB context
3. Use targeted KB queries, not broad keywords

**Source:** `.kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md`

---

## Architecture Notes

### Session vs Workspace vs Tmux

| Layer | Purpose | Persistence |
|-------|---------|-------------|
| **OpenCode Session** | Agent conversation state | Survives restarts |
| **Workspace** | File artifacts (.orch/workspace/) | Permanent |
| **Tmux Window** | Visual terminal (opt-in) | Ephemeral |

Headless mode creates the first two but not the third.

### Per-Message Model Selection

OpenCode uses **per-message** model selection, not per-session. The model must be passed with each prompt, not just at session creation. This is why SendPrompt includes the model parameter.

### Fire-and-Forget Design

Headless spawn returns immediately after sending the prompt. The orchestrator continues without blocking. This enables:
- Parallel agent spawning
- Non-blocking daemon operations  
- Better resource utilization

---

## Key Decisions (from investigations)

| Decision | Rationale | Source |
|----------|-----------|--------|
| Headless is default | Optimizes for automation, reduces TUI overhead | kn-6f7dd1 |
| Tmux is opt-in via `--tmux` | Visual monitoring when needed | kn-318507 |
| Per-message model selection | OpenCode design, not our choice | kn-a485c6 |
| Beads comments for phase tracking | Spawn-mode agnostic, works for all modes | Design investigation |
| ORCH_WORKER=1 for workers | Prevents double skill loading | kn-56f594 |

---

## Related Resources

- **Spawn guide:** `.kb/guides/spawn.md` - General spawn flow
- **Daemon guide:** `.kb/guides/daemon.md` - Daemon integration with headless
- **Status dashboard:** `http://localhost:5188` via `orch serve`
- **OpenCode API:** `pkg/opencode/client.go` - HTTP API implementation

---

## Investigations Synthesized

This guide consolidates knowledge from 15 headless-related investigations:

1. `2025-12-20-inv-implement-headless-spawn-mode-add.md` - Initial implementation
2. `2025-12-20-inv-make-headless-mode-default-deprecate.md` - Default flip
3. `2025-12-20-inv-scope-out-headless-swarm-implementation.md` - Swarm design
4. `2025-12-21-inv-headless-spawn-not-sending-prompts.md` - Binary mismatch fix
5. `2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md` - Lookup fix
6. `2025-12-22-inv-headless-spawn-mode-readiness-what.md` - Production readiness
7. `2025-12-22-inv-headless-spawn-registers-wrong-project.md` - --workdir fix
8. `2025-12-23-debug-headless-spawn-model-format.md` - Model object format
9. `2025-12-23-inv-headless-spawn-does-not-pass.md` - Model parameter threading
10. `2025-12-23-inv-orch-status-shows-headless-agents.md` - Phantom status fix
11. `2025-12-23-inv-token-limit-explosion-headless-spawn.md` - Token bloat analysis
12. `2025-12-22-inv-test-headless-mode.md` (archived) - E2E testing
13. `2025-12-22-inv-test-headless-spawn-list-files.md` (archived) - Filesystem verification
14. `2025-12-22-inv-test-headless-spawn.md` (archived) - Default mode verification
15. `2025-12-23-inv-test-headless-spawn-after-fix.md` (archived) - Post-fix validation
