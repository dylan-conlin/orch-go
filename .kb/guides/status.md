# Orch Status Command

**Purpose:** Single authoritative reference for the `orch status` CLI command. Read this before debugging status issues, understanding agent detection, or interpreting output.

**Last verified:** 2026-01-29

**Synthesized from:** 15+ investigations (Dec 20, 2025 - Jan 29, 2026) addressing stale sessions, performance, liveness detection, cross-project visibility, session cleanup, drift metrics, frontier vs status disagreement, and stuck agent reporting.

---

## Overview

`orch status` shows swarm status including active/queued/completed agent counts, per-account usage percentages, and individual agent details.

```bash
orch status              # Show active agents only
orch status --all        # Include phantom and completed agents
orch status --project X  # Filter by project
orch status --json       # Output as JSON for scripting
```

---

## Orch Status vs Orch Frontier

**Two commands with distinct purposes:**

| Command | Purpose | Primary Data Source | View |
|---------|---------|---------------------|------|
| `orch status` | Monitor agent **execution state** | OpenCode sessions, tmux windows | Running agents, runtime, tokens, processing state |
| `orch frontier` | Monitor **work decidability state** | Beads issues, dependencies | Ready work, blocked work (with leverage), active/stuck agents |

**Key distinctions:**

- **orch status** answers: "What agents are running? How long? What resources are they using?"
- **orch frontier** answers: "What work is ready? What's blocking the most work? Are agents stuck?"

**When to use which:**

- **Use `orch status`** when debugging agent issues, checking resource usage, or monitoring swarm health
- **Use `orch frontier`** when deciding what to work on next, identifying high-leverage blockers, or detecting stuck agents

**Overlap:** Both show active agents, but with different perspectives:
- `orch status` shows agents from OpenCode/tmux perspective (execution infrastructure)
- `orch frontier` shows agents from beads perspective (work tracking), including stuck agent detection (>2h runtime)

**Known issues (as of 2026-01-29):**
- ~~Frontier and status could show different skills for same agent~~ (FIXED: aligned skill inference functions)
- ~~Frontier reported agents for closed issues as stuck~~ (FIXED: now filters by beads status)
- Different time windows: frontier uses 3h for active agents, status uses 30min for idle sessions

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         orch status                              │
│                                                                  │
│   1. Fetch OpenCode sessions (single API call)                  │
│   2. Enumerate tmux windows (primary source for "active")       │
│   3. Batch fetch beads comments (parallel goroutines)           │
│   4. Get token usage per session                                │
│   5. Assess context exhaustion risk                             │
│   6. Format output based on terminal width                      │
└─────────────────────────────────────────────────────────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                 ▼
       ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
       │  OpenCode   │   │   Beads     │   │    Tmux     │
       │  Sessions   │   │   Issues    │   │   Windows   │
       │ (liveness)  │   │  (phase)    │   │  (active)   │
       └─────────────┘   └─────────────┘   └─────────────┘
