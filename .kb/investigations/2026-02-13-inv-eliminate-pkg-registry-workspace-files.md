## Summary (D.E.K.N.)

**Delta:** `pkg/registry/` (529 lines) eliminated entirely -- workspace files (AGENT_MANIFEST.json, .session_id) already serve all lookup needs.

**Evidence:** Build passes (`go build`, `go vet ./...`), zero `pkg/registry` references remain, all tests pass (no new failures).

**Knowledge:** The agent registry was write-heavy/read-light -- only 4 consumers, and each read was for data already persisted in workspace files. Adding `Model` field to existing `AgentManifest` struct closed the last gap.

**Next:** Close orch-go-352. Phase 3 of lifecycle-ownership-boundaries can proceed (session registry consolidation).

**Authority:** implementation - Scoped removal within existing workspace file patterns, no cross-boundary impact.

---

# Investigation: Eliminate Pkg Registry Workspace Files

**Question:** Can `pkg/registry/` be removed by migrating all reads to workspace file reads?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` (Phase 2)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| lifecycle-ownership-boundaries decision | implements Phase 2 | Yes | None |

## Findings

### Finding 1: Registry has only 4 consumers, all doing simple lookups

**Evidence:** `grep -r "pkg/registry"` found imports in: spawn_cmd.go (register on spawn), status_cmd.go (list active/completed), abandon_cmd.go (find by beads ID), clean_cmd.go (remove inactive entries).

**Source:** `cmd/orch/spawn_cmd.go`, `cmd/orch/status_cmd.go`, `cmd/orch/abandon_cmd.go`, `cmd/orch/clean_cmd.go`

**Significance:** All 4 use cases are simple reads that workspace files already support. No complex querying or state management.

### Finding 2: AGENT_MANIFEST.json already captures all registry metadata except Model

**Evidence:** Manifest struct contains: workspace_name, skill, beads_id, project_dir, git_baseline, spawn_time, tier, spawn_mode. Registry additionally stored model and session_id, but session_id is in `.session_id` file.

**Source:** `pkg/spawn/session.go:AgentManifest` struct, `.orch/workspace/*/AGENT_MANIFEST.json`

**Significance:** Adding one `Model` field to AgentManifest closes the data gap entirely.

### Finding 3: Registry's Abandon/Complete/Remove methods have zero callers

**Evidence:** `grep -r "\.Abandon\|\.Complete\|\.Remove" pkg/registry/` shows only definitions, no call sites outside the package itself.

**Source:** `pkg/registry/registry.go`

**Significance:** Half the registry code is dead weight -- lifecycle state management that nothing uses.

## Synthesis

**Key Insights:**

1. **Workspace files are the natural source of truth** - They're written at spawn time and persist until cleanup, making them more reliable than a separate JSON file that requires merge-on-save with file locking.

2. **Registry was accumulating complexity for zero benefit** - File locking, merge-on-save, slot reuse -- all serving a single JSON file that duplicated workspace data.

3. **Status command migration was the critical path** - Phase 1 agent collection needed workspace scanning instead of registry listing. The replacement is simpler (fewer lines) and more accurate (reads source-of-truth files directly).

## References

**Files Modified:**
- `cmd/orch/spawn_cmd.go` - Removed registerAgent() calls and function
- `cmd/orch/status_cmd.go` - Replaced Phase 1 with workspace scanning
- `cmd/orch/abandon_cmd.go` - Replaced registry lookup with workspace file lookup
- `cmd/orch/clean_cmd.go` - Removed cleanInactiveRegistryEntries()
- `pkg/spawn/session.go` - Added Model field to AgentManifest
- `pkg/spawn/context.go` - Set Model in manifest write

**Files Deleted:**
- `pkg/registry/registry.go` (529 lines)
- `pkg/registry/registry_test.go`

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` - Phase 2 of 4
