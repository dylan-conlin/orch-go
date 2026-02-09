# Decision Index

**Purpose:** Keep high-value architectural decisions discoverable so stale-review sessions can validate relevance instead of re-deriving context.

**Last reviewed:** 2026-02-09

---

## Superseded During This Review

- `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` -> superseded by `.kb/decisions/2026-01-21-cross-project-daemon-architecture.md`
- `.kb/decisions/2026-01-26-disk-cleanup-chrome-code-sign-clone.md` -> superseded by `.kb/decisions/2026-01-28-two-tier-disk-cleanup-infrastructure.md`
- `.kb/decisions/2026-01-17-event-sourced-monitoring-architecture.md` -> superseded by `.kb/decisions/2026-02-07-agent-completion-lifecycle-separation.md` for completion authority

---

## Surviving Decisions by Domain

### Operating Principles and Calibration

- `.kb/decisions/2026-01-14-trust-calibration-assert-knowledge.md`
- `.kb/decisions/2026-01-14-understanding-lag-pattern.md`
- `.kb/decisions/2026-01-14-verification-bottleneck-principle.md`
- `.kb/decisions/2026-01-14-models-track-architecture.md`

### Spawn, Orchestration, and Authority Boundaries

- `.kb/decisions/2026-01-21-strategic-first-gate-advisory-only.md`
- `.kb/decisions/2026-01-28-orchestrator-action-space-architectural-constraint.md`
- `.kb/decisions/2026-01-31-investigation-lineage-enforcement.md`

### Daemon, Completion, and Workspace Lifecycle

- `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md`
- `.kb/decisions/2026-01-17-file-based-workspace-state-detection.md`
- `.kb/decisions/2026-01-17-three-tier-workspace-hierarchy.md`
- `.kb/decisions/2026-01-17-five-tier-completion-escalation-model.md`
- `.kb/decisions/2026-01-21-cross-project-daemon-architecture.md`

### Dashboard and Runtime Integration

- `.kb/decisions/2026-01-24-readable-frontier-over-graph-visualization.md`
- `.kb/decisions/2026-01-30-strategic-center-dashboard-architecture.md`
- `.kb/decisions/2026-01-30-dynamic-hud-pattern-opencode.md`
- `.kb/decisions/2026-01-30-sse-reconnection-resilience-patterns.md`
- `.kb/decisions/2026-01-30-opencode-binary-resolution-patterns.md`

### Substrate and Local Environment Constraints

- `.kb/decisions/2026-01-21-colima-over-docker-desktop.md`
- `.kb/decisions/2026-01-28-two-tier-disk-cleanup-infrastructure.md`
- `.kb/decisions/2026-01-30-decidability-graph-substrate-options.md`

---

## How to Use This Index

1. Run `kb reflect --type stale --format json`.
2. For each candidate, check whether it belongs in one of the domains above.
3. If outdated, mark `status: superseded` and add `superseded_by` in frontmatter.
4. If still active, link it from the guide/model where that behavior is explained.
