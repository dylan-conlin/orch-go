# Harness Engineering: Structural Governance for Multi-Agent Codebases

*What happens when 50 AI agents/day commit to the same codebase — and why better models don't fix it.*

---

## The Problem: Accretion

Every agent does the right thing. The codebase degrades anyway.

We run 50+ autonomous AI agents per day on a single Go codebase (orch-go, ~47,600 lines, 125 files). Each agent gets a task, writes correct code, passes tests, and commits. Individually, every commit is reasonable. Collectively, the codebase is falling apart.

Our clearest example: `daemon.go` grew from 667 lines to 1,559 lines in 60 days. Not from one bad commit — from 30 individually correct ones. Each agent added a locally reasonable capability: stuck detection, health checks, auto-complete, agreement verification, phase timeouts, orphan recovery. No single commit was wrong. The aggregate was structural degradation.

We call this **accretion** — the tendency of multi-agent codebases toward structural degradation. It's not a quality problem. It's a coordination problem. And it doesn't go away with better agents. The 30 agents that grew daemon.go weren't struggling. They were excelling — each one individually. The problem is that individual excellence doesn't compose into collective coherence without structural coordination. Model improvement doesn't fix this, because coordination failure is an emergent property of the system, not a deficiency of its parts. (We observed that daemon.go growth accelerated as agent capabilities improved, but we haven't isolated model capability as the variable — faster iteration cycles, more features in flight, and team familiarity are confounds.)

### The anatomy of accretion

Two forces drive it:

**Feature gravity.** New code lands in existing run functions because that's where the Cobra command lives. `runDaemonLoop()` became a 702-line gravitational center for anything daemon-related. There's no friction pushing code elsewhere — the function exists, it works, and the agent's task is urgent.

**Missing shared infrastructure.** Without shared packages for common operations, each agent must be self-contained. We found 6 cross-cutting concerns independently reimplemented across 4–9 files each: workspace scanning, beads querying, output formatting, project resolution, filtering, ID extraction. That's roughly 2,100 lines of duplicated infrastructure. The agents weren't lazy — the shared packages didn't exist.

### The scale of the problem

Three entropy spirals over 12 weeks. 1,625 commits lost in a single crash. Here's the 12-week trajectory:

| Date | Files | Total Lines | Files >800 lines | % Bloated |
|------|-------|-------------|-------------------|-----------|
| Dec 22 | 10 | 7,315 | 1 | 10% |
| Jan 5 | 46 | 20,931 | 5 | 11% |
| Jan 19 | 62 | 34,607 | 13 | 21% |
| Feb 16 | 70 | 34,977 | 13 | 19% |
| Mar 8 | 125 | 47,605 | 12 | 10% |

The percentage of bloated files dropped from 21% to 10% — but this is an artifact of file proliferation, not file shrinkage. The denominator grew from 62 to 125 (extraction creates new files). The 12 files over 800 lines are still there, still growing:

| File | Lines |
|------|-------|
| daemon.go | 1,559 |
| status_cmd.go | 1,361 |
| review.go | 1,353 |
| stats_cmd.go | 1,351 |
| clean_cmd.go | 1,270 |
| spawn_cmd.go | 1,160 |

Six files over 1,000 lines. The problem is spreading, not concentrating.

### Why this is a coordination failure, not a compliance failure

This distinction matters. Two failure modes exist in multi-agent systems, and they respond oppositely to model improvements:

| | Compliance Failure | Coordination Failure |
|---|---|---|
| **What breaks** | Agent doesn't follow instructions | Agents each follow instructions but collectively produce entropy |
| **Example** | Agent ignores the 1,500-line convention | 30 agents each add correct code; daemon.go grows +892 lines |
| **Fixed by better models?** | Yes | No — model improvement doesn't help coordination (and faster agents may accelerate accretion, though we haven't isolated this variable) |

The daemon.go evidence is coordination failure. Each of the 30 commits followed instructions. Each was locally rational. Each passed review. The problem was the absence of structural coordination — no shared packages, no deduplication detection, no cross-agent awareness that workspace scanning was already implemented four times.

