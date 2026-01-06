# Session Synthesis

**Agent:** og-inv-investigate-orch-status-21dec
**Issue:** orch-go-jc72
**Duration:** 2025-12-21 10:00 → 2025-12-21 11:30
**Outcome:** success

---

## TLDR

Investigated why orch status shows 27 agents when only 1 OpenCode session exists. Found ghost sessions come from four-layer architecture (OpenCode memory/disk, registry, tmux) with no coordinated cleanup - test confirmed 2 in-memory, 238 on disk, 3 in registry, 27 shown in status.

---

## Delta (What Changed)

### Files Created

- `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md` - Complete investigation with findings, synthesis, and recommendations

### Files Modified

- None (investigation only, no code changes)

### Commits

- `ac5d49e` - Investigation: orch status ghost sessions from four-layer architecture

---

## Evidence (What Was Observed)

- OpenCode has 2 in-memory sessions (GET /session) but 238 disk-persisted sessions for orch-go directory (GET /session with x-opencode-directory header)
- Registry shows 3 active agents but orch status displays 27 total
- Code review confirmed no reconciliation between registry, tmux, and OpenCode (pkg/registry/registry.go has no reconcile functions)
- Session.remove() in OpenCode calls Storage.remove() for disk cleanup (src/session/index.ts:remove)
- orch clean only modifies registry, never checks tmux or OpenCode (cmd/orch/main.go:1665-1707)

### Tests Run

```bash
# In-memory OpenCode sessions
curl -s http://127.0.0.1:4096/session | jq 'length'
# Result: 2

# Disk-persisted sessions for orch-go directory
curl -s -H "x-opencode-directory: $PWD" http://127.0.0.1:4096/session | jq 'length'
# Result: 238

# Active agents in registry
cat ~/.orch/agent-registry.json | jq '.agents | map(select(.status == "active")) | length'
# Result: 3

# Session directories on disk
ls ~/.local/share/opencode/storage/session/ | wc -l
# Result: 15
```

---

## Knowledge (What Was Learned)

### New Artifacts

- `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md` - Four-layer architecture analysis with reconciliation recommendations

### Decisions Made

- Decision 1: Ghost sessions are architectural, not bugs in individual components
- Decision 2: Fix should be in orch clean rather than each component (minimal breaking changes)

### Constraints Discovered

- Four-layer architecture (OpenCode memory, OpenCode disk, registry, tmux) requires coordinated cleanup
- ListSessions behavior differs by header: WITH x-opencode-directory returns disk sessions (238), WITHOUT returns memory (2)
- Registry has no visibility into tmux window state or OpenCode session state

### Externalized via `kn`

- `kn constrain "orch-go agent state exists in four layers..."` - kn-7829b4
- `kn decide "OpenCode ListSessions WITH x-opencode-directory..."` - kn-e2bef6
- `kn tried "orch clean to remove ghost sessions automatically" --failed "..."` - kn-d5132d

---

## Next (What Should Happen)

**Recommendation:** close

### If Close

- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] D.E.K.N. summary filled
- [x] Self-review passed
- [x] Knowledge externalized via kn commands
- [x] Ready for `orch complete orch-go-jc72`

**Implementation Path:**

1. Create beads issue: "Enhance orch clean with four-layer reconciliation" (type: feature)
   - For each registry active agent: verify tmux window exists (if not headless) AND OpenCode session exists via GET /session/{id}
   - Mark as abandoned if either check fails
   - Add --verify-opencode flag for disk orphan cleanup
   - Add --dry-run for preview before cleaning

2. Consider follow-up investigation: "Why does OpenCode have 238 disk sessions but only 2 in memory?"
   - Test if DELETE /session actually removes disk directories
   - Check if Storage.remove() fails silently
   - Determine if this is expected behavior for session history retention

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-orch-status-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md`
**Beads:** `bd show orch-go-jc72`
