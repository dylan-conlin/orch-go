# Session Synthesis

**Agent:** og-debug-verify-whether-orch-13feb-8d46
**Issue:** orch-go-g8k
**Duration:** 2026-02-13
**Outcome:** success

---

## TLDR

Verified that `orch clean` does NOT kill live headless OpenCode sessions in normal usage. Normal headless spawns are protected by `.session_id` workspace tracking (Layer 1 defense). The only edge-case vulnerability is in `--verify-opencode` for sessions that lack `.session_id` and are idle >5min. The prior kb tried claim ("only touches registry") is outdated — clean now has 7 distinct cleanup actions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-13-inv-verify-whether-orch-clean-kills-headless-sessions.md` - Complete investigation with 6 findings, code path analysis for all 7 cleanup actions

### Files Modified
- None (investigation-only task)

### Commits
- Investigation file + SYNTHESIS.md

---

## Evidence (What Was Observed)

- `clean_cmd.go:295-511` dispatches to 7 independent cleanup paths based on flags; default (no flags) is read-only
- `headless.go:60` writes `.session_id` to workspace, which `cleanOrphanedDiskSessions()` uses as Layer 1 protection
- `purgeGhostAgents()` (line 1141) only calls `agentReg.Remove()`, NOT `DeleteSession()` — ghosts flag is registry-only
- `cleanPhantomWindows()` (line 740) only calls `tmux.KillWindow()` — phantoms flag is tmux-only
- `cleanOrphanedDiskSessions()` (line 574-588) has 3-layer defense but Layer 2 (5-min recency check) gates Layer 3 (`IsSessionProcessing`) — idle sessions skip the processing check
- `CleanStaleSessions()` (cleanup/sessions.go:64) checks `IsSessionProcessing()` for ALL candidates (stronger than `--verify-opencode`)
- Daemon periodic cleanup defaults: 7-day threshold, 6-hour interval, preserves orchestrators

### Tests Run
```bash
# Code audit (no code changes to test)
# Verified all 7 code paths via source reading
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-13-inv-verify-whether-orch-clean-kills-headless-sessions.md` - Complete analysis of all orch clean code paths

### Decisions Made
- N/A (investigation only, no code changes)

### Constraints Discovered
- `--verify-opencode` skips `IsSessionProcessing()` for sessions idle > 5 minutes that lack workspace `.session_id` — minor defense-in-depth gap
- `--stale` archival can make sessions "orphaned" from `--verify-opencode`'s perspective in subsequent runs (but only for completed workspaces)

### Externalized via `kn`
- N/A (findings are in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with 6 findings)
- [x] No code changes to test
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-g8k`

### Follow-up Work (optional, not blocking)
- Consider adding `IsSessionProcessing()` check to non-recent path in `cleanOrphanedDiskSessions` as defense-in-depth
- Update kb tried entry for "orch clean to remove ghost sessions automatically" to reflect current 7-action architecture

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What happens if `WriteSessionID` fails silently (e.g., disk full, race condition)? The headless backend logs a warning but continues — session would be untracked
- Does OpenCode `DELETE /session/{id}` terminate an in-flight agent or just remove metadata? This affects severity of `--verify-opencode` gap
- Behavior under concurrent `orch clean --all` + active spawn (race conditions between cleanup and spawn)

**What remains unclear:**
- How often `WriteSessionID` failures actually occur in production
- Whether there are any interactive sessions (no workspace) that could be affected by `--verify-opencode`

---

## Verification Contract

**Investigation artifact:** `.kb/investigations/2026-02-13-inv-verify-whether-orch-clean-kills-headless-sessions.md`
**Key outcome:** Bug NOT confirmed for normal usage. Edge case identified in `--verify-opencode` for untracked sessions.
**VERIFICATION_SPEC.yaml:** See workspace root

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-verify-whether-orch-13feb-8d46/`
**Investigation:** `.kb/investigations/2026-02-13-inv-verify-whether-orch-clean-kills-headless-sessions.md`
**Beads:** `bd show orch-go-g8k`
