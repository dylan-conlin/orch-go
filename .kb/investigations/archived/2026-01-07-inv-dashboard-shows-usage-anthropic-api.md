## Summary (D.E.K.N.)

**Delta:** Anthropic usage API returns null for all fields when no activity in current billing period; dashboard was showing 0% instead of N/A.

**Evidence:** API curl returned `{"five_hour":null,"seven_day":null,...}`; Go backend used `float64` (defaults to 0); frontend displayed "0%".

**Knowledge:** Null usage data is expected behavior from Anthropic API; must use pointer types to preserve null distinction through the data pipeline.

**Next:** Complete - fix implemented and verified via browser screenshot.

**Promote to Decision:** recommend-no (tactical fix, not architectural pattern)

---

# Investigation: Dashboard Shows 0% When Usage Data Unavailable

**Question:** Why does the dashboard show 0% usage when Anthropic API returns null for all usage fields?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Anthropic API Returns Null for Inactive Periods

**Evidence:** API call to `https://api.anthropic.com/api/oauth/usage` returns:
```json
{"five_hour":null,"seven_day":null,"seven_day_opus":null,...}
```

**Source:** Direct curl to API endpoint with valid OAuth token

**Significance:** Null is the expected response when no usage activity in current billing period - this is not an error.

---

### Finding 2: Go Backend Lost Null Distinction

**Evidence:** `UsageAPIResponse` struct in `serve_system.go` used `float64` fields which default to `0` in Go, making null indistinguishable from zero.

**Source:** `cmd/orch/serve_system.go:24-34`

**Significance:** The data pipeline collapsed null → 0, preventing frontend from knowing data was unavailable.

---

### Finding 3: Frontend Had No Unavailable State

**Evidence:** `usage.ts` types used `number` without null, and `+layout.svelte` displayed any number as percentage.

**Source:** `web/src/lib/stores/usage.ts`, `web/src/routes/+layout.svelte`

**Significance:** Even if backend sent null, frontend couldn't handle it.

---

## Synthesis

**Key Insights:**

1. **Null preservation through pipeline** - Each layer (API → Go → JSON → TypeScript → UI) must preserve null semantics for "unavailable" to display correctly.

2. **Expected API behavior** - Null usage is normal for Anthropic API, not an error condition.

3. **Type design matters** - Using `*float64` (pointer) vs `float64` in Go is the difference between preserving and losing null.

**Answer to Investigation Question:**

Dashboard showed 0% because Go's `float64` defaults to 0, losing the distinction between "0% usage" and "usage data unavailable (null)". Fixed by using `*float64` pointers in Go and `number | null` in TypeScript.

---

## Implementation (Complete)

**Changes Made:**

1. `cmd/orch/serve_system.go` - Changed percentage fields to `*float64` pointers
2. `web/src/lib/stores/usage.ts` - Updated types to `number | null`
3. `web/src/routes/+layout.svelte` - Added `formatPercent()` helper returning "N/A" for null, muted styling

**Verification:** Browser screenshot confirms "N/A | N/A @personal" displays in header.

---

## References

**Files Examined:**
- `cmd/orch/serve_system.go` - Backend API response structure
- `pkg/usage/usage.go` - Usage fetching logic
- `web/src/lib/stores/usage.ts` - Frontend usage store
- `web/src/routes/+layout.svelte` - Dashboard header UI

**Commands Run:**
```bash
# Verified API returns null
curl -s https://api.anthropic.com/api/oauth/usage -H "Authorization: Bearer $TOKEN"

# Verified profile shows valid account
curl -s https://api.anthropic.com/api/oauth/profile -H "Authorization: Bearer $TOKEN"
```

**Commit:** `7116f6e1` - fix(dashboard): show N/A instead of 0% when usage data unavailable
