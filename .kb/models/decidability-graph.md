# Model: Decidability Graph

**Domain:** Work Coordination / Authority Boundaries / Daemon Operation
**Last Updated:** 2026-01-19
**Synthesized From:** Strategic Orchestrator Model, Questions as First-Class Entities decision, daemon overnight run observations, Petri net / HTN / Active Learning analogies

---

## Summary (30 seconds)

A **decidability graph** encodes not just data dependencies ("B needs A's output") but **decision dependencies** ("B requires judgment that exceeds current authority"). Traditional workflow systems treat human decisions as external events. Decidability graphs make authority boundaries a first-class property of the work representation itself. This explains why daemon overnight runs hit walls - not from missing data, but from missing authority to traverse certain edges.

---

## Core Mechanism

### The Key Insight

Work graphs traditionally encode:
- **Nodes** = tasks
- **Edges** = "B depends on A's output"

Decidability graphs add a second dimension:
- **Node types** = what kind of resolution is needed
- **Edge authority** = who can traverse this edge

The daemon can traverse freely through Work→Work edges. When it hits a Question or Gate node, it must stop and surface - the edge requires authority the daemon doesn't have.

### Node Taxonomy

| Node Type | Characteristics | Daemon Behavior | Resolution Shape |
|-----------|-----------------|-----------------|------------------|
| **Work** | Executable with current context, resolution shape known | Traverse freely | Converges (task completes) |
| **Question** | Resolution uncertain, might branch/dissolve/reframe | Surface, don't resolve | Open (might fracture, collapse, or converge) |
| **Gate** | Judgment required, tradeoffs, irreversibility | Stop, accumulate options | Binary (decision made) |

### The Work/Question/Gate Distinction

**Work nodes** have implicit resolution shape. A bug gets fixed. A feature gets implemented. The "shape" of completion is known before starting.

**Question nodes** have open resolution space:
- Might get answered (converge to understanding)
- Might fracture into sub-questions (diverge)
- Might dissolve when reframed ("wrong question")
- Might reveal they were actually Gates

**Gate nodes** require judgment with consequences:
- Tradeoffs between options
- Irreversible choices
- Authority-specific decisions (only Dylan can decide X)

### Edge Authority

Edges carry authority requirements, not just data flow:

```
         ┌─────────────────────────────────────────────────┐
         │                                                 │
         │   Work ──daemon──▶ Work ──daemon──▶ Work       │
         │     │                                           │
         │     │ (question blocks)                         │
         │     ▼                                           │
         │  Question ──orchestrator──▶ Work               │
         │     │                                           │
         │     │ (gate blocks)                             │
         │     ▼                                           │
         │   Gate ────dylan────▶ Work                     │
         │                                                 │
         └─────────────────────────────────────────────────┘
```

**Authority levels:**
1. **Daemon** - Can traverse Work→Work edges. Spawns, monitors, completes within defined parameters.
2. **Orchestrator** - Can traverse Question edges. Scopes context for resolution.
3. **Dylan** - Can traverse Gate edges. Makes judgment calls with irreversible consequences.

### The Irreducible Function: Context Scoping

**Key insight (discovered 2026-01-19):** The hierarchy isn't about reasoning capability - workers can do any kind of reasoning (factual, design, even framing) IF they have the right context loaded. The irreducible orchestrator function is **deciding what context to load**.

| Old Model | Refined Model |
|-----------|---------------|
| Workers *can't* answer framing questions | Workers *don't have context* to answer framing questions |
| Orchestrator *does* synthesis | Orchestrator *scopes* what gets synthesized |
| Authority is role-based | Authority is context-scoping-based |
| Hierarchy about capability | Hierarchy about who scopes whom |

**Why "spawn architect to think for me" felt wrong:**
Not because architect can't think - but because orchestrator was abdicating the *scoping decision*. The question "what context does this need?" is the orchestrator's job. Once scoped, a worker can execute.

**The authority chain is about scoping:**
- Daemon: Executes pre-scoped work (context already defined by spawn)
- Orchestrator: Scopes context for workers (decides what frames/knowledge to load)
- Dylan: Scopes context for orchestrator (or overrides scoping decisions)

**Implication:** A "frame-evaluator" worker is possible - spawn with multiple frames as context, instruction to evaluate from outside. The orchestrator's contribution was *deciding those were the relevant frames to compare*.

**What remains irreducibly human (Dylan):**
- Overriding scoping decisions ("you're looking at the wrong thing")
- Value judgments that determine which frames matter
- Accountability for where the system points its attention

