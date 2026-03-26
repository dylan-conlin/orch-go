## Summary (D.E.K.N.)

**Delta:** The fallback cascade should preserve Anthropic first (alternate Opus account, then Sonnet) and only cross to GPT-5.4 for `feature-impl` overflow, not for reasoning-heavy skills.

**Evidence:** `pkg/daemon/skill_inference.go` still hardcodes `opus` for architect/investigation/debugging, `pkg/model/model.go` makes Sonnet the general default, and the Mar 26 routing thread only validates GPT-5.4 as `feature-impl` overflow while warning that reasoning-heavy routing remains under-tested.

**Knowledge:** orch-go already has the ingredients for a safe cascade, but it does not yet track Opus-specific weekly exhaustion or codify when a cross-provider downgrade is allowed.

**Next:** Implement the fallback policy as routing logic plus capacity/telemetry plumbing so the system can distinguish "Opus exhausted, Sonnet still viable" from "Anthropic path unhealthy, use OpenAI overflow or stop."

**Authority:** architectural - The decision crosses account routing, model routing, daemon defaults, and observability, so it needs one coherent policy before implementation.

---

# Investigation: Design Model Fallback Cascade Opus

**Question:** When a spawn would normally use Opus but Anthropic capacity or rate limits block that path, what should the routing priority order be and which fallbacks are safe to automate?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode architect worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/model-access-spawn-paths/model.md` | extends | yes | none |
| `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md` | extends | yes | supersedes older Gemini-secondary guidance |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Current routing already prefers Opus only for reasoning-heavy skills

**Evidence:** The daemon maps `systematic-debugging`, `investigation`, `architect`, `codebase-audit`, and `research` to the `opus` alias, but `feature-impl` is intentionally left unmapped so the resolve pipeline uses the global default model instead. The global default model is Sonnet, not Opus.

**Source:** `pkg/daemon/skill_inference.go:271`, `pkg/daemon/skill_inference.go:279`, `pkg/model/model.go:91`

**Significance:** The fallback problem is not "all spawns lose Opus"; it is mainly about the reasoning-heavy lane. Implementation work should not introduce a global downgrade mechanism that perturbs `feature-impl`, because that lane already defaults to Sonnet.

---

### Finding 2: Anthropic-first fallback is supported by current architecture, but Opus-specific headroom is not modeled

**Evidence:** The current resolver already picks accounts by tier-weighted effective headroom using 5-hour and 7-day capacity, which is the right first move for preserving Opus quality on another Anthropic account. But the raw Anthropic usage response includes `seven_day_opus` and orch-go discards it: `usageAPIResponse` defines the field, `CapacityInfo` does not store it, the parser ignores it, and the `/api/usage` handler exposes `weekly_opus_percent` fields without populating them.

**Source:** `pkg/spawn/resolve.go:537`, `pkg/account/capacity.go:24`, `pkg/account/capacity.go:74`, `pkg/account/capacity.go:225`, `cmd/orch/serve_system.go:29`

**Significance:** orch-go can already ask "which Anthropic account is healthiest?" but not "is Opus itself exhausted while Sonnet remains viable?" That missing distinction is the key design gap for a correct fallback cascade.

---

### Finding 3: Cross-provider fallback is only validated for implementation overflow, not for reasoning-heavy work

**Evidence:** The model-selection guide still documents an older rate-limit escalation of "switch account, then Gemini," but the Mar 26 routing thread updates the empirical picture: GPT-5.4 via ChatGPT Pro / Codex OAuth hit 80% first-attempt and 100% with retry on five real tasks, which is enough to promote it to `feature-impl` overflow only. The same thread explicitly keeps Opus as the default for architect/investigation/debugging because reasoning-heavy behavior remains under-tested, with one transient silent death and weaker scope control than Opus.

**Source:** `.kb/guides/model-selection.md:162`, `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md:76`, `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md:78`, `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md:80`

**Significance:** A safe automatic cascade must stop before silently routing architect/investigation/debugging work onto GPT-5.4. That is a policy boundary, not just a technical limitation.

---

## Synthesis

**Key Insights:**

1. **Preserve capability before changing provider** - Because only reasoning-heavy skills are Opus-pinned today, the first fallback should try to preserve both provider and model family semantics: another Anthropic account first, then Sonnet on the same backend, before considering OpenAI.

2. **The missing signal is Opus-specific exhaustion** - The raw API already exposes `seven_day_opus`, but orch-go collapses everything into generic Claude headroom. Without that signal, the system cannot distinguish "fallback to Sonnet" from "switch provider or stop."

3. **Cross-provider fallback should be skill-scoped, not universal** - GPT-5.4 is now real enough to carry `feature-impl` overflow, but the evidence base does not yet support automatic fallback for architect, investigation, or debugging.

**Answer to Investigation Question:**

The routing priority order should be:

1. **Stay on Opus by switching to the healthiest Anthropic account** when another account has usable Opus headroom.
2. **Downgrade to Sonnet on the Claude backend** when the Anthropic path is still healthy enough for Claude work but Opus-specific capacity is exhausted or degraded.
3. **Use GPT-5.4 only as `feature-impl` overflow** when the Anthropic path itself is unavailable and the task is in the already-validated implementation lane.
4. **Stop and surface the constraint** for architect, investigation, systematic-debugging, research, and codebase-audit instead of silently routing them to GPT-5.4 or older Gemini guidance.

This ordering follows the principles of premise-before-solution and evidence hierarchy: preserve the validated path first, degrade within the same provider second, and only cross providers where empirical evidence exists. The main limitation is that orch-go does not currently ingest `seven_day_opus`, so implementing step 2 requires new capacity plumbing rather than just a resolver tweak.

---

## Structured Uncertainty

**What's tested:**

- ✅ Reasoning-heavy skills currently force `opus`, while `feature-impl` falls through to the default model (verified by reading `pkg/daemon/skill_inference.go` and `pkg/model/model.go`)
- ✅ Anthropic account routing already uses a same-provider capacity heuristic based on 5-hour and 7-day headroom (verified by reading `pkg/spawn/resolve.go`)
- ✅ The raw capacity API exposes `seven_day_opus`, but orch-go does not propagate it into routing or dashboard output (verified by reading `pkg/account/capacity.go` and `cmd/orch/serve_system.go`)
- ✅ GPT-5.4 overflow validation is explicitly limited to `feature-impl`, while reasoning-heavy fallback remains under-tested (verified by reading the Mar 26 routing thread)

**What's untested:**

- ⚠️ Whether Sonnet should be used automatically for every Opus failure mode, or only when `seven_day_opus` is exhausted but generic Claude headroom remains
- ⚠️ Whether a second Anthropic account is usually available in the current single-Max-subscription setup, or whether step 1 will often be skipped in practice
- ⚠️ Whether GPT-5.4 can be promoted beyond `feature-impl` after a focused benchmark on architect/investigation/debugging work

**What would change this:**

- If a benchmark shows GPT-5.4 matches Opus reliability and scope control on reasoning-heavy tasks, step 4 could become a broader OpenAI fallback instead of a hard stop
- If Anthropic capacity data shows `seven_day_opus` is redundant with generic weekly limits, the Sonnet-specific branch may not need special telemetry
- If operational policy eliminates multi-account Anthropic routing entirely, the cascade should start at Sonnet instead of account switching

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Adopt an Anthropic-first, skill-scoped fallback cascade (`Opus alternate account` -> `Sonnet` -> `GPT-5.4 feature-impl only` -> `stop`) | architectural | It spans account routing, model fallback rules, skill policy, and operator-visible telemetry |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside -> implementation
- Reaches to other components/agents -> architectural
- Reaches to values/direction/irreversibility -> strategic

### Recommended Approach ⭐

**Anthropic-first staged degradation** - Codify a four-step cascade that preserves Opus first, preserves Claude second, and only crosses to GPT-5.4 where the evidence base already supports it.

**Why this approach:**
- It matches the current architecture instead of fighting it: reasoning-heavy skills are already a special lane, and `feature-impl` is already separate.
- It respects the evidence boundary from the Mar 26 benchmark instead of pretending all models are interchangeable.
- It turns the existing but discarded `seven_day_opus` signal into a concrete routing distinction between "use Sonnet" and "leave Anthropic."

**Trade-offs accepted:**
- We do not get a single universal auto-fallback path for every skill.
- We defer automatic GPT-5.4 fallback for architect/investigation/debugging until a dedicated benchmark closes the reliability gap.

**Implementation sequence:**
1. Extend capacity plumbing to carry `seven_day_opus` through `pkg/account`, cache, and dashboard-facing structs, because the cascade needs that signal to know when Sonnet is the right downgrade.
2. Add a routing policy layer that classifies the failure mode (`alternate Opus available`, `Opus exhausted but Claude healthy`, `Anthropic path unhealthy`) and picks the next model accordingly.
3. Gate GPT-5.4 fallback by skill so only `feature-impl` can cross providers automatically, while reasoning-heavy skills emit a surfaced constraint or explicit manual escalation.
4. Update docs and spawn/status surfaces so the chosen fallback reason is visible to operators and future agents.

### Alternative Approaches Considered

**Option B: Universal Sonnet fallback for all Opus failures**
- **Pros:** Simple to explain, keeps everything on Claude backend, lower implementation cost
- **Cons:** Conflates Opus-specific exhaustion with full Anthropic unavailability and loses the chance to preserve Opus on another account; also hides whether the Anthropic path is actually unhealthy
- **When to use instead:** If product policy decides multi-account Anthropic routing is gone and `seven_day_opus` proves unnecessary

**Option C: Universal GPT-5.4 fallback for any Opus miss**
- **Pros:** Single cross-provider overflow path, aligns with the new ChatGPT Pro subscription, avoids Anthropic bottlenecks entirely
- **Cons:** Violates the current evidence boundary because GPT-5.4 is only validated for `feature-impl`; would auto-route architect/investigation/debugging onto a weaker and under-tested path
- **When to use instead:** After a benchmark shows GPT-5.4 is reliable enough on reasoning-heavy worker sessions

**Rationale for recommendation:** Option A is the only design that preserves current quality guarantees while still acknowledging the new OpenAI overflow path. Option B is too coarse, and Option C outruns the evidence.

---

### Implementation Details

**What to implement first:**
- Add Opus-specific capacity fields to `CapacityInfo`, cache serialization, and any UI/API structs that surface usage
- Create a single fallback decision function so policy does not fragment across daemon inference, manual spawn paths, and UI guidance
- Update model-selection guidance to replace the stale Gemini-secondary advice with the new Anthropic-first / GPT-5.4-overflow policy

**Things to watch out for:**
- ⚠️ Do not auto-trigger Docker or other infrastructure workarounds; the worker protocol requires surfaced constraints before circumvention
- ⚠️ Avoid defect class 5 (contradictory authority signals) by keeping the cascade in one canonical resolver instead of sprinkling fallback logic across CLI, daemon, and docs
- ⚠️ Avoid defect class 2 (multi-backend blindness) by making the fallback reason explicit when the route changes from Claude backend to OpenAI/OpenCode

**Areas needing further investigation:**
- Measure how often `seven_day_opus` diverges from generic weekly Claude capacity in real usage
- Benchmark GPT-5.4 on architect/investigation/systematic-debugging before expanding automatic fallback scope
- Decide whether single-Max provisioning should remove the alternate-account branch from the default cascade

**Success criteria:**
- ✅ When Opus-specific capacity is exhausted but generic Claude capacity remains, the system chooses Sonnet and reports why
- ✅ When Anthropic capacity is unavailable and the skill is `feature-impl`, the system can route to GPT-5.4 with an operator-visible reason
- ✅ When Anthropic capacity is unavailable and the skill is reasoning-heavy, the system stops or escalates instead of silently downgrading
- ✅ Status/usage surfaces show enough capacity data to explain which fallback branch fired

---

## References

**Files Examined:**
- `pkg/daemon/skill_inference.go` - Checked which skills currently require Opus
- `pkg/model/model.go` - Verified the global default model and alias definitions
- `pkg/spawn/resolve.go` - Verified account and backend routing heuristics
- `pkg/account/capacity.go` - Verified which Anthropic capacity signals are parsed and which are dropped
- `cmd/orch/serve_system.go` - Checked how usage/capacity is exposed to operators
- `.kb/models/model-access-spawn-paths/model.md` - Verified current backend/model architecture and fallback constraints
- `.kb/guides/model-selection.md` - Checked existing rate-limit escalation guidance
- `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md` - Verified the newest GPT-5.4 routing evidence

**Commands Run:**
```bash
# Verify project location
pwd

