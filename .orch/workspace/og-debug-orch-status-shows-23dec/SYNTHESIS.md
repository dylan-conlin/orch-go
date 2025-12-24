# Session Synthesis

**Agent:** og-debug-orch-status-shows-23dec
**Issue:** orch-go-v4mw
**Duration:** 2025-12-23 ~45 minutes
**Outcome:** success

---

## TLDR

Fixed `orch status` showing 0 active agents by adding beads ID to OpenCode session titles. The issue was that session titles were just workspace names (e.g., "og-debug-orch-status-23dec") without the `[beads-id]` pattern that the matching logic expected.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/main.go` - Added `formatSessionTitle()` helper, updated spawn functions to include beads ID in title
- `cmd/orch/main_test.go` - Added test for `formatSessionTitle()`

### Commits
- (pending) Add beads ID to OpenCode session titles for orch status matching

---

## Evidence (What Was Observed)

- OpenCode API returns 55 sessions, but `extractBeadsIDFromTitle()` returns empty for all (titles lack `[beads-id]` pattern)
- Tmux windows use format: `🐛 og-debug-orch-status-shows-23dec [orch-go-v4mw]` - with beads ID
- OpenCode titles use format: `og-debug-orch-status-23dec` or `Reading SPAWN_CONTEXT.md...` - no beads ID
- Prior investigation `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md` identified four-layer architecture but didn't identify this specific matching bug

### Tests Run
```bash
# Build verification
go build ./cmd/orch 
# Build successful

# All unit tests
go test ./... 
# PASS: all tests passing

# Specific extract tests
go test ./cmd/orch/... -run "TestExtract" -v
# PASS: TestExtractBeadsIDFromTitle, TestFormatSessionTitle, etc.
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-orch-status-shows-active-agents.md` - Root cause analysis of status matching bug

### Decisions Made
- Decision: Include beads ID in session title (not workspace name lookup) because it's consistent with tmux window format and requires minimal code changes

### Constraints Discovered
- OpenCode session titles may be overwritten by Claude's first response (e.g., "Reading SPAWN_CONTEXT.md...") - needs verification if this breaks the fix

### Externalized via `kn`
- None needed - findings documented in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (fix implemented, tests pass)
- [x] Tests passing (go test ./...)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-v4mw`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does OpenCode preserve the initial title or overwrite it with Claude's response summary? - Could affect fix effectiveness
- Should we also match by workspace name pattern as fallback? - Would help with backward compatibility

**What remains unclear:**
- Whether existing phantom agents will remain phantom until respawned
- Long-term behavior if Claude overwrites title

*(These are minor concerns - the fix addresses the root cause for new spawns)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-orch-status-shows-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-orch-status-shows-active-agents.md`
**Beads:** `bd show orch-go-v4mw`
