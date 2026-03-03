---
owner: "[Owner name]"
started: "YYYY-MM-DD"
last_updated: "YYYY-MM-DD HH:MM"
phase: "[Planning/Implementation/Testing/Complete]"
status: "[Active/Blocked/Paused/Complete]"
template_version: "v4-frontmatter"
---

<!-- Pattern: TLDR Structure - See ~/.orch/patterns/tldr-structure.md -->

**TLDR:** [Problem/goal.] [Current status.] [Next action or blocker.]

---

# Workspace: [workspace-name]

<!-- Phase tracks workflow stage. Status tracks ability to continue.
Valid: Implementation+Active, Implementation+Blocked, Complete+Complete, Testing+Paused
Invalid: Complete+Active (contradictory) -->

---

## Summary

- **Current Goal:** [What you're working on now]
- **Next Step:** [Next action - must align with Phase/Status]
- **Blocking Issue:** [None / What's blocking - required if Status=Blocked]

---

## Session

**Scope:** [Small/Medium/Large] | **Started:** [timestamp] | **Last Activity:** [timestamp]

---

## Context

**Background:** [What problem are we solving?]

**Related beads:** [bd-xxx IDs or None]

**Dependencies:** [What must exist first]

---

## Tasks

- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

*Mark complete with actual time: `- [x] Task 1 (0.5h)`*
*Discovered issues → `bd create --discovered-from <this-workspace>`*

---

## Recovery

**For context overflow or crash - how to resume:**

- **Latest commit:** [sha] "[message]"
- **Next action:** [Specific task to continue]
- **Blockers:** [None / List]

```bash
orch resume <agent-id> "Continue from [task]"
```

---

## Decisions

**[YYYY-MM-DD]:** [Context]
- **Decision:** [What was chosen]
- **Why:** [Reasoning]
- **Trade-offs:** [What we're accepting]

---

## References

- [Links to related workspaces, investigations, decisions]

---

## Handoff Notes

**For the next person (orchestrator, Dylan, future agent):**

**Key context:**
- [Important background not in sections above]
- [Judgment calls made]

**Gotchas:**
- ⚠️ [Unexpected behavior or edge cases]

**Next steps:**
- [Recommended action with reasoning]

**Completion checklist:**
- [ ] All tests passing (or N/A)
- [ ] Phase: Complete, Status: Complete
- [ ] Changes committed
- [ ] Call `/exit` to close session

---

## Notes

[Additional observations]
