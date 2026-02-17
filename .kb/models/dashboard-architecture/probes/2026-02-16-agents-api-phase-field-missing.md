# Probe: Agents API Missing phase and phase_reported_at Fields

**Date:** 2026-02-16
**Status:** Active
**Issue:** orch-go-995

## Question

Why is the "Ready to Complete" section not showing completed agents on the work-graph page? The issue description states that the agents API returns `status` instead of `phase`.

## What I Tested

1. **Frontend code** (`web/src/routes/work-graph/+page.svelte:338-376`):
   - Line 345: Checks `if (agent.phase?.toLowerCase() !== 'complete') continue;`
   - Line 351: Uses `agent.phase_reported_at || agent.updated_at || agent.spawned_at` for completion timestamp

2. **Backend API struct** (`cmd/orch/serve_agents.go:27-54`):
   - Line 34: `Phase` field IS defined in `AgentAPIResponse` struct with `json:"phase,omitempty"`
   - NO `PhaseReportedAt` field in the struct definition

3. **Phase population** (`cmd/orch/serve_agents.go:778-804`):
   - Line 782: `agents[i].Phase = phaseStatus.Phase` - Phase IS being set from beads comments
   - Line 786-787: `phaseReportedAtMap[agents[i].BeadsID] = *phaseStatus.PhaseReportedAt` - Timestamp tracked internally but NOT added to API response

## What I Observed

The agents API DOES populate the `Phase` field from beads comments for active agents. However:

1. **Missing field in API response**: The `PhaseReportedAt` field is tracked internally in a map (line 787) but is NOT included in the `AgentAPIResponse` struct, so it's never sent to the frontend.

2. **Frontend depends on missing field**: The frontend code at line 351 expects `agent.phase_reported_at` to determine completion time, but this field doesn't exist in the JSON response.

3. **Completed workspaces don't get phase**: For archived/completed workspaces (lines 641-704), the code intentionally skips beads data fetch for performance (line 699-702), so these agents never get their `Phase` field populated.

## Model Impact

**Extends** the dashboard architecture model with a new finding:

The Ready to Complete section failure is caused by a **missing API field**, not a missing phase value. The `Phase` field IS populated from beads comments for active agents, but:

1. The frontend expects both `agent.phase` (for filtering) AND `agent.phase_reported_at` (for sorting/display)
2. Only `agent.phase` exists in the API response - `phase_reported_at` is tracked internally but never serialized
3. Without `phase_reported_at`, the frontend can't construct `ReadyToCompleteItem.completionAt`, causing the item to be filtered out (line 352: `if (!completionAt) continue;`)

## Fix Required

Add `PhaseReportedAt` field to `AgentAPIResponse` struct and populate it from the internal `phaseReportedAtMap`:

1. Add field to struct: `PhaseReportedAt string json:"phase_reported_at,omitempty"`
2. Populate after phase extraction: `agents[i].PhaseReportedAt = phaseStatus.PhaseReportedAt.Format(time.RFC3339)`
