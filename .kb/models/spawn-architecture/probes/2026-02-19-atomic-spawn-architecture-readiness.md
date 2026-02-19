# Probe: Atomic Spawn Architecture Readiness

**Date:** 2026-02-19
**Status:** Complete
**Model:** Spawn Architecture
**Issue:** orch-go-1083

## Question

Is the current spawn infrastructure ready for atomic spawn with rollback? What are the exact code paths, existing partial state risks, and feasibility of the 4-step atomic spawn contract (beads tag, workspace manifest, OpenCode session metadata, spawn evidence)?

## What I Tested

### 1. Current Write Ordering in Spawn Pipeline

Examined the spawn pipeline end-to-end across all spawn modes (headless, tmux, inline, claude):

**Step 4 (beads tracking) - `pkg/orch/extraction.go:SetupBeadsTracking()`:**
- Creates beads issue or resolves existing issue
- Updates status to `in_progress`
- No `orch:agent` label applied

**Step 13 (workspace writes) - `pkg/spawn/context.go:WriteContext()`:**
- Creates workspace directory
- Writes SPAWN_CONTEXT.md
- Writes `.tier`, `.spawn_time`, `.beads_id`, `.spawn_mode` dotfiles
- Writes `AGENT_MANIFEST.json` (no SessionID field)
- No rollback on failure

**Step 14 (session creation) - varies by backend:**
- Headless: `client.CreateSession()` with metadata {beads_id, workspace_path, tier, spawn_mode}
- Tmux: `tmux.CreateWindow()` + `client.FindRecentSessionWithRetry()` to capture session ID
- Claude: `spawn.SpawnClaude()` - no OpenCode session at all
- Inline: Same as headless

**Post-session (session ID write):**
- `spawn.WriteSessionID()` writes `.session_id` dotfile
- Written after session creation, not atomic with other workspace files

### 2. Existing Partial State Risks

Traced failure scenarios:

| Failure Point | Partial State Left Behind |
|---|---|
| Beads issue created, workspace write fails | Issue exists in `in_progress` with no agent |
| Workspace written, session creation fails | Workspace dir exists with SPAWN_CONTEXT.md but no session |
| Session created, session ID write fails | Session running but not discoverable via workspace |
| Tmux window created, OpenCode session not found | Window exists but no session ID captured |

### 3. Existing Infrastructure Analysis

**Beads label API:** Fully implemented
- `client.AddLabel(id, label)` / `client.RemoveLabel(id, label)` - RPC + CLI fallback
- `ListArgs.Labels` / `ListArgs.LabelsAny` - filtering support
- `FallbackRemoveLabel()` for rollback via CLI

**AgentManifest struct:** Missing SessionID field
- Currently: WorkspaceName, Skill, BeadsID, ProjectDir, GitBaseline, SpawnTime, Tier, SpawnMode, Model
- SessionID written to separate `.session_id` dotfile
- `ReadAgentManifestWithFallback()` already reads from OpenCode metadata as primary source

**OpenCode session metadata:** Already written at spawn for headless/inline
- `metadata := map[string]string{"beads_id", "workspace_path", "tier", "spawn_mode"}`
- `client.CreateSession(title, dir, model, metadata, ttl)`
- Not written for tmux/claude modes (session created by TUI/CLI, not API)

**Workspace batch lookup:** Not implemented
- ADR calls for `workspace.LookupByBeadsIDs()` but no such function exists
- Would need to scan `.orch/workspace/*/AGENT_MANIFEST.json` or use index

## What I Observed

### Key Findings

1. **spawn_cmd.go is 798 lines** - Under 1,500 threshold, no extraction needed before atomic spawn
2. **3 of 4 ADR requirements have existing infrastructure:**
   - Beads labeling: full API exists
   - Workspace manifest: exists as AGENT_MANIFEST.json (needs SessionID field)
   - Session metadata: already written for headless/inline modes
3. **1 new capability needed:** Workspace batch lookup by beads ID
4. **Claude/tmux backends don't write session metadata** - Session created by TUI/CLI, metadata requires API call after the fact
5. **No rollback logic exists** - All writes are fire-and-forget
6. **Manifest is written before session creation** - SessionID can't be included at initial write time

### Backend-Specific Challenges

| Backend | Creates Session Via | Can Write Metadata? | Can Capture SessionID? |
|---|---|---|---|
| Headless | OpenCode HTTP API | Yes (at creation) | Yes (immediate) |
| Inline | OpenCode HTTP API | Yes (at creation) | Yes (immediate) |
| Tmux | `opencode attach` TUI | No (TUI creates it) | Maybe (retry-based discovery) |
| Claude | Claude CLI binary | No (no OpenCode session) | No (no session at all) |

## Model Impact

### Confirms

- **Invariant 1 (workspace name = kebab-case):** Confirmed at `pkg/spawn/workspace.go:GenerateWorkspaceName()`
- **Invariant 2 (beads ID required for phase reporting):** Confirmed, `--no-track` creates untracked IDs
- **Invariant 5 (session scoping per-project):** Confirmed via `x-opencode-directory` header

### Extends

- **The manifest IS the workspace binding** - AGENT_MANIFEST.json is already 90% of what the ADR calls "workspace manifest". Adding SessionID field and a batch lookup function completes it.
- **Atomic spawn must handle 4 backends differently** - The ADR pseudocode assumes a single path (API-created session). In reality, tmux and claude backends create sessions through different mechanisms, so the "atomic" guarantee has different shapes per backend.
- **Rollback for beads label is straightforward** (RemoveLabel exists), **rollback for workspace is straightforward** (os.RemoveAll), but **rollback for OpenCode sessions is harder** (no delete session API found).

### New Constraint Discovered

- **Claude backend cannot participate in session metadata** - No OpenCode session exists for `--backend claude` spawns. The atomic contract for claude mode can only guarantee beads tag + workspace manifest, not session metadata.
