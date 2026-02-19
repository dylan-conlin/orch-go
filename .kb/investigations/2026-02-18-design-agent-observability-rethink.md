# Investigation: Strategic Redesign of Agent Observability Architecture

**Date:** 2026-02-18
**Status:** Active
**Investigator:** Claude Opus (og-arch-strategic-redesign-agent-18feb-0d2d)
**Issue:** orch-go-1058

---

## Context

Dylan spent all of Feb 18 firefighting observability bugs:
- price-watch agents reported as dead when they were running fine
- 84/85 agents showing empty metadata in dashboard
- Cross-repo visibility broken despite 6 weeks of fixes

The incremental fix approach has failed. The system has 5 interacting data sources that are reconciled at query time, and the reconciliation logic is where every bug lives.

---

## Current Architecture Analysis

### The Five Data Sources

| Source | What It Provides | Lifecycle | Authority Level |
|--------|-----------------|-----------|-----------------|
| **Workspace scanning** (.orch/workspace/) | SPAWN_CONTEXT.md, SYNTHESIS.md, beads_id, project_dir | Persistent | High (artifact record) |
| **Tmux window discovery** | Window existence, emoji-prefixed names with beads_id | Transient (until closed) | Low (UI only) |
| **OpenCode session API** | Session list, status (busy/idle), token counts, messages | Persistent (no TTL) | Medium (operational) |
| **Beads enrichment** | Phase comments, issue status (open/closed), task titles | Persistent | Highest (canonical) |
| **Agent registry / state.db** | Legacy, stale, not written to | Effectively dead | None |

### The Reconciliation Problem

The core issue is that `serve_agents.go` attempts to reconcile these 5 sources into a unified view, but:

1. **Each source has different lifecycle semantics** — OpenCode sessions persist indefinitely, tmux windows are transient, workspace files are permanent, beads is canonical
2. **No single source has complete truth** — Workspace knows project_dir, OpenCode knows if busy, beads knows if complete
3. **Performance optimizations break correctness** — Beads enrichment skipped for "stale" sessions, workspace scan dedup misses tmux agents
4. **Cross-project visibility requires querying multiple OpenCode directories** — The x-opencode-directory header scopes session listing

### Specific Bugs From Feb 18

1. **84/85 empty metadata:** OpenCode sessions return correct data, but the path from OpenCode → workspace cache → beads enrichment fails silently for most agents
2. **price-watch "dead" agents:** Cross-project agents spawn with `--workdir /other/project` but OpenCode's `--attach` mode uses server's cwd, so session.directory is wrong
3. **tmux agents invisible:** Workspace scan runs before tmux scan and claims agents as "completed" when SPAWN_CONTEXT.md exists

---

## Strategic Questions Answered

### Q1: Should there be ONE authoritative source of truth?

**Answer: No — but the reconciliation model is wrong.**

The current model attempts to merge 5 sources into one unified view. This is backwards. Instead:

- **State (beads + workspace files):** Canonical for work completion and agent identity
- **Infrastructure (OpenCode sessions, tmux windows):** Consulted for liveness only, never for identity or completion

**The key insight from the agent-lifecycle-state-model:** "The reconciliation burden comes from treating infrastructure as state." When we ask OpenCode "is this agent done?", we're asking the wrong source. OpenCode only knows if the session is busy/idle — it has no concept of work completion.

**Recommendation:** Invert the model. Start from beads issues (the work), not from sessions (the infrastructure).

### Q2: What's the minimal data model the dashboard needs?

**Core fields (must have):**
| Field | Source | Why |
|-------|--------|-----|
| `id` | beads_id from SPAWN_CONTEXT.md or session title | Unique agent identity |
| `status` | Priority Cascade (beads closed > Phase: Complete > workspace SYNTHESIS > session activity) | What to do next |
| `phase` | Beads comments | Where in work lifecycle |
| `project` | SPAWN_CONTEXT.md PROJECT_DIR | Which repo |
| `task` | Beads issue title | What work |
| `runtime` | Session created_at | How long |

**Secondary fields (nice to have):**
| Field | Source | Why |
|-------|--------|-----|
| `is_processing` | OpenCode SSE (session.status = busy) | Real-time feedback |
| `tokens` | OpenCode messages API | Resource usage |
| `skill` | Extracted from workspace name | Quick categorization |
| `synthesis` | SYNTHESIS.md content | Completion context |

**Fields to remove:**
- `window` — tmux window is UI, not state
- Registry-derived data — registry is dead

### Q3: Can we eliminate multi-source reconciliation entirely?

