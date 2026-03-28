# Probe: Thread Promote Claims.yaml Gap

**Model:** knowledge-accretion
**Date:** 2026-03-28
**Status:** Complete
**claim:** KA-03
**verdict:** extends

---

## Question

Does the thread promotion lifecycle correctly bootstrap all knowledge unit artifacts? Specifically, the knowledge-accretion model predicts that model directories are knowledge units containing model.md (understanding), claims.yaml (assertions), and probes/ (evidence). Does `orch thread promote` create the complete unit?

---

## What I Tested

```bash
# 1. Read scaffoldPromotionArtifact() in cmd/orch/thread_cmd.go:477-499
# 2. Ran the existing TestThreadPromoteCmd_Model test
go test ./cmd/orch/ -run TestThreadPromoteCmd_Model -v

# 3. Checked for claims.yaml after promotion
# Result: model.md created, probes/ created, claims.yaml NOT created

# 4. Verified downstream failure path: claims.LoadFile() in pkg/claims/claims.go:103
# returns "no such file or directory" when claims.yaml is missing
```

---

## What I Observed

- `scaffoldPromotionArtifact()` created `model.md` and `probes/` but NOT `claims.yaml`
- The `kb create model` command (`cmd/orch/kb_create.go:18`) has the same gap — also missing claims.yaml
- Downstream consumers (`orch complete` claim pipeline, daemon probe generation, orient edge surfacing) all call `claims.LoadFile()` which fails hard on missing file
- `claims.ScanAll()` silently skips models without claims.yaml, which means newly promoted models are invisible to orient edges and daemon probe demand

**Fix applied:** Added claims.yaml seed creation to `scaffoldPromotionArtifact()` using `claims.SaveFile()`. Creates a properly structured empty claims file (model name, version 1, today's date, empty claims array).

---

## Model Impact

- [ ] **Confirms** invariant: knowledge units require model.md + claims.yaml + probes/ as a triple
- [x] **Extends** model with: The promotion lifecycle had an incomplete bootstrap — claims.yaml was never part of the scaffold, creating a gap between artifact creation and claim infrastructure readiness. This extends KA-03 (knowledge must be structured for machine consumption) by showing that the gap doesn't just affect content quality — it breaks downstream automation entirely.

---

## Notes

- The `kb create model` command has the same bug (no claims.yaml creation). Filed as discovered work.
- The fix is minimal (8 lines) because the `claims` package already has `SaveFile()` — the plumbing existed, the scaffold just didn't call it.
