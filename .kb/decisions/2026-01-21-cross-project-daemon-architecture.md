# Decision: Cross-Project Daemon Architecture

**Date:** 2026-01-21
**Status:** Implemented
**Decision Maker:** Dylan + Claude

## Context

The orch daemon was originally designed for single-project operation, running from a fixed working directory and polling only that project's beads issues. This required:
- Running multiple daemon instances (one per project)
- Manual coordination of which projects get daemon coverage
- No shared capacity management across projects

**Pain point:** When running an orchestrator in Docker (for rate limit escape hatch), it cannot spawn agents directly (Docker not available in Claude Code sandbox). The orchestrator must delegate via `bd create -l triage:ready` for the daemon to pick up. But a single-project daemon only sees one project's issues.

## Question

How should the daemon poll multiple beads directories to support spawning agents across different projects?

## Decision

**Use `kb projects list` for project discovery with global capacity pool.**

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  orch daemon run --cross-project                            │
│                                                             │
│  1. kb projects list --json → [Project{Name, Path}, ...]   │
│  2. For each project (sorted by name):                      │
│     - ListReadyIssuesForProject(projectPath)                │
│     - Collect issues with triage:ready label                │
│  3. Sort all issues by priority (P1 > P2 > P3)              │
│  4. For highest priority issue:                             │
│     - SpawnWorkForProject(beadsID, projectPath)             │
│     - Passes --workdir to orch work                         │
│  5. Global capacity pool limits total concurrent agents     │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Choices

| Choice | Decision | Rationale |
|--------|----------|-----------|
| Project registry | `kb projects list` | Reuses existing infrastructure, no new config file |
| Capacity model | Global (shared) | Prevents runaway spawning (N projects × M agents = too many) |
| Polling strategy | Sequential iteration | Simple, deterministic, adequate for ~20 projects |
| Error handling | Log warning, continue | One project failing shouldn't block others |
| Activation | `--cross-project` flag | Backward compatible, opt-in |

### Implementation Files

| File | Purpose |
|------|---------|
| `pkg/daemon/projects.go` | `ListProjects()` - parse kb projects list |
| `pkg/daemon/issue_adapter.go` | `ListReadyIssuesForProject()`, `SpawnWorkForProject()` |
| `pkg/daemon/daemon.go` | `CrossProjectOnce()`, `CrossProjectPreview()` |
| `cmd/orch/daemon.go` | `--cross-project` flag for run/preview/once |

## Alternatives Considered

### Option B: Per-project daemon instances (status quo)
- **Pros:** Isolation, simpler per-daemon logic
- **Cons:** Requires running N daemons, no shared capacity management
- **Rejected because:** Doesn't solve containerized orchestrator delegation

### Option C: Separate ~/.orch/projects.yaml registry
- **Pros:** Independent of kb, daemon-specific config per project
- **Cons:** Another config file, duplicates kb functionality
- **Rejected because:** kb projects already exists and works

## Consequences

**Positive:**
- Single daemon covers all projects
- Containerized orchestrators can delegate via `bd create -l triage:ready`
- Global capacity prevents resource exhaustion
- Backward compatible (flag defaults to false)

**Negative:**
- Projects must be kb-registered to be daemon-visible
- Polling latency increases with more projects (acceptable for 60s interval)
- All projects share same capacity limit (no per-project quotas)

## Evidence

- Prior investigation: `.kb/investigations/2026-01-06-inv-cross-project-daemon-single-daemon.md`
- Implementation investigation: `.kb/investigations/2026-01-21-inv-cross-project-daemon-poll-multiple.md`
- Tests: 12 cross-project tests passing in `pkg/daemon/daemon_test.go`
- Production: Successfully polling 17 projects via launchd daemon

## Related

- `.kb/guides/daemon.md` - Daemon guide with cross-project section
- `.kb/constraints/kb-4c03e4-docker-spawn-host-only.md` - Container spawn constraint
