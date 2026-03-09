# Session Synthesis

**Agent:** og-inv-natural-orphan-baseline-09mar-70d3
**Issue:** orch-go-80rg8
**Duration:** 2026-03-09
**Outcome:** success

---

## Plain-Language Summary

The knowledge-physics model claimed 85.5% of investigations are orphaned and this signals systemic under-synthesis. This probe found that rate is misleading — 83% of the corpus comes from before the model/probe system existed, when orphaning was structurally guaranteed. In the model era (Feb-Mar 2026), the orphan rate is 52%, and ~80% of those orphans are naturally expected (implementation work filed as investigations, one-off audits, design docs, negative results). Only ~10% of total investigations represent genuine knowledge loss — findings that should have fed a model but didn't. The probe system is the real fix: when agents create probes instead of standalone investigations, the findings are structurally connected to their parent model by directory placement. This dropped investigation volume 76% while increasing model-connected work. The natural orphan baseline for an exploratory system is 40-50% — analogous to 5-15% dead code being normal in codebases.

---

## TLDR

Decomposed the 85.5% knowledge orphan rate into six categories and found the natural baseline is 40-50%. The headline rate is inflated by pre-model era artifacts; model-era rate is 52%. The actionable signal is "genuinely lost" knowledge at ~10% of investigations, not the raw orphan rate.

---

## Delta (What Changed)

### Files Created
- `.kb/models/knowledge-physics/probes/2026-03-09-probe-natural-orphan-baseline-categorization.md` — Full probe with orphan taxonomy, era-adjusted rates, and natural baseline analysis

### Files Modified
- `.kb/models/knowledge-physics/model.md` — Updated invariants #2, #3, #4; answered open question #1; partially answered #3; added probe reference and evolution entry

### Commits
- (pending) Knowledge physics probe and model update

---

## Evidence (What Was Observed)

- 1,166 total investigations: 969 pre-model era (Dec 2025 - Jan 2026), 196 model era (Feb-Mar 2026)
- Pre-model orphan rate: 94.7% — structurally impossible to connect to models that didn't exist
- Model-era orphan rate: 52.0% — within healthy range
- 35-file sample categorization: ~80% natural orphans (implementation, audit, design, exploratory, negative results), ~20% genuinely lost
- Probe displacement effect: investigations dropped from 548/month (Jan) to 129/month (Feb) as probes rose to 160/month
- 189 total probes with 122 confirms, 165 extends, 57 contradicts verdicts

### Tests Run
```bash
# Era-adjusted orphan measurement
find .kb/investigations -name "*.md" | wc -l  # 1166
grep -roh '\.kb/investigations/[^)| "]*\.md' .kb/ | sort -u | wc -l  # 885 referenced
comm -23 /tmp/all_inv.txt /tmp/all_referenced_inv.txt | wc -l  # 1021 strict orphans

# Model-era breakdown
# Pre-model: 918/969 = 94.7% orphan
# Model-era: 102/196 = 52.0% orphan

# Filename pattern categorization of 1,021 orphans
# Implementation patterns: 152 (14.9%)
# Fix/debug: 114 (11.2%)
# Genuine investigation: 211 (20.7%)
# Housekeeping: 44 (4.3%)
# Update/change: 28 (2.7%)
```

---

## Architectural Choices

No architectural choices — this was an investigation/probe, not implementation work.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/knowledge-physics/probes/2026-03-09-probe-natural-orphan-baseline-categorization.md` — Orphan taxonomy and healthy baseline

### Constraints Discovered
- The raw orphan rate is a misleading metric because it mixes eras with fundamentally different structural properties
- "Genuinely lost" rate (~10%) is a better health signal than raw orphan rate (~87%)

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for commands run, expectations, and outcomes.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (probe file, model updates)
- [x] Probe-to-model merge done
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-80rg8`

---

## Unexplored Questions

- **Could a `--model` flag on `kb create investigation` reduce the orphan rate below 40%?** The probe system proves structural coupling works. Applying the same principle to investigations could further reduce genuinely lost findings.
- **What's the "genuinely lost" rate trending over time?** Sampling 10 files per month could track whether the rate is stable, increasing, or decreasing.
- **Implementation-as-investigation routing problem** — 30-45% of orphans are implementation work. Better skill routing (daemon identifying "add X" tasks as feature-impl, not investigation) would mechanically reduce the orphan rate.

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-natural-orphan-baseline-09mar-70d3/`
**Probe:** `.kb/models/knowledge-physics/probes/2026-03-09-probe-natural-orphan-baseline-categorization.md`
**Beads:** `bd show orch-go-80rg8`
