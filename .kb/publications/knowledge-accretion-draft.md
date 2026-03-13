# Knowledge Accretion: What Happens When No One Remembers

*Why shared systems degrade from correct contributions — a diagnostic framework derived from 1,166 investigations, tested against 15 counterexamples, and observed in two substrates (code and knowledge) where amnesiac agents contribute.*

---

## The Pattern You Already Know

You've seen it. A shared wiki where every page was written correctly but nobody can find anything. A codebase where every commit passed review but the architecture is incoherent. A Confluence space with 2,000 pages and no table of contents. A database with 40 columns nobody can explain.

The individual contributions are fine. The aggregate is a mess.

This isn't a quality problem. It's a coordination problem. And it follows dynamics I've spent three months measuring, formalizing, and trying to break — in one system, operated by one person, across two substrates.

---

## The System That Produced This

I should explain how I arrived at these claims, because the method matters as much as the findings.

Over three months, I built a system where AI agents do empirical knowledge work. The system operates on three artifact types in a cycle:

**Investigations** are the raw material — an agent asks a question, does empirical work, records findings. Creating one is cheap. The system produced 1,166 in three months.

**Models** formalize understanding from multiple investigations into testable claims — a summary, core mechanism, critical invariants, failure modes, and open questions. The system has 32 models.

**Probes** are hypothesis tests. When an agent works in a domain with an existing model, it tests the model's claims against current evidence, reporting a verdict: confirms, contradicts, or extends. Probes live inside their parent model's directory, creating a structural connection between test and claim. 187 probes so far.

The cycle: investigate, formalize claims into models, test claims with probes, update models from probe findings. When investigations accumulate without a model (3+ in the same area), the system flags a synthesis opportunity and a model gets created. Future agents receive that model's claims as context. The cycle continues.

The 32nd model the system produced describes the dynamics governing the other 31. It's a model of how shared systems degrade or thrive based on their structural properties — and it survived 15+ falsification attempts, though all those attempts were conducted by agents operating inside the framework itself (a closed-loop limitation I'll address below).

---

## Five Conditions

This is a diagnostic framework — structural conditions that diagnose whether a shared resource is at risk of degradation. The closest comparison in type (not in evidence base) is Elinor Ostrom's institutional analysis of commons governance. Ostrom identified structural conditions, empirically derived from studying hundreds of commons across dozens of countries over decades, that predict whether a fishery collapses or a forest sustains. Her design principles don't guarantee outcomes — they diagnose structural risk.

Knowledge accretion aspires to work the same way, but from a far narrower evidence base — one system, one operator, three months. Five conditions diagnose whether a shared system is at risk of degrading from individually correct contributions:

1. **Multiple agents write to the substrate.** Not one person, but many contributors making independent changes.

2. **Agents are amnesiac.** No contributor has full context of all prior work. AI agents start fresh each session. Humans forget. New team members weren't there.

3. **Contributions are locally correct.** Each change passes local validation in isolation. This isn't about bad work — it's about good work that doesn't compose.

4. **Contributions must compose non-trivially.** The substrate requires coherence between contributions — not just accumulation. A shared codebase requires functions to interoperate. A knowledge base requires findings to be consistent. An append-only log does not — entries are independent.

5. **No structural coordination mechanism exists.** Nothing enforces that locally correct + locally correct = globally correct. No compiler catching conflicts, no schema enforcing consistency, no review process detecting redundancy.

When all five conditions hold, four dynamics emerge — at least in the two substrates we've measured (code and knowledge).

**Accretion:** individually correct contributions compose into structural degradation. Not a quality problem — a structural property of uncoordinated compositional systems.

**Attractors:** structural destinations that route contributions toward coherent organization. A well-named package pulls code toward it. A well-structured model of understanding pulls findings toward it.

**Gates:** enforcement that blocks wrong paths. Pre-commit hooks, build checks, required flags, schema validation — mechanisms where compliance is binary, not probabilistic.

**Entropy measurement:** detection of when composition is failing. Lines-per-file, orphan rates, duplication counts — metrics that make degradation visible before it becomes structural.

---

## Why Five, Not Four