```

---

## Agent Status Determination

**Key insight:** Status is determined by combining three data sources with specific semantics.

### Data Sources

| Source | What It Tells Us | Reliability |
|--------|------------------|-------------|
| **Tmux windows** | Agent has terminal | High - if window exists, agent is running |
| **OpenCode sessions** | Agent has Claude session | Medium - sessions persist after completion |
| **Beads comments** | Agent reported phase | Authoritative for completion |

### Status Priority

```
1. IsCompleted = true  (beads issue closed)      → "completed"
2. IsPhantom = true    (beads open, no session)  → "phantom"
3. IsProcessing = true (session generating)      → "running"
4. Otherwise                                      → "idle"
```

### Agent Classification

| Classification | Meaning | Shown By Default? |
|----------------|---------|-------------------|
| **Active** | Running OpenCode session (tmux or headless) | Yes |
| **Running** | Actively generating response (finish=null) | Yes (subset of active) |
| **Idle** | Has session but not currently generating | Yes (subset of active) |
| **Phantom** | Beads issue open but no running session | No (use `--all`) |
| **Completed** | Beads issue closed | No (use `--all`) |

---

## Key Evolution & Fixes

The status command evolved through multiple investigations to handle:

### 1. Stale Session Problem (Dec 20-22)

**Problem:** `orch status` showed 27+ agents when only 1-2 were actually running.

**Root cause:** OpenCode's `/session` endpoint returns ALL sessions ever created for a directory when using `x-opencode-directory` header.

**Fix:** Call `ListSessions("")` (no directory header) to get only in-memory sessions, not historical disk sessions.

**Key insight:** OpenCode has a four-layer architecture:
1. In-memory session cache
2. Disk-persisted sessions
3. Orch registry
4. Tmux windows

Without coordinated cleanup, these can become out of sync.

### 2. Performance (Dec 23)

**Problem:** `orch status` took 11+ seconds.

**Root cause:** Sequential `bd show` and `bd comments` calls (O(N) subprocess overhead).

**Fix:** 
- Batch issue fetching with `bd list --status open --json`
- Parallel comment fetching using goroutines
- Result: 12.2s → ~1s (11x improvement)

### 3. Active Detection (Dec 23)

**Problem:** OpenCode sessions have no status field - can't tell if agent is working.

**Solution:** Check `/session/{id}/message` endpoint. Last assistant message with `finish: null` and `completed: 0` = actively generating.

**Key insight:** SSE busy/idle events are unreliable (false positives during normal operation).

### 4. Title Format (Dec 23)

**Problem:** Agents showed 0 active despite running sessions.

**Root cause:** `extractBeadsIDFromTitle()` expects `[beads-id]` pattern, but session titles were just workspace names.

**Fix:** Include beads ID in session title when spawning: `"workspace-name [beads-id]"`

### 5. Cross-Project Visibility (Jan 5)

**Problem:** Different phases shown depending on which directory you run `orch status` from.

**Root cause:** Beads comments lookup used current working directory, not the agent's actual project.

**Fix:** Three-strategy project directory resolution:
1. Use session.Directory (if valid, not "/")
2. Look up workspace from current project's `.orch/workspace/`
3. Derive from beads ID prefix (e.g., `orch-go-xxxx` → `~/Documents/personal/orch-go`)

### 6. Session Cleanup on Complete (Jan 6)

**Problem:** Completed agents appeared in `orch status --all` until 30-minute idle window expired.

**Root cause:** `orch complete` closes beads issue and tmux window but does NOT delete the OpenCode session. Session persists and is matched to closed beads ID.

**Evidence:** 136+ persisted OpenCode sessions found with orch-go beads IDs. `orch abandon` correctly deletes sessions (line 169), but `orch complete` did not.

**Fix:** Add `client.DeleteSession(sessionID)` to `complete_cmd.go` after tmux window cleanup, following the pattern from `abandon_cmd.go:165-174`.

**Key insight:** Agent state exists in 4 layers (OpenCode memory, OpenCode disk, registry, tmux). All layers must be cleaned on completion.

### 7. Session Drift Metrics (Jan 7)

**Problem:** Orchestrators needed visibility into session behavior for drift detection.

**Root cause:** No way to see how long the current session had been running or how many spawns occurred.

**Fix:** Added SESSION METRICS section to `orch status` output showing:
- Time in session (duration since session start)
- Last spawn time (time since most recent spawn)
- Spawn count (number of agents spawned in current session)

**Key insight:** Core drift signals (time, spawns) are derivable from existing `SpawnRecord` infrastructure. File reads tracking deferred - requires OpenCode plugin event infrastructure.

### 8. Observability Gaps (Jan 17)

**Problem:** MODE column (tmux vs headless) was omitted in narrow terminal format, reducing visibility into escape hatch usage.

**Root cause:** Narrow format (<120 chars) drops MODE to fit smaller terminals.

**Gaps identified:**
- Narrow format shows: SOURCE, BEADS ID, MODEL, STATUS, PHASE, SKILL, RUNTIME, TOKENS (no MODE)
- No `--mode` or `--backend` filter flag in `orch status`
- Escape hatch stats tracked in `orch stats` but not filterable in status view

**Evidence:** 
- Wide format shows MODE at position 3 (`status_cmd.go:1049`)
- Narrow format omits MODE column (`status_cmd.go:1132`)
- Stats show escape hatch usage (50% spawn rate in Jan 2026)

**Mitigation:** Use `orch status --json | jq '.agents[] | select(.mode=="claude")'` for filtering, or widen terminal.

**Related:** `.kb/investigations/2026-01-17-inv-analyze-orch-status-observability-gaps.md`

### 9. Frontier vs Status Disagreement (Jan 29)

**Problem:** `orch frontier` showed different skill than `orch status` for same agent.

**Root cause:** Two separate skill inference functions had conflicting mappings:
- `pkg/daemon/skill_inference.go` mapped bug→systematic-debugging
- `cmd/orch/spawn_skill_inference.go` mapped bug→architect (stale)

**Fix:** Aligned both functions to use bug→systematic-debugging per decision 2026-01-23.

**Key insight:** Workspace name is "ground truth" for skill used. Frontier extracts skill from workspace name, so it showed actual skill, while daemon printed its own (incorrect) inference.

**Related:** `.kb/investigations/2026-01-29-inv-orch-frontier-disagrees-orch-status.md`

### 10. Stuck Agent False Positives (Jan 29)

**Problem:** `orch frontier` reported agents for closed issues as "stuck" (e.g., `orch-go-20997` showing as stuck after 20h despite being closed).

**Root cause:** Frontier's stuck agent detection only checked OpenCode session age (>2h threshold), not beads issue status. Sessions persist after issues close.

**Fix:** Added filtering step in `getActiveAndStuckAgents()` to batch-check beads status via `bd show ... --json` and exclude agents whose issues are closed.

**Key insight:** Session lifetime ≠ issue lifetime. OpenCode sessions can outlive completed work. Stuck detection must validate issue context, not just session age.

**Related:** `.kb/investigations/2026-01-29-inv-orch-frontier-reports-stuck-agents.md`

---

## Output Columns

### Wide Format (>120 chars)

| Column | Source | Description |
|--------|--------|-------------|
| BEADS ID | Beads | Issue ID for tracking |
| STATUS | Computed | running/idle/phantom/completed |
| PHASE | Beads comments | Last reported phase (Planning/Implementing/Complete) |
| TASK | Beads issue | Issue title (truncated) |
| SKILL | Workspace name | Skill used (feat, inv, debug, etc.) |
| RUNTIME | OpenCode session | Time since spawn |
| TOKENS | OpenCode session | Total (input/output) |
| RISK | Computed | Context exhaustion warning (if applicable) |

### Narrow Format (80-100 chars)

Drops TASK column, abbreviates SKILL.

### Card Format (<80 chars)

Multi-line blocks per agent for very narrow terminals.

---

## Common Problems

### "Status shows more agents than expected"

**Likely cause:** Stale OpenCode sessions in memory.

**Check:** `curl -s http://localhost:4096/session | jq 'length'` vs `orch status --json | jq '.agents | length'`

