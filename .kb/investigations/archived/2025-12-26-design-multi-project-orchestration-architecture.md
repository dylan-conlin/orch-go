# Design: Multi-Project Orchestration Architecture

**Status:** Complete
**Phase:** Complete
**Created:** 2025-12-26

## TLDR

Multi-project orchestration should embrace the "global visibility, project-scoped operations" mental model. The OpenCode server is inherently global (port 4096); the dashboard shows all projects; individual commands (`orch status`, `orch abandon`) should filter by cwd and provide clear cross-project errors. This matches the existing direction already implemented for dashboard cross-project agent visibility.

## Design Question

How should multi-project orchestration work when running agents across multiple repos (e.g., orch-go + blog)? Should components filter by current project, show global visibility, or use a hybrid approach?

## Problem Framing

### Current State

The orchestration system has evolved with some multi-project awareness but without a coherent mental model:

**Global components (by design):**
- OpenCode server (port 4096) - shows all sessions across all projects
- Dashboard (`orch serve` on port 3348) - aggregates agents from all projects
- Focus (`~/.orch/focus.json`) - single north star priority across projects
- Accounts (`~/.orch/accounts.yaml`) - Claude Max accounts are user-wide

**Per-project components:**
- Beads database (`.beads/bd.sock`) - each project has its own issue tracking
- Workspace artifacts (`.orch/workspace/`) - agents create artifacts in their project
- Knowledge base (`.kb/`) - investigations and decisions are per-project
- SPAWN_CONTEXT.md contains `PROJECT_DIR` - agents know which project they work in

**The confusion:**
1. `orch status` shows sessions from ALL projects but uses cwd for beads lookups → cross-project agents show without phase/task info
2. `orch abandon` tries `bd show` in cwd, fails for cross-project beads IDs → error message is confusing
3. Dashboard shows all agents but "Waiting for activity..." for cross-project agents whose workspaces are in different directories
4. User mental model unclear: "Am I working in a project context or globally?"

### Success Criteria

1. **Clear mental model:** Orchestrators understand when they're seeing global vs project data
2. **Predictable behavior:** Commands behave consistently regardless of cwd
3. **Graceful cross-project:** Operations that need it work; operations that don't give clear errors
4. **No breaking changes:** Existing workflows continue to work

### Constraints

- **Beads is per-project:** Each project has its own `.beads/bd.sock` - this is fundamental
- **OpenCode is global:** Single server on port 4096 - changing this would be major architecture change
- **Workspaces are per-project:** Agents create artifacts in `{project}/.orch/workspace/`
- **Prior decision:** "Dual-mode architecture (tmux for visual, HTTP for programmatic) is the correct design"
- **Prior decision:** "Keep beads as external dependency with abstraction layer"

### Scope

**IN:** How commands/dashboard handle multi-project visibility
**OUT:** Changing fundamental per-project vs global architecture of components

## Exploration

### Approach 1: Full Project Isolation

**Mechanism:** Every command and UI scoped to current working directory.
- `orch status` only shows agents from cwd project
- Dashboard only shows cwd project agents
- Cross-project operations require `cd` or `--project` flag

**Pros:**
- Simple mental model: "I'm working in one project"
- No cross-project beads errors
- Clear boundaries