The original formulation had four conditions. A systematic falsifiability probe — testing the theory against 15+ candidate counterexamples — revealed that without condition 4 (non-trivial composition), the theory over-predicts.

Append-only logs meet conditions 1-3 and 5. Multiple amnesiac agents write entries with no coordination mechanism. But logs don't degrade, because entries don't need to compose. Each entry is independent. Sensor data, voting systems, and sediment layers work the same way — additions stack without requiring coherence.

The addition of condition 4 prevents this over-prediction. It separates compositional substrates (codebases, knowledge bases, database schemas, API surfaces) from additive substrates (logs, sensor data, event streams) where contributions are independent and can't compose incorrectly.

The refined claim: these conditions produce accretion in shared mutable substrates where contributions must compose non-trivially — confirmed in two substrates so far (code and knowledge), with external evidence from human systems (scientific literature, corporate wikis, shared drives) consistent with the pattern. Remove the compositional requirement and the prediction fails — which is exactly what a falsifiable theory should do.

---

## Trying to Break It

A theory that explains everything explains nothing. So I tried to find systems that meet all five conditions but don't exhibit accretion. I tested 15+ candidates across three domains.

**Natural systems** were the most promising counterexamples. Ant colonies appear to meet all five conditions — multiple amnesiac agents making locally correct contributions to a shared substrate with no central coordination. But they produce coherent structures, not entropy.

The resolution: stigmergy. Pheromone trails are substrate-embedded coordination. The environment itself mediates between agents. Ants don't need persistent memory because the environment remembers for them. Condition 5 doesn't hold — coordination exists, it's just implicit in the substrate rather than explicit in rules.

The same resolution applies to termite mounds (cement pheromones), immune systems (cytokines, MHC presentation), and coral reefs (self-similar additions that don't require compositional coherence — condition 4 doesn't hold).

**Engineered systems** with apparent absence of coordination all turned out to have hidden coordination. CRDTs guarantee convergence through mathematical properties — the data type itself is the coordination mechanism. Blockchains have consensus protocols. Assembly lines have interface specifications. Each has coordination, just not the kind you notice at first glance.

**Human systems** that genuinely lack coordination exhibit exactly the predicted dynamics. Scientific literature: exponential paper growth, linear knowledge growth, replication crisis, duplicative systematic reviews (increased 2,728% vs 153% for all publications). Corporate wikis: orphan pages, stale docs, naming drift. Shared file systems: 85% of shared drive data is dark or ROT — Redundant, Obsolete, Trivial (Veritas, 2016). Wikipedia itself, despite extensive coordination mechanisms, still has ~15% orphan articles (~8.8 million pages with no incoming links).

Every candidate falls into one of three buckets: hidden coordination (conditions don't fully hold), trivial composition (additions don't need to cohere), or observed accretion (consistent with the theory). No clean counterexample survived — though a framework with categories this broad may be absorbing counterexamples through reclassification rather than genuine resilience. That's a limitation of post-hoc analysis.

This doesn't mean the theory is capital-T True. It means it's conditionally predictive — like Ostrom's commons principles or Conway's Law. It predicts where accretion will concentrate (at coordination gaps), what interventions will reduce it (gates at compositional boundaries), and that removing coordination will introduce it. It does not predict the form, rate, or exact threshold of accretion. Those are substrate-specific.

---

## Evidence: Code

I've run 50+ autonomous AI agent sessions per day on a single Go codebase (~47,600 lines, 125 files) for 12 weeks. Code has mature measurement tools, so the evidence is most detailed here.

**Accretion in the wild.** `daemon.go` grew from 667 to 1,559 lines in 60 days from 30 individually correct commits. Each added a locally reasonable capability. No single commit was wrong. The velocity accelerated: ~65 lines/week in month one, 345 lines/week by month three.

Separately, 6 cross-cutting concerns were independently reimplemented across 4-9 files each, producing ~2,100 lines of duplicated infrastructure. Workspace scanning alone was implemented five times by five agents unaware of each other's work.