**Answer: No, but we can simplify it dramatically.**

The fundamental problem is that we discover agents from infrastructure (OpenCode sessions, tmux windows) and then try to figure out what work they represent. This is backwards.

**Proposed model: Work-centric discovery**

```
1. Start from BEADS issues with status=in_progress (the work that's happening)
2. For each issue, find the agent working on it:
   a. Check workspace for .orch/workspace/{name}/ matching beads_id
   b. If workspace exists, read SPAWN_CONTEXT.md for session_id
   c. Query OpenCode for that session_id (liveness only)
3. Agents without beads issues (--no-track) are a separate category
```

This inverts the current model where we start from sessions and work backwards to figure out what work they represent.

### Q4: How should cross-repo agents work?

**Current problem:** OpenCode `--attach` mode uses server's cwd for session.directory, not the CLI's `--workdir`. So a price-watch agent spawned from orch-go shows directory="/Users/dylanconlin/Documents/personal/orch-go".

**Root cause:** This is an OpenCode architectural decision (server owns session directory). We can't change it without upstream changes.

**Proposed solution: Workspace is the source of truth for project_dir**

1. When spawning with `--workdir`, write PROJECT_DIR to SPAWN_CONTEXT.md (already done)
2. For agent discovery, scan all registered kb projects' workspaces (already done)
3. Never use OpenCode session.directory for project — use workspace cache lookup

This is already partially implemented (`buildMultiProjectWorkspaceCache`, `lookupProjectDir`), but the implementation has bugs:
- `getKBProjects()` fails silently and returns empty slice
- Workspace cache TTL (30s) means stale data during high spawn activity
- Cross-project beads comments aren't fetched with correct project context

### Q5: What happens when the source of truth is unavailable?

**Graceful degradation hierarchy:**

| Source Unavailable | Fallback | Impact |
|-------------------|----------|--------|
| Beads daemon dead | CLI fallback (beads.FallbackStats) | Slower, still works |
| OpenCode server dead | Show workspace-based agents only | No liveness, no tokens |
| kb CLI unavailable | Single-project mode only | No cross-project visibility |
| Workspace missing | Agent invisible | Must check tmux directly |

