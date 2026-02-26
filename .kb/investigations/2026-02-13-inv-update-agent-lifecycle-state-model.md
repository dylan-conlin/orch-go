## Summary (D.E.K.N.)

**Delta:** Added state vs infrastructure distinction to the agent lifecycle state model, reframing the four-layer table with explicit categories and explaining why conflating them creates reconciliation burden.

**Evidence:** Model file updated with Category column, new section on state vs infrastructure, workspace files as explicit state layer, and three-bucket ownership model reference.

**Knowledge:** The reconciliation burden (phantom agents, ghost sessions, orphan infrastructure) stems from treating infrastructure layers as authoritative state — they should only be consulted as fallback after state layers.

**Next:** Close — model updated per decision doc requirements.

**Authority:** implementation - Additive documentation update within existing model, no architectural changes.

---

# Investigation: Update Agent Lifecycle State Model

**Question:** How should the agent lifecycle state model be updated to distinguish state layers from infrastructure layers?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` | implements | Yes - read decision doc | None |
| `.kb/models/agent-lifecycle-state-model/model.md` (pre-update) | extends | Yes - read model | None |

## Findings

### Finding 1: Four-layer table needed Category column

**Evidence:** Original table had Layer/Storage/Lifecycle/What It Knows/Authority Level columns but no categorization of state vs infrastructure.

**Source:** `.kb/models/agent-lifecycle-state-model/model.md:21-27` (pre-update)

**Significance:** Without explicit categorization, readers can't tell which layers orch should own vs merely use.

---

### Finding 2: Workspace files were missing as an explicit layer

**Evidence:** The original four-layer model listed beads, OpenCode on-disk, OpenCode in-memory, and tmux — but workspace files (`.orch/workspace/`) are a distinct state layer containing SPAWN_CONTEXT, SYNTHESIS, .tier files.

**Source:** Decision doc explicitly lists workspace files as a state layer alongside beads.

**Significance:** Workspace files are persistent, orch-controlled artifacts that survive infrastructure restarts. They belong in the state category.

---

### Finding 3: Reconciliation burden explanation was missing

**Evidence:** The model documented failure modes (phantom agents, ghost sessions) but didn't explain the root cause: treating infrastructure as state.

**Source:** Decision doc Section "The Core Reframe: State vs Infrastructure"

**Significance:** Naming the pattern (infrastructure-as-state) gives future agents a framework to understand why these failure modes exist and how the ownership model addresses them.

---

## Synthesis

**Key Insights:**

1. **State vs infrastructure is a categorization, not restructuring** - The existing four layers remain; the new Category column labels them without changing the model's structure.

2. **Workspace files deserve explicit recognition** - They were implicitly part of the model but not called out as a layer. Adding them makes the state story complete.

3. **The ownership model connects to the reconciliation burden** - Own/Accept/Lobby directly maps to what orch should control vs what it should work around vs what it should push upstream.

**Answer to Investigation Question:**

The model was updated by: (1) adding a Category column to the four-layer table labeling each as State or Infrastructure, (2) adding workspace files as an explicit state layer, (3) adding a new "State vs Infrastructure" section explaining why the distinction matters and summarizing the Own/Accept/Lobby ownership model, (4) updating the Constraints section to group layers by category, and (5) referencing the lifecycle ownership boundaries decision.

---

## References

**Files Examined:**
- `.kb/models/agent-lifecycle-state-model/model.md` - The model being updated
- `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` - The decision driving this update

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` - Source of state vs infrastructure reframe
