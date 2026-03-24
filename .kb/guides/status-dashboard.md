# Status and Dashboard

**Purpose:** Single authoritative reference for how agent status is determined and displayed. Read this before debugging "wrong status" or dashboard issues.

**Last verified:** Jan 4, 2026

---

## Architecture

```
                    ┌─────────────────┐
                    │   Dashboard     │  (http://localhost:5188)
                    │   (SvelteKit)   │
                    └────────┬────────┘
                             │ polls
                             ▼
                    ┌─────────────────┐
                    │   orch serve    │  (http://localhost:3348)
                    │   (Go API)      │
                    └────────┬────────┘
                             │ queries
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
      ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
      │  OpenCode   │ │   Beads     │ │  Workspaces │
      │  Sessions   │ │   Issues    │ │  (.orch/)   │
      └─────────────┘ └─────────────┘ └─────────────┘
```

---

## Source of Truth

**Beads is the source of truth for agent status.**

| Question | Check | NOT this |
|----------|-------|----------|
| Is agent complete? | Beads issue closed | OpenCode session exists |
| What phase is agent in? | Beads comments | Session idle time |
| Did agent finish successfully? | Phase: Complete comment | SYNTHESIS.md exists |

**Key insight:** OpenCode sessions persist indefinitely. A session existing says nothing about whether the agent is done. Only beads matters.

---

## Status Determination Logic

`orch serve` determines status in this priority order:

1. **Beads issue closed** → `completed`
2. **Phase: Complete comment** → `completed`
3. **SYNTHESIS.md exists** → `completed` (fallback for untracked)
4. **Recent activity (< 1 min)** → `active`
5. **No recent activity** → `idle`

```go
// Simplified logic from serve_agents.go
if beadsIssue.Status == "closed" {
    return "completed"
}
if hasPhaseCompleteComment(beadsID) {
    return "completed"
}
if hasSynthesisMD(workspace) {
    return "completed"
}
if timeSinceUpdate < 1*time.Minute {
    return "active"
}
return "idle"
```

---

## CLI vs Dashboard

| Command | Shows | Source |
|---------|-------|--------|
| `orch status` | Active agents with phase, tokens, risk | OpenCode API + beads |
| `orch serve` API | All agents for dashboard | OpenCode + beads + workspaces |
| Dashboard | Visual agent cards | Polls orch serve API |

**`orch status`** is authoritative for CLI users. If it disagrees with dashboard, trust `orch status`.

---

## Common Problems

### "Dashboard shows agent as active but orch status shows nothing"

**Cause:** Dashboard is caching old state or browser hasn't refreshed.

**Fix:** Hard refresh browser (Cmd+Shift+R). If still wrong, restart `orch serve`:
```bash
pkill -f "orch serve" && orch serve &
```

### "Dashboard shows agent as active but it's done"

**Cause:** `orch complete` wasn't run, so beads issue is still open.

**Fix:** Run `orch complete <id>`.

**NOT the fix:** Deleting OpenCode sessions. The dashboard checks beads, not sessions.

### "Agent shows as idle but is actually working"

**Cause:** The "activity" threshold (1 minute) was exceeded between tool calls.

**This is normal.** Agents go idle during thinking, loading, waiting for tools. Only Phase: Complete indicates actual completion.

### "Lots of completed agents cluttering dashboard"

**Cause:** Completed agents are shown for reference.

**Fix:** Dashboard has filters. Or use `orch clean` to archive old workspaces (doesn't delete, just moves).

### "Status shows 'at-risk' for healthy agent"

**Cause:** Agent has been idle > 30 minutes without Phase: Complete.

**Action:** Check if agent is stuck (out of context), completed but didn't report, or legitimately thinking. Use `orch send <id> "status?"` to probe.

---

## Port Reference

| Port | Service | Purpose |
|------|---------|---------|
| 4096 | OpenCode | Claude API proxy, session management |
| 3348 | orch serve | Dashboard API |
| 5188 | Dashboard | SvelteKit dev server (vite) |

**If dashboard doesn't load:**
1. Check OpenCode: `curl http://localhost:4096/session`
2. Check orch serve: `curl http://localhost:3348/api/agents`
3. Check dashboard: `curl http://localhost:5188`

Or use `orch doctor` which checks all three.

---

## API Endpoints

| Endpoint | Returns |
|----------|---------|
| `GET /api/agents` | All agents with status, phase, tokens |
| `GET /api/beads/stats` | Issue counts (ready, blocked, open) |
| `GET /api/usage` | Claude Max usage percentage |
| `GET /api/daemon/status` | Daemon running state |
| `GET /api/briefs/:beads-id` | BRIEF.md content and read status for review queue |
| `POST /api/briefs/:beads-id` | Mark a brief as read (UI-only state, not comprehension gate) |

---

## Key Decisions (from kn)

- **Dashboard uses beads as source of truth** - not session time
- **Phase: Complete is the only reliable signal** - SSE busy→idle has false positives
- **SYNTHESIS.md is fallback for untracked** - when no beads issue exists
- **orch doctor checks all services** - OpenCode, orch serve, beads daemon

---

## Debugging Checklist

Before spawning an investigation about status/dashboard issues:

1. **Check kb:** `kb context "dashboard status"`
2. **Check this doc:** You're reading it
3. **Check beads:** `bd show <id>` - what's the actual status?
4. **Check CLI:** `orch status` - what does it show?
5. **Check services:** `orch doctor` - are they healthy?

If those don't answer your question, then investigate. But update this doc with what you learn.
