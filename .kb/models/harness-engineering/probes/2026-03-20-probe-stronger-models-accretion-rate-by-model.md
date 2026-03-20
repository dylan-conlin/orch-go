# Probe: Stronger Models Coordination Gates — Accretion Rate by Model [HE-08]

**Claim:** Stronger models may need more coordination gates, not fewer — compliance simplifies with model capability but coordination grows as agents get faster and more confident.

**Falsification criterion:** Stronger models produce less accretion per agent-session than weaker models on the same codebase (controlled experiment, N>50 sessions per model).

**Verdict:** UNTESTABLE with current data. Instrumentation gap prevents falsification.

## Findings

### 1. Model field never populated in events

All 784 `session.spawned` events have `model: null`. The `accretion.delta` schema doesn't include a model field at all. There is no way to join accretion data to model identity.

**Root cause:** `pkg/events/logger.go` has a `data.Model` field in the event struct, but spawn pipeline never populates it. The `--model` flag resolves via `pkg/model.Resolve()` but the resolved model string isn't passed through to the event emission path.

### 2. No model diversity in population

All 784 spawns use `spawn_mode: "claude"` (Anthropic backend). Zero non-Anthropic spawns exist. Even if model field were populated, there's no weaker-model comparison group.

This is structural — Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026), making Claude CLI the only path for Anthropic models, and non-Anthropic models have 67-87% stall rates making them impractical for production work.

### 3. Accretion data exists but is model-blind (N=363)

363 `accretion.delta` events are available, broken down by skill:

| Skill | N | Avg Code Net | Avg Code Added | Risk Sessions |
|---|---|---|---|---|
| feature-impl | 149 | 870 | 1,259 | 72 (48%) |
| systematic-debugging | 54 | 375 | 598 | 28 (52%) |
| investigation | 52 | 530 | 760 | 14 (27%) |
| architect | 7 | 2,278 | 2,420 | 5 (71%) |

Feature-impl agents produce the most accretion. Architect sessions produce the highest per-session code additions but have very small N.

### 4. Indirect evidence supports the claim's direction

**daemon.go case study:** +892 lines from 30 individually-correct commits — the canonical coordination failure. All commits were by Opus-class agents (the only agents in the system). Weaker agents (non-Anthropic) would have produced fewer commits total due to stall rates, meaning less accretion pressure.

**Compliance vs coordination divergence:** Opus stall rate ~4% vs non-Anthropic 67-87%. Stronger models clearly solve compliance (following instructions, completing tasks). But completion rate × accretion-per-session = total system accretion. A model that completes 25x more sessions while producing similar per-session code additions would produce 25x more coordination pressure.

**Back-of-envelope:** If Opus completes ~96% of sessions and non-Anthropic completes ~20%, and per-session code additions are similar (~870 lines for feature-impl), then per-100-spawns: Opus adds ~83,520 lines, non-Anthropic adds ~17,400 lines. The coordination burden scales with total successfully-added code, not per-session behavior.

### 5. What would make this testable

To run the controlled experiment (N>50 per model):
1. **Populate model field** in `session.spawned` and `accretion.delta` events — requires wiring `spawnModel` through to event emission in spawn pipeline
2. **Run comparable tasks on different models** — same skill, same codebase, different model strengths
3. **Measure both completion rate AND accretion** — the claim is about system-level coordination, not per-session behavior

The falsification criterion as stated ("less accretion per agent-session") may be the wrong metric. If stronger models produce slightly more accretion per session but 25x more sessions, the per-session metric would miss the coordination problem entirely.

## Assessment

**Claim status:** Remains **unconfirmed** — indirect evidence is consistent with the claim but no controlled experiment exists.

**Falsification criterion assessment:** The stated criterion ("less accretion per agent-session") may be poorly specified. The claim is about coordination pressure at the system level, not per-session accretion. A better falsification would be: "Total system accretion rate (lines/week) does NOT increase when switching from weaker to stronger models, holding task volume constant."

**Instrumentation gap identified:** Model identity is not tracked in events. This is a prerequisite for any model-comparative analysis.

- [x] **Neither confirms nor contradicts** — data insufficient to test. Identifies instrumentation gap (model field unpopulated) as prerequisite for future testing.
