## Summary (D.E.K.N.)

**Delta:** Fixed NaNm runtime display by adding null/undefined guards to formatDuration().

**Evidence:** API returns `spawned_at: null` for completed agents; formatDuration() now returns '-' instead of 'NaNm'.

**Knowledge:** Completed agents in registry lack timestamp data; UI must handle missing data gracefully.

**Next:** Close - fix implemented and verified.

**Confidence:** High (95%) - Simple fix with clear root cause.

---

# Investigation: Fix NaNm Runtime Display Agent

**Question:** Why do completed agent cards show "NaNm" for runtime?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%+)

---

## Findings

### Finding 1: formatDuration() doesn't guard against invalid inputs

**Evidence:** The function directly calculates `Date.now() - new Date(isoDate).getTime()` without checking if isoDate is valid. When isoDate is null/undefined/empty, `new Date()` returns Invalid Date and `.getTime()` returns NaN.

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:46-54`

**Significance:** This is the root cause of "NaNm" display.

---

### Finding 2: Completed agents lack spawned_at timestamps

**Evidence:** 
```json
{
  "id": "og-arch-alpha-opus-synthesis-20dec",
  "status": "completed",
  "spawned_at": null,
  "runtime": null
}
```

**Source:** `curl http://localhost:3348/api/agents`

**Significance:** The API returns null for completed agent timestamps, so the UI must handle this gracefully.

---

### Finding 3: Active agents have valid data

**Evidence:**
```json
{
  "id": "og-feat-fix-nanm-runtime-24dec",
  "status": "active", 
  "spawned_at": "2025-12-24T08:01:54-08:00",
  "runtime": "1m 51s"
}
```

**Source:** `curl http://localhost:3348/api/agents`

**Significance:** The fix only affects completed agents; active agents continue to work correctly.

---

## Synthesis

**Key Insights:**

1. **Root cause is missing data, not display logic** - The API returns null timestamps for completed agents.

2. **Simple guard is sufficient** - Adding null check and isNaN guard handles all edge cases.

3. **Fallback display is '-'** - A hyphen is appropriate for "no data available" in duration context.

**Answer to Investigation Question:**

Completed agent cards show "NaNm" because formatDuration() receives null/undefined spawned_at values from the API for completed agents. The fix adds guards at the start of the function to return '-' when input is missing or invalid.

---

## References

**Files Examined:**
- `web/src/lib/components/agent-card/agent-card.svelte:46-54` - formatDuration function
- `web/src/lib/stores/agents.ts:15-45` - Agent interface definition

**Commands Run:**
```bash
# Verify API data
curl -s http://localhost:3348/api/agents | jq '[.[] | select(.status == "completed")] | .[0:2]'

# TypeScript check
npm run check
```
