# Session Synthesis

**Agent:** og-arch-fix-dashboard-show-07jan-a3b7
**Issue:** orch-go-hq6sg
**Duration:** 2026-01-07 → 2026-01-07
**Outcome:** success

---

## TLDR

Fixed dashboard to show agents older than 2h instead of hiding them entirely. Added `is_stale` boolean field to API response and visual indicator (📦 badge) in dashboard so users know phase/task data may be outdated.

---

## Delta (What Changed)

### Files Created
- None

### Files Modified
- `cmd/orch/serve_agents.go` - Added `IsStale` field to `AgentAPIResponse`, changed logic from `continue` to marking agents as stale
- `web/src/lib/stores/agents.ts` - Added `is_stale?: boolean` to Agent interface
- `web/src/lib/components/agent-card/agent-card.svelte` - Added visual indicator (📦 badge with tooltip) for stale agents
- `.kb/investigations/2026-01-07-inv-fix-dashboard-show-older-agents.md` - Investigation documentation

### Commits
- Not yet committed (pending verification)

---

## Evidence (What Was Observed)

- `serve_agents.go:328-335` had `continue` statement that completely excluded agents older than 2h (`beadsFetchThreshold`)
- The 2h threshold was a performance optimization to avoid 400+ beads RPC calls (would take 3+ seconds)
- Frontend already has progressive disclosure (Active/Recent/Archive sections) that can handle stale agents
- Existing `IsStale` pattern in `review.go` provided precedent for the API field approach

### Tests Run
```bash
# Build verification
go build ./cmd/orch
# SUCCESS

# Frontend build
cd web && bun run build
# SUCCESS

# Related tests
go test ./cmd/orch/... -run "Stale|Agent" -v
# PASS: all tests passing
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-fix-dashboard-show-older-agents.md` - Investigation and solution documentation

### Decisions Made
- Decision: Use `is_stale` boolean field instead of new status type, because it's minimal change, follows existing patterns, and preserves current status logic
- Decision: Set stale agents' status to "idle" (they're not actively running), mark with `is_stale: true`
- Decision: Skip beads fetch for stale agents (preserves performance optimization)

### Constraints Discovered
- The 2h beadsFetchThreshold exists for performance reasons and must be respected for beads fetch, but agents can still be shown with `is_stale` marker

### Externalized via `kn`
- None required (tactical bug fix)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (API + frontend changes)
- [x] Tests passing (go test, bun run build)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-hq6sg`

---

## Unexplored Questions

Straightforward session, no unexplored territory.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus
**Workspace:** `.orch/workspace/og-arch-fix-dashboard-show-07jan-a3b7/`
**Investigation:** `.kb/investigations/2026-01-07-inv-fix-dashboard-show-older-agents.md`
**Beads:** `bd show orch-go-hq6sg`
