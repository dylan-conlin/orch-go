# Investigation: Workspace, Session, and Resumption Architecture

**Status:** complete
**Date:** 2026-01-06
**Context:** Meta-orchestrator session investigating "orphaned workspaces" led to complete mental model of spawn/session architecture

---

## Summary

Built complete mental model of how orch workspaces, OpenCode sessions, and tmux windows relate. Key findings:
1. "Orphaned workspaces" was a misunderstanding - light tier workers don't produce SYNTHESIS.md by design
2. OpenCode sessions are persistent and resumable via HTTP API
3. Several gaps exist in tooling for session management

---

## The Three Layers

| Layer | Purpose | Location | Persistence |
|-------|---------|----------|-------------|
| **Workspace** | File-based spawn state | `.orch/workspace/{name}/` | Filesystem |
| **OpenCode Session** | Conversation + agent state | In-memory + disk | Survives restarts |
| **Tmux Window** | Visual terminal access | tmux session | Ephemeral |

### Workspace Contents

```
.orch/workspace/{name}/
├── SPAWN_CONTEXT.md           # Worker context (or ORCHESTRATOR_CONTEXT.md / META_ORCHESTRATOR_CONTEXT.md)
├── .tier                      # "light", "full", or "orchestrator"
├── .session_id                # OpenCode session ID (key link for resumption)
├── .spawn_time                # Nanosecond timestamp
├── .beads_id                  # Beads issue ID (workers only, not orchestrators)
├── SYNTHESIS.md               # Completion artifact (full tier only)
└── SESSION_HANDOFF.md         # Completion artifact (orchestrator tier only)
```

### Links Between Layers

```
.session_id file         Session title contains workspace name
      ↓                              ↓
WORKSPACE ←─────────────────→ OPENCODE SESSION ←───→ TMUX WINDOW
                                     ↑
                                HTTP API
                            /session/{id}
```

- **Workspace → Session:** `.session_id` file stores OpenCode session ID
- **Session → Workspace:** Session title format: `{workspace-name} [{beads-id}]`
- **Tmux → Session:** `opencode --attach <server> --session <id>`

---

## The Tier System

### Tiers and Expected Artifacts

| Tier | Context File | Completion Artifact | Beads Tracked? |
|------|--------------|---------------------|----------------|
| `light` | SPAWN_CONTEXT.md | None required | Yes |
| `full` | SPAWN_CONTEXT.md | SYNTHESIS.md | Yes |
| `orchestrator` | ORCHESTRATOR_CONTEXT.md | SESSION_HANDOFF.md | No |

### Tier Assignment by Skill

**Full tier** (produce knowledge artifacts):
- investigation, architect, research, codebase-audit
- design-session, systematic-debugging

**Light tier** (implementation-focused):
- feature-impl, reliability-testing, issue-creation

**Orchestrator tier** (coordination):
- orchestrator, meta-orchestrator

Unknown skills default to `full` (conservative).

### Interpreting Workspace State

| Context File | .tier | Artifact Present | State |
|--------------|-------|------------------|-------|
| SPAWN_CONTEXT.md | light | (none expected) | ✅ Completed |
| SPAWN_CONTEXT.md | full | SYNTHESIS.md | ✅ Completed |
| SPAWN_CONTEXT.md | full | (missing) | ❌ Incomplete/Failed |
| ORCHESTRATOR_CONTEXT.md | orchestrator | SESSION_HANDOFF.md | ✅ Completed |
| ORCHESTRATOR_CONTEXT.md | orchestrator | (missing) | ⚠️ Active or Abandoned |
| META_ORCHESTRATOR_CONTEXT.md | orchestrator | (missing) | ⚠️ Active (expected for interactive) |

---

## Key Finding: Sessions Are Resumable

**Tested and proven:** Sent HTTP message to "completed" session, Claude responded with full context.

```bash
# This works - session was waiting at /exit, woke up immediately
curl -X POST "http://localhost:4096/session/{id}/prompt_async" \
  -H "Content-Type: application/json" \
  -d '{"parts": [{"type": "text", "text": "continue"}], "agent": "build"}'
```

**Implications:**
- "Completed" workers that haven't been cleaned up can be resumed
- Sessions persist through OpenCode server restarts (disk-backed)
- The only truly dead sessions are explicitly closed or garbage-collected

---

## Spawn Modes

### Headless (default for workers)
```bash
orch spawn feature-impl "task"
```
- Creates workspace, starts OpenCode session via HTTP API
- No tmux window - session runs server-side
- Returns immediately, agent works in background
- Can resume via `orch resume` or direct API

