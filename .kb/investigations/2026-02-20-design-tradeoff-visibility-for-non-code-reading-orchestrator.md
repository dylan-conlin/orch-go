# Design: Tradeoff Visibility for Non-Code-Reading Orchestrator

**Date:** 2026-02-20
**Phase:** Complete
**Beads:** orch-go-1158
**Skill:** architect
**Type:** design investigation

---

## Design Question

How does the system surface architectural tradeoffs to a non-code-reading orchestrator? Dylan can't push back on implementation approaches he's never told about. The 6-week registry drift cycle (Dec 21 - Feb 18) happened partly because Dylan's feature requests implicitly demanded multi-source reconciliation, and no agent surfaced the tradeoff ("cache locally = fast but drifts" vs "query directly = slow but correct").

## Problem Framing

### Success Criteria

A good answer:
1. Identifies recurring tradeoff classes in this codebase with evidence
2. Shows exactly where in the current system tradeoffs get lost
3. Recommends a concrete mechanism that would have caught the registry cycle earlier
4. Fits within existing infrastructure (models, probes, skills, SYNTHESIS.md) — not a new system
5. Is enforceable (gate, not reminder) per the Gate Over Remind principle

### Constraints

- Dylan does not read code (design constraint)
- Dylan interacts with AI orchestrators conversationally (not CLI)
- Models and probes are the current mechanism for externalized architectural understanding
- The system already has 20+ principles, 11 models, 55+ decisions — adding complexity must justify itself
- Workers are task-scoped; they optimize for their task, not system health (Accretion Gravity)
- Session Amnesia means agents can't remember past tradeoffs without externalization

### Scope

**In scope:**
- What tradeoff classes recur in this codebase
- Current surfacing mechanisms and their gaps
- Concrete recommendation for closing the gap
- Changes to model format, skill guidance, or verification gates

**Out of scope:**
- Implementation of recommended changes (this produces the design; feature-impl implements)
- Changes to beads/bd itself
- Changes to OpenCode

---

## Exploration (Fork Navigation)

### Evidence: Seven Recurring Tradeoff Classes

Based on exhaustive search of 55+ decisions and 200+ investigations:

| # | Tradeoff Class | Frequency | Damage When Missed | Key Evidence |
|---|----------------|-----------|-------------------|--------------|
| 1 | **Cache vs direct query** | 5 iterations over 6 weeks | 238 dead agents, ghost status | `2026-02-18-two-lane-agent-discovery.md` |
| 2 | **Velocity vs verification** | 3 entropy spirals | 1,625+ commits lost | `2026-02-14-verifiability-first-hard-constraint.md` |
| 3 | **Speed vs correctness (bulk)** | Recurring in daemon, cleanup, status | Capacity stalls, stale display | `2026-01-17-file-based-workspace-state-detection.md` |
| 4 | **Simplicity vs completeness** | Ongoing in state lifecycle | Ghost agents, partial state | `2026-01-15-ghost-visibility-over-cleanup.md` |
| 5 | **Spawn mode selection** | Per-spawn decision | Cost overruns, reliability issues | `2026-01-09-dual-spawn-mode-architecture.md` |
| 6 | **Persistence boundary** | At each new integration point | Hard dependencies, partial state | `2026-02-18-two-lane-agent-discovery.md` |
| 7 | **In-memory vs durable dedup** | At daemon, spawn | Duplicate spawns, resource waste | `2026-02-14-daemon-duplicate-spawn-ttl-fragility` probe |

**Meta-pattern:** Classes #1, #2, and #3 share a common DNA: **choosing the fast path without declaring the correctness cost.** The fast path is always locally reasonable ("just a small cache to speed things up"), and the correctness cost only becomes visible after accumulated drift.

### Evidence: Six Gaps in Current Surfacing

