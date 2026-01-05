# Decision Authority Guide

**Purpose:** Define when agents can decide autonomously vs when to escalate to orchestrator/human. Removes ambiguity that leads to overly cautious agents or overreaching decisions.

**Last verified:** Jan 5, 2026

---

## The Core Distinction

| Escalate (Human/Orchestrator decides) | Agent Decides |
|---------------------------------------|---------------|
| **Strategic** - Direction, priorities, goals | **Tactical** - How to achieve given goal |
| **User-facing** - Behavior users interact with | **Internal** - Implementation users never see |
| **Irreversible** - Can't easily undo | **Reversible** - Can change later |
| **Cross-boundary** - Multiple projects/systems | **Single scope** - One project/component |
| **Resource commitment** - Time, cost, effort | **Within budget** - Already allocated resources |
| **Ambiguous tradeoffs** - Unclear which is better | **Clear criteria** - Obvious best choice |

---

## Agent Authority (Decide Freely)

Agents have full authority over implementation details within their scope:

### Code & Architecture (within scope)
- File organization and structure
- Naming conventions (variables, functions, files)
- Code patterns and idioms
- Refactoring for readability/maintainability
- Error handling approach
- Logging placement and verbosity

### Testing
- Which tests to write
- Test naming and organization
- Test utilities and helpers
- Mock/stub implementation
- Test coverage focus areas

### Documentation
- Code comments and docstrings
- Internal documentation wording
- README updates for changes made
- Investigation file structure

### Tool Selection (within patterns)
- Using tools already in the project
- Standard library choices
- Formatting and linting configuration
- Build script modifications

### Process
- Order of implementation steps
- When to checkpoint (within session scope)
- How to break down work
- Investigation methodology

---

## Escalation Required

These require orchestrator or human decision:

### Strategic Direction
- New features or capabilities
- Changing product behavior
- Deprecating functionality
- Prioritization between options

### User-Facing Changes
- UI/UX decisions
- API contract changes
- Error message wording users see
- Default values that affect behavior

### Irreversible Decisions
- Data migration approaches
- Database schema changes (non-additive)
- External service integrations
- Security model changes

### Cross-Boundary Impact
- Changes affecting multiple projects
- Shared library modifications
- Protocol or format changes
- Infrastructure decisions

### Resource Allocation
- Adding dependencies (new libraries)
- Platform or runtime requirements
- Cost-incurring features (API calls, storage)
- Significant time investment (>2x estimate)

### Ambiguous Trade-offs
- Performance vs maintainability
- Security vs usability
- Completeness vs shipping
- Technical debt vs velocity

---

## Decision Tree

```
Is it within your spawned task scope?
├── NO → Escalate (out of scope)
└── YES ↓

Does it change user-facing behavior?
├── YES → Escalate (user impact)
└── NO ↓

Is it easily reversible?
├── NO → Escalate (irreversible)
└── YES ↓

Does it cross project boundaries?
├── YES → Escalate (cross-boundary)
└── NO ↓

Are the trade-offs clear?
├── NO → Escalate (ambiguous)
└── YES ↓

Agent can decide.
```

---

## Examples

### Agent Can Decide

| Situation | Why Agent Decides |
|-----------|-------------------|
| "Should I use a helper function or inline this code?" | Implementation detail, reversible |
| "Which testing pattern: table-driven or individual tests?" | Testing strategy, reversible |
| "Should I add a comment explaining this logic?" | Documentation, reversible |
| "How should I organize these files in the package?" | File organization, reversible |
| "Should I extract this into a separate function?" | Refactoring, reversible |

### Escalate to Orchestrator

| Situation | Why Escalate |
|-----------|--------------|
| "Should I add a new CLI flag for this feature?" | User-facing behavior change |
| "This would be easier if I changed the database schema" | Irreversible, cross-boundary |
| "I could use a new library that does this better" | Resource (new dependency) |
| "The task says X but Y would be better" | Strategic direction |
| "This bug fix reveals a deeper architectural issue" | Scope expansion needed |

### Escalate to Human (Dylan)

| Situation | Why Human Decides |
|-----------|-------------------|
| "Should we support this platform?" | Strategic, resource commitment |
| "Is this security trade-off acceptable?" | Risk assessment |
| "Should we change the mental model for how X works?" | User-facing, strategic |
| "This feature could be built 3 different ways with unclear trade-offs" | Ambiguous trade-offs |

---

## Surface Before Circumvent

Before working around ANY constraint:

1. **Surface it first** - Report via `bd comment` or in your workspace
2. **Wait for acknowledgment** - Unless urgent, wait for orchestrator response
3. **Document the reasoning** - Why you're considering the workaround

This applies to:
- System constraints (API limits, tool limitations)
- Architectural patterns that seem inconvenient
- Process requirements that feel like overhead
- Prior decisions (from `kb context`) that conflict with your approach

**Why:** Working around constraints without surfacing:
- Prevents system from learning about recurring friction
- Bypasses stakeholders who should know
- Creates hidden technical debt

---

## Uncertainty Default

**When uncertain: Escalate.**

- Escalating unnecessarily wastes a few minutes
- Deciding wrongly can cost hours or create tech debt
- Better to ask than guess wrong

**Format for escalation:**
```
QUESTION: [Clear statement of the decision needed]

Options:
1. [Option A] - [pros/cons]
2. [Option B] - [pros/cons]

Recommendation: [Which you'd choose and why]
```

Providing options + recommendation is faster for orchestrator than open-ended questions.

---

## Related

- **spawn.md** - How spawning works (references this for authority levels)
- **Orchestrator skill** - Defines what orchestrators delegate vs retain
- **Principles** - Gate Over Remind, Surface Before Circumvent

---

## Provenance

This guide crystallizes patterns observed across agent sessions:

| Pattern | When Observed |
|---------|---------------|
| Agents asking about workspace naming | Multiple agents, Dec 2025 - should decide |
| Agents changing API contracts | Jan 2026 - should escalate |
| Agents adding new dependencies | Multiple incidents - should escalate |
| Agents refactoring code structure | Normal behavior - should decide |
| Agents uncertain about scope boundaries | Common question - escalate |

The criteria emerged from analyzing what made agents productive vs what caused rework.