The analogy: a company of 30 brilliant engineers with no architecture review still produces spaghetti — possibly faster than 30 mediocre engineers, because each builds more in less time.

**Implication:** If governance were only about compliance, it would become easier as models improve. But coordination is an emergent property of multi-agent systems. It doesn't resolve with individual capability. This suggests harness engineering may be a permanent discipline, not a transitional one — though the evidence base is one system over 12 weeks.

---

## The Evidence: What We Measured

### Experiment 1: daemon.go — accretion without gates

daemon.go is our most instructive failure. Growth trajectory:

| Period | Lines Added | Velocity |
|--------|-------------|----------|
| Dec 22 → Jan 19 | +261 | ~65/week |
| Jan 19 → Feb 16 | +215 | ~54/week |
| Feb 23 → Mar 2 | +176 | 176/week |
| Mar 2 → Mar 8 | +345 | 345/week |

The velocity is *accelerating*. No inflection point is visible — the curve bends upward, not downward, after we deployed our first gates. We'll explain why below.

### Experiment 2: spawn_cmd.go — attractor works, then fails

`spawn_cmd.go` is the counter-case. When we created `pkg/spawn/backends/` on Feb 13, it acted as a **structural attractor** — a destination package that pulls code toward it by naming and import convention. spawn_cmd.go shrank from 2,432 lines to 677 lines. A -1,755 line extraction. The attractor worked.

Then it regrew. By March 8, spawn_cmd.go was back to 1,160 lines — +483 lines in 3 weeks, roughly 160 lines/week. At that velocity, it would re-cross 1,500 lines by late March.

**The lesson:** Attractors break the accretion cycle for the initial extraction. They don't prevent re-accretion. The Cobra command definition still lives in spawn_cmd.go, still creating feature gravity. New code lands there because the function exists and the task is urgent.

Extraction without routing is a pump. You can cool a room without insulation — it just heats right back up.

### Experiment 3: soft instructions fail under pressure

We ran 265 contrastive trials across 7 agent skills to measure how well written instructions change agent behavior.[^methodology] We tested three types of content:

| Content Type | Transfer Mechanism | Measured Effect |
|---|---|---|
| **Knowledge** (facts, templates) | Direct — agent reads and applies | +5 point lift |
| **Stance** (attention primers) | Indirect — shifts what agents notice | +2 to +7 on cross-source scenarios |
| **Behavioral** (MUST/NEVER rules) | Unreliable — dilutes at scale | Inert at 10+ co-resident rules |

The critical finding: behavioral constraints — the things you'd most want to enforce — are the content type most vulnerable to dilution. Our orchestrator skill had 87 behavioral constraints. We classified approximately 83 as non-functional based on observed compliance rates in production (constraints that were violated in >50% of sessions where they were relevant). They competed with the system prompt at a 17:1 signal disadvantage.

A convention in documentation without mechanical enforcement is a suggestion with a half-life proportional to context window pressure. daemon.go grew past the stated 1,500-line convention while that convention existed in CLAUDE.md. The agents read it. They understood it. They added 200 lines to an 1,800-line file anyway because the task was urgent and no gate blocked them.

### The honest assessment: gates haven't bent the curve yet

As of March 8, 2026, total lines in `cmd/orch/` grew from 34,977 to 47,605 in 3 weeks (+12,628, ~4,200/week). The aggregate growth rate hasn't slowed since gate deployment.

Why? Two gaps in our deployed gates:

1. **The completion gate exempts pre-existing bloat.** Files already over 1,500 lines receive warnings, not blocks. daemon.go at 1,559 lines will never be blocked by the completion gate. Every agent adding 50+ lines gets a non-blocking warning. The gate structurally cannot enforce on the files that need enforcement most. This is a ratchet — once bloated, always exempted.

2. **Gates without attractors leave agents stuck.** When a gate warns that a file is too large but no destination package exists for extracted code, the agent sees the warning but has nowhere to go. We've since created structural attractors (`pkg/workspace/`, `pkg/display/`, `pkg/beadsutil/`) and wired the pre-commit gate — the 30-day forward measurement will show whether this combination works.

The model is correct in theory. The gates as deployed are too late, too narrow, and self-exempting. Gates haven't been given a fair test yet because the blocking gates for pre-existing bloat literally don't exist.

