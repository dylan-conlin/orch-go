<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch complete` was checking workspace directory before registry, causing orchestrator session names to fall through to beads ID lookup and fail.

**Evidence:** Before fix: `orch complete og-orch-xxx` tried bd show on workspace name when workspace not in current dir. After fix: registry lookup first successfully identifies orchestrator sessions, even cross-project.

**Knowledge:** The orchestrator session registry is the authoritative source for orchestrator sessions, not just workspace directories. Cross-project orchestrators require registry lookup since their workspaces are in different directories.

**Next:** Fix implemented and tested. Close issue.

---

# Investigation: Orch Complete Fails Orchestrator Sessions

**Question:** Why does `orch complete` fail for orchestrator sessions with "bd show" errors?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Workspace lookup order was wrong

**Evidence:** In `complete_cmd.go`, the code:
1. Called `findWorkspaceByName(currentDir, identifier)` - only checks current project's `.orch/workspace/`
2. If not found, treated identifier as beads ID and called `resolveShortBeadsID(identifier)`
3. This caused workspace names like `og-orch-xxx` to be passed to `bd show`, which fails

**Source:** `cmd/orch/complete_cmd.go` lines 107-134

**Significance:** Orchestrator workspace names look like workspace identifiers, not beads IDs. When `findWorkspaceByName` fails (e.g., wrong directory), the fallback to beads lookup is inappropriate.

---

### Finding 2: Orchestrator session registry exists but wasn't used for lookup

**Evidence:** 
- Registry at `~/.orch/sessions.json` tracks orchestrator sessions with `ProjectDir`
- `registry.Get(workspaceName)` can find sessions by workspace name
- Complete command only used registry for unregistering, not for finding sessions

**Source:** `pkg/session/registry.go` - `Get()` method at line 229

**Significance:** The registry is the authoritative source for orchestrator sessions and includes the correct `ProjectDir` for cross-project cases.

---

### Finding 3: Cross-project orchestrator sessions require registry lookup

**Evidence:** 
- Orchestrator in project A can manage work in project B
- Workspace exists in project B's `.orch/workspace/` directory
- Current directory is project A, so `findWorkspaceByName(currentDir, ...)` fails
- Registry stores `ProjectDir` which points to the correct project

**Source:** Tested with `pw-orch-resume-p1-material-05jan` in orch-go directory

**Significance:** Registry-first lookup is essential for cross-project orchestrator completion.

---

## Synthesis

**Key Insights:**

1. **Registry is authoritative for orchestrators** - The session registry contains the correct `ProjectDir` for every orchestrator session, making it the authoritative source for finding them.

2. **Workspace name vs beads ID disambiguation** - By checking registry first, we can distinguish orchestrator workspace names from beads IDs without trying beads lookup.

3. **Cross-project support enabled** - Registry-first lookup naturally handles cross-project orchestrators since `ProjectDir` is stored with each session.

**Answer to Investigation Question:**

`orch complete` failed for orchestrator sessions because it checked the current directory's workspace folder before the registry. When an orchestrator workspace wasn't in the current directory (cross-project case), the code fell through to beads ID lookup, treating workspace names like `og-orch-xxx` as beads IDs and failing with bd show errors.

The fix adds registry lookup as the FIRST step in the lookup chain:
1. Check orchestrator session registry (handles cross-project)
2. Check workspace directory in current/specified project
3. Only fall back to beads ID lookup for worker sessions

---

## Structured Uncertainty

**What's tested:**

- ✅ Registry-first lookup finds orchestrator sessions (verified: test passes, smoke test passes)
- ✅ Cross-project orchestrator completion works (verified: completed pw-orch from orch-go directory)
- ✅ Worker completion still works via beads ID lookup (verified: existing tests pass)
- ✅ Backward compatibility with workspace directory lookup (verified: TestOrchestratorWorkspaceDetection passes)

**What's untested:**

- ⚠️ Legacy workspaces without registry entries (graceful degradation to directory lookup)
- ⚠️ Performance impact of registry lookup (likely minimal, file is small)

**What would change this:**

- Finding would be wrong if there's a valid use case where beads IDs look like workspace names
- Approach would need revision if registry becomes a performance bottleneck

---

## Implementation Recommendations

**Purpose:** Document the implemented fix.

### Recommended Approach ⭐

**Registry-first lookup** - Check orchestrator session registry before workspace directory or beads ID lookup.

**Why this approach:**
- Registry is authoritative source for orchestrator sessions
- Handles cross-project orchestrators naturally
- Prevents workspace names from being misinterpreted as beads IDs

**Trade-offs accepted:**
- Adds one file read for every `orch complete` call
- Registry must be kept in sync with actual sessions

**Implementation sequence:**
1. ✅ Check registry for identifier
2. ✅ If found, use registry's ProjectDir to locate workspace
3. ✅ If not in registry, check workspace directory
4. ✅ Only fall back to beads lookup for workers

### Implementation Details

**What was implemented:**
- Modified `runComplete()` in `cmd/orch/complete_cmd.go` to check registry first
- Added registry lookup step before workspace directory lookup
- Uses registry's `ProjectDir` to find cross-project workspaces

**Things to watch out for:**
- ⚠️ Registry file locking - uses existing lock mechanism
- ⚠️ Missing registry entries - gracefully falls through to directory lookup

**Success criteria:**
- ✅ `orch complete og-orch-xxx` no longer fails with bd show error
- ✅ Cross-project orchestrator completion works
- ✅ All existing tests pass
- ✅ New test added for registry-first lookup

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` - Main completion logic
- `pkg/session/registry.go` - Registry API
- `cmd/orch/shared.go` - Helper functions (findWorkspaceByName, etc.)

**Commands Run:**
```bash
# Reproduce bug before fix
orch complete og-orch-nonexistent-05jan
# Error: failed to parse bd show output

# Verify fix
orch complete og-orch-complete-orchestrator-session-05jan
# Success: Orchestrator session (from registry)

# Cross-project test
orch complete pw-orch-resume-p1-material-05jan
# Success: Found via registry, completed
```

---

## Investigation History

**2026-01-05 15:00:** Investigation started
- Initial question: Why does orch complete fail for orchestrator sessions?
- Context: Orchestrator sessions don't have beads tracking

**2026-01-05 15:30:** Root cause identified
- Workspace lookup order was wrong - beads lookup happened before registry check
- Cross-project case not handled

**2026-01-05 15:45:** Fix implemented
- Added registry-first lookup in runComplete()
- Added test for registry-first lookup

**2026-01-05 16:00:** Investigation completed
- Status: Complete
- Key outcome: Registry-first lookup fixes orchestrator completion for both same-project and cross-project cases
