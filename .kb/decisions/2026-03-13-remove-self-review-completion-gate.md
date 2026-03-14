# Decision: Remove self_review Completion Gate

**Date:** 2026-03-13
**Status:** Accepted
**Issue:** orch-go-ntkcz

## Context

The `self_review` gate (V1 completion gate) ran 4 automated checks at `orch complete` time: debug statements, commit format, placeholder data, and orphaned Go files. It was introduced on 2026-03-06 to reduce cross-cutting behavioral weight by 357 lines across 5 skills.

After running for ~1 week across 71 events (27 failures, 44 bypasses), the gate accumulated the highest bypass count of any gate in the system. Retrospective audit confirmed 79% false positive rate with 0 true positives.

## Decision

**Chosen:** Remove the gate entirely

**Rejected alternatives:**
- **Redesign:** The dominant FP pattern (`fmt.Print` in a CLI tool project) is structurally unfixable with regex — would need code context analysis (verbose guards, file purpose)
- **Accept as noise:** Normalizes override behavior (44 bypasses), which undermines the gate system's credibility

## Evidence

- **0 true positives** across 71 events — gate never caught a real bug
- **79% FP rate** (15/19 failures classified as FP in retrospective audit)
- **FP patterns:** fmt.Print as CLI output (755), Python print() in scripts (373), console.error for error handling (145), experiment output files as orphans
- **Prior fixes attempted:** SkipCLIFiles, *_output.go exclusion, baseline-scoped diff — none resolved the fundamental issue
- **Source:** `.kb/investigations/2026-03-11-inv-gate-retrospective-accuracy-audit.md`

## Consequences

- **Positive:** Eliminates highest-bypass gate, stops normalizing overrides, removes `--skip-self-review` ceremony
- **Mitigation:** Manual self-review checklist still in skill phases; `go vet`/`go build` gates (V2, 0% FP) catch real compilation issues
- **Risk:** If agents leave debug statements, no automated catch until code review. Accepted because the gate never caught any in practice.
