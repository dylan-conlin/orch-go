# Decision: Registry is Spawn-Time Metadata Cache

**Date:** 2026-01-12
**Status:** Accepted
**Deciders:** Dylan, Architect Agent (orch-go-6a0p1)

## Context

A P1 bug report (orch-go-6a0p1) noted that `orch abandon` doesn't remove agents from the registry, leaving them shown as "active" indefinitely. Initial assumption was this required adding a single missing function call (`agentReg.Abandon()`).

Strategic-first gate triggered due to 11+ prior registry investigations showing it as a "caching layer, not source of truth." Architect investigation revealed this isn't a simple bug fix but a systemic pattern requiring architectural clarity.

**Investigation findings:**
- Registry defines state transition methods (`Abandon()`, `Complete()`, `Remove()`) but they have ZERO production usage
- All lifecycle commands follow "write on spawn, read for lookups, never update" pattern:
  - `spawn_cmd.go`: Writes metadata (Register + Save)
  - `status_cmd.go`: Reads for listing (ListActive, ListCompleted)
  - `abandon_cmd.go`: Reads for lookups (Find) but never calls Abandon()
  - `complete_cmd.go`: Doesn't import pkg/registry at all
  - `clean_cmd.go`: Doesn't import pkg/registry at all

**The pattern emerged organically:** 12+ investigations confirmed registry is spawn-time snapshot, with agent state derived from authoritative sources (OpenCode API for session state, beads for issue status).

## Decision

**The registry is a spawn-time metadata cache for agent lookups, NOT a lifecycle state tracker.**

State transition methods (`Abandon()`, `Complete()`, `Remove()`) are deprecated and should not be integrated into lifecycle commands.

### Actions Taken

1. ✅ Added comprehensive doc comment to Registry struct explaining spawn-cache contract
2. ✅ Marked `Abandon()`, `Complete()`, `Remove()` methods as deprecated with rationale
3. ✅ Document state comes from OpenCode API + beads, not registry
4. ✅ Close orch-go-6a0p1 with explanation (expected behavior given design)

## Rationale

### Why document current reality vs implement full lifecycle?

**Aligns with actual implementation:**
- Systemic pattern across codebase (not isolated oversight)
- Complete and clean commands work fine without registry updates (proven by production usage)
- Commands already derive state from authoritative sources

**Minimal risk:**
- No behavior changes required
- Preserves working parts (session ID lookups, workspace name mappings)
- Clear justification for closing bug report

**State methods become dead code but that's acceptable:**
- Can be removed in future cleanup if desired
- Explicit deprecation prevents confusion about why they exist
- Tests can remain as documentation of "what was designed but never integrated"

### Why not implement full lifecycle management?

**High cost, unclear value:**
- Would require updating 3 commands (abandon, complete, clean)
- Risk of registry locking issues under concurrent operations
- No evidence that complete/clean need registry state (they work fine today)
- Would create state synchronization burden between registry, OpenCode API, and beads

**Fighting the design that emerged:**
- 12+ investigations showed this pattern working well
- Commands organically use registry for metadata lookups, not state tracking
- Forcing state management would be fighting proven usage patterns

### Why not remove registry entirely?

**Registry provides value for lookups:**
- `status_cmd.go` uses ListActive/ListCompleted for displaying agents
- `abandon_cmd.go` uses Find() to map beads ID → session ID
- Workspace name → session ID mapping useful for various operations
- Removing would require refactoring multiple commands

**State staleness is acceptable:**
- Commands don't rely on registry state field for decisions
- Actual state derived from OpenCode API (session.status) and beads (issue.status)
- Registry's "active" status is just metadata from spawn time

## Consequences

**Positive:**
- Clarifies registry's actual role (prevents future confusion)
- Unblocks bug report with clear architectural justification
- Preserves working functionality (lookups still work)
- Enables focused cleanup later (deprecated methods can be removed)
- Documents the design that emerged from production usage

**Negative:**
- Registry shows stale "active" status for completed agents (acceptable: not used for decisions)
- State transition methods remain as dead code until cleanup (acceptable: marked deprecated)
- Doesn't match original design intent (methods suggest full lifecycle tracking)

**Neutral:**
- Bug orch-go-6a0p1 closed as "expected behavior" (requires explanation to prevent reopening)

## Alternatives Considered

### Implement Full Registry Lifecycle Management

**Description:** Update abandon_cmd, complete_cmd, clean_cmd to call Abandon(), Complete(), Remove() methods.

**Pros:**
- Makes registry state accurate
- Fulfills original design intent
- Fixes all related staleness issues

**Cons:**
- High implementation cost (3 commands to update)
- High risk (registry locking under concurrent operations)
- Unclear value (complete/clean don't need registry today)
- Creates state synchronization burden

**Rejected because:** Cost exceeds benefit. Commands work fine without registry state updates.

### Remove Registry Entirely

**Description:** Delete pkg/registry, refactor commands to use OpenCode API + beads directly.

**Pros:**
- Eliminates dead code (state transition methods)
- Removes confusion about registry's purpose
- Forces commands to use authoritative sources

**Cons:**
- Breaks existing lookups in status/abandon (would need refactoring)
- Larger change scope than documentation
- Loses useful metadata cache (workspace name → session ID mapping)

**Rejected because:** Registry provides value for lookups. Removing it is higher risk than documenting its actual role.

### Add Registry State Management Selectively (Only Abandon)

**Description:** Add `agentReg.Abandon()` call to abandon_cmd but leave complete/clean unchanged.

**Pros:**
- Minimal change (one function call)
- "Fixes" the immediate bug report

**Cons:**
- Creates inconsistent behavior (abandon updates registry, complete doesn't)
- Doesn't address systemic pattern
- Would leave confusion: "Why does abandon update but complete doesn't?"

**Rejected because:** Inconsistency is worse than consistent "never update" pattern. Partial fix misleads future developers.

## Follow-Up

- [x] Update Registry struct documentation
- [x] Deprecate Abandon(), Complete(), Remove() methods
- [x] Close orch-go-6a0p1 with explanation
- [ ] Optional future cleanup: Remove deprecated methods after confirming no external usage

## References

- **Investigation:** `.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md`
- **Synthesis:** `.orch/workspace/og-arch-registry-abandonment-workflow-11jan-c1c7/SYNTHESIS.md`
- **Issue:** orch-go-6a0p1 (architect session)
- **Prior Knowledge:** SPAWN_CONTEXT.md lines 32-109 (11 registry investigations showing "caching layer" pattern)
- **Code References:**
  - `pkg/registry/registry.go:75-110` - Registry struct with new documentation
  - `pkg/registry/registry.go:432-490` - Deprecated state transition methods
  - `cmd/orch/spawn_cmd.go` - Registry write path
  - `cmd/orch/abandon_cmd.go` - Registry read path (Find)
  - `cmd/orch/status_cmd.go` - Registry read path (List)
