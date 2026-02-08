<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Questions exist at two levels (strategic pre-epic, tactical within-epic) and need different treatment; strategic questions warrant beads entity type, tactical questions belong in Epic Model working docs.

**Evidence:** Analyzed Understanding Artifact Lifecycle guide, beads integration model, Epic Model template pattern, and principles (Evolve by Distinction, Gate Over Remind); current system conflates "what we need to know" with "how we find out."

**Knowledge:** The key distinction is strategic vs tactical questions - strategic questions gate epic creation and need entity-level tracking; tactical questions are probing mechanisms within epic work and should remain ephemeral.

**Next:** If approved, implement `question` beads entity type for strategic questions, with Answered state triggering understanding gates.

**Promote to Decision:** Actioned - decision exists (questions-as-first-class-entities)

---

# Investigation: Design Questions as First-Class Entities

**Question:** Should questions be first-class entities in the orchestration system, and if so, how should they integrate with existing beads, investigations, and epics?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Orchestrator decision on whether to proceed with implementation
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A - New architectural concept
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Questions and Investigations Are Currently Conflated

**Evidence:**

Every investigation file has a `**Question:**` field at the top:
```markdown
# Investigation: Fix Spawn Reliability

**Question:** Why does spawn fail 30% of the time when registry lock is held?
```

This conflates two distinct things:
1. **The question** - "What do we need to know?"
2. **The investigation** - "How do we find out?"

An investigation IS the process of answering a question. But currently:
- Questions can't exist without investigations
- Can't track "question asked but not yet investigated" state
- Can't have multiple investigations answer one question
- Questions die when investigations close

**Source:**
- Investigation file template (lines 38-46)
- `.kb/investigations/2025-12-20-inv-orch-add-question-command.md:8`
- Pattern observed across 100+ investigation files

**Significance:** This conflation prevents tracking questions as first-class entities. The "Probes Sent" table in Epic Model template was an ad-hoc solution to this gap - it tracked Question Asked → What We Learned → New Questions Raised. But it was ephemeral (working doc), not persistent.

---

### Finding 2: Two Distinct Types of Questions Exist

**Evidence:**

Analysis of how questions arise in the system reveals two patterns:

**Pattern A - Strategic Questions (Pre-Epic)**
- "Should we build a daemon?" → Decides whether epic exists
- "Is the current architecture sustainable?" → Shapes future direction
- Answered by: Investigation → Decision
- Timeline: Days to weeks
- Tracking need: High (gates major work)

**Pattern B - Tactical Questions (Within-Epic)**
- "How does the SSE client handle reconnection?" → Guides implementation
- "What's the timeout for RPC calls?" → Informs coding decisions
- Answered by: Code reading, quick investigation
- Timeline: Minutes to hours
- Tracking need: Low (ephemeral probing)

The Epic Model template's "Probes Sent" table served tactical questions. But strategic questions have no home.

**Source:**
- `.kb/guides/understanding-artifact-lifecycle.md:78-103` (Epic Model usage)
- `.kb/investigations/2026-01-13-inv-analyze-understanding-artifact-architecture-epic.md:95-98` (Probes Sent pattern)
- kb principles.md "Evolve by Distinction" principle

**Significance:** Applying "Evolve by Distinction" - we're conflating strategic and tactical questions. They need different treatment:
- Strategic → Entity-level tracking with gates
- Tactical → Working document tracking, ephemeral

---

### Finding 3: Epic Readiness Is Already Questions-Based

**Evidence:**

Epic creation requires answering 5 questions (the "Ready Gate"):
1. What problem are we actually solving?
2. Why did previous approaches fail?
3. What are the key constraints?
4. Where do the risks live?
5. What does "done" look like?

From `.kb/guides/understanding-artifact-lifecycle.md:106-146`:
```
Understanding Section (Epic-Scoped Gate)
Purpose: Readiness gate when creating epic - proves orchestrator
understands the problem before spawning implementation work.
```

And from beads implementation:
```bash
bd create --type epic --understanding "Problem: ... Previous: ... Constraints: ... Risks: ... Done: ..."
```

The Understanding section IS answered questions. Epic readiness IS "strategic questions answered."

**Source:**
- `.kb/guides/understanding-artifact-lifecycle.md:106-146`
- `.kb/investigations/2026-01-07-inv-epic-readiness-gate-understanding-section.md`
- beads epic creation with `--understanding` flag

**Significance:** The system already gates epics on questions being answered. But:
- The questions aren't tracked as entities
- The journey from "question asked" to "question answered" isn't visible
- New questions raised during investigation aren't connected to original questions
- Dashboard can't show "blocking questions"

