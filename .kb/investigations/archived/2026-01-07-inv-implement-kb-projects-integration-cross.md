<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented getKBProjects() function and integrated it into extractUniqueProjectDirs() to enable cross-project workspace scanning.

**Evidence:** Tests pass, build succeeds. kb projects list --json returns 17 projects including price-watch which has 84 workspaces that were previously invisible.

**Knowledge:** OpenCode --attach mode uses server cwd, not spawn --workdir. Solution: source project dirs from kb projects registry instead of relying on session directories alone.

**Next:** None - implementation complete, ready for integration testing.

**Promote to Decision:** recommend-no (tactical fix implementing prior investigation's recommendation)

---

# Investigation: Implement KB Projects Integration for Cross-Project Visibility

**Question:** How to implement the kb projects integration to fix cross-project agent visibility?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: kb projects list --json provides structured project data

**Evidence:** 
```bash
kb projects list --json
# Returns: [{"name":"kb-cli","path":"/Users/dylanconlin/Documents/personal/kb-cli"},...]
```

**Source:** Direct command execution

**Significance:** JSON format enables reliable parsing without text manipulation.

---

### Finding 2: 14 projects have .orch/workspace directories with agents

**Evidence:**
- orch-knowledge: 800 workspaces
- orch-cli: 320 workspaces  
- price-watch: 84 workspaces (the specific project from reproduction)
- beads-ui-svelte: 72 workspaces
- 10 other projects with 1-43 workspaces each

**Source:** `ls -1 "$proj/.orch/workspace" | wc -l` for each kb project

**Significance:** Confirms significant cross-project workspace data that was being missed.

---

### Finding 3: extractUniqueProjectDirs is the correct integration point

**Evidence:** Function already collects directories from OpenCode sessions and current project. Adding kb projects at the end follows the existing pattern.

**Source:** serve_agents_cache.go:242-272

**Significance:** Minimal code change with maximum impact - adds ~30 lines including tests.

---

## Synthesis

**Key Insights:**

1. **kb projects is an ideal source** - User-maintained registry of projects, already used for cross-project orchestration. No new dependencies.

2. **Graceful degradation is essential** - kb CLI may not be available in all environments. Silent fallback to empty slice ensures dashboard works regardless.

3. **Existing test structure needed updating** - Tests had hardcoded expected counts. Refactored to test relative behavior instead.

**Answer to Investigation Question:**

Implemented by:
1. Adding `kbProject` struct for JSON parsing
2. Adding `getKBProjects()` function with exec.Command + JSON unmarshal + graceful fallback
3. Modifying `extractUniqueProjectDirs()` to merge kb projects with deduplication
4. Updating tests to account for variable kb project counts

---

## Implementation Details

**Files Changed:**
- `cmd/orch/serve_agents_cache.go` - Added getKBProjects() and integrated into extractUniqueProjectDirs()
- `cmd/orch/serve_agents_cache_test.go` - Added new tests
- `cmd/orch/serve_agents_test.go` - Updated TestExtractUniqueProjectDirs for new behavior

**New Imports:**
- `encoding/json` - For parsing kb projects list output
- `log` - For warning on graceful fallback
- `os/exec` - For running kb CLI command

---

## References

**Files Examined:**
- cmd/orch/serve_agents_cache.go - extractUniqueProjectDirs, buildMultiProjectWorkspaceCache
- cmd/orch/serve_agents_test.go - existing test patterns

**Commands Run:**
```bash
# Verify kb projects output format
kb projects list --json

# Count workspaces per project
for proj in $(kb projects list --json | jq -r '.[] | .path'); do
  ls -1 "$proj/.orch/workspace" 2>/dev/null | wc -l
done

# Run tests
go test ./cmd/orch/... -run "TestGetKBProjects|TestExtractUniqueProjectDirs"
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md - Root cause analysis
- **Decision:** This implements Option C from the prior investigation

---

## Investigation History

**2026-01-07 23:35:** Investigation started
- Implementing kb projects integration as designed in prior investigation

**2026-01-07 23:45:** Implementation complete
- getKBProjects() implemented with JSON parsing and graceful fallback
- extractUniqueProjectDirs() updated to include kb projects
- Tests written and passing

**2026-01-07 23:50:** Verification complete
- All tests pass
- Build successful
- price-watch's 84 workspaces now discoverable