| # | Gap | Evidence | Impact |
|---|-----|----------|--------|
| 1 | SYNTHESIS.md "Decisions Made" not read by any gate | `VerifySynthesis()` checks file size only | Tradeoffs captured but never surfaced |
| 2 | Explain-back gate checks presence, not tradeoff comprehension | `RunExplainBackGate()` gates on `explanation == ""` | Orchestrator passes with "Added the feature" |
| 3 | Architect investigations not linked to feature-impl completion | `DesignNotes` is freeform, not structured forks | Worker can diverge from architect recommendation silently |
| 4 | bd comment protocol carries phase status, not architectural content | Phase comments are `Phase: Implementing - doing X` | Real-time stream is blind to tradeoffs |
| 5 | CONSTRAINT comments have no acknowledgment gate | Instructions say "wait for acknowledgment" — no enforcement | Agents surface and continue in same turn |
| 6 | Models lack "pressure points" — what breaks if you change something | 11 models, none have pressure point section | Feature requests can't be checked against architectural fragility |

### Fork 1: Where should tradeoff surfacing live?

**Options:**
- A: In the model format (add "Pressure Points" section to all models)
- B: In SYNTHESIS.md (add required "Tradeoffs Accepted" section)
- C: In the completion pipeline (orch complete parses and surfaces tradeoffs)
- D: In spawn-time context (inject architectural pressure points at spawn)
- E: All of the above (layered defense)

**Substrate says:**
- Principle "Gate Over Remind": Tradeoff surfacing must be structural, not instructional
- Principle "Capture at Context": Capture tradeoffs when agents make them, not at session end
- Principle "Surfacing Over Browsing": Bring tradeoff context to the orchestrator, don't make them search
- Principle "Infrastructure Over Instruction": Make tools surface tradeoffs, don't rely on agents remembering
- Decision "Models Track Architecture": Models should capture architectural pressure points
- Model "Agent Lifecycle State Model": Demonstrates the cost of missing pressure points

**Recommendation:** Option E (layered defense), but with clear priority ordering:

