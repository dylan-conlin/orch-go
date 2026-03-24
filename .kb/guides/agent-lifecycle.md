# Agent Lifecycle Guide

**Purpose:** Single authoritative reference for agent state management, display, and coordination in the orch-go system.

**Last verified:** Feb 26, 2026

**Synthesized from:** 45+ investigations (Dec 20, 2025 - Jan 17, 2026)

---

## Quick Reference

### Agent States

| State | Description | UI Indicator |
|-------|-------------|--------------|
| `active` | Agent currently running | Green badge |
| `completed` | Agent finished successfully | Blue badge |
| `abandoned` | Agent was stopped before completion | Yellow badge |

### Display States (for UI)

| Display State | Condition | Indicator |
|---------------|-----------|-----------|
| `running` | active + is_processing | Yellow pulse |
| `ready-for-review` | active + phase=Complete | Ready badge |
| `idle` | active + no activity > 60s | Idle indicator |
| `waiting` | active + awaiting input | Waiting... |
| `completed` | status=completed | Blue badge |
| `abandoned` | status=abandoned | Yellow badge |

---

## The Flow

```
orch spawn                    bd comment "Phase: Complete"           orch complete
     │                                    │                                │
     ▼                                    ▼                                ▼
┌─────────┐    agent works    ┌──────────────────┐    orchestrator    ┌─────────┐
│ Spawned │ ───────────────►  │ Phase: Complete  │ ────────────────►  │ Closed  │
└─────────┘                   └──────────────────┘                    └─────────┘
     │                                    │                                │
     ▼                                    ▼                                ▼
  Creates:                           Agent reports:                   Orchestrator:
  - OpenCode session                 - bd comment with phase          - Verifies work
  - Beads issue                      - SYNTHESIS.md (full tier)       - Closes beads issue
  - Workspace directory              - BRIEF.md (full tier)           - Copies BRIEF to .kb/briefs/
                                     - Git commits                    - Thread back-propagation
```

---

## Architecture: Four-Layer State Model

Agent state exists across four independent layers:

| Layer | Storage | Lifecycle | What It Knows |
|-------|---------|-----------|---------------|
| **tmux windows** | Runtime (volatile) | Until window closed | Agent visible, window ID |
| **OpenCode in-memory** | Server process | Until server restart | Session ID, current status |
| **OpenCode on-disk** | `.opencode/` files | Persistent | Full message history |
| **beads comments** | `.beads/issues.jsonl` | Persistent | Phase transitions, metadata |

**Key insight:** The registry was a fifth layer attempting to cache all four, which caused drift. The solution is to query authoritative sources directly.

### Source of Truth by Concern

| Question | Source | NOT this |
|----------|--------|----------|
| Is agent complete? | Beads issue status = "closed" | OpenCode session exists |
| What phase is agent in? | Beads comments (Phase: X) | Dashboard shows "active" |
| Did agent finish? | Phase: Complete comment exists | Session went idle |
| Is agent processing? | SSE session.status = busy | Session exists |

**Beads is the source of truth for agent status.** OpenCode sessions persist to disk indefinitely. An OpenCode session existing means nothing about whether the agent is done. Only beads matters.

### Layer Cleanup on Completion

When `orch complete` runs, it must clean up all four layers in the correct order:

```
1. Close beads issue (authoritative "done" signal)
2. Delete OpenCode session (prevents ghost agents)
3. Export transcript if needed
4. Archive workspace to archived/
5. Close tmux window
6. Invalidate serve cache
```

**Critical:** Delete OpenCode session BEFORE status checks. Sessions persist to disk and appear as "running" agents in `orch status` if not deleted.

**Reference:** `.kb/guides/completion.md` for full cleanup details.

---

## Pre-Spawn Duplicate Prevention

Before spawning, check if work is already done:

