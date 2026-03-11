---
name: skill-artifact-drift
description: Skills and artifacts (SKILL.md, CLAUDE.md, models) drift out of sync as features land — recurring problem, prior systemic fixes haven't stuck
type: project
---

Recurring drift: as agents ship features, the skills, CLAUDE.md, models, and other artifacts that reference those features don't get updated. Dylan flags this periodically when he remembers. Prior attempts to solve systemically haven't worked. Examples of what drifts: new commands not in CLAUDE.md, new spawn flags not in orchestrator skill, models not updated with new evidence.

This is a coordination failure in our own taxonomy — each agent correctly ships its feature but doesn't update the artifacts that reference the changed surface area.
