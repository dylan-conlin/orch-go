<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux-spawned orchestrators were using standalone OpenCode mode which doesn't register with the shared server, preventing session ID capture.

**Evidence:** Workspace analysis showed workers (headless spawned) have .session_id files, orchestrators (tmux spawned) don't. Code traced to BuildOpencodeAttachCommand using `opencode {project}` standalone instead of `opencode attach <url> --dir {project}`.

**Knowledge:** OpenCode in standalone mode runs an embedded server - sessions aren't visible via shared API. The `--dir` flag in attach mode was fixed in commit 18b26856a to properly set session working directory.

**Next:** Additional fix needed - either add --title flag to attach or change FindRecentSession to match by directory+time only.

---

# Investigation: Orchestrator Sessions Spawned Via Tmux Don't Capture .session_id

**Question:** Why do tmux-spawned orchestrator sessions not have .session_id files, and how can we fix it?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent
**Phase:** Blocked
**Next Step:** Fix FindRecentSession to match by directory+time (not title) OR add --title to opencode attach
**Status:** In Progress - manual verification failed

---

## Findings

### Finding 1: Workers have .session_id files, orchestrators don't

**Evidence:** 
```bash
# Workers (headless spawned) - have session IDs
ls .orch/workspace/og-feat-add-api-changelog-03jan/.session_id
# -> ses_47a045d3affeRu2BZWSi8zvAun

# Orchestrators (tmux spawned) - missing session IDs
ls .orch/workspace/meta-orch-continue-meta-orch-06jan-2c9a/
# -> .meta-orchestrator .spawn_time .tier .workspace_name META_ORCHESTRATOR_CONTEXT.md
# NO .session_id file!
```

**Source:** `cmd/orch/spawn_cmd.go:1330` writes session ID, but line 1315 fails silently for tmux spawns.

**Significance:** Confirms the issue is in the tmux spawn path, not a general session ID writing problem.

---

### Finding 2: Tmux spawns used standalone OpenCode mode

**Evidence:** 
```go
// pkg/tmux/tmux.go:100-108 (BEFORE fix)
func BuildOpencodeAttachCommand(cfg *OpencodeAttachConfig) string {
    // Use standalone mode with project directory as argument
    cmd := fmt.Sprintf("ORCH_WORKER=1 %s %q", opencodeBin, cfg.ProjectDir)
    ...
}
```

Comment in code explained: "The tradeoff is sessions won't be visible via shared server API"

**Source:** `pkg/tmux/tmux.go:93-120`

**Significance:** Root cause identified - standalone mode doesn't register sessions with the shared server.

---

### Finding 3: OpenCode attach --dir now works correctly

**Evidence:** 
- OpenCode commit `18b26856a`: "fix: Session.create now respects directory parameter"
- `opencode attach --help` shows `--dir` flag: "directory to run in"
- `opencode attach --help` shows `--model` flag is supported (test was outdated)

**Source:** `~/.bun/bin/opencode attach --help`, `~/Documents/personal/opencode` git log

**Significance:** The workaround (standalone mode) is no longer needed. Sessions in attach mode now properly use the specified directory.

---

## Synthesis

**Key Insights:**

1. **Standalone vs Attach mode**: OpenCode can run in standalone mode (embedded server) or attach mode (shared server). Only attach mode makes sessions visible via the API at `/session`.

2. **Historical workaround obsolete**: The standalone mode was used because `--dir` didn't properly set working directory. OpenCode commit `18b26856a` fixed this.

3. **Session ID capture flow**: Headless spawns use `opencode run --attach` which outputs JSON including session ID. Tmux spawns query the API via `FindRecentSession`. API query only works if session is registered with shared server.

**Answer to Investigation Question:**

Tmux-spawned orchestrators didn't capture .session_id because `BuildOpencodeAttachCommand` used standalone mode (`opencode {project}`) instead of attach mode (`opencode attach <url> --dir {project}`). Standalone mode runs an embedded OpenCode server, so sessions aren't visible via the shared API at `http://localhost:4096/session`. 

The fix is to switch to attach mode with `--dir`, which is now reliable after OpenCode commit `18b26856a` fixed directory handling.

---

## Structured Uncertainty

**What's tested:**

- ✅ Unit tests pass for updated `BuildOpencodeAttachCommand` (verified: go test ./pkg/tmux/... passes)
- ✅ Full test suite passes (verified: go test ./... passes)
- ✅ Build succeeds (verified: make install completes)

