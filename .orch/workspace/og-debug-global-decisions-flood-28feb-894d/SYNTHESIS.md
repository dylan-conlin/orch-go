# Session Synthesis

**Agent:** og-debug-global-decisions-flood-28feb-894d
**Issue:** orch-go-9ksl
**Outcome:** success

---

## Plain-Language Summary

When agents spawn into non-orch projects (toolshed, price-watch, kb-cli), `kb context` was flooding them with 15-30 irrelevant global decisions from `~/.kb/` that matched on common words like "fix", "design", "strategic". The fix adds a `MinGlobalScore = 3.0` threshold to `getGlobalStoreContext()` in kb-cli, requiring global results to have at least a title match, filename match, or multiple meaningful body matches to survive. This reduces global noise from 15-30 results to 1-3 while preserving genuinely relevant global knowledge (e.g., "premise before solution" at score 12.62 still appears when searched for directly).

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## TLDR

Fixed global decision flooding in kb context by adding MinGlobalScore threshold (3.0) to `getGlobalStoreContext()`. Results: "fix" query drops from 15→1 global results, "rate limiting api" from 17→1, while genuinely relevant results (title/filename matches) still appear.

---

## Delta (What Changed)

### Files Modified
- `kb-cli/cmd/kb/context.go` - Added `MinGlobalScore` constant (3.0), `filterArtifactsByMinScore()` helper, applied filtering to all artifact types in `getGlobalStoreContext()`
- `kb-cli/cmd/kb/context_test.go` - Updated 2 existing tests (test files now have keywords in titles to score above threshold), added 3 new tests (`TestGetGlobalStoreContextFiltersLowScoreResults`, `TestFilterArtifactsByMinScore`, `TestFilterArtifactsByMinScoreEmptyInput`, `TestMinGlobalScoreConstant`)
- `kb-cli/build/kb` - Rebuilt binary with fix

---

## Evidence (What Was Observed)

- **Before fix**: `kb context "fix"` from kb-cli returned 15 global results (8 decisions, 3 guides, 3 models, 1 investigation), all scoring below 3.28
- **Before fix**: `kb context "rate limiting api"` returned 17 global results, most scoring below 1.5
- **After fix**: `kb context "fix"` returns 1 global result (score 3.28 - Strategic-First Orchestration, which has high keyword coverage)
- **After fix**: `kb context "rate limiting api"` returns 1 global result (score 12.45 - Claude Code Cross-Account Rate Limit Bug, genuinely relevant)
- **After fix**: `kb context "premise before solution"` correctly returns the relevant global decision at score 12.62

### Tests Run
```bash
go test ./cmd/kb/ -run "TestContext|TestGetGlobalStore|TestFilterArtifacts|TestMinGlobalScore|TestSortArtifacts" -v
# 22 tests passed, 0 failed
```

---

## Architectural Choices

### MinGlobalScore = 3.0 threshold applied uniformly to all global artifact types
- **What I chose:** Single constant threshold at 3.0 for all global results (decisions, investigations, guides, models)
- **What I rejected:** (1) Different thresholds per artifact type, (2) Only filtering stemmed-only matches like local `SearchArtifacts`
- **Why:** A uniform threshold is simpler and the scoring system already provides differentiation (title match = 10.0+, filename = 3.0, body = ~1.0). The `StemmedOnly` filter from `SearchArtifacts` wouldn't help here because common words like "fix" are exact matches (not stemmed), yet still noise. Score 3.0 requires at least a filename match or title match — single body-only matches can't reach it.
- **Risk accepted:** Global results with score 2.0-2.99 will be filtered even if borderline relevant. This is acceptable because local project results always take priority and truly relevant global results score much higher (10+).

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Global store results need fundamentally different filtering than local results because they match on domain-generic vocabulary that provides zero project-specific value

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (22/22 context tests)
- [x] Binary rebuilt and installed
- [x] Ready for `orch complete orch-go-9ksl`

---

## Unexplored Questions

- Should the `mergeContextResults()` function re-sort combined results by score before truncating? Currently local results always come first regardless of score. A score-aware merge could surface high-scoring global results alongside lower-scoring local ones.
- Should `MinGlobalScore` be configurable via `~/.orch/config.yaml`?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-global-decisions-flood-28feb-894d/`
**Beads:** `bd show orch-go-9ksl`
