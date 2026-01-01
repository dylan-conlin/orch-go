<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed cross-project investigation/decision file discovery by passing the found project directory to search functions instead of only searching in the dashboard server's directory.

**Evidence:** Code analysis shows `findInvestigationByWorkspace` and `findDecisionByWorkspace` only searched in `serveEffectiveDir`, but workspace lookup already correctly found cross-project workspaces via `discoverAllProjectDirs()`. Build passes after fix.

**Knowledge:** Cross-project agent visibility requires all lookups (workspace, investigations, decisions) to search the same set of project directories. The workspace lookup pattern was correct; artifact lookups needed to follow the same pattern.

**Next:** Close - fix implemented, tests pass, no breaking changes.

---

# Investigation: Dashboard Artifact Viewer Cross Project Discovery

**Question:** Why does the dashboard artifact viewer fail to find cross-project investigation files?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** og-debug-dashboard-artifact-viewer-01jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** None (builds on prior investigation but doesn't replace it)

---

## Findings

### Finding 1: Workspace lookup correctly discovers cross-project directories

**Evidence:** The `handleAgentArtifact` function at lines 1311-1332 uses `discoverAllProjectDirs()` to find workspaces across all projects, not just the current one.

**Source:** `cmd/orch/serve.go:1311-1332`
```go
allProjectDirs := discoverAllProjectDirs()
projectDirs := make(map[string]bool)
if serveEffectiveDir != "" && serveEffectiveDir != "unknown" {
    projectDirs[serveEffectiveDir] = true
}
for _, dir := range allProjectDirs {
    if dir != "" {
        projectDirs[dir] = true
    }
}
// Find workspace across all project directories
var workspacePath string
for projectDir := range projectDirs {
    candidatePath := filepath.Join(projectDir, ".orch", "workspace", workspaceID)
    if _, err := os.Stat(candidatePath); err == nil {
        workspacePath = candidatePath
        break
    }
}
```

**Significance:** This is the correct pattern - it discovers all projects via OpenCode session storage and searches each one. The artifact discovery functions should follow this same pattern.

---

### Finding 2: Investigation/decision search only used server's directory

**Evidence:** `findInvestigationByWorkspace` only searched in `serveEffectiveDir`:
```go
searchDirs := []string{
    filepath.Join(serveEffectiveDir, ".kb", "investigations", "simple"),
    filepath.Join(serveEffectiveDir, ".kb", "investigations"),
}
```
Similarly, `findDecisionByWorkspace` only searched in `filepath.Join(serveEffectiveDir, ".kb", "decisions")`.

**Source:** 
- `cmd/orch/serve.go:1533-1535` (before fix)
- `cmd/orch/serve.go:1595` (before fix)

**Significance:** This is the root cause. When an agent in project B creates an investigation, the dashboard server in project A couldn't find it because it only looked in project A's `.kb/` directory.

---

### Finding 3: Project directory was already available during workspace lookup

**Evidence:** The workspace lookup loop already knew which project directory contained the workspace, but this information was discarded after the lookup:
```go
for projectDir := range projectDirs {
    candidatePath := filepath.Join(projectDir, ".orch", "workspace", workspaceID)
    if _, err := os.Stat(candidatePath); err == nil {
        workspacePath = candidatePath
        break  // projectDir is lost here!
    }
}
```

**Source:** `cmd/orch/serve.go:1327-1332` (before fix)

**Significance:** The fix is straightforward - capture `projectDir` when the workspace is found and pass it to the artifact search functions.

---

## Synthesis

**Key Insights:**

1. **Cross-project agent visibility pattern** - The codebase already has a working pattern via `discoverAllProjectDirs()` which scans OpenCode's session storage to find all projects with agents.

2. **Inconsistent application of pattern** - Workspace discovery used this pattern correctly, but investigation/decision discovery hardcoded `serveEffectiveDir`.

3. **Simple parameter threading fix** - By capturing the project directory during workspace lookup and passing it to artifact search functions, cross-project discovery works correctly.

**Answer to Investigation Question:**

The dashboard artifact viewer failed to find cross-project investigation files because `findInvestigationByWorkspace` and `findDecisionByWorkspace` only searched in the dashboard server's own directory (`serveEffectiveDir`). The fix adds a `projectDir` parameter to these functions and passes the discovered project directory from the workspace lookup.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes after changes (verified: `go build ./cmd/orch/...` succeeded)
- ✅ Existing tests pass (verified: `go test ./cmd/orch/... -run "Artifact|Investigation|Decision"`)
- ✅ Code inspection confirms pattern consistency between workspace and artifact lookup

**What's untested:**

- ⚠️ End-to-end cross-project scenario (would require running dashboard from project A while agent runs in project B)
- ⚠️ Edge case: multiple projects have investigation files with matching keywords

**What would change this:**

- Finding would be wrong if investigation files are stored in a location other than `.kb/investigations/` within the project directory
- Fix might need adjustment if the workspace path format changes

---

## Implementation Recommendations

### Recommended Approach ⭐

**Parameter threading** - Pass the discovered project directory from workspace lookup to artifact search functions.

**Why this approach:**
- Minimal change surface (3 functions modified)
- Follows existing pattern in codebase
- No architectural changes needed

**Trade-offs accepted:**
- Added parameter to 3 functions
- Functions now have backward-compatible fallback (use `serveEffectiveDir` if no projectDir passed)

**Implementation sequence:**
1. Add `projectDir` parameter to `findInvestigationByWorkspace` and `findDecisionByWorkspace`
2. Capture `foundProjectDir` during workspace lookup
3. Pass `foundProjectDir` to artifact search calls
4. Update `collectArtifactLinks` to pass through the project directory

### Alternative Approaches Considered

**Option B: Search all project directories in artifact functions**
- **Pros:** No parameter threading needed, more flexible
- **Cons:** Performance impact (searching N directories), might return wrong file if multiple matches
- **When to use instead:** If cross-project fallback search is needed as a safety net

---

## References

**Files Examined:**
- `cmd/orch/serve.go:1286-1410` - `handleAgentArtifact` endpoint
- `cmd/orch/serve.go:1495-1563` - `findInvestigationByWorkspace` function
- `cmd/orch/serve.go:1566-1617` - `findDecisionByWorkspace` function
- `cmd/orch/serve.go:4410-4483` - deliverables endpoint with similar pattern
- `cmd/orch/serve.go:419-488` - `discoverAllProjectDirs` function

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test verification
go test ./cmd/orch/... -v -run "Artifact|Investigation|Decision"
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-30-inv-dashboard-artifact-viewer-404s-workspaceid.md` - Prior investigation that fixed 404s from beads ID suffix in workspace name

---

## Investigation History

**2026-01-01 12:xx:** Investigation started
- Initial question: Why doesn't dashboard find cross-project investigation files?
- Context: Agent spawned with `--workdir` creates investigation in target project, but dashboard (running from orchestrator project) can't find it

**2026-01-01 12:xx:** Root cause identified
- `findInvestigationByWorkspace` only searched `serveEffectiveDir`
- Workspace lookup already had correct cross-project pattern

**2026-01-01 12:xx:** Fix implemented
- Added `projectDir` parameter to search functions
- Captured and passed project directory from workspace lookup
- Build passes, tests pass

**2026-01-01 12:xx:** Investigation completed
- Status: Complete
- Key outcome: Cross-project investigation/decision discovery fixed by passing project directory to search functions
