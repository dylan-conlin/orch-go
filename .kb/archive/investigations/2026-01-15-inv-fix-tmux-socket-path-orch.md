<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** orch serve fails to query orchestrator session when running inside overmind's tmux because tmux commands default to overmind's socket instead of main tmux socket.

**Evidence:** Manual testing confirmed `tmux display-message -t orchestrator` returns empty from overmind but works with explicit `-S /tmp/tmux-501/default` socket specification.

**Knowledge:** Default tmux behavior uses $TMUX environment variable to determine server; when inside overmind's tmux, this targets wrong server. Main socket path follows `/tmp/tmux-$(id -u)/default` pattern on macOS.

**Next:** Implement global socket detection in findTmux() that prepends `-S` flag to target main tmux by default, with tmuxCommandCurrent() override for operations on current window.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural pattern)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix Tmux Socket Path Orch

**Question:** How should orch-go detect and use the correct tmux socket when running inside overmind's tmux server?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Two tmux servers are running simultaneously

**Evidence:** 
- Main tmux server: pid 851, socket at `/tmp/tmux-501/default`
- Overmind tmux server: pid 55715, socket at `/tmp/tmux-501/overmind-orch-go-wj2fhnaZaFtAryPSTvKc2`
- Environment variable `$TMUX` shows: `/private/tmp/tmux-501/overmind-orch-go-wj2fhnaZaFtAryPSTvKc2,55715,0`

**Source:** 
- `ps aux | grep tmux` - shows both servers
- `echo $TMUX` - confirms current context is overmind's tmux
- `ls -la /tmp/tmux-501/` - shows all socket files

**Significance:** When orch serve runs inside overmind, it inherits the overmind tmux environment, causing all tmux commands to target the wrong server.

---

### Finding 2: Commands without explicit socket fail to find orchestrator session

**Evidence:**
- `tmux display-message -t orchestrator -p '#{window_index}'` returns empty (FAILS)
- `tmux -S /tmp/tmux-501/default display-message -t orchestrator -p '#{window_index}'` returns "2" (WORKS)
- `tmux list-sessions` from overmind shows only "orch-go" session, not main tmux sessions

**Source:** 
- Manual testing from within overmind tmux pane
- Commands tested in current agent session

**Significance:** This confirms the root cause - tmux commands need explicit `-S` socket specification to target main tmux when running inside overmind.

---

### Finding 3: Most tmux operations target main tmux, not current tmux

**Evidence:**
- `GetTmuxCwd("orchestrator")` - targets orchestrator session in main tmux (follower.go:243, 261)
- `SessionExists("orchestrator")` - checks for sessions in main tmux (tmux.go:315)
- `GetCurrentWindowName()` - targets current window (tmux.go:73) - RARE EXCEPTION
- `RenameCurrentWindow()` - operates on current window (tmux.go:107, 123) - RARE EXCEPTION

**Source:**
- Analyzed all 9 call sites of `tmuxCommand()` in codebase
- pkg/tmux/tmux.go lines 55-876
- pkg/tmux/follower.go lines 243, 261

**Significance:** The default behavior should be to target the main tmux socket, since that's where orchestrator and worker sessions live. Only 2 functions (GetCurrentWindowName, RenameCurrentWindow) need current socket.

---

## Synthesis

**Key Insights:**

1. **Socket detection is environment-dependent** - The correct socket path depends on whether we're querying main tmux (orchestrator/workers) or current tmux (process's own window). Main tmux socket can be discovered via `/tmp/tmux-$(id -u)/default` pattern.

2. **Default should target main tmux** - 7 out of 9 tmuxCommand() call sites target main tmux for orchestrator/worker operations. Only 2 call sites (GetCurrentWindowName, RenameCurrentWindow) need current tmux.

3. **Detection strategy: Check $TMUX environment** - If $TMUX contains "overmind", we're inside overmind's tmux and need to explicitly specify main socket. Otherwise, default behavior is correct.

**Answer to Investigation Question:**

orch-go should detect the main tmux socket path and use it by default in `tmuxCommand()`. When running inside overmind's tmux (detected via $TMUX environment variable), prepend `-S /tmp/tmux-$(id -u)/default` to tmux command arguments. Only `GetCurrentWindowName()` and `RenameCurrentWindow()` need special handling to use the current socket.

