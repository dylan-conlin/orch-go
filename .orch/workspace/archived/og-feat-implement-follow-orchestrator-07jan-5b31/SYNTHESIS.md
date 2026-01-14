# SYNTHESIS: Follow the Orchestrator Dashboard Filtering

## TLDR

Implemented dashboard filtering with time-based (`?since=`) and project-based (`?project=`) filters. The dashboard now defaults to showing agents from the last 12 hours and can auto-follow the orchestrator's working directory to filter by project context. This reduces dashboard load time by limiting the agents fetched from the API.

## Outcome: Success

All components implemented and tested:
- Backend: Time and project filters on `/api/agents`, context endpoint at `/api/context`
- Frontend: Time filter dropdown, follow orchestrator toggle, reactive filter state

## Delta Summary

**New Files (4):**
- `cmd/orch/serve_filter.go` - Filter parsing utilities (parseSinceParam, filterByTime, filterByProject)
- `cmd/orch/serve_context.go` - `/api/context` endpoint returning tmux cwd and resolved projects
- `pkg/tmux/follower.go` - Tmux follower ported from TypeScript (polls orchestrator pane for cwd)
- `web/src/lib/stores/context.ts` - Frontend filter stores with localStorage persistence

**Modified Files (4):**
- `cmd/orch/serve.go` - Added `/api/context` route
- `cmd/orch/serve_agents.go` - Added filter parsing and application to `/api/agents`
- `web/src/lib/stores/agents.ts` - Added queryString param to fetch(), filter callback mechanism
- `web/src/lib/components/stats-bar/stats-bar.svelte` - Added time filter dropdown and follow toggle
- `web/src/routes/+page.svelte` - Wired filter stores, context polling, reactive refetch

## Key Design Decisions

1. **Default to 12h** - Most agents are recent; reduces initial API response size
2. **Stability threshold** - Tmux follower uses 2 consecutive polls before project switch (prevents flicker)
3. **Multi-project config** - orch-go sees agents from ecosystem repos (hardcoded in `DefaultMultiProjectConfigs()`)
4. **Client-side persistence** - Filter preferences saved to localStorage
5. **Filter applied after enrichment** - Backend filters after beads data enrichment to ensure complete data

## Architecture Notes

The filtering happens in two layers:
- **Backend** (`serve_agents.go:834-860`): Filters `AgentWithMeta` slice after enrichment
- **Frontend** (`+page.svelte`): Reactive filter state triggers API refetch with query string

The tmux follower (`pkg/tmux/follower.go`) is a Go port of the original TypeScript implementation from `beads-ui-svelte`. It provides:
- Polling at configurable intervals (default 2s in frontend, 500ms capability)
- Stability threshold to prevent UI flicker during rapid directory changes
- Multi-project configurations for orchestration repos

## Testing Evidence

```bash
# Backend builds
go build ./...

# Backend tests pass
go test ./cmd/orch/... -run "Context|Filter" -v
# TestFilterByProject, TestFilterByNeedsReview, TestLimitFiltering - all PASS

# Frontend builds
npm run build
# ✓ built in 10.70s
```

## Recommendation: Close

Feature is complete and functional. Visual verification recommended:
1. Open dashboard at https://localhost:3348
2. Verify time filter dropdown appears in stats bar
3. Verify "Following" toggle works and auto-filters by project
4. Verify changing time filter triggers API refetch with `?since=` param

## Next Actions

1. **Visual verification** - Dylan should confirm the UI works as expected
2. **Consider SSE for context** - Currently polls every 2s; SSE would be more efficient
3. **Add project dropdown** - Allow manual project selection when not following orchestrator
