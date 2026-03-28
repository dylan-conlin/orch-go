# Session Synthesis

**Agent:** og-research-thread-promote-doesn-28mar-dcad
**Issue:** orch-go-dy8ev
**Duration:** 2026-03-28 11:27 → 2026-03-28 11:35
**Outcome:** success

---

## Plain-Language Summary

When `orch thread promote` creates a model from a converged thread, it was creating `model.md` and `probes/` but not `claims.yaml`. This meant every newly promoted model was invisible to the claims infrastructure — orient couldn't surface edges, the daemon couldn't generate probe demand, and `orch complete` would crash when trying to update claims. The fix adds a seed `claims.yaml` (empty but properly structured) to the model scaffold, so the claims pipeline works from the moment a model is created.

## TLDR

Added seed `claims.yaml` creation to `scaffoldPromotionArtifact()` in the thread promote command. 8 lines of implementation, test-first approach with existing test suite extended.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/thread_cmd.go` - Added `claims` import and seed claims.yaml creation in `scaffoldPromotionArtifact()`
- `cmd/orch/thread_cmd_test.go` - Added claims.yaml existence and content assertions to `TestThreadPromoteCmd_Model` and `TestThreadPromoteCmd_Decision`

### Files Created
- `.kb/models/knowledge-accretion/probes/2026-03-28-probe-thread-promote-claims-yaml-gap.md` - Probe documenting the gap

---

## Evidence (What Was Observed)

- `scaffoldPromotionArtifact()` (thread_cmd.go:477) created model.md + probes/ but not claims.yaml
- `claims.ScanAll()` (claims.go:136) silently skips models without claims.yaml — newly promoted models invisible to orient
- `claims.LoadFile()` (claims.go:103) returns hard error on missing file — breaks completion pipeline
- `kb create model` (kb_create.go:18) has the same gap (separate issue)

### Tests Run
```bash
go test ./cmd/orch/ -run TestThreadPromoteCmd -count=1
# PASS: all 7 promote tests pass (0.315s)

go test ./pkg/claims/ ./pkg/thread/ -count=1
# PASS: both packages clean
```

---

## Architectural Choices

No architectural choices — fix used existing `claims.SaveFile()` from pkg/claims.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Model directories need three files to be a complete knowledge unit: model.md, claims.yaml, probes/. Two code paths create these directories (thread promote, kb create model) and both were missing claims.yaml.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing
- [x] Probe has Status: Complete
- [x] Ready for `orch complete orch-go-dy8ev`

---

## Unexplored Questions

- Should `kb create model` also get the same fix? (Filed as discovered work)
- Should there be a lint check that every model directory has claims.yaml?

---

## Friction

No friction — smooth session.

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-research-thread-promote-doesn-28mar-dcad/`
**Beads:** `bd show orch-go-dy8ev`
