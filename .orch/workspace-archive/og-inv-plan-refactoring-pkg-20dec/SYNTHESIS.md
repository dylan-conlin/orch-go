# Session Synthesis

**Agent:** og-inv-plan-refactoring-pkg-20dec
**Issue:** orch-go-n9h
**Duration:** 2025-12-20
**Outcome:** success

---

## TLDR

Investigated how to refactor pkg/registry to cache Beads issue state. Recommended extending Agent struct with TTL-cached Phase/Issue fields to reduce bd CLI calls from ~9 per operation to ~2-3, with highest impact on `orch wait` polling loops (360 → 36 calls for 30-min wait).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-20-inv-plan-refactoring-pkg-registry-act.md` - Comprehensive refactoring plan with implementation recommendations

### Files Modified
- None (investigation only, no implementation)

### Commits
- (pending) - Investigation file to be committed

---

## Evidence (What Was Observed)

- Found 9 distinct `exec.Command("bd", ...)` calls across codebase (verified via rg)
- `pkg/registry/registry.go:37-60` Agent struct does not cache any Beads state
- `cmd/orch/wait.go:164` polls `GetPhaseStatus()` every 5 seconds in a loop
- `cmd/orch/review.go:91` calls `VerifyCompletion()` for each completed agent
- Registry already has file locking (`pkg/registry/registry.go:243-258`) and merge semantics (`pkg/registry/registry.go:262-298`)

### Tests Run
```bash
# Found all bd CLI call sites
rg 'exec\.Command\("bd"' --type go -A 1
# Result: 9 locations in 5 files

# Verified verify function usage patterns
rg "verify\.(GetIssue|GetPhaseStatus|VerifyCompletion)" --type go -A 2
# Result: 7 call sites showing polling and batch access patterns
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-20-inv-plan-refactoring-pkg-registry-act.md` - Full refactoring plan

### Decisions Made
- Decision 1: TTL-based caching in Agent struct (vs separate cache layer) because it leverages existing registry infrastructure
- Decision 2: Cache Phase + Issue status fields (vs full comment history) because these are the actual consumption patterns
- Decision 3: 30-second TTL recommended (start value, to be tuned) because it balances freshness vs call reduction

### Constraints Discovered
- Registry uses file locking for concurrent access - cache updates must respect this pattern
- Agents without BeadsID (--no-track spawns) cannot be cached - need to handle gracefully
- Cross-process cache consistency is a concern - stale cache in one process could conflict

### Externalized via `kn`
- (none - findings are captured in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with refactoring plan)
- [x] Tests passing (investigation, not implementation)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-n9h`

### Follow-up Work
The investigation recommends a concrete implementation path. If desired:

**Issue:** Implement Beads state caching in pkg/registry
**Skill:** feature-impl
**Context:**
```
Extend Agent struct with CachedPhase, CachedPhaseSummary, PhaseCheckedAt fields.
Add GetCachedPhase(beadsID, maxAge) method with TTL-based refresh.
Update orch wait/review to use cache. See investigation file for full design.
```

---

## Session Metadata

**Skill:** investigation
**Model:** (agent-determined)
**Workspace:** `.orch/workspace/og-inv-plan-refactoring-pkg-20dec/`
**Investigation:** `.kb/investigations/2025-12-20-inv-plan-refactoring-pkg-registry-act.md`
**Beads:** `bd show orch-go-n9h`