```go
// Check for Phase: Complete before spawning
comments := bd.GetComments(beadsID)
for _, c := range comments {
    if strings.Contains(c, "Phase: Complete") {
        return errors.New("work already complete, not respawning")
    }
}
```

**Why:** Prevents duplicate spawns when:
- Agent finished but `orch complete` wasn't run
- Daemon retry triggers spawn for closed issue
- Manual respawn of already-done work

**Code reference:** `pkg/spawn/pre_spawn_check.go`

---

## Dual-Mode Architecture

The system uses two modes that serve different needs:

| Mode | Purpose | When Used |
|------|---------|-----------|
| **tmux** | Visual agent access | Orchestrator needs to see real-time agent activity |
| **HTTP API** | Programmatic state query | Dashboard, automation, `orch status` |

**Why both are needed:**
- tmux provides window-per-agent isolation for parallel visual monitoring
- HTTP API provides programmatic state access without TUI overhead
- Neither can replace the other - they serve complementary needs

**The "return to tmux" pattern** happens because orchestrators need to SEE what agents are doing. HTTP API gives you data; tmux gives you visibility.

---

## Workspace Structure

Each spawned agent has a workspace:

```
.orch/workspace/{name}/
├── SPAWN_CONTEXT.md           # Worker context
├── .tier                      # "light", "full", or "orchestrator"
├── .session_id                # OpenCode session ID (for resumption)
├── .spawn_time                # Nanosecond timestamp
├── .beads_id                  # Beads issue ID
├── SYNTHESIS.md               # Completion artifact (full tier only)
└── SESSION_HANDOFF.md         # Completion artifact (orchestrator tier only)
```

### Tier System

| Tier | Context File | Completion Artifact | Beads Tracked? |
|------|--------------|---------------------|----------------|
| `light` | SPAWN_CONTEXT.md | None required | Yes |
| `full` | SPAWN_CONTEXT.md | SYNTHESIS.md | Yes |
| `orchestrator` | ORCHESTRATOR_CONTEXT.md | SESSION_HANDOFF.md | No |

**Workspace Path Pattern:** `{PROJECT_DIR}/.orch/workspace/{agent.id}/`

The agent `id` IS the workspace name. PROJECT_DIR is stored in SPAWN_CONTEXT.md for cross-project visibility.

---

## Dashboard Implementation

### SSE Event Handling

The dashboard receives real-time updates via Server-Sent Events:

| Event Type | Contains | Use For |
|------------|----------|---------|
| `session.status` | `{sessionID, status: {type: busy|idle}}` | Update `is_processing` |
| `message.part` | `{sessionID, part: {...}}` | Update `current_activity` |
| `session.created` | Session metadata | Add new agent to list |

**Processing state detection:**
```typescript
// When session.status is 'busy' or message.part fires
agent.is_processing = true

// When session.status is 'idle'
agent.is_processing = false
agent.current_activity = undefined
```

### Dashboard Status Logic

The dashboard (`orch serve`) determines status in this order:

1. **Check beads issue status** - If closed → show "completed"
2. **Check Phase: Complete comment** - If present → show "completed" 
3. **Check SYNTHESIS.md** - If exists in workspace → show "completed"
4. **Fall back to session state** - active/idle based on recent activity

**If dashboard shows wrong status:**
1. Check beads: `bd show <id> --json | jq '.status'`
2. If beads says closed, dashboard should show completed (refresh browser)
3. If beads says open but agent is done, run `orch complete <id>`

### Agent Card Layout

Cards should maintain consistent height using reserved space pattern:

```svelte
<!-- WRONG: Conditional rendering causes height jitter -->
{#if agent.current_activity}
  <ActivitySection />
{/if}

<!-- RIGHT: Always render with placeholder -->
{#if agent.status === 'active'}
  {#if agent.current_activity}
    <ActivitySection />
  {:else}
    <span>Waiting for activity...</span>
  {/if}
{/if}
```

### Stable Sort for Grid Layout

