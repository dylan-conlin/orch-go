# Decision: Verification Levels (V0-V3)

**Date:** 2026-02-20
**Status:** Accepted
**Deciders:** Dylan
**Blocks:** verification level, completion gates, verify, orch complete, skip flags, force, V0, V1, V2, V3

## Context

The orchestrator and Dylan couldn't communicate about verification. "Verify this works" meant 5 different things. The system had 14 completion gates that all fired by default, requiring `--skip-*` flag combinations or `--force` to bypass — leading to gate proliferation where `--force` became the happy path.

An audit (orch-go-1153) found the 14 gates were real (not theater), but three implicit level systems (spawn tier, checkpoint tier, skill-based auto-skips) encoded the same concept in uncoordinated vocabularies.

A companion investigation (orch-go-1158) on tradeoff visibility concluded that architectural tradeoff surfacing belongs upstream (at spawn/triage time via model Pressure Points), not as additional completion gates.

## Decision

### Four Verification Levels

One concept — verification level — declared at spawn time, determines everything at completion time.

| Level | Name | Gates That Fire | Typical Work |
|-------|------|-----------------|--------------|
| **V0** | Acknowledge | Phase Complete | Config, README, issue creation |
| **V1** | Artifacts | V0 + Synthesis, Handoff Content, Skill Output, Phase Gates, Constraint, Decision Patch Limit | Investigations, architect, research |
| **V2** | Evidence | V1 + Test Evidence, Git Diff, Build, Accretion | Features, bug fixes, debugging |
| **V3** | Behavioral | V2 + Visual Verification, Explain-Back, Behavioral gate | UI features, critical behavioral changes |

Each level is a strict superset of the one below. The 14 existing gates are reorganized into levels, not replaced.

### Declaration: Infer + Override

Default level is deterministic from (skill, issue type):

```
V_default = max(skill_level, issue_type_level)
```

Skill defaults:
- issue-creation → V0
- investigation/architect/research/codebase-audit → V1
- feature-impl/systematic-debugging/reliability-testing → V2

Issue type minimums:
- feature/bug/decision → min V2
- investigation/probe → min V1
- task/question → no minimum

Orchestrator can override at spawn time: `orch spawn feature-impl "update README" --verify-level V0`

### Three Blocking Question Resolutions

**Storage:** AGENT_MANIFEST.json. The level is spawn metadata — one place for spawn decisions.

**Override timing:** Spawn-only. The level is immutable after spawn. Skip flags handle edge cases at completion. Completion-time override would become the new `--force` with a friendlier name.

**Web file auto-elevation:** Warn, don't elevate. If a V2 agent touches web files, completion outputs a warning but doesn't block. Preserves orchestrator judgment and the spawn-only immutability invariant. The orchestrator learns to declare V3 upfront for UI work.

### Tradeoff Visibility Integration

Tradeoff visibility (from orch-go-1158) is embedded within levels, not added as separate gates:

- **Model Pressure Points** → upstream of levels (injected in SPAWN_CONTEXT via kb context). Not a gate — orchestrator context for triage judgment.
- **Architectural Choices in SYNTHESIS.md** → content the existing synthesis gate surfaces at V1+. Not a new gate.
- **Tradeoff comprehension** → naturally part of V3 explain-back.

### Feature Requests Flow Through Models (Q1161 Resolution)

The tradeoff visibility gap is upstream: feature requests should be checked against model Pressure Points before becoming tasks. This is orchestrator judgment at triage time, not automation. The orchestrator reads pressure points (surfaced by kb context) and considers whether the task conflicts with known architectural fragility before spawning.

This changes the orchestrator skill's pre-spawn checks to include: "does this task conflict with architectural pressure in the target domain?"

## Rationale

### Why levels over the current gate system?

The current system requires the orchestrator to reason about which of 14 gates will fire and which skip flags to use. Levels replace that with a single concept: "this is V2 work." The gates still exist but the level pre-selects the relevant subset.

### Why not add more gates?

Historical failure mode: gates proliferate → flag combos needed → `--force` becomes default → theater. We've lived through this. The design reorganizes existing gates rather than adding new ones.

### Why spawn-only immutability?

The whole point is upfront judgment. Changing the level at completion is retroactive justification, not reasoning. Skip flags already handle "something unexpected happened."

### Why warn-not-elevate for web changes?

Auto-elevation after spawn contradicts spawn-only immutability. Warning preserves orchestrator judgment while surfacing the gap. The orchestrator learns rather than being bypassed.

## Consequences

**Positive:**
- Shared vocabulary: "V2" means the same thing to orchestrator and human
- Common case requires zero flags at completion
- `--force` usage should drop to near-zero
- Tradeoff visibility integrated without gate proliferation
- Orchestrator triage becomes more substantive (checking pressure points)

**Negative:**
- Migration effort: three implicit systems → one explicit system
- One new flag at spawn time (`--verify-level`)
- Model Pressure Points sections need to be authored

**Risks:**
- Orchestrator always using V0 override to avoid friction (monitor usage)
- Skip flags at completion becoming routine (same pattern as `--force`)

## Implementation Phasing

1. Add `VerifyLevel` to spawn config and AGENT_MANIFEST.json. Infer from skill+issue type.
2. Add `GatesForLevel()` to `pkg/verify/check.go`. Completion selects gates based on level.
3. Add `--verify-level` flag to `orch spawn`. Surface level in `orch status`.
4. Remove redundant auto-skip logic. Monitor skip flag usage.

## References

- Investigation: `.kb/investigations/2026-02-20-inv-architect-verification-levels.md`
- Audit: `.kb/investigations/2026-02-20-audit-verification-infrastructure-end-to-end.md`
- Tradeoff visibility: `.kb/investigations/2026-02-20-design-tradeoff-visibility-for-non-code-reading-orchestrator.md`
- Models: `verifiability-first-development.md`, `control-plane-bootstrap.md`
