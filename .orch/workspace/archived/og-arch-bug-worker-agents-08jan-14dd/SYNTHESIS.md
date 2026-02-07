# Session Synthesis

**Agent:** og-arch-bug-worker-agents-08jan-14dd
**Issue:** orch-go-2pyaw
**Duration:** 2026-01-08 ~14:30 → ongoing
**Outcome:** partial - Root cause not in orch-go code; needs reproduction with monitoring

---

## TLDR

Investigated bug where tmux switches from 'orchestrator' to 'workers-orch-go' during headless spawns. **Finding: orch-go headless spawn code does NOT interact with tmux at all** - the bug cause is external (possibly tmux hooks, OpenCode plugins, or background scripts). Needs reproduction with monitoring to identify actual trigger.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-08-inv-bug-worker-agents-cause-tmux.md` - Investigation documenting code analysis and findings

### Files Modified
- None

### Commits
- None yet (no code changes needed - this was pure investigation)

---

## Evidence (What Was Observed)

- **Headless spawn path** (`runSpawnHeadless` at spawn_cmd.go:1308-1404) uses only `exec.Command` to run opencode CLI - no tmux imports or calls
- **Only tmux spawn path** has tmux interactions:
  - `tmux select-window` at spawn_cmd.go:1577 (focuses window within session)
  - `tmux.Attach()` at spawn_cmd.go:1607 (only if attach=true)
- **switch-client only called in Attach()** - and Attach() is only called when attach flag is true in tmux mode
- **EnsureWorkersSession creates sessions with `-d` (detached)** - doesn't switch client
- **Grep confirmed**: only 2 places have switch-client: tmux.go:537 and tmux_test.go

### Tests Run
```bash
# Search for tmux switch commands in codebase
grep -r "switch-client\|attach-session" /Users/dylanconlin/Documents/personal/orch-go --include="*.go"
# Found: Only in pkg/tmux/tmux.go (production) and tests

# Check headless spawn function
# Result: No tmux imports, no switch-client calls
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-bug-worker-agents-cause-tmux.md` - Full investigation with D.E.K.N. summary

### Decisions Made
- The bug is NOT in orch-go headless spawn code - need to look elsewhere

### Constraints Discovered
- Headless spawn is completely tmux-agnostic by design
- Session switch (orchestrator→workers) requires switch-client which only runs in Attach()
- select-window only changes focus WITHIN a session, not between sessions

### Externalized via `kn`
- Not yet - will add constraint when root cause identified

---

## Next (What Should Happen)

**Recommendation:** escalate - Need Dylan to reproduce with monitoring

### If Escalate
**Question:** What external process/script is causing the tmux session switch?

**Investigation steps needed:**
1. Check `~/.tmux.conf` for hooks (session-created, window-created)
2. Run `tmux list-clients` before and after spawn to see what changes
3. Monitor process tree during spawn for unexpected tmux calls
4. Check if bug correlates with specific conditions (multiple terminals, specific projects, etc.)

**Options:**
1. **Have Dylan reproduce with monitoring** - Run spawn while watching `tmux list-clients` in another terminal
2. **Add defensive code** - Make orch explicitly NOT switch sessions even if something else tries
3. **Check tmux hooks** - Look for hooks in ~/.tmux.conf that react to session/window events

**Recommendation:** Option 1 first - we need environmental evidence since code analysis is complete. If high-impact, Option 2 as stopgap.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does Dylan have custom tmux hooks that could trigger on session/window creation?
- Are there any background scripts watching for agent spawns?
- Could OpenCode plugins indirectly trigger tmux operations?

**Areas worth exploring further:**
- The orchestrator-session.ts plugin runs `orch session start` on session.created - could this have side effects?
- tmuxinator's startup_window directive - does it ever auto-switch clients?

**What remains unclear:**
- Exact timing of when the switch happens (during spawn, after spawn, etc.)
- Whether it happens with all spawns or only specific conditions
- Whether multiple terminal windows affect the behavior

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-20250514
**Workspace:** `.orch/workspace/og-arch-bug-worker-agents-08jan-14dd/`
**Investigation:** `.kb/investigations/2026-01-08-inv-bug-worker-agents-cause-tmux.md`
**Beads:** `bd show orch-go-2pyaw`
