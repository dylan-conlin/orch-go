# Understanding Orchestrators

To manage orchestrators effectively, you must deeply understand what they know, what they produce, and how they fail.

---

## What Orchestrators Know

The orchestrator skill (which you inherit) contains:

| Domain | What They Know |
|--------|----------------|
| **Delegation** | ABSOLUTE DELEGATION RULE - never do spawnable work |
| **Worker Management** | Skill selection, spawning, completion verification |
| **Triage** | Issue labeling, daemon workflow, batch processing |
| **Synthesis** | Combining findings from multiple workers |
| **Session Lifecycle** | Focus blocks, session start/end, handoffs |
| **Autonomy** | When to act vs when to ask |

**Read the full orchestrator skill** (inherited via dependencies) to understand their complete operational context.

---

## What Orchestrators Produce

| Artifact | Purpose | Location |
|----------|---------|----------|
| SESSION_HANDOFF.md | Session summary for next orchestrator | `~/.orch/SESSION_HANDOFF.md` or project `.orch/` |
| Beads comments | Progress tracking during session | `bd comment` on active issues |
| Git commits | Code integration after worker completion | Project repo |
| Follow-up issues | Discovered work during session | `.beads/issues.jsonl` |

**SESSION_HANDOFF.md structure:**
- Summary (D.E.K.N. format)
- What happened this session
- Key insights
- Backlog state
- Friction encountered
- Next session start instructions

---

## Orchestrator Failure Modes

| Failure Mode | Symptoms | Your Response |
|--------------|----------|---------------|
| **Doing spawnable work** | Reading code, debugging, implementing | Remind of ABSOLUTE DELEGATION RULE |
| **Micromanaging workers** | Excessive guidance, not letting workers struggle | Let workers fail, learn from it |
| **Missing synthesis** | Completing workers without extracting insights | Require synthesis before close |
| **Context exhaustion** | 2+ hours without checkpoint, degraded output | Spawn fresh orchestrator session |
| **Scope creep** | Working on issues outside current focus | Redirect to focus or spawn new session |
| **Compensating for gaps** | Providing context system should surface | Note gap, let system fail, create improvement |
| **Frame collapse from vague goal** | Exploration → investigation → debugging (dropped 2 levels) | Check original goal specificity; refine goals before next spawn (see "Vague Goals Cause Frame Collapse") |

---

## Orchestrator Escalation Triggers

Orchestrators should escalate to you (meta-orchestrator) when:

- Strategic focus decisions (which epic? which project?)
- Cross-project prioritization conflicts
- System-level decisions (tooling changes, process improvements)
- Unusual failures requiring pattern analysis
- Completion of major work requiring synthesis across sessions

Orchestrators should NOT escalate for:
- Tactical decisions within focus
- Worker skill selection
- Individual issue triage
- Routine completions
