<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard agents filter was using s.Directory (OpenCode session directory) for early filtering, which is wrong for --workdir spawns since s.Directory is the orchestrator's cwd, not the target project.

**Evidence:** Code inspection showed filterByProject(s.Directory, ...) at line 345 used session directory, while agent.ProjectDir at line 727 is correctly populated from workspace cache.

**Knowledge:** Cross-project visibility requires filtering AFTER project_dir is populated from workspace cache, not before using session directory.

**Next:** Fix deployed - removed early project filter, kept late filter using agent.ProjectDir.

**Promote to Decision:** recommend-no (bug fix, not architectural change)

---

# Investigation: Dashboard Agents Filter Session Directory

**Question:** Why does the dashboard project filter not show agents spawned with --workdir to other projects?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Early filter uses wrong directory

**Evidence:** Line 345 of serve_agents.go had:
```go
if projectFilterParam != "" && !filterByProject(s.Directory, projectFilterParam) {
    // Session is from a different project, skip it
    continue
}
```

**Source:** cmd/orch/serve_agents.go:345

**Significance:** `s.Directory` is the OpenCode session directory, which for --attach sessions is the orchestrator's cwd (e.g., orch-go) regardless of the `--workdir` flag used during spawn. This causes agents spawned to other projects to be filtered out when using project filter.

---

### Finding 2: Correct project_dir populated later from workspace cache

**Evidence:** Line 727 correctly populates ProjectDir from workspace cache:
```go
// Populate project_dir from beadsProjectDirs lookup (for workspace path construction)
if agentProjectDir, ok := beadsProjectDirs[agents[i].BeadsID]; ok {
    agents[i].ProjectDir = agentProjectDir
}
```

**Source:** cmd/orch/serve_agents.go:727

**Significance:** The workspace cache extracts PROJECT_DIR from SPAWN_CONTEXT.md, which has the correct target project directory from --workdir flag. This value is correct for filtering.

---

### Finding 3: Late filter already uses correct field

**Evidence:** Line 894 uses the correct field:
```go
if projectFilterParam != "" && !filterByProject(agent.ProjectDir, projectFilterParam) {
    continue
}
```

**Source:** cmd/orch/serve_agents.go:894

**Significance:** The late filter already uses agent.ProjectDir which has the correct value from workspace cache. The fix simply needs to remove the early filter that uses the wrong field.

---

## Synthesis

**Key Insights:**

1. **Timing matters for cross-project visibility** - The session directory is available immediately but incorrect for --workdir spawns. The correct project_dir is only available after workspace cache lookup.

2. **Redundant filtering** - The early filter was redundant with the late filter, but used the wrong field. Removing it simplifies the code and fixes the bug.

3. **Performance tradeoff acceptable** - Removing early project filtering may process more sessions before late filtering, but this is a small cost for correct behavior.

**Answer to Investigation Question:**

The dashboard project filter wasn't showing --workdir agents because early filtering used s.Directory (session directory = orchestrator's cwd) instead of waiting for the correct project_dir from workspace cache. The fix removes the early project filter, keeping only the late filter which uses agent.ProjectDir correctly populated from workspace cache.

---

## Structured Uncertainty

**What's tested:**

- ✅ filterByProject function works correctly (unit tests added in serve_filter_test.go)
- ✅ All existing tests pass after fix
- ✅ Build succeeds

**What's untested:**

- ⚠️ Live dashboard verification with actual --workdir spawns (would need manual testing)
- ⚠️ Performance impact of removing early filter (expected minimal)

**What would change this:**

- If performance degrades significantly, could add early filtering based on workspace name pattern instead of session directory

---

## Implementation Recommendations

### Recommended Approach ⭐

**Remove early project filter** - Delete the project filter from early filtering block, keep only time filter. Late filter using agent.ProjectDir handles project filtering correctly.

**Why this approach:**
- Correct behavior for --workdir spawns
- Minimal code change (just remove ~4 lines)
- Late filter already exists and uses correct field

**Trade-offs accepted:**
- Slight performance impact from processing more sessions before late filter
- Acceptable because workspace cache and beads batch fetch are already the main costs

**Implementation sequence:**
1. Remove project filter from early filtering (lines 344-348)
2. Add comment explaining why project filter is not applied early
3. Keep late filter unchanged at line 894

---

## References

**Files Examined:**
- cmd/orch/serve_agents.go - Main handler with filtering logic
- cmd/orch/serve_filter.go - Filter utility functions

**Commands Run:**
```bash
# Verify filterByProject usage
grep -n "filterByProject" cmd/orch/serve_agents.go

# Run tests
go test ./cmd/orch/... -run "Test.*Filter"

# Build
go build ./cmd/orch
```

---

## Investigation History

**2026-01-07 16:45:** Investigation started
- Initial question: Why doesn't project filter show --workdir agents?
- Context: Cross-project visibility fix incomplete - agents have correct project_dir but filter uses wrong field

**2026-01-07 17:00:** Root cause identified
- Early filter at line 345 uses s.Directory (wrong)
- Late filter at line 894 uses agent.ProjectDir (correct)
- Fix: Remove early project filter

**2026-01-07 17:10:** Investigation completed
- Status: Complete
- Key outcome: Removed early project filter, fixed cross-project dashboard visibility
