# Strategic Decisions

Meta-orchestrator makes WHICH decisions. Orchestrator makes HOW decisions.

---

## The WHICH vs HOW Distinction

| Decision Type | Who Decides | Examples |
|---------------|-------------|----------|
| **WHICH** (Strategic) | Meta-orchestrator | Which epic? Which project? What's the goal this week? |
| **HOW** (Tactical) | Orchestrator | Which skill? How to verify? When to spawn? |

**The test:** "Is this about direction or execution?"
- Direction → Meta-orchestrator
- Execution → Orchestrator

---

## Strategic Decision Categories

### 1. Focus Allocation

- Which project gets attention today/this week?
- Which epic is highest priority?
- When to shift focus vs stay the course?

**Inputs:**
- Cross-project backlog state (`orch status` + `bd ready` across repos)
- Strategic goals (what matters most right now)
- Resource constraints (rate limits, time available)

### 2. Cross-Project Prioritization

When multiple projects compete:
- Which is most urgent?
- Which has momentum worth preserving?
- Which is blocking other work?

### 3. System Evolution

- Should we add a new skill?
- Should we update the orchestrator skill?
- Should we add a new tool/command?
- Should we change a process?

**Trigger:** Recurring friction in orchestrator handoffs, same failure mode 3+ times.

### 4. Milestone Decisions

- Is this epic actually complete?
- Should we ship or wait?
- Is the integration audit sufficient?

---

## What You Don't Decide

| Decision | Who Decides |
|----------|-------------|
| Which worker skill for this issue | Orchestrator |
| When to complete an agent | Orchestrator |
| Whether this issue is ready for daemon | Orchestrator |
| Implementation approach for a feature | Worker |
| Debugging strategy for a bug | Worker |

**If you're making these decisions, you've dropped a level.** Spawn an orchestrator and let them handle it.

---

## Decision Capture

Strategic decisions should be captured for future sessions:

| Type | Capture Location |
|------|------------------|
| Quick decision | `kb quick decide "X" --reason "Y"` |
| Significant architectural | `.kb/decisions/YYYY-MM-DD-title.md` |
| Process change | Update relevant skill or CLAUDE.md |
| Constraint | `kb quick constrain "X" --reason "Y"` |

**The test:** Will the next meta-orchestrator need this? → Externalize it.
