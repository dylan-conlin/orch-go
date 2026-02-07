# Session Synthesis

**Agent:** {workspace-name}
**Issue:** {beads-id}
**Duration:** {start-time} → {end-time}
**Outcome:** {success | partial | blocked | failed}

---

<!--
## How to Use This Template (Progressive Synthesis)

**Fill this file AS YOU WORK, not at the end.**

The anti-pattern: "I'll synthesize everything when I'm done" → leads to lost details,
incomplete sections, and the cognitive load of reconstructing what you observed.

**Progressive documentation pattern:**
1. **BEFORE:** Fill metadata (agent, issue, duration start)
2. **DURING:** Add to Delta and Evidence sections as you go
3. **AFTER:** Synthesize Knowledge and Next sections
4. **COMMIT:** Final review, fill TLDR, update outcome

**Section timing:**
| Section | When to Fill |
|---------|--------------|
| TLDR | Last (after you know what happened) |
| Delta | During work (as you create/modify files) |
| Evidence | During work (as you observe things) |
| Knowledge | After implementation (patterns noticed) |
| Issues Created | During/after work (as you discover work) |
| Next | After validation (what should happen) |
| Unexplored | Anytime (capture questions as they emerge) |

**Why this matters:**
- Details are lost if not captured immediately
- "I'll remember" → you won't (session amnesia)
- Progressive fill reduces end-of-session cognitive load
- Sections like "Unexplored Questions" need real-time capture
-->

## TLDR

[1-2 sentence summary. What was the goal? What was achieved?]

---

## Delta (What Changed)

### Files Created
- `path/to/file.go` - Brief description

### Files Modified
- `path/to/existing.go` - What was changed

### Commits
- `abc1234` - Commit message summary

---

## Evidence (What Was Observed)

- Observation 1 with source reference (file:line or command output)
- Observation 2 with source reference
- Key finding that informed decisions

### Tests Run
```bash
# Command and result
go test ./... 
# PASS: all tests passing
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/YYYY-MM-DD-*.md` - Brief description

### Decisions Made
- Decision 1: [choice] because [rationale]

### Constraints Discovered
- Constraint 1 - Why it matters

### Externalized via `kn`
- `kn decide "X" --reason "Y"` - [if applicable]
- `kn constraint "X" --reason "Y"` - [if applicable]
- `kn tried "X" --failed "Y"` - [if applicable]

---

## Issues Created

**Discovered work tracked during this session:**

- `orch-go-XXXXX` - Brief description of bug/task/feature/question
- `orch-go-YYYYY` - Brief description

*(If no discovered work: "No discovered work during this session")*

**Note:** Every session should either create issues for discovered work OR explicitly note that none was found. Bugs, tech debt, enhancement ideas, and strategic questions should all be tracked.

---

## Next (What Should Happen)

**Recommendation:** {close | spawn-follow-up | escalate | resume}

### If Close
- [ ] All deliverables complete
- [ ] Tests passing
- [ ] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete {issue-id}`

### If Spawn Follow-up
**Issue:** {new-issue-title}
**Skill:** {recommended-skill}
**Context:**
```
{Brief context for next agent - 2-3 sentences max}
```

### If Escalate
**Question:** {what needs decision from orchestrator}
**Options:**
1. {option A} - pros/cons
2. {option B} - pros/cons

**Recommendation:** {which option and why}

### If Resume
**Next Step:** {what to do when resuming}
**Blocker:** {what prevented completion}
**Context to Reload:**
- {key file to re-read}
- {state to remember}

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- [Question 1 - why it's interesting]
- [Question 2 - why it's interesting]

**Areas worth exploring further:**
- [Area 1]
- [Area 2]

**What remains unclear:**
- [Uncertainty 1]
- [Uncertainty 2]

*(If nothing emerged, note: "Straightforward session, no unexplored territory")*

---

## Session Metadata

**Skill:** {skill-name}
**Model:** {model-used}
**Workspace:** `.orch/workspace/{workspace-name}/`
**Investigation:** `.kb/investigations/YYYY-MM-DD-*.md`
**Beads:** `bd show {issue-id}`
