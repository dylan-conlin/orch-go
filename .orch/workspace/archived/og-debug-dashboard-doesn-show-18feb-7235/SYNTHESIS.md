# Synthesis: Dashboard Not Showing Newly Spawned Agents (orch-go-1066)

## Problem
After the beads-first discovery rewrite (issues 1058-1062), newly spawned OpenCode agents were invisible in the dashboard (`GET /api/agents`) while `orch status` could see them. The first agent spawned after the rewrite was invisible; a second agent spawned minutes later appeared.

## Root Cause (Two Bugs)

### Bug 1: Dashboard discovery too narrow
`listInProgressIssues` in `serve_agents_discovery.go` only returned issues with `in_progress` status. The beads-first rewrite removed fallback discovery paths that the old 5-source reconciliation had, making dashboard entirely dependent on beads issue status.

### Bug 2: Auto-created issues never transitioned to in_progress
`SetupBeadsTracking` in `pkg/orch/extraction.go` only called `UpdateIssueStatus("in_progress")` when `beadsIssueFlag != ""` (i.e., when `--issue` flag was provided). Auto-created issues (spawned without `--issue`) kept `open` status permanently.

Together: auto-created issues stayed `open`, and the dashboard only looked for `in_progress` issues.

## Fix

1. **Expanded dashboard discovery**: Renamed `listInProgressIssues` to `listActiveIssues`, changed filter to include both `open` and `in_progress` statuses
2. **Fixed status transition**: Changed condition from `beadsIssueFlag != ""` to `beadsID != ""` so auto-created issues also get updated to `in_progress`
3. **Added regression test**: `TestHandleAgentsOpenStatusIssueVisible` verifies agents with `open` status beads issues are visible in the dashboard

## Why the Second Agent Was Visible
The second agent (orch-go-1066) was spawned with `--issue orch-go-1066` (explicit issue flag), so its status was correctly updated to `in_progress`. The first agent (orch-go-1064) was auto-created without `--issue`, staying in `open` status.

## Files Changed
- `cmd/orch/serve_agents_discovery.go` - `listActiveIssues` with expanded status filter
- `cmd/orch/serve_agents_handlers.go` - Updated call site
- `pkg/orch/extraction.go` - Fixed status update condition
- `cmd/orch/serve_agents_test.go` - Updated tests + regression test
