# Knowledge Physics: What Happens When Understanding Has to Survive Without Memory

*A system where AI agents investigate, test hypotheses, and formalize understanding. The knowledge compounds instead of evaporating. The 32nd model it produced describes the physics of the other 31.*

---

## 1. The Meta-Story

Over the past three months, I built a system where AI agents do empirical knowledge work. An agent investigates a question. It records what it finds. If an existing model of that domain exists, the agent tests its findings against the model's claims — confirming, contradicting, or extending them. If the findings warrant it, the model gets updated. If enough unconnected findings accumulate in the same area, a new model gets created to synthesize them.

The system has produced 1,166 investigations, 187 probes (hypothesis tests against models), and 32 models across three months of operation. The 32nd model is called "knowledge physics." It describes the dynamics that govern how the other 31 behave — how knowledge accretes, what pulls findings toward structure, what happens when no one enforces the rules, and how to measure whether the system is compounding understanding or just accumulating files.

That model is what this publication is about. But the model is less interesting than how it was produced. The system that generated it is an automated version of the scientific method: observe, hypothesize, test, formalize. The difference from traditional scientific practice is that the investigators are AI agents with no memory between sessions, operating on shared artifacts in a Git repository. Every session starts from zero. The knowledge persists in the substrate — the files — not in the contributors.

This turns out to be the same situation faced by any organization where contributors come and go but the work has to compound. The physics are the same whether the amnesiac contributors are AI agents cycling through sessions, postdocs rotating through a lab, or engineers cycling through a team. The substrate doesn't care who's writing to it. It cares whether the contributions compose.

---

## 2. The Problem: Institutional Amnesia

Organizations re-learn things they already know. A new team member investigates a design question that was settled six months ago. A researcher repeats a failed experiment because the negative result was never recorded where anyone could find it. A developer reimplements a utility that already exists in three other files because no one told them — and no one could, because the people who wrote those files have moved on.

This is institutional amnesia: the gap between what an organization has learned and what its current contributors can access. It's expensive in every domain. In regulated R&D, unreproduced experiments waste months and compromise audit trails. In software, duplicated infrastructure and re-investigated bugs compound into structural degradation. In any organization with turnover, the cost scales with the rate of contributor cycling.

AI agents have the same problem at 100x speed. A Claude Code session investigates a bug, discovers the root cause, fixes it, and closes. The next session on a related problem starts from zero. There's no memory transfer. If the first session's findings aren't recorded somewhere structural, the second session repeats the investigation. With 50+ agent sessions per day on a single codebase, the re-investigation rate compounds fast.

The conventional response is documentation: write things down, maintain a wiki, keep a knowledge base. This works when the documentation is maintained, discoverable, and current. In practice, documentation accretes — it grows without pruning, contradicts itself, and becomes stale. The problem isn't that people don't document. It's that documentation without structure is just a slower form of forgetting.

What's needed isn't more documentation. It's a system where understanding compounds structurally — where new contributions build on and are connected to existing understanding, where contradictions get detected and resolved, and where the structure itself guides contributors toward what's already known.

---

## 3. The System: Investigation, Probe, Model

The system operates on three artifact types, each with a defined role in a cycle:

**Investigations** are the raw material. An agent (or a human) asks a question, does empirical work — reading code, running experiments, checking behavior — and records the findings. An investigation is self-contained: it states its question, describes what was tested, reports what was observed, and assesses the impact on existing understanding. Creating an investigation is cheap. The system has produced 1,166 of them in three months.

**Models** are the synthesized understanding. A model takes findings from multiple investigations and formalizes them into claims — specific, testable statements about how something works. A model has a summary, a core mechanism section explaining the dynamics, critical invariants that must hold, failure modes that describe how the system breaks, and open questions marking the boundaries of current understanding. The system has 32 models. Each one is a structured representation of what the system knows about a domain.