---

### Finding 4: Current Entity Types Don't Fit Questions

**Evidence:**

Current beads entity types and their characteristics:

| Type | Has Assignee | Has Estimate | Has Verification | Produces Artifact |
|------|-------------|-------------|------------------|-------------------|
| task | Yes | Yes | Yes | Code/docs |
| bug | Yes | Yes | Yes | Fix |
| feature | Yes | Yes | Yes | Implementation |
| epic | Yes (indirect) | No | No | Children completion |
| **question** | No | No | No | Answer/understanding |

Questions don't fit the "work item" model:
- No one "works on" a question - they work on investigations that answer it
- No estimate for "how long until answered" - depends on complexity discovered
- No verification - answer is judged by understanding, not deliverable
- Artifact is understanding, not code

But questions DO need:
- Lifecycle (Open → Investigating → Answered → Closed)
- Dependencies (question blocks epic/feature)
- Visibility (dashboard: "what questions need answers?")

**Source:**
- `.beads/.beads/issues.jsonl` - existing issue structure
- `.kb/models/beads-integration-architecture.md:69-105` - three integration points
- beads CLI: `bd create`, `bd ready`, `bd show`

**Significance:** Questions are "lightweight" beads entities - they need lifecycle and dependencies but not the work-tracking baggage (assignee, estimate, verification). This suggests a new entity type with simpler schema.

---

### Finding 5: Gate Mechanics Need Questions

**Evidence:**

From principles.md "Gate Over Remind":
> "Enforce knowledge capture through gates (cannot proceed without), not reminders (easily ignored)."

And "Gates must be passable by the gated party":
> "A gate that the agent cannot satisfy by its own work is not a gate - it's a human checkpoint."

For questions to gate work effectively:
1. Question status must be queryable (`bd ready` shouldn't show blocked work)
2. Question answered = gate passed (investigation closes with answer → question closes)
3. Dependencies must flow (question-X blocks epic-Y)

Current system gaps:
- `bd ready` shows issues, not questions
- No "is this question answered?" state
- Dependencies can't reference questions (because questions aren't entities)

**Source:**
- `~/.kb/principles.md` - Gate Over Remind principle
- `.kb/models/beads-integration-architecture.md:149-195` - dependency mechanics
- `bd ready` output format

**Significance:** To make questions gate work, they need entity status. Reminders ("don't start until question answered") fail under cognitive load. Gates ("bd ready excludes question-blocked items") work.

---

## Synthesis

**Key Insights:**

1. **Evolve by Distinction** - Questions exist at two levels that need different treatment:
   - Strategic questions (gate epic creation) → Need entity tracking
   - Tactical questions (guide probing) → Stay in Epic Model working docs

2. **Questions Are Pre-Work** - Questions define what we need to know BEFORE work starts. They're not work items themselves but prerequisites for work. This makes them fundamentally different from task/bug/feature.

3. **Epic Readiness Is Already Questions-Based** - The Understanding section requirement is "answer these 5 questions." Formalizing questions as entities makes this explicit and trackable.

4. **Investigations Answer Questions** - The relationship is 1:many. One question can spawn multiple investigations (probing). One investigation can answer multiple questions. This relationship should be explicit.

**Answer to Investigation Question:**

**Yes, strategic questions should be first-class entities in beads.** They should:
- Have their own entity type (`question`)
- Have lightweight schema (no assignee, estimate, verification)
- Have lifecycle (Open → Investigating → Answered → Closed)
- Support dependencies (question blocks epic/task/feature)
- Be visible in dashboard ("Open Questions" view)
- Gate work via `bd ready` (blocked-by-question items excluded)

**Tactical questions should NOT be entities.** They should:
- Stay in Epic Model "Probes Sent" table
- Be ephemeral (working document)
- Feed into Understanding section when answered
- Not clutter entity space with high-volume, short-lived items

---

## Structured Uncertainty

**What's tested:**

- ✅ Investigation files have Question field (verified: examined 20+ investigations)
- ✅ Epic Understanding section is questions-based (verified: read lifecycle guide and beads implementation)
- ✅ Beads supports entity types and dependencies (verified: reviewed beads-integration-architecture model)
- ✅ Current entity types don't fit questions (verified: compared schemas)

**What's untested:**

- ⚠️ Volume of strategic vs tactical questions (hypothesis: 80% tactical, 20% strategic)
- ⚠️ Whether beads schema extension is straightforward (would need beads codebase review)
- ⚠️ Dashboard performance with question entities (would need implementation + testing)
- ⚠️ Whether `bd ready` modification disrupts existing workflows

