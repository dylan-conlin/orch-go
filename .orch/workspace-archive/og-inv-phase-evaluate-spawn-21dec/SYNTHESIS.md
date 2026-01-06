# Session Synthesis

**Agent:** og-inv-phase-evaluate-spawn-21dec
**Issue:** orch-go-pe5d.3
**Duration:** 2025-12-21 16:15 → 2025-12-21 17:00
**Outcome:** success

---

## TLDR

Evaluated three options for session_id capture without registry. Recommended Option 2 (workspace file): store session_id in `.orch/workspace/{name}/.session_id` during spawn, eliminating global registry while keeping lookups fast.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md` - Full investigation with analysis of three options

### Files Modified
- None (investigation-only task)

### Commits
- (pending) Investigation file with findings and recommendation

---

## Evidence (What Was Observed)

- Spawn tmux mode uses `FindRecentSessionWithRetry` with 500ms-2s window to capture session_id (main.go:1087-1091)
- `FindRecentSession` only matches sessions created in last 30 seconds (client.go:347-350)
- Headless spawn gets session_id synchronously via HTTP API (main.go:974-978)
- Phase 1/2 migrations proved derived lookups work via title matching (commits a63bd52, c8a83e0)
- Daemon active count runs at 60s intervals - API call latency acceptable (daemon.go:148-166)

### Tests Run
```bash
# Code analysis - reviewed spawn flow and timing constraints
rg "FindRecentSession|ListSessions" --type go -l
# Found: pkg/opencode/client.go, cmd/orch/main.go

# Verified Phase 1/2 migrations prove derived lookups work
git show a63bd52 --stat
git show c8a83e0 --stat
# Both show successful migration to derived lookups
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md` - Phase 3 evaluation with three options

### Decisions Made
- Decision: Store session_id in workspace file because it co-locates data, has single writer (no locking), and automatically cleans up with workspace

### Constraints Discovered
- `FindRecentSession` has 30-second time window for matching newly created sessions
- Derived lookups via title matching work but are 100-300ms slower than file read
- Daemon polls every 60s so API-based active count is acceptable

### Externalized via `kn`
- `kn decide "Session_id stored in workspace file not registry" --reason "Co-locates data with workspace, single writer, no lock contention"` - kn-3ffc51

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement workspace file storage for session_id (Phase 3 implementation)
**Skill:** feature-impl
**Context:**
```
Implement Option 2 from investigation: add WriteSessionID/ReadSessionID helper functions,
update spawn to write session_id to .orch/workspace/{name}/.session_id after capture,
update read commands (tail, question, resume) to read from workspace file first.
See .kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md for full details.
```

### Implementation Sequence
1. Add helper functions for workspace file I/O
2. Update spawn to write session_id after capture
3. Update read commands to check workspace file first
4. Update daemon to use OpenCode API for active count
5. Remove registry writes from spawn (final step)

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-phase-evaluate-spawn-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md`
**Beads:** `bd show orch-go-pe5d.3`