**Attractors work, then fail without gates.** When we created `pkg/spawn/backends/`, it acted as a structural attractor — a destination that pulls code toward it through naming and import convention. `spawn_cmd.go` shrank from 2,432 to 677 lines. Then it regrew to 1,160 lines in 3 weeks. The gravitational center (the Cobra command definition) still lived in the original file. Extraction without routing is a pump — you can cool a room without insulation, it just heats back up.

**Soft instructions fail under pressure.** 265 contrastive trials across 7 agent skills measured how written instructions change behavior. Behavioral rules (MUST/NEVER) became inert at 10+ co-resident constraints. Our project charter stated a 1,500-line convention. The agents read it, understood it, and added 200 lines to an 1,800-line file anyway because the task was urgent and no gate blocked them.

**Every convention without a gate was violated.** This is the invariant that keeps confirming. Documentation conventions, commit message formats, file size limits — without mechanical enforcement, every single one degraded under production pressure.

---

## Evidence: Knowledge

If the same dynamics appear in a fundamentally different medium, the framework is diagnosing system properties, not substrate properties. Knowledge provides that second confirmation.

**Accretion is measurable in our system.** The system's 1,166 investigations have an 87.6% orphan rate — work product with no structural connection to any model. But this decomposes: the pre-model era (before the probe system existed) has a 94.7% orphan rate across 83% of the corpus. The model era rate is 52.0%.

A 35-file sample revealed that ~80% of orphans are naturally expected — implementation tasks miscategorized as investigations, one-off explorations, negative results. The actionable "genuinely lost" rate is ~10%, comparable to healthy dead code rates in mature codebases.

**The probe displacement.** When probes activated, investigation volume dropped 76% while probes rose to 160/month. The system shifted from producing disconnected findings to producing structurally-connected hypothesis tests. The orphan rate dropped from 94.7% to 52% — not through cleanup, but through architectural change that prevents orphaning at creation time.

This is the knowledge equivalent of introducing import statements. Before probes, investigations were functions defined in random files. After probes, findings are structurally coupled to the models they test — probes live in `.kb/models/{name}/probes/`, creating a directory-level connection between evidence and claim.

**Zero hard gates, measured consequences.** Every knowledge convention is violated at significant rates: 48% of investigations skip prior work citation. Probe-to-model merge is advisory only — contradiction verdicts accumulated unresolved until a batch cleanup. Quick entry deduplication has no automated checking (confirmed duplicates exist). 1 of 56 decisions has an enforcement check (1.8%).

The code substrate's invariant — "every convention without a gate will eventually be violated" — confirmed in the knowledge substrate across every measured transition.

**Three model behaviors.** Across 32 models: **attractors** that increase investigation density toward them after creation (the daemon model attracted 34 probes over 21 days — sustained gravitational pull), **capstones** that decrease density by settling a topic, and **dormant** models with zero ongoing engagement.

---

## The Compliance/Coordination Distinction

This is the sharpest claim and the one with the most practical consequence.

Agent governance addresses two failure modes that respond oppositely to model improvement:

**Compliance failure:** an agent doesn't follow instructions. Fixed by better models — smarter agents follow instructions more reliably.

**Coordination failure:** agents each follow instructions correctly but collectively produce entropy. Not fixed by better models. Made *worse* by faster, more confident agents that accrete more per session with higher confidence.

We tested this directly. Two AI agents, same task, same codebase, no communication. Simple task (add a `FormatBytes` function): 10 trials, 100% merge conflict rate. Both agents wrote correct code. Both appended to the same file location. Both generated near-identical commit messages independently. Fisher's exact test: p=1.0. The conflict rate was identical regardless of model capability.

Complex task (multi-file table renderer): the more capable model made *deeper* semantic conflicts. It anticipated Unicode edge cases and made more sophisticated design choices — producing implementations that were individually superior but collectively incompatible. In this single trial, greater capability created greater divergence — though one trial can't establish a general trend.

The `daemon.go` evidence is coordination failure at codebase scale. 30 correct commits, 892 lines of growth, workspace scanning implemented five times. No agent was wrong. The architecture was missing.

The analogy: a company of 30 brilliant engineers with no architecture review still produces spaghetti — possibly faster than 30 mediocre engineers, because each builds more in less time.

