# Session Synthesis

**Agent:** og-inv-probe-harness-engineering-22mar-c823
**Issue:** orch-go-9ld9s
**Outcome:** success

---

## Plain-Language Summary

The harness-engineering model has 13 claims, all about behavioral enforcement — how to constrain agent behavior with gates, hooks, and rules. Zero claims address the complementary strategy: constraining the problem surface so enforcement isn't needed. This is a structural blind spot. The model already USES 7 problem-constraint patterns (spawn tiers, verification levels, --explore decomposition, daemon routing, hotspot extraction, domain harnesses, issue type scoping) without naming them. autoresearch (48k stars, 1 file, 1 metric, zero governance) proves the extreme case: when the problem is tightly constrained, all enforcement machinery becomes unnecessary. New claim HE-14, section 9, invariant 8, and constraint added to the model.

---

## TLDR

Probed the harness-engineering model for coverage of problem-surface constraints (narrowing what agents work on). Found 0/13 claims covering this strategy despite 7 existing orch-go patterns that use it. Extended the model with HE-14, new section, invariant, and constraint.

---

## Delta (What Changed)

### Files Created
- `.kb/models/harness-engineering/probes/2026-03-22-probe-problem-surface-constraints-blind-spot.md` — Probe documenting the blind spot with evidence table and proposed claim

### Files Modified
- `.kb/models/harness-engineering/model.md` — Added §9 (Problem-Surface Constraints), Critical Invariant #8, Constraint (Why Narrow Problem Before Enforcement), Evolution entry, Probe reference
- `.kb/models/harness-engineering/claims.yaml` — Added HE-14 claim with 5 evidence sources, falsification criteria, and 3 tension mappings

---

## Evidence (What Was Observed)

- **0/13 claims about problem constraints:** Full text search and manual claim-by-claim review confirmed zero coverage of problem-surface constraint strategy
- **7 unnamed patterns found:** Codebase search verified spawn tiers (`pkg/spawn/config.go:17-47`), verify levels (`pkg/spawn/verify_level.go:6-55`), --explore (`cmd/orch/spawn_cmd.go:243-264`), daemon routing (`pkg/daemon/coordination.go:40-98`), hotspot extraction (`pkg/daemon/coordination.go:47-77`), domain harness (`.harness/openscad/CLAUDE.md:124-132`), issue type scoping (`pkg/spawn/verify_level.go:46-55`)
- **V0 eliminates ~11/14 gates:** Light-tier tasks with V0 verification skip synthesis, test evidence, build, git diff, and behavioral gates — implicit problem constraint
- **Fowler quote dual interpretation:** Line 142 says "constraining the solution space" — model adopted "add enforcement" interpretation exclusively, missing "simplify the problem" interpretation
- **autoresearch (external evidence):** 1,225 lines total, zero governance, 48k stars — constraint design IS the architecture

### Tests Run
```bash
# Text searches confirming zero problem-constraint coverage
grep -i "problem.*(constraint|surface|scope|narrow|simplif)" model.md  # 0 matches
grep -i "light|tier.*light|single.file|domain.harness" model.md  # 0 matches

# Code verification of 7 patterns
# Confirmed via Read tool: config.go, coordination.go, verify_level.go, spawn_cmd.go, CLAUDE.md
```

---

## Architectural Choices

No architectural choices — task was investigation/probe, not implementation.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/harness-engineering/probes/2026-03-22-probe-problem-surface-constraints-blind-spot.md` — Probe with full evidence table

### Constraints Discovered
- The harness-engineering model's taxonomy (hard/soft) has a gap: problem-surface constraints are neither hard nor soft — they work by making enforcement unnecessary, not harder or softer
- The Fowler quote "constraining the solution space" supports BOTH adding enforcement AND simplifying problems — model previously only used one interpretation

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification criteria.

Key outcomes:
- Probe file exists with all 4 required sections (Question, What I Tested, What I Observed, Model Impact)
- Model updated with HE-14 claim, §9, invariant #8, constraint
- claims.yaml updated with full HE-14 entry including falsification criteria

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Probe file created with all required sections
- [x] Model merged with probe findings
- [x] claims.yaml updated
- [x] Ready for `orch complete orch-go-9ld9s`

---

## Unexplored Questions

- **Can V0-level problem constraints be applied more broadly?** V0 already eliminates ~11/14 gates. What other task types could be constrained tightly enough to use V0?
- **Would an "orch optimize" tight-loop mode work?** autoresearch's pattern (1 file, 1 metric, keep/discard) could be a new orch-go command for narrow optimization tasks
- **How does the domain harness (OpenSCAD) interact with the general completion pipeline?** Currently runs both domain 5-layer + general 14-gate — could problem constraints in the domain harness reduce general pipeline overhead?

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-probe-harness-engineering-22mar-c823/`
**Probe:** `.kb/models/harness-engineering/probes/2026-03-22-probe-problem-surface-constraints-blind-spot.md`
**Beads:** `bd show orch-go-9ld9s`