---

## The Framework: Harness Engineering

We use the term "harness" deliberately. It comes from OpenAI's Codex team, who built roughly 1 million lines of code with zero manually-written source over 5 months by investing primarily in the test and constraint infrastructure. Their reframing: **agent failure is a harness bug, not an agent bug.** When daemon.go independently reimplements workspace scanning for the fifth time, the architecture is missing — not the agent.

Martin Fowler and Birgitta Bockeler added a critical observation: this approach requires "constraining the solution space" — the opposite of what most expect from AI coding. Rigor doesn't disappear when you stop writing code manually; it migrates to environment design and constraint specification.

We arrived at the same insight from the opposite direction. OpenAI designed gates before code (greenfield advantage). We discovered the need through pain — 3 entropy spirals, 1,625 lost commits, and daemon.go growing +892 lines past its pre-extraction baseline.

### The hard/soft taxonomy

Every component of the development environment that constrains agent behavior is a harness. The single most useful distinction: **hard** vs **soft**.

| Property | Hard Harness | Soft Harness |
|----------|-------------|--------------|
| Enforcement | Deterministic — passes or fails | Probabilistic — influences via context |
| Bypass | Cannot be ignored without escape hatch | Drifts under pressure |
| Measurement | Unnecessary — outcome is binary | Requires contrastive testing |
| Cost | Higher upfront (code, infrastructure) | Lower upfront (prose, templates) |
| Degradation | Stable unless code is modified | Dilutes at scale (10+ constraints = inert) |

**Hard harness** examples: pre-commit hooks, compilation (`go build`), spawn gates that refuse to create agents for degraded files, structural tests enforcing package boundaries, Claude Code deny rules preventing agents from editing the files that define their own constraints.

**Soft harness** examples: CLAUDE.md conventions, skill documents, knowledge base guides, advisory context injected at spawn time.

Hard harness doesn't need measurement — a build passes or fails. Soft harness needs contrastive testing to know whether it works at all. The default assumption for soft harness should be "probably doesn't work" until proven otherwise.

