# Project Config Migration

This guide covers migration from legacy flat policy keys in `.orch/config.yaml` to the nested typed schema.

## What changed

Legacy flat keys are deprecated and will be removed after the migration window.
Use nested keys under `daemon`, `dashboard`, `spawn`, and `completion`.

When `orch` loads `.orch/config.yaml`, it now:

- Detects deprecated flat keys.
- Maps them to nested keys in memory.
- Rewrites `.orch/config.yaml` with nested keys (auto-migration).
- Prints a deprecation notice to stderr.

## Legacy key mapping

| Legacy flat key | New nested key |
|---|---|
| `daemon_cleanup_interval_minutes` | `daemon.cleanup.interval_minutes` |
| `daemon_cleanup_sessions_age_days` | `daemon.cleanup.sessions_age_days` |
| `daemon_cleanup_workspaces_age_days` | `daemon.cleanup.workspaces_age_days` |
| `daemon_dead_session_interval_minutes` | `daemon.dead_session.interval_minutes` |
| `daemon_max_dead_session_retries` | `daemon.dead_session.max_retries` |
| `daemon_orphan_reap_interval_minutes` | `daemon.orphan_reap.interval_minutes` |
| `daemon_dashboard_watchdog_interval_seconds` | `daemon.dashboard_watchdog.interval_seconds` |
| `daemon_dashboard_watchdog_failures_before_restart` | `daemon.dashboard_watchdog.failures_before_restart` |
| `daemon_dashboard_watchdog_restart_cooldown_minutes` | `daemon.dashboard_watchdog.restart_cooldown_minutes` |
| `dashboard_agents_active_minutes` | `dashboard.agents.active_minutes` |
| `dashboard_agents_ghost_display_hours` | `dashboard.agents.ghost_display_hours` |
| `dashboard_agents_dead_minutes` | `dashboard.agents.dead_minutes` |
| `dashboard_agents_stalled_minutes` | `dashboard.agents.stalled_minutes` |
| `dashboard_agents_beads_fetch_hours` | `dashboard.agents.beads_fetch_hours` |
| `spawn_context_quality_threshold` | `spawn.context_quality.threshold` |
| `completion_auto_rebuild_timeout_seconds` | `completion.auto_rebuild.timeout_seconds` |
| `completion_transcript_export_timeout_seconds` | `completion.transcript_export.timeout_seconds` |
| `completion_cache_invalidate_timeout_seconds` | `completion.cache_invalidate.timeout_seconds` |

## Example

Before:

```yaml
daemon_cleanup_interval_minutes: 360
spawn_context_quality_threshold: 20
completion_auto_rebuild_timeout_seconds: 120
```

After:

```yaml
daemon:
  cleanup:
    interval_minutes: 360
spawn:
  context_quality:
    threshold: 20
completion:
  auto_rebuild:
    timeout_seconds: 120
```

## Migration window

- Current behavior: flat keys still load, emit deprecation warnings, and auto-migrate.
- Future behavior: flat keys will be removed after the migration window.
