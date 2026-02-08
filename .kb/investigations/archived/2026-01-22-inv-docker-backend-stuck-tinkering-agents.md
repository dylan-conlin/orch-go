## Summary (D.E.K.N.)

**Delta:** Docker spawns hang intermittently because `cat file | claude` pipes interact poorly with Docker's `-it` flag; stdin redirection (`claude < file`) works reliably.

**Evidence:** Tested both patterns via tmux: pipe version sometimes hangs at "Tinkering...", redirection version processes immediately with full TUI functionality.

**Knowledge:** When Docker allocates a TTY (`-t`) and a subprocess pipes to another command, terminal control and EOF detection become unreliable; direct stdin redirection avoids the subprocess and works consistently.

**Next:** Fix implemented in docker.go - change `cat %q | claude` to `claude < %q`.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural pattern)

---

# Investigation: Docker Backend Stuck Tinkering Agents

**Question:** Why do Docker-spawned agents intermittently hang at "Tinkering..." and how to fix it?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** systematic-debugging spawn
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: "Tinkering..." is API response waiting, not terminal input waiting

**Evidence:** Claude CLI shows "Tinkering..." or "Marinating..." while waiting for API response, not during input phase. Running containers that completed work also showed this spinner during processing.

**Source:** Live observation of running Docker containers via `tmux capture-pane`

**Significance:** The architect's hypothesis about "waiting for terminal input" was incorrect - the issue is somewhere in the stdin handling, not Claude's input detection.

---

### Finding 2: Pipe pattern (`cat file | claude`) with `-it` is unreliable

**Evidence:** Testing showed that `docker run -it ... bash -c 'cat file | claude'` sometimes hangs indefinitely, while other times completes successfully. The intermittent nature (2 of 3 failures) suggests a race condition or timing issue.

**Source:** Manual reproduction tests and original bug report

**Significance:** With pipes, there are two processes (`cat` and `claude`) sharing terminal control. Docker's TTY allocation with `-t` may interfere with proper pipe/EOF handling when multiple processes are involved.

---

### Finding 3: Stdin redirection (`claude < file`) works reliably

**Evidence:** Testing `docker run -it ... bash -c 'claude < file'` showed:
- TUI displays correctly
- Claude receives input properly
- Tool calls execute (Search, Bash, Read observed)
- Agent completes work without hanging

**Source:** Manual tests via tmux with both process substitution and file redirection

**Significance:** Stdin redirection has simpler data flow - no subprocess needed, `claude` reads directly from the file descriptor. This avoids the terminal control issues that occur with pipes.

---

## Synthesis

**Key Insights:**

1. **TTY + pipe conflict** - Docker's `-it` flag allocates a pseudo-TTY, but the pipe pattern creates two processes that must coordinate terminal control. This coordination can fail intermittently.

2. **Redirection is simpler** - With `command < file`, there's only one process (`command`) reading from the file descriptor. No subprocess coordination needed.

3. **TUI still works** - The `-t` flag is still needed for Claude's TUI to display, and stdin redirection works correctly with it because there's no pipe coordination issue.

**Answer to Investigation Question:**

Docker spawns hang because the `cat file | claude` pattern has two processes sharing terminal control under Docker's `-t` flag, which creates intermittent failures. The fix is to use stdin redirection (`claude < file`) instead, which has only one process and works reliably.

---

## Structured Uncertainty

**What's tested:**

- ✅ Stdin redirection works with `-it` (verified: multiple tests showed TUI display, tool execution, task completion)
- ✅ Pipe pattern can hang (verified: reproduced "Tinkering..." hang, matches original bug report)
- ✅ Fix doesn't break TUI functionality (verified: observed full TUI with tool calls working)

**What's untested:**

- ⚠️ Long-running agent sessions (only tested short interactions, not full multi-hour sessions)
- ⚠️ Edge cases with very large context files (tested with 44KB file, larger might behave differently)

**What would change this:**

- If stdin redirection starts showing same intermittent failures, the root cause is different
- If removing `-t` flag proves more reliable, that would suggest a different approach

---

## References

**Files Examined:**
- `pkg/spawn/docker.go:85-109` - Docker spawn command generation
- `pkg/spawn/claude.go:50-55` - Claude spawn pattern (for comparison)

**Commands Run:**
```bash
# Test pipe pattern with -it
docker run -it --rm ... bash -c 'cat file | claude --dangerously-skip-permissions'

# Test stdin redirection with -it
docker run -it --rm ... bash -c 'claude --dangerously-skip-permissions < file'
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-22-inv-analyze-spawn-reliability-pattern-multiple.md` - Broader spawn reliability analysis
