<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch send` tmux fallback fails because `SendKeys`, `SendKeysLiteral`, and `SendEnter` use raw `exec.Command("tmux", ...)` instead of the `tmuxCommand()` helper, causing commands to target overmind's tmux server instead of the main tmux server where worker windows live.

**Evidence:** Code review of `pkg/tmux/tmux.go` shows SendKeys (line 562), SendKeysLiteral (line 568) bypass the tmuxCommand() helper (lines 104-116) which adds `-S mainSocket` flag when running inside overmind.

**Knowledge:** Tmux socket targeting is critical when running inside overmind; all tmux commands must use the `tmuxCommand()` helper to ensure they target the correct tmux server.

**Next:** Update `SendKeys`, `SendKeysLiteral`, and all other functions in `pkg/tmux/tmux.go` that use raw `exec.Command("tmux", ...)` to use the `tmuxCommand()` helper instead.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Test Spawn Orch Send Debugging

**Question:** Why does `orch send` paste messages into tmux windows but fail to submit them (Enter key not working)?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** Dylan/Claude agent
**Phase:** Complete
**Next Step:** None - fix identified, implementation needed
**Status:** Complete

---

## Findings

### Finding 1: Tmux socket detection is implemented but not used consistently

**Evidence:** The `tmuxCommand()` helper function (lines 104-116 of `pkg/tmux/tmux.go`) correctly detects and adds the `-S mainSocket` flag when running inside overmind's tmux. However, several functions bypass this helper and use `exec.Command("tmux", ...)` directly.

**Source:**
- `pkg/tmux/tmux.go:104-116` - `tmuxCommand()` helper with socket detection
- `pkg/tmux/tmux.go:28-62` - `detectMainSocket()` function
- `pkg/tmux/tmux.go:562-564` - `SendKeys` uses raw `exec.Command`
- `pkg/tmux/tmux.go:568-571` - `SendKeysLiteral` uses raw `exec.Command`

**Significance:** This explains why `orch send` fails when invoked from within overmind. The send-keys command targets the wrong tmux server.

---

### Finding 2: Multiple functions have the same bug

**Evidence:** The following functions in `pkg/tmux/tmux.go` use raw `exec.Command("tmux", ...)` instead of `tmuxCommand()`:
- Line 536: `CreateWindow`
- Line 562: `SendKeys`
- Line 568: `SendKeysLiteral`
- Line 579: `SelectWindow`
- Line 585: `KillSession`
- Line 591: `GetPaneContent`
- Line 676: `WindowExists`
- Line 692: `KillWindow`
- Line 698: `KillWindowByID`
- Line 907: `WindowExistsByID`
- Line 926-932: `CaptureLines`

**Source:** `pkg/tmux/tmux.go` - manual code review

**Significance:** This is a pervasive pattern. All these functions will fail when invoked from overmind's context.

---

### Finding 3: `orch send` uses the tmux fallback path when OpenCode session resolution fails

**Evidence:** In `cmd/orch/send_cmd.go:51-75`, the `runSend` function:
1. First tries to resolve session ID via OpenCode API
2. If that fails, calls `findTmuxWindowByIdentifier` and `sendViaTmux`
3. `sendViaTmux` calls `tmux.SendKeysLiteral` and `tmux.SendEnter`

**Source:** `cmd/orch/send_cmd.go:51-75, 117-148`

**Significance:** The bug only manifests in the tmux fallback path, but this is a common path for tmux-spawned agents where OpenCode session ID wasn't captured.

---

## Synthesis

**Key Insights:**

1. **Socket detection works but isn't consistently used** - The infrastructure for detecting and targeting the correct tmux socket exists in `tmuxCommand()`, but many functions don't use it.

2. **Overmind creates a separate tmux server** - When running inside overmind, there are two tmux servers: overmind's (which runs services) and the main one (where worker windows are). Commands must target the correct one.

3. **The fix is straightforward** - Update all functions to use `tmuxCommand()` instead of raw `exec.Command("tmux", ...)`. The helper already handles the socket detection logic.

**Answer to Investigation Question:**

`orch send` pastes messages but fails to submit them because `SendKeys`, `SendKeysLiteral`, and `SendEnter` functions use raw `exec.Command("tmux", ...)` instead of the `tmuxCommand()` helper. When invoked from within overmind (e.g., from orch serve or the dashboard), these commands target overmind's tmux server instead of the main tmux server where worker windows exist. The message appears in the wrong place (or not at all), and the Enter key is never sent to the actual worker window.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code review confirms `SendKeys`, `SendKeysLiteral`, `SendEnter` use raw `exec.Command` (verified: read source code)
- ✅ `tmuxCommand()` helper exists and adds `-S mainSocket` flag when needed (verified: read source code lines 104-116)
- ✅ Socket detection via `detectMainSocket()` checks for overmind in `$TMUX` env (verified: read source code lines 28-62)

**What's untested:**

- ⚠️ Actually running `orch send` from overmind context to confirm the failure mode (test spawn verification, not implementation debugging)
- ⚠️ Verifying the fix works after implementation (deferred to fix implementation agent)

**What would change this:**

- Finding would be wrong if `SendKeys` somehow doesn't go through overmind's tmux server when run from overmind
- Finding would be wrong if there's another layer of indirection not visible in the code

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Update all tmux functions to use tmuxCommand()** - Convert raw `exec.Command("tmux", ...)` calls to use the `tmuxCommand()` helper throughout `pkg/tmux/tmux.go`.

**Why this approach:**
- The helper already exists and handles socket detection correctly
- Consistent usage ensures all tmux operations target the correct server
- Low risk - no behavioral change when not running inside overmind

**Trade-offs accepted:**
- Minor performance overhead of socket detection (already cached via `sync.Once`)
- All functions now return an error for tmux not found (acceptable - callers should handle this)

**Implementation sequence:**
1. Update `SendKeys` and `SendKeysLiteral` (primary bug)
2. Update `SendEnter` (indirectly fixed by step 1)
3. Update remaining functions (comprehensive fix)
4. Add unit tests verifying the fix

### Alternative Approaches Considered

**Option B: Only fix SendKeys/SendKeysLiteral**
- **Pros:** Minimal change, fixes immediate bug
- **Cons:** Leaves other functions broken, inconsistent codebase
- **When to use instead:** If time-critical and need immediate fix

**Rationale for recommendation:** The comprehensive fix is low-risk and prevents future bugs in other functions.

---

## References

**Files Examined:**
- `pkg/tmux/tmux.go` - Main tmux package, source of bug
- `cmd/orch/send_cmd.go` - Send command implementation
- `cmd/orch/shared.go` - Helper functions including `findTmuxWindowByIdentifier`

**Commands Run:**
```bash
# Search for relevant code
rg "findTmuxWindowByIdentifier" --type go

# Read source files
# (via Read tool)
```

**Related Artifacts:**
- **Investigation:** `.orch/workspace/og-debug-fix-orch-send-19jan-ba67/SPAWN_CONTEXT.md` - Original bug report context

---

## Investigation History

**2026-01-19:** Investigation started
- Initial question: Why does orch send fail to submit messages?
- Context: Test spawn for orch send debugging

**2026-01-19:** Root cause identified
- Found that `SendKeys`, `SendKeysLiteral` bypass `tmuxCommand()` helper
- This causes commands to target wrong tmux server when inside overmind

**2026-01-19:** Investigation completed
- Status: Complete
- Key outcome: Fix identified - update tmux functions to use `tmuxCommand()` helper