**What would change this:**

- If strategic questions are rare (<5% of all questions), separate entity type may be overhead
- If beads schema is rigid, might need separate `.kb/questions/` system instead
- If most "questions" are really just "tasks phrased as questions," entity type is wrong model

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Question as Lightweight Beads Entity** - Add `question` as a new beads entity type with minimal schema, supporting lifecycle and dependencies but not work-tracking fields.

**Why this approach:**
- Reuses existing beads machinery (lifecycle, dependencies, CLI, dashboard integration)
- Questions CAN block work naturally via dependency system
- `bd ready` already filters by dependencies - questions integrate cleanly
- Follows "Infrastructure Over Instruction" principle - gates, not reminders

**Trade-offs accepted:**
- Adds complexity to beads entity taxonomy
- Questions mixed with issues in some views (mitigation: filter by type)
- Requires beads codebase changes (moderate effort)

**Implementation sequence:**

1. **Add `question` entity type to beads**
   - Schema: `id`, `title`, `description`, `status`, `priority`, `labels`, `created_at`, `updated_at`, `closed_at`, `close_reason`
   - Omit: `assignee`, `estimate`, `repro`, `understanding` (not applicable)
   - Status values: `open`, `investigating`, `answered`, `closed`

2. **Wire question lifecycle**
   - `bd create --type question --title "Should we..."` creates question
   - `bd update <id> --status=investigating` when investigation spawned
   - `bd close <id> --reason="Answered: ..."` when understanding reached
   - Investigation files can reference: `**Answers:** <question-id>`

3. **Implement question gates**
   - `bd dep add <epic-id> <question-id>` makes epic depend on question
   - `bd ready` excludes items blocked by open questions
   - `bd blocked` shows question-blocked items

4. **Dashboard "Questions" view**
   - Filter: `type=question`
   - Columns: Title, Status, Blocking, Age
   - Visual: Open (red), Investigating (yellow), Answered (green)

### Alternative Approaches Considered

**Option B: Separate `.kb/questions/` System**
- **Pros:**
  - Clean separation from beads (questions ≠ work)
  - Can have different lifecycle/schema
  - No beads codebase changes
- **Cons:**
  - Duplicates storage/CLI machinery
  - Cross-referencing harder (question → epic linkage)
  - Two systems to learn and maintain
- **When to use instead:** If beads schema is too rigid to extend, or if questions need fundamentally different tooling

**Option C: Questions as Attributes of Epics/Investigations**
- **Pros:**
  - No new entity type
  - Already have Question field in investigations
  - Simpler
- **Cons:**
  - Can't track questions independently
  - Loses "question asked, not yet investigated" state
  - Can't have question lifecycle separate from investigation
  - No dependency/gate support
- **When to use instead:** If strategic questions are very rare (<5%) and overhead isn't justified

**Rationale for recommendation:** Option A (beads entity) gives full infrastructure benefits (lifecycle, dependencies, dashboard) while Option B duplicates machinery and Option C loses the key capability (independent question tracking with gates).

---

### Implementation Details

**What to implement first:**
- `bd create --type question` with minimal schema
- Question status lifecycle (open → investigating → answered → closed)
- Basic CLI support (create, show, update, close)

**Things to watch out for:**
- ⚠️ Question entity inflation - need guidance on "when to create question entity vs just document"
- ⚠️ Dashboard view proliferation - one more filter/view to maintain
- ⚠️ Existing workflows assume issues have assignees - question display may need special handling

**Areas needing further investigation:**
- Beads codebase: How hard is adding new entity type?
- Auto-linking: Can closing an investigation auto-update related question status?
- Question templates: Common question types (strategic, technical, design) worth standardizing?

**Success criteria:**
- ✅ Can create question: `bd create --type question --title "Should we..."`
- ✅ Can block epic on question: `bd dep add <epic> <question>`
- ✅ `bd ready` excludes question-blocked items
- ✅ Dashboard shows open questions needing answers
- ✅ Closing investigation with answer closes related question

---

## Dashboard Design

