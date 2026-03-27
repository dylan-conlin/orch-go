# Brief: orch-go-cwj26

## Frame

The dashboard said 4% of feature-impl agents were producing SYNTHESIS.md. That sounded terrible — 96% non-compliance on a core completion ritual. The question was whether agents were ignoring the protocol, or whether the protocol was arguing with itself about what to ask for.

## Resolution

It turned out to be three subsystems that evolved different answers to the same question. The spawn tier system says feature-impl is light tier — no synthesis needed. The feature-impl skill's completion section agrees by omission (it lists six steps, none involving synthesis). But the worker-base protocol, which is always loaded as a dependency, includes a full synthesis/brief/verification ritual for full-tier spawns. And the compliance metrics count synthesis rates flat across all tiers, so correct light-tier behavior (no synthesis, by design) shows up as non-compliance.

The surprise was that the diagnostic code — the part closest to completion time — already gets this right. `classifyCompleted()` checks `IsFullTier && !HasSynthesis` before flagging a gap. The metric aggregation upstream just doesn't carry the tier partition. The fix is making all three layers say the same thing: synthesis required when full tier, optional when light tier. The highest-leverage change is adding a tier-conditional block to the feature-impl skill's completion section so it stops contradicting the worker-base protocol for full-tier spawns.

## Tension

Feature-impl's default tier (light) was validated last December: 77% overhead reduction, no quality drop. But the system has no way to know when a specific feature-impl task crosses the line from "code delivery" to "knowledge-producing work." Right now that judgment is the orchestrator's, expressed via `--tier full`. Whether that's the right decision surface — or whether the task description itself should trigger the tier — is a design question that didn't fit in this architect pass.