To prevent card jostling on updates:
- **Active/Recent sections:** Sort by `spawned_at` (immutable)
- **Archive section:** Sort by `updated_at` (volatile, shows recency)

```typescript
function sortAgents(agents: Agent[], useStableSort: boolean) {
  if (useStableSort) {
    // Use spawned_at - never changes, prevents jostling
    return sort(agents, a => a.spawned_at);
  } else {
    // Use updated_at - shows most recently modified
    return sort(agents, a => a.updated_at);
  }
}
```

---

## Display State Computation

Centralized in `agents.ts`:

```typescript
type DisplayState = 'running' | 'ready-for-review' | 'idle' | 'waiting' | 'completed' | 'abandoned';

function computeDisplayState(agent: Agent): DisplayState {
  if (agent.status === 'completed') return 'completed';
  if (agent.status === 'abandoned') return 'abandoned';
  
  if (agent.status === 'active') {
    if (agent.phase?.toLowerCase() === 'complete') return 'ready-for-review';
    if (agent.is_processing) return 'running';
    if (agent.current_activity?.timestamp) {
      const idleMs = Date.now() - agent.current_activity.timestamp;
      if (idleMs > 60000) return 'idle';
    }
    return 'waiting';
  }
  
  return 'waiting';
}
```

---

## Health & Self-Healing

Agents are equipped with a "digital nervous system" (Coaching Plugin) that monitors for behavioral degradation.

### Pain as Signal
Autonomous error correction requires that agents *feel* the friction of their own failure in real-time.
- **Signal:** Detections (Analysis Paralysis, Frame Collapse) are injected as tool-layer messages.
- **Action:** Agents must treat these signals as authoritative feedback from the hierarchy.
- **Pivot:** Upon receiving a signal, the agent should stop the current loop, reason about the failure, and pivot (e.g., change tool, escalate, or request context reset).

---

## Cross-Project Visibility

### The Problem

When agents are spawned with `--workdir /other/project`:
- Workspace goes to `/other/project/.orch/workspace/`
- Dashboard running from orch-go only sees orch-go's workspaces

### The Solution

1. **Discover projects from OpenCode sessions:**
   ```go
   // Extract unique project directories from session.Directory
   projectDirs := extractUniqueProjectDirs(sessions)
   ```

2. **Build workspace caches for each project:**
   ```go
   // Scan each project's .orch/workspace/ in parallel
   wsCache := buildMultiProjectWorkspaceCache(projectDirs)
   ```

3. **Route beads queries to correct project:**
   ```go
   // Use PROJECT_DIR from SPAWN_CONTEXT.md
   GetCommentsBatchWithProjectDirs(beadsIDs, projectDirMap)
   ```

---

## Lifecycle Phase Tracking

Agents report phase transitions via beads comments:

```bash
bd comment <beads-id> "Phase: Planning - Starting investigation"
bd comment <beads-id> "Phase: Implementing - Building feature"
bd comment <beads-id> "Phase: Complete - All tests passing"
```

