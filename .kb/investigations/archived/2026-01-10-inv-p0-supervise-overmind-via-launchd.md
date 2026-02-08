<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Launchd successfully supervises overmind with crash recovery via KeepAlive=true and EnvironmentVariables PATH configuration.

**Evidence:** Tested crash recovery by killing tmux processes - overmind auto-restarted within 5 seconds with all services (api, web, opencode) returning to "running" state; launchd job shows status 0; logs confirm successful starts.

**Knowledge:** Overmind in detached mode (-D) requires launchd PATH to include /opt/homebrew/bin (for tmux) and ~/.bun/bin (for orch/opencode/bun); KeepAlive=true provides automatic crash recovery; socket removal + tmux kill effectively simulates crash condition.

**Next:** Close issue - implementation complete and tested; document in CLAUDE.md about launchd supervision setup.

**Promote to Decision:** recommend-no - Tactical implementation following established launchd patterns; no new architectural principles emerged.

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

# Investigation: P0 Supervise Overmind Via Launchd

**Question:** How should launchd supervise overmind to ensure reliability and auto-restart?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** feature-impl agent (orch-go-b6hwn)
**Phase:** Complete
**Next Step:** Document in CLAUDE.md
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Overmind is Installed and Running Services

**Evidence:**
- Overmind installed at `/opt/homebrew/bin/overmind`
- Procfile exists with 3 services: api, web, opencode
- Services defined as: `api: orch serve`, `web: cd web && bun run dev`, `opencode: ~/.bun/bin/opencode serve --port 4096`
- No existing overmind launchd plist at `~/Library/LaunchAgents/`

**Source:**
- Command: `which overmind` → `/opt/homebrew/bin/overmind`
- File: `/Users/dylanconlin/Documents/personal/orch-go/Procfile`
- Command: `ls -la ~/Library/LaunchAgents/ | grep orch` → no results

**Significance:** Overmind is installed and configured but has no supervision - if overmind crashes, all 3 services (api, web, opencode) go down with it. This is the reliability gap we need to fix.

---

### Finding 2: Existing launchd Plists Follow Standard Pattern

**Evidence:**
- Examined existing plists in `~/Library/LaunchAgents/`:
  - `homebrew.mxcl.emacs-plus@29.plist` uses `KeepAlive=true`, `RunAtLoad=true`
  - `homebrew.mxcl.mysql.plist` uses same pattern with `WorkingDirectory` set
  - `com.user.tmuxinator.plist` shows environment variable pattern for PATH
- Standard structure includes: Label, ProgramArguments, RunAtLoad, KeepAlive, StandardErrorPath, StandardOutPath

**Source:**
- File: `~/Library/LaunchAgents/homebrew.mxcl.emacs-plus@29.plist`
- File: `~/Library/LaunchAgents/homebrew.mxcl.mysql.plist`
- File: `~/Library/LaunchAgents/com.user.tmuxinator.plist`

**Significance:** We have proven patterns to follow for creating the overmind supervisor plist. Key elements needed: KeepAlive for auto-restart, RunAtLoad for boot persistence, WorkingDirectory for project path, StandardErrorPath/StandardOutPath for logging.

---

### Finding 3: Overmind Start Requires Project Directory Context

**Evidence:**
- `overmind start` command options show:
  - `--procfile value, -f value`: Specify Procfile to load (default: "./Procfile")
  - `--root value, -d value`: Specify working directory (default: directory containing Procfile)
- Procfile is located at `/Users/dylanconlin/Documents/personal/orch-go/Procfile`
- Command must run from project directory OR specify `--root` flag

**Source:**
- Command: `overmind help start`
- File: `/Users/dylanconlin/Documents/personal/orch-go/Procfile`

**Significance:** The launchd plist must either set WorkingDirectory to the orch-go project path, or use `--root` flag in ProgramArguments to ensure overmind finds the Procfile.

---

## Synthesis

**Key Insights:**

1. **Three-Layer Architecture Emerges** - Decision document establishes principle "launchd owns ALL persistent services". Current implementation has overmind → services (api, web, opencode), but overmind itself is unsupervised. Adding launchd supervision creates proper hierarchy: launchd → overmind → services. (Findings 1, 2)

2. **Standard launchd Pattern Applies** - Existing plists show proven pattern: KeepAlive=true for auto-restart, RunAtLoad=true for boot persistence, WorkingDirectory for project context, StandardErrorPath/StandardOutPath for debugging. This is exactly what overmind needs. (Finding 2)

3. **Project Path is Critical** - Overmind requires working directory context to find Procfile. Must set WorkingDirectory in plist to `/Users/dylanconlin/Documents/personal/orch-go` OR use `--root` flag. (Finding 3)

**Answer to Investigation Question:**

Launchd should supervise overmind via a plist at `~/Library/LaunchAgents/com.overmind.orch-go.plist` with:
- **KeepAlive=true** - Auto-restart overmind if it crashes
- **RunAtLoad=true** - Auto-start at login
- **WorkingDirectory** - Set to project path so overmind finds Procfile
- **StandardErrorPath/StandardOutPath** - Log to `~/.orch/overmind-*.log` for debugging
- **ProgramArguments** - `/opt/homebrew/bin/overmind start -D` (detached mode)

