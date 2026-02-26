## Summary (D.E.K.N.)

**Delta:** Implemented beads stats display in dashboard stats bar showing ready/blocked issue counts.

**Evidence:** All tests pass (5 Playwright tests, Go build passes), feature displays beads stats with ready count and blocked indicator.

**Knowledge:** `bd stats --json` provides all needed beads stats; pattern follows existing usage/focus/servers stores.

**Next:** Feature complete - close issue.

**Confidence:** High (90%) - straightforward implementation following established patterns.

---

# Investigation: Add Beads Stats Dashboard Stats

**Question:** How to display beads issue stats (ready, blocked, open) in dashboard stats bar?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: `bd stats --json` provides all needed data

**Evidence:** `bd stats --json` output:
```json
{
  "summary": {
    "total_issues": 1167,
    "open_issues": 194,
    "in_progress_issues": 16,
    "blocked_issues": 17,
    "ready_issues": 177,
    "closed_issues": 955,
    "average_lead_time_hours": 37.47
  }
}
```

**Source:** `bd stats --json` command output

**Significance:** Single command provides all beads stats needed for dashboard display.

---

### Finding 2: Existing store pattern works well

**Evidence:** usage, focus, and servers stores all use same pattern:
- Create writable store
- Fetch from API endpoint
- Refresh every 60 seconds

**Source:** `web/src/lib/stores/usage.ts`, `web/src/lib/stores/focus.ts`, `web/src/lib/stores/servers.ts`

**Significance:** Following established pattern ensures consistency and reduces implementation risk.

---

### Finding 3: Stats bar layout supports additional indicators

**Evidence:** Stats bar already displays: active, recent, archive, errors, focus, servers - beads indicator follows same pattern.

**Source:** `web/src/routes/+page.svelte` lines 330-400

**Significance:** No layout changes needed, just add another indicator following the pattern.

---

## Implementation

**Changes made:**

1. **Backend** (`cmd/orch/serve.go`):
   - Added `/api/beads` endpoint
   - Shells out to `bd stats --json`
   - Returns BeadsAPIResponse with key stats

2. **Frontend Store** (`web/src/lib/stores/beads.ts`):
   - Created beads store following usage/focus pattern
   - Fetches from `/api/beads`

3. **Stats Bar** (`web/src/routes/+page.svelte`):
   - Import beads store
   - Fetch on mount and every 60s
   - Display ready count with blocked indicator

4. **Tests** (`web/tests/stats-bar.spec.ts`):
   - Test beads indicator displays with mocked data
   - Test blocked count hidden when zero

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Added /api/beads endpoint
- `web/src/lib/stores/beads.ts` - New beads store
- `web/src/routes/+page.svelte` - Stats bar display
- `web/tests/stats-bar.spec.ts` - Playwright tests