**Phase parsing:**
```go
phasePattern := regexp.MustCompile(`(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?`)
```

**Completion detection:**
- `Phase: Complete` is the authoritative signal
- Session status (busy/idle) cannot reliably indicate completion
- `orch complete` verifies phase before closing

---

## Agent Detail Panel

When clicking an agent card, show a slide-out panel with:

### Header
- Status badge + phase badge + runtime
- Processing indicator (yellow pulse when active)

### Identifiers (copyable)
- Workspace ID (agent name)
- Session ID (for API access)
- Beads ID (for CLI commands)

### State-Aware Content

| Agent State | Show |
|-------------|------|
| Active | Live output streaming, current activity |
| Completed | Synthesis (TLDR, outcome, recommendation) |
| Abandoned | Failure report if exists |

### Actions (state-aware)

| State | Actions |
|-------|---------|
| Active | Send message, Abandon, Open workspace |
| Completed | Complete, View synthesis, Open workspace |
| Abandoned | View failure report, Respawn |

---

## Multi-Agent Synthesis

### Workspace Isolation

Each agent operates in its own `.orch/workspace/{name}/` directory, preventing file-level conflicts.

### Registry Merge

For concurrent registry access:
- Uses file locking with `syscall.Flock`
- Merge logic compares `UpdatedAt` timestamps
- Newer wins on conflict

### SYNTHESIS.md Pattern (D.E.K.N.)

Each agent produces a SYNTHESIS.md following D.E.K.N.:
- **D**elta: What changed (files, commits)
- **E**vidence: What was observed (test output)
- **K**nowledge: What was learned (decisions, constraints)
- **N**ext: What should happen (recommendation)

### Batch Review

`orch review` aggregates multiple agent outputs:
- Groups completions by project
- Shows synthesis cards for each agent
- Enables efficient triage of parallel agent work

---

## Common Problems

### "Dashboard shows agent as active but it's done"

**Cause:** `orch complete` wasn't run, so beads issue is still open.

**Fix:** Run `orch complete <id>`. If that's blocked, check what gate is blocking.

**NOT the fix:** Deleting OpenCode sessions. That treats the symptom, not the cause.

### "orch complete is blocked / requires flags"

**Cause:** Gates were added that require verification before closing.

**See:** `.kb/guides/completion-gates.md` for full reference on all gates.

**Quick bypass:** `orch complete <id> --force` skips all verification gates.

### "Agent went idle but didn't report Phase: Complete"

**Cause:** Agent ran out of context, crashed, or didn't follow the completion protocol.

**This is expected behavior.** Session idle ≠ work complete. Only agents that explicitly run `bd comment <id> "Phase: Complete"` are considered done.

**Fix:** Check workspace for what agent accomplished, then either:
- `orch complete <id> --force` if work is done
- `orch abandon <id>` if work is incomplete

**Important:** `orch abandon` must remove the `orch:agent` label AND assignee from the beads issue, not just reset status. Otherwise the issue remains invisible to future daemon spawning.

**Important:** `orch clean` must check tmux pane process liveness before closing windows. A window may appear idle but still have an active Claude Code process.

### "Lots of zombie agents in dashboard"

**Cause:** Agents finished but `orch complete` was never run.

**Fix:** Complete or abandon each one. Don't delete OpenCode sessions as a workaround.

**Prevention:** Complete agents promptly. Don't let them accumulate.

### NaN Runtime Display

**Cause:** `formatDuration()` receives null/undefined `spawned_at` for completed agents.

**Fix:** Guard against invalid inputs:
```typescript
function formatDuration(isoDate: string | null): string {
  if (!isoDate) return '-';
  const time = new Date(isoDate).getTime();
  if (isNaN(time)) return '-';
  // ... rest of formatting
}
```

### Agent Cards Growing/Shrinking

**Cause:** Conditional rendering of bottom sections based on content presence.

**Fix:** Always render section container with placeholder (see "Agent Card Layout" above).

### Excess Whitespace in Completed Cards

**Cause:** TLDR displayed twice (in title AND synthesis section).

**Fix:** Synthesis section only shows outcome badge:
```svelte
{#if agent.status === 'completed' && agent.synthesis?.outcome}
  <OutcomeBadge outcome={agent.synthesis.outcome} />
{/if}
```

---

## Agent Liveness Detection

### Phase-Based Liveness (Feb 2026)

Agent liveness is determined by **beads phase comments**, not tmux or OpenCode session state.

| Backend | Liveness Signal | Why |
|---------|----------------|-----|
| OpenCode (headless) | OpenCode session status | Has session_id in manifest |
| Claude CLI (tmux) | Beads phase comments | No OpenCode session exists |

**Why not tmux liveness:** Tmux liveness violated the two-lane decision (tmux is UI-only, not state). Phase comments are the authoritative heartbeat — they work for all backends without tmux dependency.

**Known gap:** `queryTrackedAgents` only checks OpenCode sessions for liveness. Claude-backend agents with no OpenCode session appear dead even when running in tmux. Phase comments from beads fill this gap as a liveness proxy.

### Scan Ordering (serve_agents.go)

Scan ordering determines which state layer claims an agent first — the first scan to find a beads_id wins the duplicate check, and subsequent scans defer. **Tmux scan must run before completed workspace scan** or tmux agents get misidentified as completed.

### Query Engine Internals

- `queryTrackedAgents` extracts phase from beads comments via **per-issue RPC calls** (no batch API exists). Acceptable for typical agent counts (<20).
- Uses `beads.FallbackListWithLabel` for CLI fallback, consistent with existing Fallback* pattern in `pkg/beads/client.go`.

---

## No Local Agent State (Invariant)

orch-go must not maintain local agent state — no registries, projection DBs, SSE materializers, or caches for agent discovery. Query beads and OpenCode directly.

**Five failed iterations:** registry.json, sessions.json, state.db, workspace cache, and multi-source reconciliation all drifted from reality, causing ghost agents, phantom status, and days of debugging. The fix is structural extraction (strangler fig) — create attractor packages via extraction, agents naturally route there without caching.

---

## Skill Compliance Lessons

### Textual-Only Delegation Constraints Don't Work

Textual-only orchestrator delegation constraints in skills failed. An agent collapsed into implementation within 30 seconds of an ambiguous directive ("work toward the focus"). The orchestrator skill's action space table had zero enforcement — pre-response orientation checks never surfaced. **Lesson:** Behavioral constraints require enforcement mechanisms (hooks, gates), not just instructional text.

---

## Key Decisions (Settled)

These are settled. Don't re-investigate:

- **Dashboard uses beads as source of truth** - not session state
- **SSE busy→idle cannot detect completion** - agents go idle for many reasons
- **Phase: Complete is the only reliable signal** - from beads comments
- **SYNTHESIS.md is fallback for untracked agents** - when no beads issue exists
- **Dual-mode architecture (tmux + HTTP)** - each serves distinct, irreplaceable needs
- **Stable sort using spawned_at** - prevents card jostling in Active/Recent sections
- **Phase-based liveness over tmux liveness** - works for all backends, no tmux dependency
- **No local agent state** - query authoritative sources directly, never cache agent state

---

## What Lives Where

| Thing | Location | Lifecycle |
|-------|----------|-----------|
| OpenCode session | OpenCode's internal storage | Persists until deleted |
| Beads issue | `.beads/` | Created at spawn, closed at complete |
| Workspace | `.orch/workspace/<name>/` | Created at spawn, persists forever |
| SPAWN_CONTEXT.md | Workspace | Created at spawn |
| SYNTHESIS.md | Workspace | Created by agent before completion |

---

## Debugging Checklist

Before spawning an investigation about lifecycle issues:

1. **Check kb:** `kb context "agent lifecycle"` or `kb context "completion"`
2. **Check this doc:** You're reading it
3. **Check beads:** `bd show <id>` - what's the actual status?
4. **Check recent post-mortems:** `.kb/post-mortems/`

If those don't answer your question, then investigate. But update this doc with what you learn.

---

## Related Documentation

- **Skill:** `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator policies
- **Guide:** `.kb/guides/completion-gates.md` - Completion verification gates
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Artifact types

---

## History

- **Jan 6, 2026:** Comprehensive update synthesizing 17 agent investigations. Added four-layer state model, display state computation, SSE handling, cross-project visibility, and UI layout patterns.
- **Jan 4, 2026:** Created after spending 1 hour debugging a problem that was already documented in kn. Synthesized from 20+ investigations about sessions/completion/lifecycle.
- **Jan 4, 2026:** Disabled repro verification and dependency check gates - they blocked completion without clear benefit.
