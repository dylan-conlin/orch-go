<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation recommendations should be classified at three levels (Implementation/Architectural/Premise) mapping to existing authority boundaries (Worker/Orchestrator/Dylan), with architects generating gate-blocking questions as a distinct "Question Generation" phase that precedes design synthesis.

**Evidence:** Analyzed decidability-graph.md model (Work/Question/Gate taxonomy), worker-authority-boundaries decision (workers create nodes, orchestrators create edges), and existing question subtype encoding (factual/judgment/framing labels).

**Knowledge:** The "what questions should we be asking" phase is a distinct output of architect work - surfacing strategic unknowns (Questions) and gate-level decisions (Gates) that must be resolved before implementation proceeds. This completes the loop: Architect surfaces → Orchestrator/Dylan resolves → Design unblocked.

**Next:** Create decision record codifying the three-level recommendation classification and question generation as explicit architect output.

**Promote to Decision:** Actioned - decision exists (recommendation-authority-classification)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Design Decision Authority Flow and Question Generation

**Question:** How should investigation/architect recommendations be classified by authority level, and how should the architect workflow include explicit question generation for gate-level decisions?

**Started:** 2026-01-30
**Updated:** 2026-01-30
**Owner:** og-arch-design-decision-authority-30jan-eb8a
**Phase:** Complete
**Next Step:** None - ready for decision record creation
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** `.kb/decisions/2026-01-19-worker-authority-boundaries.md` - Extends worker authority to include recommendation classification
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Decidability Graph Already Defines Authority Levels for Graph Traversal

**Evidence:** The decidability-graph.md model establishes three authority levels for traversing edges in the work graph:

| Authority | Can Traverse | Examples |
|-----------|--------------|----------|
| Daemon | Work→Work edges | Bug fixes, tactical tasks with clear scope |
| Orchestrator | Question edges | Judgment calls, synthesis across agents |
| Dylan | Gate edges | Irreversible choices, value judgments, strategic direction |

The model also defines question subtypes (`subtype:factual`, `subtype:judgment`, `subtype:framing`) that determine who can resolve them:
- Factual → Daemon can spawn investigation to answer
- Judgment → Orchestrator synthesizes tradeoffs
- Framing → Dylan reframes the problem space

**Source:** `.kb/models/decidability-graph.md:26-75` (Node Taxonomy, Edge Authority sections)

**Significance:** The authority model for *traversing* the graph already exists. What's missing is an authority model for *investigation outputs* - how should recommendations from investigations be classified so they flow to the right resolver?

---

### Finding 2: Worker Authority Boundaries Define Node vs Edge Creation

**Evidence:** The worker-authority-boundaries decision establishes:
- **Workers CAN:** Create any issue type, label tactical work `triage:ready`, label uncertain work `triage:review`, surface questions in SYNTHESIS.md
- **Workers CANNOT:** Add dependencies to existing issues, close issues outside scope, override orchestrator decisions, label strategic questions `triage:ready`

The key rule: "Workers expand the graph (create nodes), orchestrators constrain it (create blocking edges)."

**Source:** `.kb/decisions/2026-01-19-worker-authority-boundaries.md:19-43`

**Significance:** Investigation recommendations are a form of "expanding the graph" - they propose new nodes (decisions, questions, gates). The question is: should workers classify *what kind* of node they're proposing so the right authority can act on it?

---

### Finding 3: "Promote to Decision" Field Has No Authority Classification

**Evidence:** The investigation template has a "Promote to Decision" field with values: `recommend-yes`, `recommend-no`, `unclear`. But this field doesn't classify *what kind* of decision or *who* should make it.

From the meta-failure investigation:
- 107 investigations with "recommend-no" vs ~10 with "recommend-yes"
- No differentiation between implementation decisions (worker), architectural decisions (orchestrator), and strategic decisions (Dylan)
- `kb reflect --type promote` only checks kn entries, not investigation files

**Source:** `.kb/investigations/2026-01-14-inv-meta-failure-decision-documentation-gap.md:99-140`

**Significance:** Without authority classification, all recommendations go to the same queue (orchestrator review) regardless of whether they're implementation details the worker could decide or strategic questions requiring Dylan's input. This creates unnecessary escalation for simple decisions and may bury critical decisions in the same queue as tactical ones.