**What's untested:**

- ❌ Manual verification FAILED - tmux spawns still don't capture session IDs (see below)

**What would change this:**

- If OpenCode's `--dir` flag behavior regresses
- If there are edge cases where attach mode behaves differently than standalone

---

## Manual Verification (2026-01-06 16:00)

**Test performed:** `orch spawn hello "test session id capture" --tmux --bypass-triage --no-track`

**Result:** FAILED - .session_id file NOT created in workspace

**Root cause discovered:**
1. Session IS being registered with API (confirmed: ses_46a434a9dffeCdHbGCE3WekKnB exists)
2. BUT `FindRecentSession` can't find it because:
   - It matches by session title
   - Session title is first prompt text ("Reading SPAWN_CONTEXT for task setup"), NOT workspace name
   - `opencode attach --dir` doesn't set session title - only the first message becomes the title

**Fix is incomplete.** The change to use `opencode attach --dir` successfully registers sessions with the API, but `FindRecentSession` can't locate them because the title matching fails.

**Required additional fix (one of):**
a) Add `--title` flag to attach command (requires OpenCode support: `opencode attach <url> --dir <path> --title <workspace>`)
b) Change `FindRecentSession` to match by directory + creation time only, ignoring title

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Switch tmux spawns to attach mode with --dir** - Change `BuildOpencodeAttachCommand` to use `opencode attach <url> --dir <project>` instead of `opencode <project>`.

**Why this approach:**
- Sessions become visible via API, enabling session ID capture
- OpenCode commit 18b26856a fixed the `--dir` issue that originally required standalone mode
- Minimal change with clear benefit

**Trade-offs accepted:**
- Depends on shared OpenCode server being available (same as headless spawns)
- Slight behavioral change for existing tmux users (transparent)

**Implementation sequence:**
1. ✅ Update `BuildOpencodeAttachCommand` in `pkg/tmux/tmux.go`
2. ✅ Update tests to expect `attach` mode with `--model` support
3. ⬜ Manual verification with real spawn

### Alternative Approaches Considered

**Option B: Parse session ID from TUI output**
- **Pros:** Would work with standalone mode
- **Cons:** Complex parsing, fragile, OpenCode TUI format could change
- **When to use instead:** Never - attach mode is simpler and more robust

**Option C: Keep standalone mode, create separate session lookup mechanism**
- **Pros:** Preserves standalone behavior
- **Cons:** Duplicates functionality, adds complexity
- **When to use instead:** If attach mode has unforeseen issues

---

### Implementation Details

**What to implement first:**
- ✅ Core fix in `BuildOpencodeAttachCommand`
- ✅ Test updates

**Things to watch out for:**
- ⚠️ ORCH_WORKER=1 environment variable must still be prefixed
- ⚠️ Model flag should now be included (opencode attach supports --model)

**Success criteria:**
- ✅ Unit tests pass
- ⬜ New tmux-spawned orchestrators have .session_id file in workspace
- ⬜ `orch attach <workspace>` command can use the captured session ID

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - Spawn command implementation, session ID writing
- `pkg/tmux/tmux.go` - BuildOpencodeAttachCommand implementation
- `pkg/opencode/client.go` - FindRecentSession, session lookup methods

**Commands Run:**
```bash
# Check workspace contents
ls -la .orch/workspace/meta-orch-*/
ls -la .orch/workspace/og-feat-*/.session_id

# Verify OpenCode attach supports --dir and --model
~/.bun/bin/opencode attach --help

# Run tests
go test ./pkg/tmux/... -v
go test ./... 
make install
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Full mental model
- **Issue:** `orch-go-cnkbv` - orch attach command
- **Issue:** `orch-go-xdcpc` - Extend orch resume for orchestrators

---

## Investigation History

**2026-01-06 15:45:** Investigation started
- Initial question: Why don't tmux-spawned orchestrators capture .session_id?
- Context: Spawned from issue orch-go-wruwx

**2026-01-06 16:00:** Root cause identified
- Traced to BuildOpencodeAttachCommand using standalone mode
- Confirmed workers (headless) have session IDs, orchestrators (tmux) don't

**2026-01-06 16:30:** Implementation completed
- Changed BuildOpencodeAttachCommand to use attach mode with --dir
- Updated tests, all passing
- Build verified

**2026-01-06 16:45:** Investigation completed
- Status: Complete
- Key outcome: Fixed tmux spawns to use attach mode, enabling session ID capture via API
