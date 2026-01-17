# Synthesis: Add Daemon Utilization Metrics to Dashboard

**Agent:** og-feat-add-daemon-utilization-17jan-a32f
**Issue:** orch-go-zeeua
**Date:** 2026-01-17

## Summary

Added daemon utilization metrics to the `/api/daemon` endpoint to surface the ratio of daemon-spawned vs manual-spawned agents. This tracks where triage discipline is slipping (manual spawns bypassing the daemon workflow).

## What Was Delivered

### New Files
- `pkg/daemon/utilization.go` - Utilization metrics computation from events.jsonl
- `pkg/daemon/utilization_test.go` - Tests for utilization metrics

### Modified Files
- `cmd/orch/serve_system.go` - Extended `/api/daemon` endpoint with utilization metrics

### API Changes

The `/api/daemon` endpoint now includes an optional `utilization` field with these metrics:

```json
{
  "utilization": {
    "total_spawns": 336,
    "daemon_spawns": 90,
    "manual_spawns": 246,
    "daemon_spawn_rate": 26.78,  // % from daemon (higher is better)
    "triage_bypassed": 404,
    "triage_slip_rate": 100,     // % bypassed triage (lower is better, capped at 100)
    "auto_completions": 55,
    "analysis_period": "Last 7 days",
    "days_analyzed": 7
  }
}
```

### Query Parameters

- `days` - Analysis window in days (default: 7, max: 90)
  - Example: `GET /api/daemon?days=30`

## Key Metrics

| Metric | Purpose | Ideal Direction |
|--------|---------|-----------------|
| `daemon_spawn_rate` | % of spawns from daemon | Higher is better (more automated) |
| `triage_slip_rate` | % of spawns that bypassed triage | Lower is better (better discipline) |
| `manual_spawns` | Count of non-daemon spawns | Lower is better |
| `auto_completions` | Daemon-driven completions | Higher indicates healthy automation |

## Technical Decisions

1. **Reused events.jsonl aggregation pattern** from `stats_cmd.go` rather than creating new data sources
2. **Capped triage slip rate at 100%** since historical data can have more bypass events than spawn events (spawn failures after bypass logged)
3. **Made utilization optional** in API response - gracefully handles missing events file
4. **Added configurable time window** via `days` query param (default 7, max 90)

## Testing

```bash
# Verified API response
curl -sk https://localhost:3348/api/daemon | jq .utilization

# Tested time windows
curl -sk "https://localhost:3348/api/daemon?days=1" | jq .utilization
curl -sk "https://localhost:3348/api/daemon?days=30" | jq .utilization

# All tests pass
go test ./pkg/daemon/... -v
```

## Next Steps for Dashboard UI

The web UI at `http://localhost:5188` can now consume these metrics. Suggested display:

1. **Daemon Health Card** - Show daemon spawn rate as primary gauge
2. **Triage Discipline** - Highlight slip rate with warning thresholds
3. **Time Window Selector** - Allow switching between 1d/7d/30d views

## References

- Investigation: `.kb/investigations/2026-01-17-inv-add-daemon-utilization-metrics-dashboard.md`
- Beads issue: `orch-go-zeeua`
