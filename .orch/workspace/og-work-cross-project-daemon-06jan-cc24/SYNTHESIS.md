# Session Synthesis

**Agent:** og-work-cross-project-daemon-06jan-cc24
**Issue:** orch-go-g7hax
**Duration:** 2026-01-06 14:00 → 2026-01-06 15:15
**Outcome:** success

---

## TLDR

Investigated cross-project daemon architecture. Found that a single daemon can poll all registered projects by iterating over `kb projects list` and using existing beads/spawn patterns with `--workdir`. No architectural blockers - ready for implementation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md` - Full design investigation with findings, synthesis, and implementation recommendations

### Files Modified
- None (investigation-only session)

### Commits
- To be committed with SYNTHESIS.md

---

## Evidence (What Was Observed)

- `kb projects list` returns 17 registered projects with paths - existing mechanism for project discovery
- `pkg/beads/client.go:46-50` - `WithCwd()` option already supports targeted directory operations
- `pkg/beads/client.go:22-23` - `DefaultDir` package variable can override cwd-based discovery
- `pkg/daemon/issue_adapter.go:16-32` - Current daemon uses `beads.FindSocketPath("")` which starts from cwd (implicitly project-scoped)
- `cmd/orch/spawn_cmd.go:57,541-578` - `--workdir` flag already handles cross-project spawns
- Prior kb constraint confirms: "cross project agent visibility requires fetching beads comments from agent's project directory"

### Tests Run
```bash
# Verified kb projects returns data
kb projects list
# Result: 17 projects with paths

# Verified beads ready output structure
bd ready --json
# Result: JSON array of issues with id, title, status, priority, issue_type, labels
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md` - Complete design investigation

### Decisions Made
- Decision 1: Use `kb projects list` as project registry (not new config) because it already exists and integrates with orchestration workflow
- Decision 2: Iterate over projects in each poll cycle (simple approach) rather than aggregating databases (would violate cross-repo contamination constraint)
- Decision 3: Single capacity pool across all projects to prevent runaway spawning

### Constraints Discovered
- Projects must be kb-registered to be daemon-visible (acceptable trade-off)
- Beads cross-repo contamination must be avoided (ruled out aggregated database approach)
- Error in one project must not crash daemon or block other projects

### Externalized via `kn`
- N/A - Findings captured in investigation file

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up (create epic with implementation tasks)

### If Spawn Follow-up
**Issue:** Cross-project daemon implementation
**Skill:** feature-impl
**Context:**
```
Implement cross-project daemon per investigation findings at .kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md. 
Key steps: 1) Add project discovery function, 2) Modify ListReadyIssues to accept project path, 3) Update daemon loop for cross-project iteration, 4) Ensure spawn passes --workdir.
```

**Recommended epic structure:**
1. **Add project discovery function** - Parse `kb projects list` output
2. **Modify ListReadyIssues for targeted projects** - Add projectPath parameter, set beads.DefaultDir
3. **Update daemon loop for cross-project iteration** - Iterate over projects, aggregate issues, shared capacity
4. **Add cross-project spawn support in orch work** - Pass `--workdir` when spawning
5. **Add project-aware logging and dashboard support** - Include project name in spawn logs

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should project list be cached or refreshed every poll cycle? (refresh ensures new projects are discovered, cache is faster)
- Should daemon have `--projects` flag to filter which projects to monitor? (useful for testing/debugging)
- Does dashboard need cross-project awareness for agent visibility? (currently shows all agents regardless of project)

**Areas worth exploring further:**
- Performance impact of polling 17+ projects every 60 seconds
- Behavior when individual project beads daemon is down

**What remains unclear:**
- Interaction with launchd-managed daemon (does launchd enforce single instance?)

---

## Session Metadata

**Skill:** design-session
**Model:** opus
**Workspace:** `.orch/workspace/og-work-cross-project-daemon-06jan-cc24/`
**Investigation:** `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md`
**Beads:** `bd show orch-go-g7hax`
