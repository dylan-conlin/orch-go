# Dashboard Architecture Guide

**Status:** Stable
**Last Updated:** 2026-01-07

## Overview
The `orch-go` dashboard provides real-time visibility into agent activity across multiple projects. It uses a hybrid architecture to ensure data consistency and low latency.

## Core Patterns

### 1. Hybrid SSE + API Activity Feed
To prevent event loss on page refresh and ensure complete history, the activity feed uses two data sources:
- **SSE (Server-Sent Events):** Provides real-time updates for active agents.
- **API Fetch (`/api/session/{id}/messages`):** Fetches complete historical messages from the OpenCode storage on tab open.
- **Deduplication:** The frontend merges these streams, using the event ID to deduplicate.

### 2. Cross-Project Visibility
The dashboard is "project-aware." It correctly handles agents spawned with `--workdir` to other projects:
- **Late Filtering:** Project filtering is performed *after* the workspace cache lookup, ensuring `agent.ProjectDir` is correctly populated from `SPAWN_CONTEXT.md`.
- **Session Directory vs Project Directory:** For `--workdir` spawns, the OpenCode session directory is the orchestrator's CWD, while the `ProjectDir` is the target repo. The dashboard uses `ProjectDir` for filtering.

### 3. Beads Integration
The dashboard follows the active beads issue. When a focus is set or an issue is claimed, the dashboard prioritizes that context in the UI.

## Troubleshooting

### Registry Population Issues
If agents are missing from the dashboard, check the session registry:
- Registry is populated from `.orch/workspace/` metadata.
- Ensure `SPAWN_CONTEXT.md` exists and contains valid JSON metadata.
- Check for storage race conditions if multiple agents are spawned simultaneously.

## Related Investigations
- `2026-01-07-inv-dashboard-agents-filter-session-directory.md`
- `2026-01-07-inv-dashboard-activity-feed-implement-hybrid.md`