**Probes** are hypothesis tests. When an agent works in a domain that has an existing model, it enters probe mode: instead of open-ended investigation, the agent tests the model's claims against current evidence. A probe reports a verdict — confirms, contradicts, or extends — for each claim it can assess. Probes live inside their parent model's directory (`.kb/models/{name}/probes/`), creating a structural connection between the test and the thing being tested. The system has produced 187 probes.

**The cycle works like this:**

1. An agent receives a task in some domain.
2. The system surfaces existing models relevant to that domain via context injection.
3. If a relevant model exists, the agent enters probe mode — testing the model's claims against what it observes.
4. The probe's findings get recorded. Confirms verdicts build confidence. Contradicts verdicts trigger model updates. Extends verdicts add new claims.
5. If no model exists but findings accumulate (3+ investigations in the same area), the system flags a synthesis opportunity.
6. A model gets created to synthesize the accumulated findings.
7. Future agents in that domain receive the model's claims, entering probe mode. The cycle continues.

**Why this compounds instead of accreting.** Documentation accretes because there's no mechanism connecting new contributions to existing understanding. You add a page, and it sits alongside other pages with no structural relationship. The investigation/probe/model cycle compounds because the probe mechanism creates feedback: new work is tested against existing models, and existing models are updated based on new evidence. The structure isn't static — it evolves as understanding deepens.

