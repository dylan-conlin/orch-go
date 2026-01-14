# Session Synthesis

**Agent:** og-arch-analyze-serve-agents-04jan
**Issue:** orch-go-25s2.2
**Duration:** 2026-01-04 ~10:30 → 2026-01-04 ~11:00
**Outcome:** success

---

## TLDR

Validated and refined the prior investigation's 2-phase extraction plan for serve_agents.go (1399 lines). Recommend splitting into 3 files: serve_agents_cache.go (~470 lines), serve_agents_events.go (~230 lines), and core serve_agents.go (~700 lines). Added 2 feature items (feat-027, feat-028) to .orch/features.json.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-inv-analyze-serve-agents-go-1399.md` - Validated extraction plan with exact line counts

### Files Modified
- `.orch/features.json` - Added feat-027 (Phase 1: cache extraction) and feat-028 (Phase 2: events extraction)

### Commits
- (To be committed)

---

## Evidence (What Was Observed)

- Line count verified: `wc -l cmd/orch/serve_agents.go` returns 1399 (matches prior investigation)
- Function declarations verified via `grep -n "^func\|^type\|^const\|^var"` - 35 declarations total
- workspaceCache type is 220 lines (lines 277-496), larger than prior estimate of ~75 lines
- handleAgents function is 451 lines (lines 499-950), confirmed as "god handler" that shouldn't be split
- Events handlers (lines 954-1195) are completely independent of caching and agent logic

### Key Finding
Prior investigation estimated ~350 lines for cache extraction. Actual analysis shows ~470 lines because workspaceCache (220 lines) was partially counted in "core" when it belongs with caching infrastructure.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-inv-analyze-serve-agents-go-1399.md` - Refined extraction plan with exact line ranges

### Decisions Made
- Decision: Keep `handleCacheInvalidate` in core file because it's an HTTP handler that touches both cache types
- Decision: Include workspaceCache with beadsCache in cache file because both are performance optimization infrastructure
- Decision: Response types (AgentAPIResponse, GapAPIResponse, SynthesisResponse) stay in core because they're the API contract

### Constraints Discovered
- Constraint: `globalBeadsCache` is initialized in `runServe()` in serve.go - initialization must stay there
- Constraint: `newBeadsCache()` is called from serve.go - must remain exported (already is)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created, features.json updated)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-25s2.2`

### Implementation Guidance for feat-027 (Phase 1)
**Skill:** feature-impl
**Context:**
```
Extract ~470 lines from serve_agents.go into serve_agents_cache.go.
Move: beadsCache (lines 27-227), workspaceCache (lines 277-496), TTL constants (88-92), globals (62-64, 124).
Verify: go build ./cmd/orch/ && go test ./cmd/orch/
See: .kb/investigations/2026-01-04-inv-analyze-serve-agents-go-1399.md for exact line ranges.
```

### Implementation Guidance for feat-028 (Phase 2)
**Skill:** feature-impl  
**Context:**
```
Extract ~230 lines from serve_agents.go into serve_agents_events.go.
Move: handleEvents, handleAgentlog, handleAgentlogJSON, handleAgentlogSSE, readLastNEvents (lines 954-1195).
Depends on feat-027 completing first.
Verify: go build ./cmd/orch/ && go test ./cmd/orch/
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should serve_agents_cache_test.go be created, or keep tests unified in serve_agents_test.go?
- Is there value in extracting caching to pkg/cache/ for reuse? (Currently only serve_agents uses it)

**What remains unclear:**
- Whether Go compiler initialization order needs special handling when splitting globals across files (likely fine with package-level vars)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-analyze-serve-agents-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-inv-analyze-serve-agents-go-1399.md`
**Beads:** `bd show orch-go-25s2.2`