### Worker Authority Boundaries

**Key rule:** Workers can create nodes, only orchestrator can create blocking edges.

| Workers CAN | Workers CANNOT |
|-------------|----------------|
| Create any issue type (task, bug, feature, question) | Add dependencies to existing issues |
| Label tactical work `triage:ready` | Label strategic questions `triage:ready` |
| Label uncertain work `triage:review` | Close issues outside their scope |
| Surface questions in SYNTHESIS.md | Override orchestrator decisions |
| Make implementation decisions within scope | Create blocking relationships |

**The safety mechanism:** Triage labels. `triage:review` surfaces work without enabling daemon to act - orchestrator validates first. This allows workers to expand the graph (create nodes) while orchestrator controls graph constraints (blocking edges).

**Decision:** `.kb/decisions/2026-01-19-worker-authority-boundaries.md`

### Resolution Typing (Question Subtypes)

Not all questions are equal. Resolution type determines who can traverse:

| Question Subtype | Example | Who Resolves | How |
|------------------|---------|--------------|-----|
| **Factual** | "How does X work?" | Daemon (via investigation) | Spawn agent, answer surfaces |
| **Judgment** | "Should we use X or Y?" | Orchestrator | Synthesize tradeoffs, decide |
| **Framing** | "Is X even the right question?" | Dylan | Reframe the problem space |

A question's subtype may not be known at creation. "How does X work?" might reveal "X is the wrong abstraction" (factual→framing escalation).

### Graph Dynamics

Unlike static workflow DAGs, decidability graphs change during execution:

**Resolution effects:**

| Resolution Type | Graph Effect |
|-----------------|--------------|
| Question answered | Unblocks dependent subgraph |
| Question reframed | Collapses old subgraph, inserts new nodes |
| Question fractured | Inserts new question nodes as dependencies |
| Gate decided | Selects one branch, prunes alternatives |
| Work completed | Node marked done, edges traversable |

**The Petri net analogy:** Tokens (authority) flow through places (nodes). Transitions (edges) fire when the required token type is present. Daemon tokens can fire Work transitions. Question transitions require orchestrator tokens. Gate transitions require Dylan tokens.

### Frontier Representation

The daemon's output is a graph state report:

```
Frontier Report:
  Completed paths: [Work-A → Work-B → Work-C] ✓
  Ready frontier: [Work-D, Work-E] (daemon-traversable)
  Question-blocked: [Work-F, Work-G] waiting on Question-Q1
  Gate-blocked: [Epic-X] waiting on Gate-G1
  Questions surfaced: [Q1: "Should we refactor auth before adding feature?"]
  Gates accumulated: [G1: "Adopt event sourcing?" - options A, B, C gathered]
```

This is richer than current `orch status` which shows agent states, not graph topology.

---

## Why This Fails (Without Explicit Modeling)

### 1. Daemon Hits Walls Overnight

**What happens:** Daemon spawns all `triage:ready` issues, but by 3am has stalled on 4 agents, each waiting for decisions nobody made.

**Root cause:** Work was labeled ready when it actually depended on unresolved questions. The graph had Question nodes, but they weren't represented - just implicit in the work descriptions.

**Without the model:** "Daemon stalled" → add more issues → same pattern
**With the model:** "Question-blocked frontier" → surface questions first → unblock graph

### 2. Orchestrator Resolves Wrong Level

