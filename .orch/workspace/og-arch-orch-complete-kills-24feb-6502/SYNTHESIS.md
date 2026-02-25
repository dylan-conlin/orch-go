# Session Synthesis

**Agent:** og-arch-orch-complete-kills-24feb-6502
**Issue:** orch-go-1216
**Outcome:** success

---

## Plain-Language Summary

All tmux window cleanup code (`orch complete`, `orch clean`, `orch abandon`, `orch review done`) was killing windows by their index-based target (`session:3`) instead of their stable unique ID (`@1234`). Tmux window indices shift when earlier windows are killed (especially with `renumber-windows` on), so concurrent completion operations could target the wrong window — killing an unrelated agent. The fix changes all 4 call sites to use `KillWindowByID(window.ID)` instead of `KillWindow(window.Target)`. The safer function already existed; it just wasn't being used.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and acceptance criteria.

Key outcomes:
- All 4 callers migrated from `KillWindow(window.Target)` to `KillWindowByID(window.ID)`
- `go build ./cmd/orch/` passes
- `go vet ./cmd/orch/` passes
- `go test ./cmd/orch/` passes (all tests)
- `go test ./pkg/tmux/` passes (all tests)
- Zero remaining callers of index-based `KillWindow` in Go code

---

## TLDR

Fixed `orch complete` (and 3 other commands) killing wrong tmux window by changing from unstable window index targeting to stable window ID targeting. All 4 call sites now use `KillWindowByID(window.ID)` instead of `KillWindow(window.Target)`.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/complete_cleanup.go:37` - Changed `KillWindow(window.Target)` → `KillWindowByID(window.ID)`
- `cmd/orch/clean_cmd.go:496` - Changed `KillWindow(pw.window.Target)` → `KillWindowByID(pw.window.ID)`
- `cmd/orch/abandon_cmd.go:200-202` - Changed `KillWindow(windowInfo.Target)` → `KillWindowByID(windowInfo.ID)`
- `cmd/orch/review.go:921` - Changed `KillWindow(window.Target)` → `KillWindowByID(window.ID)`

### Files Created
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-orch-complete-kills-wrong-tmux-window.md` - Probe documenting the bug and fix

---

## Evidence (What Was Observed)

- `pkg/tmux/tmux.go:508-538` - `CreateWindow` returns both `windowTarget` (session:index) and `windowID` (@-prefixed stable ID)
- `pkg/tmux/tmux.go:694` - `KillWindow` takes `session:index` format
- `pkg/tmux/tmux.go:703` - `KillWindowByID` takes `@ID` format (stable)
- `pkg/tmux/tmux.go:817` - `ListWindows` builds `Target` from `session:index` (unstable)
- All 4 cleanup callers used `window.Target` (unstable) despite `window.ID` (stable) being available
- `DefaultLivenessChecker.WindowExists` in `clean_cmd.go:96` already correctly used `WindowExistsByID`

### Tests Run
```bash
go build ./cmd/orch/    # PASS
go vet ./cmd/orch/      # PASS
go test ./cmd/orch/     # PASS (all tests, 2.5s)
go test ./pkg/tmux/     # PASS (all tests, 0.39s)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-orch-complete-kills-wrong-tmux-window.md` - Probe confirming the bug

### Constraints Discovered
- **tmux window indices are NOT stable identifiers** - they change with `renumber-windows` and concurrent window operations
- **Always use @-prefixed window IDs for mutating operations** (kill, send-keys to wrong target, etc.)
- `WindowInfo.Target` should only be used for display/logging, never for targeting tmux operations

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Probe file created with Status: Complete
- [x] Ready for `orch complete orch-go-1216`

---

## Unexplored Questions

- **Should `WindowInfo.Target` be removed from the struct?** It's now unused for operations but might still be useful for display. Low priority.
- **Should `KillWindow` and `WindowExists` be deprecated?** They have no remaining callers. Could add deprecation comments.

*(Both are cosmetic cleanup, not blocking.)*

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-orch-complete-kills-24feb-6502/`
**Probe:** `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-orch-complete-kills-wrong-tmux-window.md`
**Beads:** `bd show orch-go-1216`
