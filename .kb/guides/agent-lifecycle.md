# Agent Lifecycle

**Purpose:** Single authoritative reference for how agents move through spawn → work → complete → dashboard. Read this before debugging lifecycle issues.

**Last verified:** Jan 4, 2026

---

## The Flow

```
orch spawn                    bd comment "Phase: Complete"           orch complete
     │                                    │                                │
     ▼                                    ▼                                ▼
┌─────────┐    agent works    ┌──────────────────┐    orchestrator    ┌─────────┐
│ Spawned │ ───────────────►  │ Phase: Complete  │ ────────────────►  │ Closed  │
└─────────┘                   └──────────────────┘                    └─────────┘
     │                                    │                                │
     ▼                                    ▼                                ▼
  Creates:                           Agent reports:                   Orchestrator:
  - OpenCode session                 - bd comment with phase          - Verifies work
  - Beads issue (unless --no-track)  - SYNTHESIS.md (full tier)       - Closes beads issue
  - Workspace directory              - Git commits                    - Rebuilds if needed
```

---

## Source of Truth

**Beads is the source of truth for agent status.**

| Question | Source | NOT this |
|----------|--------|----------|
| Is agent complete? | Beads issue status = "closed" | OpenCode session exists |
| Is agent working? | Beads comments show recent Phase updates | Dashboard shows "active" |
| Did agent finish? | Phase: Complete comment exists | Session went idle |

**Key insight:** OpenCode sessions persist to disk indefinitely. An OpenCode session existing means nothing about whether the agent is done. Only beads matters.

---

## Dashboard Status Logic

The dashboard (`orch serve`) determines status in this order:

1. **Check beads issue status** - If closed → show "completed"
2. **Check Phase: Complete comment** - If present → show "completed" 
3. **Check SYNTHESIS.md** - If exists in workspace → show "completed"
4. **Fall back to session state** - active/idle based on recent activity

**If dashboard shows wrong status:**
1. Check beads: `bd show <id> --json | jq '.status'`
2. If beads says closed, dashboard should show completed (refresh browser)
3. If beads says open but agent is done, run `orch complete <id>`

---

## Common Problems

### "Dashboard shows agent as active but it's done"

**Cause:** `orch complete` wasn't run, so beads issue is still open.

**Fix:** Run `orch complete <id>`. If that's blocked, check what gate is blocking (see below).

**NOT the fix:** Deleting OpenCode sessions. That treats the symptom, not the cause.

### "orch complete is blocked / requires flags"

**Cause:** Gates were added that require verification before closing.

**See:** `.kb/guides/completion-gates.md` for full reference on all 11 gates.

**Quick bypass:** `orch complete <id> --force` skips all verification gates.

**If completion requires multiple flags to work, that's a smell.** The gates may be adding friction without value.

### "Agent went idle but didn't report Phase: Complete"

**Cause:** Agent ran out of context, crashed, or didn't follow the completion protocol.

**This is expected behavior.** Session idle ≠ work complete. Only agents that explicitly run `bd comment <id> "Phase: Complete"` are considered done.

**Fix:** Check workspace for what agent accomplished, then either:
- `orch complete <id> --force` if work is done
- `orch abandon <id>` if work is incomplete

### "Lots of zombie agents in dashboard"

**Cause:** Agents finished but `orch complete` was never run (gates blocked it, or orchestrator didn't complete them).

**Fix:** Complete or abandon each one. Don't delete OpenCode sessions as a workaround.

**Prevention:** Complete agents promptly. Don't let them accumulate.

---

## Key Decisions (from kn)

These are settled. Don't re-investigate:

- **Dashboard uses beads as source of truth** - not session state
- **SSE busy→idle cannot detect completion** - agents go idle for many reasons
- **Phase: Complete is the only reliable signal** - from beads comments
- **SYNTHESIS.md is fallback for untracked agents** - when no beads issue exists

---

## What Lives Where

| Thing | Location | Lifecycle |
|-------|----------|-----------|
| OpenCode session | OpenCode's internal storage | Persists until deleted |
| Beads issue | `.beads/` | Created at spawn, closed at complete |
| Workspace | `.orch/workspace/<name>/` | Created at spawn, persists forever |
| SPAWN_CONTEXT.md | Workspace | Created at spawn |
| SYNTHESIS.md | Workspace | Created by agent before completion |

---

## Debugging Checklist

Before spawning an investigation about lifecycle issues:

1. **Check kb:** `kb context "agent lifecycle"` or `kb context "completion"`
2. **Check this doc:** You're reading it
3. **Check beads:** `bd show <id>` - what's the actual status?
4. **Check recent post-mortems:** `.kb/post-mortems/`

If those don't answer your question, then investigate. But update this doc with what you learn.

---

## History

- **Jan 4, 2026:** Created after spending 1 hour debugging a problem that was already documented in kn. Synthesized from 20+ investigations about sessions/completion/lifecycle.
- **Jan 4, 2026:** Disabled repro verification and dependency check gates - they blocked completion without clear benefit.
