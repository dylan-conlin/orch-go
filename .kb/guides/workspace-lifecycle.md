# Workspace Lifecycle Guide

**Purpose:** Authoritative reference for workspace creation, state management, cleanup, and cross-reference operations.

**Last verified:** Jan 17, 2026

**Synthesized from:** 10 investigations (Dec 2025 - Jan 2026) on workspace lifecycle, cleanup, name collisions, session architecture

---

## Quick Reference

### Workspace Types

| Type | Location | Naming | Completion Artifact | Beads Tracked |
|------|----------|--------|---------------------|---------------|
| **Worker** | `{project}/.orch/workspace/og-{skill}-{slug}-{date}-{hex}/` | 4-char hex | SYNTHESIS.md (full) or none (light) | Yes |
| **Spawned Orchestrator** | `{project}/.orch/workspace/og-orch-{slug}-{date}-{hex}/` | 4-char hex | SESSION_HANDOFF.md | No |
| **Interactive Session** | `~/.orch/session/{date}/` | Date-based | SESSION_HANDOFF.md | No |

### Lifecycle States

```
Spawn → Execute → Complete → Archive
  │        │          │          │
[Create] [Active]  [Resolved]  [Cleaned]
```

---

## The Three-Layer Model

Workspace state exists across three independent layers:

| Layer | Storage | Lifecycle | What It Knows |
|-------|---------|-----------|---------------|
| **Workspace** | `{project}/.orch/workspace/` | Filesystem | Spawn context, tier, session link |
| **OpenCode Session** | In-memory + `.opencode/` | Persistent | Conversation history, current status |
| **Beads Issue** | `.beads/issues.jsonl` | Until closed | Phase transitions, completion state |

### Links Between Layers

```
.session_id file         Session title contains workspace name
      ↓                              ↓
WORKSPACE ←─────────────────→ OPENCODE SESSION
                                     ↑
                                HTTP API
                            /session/{id}
```

- **Workspace → Session:** `.session_id` file stores OpenCode session ID
- **Session → Workspace:** Session title format: `{workspace-name} [{beads-id}]`

---

## Workspace Contents

Every workspace contains these files:

### Core Files

| File | Purpose | Created By |
|------|---------|------------|
| `SPAWN_CONTEXT.md` | Worker execution context | Spawn |
| `ORCHESTRATOR_CONTEXT.md` | Spawned orchestrator context | Spawn |
| `META_ORCHESTRATOR_CONTEXT.md` | Meta-orchestrator context | Spawn |

### Metadata Files

| File | Purpose | Created By |
|------|---------|------------|
| `.tier` | light/full/orchestrator - verification rules | Spawn |
| `.session_id` | OpenCode session ID link | Spawn |
| `.beads_id` | Beads issue tracking link | Spawn |
| `.spawn_time` | Nanosecond timestamp for age calculation | Spawn |
| `.spawn_mode` | How session was spawned (opencode, claude, inline) | Spawn |

### Completion Artifacts

| File | Purpose | Created By |
|------|---------|------------|
| `SYNTHESIS.md` | Full tier completion artifact | Agent (full tier only) |
| `SESSION_HANDOFF.md` | Orchestrator completion artifact | Agent (orchestrator tier only) |
| `.review-state.json` | Synthesis recommendation review state | `orch review done` |

---

## Tier System

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
| SPAWN_CONTEXT.md | light | (none expected) | Completed |
| SPAWN_CONTEXT.md | full | SYNTHESIS.md | Completed |
| SPAWN_CONTEXT.md | full | (missing) | Incomplete/Failed |
| ORCHESTRATOR_CONTEXT.md | orchestrator | SESSION_HANDOFF.md | Completed |
| ORCHESTRATOR_CONTEXT.md | orchestrator | (missing) | Active or Abandoned |

---

## Workspace Creation

### Naming Convention

**Spawned workspaces:** `og-{skill}-{slug}-{date}-{hex}`
- `og` = orch-go prefix
- `skill` = first 4 chars of skill name
- `slug` = first 20 chars of task slug
- `date` = DDmon format (e.g., 17jan)
- `hex` = 4-char random hex (65,536 variations per day)

**Interactive sessions:** `{date}` (e.g., `2026-01-17`)

### Code Reference

```go
// pkg/spawn/config.go
func GenerateWorkspaceName(skill, taskSlug string) string {
    hex := make([]byte, 2)
    rand.Read(hex)
    return fmt.Sprintf("og-%s-%s-%s-%x",
        skill[:4], slug[:20], time.Now().Format("02jan"), hex)
}
```

### Why Hex Suffix?

**Problem solved:** Prior to fix (Jan 5, 2026), same-day sessions overwrote each other's artifacts.

**Solution:**
- 4-character random hex suffix via `crypto/rand`
- 65,536 possible suffixes per day per task
- Workspace existence check before spawn

**Reference:** `.kb/investigations/2026-01-05-debug-orchestrator-workspace-name-collision-bug.md`

---

## Workspace Cleanup

### Available Commands

| Command | Purpose | What It Does |
|---------|---------|--------------|
| `orch clean --workspaces` | Archive old workspaces | Moves >7 day workspaces to `archived/` |
| `orch clean --workspaces --workspace-days 14` | Custom retention | Uses 14-day threshold |
| `orch clean --workspaces --dry-run` | Preview | Shows what would be archived |
| `orch doctor --sessions` | Cross-reference | Detects orphaned workspaces/sessions |

