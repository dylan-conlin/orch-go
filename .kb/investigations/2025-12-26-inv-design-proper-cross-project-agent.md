## Summary (D.E.K.N.)

**Delta:** Cross-project agent visibility requires PROJECT_DIR-aware aggregation across multiple project workspaces.

**Evidence:** Analysis of serve.go shows current buildWorkspaceCache only scans current project's .orch/workspace/; agents spawned via `--workdir` have workspaces in the target project, not the orchestration home.

**Knowledge:** The dashboard needs to aggregate workspaces across ALL known projects to show cross-project agents correctly. OpenCode sessions are global (port 4096) but workspaces and beads issues are per-project.

**Next:** Implement multi-project workspace aggregation in serve.go, with registry of known project directories.

**Confidence:** High (85%) - Clear understanding of the problem; solution requires careful design to avoid performance issues.

---

# Investigation: Design Proper Cross-Project Agent Visibility for Dashboard

**Question:** How should the dashboard aggregate agent data across multiple projects to show agents spawned with `--workdir` correctly?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current Architecture is Single-Project

**Evidence:** 
- `buildWorkspaceCache()` in serve.go:276-348 only scans `projectDir/.orch/workspace/`
- `projectDir` is set from `sourceDir` (build-time) or `os.Getwd()` at serve.go:368-370
- OpenCode sessions are listed globally (port 4096, no directory filter at serve.go:385)
- When an agent is spawned with `--workdir /other/project`, its workspace goes to `/other/project/.orch/workspace/`
- But the dashboard running from orch-go only sees orch-go's workspaces

**Source:** 
- cmd/orch/serve.go:276-385
- pkg/spawn/context.go:271-278 (workspace creation uses cfg.ProjectDir)

**Significance:** This is the root cause of "Waiting for activity..." - the dashboard can't find workspace metadata for cross-project agents.

---

### Finding 2: PROJECT_DIR is Already Tracked in SPAWN_CONTEXT.md

**Evidence:**
- SPAWN_CONTEXT.md template includes `PROJECT_DIR: {{.ProjectDir}}` at context.go:59
- `extractProjectDirFromWorkspace()` already exists in review.go:240-261
- `buildWorkspaceCache()` already extracts PROJECT_DIR into `beadsToProjectDir` map (serve.go:332-334)
- `GetCommentsBatchWithProjectDirs()` in verify/check.go:741-784 already handles cross-project beads queries

**Source:**
- pkg/spawn/context.go:59
- cmd/orch/review.go:240-261
- cmd/orch/serve.go:332-334
- pkg/verify/check.go:741-784

**Significance:** The infrastructure for cross-project awareness exists but only works when the workspace is in the current project. For true cross-project visibility, we need to discover workspaces across projects.

---

### Finding 3: OpenCode Sessions Contain Directory Information

**Evidence:**
- OpenCode Session struct has `Directory string` field (opencode/types.go)
- `ListSessions("")` returns all sessions regardless of directory (serve.go:385)
- Each session's Directory field tells us where it was created
- This could be used as an index to discover which projects have active agents

**Source:**
- pkg/opencode/types.go (Session struct)
- cmd/orch/serve.go:385

**Significance:** We can use OpenCode session directories to dynamically discover which projects have active agents, avoiding the need for a static project registry.

---

### Finding 4: Beads Issues Are Per-Project

**Evidence:**
- Beads socket is at `{project}/.beads/bd.sock`
- `FindSocketPath()` walks up directory tree looking for `.beads/bd.sock` (beads/client.go:66-94)
- `GetCommentsBatchWithProjectDirs()` groups beads queries by project directory for efficient RPC client reuse
- Each project has its own beads database, issues don't cross project boundaries

**Source:**
- pkg/beads/client.go:66-94
- pkg/verify/check.go:741-784

**Significance:** Cross-project agent visibility requires querying beads from multiple project directories, not just the current one.

---

## Synthesis

**Key Insights:**

1. **Dynamic Project Discovery** - Rather than maintaining a static registry of projects, we can discover active projects from OpenCode session directories. This is more robust and self-healing.

2. **Multi-Project Workspace Aggregation** - The dashboard needs to scan `.orch/workspace/` directories across ALL projects with active sessions, not just the current project.

3. **Beads Query Routing** - The existing `GetCommentsBatchWithProjectDirs()` pattern handles cross-project beads queries efficiently. The missing piece is populating the projectDirs map from multiple workspace directories.

4. **Cache Locality** - Building separate workspace caches per project-directory keeps the architecture clean and allows parallel scanning.

**Answer to Investigation Question:**

The dashboard should aggregate agent data across multiple projects by:
1. Using OpenCode session directories to discover which projects have active agents
2. Building workspace caches for each unique project directory
3. Merging workspace metadata across all project caches
4. Using the merged beadsToProjectDir map for cross-project beads queries

This approach is dynamic (discovers projects from active sessions), efficient (parallel workspace scanning), and maintains the existing cross-project beads infrastructure.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The analysis shows a clear understanding of the problem and the existing infrastructure. The solution builds on proven patterns already in the codebase.

**What's certain:**

- ✅ OpenCode sessions contain project directory information
- ✅ PROJECT_DIR is already embedded in SPAWN_CONTEXT.md
- ✅ Cross-project beads queries work via GetCommentsBatchWithProjectDirs
- ✅ The issue is that buildWorkspaceCache only scans one project

**What's uncertain:**

- ⚠️ Performance impact of scanning multiple project workspace directories
- ⚠️ Whether all projects have `.orch/workspace/` (only those with spawned agents)
- ⚠️ Race conditions between workspace creation and dashboard polling

