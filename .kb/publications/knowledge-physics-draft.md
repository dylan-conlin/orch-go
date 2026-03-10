# Knowledge Physics: Why Stronger AI Agents Need More Coordination, Not Less

*A system where AI agents investigate, test hypotheses, and formalize understanding — and the physics that govern what happens when they don't coordinate.*

---

## The Experiment That Explains Everything

Give two AI agents the same task on the same codebase. Same starting point, same instructions, no communication between them.

We did this 20 times. The task was simple: add a `FormatBytes` function to a Go package. Each agent independently wrote the code, wrote tests, and committed. Every single run — all 20 — scored perfectly on individual metrics. Completion, compilation, tests passing, no regressions, spec compliance. The agents followed every instruction correctly.

Then we merged their work. Conflict rate: 100%. Every trial, without exception.

Both agents appended their function to the same location in the same file. Both appended their tests to the same location in the test file. Both produced nearly identical commit messages — independent convergence on `feat: add FormatBytes function for human-readable byte formatting`. Git cannot auto-merge identical insertions at the same position. Twenty perfect individual performances. Twenty merge failures.

This wasn't a model capability issue. We tested across capability levels — Claude Haiku (fast, smaller) and Claude Opus (slower, more capable). Same result. Fisher's exact test: p=1.0. The coordination failure rate is identical regardless of model capability.

We escalated to a harder task. A multi-file table renderer: `VisualWidth` function in one file, `RenderTable` in a new file, tests for both. Deliberately ambiguous design choices — column separators, header styles, extra-column handling — left to each agent's judgment.

Both agents scored 10/10 on individual metrics. Again, perfect compliance. Then the merge: conflict across all four files. But now something new appeared. Beyond the textual conflicts that Git couldn't resolve, there were *semantic* conflicts — the two implementations made incompatible design decisions. Haiku expanded the table when rows had extra columns. Opus ignored extras and truncated to header width. Even if a human resolved the merge markers, one agent's tests would fail against the other's implementation.

The complex task also revealed a genuine capability difference that compliance scoring can't detect. Opus anticipated that `len(string)` returns byte count, not character count, and implemented rune-counting for Unicode correctness — unprompted, with Unicode test cases the spec never mentioned. Haiku used `len()` directly. Both passed their own tests. Both scored 10/10. But Haiku's implementation is subtly wrong for any non-ASCII input.

Here's the point: **stronger models don't reduce coordination failures. They may intensify them.** Opus wrote a more sophisticated implementation, with more design opinions, creating deeper semantic conflicts on merge. The capability gap between the models showed up in edge case handling and design judgment — exactly the dimensions where independent agents diverge most.

This is the central finding behind everything that follows.

---

## The Problem: Institutional Amnesia

Organizations re-learn things they already know. A new team member investigates a design question settled six months ago. A researcher repeats a failed experiment because the negative result was never recorded. A developer reimplements a utility that exists in three other files because the people who wrote them have moved on.

This is institutional amnesia: the gap between what an organization has learned and what its current contributors can access. It's expensive everywhere. In regulated R&D, unreproduced experiments waste months. In software, duplicated infrastructure compounds into structural degradation. The cost scales with contributor cycling rate.

AI agents have this problem at 100x speed. A Claude Code session investigates a bug, discovers the root cause, fixes it, closes. The next session on a related problem starts from zero. With 50+ agent sessions per day on a single codebase, the re-investigation rate compounds fast.

The conventional response is documentation. The problem isn't that people don't document. It's that documentation without structure is just a slower form of forgetting. What's needed is a system where understanding compounds structurally — where new contributions connect to existing understanding, contradictions get detected, and the structure itself guides contributors toward what's already known.

---

## The System: Investigation, Probe, Model

Over three months, I built a system where AI agents do empirical knowledge work. The system operates on three artifact types in a cycle.

**Investigations** are the raw material — an agent asks a question, does empirical work, records findings. Creating one is cheap. The system produced 1,166 in three months.

