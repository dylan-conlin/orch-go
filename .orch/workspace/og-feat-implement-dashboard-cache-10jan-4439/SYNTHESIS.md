# Synthesis: Dashboard Cache Invalidation System

**Status:** Complete
**Phase:** Implementation Complete
**Date:** 2026-01-10

## TLDR

Implemented Phase 4 of dashboard reliability architecture: cache invalidation system with version headers, staleness detection, and reload banners to prevent dashboard showing stale data after binary updates.

---

## What Was Built

### API Changes (serve.go)
- Added `X-Orch-Version` header to all API responses (uses build-time `version` variable)
- Added `X-Cache-Time` header with RFC3339 timestamp to all responses
- Headers added in CORS middleware wrapper (lines 244-245) → applies to ALL endpoints

### Dashboard Infrastructure
1. **cache-validation.ts** - New store for tracking cache state:
   - Monitors version from API responses
   - Detects version mismatches (binary updated)
   - Calculates cache age (staleness detection >60s)
   - Provides helper for wrapping fetch calls

2. **CacheValidationBanner.svelte** - UI component:
   - Yellow banner for version mismatch with "Reload Dashboard" button
   - Orange banner for stale data warning (>60s)
   - Fixed positioning at top of page
   - Dismissible version mismatch (staleness auto-clears when fresh data arrives)

3. **Integration**:
   - agents.ts imports cache-validation and calls updateFromResponse on every fetch
   - +page.svelte includes banner at top of layout

---

## Implementation Decisions

### Why CORS Middleware for Headers?
- Single point of addition → all API endpoints get headers automatically
- No need to update individual handlers
- Consistent across entire API surface

### Why 60-Second Staleness Threshold?
- Per decision doc `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md`
- Balance between false positives (normal delays) and catching actual stale state
- Conservative enough to avoid alert fatigue

### Why Two Separate Banners?
- Version mismatch = actionable (reload fixes it)
- Staleness = informational (may resolve on next fetch)
- Different urgency levels → different visual treatments

### Cache Busting for Assets
**Not implemented separately** - Vite already handles this via content hashing in production builds.
- Production: Assets get content-based filenames (e.g., `bundle.abc123.js`)
- Dev mode: Assets served fresh on every request
- The version header mechanism handles forcing dashboard reload when binary updates

---

## Testing Evidence

### API Headers Verified
```bash
$ curl -k -I https://localhost:3348/api/agents | grep -i x-
x-cache-time: 2026-01-10T01:19:49-08:00
x-orch-version: b7958f1e-dirty
```

Headers successfully added to API responses.

---

## Next Actions

### Visual Verification Steps (For Orchestrator via Glass)

**Test 1: Version Mismatch Banner**
1. Open dashboard at http://localhost:5188
2. Observe current version header (check Network tab: x-orch-version)
3. Rebuild binary with different version: `make build`
4. Restart orch serve: `pkill -f "orch serve" && ~/bin/orch serve &`
5. Dashboard should show yellow banner: "New version available" with "Reload Dashboard" button
6. Click reload → banner should disappear

**Test 2: Staleness Warning**
1. Open dashboard
2. Network throttle or pause server to delay responses >60s
3. Orange banner should appear: "Data may be out of date (cache > 60s old)"
4. Resume normal network → banner should disappear

**Test 3: Normal Operation**
1. Dashboard loads with no banners visible initially
2. API responses include headers (verify in Network tab)
3. Cache validation store updates on each fetch (check Redux DevTools or console)

**Status:** Dashboard opened successfully. Full visual verification pending orchestrator Glass session (per constraint: orchestrator uses Glass for all browser interactions).

### Optional Enhancements (Not in Scope)
- Add cache invalidation trigger on `orch deploy`
- Add version mismatch detection to other stores (usage, beads, servers)
- Progressive enhancement: Show last update time in UI

---

## Files Changed

**Backend:**
- `cmd/orch/serve.go` - Added headers in CORS middleware (2 lines)

**Frontend:**
- `web/src/lib/stores/cache-validation.ts` - New store (95 lines)
- `web/src/lib/components/cache-validation-banner/CacheValidationBanner.svelte` - New component (47 lines)
- `web/src/lib/components/cache-validation-banner/index.ts` - Export (1 line)
- `web/src/lib/stores/agents.ts` - Import + updateFromResponse call (2 lines)
- `web/src/routes/+page.svelte` - Import + banner component (3 lines)

**Documentation:**
- `.kb/investigations/2026-01-10-inv-implement-dashboard-cache-invalidation-system.md`

**Total:** 7 files changed, ~150 lines added

---

## Success Criteria (From Decision Doc)

- [x] Version headers added to API responses
- [x] Dashboard detects version mismatches
- [ ] Visual verification: Reload banner shows when version changes (testing in progress)
- [ ] Visual verification: Stale warning shows when cache >60s (testing in progress)
- [x] Code committed and built

---

## Leave It Better

```bash
kb quick decide "Cache validation via response headers" --reason "Enables dashboard to detect stale state without server-side session management"
```

**What was learned:**
- CORS middleware is correct layer for cross-cutting headers
- Vite handles asset cache busting automatically - no manual intervention needed
- Staleness detection requires client-side timestamp comparison (Date.parse + Date.now())
