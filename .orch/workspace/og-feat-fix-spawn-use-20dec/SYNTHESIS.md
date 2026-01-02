# Session Synthesis

**Agent:** og-feat-fix-spawn-use-20dec
**Issue:** orch-go-u82
**Duration:** 2025-12-20 ~16:00 → ~16:45
**Outcome:** success

---

## TLDR

Fixed spawn to use standalone mode with TUI instead of headless prompt arg. Ported Python orch-cli's approach (spawn.py:959-1032): start opencode without prompt arg, wait for TUI ready, send prompt via tmux send-keys.

---

## Delta (What Changed)

### Files Created

- `.kb/investigations/2025-12-20-inv-fix-spawn-use-standalone-mode.md` - Investigation documenting the fix

### Files Modified

- `pkg/tmux/tmux.go` - Added `BuildStandaloneCommand()` returning command string (not exec.Cmd)
- `pkg/tmux/tmux_test.go` - Added test for `BuildStandaloneCommand()`
- `cmd/orch/main.go` - Updated `runSpawnInTmux()` to use standalone mode workflow

### Commits

- `5ff9c11` - feat: use standalone mode for TUI spawn (port Python approach)

---

## Evidence (What Was Observed)

- Python orch-cli uses standalone mode because "attach mode has issues with project/session routing" (spawn.py:961)
- Python explicitly rejected CLI prompt arg because "--prompt flag has inconsistent submit behavior"
- WaitForOpenCodeReady, IsOpenCodeReady, SendPromptAfterReady already existed in orch-go (just needed to wire them in)
- Existing functions matched Python's timing (15s timeout, 200ms poll, 1s post-ready delay)

### Tests Run

```bash
go test ./...
# PASS: all tests passing (16 packages)

go test ./pkg/tmux/... -v -run TestBuildStandaloneCommand
# PASS: TestBuildStandaloneCommand (0.00s)
```

---

## Knowledge (What Was Learned)

### New Artifacts

- `.kb/investigations/2025-12-20-inv-fix-spawn-use-standalone-mode.md` - Documents the fix and rationale

### Decisions Made

- Use string return from BuildStandaloneCommand (not exec.Cmd) because command is sent via tmux send-keys
- Add ORCH_WORKER env var to match Python behavior
- Warn but don't fail if TUI-ready wait times out (agent may still work)

### Constraints Discovered

- TUI readiness detection requires visual indicators (┃ + build/agent + alt+x/commands)
- Post-ready delay (1s) is necessary for input focus to settle

### Externalized via `kn`

- None (constraints already documented in investigation)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close

- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-u82`

---

## Session Metadata

**Skill:** feature-impl
**Model:** anthropic/claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-feat-fix-spawn-use-20dec/`
**Investigation:** `.kb/investigations/2025-12-20-inv-fix-spawn-use-standalone-mode.md`
**Beads:** `bd show orch-go-u82`
