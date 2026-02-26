## Summary (D.E.K.N.)

**Delta:** Added focus drift indicator to dashboard stats bar showing current focus goal and drift status.

**Evidence:** Go tests pass, TypeScript check passes, API endpoint returns focus data correctly.

**Knowledge:** Focus package already provides CheckDrift() which compares active beads IDs against focused issue.

**Next:** Close - feature implemented and working.

**Confidence:** High (90%) - API tested manually, TypeScript compiles, tests pass.

---

# Investigation: Add Focus Drift Indicator Dashboard

**Question:** How to display focus goal and drift status in the dashboard stats bar?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Focus package provides all needed functionality

**Evidence:** `pkg/focus/focus.go` exports:
- `Focus` struct with Goal, BeadsID, SetAt fields
- `Store.Get()` returns current focus
- `Store.CheckDrift(activeIssues)` compares active work against focus

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/focus/focus.go:22-46`

**Significance:** No new backend logic needed - just need to expose via API endpoint.

---

### Finding 2: Dashboard stats bar is in +page.svelte

**Evidence:** Stats bar is at lines 322-371 of +page.svelte, uses a flex container with dividers between stats sections.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte:322-371`

**Significance:** Pattern is clear - add new section after errors, before connect button.

---

### Finding 3: Usage store pattern provides template

**Evidence:** `web/src/lib/stores/usage.ts` shows standard pattern:
- Define interface for API response
- Create writable store with fetch() method
- Export helper functions for formatting

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/usage.ts`

**Significance:** Follow same pattern for focus store.

---

## Implementation

### Changes Made

1. **`cmd/orch/serve.go`** - Added `/api/focus` endpoint:
   - `FocusAPIResponse` struct with goal, beads_id, set_at, is_drifting, has_focus
   - `handleFocus()` reads focus from store, checks drift against active sessions
   - Registered route and added to console output

2. **`web/src/lib/stores/focus.ts`** - New store:
   - `FocusInfo` interface matching API response
   - `createFocusStore()` with fetch() method
   - `getDriftEmoji()` helper (🎯 = focused, ⚠️ = drifting)

3. **`web/src/routes/+page.svelte`** - Added to stats bar:
   - Import focus store and helper
   - Fetch focus data on mount and every 60 seconds
   - Display indicator after errors section when has_focus is true
   - Shows emoji + "focused"/"drifting" text with appropriate colors

---

## Verification

- ✅ `go build ./...` succeeds
- ✅ `go test ./...` all pass
- ✅ `bun run check` (svelte-check) shows 0 errors
- ✅ `curl http://127.0.0.1:3348/api/focus` returns expected JSON
- ✅ Playwright tests: 18 passed, 4 skipped, 1 pre-existing flaky test

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Added API endpoint
- `web/src/lib/stores/focus.ts` - New store (created)
- `web/src/routes/+page.svelte` - UI integration

**API Endpoint:**
```bash
curl http://127.0.0.1:3348/api/focus
# Returns: {"goal":"...","beads_id":"...","set_at":"...","is_drifting":false,"has_focus":true}
```
