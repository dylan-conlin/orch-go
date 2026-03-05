## Summary (D.E.K.N.)

**Delta:** Coordination plans decompose into three distinct artifacts with different persistence needs: the graph structure (beads issues + deps), the strategic narrative (a new `.kb/plans/` artifact type), and the cross-project binding (enriched ECOSYSTEM.md sections). A session-end gate + `orch orient` consumption completes the loop.

**Evidence:** Cross-referenced decidability graph model, ATC audit (Finding 3.1 — debrief pipeline is biggest gap), session debrief probe (interactive sessions have no durable comprehension artifact), beads integration guide (cross-project is "epic in primary repo, ad-hoc in secondary"), existing orient data sources (facts only, no comprehension), focus.json (single global goal, no phased plan concept).

**Knowledge:** The plan IS a decidability graph expressed in natural language. Beads already captures Work/Question/Gate nodes and blocking edges. What's missing is the *coordination intent* — the phasing rationale, cross-project awareness, and strategic narrative that explains WHY this sequence. This is the "comprehension" gap the debrief probe identified: facts without meaning.

**Next:** Implement `.kb/plans/` artifact type, `orch plan` command (create/show/consume), session-end gate for plan externalization, and `orch orient` integration to inject active plans into session start.

**Authority:** architectural — Creates new artifact type, touches spawn/orient/completion infrastructure, establishes cross-project coordination pattern

---

# Investigation: Design — How Orchestrator Coordination Plans Persist Across Sessions

**Question:** When an orchestrator produces a phased execution plan with Work nodes, blocking edges, cross-project boundaries, and phasing gates, how should that plan persist so the next orchestrator can consume it without re-deriving the sequencing?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-vptji)
**Phase:** Complete
**Next Step:** None — create implementation issues for recommended approach
**Status:** Complete

**Patches-Decision:** N/A (new artifact type)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/decidability-graph/model.md` | Source (plans ARE decidability graphs) | Yes — Work/Question/Gate taxonomy confirmed | None |
| `.kb/investigations/2026-02-28-inv-atc-lens-feature-audit.md` | Extends (Finding 3.1: debrief pipeline is biggest ATC gap) | Yes — post-flight knowledge capture confirmed missing | None |
| `.kb/models/orchestrator-session-lifecycle/probes/2026-02-28-probe-session-debrief-artifact-design.md` | Extends (interactive sessions have no durable comprehension artifact) | Yes — gap confirmed between facts, tactical memory, and comprehension | None |
| `.kb/investigations/2026-03-05-inv-design-systematic-mapping-decidability-graph.md` | Extends (authority boundary enforcement mapping) | Yes — 22 boundaries catalogued | None |
| `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` | Confirms (question beads block dependent work) | Yes — question gating mechanics verified | None |

---

## Findings

### Finding 1: A Coordination Plan Decomposes Into Three Distinct Parts With Different Persistence Needs

**Evidence:** Analyzing the real cross-project orchestration example (Toolshed/PriceWorks):

```
Phase 1: Wire PriceCurvePanel (Toolshed, no blockers)
Phase 2: Forward simulation (PW geometry storage BLOCKS this → PW API → Toolshed UI)
Phase 3: Strategic landscape (PW landscape API + inverse solver → Toolshed dashboard)
```

This contains three distinguishable things:

1. **Graph structure** — Work nodes (Wire PriceCurvePanel, PW geometry storage, Forward simulation API, etc.), blocking edges (geometry BLOCKS simulation), and cross-project node locations (Toolshed vs PriceWorks). This is *mechanical* — it's the shape of the work.

2. **Strategic narrative** — "Phase 2 is gated on geometry because simulation is meaningless without real dimensions" and "Phase 3 requires both PW APIs to converge before Toolshed can build the dashboard view." This is *comprehension* — the WHY behind the sequence, the reasoning that would be expensive to re-derive.

3. **Cross-project binding** — Which repos are involved, what work lives where, what interfaces cross the boundary. This is *coordination topology* — the map of where graph nodes physically reside.

Each needs different persistence:

| Part | Persistence Need | Why Separate |
|------|-----------------|--------------|
| Graph structure | Survives session, queryable, blocks work | Already has a home: beads issues + deps |
| Strategic narrative | Survives session, human-readable, aids comprehension | No current home — evaporates with session |
| Cross-project binding | Survives session, guides routing | Partially exists in ECOSYSTEM.md |

**Source:** Task description (real Toolshed/PriceWorks example), decidability graph model (node/edge taxonomy), beads integration guide (dependency mechanics)

**Significance:** The design must address all three parts separately because they have different consumers. The daemon consumes graph structure (what's ready?). The orchestrator consumes strategic narrative (why this sequence?). The spawn system consumes cross-project binding (where does this work live?).

---

### Finding 2: Beads Already Captures Graph Structure — The Gap Is Comprehension

**Evidence:** Beads issues with dependencies already encode the decidability graph:

```bash
# Work nodes
bd create "Wire PriceCurvePanel" --type task -l triage:ready
bd create "PW geometry storage" --type task -l project:price-watch
bd create "Forward simulation API" --type task

