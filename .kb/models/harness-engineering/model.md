# Model: Harness Engineering

**Domain:** Multi-Agent Code Quality Practices
**Last Updated:** 2026-03-10
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
- Fowler/Bockeler: "Harness Engineering" (https://martinfowler.com/articles/exploring-gen-ai/harness-engineering.html) — Verification gap, relocating rigor

---

## Summary (30 seconds)

Harness engineering is a working label for the practice of making wrong paths mechanically impossible for AI agents, rather than instructing agents to choose right paths. It is not a new discipline — it is software architecture, CI/CD enforcement, and tech debt management applied to multi-agent workflows. It operates through two fundamentally different enforcement types: **hard harness** (deterministic, mechanically enforced, cannot be ignored — pre-commit hooks, spawn gates, `go build`, Go package structure, structural tests) and **soft harness** (probabilistic, context-dependent, driftable — skills, CLAUDE.md, knowledge bases, SPAWN_CONTEXT.md). Hard harness matters more because agents under pressure drift from soft instructions — contrastive testing (265 trials, 7 skills) showed behavioral constraints dilute to bare parity at 10+ co-resident items, and stance transfers only as attention primers, not action directives. Accretion is entropy: individually correct agent commits compose into structural degradation when shared infrastructure is missing — daemon.go regrew +892 lines past its pre-extraction baseline in 60 days from 30 correct commits. OpenAI arrived at the same framework from greenfield (designed gates before code); we arrived through pain (retrofit after 3 entropy spirals, 1,625 lost commits). A useful lens for thinking about agent failures: **compliance failure** (agent doesn't follow instructions) vs **coordination failure** (agents each follow instructions correctly but collectively produce problems). In practice these are often mixed — not a clean partition. The claim that stronger models make coordination worse is plausible but uncontrolled (observed in one system, no experiment isolating model capability as the variable).

---

## Core Mechanism

### 1. The Harness Taxonomy (Hard vs Soft)

A harness is everything in the development environment that constrains and guides agent behavior. The term comes from OpenAI's Codex team, who built ~1M lines of code with zero manually-written source over 5 months by investing primarily in the harness. The key reframing: **agent failure is a harness bug, not an agent bug.** When daemon.go independently reimplements workspace scanning for the 5th time, the architecture is missing — not the agent.

Every harness component is either hard or soft:

| Property | Hard Harness | Soft Harness |
|----------|-------------|--------------|
| **Enforcement** | Deterministic — passes or fails | Probabilistic — influences via context |
| **Bypass** | Cannot be ignored without escape hatch | Can be drifted from under pressure |
| **Measurement** | Unnecessary — outcome is binary | Requires contrastive testing to validate |
| **Cost** | Higher upfront (code, infrastructure) | Lower upfront (prose, templates) |
| **Degradation** | Stable unless code is modified | Dilutes at scale (5+ constraints = inert) |

**Cross-language portability (Mar 8 probe — opencode TypeScript fork):** The framework (taxonomy, invariants, failure modes) is language-independent. The gate inventory is language-specific. 5 of 8 harness patterns translate directly to TypeScript (deny rules, control plane lock, hook registration, beads close hook, pre-commit accretion gate). 3 need adaptation: build gate has no TypeScript equivalent (bun typecheck has `any` escape hatch), architecture lint needs different tooling (ts-morph/eslint vs go/ast), hotspot analysis needs generated-file exclusion (4 of 10 top opencode hotspots are *.gen.ts code-generated files). "Unfakeability" is a property of structural coupling (schema↔migration, source↔binary), not compilation specifically — TypeScript's Drizzle migration gate is equally unfakeable despite not being a compiler.

**Existing hard harness (orch-go, verified):**

| Mechanism | What It Prevents | Source | Status |
|-----------|-----------------|--------|--------|
| Pre-commit growth gate | Accretion past 800/600 line thresholds | `pkg/verify/accretion_precommit.go`, `scripts/pre-commit-exec-start-cleanup.sh` | **Shipped** — hard block >1500, warning-only >800 (+30 net) / >600 (+50 net). Wired via `orch precommit accretion` (orch-go-t7te8) |
| Spawn hotspot gate | Feature-impl/debugging on CRITICAL (>1500 line) files | `pkg/spawn/gates/hotspot.go` | Shipped, blocking |
| Build gate (`go build`) | Broken compilation reaching completion | `pkg/verify/check.go` — unfakeable gate (Go-specific; TypeScript has no equivalent — see cross-language probe) | Shipped |
| Completion accretion gate | Agent-caused growth past thresholds | `pkg/verify/accretion.go` (800/1500 thresholds, ±50 delta) | Shipped, **exempts pre-existing bloat** — files already over 1500 get warning, not block (Mar 8 probe) |
| Architecture lint tests | Forbidden lifecycle state packages/imports | `cmd/orch/architecture_lint_test.go` (4 tests) | Shipped, not in CI |
| Spawn rate limiter | Velocity exceeding verification bandwidth | `pkg/spawn/gates/ratelimit.go` | Shipped |
| Spawn concurrency gate | Too many parallel agents | `pkg/spawn/gates/concurrency.go` | Shipped |
| Claude Code deny hooks | Forbidden tool usage at spawn time | `~/.orch/hooks/*.py` (12 scripts, 5 denials) | Shipped, mutable |

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
- Counter-evidence (partially): `spawn_cmd.go` *shrank* -1,755 lines (2,432→677) after `pkg/spawn/backends/` was created (Feb 13) — proving attractors work for initial extraction. However, spawn_cmd.go regrew to 1,160 lines by Mar 8 (+483 in 3 weeks, ~160 lines/week), demonstrating that attractors break the accretion cycle temporarily but re-accretion follows without blocking gates. (Mar 8 probe corrected from -840 claim)
- **Extraction is net-positive on lines** (Mar 10 probe): 5 of 6 recent extraction commits ADDED net lines (+12 to +65 each) through file headers, package declarations, and import duplication. Only the daemon extraction was net-negative (-214) because it also removed dead code. Extraction distributes entropy but doesn't reduce it without concurrent dead code removal.
- **New bloated files emerge as fast as old ones are extracted** (Mar 10 probe): In 2 weeks, 3 files crossed 800 lines (userconfig.go 611→975, +page.svelte 778→1,201, hotspot.go 858→1,050) while 3 were extracted below threshold. Net bloated count barely changed.
- 6 cross-cutting concerns independently reimplemented across 4-9 files (~2,100 lines of duplicated infrastructure): workspace scanning, beads querying, output formatting, project resolution, filtering, ID extraction.

**Two forces drive accretion:**

1. **Feature gravity** — New capabilities land in run functions because that's where the Cobra command lives. `runDaemonLoop()` at 702 lines is the gravitational center for anything daemon-related. There's no friction pushing code elsewhere.

2. **Missing shared infrastructure** — Without `pkg/workspace/`, `pkg/display/`, or shared beads querying, each command file must be self-contained. This isn't agent laziness — the shared packages don't exist. Self-containment is the only option.

**Prevention requires both:**
- **Structural attractors** — packages that pull code toward them (e.g., `pkg/spawn/backends/` caused spawn_cmd.go to shrink)
- **Gates** — enforcement that blocks the old path (pre-commit growth gate catches accretion during authoring)

Attractors without gates → agents still put code in the old location by habit.
Gates without attractors → agents are blocked with nowhere to put code.

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

### 5. The Measurement Layer

Hard and soft harness require fundamentally different measurement:

| Harness Type | Measurement Need | Method |
|---|---|---|
| Hard | None — outcome is deterministic | Build passes or fails |
| Soft (knowledge) | Moderate — verify agents use facts | Single-turn contrastive tests (+5 point lift) |
| Soft (stance) | High — verify attention priming | Multi-scenario contrastive tests (bare 0% → stance 83% on S09, but only cross-source scenarios) |
| Soft (behavioral) | Critical — verify not diluted | Compliance rate (87 constraints → bare parity 5/7 scenarios) |

**The completion verification pipeline through harness lens:**

| Gate Type | Count | Harness Category | Provenance |
|-----------|-------|-------------------|------------|
| Execution-based (Build, Vet, Staticcheck) | 3 | **Hard** | Deterministic — cannot be faked |
| Evidence-based (Phase, Synthesis, Test Evidence, etc.) | 10 | **Structured soft** | Pattern matching — detects theater, not correctness |
| Judgment-based (Explain-back, Behavioral) | 2 | **Human soft** | Human comprehension — valid because human takes responsibility |

As of Mar 2026, 3 of 15 completion gates run code (up from 1). Vet and staticcheck were added as independent hard gates. Further expansion (actually running tests) would continue increasing hard harness surface.

**The measurement design principle:** Every soft harness component should be contrastively validated before deployment. If adding content to a skill doesn't measurably change behavior, it's dead weight crowding out effective content.

### 6. Implementation Layers

Each layer builds on the previous. Lower layers are more immediately actionable:

| Layer | What | Status | Mechanism |
|-------|------|--------|-----------|
| **0** | Pre-commit growth gate | **Shipped** (orch-go-hhq9a, corrected orch-go-34vn0) | `orch precommit accretion`, warning-only, >800→≥30 net lines, >600→≥50 net lines |
| **1** | Structural tests for package boundaries | **Partially shipped** | `architecture_lint_test.go` (4 tests for lifecycle state), not in CI |
| **2** | Duplication detector | Not started | Static analysis finding pattern similarity across files |
| **3** | Periodic entropy agent | Not started | Background agent reviewing growth trends weekly |
| **4** | Gates that generate gates | Aspirational | Entropy agent drafts structural tests for recurring patterns |

**Layer 0 status (Mar 10):** Fully shipped. The pre-commit hook calls `orch precommit accretion` which runs `CheckStagedAccretion`. Hard block at >1500 lines. Warning-only at >800 lines (+30 net delta) and >600 lines (+50 net delta). Override: `FORCE_ACCRETION=1 git commit ...`. The completion accretion gate (`pkg/verify/accretion.go`) IS also active but exempts pre-existing bloated files. **However, pre-commit gate was only wired on Mar 10 — zero post-gate velocity data exists.** Weekly cmd/orch/ growth was accelerating prior to wiring: 370 → 1,473 → 6,264 → 6,131 lines/week (Feb 10–Mar 10).

**Layer 1 needs extension.** Current structural tests enforce only the no-lifecycle-state constraint (from two-lane architecture decision). Missing: function size limits for cmd/orch/, package boundary enforcement, cross-cutting duplication detection. These 4 tests also aren't in CI — they require manual `go test` execution.

**Layer 2 would convert the "agent failure = harness bug" principle into automation.** When function similarity > threshold across files, create a beads issue: "shared infrastructure missing for workspace scanning." The detector is hard harness (deterministic); the response is initially soft (recommendation).

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

### 8. Two Failure Modes: Compliance vs Coordination (a lens, not a partition)

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

**Which gates are which:**

| Gate | Type | Trajectory |
|------|------|-----------|
| Pre-commit growth gate | Compliance | Simplifies — smarter agents self-limit |
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

7. **Stronger models may need more coordination gates, not fewer.** Hypothesis: compliance gates simplify with model capability, but coordination gates grow in importance as agents get faster. Observed in one system (faster agents produced more code per session). Not experimentally controlled — this is a plausible claim, not a validated one.

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

**The broader pattern:** 12 files in cmd/orch/ exceed 800 lines (total: ~14,000 lines). 6 exceed 1,000. The exemption means the completion gate is primarily useful for files approaching the threshold, not files that have already passed it. This is the wrong coverage profile — the already-bloated files are where accretion pressure is strongest (feature gravity).

### 7. Measurement Artifacts in Soft Harness

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

**2026-03-08 (evening):** Compliance vs coordination failure mode distinction crystallized. daemon.go +892 was coordination failure (30 agents each correct, collectively incoherent), not compliance failure. Stronger models fix compliance but worsen coordination — faster agents accrete more confidently. Harness engineering reframed as permanent discipline (coordination infrastructure) rather than transitional (training wheels). Publication plan created with 4 phases: deepen model → cross-language evidence → publication draft → portable tooling.

---

## References

**Investigations:**
- `.kb/investigations/2026-03-07-inv-analyze-accretion-pattern-orch-go.md` — Primary accretion evidence
- `.kb/investigations/2026-03-07-inv-add-pre-commit-growth-gate.md` — Layer 0 implementation
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` — Four-layer enforcement design
- `.kb/investigations/2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md` — Cross-investigation synthesis
- `.kb/investigations/2026-02-14-inv-add-claude-md-accretion-boundaries.md` — CLAUDE.md as soft harness: progressive disclosure pattern (20→4 lines)
- `.kb/investigations/2026-02-14-inv-fix-claude-md-remove-deleted.md` — CLAUDE.md documentation drift: stale refs to deleted pkg/registry/, duplicated sections

**Thread:**
- `.kb/threads/2026-03-07-harness-engineering-structural-enforcement-agent.md`

**Related Models:**
- `.kb/models/architectural-enforcement/model.md` — Hard harness mechanisms
- `.kb/models/entropy-spiral/model.md` — Soft harness failure at scale
- `.kb/models/skill-content-transfer/model.md` — Three-type vocabulary, contrastive measurement
- `.kb/models/extract-patterns/model.md` — Extraction mechanics
- `.kb/models/completion-verification/model.md` — 14-gate pipeline, gate type taxonomy

**Decisions:**
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md`
- `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md`

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
- `pkg/verify/precommit.go` — Pre-commit growth gate (Layer 0)
- `cmd/orch/architecture_lint_test.go` — Structural tests (Layer 1, partial)

## Probes

- 2026-03-07: Completion verification through harness lens — Confirms hard/soft taxonomy: 1 of 14 gates is execution-based (hard), 11 evidence-based (structured soft), 2 judgment-based (human soft). Build gate is the only unfakeable gate.
- 2026-03-08: 30-day accretion trajectory measurement — **Gates have NOT bent the line count curve.** daemon.go hit 1,559 (CRITICAL) despite all deployed gates. Completion accretion gate exempts pre-existing bloat. Pre-commit accretion gate exists in code but is NOT wired into the hook. spawn_cmd.go shrank -1,755 (not -840 as claimed) then regrew +483 in 3 weeks. Total cmd/orch/: 47,605 lines across 125 files, 12 files >800 lines. Fix:feat ratio spike (1.21) was transient, reverted to 0.36. Confirms invariants #2 and #4. Extends model with gate exemption failure mode and dead code enforcement gap.
- 2026-03-08: Cross-language harness portability (Go → TypeScript) — **Framework is language-independent, gates are not.** 5/8 harness patterns translate directly to TypeScript. Build gate (`go build`) has no TypeScript equivalent — `bun typecheck` has `any` escape hatch and is pre-push only. "Unfakeability" is structural coupling (schema↔migration, source↔binary), not compilation. Generated code creates false positives: 4/10 top opencode hotspots are *.gen.ts files. TypeScript has own domain-specific hard harness (Drizzle migration gate) that Go lacks. Extends model with cross-language portability analysis and generated-code blind spot.
- 2026-03-08: Publication draft model synthesis — **All core claims survived synthesis into publication format.** Compliance vs coordination failure distinction is the strongest novel claim for external audiences. Three claims need more evidence before strong external assertion: "stronger models need more coordination gates" (no controlled experiment), soft harness budget curve (shape unknown), cross-language portability (dry-run only, not 30-day operation). Honest negative evidence (gates haven't bent the curve) strengthens credibility. spawn_cmd.go correction (-1,755, not -840) confirmed.
- 2026-03-10: Publication polish — related work positioning — **Compliance/coordination distinction is novel in published literature.** Positioned against 4 sources: OpenAI (harness = environment setup), Anthropic (harness = session orchestration), Fowler (harness = verification), MAST/Cemri et al. (observes coordination failures but prescribes model solutions). MAST's FC1/FC2/FC3 maps to compliance/coordination/verification but they don't recognize opposite model-improvement trajectories. "Deeper social reasoning" is a compliance answer to a coordination question. The field uses "harness" with 3 distinct meanings; ours (architecture-as-governance) is the only one addressing concurrent multi-agent coordination structurally.
- 2026-03-10: Health score calibration vs structural improvement — **89% of score improvement (37→73) is calibration artifact, not structural.** Threshold scaling (accretion 20→92.8, hotspot 15→46.4) and bloat% formula change account for +32.2 of +36 points. Baseline values under new formula would score 69.2 — already above the 65 gate. Accretion velocity increasing (370→6,131 lines/week). New bloated files emerging as fast as old ones extracted. Extraction is net-positive on lines (5/6 commits added lines). Pre-commit gate wired today, zero post-gate data. Extends model with score-calibration-as-soft-harness failure mode.
- 2026-03-10: Blog post uncontaminated claim review — **Both published posts ("Soft Harness Doesn't Work," "Building Blind") have mild-to-moderate overclaiming, primarily implicit novelty.** 6 overclaimed, 3 unsupported, 5 fine, 2 fine-but-citable instances across both posts. Main issue: well-established concepts (affordances/Norman, PDCA/Deming, falsificationism/Popper, Conway's Law, nudge theory) described without citation, creating impression of original discovery. Threshold claims (5+ constraints, 10+ inert) stated as general findings from N=7 skills — insufficient for precise inflection points. Recommended: inline acknowledgments ("essentially Conway's Law for LLM agents"), soften thresholds to "in my system," add methodology footnote for 265-trial claim. Posts stay in first-person experiential framing which mitigates risk. Self-critical honesty ("I was wrong") is a strength. The specific context (AI agent orchestration) is genuinely novel even when the conceptual frameworks are not.
