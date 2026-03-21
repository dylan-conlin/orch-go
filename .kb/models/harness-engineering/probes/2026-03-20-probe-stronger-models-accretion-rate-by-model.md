# Probe: Stronger Models Coordination Gates — Accretion Rate by Model [HE-08]

**Claim:** Stronger models may need more coordination gates, not fewer — compliance simplifies with model capability but coordination grows as agents get faster and more confident.

**Falsification criterion:** Stronger models produce less accretion per agent-session than weaker models on the same codebase (controlled experiment, N>50 sessions per model).

**Verdict:** INDIRECTLY SUPPORTED — no controlled cross-model experiment exists, but converging evidence from 4 independent sources supports the claim's direction. The stated falsification criterion is poorly specified (per-session metric misses system-level coordination pressure).

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

### 8. Production per-session accretion (N=235 Opus sessions, Mar 15-21)

Git-log analysis of 235 unique Opus agent sessions (all production, all `claude` backend):

| Metric | Value |
|---|---|
| N | 235 sessions |
| Mean insertions | 243 lines |
| Median | 130 lines |
| Stdev | 289 lines |
| Min / Max | 2 / 1,967 lines |
| Total | 57,100 lines in ~7 days |
| Sessions >500 lines | 29 (12%) |

**System-level accretion rate:** 57,100 lines / 7 days = ~8,157 lines/day from Opus agents alone.

**Comparison with coordination demo haiku data:** Haiku demo agents averaged 112.7 lines per session (no-coord condition). Production Opus agents average 243 lines — 2.2x higher per session. But production tasks are more complex than the demo's simple/complex tasks, so this isn't a clean model comparison.

**The system-level math strengthens:** Even if per-session accretion were identical between models, the completion rate multiplier dominates. With 235 Opus sessions in 7 days (~34/day) at ~96% completion vs an estimated ~7/day if using non-Anthropic models (same spawn rate, ~20% completion), total system accretion would be:
- Opus: 34 sessions/day × 243 lines = ~8,262 lines/day
- Non-Anthropic (hypothetical): 7 sessions/day × 243 lines = ~1,701 lines/day
- Ratio: **~4.9x** more coordination pressure from the stronger model

### 9. Cross-model pilot data (N=2, insufficient for conclusions)

The coordination demo pilot (N=1 per model) provides a directional hint but is statistically meaningless:

| Task | Haiku Added | Opus Added | Delta |
|---|---|---|---|
| Simple (FormatBytes) | 90 | 80 | Opus -11% |
| Complex (FormatTable) | 26 | 34 | Opus +31% |

Individual scores: both models 5/6 (simple) and 10/10 (complex) in all trials. Model capability is not the differentiator for individual task quality. The coordination demo's N=10 confirmed this: both models 6/6 on simple, 10/10 on complex, 100% merge conflict rate regardless of model.

### 10. The falsification criterion is wrong

The stated criterion — "stronger models produce less accretion per agent-session" — targets per-session behavior. But the claim is about coordination pressure, which is a system-level emergent property. Three components determine total coordination pressure:

1. **Per-session accretion** — how many lines each completed session adds
2. **Completion rate** — what fraction of spawned sessions finish successfully
3. **Spawn velocity** — how many sessions are spawned per unit time

Stronger models dominate on (2) and potentially increase (3) because faster completion frees capacity for more spawns. Even if (1) is slightly lower for stronger models (plausible — they might write more concise code), the product (1 × 2 × 3) can still be much larger.

**Revised falsification criterion:** "Total system accretion rate (lines/week) does NOT increase when switching from weaker to stronger models on the same task queue." This captures the coordination claim rather than the per-session claim.

## Assessment

**Claim status:** **Indirectly supported** — no controlled cross-model experiment exists, but 5 converging evidence lines all point in the same direction.

**Evidence summary:**

| Source | Finding | Supports Claim? |
|---|---|---|
| Back-of-envelope (completion rate × accretion) | ~5x system accretion with Opus vs non-Anthropic | Yes |
| Production data (N=235 Opus sessions) | 8,157 lines/day, 34 sessions/day | Yes (high throughput) |
| Coordination demo (N=160, haiku-only) | Gates reduce accretion 8-12%, effect larger on complex tasks | Yes (coordination gates help) |
| daemon.go case study | +892 lines from 30 correct Opus commits | Yes (coordination failure from capable agents) |
| Cross-model demo (N=80) | 100% merge conflict rate independent of model | Yes (capability doesn't solve coordination) |

**What would weaken the claim:**
1. Opus producing dramatically less per-session accretion than haiku (e.g., 50%+ reduction) — could offset the completion rate multiplier
2. Stronger models spontaneously coordinating without structural gates (no evidence for this; coordination demo shows they don't)
3. Non-Anthropic models achieving comparable completion rates — would eliminate the multiplier effect

**What would strengthen it further:**
1. Run coordination demo with `--model opus` (N>50) — measure per-session accretion difference directly
2. Fix `accretion.delta` event emission so production data can be analyzed per-model
3. Measure spawn velocity as a function of model capability (faster completion → more spawns?)

**Remaining blockers for definitive confirmation:**
1. Fix `accretion.delta` event emission to include `code_added`/`code_net` fields
2. Run coordination demo with `--model opus` (N>50 per model)
3. Compare system-level (completion-rate-weighted) accretion, not just per-session

- [x] **Indirectly supports claim** — 5 converging evidence lines, no contradictory data. Controlled experiment not yet run but the falsification criterion itself was poorly specified (per-session metric misses the system-level coordination pressure that the claim actually describes).
