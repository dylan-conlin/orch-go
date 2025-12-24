## Summary (D.E.K.N.)

**Delta:** Added servers status panel to dashboard showing running/total server count across projects.

**Evidence:** API endpoint returns correct data, Svelte store works, stats bar displays indicator, all tests pass.

**Knowledge:** The pattern for adding dashboard panels is: API endpoint in serve.go → Svelte store → page.svelte import + UI.

**Next:** Close issue, servers status is now visible in dashboard.

**Confidence:** High (95%) - Implementation complete and tested.

---

# Implementation: Add Servers Status Panel Dashboard

**Question:** How should we add servers status visibility to the dashboard?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%+)

---

## Deliverables

### 1. API Endpoint: `/api/servers`

**File:** `cmd/orch/serve.go`

Added `handleServers` function that:
- Reads port allocations from `~/.orch/ports.yaml` via port registry
- Lists running workers sessions via tmux
- Returns project-grouped data with running status

Response structure:
```json
{
  "projects": [
    {
      "project": "orch-go",
      "ports": [{"service": "web", "port": 5188}, {"service": "api", "port": 3348}],
      "running": true,
      "session": "workers-orch-go"
    }
  ],
  "total_count": 3,
  "running_count": 1,
  "stopped_count": 2
}
```

### 2. Svelte Store: `servers.ts`

**File:** `web/src/lib/stores/servers.ts`

Created store following the same pattern as `usage.ts`:
- `fetch()` method to get data from API
- TypeScript interfaces for type safety
- Error handling with error property

### 3. Dashboard Integration

**File:** `web/src/routes/+page.svelte`

Added to stats bar:
- Import `servers` store
- Fetch on mount and every 60 seconds
- Display indicator showing `🖥️` when servers running, `💤` when all stopped
- Show running count / total count

---

## Files Changed

1. `cmd/orch/serve.go` - Added `/api/servers` endpoint and handler
2. `web/src/lib/stores/servers.ts` - New store file
3. `web/src/routes/+page.svelte` - Import store, fetch, display in stats bar

---

## Testing

- Go tests: `go test ./...` - All pass
- Svelte check: `npm run check` - No errors
- Build: `go build ./cmd/orch/` - Success
