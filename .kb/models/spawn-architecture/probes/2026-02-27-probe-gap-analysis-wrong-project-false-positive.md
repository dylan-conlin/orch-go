# Probe: Gap Analysis Wrong-Project False Positive Fix

**Model:** spawn-architecture
**Date:** 2026-02-27
**Status:** Complete

---

## Question

The Feb 27 cross-repo spawn context quality audit (probe) found that gap analysis scores 90-95% quality when 100% of matches are from the wrong project. Does adding project-relevance checking to `AnalyzeGaps` correctly detect wrong-project knowledge injection and reduce the quality score?

---

## What I Tested

### Test 1: Reproduction of original bug scenario

```go
// Simulate toolshed spawn receiving orch-go knowledge (exact scenario from probe)
result := &KBContextResult{
    Matches: []KBContextMatch{
        {Type: "model", Path: "/home/user/projects/orch-go/.kb/models/spawn-architecture/model.md"},
        {Type: "model", Path: "/home/user/projects/orch-go/.kb/models/dashboard/model.md"},
        {Type: "guide", Path: "/home/user/projects/orch-go/.kb/guides/model-selection.md"},
        {Type: "decision", Path: "/home/user/projects/orch-go/.kb/decisions/2026-02-01-slide-out.md"},
        {Type: "constraint", Title: "Dashboard max-h-64"},  // kn entry, no path
    },
}
analysis := AnalyzeGaps(result, "pricing strategy", "/home/user/projects/toolshed")
```

**Before fix:** ContextQuality=90, HasGaps=false
**After fix:** ContextQuality<=20, HasGaps=true, GapTypeWrongProject=critical, WrongProjectCount=4

### Test 2: All wrong-project matches → quality 0

```bash
go test ./pkg/spawn/ -run TestAnalyzeGaps_AllWrongProject -v
# PASS: ContextQuality=0 when all 3 matches from wrong project
```

### Test 3: Mixed correct/wrong → proportional penalty

```bash
go test ./pkg/spawn/ -run TestAnalyzeGaps_MixedCorrectAndWrong -v
# PASS: 1 of 5 wrong → WrongProjectCount=1, severity=warning, quality >= 40
```

### Test 4: Global ~/.kb/ knowledge not flagged

```bash
go test ./pkg/spawn/ -run TestIsWrongProjectMatch -v
# PASS: Global .kb path correctly treated as acceptable
```

### Test 5: Empty projectDir → detection skipped

```bash
go test ./pkg/spawn/ -run TestAnalyzeGaps_NoProjectDir -v
# PASS: WrongProjectCount=0 when no projectDir (backward compatible)
```

### Full test suite
```bash
go test ./pkg/spawn/ -count=1  # PASS (0.468s)
go test ./pkg/orch/ -count=1   # PASS (0.010s)
go test ./cmd/orch/ -count=1   # PASS (6.645s)
go build ./cmd/orch/           # OK
go vet ./cmd/orch/ ./pkg/spawn/ ./pkg/orch/  # OK
```

---

## What I Observed

1. **Path-based detection works**: Matches with `.kb/` paths from a different project than `projectDir` are correctly flagged. Matches from global `~/.kb/` are correctly excluded. Matches without paths (kn entries) are given benefit of the doubt.

2. **Scoring is proportional**: `calculateContextQuality` now uses `effectiveMatches = TotalMatches - WrongProjectCount` for base points, and applies a `relevanceRatio` penalty to category bonuses. All-wrong-project → 0, half-wrong → roughly half, no-wrong → unchanged.

3. **Severity scales with ratio**: >50% wrong-project → critical, <=50% → warning. This matches the real-world severity: a majority-wrong context is fundamentally broken.

4. **Backward compatible**: When `projectDir=""` (same-project spawns), wrong-project detection is completely skipped. All existing tests pass without modification (they pass `""` for projectDir).

---

## Model Impact

- [x] **Extends** model with: Gap analysis now checks content relevance in addition to category population. The `AnalyzeGaps` function accepts `projectDir` and uses path-based checking to detect wrong-project knowledge injection. Matches with `.kb/` paths from different projects are counted as `WrongProjectCount` and excluded from quality scoring. This closes the false-positive gap where cross-repo spawns could score 90%+ quality with 100% irrelevant content.

---

## Notes

- This fix is defense-in-depth alongside the CWD fix (orch-go-1goo/16a1). The CWD fix prevents wrong-project knowledge from being queried. This fix detects it if it gets through.
- kn entries (constraints, decisions) without Path fields cannot be checked for project relevance. If the CWD bug injects wrong-project kn entries, gap analysis won't detect those specifically. But the path-based matches (models, guides, investigations) are typically the heaviest wrong-project injection, and detecting those is sufficient for a quality signal.
- The `GapAPIResponse` in the dashboard API now includes `WrongProjectCount` in match stats, giving operators visibility into wrong-project injection via the web UI.