**Fix:** The 30-minute idle filter should handle this. If sessions persist, restart OpenCode server.

### "Agent shows wrong phase"

**Likely cause:** Cross-project visibility issue (comments looked up in wrong project).

**Check:** Run from agent's project directory and compare output.

**Fix:** The three-strategy project directory resolution should handle this. If persists, check that beads ID prefix matches project name convention.

### "Status is slow"

**Likely cause:** Regression in batch fetching.

**Check:** Time breakdown:
```bash
time curl -s http://localhost:4096/session  # Should be <50ms
time bd list --status open --json           # Should be <200ms
time orch status                            # Should be <2s
```

**Fix:** Ensure batch functions are being used (not sequential `bd show` per agent).

### "Agent shows as phantom but is running"

**Likely cause:** Session title doesn't have beads ID in brackets.

**Check:** `curl -s http://localhost:4096/session | jq '.[].title'` - look for `[beads-id]` pattern.

**Fix:** Respawn agent - new spawns include beads ID in title.

### "Active count doesn't match visible agents"

**Likely cause:** Filtering. Active count is computed before `--all` and `--project` filters.

**Expected:** The SWARM STATUS line shows totals, then filtered agents are displayed below.

---

## Constraints & Decisions

These are settled. Don't re-investigate:

- **Tmux windows are primary source for "active"** - If window exists, agent is running
- **30-minute idle threshold for OpenCode sessions** - Filters completed-but-cached sessions
- **Beads comments are authoritative for phase** - Not session activity time
- **Session titles must include `[beads-id]` for matching** - Pattern established for tmux windows
- **Batch/parallel beads fetching required** - Sequential calls cause O(N) slowdown
- **Cross-project needs three-strategy lookup** - Session.Directory is "/" for spawned agents

