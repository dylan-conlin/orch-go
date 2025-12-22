# Session Synthesis

**Agent:** og-debug-orch-send-still-21dec
**Issue:** orch-go-kszt
**Duration:** 2025-12-21
**Outcome:** success

---

## TLDR

Fixed `orch send` to work with tmux agents by adding a tmux send-keys fallback when OpenCode session ID cannot be resolved. The root cause was that session titles don't contain beads IDs, making API-based resolution fail for tmux-spawned agents.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/main.go` - Refactored `runSend` to try OpenCode API first, then fall back to tmux send-keys when session ID cannot be resolved. Added three new functions: `sendViaOpenCodeAPI`, `sendViaTmux`, and `findTmuxWindowByIdentifier`.

### Commits
- (pending) - Add tmux send-keys fallback for orch send to support tmux agents

---

## Evidence (What Was Observed)

- `resolveSessionID` fails for tmux agents because session titles contain workspace names (e.g., `og-debug-orch-send-still-21dec`), not beads IDs (e.g., `orch-go-kszt`). See `cmd/orch/main.go:1317-1327`.
- The `runTail` command works for tmux agents because it has a fallback to `tmux.CaptureLines` when API lookup fails. See `cmd/orch/main.go:468-473`.
- Window names DO contain beads IDs in format `[beads-id]` (e.g., `🐛 og-debug-... [orch-go-kszt]`), so `tmux.FindWindowByBeadsID` can locate them.

### Tests Run
```bash
# Build verification
go build -o build/orch-test ./cmd/orch 
# Success

# All tests pass
go test ./...
# PASS: all packages

# Smoke test: tmux fallback path
./build/orch-test send orch-go-hrhw "hello, can you hear me?"
# ✓ Message sent to orch-go-hrhw (via tmux workers-orch-go:5)

# Smoke test: API path (workspace with session ID)
./build/orch-test send og-inv-quick-test-verify-21dec "hello, testing API path"
# ✓ Message sent to session ses_4bc758a0affevWoGLNGREjeAKM (via API)

# Smoke test: error handling
./build/orch-test send nonexistent-id-xyz "hello"
# Error: failed to resolve session and no tmux window found: ...
```

---

## Knowledge (What Was Learned)

### Root Cause Analysis
The `orch send` command relied solely on `resolveSessionID` which:
1. Checks for `ses_xxx` prefix (direct session ID)
2. Searches `.orch/workspace/` for matching directory with `.session_id` file
3. Lists OpenCode sessions and matches by title containing the identifier
4. Finds tmux window, then still tries to match API session by title

The problem: Step 4 finds the tmux window but still requires an API session match, which fails because session titles don't contain beads IDs.

### Solution Pattern
Follow the `runTail` pattern: when OpenCode API lookup fails, fall back to direct tmux operations. For `send`, this means using `tmux send-keys` to type the message into the tmux pane.

### Constraints Discovered
- OpenCode session titles are set to workspace names, not beads IDs
- The `.session_id` file in workspaces is sometimes missing (retry mechanism isn't 100% reliable)
- Tmux window names reliably contain `[beads-id]` suffix, making `FindWindowByBeadsID` reliable

### Externalized via `kn`
- N/A (straightforward bug fix, pattern follows existing `runTail` implementation)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (runSend refactored with tmux fallback)
- [x] Tests passing (go test ./...)
- [x] Smoke tests confirm both API and tmux paths work
- [x] Ready for `orch complete orch-go-kszt`

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude
**Workspace:** `.orch/workspace/og-debug-orch-send-still-21dec/`
**Investigation:** (investigation file creation skipped - straightforward bug fix)
**Beads:** `bd show orch-go-kszt`