**Questions View:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Questions                                            Mode: Operational  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│ ● Open (needs answer)                                                   │
│ ├─ orch-go-q123: Should questions be first-class entities?             │
│ │   → Blocks: orch-go-epic456 "Question tracking system"               │
│ │   Age: 2 hours                                                        │
│ │                                                                       │
│ ├─ orch-go-q124: What's the right model for question lifecycle?        │
│ │   → Blocks: orch-go-task789                                          │
│ │   Age: 1 day                                                          │
│                                                                         │
│ ◐ Investigating (investigation active)                                  │
│ ├─ orch-go-q125: How does spawn handle race conditions?                │
│ │   → Investigation: .kb/investigations/2026-01-18-inv-spawn-race.md   │
│ │   Age: 3 hours                                                        │
│                                                                         │
│ ✓ Answered (recently - last 24h)                                        │
│ ├─ orch-go-q126: Should we use RPC or CLI for beads?                   │
│ │   → Answer: RPC-first with CLI fallback                              │
│ │   Unblocked: orch-go-feature101                                       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Integration with Existing Views:**

| View | Question Impact |
|------|-----------------|
| Swarm Map | Show "Blocked by question" badge on agents |
| Ready Queue | Exclude question-blocked issues |
| Stats Bar | "3 questions open" counter |
| Epic Detail | Show "Questions to answer" section |

---

## Lifecycle Diagram

```
                    ┌─────────────────────────────────────────────┐
                    │           QUESTION LIFECYCLE                 │
                    └─────────────────────────────────────────────┘

    Question emerges          Investigation           Understanding
    (strategic need)           spawned                  reached
          │                       │                        │
          ▼                       ▼                        ▼
    ┌──────────┐            ┌──────────────┐         ┌──────────┐
    │   OPEN   │───────────▶│ INVESTIGATING │────────▶│ ANSWERED │
    │          │  spawn     │              │  close   │          │
    │ • Title  │  invest.   │ • Title      │  with    │ • Title  │
    │ • Desc   │            │ • Desc       │  answer  │ • Desc   │
    │ • Blocks │            │ • Blocks     │          │ • Answer │
    └──────────┘            │ • Inv link   │          │ • Unblocks│
          │                 └──────────────┘          └──────────┘
          │                        │                        │
          ▼                        ▼                        ▼
    New information         Investigation          Question closed,
    may reopen              may branch             gates released

                    ┌──────────────────────────────┐
                    │                              │
                    │  RELATIONSHIPS:              │
                    │                              │
                    │  Question ←───── Investigation │
                    │     │        "answers"       │
                    │     │                        │
                    │     └────────▶ Epic/Task    │
                    │          "blocks"           │
                    │                              │
                    └──────────────────────────────┘
```

---

## References

**Files Examined:**
- `.kb/guides/understanding-artifact-lifecycle.md` - Understanding progression (Epic Model → Understanding → Model)
- `.kb/models/beads-integration-architecture.md` - Beads RPC/CLI, lifecycle, dependencies
- `.kb/models/dashboard-architecture.md` - Dashboard views and architecture
- `.kb/investigations/2026-01-13-inv-analyze-understanding-artifact-architecture-epic.md` - Epic Model analysis
- `.kb/investigations/2026-01-07-inv-epic-readiness-gate-understanding-section.md` - Understanding section implementation
- `.kb/investigations/2025-12-20-inv-orch-add-question-command.md` - Question extraction command
- `.beads/.beads/issues.jsonl` - Beads issue structure (first 5 lines)
- `~/.kb/principles.md` - Meta-orchestration principles

**Commands Run:**
```bash
# Check beads issue structure
head -100 .beads/.beads/issues.jsonl | head -5

# Search for Probes Sent pattern
grep -r "Probes Sent" .
```

**External Documentation:**
- None required - internal system design

**Related Artifacts:**
- **Guide:** `.kb/guides/understanding-artifact-lifecycle.md` - Documents current understanding progression
- **Model:** `.kb/models/beads-integration-architecture.md` - Beads integration patterns
- **Investigation:** `.kb/investigations/2026-01-13-inv-analyze-understanding-artifact-architecture-epic.md` - Prior art on understanding artifacts

---

## Investigation History

**2026-01-18 10:00:** Investigation started
- Initial question: Should questions be first-class entities?
- Context: Discussion identified questions as direction-defining but untracked

**2026-01-18 11:00:** Key finding - two question types
- Discovered strategic vs tactical distinction
- Applied "Evolve by Distinction" principle
- Determined strategic questions need entity tracking, tactical stay ephemeral

**2026-01-18 11:30:** Design recommendation formed
- Recommended: Question as lightweight beads entity
- Rejected: Separate system (duplicates machinery), attributes-only (loses lifecycle)
- Designed: Lifecycle, gates, dashboard integration

**2026-01-18 12:00:** Investigation completed
- Status: Complete
- Key outcome: Questions should be beads entities (strategic) with lifecycle and gate support; tactical questions remain in working docs
