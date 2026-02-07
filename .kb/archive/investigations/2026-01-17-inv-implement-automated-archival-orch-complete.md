## Summary (D.E.K.N.)

**Delta:** Automated workspace archival was successfully added to `orch complete` - workspaces are now moved to `archived/` immediately after successful completion.

**Evidence:** All 8 archival-related tests pass, including collision handling and registry updates.

**Knowledge:** The archival gap identified in the Workspace Lifecycle Model is now closed. The `--no-archive` flag provides opt-out for cases where immediate archival isn't desired.

**Next:** Close issue - implementation complete.

**Promote to Decision:** recommend-no (tactical implementation of existing architectural model)

---

# Investigation: Implement Automated Archival in orch complete

**Question:** How should automated archival be implemented in `orch complete` to close the archival gap identified in the Workspace Lifecycle Model?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing archival infrastructure in clean_cmd.go

**Evidence:** The `archiveStaleWorkspaces()` function already handles moving workspaces to `archived/` directory, including name collision handling with timestamp suffixes.

**Source:** `cmd/orch/clean_cmd.go:866-1034`

**Significance:** The archival logic already exists for `orch clean --stale`, so the implementation for `orch complete` could reuse similar patterns rather than inventing new ones.

---

### Finding 2: Session registry tracks orchestrator sessions

**Evidence:** The `OrchestratorSession` struct in `pkg/session/registry.go` tracks workspace name, project directory, and status. It provides an `Update()` method for modifying session state.

**Source:** `pkg/session/registry.go:29-52`, `pkg/session/registry.go:205-220`

**Significance:** The registry can be extended with an `ArchivedPath` field to track where completed workspaces are archived, satisfying the requirement to "update session registry to reflect the new location."

---

### Finding 3: Completion flow requires careful ordering

**Evidence:** The `runComplete()` function has a specific order: (1) close beads issue, (2) delete OpenCode session, (3) export transcript, (4) clean up tmux window. Several steps read from the workspace directory.

**Source:** `cmd/orch/complete_cmd.go:884-965`

**Significance:** Archival must happen AFTER all workspace reads (OpenCode session deletion reads `.session_id`, transcript export writes to workspace) but BEFORE cleanup. This ensures no data loss.

---

## Synthesis

**Key Insights:**

1. **Reuse existing patterns** - The archival implementation mirrors `archiveStaleWorkspaces()` from clean_cmd.go: create archived directory, handle name collisions with timestamp suffix, move workspace with `os.Rename()`.

2. **Registry update is optional but valuable** - Adding `ArchivedPath` to the session registry provides a clear audit trail of where orchestrator workspaces ended up after completion.

3. **Opt-out via --no-archive** - Some workflows may need to keep the workspace in place for immediate inspection. The flag provides flexibility without changing the default behavior.

**Answer to Investigation Question:**

Automated archival is implemented by:
1. Adding `archiveWorkspace()` helper function that handles directory creation, collision handling, and workspace move
2. Calling it after OpenCode session deletion and transcript export in `runComplete()`
3. Adding `--no-archive` flag for opt-out
4. Extending `OrchestratorSession` struct with `ArchivedPath` field and updating it post-archival

---

## Structured Uncertainty

**What's tested:**

- ✅ archiveWorkspace moves workspace to archived/ (TestArchiveWorkspace)
- ✅ Empty path returns error (TestArchiveWorkspaceEmptyPath)
- ✅ Non-existent workspace returns error (TestArchiveWorkspaceNonExistent)
- ✅ Name collisions handled with timestamp suffix (TestArchiveWorkspaceNameCollision)
- ✅ Registry ArchivedPath field can be updated (TestRegistryArchivedPathUpdate)

**What's untested:**

- ⚠️ End-to-end `orch complete` with actual beads issue (requires integration test setup)
- ⚠️ Cross-filesystem archival (if workspace and archived/ are on different mounts)

**What would change this:**

- Finding would be wrong if `os.Rename()` fails silently instead of returning error
- Finding would be wrong if registry file locking doesn't work under concurrent access

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Immediate archival after completion** - Archive workspace as the final step of `orch complete` after all verification passes.

**Why this approach:**
- Closes the archival gap immediately rather than deferring to `orch clean --stale`
- Workspace is archived while context is fresh
- No accumulation of completed workspaces

**Trade-offs accepted:**
- Workspace is no longer in original location if immediate re-inspection needed
- Can use `--no-archive` to opt out when needed

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` - Main implementation target
- `cmd/orch/clean_cmd.go` - Reference for existing archival patterns
- `pkg/session/registry.go` - Session registry structure
- `.kb/models/workspace-lifecycle-model.md` - Architectural context

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/

# Run archival tests
go test ./cmd/orch/ -run "Archive|RegistryArchived" -v
```

---

## Investigation History

**2026-01-17 01:35:** Investigation started
- Initial question: How to implement automated archival in orch complete?
- Context: Implementing workspace lifecycle model's archival phase

**2026-01-17 01:55:** Implementation complete
- Status: Complete
- Key outcome: Automated archival added to `orch complete` with 8 passing tests