**This makes coordination infrastructure permanent, not transitional.** Compliance gates may simplify as models improve. Coordination gates become the primary investment as agents get more capable. The investigation/probe/model cycle is coordination infrastructure for knowledge. Models are structural destinations routing findings toward coherent understanding. Probes are coordination checks verifying new findings against existing structure. The cycle doesn't compensate for bad agents. It provides coordination that good agents cannot provide for themselves.

---

## The External Evidence

The dynamics aren't unique to my system. They're visible at industry scale:

**Knight Capital, 2012.** Developers repurposed a deprecated feature flag from 2003 code. One of eight servers didn't receive a deployment update. The dormant flag activated old trading logic — $460 million lost in 45 minutes. A locally correct change (repurpose unused flag) to a shared substrate (flag namespace) by an amnesiac writer (unaware of old server state) with no coordination mechanism (no flag lifecycle management). All five conditions held.

**Feature flags at scale.** 73% of feature flags are never removed. Average enterprise has 200-500 stale flags. The creation/removal cost asymmetry is universal: adding is always cheaper than removing, because removal requires coordinating with unknown dependents.

**Shared drives.** 85% of enterprise data on shared drives is either dark (value unknown, 52%) or ROT — Redundant, Obsolete, Trivial (33%). $3.3 trillion globally spent managing data nobody uses.

**Scientific literature.** Systematic reviews increased 2,728% while all publications grew 153%. Two-thirds of meta-analyses overlap with existing ones. Papers grow exponentially; knowledge grows linearly.

**Wikipedia.** Despite extensive coordination infrastructure (WikiProjects, bots, style guides, deletion processes), ~15% of articles are orphans with no incoming links. Accretion concentrates at the gaps in coordination coverage — exactly where the framework predicts.

Every one of these cases meets the five conditions when classified post-hoc. Every one exhibits dynamics consistent with the framework's predictions — though post-hoc classification is inherently weaker than forward prediction.

---

## The Creation/Removal Ratchet

One dynamic deserves special emphasis because it appears in every substrate studied.

Adding is always cheaper than removing. Adding a file, column, feature flag, API endpoint, investigation, or wiki page is a single-agent action. Removing one requires coordinating with unknown dependents. This asymmetry produces a ratchet: growth is easy, shrinkage requires coordination that amnesiac agents cannot provide.

73% of feature flags never removed. 85% of shared drive data untouched. 39% of enterprises cannot maintain an accurate API inventory (median enterprise manages 15,564 APIs). Once created, artifacts persist not because they're valuable but because removal coordination is expensive.

This ratchet alone may explain monotonic accretion even in systems with partial coordination. It's not that coordination is absent — it's that coordination sufficient for creation is insufficient for removal.

---

## What Kind of Theory Is This?

Not a law of physics. The name "knowledge accretion" is memorable shorthand — the body earns it through empirical evidence and honest scoping, not through claiming mathematical certainty.

This is a diagnostic framework in the tradition of Ostrom's design principles for commons governance (1990), Conway's Law (1967), and Brooks's Law (1975). Structural conditions, empirically derived, that predict outcomes without guaranteeing them.

**What it predicts:**
- Where accretion will concentrate (at gaps in coordination coverage)
- What interventions reduce it (gates at compositional boundaries, attractors providing structural destinations)
- That removing coordination will introduce accretion

**What it doesn't predict:**
- The exact form accretion takes in a new substrate
- The rate of degradation
- The threshold at which degradation becomes critical

The theory is most precisely expressed as a continuous risk model rather than binary conditions: `accretion_risk = f(amnesia_level x compositional_complexity / coordination_strength)`. This explains partial accretion in partially-coordinated systems and avoids debates about where binary thresholds fall.

---

## Coordination Comes From Three Sources

The falsifiability probe revealed an important taxonomy. Coordination isn't just "rules humans write." It comes from three sources:

**Explicit coordination:** Engineered rules and enforcement. Type systems, schemas, CI pipelines, code review, pre-commit hooks. This is what software teams build.

**Substrate-embedded coordination:** Mathematical or physical properties of the substrate that guarantee coherence by construction. CRDTs converge because the data type makes conflict impossible. Strongly-typed languages prevent certain classes of composition error through the type system.

