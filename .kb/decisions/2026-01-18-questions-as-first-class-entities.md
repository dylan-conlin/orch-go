# Decision: Questions as First-Class Entities

**Date:** 2026-01-18
**Status:** Accepted
**Context:** Synthesized from investigation on questions-based epic gating and artifact lifecycle analysis

## Summary

Strategic questions (pre-epic, gate-worthy) become beads entities of type `question`; tactical questions (within-epic probing) remain ephemeral in Epic Model "Probes Sent" tables. This operationalizes the "Premise Before Solution" principle through infrastructure rather than instruction.

## The Problem

Questions exist at two levels but currently share no infrastructure:

1. **Strategic questions** ("Should we build X?") - Gate epic creation, define direction
2. **Tactical questions** ("How does Y work?") - Guide implementation, inform decisions

Current system gaps:
- Questions can't exist without investigations (conflating "what to know" with "how to find out")
- Can't track "question asked but not yet investigated" state
- No way to block work on unanswered questions
- `bd ready` shows all issues regardless of prerequisite understanding

**Pain point:** Epic orch-go-erdw was created from "How do we X?" without validating the premise. Architect later found the premise was wrong. Work was spawned before the question was answered.

## The Decision

### Strategic Questions → Beads Entity

Add `question` as a new beads entity type for strategic, gate-worthy questions:

**Schema (minimal):**
- `id`, `title`, `description`, `status`, `priority`, `labels`
- `created_at`, `updated_at`, `closed_at`, `close_reason`
- Omit: `assignee`, `estimate`, `repro`, `understanding` (not work items)

**Status lifecycle:**
```
Open → Investigating → Answered → Closed
```

**Gate mechanics:**
- `bd dep add <epic-id> <question-id>` - Epic depends on question
- `bd ready` - Excludes question-blocked items
- `bd blocked` - Shows question-blocked work

### Tactical Questions → Ephemeral (No Change)

Tactical questions stay in Epic Model "Probes Sent" table:
- Working document tracking
- Short-lived (minutes to hours)
- Feed into Understanding section when answered
- Don't clutter entity space

### Decision Criteria: When to Create Question Entity

Create `question` entity when:
- Question gates whether epic/feature should exist
- Answer requires investigation spanning hours/days
- Multiple work items depend on the answer
- Question frames direction, not implementation

Keep ephemeral when:
- Question guides specific implementation step
- Answer found in minutes via code reading
- Question is scoped to single task
- Question is implementation detail, not direction

## Why This Design

### Principle: Evolve by Distinction

We were conflating:
- Questions (what we need to know)
- Investigations (how we find out)

Strategic vs tactical questions have different:
- Timelines (days vs minutes)
- Tracking needs (entity vs ephemeral)
- Gate implications (blocks epics vs informs tasks)

Distinguishing them enables appropriate treatment for each.

### Principle: Gate Over Remind

From principles.md: "Enforce knowledge capture through gates, not reminders."

Without infrastructure:
- "Don't start epic until question answered" → reminder (fails under cognitive load)

With infrastructure:
- `bd ready` excludes question-blocked items → gate (enforced automatically)

### Principle: Infrastructure Over Instruction

The Understanding section already requires answering 5 questions before epic creation. But those questions weren't tracked entities - just a checklist. Making questions entities:
- Tracks the journey from "asked" to "answered"
- Links investigations to the questions they answer
- Shows dashboard view of "blocking questions"
- Enables `bd ready` filtering

### Trade-offs Accepted

1. **Added entity type complexity** - Beads taxonomy grows from 4 to 5 types
   - Mitigation: Question schema is minimal (no assignee, estimate, verification)

2. **Question entity inflation risk** - May over-create question entities
   - Mitigation: Clear criteria ("gates epic creation" not "guides implementation")

3. **Dashboard view proliferation** - One more filter/view to maintain
   - Mitigation: Questions view consolidates "what needs answers" in one place

4. **Beads codebase changes required** - Moderate implementation effort
   - Accepted because alternative (separate system) duplicates machinery

## Implementation

### CLI Commands

```bash
# Create strategic question
bd create --type question --title "Should we adopt event sourcing?"

# Link to blocking question
bd dep add <epic-id> <question-id>

# View open questions
bd list --type question --status open

# Update status when investigation starts
bd update <question-id> --status investigating

# Close with answer
bd close <question-id> --reason "Answered: Yes, for audit trail requirements"
```

### Dashboard Integration

**Questions view:**
- Open (needs answer) - red
- Investigating (investigation active) - yellow
- Answered (recently) - green

**Other views:**
- Swarm Map: "Blocked by question" badge on agents
- Ready Queue: Excludes question-blocked issues
- Stats Bar: "3 questions open" counter

### Lifecycle Diagram

```
Question emerges → OPEN → Investigation spawned → INVESTIGATING → Understanding reached → ANSWERED → CLOSED
       ↓                         ↓                        ↓
   Blocks epics            Links to inv file        Unblocks work
```

## Evidence

- **Investigation:** `.kb/investigations/2026-01-18-inv-design-questions-first-class-entities.md`
- **Guide:** `.kb/guides/understanding-artifact-lifecycle.md` (Understanding section as answered questions)
- **Model:** `.kb/models/beads-integration-architecture.md` (dependency mechanics)
- **Concrete example:** Epic orch-go-erdw created without premise validation - questions as entities would have prevented spawn

## Related Decisions

- `2026-01-17-five-tier-completion-escalation-model.md` - Knowledge work surfacing pattern
- `2026-01-12-models-as-understanding-artifacts.md` - Understanding artifacts lifecycle
- `2026-01-14-verification-bottleneck-principle.md` - Gatekeeping patterns
