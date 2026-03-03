# Reviewing Orchestrator Handoffs

SESSION_HANDOFF.md is to orchestrators what SYNTHESIS.md is to workers. You review handoffs to understand what happened and decide next steps.

---

## The Review Workflow

```bash
# See completed orchestrator sessions
orch review --orchestrators

# Review specific handoff
cat ~/.orch/SESSION_HANDOFF.md
# or
cat <project>/.orch/SESSION_HANDOFF.md
```

---

## What to Look For

### 1. Summary (D.E.K.N.)

| Section | Question |
|---------|----------|
| **Delta** | What changed? Was meaningful progress made? |
| **Evidence** | Is there provenance? Can claims be verified? |
| **Knowledge** | What was learned? Should this become a decision/principle? |
| **Next** | Is the next step clear? Can another orchestrator resume? |

### 2. Friction Section

- What was harder than it should have been?
- Are there recurring friction patterns across sessions?
- Does this suggest system improvements?

### 3. Backlog State

- What's the current state of ready work?
- Were follow-up issues created?
- Are there blocked issues that need attention?

### 4. Gap Analysis

- Did the orchestrator ask for context the system should have surfaced?
- Did they compensate for missing knowledge?
- Should we create improvements to prevent this friction?

---

## Handoff Quality Checklist

- [ ] D.E.K.N. summary is substantive (not placeholder)
- [ ] What happened is clear without conversation context
- [ ] Friction captured (or explicitly "none")
- [ ] Next step is actionable
- [ ] Git pushed (work is not stranded locally)

---

## After Review

| Finding | Action |
|---------|--------|
| Clear next step | Spawn new orchestrator session if needed |
| Recurring friction | Create issue for system improvement |
| Knowledge gap | Add to kb quick or kb create |
| Strategic insight | Capture in decision record |
| Degraded session | Note pattern, don't penalize |

---

## Pattern Detection Across Handoffs

Over time, look for:

- **Recurring friction** - Same complaint in multiple sessions → system issue
- **Missing learnings** - Sessions without kb quick externalization → capture discipline slipping
- **Abandoned sessions** - Started but no handoff → investigate cause
- **Context exhaustion** - Sessions ending with degraded output → session scope too long

Use `kb reflect --type orchestrator` to automate pattern detection (when implemented).

---

## The Spawn Improvement Loop

**Meta-orchestrator's primary workflow:** Spawn → Observe → Review Handoff → Diagnose Friction → Improve Next Spawn

This is the continuous improvement cycle that makes the system better over time.

```
           ┌─────────────────────────────────────────┐
           │                                         │
           ▼                                         │
┌───────────────────┐    ┌───────────────────┐       │
│  SPAWN            │───▶│  OBSERVE/MONITOR  │       │
│  (with refined    │    │  (during session) │       │
│   goal)           │    │                   │       │
└───────────────────┘    └─────────┬─────────┘       │
                                   │                 │
                                   ▼                 │
┌───────────────────┐    ┌───────────────────┐       │
│  IMPROVE          │◀───│  DIAGNOSE         │       │
│  - Update spawn   │    │  - Review handoff │       │
│    context        │    │  - Identify gaps  │       │
│  - Refine goal    │    │  - Note patterns  │       │
│  - Add context    │    │                   │       │
└─────────┬─────────┘    └───────────────────┘       │
          │                                          │
          └──────────────────────────────────────────┘
```

**Each phase:**

| Phase | What You Do | Output |
|-------|-------------|--------|
| **Spawn** | Refine goal, add context, spawn orchestrator | ORCHESTRATOR_CONTEXT.md |
| **Observe** | Monitor during session (if real-time), or skip to Review | Frame corrections (if needed) |
| **Review** | Read SESSION_HANDOFF.md, check D.E.K.N. | Understanding of what happened |
| **Diagnose** | Identify friction, gaps, frame collapse | Improvement opportunities |
| **Improve** | Update templates, add context, refine goals | Better next spawn |

**What to improve after each cycle:**

| Friction Observed | Improvement |
|-------------------|-------------|
| Orchestrator asked for context | Add to ORCHESTRATOR_CONTEXT.md template |
| Frame collapsed to worker | Refine goal specificity (see "Vague Goals Cause Frame Collapse") |
| Same failure mode 3+ times | Update orchestrator skill |
| Knowledge gap | Create kb quick entry or kb investigation |
| Process unclear | Update documentation |

**The discipline:** Every spawn should be better than the last. If you're not improving the loop, you're just watching cycles.
