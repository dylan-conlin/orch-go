# Model: Harness Engineering

**Domain:** Multi-Agent Code Quality Practices
**Last Updated:** 2026-03-20
**Validation Status:** WORKING HYPOTHESIS — practices are grounded in one system (orch-go, 3 months). Independent external review (Codex, Mar 10) identified the framework as "software architecture + CI/policy enforcement + tech debt management with agent vocabulary" — strongest as internal operating model, weakest when claiming to be a new discipline. The practices (hard/soft distinction, gate layering, attractor + gate pattern) work in this system. The generalizations are untested. See `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md`.
**Synthesized From:**
- `.kb/investigations/2026-03-07-inv-analyze-accretion-pattern-orch-go.md` — Accretion structural analysis (daemon.go +892 lines, 6 cross-cutting concerns)
- `.kb/threads/2026-03-07-harness-engineering-structural-enforcement-agent.md` — Framework formulation and implementation sequence
- `.kb/models/skill-content-transfer/model.md` — Three-type vocabulary (knowledge/stance/behavioral), 265 contrastive trials
- `.kb/models/architectural-enforcement/model.md` — Four-layer gate mechanisms, threshold calibration
- `.kb/models/entropy-spiral/model.md` — Feedback loops, control plane immutability, 1,625 lost commits
- `.kb/models/extract-patterns/model.md` — Extraction as temporary entropy reduction
- `.kb/models/completion-verification/model.md` — 14-gate pipeline, gate type taxonomy (execution/evidence/judgment)
- OpenAI: "Harness Engineering" (https://openai.com/index/harness-engineering/) — Codex team, ~1M lines, zero manual code
- `.kb/plans/2026-03-11-measurement-instrumentation.md` — Measurement audit: 52% field gaps, 0 gate events, survivorship bias architecture
- `.kb/threads/2026-03-11-measurement-as-first-class-harness.md` — Thread: enforcement without measurement is theological
- `.kb/global/models/signal-to-design-loop/probes/2026-03-20-probe-domain-gate-coverage-gap-physical-cad.md` — LED gate stack: 4-layer gates pass geometry that is non-functional (compositional correctness gap cross-domain evidence)
- Fowler/Bockeler: "Harness Engineering" (https://martinfowler.com/articles/exploring-gen-ai/harness-engineering.html) — Verification gap, relocating rigor

---

## Summary (30 seconds)

Harness engineering is a working label for the practice of making wrong paths mechanically impossible for AI agents, rather than instructing agents to choose right paths. It is not a new discipline — it is software architecture, CI/CD enforcement, and tech debt management applied to multi-agent workflows. It operates through two fundamentally different enforcement types: **hard harness** (deterministic, mechanically enforced, cannot be ignored — pre-commit hooks, spawn gates, `go build`, Go package structure, structural tests) and **soft harness** (probabilistic, context-dependent, driftable — skills, CLAUDE.md, knowledge bases, SPAWN_CONTEXT.md). Hard harness matters more because agents under pressure drift from soft instructions — contrastive testing (265 trials, 7 skills) showed behavioral constraints dilute to bare parity at 10+ co-resident items, and stance transfers only as attention primers, not action directives. **Every harness layer requires both an enforcement surface and a measurement surface** — enforcement without measurement is theological (you believe the gate works), measurement without enforcement is observational (you see problems but can't intervene). Evidence: dupdetect cost 111s invisibly (no measurement), 52% of completions lacked fields for analysis (survivorship bias), accretion.delta covered 4.7% of cases (silently broken). Accretion is entropy: individually correct agent commits compose into structural degradation when shared infrastructure is missing — daemon.go regrew +892 lines past its pre-extraction baseline in 60 days from 30 correct commits. OpenAI arrived at the same framework from greenfield (designed gates before code); we arrived through pain (retrofit after 3 entropy spirals, 1,625 lost commits). A useful lens for thinking about agent failures: **compliance failure** (agent doesn't follow instructions) vs **coordination failure** (agents each follow instructions correctly but collectively produce problems). In practice these are often mixed — not a clean partition. Both are instances of the broader **compositional correctness gap** — individually valid components compose into non-functional wholes because gates validate at the component level while failure emerges at the composition level. This pattern appears across domains: operation→assembly (sheet metal DFM — operations pass individually, assembled part interferes), geometry→function (LED gate stack — 4-layer gates pass valid manifold STL that doesn't work as LED enclosure), agent→system (daemon.go +892 from 30 correct commits). The claim that stronger models make coordination worse is plausible but uncontrolled (observed in one system, no experiment isolating model capability as the variable).

---

## Core Mechanism

### 1. The Harness Taxonomy (Hard vs Soft)

A harness is everything in the development environment that constrains and guides agent behavior. The term comes from OpenAI's Codex team, who built ~1M lines of code with zero manually-written source over 5 months by investing primarily in the harness. The key reframing: **agent failure is a harness bug, not an agent bug.** When daemon.go independently reimplements workspace scanning for the 5th time, the architecture is missing — not the agent.

Every harness component is either hard or soft:

| Property | Hard Harness | Soft Harness |
|----------|-------------|--------------|
| **Enforcement** | Deterministic — passes or fails | Probabilistic — influences via context |
| **Bypass** | Cannot be ignored without escape hatch | Can be drifted from under pressure |
| **Measurement** | Outcome is binary, but cost/coverage are not (dupdetect: 111s invisible, accretion.delta: 4.7% coverage) | Requires contrastive testing to validate effectiveness |
| **Cost** | Higher upfront (code, infrastructure) | Lower upfront (prose, templates) |
| **Degradation** | Stable unless code is modified | Dilutes at scale (5+ constraints = inert) |

**Cross-language portability (Mar 8 probe — opencode TypeScript fork):** The framework (taxonomy, invariants, failure modes) is language-independent. The gate inventory is language-specific. 5 of 8 harness patterns translate directly to TypeScript (deny rules, control plane lock, hook registration, beads close hook, pre-commit accretion gate). 3 need adaptation: build gate has no TypeScript equivalent (bun typecheck has `any` escape hatch), architecture lint needs different tooling (ts-morph/eslint vs go/ast), hotspot analysis needs generated-file exclusion (4 of 10 top opencode hotspots are *.gen.ts code-generated files). "Unfakeability" is a property of structural coupling (schema↔migration, source↔binary), not compilation specifically — TypeScript's Drizzle migration gate is equally unfakeable despite not being a compiler.

**Existing hard harness (orch-go, verified):**

| Mechanism                 | What It Prevents                                      | Source                                                                                                         | Status                                                                                                                             |
|---------------------------|-------------------------------------------------------|----------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------|
| Pre-commit growth gate    | Accretion past 800/600 line thresholds                | `pkg/verify/accretion_precommit.go`, `scripts/pre-commit-exec-start-cleanup.sh`                                | **Shipped, advisory** (reclassified Mar 17) — warn >1500, warning-only >800 (+30 net) / >600 (+50 net). 2-week probe: 100% bypass rate on blocks, 75% hotspot reduction through event-driven extraction. Decision: `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` |
| Spawn hotspot gate        | Feature-impl/debugging on CRITICAL (>1500 line) files | `pkg/spawn/gates/hotspot.go`                                                                                   | **Shipped, advisory** (reclassified Mar 17) — warns + emits event, daemon handles routing. Decision: `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` |
| Build gate (`go build`)   | Broken compilation reaching completion                | `pkg/verify/check.go` — unfakeable gate (Go-specific; TypeScript has no equivalent — see cross-language probe) | Shipped                                                                                                                            |
| Completion accretion gate | Agent-caused growth past thresholds                   | `pkg/verify/accretion.go` (800/1500 thresholds, ±50 delta)                                                     | **Shipped, advisory** (reclassified Mar 17) — warns on agent-caused bloat, exempts pre-existing bloat. Decision: `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` |
| Architecture lint tests   | Forbidden lifecycle state packages/imports            | `cmd/orch/architecture_lint_test.go` (4 tests)                                                                 | Shipped, not in CI                                                                                                                 |
| Spawn rate limiter        | Velocity exceeding verification bandwidth             | `pkg/spawn/gates/ratelimit.go`                                                                                 | Shipped                                                                                                                            |
| Spawn concurrency gate    | Too many parallel agents                              | `pkg/spawn/gates/concurrency.go`                                                                               | Shipped                                                                                                                            |
| Duplication detector      | Cross-file function similarity at completion          | `pkg/dupdetect/`, `pkg/verify/duplication.go`                                                                  | **Shipped** — AST fingerprinting, completion advisory, pre-commit check. O(n²)→O(M×N) after measurement revealed 111s cost         |
| Claude Code deny hooks    | Forbidden tool usage at spawn time                    | `~/.orch/hooks/*.py` (12 scripts, 5 denials)                                                                   | Shipped, mutable                                                                                                                   |

**Existing soft harness (orch-go):**

| Mechanism | What It Influences | Measured? | Measured Effectiveness |
|-----------|-------------------|-----------|-----------------------|
| SKILL.md content | Agent procedure, vocabulary, routing | Yes — 265 contrastive trials | Knowledge: +5 lift. Stance (attention primers): +2 to +7. Behavioral: inert at 10+ |
| CLAUDE.md | Codebase conventions, constraints | No | Unknown — daemon.go grew past stated 1500-line convention. Maintenance investigations (Feb 2026) show CLAUDE.md requires active upkeep: documentation drift (stale pkg/registry/ refs, duplicated sections) and progressive disclosure (accretion boundaries section compressed from 20→4 lines after guarded-file reminder). |
| .kb/ knowledge | Prior findings, decisions, models | No | Unknown |
| SPAWN_CONTEXT.md | Hotspot awareness, skill context | No | Advisory injection, no compliance gate |
| Coaching plugin | Mid-session correction | Partial | Only works for OpenCode spawns, not Claude CLI/tmux |

**The critical asymmetry:** Hard harness doesn't need measurement — a build passes or fails. Soft harness needs contrastive testing to know whether it works at all. The default assumption for soft harness should be "probably doesn't work" until proven otherwise.

### 2. File Growth from Uncoordinated Agents

Files grow when multiple agents add code without shared awareness of prior additions. Each commit is locally rational; the aggregate effect is structural degradation. (Previously labeled "accretion as thermodynamics" — the thermodynamic analogy adds no predictive value beyond "things grow when agents don't coordinate.")

**Primary evidence:**
- `daemon.go` grew +892 lines (667→1559) in 60 days from 30 individually-correct commits. Each added a locally-reasonable capability (stuck detection, health checks, auto-complete, agreement checks, phase timeouts, orphan recovery).
- Extraction is temporary: Jan 2026 extraction reduced main.go by -1058 lines, clean_cmd.go by -670 lines. But daemon.go regrew past its pre-extraction baseline within 2 months.
- Counter-evidence (partially): `spawn_cmd.go` *shrank* -1,755 lines (2,432→677) after `pkg/spawn/backends/` was created (Feb 13) — proving attractors work for initial extraction. However, spawn_cmd.go regrew to 1,160 lines by Mar 8 (+483 in 3 weeks, ~160 lines/week), demonstrating that attractors break the accretion cycle temporarily but re-accretion follows without blocking gates. (Mar 8 probe corrected from -840 claim). **Update (Mar 17 probe):** After pre-commit gate wired + daemon extraction triggers, spawn_cmd.go back down to 542 lines — suggesting attractors WITH gates CAN hold. 1-week data, re-accretion monitoring continues.
- **Extraction is net-positive on lines** (Mar 10 probe): 5 of 6 recent extraction commits ADDED net lines (+12 to +65 each) through file headers, package declarations, and import duplication. Only the daemon extraction was net-negative (-214) because it also removed dead code. Extraction distributes entropy but doesn't reduce it without concurrent dead code removal.
- **New bloated files emerge as fast as old ones are extracted** (Mar 10 probe): In 2 weeks, 3 files crossed 800 lines (userconfig.go 611→975, +page.svelte 778→1,201, hotspot.go 858→1,050) while 3 were extracted below threshold. Net bloated count barely changed.
- 6 cross-cutting concerns independently reimplemented across 4-9 files (~2,100 lines of duplicated infrastructure): workspace scanning, beads querying, output formatting, project resolution, filtering, ID extraction.

**Two forces drive accretion:**

1. **Feature gravity** — New capabilities land in run functions because that's where the Cobra command lives. `runDaemonLoop()` at 702 lines is the gravitational center for anything daemon-related. There's no friction pushing code elsewhere.

2. **Missing shared infrastructure** — Without `pkg/workspace/`, `pkg/display/`, or shared beads querying, each command file must be self-contained. This isn't agent laziness — the shared packages don't exist. Self-containment is the only option.

**Prevention requires both:**
- **Structural attractors** — packages that pull code toward them (e.g., `pkg/spawn/backends/` caused spawn_cmd.go to shrink)
- **Gates** — signaling that triggers extraction cascades (pre-commit growth gate emits events that drive daemon extraction)

Attractors without gates → agents still put code in the old location by habit.
Gates without attractors → agents are warned with nowhere to put code.

**Gate mechanism update (Mar 17 probe):** Gates work through signaling, not blocking. 2-week measurement: 55 gate firings, 2 blocks, both bypassed in <16 seconds (100% bypass rate). But the event emission triggered daemon extraction cascades that reduced hotspots 75% (12→3 files). The blocking path adds friction agents route around instantly; the signaling path triggers daemon responses that produce structural improvement. All accretion gates reclassified from blocking to advisory. See `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md`.

**The thermodynamic analogy:** Attractors are low-energy states that code naturally flows toward. Gates are activation energy barriers preventing code from accumulating in high-entropy states. Extraction without attractors/gates is cooling a room without insulation — it heats right back up.

### 3. Three-Type Vocabulary Transfer

The skill content transfer model (265 trials, 7 skills) discovered three content types that transfer through fundamentally different mechanisms. These map directly to harness components:

| Skill Content Type | Transfer Mechanism | Harness Equivalent | Harness Function |
|---|---|---|---|
| **Knowledge** (facts, routing tables, templates) | Direct — agent reads and applies (+5 lift) | **Context** | What agents see |
| **Stance** (attention primers, not action directives) | Indirect — shifts what agents notice (+2 to +7 on cross-source scenarios) | **Attractors** | Where agents naturally route |
| **Behavioral** (MUST/NEVER prohibitions) | Unreliable — dilutes at 5+, inert at 10+ | **Constraints/Gates** | What agents can't do |

**The mapping reveals the design error:** We put constraint-type content (behavioral prohibitions) in context-type containers (skill documents). Constraints dilute in context. The correct mapping:
- Knowledge → keep in skill/context (resilient, no dilution limit until ~50+)
- Stance → keep in skill/context (only attention primers — "look for X" — not action directives — "do X")
- Behavioral → move to hard harness (every MUST/NEVER should be a hook, gate, or structural test)

**The why behind attractor-as-stance:** When `pkg/spawn/backends/` exists, agents put spawn code there — not because a skill says to, but because the package name primes their attention. Package structure is an attention primer at the architectural level. This is why attractors work: they're persistent, always-visible stance that doesn't compete with system prompt.

### 4. The OpenAI Parallel

OpenAI's Codex team (~1M lines, 1,500 PRs, 3-7 engineers, 5 months, zero manual code) and our accretion discovery are the same insight from opposite directions:

| Dimension | OpenAI (Greenfield) | orch-go (Retrofit) |
|-----------|--------------------|--------------------|
| **Discovery path** | Designed harness before code | Discovered need through pain (3 spirals, 1,625 lost commits) |
| **Scale** | ~1M lines, ~1,500 PRs | ~80K lines, ~6,000 commits |
| **Structural tests** | Custom linters enforcing layered deps (Types→Config→Repo→Service→Runtime→UI) at CI | `architecture_lint_test.go` (4 tests, not in CI) |
| **Entropy management** | "Automated garbage collection" — background agents weekly | Aspirational Layer 3 — not implemented |
| **Documentation** | AGENTS.md as progressive disclosure (~100-line TOC, 88 files) | CLAUDE.md as monolith + .kb/ knowledge system |
| **Agent failure model** | "Agent failure = harness bug — fix the environment" | Same: 5th workspace scanner = missing `pkg/workspace/` |
| **Verification gap** | Noted by Fowler: architectural verification but not behavioral | 14-gate completion pipeline addresses this |

**Three transferable practices:**

1. **Structural tests as first-class artifacts.** OpenAI enforces dependency direction at CI level. Our `architecture_lint_test.go` tests one constraint (no lifecycle state packages). Gap: no tests for function size limits, package boundary violations, or cross-cutting duplication.

2. **Agent failure = harness bug.** Every duplication is a bug report against the architecture, not the agent. The 5th workspace scanner means `pkg/workspace/` is missing.

3. **Entropy management as continuous practice.** OpenAI runs periodic agents scanning for constraint violations. We have detection (hotspot analysis) but not proactive management.

**What OpenAI doesn't address:** How to know WHEN to add new gates. They designed gates before code (greenfield advantage). For retrofit systems, accretion signals (duplication count, regrowth rate, detection-without-prevention) must feed an automated gate discovery system.

**Fowler/Bockeler's critical addition:** The harness approach requires "constraining the solution space" — the opposite of what most expect from AI coding. "Relocating rigor" — rigor doesn't disappear when you stop writing code manually; it migrates to environment design and constraint specification. Also identifies the verification gap: OpenAI doesn't describe functional verification. Our completion pipeline (14 gates, 3 types) addresses this.

**Anthropic's "Effective Harnesses for Long-Running Agents":** Extends OpenAI's framing to multi-session continuity — progress files, mandatory commits, structured feature lists. Single-agent only. Does not address concurrent multi-agent coordination.

**MAST taxonomy (Cemri et al., arXiv:2503.13657):** Closest academic work to our coordination concern. 1,600+ traces, 14 failure modes in 3 categories: FC1 (system design, ~44%) ≈ compliance, FC2 (inter-agent misalignment, ~32%) ≈ coordination, FC3 (task verification, ~24%) ≈ Fowler's verification gap. However, MAST prescribes "deeper social reasoning abilities" — model improvement — for coordination failures that are actually architectural. Their own finding ("a well-designed MAS can result in performance gain when using the same underlying model") supports the architectural interpretation without developing it.

**Context in existing literature (Mar 10 probes):** The term "harness" is used with different emphases: environment setup (OpenAI/Anthropic), verification (Fowler). Our usage focuses on multi-agent coordination. Whether the compliance/coordination distinction adds something beyond existing coordination cost literature is an open question — the claim of novelty was identified as overclaim by independent review. Blog claim review (Mar 10) identified that key concepts map to established literature: "structural attractors" = affordances (Norman, 1988) + nudge theory (Thaler/Sunstein, 2008) + Conway's Law (1967); "dilution curve" = known prompt-length vs. instruction-following degradation; "architecture doing the work of instruction" = Christopher Alexander's Pattern Language (1977). The specific APPLICATION to LLM agent orchestration is novel; the underlying concepts are not.

### 5. The Measurement Surface (Paired with Enforcement)

**Core claim (Mar 11):** Every harness layer needs BOTH an enforcement surface and a measurement surface. Enforcement without measurement is theological — you believe the gate works but can't prove it. Measurement without enforcement is observational — you can see what's happening but can't intervene. The harness engineering model was previously framed as "build enforcement layers." The correct framing is: each layer is a pair of enforcement + measurement, and one without the other is incomplete.

**Evidence that enforcement alone is insufficient:**

| Enforcement Layer | Measurement Gap Found | Consequence |
|---|---|---|
| Duplication detector (hard, deterministic) | No timing telemetry | 111s per completion — invisible until manually profiled. O(n²) cost was architectural, not a bug. |
| Completion pipeline (hard+soft, 15 gates) | 52% of agent.completed events lacked skill/outcome fields | Cannot calculate gate accuracy — no denominator data. Measured survivors, not decisions. |
| Spawn gates (hard, blocking) | 0 gate_decision events emitted | Knew gates existed but not how often they fired, what they blocked, or whether blocks were correct |
| Accretion delta (hard, per-completion) | 4.7% coverage (path filter bug) | 95% of completions silently skipped — gate appeared active but was nearly blind |
| Health score gate (soft masquerading as hard) | Score formula calibrated to pass existing state | 89% of 37→73 improvement was recalibration, not structural. Gate that never fires = false assurance |

**The survivorship bias architecture:** Before Mar 11 instrumentation, the system measured outcomes (completions) but not decisions (gate evaluations, blocks, bypasses). This created survivorship bias — you see what got through, not what was filtered or why. You cannot evaluate gate precision (false positive rate) without logging both blocked and passed cases.

**What each harness type needs measured:**

| Harness Type | Enforcement Surface | Measurement Surface |
|---|---|---|
| Hard (build, gates) | Deterministic pass/fail | **Cost** (pipeline timing), **coverage** (% of events with data), **precision** (false positive rate from gate_decision events) |
| Soft (knowledge) | Context injection | **Effectiveness** (single-turn contrastive tests, +5 lift) |
| Soft (stance) | Attention priming | **Reach** (multi-scenario contrastive tests, bare 0% → stance 83% on cross-source) |
| Soft (behavioral) | MUST/NEVER prohibitions | **Compliance rate** (87 constraints → bare parity at 5/7 scenarios — confirming dilution) |

**The previous claim "hard harness doesn't need measurement" was wrong.** Hard harness outcome is binary (pass/fail), but its operational properties are not: cost (111s dupdetect), coverage (4.7% accretion.delta), precision (false positive rate — now measured: 0% for signal gates, 79% for self_review), and volume (30 gate_decision events since Mar 11 instrumentation). A gate that passes/fails deterministically but costs 111s per run, covers 4.7% of cases, and logs nothing about its decisions is not well-understood — it's merely deterministic.

**Retrospective accuracy audit (Mar 11, Phase 3):** Sampled 173 gate blocks/failures across 8 signal gates. Signal gate aggregate false positive rate: **0%** (0/173). Gates split into two categories: *correctness gates* (build, vet, phase_complete, synthesis, accretion_precommit — 11 events, 0% FP, catch real defects) and *discipline gates* (explain_back, verified, triage — 162 events, 0% FP, measure human process compliance). Discipline gates cannot be "wrong" in the traditional sense — they measure whether a human did something, not whether code is correct. Confidence intervals are wide for low-volume gates (build/vet: [0%, 63%] at 95% CI, n=3) but narrow for high-volume ones (explain_back: [0%, 7%], n=52). The noise-classified self_review gate has 79% FP rate (15/19 failures are intentional CLI output, console.error, or pre-existing code). See `.kb/investigations/2026-03-11-inv-gate-retrospective-accuracy-audit.md`.

**The completion verification pipeline through harness lens:**

| Gate Type | Count | Harness Category | Provenance | Measured FP Rate |
|-----------|-------|-------------------|------------|------------------|
| Execution-based (Build, Vet, Staticcheck) | 3 | **Hard** | Deterministic — cannot be faked | 0% (n=6, wide CI) |
| Evidence-based (Phase, Synthesis, Test Evidence, etc.) | 10 | **Structured soft** | Pattern matching — detects theater, not correctness | 0% for phase_complete/synthesis (n=4); 79% for self_review (n=19) |
| Judgment-based (Explain-back, Behavioral) | 2 | **Human soft** | Human comprehension — valid because human takes responsibility | 0% (n=109, narrow CI) — discipline gates, not correctness gates |

As of Mar 2026, 3 of 15 completion gates run code (up from 1). Vet and staticcheck were added as independent hard gates. Further expansion (actually running tests) would continue increasing hard harness surface.

**Measurement instrumentation status (Mar 11):** Phase 1-3 of measurement plan shipped:
- agent.completed field coverage: 52% → ~100% (bare duplicate events eliminated, all 5 emission paths enriched)
- spawn.gate_decision events: now logged at all gate block/bypass points
- daemon.architect_escalation events: now logged with hotspot match details
- duplication.detected events: now logged with file pairs and similarity scores
- accretion.delta coverage: 4.7% → ~100% (git baseline fix)
- Completion pipeline timing: per-step duration and skip-reason instrumented

Phase 4 (correlation — "do gates improve quality?") blocked on 2-4 weeks of data accumulation. Checkpoint: Mar 24.

**Falsification criteria (Mar 11 design):** Four tests that can kill the model:

| Criterion | Measurement | Threshold | Status |
|-----------|-------------|-----------|--------|
| Gates are ceremony | Accretion velocity pre/post gate | Post-gate velocity <50% of pre-gate for 2+ weeks | **Inconclusive at 1 week:** raw velocity -25% (6,131→4,597/wk), but per-commit velocity only -5.6% (confounded by lower activity). Structural improvement dramatic: hotspot count 12→3 (75% reduction). Gate blocks rare (3.6%) and all bypassed. Velocity metric may be wrong — structural health (hotspot count) better captures gate effect. Checkpoint Mar 24 still needed with commit-normalized data. |
| Gates improve agent quality | Enforced vs bypassed cohort comparison | Measurable quality difference between enforced and bypassed cohorts | **No measurable difference (Mar 17).** 529 spawns, 258 with gates. Enforced: 81.1% completion, 100% verification. Bypassed: 74.2% completion, 100% verification. Delta within noise. Gates don't filter for individual agent quality — they create systemic pressure (extraction cascades, hotspot reduction). This is the "signaling infrastructure" finding: gates work by making structural problems visible, not by blocking bad agents. |
| Gates are irrelevant | Gate fire rate (evaluations / spawns) | Fire rate <5% = irrelevant | Falsified: 69.5% fire rate from legacy events |
| Soft harness is inert | Controlled A/B removal test | Removal causes no outcome difference | Not measurable (no experiments yet) |
| Framework is anecdotal | Second system deployment | No benefit in second system | Not measurable (single system) |

**Design principles:**
1. Every enforcement component should have a corresponding measurement surface before being treated as "working."
2. Every soft harness component should be contrastively validated before deployment. If adding content to a skill doesn't measurably change behavior, it's dead weight.
3. Measurement infrastructure is itself subject to the hard/soft taxonomy — a measurement that can be gamed or miscalibrated is soft measurement (health score formula). A measurement that reads from deterministic sources (git diff, event logs) is hard measurement.
4. Falsification requires both a measurement surface AND a threshold. A measurement without a pass/fail criterion is observation, not testing.

### 6. Implementation Layers

Each layer builds on the previous. Lower layers are more immediately actionable. Each layer now tracked as enforcement+measurement pair:

| Layer | Enforcement Surface | Status | Measurement Surface | Status |
|-------|-------------------|--------|-------------------|--------|
| **0** | Pre-commit growth gate | **Shipped** | accretion.delta events, pipeline timing | **Shipped** (Mar 11) |
| **1** | Structural tests (arch lint) | **Partial** (4 tests, not in CI) | Test pass/fail in pre-commit | Not started |
| **2** | Duplication detector | **Shipped** (Mar 11) | duplication.detected events, pipeline timing | **Shipped** (Mar 11) |
| **3** | Periodic entropy agent | Not started | Growth trend tracking (orch stats) | **Partial** |
| **4** | Gates that generate gates | Aspirational | Gate accuracy correlation (Phase 4) | Blocked on data |

**Layer 0 status (Mar 17, 1-week data):** Enforcement fully shipped. Measurement now paired: accretion.delta coverage fixed from 4.7% → ~100% (git baseline bug), pipeline timing instrumented. The pre-commit hook calls `orch precommit accretion` which runs `CheckStagedAccretion`. Hard block at >1500 lines. Warning-only at >800 lines (+30 net delta) and >600 lines (+50 net delta). Override: `FORCE_ACCRETION=1 git commit ...`. **Pre-commit gate wired Mar 10.** First-week data (Mar 17 probe): 55 gate decisions — 51 allow, 2 block, 2 bypass. Both blocks were immediately force-bypassed (100% bypass rate). Direct blocking negligible. Indirect effect substantial: combined with daemon extraction triggers and hotspot enforcement, 11 of 12 previously-bloated files shrank below 800 lines (12→3 hotspot count, 75% reduction). daemon.go went from 1,559→197 lines. Gate's primary mechanism is extraction pressure, not blocking. Raw velocity -25% but confounded by -19% commit activity; per-commit reduction only 5.6%. **Checkpoint Mar 24 remains: need commit-normalized data across 2+ weeks.**

**Layer 1 needs extension.** Current structural tests enforce only the no-lifecycle-state constraint (from two-lane architecture decision). Missing: function size limits for cmd/orch/, package boundary enforcement, cross-cutting duplication detection. These 4 tests also aren't in CI — they require manual `go test` execution.

**Layer 2 status (Mar 11):** Enforcement shipped (AST fingerprinting, completion advisory, pre-commit check). Measurement shipped (duplication.detected events with file pairs and similarity scores). Original O(n²) implementation cost 111s per completion — invisible until measurement surface (pipeline timing) was added. Fixed to O(M×N) scoped comparison (43x speedup). This is the canonical example of why enforcement needs measurement: a deterministic gate that silently costs 2 minutes per agent completion is worse than no gate.

**Layer 3 is OpenAI's "garbage collection" pattern.** A periodic agent reviewing duplication detector output, growth trends, and structural test results. Produces recommendations: "pkg/workspace/ needed," "daemon.go periodic tasks should extract."

**Layer 4 is the meta-layer.** When the entropy agent identifies a pattern 3+ times, it drafts the structural test that would prevent it. The harness extending itself. This is aspirational.

### 7. Conventions Without Gates Tend to Erode

In orch-go, conventions without enforcement have been violated. This is consistent with known principles (unenforced policy decays, adding is cheaper than removing). The following observations are from this system:

1. **No persistent memory.** Each agent session starts fresh. Conventions documented in context compete with system prompt (17:1 signal disadvantage) and task pressure.

2. **Volume overwhelms vigilance.** At 45+ commits/day, no human can verify convention compliance. Unverified conventions are unenforced conventions.

3. **Locally rational violations compound.** Each violation is small and justifiable. "I added workspace scanning inline because the task was urgent." Multiply by 30 agents and 60 days: +2,100 lines of duplicated infrastructure.

**Useful mappings (not novel — this is standard software architecture applied to agents):**
- Package structure routes where agents put code
- Import boundaries constrain dependencies
- Structural tests enforce invariants
- Pre-commit hooks prevent known-bad patterns
- Escape hatches (`--force-hotspot --architect-ref`) allow bypass with justification

### 8. The Compositional Correctness Gap

#### Compliance vs Coordination (a lens, not a partition)

A useful way to think about agent failures, though in practice most failures are hybrid:

| Property | Compliance Failure | Coordination Failure |
|----------|-------------------|---------------------|
| **What breaks** | Agent doesn't follow instructions | Agents each follow instructions correctly but collectively produce entropy |
| **Example** | Agent ignores 1,500-line convention | 30 agents each add locally-correct code, daemon.go grows +892 lines |
| **Root cause** | Insufficient capability or context pressure | No shared memory, no structural coordination across agents |
| **Solved by stronger models?** | Yes — Opus stall rate ~4% vs non-Anthropic 67-87% | No — made *worse* by faster, more confident agents |
| **Harness layers** | Layer 0-1 (compliance gates) | Layer 2-4 (coordination gates) |
| **Trajectory** | Simplifies with model improvement | Becomes more important with model improvement |

**The daemon.go evidence is coordination failure, not compliance failure.** Each of the 30 commits followed instructions. Each was locally rational. Each passed review. The problem was the absence of structural coordination — no shared `pkg/workspace/`, no deduplication detection, no cross-agent awareness that workspace scanning was already implemented 4 times.

**The analogy:** A company of 30 brilliant engineers with no architecture review still produces spaghetti — possibly faster than 30 mediocre engineers, because each builds more in less time. Architecture review isn't compensating for incompetence. It's providing the coordination layer that individual competence cannot.

**Note on permanence claims:** A previous version of this model claimed harness engineering is "a permanent discipline, not transitional." This is plausible but unvalidated — it's a prediction about the future trajectory of model capabilities vs coordination needs, based on 3 months of observation in one system. The practices are useful now; whether they remain necessary as models improve is an open question.

#### Generalization: Compositional Correctness Gap

Compliance vs coordination is one instance of a broader failure mode class: the **compositional correctness gap**. Individually valid components compose into non-functional wholes because validation gates operate at the component level while failure emerges at the composition level. This pattern appears at multiple abstraction scales:

| Scale | Components Validated | Composition That Fails | What No Gate Checks |
|-------|---------------------|----------------------|-------------------|
| **Operation → Assembly** (sheet metal DFM) | Individual cuts, bends, hardware insertions each pass DFM rules | Composed assembly: bend line crosses hardware location, cuts weaken fold region, hardware collides after bending | Inter-operation interference in physical assembly |
| **Geometry → Function** (LED gate stack) | Parameters valid, CGAL manifold, polygon budget met, build plate fit | Cut-channel LED routing produces disconnected channels — valid STL, non-functional enclosure | Connectivity/routing across geometric features |
| **Agent → System** (harness engineering) | Each commit compiles, passes review, is locally rational | 30 correct commits produce +892 lines, 6 duplicated concerns, structural degradation | Cross-agent coherence over time |

**The structure is identical in all three cases:**
1. Every component passes all applicable validation gates
2. The composed whole fails at a property no gate measures
3. The failure is invisible to every existing gate because gates validate at a different abstraction level than function

**Cross-domain evidence:**

**LED magnetic letters (Mar 20 probe, ~150 renders):** An OpenSCAD 4-layer gate stack (parameter validation → geometry check → printability → intent alignment) was tested on LED channel routing for letter-shaped enclosures. The cut-channel approach (intersection of zigzag pattern with inner letter profile) passed all 4 gate layers for every letter tested — valid parameters, manifold geometry, printable solid, within polygon budget. But the channels are disconnected for every non-rectangular letter (A, M, W, H, O, L). Diagonal strokes clip horizontal channels into isolated segments. The LED strip has no continuous path. A functionally broken design that passes every gate. The guide-rail approach (raised horizontal rails clipped to inner profile) also passes all gates AND produces functional routing — but no gate can distinguish the two. See `.kb/global/models/signal-to-design-loop/probes/2026-03-20-probe-domain-gate-coverage-gap-physical-cad.md`.

**Sheet metal DFM (SendCutSend domain, now directly tested):** Individual manufacturing operations each have well-defined validation rules — minimum bend radius, minimum hole-to-edge distance, minimum tab width, hardware insertion force limits. Each operation passes its own DFM check. But composed assembly reveals interference: a bend line crossing a hardware location deforms the insert, cut patterns weakening a fold region cause cracking, hardware placements that clear in flat layout collide after bending. **Direct evidence (Mar 20 probe):** SCS's AI Part Builder (powered by Smithy, api.smithy.cc — a third-party geometry engine) was tested with "L-bracket with PEM nuts near the bend line." Smithy generated valid geometry with PEM holes near the bend line and zero DFM warnings. The integration is a one-way handoff: Smithy generates STEP → one-way gate ("Model cannot be edited after continuing") → SCS quoting. DFM issues are discovered only after the user loses the ability to edit. **0% recall** on hardware+bend conflicts — a regression from SCS's own DFM tools (46.2% recall in Fin bot analysis). The gap is specifically at the third-party integration boundary: Smithy validates geometry, SCS validates manufacturing, neither validates that the geometry is manufacturable. See `.kb/models/smithy-geometry-engine/model.md` for full Smithy model and `.kb/models/harness-engineering/probes/2026-03-20-probe-scs-ai-part-builder-compositional-correctness.md`.

**daemon.go coordination failure (existing evidence):** 30 agent commits, each individually correct, composing into +892 lines of structural degradation with 6 cross-cutting concerns independently reimplemented. The build gate, review, and local rationality all passed — the composition property (system coherence) was unchecked.

**Why naming this matters:** "Coordination failure" accurately describes the agent case but doesn't capture the cross-domain pattern. A sheet metal assembly doesn't "coordinate" — its operations are sequenced by a machine. LED channel routing doesn't involve multiple agents. The unifying concept is that **validation gates check component properties while failure emerges from composition properties** — and this gap exists in any domain where validation and function operate at different abstraction levels.

**Implications for gate design:** The compositional correctness gap predicts where enforcement will fail: wherever the gate taxonomy covers components but not their composition. Closing the gap requires gates that operate at the composition level — assembly simulation for DFM, connectivity analysis for CAD routing, cross-agent deduplication for code. These gates are harder to build (they require domain-semantic understanding of what "correct composition" means), which is why the gap persists.

#### Which gates are which

| Gate | Type | Trajectory |
|------|------|-----------|
| Pre-commit growth gate | **Signaling** (reclassified Mar 17) | Neither compliance nor coordination — creates visibility that triggers extraction. 100% bypass rate on blocks, but 75% hotspot reduction through indirect pressure. |
| Build gate (`go build`) | Compliance | Permanent — compiler is always needed |
| Spawn hotspot gate | Coordination | Permanent — prevents uncoordinated work on degraded areas |
| Architecture lint | Coordination | Grows — more structural invariants as system matures |
| Duplication detector | Coordination | Grows — catches cross-agent redundancy |
| Entropy agent | Coordination | Grows — system-level health monitoring |
| Gates that generate gates (Layer 4) | Coordination | The endgame — coordination infrastructure that extends itself |

---

## Critical Invariants

1. **Hard harness for enforcement, soft harness for orientation.** Behavioral prohibitions in skill documents produce the worst of both — unreliable enforcement that dilutes reliable knowledge transfer. The orchestrator skill had 87 behavioral constraints; ~83 were non-functional.

2. **Every convention without a gate will eventually be violated.** A convention in CLAUDE.md without infrastructure enforcement is a suggestion with a half-life proportional to context window pressure. daemon.go grew past the stated 1,500-line convention.

3. **"Agent failure is harness failure" — useful heuristic, not universal truth.** Asking "what's missing from the harness?" before "what's wrong with the agent?" is a productive default. But as stated universally, this is unfalsifiable — if every bad outcome is reclassified as a harness bug, the framework absorbs any result. Some failures are genuinely agent-level (model limitations, context window issues). Use as a design heuristic, not an axiom.

4. **Extraction without routing is a pump.** Moving code out of a file without creating an attractor (destination package) results in re-accretion. The gravitational center must be relocated, not just temporarily emptied. daemon.go +892 lines post-extraction proves this.

5. **Prevention > Detection > Rejection.** Each layer further from authoring has higher cost. Pre-commit gate (prevention) < spawn gate (early detection) < completion gate (late detection + wasted work).

6. **Mutable hard harness is soft harness with extra steps.** All current defenses (spawn gates, verify gates, hooks, architecture lint) are source code agents can modify. This is the entropy spiral's core vulnerability. True immutability requires infrastructure that's architecturally unreachable by agents.

7. **Enforcement without measurement is theological; enforcement with measurement is empirical.** A gate you can't measure is an assertion you can't test. The dupdetect gate (111s invisible cost, 4.7% accretion coverage) demonstrates that even hard, deterministic enforcement can be operationally broken without a measurement surface. Every harness layer must be a pair: enforcement surface + measurement surface. One without the other is incomplete. (Evidence: Mar 11 instrumentation audit — 52% field gaps, 0 gate_decision events, survivorship bias architecture.)

8. **Stronger models may need more coordination gates, not fewer.** Hypothesis: compliance gates simplify with model capability, but coordination gates grow in importance as agents get faster. Observed in one system (faster agents produced more code per session). Not experimentally controlled — this is a plausible claim, not a validated one. **Instrumentation update (Mar 20):** model field now populated in `session.spawned` events (all backends). `accretion.delta` events have model field in schema but line-count fields (`code_added`, `code_net`) are not populated — only 3 of 425 accretion.delta events have model data, none with line counts. Controlled comparison still blocked. **Coordination demo evidence (N=160, haiku-only):** coordination conditions reduce per-agent accretion 8-12% vs no-coord baseline (Cohen's d: 0.17-0.29, small effect). Placement (file routing) most effective at -12.2%. Effect larger on complex tasks (-14.2%) than simple (-8.3%). This shows coordination gates reduce accretion even for a single model tier — the cross-model comparison (haiku vs opus) has not been run. **Back-of-envelope math** suggests the system-level effect is large: if Opus completes ~96% vs non-Anthropic ~20%, total system accretion scales ~5x even if per-session accretion is identical. The stated falsification criterion ("less accretion per agent-session") may target the wrong metric — coordination is a system-level property, not per-session.

---

## Why This Fails

### 1. Soft Harness Masquerading as Hard

**What happens:** Documentation says "files >1,500 lines require extraction." Agent reads this, understands it, adds 200 lines to an 1,800-line file anyway because the task is urgent and no gate blocks it.

**Evidence:** daemon.go grew 667→1559 lines while the >1,500 convention existed in CLAUDE.md. Only after the spawn hotspot gate was implemented did blocking actually occur.

### 2. Gate Calibration Death Spiral

**What happens:** Gate too strict → high false positive rate → `--force` reflex → gate becomes noise → no enforcement.

**Evidence:** Original strategic-first hotspot gate blocked ALL non-architect spawns. Build fixes, investigations, and low-risk work all blocked.

**Fix:** "The fix for an ignored gate is never 'make it louder' — it's 'make it more precise.'" Tiered enforcement (warning at 800, hard gate at 1,500) with skill-based exemptions.

### 3. Attractors Without Gates (and Vice Versa)

**What happens (attractors without gates):** `pkg/daemon/` exists (896 lines) but new features still land in `cmd/orch/daemon.go` because the Cobra command lives there. Attractor exists but no gate prevents the old path.

**What happens (gates without attractors):** Pre-commit warns status_cmd.go is too large, but no `pkg/display/` or `pkg/workspace/` exists for extracted code. Agent sees warning but has nowhere to go.

### 4. Mutable Control Plane

**What happens:** All defenses live inside the system agents can modify. Three entropy spirals (1,625 lost commits) occurred with mutable infrastructure. Circuit breaker is currently disabled.

**Mitigation (partial):** Compiled binary provides temporal buffer. Full resolution requires `chflags uchg` on hook files, removing `Edit(*/.claude/*)` from allow list, re-enabling circuit breaker.

### 5. Generated Code False Positives

**What happens:** Hotspot analysis, accretion gates, and architect routing treat all files equally. In TypeScript (and Go with protobuf), large generated files trigger false positives — routing architect sessions for files no agent authored.

**Evidence (Mar 8 probe):** `orch hotspot` on opencode found 48 bloated files and 155 hotspots. 4 of the top 10 were `*.gen.ts` code-generated SDK files (5,070, 3,909, 3,318 lines). These would trigger spawn gate blocking and architect routing despite being machine-generated.

**The broader pattern:** Any codebase with code generation (OpenAPI codegen, GraphQL, protobuf, icon component generators) will have inflated hotspot counts. Harness tooling needs a generated-code exclusion mechanism (`.orchignore` or pattern-based filtering) to maintain gate precision.

### 6. Score Calibration as Soft Harness in Disguise (RESOLVED)

**What happened:** Health score gate (blocked feature-impl when score < 65) used a formula calibrated to produce passing scores. 89% of the 37→73 improvement was formula changes, not structural improvement. At baseline values the new formula scored 69.2 — above the gate threshold — with zero extractions.

**Resolution (Mar 11):** Gate removed entirely from spawn path. Health score remains as diagnostic metric (`orch health` / `orch doctor`) but no longer pretends to gate spawns. Real enforcement comes from pre-commit accretion gate and hotspot blocking. The principle: a gate whose trigger condition is recalibrated to pass existing state is functionally equivalent to removing the gate — so remove it honestly rather than leaving a false signal.

**The broader pattern:** Honesty over ceremony. An advisory gate that never fires provides false assurance. Better to have fewer gates that actually enforce than more gates that create the appearance of enforcement.

### 6b. Gate Exemptions as Permanent Bypasses

**What happens:** Gate designed to prevent accretion exempts "pre-existing bloat" — files already over the threshold receive warnings instead of blocks. Once a file crosses 1,500 lines, it can never be blocked again. This creates a ratchet: bloat begets exemption begets more bloat.

**Evidence (Mar 8 probe):** daemon.go at 1,559 lines. The completion accretion gate (`pkg/verify/accretion.go` lines 128-136) downgrades from ERROR to WARNING when `preChangeLines > AccretionCriticalThreshold`. Every future agent adding 50+ lines to daemon.go gets a non-blocking warning. The gate structurally cannot enforce on the files that need enforcement most.

**The broader pattern (Mar 8):** 12 files in cmd/orch/ exceeded 800 lines (total: ~14,000 lines). 6 exceeded 1,000. The exemption means the completion gate is primarily useful for files approaching the threshold, not files that have already passed it. **Update (Mar 17 probe):** Hotspot count dropped 12→3 (75% reduction). 11 of 12 previously-bloated files shrank below 800 via extraction. daemon.go: 1,559→197 (-87%). The exemption issue is partially resolved by extraction removing the pre-existing bloat rather than enforcing against it. However, new extraction targets are emerging (stats_aggregation.go at 959 lines).

### 7. Enforcement Without Measurement (Invisible Cost)

**What happens:** Hard harness is shipped and assumed to work because its outcome is deterministic. But cost, coverage, and precision go unmeasured, creating invisible operational burdens.

**Evidence:**
- Duplication detector: deterministic (finds duplicates or doesn't), but cost 111s per completion. O(n²) function comparison across entire project. No timing telemetry existed to surface this — discovered only by manual profiling after agents started timing out.
- Accretion delta: deterministic (computes line changes), but path filter bug restricted it to workspace dir. 95.3% of completions silently skipped. The gate appeared active in code but was nearly blind operationally.
- Spawn gates: deterministic (block or allow), but 0 events logged about decisions. Could not answer "how many spawns did the hotspot gate block this week?" — the denominator for gate accuracy was unrecorded.
- Agent.completed: 52% of events lacked skill/outcome fields. Survivorship bias: measured completions but not the decisions that led to them.

**Fix:** Pair every enforcement layer with measurement from day one. The measurement plan (Mar 11) shipped instrumentation across all 4 gaps: pipeline timing, gate_decision events, accretion coverage fix, field enrichment.

### 8. Measurement Artifacts in Soft Harness

**What happens:** Soft harness appears to work based on flawed measurement, creating false confidence.

**Evidence:** The "detection-to-action gap" (agents detect but still approve completion) was a measurement artifact — negation indicators failed when agents refused by naming what they refused. Re-scoring with positive refusal detection showed 0/6 → 6/6. Even the measurement infrastructure for soft harness is itself soft.

---

## Constraints

### Why Hard Over Soft?

**Constraint:** Instruction hierarchy (system > user) means skill content is structurally subordinate to system prompt defaults. Behavioral constraints compete at 17:1 disadvantage.

**This enables:** Simple skill documents focused on knowledge and stance
**This constrains:** All enforcement must be infrastructure, not prose

### Why Both Attractors and Gates?

**Constraint:** Gates without attractors block agents with nowhere to go. Attractors without gates are optional.

**This enables:** Agents blocked from wrong paths AND guided toward right paths
**This constrains:** Cannot ship a gate without ensuring the alternative path exists

### Why Entropy Management Is Continuous?

**Constraint:** Extraction is temporary entropy reduction. daemon.go regrew past pre-extraction baseline in 60 days.

**This enables:** Proactive detection before CRITICAL thresholds
**This constrains:** Cannot treat extraction as "done" — must monitor for re-accretion

### Why Enforcement Needs Measurement?

**Constraint:** Hard harness outcome is binary but its operational properties (cost, coverage, precision) are continuous and can silently degrade.

**This enables:** Data-backed evaluation of gate effectiveness (Mar 24 checkpoint)
**This constrains:** Cannot ship enforcement without corresponding measurement surface — must instrument before declaring "shipped"

### Why Package Structure Matters More Than Instructions?

**Constraint:** Package structure is persistent across all agent sessions — always visible, always enforced (by the compiler). Instructions compete with system prompt and degrade under pressure.

**This enables:** Architecture-as-communication design philosophy
**This constrains:** Cannot compensate for bad architecture with good instructions

---

## Relationship to Other Models

```
                    Harness Engineering
                   (this model — the frame)
                          │
           ┌──────────────┼──────────────────┐
           │              │                  │
    Architectural    Entropy Spiral    Skill Content
    Enforcement      (why soft          Transfer
    (hard harness     harness fails    (vocabulary for
     mechanisms)      under pressure)   classifying
           │              │             harness content)
           │              │                  │
           └──────────────┼──────────────────┘
                          │
              ┌───────────┴───────────┐
              │                       │
       Extract Patterns       Completion Verification
       (temporary entropy     (14-gate pipeline:
        reduction)             1 hard, 11 evidence,
                               2 judgment)
```

- **Architectural Enforcement** details how hard harness mechanisms work (spawn gates, completion gates, coaching, CLAUDE.md).
- **Entropy Spiral** explains what happens when soft harness fails at scale — locally correct changes compose into globally incoherent systems.
- **Skill Content Transfer** provides the vocabulary (knowledge/stance/behavioral → context/attractors/constraints) for classifying harness content.
- **Extract Patterns** describes extraction mechanics — necessary but not sufficient without attractors and gates.
- **Completion Verification** shows the gate type taxonomy: 1 execution-based (hard), 11 evidence-based (structured soft), 2 judgment-based (human soft). The sharp boundary at execution is the harness engineering frontier.

---

## Evolution

**2025-12-21 to 2026-02-12:** Three entropy spirals (1,625 lost commits). System learned intellectually (post-mortems) but not structurally (no gates implemented between spirals). Origin story — pain as discovery mechanism.

**2026-01-03 to 2026-01-08:** Code extraction round. main.go -1058, clean_cmd.go -670. Redistribution without deduplication. Moved the problem.

**2026-02-14:** Four-layer architectural enforcement designed. First structural thinking about hard harness.

**2026-02-26:** Three-layer hotspot enforcement decision. `--force-hotspot` + `--architect-ref`. Investigation→architect→implementation sequence infrastructure-enforced.

**2026-03-01 to 2026-03-06:** Skill content transfer experiments (265 trials, 7 skills). Behavioral dilution empirically confirmed. Three-type vocabulary crystallized the hard/soft distinction.

**2026-03-07 (morning):** Accretion structural analysis. daemon.go +892 lines, 6 cross-cutting concerns. "Extraction without routing is a pump." Pre-commit growth gate thresholds corrected.

**2026-03-07 (afternoon):** OpenAI parallel discovered. Thread synthesized. This model created, unifying accretion, skill dilution, OpenAI practices, and existing enforcement models into the harness engineering framework.

**2026-03-08:** Harness engineering plan completed — 13/13 issues across 6 phases shipped. Layers 0-5 implemented: structural attractors (pkg/workspace/, pkg/display/, pkg/beadsutil/), structural tests (function size lint, package boundaries), pre-commit hardening (>1500 blocking gate), duplication detector (AST fingerprinting + beads auto-issue), entropy agent (orch entropy + weekly launchd), control plane immutability (chflags uchg + deny rules + orch harness lock/unlock/verify). MVH checklist produced. `orch harness init` automates Day 1 governance.

**2026-03-11:** Measurement reframed as first-class paired surface. Instrumentation audit (52% field gaps, 0 gate events, 111s invisible dupdetect cost, 4.7% accretion coverage) proved enforcement without measurement is operationally broken. Model updated from "3-layer enforcement stack" to "paired enforcement+measurement surfaces." Invariant #7 added. New failure mode (#7: Enforcement Without Measurement) added. Phase 1-3 of measurement plan shipped: field enrichment, gate_decision events, duplication.detected events, pipeline timing, accretion coverage fix.

**2026-03-08 (evening):** Compliance vs coordination failure mode distinction crystallized. daemon.go +892 was coordination failure (30 agents each correct, collectively incoherent), not compliance failure. Stronger models fix compliance but worsen coordination — faster agents accrete more confidently. Harness engineering reframed as permanent discipline (coordination infrastructure) rather than transitional (training wheels). Publication plan created with 4 phases: deepen model → cross-language evidence → publication draft → portable tooling.

**2026-03-17:** First post-gate effectiveness measurement (1 week since Mar 10 wiring). Raw velocity -25% (6,131→4,597/wk), but confounded by lower commit activity (-19%); per-commit velocity only -5.6%. Gate's direct blocking negligible (2 blocks, both bypassed). Indirect effect dramatic: hotspot count 12→3, daemon.go 1,559→197. Gate works as coordination mechanism (extraction pressure) not compliance mechanism (blocking). Falsification criterion #1 inconclusive — velocity metric may be wrong measure; structural health (hotspot count, file size Gini) better captures gate effect. Checkpoint Mar 24 needs commit-normalized data.

**2026-03-20:** §8 generalized from "compliance vs coordination" to "compositional correctness gap." Two independent cross-domain evidence sources (LED magnetic letters gate stack, SendCutSend sheet metal DFM) showed the same structure as daemon.go coordination failure: individually valid components compose into non-functional wholes because validation gates operate at the component level. Named concept added. Three-scale evidence table added (operation→assembly, geometry→function, agent→system).

---

## References

**Investigations:**
- `.kb/investigations/2026-03-07-inv-analyze-accretion-pattern-orch-go.md` — Primary accretion evidence
- `.kb/investigations/2026-03-07-inv-add-pre-commit-growth-gate.md` — Layer 0 implementation
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` — Four-layer enforcement design
- `.kb/investigations/2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md` — Cross-investigation synthesis
- `.kb/investigations/2026-02-14-inv-add-claude-md-accretion-boundaries.md` — CLAUDE.md as soft harness: progressive disclosure pattern (20→4 lines)
- `.kb/investigations/2026-02-14-inv-fix-claude-md-remove-deleted.md` — CLAUDE.md documentation drift: stale refs to deleted pkg/registry/, duplicated sections
- `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md` — Compositional correctness gap synthesis (LED gates + DFM + daemon.go)

**Thread:**
- `.kb/threads/2026-03-07-harness-engineering-structural-enforcement-agent.md`

**Related Models:**
- `.kb/models/architectural-enforcement/model.md` — Hard harness mechanisms
- `.kb/models/entropy-spiral/model.md` — Soft harness failure at scale
- `.kb/models/skill-content-transfer/model.md` — Three-type vocabulary, contrastive measurement
- `.kb/models/extract-patterns/model.md` — Extraction mechanics
- `.kb/models/completion-verification/model.md` — 14-gate pipeline, gate type taxonomy
- `.kb/models/smithy-geometry-engine/model.md` — Smithy (SCS AI Part Builder's geometry engine): compositional correctness gap at third-party integration boundary

**Decisions:**
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md`
- `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md`

**Probes:**
- 2026-03-17: Pre-commit accretion gate 1-week effectiveness — raw velocity -25% but per-commit only -5.6% (activity confound). Hotspot count 12→3 (75% reduction). Gate works via extraction pressure, not blocking. Falsification criterion #1 inconclusive. Gate effectiveness cohort (529 spawns): enforced 81.1% vs bypassed 74.2% completion (noise), both 100% verification. Gates are signaling infrastructure, not quality gates.
- 2026-03-13: Duplication detector precision — AST fingerprinting precision measurement
- 2026-03-13: Hotspot gate cost/precision — gate cost and false positive measurement
- 2026-03-11: Measurement surface design falsification — paired enforcement+measurement validation
- 2026-03-08: 30-day accretion trajectory — baseline established, pre-commit gate dead code identified
- 2026-03-10: Health score calibration — 89% improvement from recalibration, gate removed
- 2026-03-08: Publication draft — model synthesis for blog post
- 2026-03-08: Cross-language harness portability — TypeScript fork validation
- 2026-03-10: Blog post uncontaminated claim review — external review preparation

**External:**
- OpenAI: https://openai.com/index/harness-engineering/
- Fowler/Bockeler: https://martinfowler.com/articles/exploring-gen-ai/harness-engineering.html
- Anthropic: https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents — Single-agent, multi-session. "Harness" = orchestration framework for context continuity. Does NOT address multi-agent coordination.
- Cemri et al., "Why Do Multi-Agent LLM Systems Fail?" arXiv:2503.13657 — MAST taxonomy: 1,600+ traces, 14 failure modes, 3 categories. FC2 (inter-agent misalignment) = ~32% of failures. Frames solution as "deeper social reasoning" (model improvement), not architecture. Their FC1/FC2/FC3 maps to our compliance/coordination/verification but they don't recognize opposite model-improvement trajectories.

**Primary Evidence (Verify These):**
- `cmd/orch/daemon.go` — +892 lines in 60 days (feature gravity evidence)
- `cmd/orch/spawn_cmd.go` — shrank -840 lines after `pkg/spawn/backends/` (attractor evidence)
- `pkg/spawn/gates/hotspot.go` — Spawn gate with `--architect-ref` verification
- `pkg/verify/accretion.go` — Completion accretion gate (800/1500 thresholds, ±50 delta)
- `pkg/verify/*_precommit.go` — Pre-commit growth gates (Layer 0): accretion, duplication, model-stub
- `cmd/orch/architecture_lint_test.go` — Structural tests (Layer 1, partial)

## Probes

- 2026-03-07: Completion verification through harness lens — Confirms hard/soft taxonomy: 1 of 14 gates is execution-based (hard), 11 evidence-based (structured soft), 2 judgment-based (human soft). Build gate is the only unfakeable gate.
- 2026-03-08: 30-day accretion trajectory measurement — **Gates have NOT bent the line count curve.** daemon.go hit 1,559 (CRITICAL) despite all deployed gates. Completion accretion gate exempts pre-existing bloat. Pre-commit accretion gate exists in code but is NOT wired into the hook. spawn_cmd.go shrank -1,755 (not -840 as claimed) then regrew +483 in 3 weeks. Total cmd/orch/: 47,605 lines across 125 files, 12 files >800 lines. Fix:feat ratio spike (1.21) was transient, reverted to 0.36. Confirms invariants #2 and #4. Extends model with gate exemption failure mode and dead code enforcement gap.
- 2026-03-08: Cross-language harness portability (Go → TypeScript) — **Framework is language-independent, gates are not.** 5/8 harness patterns translate directly to TypeScript. Build gate (`go build`) has no TypeScript equivalent — `bun typecheck` has `any` escape hatch and is pre-push only. "Unfakeability" is structural coupling (schema↔migration, source↔binary), not compilation. Generated code creates false positives: 4/10 top opencode hotspots are *.gen.ts files. TypeScript has own domain-specific hard harness (Drizzle migration gate) that Go lacks. Extends model with cross-language portability analysis and generated-code blind spot.
- 2026-03-08: Publication draft model synthesis — **All core claims survived synthesis into publication format.** Compliance vs coordination failure distinction is the strongest novel claim for external audiences. Three claims need more evidence before strong external assertion: "stronger models need more coordination gates" (no controlled experiment), soft harness budget curve (shape unknown), cross-language portability (dry-run only, not 30-day operation). Honest negative evidence (gates haven't bent the curve) strengthens credibility. spawn_cmd.go correction (-1,755, not -840) confirmed.
- 2026-03-10: Publication polish — related work positioning — **Compliance/coordination distinction is novel in published literature.** Positioned against 4 sources: OpenAI (harness = environment setup), Anthropic (harness = session orchestration), Fowler (harness = verification), MAST/Cemri et al. (observes coordination failures but prescribes model solutions). MAST's FC1/FC2/FC3 maps to compliance/coordination/verification but they don't recognize opposite model-improvement trajectories. "Deeper social reasoning" is a compliance answer to a coordination question. The field uses "harness" with 3 distinct meanings; ours (architecture-as-governance) is the only one addressing concurrent multi-agent coordination structurally.
- 2026-03-10: Health score calibration vs structural improvement — **89% of score improvement (37→73) is calibration artifact, not structural.** Threshold scaling (accretion 20→92.8, hotspot 15→46.4) and bloat% formula change account for +32.2 of +36 points. Baseline values under new formula would score 69.2 — already above the 65 gate. Accretion velocity increasing (370→6,131 lines/week). New bloated files emerging as fast as old ones extracted. Extraction is net-positive on lines (5/6 commits added lines). Pre-commit gate wired today, zero post-gate data. Extends model with score-calibration-as-soft-harness failure mode.
- 2026-03-10: Blog post uncontaminated claim review — **Both published posts ("Soft Harness Doesn't Work," "Building Blind") have mild-to-moderate overclaiming, primarily implicit novelty.** 6 overclaimed, 3 unsupported, 5 fine, 2 fine-but-citable instances across both posts. Main issue: well-established concepts (affordances/Norman, PDCA/Deming, falsificationism/Popper, Conway's Law, nudge theory) described without citation, creating impression of original discovery. Threshold claims (5+ constraints, 10+ inert) stated as general findings from N=7 skills — insufficient for precise inflection points. Recommended: inline acknowledgments ("essentially Conway's Law for LLM agents"), soften thresholds to "in my system," add methodology footnote for 265-trial claim. Posts stay in first-person experiential framing which mitigates risk. Self-critical honesty ("I was wrong") is a strength. The specific context (AI agent orchestration) is genuinely novel even when the conceptual frameworks are not.
- 2026-03-11: Measurement surface design for falsification — **Infrastructure is 80% ready; 3 targeted additions close the gaps.** Audited 40+ event types, 1400 lines of stats code, 4,831 events. Stats already compute gate_decision aggregation (GateDecisionStats) and gate effectiveness correlation (GateEffectivenessStats), but both show zeros because gate_decision events only just shipped (3 events in 7 days). Legacy bypass events show 69.5% fire rate (141/203 spawns) — gates fire on majority of spawns, falsifying "gates are irrelevant." Missing: (1) "allow" gate events for true fire rate, (2) accretion snapshots for velocity trending, (3) harness API endpoint for dashboard. Soft harness compliance NOT measurable from events — requires controlled A/B experiments. Designed 5 implementation components: gate allow events, accretion snapshots, harness API, CLI report, dashboard visualization. Confirms invariant #7 (enforcement without measurement is theological). Extends model with 4 falsification criteria and measurable thresholds.
- 2026-03-11: Retrospective accuracy audit (Phase 3) — **Signal gates have 0% false positive rate across 173 samples.** Audited all blocks/failures for 8 signal gates (build, vet, phase_complete, synthesis, explain_back, verified, triage, accretion_precommit). Zero false positives found. Gates split into correctness gates (11 events, catch real defects) and discipline gates (162 events, measure human process compliance). Discipline gates (explain_back, verified, triage) have 100% eventual-completion rate — they enforce process without blocking correct work. Low-volume gates (build/vet n=3) have wide confidence intervals; Phase 4 prospective tracking needed. self_review (NOISE) confirmed at 79% FP rate (15/19 failures are intentional CLI output or pre-existing code). Extends model with gate accuracy data and correctness/discipline gate taxonomy.
- 2026-03-20: Stronger models accretion rate by model [HE-08] — **Claim remains unconfirmed; instrumentation partially closed, controlled experiment not yet run.** Model field wired into `session.spawned` (working) but `accretion.delta` events lack line-count fields (only 3/425 have model, none with code_added/code_net). Coordination demo (N=160, haiku-only) shows coordination conditions reduce accretion 8-12% vs no-coord baseline (Cohen's d 0.17-0.29, small). Placement most effective (-12.2%). Effect larger on complex tasks. Back-of-envelope: system-level accretion scales ~5x with Opus completion rate (~96%) vs non-Anthropic (~20%). Falsification criterion ("less accretion per agent-session") may be wrong metric — coordination is system-level. Next: fix accretion.delta line-count emission, then run coordination demo with `--model opus` for haiku-vs-opus controlled comparison (N>50 per model).
- 2026-03-20: Compositional correctness gap — **Compliance vs coordination is one instance of a broader cross-domain failure mode class.** LED magnetic letters gate stack (~150 renders): cut-channel LED routing passes all 4 gate layers (parameter, geometry, printability, intent) but produces disconnected channels — valid manifold STL, non-functional enclosure. Sheet metal DFM (SendCutSend): individual operations (cuts, bends, hardware) pass DFM rules but composed assembly interferes. Same structure as daemon.go +892 from 30 correct commits. Named concept "compositional correctness gap": gates validate at component level, failure emerges at composition level. §8 generalized with three-scale evidence table (operation→assembly, geometry→function, agent→system). See `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md`.
- 2026-03-20: SCS AI Part Builder compositional correctness gap (CCG-DFM) — **Confirms compositional correctness gap in production commercial DFM tooling.** SCS's AI Part Builder (powered by Smithy, api.smithy.cc) tested with 4 parts. PEM-near-bend-line test: zero DFM warnings, valid geometry, unmanufacturable part. Integration is iframe with one-way handoff — Smithy validates geometry, SCS validates manufacturing, neither validates manufacturability. **0% recall** on hardware+bend conflicts (regression from SCS's own DFM tools at 46.2%). Gap is at the third-party integration boundary. Three architectural paths identified (none implemented): Smithy internalizes DFM, real-time DFM API, or remove one-way gate. See `.kb/models/smithy-geometry-engine/model.md` for full Smithy model.
