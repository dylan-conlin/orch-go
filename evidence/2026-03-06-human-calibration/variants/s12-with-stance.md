## Knowledge

- The dashboard's AgentGrid component (web/src/components/AgentGrid.svelte)
  fetches from GET /api/v1/agents and renders all returned results.
- The grid uses a fixed CSS layout: 4 columns, max-height 600px,
  no pagination or virtual scrolling.
- Typical per-project agent count is 5-20. Total across all managed
  projects can reach 40-80 depending on activity level.
- The /api/v1/agents endpoint is consumed by both the dashboard UI
  and programmatic callers (orch status, monitoring scripts).

## Stance

Every data path has implicit consumers. When modifying a query or
data source, trace who reads the result and what assumptions they
make about shape, volume, or timing. A locally correct change can
break a downstream consumer's implicit contract.
