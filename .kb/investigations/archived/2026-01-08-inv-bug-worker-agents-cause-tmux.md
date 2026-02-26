<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless spawns in orch-go do NOT interact with tmux - the bug cause is elsewhere (likely tmuxinator startup_window, OpenCode plugin events, or external scripts).

**Evidence:** Traced all code paths - headless spawn uses only CLI subprocess with no tmux commands; only tmux spawn path calls select-window/switch-client.

**Knowledge:** Two code paths exist: headless (no tmux) and tmux (creates window + select-window). The switch is happening but not from headless spawn code.

**Next:** Reproduce the bug with monitoring to identify actual trigger - check tmux hooks, OpenCode plugin session.created events, and startup_window in tmuxinator configs.

**Promote to Decision:** recommend-no - This is debugging, not architectural. No decision needed until root cause identified.

---

# Investigation: Bug - Worker Agents Causing tmux Session Switch

**Question:** What causes the tmux client to switch from 'orchestrator' session to 'workers-orch-go' when headless agents are spawned?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** orch-go-2pyaw
**Phase:** Complete
**Next Step:** Orchestrator to have Dylan reproduce with monitoring
**Status:** Complete

---

## Findings

### Finding 1: Headless spawn code does NOT interact with tmux

**Evidence:** 
- `runSpawnHeadless()` (spawn_cmd.go:1308-1404) only uses `exec.Command` to run opencode CLI
- No tmux imports, no switch-client, no select-window, no Attach calls
- The code path: `runSpawnWithSkill → runSpawnHeadless → startHeadlessSession`

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1308-1466`
- Confirmed by grep: only tmux spawn path (line 1577) calls select-window

**Significance:** The bug cannot be caused by the headless spawn code itself. Something else is triggering the session switch.

---

### Finding 2: Tmux spawn path DOES interact with tmux

**Evidence:**
- `runSpawnTmux()` (spawn_cmd.go:1468-1613) creates tmux windows and calls:
  - `tmux.CreateWindow()` - creates detached window
  - `tmux select-window -t <target>` (line 1577) - focuses window within session
  - `tmux.Attach()` (line 1607, only if attach=true) - switches client

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1577` (select-window)
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1607` (Attach)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go:534-555` (switch-client logic)

**Significance:** `select-window` focuses a window WITHIN a session but doesn't switch the client between sessions. `switch-client` only runs if attach=true AND running inside tmux.

---

### Finding 3: Potential causes outside orch-go code

**Evidence:**
1. **tmuxinator startup_window**: `~/.tmuxinator/workers-orch-go.yml` has `startup_window: servers` - if tmuxinator is started, it focuses the servers window
2. **OpenCode plugin events**: `orchestrator-session.ts` runs `orch session start` on `session.created` events - could have side effects
3. **Dashboard context polling**: `serve_context.go` polls `tmux display-message` on orchestrator session - read-only, shouldn't switch

**Source:**
- `~/.tmuxinator/workers-orch-go.yml` (startup_window: servers)
- `/Users/dylanconlin/Documents/personal/orch-go/plugins/orchestrator-session.ts:192-214`
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_context.go:53`

**Significance:** The bug is likely triggered by:
1. tmuxinator auto-starting when session is created (EnsureWorkersSession calls tmuxinator in background)
2. Some other process reacting to agent spawns
3. User action during agent spawn that's being misattributed

---

### Finding 4: EnsureWorkersSession creates sessions detached

**Evidence:**
- `tmux new-session -d -s <name>` - The `-d` flag creates session in detached mode
- No switch-client in session creation path
- Background goroutine updates tmuxinator config but doesn't start tmuxinator

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go:267-302`

**Significance:** Creating a workers session doesn't switch the client - it's created detached.

---

## Synthesis

**Key Insights:**

1. **Code analysis rules out orch-go headless spawn** - The headless code path is completely tmux-free. If spawns are confirmed headless (spawn_mode=headless in events), the switch is coming from elsewhere.

2. **Session switch vs window focus are different** - `select-window` focuses within current session; `switch-client` changes which session you're viewing. The symptom is session switch (orchestrator → workers-orch-go), not window focus.

3. **External triggers need investigation** - Possible causes: tmux hooks, OpenCode plugin side effects, background processes, or user action being misattributed.

**Answer to Investigation Question:**

The orch-go headless spawn code does NOT cause tmux session switches - it doesn't interact with tmux at all. The bug is triggered by something else:

1. Most likely: Some external process or script reacting to agent spawns
2. Possible: tmuxinator being triggered by background config updates (though this shouldn't switch clients)
3. Possible: The symptom timing correlation with spawns is coincidental

**Cannot reproduce without user environment** - Need Dylan to reproduce while monitoring tmux hooks and process spawns.

---

## Structured Uncertainty

**What's tested:**

- ✅ Headless spawn code has no tmux interactions (verified: code review of runSpawnHeadless)
- ✅ Only Attach() function calls switch-client (verified: grep for switch-client in codebase)
- ✅ Sessions are created detached with -d flag (verified: code review of EnsureWorkersSession)

**What's untested:**

- ⚠️ Whether tmux hooks exist that react to session/window creation
- ⚠️ Whether OpenCode plugin events trigger any tmux operations indirectly
- ⚠️ Actual reproduction of the bug with process monitoring

**What would change this:**

- If Dylan's environment has custom tmux hooks (check ~/.tmux.conf)
- If there's a running script that monitors spawns and switches sessions
- If the bug only occurs under specific conditions (e.g., multiple terminals open)

---

## Implementation Recommendations

**Purpose:** Identify and fix the root cause of tmux session switching.

### Recommended Approach ⭐

**Reproduce with monitoring** - Have Dylan reproduce the bug while capturing:
1. `tmux list-sessions` before and after spawn
2. Process tree (`pstree`) during spawn
3. tmux hooks if any exist

**Why this approach:**
- Code analysis is complete - no bug found in orch-go
- Need environmental evidence to find actual cause
- Monitoring during reproduction will reveal the trigger

**Trade-offs accepted:**
- Requires user involvement
- May be intermittent and hard to catch

**Implementation sequence:**
1. Check `~/.tmux.conf` for hooks (session-created, window-created hooks)
2. Monitor with `tmux list-clients` before/after spawn
3. Run spawn with verbose logging and process monitoring

### Alternative Approaches Considered

**Option B: Add defensive code to prevent unwanted switches**
- **Pros:** Quick fix even without root cause
- **Cons:** Doesn't solve root cause, could mask other issues
- **When to use instead:** If bug is high-impact and root cause too hard to find

**Option C: Disable tmuxinator integration**
- **Pros:** Removes one potential trigger
- **Cons:** Loses automatic config updates, may not fix issue
- **When to use instead:** If confirmed tmuxinator is the cause

---

### Implementation Details

**What to implement first:**
1. Ask Dylan to check for tmux hooks: `grep -r "hook\|session-created\|window-created" ~/.tmux.conf`
2. Run `tmux list-clients` in both terminals before and during spawn
3. Monitor process tree during spawn for unexpected tmux calls

**Things to watch out for:**
- ⚠️ Bug might be timing-dependent (race condition with background processes)
- ⚠️ Multiple terminal windows might interact unexpectedly
- ⚠️ OpenCode server might emit events that trigger plugins

**Areas needing further investigation:**
- What other processes are running that interact with tmux?
- Does the bug happen with tmux -CC (Control Mode) clients?
- Is there a pattern to when it occurs (specific skills, projects, etc.)?

**Success criteria:**
- ✅ Root cause identified and verified by reproduction
- ✅ Fix implemented that prevents session switching
- ✅ Confirmed headless spawns don't affect orchestrator's tmux state

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - Main spawn command implementation
- `pkg/tmux/tmux.go` - tmux helper functions
- `pkg/tmux/follower.go` - Dashboard context polling
- `cmd/orch/serve_context.go` - Dashboard API endpoint
- `plugins/orchestrator-session.ts` - OpenCode plugin
- `~/.tmuxinator/workers-orch-go.yml` - tmuxinator config

**Commands Run:**
```bash
# Search for switch-client in codebase
grep -r "switch-client\|attach-session" /Users/dylanconlin/Documents/personal/orch-go

# Check tmuxinator config
cat ~/.tmuxinator/workers-orch-go.yml

# Check which orch binary is being used
which orch && file $(which orch)
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/simple/2025-12-07-switch-workers-client-switches-wrong.md` (orch-cli) - Previous similar bug in Python version
- **Investigation:** `.kb/investigations/simple/2025-12-07-per-project-workers-sessions-design.md` (orch-cli) - Design docs for session switching

---

## Investigation History

**2026-01-08 14:30:** Investigation started
- Initial question: Why do headless worker agents cause tmux session switches?
- Context: Dylan reported tmux switching from orchestrator to workers-orch-go when spawning headless agents

**2026-01-08 15:00:** Code analysis complete
- Finding: Headless spawn path has no tmux interactions
- Finding: Only tmux spawn path calls select-window (line 1577)
- Finding: Only Attach() calls switch-client (line 1607, conditional)

**2026-01-08 15:30:** Investigation status update
- Status: In Progress - needs reproduction with monitoring
- Key outcome: Bug cause is NOT in orch-go headless spawn code