The key architectural decision is that probes live inside model directories. This seems like a minor file system detail, but it converts the connection between findings and models from attention-dependent (the agent has to remember to reference the model) to structural (the probe's location inherently connects it). Before the probe system existed, 94.7% of investigations were orphaned — disconnected from any model. After the probe system, the rate in the same period dropped to 52%.

---

## 4. The Theory: Knowledge Physics

The investigation/probe/model cycle produced a surprising result. The system's harness engineering model — which describes how to govern AI agents writing code — turned out to describe the same dynamics as the knowledge system itself. Code accretes (files grow from individually correct commits). Knowledge accretes (investigations accumulate without synthesis). Code has attractors (packages that pull code toward them). Knowledge has attractors (models that pull findings toward them). Code needs gates (pre-commit hooks, build checks). Knowledge needs gates (and has almost none).

The dynamics are the same. Only the substrate differs.

This led to the theory: four conditions produce substrate-independent dynamics.

**Condition 1: Multiple agents write to the substrate.** Not one person maintaining a codebase or knowledge base, but many contributors — AI agents, team members, rotating researchers — each making independent changes.

**Condition 2: Agents are amnesiac.** No contributor has full context of all prior contributions. AI agents start each session fresh. Humans forget. New team members weren't there. The substrate must carry the context that contributors cannot.

**Condition 3: Contributions are locally correct.** Each individual change passes local validation. The code compiles. The investigation is well-structured. The finding is accurate. Nothing is obviously wrong in isolation.

**Condition 4: No structural coordination mechanism exists.** Locally correct + locally correct does not equal globally correct. Without something routing contributions into a coherent whole, individually good work composes into structural degradation.

When these four conditions hold, four dynamics emerge:

### Accretion

Individually correct contributions compose into structural degradation when shared infrastructure is missing. This is entropy — not a quality problem, but a thermodynamic property of uncoordinated systems.

In code, accretion manifests as file bloat and duplication. Our `daemon.go` grew from 667 lines to 1,559 lines in 60 days from 30 individually correct commits. Each added a locally reasonable capability — stuck detection, health checks, auto-complete. No single commit was wrong. The aggregate was structural degradation. Separately, 6 cross-cutting concerns were independently reimplemented across 4-9 files each, producing roughly 2,100 lines of duplicated infrastructure.

In knowledge, accretion manifests as orphan investigations and semantic overlap. Of 1,166 investigations, 87.6% have no structural connection to any model. Quick entries contain confirmed duplicates. Multiple investigations cover the same ground without awareness of each other.

The mechanism is identical across substrates. The manifestation differs.

### Attractors

Structural destinations that route contributions toward coherent organization. In code, a well-named package pulls code toward it — when `pkg/spawn/backends/` exists, spawn-related code naturally routes there via imports and naming convention. In knowledge, a model pulls findings toward it — when agents receive a model's claims via context injection, their investigation is framed by existing understanding.

Not all attractors behave the same. We observed three distinct behaviors across 32 knowledge models:

- **Attractor models** increase investigation density toward them after creation. The daemon-autonomous-operation model attracted 34 probes across 13 dates over 21 days — sustained gravitational pull. New work flows toward the model because agents in that domain receive its claims.
- **Capstone models** decrease investigation density. They synthesize and settle a topic. The entropy-spiral model's reference rate dropped from 12.5% to 8.8% after formalization — the model absorbed the open questions.
- **Dormant models** exist but generate no probes or investigations. Seven models have zero probes. Either complete, abandoned, or forgotten.

A key asymmetry between substrates: code attractors are structurally coupled (the compiler enforces imports), while knowledge attractors are attention-primed (context injection influences but doesn't enforce). The probe system partially bridges this gap — probes live in model directories, creating structural coupling — which is why the orphan rate dropped from 94.7% to 52% when probes were introduced.

### Gates

Enforcement mechanisms that block wrong paths. In code: pre-commit hooks reject commits that grow files past thresholds, build checks reject code that doesn't compile, spawn gates refuse to create agent sessions targeting degraded files. In knowledge: almost nothing. Every knowledge convention is advisory.

This is the sharpest finding from measuring knowledge dynamics. Every knowledge transition in the system is either ungated or advisory-only:

| Transition | Status | What Happens |
|------------|--------|-------------|
| Investigation to model synthesis | Ungated | No mechanism forces synthesis when investigations accumulate |
| Probe to model update | Advisory | Skill template says "merge findings" — 48% skip it |
| Quick entry deduplication | Ungated | No check against existing entries |
| Decision to implementation | Ungated | 1 of 56 decisions has enforcement (1.8%) |
| Prior Work citation | Advisory | Template includes it — 48% of investigations skip it |
| Knowledge consistency at commit | Ungated | Pre-commit hooks run on `.go` files, not `.kb/` files |

Every convention without a gate is violated at significant rates. This is the same invariant discovered in the code substrate, now confirmed in a second substrate.

### Entropy

Measurement of whether composition is failing. Code has lines-per-file, duplication detection, fix-to-feature ratios, hotspot analysis. Knowledge has the orphan rate (measured here for the first time: 87.6% raw, 52% in the model era) and synthesis backlog (4 clusters, 17 investigations). Most knowledge entropy metrics don't exist yet — claims per model, contradiction backlog, semantic overlap — because knowledge measurement is less mature than code measurement.

---

## 5. Evidence: Code Substrate

The code substrate provides the most detailed evidence because code has mature measurement tools and the system has been operating on a Go codebase (~47,600 lines, 125 files) with 50+ autonomous AI agent sessions per day for 12 weeks.

### Accretion trajectory

`daemon.go` grew from 667 to 1,559 lines in 60 days. The growth came from 30 individually correct commits, each adding a locally reasonable capability. The velocity accelerated over time:

| Period | Lines Added | Velocity |
|--------|-------------|----------|
| Dec 22 - Jan 19 | +261 | ~65/week |
| Jan 19 - Feb 16 | +215 | ~54/week |
| Feb 23 - Mar 2 | +176 | 176/week |
| Mar 2 - Mar 8 | +345 | 345/week |

Extraction is temporary without structural prevention. `spawn_cmd.go` shrank from 2,432 lines to 677 after a destination package was created — a -1,755 line extraction. Then it regrew to 1,160 lines in 3 weeks (+483 lines, ~160/week). The gravitational center (the Cobra command definition) still lived in the original file, pulling new code back. Extraction without routing is a pump — you can cool a room without insulation, but it heats right back up.

### Three entropy spirals

Three entropy spirals over 12 weeks, with 1,625 commits lost in a single crash. The system learned intellectually after each (post-mortems were written) but not structurally (no gates were implemented between spirals). The post-mortems were soft harness — documentation of what went wrong — without hard harness to prevent recurrence. The spirals only stopped when hard gates were deployed.

### Contrastive measurement: soft instructions fail under pressure

265 contrastive trials across 7 agent skills measured how well written instructions change agent behavior:

| Content Type | Mechanism | Measured Effect |
|---|---|---|
| Knowledge (facts, templates) | Direct — agent reads and applies | +5 point lift |
| Stance (attention primers) | Indirect — shifts what agents notice | +2 to +7 on cross-source scenarios |
| Behavioral (MUST/NEVER rules) | Unreliable — dilutes at scale | Inert at 10+ co-resident rules |

The critical finding: behavioral constraints — the enforcement you'd most want — are the content type most vulnerable to dilution. An orchestrator skill had 87 behavioral constraints; approximately 83 were non-functional. A convention in documentation without mechanical enforcement is a suggestion with a half-life proportional to context window pressure. `daemon.go` grew past the stated 1,500-line convention while that convention existed in the project charter. The agents read it. They understood it. They added 200 lines to an 1,800-line file anyway because the task was urgent and no gate blocked them.

### Hard vs soft harness

Every component of the development environment that constrains agent behavior is a harness. The single most useful classification: hard (deterministic — passes or fails, cannot be bypassed without an explicit escape hatch) vs soft (probabilistic — influences via context, drifts under pressure, dilutes at scale).

Hard harness doesn't need measurement — a build passes or fails. Soft harness needs contrastive testing to know whether it works at all. The default assumption for any soft harness component should be "probably doesn't work" until tested.

Deployed hard harness in the code substrate: pre-commit growth gates, compilation checks (`go build`), spawn gates refusing agent sessions on degraded files, structural tests enforcing package boundaries, control plane immutability via OS-level file locking, completion verification pipeline.

Honest assessment: gates haven't bent the aggregate code growth curve yet. Total lines grew from 34,977 to 47,605 in 3 weeks after gate deployment. The deployed gates were too late (completion time, not authoring time), too narrow (block new crossing of thresholds but exempt files already past them), and self-exempting (pre-existing bloat receives warnings, not blocks). The gate stack needed tightening — structural attractors (destination packages) plus pre-commit blocking on already-bloated files. The 30-day forward measurement from March 8, 2026 is the real test.

### Compliance failure vs coordination failure

This distinction is central to the theory's permanence claim.

**Compliance failure:** An agent doesn't follow instructions. Example: agent ignores a documented convention. Fixed by better models — smarter agents follow instructions more reliably.

**Coordination failure:** Agents each follow instructions correctly but collectively produce entropy. Example: 30 agents each add locally correct code; `daemon.go` grows +892 lines. Not fixed by better models — made worse by faster, more confident agents.

The `daemon.go` evidence is coordination failure. Each of the 30 commits followed instructions. Each was locally rational. Each passed review. The problem was the absence of structural coordination — no shared packages existed, no deduplication detection, no cross-agent awareness that workspace scanning was already implemented four times.

Controlled demonstration (March 9, 2026): two agents (Haiku and Opus, representing different capability levels) independently implemented the same function (`FormatBytes`) on the same codebase from the same baseline commit. Both achieved 6/6 on individual performance metrics — completion, compilation, tests passing, no regression, file discipline, spec match. Both produced nearly identical commit messages independently. The merge: 100% conflict. Both appended to the same insertion points in the same files. Git cannot auto-merge identical insertions at the same position. The coordination failure is structural, not a capability issue. Even perfect agents conflict when they modify the same insertion points without coordination protocol.

---

## 6. Evidence: Knowledge Substrate

The knowledge substrate provides the theoretical validation: if the same dynamics appear in a fundamentally different medium (markdown files vs compiled code), the dynamics are properties of the system configuration, not the substrate.

### 1,166 investigations measured

The system has produced 1,166 investigations across three months. The raw orphan rate (investigations not connected to any model) is 87.6%. But this rate decomposes:

| Era | Total | Orphaned | Orphan Rate |
|-----|-------|----------|-------------|
| Pre-model era (Dec 2025 - Jan 2026) | 969 | 918 | 94.7% |
| Model era (Feb - Mar 2026) | 196 | 102 | 52.0% |
| All | 1,166 | 1,021 | 87.6% |

The pre-model era constitutes 83% of the corpus. Those investigations were created before the model/probe system existed — they were structurally impossible to connect to models. The meaningful rate is the model-era rate: 52%.

### Orphan taxonomy: not all orphans are pathological

A 35-file sample revealed six orphan categories:

| Category | Rate | Natural? |
|----------|------|----------|
| Implementation-as-investigation (wrong skill routing) | 30-45% | Yes |
| Audit/design snapshots | 25-33% | Yes |
| Exploratory (one-off questions) | 15-20% | Yes |
| Genuinely lost (should feed a model, didn't) | ~20% of orphans | No |
| Negative results (valuable to record) | 5-7% | Yes |
| Superseded (later work covers same ground) | 3-5% | Yes |

Approximately 80% of orphans are naturally expected. The actionable signal is the "genuinely lost" rate — roughly 10% of total investigations. This is comparable to healthy dead code rates in mature codebases (5-15%). A 40-50% orphan rate is healthy for an exploratory knowledge system; above 60% signals under-synthesis.

### Probe displacement: the system self-healed

When the probe system activated in February 2026, investigation volume dropped 76% (from 548/month to 129/month) while probes rose to 160/month. The system shifted from producing disconnected investigations to producing structurally-connected probes — hypothesis tests that live inside their parent model's directory.

| Month | Investigations | Probes | Models |
|-------|---------------|--------|--------|
| Dec 2025 | 421 | 0 | 0 |
| Jan 2026 | 548 | 0 | 16 |
| Feb 2026 | 129 | 160 | 9 |
| Mar 2026 | 67 | 29 | 7 |

This is the knowledge equivalent of introducing `import` statements. Before probes, investigations were like functions defined in random files with no connection to their calling package. After probes, findings are structurally coupled to the models they test. The orphan rate dropped from 94.7% to 52% — not through retroactive cleanup, but through architectural change that prevents orphaning at creation time.

### Three model behaviors

Across 32 models, three distinct behaviors emerged:

- **Attractor:** daemon-autonomous-operation attracted 34 probes across 13 dates over 21 days. The model actively pulls new work — agents spawned in the domain receive the model's claims via context injection, framing their investigation. Harness-engineering showed a launch burst pattern: 2% pre-model reference rate jumped to 100% post-creation.

- **Capstone:** entropy-spiral's reference rate dropped from 12.5% to 8.8% after formalization. The model synthesized and settled the topic. There was less to investigate.

- **Dormant:** 7 models with zero probes. Complete, abandoned, or forgotten — no mechanism distinguishes these states.

### Zero hard gates, measured consequences

Every knowledge convention is violated at significant rates. Prior Work tables (48% skip rate). Probe-to-model merge (advisory only — contradiction verdicts accumulated before batch resolution). Quick entry deduplication (confirmed duplicates exist, no automated checking). Decision enforcement (1 of 56 decisions has a check: 1.8%).

The code substrate's invariant — "every convention without a gate will eventually be violated" — is confirmed in the knowledge substrate across every measured transition.

### First hard gate experiment

The first hard knowledge gate has been designed: `kb create investigation --model X` as a required flag, with `--orphan` for explicit opt-out. If adding structural coupling at creation time drops the orphan rate further (as probes did from 94.7% to 52%), the system has causal evidence that the physics predict interventions — they don't just describe patterns.

This experiment is in progress. The probe system demonstrated that structural coupling works in knowledge (directory-level connection reduced orphans by 42 percentage points). A creation-time gate would test whether earlier coupling — at investigation creation rather than probe creation — further reduces the genuinely-lost rate.

---

## 7. The Sharp Claim

Stronger models and stronger contributors need *more* coordination infrastructure, not less.

If governance were only about compliance — getting agents to follow instructions — it would become obsolete as models improve. Smarter agents follow instructions better. But the daemon.go evidence is not compliance failure. Thirty agents each followed instructions correctly and collectively produced 892 lines of uncoordinated growth. The coordination demo confirmed it empirically: two agents each achieved perfect individual scores (6/6), then produced 100% merge conflict.

Coordination failure is an emergent property of multi-agent systems. It doesn't resolve with individual capability. A company of 30 brilliant engineers with no architecture review still produces spaghetti — possibly faster than 30 mediocre engineers, because each builds more in less time. Architecture review isn't compensating for incompetence. It's providing the coordination layer that individual competence cannot.

This makes the system permanent, not transitional.

Compliance gates (pre-commit checks, build validation) may simplify over time as models get better at self-limiting. Coordination gates (structural tests, duplication detection, entropy measurement, synthesis enforcement) become the primary investment as agents get more capable and autonomous. A more capable agent accretes more code per session with higher confidence. A more capable researcher produces more findings per session with higher quality — each one locally correct, collectively uncoordinated.

The investigation/probe/model cycle is coordination infrastructure for knowledge. Models are coordination points — structural destinations that route findings toward coherent understanding. Probes are coordination checks — tests that verify new findings against existing models. The cycle doesn't compensate for bad agents. It provides the structural coordination that good agents cannot provide for themselves.

---

## 8. Running It Yourself

### What you need

The investigation/probe/model cycle separates cleanly into substrate (the knowledge system) and orchestration (the infrastructure that drives it at scale). You don't need the full orchestration stack. Five components form the minimal substrate:

| Component | Role | Why Irreducible |
|-----------|------|----------------|
| An AI agent runtime (Claude Code, Cursor, or similar) | The entity that investigates, tests, observes, writes | The cycle needs someone to do the empirical work |
| `kb` CLI | Knowledge artifact management — context retrieval, artifact creation | Stores conventions; could be replaced by raw directory knowledge |
| Git | Version control for `.kb/` artifacts | Audit trail, collaboration, diff visibility |
| `.kb/` directory | The shared mutable substrate | Where artifacts live |
| An investigation skill | Cycle conventions — probe mode, templates, merge protocol | Without it, the agent doesn't know the cycle |

The full orchestration stack (agent spawning, issue tracking, autonomous daemon, completion verification, dashboard) adds reliability at scale but isn't required for the cycle itself. Empirically tested: every step of the cycle works without the orchestration binary — the agent runs `kb context` itself (1 bash call), reads model files directly (1-3 read calls), creates probes by writing files to model directories. The gap is convenience (2-4 extra tool calls per session), not capability.

### The first user

The system's evidence comes from a solo developer using AI agents on a complex project. The first external user who replicates this profile has the lowest adoption friction:

- Already uses an AI coding assistant (Claude Code, Cursor, Copilot)
- Works on a complex project where they forget their own prior decisions
- Comfortable with Git and CLI tools
- Frustrated by "I solved this before but can't remember how"

This is the Solo Technical Researcher — a developer or researcher working alone on a long-running project where understanding needs to compound across sessions. They don't need team features, organizational buy-in, or a training program. They need a `.kb/` directory and conventions for how to use it.

### How to start

**1. Initialize.** `kb init` creates the `.kb/` directory structure. Add a `models/` directory — this is where synthesized understanding lives.

**2. Investigate.** When you or your agent encounters a question worth recording: `kb create investigation "my-question"`. Do the empirical work. Record findings. This is the cheapest artifact to create and the starting point for everything.

**3. Synthesize.** After 3+ investigations in the same area, create a model. A model has: a summary (30-second version), a core mechanism (how it works), critical invariants (what must hold), failure modes (how it breaks), and open questions (what you don't know yet). The model is the fundamental unit — without it, investigations are homeless.

**4. Probe.** When you work in a domain with an existing model, enter probe mode. Test the model's claims against what you observe. Record the verdict: confirms, contradicts, or extends. Put the probe file in `.kb/models/{name}/probes/`. This structural placement is what prevents orphaning.

**5. Close the loop.** When a probe contradicts a model claim, update the model. When a probe extends it, add the new claim. The model evolves based on evidence, not opinion.

The system produces value from day one (recorded investigations are searchable context for future sessions), but the compounding effect becomes visible after you have your first model being tested by probes. That's the "magic moment" — when you see an agent discover something you already knew, confirm it against the model, and extend the model with a new finding you didn't know.

### What to expect

Adoption patterns from comparable tools (ADRs, Zettelkasten, electronic lab notebooks) follow a consistent sequence: individual practitioner solves their own problem, nearby collaborators see value through exposure, champions carry the practice to new contexts, institutional endorsement codifies it. ADRs took 7 years from individual projects (2011) to ThoughtWorks "Adopt" (2018) to UK Government mandate (2025). Bottom-up adoption is stickier than top-down mandates.

The physics become visible once you have enough volume. With 3-5 investigations per week, expect to see your first synthesis opportunity within 2-3 weeks and your first model within a month. The attractor effect — where a model actively pulls new work toward it — becomes noticeable after the model has been probed 3-5 times.

---

## 9. Honest Gaps

These are the things we don't know yet. They're listed because honest boundaries are more useful than confident claims without evidence.

### Soft harness budget curve

We know 0 behavioral constraints means no guidance and 10+ means inert. Where's the useful range? Is it 3 constraints? 5? Does it vary by model capability? The 265 contrastive trials showed the extremes but didn't map the curve. This matters because most governance will necessarily be soft harness — not everything can be a hard gate.

### Cross-substrate validation

Code and knowledge are two confirmed instances of the physics. The theory predicts the same dynamics in database schemas, config systems, API surfaces, and documentation. No third substrate has been tested empirically. Until it has, the generalization claim rests on structural argument (the four conditions hold for these substrates) rather than measurement.

### Knowledge pre-commit hooks

Code has pre-commit hooks that validate syntax, lint, and compilation. Knowledge equivalents would need to check: does this investigation cite prior work? Does this probe reference its model? Does the model contradict itself? The technical challenge is that knowledge validation is semantic, not syntactic. Current approaches either require LLM calls at commit time (expensive, slow, non-deterministic) or structural heuristics (fast but shallow). The right approach is unclear.

### Whether hard knowledge gates reduce entropy or add ceremony

Code gates (pre-commit, spawn, completion) have demonstrated prevention of measurable damage. But knowledge creation is more exploratory than code writing. Mandatory model-linking might slow legitimate exploration — the investigation you file without a model attachment might be the seed of a model that doesn't exist yet. The first hard gate experiment (requiring `--model` flag with `--orphan` opt-out) will test this tradeoff.

### Minimum time to visible physics

This system shows the dynamics after 1,166 investigations over three months. A solo researcher generating 3-5 investigations per week might not see the dynamics (orphan accumulation, attractor pull, synthesis opportunities) for weeks. The "magic moment" — when compounding value becomes obvious — may need to be accelerated through seeded examples or guided onboarding for first-time users.

### Governance health metric

Is there a composite 0-100 score for knowledge system health? Components would include: orphan rate, probe freshness, contradiction backlog, synthesis debt. We don't know whether these collapse into a single useful number or should be tracked independently. The code equivalent (combining lines-per-file, duplication rate, fix-to-feature ratio) hasn't been validated as a composite either.

### The mutable control plane

All defenses live inside the system contributors can modify. In code, we mitigated this with OS-level file locking — agents can't edit the files that define their own constraints. In knowledge, no equivalent exists. A probe could contradict a model and the model could remain unchanged indefinitely. True immutability for knowledge governance — enforcement that can't be drifted from — remains an open architectural problem.

---

*This is based on three months of operating a knowledge system: 1,166 investigations, 187 probes, 32 models, a parallel codebase governance system (~47,600 lines of Go, 125 files, 50+ agent sessions/day), 265 contrastive trials across 7 skills, 3 entropy spirals, 1,625 lost commits, and one controlled coordination failure demonstration. The knowledge physics model, evidence, and investigation/probe/model cycle conventions are available at [repo link].*
