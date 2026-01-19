# Session Synthesis

**Agent:** og-inv-test-spawn-orch-19jan-524c
**Issue:** (ad-hoc spawn - no beads tracking)
**Duration:** 2026-01-19 → 2026-01-19
**Outcome:** success

---

## TLDR

Identified root cause of `orch send` tmux fallback failure: `SendKeys`, `SendKeysLiteral`, and other functions in `pkg/tmux/tmux.go` bypass the `tmuxCommand()` helper, causing commands to target overmind's tmux server instead of the main tmux server where worker windows live.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-19-inv-test-spawn-orch-send-debugging.md` - Investigation documenting root cause analysis

### Files Modified
- None (investigation only)

### Commits
- (pending final commit)

---

## Evidence (What Was Observed)

- `pkg/tmux/tmux.go:562-564` - `SendKeys` uses `exec.Command("tmux", ...)` directly, bypassing `tmuxCommand()`
- `pkg/tmux/tmux.go:568-571` - `SendKeysLiteral` uses `exec.Command("tmux", ...)` directly
- `pkg/tmux/tmux.go:104-116` - `tmuxCommand()` helper correctly adds `-S mainSocket` flag when inside overmind
- `pkg/tmux/tmux.go:28-62` - `detectMainSocket()` detects overmind by checking if `$TMUX` contains "overmind"
- Multiple other functions (11 total) also bypass the helper

### Tests Run
```bash
# Code review via Read tool
# No code changes to test - investigation only
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-19-inv-test-spawn-orch-send-debugging.md` - Root cause analysis of orch send tmux failure

### Decisions Made
- Decision: Fix should update ALL tmux functions to use `tmuxCommand()`, not just SendKeys/SendKeysLiteral, because the inconsistency is a pervasive pattern (11 functions affected)

### Constraints Discovered
- Constraint: When running inside overmind, tmux commands MUST use the `-S mainSocket` flag to target the correct tmux server. Functions that use raw `exec.Command("tmux", ...)` will target overmind's tmux server instead.

### Externalized via `kn`
- N/A (constraint already documented in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, implementation is separate issue)

### If Close
- [x] All deliverables complete (investigation file with root cause + recommendations)
- [x] Tests passing (investigation only - no code changes)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for orchestrator review

### Recommended Implementation
The debug agent working on `orch-go-hudeh` should:
1. Update `SendKeys` to use `tmuxCommand()` helper
2. Update `SendKeysLiteral` to use `tmuxCommand()` helper
3. Update remaining 9 functions that bypass the helper
4. Run `go test ./pkg/tmux/...` to verify no regressions
5. Smoke test: Run `orch send` from overmind context to verify fix

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why were some functions written to use `tmuxCommand()` while others use raw `exec.Command`? Was this intentional or incremental oversight?

**Areas worth exploring further:**
- Whether the `tmuxCommandCurrent()` variant (for current-context operations) is being used correctly throughout

**What remains unclear:**
- Whether there are any edge cases where bypassing `tmuxCommand()` was intentional

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-test-spawn-orch-19jan-524c/`
**Investigation:** `.kb/investigations/2026-01-19-inv-test-spawn-orch-send-debugging.md`
**Beads:** (ad-hoc spawn - no beads tracking)