# Create investigation artifact
kb create investigation design-model-fallback-cascade-opus --orphan

# Inspect issue context
bd show orch-go-2mlhl
```

**External Documentation:**
- None

**Related Artifacts:**
- **Model:** `.kb/models/model-access-spawn-paths/model.md` - Current backend/model routing substrate
- **Guide:** `.kb/guides/model-selection.md` - Existing operator-facing fallback guidance that now needs updating
- **Thread:** `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md` - Latest evidence on GPT-5.4 routing scope

---

## Investigation History

**2026-03-26 09:56:** Investigation started
- Initial question: What should the fallback priority order be when Opus is rate-limited?
- Context: Need a routing policy that reflects current Anthropic/OpenAI evidence instead of older Gemini-era assumptions

**2026-03-26 10:05:** Routing split clarified
- Verified that only reasoning-heavy skills are forced to Opus while `feature-impl` already defaults to Sonnet

**2026-03-26 10:12:** Primary design gap identified
- Verified that raw capacity data exposes `seven_day_opus` but current structs and handlers drop it, which blocks a correct Sonnet-vs-provider-fallback decision

**2026-03-26 10:20:** Investigation completed
- Status: Complete
- Key outcome: Recommended an Anthropic-first fallback cascade with GPT-5.4 restricted to `feature-impl` overflow until reasoning-heavy benchmarks exist