**Models** formalize understanding from multiple investigations into testable claims — a summary, core mechanism, critical invariants, failure modes, and open questions. The system has 32 models.

**Probes** are hypothesis tests. When an agent works in a domain with an existing model, it tests the model's claims against current evidence, reporting a verdict: confirms, contradicts, or extends. Probes live inside their parent model's directory (`.kb/models/{name}/probes/`), creating a structural connection between test and claim. 187 probes so far.

The cycle: an agent receives a task, the system surfaces relevant models via context injection, the agent enters probe mode and tests claims against observation, findings get recorded and merged back into models. When findings accumulate without a model (3+ investigations in the same area), the system flags a synthesis opportunity and a model gets created. Future agents receive that model's claims. The cycle continues.

This is the automated scientific method: observe, hypothesize, test, formalize. The 32nd model the system produced describes the dynamics governing how the other 31 behave. That model is what this publication is about.

---

## The Theory: Knowledge Physics

The coordination demo reveals something that extends far beyond merge conflicts. The same dynamics appear in every shared system we've measured — code, knowledge, and likely any substrate where amnesiac contributors make independent changes.

Four conditions produce these dynamics:

1. **Multiple agents write to the substrate.** Not one person, but many contributors making independent changes.
2. **Agents are amnesiac.** No contributor has full context of all prior work. AI agents start fresh each session. Humans forget. New team members weren't there.
3. **Contributions are locally correct.** Each change passes local validation in isolation.
4. **No structural coordination mechanism exists.** Locally correct + locally correct ≠ globally correct.

When these conditions hold, four dynamics emerge — regardless of what the substrate is made of.

### Accretion

Individually correct contributions compose into structural degradation. This is entropy, not a quality problem — a thermodynamic property of uncoordinated systems.

### Attractors

Structural destinations that route contributions toward coherent organization. A well-named package pulls code toward it. A well-structured model pulls findings toward it.

### Gates

Enforcement that blocks wrong paths. Pre-commit hooks, build checks, required flags — mechanisms where compliance is binary, not probabilistic.

### Entropy measurement

Detection of when composition is failing. Lines-per-file, orphan rates, duplication counts — metrics that make degradation visible before it becomes structural.

These dynamics are substrate-independent. They emerge from the system configuration (the four conditions), not from properties of the medium.

---

## Evidence: Code Substrate

We've run 50+ autonomous AI agent sessions per day on a Go codebase (~47,600 lines, 125 files) for 12 weeks. The evidence is detailed because code has mature measurement tools.

### Accretion in the wild

`daemon.go` grew from 667 to 1,559 lines in 60 days from 30 individually correct commits. Each added a locally reasonable capability — stuck detection, health checks, auto-complete. No single commit was wrong. The aggregate was structural degradation. The velocity accelerated: ~65 lines/week in the first month, 345 lines/week by the third.

Separately, 6 cross-cutting concerns were independently reimplemented across 4–9 files each, producing roughly 2,100 lines of duplicated infrastructure. Workspace scanning alone was implemented five times by five agents who had no awareness of each other's work.

Extraction is temporary without structural prevention. `spawn_cmd.go` shrank from 2,432 to 677 lines after a destination package was created — a -1,755 line extraction. Then it regrew to 1,160 lines in 3 weeks. The gravitational center (the command definition) still lived in the original file. Extraction without routing is a pump.

### Soft instructions fail under pressure

265 contrastive trials across 7 agent skills measured how written instructions change behavior:

| Content Type | Transfer Mechanism | Effect |
|---|---|---|
| Knowledge (facts, templates) | Direct application | +5 point lift |
| Stance (attention primers) | Shifts what agents notice | +2 to +7 |
| Behavioral (MUST/NEVER rules) | Unreliable at scale | Inert at 10+ rules |

Our orchestrator skill had 87 behavioral constraints. Approximately 83 were non-functional. `daemon.go` grew past the stated 1,500-line convention while that convention existed in the project charter. The agents read it, understood it, and added 200 lines to an 1,800-line file anyway because the task was urgent and no gate blocked them.

