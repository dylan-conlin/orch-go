<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The registry abandonment issue is NOT a simple oversight - it's a systemic pattern where registry state management methods exist but are never called by lifecycle commands.

**Evidence:** Registry has Abandon(), Complete(), Remove() methods (pkg/registry/registry.go:432-490) but only Abandon() is called in tests; spawn writes to registry, status/abandon read from it, but complete/clean don't even import registry package.

**Knowledge:** The registry was designed with state transitions but only half-implemented: spawn registers agents (write), status/abandon query them (read), but no commands update state on completion/abandonment, causing registry to become permanently stale.

**Next:** Recommend documenting current reality: registry is spawn-time snapshot only, not lifecycle tracker; optionally create epic for full state management if needed.

**Promote to Decision:** Actioned - decision exists (registry-contract-spawn-cache-only)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Registry Abandonment Workflow Validate Simple

**Question:** Is the registry abandonment bug (orch abandon doesn't remove agent from registry) a simple oversight requiring one missing function call, or a symptom of deeper registry state management issues?

**Started:** 2026-01-11
**Updated:** 2026-01-11
**Owner:** Architect agent (spawned from orch-go-6a0p1)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Registry State Transition Methods Exist But Are Never Called

**Evidence:** 
- Registry defines state transition methods: `Abandon()` at line 432, `Complete()` at line 450, `Remove()` at line 470 (pkg/registry/registry.go)
- Grep for `.Abandon()` in production code: ZERO results (only found in registry_test.go)
- Grep for `.Complete()` in production code: ZERO results
- Grep for `agentReg.` usage shows only: Register (spawn_cmd.go), Find (abandon_cmd.go), ListActive/ListCompleted (status_cmd.go)

**Source:** 
- pkg/registry/registry.go:432-490 (method definitions)
- `rg "\.Abandon\(" --type go` (found only test usage)
- `rg "agentReg\." --type go` (found only read/register operations)

**Significance:** The registry was designed with state transition methods but these methods were never integrated into the lifecycle commands, indicating incomplete implementation rather than a single missing call.

---

### Finding 2: Lifecycle Commands Have Inconsistent Registry Integration

**Evidence:**
- **spawn_cmd.go**: Imports registry, calls `agentReg.Register()` and `agentReg.Save()` (writes to registry)
- **abandon_cmd.go**: Imports registry, calls `agentReg.Find()` (reads from registry) but NEVER calls `agentReg.Abandon()`
- **complete_cmd.go**: Does NOT import pkg/registry at all (no registry interaction)
- **clean_cmd.go**: Does NOT import pkg/registry at all (no registry interaction)
- **status_cmd.go**: Imports registry, calls `agentReg.ListActive()` and `ListCompleted()` (reads from registry)

**Source:**
- cmd/orch/spawn_cmd.go (grep for agentReg.Register)
- cmd/orch/abandon_cmd.go:122-126 (finds agent), 285-301 (updates orchestrator session registry, NOT agent registry)
- cmd/orch/complete_cmd.go:1-1133 (no pkg/registry import)
- cmd/orch/clean_cmd.go:1-1119 (no pkg/registry import)
- cmd/orch/status_cmd.go (grep for agentReg.List)

**Significance:** Registry integration follows a "write on spawn, read for lookups, never update" pattern, indicating this is not an oversight in one command but a systemic pattern across the codebase.

---

### Finding 3: Registry Becomes Permanently Stale After Spawn

**Evidence:**
- Agent registered with `Status: StateActive` on spawn (registry.go:346)
- Status never updated when agent completes (complete_cmd doesn't touch registry)
- Status never updated when agent abandoned (abandon_cmd finds agent but doesn't call Abandon method)
- Status never updated when agent cleaned (clean_cmd doesn't touch registry)
- Result: All agents remain `StateActive` in registry forever, even after completion/abandonment

**Source:**
- pkg/registry/registry.go:343-350 (Register sets StateActive)
- cmd/orch/complete_cmd.go (no registry updates)
- cmd/orch/abandon_cmd.go:64-325 (no agentReg.Abandon() call)
- cmd/orch/clean_cmd.go (no registry updates)

**Significance:** The registry is effectively a "spawn-time snapshot" that never reflects post-spawn lifecycle changes, making it unsuitable for accurate agent state tracking beyond the initial spawn.

---

## Synthesis

**Key Insights:**

1. **Half-Implemented Design Pattern** - The registry was designed with full state lifecycle management (Abandon, Complete, Remove methods), but only the write path (Register) and read path (Find, List) were implemented in commands. The update path was never integrated, leaving the state transition methods orphaned.

2. **Systemic Not Isolated** - This isn't a bug in one command. All three lifecycle commands (abandon, complete, clean) exhibit the same pattern: they don't update registry state. Complete and clean don't even import the registry package. This consistency suggests a design decision (intentional or unintentional) rather than an oversight.

3. **Registry as Spawn Snapshot vs Lifecycle Tracker** - The current implementation treats registry as a spawn-time cache for metadata (session IDs, tmux windows, beads IDs) used for lookups, not as a source of truth for agent state. This works for the read-only use cases (status command showing "what's spawned?") but breaks any assumption that registry reflects current agent state.

**Answer to Investigation Question:**

This is NOT a simple oversight requiring one missing function call. The registry abandonment bug is a symptom of a deeper architectural pattern where the registry's state management design was never fully implemented across lifecycle commands. Evidence: (1) state transition methods exist but are never called in production code (Finding 1), (2) all lifecycle commands follow "read but don't update" pattern (Finding 2), (3) registry becomes permanently stale after spawn (Finding 3). Adding `agentReg.Abandon()` to abandon_cmd would fix the immediate symptom but leave complete and clean with the same underlying issue, creating inconsistent behavior where abandonment updates registry but completion doesn't. A coherent solution requires either: (A) implementing full registry lifecycle management across all commands, (B) removing registry state management entirely, or (C) documenting current reality (registry is spawn cache only).

---

## Structured Uncertainty

**What's tested:**

- ✅ Registry state methods exist but aren't called (verified: grep search across codebase showed zero production usage)
- ✅ Lifecycle commands don't update registry (verified: code inspection of abandon_cmd.go, complete_cmd.go, clean_cmd.go)
- ✅ Spawn writes to registry (verified: found agentReg.Register() and agentReg.Save() calls in spawn_cmd.go)

**What's untested:**

- ⚠️ Whether registry was intentionally designed as spawn-only cache (no documentation found, assuming based on implementation pattern)
- ⚠️ Whether there are other commands beyond the 3 examined that DO update registry state (searched main lifecycle commands but didn't audit entire codebase)
- ⚠️ Whether fixing this inconsistently (updating only abandon) would break anything that expects stale registry (assumed safe but not validated)

**What would change this:**

- Finding would be wrong if there exist commands that DO call Abandon/Complete/Remove methods (would indicate selective usage not systemic pattern)
- Recommendation would change if kb context showed explicit decision to treat registry as spawn-only (would move from "document current reality" to "decision already documented")
- Severity assessment would change if registry staleness is causing production issues beyond this bug report (currently treating as design debt not urgent bug)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Document Current Reality: Registry as Spawn-Time Cache** - Accept that registry is a spawn-time snapshot for metadata lookups, not a lifecycle state tracker, and document this explicitly in code comments and architecture docs.

**Why this approach:**
- Aligns with what the system actually does (Finding 2: systemic "read but don't update" pattern)
- Minimal implementation risk - no behavior changes, just clarifies intent
- Preserves the working parts (registry helps with session ID lookups in abandon/status commands)
- Unblocks the immediate bug fix with clear justification: "registry is spawn cache, not lifecycle tracker"

**Trade-offs accepted:**
- Registry shows stale "active" status for completed agents (acceptable: status command derives actual state from OpenCode API + beads, not from registry alone)
- Registry's Abandon/Complete/Remove methods become dead code (acceptable: can mark as deprecated and remove in future cleanup)
- Doesn't solve the original bug report expectation (but clarifies why behavior is correct given design)

**Implementation sequence:**
1. Add doc comment to Registry struct explaining spawn-cache contract (pkg/registry/registry.go:76)
2. Mark Abandon/Complete/Remove methods as deprecated with comment explaining they're not integrated
3. Update SPAWN_CONTEXT notes if registry is mentioned (clarify its actual role)
4. Close bug orch-go-6a0p1 with explanation: registry is spawn cache, not lifecycle tracker; expected behavior

### Alternative Approaches Considered

**Option B: Implement Full Registry Lifecycle Management**
- **Pros:** Makes registry state accurate, fulfills original design intent, fixes all related staleness issues
- **Cons:** High implementation cost (update 3 commands: abandon, complete, clean), high risk (registry locking issues under concurrent operations), unclear value (complete/clean don't need registry today per Finding 2)
- **When to use instead:** If registry staleness is causing production issues beyond this bug, or if future features require accurate agent state tracking in registry

**Option C: Remove Registry Entirely**
- **Pros:** Eliminates dead code (state transition methods), removes confusion about registry's purpose, forces commands to use OpenCode API + beads as source of truth
- **Cons:** Breaks existing lookups in status/abandon commands (would need refactoring), requires migration plan for existing registry.json files, larger change scope than Option A
- **When to use instead:** If investigation reveals registry provides no value even for metadata lookups (not found in this investigation - status/abandon do use it for session ID lookups)

**Rationale for recommendation:** Option A (document current reality) addresses the investigation findings with minimal risk while unblocking the immediate bug. Finding 2 shows the system has a consistent pattern (spawn writes, others read), not broken behavior. Finding 3 shows staleness is acceptable because commands derive real state from OpenCode API + beads. Option B (full lifecycle) has high cost with unclear benefit (complete/clean work fine without registry updates). Option C (remove) is higher risk than needed (status/abandon rely on registry for session ID lookups per code inspection).

---

### Implementation Details

**What to implement first:**
- Add package-level doc comment to Registry struct clarifying "spawn-time cache" contract
- Mark Abandon(), Complete(), Remove() methods as deprecated in code comments
- Close bug orch-go-6a0p1 with explanation (not a bug, expected behavior given registry design)

**Things to watch out for:**
- ⚠️ Ensure bug close explanation is clear about WHY this isn't a bug (prevent reopening with same confusion)
- ⚠️ If marking methods deprecated, check if removing them would break any test assumptions (registry_test.go calls Abandon)
- ⚠️ Dylan may have use cases for registry state tracking not surfaced in this investigation - confirm before closing bug

**Areas needing further investigation:**
- Why were state transition methods designed but never integrated? (Could search git history for original intent)
- Are there other code paths that assume registry state is accurate? (Did lightweight search, could do deeper audit)
- Would removing deprecated methods break anything? (Low risk but could validate before cleanup)

**Success criteria:**
- ✅ Code comments clearly explain registry is spawn-time cache for metadata lookups
- ✅ Bug orch-go-6a0p1 closed with explanation that satisfies the orchestrator
- ✅ No confusion about registry's role in future development (documented contract prevents retry of same fix attempt)

---

## References

**Files Examined:**
- cmd/orch/abandon_cmd.go:64-326 - Analyzed runAbandon() to find missing agentReg.Abandon() call
- pkg/registry/registry.go:432-490 - Examined state transition method definitions
- cmd/orch/complete_cmd.go:1-1133 - Checked for registry imports and state updates
- cmd/orch/clean_cmd.go:1-1119 - Checked for registry imports and state updates
- cmd/orch/spawn_cmd.go - Confirmed registry write operations on spawn
- cmd/orch/status_cmd.go - Confirmed registry read operations for listing

**Commands Run:**
```bash
# Search for Abandon method calls in production code
rg "\.Abandon\(" --type go

# Search for Complete method calls (filtered out comments)
rg "\.Complete\(" --type go | grep -v "Phase: Complete"

# Search for all agentReg usage patterns
rg "agentReg\." --type go

# Search for registry design documentation
rg "Registry was designed" --type md
```

**External Documentation:**
- None referenced (investigation focused on codebase analysis)

**Related Artifacts:**
- **Issue:** orch-go-6a0p1 - Original bug report that triggered this investigation
- **Prior Knowledge:** SPAWN_CONTEXT.md lines 32-109 - Contains 11 registry investigations and constraints showing registry is "caching layer, not source of truth"

---

## Investigation History

**2026-01-11 Initial:** Investigation started
- Initial question: Is registry abandonment bug a simple oversight (missing function call) or symptom of deeper pattern?
- Context: Bug orch-go-6a0p1 reported that orch abandon doesn't remove agent from registry; strategic-first gate triggered due to 11 prior registry investigations

**2026-01-11 Finding:** Discovered systemic pattern across lifecycle commands
- All three lifecycle commands (abandon, complete, clean) exhibit same "read but don't update" pattern
- Registry state transition methods exist but are completely unused in production code

**2026-01-11 Complete:** Investigation completed
- Status: Complete
- Key outcome: This is NOT a simple fix - it's a half-implemented design pattern requiring architectural decision on registry's actual role