### Cleanup Strategy

```bash
# 1. Run diagnostics
orch doctor --sessions

# 2. Preview what would be archived
orch clean --workspaces --dry-run

# 3. Archive old workspaces
orch clean --workspaces
```

### File-Based Completion Detection

For high-performance bulk operations (like `orch clean`), completion is inferred from the filesystem:

1. **Full Tier:** `SYNTHESIS.md` exists
2. **Orchestrator Tier:** `SESSION_HANDOFF.md` exists
3. **Light Tier:** `.beads_id` exists (assumed complete if no active session found)

**Why file-based?**
- Beads API calls are slow (5+ seconds when daemon cold)
- File-based detection completes in <1 second for 300+ workspaces
- Metadata files are self-describing

---

## Session Cross-Reference

### Three-Way Check

`orch doctor --sessions` cross-references three independent state stores:

1. **Workspace files** - `.orch/workspace/*/`
2. **OpenCode sessions** - via `ListDiskSessions()` API
3. **Orchestrator registry** - `~/.orch/sessions.json`

### Detection Types

| Type | Meaning | Action |
|------|---------|--------|
| Orphaned workspace | Session was deleted but workspace remains | Archive or delete |
| Orphaned session | Session exists without workspace | Usually fine (interactive) |
| Zombie session | Active in registry but idle >30min | May need intervention |
| Registry mismatch | Session ID in registry not found in OpenCode | Registry is stale |

### Cross-Reference Algorithm

```go
// 1. Build workspace → session map
for each workspace:
    sessionID = readFile(".session_id")
    workspaceMap[workspaceName] = sessionID

// 2. Get OpenCode sessions
sessions = client.ListDiskSessions()

// 3. Load registry
registry = loadFile("~/.orch/sessions.json")

// 4. Cross-reference
for sessionID in workspaceMap.Values():
    if !sessions.Contains(sessionID):
        report("orphaned workspace")

for session in sessions:
    if !workspaceMap.Has(session.Directory):
        report("orphaned session")
```

---

## Workspace Commands

### Resume by Workspace

```bash
# Resume by beads ID (workers)
orch resume <beads-id>

# Resume by workspace name (orchestrators)
orch resume --workspace <name>

# Resume by session ID (direct)
orch resume --session <id>
```

### Attach to Workspace

```bash
# Attach to workspace TUI (supports partial name matching)
orch attach <workspace>
orch attach auth  # Matches "og-feat-auth-17jan-abc1"
```

### Workspace Status

```bash
# List all active agents
orch status

# Cross-reference workspaces and sessions
orch doctor --sessions
```

---

## Storage Locations

| Location | Scope | Purpose |
|----------|-------|---------|
| `{project}/.orch/workspace/` | Project-local | Worker and spawned orchestrator execution |
| `{project}/.orch/workspace/archived/` | Project-local | Completed workspaces after cleanup |
| `~/.orch/session/` | Global | Interactive human sessions |
| `~/.orch/sessions.json` | Global | Orchestrator session registry |

### Why Project-Local for Workers?

Workers need project-local workspaces to:
- Run project-specific tests
- Access codebase with correct paths
- Keep evidence close to the code
- Enable `orch complete` to run project-local tests

### Why Global for Interactive?

Interactive sessions use global location because:
- Humans work across multiple projects
- Daily continuity is more useful than goal-atomic workspaces
- One SESSION_HANDOFF.md per day simplifies "Landing the Plane"

---

## Common Problems

### "340+ workspaces accumulated"

**Cause:** `orch complete` closes beads issue but doesn't archive workspace. Archival requires manual `orch clean --workspaces`.

**Fix:**
```bash
orch clean --workspaces --dry-run  # Preview
orch clean --workspaces            # Archive
```

### "Workspace name collision"

**No longer a problem.** Hex suffix (Jan 5, 2026 fix) prevents collisions.

### "Can't find agent workspace"

**Use partial matching:**
```bash
orch attach auth  # Matches "og-feat-auth-17jan-abc1"
```

### "Orphaned workspace detected"

**Cause:** OpenCode session was deleted but workspace remains.

**Evaluation:**
1. Check if SYNTHESIS.md exists (completed work)
2. If completed, archive: `orch clean --workspaces`
3. If incomplete, review manually

---

## Key Decisions (Settled)

These are settled. Don't re-investigate:

- **Three workspace types** - Worker, Spawned Orchestrator, Interactive Session
- **File-based state detection** - SYNTHESIS.md/.tier/.beads_id, not API calls
- **Hex suffix for uniqueness** - 4-char random prevents name collisions
- **Project-local for workers** - Keeps evidence close to code
- **Global for interactive** - Humans work across projects
- **Manual archival** - `orch clean --workspaces` is opt-in, not automatic

---

## Related Documentation

- **Model:** `.kb/models/workspace-lifecycle-model/model.md` - Comprehensive workspace lifecycle model
- **Guide:** `.kb/guides/agent-lifecycle.md` - Agent state management
- **Guide:** `.kb/guides/completion.md` - Completion verification
- **Decision:** `.kb/decisions/2026-01-17-three-tier-workspace-hierarchy.md` - Workspace type decision
- **Decision:** `.kb/decisions/2026-01-17-file-based-workspace-state-detection.md` - State detection decision

---

## History

- **Jan 17, 2026:** Created from synthesis of 10 workspace investigations. Establishes authoritative reference for workspace lifecycle, cleanup, and cross-reference operations.