---

## JSON Output Schema

```json
{
  "swarm": {
    "active": 3,
    "processing": 1,
    "idle": 2,
    "phantom": 0,
    "queued": 0,
    "completed_today": 5
  },
  "accounts": [
    {
      "name": "current",
      "email": "...",
      "used_percent": 45.2,
      "reset_time": "2d 5h",
      "is_active": true
    }
  ],
  "orchestrator_sessions": [...],
  "agents": [
    {
      "session_id": "ses_...",
      "beads_id": "orch-go-xxxx",
      "skill": "feature-impl",
      "runtime": "1h23m",
      "phase": "Implementing",
      "task": "Add feature X",
      "project": "orch-go",
      "is_phantom": false,
      "is_processing": true,
      "is_completed": false,
      "tokens": {
        "input_tokens": 8500,
        "output_tokens": 4200,
        "total_tokens": 12700
      },
      "context_risk": null
    }
  ]
}
```

---

## Related Resources

- **Status determination for dashboard:** `.kb/guides/status-dashboard.md`
- **Dashboard architecture:** `.kb/guides/dashboard.md`
- **Agent lifecycle:** `.kb/guides/agent-lifecycle.md`
- **Spawn command:** `.kb/guides/spawn.md`

---

## Source Investigations

| Date | Investigation | Key Contribution |
|------|---------------|------------------|
| 2025-12-20 | `inv-enhance-status-command-swarm-progress.md` | Added swarm metrics, account usage, --json flag |
| 2025-12-21 | `inv-investigate-orch-status-showing-stale.md` | Identified four-layer architecture |
| 2025-12-21 | `inv-orch-status-showing-stale-sessions.md` | Fixed x-opencode-directory header issue |
| 2025-12-22 | `debug-orch-status-stale-sessions.md` | Activity-based liveness (30 min threshold) |
| 2025-12-22 | `inv-update-orch-status-use-islive.md` | Identified state.GetLiveness() API |
| 2025-12-23 | `inv-orch-status-can-detect-active.md` | Messages endpoint for processing detection |
| 2025-12-23 | `inv-orch-status-shows-active-agents.md` | Session title format fix (`[beads-id]`) |
| 2025-12-23 | `inv-orch-status-takes-11-seconds.md` | Batch/parallel beads fetching |
| 2025-12-24 | `inv-fix-status-filter-test-expects.md` | Test synchronization (already fixed) |
| 2026-01-05 | `debug-fix-orch-status-showing-different.md` | Cross-project directory resolution |
| 2026-01-06 | `inv-orch-status-shows-completed-agents.md` | Session cleanup in orch complete |
| 2026-01-07 | `inv-orch-status-surface-drift-metrics.md` | Session drift metrics display |
| 2026-01-17 | `inv-analyze-orch-status-observability-gaps.md` | MODE column omission, filter gaps |
| 2026-01-29 | `inv-orch-frontier-disagrees-orch-status.md` | Skill inference alignment |
| 2026-01-29 | `inv-orch-frontier-reports-stuck-agents.md` | Stuck agent false positives fix |

---

## Code Reference

| File | Purpose |
|------|---------|
| `cmd/orch/status_cmd.go` | Main status command implementation |
| `pkg/opencode/client.go` | ListSessions, IsSessionProcessing |
| `pkg/verify/check.go` | GetCommentsBatch, ParsePhaseFromComments |
| `pkg/tmux/tmux.go` | ListWorkersSessions, ListWindows |
