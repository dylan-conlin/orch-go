# Session Synthesis

**Agent:** og-debug-fix-gap-analysis-27feb-7b71
**Issue:** orch-go-3go1
**Outcome:** success

---

## Plain-Language Summary

Gap analysis was giving high quality scores (90-95/100) even when all the knowledge injected into a spawn came from the wrong project. For example, a toolshed agent building a pricing panel would receive orch-go's dashboard architecture knowledge and gap analysis would report "quality: 90/100, no gaps" because it only checked whether categories (models, decisions, constraints) were populated, not whether the content was actually from the target project.

The fix adds project-relevance checking: `AnalyzeGaps` now accepts a `projectDir` parameter and checks whether each match's file path belongs to the target project. Wrong-project matches are excluded from quality scoring and flagged with a new `wrong_project` gap type. All-wrong-project spawns now correctly score 0/100 instead of 90+/100, giving operators a clear signal that the spawn context is degraded.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace directory.

Key outcomes:
- `TestAnalyzeGaps_WrongProject`: 4/5 matches wrong-project → quality <=20, critical gap
- `TestAnalyzeGaps_AllWrongProject`: all wrong → quality 0
- `TestAnalyzeGaps_MixedCorrectAndWrong`: 1/5 wrong → warning gap, quality >=40
- All existing tests pass unchanged (backward compatible)
- Full build and vet clean

---

## Delta (What Changed)

### Files Modified
- `pkg/spawn/gap.go` - Added `GapTypeWrongProject`, `WrongProjectCount` to MatchStatistics, project-relevance checking in `AnalyzeGaps`, proportional quality penalty in `calculateContextQuality`, new functions `countWrongProjectMatches` and `isWrongProjectMatch`
- `pkg/spawn/gap_test.go` - Added 6 new test functions covering wrong-project detection
- `pkg/orch/extraction.go` - Updated 3 `AnalyzeGaps` call sites to pass `projectDir`
- `pkg/spawn/skill_requires.go` - Updated 2 `AnalyzeGaps` call sites to pass `projectDir`

### Files Created
- `.kb/models/spawn-architecture/probes/2026-02-27-probe-gap-analysis-wrong-project-false-positive.md`

---

## Evidence (What Was Observed)

- The probe from earlier today confirmed: toolshed-74 scored quality 90% with 100% orch-go knowledge
- `isWrongProjectMatch` correctly distinguishes: target project paths (not flagged), global ~/.kb/ paths (not flagged), other project .kb/ paths (flagged), paths without .kb/ (not flagged), no-path kn entries (not flagged)
- `calculateContextQuality` with WrongProjectCount=5, TotalMatches=5 returns 0
- `calculateContextQuality` with WrongProjectCount=3, TotalMatches=6 returns 20-55 range (proportional)
- `calculateContextQuality` with WrongProjectCount=0 returns same values as before the change

### Tests Run
```bash
go test ./pkg/spawn/ -count=1  # PASS (0.468s) - 100+ tests
go test ./pkg/orch/ -count=1   # PASS (0.010s)
go test ./cmd/orch/ -count=1   # PASS (6.645s)
go build ./cmd/orch/           # OK
go vet ./cmd/orch/ ./pkg/spawn/ ./pkg/orch/  # OK
```

---

## Architectural Choices

### Path-based detection only (not title-based)
- **What I chose:** Detect wrong-project matches by checking if `KBContextMatch.Path` is under `projectDir`
- **What I rejected:** Also checking `[project]` prefix in titles for non-path matches
- **Why:** kn entries (constraints, decisions) don't have paths. Trying to infer project from title content would be brittle and produce false positives. The path-based check catches models, guides, and investigations — which are the heaviest wrong-project injection. kn entries are lightweight and the CWD fix (separate issue) is the proper fix for those.
- **Risk accepted:** kn entries from wrong project won't be detected. Mitigation: CWD fix is the root cause fix; this is defense-in-depth.

### Proportional penalty instead of binary pass/fail
- **What I chose:** Scale quality score based on `effectiveMatches/totalMatches` ratio
- **What I rejected:** Binary: any wrong-project match → quality=0
- **Why:** A single wrong-project match among many correct ones shouldn't zero out the score. The severity (warning vs critical) scales with the ratio, giving operators proportional signal.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/spawn-architecture/probes/2026-02-27-probe-gap-analysis-wrong-project-false-positive.md`

### Constraints Discovered
- kn entries (constraints, decisions) don't carry project-path metadata, making them invisible to project-relevance checking

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (new + existing)
- [x] Probe file created
- [x] Ready for `orch complete orch-go-3go1`

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-fix-gap-analysis-27feb-7b71/`
**Beads:** `bd show orch-go-3go1`