---

## Decision Forks

### Fork 1: Which operations need main socket vs current socket?

**Options:**
- A: All operations use main socket
- B: Operations with `-t sessionName` use main socket, operations without use current
- C: Explicit per-call-site decision

**Substrate says:**
- Evidence from codebase: 7/9 call sites target main tmux (orchestrator/worker sessions)
- Only GetCurrentWindowName/RenameCurrentWindow operate on current context

**RECOMMENDATION:** Option A (all operations use main socket by default) with special cases for GetCurrentWindowName/RenameCurrentWindow.

**Trade-off accepted:** GetCurrentWindowName/RenameCurrentWindow need special handling (use separate function or flag).

---

### Fork 2: How to implement socket detection?

**Options:**
- A: Detect in tmuxCommand() and auto-add socket based on args
- B: Create tmuxCommandMain() and tmuxCommandCurrent() functions
- C: Add socket parameter to tmuxCommand(socket string, args...)
- D: Cache socket path globally in findTmux()

**Substrate says:**
- Principle: "Coherence over patches" - choose approach that makes intent clear across many call sites
- Evidence: Default behavior should target main tmux (7/9 call sites)

**RECOMMENDATION:** Option D (cache socket path globally) - detect main socket in findTmux() when running inside overmind, and use it by default. Add tmuxCommandCurrent() for the 2 exceptions.

**Trade-off accepted:** Global state (cached socket path), but provides transparent fix for all call sites.

---

## Structured Uncertainty

**What's tested:**

- ✅ Main tmux commands fail from overmind context (verified: ran tmux display-message)
- ✅ Explicit socket specification fixes the issue (verified: added -S flag, command succeeded)
- ✅ Two tmux servers running simultaneously (verified: ps aux | grep tmux)

**What's untested:**

- ⚠️ Performance impact of socket detection (not benchmarked)
- ⚠️ Behavior on Linux systems (only tested on macOS)
- ⚠️ Behavior when main tmux uses non-default socket name

**What would change this:**