**What would increase confidence to Very High (95%+):**

- Benchmarking multi-project workspace scanning (should be negligible with ~5 projects)
- Testing with actual cross-project spawns
- Handling edge cases (project directory no longer exists, permissions issues)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Multi-Project Workspace Aggregation via Session Directory Discovery**

Build workspace caches from ALL unique project directories discovered in OpenCode sessions, then merge them for a unified view.

**Why this approach:**
- Dynamic discovery means no static project registry to maintain
- Reuses existing buildWorkspaceCache architecture
- Scales naturally with number of active projects
- Self-healing when projects are added/removed

**Trade-offs accepted:**
- Slight increase in workspace scanning (now scanning N project directories instead of 1)
- Session list fetch required before workspace scan (already happening)

**Implementation sequence:**

1. **Extract unique project directories from OpenCode sessions** - Collect `session.Directory` from all sessions
2. **Build workspace caches for each unique project directory** - Parallel scanning with goroutines
3. **Merge workspace metadata into unified maps** - Combine beadsToWorkspace and beadsToProjectDir
4. **Pass merged maps to existing cross-project beads infrastructure** - No changes needed

### Alternative Approaches Considered

**Option B: Static Project Registry**
- **Pros:** Simpler implementation, predictable behavior
- **Cons:** Requires manual maintenance, stale registry possible, doesn't scale with ad-hoc projects
- **When to use instead:** If dynamic discovery proves too slow

**Option C: Central Workspace Directory**
- **Pros:** Single scan location, simpler architecture
- **Cons:** Major breaking change, requires migration, violates per-project isolation
- **When to use instead:** Never - this is a fundamental architecture change

**Rationale for recommendation:** Option A (dynamic discovery) requires minimal code changes, builds on existing infrastructure, and is self-maintaining. Options B and C have significant downsides.

---

### Implementation Details

**What to implement first:**
1. Helper function to extract unique project directories from sessions
2. Multi-project workspace cache aggregation in `handleAgents()`
3. Update workspace lookups to use merged cache

**Things to watch out for:**
- ⚠️ Parallel workspace scanning must be safe (no shared mutable state)
- ⚠️ Some session directories may not exist (session from terminated project)
- ⚠️ Dashboard must handle missing workspaces gracefully (agent spawned but workspace not yet created)

**Areas needing further investigation:**
- Performance benchmarking with real multi-project workloads
- Whether to add project filtering API parameter
- Caching across requests (currently rebuilds cache per request)

**Success criteria:**
- ✅ Agents spawned with `--workdir` to different projects show correct status in dashboard
- ✅ Phase information from beads comments loads correctly for cross-project agents
- ✅ No performance regression (dashboard response time stays under 500ms)
- ✅ All existing tests continue to pass

---

## API Shape (Design Specification)

### No API Changes Required

The existing `/api/agents` endpoint returns `AgentAPIResponse` structs with a `Project` field. Cross-project agents already have this field populated via `extractProjectFromBeadsID()`. The change is internal to how we populate the agent data.

### Internal Changes

**New Functions:**

```go
// extractUniqueProjectDirs collects unique project directories from OpenCode sessions.
// Returns a deduplicated slice of directory paths that have active agents.
func extractUniqueProjectDirs(sessions []opencode.Session) []string

// buildMultiProjectWorkspaceCache builds workspace caches for multiple project directories
// and merges them into a unified cache. Scans in parallel for performance.
func buildMultiProjectWorkspaceCache(projectDirs []string) *workspaceCache
```

**Modified Functions:**

```go
// handleAgents - add multi-project workspace aggregation
// Before: wsCache := buildWorkspaceCache(projectDir)
// After:  projectDirs := extractUniqueProjectDirs(sessions)
//         wsCache := buildMultiProjectWorkspaceCache(projectDirs)
```

### Data Flow

```
Request: GET /api/agents
    ↓
1. List OpenCode sessions (global, port 4096)
    ↓
2. Extract unique project directories from session.Directory fields
    ↓
3. For each project directory (parallel):
   - Scan {project}/.orch/workspace/
   - Extract beadsID and PROJECT_DIR from SPAWN_CONTEXT.md
    ↓
4. Merge all workspace metadata into unified cache
    ↓
5. Use merged cache for:
   - beadsToWorkspace lookups (find workspace path)
   - beadsToProjectDir lookups (route beads queries)
    ↓
6. Build AgentAPIResponse list with correct cross-project data
```

---

## References

**Files Examined:**
- cmd/orch/serve.go - Dashboard API server, handleAgents, buildWorkspaceCache
- cmd/orch/main.go:2150-2300 - orch status cross-project handling
- cmd/orch/review.go - extractProjectDirFromWorkspace implementation
- pkg/spawn/context.go - SPAWN_CONTEXT.md template with PROJECT_DIR
- pkg/verify/check.go - GetCommentsBatchWithProjectDirs cross-project beads
- pkg/beads/client.go - FindSocketPath per-project discovery
- pkg/opencode/client.go - Session listing and types

**Related Artifacts:**
- **Decision:** .kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md - Beads is external dependency
- **Prior Knowledge:** Skills own domain behavior, spawn owns orchestration infrastructure

---

## Investigation History

**2025-12-26 12:00:** Investigation started
- Initial question: How to show cross-project agents in dashboard correctly?
- Context: Agents spawned with --workdir show "Waiting for activity..."

**2025-12-26 12:30:** Exploration phase complete
- Analyzed serve.go, main.go, review.go for cross-project patterns
- Identified root cause: single-project workspace scanning

**2025-12-26 13:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Design for multi-project workspace aggregation via session directory discovery
