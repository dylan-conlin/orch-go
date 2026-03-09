# Harness Engineering: Structural Governance for Multi-Agent Codebases

*What happens when 50 AI agents/day commit to the same codebase — and what we built to survive it.*

---

## The Problem: Accretion

Every agent does the right thing. The codebase degrades anyway.

We run 50+ autonomous AI agents per day on a single Go codebase (orch-go, ~47,600 lines, 125 files). Each agent gets a task, writes correct code, passes tests, and commits. Individually, every commit is reasonable. Collectively, the codebase is falling apart.

Our clearest example: `daemon.go` grew from 667 lines to 1,559 lines in 60 days. Not from one bad commit — from 30 individually correct ones. Each agent added a locally reasonable capability: stuck detection, health checks, auto-complete, agreement verification, phase timeouts, orphan recovery. No single commit was wrong. The aggregate was structural degradation.

We call this **accretion** — the thermodynamic tendency of multi-agent codebases toward entropy. It's not a quality problem. It's a coordination problem. And it doesn't go away with better agents. In fact, it gets worse.

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
| **Fixed by better models?** | Yes | No — made *worse* by faster, more confident agents |

The daemon.go evidence is coordination failure. Each of the 30 commits followed instructions. Each was locally rational. Each passed review. The problem was the absence of structural coordination — no shared packages, no deduplication detection, no cross-agent awareness that workspace scanning was already implemented four times.

The analogy: a company of 30 brilliant engineers with no architecture review still produces spaghetti — possibly faster than 30 mediocre engineers, because each builds more in less time.

**Implication:** If governance were only about compliance, it would be obsolete when models get good enough. But coordination is an emergent property of multi-agent systems. It doesn't resolve with individual capability. This makes harness engineering a permanent discipline, not a transitional one.

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

We ran 265 contrastive trials across 7 agent skills to measure how well written instructions change agent behavior. We tested three types of content:

| Content Type | Transfer Mechanism | Measured Effect |
|---|---|---|
| **Knowledge** (facts, templates) | Direct — agent reads and applies | +5 point lift |
| **Stance** (attention primers) | Indirect — shifts what agents notice | +2 to +7 on cross-source scenarios |
| **Behavioral** (MUST/NEVER rules) | Unreliable — dilutes at scale | Inert at 10+ co-resident rules |

The critical finding: behavioral constraints — the things you'd most want to enforce — are the content type most vulnerable to dilution. Our orchestrator skill had 87 behavioral constraints. Approximately 83 were non-functional. They competed with the system prompt at a 17:1 signal disadvantage.

A convention in documentation without mechanical enforcement is a suggestion with a half-life proportional to context window pressure. daemon.go grew past the stated 1,500-line convention while that convention existed in CLAUDE.md. The agents read it. They understood it. They added 200 lines to an 1,800-line file anyway because the task was urgent and no gate blocked them.

### Experiment 4: fix:feat ratio is transient

We tracked the ratio of fix commits to feature commits as a proxy for system health. During the week we deployed most of our gates (Feb 23), the ratio spiked to 1.21 — more fixes than features, expected during infrastructure work. The following week it reverted to 0.36.

There is no sustained shift. Gate deployment didn't change the steady-state ratio. This tells us gates are enforcement mechanisms, not culture changers. They prevent specific violations; they don't make agents more architecturally aware.

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

The thermodynamic analogy works: attractors are low-energy states that code naturally flows toward. Gates are activation energy barriers preventing code from accumulating in high-entropy states. Extraction without both is cooling a room without insulation.

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

The trajectory here is important: Layers 0–1 are **compliance gates** — they simplify with model improvement as smarter agents self-limit. Layers 2–4 are **coordination gates** — they become more important as agents get faster and more autonomous. A more capable agent accretes more code per session with higher confidence.

This is why harness engineering is permanent infrastructure. The compliance gates may simplify. The coordination gates are the endgame.

### Cross-language portability

We tested the framework against a TypeScript codebase (our OpenCode fork, ~48 bloated files, 155 hotspots) to see what translates.

**The framework is language-independent. The gates are not.**

5 of 8 harness patterns translate directly to TypeScript with zero adaptation: deny rules, control plane lock, Claude Code hook registration, beads close hook, pre-commit accretion gate. These operate at the OS level, tool level, or git level — none examine language-specific constructs.

3 patterns need adaptation:

- **Build gate:** Go's `go build` is unfakeable — the binary won't exist if it fails. TypeScript's `bun typecheck` has escape hatches (`any`, `@ts-ignore`) and runs at pre-push, not pre-commit. No TypeScript mechanism has equivalent enforcement strength.

- **Architecture lint:** Go's `go/ast` package provides direct AST access. TypeScript needs ts-morph or eslint with custom rules.

- **Hotspot analysis:** Generated code creates false positives. 4 of the top 10 hotspot files in the TypeScript project were `*.gen.ts` code-generated SDK files (5,070, 3,909, 3,318 lines). These would trigger spawn gate blocking and architect routing despite being machine-generated. Any codebase with code generation (OpenAPI, GraphQL, protobuf) needs a generated-file exclusion mechanism.

**The deeper insight:** "Unfakeability" is a property of structural coupling, not compilation specifically. Go's `go build` is unfakeable because source → binary is tightly coupled. TypeScript's Drizzle migration gate (schema change without migration = blocked commit) is equally unfakeable because schema → migration is tightly coupled. Each ecosystem has its own structurally coupled hard gates — the framework should catalog these per language rather than assuming Go's inventory is universal.

---

## What's Working / What Isn't

### Working

