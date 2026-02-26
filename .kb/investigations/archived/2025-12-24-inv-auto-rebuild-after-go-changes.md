## Summary (D.E.K.N.)

**Delta:** Added auto-rebuild functionality to `orch complete` flow - detects Go file changes in recent commits, runs `make install`, and restarts `orch serve` if running.

**Evidence:** Implementation tested via unit tests for file pattern matching. All tests pass including new `TestHasGoChangesDetection` test.

**Knowledge:** The solution uses `git diff --name-only HEAD~5..HEAD` to detect changes. Pattern matching targets `cmd/orch/*.go` and `pkg/*.go` files. Process restart uses `nohup` for background spawning.

**Next:** Close - feature implemented and ready for use.

**Confidence:** High (85%) - Pattern matching tested, but restart logic not tested in isolation (would restart actual serve).

---

# Investigation: Auto Rebuild After Go Changes

**Question:** How to auto-rebuild orch binary and restart orch serve when agents commit Go changes?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: runComplete flow in main.go

**Evidence:** The `runComplete` function (main.go:2404-2549) handles agent completion. Key steps:
1. Verify issue exists
2. Verify phase status (unless --force)
3. Check liveness
4. Close beads issue
5. Clean up tmux window
6. Log completion event

**Source:** cmd/orch/main.go:2404-2549

**Significance:** The best integration point is after tmux cleanup and before logging. This ensures verification passed before triggering rebuild.

---

### Finding 2: Go file detection via git

**Evidence:** `git diff --name-only HEAD~5..HEAD` reliably shows changed files in recent commits. Pattern matching for `cmd/orch/*.go` and `pkg/*.go` identifies rebuild-worthy changes.

**Source:** Testing in shell: `git diff --name-only HEAD~5..HEAD | grep -E '(cmd/orch/.*\.go|pkg/.*\.go)'`

**Significance:** Using last 5 commits catches typical agent work while avoiding excessive history scanning.

---

### Finding 3: orch serve restart mechanism

**Evidence:** Process identified via `pgrep -f "orch.*serve"`. Uses SIGTERM for graceful shutdown, then spawns new process with `nohup` for background execution.

**Source:** Testing: `pgrep -lf "orch.*serve"` shows PID and command

**Significance:** Background spawning via nohup ensures the serve process outlives the completion command.

---

## Synthesis

**Key Insights:**

1. **Integration point** - Adding auto-rebuild after tmux cleanup but before event logging ensures we only rebuild on successful completions.

2. **Pattern matching** - Simple string prefix/suffix matching is sufficient for Go file detection without complex glob parsing.

3. **Process restart** - Using nohup with /dev/null redirects avoids blocking the completion command.

**Answer to Investigation Question:**

Auto-rebuild implemented in `orch complete` flow. When verification passes and beads issue closes successfully, the system checks recent commits for Go file changes. If detected, runs `make install` and restarts `orch serve` if running.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Pattern matching logic is well-tested via unit tests. The git and process management commands are standard and reliable.

**What's certain:**

- Pattern matching correctly identifies cmd/orch and pkg Go files
- git diff command works for recent commits
- Process detection via pgrep is reliable

**What's uncertain:**

- Restart behavior in edge cases (multiple serve processes, serve started with different flags)
- Behavior when make install fails mid-way

**What would increase confidence to Very High:**

- Integration test with mock serve process
- Testing make install failure scenarios

---

## Implementation Recommendations

### Recommended Approach: Inline in runComplete

**Why this approach:**
- Simple, contained change
- No new packages needed
- Logical flow: verify -> close -> rebuild -> log

**Trade-offs accepted:**
- Functions are in main.go rather than separate package
- Restart spawns with default flags (port 3348)

**Implementation sequence:**
1. Add hasGoChangesInRecentCommits() helper
2. Add runAutoRebuild() helper
3. Add restartOrchServe() helper
4. Call from runComplete after tmux cleanup

---

## References

**Files Examined:**
- cmd/orch/main.go:2404-2549 - runComplete function
- cmd/orch/serve.go - serve command implementation

**Commands Run:**
```bash
git diff --name-only HEAD~5..HEAD
pgrep -lf "orch.*serve"
go test ./cmd/orch/... -run TestHasGoChangesDetection -v
```

---

## Investigation History

**2025-12-24 10:00:** Investigation started
- Initial question: How to auto-rebuild after Go changes in orch complete

**2025-12-24 10:15:** Found integration point in runComplete

**2025-12-24 10:30:** Implementation complete
- Three helper functions added
- Unit tests pass
- All package tests pass
