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
1. **~~Populate model field~~ DONE** in `session.spawned` and `accretion.delta` events — model field added to `AccretionDeltaData` struct and wired from agent manifest at completion time (orch-go-2jtbe, 2026-03-20)
2. **Run comparable tasks on different models** — same skill, same codebase, different model strengths. The coordination demo (`experiments/coordination-demo/`) now measures accretion per trial via `measure-accretion.sh`.
3. **Measure both completion rate AND accretion** — the claim is about system-level coordination, not per-session behavior

The falsification criterion as stated ("less accretion per agent-session") may be the wrong metric. If stronger models produce slightly more accretion per session but 25x more sessions, the per-session metric would miss the coordination problem entirely.

### 6. Coordination demo accretion data (N=160, haiku-only)

Initial accretion measurement from redesign experiment (all haiku, 4 conditions x 2 tasks x 10 trials x 2 agents):

| Condition | N | Avg Added | Stdev | Median | Avg Duration | Lines/min |
|---|---|---|---|---|---|---|
| no-coord | 40 | 112.7 | 52.5 | 86 | 75s | 107.6 |
| placement | 40 | 99.0 | 41.0 | 84 | 70s | 99.9 |
| context-share | 40 | 100.5 | 42.8 | 86 | 68s | 98.0 |
| messaging | 40 | 104.1 | 46.7 | 87 | 87s | 82.0 |

**Effect sizes vs no-coord baseline:**

| Condition | Reduction | Cohen's d | Significance |
|---|---|---|---|
| placement | -12.2% | 0.291 | Small effect |
| context-share | -10.8% | 0.254 | Small effect |
| messaging | -7.7% | 0.174 | Negligible |

**Task complexity interaction:** Complex tasks show larger coordination benefit (-14.2% for placement) than simple tasks (-8.3%). This is directionally consistent with the claim — coordination gates matter more as tasks grow in scope.

**Per-trial total accretion (both agents combined):** no-coord mean=225.4 (stdev=73.5), placement mean=197.9 (stdev=61.8). Coordination reduces variance as well as mean — suggesting more predictable outcomes.

**Messaging overhead:** Messaging condition adds coordination overhead (+16% duration vs no-coord) while producing the smallest accretion reduction. The coordination cost exceeds the benefit at this scale.

**Limitation:** All haiku. The cross-model comparison (haiku vs opus) has not been run. Haiku is a weaker model — if opus produces MORE accretion per agent with the same coordination conditions, it would support the claim. If less, it would contradict.

### 7. Instrumentation gap partially remains

The probe initially stated the instrumentation gap was CLOSED. Verification shows it's only partially closed:
- `session.spawned` events: model field populated correctly (confirmed: recent events show `"model":"anthropic/claude-opus-4-5-20251101"`)
- `accretion.delta` events: model field exists in schema but **line-count fields (`code_added`, `code_net`) are not populated** in new events. Only 3 of 425 accretion.delta events have model data, and none contain accretion measurements. The wiring connects model identity to completion events but doesn't connect accretion measurements to model identity.

## Assessment

**Claim status:** Remains **unconfirmed** — instrumentation partially closed, controlled experiment not yet run.

**Falsification criterion assessment:** The stated criterion ("less accretion per agent-session") may be poorly specified. The claim is about coordination pressure at the system level, not per-session accretion. A better falsification would be: "Total system accretion rate (lines/week) does NOT increase when switching from weaker to stronger models, holding task volume constant."

**Indirect evidence direction:** All available evidence is directionally consistent with the claim:
1. Back-of-envelope: ~5x system accretion with stronger models (completion rate multiplier)
2. Coordination demo: coordination gates reduce accretion 8-12% even within a single model tier
3. daemon.go case study: +892 lines from 30 individually-correct Opus commits (the only model in use)
4. Task complexity interaction: coordination benefits increase with task complexity, suggesting they'd increase further with model capability

**What would falsify:** Opus producing LESS per-session accretion than haiku on identical tasks in the coordination demo. This is plausible (stronger models might write more concise code) but would only falsify the per-session metric — the system-level claim (completion rate × per-session accretion) could still hold.

**Remaining blockers:**
1. Fix `accretion.delta` event emission to include `code_added`/`code_net` fields
2. Run coordination demo with `--model opus` (N>50 per model)
3. Compare both per-session AND system-level (completion-rate-weighted) accretion

- [x] **Neither confirms nor contradicts** — instrumentation partially closed, experiment infrastructure ready but controlled comparison not yet run. Indirect evidence is directionally consistent with the claim.
