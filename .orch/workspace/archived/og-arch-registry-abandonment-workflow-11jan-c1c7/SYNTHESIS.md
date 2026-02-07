# Session Synthesis

**Agent:** og-arch-registry-abandonment-workflow-11jan-c1c7
**Issue:** orch-go-6a0p1
**Duration:** 2026-01-11
**Outcome:** success

---

## TLDR

Investigated whether registry abandonment bug is a simple oversight or systemic pattern. Found it's a systemic issue: registry has state transition methods (Abandon, Complete, Remove) that are never called by lifecycle commands, making it a spawn-time cache that becomes permanently stale rather than a lifecycle tracker.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md` - Full investigation with findings, synthesis, and recommendations

### Files Modified
- None (investigation-only session, no code changes)

### Commits
- (Pending: will commit investigation file)

---

## Evidence (What Was Observed)

- Registry defines Abandon() (line 432), Complete() (line 450), Remove() (line 470) but grep shows ZERO production usage
- spawn_cmd.go calls agentReg.Register() and Save() (writes to registry)
- abandon_cmd.go calls agentReg.Find() but NEVER calls agentReg.Abandon() (reads but doesn't update)
- complete_cmd.go and clean_cmd.go don't even import pkg/registry (no interaction at all)
- Pattern is consistent across all lifecycle commands: write on spawn, read for lookups, never update state

### Tests Run
```bash
# Search for state transition method calls
rg "\.Abandon\(" --type go
# Result: Only found in registry_test.go, zero production usage

rg "\.Complete\(" --type go | grep -v "Phase: Complete"
# Result: Zero results

rg "agentReg\." --type go
# Result: Only Register, Find, List operations - no Abandon/Complete/Remove
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md` - Documents systemic pattern and recommends documenting registry as spawn-time cache

### Decisions Made
- Decision 1: This is NOT a simple fix requiring one function call, but a deeper pattern requiring architectural decision
- Decision 2: Recommend documenting current reality (registry as spawn cache) rather than implementing full lifecycle management

### Constraints Discovered
- Registry is effectively a spawn-time snapshot - any assumption that it tracks current agent state is false
- Complete and clean commands work fine without registry updates (derive state from OpenCode API + beads)
- Fixing this inconsistently (updating only abandon) would create confusion where abandonment updates registry but completion doesn't

### Externalized via `kb`
- Investigation document contains D.E.K.N. summary recommending promotion to decision
- Should create `kb quick decide` or formal decision if orchestrator accepts recommendation

---

## Next (What Should Happen)

**Recommendation:** close (with decision on registry's actual contract)

### Recommendation from Investigation

The investigation recommends **documenting current reality**: registry is a spawn-time cache for metadata lookups, not a lifecycle state tracker.

**Three options presented:**
1. **⭐ Document current reality** (recommended) - Clarify registry is spawn cache, mark state methods as deprecated
2. **Implement full lifecycle management** - Update all commands to call Abandon/Complete/Remove (high cost, unclear value)
3. **Remove registry entirely** - Eliminate confusion but breaks existing lookups in status/abandon

**Rationale for recommendation:**
- Aligns with actual implementation pattern (systemic "read but don't update")
- Minimal risk - no behavior changes, just clarifies intent
- Unblocks immediate bug with clear justification
- State methods become dead code (can deprecate/remove later)

**Next steps if recommendation accepted:**
1. Add doc comment to Registry struct explaining spawn-cache contract
2. Mark Abandon/Complete/Remove methods as deprecated
3. Close orch-go-6a0p1 with explanation (not a bug, expected behavior)
4. Optional: Create beads issue for future cleanup (remove deprecated methods)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why were state transition methods designed but never integrated? (Could search git history for original intent)
- Are there other code paths that assume registry state is accurate? (Did lightweight search, could do deeper audit)
- Would removing deprecated methods break anything beyond tests? (Low risk but could validate)

**Areas worth exploring further:**
- Git blame on registry.go to understand original design intent
- Broader audit of all registry usage across the codebase (investigation focused on main lifecycle commands)

**What remains unclear:**
- Whether Dylan has use cases for registry state tracking not surfaced in this investigation (should confirm before closing bug)
- Whether the "half-implemented" pattern was intentional (never needed full lifecycle) or accidental (started but never finished)

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4.5
**Workspace:** `.orch/workspace/og-arch-registry-abandonment-workflow-11jan-c1c7/`
**Investigation:** `.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md`
**Beads:** `bd show orch-go-6a0p1`
