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

There were **two separate causes** of SIGKILL (exit 137) for `orch spawn --tmux`:

1. **Stale binary** - Binary at `~/bin/orch` didn't have latest code changes
2. **launchd KeepAlive conflict** - `orch serve` daemon using `~/bin/orch` with `KeepAlive: true` caused SIGKILL when binary was replaced

**Resolution:** 
1. Added git post-commit hook to auto-rebuild on Go file changes
2. Changed `orch serve` launchd plist to use `build/orch` instead of `~/bin/orch`

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

## Second Occurrence: launchd KeepAlive Conflict

After fixing the stale binary issue, SIGKILL returned. Further investigation revealed a second root cause.

### Finding: launchd `orch serve` daemon causes SIGKILL

**Evidence:**
```bash
# Same binary, different paths - different behavior
./build/orch spawn --tmux investigation "test" # SUCCESS
/Users/dylanconlin/bin/orch spawn --tmux investigation "test" # EXIT 137

# Even copying the working binary didn't help
cp ./build/orch ~/bin/orch
~/bin/orch spawn --tmux investigation "test" # EXIT 137

# But renaming works!
cp ./build/orch ~/bin/orch-new
~/bin/orch-new spawn --tmux investigation "test" # SUCCESS

# Stopping orch serve fixed it
launchctl stop com.orch-go.serve
~/bin/orch spawn --tmux investigation "test" # SUCCESS
```

**Root Cause:** The `com.orch-go.serve` launchd daemon was configured with:
- `ProgramArguments: /Users/dylanconlin/bin/orch serve`
- `KeepAlive: true`

When `make install` replaced `~/bin/orch`, launchd detected the binary change and restarted the daemon. This restart somehow sent SIGKILL to the running spawn process.

**Resolution:** Changed the plist to use the build directory binary instead:
```xml
<string>/Users/dylanconlin/Documents/personal/orch-go/build/orch</string>
```

Now `make install` can update `~/bin/orch` without affecting the serve daemon.

---

## Prevention Measures Added

1. **Git post-commit hook** - Auto-runs `make install` when Go files change
   - Location: `.git/hooks/post-commit`
   - Prevents stale binary issues

2. **Self-describing binary** - `orch version --source` shows build info and staleness
   - Embeds source directory and git hash at build time
   - Can detect when binary is out of sync with source

3. **Separate daemon binary** - `orch serve` uses `build/orch` not `~/bin/orch`
   - Prevents launchd KeepAlive from interfering with spawn processes

---

## Related

- The new code uses `pkg/tmux` package properly for TUI readiness detection
- WaitForOpenCodeReady polls pane content for visual indicators (prompt box + agent selector)
- PostReadyDelay of 1s allows input focus to settle before typing
- Issue `orch-go-xvcu` tracked the second occurrence