**Spawn hotspot gate** blocks feature-impl and systematic-debugging skills from spawning on CRITICAL files (>1,500 lines). This prevents new work from landing on already-degraded files. It routes through architect review first: `--force-hotspot --architect-ref <closed-architect-issue>` provides due process — bypass requires proof of prior review.

**`orch harness init`** automates Day 1 governance for new projects: deny rules, hook registration, beads close hook, pre-commit gate wiring, and control plane lock. A new project goes from zero enforcement to minimum viable harness in under an hour.

**Control plane immutability** via OS-level file locking (`chflags uchg`). Agents cannot modify the files that define their own constraints. Before this, three entropy spirals occurred with mutable infrastructure.

**The hard/soft taxonomy as a design tool.** Once you can classify each harness component, the design conversation changes from "should we add this rule?" to "is this a hard or soft component, and are we putting it in the right container?"

### Not working

**Gates haven't bent the accretion curve.** Total lines, individual file sizes, and weekly velocity all show no deceleration after gate deployment. The deployed gates are too late (completion, not pre-commit), too narrow (block new crossing of 1,500 but exempt files already there), and self-exempting (pre-existing bloat skip creates a ratchet).

**Completion gate exempts pre-existing bloat.** Files already over 1,500 lines get warnings, not blocks. This means the gate primarily protects files approaching the threshold, not files that already crossed it. That's exactly backwards — the already-bloated files are where accretion pressure is strongest.

**Soft harness budget is unknown.** We know 10+ behavioral constraints are inert. We know knowledge transfers at +5 per item with no observed dilution limit. We don't know the exact curve — is it 5 effective behavioral slots? 7? How do constraints from multiple sources interact?

**Generated code creates hotspot false positives.** Any codebase with code generation (OpenAPI codegen, GraphQL schemas, protobuf, icon component generators) will have inflated hotspot counts. Harness tooling needs a generated-code exclusion mechanism.

---

## Open Questions

These are the things we don't know yet. We're listing them because honest gaps are more useful than confident claims without evidence.

### Governance health metric

We've designed but not implemented a composite 0–100 score for harness effectiveness. The components would include: escape hatch frequency (how often agents use `--force` bypasses), accretion velocity (line count growth rate), gate firing rate (how often gates block vs warn), and structural test coverage. We don't know yet whether this collapses into a single useful number or whether the components should be tracked independently.

### Soft harness budget curve

We know the extremes: 0 behavioral constraints = no guidance; 10+ = inert. Where's the useful range? Is it 3 constraints? 5? Does it depend on the constraint's semantic distance from the system prompt? Does it vary by model capability? We need more contrastive experiments with controlled constraint counts to find the curve.

### Layer 4: gates that generate gates

The aspiration is a system that extends its own enforcement. When the entropy agent identifies a pattern 3+ times — the same cross-cutting concern reimplemented independently — it drafts the structural test that would prevent it. The harness extending itself. We haven't built this, and we're not sure it's achievable without creating a gate calibration death spiral (gate too strict → high false positive rate → `--force` reflex → gate becomes noise).

The fix for an ignored gate is never "make it louder" — it's "make it more precise." That principle may apply to auto-generated gates too.

### Cross-language portability depth

We've shown 5 of 8 patterns translate between Go and TypeScript. But we haven't tested: whether accretion thresholds (800/1,500 lines) are appropriate for TypeScript (files may naturally be larger); whether Python has worse or better hard harness than either; whether completion verification can be made language-agnostic beyond Go's build/vet/staticcheck; or whether the full framework works in practice on a non-Go project running agents for 30+ days.

### The mutable control plane problem

All our defenses live inside the system agents can modify. OS-level file locking provides a temporal buffer, but it's not architecturally immutable — any process with sufficient privileges can unlock it. True immutability would require infrastructure that's architecturally unreachable by agents. We don't know what that looks like yet. It may require enforcement at a level above the codebase entirely — CI/CD pipelines, pre-merge hooks on remote repositories, or organizational infrastructure.

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

## Where We Are

We're 12 weeks and 3 entropy spirals into this. The framework is clear: hard harness for enforcement, soft harness for orientation, attractors and gates together, coordination gates as permanent infrastructure. All 5 enforcement layers are now shipped — from pre-commit blocking through AST duplication detection to weekly entropy analysis. The evidence is real but incomplete — the full gate stack has only been deployed for days, not weeks. The 30-day forward measurement is the real test.

The 30-day forward measurement starts from March 8, 2026. The baseline: daemon.go at 1,559 lines, 47,605 total lines across 125 files, 12 files over 800 lines, ~4,200 lines/week growth velocity. What would constitute "bending the curve": daemon.go stabilizes or decreases below 1,500; files over 800 drops below 10; weekly velocity drops below 2,000.

We're publishing now — before the 30-day results — because the framework, the taxonomy, and the failure modes are already useful to teams running into the same problems. The evidence supports the diagnosis (accretion as coordination failure) even if the treatment (gates + attractors) is still proving itself.

The deepest insight from this work: in multi-agent systems, codebase architecture is governance. Package structure is a routing table for agentic contributions. Import boundaries are jurisdiction lines. Structural tests are constitutional constraints. And every convention without a gate will eventually be violated.

---

*This is based on 12 weeks of operating orch-go: ~47,600 lines of Go, 125 files, 50+ autonomous AI agent sessions/day, 3 entropy spirals, 1,625 lost commits, 265 contrastive trials across 7 skills, and cross-language validation against a TypeScript fork. The harness engineering model, evidence probes, and minimum viable harness checklist are open at [repo link].*