### Three entropy spirals

Three entropy spirals in 12 weeks, with 1,625 commits lost in a single crash. The system learned intellectually after each — post-mortems were written — but not structurally. No gates were implemented between spirals. The post-mortems were soft harness (documentation) without hard harness (enforcement). The spirals stopped only when hard gates were deployed.

### Hard vs soft harness

Every harness component is either hard (deterministic — passes or fails, cannot be bypassed without explicit escape hatch) or soft (probabilistic — influences via context, drifts under pressure). Hard harness doesn't need measurement. Soft harness needs contrastive testing to know whether it works at all.

---

## Evidence: Knowledge Substrate

If the same dynamics appear in a fundamentally different medium, the physics are properties of the system configuration, not the substrate. Knowledge provides that second confirmation.

### 1,166 investigations, measured

The raw orphan rate (investigations not connected to any model) is 87.6%. But this decomposes:

| Era | Total | Orphan Rate |
|-----|-------|-------------|
| Pre-model era (before probes existed) | 969 | 94.7% |
| Model era (with probes) | 196 | 52.0% |

The pre-model era is 83% of the corpus — investigations created before the structural connection mechanism existed. The meaningful rate is 52%.

A 35-file sample revealed that ~80% of orphans are naturally expected (implementation-as-investigation, one-off explorations, negative results). The actionable "genuinely lost" rate is roughly 10% — comparable to healthy dead code rates in mature codebases.

### The probe displacement

When probes activated, investigation volume dropped 76% while probes rose to 160/month. The system shifted from producing disconnected findings to producing structurally-connected hypothesis tests. The orphan rate dropped from 94.7% to 52% — not through cleanup, but through architectural change that prevents orphaning at creation time.

This is the knowledge equivalent of introducing `import` statements. Before probes, investigations were functions defined in random files. After probes, findings are structurally coupled to the models they test.

### Zero hard gates, measured consequences

Every knowledge convention is violated at significant rates:

| Convention | Violation Rate |
|-----------|---------------|
| Prior Work citation in investigations | 48% skip it |
| Probe-to-model merge after contradicting findings | Advisory only — verdicts accumulated unresolved |
| Quick entry deduplication | No automated checking; confirmed duplicates exist |
| Decision enforcement | 1 of 56 decisions has a check (1.8%) |

The code substrate's invariant — "every convention without a gate will eventually be violated" — confirmed in the knowledge substrate across every measured transition.

### Three model behaviors