**The design error we made (and that you'll make too):** We put behavioral constraint content (MUST/NEVER prohibitions) inside context-type containers (skill documents). Constraints dilute in context. The correct mapping:

- **Knowledge** (facts, routing tables) → keep in skills/context — resilient, no dilution limit
- **Stance** (attention primers) → keep in skills/context — only "look for X," not "do X"
- **Behavioral** (MUST/NEVER) → move to hard harness — every prohibition should be a hook, gate, or structural test

### Attractors + gates: both required

Neither attractors nor gates work alone.

**Attractors without gates:** `pkg/daemon/` exists as an extracted package (896 lines of daemon logic), but new features still land in `cmd/orch/daemon.go` because the Cobra command lives there. The attractor exists. No gate prevents the old path.

**Gates without attractors:** The pre-commit gate warns that status_cmd.go is too large, but no destination package exists for the extracted code. The agent sees the warning but has nowhere to go.

**Both together:** `pkg/spawn/backends/` (attractor) plus completion accretion gate (enforcement) caused spawn_cmd.go to shrink -1,755 lines. The attractor provides the low-energy destination state; the gate provides the activation energy barrier preventing the old path.

A loose metaphor helps here (it's a metaphor, not a physics claim): attractors are destinations that code naturally flows toward. Gates are barriers preventing code from accumulating in high-complexity files. Extraction without both is cooling a room without insulation.

### Five invariants

After 12 weeks of running 50+ agents/day, these are the invariants we'd bet on:

1. **Hard harness for enforcement, soft harness for orientation.** Behavioral prohibitions in skill documents produce the worst of both — unreliable enforcement that dilutes reliable knowledge transfer.

2. **Every convention without a gate will eventually be violated.** A convention in documentation without infrastructure enforcement is a suggestion with a half-life proportional to context window pressure.

3. **Agent failure is harness failure.** The first question for any wrong agent outcome is "what's missing from the harness?" not "what's wrong with the agent?" The harness is the modifiable variable.

4. **Extraction without routing is a pump.** Moving code out of a file without creating an attractor results in re-accretion. The gravitational center must be relocated, not just temporarily emptied.

5. **Prevention > Detection > Rejection.** Each layer further from authoring has higher cost. Pre-commit gate (prevention) < spawn gate (early detection) < completion gate (late detection + wasted agent work).

### Five enforcement layers

Each layer builds on the previous. Lower layers are more immediately actionable:

| Layer | What | Status (ours) | Mechanism |
|-------|------|--------|-----------|
| **0: Pre-commit** | Growth gate at authoring time | Shipped — blocking at >1,500 lines | `CheckStagedAccretion` blocks commits adding to files past threshold |
| **1: Structural tests** | Package boundary enforcement | Shipped — function size lint, package boundaries, 4 architecture tests | Tests asserting architectural invariants (no forbidden imports, function size limits) |
| **2: Duplication detector** | Cross-agent redundancy detection | Shipped — AST fingerprinting + auto-issue creation | `pkg/dupdetect/` finds function similarity across files, creates beads issues |
| **3: Entropy agent** | Periodic system-level health monitoring | Shipped — `orch entropy` + weekly launchd scheduling | Analyzes fix:feat ratio, velocity, bloat, override trends; generates recommendations |
| **4: Self-extending gates** | Gates that generate gates | Aspirational | Entropy agent drafts structural tests for recurring patterns |

The trajectory here is important: Layers 0–1 are **compliance gates** — they may simplify with model improvement as smarter agents self-limit. Layers 2–4 are **coordination gates** — they don't simplify with model improvement because coordination is a system property, not an agent capability. A more capable agent may accrete more code per session, though we haven't measured this directly.

This is why harness engineering is permanent infrastructure. The compliance gates may simplify. The coordination gates are the endgame.

We tested the framework against one TypeScript codebase and found that 5 of 8 harness patterns translated directly. The 3 that didn't (build gate, architecture lint, hotspot analysis) are language-specific in implementation but not in concept. Two languages is not enough to claim language-independence — it's enough to say the patterns aren't Go-specific. The deeper insight: "unfakeability" is a property of structural coupling (schema↔migration, source↔binary), not compilation specifically. Each ecosystem has its own structurally coupled hard gates.

---

## Related Work: Three Meanings of "Harness"

The term "harness" has gained currency in AI engineering, but different groups mean fundamentally different things by it. Untangling these usages clarifies what our work adds.

**Harness as environment setup.** OpenAI's Codex team used "harness" to describe the test infrastructure and constraint scaffolding that enabled their zero-manual-code approach (~1M lines, 1,500 PRs, 3–7 engineers, 5 months). Their reframing — "agent failure is a harness bug, not an agent bug" — is foundational to our work. Anthropic's "Effective Harnesses for Long-Running Agents" extends this to multi-session continuity: progress files, mandatory commits, structured feature lists, incremental completion. Both are primarily single-agent approaches. They solve the problem of making *one agent* effective across sessions. They don't address what happens when 50 agents are effective simultaneously on the same codebase.

**Harness as verification.** Fowler and Böckeler's analysis of the OpenAI approach identifies what they call the "verification gap" — architectural quality without behavioral verification — and the concept of "relocating rigor." Rigor doesn't disappear when you stop writing code manually; it migrates to environment design and constraint specification. This framing is precisely correct and influenced our work directly. But it stops at the single-agent boundary. "Does the harness verify the right things?" is a different question from "how do multiple agents coordinate *through* the harness?"

**Harness as governance.** Our usage is distinct: architecture-as-governance for concurrent multi-agent systems. Package structure as a routing table for agentic contributions. Import boundaries as jurisdiction lines. Structural tests as constitutional constraints. The harness isn't just the environment an agent runs in — it's the coordination infrastructure that prevents *N* individually-correct agents from producing collectively-incoherent output.

### The coordination gap in the literature

The closest academic work to our coordination concern is Cemri et al.'s MAST taxonomy, which analyzed 1,600+ traces of multi-agent LLM systems and identified 14 failure modes across three categories. Their FC2 category — inter-agent misalignment — accounts for roughly 32% of all failures, including conversation resets, task derailment, ignored inputs, and reasoning-action mismatches. This maps directly to what we call coordination failure.

But MAST frames the solution as requiring "deeper social reasoning abilities" from agents — essentially, better models. They don't consider the possibility that the coordination failures they observe are *architectural*, not cognitive. Their own finding — "a well-designed MAS can result in performance gain when using the same underlying model" — supports the architectural interpretation, but they don't develop it.

What no existing work distinguishes: compliance failure (agent doesn't follow instructions — fixed by better models) versus coordination failure (agents each follow instructions but collectively produce entropy — not helped by better models). MAST's FC1 (system design, ~44% of failures) maps roughly to compliance. Their FC2 (~32%) maps to coordination. Their FC3 (task verification, ~24%) maps to Fowler's verification gap. But MAST taxonomizes by symptom, not by response to model improvement. The result: they prescribe model-level solutions for what is actually an architectural problem. "Deeper social reasoning" is a compliance answer to a coordination question.

This distinction — that compliance and coordination failures respond differently to model improvement — is the core framing of our work. The distinction itself is well-known in other fields: mechanism design, organizational theory, and distributed systems all separate individual compliance from collective coordination. It seems underappreciated in the AI coding agent community, where most harness work focuses on making individual agents more effective. What we're adding is the application to AI coding agents and the observation that coordination failures don't resolve with model improvement.

---

## Getting Started

If you're running 10+ AI agents on a shared codebase, here's the minimum viable harness — Day 1 governance that prevents the first entropy spiral.

### Day 0: Structural scaffold (~30 min)

Create the directory structure: workspace storage, knowledge base, issue tracking, project charter (CLAUDE.md). If you use our tooling, `orch init` does this. If not, create equivalent directories for agent session storage and knowledge externalization.

### Day 1: Behavioral enforcement (~2–4h)

These seven items are the difference between "has directories" and "safe for autonomous agents":

1. **Deny rules.** Prevent agents from modifying the files that define their own constraints. This is the recursive vulnerability — without it, your first agent can edit away its own guardrails.

2. **Gate: agents can't self-close issues.** Agents report completion; a verification pipeline closes issues. Without this gate, agents bypass verification by closing their own work.

3. **Gate: no `git add -A`.** Agents must stage files by name. Without this, agents stage secrets, unrelated changes, and build artifacts.

4. **Event emission hook.** When issues are closed, emit completion events. Without this, agent lifecycle is invisible to monitoring.

5. **Pre-commit growth gate.** The one hard gate — warns when files grow past size thresholds. This is the canary. It won't stop accretion alone, but it makes accretion visible at authoring time instead of review time.

6. **Control plane lock.** Make enforcement files immutable at the OS level. Mutable hard harness is soft harness with extra steps.

7. **Governance sections in project charter.** Document authority boundaries (what agents can decide vs. escalate), accretion limits, and key conventions. This is soft harness — it won't prevent violations alone, but it provides the orientation that makes hard harness legible.

### Week 1: Verification & observability (~4–8h)

Run one full agent lifecycle with human observation. Watch gates fire against real tool use. Confirm deny rules block control plane edits. Confirm the pre-commit gate warns on growth. This behavioral verification is critical — code that exists but has never been observed working is enforcement theater.

---

## The Honest Assessment

### What's working

**Spawn hotspot gate** blocks agents from spawning on degraded files (>1,500 lines), routing through architect review first. Bypass requires proof of prior review — due process, not bureaucracy.

**Control plane immutability** via OS-level file locking. Agents cannot modify the files that define their own constraints. Before this, three entropy spirals occurred with mutable infrastructure.

**The hard/soft taxonomy as a design tool.** Once you can classify each harness component, the conversation changes from "should we add this rule?" to "is this a hard or soft component, and are we putting it in the right container?"

### What isn't

**Gates haven't bent the accretion curve.** Total lines, individual file sizes, and weekly velocity all show no deceleration after gate deployment. The deployed gates were too late (completion, not pre-commit), too narrow (exempt files already over threshold), and self-exempting. We've since wired blocking pre-commit gates and created structural attractors — the 30-day forward measurement will show whether the combination works.

**Completion gate exempts pre-existing bloat.** Files already over 1,500 lines get warnings, not blocks. That's exactly backwards — the already-bloated files are where accretion pressure is strongest.

---

## Open Questions

These are the things we don't know yet. We're listing them because honest gaps are more useful than confident claims without evidence.

### Governance health metric

We've designed but not yet validated a composite 0–100 score for harness effectiveness: escape hatch frequency, accretion velocity, gate firing rate, structural test coverage. We don't know yet whether this collapses into a single useful number or whether the components should be tracked independently.

### The mutable control plane problem

All our defenses live inside the system agents can modify. OS-level file locking provides a temporal buffer, but it's not architecturally immutable — any process with sufficient privileges can unlock it. True immutability would require infrastructure that's architecturally unreachable by agents. We don't know what that looks like yet. It may require enforcement at a level above the codebase entirely — CI/CD pipelines, pre-merge hooks on remote repositories, or organizational infrastructure.

---

## Where We Are

We're 12 weeks and 3 entropy spirals into this. The framework is clear: hard harness for enforcement, soft harness for orientation, attractors and gates together, coordination gates as permanent infrastructure. All 5 enforcement layers are now shipped — from pre-commit blocking through AST duplication detection to weekly entropy analysis. The evidence is real but incomplete — the full gate stack has only been deployed for days, not weeks. The 30-day forward measurement is the real test.

The 30-day forward measurement starts from March 8, 2026. The baseline: daemon.go at 1,559 lines, 47,605 total lines across 125 files, 12 files over 800 lines, ~4,200 lines/week growth velocity. What would constitute "bending the curve": daemon.go stabilizes or decreases below 1,500; files over 800 drops below 10; weekly velocity drops below 2,000.

We're publishing now — before the 30-day results — because the framework, the taxonomy, and the failure modes are already useful to teams running into the same problems. The evidence supports the diagnosis (accretion as coordination failure) even if the treatment (gates + attractors) is still proving itself.

The field is converging on "harness" as a concept, but the coordination problem — what happens when many individually effective agents share a substrate — is underexplored in the AI engineering literature we've seen. Most harness work assumes the challenge is making individual agents effective — better prompts, better tools, better verification. Our experience says the harder problem is what happens *after* you make them effective. Fifty capable agents with no coordination infrastructure produce entropy faster than five mediocre agents ever could.

The deepest insight from this work: in multi-agent systems, codebase architecture is governance. Package structure is a routing table for agentic contributions. Import boundaries are jurisdiction lines. Structural tests are constitutional constraints. And every convention without a gate will eventually be violated.

Accretion isn't unique to code. We've observed similar dynamics — attractors, gates, entropy spirals, the compliance/coordination split — in knowledge systems where amnesiac agents contribute to a shared understanding rather than a shared codebase. The patterns look similar in both substrates we've measured (Go codebase and knowledge base), though two data points are suggestive, not conclusive. That's the subject of our next piece.

---

*This is based on 12 weeks of operating orch-go: ~47,600 lines of Go, 125 files, 50+ autonomous AI agent sessions/day, 3 entropy spirals, 1,625 lost commits, 265 contrastive trials across 7 skills, and cross-language validation against a TypeScript fork. The harness engineering model, evidence probes, and minimum viable harness checklist are open at [repo link].*

[^methodology]: Each contrastive trial compared agent output on the same task scenario with and without the skill document loaded. The control condition used a base system prompt with no skill-specific instructions; the treatment condition injected the full skill document. Output was scored by an automated rubric calibrated against human ratings on 3 of 4 validation scenarios (Spearman rho > 0.74). All 7 skills were from our orchestrator system; we did not test skills from other systems. The 265 number reflects total scenario × skill combinations, not independent experiments — many share the same underlying model and task distribution.

### References

- OpenAI, "Harness Engineering" (2025). https://openai.com/index/harness-engineering/
- Fowler, M. & Böckeler, B., "Harness Engineering" (2025). https://martinfowler.com/articles/exploring-gen-ai/harness-engineering.html
- Anthropic, "Effective Harnesses for Long-Running Agents" (2026). https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents
- Cemri, M. et al., "Why Do Multi-Agent LLM Systems Fail?" (2025). arXiv:2503.13657
