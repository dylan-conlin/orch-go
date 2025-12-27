<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-project completion breaks because beads issues are per-repo; the recommended solution is auto-detecting project from workspace metadata with `--workdir` as fallback.

**Evidence:** Code analysis shows SPAWN_CONTEXT.md already stores PROJECT_DIR; current complete uses current directory for beads lookup; orch abandon already has `--workdir` pattern.

**Knowledge:** Beads issues are per-repo (socket at `.beads/bd.sock`); workspace metadata is the single source of truth for agent-to-project mapping; consistency with existing `--workdir` pattern in spawn/abandon is important.

**Next:** Implement Option A: auto-detect from workspace metadata, fallback to `--workdir` flag. See Implementation Recommendations section.

---

# Investigation: Cross-Project Completion UX Design

**Question:** How should `orch complete` handle agents spawned in other repos (glass, kb-cli, etc) when the orchestrator is running from orch-go?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Beads Issues Are Per-Repo by Design

**Evidence:** 
- Beads daemon socket is at `.beads/bd.sock` in each project root
- `beads.FindSocketPath(dir)` walks up from the given directory to find `.beads/bd.sock`
- All beads operations (Show, Comments, Close) are routed through this socket
- When orchestrator runs `orch complete` from orch-go, it can only access orch-go's beads database

**Source:** 
- `pkg/beads/client.go:78-106` - `FindSocketPath()` implementation
- `pkg/verify/check.go:47-61` - `GetCommentsWithDir()` using FindSocketPath

**Significance:** This is the root cause of the friction - beads is intentionally per-project for data isolation. We cannot change beads to be global; we must teach orch commands to reach into other projects.

---

### Finding 2: Workspace Metadata Already Stores Project Directory

**Evidence:**
- `SPAWN_CONTEXT.md` contains `PROJECT_DIR: /absolute/path/to/project`
- This is written by `pkg/spawn/context.go:76` in the template
- Function `extractProjectDirFromWorkspace()` already exists in `cmd/orch/review.go:258-279` to extract this

**Source:**
- `pkg/spawn/context.go:76` - `PROJECT_DIR: {{.ProjectDir}}` in template
- `cmd/orch/review.go:258-279` - `extractProjectDirFromWorkspace()` implementation

**Significance:** We don't need to add new metadata - the project directory is already stored. We just need to use it in the completion workflow.

---

### Finding 3: Existing Pattern from `orch abandon --workdir`

**Evidence:**
- `orch abandon` already supports `--workdir` flag for cross-project abandonment
- Sets `beads.DefaultDir` to the target project directory
- Has helpful error messages suggesting `--workdir` when project mismatch detected

**Source:**
- `cmd/orch/main.go:654-685` - abandon command with `--workdir` flag
- `cmd/orch/main.go:687-843` - `runAbandon()` implementation

**Significance:** There's already a precedent for cross-project operations. The `--workdir` pattern is proven and understood. This should be extended to `orch complete`.

---

### Finding 4: Current Complete Has Cross-Project Detection But No Resolution

**Evidence:**
- `runComplete()` tries to get issue and provides helpful error on mismatch:
  ```
  Hint: The issue ID suggests it belongs to project 'glass', but you're in 'orch-go'.
  Try: cd ~/path/to/glass && orch complete glass-xyz
  ```
- But it doesn't auto-detect from workspace metadata or offer `--workdir`

**Source:**
- `cmd/orch/main.go:2948-2968` - runComplete with cross-project error hint

**Significance:** The problem is recognized but not solved - user must manually cd. The metadata exists to auto-resolve.

---

### Finding 5: Review Command Already Handles Cross-Project Comments

**Evidence:**
- `GetCommentsBatchWithProjectDirs()` accepts a map of beadsID → projectDir
- Uses different beads sockets for different projects
- `extractProjectDirFromWorkspace()` extracts PROJECT_DIR from SPAWN_CONTEXT.md

**Source:**
- `pkg/verify/check.go:732-781` - `GetCommentsBatchWithProjectDirs()`
- `cmd/orch/serve.go:572-574` - comment about cross-project agent visibility

**Significance:** The infrastructure for cross-project beads access already exists. We can reuse this pattern.

---

## Synthesis

**Key Insights:**

1. **The data is there** - PROJECT_DIR is already in SPAWN_CONTEXT.md for every agent. We just need to read it and use it.

2. **The pattern is established** - `orch abandon --workdir` already solves this problem. Extending to `orch complete` provides consistency.

3. **Auto-detection is better UX** - Rather than requiring `--workdir` every time, we can auto-detect from workspace metadata and only require `--workdir` as a fallback.

**Answer to Investigation Question:**

The solution is a hybrid approach: **auto-detect project from workspace metadata, fallback to `--workdir` flag**.

When running `orch complete glass-xyz`:
1. Find workspace by beadsID in current project's `.orch/workspace/`
2. Extract `PROJECT_DIR` from `SPAWN_CONTEXT.md`
3. Use that directory for beads operations (set `beads.DefaultDir`)
4. If workspace not found, check if `--workdir` was provided
5. If neither, show helpful error with suggestion

This is superior to the alternatives (always requiring `--workdir`, or a new cross-repo query syntax) because:
- Zero friction for the happy path (auto-detection works)
- Consistent with existing `orch abandon --workdir` pattern
- No changes to beads CLI required
- Workspace metadata already exists

