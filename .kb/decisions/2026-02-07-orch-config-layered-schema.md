---
status: active
blocks:
  - keywords:
      - orch config layering
      - config precedence
      - global project cli overrides
      - spawn defaults
      - context quality threshold
      - daemon cleanup thresholds
---

# Decision: Layered Orch Config Schema (Global + Project + CLI)

**Date:** 2026-02-07
**Status:** Decided
**Context:** orch-go-21426 (review `~/.orch/config.yaml` scope and layering)
**Authority:** Architectural - affects spawn, daemon, serve, complete, and cross-project behavior

---

## Problem

`orch` already has two config files (`~/.orch/config.yaml` and `.orch/config.yaml`), but policy is split between config, command flags, and hardcoded defaults. This makes behavior hard to predict and tune across projects.

Examples of policy currently not centralized:
- Daemon cleanup/dead-session/watchdog defaults live in flags.
- Dashboard activity/dead/stalled thresholds are hardcoded.
- Context gate threshold default is code constant.
- Completion timeouts are hardcoded.

---

## Decision

Adopt a **single layered configuration contract** for all orchestrator policy:

1. **Built-in defaults** (code)
2. **Global user config** (`~/.orch/config.yaml`)
3. **Project override config** (`.orch/config.yaml`)
4. **CLI flags** (ephemeral invocation overrides)

This is the precedence for standard values. For safety constraints, stricter bounds win.

---

## Layering Rules

### Standard values

`effective = CLI > project > global > built-in`

Examples:
- Spawn backend default
- Spawn tier default
- Model defaults
- Daemon poll interval

### Safety-bounded values

For threshold/limit values where unsafe overrides are possible, enforce guardrails:

- `effective_min = max(global.min, project.min)`
- `effective_max = min(global.max, project.max)`
- CLI may tighten, but may not relax outside effective bounds unless explicit force+reason semantics exist.

Examples:
- Context quality gating threshold
- Dead session retry cap
- Completion verification timeouts

---

## Scope Ownership

### Global (`~/.orch/config.yaml`) owns machine/user policy

- User-wide defaults and safety posture
- Daemon operating policy
- Model routing policy
- Completion policy defaults
- Dashboard preference defaults

### Project (`.orch/config.yaml`) owns repo-local behavior

- Backend/model defaults specific to this repository
- Spawn defaults specific to this codebase
- Server metadata (`servers`, `domain`)
- Optional local threshold overrides within global bounds

### CLI flags own one-off execution intent

- Temporary deviations for a single run
- Must remain highest priority for normal values
- Must remain auditable through existing event logs

---

## Recommended Schema

```yaml
version: 1

# Existing keys retained (compat)
backend: opencode
disabled_backends: [docker]
auto_export_transcript: true
default_model: openai/gpt-5.3-codex
skill_models:
  investigation: openai/gpt-5.3-codex

spawn:
  default_tier: skill-default   # skill-default | light | full
  default_mode: headless        # headless | tmux | inline
  include_servers: auto         # auto | always | never
  default_validation: tests     # none | tests | smoke-test
  default_variant: auto         # auto | high | max | none
  max_agents: 5
  auto_init: false
  context_quality:
    gate_on_gap: false
    threshold: 20
    min_threshold: 10

model_routing:
  default_model: openai/gpt-5.3-codex
  skill_models:
    architect: openai/gpt-5.3-codex
  backend_defaults:
    opencode: deepseek
    claude: opus
  rules:
    - when: issue_label_prefix:model
      action: use_label_value

daemon:
  poll_interval: 60
  max_agents: 3
  label: triage:ready
  verbose: true
  reflect:
    enabled: true
    interval_minutes: 60
    create_issues: true
  cleanup:
    enabled: true
    interval_minutes: 360
    sessions:
      enabled: true
      age_days: 7
    workspaces:
      enabled: true
      age_days: 7
    investigations:
      enabled: true
    preserve_orchestrator: true
  dead_session:
    enabled: true
    interval_minutes: 10
    max_retries: 2
  orphan_reap:
    enabled: true
    interval_minutes: 5
  dashboard_watchdog:
    enabled: true
    interval_seconds: 30
  sort_mode: priority

dashboard:
  agents:
    active_minutes: 10
    dead_minutes: 3
    stalled_minutes: 15
    ghost_display_hours: 4
    beads_fetch_hours: 2

completion:
  require_phase_complete: true
  auto_rebuild:
    enabled: true
    timeout_seconds: 120
  transcript_export:
    enabled: true
    timeout_seconds: 10
```

Project overrides should support only project-scoped subsets (for example `spawn`, `model_routing.backend_defaults`, `model_routing.skill_models`, `spawn_mode`, `servers`, `domain`) and must not own machine-global daemon launchd policy.

---

## Why This Design

1. **Preserves existing architecture**: keeps both global and project config files already used by orch.
2. **Moves policy out of code constants**: thresholds/timeouts/defaults become explicit and reviewable.
3. **Maintains ergonomic overrides**: CLI flags still win for one-off runs.
4. **Adds safety guardrails**: prevents accidental policy weakening via local overrides.

---

## Trade-offs Accepted

- **Larger schema surface area** in exchange for explicitness and predictability.
- **Migration work required** to map legacy flat keys and flag defaults into nested structure.
- **Validation complexity increases** because some fields are standard precedence while others are bounded.

---

## Migration Plan

1. Introduce typed v1 schema structs and merge engine (no behavior change yet).
2. Continue reading legacy keys and map them to v1 effective config.
3. Migrate consumers command-by-command (`spawn`, `daemon`, `serve`, `complete`).
4. Emit deprecation warnings for legacy keys after parity is verified.
5. Remove legacy branches after two release cycles.

---

## Evidence

- `pkg/userconfig/userconfig.go` - current global schema and getters
- `pkg/config/config.go` - current project schema and defaults
- `cmd/orch/backend.go` - backend precedence chain
- `cmd/orch/spawn_cmd.go` - model resolution precedence
- `cmd/orch/spawn_usage.go` - tier resolution precedence
- `cmd/orch/daemon.go` - daemon policy currently flag-centric
- `cmd/orch/serve_agents_collect.go` - dashboard thresholds currently hardcoded
- `pkg/spawn/gap.go` - context gap default threshold constant
- `cmd/orch/complete_helpers.go` and `cmd/orch/complete_actions.go` - completion timeout constants

---

## Related

- Investigation: `.kb/investigations/2026-02-07-inv-review-orch-config-yaml-design.md`
- Decision: `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md`
- Decision: `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md`
