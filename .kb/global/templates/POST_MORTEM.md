# Agent Investigation: [Brief Description]

**Date:** YYYY-MM-DD
**Agent ID:** [From orch status or "N/A" for orchestrator issues]
**Project:** [Project name]
**Skill Used:** [Skill that was spawned]
**Type:** [Bug | Pattern | Process Gap | Tooling Issue]

---

## Quick Summary (30 seconds)

**What happened:** [One sentence - agent did X but Y happened]

**Root cause:** [One sentence - because Z was missing/wrong]

**Recommendation:** [One sentence - should do W]

**Action:** [Added to ROADMAP | Applied immediately | Won't fix - see Resolution section]

---

## What Went Wrong
[1-2 sentences describing the issue caught at verification]

## Root Cause
[What spawn context was missing/unclear/incorrect]
- Missing context: [e.g., "Architecture assumptions not stated"]
- Unclear authority: [e.g., "Agent didn't know when to escalate"]
- Missing verification: [e.g., "No checklist to self-verify correctness"]
- Other: [Skill instructions, deliverables, etc.]

## What Agent Should Have Known/Checked
[What information would have prevented this issue]

## Evidence
[Workspace location, git commits, verification findings]

## Impact Assessment

**Frequency:** [Every spawn | Common (30-50%) | Occasional (10-30%) | Rare (<10%) | One-time]

**Severity:**
- [ ] Critical - Blocks work completely
- [ ] High - Requires manual intervention every time
- [ ] Medium - Causes friction/wasted time
- [ ] Low - Minor inconvenience

**Affected scope:**
- [ ] All agents (global pattern)
- [ ] Specific skill: [skill-name]
- [ ] Specific project: [project-name]
- [ ] Meta-orchestration only

---

## Recommendations

[What should change - descriptive, not a task list]

**Changes needed:**
1. **Spawn Template:** [What section/field needs updating]
2. **Skill ([skill-name]):** [What instruction/verification needs adding]
3. **Documentation:** [What needs documenting in CLAUDE.md or docs/]

**Estimated effort:** [< 30 min | 1-2 hours | Half day | Full day]

**Priority:** [P1 - Critical | P2 - High | P3 - Medium | P4 - Low]

---

## Suggested ROADMAP Entry

[Copy-paste ready text for adding to .orch/ROADMAP.org Untriaged section]

```org
** TODO [Task title - keep concise]
Mode: [suggested skill]
:PROPERTIES:
:Created: YYYY-MM-DD
:Investigation: .orch/knowledge/spawning-lessons/YYYY-MM-DD-brief-description.md
:Priority: [1-4]
:Estimated-effort: [from above]
:Type: [bug | pattern | infrastructure | documentation]
:END:

**Problem:** [Copy from "What Went Wrong" - 1-2 sentences]

**Goal:** [What success looks like - specific, testable]

**Context:** See linked investigation for full analysis and evidence.

**Implementation notes:** [Any hints for agent doing the work - optional]
```

## Classification
- [ ] Global (spawn mechanics) → ~/.orch/knowledge/spawning-lessons/
- [ ] Project-specific → {project}/.orch/knowledge/agent-lessons/

**Rationale:** [Why this classification - helps future sessions understand scope]

---

## Resolution Status

**Added to ROADMAP:**
- [ ] No - Not yet triaged
- [ ] Yes - .orch/ROADMAP.org:L#### (line number)
- [ ] Won't Fix - See reason below

**Current status:**
- [ ] 🔴 Unplanned - Investigation complete, not yet added to ROADMAP
- [ ] 🟡 Planned - ROADMAP item created, work not started
- [ ] 🔵 In Progress - Work started (ROADMAP item TODO)
- [ ] 🟢 Complete - ROADMAP item marked DONE
- [ ] ⚫ Won't Fix - Decided not to implement

**Completion details:** [Fill when status changes to Complete]
- **Date completed:** YYYY-MM-DD
- **Implementation:** [Commit hash(es) and brief summary]
- **Verification:** [How fix was verified]

**Won't fix reason:** [Fill if status is Won't Fix]
- [Why decided not to implement - one-time issue, already fixed elsewhere, etc.]

---

## Usage Notes

**When to create investigation:**
- Verification catches successful-but-wrong implementation
- Agent blocked due to missing spawn context
- Agent asks questions that should have been in spawn prompt
- Unclear authority boundaries cause incorrect decisions
- Recurring patterns that cause friction
- Process gaps that slow down orchestration

**Scope - capture when:**
- Missing context in spawn prompt (architectural, domain)
- Unclear authority boundaries (decide vs escalate)
- Missing verification criteria (can't self-verify)
- Ambiguous deliverables (wrong artifact type/location)
- Skill instruction gaps (unclear, incomplete)
- Tooling friction (commands fail, unclear errors)

**Don't capture:**
- External tooling failures (not our system)
- User requirement changes mid-work
- Agent implementation bugs (code logic errors)
- One-off edge cases with no pattern

**Classification heuristic:**
- **Global:** Spawn template structure, skill instructions, authority patterns, verification approaches
- **Project:** Domain knowledge, project architecture, project-specific tools/constraints

---

## Workflow

### 1. Investigation Phase (5-10 min)

1. **Create investigation** from this template
2. **Fill all sections** through "Suggested ROADMAP Entry"
3. **Save** to appropriate location:
   - Global: `~/.orch/knowledge/spawning-lessons/YYYY-MM-DD-description.md`
   - Project: `{project}/.orch/knowledge/agent-lessons/YYYY-MM-DD-description.md`
4. **Mark status** as 🔴 Unplanned

### 2. Triage Decision

**Option A: Quick fix (< 30 min)**
- Apply changes immediately (same session)
- Update Resolution Status: 🟢 Complete
- Add commit hash and verification notes
- Investigation is done

**Option B: Requires planning (> 30 min)**
- Copy "Suggested ROADMAP Entry" section
- Add to `.orch/ROADMAP.org` Untriaged section
- Update investigation Resolution Status: 🟡 Planned
- Add ROADMAP line number to investigation
- Investigation is done (task tracking happens in ROADMAP)

**Option C: Won't fix**
- Mark status: ⚫ Won't Fix
- Document reason in "Won't fix reason" field
- Investigation is done

### 3. Implementation (happens via ROADMAP)

- When ROADMAP item marked DONE, return to investigation
- Update Resolution Status: 🟢 Complete
- Add completion details (date, commits, verification)
- Investigation lifecycle complete

### 4. Querying Investigations

**Find unplanned investigations:**
```bash
rg "🔴 Unplanned" ~/.orch/knowledge/spawning-lessons/
```

**Find investigations linked to ROADMAP:**
```bash
rg ":Investigation:" .orch/ROADMAP.org
```

**Find completed investigations:**
```bash
rg "🟢 Complete" ~/.orch/knowledge/spawning-lessons/
```

---

## Key Insight

**Investigations are documentation, not tasks.**

- Investigation = Analysis + Evidence + Recommendations (read-only after triage)
- ROADMAP = Tasks + Status + Implementation tracking (active work)
- This template creates investigations that link to ROADMAP, not replace it
