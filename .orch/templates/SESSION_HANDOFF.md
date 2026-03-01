# Session Handoff

**Orchestrator:** {workspace-name}
**Focus:** {session-goal}
**Duration:** {start-time} → {end-time}
**Outcome:** {success | partial | blocked | failed}

---

<!--
## How to Use This Template (Progressive Synthesis)

**Fill this file AS YOU WORK, not at the end.**

The anti-pattern: "I'll write the handoff when I'm done" → leads to lost context,
incomplete sections, and the cognitive load of reconstructing what happened.

**Progressive documentation pattern:**
1. **SESSION START:** Fill metadata (workspace, focus, duration start)
2. **DURING:** Add to Spawns, Evidence, Friction as you go
3. **BEFORE HANDOFF:** Synthesize Knowledge, fill Next section
4. **FINAL:** Write TLDR, update outcome, review for patterns

**Section timing:**
| Section | When to Fill |
|---------|--------------|
| TLDR | Last (after you know what happened) |
| Spawns | During work (as you spawn/complete agents) |
| Evidence | During work (as you observe patterns) |
| Knowledge | Before handoff (what emerged) |
| Friction | Anytime (capture frustrations immediately) |
| Next | Before handoff (what should happen) |
| Unexplored | Anytime (capture questions as they emerge) |

**Why this matters:**
- Details are lost if not captured immediately
- "I'll remember" → you won't (session amnesia)
- Progressive fill reduces end-of-session cognitive load
- Friction section needs real-time capture (you'll rationalize it away later)
-->

## TLDR

[1-2 sentence summary. What was the focus? What was achieved?]

---

## Spawns (Agents Managed)

### Completed
| Agent | Issue | Skill | Outcome | Key Finding |
|-------|-------|-------|---------|-------------|
| {workspace} | {beads-id} | {skill} | {success/partial/failed} | {1-line insight} |

### Still Running
| Agent | Issue | Skill | Phase | ETA |
|-------|-------|-------|-------|-----|
| {workspace} | {beads-id} | {skill} | {phase} | {estimate} |

### Blocked/Failed
| Agent | Issue | Blocker | Next Step |
|-------|-------|---------|-----------|
| {workspace} | {beads-id} | {what blocked} | {spawn-fresh/escalate/defer} |

---

## Evidence (What Was Observed)

### Patterns Across Agents
- [Pattern 1: e.g., "3 agents hit the same auth issue"]
- [Pattern 2: e.g., "Investigation agents consistently took 2+ hours"]

### Completions
- **{beads-id}:** {what SYNTHESIS.md revealed}
- **{beads-id}:** {what SYNTHESIS.md revealed}

### System Behavior
- [Observation about orch/beads/kb tooling]
- [Observation about spawn patterns]

---

## Knowledge (What Was Learned)

### Decisions Made
- **{topic}:** {decision} because {rationale}

### Constraints Discovered
- {constraint} - why it matters

### Externalized
- `kb quick decide "X" --reason "Y"` - [if applicable]
- `kb quick constrain "X" --reason "Y"` - [if applicable]
- `.kb/decisions/YYYY-MM-DD-*.md` - [if created]

### Artifacts Created
- `.kb/investigations/YYYY-MM-DD-*.md` - {brief description}
- `.kb/decisions/YYYY-MM-DD-*.md` - {brief description}

### Model Impact
- Did any work this session change system architecture? (not just implementation)
- If yes, which models in `.kb/models/` need updating?
- If unsure, check: would an agent reading the model tomorrow be misled?

---

## Friction (What Was Harder Than It Should Be)

<!--
This section is critical for meta-orchestrator reflection.
Capture frustrations AS THEY HAPPEN. You'll rationalize them away later.
-->

### Tooling Friction
- [Tool gap or UX issue]
- [Missing command or awkward workflow]

### Context Friction
- [Information that should have been surfaced but wasn't]
- [Had to manually provide context that system should know]

### Skill/Spawn Friction
- [Skill guidance was unclear or wrong]
- [Spawn context was insufficient]

### Process Friction
- [Workflow that felt bureaucratic]
- [Gate that blocked without clear benefit]

*(If smooth session: "No significant friction observed")*

---

## Focus Progress

### Where We Started
- {state of focus goal at session start}
- {key blockers or open questions}

### Where We Ended
- {state of focus goal now}
- {what shifted or became clearer}

### Scope Changes
- [If focus shifted mid-session, note why]
- [If scope expanded/contracted, note the trigger]

---

## Next (What Should Happen)

**Recommendation:** {continue-focus | shift-focus | escalate | pause}

### If Continue Focus
**Immediate:** {first thing next orchestrator should do}
**Then:** {subsequent priorities}
**Context to reload:**
- {key file or artifact to read}
- {state to remember}

### If Shift Focus
**New focus:** {recommended focus}
**Why shift:** {rationale}
**Handoff for current focus:**
- {what's left}
- {who/when to resume}

### If Escalate
**Question for meta-orchestrator:** {what needs decision}
**Options:**
1. {option A} - pros/cons
2. {option B} - pros/cons
**Recommendation:** {which and why}

### If Pause
**Why pausing:** {rationale}
**Resume conditions:** {what needs to happen before resuming}
**State to preserve:**
- {local branches, uncommitted work}
- {running agents to monitor}

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- [Question 1 - why it's interesting]
- [Question 2 - why it's interesting]

**Patterns worth investigating:**
- [Pattern 1 - observed but not analyzed]
- [Pattern 2 - observed but not analyzed]

**System improvement ideas:**
- [Tooling idea]
- [Process idea]

*(If nothing emerged: "Focused session, no unexplored territory")*

---

## Session Metadata

**Agents spawned:** {count}
**Agents completed:** {count}
**Issues closed:** {list}
**Issues created:** {list}
**Issues blocked:** {list}

**Repos touched:** {list}
**PRs:** {submitted/merged}
**Commits (by agents):** {approximate count}

**Workspace:** `.orch/workspace/{workspace-name}/`
**Transcript:** `.orch/workspace/{workspace-name}/TRANSCRIPT.md`