**Cons:**
- Breaks existing cross-project agent spawning (`--workdir`)
- Loses global swarm visibility (the dashboard's value)
- Forces orchestrators to switch directories constantly
- Against existing direction (dashboard already aggregates)

**Complexity:** High - requires major refactoring of dashboard aggregation

### Approach 2: Global Visibility Everywhere

**Mechanism:** Everything shows all projects, with project indicators.
- `orch status` shows all sessions with project column
- Dashboard shows all agents with project badges
- Cross-project operations automatically route to correct beads

**Pros:**
- Simple mental model: "I see everything"
- Swarm visibility preserved
- No "I can see it but can't act on it" frustration

**Cons:**
- Beads cross-project routing is complex (which `.beads/bd.sock`?)
- Noise when working focused on one project
- Harder to know "what's happening in MY project"

**Complexity:** Medium-High - requires cross-project beads routing for operations

### Approach 3: Hybrid - Global Visibility, Project-Scoped Operations ⭐

**Mechanism:** Visibility is global, but operations are project-scoped.
- `orch status` shows all sessions with project column (existing behavior)
- Dashboard shows all agents with project badges (existing behavior)
- Operations (`orch abandon`, `bd close`) require being in correct project or error with clear message
- Add `--project` flag for explicit cross-project operations

**Pros:**
- Clear mental model: "I see everything; I act on my project"
- Preserves swarm visibility
- Minimal changes to existing behavior
- Clear error messages guide users

**Cons:**
- Must be in correct directory for some operations
- Error messages must be helpful

**Complexity:** Low - mostly improving error messages and adding optional flags

### Approach 4: Orchestration Home Pattern

**Mechanism:** Designate a single "orchestration home" directory for all orchestration.
- All `orch` commands run from home (e.g., `~/orch-home/`)
- Home has registry of known projects
- All beads operations go to correct project automatically

**Pros:**
- Consistent location for orchestration
- Could solve cross-project beads routing

**Cons:**
- Breaking change to existing workflows
- Requires maintaining project registry
- Goes against current "work in project" pattern
- Already rejected in prior discussion (see SKILL.md "orch-go as orchestration home")

**Complexity:** High - fundamental workflow change

## Synthesis

### Recommendation: Approach 3 - Global Visibility, Project-Scoped Operations ⭐

**Why this approach:**

1. **Matches existing direction:** The dashboard already implements multi-project workspace aggregation (see `2025-12-26-inv-design-proper-cross-project-agent.md`). This extends that philosophy to CLI commands.

2. **Minimal disruption:** Current `orch status` already shows cross-project sessions. We're not changing behavior, just making it more explicit and improving error messages.

3. **Respects beads architecture:** Beads is per-project by design. Rather than fighting this, we make it explicit: "You can see this agent, but you must be in its project to act on it."

4. **Clear user mental model:** "I see the whole swarm. I act on my project."

5. **Guided by Session Amnesia principle:** Clear error messages with actionable guidance help the next orchestrator understand what to do.

### Trade-offs Accepted

- **Must cd for cross-project operations:** Users need to be in the correct project directory for operations like `orch abandon`. This is acceptable because:
  - Beads issue tracking is fundamentally per-project
  - Clear error message with suggested command mitigates friction
  - Cross-project operations are less common than same-project operations

- **Potential noise in status:** Showing all projects in `orch status` could be noisy. Mitigated by:
  - Adding `--project` / `-p` filter flag (already exists in status command)
  - Project column makes it clear which project each agent belongs to

### When This Would Change

- If beads becomes global (cross-project unified database) → could enable global operations
- If orchestration home pattern is adopted → operations would route automatically
- If per-project isolation becomes a requirement → would need Approach 1

### Implementation Specification

#### Error Message Improvements

**Current behavior (confusing):**
```
$ cd ~/blog && orch abandon orch-go-xyz1
Error: failed to get beads issue: issue not found
```

**Improved behavior:**
```
$ cd ~/blog && orch abandon orch-go-xyz1
Error: Issue orch-go-xyz1 not found in current project (blog).
This appears to be an issue from project 'orch-go'.

To abandon this agent, run:
  cd ~/Documents/personal/orch-go && orch abandon orch-go-xyz1
```

**Implementation:** In `runAbandon()` and similar commands:
1. When beads lookup fails, extract project name from beads ID (already have `extractProjectFromBeadsID`)
2. Check if cwd project matches beads ID project
3. If mismatch, provide helpful error with suggested command

#### Dashboard Enhancements

**Current behavior (partially working):**
- Multi-project agent visibility implemented via `extractUniqueProjectDirs()` and `buildMultiProjectWorkspaceCache()`
- Agents show correct status from cross-project workspaces

**Already done (from prior investigation):**
- Dashboard aggregates workspaces across all projects with active OpenCode sessions
- Phase/task info populates correctly for cross-project agents via `beadsProjectDirs` routing

**Remaining enhancement:**
- Add visual project grouping/filtering in UI (optional enhancement)
- Consider persistent project filter preference

#### CLI Command Enhancements

**orch status (minimal changes):**
- Already has `--project` / `-p` flag for filtering
- Already shows cross-project agents
- Add column header clarification or legend

**orch abandon (error message improvement):**
- Detect cross-project beads ID mismatch
- Provide actionable error message with correct cd command

**orch complete (error message improvement):**
- Same pattern as abandon
- Guide user to correct project directory

#### File Targets

| File | Action | Description |
|------|--------|-------------|
| `cmd/orch/main.go` | Modify | Improve `runAbandon()` error messages |
| `cmd/orch/complete.go` | Modify | Improve cross-project error messages |
| `cmd/orch/serve.go` | No change | Already has multi-project aggregation |
| `pkg/verify/check.go` | Modify (optional) | Add project-aware `GetIssueWithProject()` |

### Acceptance Criteria

- [ ] `orch abandon` for cross-project beads ID gives helpful error with suggested command
- [ ] `orch complete` for cross-project beads ID gives helpful error
- [ ] Dashboard continues to show all agents correctly (no regression)
- [ ] `orch status` continues to work (no regression)
- [ ] Error messages include project name and suggested command
- [ ] All existing tests pass

### Out of Scope

- Beads cross-project routing (would require beads architecture changes)
- Per-project `orch serve` instances (would fragment visibility)
- Automatic project switching (complex, error-prone)
- UI project selector in dashboard (future enhancement)

## Unexplored Questions

1. **Should `orch spawn --workdir` create beads issue in target project or current project?**
   - Currently: Creates in current project with `--workdir` target
   - Could argue either way; cross-project tracking has tradeoffs

2. **Should focus support project-aware goals?**
   - Current `orch focus` is global
   - Multi-project focus with priority ordering could be useful

3. **How does daemon handle cross-project issues?**
   - Daemon polls beads in its working directory
   - Cross-project agents spawned via daemon work because they create their own beads context

4. **Should we show project context in session titles?**
   - OpenCode session titles don't include project name
   - Could help with quick identification in global views

## References

**Files Examined:**
- `cmd/orch/serve.go:338-442` - `extractUniqueProjectDirs()`, `buildMultiProjectWorkspaceCache()`
- `cmd/orch/main.go:673-748` - `runAbandon()` current implementation
- `cmd/orch/main.go:2100-2360` - `runStatus()` cross-project handling
- `pkg/beads/client.go:75-106` - `FindSocketPath()` per-project socket discovery
- `pkg/verify/check.go:37-61` - `GetCommentsWithDir()` cross-project comments
- `pkg/focus/focus.go:77-80` - `DefaultPath()` global focus storage

**Related Investigations:**
- `2025-12-26-inv-design-proper-cross-project-agent.md` - Dashboard multi-project aggregation
- Prior SPAWN_CONTEXT template with PROJECT_DIR embedding

**Related Decisions:**
- "Dual-mode architecture (tmux for visual, HTTP for programmatic) is the correct design"
- "Keep beads as external dependency with abstraction layer"

**Principle Applied:**
- **Session Amnesia** - Error messages must be self-contained and actionable for the next orchestrator who has no memory of how the situation arose
- **Surfacing Over Browsing** - Dashboard surfaces all agents globally rather than requiring navigation to each project
