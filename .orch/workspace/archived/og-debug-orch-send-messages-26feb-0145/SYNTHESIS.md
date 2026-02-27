# Session Synthesis

**Agent:** og-debug-orch-send-messages-26feb-0145
**Issue:** orch-go-w1qt
**Outcome:** success

---

## Plain-Language Summary

`orch send` to tmux-backend (Claude CLI) agents failed because of two bugs working together. First, the session resolution code treated tmux window IDs (like `@339`) stored in Claude-backend workspaces as valid OpenCode session IDs, silently sending the message to a non-existent API endpoint that returned 204 OK. Second, when messages did reach the tmux send-keys fallback path, there was no delay between typing text and pressing Enter — the TUI hadn't finished processing the pasted characters when Enter arrived, so the submit was missed. The Python orch-cli already solved both issues (1-second delay in send.py, window ID targeting), but the Go rewrite dropped these patterns.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## TLDR

Fixed `orch send` for tmux agents by (1) filtering out tmux window IDs from OpenCode session resolution so messages actually reach the tmux path, (2) adding a 500ms delay between text and Enter in tmux send-keys to let TUIs process pasted text before submit, and (3) using stable window IDs instead of volatile session:index for targeting.

---

## Delta (What Changed)

### Files Modified
- `pkg/tmux/tmux.go` - Added `SendTextAndSubmit()` function with configurable delay between text and Enter, plus `DefaultSendDelay` constant (500ms)
- `pkg/tmux/tmux_test.go` - Added integration test `TestSendTextAndSubmit` and `TestDefaultSendDelay`
- `cmd/orch/send_cmd.go` - Updated `sendViaTmux()` to use stable window ID targeting and `SendTextAndSubmit` with delay
- `cmd/orch/shared.go` - Added `isOpenCodeSessionID()` helper; updated `resolveSessionID()` to skip non-OpenCode session IDs (like tmux window IDs); updated `findTmuxWindowByIdentifier()` to search orchestrator and meta-orchestrator sessions
- `pkg/spawn/claude.go` - Updated `SendClaude()` to use `SendTextAndSubmit` with delay

---

## Evidence (What Was Observed)

- Python orch-cli `send.py:101-103` has explicit 1-second delay with comment: "Without this sleep, Enter gets processed before message is fully pasted"
- Python orch-cli `complete.py:65-66` has 0.5-second delay with same pattern
- Python orch-cli uses stable `window_id` (not session:index) for targeting
- Claude-backend workspaces store tmux window IDs (e.g., `@339`) in `.session_id`, not OpenCode session IDs
- Before fix: `orch send orch-go-w1qt "test"` → "✓ Message sent to session @339 (via API)" (silently dropped)
- After fix: `orch send orch-go-w1qt "test"` → "✓ Message sent to orch-go-w1qt (via tmux @339)" (delivered)

### Tests Run
```bash
go test -count=1 -v ./pkg/tmux/
# PASS: 37 tests including TestSendTextAndSubmit (2.04s) and TestDefaultSendDelay

go vet ./cmd/orch/ ./pkg/tmux/ ./pkg/spawn/
# No issues
```

---

## Architectural Choices

### Delay value: 500ms vs 1s
- **What I chose:** 500ms default delay
- **What I rejected:** 1-second delay (Python orch-cli's value)
- **Why:** The Python comment says "1.0s is reliable" for initial spawn (cold TUI). For follow-up messages to a warm TUI, 500ms should be sufficient. The delay is configurable via the `SendTextAndSubmit` parameter.
- **Risk accepted:** 500ms might occasionally be insufficient under heavy system load

### Session ID filtering: prefix check vs format validation
- **What I chose:** Simple `strings.HasPrefix(id, "ses_")` check
- **Why:** OpenCode session IDs consistently start with `ses_`. Tmux window IDs start with `@`. This is a reliable discriminator without needing regex or complex parsing.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Claude-backend agents store tmux window IDs in `.session_id` files, not OpenCode session IDs — any code reading session_id must distinguish between the two formats
- TUI applications (Claude Code, OpenCode) need a delay between receiving literal text and the Enter key via tmux send-keys — without it, the submit is intermittently missed

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (37/37 in pkg/tmux)
- [x] Fix verified via smoke test (message delivered and received)
- [x] Ready for `orch complete orch-go-w1qt`

---

## Unexplored Questions

- Should `SendTextAndSubmit` have adaptive delay based on message length? Longer messages take more time for the TUI to process.
- Should `resolveSessionID` be refactored to make the OpenCode vs tmux session type explicit throughout the resolution pipeline?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-orch-send-messages-26feb-0145/`
**Beads:** `bd show orch-go-w1qt`
