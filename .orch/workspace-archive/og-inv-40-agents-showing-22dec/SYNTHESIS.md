# Session Synthesis

**Agent:** og-inv-40-agents-showing-22dec
**Issue:** orch-go-vudu
**Duration:** 2025-12-22 → 2025-12-22
**Outcome:** success

---

## TLDR

Investigated why `orch status` shows 41 "active" agents when most are phantoms. Found the count comes from persistent tmux windows across all workers-* sessions - agents complete but their windows are never closed, and `orch clean` doesn't actually close them.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md` - Full investigation findings

### Files Modified
- None

### Commits
- (Investigation file committed)

---

## Evidence (What Was Observed)

- `orch status` reports 41 active agents
- Tmux windows across all workers-* sessions: 41 (exact match)
- OpenCode API `/session` returns only 4 sessions
- All 27 orch-go agent windows show `/status` prompt (idle, not working)
- `orch clean --dry-run` reports 117 cleanable workspaces but does not kill windows

### Tests Run
```bash
# Count tmux windows vs status count
$ orch status | head -5
SWARM STATUS
  Active:    41
  Completed: 0 (today)

# Verify OpenCode session count
$ curl -s http://127.0.0.1:4096/session | jq 'length'
4

# Check window activity
$ tmux capture-pane -t "workers-orch-go:2" -p | tail -1
  ~/Documents/personal/orch-go:master • 1 LSP  /status

# Test clean behavior
$ orch clean --dry-run
Found 117 cleanable workspaces
# Note: Does not kill any windows
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md` - Root cause analysis

### Decisions Made
- None (this is an investigation, not an implementation)

### Constraints Discovered
- `orch status` counts ALL workers-* tmux windows as "active" regardless of actual agent state
- `orch clean` identifies completions but doesn't close windows or reduce agent count
- Cleanup lifecycle is incomplete: spawn → work → complete has no cleanup step for windows

### Externalized via `kn`
- `kn constrain "orch status counts ALL workers-* tmux windows as active" --reason "Discovered during phantom agent investigation - status inflated by persistent windows"` - Created: kn-c42b54

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

**Issue:** Add --windows flag to orch clean to actually close tmux windows

**Skill:** feature-impl

**Context:**
```
The 41 "active" agent count comes from persistent tmux windows that are never closed.
orch clean identifies 117 cleanable workspaces but doesn't kill windows.
Add ability for orch clean to find tmux windows by beads ID and kill them.
See investigation: .kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch complete` automatically close the tmux window after verification passes?
- Should `orch status` filter to show only current-project agents instead of all workers-*?
- What's the UX impact of auto-closing windows vs manual cleanup?

**Areas worth exploring further:**
- Cross-project window aggregation - is global view the right default?
- Should there be a "preserve window" option for debugging?

**What remains unclear:**
- User preference on automatic window cleanup vs manual control

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-40-agents-showing-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md`
**Beads:** `bd show orch-go-vudu`
