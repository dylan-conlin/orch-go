# Probe: Architect-Gate Enforcement for Hotspot Areas

**Status:** Complete
**Date:** 2026-02-24
**Triggered by:** orch-go-1187 (architect: design enforcement for investigation → architect → implementation)
**Model:** spawn-architecture

## Question

Can the existing spawn gate infrastructure enforce the investigation → architect → implementation sequence for hotspot areas, and what are the enforcement gaps that allowed orch-go-1182/1183 to bypass architectural review?

## What I Tested

### Test 1: Current --force-hotspot bypass behavior

**Code reviewed:** `pkg/spawn/gates/hotspot.go:59-63`

```go
if forceHotspot {
    fmt.Fprintln(os.Stderr, "⚠️  --force-hotspot: Bypassing CRITICAL hotspot block")
    fmt.Fprintln(os.Stderr, "")
    return result, nil
}
```

**Observation:** When `--force-hotspot` is passed, the gate prints a warning and returns `nil` error — no verification that an architect has reviewed the area. The bypass is unconditional. No logging, no issue reference, no accountability trail.

### Test 2: Daemon-driven spawns skip hotspot check entirely

**Code reviewed:** `pkg/spawn/gates/hotspot.go:50-53`

```go
if daemonDriven {
    return result, nil
}
```

**Observation:** Daemon-driven spawns return the result but never check `HasCriticalHotspot`. The result is returned for telemetry but the gate never blocks. If daemon infers `feature-impl` for an issue targeting a CRITICAL hotspot file, the spawn proceeds without any warning or block.

### Test 3: Daemon skill inference for features/tasks

**Code reviewed:** `pkg/daemon/skill_inference.go:29-43`

```go
case "feature":
    return "feature-impl", nil
case "task":
    return "feature-impl", nil
```

**Observation:** Feature and task issues always infer to `feature-impl`. The daemon has no hotspot awareness — it doesn't check if the issue targets hotspot files before choosing skill. Bugs correctly default to `architect`, but features/tasks go straight to implementation.

### Test 4: Pre-flight check chain

**Code reviewed:** `pkg/orch/extraction.go:375-379`

```go
if hotspotCheckFunc != nil {
    if _, err := gates.CheckHotspot(preCheckDir, input.Task, input.SkillName, input.DaemonDriven, forceHotspot, hotspotCheckFunc); err != nil {
        return nil, err
    }
}
```

**Observation:** The hotspot gate is the LAST check in the pre-flight chain. It runs after triage, verification, concurrency, and rate limit checks. This is correct — no point running expensive hotspot analysis if earlier checks fail.

## What I Observed

### Three enforcement gaps that enabled the orch-go-1182/1183 failure:

**Gap 1: --force-hotspot has no accountability requirement**
The flag is a simple boolean bypass. No architect issue reference, no logged reason, no verification. The orch-go-1182 case: hotspot gate fired → orchestrator passed --force-hotspot → worker implemented a solution that violated the two-lane decision. The bypass required zero evidence that an architect had reviewed the area.

**Gap 2: Daemon-driven spawns skip the hotspot gate entirely**
If the daemon creates a follow-up feature-impl from an investigation that discovered issues in a hotspot area, the spawn proceeds with no hotspot check. The daemon's comment "triage already happened" doesn't hold — triage checks issue validity, not whether the target area needs architectural review.

**Gap 3: Investigation skill has no hotspot-aware routing**
The investigation skill's D.E.K.N. summary includes a "Next" field for follow-up recommendations. But there's no guidance for investigation agents to check whether their findings affect hotspot files, nor to recommend architect (rather than feature-impl) when they do.

### What worked correctly:
- Hotspot detection infrastructure is solid (4 signals: fix density, investigation clustering, bloat, coupling)
- Skill exemptions are correct (architect/investigation/capture-knowledge/codebase-audit exempt)
- Completion accretion gate functions properly (800 warning, 1500 error, net-negative bypass)
- Error message on block correctly says "Spawn architect to design extraction first"

## Model Impact

**Extends** the spawn-architecture model with a new invariant:

**Proposed Invariant: Architect-Gated Hotspot Override**
> When implementation skills (feature-impl, systematic-debugging) target CRITICAL hotspot files, `--force-hotspot` requires `--architect-ref <issue-id>` where the referenced architect issue is type=architect and status=closed. This converts the escape hatch from "bypass with flag" to "bypass with proof of architectural review."

**Relates to:**
- Completion verification model: Accretion enforcement gap analysis (2026-02-19) identified Layer 1 spawn gate gaps — this probe extends that finding with the architect-reference requirement
- Agent lifecycle state model: The orch-go-1182/1183 violation was caught by the tmux-liveness two-lane probe — this probe addresses the root cause (missing enforcement of architect-first sequence)

**Confirms:**
- Spawn-architecture Invariant: "Skills own domain behavior, spawn owns orchestration infrastructure" — enforcement belongs in spawn gates, not skill templates
- Principle: "Gate Over Remind" — advisory routing in orchestrator skill is insufficient
- Principle: "Infrastructure Over Instruction" — the orchestrator skill already says "use architect first for hotspots" but instructions didn't prevent the violation