# Blocking edges
bd dep add <simulation-api> <geometry-storage>   # simulation blocked by geometry
bd dep add <toolshed-ui> <simulation-api>         # UI blocked by API

# Question nodes
bd create "Can we use existing PW dimension API?" --type question -l subtype:factual
bd dep add <simulation-api> <dimension-question>  # API blocked by question
```

`bd ready` would correctly show only Phase 1 work. `bd blocked` would show Phase 2/3 work with their blocking reasons. The daemon would process Phase 1, hit Phase 2's question node, surface it for orchestrator.

**What's missing:** After creating all these issues, a new orchestrator session sees a flat list of issues with dependencies. It can reconstruct THAT something blocks something else, but not WHY the plan was structured this way, what the phases mean, or what the strategic intent is. The graph structure is preserved; the planning intelligence that produced the graph is lost.

This maps precisely to the session debrief probe's finding (2026-02-28): the gap is between *facts* (what `orch orient` provides — throughput, ready work, model state) and *comprehension* (why work matters, how threads connect).

**Source:** Beads integration guide (dependency mechanics), session debrief probe (facts vs comprehension gap), decidability graph model (node types)

**Significance:** The solution must NOT reinvent beads' graph mechanics. Beads works. What's needed is a *companion artifact* that carries the strategic narrative alongside the graph. The graph is the skeleton; the narrative is the muscle that explains how it moves.

---

### Finding 3: Cross-Project Coordination Is Currently Ad-Hoc — The Binding Layer Is Missing

**Evidence:** From `kb context`:
> Cross-project epics use Option A: epic in primary repo, ad-hoc spawns with --no-track in secondary repos, manual bd close with commit refs (`kn-43aa5e`)

The ECOSYSTEM.md provides a static registry of repos and their relationships, but has no dynamic coordination state. Focus.json tracks a single global goal — no phased plan concept. The `orch orient` command reads from the current project's `.kb/` and `bd ready`, not cross-project.

For the Toolshed/PriceWorks example:
- PriceCurvePanel work lives in Toolshed's beads
- Geometry storage work lives in PriceWorks' beads
- The blocking relationship between them (geometry BLOCKS simulation) can't be expressed in a single beads DB — they're in different repos
- The orchestrator must mentally track which beads DB to query for which phase

**Source:** kb quick entry `kn-43aa5e`, ECOSYSTEM.md content, pkg/focus/focus.go (single focus goal), pkg/orient/orient.go (project-scoped data sources)

**Significance:** Cross-project plan persistence requires a mechanism above the beads layer. Beads is per-project by design (and that's correct — multi-repo hydration was tried and abandoned as dangerous). The coordination layer that binds cross-project work together must live in a project-independent location (`~/.orch/` or `.kb/`).

---

### Finding 4: `orch orient` Is the Natural Consumption Point — But Currently Reads Only Facts

**Evidence:** From reading `pkg/orient/orient.go`, orient assembles:
- Throughput metrics (from `~/.orch/events.jsonl`)
- Previous session debrief (from `.kb/sessions/`)
- Ready issues (from `bd ready`)
- Model freshness (from `.kb/models/`)
- Focus goal (from `~/.orch/focus.json`)

The `PreviousSession *DebriefSummary` field was recently added (per the debrief probe design), which means orient already has the concept of "what happened before." But it doesn't have "what's the plan" — the forward-looking strategic intent.

Orient's data sources are all backward-looking (what happened) or present-tense (what's ready). There's no forward-looking source that says "here's where we're heading and why this is the sequence."

**Source:** `pkg/orient/orient.go`, `cmd/orch/orient_cmd.go`

**Significance:** Adding an `ActivePlan *PlanSummary` field to `OrientationData` would complete the fact→comprehension→direction progression. Orient would tell the orchestrator: "Here's what happened (debrief), here's what's ready (facts), and here's where we're heading (plan)."

---

### Finding 5: Session Debrief Already Captures Plan-Like Comprehension — But Inconsistently

**Evidence:** The session debrief template (`.kb/sessions/TEMPLATE.md`) includes:
- "What's Next" — 1-3 proposed threads for next session
- "What's In Flight" — active agents, pending review, open questions

Looking at the actual debrief (`2026-03-04-debrief.md`), "What's Next" contains:
```
1. [P2] Phase 5: Audit and migrate 10 orch-knowledge beads issues to orch-go
2. [P2] Model drift: Agent Lifecycle
3. [P2] Model drift: Agent Orchestration
```

This is plan-like but lacks: phasing rationale, blocking relationships, cross-project awareness, and the strategic narrative explaining WHY this sequence. It's a prioritized list, not a coordination plan.

**Source:** `.kb/sessions/TEMPLATE.md`, `.kb/sessions/2026-03-04-debrief.md`

**Significance:** The debrief's "What's Next" is the embryonic form of a coordination plan. The gap is evolving it from a prioritized list to a phased plan with blocking relationships and strategic rationale. This could be a natural promotion path: debrief "What's Next" → coordination plan when the sequence has enough structure.

---

## Synthesis

**Key Insights:**

1. **Plans decompose into graph + narrative + binding** — The graph structure (Work/Question/Gate nodes with blocking edges) goes to beads. The strategic narrative (phasing rationale, sequencing logic, cross-project awareness) needs a new artifact type. The cross-project binding (which repos hold which graph nodes) enriches the narrative with routing information. Trying to force all three into beads would violate "Evolve by Distinction" — they have different consumers, different lifecycles, and different update frequencies.

2. **The plan artifact is a COMPANION to beads, not a replacement** — This follows the existing three-layer architecture pattern (beads tracks work, kb persists knowledge, workspace holds ephemeral state). A coordination plan is durable knowledge about work structure — it belongs in `.kb/`, not `.beads/` (which tracks the work itself) and not `.orch/` (which is infrastructure state). The plan references beads IDs but doesn't duplicate issue tracking.

3. **`orch orient` completes the comprehension loop** — The pattern becomes: orchestrator produces plan → plan creates beads graph + kb narrative → session ends → next session starts → `orch orient` reads plan narrative + beads graph state → orchestrator has full context without re-derivation. This is the ATC "briefing→flight→debrief→update briefings" cycle the ATC audit identified as the biggest gap.

4. **Behavioral grammars say the gate must be automated** — "Manual discipline will leak" (behavioral grammars Claim 3). If plan externalization is a skill instruction ("remember to write down your plan"), it will be skipped under cognitive load. The gate should be at session end: did the orchestrator produce a plan artifact for any multi-session, cross-project, or phased work?

5. **Cross-project binding is the hardest part — and the least important to automate first** — Beads is per-project by design. Cross-project plan persistence requires a manual binding step: the plan narrative names the repos and references cross-project beads IDs. Automating this (e.g., `bd dep add --cross-project`) is a future evolution, not a prerequisite. The manual approach already works for cross-project epics (`kn-43aa5e`).

**Answer to Investigation Question:**

A coordination plan persists as:

1. **Graph structure** → Beads issues with dependencies (already works)
2. **Strategic narrative** → New `.kb/plans/{date}-plan-{slug}.md` artifact with phasing rationale, blocking logic, and cross-project binding
3. **Enforcement** → Session-end advisory gate (completion pipeline checks for plan artifact when work spans phases/projects)
4. **Consumption** → `orch orient` reads active plans and injects summary into session start; `orch plan show` provides detailed view

The plan artifact is a kb artifact, not a beads entity. Beads tracks what work exists and what blocks what. The plan explains WHY the work is structured this way and HOW it should be sequenced. Conflating these would mean either beads carries prose narratives (wrong) or kb tracks work status (wrong).

---

## Structured Uncertainty

**What's tested:**

- ✅ Beads dependency mechanics work for encoding plan graph structure (verified: decidability graph dogfooding 2026-01-19, question blocking 2026-01-18)
- ✅ Orient already has `PreviousSession` field for backward-looking comprehension (verified: read orient.go struct)
- ✅ Session debriefs capture embryonic plan-like content in "What's Next" (verified: read 2026-03-04-debrief.md)
- ✅ Cross-project beads operations work via `WithCwd(dir)` / `FallbackCreateInDir()` (verified: read pkg/beads/ implementation)

**What's untested:**

- ⚠️ Whether orchestrators will consistently produce plan artifacts (behavioral compliance is the perpetual challenge)
- ⚠️ Whether the plan artifact format captures enough context for a fresh orchestrator to resume without re-derivation
- ⚠️ Whether `orch orient` injection of plan summaries adds meaningful comprehension or just adds noise
- ⚠️ How plan artifacts age — do they become stale? When should they be marked superseded?

**What would change this:**

- Evidence that orchestrators consistently ignore plan artifacts (would need stronger gate, not advisory)
- Evidence that plan narratives go stale faster than they're consumed (would need auto-staleness detection)
- Evidence that cross-project beads deps become necessary before narrative-level binding (would need beads cross-project dependency mechanism first)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create `.kb/plans/` artifact type | architectural | New artifact type in kb taxonomy, affects template ownership, touches multiple systems |
| Add `orch plan` CLI commands | implementation | Standard CLI addition within orch-go |
| Add session-end advisory gate for plan externalization | architectural | Touches completion pipeline, affects orchestrator workflow |
| Integrate active plans into `orch orient` | implementation | Extends existing orient data sources |
| Cross-project binding in plan narrative (manual) | implementation | Uses existing ECOSYSTEM.md pattern, no new infrastructure |

### Recommended Approach ⭐

**Companion Plan Artifact** — Create a `.kb/plans/` artifact type that carries strategic narrative alongside beads' graph structure, consumed by `orch orient` at session start.

**Why this approach:**
- Follows existing three-layer architecture (beads=work, kb=knowledge, workspace=ephemeral)
- Doesn't duplicate beads' graph mechanics — companion, not replacement
- Natural consumption via `orch orient` (the existing session-start briefing point)
- Follows "Evolve by Distinction" principle — plans are distinct from debriefs, investigations, and decisions
- Template ownership: orch-go owns coordination artifacts (per existing decision)

**Trade-offs accepted:**
- Plans are a new artifact type, adding taxonomy complexity (mitigated: minimal schema, clear distinction from existing types)
- Manual cross-project binding (mitigated: matches current working pattern per `kn-43aa5e`)
- Advisory gate, not blocking (mitigated: behavioral grammars say start advisory, graduate to blocking after measuring compliance)

**Implementation sequence:**

1. **Plan artifact template + `kb create plan`** — Define `.kb/plans/{date}-plan-{slug}.md` format. Template sections: TLDR, Phases (with beads IDs), Blocking Logic, Cross-Project Binding, Strategic Rationale, Status (active/superseded). Establish `kb create plan` command. This is foundational — everything else depends on having the artifact format.

2. **`orch plan` CLI commands** — `orch plan show` (display active plans with beads graph state overlay), `orch plan create` (interactive plan creation that generates beads issues + plan artifact together), `orch plan status` (show plan progress by checking beads issue statuses). These make plans first-class in the orchestration workflow.

3. **`orch orient` integration** — Add `ActivePlans []PlanSummary` to `OrientationData`. Orient scans `.kb/plans/` for status=active plans, extracts TLDR + current phase progress (by querying beads for referenced issue statuses). Inject into orient output so next orchestrator sees "Active plan: Toolshed/PW integration — Phase 1 complete, Phase 2 blocked on PW geometry storage."

4. **Session-end advisory gate** — In orchestrator completion/debrief flow, check: did this session produce multi-phase or cross-project work? If yes, is there a plan artifact? If no, warn (advisory, not blocking). Graduate to blocking after 30 days if compliance is poor. This follows the three-layer hotspot enforcement pattern: start advisory, measure, escalate.

### Plan Artifact Format

```markdown
# Coordination Plan: {title}

