# Probe: Verification Levels Design — Unifying Three Implicit Level Systems

**Date:** 2026-02-20
**Status:** Complete
**Beads:** orch-go-1157
**Model:** completion-verification

## Question

The completion-verification model describes gates as "independent and cumulative" — all gates fire, all are reported, individual gates auto-skip based on heuristics. But the codebase actually encodes three separate implicit level systems (spawn tier, checkpoint tier, skill-based auto-skips). Can these be unified into a single "verification level" concept that both human and orchestrator understand?

## What I Tested

1. **Mapped all auto-skip paths in pkg/verify/:** Traced every conditional that determines whether a gate fires:
   - `check.go:532` — synthesis auto-skipped for knowledge-producing skills
   - `test_evidence.go:50-75` — test evidence auto-skipped for investigation/architect/research
   - `check.go:371-381` — visual verification skipped when no web changes
   - `check.go:385-396` — test evidence skipped based on skill type
   - `check.go:401-411` — git diff skipped for orchestrator tier
   - `check.go:416-426` — accretion skipped for orchestrator tier
   - `check.go:431-439` — build runs regardless of tier (only gate that ignores tier)

2. **Mapped checkpoint tier requirements** (`checkpoint.go:155-196`):
   - Tier 1 (feature/bug/decision) → gate1 + gate2
   - Tier 2 (investigation/probe) → gate1 only
   - Tier 3 (task/question) → no checkpoint

3. **Mapped spawn tier defaults** (`spawn/config.go:31-46`):
   - investigation/architect/research/codebase-audit/systematic-debugging → full tier
   - feature-impl/reliability-testing/issue-creation → light tier

4. **Traced the auto-skip decision tree for a typical investigation completion:**
   - Phase Complete: fires (all agents)
   - Synthesis: auto-skipped (IsKnowledgeProducingSkill → true)
   - Skill Output: fires
   - Phase Gates: fires
   - Constraint: fires
   - Decision Patch: fires
   - Test Evidence: auto-skipped (investigation excluded)
   - Git Diff: fires (but usually passes — investigations don't claim file changes)
   - Build: fires only if Go files changed
   - Accretion: fires (but usually passes)
   - Visual: skipped (no web changes)
   - Explain-Back: required (Tier 2 → gate1)
   - Behavioral: not required (Tier 2 → no gate2)

## What I Observed

**Finding 1: The three systems converge on ~4 natural levels.** Every combination of (skill type, issue type, change type) falls into one of four verification intensities. The systems don't conflict — they independently encode the same spectrum.

**Finding 2: The "all gates fire" model claim is wrong.** The model says gates are "independent and cumulative." In practice, 5-8 gates auto-skip for any given completion. The gates are cumulative in code structure but level-selective in practice.

**Finding 3: Auto-skip logic is scattered across 6 different files** with no centralized documentation of which gates fire for which work types. An orchestrator reasoning about "what will orch complete check?" must consult:
- `check.go` (tier-based skips)
- `test_evidence.go` (skill-based skips)
- `escalation.go` (knowledge-producing skill detection)
- `checkpoint/checkpoint.go` (issue-type-based requirements)
- `spawn/config.go` (tier defaults)
- `complete_cmd.go` (SkipConfig and flag processing)

**Finding 4: The build gate is the only gate that ignores all level signals.** It fires for any completion that changed Go files, regardless of tier, skill, or issue type. This is correct behavior (a broken build should always be caught) but inconsistent with the "level determines gates" design unless we make build an unconditional baseline.

## Model Impact

**Contradicts:** Model claim that gates are "independent and cumulative." They are structurally independent but functionally level-selective — the auto-skip logic creates implicit levels.

**Extends:** The model should document the four verification levels (V0-V3) as the primary concept, with individual gates as implementation details of each level.

**Confirms:** Model claim that "targeted bypass system shows mature engineering." The SkipConfig system works correctly as an escape hatch. The design recommendation preserves it for edge cases.

**Recommends model update:**
1. Replace "3 gates" / "14 gates" narrative with "4 verification levels, each checking a subset of 14 gates"
2. Document which level is default for each (skill, issue type) combination
3. Move gate inventory to a reference section; lead with levels as the primary concept
4. Note: build gate is unconditional (fires regardless of level)

## Investigation Reference

Full design: `.kb/investigations/2026-02-20-inv-architect-verification-levels.md`
