# Session Synthesis

**Agent:** og-arch-dashboard-activity-feed-07jan-1787
**Issue:** orch-go-i2911
**Duration:** 2026-01-07 09:00 → 2026-01-07 10:15
**Outcome:** success

---

## TLDR

Investigated dashboard activity feed event loss and designed hybrid SSE + API architecture: SSE for real-time updates, OpenCode API for historical events on-demand. OpenCode already persists all session data to disk and exposes it via GET /session/:sessionID/message endpoint.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md` - Architecture investigation with recommendation

### Files Modified
- `.orch/features.json` - Added feat-040 for dashboard activity feed historical event loading

### Commits
- (pending) - architect: dashboard activity feed persistence design

---

## Evidence (What Was Observed)

- Current implementation stores 1000 events globally across all agents (`agents.ts:363-364`), diluting per-agent visibility
- OpenCode stores all session data to `~/.local/share/opencode/storage/` with messages and parts persisted to JSON files
- OpenCode exposes GET `/session/:sessionID/message` endpoint returning `MessageV2.WithParts[]` (`server.ts:1067-1104`)
- Part types (text, tool, reasoning, step-start, step-finish) map directly to dashboard filter categories
- Activity tab already filters by session ID (`activity-tab.svelte:76-86`) - just needs history fetched

### Tests Run
```bash
# Verified OpenCode storage structure
ls ~/.local/share/opencode/storage/
# message migration part project session session_diff todo

# Verified session message files exist
ls ~/.local/share/opencode/storage/message/ | head -20
# ses_466c3cf12ffeqq8yuKoDU108mH...
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md` - Full architecture design

### Decisions Made
- Decision: Fetch historical events from OpenCode API on-demand (not cache in orch-go or browser)
  - Rationale: OpenCode is source of truth, already persists everything, avoids duplicate storage

### Constraints Discovered
- Dashboard SSE already exhausts HTTP/1.1 connection pool (6 per origin) - prior constraint
- OpenCode API requires session to be in same project OR x-opencode-directory header for cross-project

### Externalized via `kn`
- None needed - investigation establishes the pattern

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Feature added to features.json (feat-040)
- [x] Ready for `orch complete orch-go-i2911`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle very long sessions (1000+ messages) - pagination needed?
- Should history be preloaded on dashboard load or lazy-loaded per agent tab?

**Areas worth exploring further:**
- Cross-project session fetching may need orch-go proxy with x-opencode-directory header
- Performance benchmarking for API fetch time

**What remains unclear:**
- Browser memory impact of storing historical events per-agent (untested)

---

## Session Metadata

**Skill:** architect
**Model:** opus (assumed)
**Workspace:** `.orch/workspace/og-arch-dashboard-activity-feed-07jan-1787/`
**Investigation:** `.kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md`
**Beads:** `bd show orch-go-i2911`
