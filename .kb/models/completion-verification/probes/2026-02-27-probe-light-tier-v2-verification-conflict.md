# Probe: LIGHT Tier / V2 Verification Level Conflict

**Date:** 2026-02-27
**Status:** Complete
**Model:** Completion Verification Architecture
**Beads:** orch-go-i9qi

## Question

The Completion Verification model states: "well-configured spawns should require zero skip flags." Is this invariant violated by the LIGHT tier + V2 level combination? Specifically: feature-impl defaults to TierLight (no SYNTHESIS.md) but V2 (which includes GateSynthesis). This means every LIGHT feature-impl completion requires `--skip-synthesis`.

## What I Tested

### 1. Verification level determination for feature-impl

Code path in `pkg/spawn/verify_level.go`:
```
DefaultVerifyLevel("feature-impl", "") → V2
DefaultVerifyLevel("feature-impl", "feature") → max(V2, V2) → V2
DefaultVerifyLevel("feature-impl", "bug") → max(V2, V2) → V2
```

### 2. Gates that fire at V2

From `pkg/verify/level.go` GatesForLevel("V2"):
- V0: GatePhaseComplete
- V1: GateSynthesis, GateHandoffContent, GateSkillOutput, GatePhaseGate, GateConstraint, GateDecisionPatchLimit, GateArchitecturalChoices
- V2: GateTestEvidence, GateGitDiff, GateBuild, GateVet, GateAccretion

**GateSynthesis is included at V1+ and therefore fires at V2.**

### 3. Tier determination for feature-impl

From `pkg/spawn/config.go`:
```
SkillTierDefaults["feature-impl"] = TierLight
```

### 4. How modern verification handles LIGHT + V2

In `pkg/verify/check.go:551-576` (verifyCompletionWithLevelAndComments):
```go
if workspacePath != "" && ShouldRunGate(verifyLevel, GateSynthesis) {
    // Checks synthesis — NO tier check here
    if IsKnowledgeProducingSkill(skillName) {
        // auto-skip for knowledge skills
    } else {
        // REQUIRES SYNTHESIS.md — feature-impl hits this path
    }
}
```

**The modern path does NOT check tier. Feature-impl at V2 WILL fail the synthesis gate.**

### 5. Legacy verification path

In `pkg/verify/check.go:653`:
```go
if workspacePath != "" && tier != "light" {
    // Check SYNTHESIS.md — LIGHT tier correctly skipped here
}
```

**The legacy path correctly skips synthesis for LIGHT tier.**

### 6. Partial workaround already exists

Line 388 in check.go:
```go
if !isOrch && ShouldRunGate(verifyLevel, GateArchitecturalChoices) && tier != "light" {
```
The GateArchitecturalChoices gate already has a `tier != "light"` suppression, showing someone already hit this class of problem for a different gate.

### 7. SPAWN_CONTEXT.md tells LIGHT agents to skip SYNTHESIS.md

From `pkg/spawn/context.go:116`:
```
⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
```

So agents are told not to produce it, but V2 verification requires it.

## What I Observed

**The conflict is confirmed.** Two uncoordinated systems make contradictory promises:

| System | Says | About SYNTHESIS.md |
|--------|------|--------------------|
| Spawn tier (LIGHT) | "NOT required" | Agent told to skip |
| Verify level (V2) | GateSynthesis fires | Verification demands it |

The SPAWN_CONTEXT instructs the agent: "SYNTHESIS.md is NOT required." Then `orch complete` runs GateSynthesis because the level is V2. The orchestrator must use `--skip-synthesis` every time.

This directly violates the decision doc's stated goal: "Common case requires zero flags at completion."

## Model Impact

**Confirms invariant violation:** The model states "well-configured spawns should require zero skip flags." For every feature-impl agent (the most common spawn type), the orchestrator must use `--skip-synthesis` — the exact opposite of the design goal.

**Extends model understanding:** The root cause is that V0-V3 was designed to *replace* the tier system (the decision doc says "three implicit systems → one explicit system") but the migration was incomplete. The tier still exists, still controls agent instructions (SPAWN_CONTEXT), and still controls one gate (architectural choices) — but the V0-V3 system ignores it for the synthesis gate.

**Contradict: "Each level is a strict superset"** is technically true for the gate *definitions*, but the *intent* of V2 (for implementation work) conflicts with the *intent* of TierLight (implementation work doesn't need synthesis). Both systems agree on the *work type* but disagree on what's required.
