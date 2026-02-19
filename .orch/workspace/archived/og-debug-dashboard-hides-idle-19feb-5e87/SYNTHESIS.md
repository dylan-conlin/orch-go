# SYNTHESIS: Dashboard hides idle/at-risk agents (orch-go-1112)

## TLDR

Fixed three compounding issues that caused the dashboard to hide idle agents with AT-RISK/CRITICAL context exhaustion flags, while `orch status --all` correctly showed them.

## Outcome: success

## Root Cause

Three compounding factors hid idle/at-risk agents from the dashboard:

1. **Status mapping**: Backend maps `idle` sessions to `dead` status in the API response
2. **Time filter**: Default `?since=12h` query parameter excluded dead agents whose `updated_at` was older than 12 hours
3. **Missing context risk**: Dashboard API never computed `verify.AssessContextRisk()`, so at-risk agents had no risk data even if they passed the time filter

## Delta

### Backend (`cmd/orch/`)
- **serve_agents_types.go**: Added `ContextRisk *verify.ContextExhaustionRisk` field to `AgentAPIResponse`
- **serve_agents_handlers.go**:
  - Added context risk assessment loop after token fetching (reuses existing `verify.AssessContextRisk`)
  - Modified time filter to bypass for agents needing attention (dead, awaiting-cleanup, or at-risk)

### Frontend (`web/src/lib/`)
- **stores/agents.ts**: Added `ContextRisk` interface, `context_risk` field on `Agent`, new `atRiskAgents` derived store
- **components/needs-attention/needs-attention.svelte**: Added "Context At-Risk" section with agent cards, included in attention count (deduped against dead/awaiting-cleanup)
- **components/agent-card/agent-card.svelte**: Added context risk badge (CRITICAL red / AT-RISK orange) with tooltip showing token usage details

## Evidence

- `go build ./cmd/orch/` passes
- `go vet ./cmd/orch/` passes
- `npm run build` succeeds (frontend compiles clean)
- All pre-existing svelte-check errors are in unrelated files (work-graph, knowledge-tree)

## Knowledge

- The dashboard's time filter was the primary blocker - dead agents with old timestamps were silently excluded
- Context risk was already computed in `orch status` but never surfaced in the dashboard API
- The fix follows the existing pattern: NeedsAttention component already handles dead/stalled/awaiting-cleanup agents

## Next Actions

- Verify on live dashboard with idle agents present
- Consider adding a "show all" toggle to bypass time filter entirely