**Environmental coordination:** The environment mediates between agents. Stigmergy — pheromone trails guiding ant behavior. Physical constraints shaping construction. Chemical gradients directing cellular behavior.

Digital substrates lack environmental coordination entirely. A `.go` file doesn't resist bloat through physics. A `.kb/` directory doesn't resist orphan investigations through chemistry. This is why digital substrates require *engineered* coordination while biological substrates often have implicit coordination built into the medium.

CRDTs are the notable exception — a digital substrate with coordination embedded in its mathematical structure. Whether type systems qualify as substrate-embedded coordination (and whether strongly-typed languages therefore exhibit less accretion than weakly-typed ones) is a testable prediction I haven't yet tested.

---

## Honest Gaps

**One system, one operator.** This framework was derived from one system operated by one person. The falsifiability probe tested it against external evidence, but no one else has applied the five-condition diagnostic to their own system and reported results. First external validation requires someone else running the investigation/probe/model cycle and observing whether accretion dynamics appear.

**Two confirmed substrates (plus one adversarial).** Code and knowledge are empirically confirmed. A third substrate — operational security — was confirmed in a separate project, extending the theory to adversarial substrates where entropy is invisible to internal measurement and failure is binary rather than gradual. But database schemas, config systems, API surfaces, and documentation are still theoretical predictions, not empirical confirmations.

**Knowledge gates are unproven.** The knowledge substrate has zero hard gates. Whether deploying them (e.g., requiring investigations to reference a model) would reduce the genuinely-lost rate is an active experiment, not a demonstrated result.

**Minimum time to visible dynamics.** This system shows clear dynamics after 1,166 investigations over months. A solo researcher generating 3-5 investigations per week may not see the compounding effect for weeks. The "magic moment" — when the system's value becomes obvious — may need acceleration through seeded examples for first-time users.

**Conditions 1-3 have low discriminating power.** The theory's predictive content comes almost entirely from conditions 4 (compositional complexity) and 5 (absent coordination). Conditions 1-3 describe the context — they're common in most modern systems. This means the theory's core content is: "compositional substrates without coordination degrade from locally correct contributions." The specific mechanism (local correctness composing into global degradation) and the specific remedy (gates at compositional boundaries) are the contributions. The five-condition formulation is a diagnostic checklist, not five independent variables with equal weight.

---

## Running It Yourself

The investigation/probe/model cycle separates cleanly into substrate (the knowledge system) and orchestration (infrastructure for running it at scale). Five components form the minimal substrate:

| Component | Role |
|-----------|------|
| AI agent runtime (Claude Code, Cursor, etc.) | Does the empirical work |
| `kb` CLI or equivalent | Artifact management, context retrieval |
| Git | Audit trail, version control |
| `.kb/` directory | The shared mutable substrate |
| Investigation skill/conventions | The cycle protocol |

**How to start:** Initialize a `.kb/` directory with `models/` and `investigations/` subdirectories. When you encounter a question worth recording, create an investigation. After 3+ investigations in the same area, create a model with testable claims. When working in a domain with an existing model, run a probe — test the claims against current evidence, record the verdict, update the model.

The compounding becomes visible after your first model is tested by probes. Before that, you're accumulating raw material. After that, you're building structural understanding that compounds across sessions.

**Who this is for:** A solo developer or researcher working on a complex long-running project with AI agents. Someone who forgets their own prior decisions and is frustrated by re-investigation. No team features, organizational buy-in, or training required. Adoption patterns from comparable tools — ADRs took 7 years from individual practice to industry standard, Obsidian's 1.5 million monthly users are predominantly individual — show that bottom-up adoption is stickier than top-down mandates.

---

*Based on three months operating a knowledge system: 1,166 investigations, 187 probes, 32 models, a parallel codebase (~47,600 lines of Go, 50+ agent sessions/day), 265 contrastive trials, 3 entropy spirals, 1,625 lost commits, 20 controlled coordination trials, and a falsifiability probe testing the theory against 15+ candidate counterexamples across natural, engineered, and human systems. The 32nd model describes the physics of the other 31.*
