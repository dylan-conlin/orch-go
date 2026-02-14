# Session Synthesis

**Agent:** og-debug-orch-status-shows-13feb-14bb
**Issue:** orch-go-py1
**Duration:** 2026-02-13
**Outcome:** success

---

## TLDR

Fixed ghost agents inflating `orch status` Active count (8 → 3) by fixing two phantom detection bugs and adding `orch clean --ghosts` to purge stale registry entries. Root causes: "api-stalled" sessions not treated as phantom, and dead tmux Window references persisting from registry cache.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/status_cmd.go` - Fixed phantom detection: (1) clear Window when tmux window dead, (2) include "api-stalled" in phantom check
- `cmd/orch/clean_cmd.go` - Added `--ghosts` flag and `purgeGhostAgents()` function (~80 lines) that cross-references registry against live tmux/OpenCode
- `cmd/orch/clean_test.go` - Updated TestCleanAllFlagLogic to include ghosts flag

### Files Created
- `.kb/investigations/2026-02-13-inv-orch-status-shows-ghost-agents.md` - Root cause investigation

---

## Evidence (What Was Observed)

- Before fix: `orch status --json` showed Active=8 with zero live tmux windows and zero live OpenCode sessions
- After fix: Active=3 (actual live agents), Phantom=15 (correctly detected ghosts)
- `orch clean --ghosts --dry-run` correctly identified all 15 ghost agents across all ghost types:
  - Cross-project (pw-8966, pw-8975): api-stalled, no live backing
  - Completed (orch-go-y55): tmux-stalled, beads issue closed
  - Untracked (orch-go-untracked-*): tmux-stalled, no beads ID
  - Abandoned (orch-go-1): api-stalled, no live session

### Tests Run
```bash
go build ./cmd/orch/   # Clean build
go vet ./cmd/orch/     # No issues
go test ./cmd/orch/    # PASS (2.181s) - all tests passing
go test ./cmd/orch/ -run "TestClean|TestStatus" -v  # 5 passed, 0 failed
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Phantom detection must check ALL sentinel session IDs (tmux-stalled AND api-stalled), not just tmux-stalled
- Window field in AgentInfo must reflect live state, not registry cache — clearing it when tmux window is dead is critical
- Registry as spawn-time cache means ghost accumulation is inevitable without periodic reconciliation

### Verification Contract
- **Reproduction steps:** `orch status --json` → check `swarm.active` count
- **Before fix:** Active included agents with no live tmux or OpenCode backing
- **After fix:** Active only includes agents with verified live sessions
- **Regression test:** TestCleanAllFlagLogic verifies --ghosts flag included in --all

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (phantom detection fix + orch clean --ghosts)
- [x] Tests passing (go test ./cmd/orch/ - PASS 2.181s)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-py1`

---

## Unexplored Questions

- Should `orch clean --ghosts` be run automatically as part of `orch status` (filter-on-read) rather than requiring explicit cleanup?
- Should the daemon periodically run ghost reconciliation?

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-orch-status-shows-13feb-14bb/`
**Investigation:** `.kb/investigations/2026-02-13-inv-orch-status-shows-ghost-agents.md`
**Beads:** `bd show orch-go-py1`