---

### Finding 4: Architect Skill Doesn't Define Question Generation as Explicit Output

**Evidence:** The architect skill defines four phases:
1. Problem Framing
2. Exploration
3. Synthesis (fork navigation)
4. Externalization (investigation artifact + feature list review)

Phase 3 includes "Fork Navigation" which identifies decision points, but there's no explicit step to:
1. Classify forks by authority level (worker/orchestrator/Dylan)
2. Generate Question entities for unresolved forks
3. Generate Gate entities for Dylan-level decisions

The skill says to navigate forks with recommendations, but doesn't specify what to do when a fork *cannot* be navigated without higher authority input.

**Source:** Architect skill SKILL.md (Phase 3: Synthesis section) - fork navigation protocol doesn't include authority classification

**Significance:** Architects identify decision forks but don't systematically classify them by authority or create the Question/Gate entities that would block dependent work until resolved. This is the "what questions should we be asking" phase gap.

---

## Synthesis

**Key Insights:**

1. **Recommendation Authority Classification Maps to Existing Graph Model** - The decidability graph already defines three authority levels (Daemon, Orchestrator, Dylan). Investigation recommendations should classify into corresponding levels: Implementation (worker decides within scope), Architectural (orchestrator decides with cross-agent context), and Strategic/Premise (Dylan decides with value judgment). This isn't a new model - it's applying the existing traversal authority model to investigation outputs.

2. **Question Generation is the Missing Architect Phase** - The architect skill identifies decision forks but doesn't systematically generate Question/Gate entities when forks cannot be navigated. The "what questions should we be asking" phase should be explicit: after fork identification, classify each unnavigable fork as either a Question (needs investigation/synthesis) or a Gate (needs Dylan's judgment), then create the corresponding beads entity to block dependent work.

3. **Workers Should Classify, Orchestrators Should Route** - Following the "workers create nodes, orchestrators create edges" rule: workers/architects should *classify* their recommendations by authority level (metadata), but orchestrators create the actual blocking relationships. This gives orchestrators visibility into what kind of decisions are pending without workers overstepping into graph-constraining authority.

**Answer to Investigation Question:**

**Part 1: How should investigation recommendations be classified by authority?**

Use a three-level classification that maps to the decidability graph authority model:

| Level | Maps To | Who Decides | Examples | Template Field |
|-------|---------|-------------|----------|----------------|
| **Implementation** | Work traversal | Worker (within scope) | Threshold values, naming, code structure, edge case handling | `authority:implementation` |
| **Architectural** | Question traversal | Orchestrator | Cross-component design, tradeoff synthesis, pattern selection | `authority:architectural` |
| **Strategic** | Gate traversal | Dylan | Premise questions, value judgments, irreversible direction | `authority:strategic` |

Classification criteria (from existing decision-authority guide):
- **Implementation:** Reversible, single-scope, clear criteria, no user-facing impact
- **Architectural:** Cross-boundary, multiple valid approaches, requires synthesis
- **Strategic:** Irreversible, resource commitment, ambiguous tradeoffs, premise-level

**Part 2: How should architects block on gate-level decisions?**