Across 32 models, three distinct behaviors: **attractors** that increase investigation density toward them after creation (the daemon model attracted 34 probes over 21 days), **capstones** that decrease density by settling a topic (entropy-spiral's reference rate dropped from 12.5% to 8.8%), and **dormant** models with zero probes.

A key asymmetry: code attractors are structurally coupled (the compiler enforces imports), knowledge attractors are attention-primed (context injection influences but doesn't enforce). The probe system partially bridges this — probes live in model directories, converting attention priming into structural coupling.

---

## The Sharp Claim: Compliance vs Coordination

Agent governance addresses two fundamentally different failure modes, and they respond oppositely to model improvement.

**Compliance failure:** an agent doesn't follow instructions. Fixed by better models — smarter agents follow instructions more reliably.

**Coordination failure:** agents each follow instructions correctly but collectively produce entropy. Not fixed by better models. Made *worse* by faster, more confident agents.

The coordination demo proves this empirically. Both Haiku and Opus achieved perfect individual scores. The merge conflict rate was identical: 100%. On the complex task, Opus's greater capability produced *deeper* semantic conflicts — more sophisticated design judgments that diverged more fundamentally from Haiku's approach.

The `daemon.go` evidence is coordination failure. Each of 30 commits followed instructions. Each was locally rational. Each passed review. The problem was the absence of structural coordination — no shared packages, no deduplication detection, no cross-agent awareness that workspace scanning was already implemented four times.

The analogy: a company of 30 brilliant engineers with no architecture review still produces spaghetti — possibly faster than 30 mediocre engineers, because each builds more in less time. Architecture review isn't compensating for incompetence. It's providing the coordination layer that individual competence cannot.

**This makes coordination infrastructure permanent, not transitional.**

Compliance gates (pre-commit checks, build validation) may simplify as models improve. Coordination gates — structural tests, duplication detection, entropy measurement, synthesis enforcement — become the primary investment as agents get more capable. A more capable agent accretes more code per session with higher confidence. A more capable researcher produces more findings per session with higher quality — each locally correct, collectively uncoordinated.

The investigation/probe/model cycle is coordination infrastructure for knowledge. Models are coordination points — structural destinations routing findings toward coherent understanding. Probes are coordination checks — tests verifying new findings against existing structure. The cycle doesn't compensate for bad agents. It provides the structural coordination that good agents cannot provide for themselves.

---

## Running It Yourself

The investigation/probe/model cycle separates cleanly into substrate (the knowledge system) and orchestration (infrastructure for scale). Five components form the minimal substrate:

| Component | Role |
|-----------|------|
| AI agent runtime (Claude Code, Cursor) | Does the empirical work |
| `kb` CLI or equivalent | Artifact management, context retrieval |
| Git | Audit trail, collaboration |
| `.kb/` directory | The shared mutable substrate |
| Investigation skill/conventions | The cycle protocol |

The full orchestration stack (agent spawning, issue tracking, autonomous daemon, completion verification) adds reliability at scale but isn't required for the cycle itself.

**How to start:** Initialize a `.kb/` directory with `models/` and `investigations/` subdirectories. When you or your agent encounters a question worth recording, create an investigation. After 3+ investigations in the same area, create a model with testable claims. When working in a domain with an existing model, run a probe — test the claims, record the verdict, update the model. The compounding effect becomes visible after your first model is tested by probes.

**Who this is for:** The first user is a solo developer or researcher working on a complex long-running project with AI agents — someone who forgets their own prior decisions and is frustrated by re-investigation. No team features, organizational buy-in, or training required. Adoption patterns from comparable tools (ADRs, Zettelkasten, electronic lab notebooks) show bottom-up adoption is stickier than top-down mandates.

---

## Honest Gaps

**The coordination claim needs more scale.** Our controlled demo used 2 agents on 1 task. The `daemon.go` evidence spans 30 commits over 60 days. We haven't run a controlled experiment at scale showing that coordination failure rate increases with agent count or agent capability — the current evidence is structural argument plus small controlled demos.

**Cross-substrate validation is theoretical.** Code and knowledge are two confirmed instances. The theory predicts the same dynamics in database schemas, config systems, API surfaces, documentation. No third substrate has been tested empirically.

**Knowledge gates are unproven.** We've described the gate deficit in the knowledge substrate but haven't deployed hard knowledge gates yet. Whether requiring `--model` flag on investigation creation reduces the genuinely-lost rate (as probes reduced the orphan rate from 94.7% to 52%) is an active experiment.

**Soft harness budget curve is unknown.** We know 10+ behavioral constraints are inert. We don't know the useful range. 3 constraints? 5? This matters because most governance will necessarily be soft.

**The mutable control plane.** All defenses live inside the system contributors can modify. OS-level file locking provides a buffer for code, but knowledge has no equivalent. Enforcement that can't be drifted from remains an open architectural problem.

**Minimum time to visible physics.** This system shows dynamics after 1,166 investigations over months. A solo researcher at 3–5 investigations per week may not see the compounding effect for weeks. The "magic moment" may need acceleration through guided onboarding.

---

*Based on three months of operating a knowledge system: 1,166 investigations, 187 probes, 32 models, a parallel codebase (~47,600 lines of Go, 50+ agent sessions/day), 265 contrastive trials, 3 entropy spirals, 1,625 lost commits, 20 controlled coordination trials (simple task N=10 × 2 models, complex task N=1 × 2 models), and one system whose 32nd model describes the physics of the other 31.*