**Recommendation:** Each degradation should be visible in the dashboard (a status indicator showing what's available).

---

## Proposed Architecture

### Design Principle: Own / Accept / Lobby

From the agent-lifecycle-state-model:

| Bucket | What | Implication |
|--------|------|-------------|
| **Own** | State layers (beads, workspaces), verification gates | Design, maintain, evolve |
| **Accept** | Infrastructure constraints (sessions persist, no TTL, x-opencode-directory) | Work within them |
| **Lobby** | Missing infrastructure features (session TTL, metadata API) | File upstream |

### Single Authoritative Path

**Phase 1: Work-centric agent discovery**

```go
// NEW: Start from work, find infrastructure
func discoverAgentsFromWork(projectDirs []string) []Agent {
    agents := []Agent{}
    
    // 1. Get all in_progress beads issues across projects
    for _, projectDir := range projectDirs {
        issues := beads.List(projectDir, status: "in_progress")
        for _, issue := range issues {
            agent := Agent{
                BeadsID:    issue.ID,
                Task:       issue.Title,
                ProjectDir: projectDir,
            }
            
            // 2. Find workspace for this issue
            workspace := findWorkspaceByBeadsID(projectDir, issue.ID)
            if workspace != nil {
                agent.SessionID = workspace.SessionID
                agent.SpawnedAt = workspace.SpawnTime
                agent.Skill = workspace.Skill
            }
            
            // 3. Get phase from beads comments
            agent.Phase = getPhaseFromComments(issue.ID)
            
            // 4. Check liveness from OpenCode (if session exists)
            if agent.SessionID != "" {
                agent.Status = getSessionLiveness(agent.SessionID)
            }
            
            agents = append(agents, agent)
        }
    }
    
    return agents
}
```

**Phase 2: Eliminate reverse reconciliation**

The current code builds agents from sessions, then tries to match them to beads issues. This is O(n*m) reconciliation with many edge cases. The new model:

1. Beads issues ARE the source of truth for "what work exists"
2. Workspaces ARE the source of truth for "which agent works on which issue"
3. OpenCode sessions ARE the source of truth for "is the agent alive right now"
4. Tmux windows ARE NOT a source of truth — they're a UI concern

**Phase 3: Remove dead code**

| To Delete | Why |
|-----------|-----|
| Registry / state.db | Not written to, 6 days stale |
| Tmux window scanning for agent discovery | Use for UI only, not for agent state |
| Multiple paths to the same data | Consolidate to single path |

### Data Model

```go
type Agent struct {
    // Identity (from beads + workspace)
    BeadsID    string `json:"beads_id"`           // Canonical identifier
    ProjectDir string `json:"project_dir"`        // From SPAWN_CONTEXT.md
    SessionID  string `json:"session_id,omitempty"` // OpenCode session if exists
    
    // Work state (from beads)
    Task       string `json:"task"`               // Issue title
    Phase      string `json:"phase"`              // From beads comments
    Status     string `json:"status"`             // Priority Cascade result
    
    // Infrastructure state (from OpenCode, for display only)
    IsProcessing bool   `json:"is_processing"`    // Currently generating
    Runtime      string `json:"runtime"`          // Since spawn
    Tokens       *TokenStats `json:"tokens,omitempty"`
    
    // Metadata (from workspace)
    Skill      string `json:"skill,omitempty"`
    SpawnedAt  string `json:"spawned_at,omitempty"`
}
```

### Migration Path

**Step 1 (Quick win): Fix cross-project visibility**
- Ensure `getKBProjects()` handles errors properly
- Add debug logging to workspace cache building
- Fix beads comments fetch to use correct project context

**Step 2 (Medium effort): Invert discovery model**
- New function `discoverAgentsFromWork()` that starts from beads
- Keep old code path for backward compatibility
- A/B test with feature flag

**Step 3 (Large effort): Simplify serve_agents.go**
- Remove reverse reconciliation code
- Delete registry/state.db references
- Consolidate to single code path

**Step 4 (Cleanup): Delete dead code**
- Remove tmux scanning from agent discovery
- Remove registry code
- Remove unused helper functions

### What Gets Deleted vs Kept

**Delete:**
- `~/.orch/registry.json` and all registry code in `pkg/session/`
- `~/.orch/state.db` and all SQLite code
- Tmux window scanning in `handleAgents()` (move to UI-only tmux.go)
- Multiple workspace scanning paths (consolidate to one)
- `investigationDirCache` for discovery (replaced by beads-centric model)

**Keep but refactor:**
- Workspace cache (`workspaceCache`) — still needed for project_dir lookup
- Beads cache (`beadsCache`) — still needed for performance
- OpenCode client — still needed for liveness checks
- Priority Cascade in `determineAgentStatus()` — core logic is sound

**Keep as-is:**
- Beads RPC/CLI integration
- SSE streaming for real-time updates
- Dashboard frontend (only backend changes)

---

## Constraints Discovered

1. **OpenCode --attach uses server cwd:** Cannot be changed without upstream PR. Must work around via workspace metadata.

2. **Sessions persist indefinitely:** OpenCode has no TTL. Cleanup is our responsibility.

3. **x-opencode-directory scopes all session operations:** Cross-project queries require explicit directory header.

4. **Beads comments are the only reliable phase source:** Agents must use `bd comment` protocol.

5. **Tmux windows are ephemeral:** Cannot be used for state, only for UI.

---

## Recommendation

**Invert the discovery model:** Start from beads issues (work) and find infrastructure, not vice versa. This eliminates the multi-source reconciliation that causes every bug.

**Priority order:**
1. Fix cross-project visibility bugs (immediate pain)
2. Implement work-centric discovery alongside existing code
3. Cut over once validated
4. Delete dead code

**Estimated effort:**
- Step 1: 2-4 hours
- Step 2: 1-2 days
- Step 3: 1 day
- Step 4: 2-4 hours

---

## Next Actions

1. **Spawn follow-up for Step 1:** Fix cross-project visibility by ensuring `getKBProjects()` works reliably
2. **Create decision record:** Document the "work-centric discovery" architectural decision
3. **Prototype Step 2:** Implement `discoverAgentsFromWork()` in parallel with existing code

---

## References

- `.kb/models/agent-lifecycle-state-model.md` — State vs infrastructure distinction
- `.orch/workspace/archived/og-arch-cross-project-agents-07jan-1844/SYNTHESIS.md` — kb projects integration
- `.orch/workspace/archived/og-inv-dashboard-blind-claude-17feb-24d6-213340/SYNTHESIS.md` — Scan ordering bug
- `.orch/workspace/archived/og-debug-fix-dashboard-blind-17feb-1b59/SYNTHESIS.md` — Fix that didn't stick
- `cmd/orch/serve_agents.go` — Current reconciliation mess (~1400 lines)
- `cmd/orch/serve_agents_cache.go` — Workspace cache implementation