### Tmux (default for orchestrators, opt-in for workers)
```bash
orch spawn --tmux feature-impl "task"
orch spawn orchestrator "goal"  # tmux by default
```
- Creates workspace, starts OpenCode session
- Opens tmux window with `opencode --attach`
- Closing tmux PAUSES but doesn't kill session (it's server-side)

---

## Current Capabilities

| Capability | Command | Notes |
|------------|---------|-------|
| Resume worker by beads ID | `orch resume <beads-id>` | Finds workspace, sends message |
| Resume by session ID | `curl POST /session/{id}/prompt_async` | Direct API, no orch command |
| List all sessions | `curl GET /session` | OpenCode API |
| Check session exists | `curl GET /session/{id}` | Returns 200 if exists |

---

## Identified Gaps

### 1. No `orch attach <workspace>` Command
**Problem:** Can't easily open TUI for existing session from workspace name
**Impact:** When tmux window closes, no easy way to reconnect visually
**Solution:** Read `.session_id`, run `opencode --attach --session <id>`
**Issue:** orch-go-cnkbv

### 1a. ✅ `opencode attach --session` FIXED
**Problem:** `opencode attach http://localhost:4096 --session <id>` returned "Session not found" even when session exists via API
**Root cause:** Sessions are project-scoped. TUI used client's cwd, server used different project context.
**Fix:** Added `Session.getGlobal()` that searches across all projects (commit e3a23604b in Dylan's fork)
**Issue:** orch-go-1qgwg (closed)
**Status:** FIXED - requires Dylan's opencode fork (see ~/.claude/CLAUDE.md "OpenCode: Use Dylan's Fork")

### 2. `orch resume` Only Works for Workers
**Problem:** Takes beads ID, but orchestrators don't have beads IDs
**Impact:** Can't resume orchestrator sessions (the price-watch friction)
**Solution:** Accept `--workspace <name>` or `--session <id>` flags
**Issue:** orch-go-xdcpc

### 3. No Workspace ↔ Session Cross-Reference
**Problem:** Can't detect orphaned workspaces vs orphaned sessions
**Impact:** No systematic cleanup, can't detect zombies
**Solution:** `orch doctor` could compare both sources
**Issue:** orch-go-0l2f9

### 4. Registry Population Issues
**Problem:** `~/.orch/registry.json` appears empty but `orch status` shows sessions
**Impact:** Unclear source of truth for orchestrator sessions
**Status:** Needs investigation (didn't complete debugging)

### 5. No Session Cleanup Strategy
**Problem:** OpenCode sessions accumulate forever
**Impact:** Disk usage, confusion about what's active
**Solution:** Policy decision needed - archive after X days?
**Issue:** orch-go-1kk2u (updated from "orphaned workspaces" to cleanup strategy)

---

## Corrected Understanding: "Orphaned Workspaces"

**Previous belief:** 200+ workspaces without SYNTHESIS.md are orphaned/failed

**Actual state (288 workspaces analyzed):**
- 218 light-tier workers (no SYNTHESIS.md expected) ✅
- 46 workers with SYNTHESIS.md ✅
- 12 orchestrators with SESSION_HANDOFF.md ✅
- ~10 actually incomplete (test spawns, abandoned sessions)

**The "orphan problem" was a misunderstanding of the tier system.**

---

## Recommendations

1. **Document tier system in orchestrator skill** - This understanding keeps getting lost
2. **Add `orch attach <workspace>`** - Low effort, high value
3. **Extend `orch resume`** - Accept workspace name for orchestrators
4. **Add `orch doctor --sessions`** - Cross-reference workspaces and OpenCode sessions
5. **Update orch-go-1kk2u** - Change from "orphaned workspaces bug" to cleanup strategy feature

---

## Test Commands Used

```bash
# Analyze workspace state
cd .orch/workspace
for dir in */; do
  tier=$(cat "${dir}.tier" 2>/dev/null)
  has_synthesis=$([ -f "${dir}SYNTHESIS.md" ] && echo "YES" || echo "no")
  echo "${dir}: tier=$tier synthesis=$has_synthesis"
done

# Check OpenCode sessions
curl -s http://localhost:4096/session | jq '.[] | "\(.id) | \(.title)"'

# Resume a session via API
curl -X POST "http://localhost:4096/session/{id}/prompt_async" \
  -H "Content-Type: application/json" \
  -d '{"parts": [{"type": "text", "text": "continue"}], "agent": "build"}'
```

---

## References

- `pkg/spawn/config.go` - Tier constants and skill mappings
- `pkg/spawn/session.go` - Session ID file handling
- `pkg/opencode/client.go` - OpenCode HTTP API client
- `pkg/session/registry.go` - Orchestrator session registry
- `cmd/orch/spawn_cmd.go` - Spawn command implementation
- `cmd/orch/resume.go` - Resume command (beads-id only currently)
