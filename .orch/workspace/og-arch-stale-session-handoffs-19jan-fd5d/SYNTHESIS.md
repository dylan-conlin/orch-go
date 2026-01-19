# Session Synthesis

**Agent:** og-arch-stale-session-handoffs-19jan-fd5d
**Issue:** orch-go-20901
**Duration:** 2026-01-19 15:30 → 2026-01-19 16:15
**Outcome:** success

---

## TLDR

Fixed stale session handoff injection by gating cross-window scan and legacy fallback on presence of active/ directories. After explicit `orch session end`, no stale handoffs are injected; after crash (active/ exists), recovery works correctly.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/session.go` - Added `hasActiveSessionAnywhere()` and `hasWindowScopedDirectories()` helpers, gated cross-window scan and legacy fallback on active session presence
- `cmd/orch/session_resume_test.go` - Updated tests for new behavior, added `TestDiscoverSessionHandoff_NoHandoffAfterExplicitEnd`

### Files Created
- `.kb/investigations/2026-01-19-inv-stale-session-handoffs-injected-after.md` - Investigation document with root cause analysis and recommendation

### Commits
- (pending) `fix: prevent stale handoff injection after explicit session end`

---

## Evidence (What Was Observed)

- After `orch session end`, the session store shows `null` and `active/` is archived to timestamped directory (`session.go:576-613`)
- Cross-window scan (`scanAllWindowsForMostRecent`) was finding archived handoffs unconditionally, regardless of session state
- Legacy fallback (`session.go:1412-1456`) was also returning handoffs regardless of session state
- Test confirmed fix: with no `active/` directories, `orch session resume --check` returns exit code 1 (no handoff)
- Test confirmed recovery: with `active/` directories present, cross-window scan finds handoffs correctly

### Tests Run
```bash
go test -v ./cmd/orch/... -run TestDiscoverSessionHandoff
# PASS: all 11 tests passing including new TestDiscoverSessionHandoff_NoHandoffAfterExplicitEnd
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-19-inv-stale-session-handoffs-injected-after.md` - Full root cause analysis

### Decisions Made
- Decision: Use `active/` directory presence as the signal for "session in progress" because it's already the canonical marker for mid-session state
- Decision: Gate legacy fallback on both active session check AND window-scoped directory check to preserve pure-legacy setup behavior

### Constraints Discovered
- Constraint: Cross-window scan and legacy fallback must respect explicit session end signal - Without this gate, users get stale context injected after intentionally closing their session

### Externalized via `kn`
- None needed - the fix is implementation-level, not architectural

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + implementation)
- [x] Tests passing (11 session resume tests)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-20901`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Stale `active/` directories exist in `orch-go-1` and `orch-go-2` from Jan 17 - should there be a cleanup mechanism?

**Areas worth exploring further:**
- Session lifecycle management could benefit from a `orch session clean` command to remove stale active directories

**What remains unclear:**
- None - the fix is straightforward and well-tested

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-stale-session-handoffs-19jan-fd5d/`
**Investigation:** `.kb/investigations/2026-01-19-inv-stale-session-handoffs-injected-after.md`
**Beads:** `bd show orch-go-20901`
