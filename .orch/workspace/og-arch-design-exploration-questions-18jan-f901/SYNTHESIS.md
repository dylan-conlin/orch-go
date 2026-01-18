# Session Synthesis

**Agent:** og-arch-design-exploration-questions-18jan-f901
**Issue:** Ad-hoc (no tracking)
**Duration:** 2026-01-18 10:00 → 2026-01-18 12:00
**Outcome:** success

---

## TLDR

Design exploration for questions as first-class entities in the orchestration system. **Recommendation:** Add `question` as a lightweight beads entity type for strategic questions (pre-epic, gate-worthy), while tactical questions (within-epic probing) remain ephemeral in working docs. This follows "Evolve by Distinction" - strategic and tactical questions need different treatment.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-inv-design-questions-first-class-entities.md` - Design investigation with recommendation for questions as beads entities

### Files Modified
- None

### Commits
- Pending (to be committed with this SYNTHESIS.md)

---

## Evidence (What Was Observed)

- Investigation files all have `**Question:**` field, but this conflates question with investigation
- Epic Model template had "Probes Sent" table (Question Asked → What We Learned → New Questions Raised) - ad-hoc solution to question tracking gap
- Epic readiness is already questions-based: Understanding section requires answering 5 questions
- Current beads entity types (task, bug, feature, epic) don't fit questions - questions lack assignee, estimate, verification
- Beads dependencies could support questions blocking work if questions were entities

### Source Evidence
- `.kb/guides/understanding-artifact-lifecycle.md:106-146` - Understanding section as gate
- `.kb/models/beads-integration-architecture.md:69-105` - Three integration points
- `~/.kb/principles.md` - "Evolve by Distinction", "Gate Over Remind" principles
- `.kb/investigations/2026-01-13-inv-analyze-understanding-artifact-architecture-epic.md:95-98` - Probes Sent pattern

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-inv-design-questions-first-class-entities.md` - Complete design investigation with recommendation

### Key Insight: Two Types of Questions

**Strategic Questions (Pre-Epic)**
- "Should we build X?" → Decides whether epic exists
- Timeline: Days to weeks
- Tracking need: High (gates major work)
- Example: "Should questions be first-class entities?"

**Tactical Questions (Within-Epic)**
- "How does Y work?" → Guides implementation
- Timeline: Minutes to hours
- Tracking need: Low (ephemeral probing)
- Example: "What's the timeout for RPC calls?"

**The Distinction Matters:** Strategic questions need entity-level tracking with gates. Tactical questions should remain in Epic Model working docs.

### Design Recommendation

Add `question` as a new beads entity type with:
- Lightweight schema (no assignee, estimate, verification)
- Lifecycle: Open → Investigating → Answered → Closed
- Dependencies: Questions can block epics/tasks/features
- Dashboard: "Open Questions" view showing blocking questions

### Constraints Discovered
- Questions are fundamentally different from work items - they're prerequisites for work, not work itself
- Beads dependency system already supports the blocking mechanics needed
- Dashboard has two-mode design - questions would add a third filter/view

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Approved for Implementation

**Epic:** "Question Tracking System"

**Children:**
1. **beads:** Add `question` entity type with lightweight schema
2. **beads:** Wire question lifecycle (open → investigating → answered → closed)
3. **beads:** Enable dependencies between questions and other entities
4. **orch-go:** Dashboard "Questions" view showing open questions and blocking relationships

**Success Criteria:**
- `bd create --type question --title "Should we..."` works
- `bd dep add <epic> <question>` blocks epic on question
- `bd ready` excludes question-blocked items
- Dashboard shows open questions needing answers

### If Not Approved

**Alternative:** Document this design for future reference; continue using Epic Model "Probes Sent" pattern for tactical questions and informal tracking for strategic questions.

---

## Unexplored Questions

**Questions that emerged during this session:**
- How hard is it to add a new entity type to beads? (would need beads codebase review)
- Should there be auto-linking between questions and investigations? (closing investigation could auto-update question status)
- Are there common question templates worth standardizing? (strategic, technical, design questions)

**Areas worth exploring further:**
- Question entity inflation: What guidance prevents over-creating question entities?
- Cross-project questions: Can a question in orch-go block work in another project?
- Question bundles: Groups of related questions that gate an epic

**What remains unclear:**
- Volume of strategic vs tactical questions in practice (hypothesis: 80% tactical, 20% strategic)
- Whether beads schema is flexible enough for new entity type

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-exploration-questions-18jan-f901/`
**Investigation:** `.kb/investigations/2026-01-18-inv-design-questions-first-class-entities.md`
**Beads:** N/A (ad-hoc spawn)