- Finding would be wrong if tmux commands already worked from overmind (they don't)
- Socket path pattern may differ on other systems (needs cross-platform testing)
- If GetCurrentWindowName is called from main tmux context, it will still work (same socket)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Global Socket Detection with Current Context Override** - Detect main tmux socket path in findTmux(), use it by default in tmuxCommand(), and provide tmuxCommandCurrent() for operations needing current socket.

**Why this approach:**
- Fixes all 7 main-tmux call sites transparently (no code changes)
- Makes intent explicit for the 2 current-tmux call sites (requires refactor to tmuxCommandCurrent)
- Maintains backward compatibility when not running inside overmind
- Handles the common case (main tmux operations) as the default

**Trade-offs accepted:**
- Global state (cached socket path) - acceptable because socket path doesn't change during process lifetime
- Need to update 2 call sites to use tmuxCommandCurrent() - small refactor, makes intent clearer

**Implementation sequence:**
1. Add detectMainSocket() function to determine main tmux socket path
2. Modify findTmux() to cache both tmux binary path and main socket path
3. Update tmuxCommand() to prepend -S socket when main socket is detected
4. Add tmuxCommandCurrent() function that explicitly skips socket specification
5. Update GetCurrentWindowName() and RenameCurrentWindow() to use tmuxCommandCurrent()
6. Test with reproduction steps from issue

### Alternative Approaches Considered

**Option B: Add socket parameter to all call sites**
- **Pros:** Explicit, no global state, maximum flexibility
- **Cons:** Requires updating all 9 call sites, makes API more complex, easy to forget
- **When to use instead:** If socket selection needs to be dynamic per-call

**Option C: Smart detection based on command args**
- **Pros:** Fully automatic, no refactoring needed
- **Cons:** Magic behavior, hard to reason about, brittle (depends on arg parsing)
- **When to use instead:** Never - too much implicit behavior

**Option D: Always require -S flag from callers**
- **Pros:** No magic, explicit socket specification
- **Cons:** Breaks all existing code, verbose, duplicates detection logic everywhere
- **When to use instead:** New codebase with socket-aware design from start

**Rationale for recommendation:** Option A (global detection with override) provides the best balance - fixes common case transparently while making special cases explicit. Aligns with evidence that 7/9 operations target main tmux.

---

### Implementation Details

**What to implement first:**
- detectMainSocket() function - detects if running in overmind and returns main socket path
- Update findTmux() to cache main socket path alongside tmux binary path
- Modify tmuxCommand() to use main socket when detected

**Things to watch out for:**
- ⚠️ Socket path detection must work on both macOS and Linux (different /tmp layouts)
- ⚠️ Must handle case where $TMUX is not set (not running in tmux at all)
- ⚠️ Must handle case where main tmux is not running (fail gracefully)
- ⚠️ GetCurrentWindowName/RenameCurrentWindow must still work when called from main tmux context

**Areas needing further investigation:**
- None identified - solution is straightforward based on findings

**Success criteria:**
- ✅ `tmux display-message -t orchestrator -p '#{window_index}'` works from overmind context
- ✅ Dashboard /api/context endpoint returns orchestrator cwd when orch serve runs in overmind
- ✅ GetCurrentWindowName() still works correctly from any tmux context
- ✅ All existing tmux tests pass

---

## References

**Files Examined:**
- pkg/tmux/tmux.go:55-61 - tmuxCommand() function that needs socket detection
- pkg/tmux/tmux.go:73 - GetCurrentWindowName() uses current socket
- pkg/tmux/tmux.go:107,123 - RenameCurrentWindow() uses current socket  
- pkg/tmux/follower.go:243,261 - GetTmuxCwd() needs main socket
- cmd/orch/serve_context.go:53 - API endpoint that fails due to socket issue

**Commands Run:**
```bash
# Check running tmux servers
ps aux | grep tmux

# Verify current tmux context
echo $TMUX

# Test command without socket (fails from overmind)
tmux display-message -t orchestrator -p '#{window_index}'

# Test command with explicit socket (works)
tmux -S /tmp/tmux-501/default display-message -t orchestrator -p '#{window_index}'

# List available sockets
ls -la /tmp/tmux-501/

# Check which sessions visible from overmind
tmux list-sessions
```

**External Documentation:**
- tmux man page - socket specification via -S flag and -L flag
- tmux $TMUX environment variable - format: socket_path,server_pid,session_id

**Related Artifacts:**
- **Issue:** orch-go-0n4ud - Dashboard context API fails when orch serve runs in overmind

---

## Test Verification

**Before fix:**
```bash
# From overmind tmux context
$ tmux display-message -t orchestrator -p '#{window_index}'
(empty - FAILED)
```

**After fix:**
```bash
# From overmind tmux context  
$ curl -k https://localhost:3348/api/context | jq .
{
  "cwd": "/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch",
  "project_dir": "/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch",
  "project": "price-watch",
  "included_projects": ["price-watch"]
}
(SUCCESS - returns orchestrator cwd)

# Verify orchestrator session cwd matches
$ tmux -S /tmp/tmux-501/default display-message -t orchestrator -p '#{pane_current_path}'
/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch
(MATCHES)
```

**Test suite:**
```bash
$ go test ./pkg/tmux/...
PASS (all tests passing)
```

---

## Investigation History

**2026-01-15 13:25:** Investigation started
- Initial question: How should orch-go detect and use correct tmux socket when running inside overmind?
- Context: Dashboard /api/context failing when orch serve runs in overmind tmux

**2026-01-15 13:40:** Identified root cause
- Confirmed two tmux servers running (main + overmind)
- Verified tmux commands without -S flag target overmind's server
- Determined 7/9 tmuxCommand() call sites need main socket

**2026-01-15 13:50:** Implemented solution
- Added detectMainSocket() function
- Modified tmuxCommand() to auto-add -S flag
- Created tmuxCommandCurrent() for exceptions
- Updated GetCurrentWindowName/RenameCurrentWindow

**2026-01-15 13:55:** Investigation completed
- Status: Complete
- Key outcome: Socket detection implemented and verified working, Dashboard API now functions correctly from overmind context
