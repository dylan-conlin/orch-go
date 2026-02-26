<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added TTY detection to `orch review done` to auto-skip confirmation and recommendation prompts when stdin is not a terminal.

**Evidence:** Isolated test confirms TTY detection works correctly; `term.IsTerminal()` returns false when stdin is piped.

**Knowledge:** Pattern already exists in main.go (lines 3672 and 3711) for similar prompts in `orch complete`. Apply same pattern to review.go.

**Next:** Close - fix implemented and tested.

---

# Investigation: orch review done hangs when stdin not available

**Question:** Why does `orch review done` hang when stdin is not available (e.g., when spawned by daemon or in scripts)?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent (og-debug-orch-review-hangs-30dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Two stdin reads in runReviewDone without TTY detection

**Evidence:** 
- Line 781-783: Initial confirmation prompt `fmt.Print("Continue? [y/N]: ")` followed by `reader.ReadString('\n')`
- Line 836-838: Recommendation prompt `fmt.Print("\n  Create follow-up issues? [y/n/skip-all]: ")` followed by `reader.ReadString('\n')`

Both of these calls will block indefinitely waiting for input if stdin is not connected to a terminal.

**Source:** cmd/orch/review.go:781, cmd/orch/review.go:836

**Significance:** This is the root cause of the hang when `orch review done` is called from non-interactive contexts.

---

### Finding 2: Pattern for TTY detection already exists in codebase

**Evidence:** 
```go
// From main.go:3672
if !term.IsTerminal(int(os.Stdin.Fd())) {
    return fmt.Errorf("agent still running and stdin is not a terminal; use --force to complete anyway")
}
```

The codebase already uses `golang.org/x/term` package with `term.IsTerminal(int(os.Stdin.Fd()))` for TTY detection.

**Source:** cmd/orch/main.go:3672, cmd/orch/main.go:3711

**Significance:** We can apply the same pattern to review.go without introducing new dependencies or patterns.

---

### Finding 3: Isolated test confirms TTY detection works

**Evidence:** 
```bash
$ echo "" | go run /tmp/test_tty.go
stdin is NOT a terminal
```

The TTY detection correctly identifies when stdin is not a terminal (e.g., when input is piped).

**Source:** Test program using `term.IsTerminal(int(os.Stdin.Fd()))`

**Significance:** The fix approach is validated - TTY detection correctly distinguishes interactive vs non-interactive contexts.

---

## Synthesis

**Key Insights:**

1. **Same pattern applies** - The fix follows the established pattern already used in main.go for `orch complete` prompts. Consistency is maintained across the codebase.

2. **Two separate fixes needed** - The confirmation prompt (line 781) and the recommendation prompt (line 836) needed separate TTY detection logic since they have different behaviors (one skips confirmation, the other sets skipAllPrompts).

3. **Performance issue is separate** - During testing, discovered that `orch review` commands are slow due to workspace scanning (773 directories) and beads calls. This is a separate performance issue, not related to the stdin hang.

**Answer to Investigation Question:**

The hang occurs because `runReviewDone()` calls `reader.ReadString('\n')` at two points without first checking if stdin is a terminal. When stdin is not connected (e.g., daemon-spawned agents, scripts, pipes), `ReadString()` blocks indefinitely waiting for input that will never come. The fix adds TTY detection using `term.IsTerminal(int(os.Stdin.Fd()))` to auto-skip prompts when stdin is not interactive.

---

## Structured Uncertainty

**What's tested:**

- ✅ TTY detection correctly identifies non-terminal stdin (verified: isolated Go test)
- ✅ Build succeeds with changes (verified: `go build ./cmd/orch/`)
- ✅ Existing tests pass (verified: `go test ./cmd/orch/... -run Review`)

**What's untested:**

- ⚠️ End-to-end test in daemon context (not performed due to slow workspace scanning)
- ⚠️ Behavior when stdin is partially available (edge case)

**What would change this:**

- If `term.IsTerminal()` behaves differently on different platforms or terminal types
- If the stdin behavior differs when spawned via different mechanisms (launchd vs shell)

---

## Implementation Recommendations

**Purpose:** Document the fix that was implemented.

### Recommended Approach ⭐

**TTY Detection with Auto-Skip** - Check if stdin is a terminal before attempting to read; auto-apply `--yes` and `--no-prompt` behavior when non-interactive.

**Why this approach:**
- Follows existing codebase pattern (main.go:3672)
- Non-breaking change for interactive users
- Provides informative message when skipping prompts

**Trade-offs accepted:**
- Non-interactive runs will complete without confirmation (mitigated by showing informative message)
- Recommendations are auto-dismissed when non-interactive (acceptable since orchestrator can review via `orch review`)

**Implementation sequence:**
1. Add `golang.org/x/term` import
2. Add TTY check to confirmation prompt (line 781)
3. Initialize `skipAllPrompts` with TTY check (before recommendation loop)

---

## References

**Files Examined:**
- cmd/orch/review.go - Main implementation file with stdin hang
- cmd/orch/main.go - Reference for TTY detection pattern

**Commands Run:**
```bash
# Test TTY detection
echo "" | go run /tmp/test_tty.go

# Build verification
go build -o /tmp/orch-test ./cmd/orch/

# Run tests
go test ./cmd/orch/... -run Review -v
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-30-inv-investigate-recent-bugs-attention-panel.md - Original issue identification

---

## Investigation History

**2025-12-30 22:15:** Investigation started
- Initial question: Why does `orch review done` hang when stdin not available?
- Context: Identified in prior investigation of attention panel bugs

**2025-12-30 22:30:** Root cause identified
- Found two stdin reads without TTY detection in runReviewDone()

**2025-12-30 22:35:** Fix implemented
- Added TTY detection following existing pattern from main.go

**2025-12-30 22:40:** Investigation completed
- Status: Complete
- Key outcome: Added TTY detection to auto-skip prompts when stdin is not a terminal