**What happens:** Orchestrator answers a question that was actually a Gate (Dylan's judgment needed). Decision gets made, work proceeds, Dylan later says "wait, I wouldn't have chosen that."

**Root cause:** Question vs Gate distinction wasn't explicit. Looked like orchestrator-level uncertainty, was actually authority-gated.

**Without the model:** "Misaligned decision" → more communication overhead
**With the model:** "Gate detected" → escalate before resolving

### 3. Questions Treated as Work

**What happens:** "Investigate whether we should do X" spawned as investigation. Agent produces findings. Nobody synthesizes. Question remains unresolved. Dependent work stays blocked.

**Root cause:** Conflating the investigation (Work) with the question it's meant to answer (Question). Investigation completing ≠ question answered.

**Without the model:** Investigation completes → confusion about what's unblocked
**With the model:** Investigation completes → question node transitions to "evidence gathered" → orchestrator synthesizes → question answered → work unblocked

### 4. Premature Work on Uncertain Premises

**What happens:** Epic created for "How do we X?" without validating whether X is correct. Architect later finds premise was wrong. Work wasted.

**Root cause:** No gate enforcing premise validation. The graph allowed traversal into Work nodes while Question nodes were still open.

**The Premise Before Solution violation:** `bd ready` showed issues as ready because they had no *data* dependencies. The *decision* dependency (premise validation) wasn't encoded.

---

## Constraints

### Why Can't Daemon Resolve Questions?

**Constraint:** Daemon can spawn investigations but cannot synthesize answers.

**Implication:** Questions block until orchestrator engages, even if all evidence exists.

**Why this is correct:**
- Synthesis requires cross-agent context (daemon doesn't have)
- Questions may reframe during resolution (daemon can't detect)
- Wrong answers worse than slow answers (Understanding Through Engagement principle)

**This enables:** Reliable batch processing within known bounds
**This constrains:** Questions create synchronization points that require orchestrator

---

### Why Can't Orchestrator Traverse Gates?

**Constraint:** Some decisions require Dylan's authority (irreversible, high-stakes, value-laden).

**Implication:** Gates accumulate options but don't resolve until Dylan engages.

**Why this is correct:**
- Irreversibility (can't undo architecture choices)
- Value judgments (what matters more: speed vs correctness?)
- Accountability (Dylan owns the system direction)

**This enables:** Options gathered efficiently, decision made with full picture
**This constrains:** Gate nodes are hard synchronization points with Dylan

---

### Why Is Resolution Shape Unknown for Questions?

**Constraint:** Questions can fracture, collapse, or reframe - can't predict the resulting graph structure.

**Implication:** Can't plan past a question node. Subgraph after question is provisional.

**Why this is fundamental:**
- "Should we do X?" might reveal "X is wrong framing"
- Single question might become 3 sub-questions
- Answer might collapse entire planned subgraph

**The Saga pattern connection:** Long-running processes need compensation and resumability. The graph after a question isn't committed until the question resolves.

**This enables:** Honest representation of uncertainty
**This constrains:** Can't create detailed plans past unresolved questions

---

## Integration Points

### With Beads

The `question` bead type is the entity representation of Question nodes:

```bash
bd create --type question --title "Should we adopt event sourcing?"
bd dep add <epic-id> <question-id>  # Epic depends on question
bd ready                             # Excludes question-blocked items
bd ready --type question             # Show only questions (for orchestrator)
```

**Verified behavior (dogfooded 2026-01-19):**
- Questions block dependent work (`bd blocked` shows them)
- Closing question unblocks dependent work (`bd ready` includes it)
- Questions excluded from default `bd ready` (correct - they're not daemon work)

**Current gaps:**
- `bd close` requires "Phase: Complete" but questions aren't agent work (need `--force`)
- `answered` status doesn't unblock dependencies - only `closed` does
- Decidability typing (factual/judgment/framing) not yet encoded

### With Daemon

Daemon already respects dependencies via `bd ready`. What's missing:
- Surfacing WHY work is blocked (question vs data dependency)
- Frontier reporting (what's daemon-traversable vs escalation-needed)
- Question subtype awareness (can this question resolve via investigation?)

### With Orchestrator Skill

The Strategic Orchestrator Model already encodes:
- Orchestrator does comprehension (resolves questions)
- Daemon does coordination (traverses work)
- Dylan provides perspective (resolves gates)

Decidability graph makes this structural, not just role guidance.

### With Dashboard

Current views: agent states, ready queue, blocked issues.
Decidability-aware views would add:
- **Frontier view:** What's immediately traversable vs blocked-on-what
- **Question view:** Open questions with blocking scope
- **Gate view:** Accumulated options awaiting Dylan

---

## Analogies from Other Domains

### Petri Nets

- Formal model for concurrency with explicit synchronization
- Tokens flow through places, transitions fire when enabled
- **Mapping:** Tokens = authority type, places = nodes, transitions = edges
- **Portable concepts:** Reachability (can we get to X?), deadlock detection (circular question dependencies), liveness (will this question ever resolve?)

### Hierarchical Task Networks (HTN)

- Tasks decompose into subtasks, decomposition is context-dependent
- **Mapping:** Questions = tasks that can't decompose until something resolves
- **Insight:** "Planning is part of the work" - you can't fully plan past questions

### Active Learning

- Formal notion of "uncertainty too high, query the oracle"
- **Mapping:** Daemon hitting question = recognizing competence boundary
- **Insight:** Yield rather than guess. Wrong answers are expensive.

### Saga Pattern

- Long-running processes with compensation and resumability
- **Mapping:** Graph state = "where we are", not just "what to do"
- **Insight:** Partial progress is first-class. Can checkpoint at question nodes.

### OODA Loops

- Observe, Orient, Decide, Act
- **Mapping:** Decide is distinct phase with own rhythm
- **Insight:** Graph encodes *where the D happens* - Gate nodes are decision points

---

## Evolution

**2026-01-19:** Initial model created. Synthesized from overnight daemon run observations, question beads implementation, and discussion of authority boundaries in orchestration.

**Prior work:**
- Strategic Orchestrator Model (2026-01-07): Established daemon/orchestrator/Dylan authority division
- Questions as First-Class Entities (2026-01-18): Created question bead type with blocking semantics
- Premise Before Solution principle: Identified need to gate work on premise validation

**What this model adds:**
- Explicit node typing (Work/Question/Gate)
- Authority as edge property
- Graph dynamics (resolution changing structure)
- Frontier as observable state
- Formal analogies (Petri nets, HTN, Active Learning)

**2026-01-19 (later):** Dogfooded model against current beads implementation. See "Empirical Validation" section.

**2026-01-19 (discussion):** Major refinement - discovered that hierarchy is about context-scoping, not reasoning capability. Workers can answer framing questions if given right context. The irreducible orchestrator function is deciding what context to load. See "The Irreducible Function: Context Scoping" section. Captured as `kb-227b01`.

---

## Empirical Validation (Dogfooding 2026-01-19)

Created three question beads to test decidability mechanics:

| Question | ID | Purpose |
|----------|-----|---------|
| How should question subtypes be encoded? | orch-go-7hd6h | Resolution typing |
| Is Question vs Gate distinction crisp? | orch-go-2yzjl | Boundary clarity |
| Can daemon resolve factual questions? | orch-go-iz4tb | Authority levels |

**Test procedure:**
1. Created question beads via `bd create --type question`
2. Created dependent task via `bd create --type task`
3. Added dependency via `bd dep add <task> <question>`
4. Verified task blocked (`bd blocked` showed it, `bd ready` excluded it)
5. Closed question via `bd close --force`
6. Verified task unblocked (appeared in `bd ready`)

**What worked:**
- Question type accepted by beads
- Dependency mechanics work (question blocks task)
- `bd ready` correctly excludes question-blocked work
- `bd ready --type question` shows questions separately
- Closing question unblocks dependent work

**Friction discovered:**

| Issue | Constraint Created |
|-------|-------------------|
| `bd close` requires "Phase: Complete" for all types | `kb-dc4a2e` |
| `answered` status doesn't unblock (only `closed` does) | `kb-fe6173` |

**Implication for model:**
The Question lifecycle (Open → Investigating → Answered → Closed) isn't fully wired into dependency resolution. Currently only `closed` status unblocks. The `answered` intermediate state exists but has no mechanical effect.

**Recommendation:** Either wire `answered` to unblock dependencies, or simplify lifecycle to Open → Closed with close reason capturing the answer.

---

## Open Questions

1. **Resolution typing implementation:** How should question subtypes (factual/judgment/framing) be encoded? Labels? Separate field? Inferred from context?

2. **Graph visualization:** What would a decidability graph look like in the dashboard? How to show authority edges?

3. **Automatic escalation detection:** Can the system detect when a factual question has become a framing question? (Evidence gathered but no convergence)

4. **Gate accumulation:** How should options be gathered at Gate nodes? Current pattern is ad-hoc. Should there be structure?

5. **Subgraph provisioning:** When work is created past a Question, how explicit should the "provisional" status be?

---

## References

**Principles:**
- `~/.kb/principles.md` - Gate Over Remind, Premise Before Solution, Perspective is Structural

**Decisions:**
- `.kb/decisions/2026-01-19-worker-authority-boundaries.md` - Workers create nodes, orchestrators create edges
- `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question bead type
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Authority division (with 2026-01-19 refinement)
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Why orchestrator resolves questions

**Related Models:**
- `.kb/models/daemon-autonomous-operation.md` - How daemon traverses work
- `.kb/models/completion-lifecycle.md` - Work node resolution
- `.kb/models/orchestrator-session-lifecycle.md` - Orchestrator engagement patterns

**Conceptual Sources:**
- Petri nets (Carl Adam Petri, 1962)
- Hierarchical Task Networks (Erol, Hendler, Nau, 1994)
- Active Learning (query strategies for ML)
- Saga pattern (distributed transactions)
- OODA loops (John Boyd)
