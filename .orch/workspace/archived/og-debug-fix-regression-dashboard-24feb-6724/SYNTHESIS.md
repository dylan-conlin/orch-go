# Fix: Dashboard oscillates between correct status and "unassigned" (orch-go-1183)

## Root Cause

Three independent code paths check tmux state for claude-backend agents during each dashboard poll cycle:

1. `/api/agents` → `queryTrackedAgents()` → `checkTmuxWindowLiveness()` → `tmux.FindWindowByWorkspaceNameAllSessions()`
2. `/api/beads/graph` → `buildActiveAgentMap()` → `tmux.ListWorkersSessions()` + `tmux.ListWindows()` (independent tmux calls)
3. `/api/attention` → `AgentCollector` → internal HTTP call to `/api/agents`

Each tmux check shells out multiple times (list-sessions, has-session, list-windows per session). When these independent checks disagree due to intermittent tmux command failures:

- `/api/beads/graph` tmux check fails → no `active_agent` on the graph node
- `/api/agents` tmux check succeeds → agent status = "active" or "awaiting-cleanup"
- Frontend gets inconsistent data: no `active_agent` AND no `attentionBadge` → falls through to "unassigned"

Next poll cycle: both tmux checks succeed → correct display. This creates the oscillation.

## Fix (Two-Pronged)

### 1. Tmux liveness cache (query_tracked.go)

Added a 10-second TTL cache to `checkTmuxWindowLiveness()`. The cache uses `sync.RWMutex` for concurrent safety. Within the 10s TTL window, all callers see the same cached result regardless of tmux command transient failures.

**Why 10s?** The dashboard polls every 30s. A 10s TTL ensures that within any single poll cycle, all endpoints that check tmux state see consistent results, while still refreshing frequently enough to detect actual agent death promptly.

### 2. Unified agent map source (serve_beads.go)

Rewrote `buildActiveAgentMap()` (used by `/api/beads/graph`) to use `globalTrackedAgentsCache` as its primary data source instead of making independent tmux calls. This means:

- `/api/beads/graph` and `/api/agents` now share the same underlying data for tracked agents
- The independent `tmux.ListWorkersSessions()` + `tmux.ListWindows()` calls were eliminated
- OpenCode session lookup is still used as a supplementary source for untracked agents
- Phase enrichment via beads comments is still done for agents missing phase data
- Removed unused `tmux` import from serve_beads.go

## Files Changed

| File | Change |
|------|--------|
| `cmd/orch/query_tracked.go` | Added `tmuxLivenessCache` (10s TTL) with `sync.RWMutex`; rewrote `defaultCheckTmuxWindowLiveness()` to check cache before shelling out |
| `cmd/orch/serve_beads.go` | Rewrote `buildActiveAgentMap()` to use `globalTrackedAgentsCache`; removed unused `tmux` import |

## Verification

- `go build ./cmd/orch/` — passes
- `go vet ./cmd/orch/` — passes
- `go test ./cmd/orch/` — all tests pass
