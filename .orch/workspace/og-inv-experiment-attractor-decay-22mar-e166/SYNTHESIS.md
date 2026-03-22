# SYNTHESIS: Attractor Decay Experiment

## Plain-Language Summary

We tested whether placement attractors (the "put your code after function X" instructions that produce 100% coordination success) break when the codebase changes underneath them. The hypothesis was that renaming the anchor function would cause immediate failure. **The opposite happened: 9/9 trials succeeded with completely stale attractors.** Agents compensate for stale anchors by reading the actual code and adapting — finding renamed functions semantically, or falling back to secondary anchors in the placement instruction. The coordination value of attractors comes from assigning agents to different *regions* of the file, not from the specific function name used as an anchor. This means the sensor requirement (detecting when attractors go stale) is much weaker than expected — you don't need real-time freshness because agents themselves act as adaptive sensors.

## What Was Tested

Three phases of codebase mutation with ORIGINAL (stale) placement prompts, N=3 trials per phase:

1. **Rename**: `FormatDurationShort` → `FormatElapsed` (anchor function doesn't exist under original name)
2. **Reorganize**: `StripANSI` moved to `ansi.go` (anchor function in wrong file)
3. **Alternatives**: New competing functions added (semantic ambiguity)

## Key Findings

| Phase | Success Rate | Agent Adaptation Mechanism |
|-------|-------------|---------------------------|
| baseline | 100% (20/20) | N/A |
| rename | **100% (3/3)** | Semantic adaptation — found `FormatElapsed` at same position |
| reorganize | **100% (3/3)** | Anchor redundancy — used secondary anchor "BEFORE FormatDuration" |
| alternatives | **100% (3/3)** | Literal compliance — attractor still valid, ignored competing functions |

**Two resilience mechanisms discovered:**
1. **Semantic adaptation**: Agents resolve the *intent* of a placement instruction, not the literal name
2. **Anchor redundancy**: Well-designed placement instructions with multiple anchors ("after X, BEFORE Y") survive losing one

**Core insight**: Region separation, not anchor accuracy, is the load-bearing property of attractors.

## Model Impact

- **Extends** CO-02 (structural placement prevents conflicts) with resilience finding
- **Answers** open question: "Can attractor-based coordination degrade?" → Not for incremental changes
- **Weakens** implied sensor requirement — agents themselves compensate for attractor staleness

## Verification Contract

See `VERIFICATION_SPEC.yaml` for detailed verification.

**Key evidence:**
- Results: `experiments/coordination-demo/redesign/results/20260322-154258/`
- Probe: `.kb/models/coordination/probes/2026-03-22-probe-attractor-decay-degradation-curve.md`
- Model updated: `.kb/models/coordination/model.md` (Claim 2, Evidence table, Open Questions, Mechanism Dimension)
