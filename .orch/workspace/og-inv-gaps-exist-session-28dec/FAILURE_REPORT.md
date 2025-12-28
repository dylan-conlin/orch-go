# Failure Report

**Agent:** og-inv-gaps-exist-session-28dec
**Issue:** orch-go-2wbw
**Abandoned:** 2025-12-28 12:17:22
**Reason:** Binary was stale, session created in wrong directory

---

## Context

**Task:** [orch-go] investigation: What gaps exist in session-start context for or...

**What was attempted:**
[Brief description of what the agent was trying to do]

---

## Failure Summary

**Primary Cause:** Binary was stale, session created in wrong directory

**Details:**
[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]

---

## Progress Made

### Completed Steps
- [ ] [Step 1 - if any]

### Partial Progress
- [What was started but not finished]

### Artifacts Created
- [List any files created before abandonment]

---

## Learnings

**What worked:**
- [Things that went well before failure]

**What didn't work:**
- [Approaches that failed or caused issues]

**Root cause analysis:**
- [If known, why did this fail? External dependency? Tool issue? Scope creep? Context exhaustion?]

---

## Recovery Recommendations

**Can this be retried?** yes

**If yes, what should be different:**
- [Suggestion 1 - different approach]
- [Suggestion 2 - smaller scope]
- [Suggestion 3 - additional context needed]

**If spawning a new agent:**
```
orch spawn {skill} "{adjusted-task}" --issue orch-go-2wbw
```

**Context to provide:**
- [Key insight 1 for next agent]
- [Key insight 2 for next agent]
- [Pitfall to avoid]

---

## Session Metadata

**Workspace:** `.orch/workspace/og-inv-gaps-exist-session-28dec/`
**Beads:** `bd show orch-go-2wbw`
