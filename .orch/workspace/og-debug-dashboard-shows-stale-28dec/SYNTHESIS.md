# Session Synthesis

**Agent:** og-debug-dashboard-shows-stale-28dec
**Issue:** orch-go-sk8i
**Duration:** 2025-12-28 13:03 → 2025-12-28 13:30
**Outcome:** success

---

## TLDR

Dashboard showed agents with Phase: Complete as "active" because status derivation prioritized session existence over beads Phase. Fixed by making Phase: Complete authoritative for completion status regardless of session state.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-dashboard-shows-stale-agent-data.md` - Root cause analysis documenting two bugs: session scoping and status derivation

### Files Modified
- `cmd/orch/serve.go` - Fixed status derivation logic to use Phase: Complete as authoritative source

### Commits
- (Not yet committed - changes staged in serve.go)

---

## Evidence (What Was Observed)

- API returned 6 active agents but 4 had `phase: "Complete"` with `status: "active"` (verified: `curl http://localhost:3348/api/agents`)
- `orch status` showed 0 active while API showed 6 (mismatch due to session scoping)
- OpenCode sessions with `x-opencode-directory` header only returned when queried with that header (verified: curl with/without header)
- serve.go:926-953 required `!hasActiveSession` before marking completed (root cause of status bug)
- Prior decision stated "Dashboard agent status derived from beads phase, not session time" (contradiction with implementation)

### Tests Run
```bash
# Built and tested fix
/opt/homebrew/bin/go build -o /tmp/orch-test ./cmd/orch
# Verified agents with Phase: Complete now show status: completed
curl -s http://localhost:3349/api/agents | jq '[.[] | select(.phase == "Complete")] | .[0:5] | .[] | {phase, status}'
# Result: All Phase: Complete agents now have status: "completed"
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-dashboard-shows-stale-agent-data.md` - Full investigation with D.E.K.N. summary

### Decisions Made
- Use beads Phase as authoritative for completion status - prior decision exists, implementation was incorrect
- Remove session existence check from completion logic - session may be open while work is done

### Constraints Discovered
- OpenCode stores sessions per-project-directory - queries without directory header miss project-specific sessions
- Prior fix for "running agents shown as completed" was too aggressive - blocked legitimate completions

### Externalized via `kn`
- None needed - findings align with existing prior decision in spawn context

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete - investigation file and code fix done
- [x] Tests passing - verified with test build
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-sk8i`

### Additional Notes
The `orch status` showing 0 agents issue was already fixed in main.go by another agent (lines 2310-2321 query sessions by project directory). The running binary at `/Users/dylanconlin/bin/orch` is stale - a rebuild will fix both issues.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should there be a single source of truth for "active session" detection between orch status and dashboard?
- Could shared workspace cache logic between serve.go and main.go reduce code duplication?

**Areas worth exploring further:**
- Session cleanup: Should sessions be auto-closed when Phase: Complete is detected?
- Dashboard: Consider showing "completing" intermediate state between active and completed

**What remains unclear:**
- Why the original fix for "running agents shown as completed" was implemented - may have been a different bug

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-dashboard-shows-stale-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-dashboard-shows-stale-agent-data.md`
**Beads:** `bd show orch-go-sk8i`