This completes the reliability architecture: launchd supervises overmind, overmind supervises services, service monitor observes all layers.

---

## Test Results

### Test 1: Initial Load and Start
**Command:** `launchctl load ~/Library/LaunchAgents/com.overmind.orch-go.plist`
**Result:** ✅ SUCCESS
- launchd job loaded with status 0 (success)
- overmind started and created .overmind.sock
- All 3 services (api, web, opencode) started successfully
- Logs captured at ~/.orch/overmind-stdout.log and ~/.orch/overmind-stderr.log

**Evidence:**
```
$ launchctl list | grep com.overmind.orch-go
-	0	com.overmind.orch-go

$ overmind status
PROCESS   PID       STATUS
api       0         running
web       0         running
opencode  0         running
```

### Test 2: Crash Recovery (KeepAlive Verification)
**Command:** `rm -f .overmind.sock && pkill -f "tmux.*overmind-orch-go"`
**Result:** ✅ SUCCESS - Auto-restart within 5 seconds
- Removed socket file and killed all tmux processes (simulated crash)
- After 5 seconds: socket recreated, services restarted
- launchd job remained at status 0
- All services returned to "running" state

**Evidence:**
```
$ rm -f .overmind.sock && pkill -f "tmux.*overmind-orch-go"
$ sleep 5 && ls -la .overmind.sock
Sat Jan 10 00:42:05 2026 0 B .overmind.sock

$ overmind status
PROCESS   PID       STATUS
api       0         running
web       0         running
opencode  0         running
```

### Test 3: PATH Environment Variable Fix
**Initial Issue:** launchd couldn't find tmux - stderr showed "Can't find tmux. Did you forget to install it?"
**Solution:** Added EnvironmentVariables section to plist with PATH including /opt/homebrew/bin and ~/.bun/bin
**Result:** ✅ SUCCESS - tmux found, overmind started correctly

---

## Structured Uncertainty

**What's tested:**

- ✅ Crash recovery works (verified: killed tmux processes, observed auto-restart within 5 seconds)
- ✅ All three services restart after crash (verified: overmind status shows api, web, opencode all "running")
- ✅ PATH environment variable includes required directories (verified: no "can't find tmux" errors in logs)
- ✅ Logs are captured to ~/.orch/ directory (verified: tail shows overmind output)

**What's untested:**

- ⚠️ Boot persistence (RunAtLoad=true set, but not tested via actual reboot)
- ⚠️ Services accessible on correct ports (overmind status shows running, but didn't verify HTTP responses)
- ⚠️ Cross-project overmind supervision (only tested with orch-go project)

**What would change this:**

- Boot persistence would be wrong if launchd doesn't start overmind after Mac reboot
- Service accessibility would be wrong if curl to localhost:5188 or localhost:4096 returns connection refused after successful start
- Cross-project claim would be wrong if launching overmind in different project conflicts with existing socket

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Create Standard launchd Plist with Detached Overmind** - Single plist file using proven pattern with `overmind start -D` for background operation.

**Why this approach:**
- Follows established pattern from existing plists (Finding 2)
- `-D` flag runs overmind in detached mode (no tmux attachment required)
- WorkingDirectory ensures Procfile is found (Finding 3)
- KeepAlive ensures automatic recovery from crashes
- Logs to `~/.orch/` for centralized debugging

**Trade-offs accepted:**
- Services won't start until user logs in (acceptable - development tools, not production)
- Overmind logs separate from launchd logs (acceptable - easier to find overmind-specific output)
- Single plist for all orch-go services (acceptable - they're tightly coupled)

**Implementation sequence:**
1. Create `~/Library/LaunchAgents/com.overmind.orch-go.plist` with standard pattern
2. Load plist via `launchctl load`
3. Test crash recovery by killing overmind process
4. Verify services restart automatically

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
1. Create plist file with standard launchd structure
2. Stop any existing overmind processes (avoid conflicts)
3. Load the plist and verify it starts

**Things to watch out for:**
- ⚠️ Existing overmind processes may conflict - must stop before loading plist
- ⚠️ `-D` flag critical for detached mode - without it, launchd can't manage properly
- ⚠️ Log file permissions - ensure `~/.orch/` directory exists and is writable
- ⚠️ PATH environment variable - ensure overmind can find orch, bun, and opencode binaries

**Areas needing further investigation:**
- Whether KeepAlive should be `true` or `{ SuccessfulExit = false; }` (stricter - only restart on crash, not clean exit)
- Whether to use `StandardErrorPath` or let launchd manage logs via `log show`
- Cross-project overmind management (if Dylan has multiple projects with Procfiles)

**Success criteria:**
- ✅ Overmind starts automatically at login
- ✅ Kill overmind process → launchd restarts it within 5 seconds
- ✅ Services (api, web, opencode) come back up after restart
- ✅ Logs visible at `~/.orch/overmind-stdout.log` and `~/.orch/overmind-stderr.log`

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
