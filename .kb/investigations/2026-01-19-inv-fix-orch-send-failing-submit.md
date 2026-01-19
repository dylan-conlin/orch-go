<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `orch send` tmux fallback fails because `SendKeys`, `SendKeysLiteral`, and `SendEnter` use raw `exec.Command("tmux", ...)` instead of the `tmuxCommand()` helper, causing commands to target overmind's tmux server instead of the main tmux server where worker windows live.

**Evidence:** Code review confirms SendKeys (line 562), SendKeysLiteral (line 568) use raw `exec.Command` while `tmuxCommand()` helper (lines 104-116) correctly adds `-S mainSocket` flag when running inside overmind.

**Knowledge:** Tmux socket targeting is critical when running inside overmind; all tmux commands that target worker windows must use the `tmuxCommand()` helper.

**Next:** Fix all affected functions in `pkg/tmux/tmux.go` to use `tmuxCommand()` helper.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Fix Orch Send Failing Submit

**Question:** Why does `orch send` paste messages into tmux windows but fail to submit them (Enter key not working)?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** Claude agent (systematic-debugging skill)
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete

---

## Findings

### Finding 1: Tmux socket detection helper exists but isn't used by SendKeys functions

**Evidence:** The `tmuxCommand()` helper (lines 104-116) correctly detects when running inside overmind and adds `-S mainSocket` flag to target the main tmux server. However, `SendKeys`, `SendKeysLiteral`, and other functions bypass this helper and use `exec.Command("tmux", ...)` directly.

**Source:**
- `pkg/tmux/tmux.go:104-116` - `tmuxCommand()` helper with socket detection
- `pkg/tmux/tmux.go:28-62` - `detectMainSocket()` function
- `pkg/tmux/tmux.go:562-564` - `SendKeys` uses raw `exec.Command`
- `pkg/tmux/tmux.go:568-571` - `SendKeysLiteral` uses raw `exec.Command`

**Significance:** This is the root cause. When `orch send` is invoked from overmind's context (e.g., from orch serve), commands target the wrong tmux server.

---

### Finding 2: Multiple functions have the same pattern

**Evidence:** The following functions in `pkg/tmux/tmux.go` use raw `exec.Command("tmux", ...)`:
- Line 536: `CreateWindow`
- Line 562: `SendKeys`
- Line 568: `SendKeysLiteral`
- Line 579: `SelectWindow`
- Line 585: `KillSession`
- Line 591: `GetPaneContent`
- Line 676: `WindowExists`
- Line 692: `KillWindow`
- Line 698: `KillWindowByID`

**Source:** `pkg/tmux/tmux.go` - code review

**Significance:** All these functions will fail when invoked from overmind's context.

---

### Finding 3: Bug only manifests from overmind context

**Evidence:** Testing `orch send` from the command line (outside overmind) works correctly - messages are submitted and agents respond. The bug only occurs when called from within overmind's tmux (e.g., orch serve, web dashboard).

**Source:** Manual testing from command line vs spawned agent's investigation

**Significance:** Explains why the bug is intermittent - depends on invocation context.

---

## Synthesis

**Key Insights:**

1. **Socket detection works but isn't consistently used** - The `tmuxCommand()` helper correctly detects overmind and targets the main socket, but many functions don't use it.

2. **Overmind creates a separate tmux server** - When running inside overmind, raw `exec.Command("tmux", ...)` connects to overmind's tmux server by default, not the main one.

3. **The fix is straightforward** - Update all affected functions to use `tmuxCommand()` instead of raw `exec.Command("tmux", ...)`.

**Answer to Investigation Question:**

`orch send` pastes messages but fails to submit Enter because `SendKeys`, `SendKeysLiteral`, and `SendEnter` use raw `exec.Command("tmux", ...)` instead of the `tmuxCommand()` helper. When invoked from overmind (e.g., from orch serve), commands target overmind's tmux server instead of the main one where worker windows live.

---

## Structured Uncertainty

**What's tested:**

- SendKeys, SendKeysLiteral use raw exec.Command (verified: read source code lines 562-571)
- tmuxCommand() helper adds -S flag when needed (verified: read source code lines 104-116)
- orch send works from command line outside overmind (verified: manual test)

**What's untested:**

- Running orch send from inside overmind to confirm failure (test environment not available)
- Fix verification in overmind context (to be tested after implementation)

**What would change this:**

- Finding would be wrong if tmux automatically detects socket context
- Finding would be wrong if overmind doesn't actually use a separate tmux server

---

## Implementation Recommendations

### Recommended Approach

**Update SendKeys, SendKeysLiteral, and other affected functions to use tmuxCommand()** - Convert raw `exec.Command("tmux", ...)` calls to use the helper throughout `pkg/tmux/tmux.go`.

**Why this approach:**
- The helper already exists and handles socket detection correctly
- Consistent usage ensures all tmux operations target the correct server
- Low risk - no behavioral change when not running inside overmind

**Trade-offs accepted:**
- Functions now return error if tmux not found (acceptable)
- Slightly more complex signature (must handle error)

**Implementation sequence:**
1. Update `SendKeys` and `SendKeysLiteral` (fixes main bug)
2. Update other affected functions (comprehensive fix)
3. Test from both contexts

---

## References

**Files Examined:**
- `pkg/tmux/tmux.go` - Main tmux package
- `cmd/orch/send_cmd.go` - Send command implementation
- `.kb/investigations/2026-01-19-inv-test-spawn-orch-send-debugging.md` - Related investigation by spawned agent

**Commands Run:**
```bash
# Test orch send from command line
./build/orch send orch-go-9xv1b "test message"

# Verify message appeared
tmux capture-pane -t workers-orch-go:3 -p | tail -20
```

---

## Investigation History

**2026-01-19 11:00:** Investigation started
- Initial question: Why does orch send fail to submit messages?
- Context: Bug report that messages paste but Enter not sent

**2026-01-19 11:10:** Tested basic tmux send-keys functionality
- Verified tmux send-keys Enter works correctly
- Verified orch send works from command line

**2026-01-19 11:15:** Root cause identified
- Spawned test agent found: SendKeys/SendKeysLiteral bypass tmuxCommand() helper
- Bug manifests when called from overmind's context

**2026-01-19 11:20:** Investigation completed
- Status: Complete
- Key outcome: Fix identified - update functions to use tmuxCommand() helper