Architects should:
1. **Identify unnavigable forks** during Phase 3 (Synthesis) - forks where substrate doesn't provide enough context to recommend
2. **Classify the blocker** - Is this a Question (needs investigation/context) or Gate (needs Dylan's judgment)?
3. **Create beads entity** - `bd create --type question "..." -l authority:architectural` or `-l authority:strategic`
4. **Document in investigation** - Note that design is blocked on specific question/gate with ID reference
5. **Set investigation status** - `Status: BLOCKED - pending {question-id}` until resolved

The orchestrator then:
1. Reviews blocked architects
2. For Questions: Decides if it can synthesize answer or needs to spawn investigation
3. For Gates: Routes to Dylan with options accumulated from architect work

**Part 3: What is the "what questions should we be asking" phase?**

This is a **Question Generation Phase** that should be explicit in architect work:

```
Problem Framing → Exploration → Fork Identification → QUESTION GENERATION → Synthesis → Externalization
                                       ↓
                              For each fork:
                              ├── Can navigate with substrate? → Add to Synthesis
                              └── Cannot navigate? → Classify authority:
                                  ├── Needs more context? → Question entity
                                  └── Needs value judgment? → Gate entity
```

**Outputs of Question Generation Phase:**
1. **Question entities** for forks requiring investigation or synthesis
   - Factual subtype: "How does X work?" (daemon-spawnable investigation)
   - Judgment subtype: "Should we use X or Y?" (orchestrator synthesis)
2. **Gate entities** for forks requiring Dylan's judgment
   - Framing subtype: "Is X even the right question?" (Dylan reframes)
3. **Blocking relationships** documented (orchestrator creates actual deps)

This phase makes the architect's uncertainty explicit and actionable. Instead of producing a design with hidden assumptions or "it depends" cop-outs, the architect surfaces precisely what must be resolved before the design can be finalized.

**Evidence supporting this design:**
- Finding 1: Authority model exists for traversal, needs extension to outputs
- Finding 2: Workers can create nodes, classification is metadata on those nodes
- Finding 3: Current "Promote to Decision" lacks authority granularity
- Finding 4: Architect skill has fork navigation but no question generation

---

## Structured Uncertainty

**What's tested:**

- ✅ Decidability graph defines three authority levels for traversal (verified: read decidability-graph.md:26-75)
- ✅ Worker authority boundaries restrict workers from creating blocking edges (verified: read decision 2026-01-19)
- ✅ Question subtype encoding exists via labels (verified: `subtype:{factual|judgment|framing}` in CLAUDE.md)
- ✅ "Promote to Decision" field lacks authority granularity (verified: meta-failure investigation findings)

**What's untested:**

- ⚠️ Whether workers will correctly classify authority levels (requires skill update and agent testing)
- ⚠️ Whether Question Generation phase produces useful outputs (requires architect sessions with new phase)
- ⚠️ Whether orchestrator queue is helped by authority classification (usage pattern, not validated)
- ⚠️ Whether `authority:` labels integrate smoothly with existing `subtype:` labels

**What would change this:**

- If workers consistently misclassify authority levels, the classification may need to be orchestrator-only
- If Question Generation phase adds too much overhead to architect work, it may need to be optional
- If authority labels conflict with question subtypes, the encoding may need redesign
- If Dylan doesn't want this level of granularity routed to him, Gate classification may need adjustment

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Label-Based Authority Classification with Question Generation Phase** - Extend existing label conventions to include `authority:{implementation|architectural|strategic}` and add explicit Question Generation phase to architect skill.

**Why this approach:**
- Reuses existing beads label infrastructure (no schema changes, per question subtype pattern)
- Aligns with decidability graph authority levels (consistent mental model)
- Makes architect uncertainty explicit and actionable (surfaces hidden assumptions)
- Gives orchestrators visibility into pending decision types (better routing)

**Trade-offs accepted:**
- Label proliferation - another dimension on issues (acceptable: labels are flexible by design)
- Skill complexity - architect skill gets a new phase (acceptable: explicit > implicit uncertainty)
- Classification burden on workers/architects (mitigated: clear criteria from decision-authority guide)

**Implementation sequence:**
1. **Update investigation template** - Replace `Promote to Decision: recommend-yes/no` with `Recommendation Authority: implementation/architectural/strategic` field
2. **Update architect skill** - Add explicit Question Generation phase between Fork Identification and Synthesis
3. **Create decision record** - Codify the authority classification criteria and question generation protocol
4. **Update kb reflect** - Add `--type investigation-authority` to surface recommendations by authority level

### Alternative Approaches Considered

**Option B: Orchestrator-Only Classification**
- **Pros:** Workers don't need to learn classification criteria; avoids misclassification
- **Cons:** Loses worker context (they understand their findings best); creates orchestrator bottleneck
- **When to use instead:** If workers consistently misclassify, fall back to orchestrator review of all recommendations

**Option C: Question/Gate Entities Only (No Authority Label)**
- **Pros:** Simpler - just create entities, don't add metadata
- **Cons:** Loses the distinction between Architectural (orchestrator) and Strategic (Dylan) within entities; all Questions look the same
- **When to use instead:** If three-level classification proves too complex, simplify to binary (worker vs escalation)

**Option D: Implicit Classification via Entity Type**
- **Pros:** Entity type encodes authority - `task` is worker, `question` is orchestrator, custom `gate` type is Dylan
- **Cons:** Requires new beads type; conflates entity type with authority; `question` entities can be factual (daemon-level)
- **When to use instead:** If label-based approach becomes unwieldy

**Rationale for recommendation:** Label-based classification follows the established pattern from question subtype encoding (`subtype:factual/judgment/framing`). Adding `authority:implementation/architectural/strategic` gives another dimension without schema changes. The Question Generation phase makes architect uncertainty explicit - surfacing "I need X resolved before I can recommend" rather than hiding it in "it depends" hedges.

---

### Implementation Details

**What to implement first:**
1. **Decision record** - Codify authority classification and question generation protocol (immediate, provides foundation)
2. **Investigation template update** - Replace `Promote to Decision` with `Recommendation Authority` field (enables classification)
3. **Architect skill update** - Add Question Generation phase guidance (enables better architect outputs)
4. **kb reflect enhancement** - Add `--type investigation-authority` to surface recommendations by level

**Things to watch out for:**
- ⚠️ Don't confuse `authority:` labels with `subtype:` labels - authority is about who decides, subtype is about resolution shape
- ⚠️ Question Generation phase should NOT block synthesis - it runs in parallel, surfacing what can't be navigated
- ⚠️ Workers classifying as `authority:implementation` might over-use this to avoid escalation - monitor for this pattern
- ⚠️ The existing decision-authority guide has detailed criteria - leverage it rather than duplicating

**Areas needing further investigation:**
- How should orchestrator dashboard surface authority-classified recommendations? (view design)
- Should `orch frontier` include authority-level breakdown? (integration question)
- What happens when classification is disputed? (conflict resolution process)
- How does this interact with the decision gate (`blocks:` in decision frontmatter)? (integration)

**Success criteria:**
- ✅ Investigation template has `Recommendation Authority` field with clear criteria
- ✅ Architect skill documents Question Generation phase with explicit outputs
- ✅ At least one architect session uses new protocol (dogfooding)
- ✅ `kb reflect --type investigation-authority` surfaces recommendations by level
- ✅ Dylan can filter for strategic-level decisions requiring his input

---

## References

**Files Examined:**
- `.kb/models/decidability-graph.md` - Authority levels for graph traversal (Work/Question/Gate taxonomy)
- `.kb/decisions/2026-01-19-worker-authority-boundaries.md` - Workers create nodes, orchestrators create edges
- `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question entity type and gate mechanics
- `.kb/guides/decision-authority.md` - Criteria for agent vs orchestrator vs human decisions
- `.kb/investigations/2026-01-14-inv-meta-failure-decision-documentation-gap.md` - "Promote to Decision" tooling gap
- `~/.kb/principles.md` - Authority is Scoping, Escalation is Information Flow principles
- `~/.claude/skills/worker/architect/SKILL.md` - Current architect phases (no explicit question generation)

**Commands Run:**
```bash
# Create investigation file
kb create investigation design-decision-authority-flow-question

# No additional commands - this was primarily design work via document analysis
```

**External Documentation:**
- N/A - Design based on existing internal models

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-19-worker-authority-boundaries.md` - This investigation extends
- **Model:** `.kb/models/decidability-graph.md` - Authority model this builds on
- **Decision:** `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question entity pattern
- **Decision:** `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md` - Label encoding pattern
- **Investigation:** `.kb/investigations/2026-01-14-inv-meta-failure-decision-documentation-gap.md` - Identified promotion gap

---

## Investigation History

**2026-01-30 (start):** Investigation started
- Initial question: How should recommendations be classified by authority? How should architects block on gates? What is "question generation"?
- Context: Spawned as architect task to design authority flow in investigation/architect workflow

**2026-01-30 (analysis):** Reviewed existing authority models
- Found decidability graph defines traversal authority
- Found worker authority boundaries decision defines node vs edge creation
- Identified gap: no authority classification for investigation outputs

**2026-01-30 (design):** Synthesized three-part design
- Part 1: Label-based authority classification (`authority:implementation/architectural/strategic`)
- Part 2: Architect gate-blocking via Question/Gate entity creation
- Part 3: Question Generation as explicit architect phase

**2026-01-30 (complete):** Investigation completed
- Status: Complete
- Key outcome: Three-level recommendation classification mapping to decidability graph, with Question Generation as explicit architect output phase
