# Session Synthesis

**Agent:** og-debug-implement-fix-cross-25feb-9aa8
**Issue:** orch-go-1231
**Outcome:** success

---

## Plain-Language Summary

Fixed the work-graph showing 'unassigned' for cross-project in-progress issues (like toolshed-* issues). The root cause was that `buildActiveAgentMap()` in `serve_beads.go` only looked for agents in the local project — it queried local beads and the default OpenCode session scope, both of which excluded cross-project agents. Two changes: (1) extended `listTrackedIssues()` to accept project directories and query beads in each registered project, and (2) replaced `client.ListSessions("")` with the existing `listSessionsAcrossProjects()` function. Now cross-project in-progress issues correctly show their active agent data (phase, model, runtime).

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key verification: toolshed in_progress issues now return `active_agent` data in `/api/beads/graph?project_dir=.../toolshed`.

---

## TLDR

Fixed two scoping gaps in `buildActiveAgentMap()` that caused cross-project issues to show 'unassigned' — extended beads queries to span all project dirs and used `listSessionsAcrossProjects()` for OpenCode sessions.

---

## Delta (What Changed)

### Files Created
- `pkg/beads/client.go` - Added `FallbackListWithLabelInDir()` function for cross-project beads CLI queries

### Files Modified
- `cmd/orch/query_tracked.go` - Extended `listTrackedIssues()` to accept `projectDirs` parameter, added `listTrackedIssuesLocal()` and `listTrackedIssuesForDir()` helper functions
- `cmd/orch/serve_beads.go` - Replaced `client.ListSessions("")` with `listSessionsAcrossProjects(client, sourceDir)` in `buildActiveAgentMap()`

---

## Evidence (What Was Observed)

- Before fix: toolshed in_progress issues returned `active_agent: null` in `/api/beads/graph?project_dir=.../toolshed`
- After fix: all 4 toolshed in_progress issues (164, 148, 165, 163) return `active_agent: {model: "anthropic/claude-opus-4-5-20251101"}`
- Local orch-go graph still works correctly (2 in_progress issues with active_agent data)

### Tests Run
```bash
go build ./cmd/orch/
# BUILD: PASS

go test ./cmd/orch/ -run TestServe -count=1
# ok  github.com/dylan-conlin/orch-go/cmd/orch  0.034s

go test ./cmd/orch/ -count=1 -timeout 120s
# ok  github.com/dylan-conlin/orch-go/cmd/orch  2.844s

go test ./pkg/beads/ -count=1 -timeout 60s
# ok  github.com/dylan-conlin/orch-go/pkg/beads  0.015s
```

### Live Verification
```bash
curl -sk "https://localhost:3348/api/beads/graph?project_dir=.../toolshed"
# toolshed-164: active_agent=PRESENT
# toolshed-148: active_agent=PRESENT
# toolshed-165: active_agent=PRESENT
# toolshed-163: active_agent=PRESENT
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Reused existing `listSessionsAcrossProjects()` rather than duplicating session aggregation logic — consistency with `/api/agents` handler
- Added `FallbackListWithLabelInDir()` to beads package rather than modifying `FallbackListWithLabel()` — preserves backward compatibility

### Constraints Discovered
- `CLIClient.List()` doesn't support label filtering via CLI args — had to use separate fallback function for label-based queries in cross-project dirs

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Live verification confirms bug fix
- [x] Ready for `orch complete orch-go-1231`

---

## Unexplored Questions

- The `queryTrackedAgents()` function (used by `/api/agents`) now also benefits from cross-project `listTrackedIssues()` since it passes `projectDirs`. This is a bonus improvement.
- `CLIClient.List` should probably support `-l` label filtering to avoid needing separate `FallbackListWithLabelInDir` — but that's a separate enhancement.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-implement-fix-cross-25feb-9aa8/`
**Beads:** `bd show orch-go-1231`
