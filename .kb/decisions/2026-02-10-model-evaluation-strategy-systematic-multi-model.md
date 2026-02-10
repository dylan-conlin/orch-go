## Summary (D.E.K.N.)

**Delta:** Adopt systematic model evaluation: external monitoring (Marginlab alerts) + orch-specific eval harness (reference tasks across models). Run on new releases, suspected degradation, and quarterly.

**Evidence:** Opus 4.5 degraded noticeably before 4.6 (user-observed + Reddit reports). 6-model logout-fix benchmark (Jan 28) showed Codex and DeepSeek beat Opus/Sonnet/GPT/Gemini — results were non-obvious and only discovered through controlled comparison. Marginlab tracks daily SWE-bench for both Claude and Codex. Current evaluation is ad-hoc and reactive.

**Knowledge:** Model reliability is unpredictable — performance degrades silently, new releases change the landscape overnight (Opus 4.6 + GPT-5.3-Codex both dropped Feb 5). Generic benchmarks (SWE-bench) don't capture orch-specific behavior (completion protocol, spawn context adherence, Claude-dialect compatibility). The system should be designed for model portability, not single-model dependency.

**Next:** Create beads issue for `orch eval` capability. Subscribe to Marginlab alerts. Curate 3-5 reference tasks spanning skill types. Reduce Claude-dialect coupling in spawn templates over time.

---

# Decision: Model Evaluation Strategy — Systematic Multi-Model Assessment

**Date:** 2026-02-10
**Status:** Accepted

**Related-To:**
- `.kb/benchmarks/2026-01-28-logout-fix-6-model-comparison.md` — Best existing benchmark (6 models, controlled)
- `.kb/models/multi-model-evaluation-feb2026.md` — Codex production evaluation
- `.kb/investigations/archived/2026-01-17-inv-build-model-advisor-tool-live.md` — OpenRouter API for live model data
- `.kb/investigations/archived/2026-01-27-research-kimi-k2-visual-agentic-model.md` — Example of model assessment
- `.kb/decisions/2026-02-10-subscription-strategy-dual-vendor-resilience.md` — Subscription strategy this enables
- https://marginlab.ai/trackers/claude-code-historical-performance/ — External Claude tracking
- https://marginlab.ai/trackers/codex-historical-performance/ — External Codex tracking

---

## Context

**Background:** The orchestration system currently uses GPT-5.3-Codex as default spawn model and Claude Opus 4.6 for orchestration. Model selection has evolved through experience rather than systematic evaluation. The Jan 28 logout-fix benchmark was the most rigorous comparison to date — but it was ad-hoc.

**What triggered this decision:** Dylan experienced significant Opus 4.5 degradation before 4.6 released, driving interest in GPT as alternative. Reddit confirmed many Claude users were transitioning. This revealed model reliability as a serious vendor risk requiring proactive monitoring, not reactive discovery.

**Key insight:** The system is designed in "Claude dialect" (spawn templates, completion protocol, skill structures). This makes it fragile to model changes and limits the ability to leverage potentially better/cheaper alternatives. Model portability should be a design goal.

---

## The Problem with Current Approach

| Current State | Problem |
|---------------|---------|
| Ad-hoc benchmarks when curiosity strikes | Misses degradation until it's painful |
| Single-task comparisons (N=1) | Can't distinguish model strength from task fit |
| Pass/fail binary assessment | Misses quality dimensions (tokens, time, protocol compliance) |
| "Feels better/worse" subjective | No quantified baselines for comparison |
| Manual side-by-side spawns | High effort, rarely done |
| Marginlab for generic SWE-bench | Doesn't capture orch-specific behavior |
| Claude-dialect prompts | Structural bias against non-Claude models |

---

## Decision

Adopt a two-layer model evaluation strategy:

### Layer 1: External Monitoring (passive, free)

- **Subscribe to Marginlab alerts** for both Claude Code and Codex degradation trackers
- **Track model releases** from Anthropic, OpenAI, Google, and others
- **Purpose:** Early warning that "something changed" — triggers Layer 2

### Layer 2: Orch-Specific Eval Harness (active, periodic)

Build an `orch eval` command that:

1. **Takes reference tasks** — curated set of 3-5 beads issues with known-good outcomes
2. **Spawns across N models** — identical context, parallel execution
3. **Measures quality dimensions:**
   - Compiles / tests pass (binary correctness)
   - Completion protocol followed (orch-specific)
   - Token usage (cost efficiency)
   - Time to complete (throughput)
   - Artifact quality (investigation structure, decision format)
4. **Produces comparison artifact** in `.kb/benchmarks/`
5. **Tracks over time** — detect trends, not just point-in-time

**Reference task types (curate 3-5):**

| Task Type | What It Tests | Skill |
|-----------|--------------|-------|
| Feature implementation | Code generation, test writing | feature-impl |
| Debugging | Root cause analysis, not surface fixes | systematic-debugging |
| Investigation | Protocol compliance, artifact quality | investigation |
| Simple edit | Speed, efficiency, don't overthink | feature-impl (light) |
| Architecture | Reasoning depth, tradeoff analysis | architect |

**When to run:**
- New model release from any provider
- Suspected degradation (Marginlab alert or subjective feel)
- Quarterly health check
- Before changing `default_model` in config

### Layer 3: Model Portability (ongoing, long-term)

Reduce Claude-dialect coupling in the system:

- Identify Claude-specific language in spawn templates
- Create model-neutral completion protocol (or model-specific adapters)
- Test new models with progressively less Claude-specific prompting
- Goal: any capable model should be able to execute orch spawns without model-specific tuning

---

## Rationale

1. **The logout-fix benchmark was the right methodology** — controlled, multi-model, same task. It just needs to be repeatable, not ad-hoc.
2. **Generic benchmarks miss orch-specific behavior** — SWE-bench doesn't test completion protocol, spawn context adherence, or artifact production.
3. **Model landscape changes fast** — Opus 4.6 and GPT-5.3-Codex released same day. DeepSeek V3 appeared seemingly overnight. Evaluation must be lightweight enough to run frequently.
4. **Cheaper models might be good enough** — DeepSeek at $0.25/MTok beat Opus at $200/mo flat on the logout fix. Systematic eval would surface these opportunities.
5. **Claude-dialect coupling is technical debt** — Accepted for now (Jan 24 decision) but should be reduced over time for resilience.

**Trade-offs accepted:**
- Building eval harness takes engineering effort (but pays off in confidence)
- Reference tasks need curation and maintenance
- Model portability work competes with feature work
- Some Claude-specific optimization may be lost in pursuit of portability

---

## Implementation Sequence

1. **Now:** Subscribe to Marginlab alerts (both trackers)
2. **Soon (beads issue):** Design and build `orch eval` command
3. **Ongoing:** Curate reference tasks as good examples emerge from real work
4. **Long-term:** Reduce Claude-dialect coupling in spawn templates
5. **On trigger:** Run eval on new model releases and degradation alerts

---

## Success Criteria

- Can answer "which model for which skill?" with data, not intuition
- Degradation detected within 24h of occurrence (Marginlab + eval)
- New model evaluated within 1 week of release
- `default_model` changes backed by eval data, not vibes
- At least 2 models viable for each skill type (resilience)