1. **Models get "Pressure Points" section** (prevents the problem upstream — architects document where feature requests would violate invariants)
2. **SYNTHESIS.md gets "Architectural Choices" required section** (captures tradeoffs at the moment of decision)
3. **orch complete surfaces tradeoff content** (gates on orchestrator engaging with tradeoffs)
4. **SPAWN_CONTEXT injects model pressure points** (agents know upfront what's fragile)

**Trade-off accepted:** Increases cognitive overhead for agents (one more section to fill). Acceptable because: (a) only applies to decisions that actually involve tradeoffs — simple tasks won't have any, (b) the overhead is far less than 6 weeks of drift debugging.

**When this would change:** If the system moves to fully autonomous orchestration where Dylan doesn't review work. Then tradeoff surfacing would need to be agent-to-agent, not agent-to-human.

### Fork 2: Should tradeoff declaration be a gate or a prompt?

**Options:**
- A: Hard gate — orch complete fails if SYNTHESIS.md has no "Architectural Choices" section
- B: Soft prompt — orch complete warns but doesn't block
- C: Conditional gate — only gates for agents that modified architectural files or exceeded certain change thresholds
- D: Skill-level gate — only architect and feature-impl skills require tradeoff declaration

**Substrate says:**
- Principle "Gate Over Remind": Gates > reminders (argues for A or C)
- Principle "Accretion Gravity": Workers optimize for task, not system health (argues against relying on voluntary compliance)
- Evidence: The CONSTRAINT comment protocol (instruction-only) shows this doesn't work without enforcement
- Counter-evidence: Not every task involves architectural tradeoffs. Gating all tasks on tradeoff declaration would create false friction for simple bug fixes.

**Recommendation:** Option D (skill-level gate). Require "Architectural Choices" section in SYNTHESIS.md for architect, feature-impl, and systematic-debugging skills. Other skills (investigation, capture-knowledge) skip this gate.

**SUBSTRATE:**
- Principle: Gate Over Remind says enforce, not suggest
- Principle: Accretion Gravity says agents won't self-govern
- Evidence: Simple tasks (typo fixes, doc updates) genuinely have no tradeoffs to declare
- Model: Completion verification already has per-skill gates (skill output constraints)

**Trade-off accepted:** Investigation and capture-knowledge agents can make tradeoffs that aren't gated. Acceptable because: these skills produce knowledge artifacts, not code changes. Tradeoffs in knowledge work are lower-risk than tradeoffs in implementation.

**When this would change:** If investigation agents start making implementation changes (scope creep), this gate would need to expand.

### Fork 3: What should the "Pressure Points" model section contain?

**Options:**
- A: Freeform "what breaks if you change this" prose
- B: Structured table: Feature Request Pattern → Architectural Risk → Model Invariant Violated
- C: Constraint-linked format: each invariant gets an explicit "violating scenarios" list

**Substrate says:**
- Principle "Progressive Disclosure": Tables are scannable, prose is not
- Principle "Surfacing Over Browsing": `kb context` already surfaces model constraints; pressure points would be surfaced alongside them
- Evidence: The constraint "No persistent lifecycle caches" (invariant #7) would have prevented the registry cycle IF it had existed before Feb 2026
- Decision "Models Track Architecture": Pressure points are architectural, not implementation — they belong in models

**Recommendation:** Option B (structured table). Each model gets a "Pressure Points" section after "Constraints":

```markdown
## Pressure Points

| If Asked To... | Architectural Risk | Invariant at Risk |
|----------------|-------------------|-------------------|
| Cache agent state locally | Drift from authoritative sources; 6-week registry cycle | #7: No persistent lifecycle caches |
| Infer completion from session idle | False positives; session idle ≠ work complete | #3: Session existence ≠ agent still working |
| Add a fifth state layer | Reconciliation complexity; every new layer drifts | #5: Multiple sources must be reconciled |
```

**Trade-off accepted:** Additional maintenance burden on model authors. Acceptable because: (a) models are updated infrequently (when architecture changes), (b) pressure points are the highest-value content for the orchestrator — they're the "what breaks" that prevents feature-architecture conflicts.

**When this would change:** If models become auto-generated from code analysis, pressure points could be inferred rather than manually authored.

### Fork 4: How should orch complete surface tradeoff content?

**Options:**
- A: Parse SYNTHESIS.md "Architectural Choices" section, inject into explain-back prompt
- B: Add a `--tradeoffs` flag to orch complete that the orchestrator must fill
- C: Surface tradeoff content in `bd show` output so orchestrator sees it during review
- D: Require orchestrator to acknowledge each tradeoff explicitly before closing

**Substrate says:**
- Constraint "Orchestrator is AI, not Dylan": The orchestrator is an AI that calls CLI commands programmatically
- Constraint "User Interaction Model": CLI flags are for programmatic use, not human typing
- Decision "explain-back feature rework" (orch-go-w50): Designing CLI prompts was wrong; conversational is right
- Principle "Surfacing Over Browsing": Bring tradeoff content to the orchestrator

**Recommendation:** Option A (parse and inject). When `orch complete` runs, if the SYNTHESIS.md has an "Architectural Choices" section, include its content in the completion summary that the orchestrator reviews. The orchestrator then incorporates tradeoff understanding into its explain-back text.

This avoids:
- New flags the orchestrator must remember (Option B violates User Interaction Model)
- Separate review step (Option D is overhead for simple tradeoffs)
- Passive surfacing (Option C requires orchestrator to proactively check)

**SUBSTRATE:**
- Constraint: "Orchestrator is AI, not Dylan" — orchestrator calls orch complete with flags
- Decision: explain-back rework — conversational, not CLI-prompted
- Principle: Surfacing Over Browsing — bring content to the orchestrator, don't make it search

**Trade-off accepted:** Parsing SYNTHESIS.md content adds complexity to the verify pipeline. Acceptable because the pipeline already parses SYNTHESIS.md for discovered work (Next section).

**When this would change:** If SYNTHESIS.md format changes fundamentally. But the template is versioned and owned.

---

## Blocking Questions

### Q1: Should "Pressure Points" be added to all 11 models at once, or piloted on the most critical models first?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If pilot, start with agent-lifecycle-state-model and spawn-architecture (the two models most involved in the registry cycle). If all-at-once, use a synthesis sweep to add pressure points across all models in one session.

### Q2: Should the "Architectural Choices" SYNTHESIS.md section be mandatory only for new sessions, or retroactively required for in-flight work?

- **Authority:** implementation
- **Subtype:** judgment
- **What changes based on answer:** If new-only, update the SYNTHESIS.md template and skill guidance. If retroactive, existing agents would fail the gate on their next completion.

### Q3: Is the tradeoff visibility gap a symptom of a deeper problem — that feature requests don't flow through architectural models before becoming tasks?

- **Authority:** strategic
- **Subtype:** framing
- **What changes based on answer:** If yes, the fix is upstream: feature requests should be checked against model pressure points before spawning. This would be a new gate in the spawn flow, not just a completion gate. If no, the completion-time capture is sufficient.

---

## Synthesis: Recommended Approach

### The Core Insight

The registry drift cycle wasn't caused by bad engineering. Each cache was locally reasonable. The root cause is a **visibility gap**: when agents make "fast but drifts" vs "slow but correct" choices, no mechanism carries that tradeoff to the non-code-reading orchestrator. The tradeoff gets buried in code and only surfaces when drift causes visible pain.

### Layered Defense (4 Mechanisms)

**Layer 1: Model Pressure Points (prevent upstream)**

Add a "Pressure Points" section to the model format, after "Constraints". This transforms models from "here's how it works" to "here's how it works AND here's what breaks it."

Format:
```markdown
## Pressure Points

| If Asked To... | Architectural Risk | Invariant at Risk |
|----------------|-------------------|-------------------|
| [Feature request pattern] | [What goes wrong] | [Which invariant is violated] |
```

When `kb context` surfaces model summaries in SPAWN_CONTEXT, pressure points flow to agents at spawn time. An agent asked to "add caching for faster status" would see the pressure point and know the risk before starting.

**Why this is the most important layer:** It prevents the problem at the source. The registry cache was never checked against a model because the model didn't say "caching here = drift." With pressure points, the model would have said exactly that.

**Layer 2: SYNTHESIS.md "Architectural Choices" Section (capture at decision time)**

Add a required section to the SYNTHESIS.md template between "Evidence" and "Knowledge":

```markdown
## Architectural Choices

### [Choice description]
- **What I chose:** [approach taken]
- **What I rejected:** [alternative not taken]
- **Why:** [rationale]
- **Risk accepted:** [what could go wrong with this choice]

*(If no architectural choices were made, write: "No architectural choices — task was within existing patterns.")*
```

This is a Gate Over Remind implementation. The section exists in the template, and `VerifySynthesis()` can be extended to check for its presence in skills that require it (architect, feature-impl, systematic-debugging).

**Why this matters:** Currently, agents can make tradeoffs in code without ever declaring them. This section forces declaration at the moment of choice (Capture at Context).

**Layer 3: Completion Pipeline Surfaces Tradeoffs (bring to orchestrator)**

Extend `orch complete` to parse the "Architectural Choices" section from SYNTHESIS.md and include it in the completion summary. The orchestrator sees tradeoff content before writing the explain-back text.

This is a Surfacing Over Browsing implementation. Instead of hoping the orchestrator reads SYNTHESIS.md manually, the pipeline extracts and presents the relevant content.

**Layer 4: Feature Request → Model Pressure Point Check (future, requires Q3 resolution)**

If Q3 is answered "yes" (feature requests should flow through models), add a spawn-time check: when `orch spawn` receives a task description, `kb context` results include pressure points, and the spawn context explicitly flags any matches. This would be a lightweight version of "does this feature request conflict with architectural constraints?"

This layer is **not recommended for immediate implementation** — it requires the framing question (Q3) to be resolved. But it's the logical endpoint of this design.

---

## Recommendations

⭐ **RECOMMENDED:** Layered defense with Layers 1-3 implemented now, Layer 4 deferred pending Q3

- **Why:** Each layer catches tradeoffs at a different point in the lifecycle: prevention (models), capture (SYNTHESIS), surfacing (completion). Together they close the gap that allowed the registry cycle. Implementing all three now is feasible because each is a small change to an existing mechanism.
- **Trade-off:** Adds complexity to model authoring, SYNTHESIS template, and verify pipeline. Acceptable because the alternative — 6-week debugging cycles — is far more expensive.
- **Expected outcome:** Architectural tradeoffs become visible to the orchestrator at three points: before agents start (pressure points in SPAWN_CONTEXT), when agents decide (Architectural Choices in SYNTHESIS), and when the orchestrator reviews (tradeoff content in completion summary).

**Alternative: Require architect review for all feature work**
- **Pros:** Guarantees tradeoff analysis before implementation
- **Cons:** Massive overhead for simple features; architect skill already exists and captures tradeoffs when used
- **When to choose:** Only if Layers 1-3 prove insufficient after 4 weeks of use

**Alternative: Add tradeoff fields to bd comment protocol**
- **Pros:** Real-time tradeoff surfacing in the orchestrator's monitoring stream
- **Cons:** Requires bd schema changes; comments are freeform by design; over-structures the communication channel
- **When to choose:** If the completion-time surfacing (Layer 3) proves too late — i.e., tradeoffs need to be caught mid-session, not at completion

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the visibility gap that caused the 6-week registry drift cycle
- Future agents adding caching or local state should encounter these pressure points

**Suggested blocks keywords:**
- "tradeoff", "architectural choice", "cache", "pressure point"
- "model format", "SYNTHESIS template", "verify pipeline"
- "non-code-reading orchestrator", "visibility gap"

---

## Implementation-Ready Output

### File Targets

1. **`.kb/models/*/` (all 11 models)** — Add "Pressure Points" section. Start with `agent-lifecycle-state-model.md` and `spawn-architecture.md` as pilot.
2. **`.orch/templates/SYNTHESIS.md`** — Add "Architectural Choices" section between Evidence and Knowledge.
3. **`pkg/verify/check.go`** — Extend `VerifySynthesis()` to check for "Architectural Choices" section in skills that require it.
4. **`pkg/spawn/context.go`** — When generating SPAWN_CONTEXT, include model pressure points in the PRIOR KNOWLEDGE injection (this may already partially work if `kb context` surfaces the section).
5. **Skill guidance updates** — architect, feature-impl, systematic-debugging skills get instruction to fill "Architectural Choices" section.

### Acceptance Criteria

1. Agent-lifecycle-state-model has a populated "Pressure Points" section with the registry cycle tradeoff documented
2. SYNTHESIS.md template includes "Architectural Choices" section with format guidance
3. `VerifySynthesis()` gates on "Architectural Choices" presence for architect/feature-impl/systematic-debugging skills
4. `orch complete` output includes "Architectural Choices" content from SYNTHESIS.md when present

### Out of Scope

- Changes to beads/bd protocol
- Real-time tradeoff surfacing in bd comments (deferred alternative)
- Layer 4 (spawn-time pressure point matching) — deferred pending Q3
- Retroactive pressure point authoring for all 11 models (start with pilot of 2)

---

## References

- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — The decision that resolved the registry drift
- `.kb/models/agent-lifecycle-state-model.md` — The model that should have caught the drift earlier
- `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` — Full drift cycle narrative
- `~/.kb/principles.md` — Gate Over Remind, Capture at Context, Surfacing Over Browsing, Infrastructure Over Instruction
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-20-tradeoff-visibility-gap-analysis.md` — Companion probe
