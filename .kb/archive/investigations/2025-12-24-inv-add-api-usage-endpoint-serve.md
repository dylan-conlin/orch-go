## Summary (D.E.K.N.)

**Delta:** Added `/api/usage` endpoint to serve.go and integrated usage display into dashboard stats bar.

**Evidence:** Tests pass (TestHandleUsageMethodNotAllowed, TestHandleUsageJSONResponse, TestUsageAPIResponseJSONFormat), Svelte types check clean.

**Knowledge:** pkg/usage already has FetchUsage() that fetches from Anthropic API - just needed to expose via HTTP endpoint and consume in frontend.

**Next:** Close - implementation complete, ready for review.

**Confidence:** High (90%) - Simple endpoint addition building on existing pkg/usage infrastructure.

---

# Investigation: Add Api Usage Endpoint Serve

**Question:** How to add /api/usage endpoint to serve.go and display usage in dashboard?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Feature implementation worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: pkg/usage package already implements Claude Max usage fetching

**Evidence:** pkg/usage/usage.go contains FetchUsage() function that returns UsageInfo with FiveHour, SevenDay, SevenDayOpus, and Email fields.

**Source:** pkg/usage/usage.go:229-288

**Significance:** No need to implement API fetching logic - just need to expose via HTTP endpoint.

---

### Finding 2: serve.go follows consistent handler pattern

**Evidence:** Existing endpoints (handleAgents, handleEvents, handleAgentlog) all follow pattern: check method, fetch data, JSON encode response.

**Source:** cmd/orch/serve.go:138-562

**Significance:** Easy to add new endpoint following same pattern for consistency.

---

### Finding 3: Dashboard uses Svelte stores for data management

**Evidence:** web/src/lib/stores/agents.ts, agentlog.ts show pattern of creating stores with fetch methods and subscribing in components.

**Source:** web/src/lib/stores/agents.ts

**Significance:** Created usage.ts store following same pattern for frontend integration.

---

## Implementation

### Backend (cmd/orch/serve.go)

1. Added `github.com/dylan-conlin/orch-go/pkg/usage` import
2. Registered `/api/usage` endpoint with CORS handler
3. Created `UsageAPIResponse` struct with account, five_hour_percent, weekly_percent fields
4. Created `handleUsage` handler that calls usage.FetchUsage() and returns JSON

### Frontend (web/)

1. Created `web/src/lib/stores/usage.ts` with:
   - UsageInfo interface matching API response
   - usage store with fetch method
   - getUsageColor() helper (green <60%, yellow 60-80%, red >80%)
   - getUsageEmoji() helper for visual indicators

2. Updated `web/src/routes/+page.svelte`:
   - Added usage store import
   - Fetch usage on mount with 60s refresh interval
   - Display in stats bar with color-coded percentages

---

## References

**Files Modified:**
- cmd/orch/serve.go - Added /api/usage endpoint
- cmd/orch/serve_test.go - Added usage endpoint tests
- web/src/lib/stores/usage.ts - New usage store
- web/src/routes/+page.svelte - Added usage display

**Tests Added:**
- TestHandleUsageMethodNotAllowed
- TestHandleUsageJSONResponse
- TestUsageAPIResponseJSONFormat