---

## Structured Uncertainty

**What's tested:**

- ✅ SPAWN_CONTEXT.md contains PROJECT_DIR (verified: examined existing workspaces)
- ✅ `extractProjectDirFromWorkspace()` correctly parses the path (verified: code review)
- ✅ beads client supports `WithCwd()` option (verified: `pkg/beads/client.go:47-51`)
- ✅ `orch abandon --workdir` pattern works (verified: code review and documentation)

**What's untested:**

- ⚠️ Performance impact of reading SPAWN_CONTEXT.md for every complete (likely negligible - single file read)
- ⚠️ Edge case: workspace exists in multiple projects (should use most recent, not tested)
- ⚠️ Cross-project tmux window cleanup (currently searches all sessions, may still work)

**What would change this:**

- If workspace metadata can't be trusted (e.g., orphaned workspaces from deleted projects)
- If beads daemon socket permissions prevent cross-project access
- If orchestrator frequently runs from directories with no project context

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Option A: Auto-detect from workspace metadata, `--workdir` as fallback**

**Why this approach:**
- Zero friction for the common case (workspace exists, PROJECT_DIR is valid)
- Consistent with existing `orch abandon --workdir` pattern
- Uses existing metadata - no new data structures needed
- Explicit fallback for edge cases or manual override

**Trade-offs accepted:**
- Adds a file read per complete (SPAWN_CONTEXT.md) - negligible cost
- Depends on workspace not being cleaned up before completion

**Implementation sequence:**
1. Add `--workdir` flag to `orch complete` (copy from abandon)
2. In `runComplete()`, after finding workspace, extract PROJECT_DIR
3. If PROJECT_DIR differs from cwd, set `beads.DefaultDir` 
4. All beads operations use the resolved project directory
5. Update helpful error message to suggest `--workdir` as alternative to cd

### Alternative Approaches Considered

**Option B: Cross-repo beads query syntax (`bd show glass:glass-xyz`)**
- **Pros:** Universal solution for any beads command
- **Cons:** Requires changes to beads CLI; large scope; beads is a separate project
- **When to use instead:** If we want to solve this at the beads level for all tools

**Option C: Always require `--workdir` for cross-project completion**
- **Pros:** Explicit, no magic, simple implementation
- **Cons:** Friction every time; orchestrator must know paths; error-prone
- **When to use instead:** If auto-detection proves unreliable

**Option D: Global beads database / registry**
- **Pros:** Single source of truth for all issues
- **Cons:** Violates beads' per-project isolation design; major architectural change
- **When to use instead:** Never - this fundamentally changes beads' model

**Rationale for recommendation:** Option A provides the best UX (auto-detection) while maintaining full compatibility with existing patterns and requiring minimal changes. The workspace metadata already exists and is authoritative.

---

### Implementation Details

**What to implement first:**
1. Add `completeWorkdir` flag to complete command (like `abandonWorkdir`)
2. Modify `runComplete()` to:
   - Call `findWorkspaceByBeadsID()` to locate workspace
   - If found, call `extractProjectDirFromWorkspace()` to get PROJECT_DIR
   - If PROJECT_DIR differs from cwd, set `beads.DefaultDir`
3. Update error messages to mention `--workdir` option

**File targets:**
- `cmd/orch/main.go` - Add flag, modify runComplete()
- `cmd/orch/review.go` - May need to export/reuse `extractProjectDirFromWorkspace()`

**Things to watch out for:**
- ⚠️ Ensure `beads.DefaultDir` is set before any beads operations (GetIssue, GetPhaseStatus, CloseIssue)
- ⚠️ Cross-project tmux window cleanup should still work (searches all sessions already)
- ⚠️ Don't break the auto-rebuild logic (uses cwd, not beads project dir)

**Areas needing further investigation:**
- What happens if workspace is in project A but user provides `--workdir` pointing to project B?
- Should we add `--workdir` to `orch review` as well? (May already work via `extractProjectDirFromWorkspace`)

**Success criteria:**
- ✅ `orch complete glass-xyz` from orch-go auto-detects and completes successfully
- ✅ `orch complete unknown-xyz --workdir ~/glass` works as explicit override
- ✅ Error message when issue not found suggests `--workdir` option
- ✅ Existing single-project completion still works unchanged

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Complete and abandon command implementations
- `cmd/orch/review.go` - extractProjectDirFromWorkspace function
- `pkg/beads/client.go` - Beads client with FindSocketPath
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template with PROJECT_DIR
- `pkg/verify/check.go` - GetCommentsWithDir, GetCommentsBatchWithProjectDirs

**Commands Run:**
```bash
# Examined existing workspace metadata
cat .orch/workspace/*/SPAWN_CONTEXT.md | grep PROJECT_DIR
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md` - Beads is external dependency
- **Investigation:** None directly related

---

## Investigation History

**2025-12-27 08:00:** Investigation started
- Initial question: How to handle cross-project completion UX
- Context: Spawning agents with `--workdir` creates beads issues in other repos

**2025-12-27 08:30:** Exploration phase
- Analyzed beads per-repo architecture
- Found existing PROJECT_DIR metadata
- Identified `orch abandon --workdir` as precedent

**2025-12-27 09:00:** Investigation completed
- Status: Complete
- Key outcome: Recommend auto-detect from workspace metadata with `--workdir` fallback
