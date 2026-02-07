---
status: active
blocks:
  - keywords:
      - add vite dev server
      - add bun run dev
      - separate web server
      - dashboard hot reload
      - port 5188
---

# Decision: Serve Dashboard as Static Build, Eliminate bun Dev Server

**Date:** 2026-02-07
**Status:** Active
**Context:** Chronic system instability from bun dev server zombie process accumulation

## Problem

The dashboard was served by a Vite dev server (`bun run dev` on port 5188) as a separate process from the Go API server (port 3348). This caused:

- **Zombie bun processes:** 13 orphaned bun processes found consuming 40%+ CPU. Every crash/restart left orphans because bun child processes weren't cleaned up.
- **Two servers for one UI:** Unnecessary complexity — two ports, two processes, two things that can fail.
- **4-process runtime:** api + web + daemon + opencode. Each additional process is another failure mode.

**Evidence:** Feb 7 2026 session found 15 bun processes for orch-go (13 orphans with PPID=1). Killing them dropped CPU from 75% to 38%.

## Decision

Serve the dashboard as pre-built static assets from the Go binary (`orch serve` on port 3348). Eliminate `bun run dev` entirely.

## What Changed

| Aspect | Before | After |
|--------|--------|-------|
| Dashboard URL | `http://localhost:5188` | `https://localhost:3348` |
| Process count | 4 (api + web + daemon + opencode) | 3 (api + daemon + opencode) |
| Dashboard serving | Vite dev server (bun) | Go static file handler |
| Hot-reload | Yes (instant on save) | No (requires rebuild) |
| Rebuild trigger | Automatic (Vite watches files) | `orch-dashboard restart` (auto-detects changes) |
| Zombie risk | High (bun spawns child processes) | None (no bun process) |

## Tradeoffs

**Accepted:**
- No hot-reload during dashboard development. Agents don't use hot-reload anyway, and Dylan rarely edits Svelte by hand. When needed, can temporarily run `bun run dev` on a side port.

**Gained:**
- Entire category of failure eliminated (bun zombie processes)
- Simpler runtime (3 processes instead of 4)
- Single port for everything (3348)
- `orch-dashboard restart` handles web asset rebuilding automatically

## Escape Hatch

If hot-reload is needed for intensive UI work:
```bash
cd web && bun run dev --port 5188
```
This runs alongside the production setup temporarily. Kill it when done.

## Alternatives Considered

**Option A: Fix bun zombie cleanup (wrapper script)**
- Implemented as Phase 1 bandaid (`scripts/dashboard-web-dev.sh`)
- Rejected as permanent solution: manages the problem instead of eliminating it

**Option B: Keep dev server but add process supervision**
- Rejected: adds complexity to manage complexity. Wrong direction.

## References

- Phase 1 investigation: 13 zombie bun processes at 40% CPU (this conversation)
- `.kb/models/dashboard-architecture.md` — dashboard architecture overview
- `.kb/guides/resilient-infrastructure-patterns.md` — escape hatch pattern