**Created:** YYYY-MM-DD
**Status:** active | superseded | completed
**Superseded-By:** [path to replacement plan, if superseded]
**Projects:** [list of repos involved]

## TLDR

[1-2 sentence summary of what this plan achieves and the overall sequence]

## Phases

### Phase 1: {name}
**Status:** ready | in-progress | complete
**Projects:** {repo names}
**Beads:** {beads-id-1}, {beads-id-2}
**Blockers:** none

### Phase 2: {name}
**Status:** blocked | ready | in-progress | complete
**Projects:** {repo names}
**Beads:** {beads-id-3}, {beads-id-4}
**Blockers:** Phase 1 ({beads-id-1} blocks {beads-id-3})
**Cross-project:** {beads-id-3} is in {repo-A}, {beads-id-4} is in {repo-B}

### Phase 3: {name}
...

## Blocking Logic

[Strategic rationale for the phasing — WHY this sequence, not just WHAT the sequence is.
This is the comprehension that would be expensive to re-derive.
Example: "Phase 2 is gated on geometry storage because forward simulation requires
real dimensions — simulation with placeholder dimensions produces meaningless results."]

## Cross-Project Binding

| Beads ID | Project | Repo Path | Notes |
|----------|---------|-----------|-------|
| {id} | {project} | ~/path/to/repo | {context} |

## Questions / Gates

[Open Question/Gate nodes that affect plan execution.
Reference beads question IDs where they exist.]

## Evolution

[Append-only log of plan changes — when phases were modified, why the plan was updated]
- YYYY-MM-DD: Plan created from orchestrator session
- YYYY-MM-DD: Phase 1 completed, Phase 2 unblocked
```

### Alternative Approaches Considered

**Option B: Plans as beads entities (new type)**
- **Pros:** Single system for all work tracking; `bd dep add` cross-references work naturally; `bd ready` integration built-in
- **Cons:** Beads is per-project by design — cross-project plans can't live in one beads DB; plan narrative is prose, not work status; violates beads' schema simplicity (it's work tracking, not knowledge management)
- **When to use instead:** If beads adds cross-project dependency support AND plan narratives are short enough to fit in issue descriptions

**Option C: Plans in session debriefs (`.kb/sessions/`)**
- **Pros:** No new artifact type; debriefs already capture embryonic plan content in "What's Next"; orient already reads debriefs
- **Cons:** Debriefs are session-scoped (one per session), plans span many sessions; debrief format doesn't support phasing, blocking logic, or cross-project binding; conflates "what happened" with "what should happen next"
- **When to use instead:** If plans are always single-session and single-project (they're not — the problem statement requires multi-session, cross-project)

**Option D: Plans in `~/.orch/plans/` (infrastructure state)**
- **Pros:** Global location, not per-project; aligns with focus.json being in `~/.orch/`
- **Cons:** Plans are knowledge artifacts (strategic rationale, phasing logic) not infrastructure state; template ownership decision says orch-go owns coordination artifacts but they belong in kb's knowledge taxonomy; `~/.orch/` has no versioning (not in git)
- **When to use instead:** If plans need to be truly global and not version-controlled per project

**Rationale for recommendation:** Plans are durable knowledge about work structure. They contain comprehension (strategic rationale) not just facts (issue statuses). The kb layer is where comprehension lives. Putting plans in beads conflates work tracking with knowledge management. Putting them in debriefs conflates backward-looking reflection with forward-looking intent. Putting them in `~/.orch/` loses version control. `.kb/plans/` follows the established pattern: beads=work, kb=knowledge, workspace=ephemeral.

---

### Implementation Details

**What to implement first:**
- Plan artifact template (format design above) — template in `.orch/templates/PLAN.md` or via `kb create plan` command
- `orch plan show` — scan `.kb/plans/` for active plans, overlay beads status on each phase's issues, display progress
- Orient integration (`ActivePlans` field) — lowest effort, highest visibility

**Things to watch out for:**
- ⚠️ Cross-project beads ID references in plan artifacts may become stale if issues are closed/renumbered — plan should reference by title as fallback
- ⚠️ Plan proliferation risk — need clear criteria for when a plan is warranted (multi-phase AND cross-project, or multi-phase with 3+ phases, or cross-project with 2+ repos)
- ⚠️ Plan staleness — active plans with no referenced beads activity for >7 days should trigger staleness warning in orient
- ⚠️ Defect Class 5 risk (Contradictory Authority Signals) — plan narrative might disagree with beads dependency graph if edited independently. The plan should be a *companion* that references beads, not an *alternative* that contradicts it. Beads graph is authoritative for blocking; plan narrative is authoritative for rationale.

**Areas needing further investigation:**
- Should `orch plan create` auto-generate beads issues from plan phases, or should plans be written after beads issues exist? (Likely both: interactive plan creation generates issues, post-hoc plan writing references existing issues)
- How should plans handle phase completion? Manual status update in the plan file, or auto-detection from beads issue statuses? (Recommend auto-detection via `orch plan status` — reduces staleness risk)
- Should the question type be demoted from beads? The task mentions this. Current answer: no — questions as beads entities with blocking semantics are working well (verified 2026-01-19 dogfooding). The plan artifact can reference question beads IDs in its "Questions / Gates" section. Demotion would lose the blocking mechanics that make `bd ready` work.

**Success criteria:**
- ✅ Next orchestrator session starts with plan context via `orch orient` (no re-derivation of sequencing)
- ✅ Cross-project coordination plan survives across sessions (Toolshed/PW example would be persisted and recoverable)
- ✅ Plan artifact distinguishes graph structure (beads) from strategic narrative (kb) — no duplication
- ✅ Plan compliance measurable (can count sessions that should have produced plans vs sessions that did)

---

## Blocking Questions

### Q1: Should `.kb/plans/` live in a specific project or be cross-project?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If per-project, each repo has its own `.kb/plans/` and cross-project plans live in the "primary" repo (same as cross-project epics today). If cross-project (e.g., `~/.kb/plans/`), plans have a natural global home but lose per-project version control.

**Recommendation:** Per-project in the primary repo (following `kn-43aa5e` — cross-project epics in primary repo), with cross-project binding in the plan narrative naming secondary repos and their beads IDs. This matches the existing working pattern and doesn't require new cross-project infrastructure.

### Q2: Where does `kb create plan` live — in kb-cli or orch-go?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If kb-cli, plans are a first-class kb artifact type (like investigations, decisions). If orch-go, plans are coordination artifacts that happen to live in `.kb/` (like SYNTHESIS.md lives in workspaces). Template ownership decision says: kb-cli owns knowledge artifacts, orch-go owns orchestration artifacts.

**Recommendation:** orch-go owns plan creation (`orch plan create`), but the artifact lives in `.kb/plans/`. This matches the pattern: orch-go creates SPAWN_CONTEXT.md (coordination artifact) in the workspace, and orch-go would create plan artifacts in `.kb/plans/`. The template lives in `.orch/templates/PLAN.md` following existing template ownership.

---

## References

**Files Examined:**
- `.kb/models/decidability-graph/model.md` — Complete decidability graph model (node taxonomy, edge authority, graph dynamics)
- `.kb/investigations/2026-02-28-inv-atc-lens-feature-audit.md` — ATC audit findings, especially Finding 3.1 (debrief pipeline gap)
- `.kb/investigations/2026-03-05-inv-design-systematic-mapping-decidability-graph.md` — Authority boundary enforcement mapping
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-28-probe-session-debrief-artifact-design.md` — Session debrief artifact design probe
- `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` — Question beads type decision
- `.kb/guides/beads-integration.md` — Beads integration patterns including cross-project
- `.kb/guides/orchestrator-session-management.md` — Orchestrator session lifecycle
- `.kb/guides/spawned-orchestrator-pattern.md` — Spawned orchestrator delegation
- `~/.kb/principles.md` — Foundational principles (Gate Over Remind, Evolve by Distinction, Premise Before Solution)
- `~/.orch/focus.json` — Single global focus goal (no phased plan concept)
- `~/.orch/ECOSYSTEM.md` — Cross-project registry
- `pkg/orient/orient.go` — Orient data model and data sources
- `.kb/sessions/TEMPLATE.md` and `2026-03-04-debrief.md` — Session debrief format and content

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` — Questions as beads entities (plan can reference question nodes)
- **Decision:** Cross-project epic pattern (`kn-43aa5e`) — Primary repo + ad-hoc secondary (plan follows same pattern)
- **Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-28-probe-session-debrief-artifact-design.md` — Facts vs comprehension gap
- **Investigation:** `.kb/investigations/2026-02-28-inv-atc-lens-feature-audit.md` — ATC audit, Finding 3.1

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: How should orchestrator coordination plans persist across sessions?
- Context: Spawned as architect task to design plan persistence. The problem: plans with phased execution, blocking relationships, and cross-project boundaries evaporate with session amnesia.

**2026-03-05:** Exploration complete
- Read decidability graph model, ATC audit, enforcement mapping investigation, orchestrator session management, beads integration, session debrief probe, spawned orchestrator pattern, debrief artifacts, orient data model
- Identified 5 decision forks: artifact location, format, gate type, consumption mechanism, cross-project binding

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: Plans decompose into graph (beads) + narrative (new `.kb/plans/` artifact) + binding (enriched narrative sections). Companion artifact pattern follows three-layer architecture. `orch orient` injection completes the briefing→flight→debrief→update cycle. Advisory gate at session end enforces externalization.
