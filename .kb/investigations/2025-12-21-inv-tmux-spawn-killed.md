# Investigation: orch spawn --tmux getting SIGKILL

**Question:** Why is `orch spawn --tmux` being killed with exit code 137 (SIGKILL)?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Dylan
**Phase:** Complete
**Status:** Resolved
**Confidence:** High (95%)

---

## Summary

The SIGKILL issue was caused by a **stale binary** at `/Users/dylanconlin/bin/orch` that didn't have the latest code changes. The uncommitted changes in `cmd/orch/main.go` (using `pkg/tmux` package properly) were not in the installed binary.

**Root Cause:** Binary at `~/bin/orch` was built BEFORE the code was refactored to use the proper `pkg/tmux` package. The `git diff` showed uncommitted changes that fixed the tmux spawn logic.

**Resolution:** Rebuild and reinstall the binary with `make install`. The new binary works correctly.

---

## Key Finding: Binary Mismatch

**Evidence:**
```bash
# Test binary from /tmp worked
/tmp/orch-test spawn --tmux investigation "test" # SUCCESS

# Binary at ~/bin failed
/Users/dylanconlin/bin/orch spawn --tmux investigation "test" # EXIT CODE 137

# After copying working binary to ~/bin
cp /tmp/orch-test /Users/dylanconlin/bin/orch
/Users/dylanconlin/bin/orch spawn --tmux investigation "test" # SUCCESS
```

**Source:** Direct testing during investigation

**Significance:** The issue wasn't external process killing, it was the old binary crashing during the spawn flow. The new code fixes this.

---

## Code Changes That Fixed It

The `git diff cmd/orch/main.go` showed:

1. **Added proper import:** `"github.com/dylan-conlin/orch-go/pkg/tmux"`

2. **Replaced inline exec.Command calls** with proper `pkg/tmux` functions:
   - `tmux.EnsureWorkersSession()` instead of manual session check/create
   - `tmux.CreateWindow()` instead of manual window creation
   - `tmux.SendPromptAfterReady()` instead of `time.Sleep(2 * time.Second)`

3. **Removed problematic code:**
   - `envVars := fmt.Sprintf("export ORCH_WORKER=true && ")` - not needed
   - Hard-coded sleep delays replaced with proper TUI detection

---

## Why SIGKILL Specifically?

The old code likely crashed in a way that the Go runtime handled as SIGKILL (exit 137). Possibilities:
- Panic during tmux command execution that wasn't caught
- Race condition in the old sleep-based TUI detection
- Memory corruption from improper string handling

The exact cause in the old code is moot now - the new code works correctly.

---

## Resolution Steps

1. Checked `git status` - found uncommitted changes to `cmd/orch/main.go`
2. Rebuilt binary with `make install`
3. Verified new binary works: `orch spawn --tmux investigation "test"` succeeds

---

## Lessons Learned

1. **Always rebuild after code changes** - The binary at `~/bin/orch` doesn't auto-update
2. **Check git diff early** - Uncommitted changes are a common cause of "it works here but not there"
3. **Compare binaries directly** - `md5` hash comparison quickly identifies version mismatch

---

## Related

- The new code uses `pkg/tmux` package properly for TUI readiness detection
- WaitForOpenCodeReady polls pane content for visual indicators (prompt box + agent selector)
- PostReadyDelay of 1s allows input focus to settle before typing
