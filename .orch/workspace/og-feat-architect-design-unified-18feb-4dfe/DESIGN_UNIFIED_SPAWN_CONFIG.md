# Unified Spawn Config Precedence Model

## Problem

Spawn configuration comes from overlapping sources (CLI flags, beads labels, project config, user config, heuristics, hardcoded defaults). When precedence is implicit or inconsistent, spawns pick unexpected backend/model/tier and create hard-to-debug behavior. Recent audits show multiple overrides and defaults silently winning over user intent.

## Goals

- Make precedence explicit and consistent across backend/model/tier/mode/MCP.
- Preserve user intent: explicit choices always win over heuristics.
- Avoid silent overrides from implicit defaults.
- Make resolved values traceable (source + reason).

## Non-goals

- Changing the spawn workflow (manual vs daemon) or triage friction.
- Reworking skill inference or beads lifecycle.
- Implementing new config formats.

## Should We Unify Precedence?

Yes. The current behavior depends on implicit defaults (project config ApplyDefaults, user config DefaultConfig) and ad-hoc overrides (daemon setting spawnModel, infra escape hatch). This causes spawns to diverge from user intent. A unified, explicit precedence model eliminates silent overrides and reduces the class of "config source conflicts" that create repeated bugs.

## Current Config Sources (Observed)

- CLI flags: --backend, --model, --mode, --tmux/--headless/--inline, --light/--full, --mcp, --phases, --validation, --issue, --workdir, --no-track, --skip-artifact-check, --gate-on-gap, --skip-gap-gate, --gap-threshold, --orientation-frame, --design-workspace.
- Beads labels: skill:_, needs:_ (MCP inference), issue type -> skill inference.
- Project config: .orch/config.yaml (spawn_mode, models, servers, claude/opencode sections).
- User config: ~/.orch/config.yaml (backend, models, default_model, default_tier, daemon settings).
- Heuristics: infra detection (escape hatch), task scope -> tier inference, skill defaults.
- Hardcoded defaults: default backend "opencode", default model "sonnet", default mode "tdd", default validation "tests".

## Proposed Unified Precedence Model

### 1) Explicitness is the primary axis

Only _explicit_ values should outrank other sources. Defaults (including ApplyDefaults and DefaultConfig) should not be treated as explicit.

Definition of explicit:

- CLI flag provided.
- Config file exists AND key is set in file.
- Beads labels present.
- Issue fields set (type, title, description).

### 2) Precedence Layers (highest to lowest)

**Layer A: CLI flags (per-spawn explicit)**

- --backend, --model, --mode, --tmux/--headless/--inline, --light/--full, --mcp, --phases, --validation.

**Layer B: Beads issue overrides (explicit metadata)**

- skill:\* label overrides skill inference.
- needs:\* label overrides MCP selection.

**Layer C: Project config (explicit keys only)**

- .orch/config.yaml keys explicitly set (spawn_mode, models aliases, servers, claude/opencode models).

**Layer D: User config (explicit keys only)**

- ~/.orch/config.yaml keys explicitly set (backend, default_model, default_tier, models aliases).

**Layer E: Heuristics (advisory unless no explicit inputs)**

- Infrastructure detection (escape hatch), task scope -> tier inference, skill defaults.

**Layer F: Hardcoded defaults**

- Backend: opencode
- Model: sonnet
- Mode: tdd
- Validation: tests
- Tier: skill defaults

### 3) Per-Setting Precedence (normalized)

**Backend (claude/opencode)**

1. --backend
2. --model implies backend requirement (if model requires a backend, treat as explicit)
3. Project config spawn_mode (explicit key only)
4. User config backend (explicit key only)
5. Infra escape hatch (only if no explicit backend/model/config)
6. Default: opencode

**Model**

1. --model
2. Project config model (explicit key, per-backend)
3. User config default_model (explicit key)
4. Default: model.DefaultModel (sonnet)

**Tier (light/full)**

1. --light / --full
2. User config default_tier (explicit key)
3. Task scope inference (session scope or keywords)
4. Skill default

**Spawn mode (headless/tmux/inline)**

1. --inline / --tmux / --headless
2. Orchestrator skill default (tmux)
3. Default: headless

**MCP server**

1. --mcp
2. needs:\* label (explicit)
3. Default: empty

**Implementation mode (tdd/direct/verification-first)**

1. --mode
2. Default: tdd

### 4) Model/Backend Compatibility Rules

- If --model requires a specific backend and --backend conflicts, error (not warn).
- If --model implies backend and --backend not provided, select the required backend.
- Infra detection never overrides explicit backend/model. It only warns.

## Required Structural Changes (Design)

### A) Config Load with Explicitness Metadata

Introduce a load path that returns "value + explicitness":

- Project config: track whether spawn_mode is present in YAML, not just defaulted.
- User config: track whether backend/default_model/default_tier are set in YAML.

Implementation sketch:

- Parse YAML into both struct + map[string]any to detect key presence.
- Add helper: config.LoadWithMeta(projectDir) -> (cfg, meta)
- Add helper: userconfig.LoadWithMeta() -> (cfg, meta)

### B) Centralized Resolver

Create a single resolver (pkg/orch or pkg/spawn) that returns a `ResolvedSpawnSettings` object:

- Fields: Backend, Model, Tier, SpawnMode, MCP, Mode, Validation
- Metadata: Source per field (flag, issue, project-config, user-config, heuristic, default)
- Diagnostics: warnings/errors for conflicts

### C) Stop Treating Defaults as Explicit

- Avoid ApplyDefaults for precedence decisions.
- Only use defaults after explicit-resolution pass completes.
- Explicitness lives in meta object, not in defaulted config values.

## Bug Classes Addressed (10)

1. Project config exists for servers -> spawn_mode defaults to opencode and overrides user backend.
2. User config default_model is loaded into spawnModel and treated as CLI explicit, altering backend selection flow.
3. Explicit --model does not force backend; can silently use opencode even when model requires claude.
4. Project config per-backend model (OpenCode.Model/Claude.Model) is ignored, leading to unexpected model selection.
5. Project config default OpenCode.Model = flash, which is invalid, but defaults can be interpreted as explicit.
6. Infrastructure escape hatch can override backend when explicit intent is not clearly distinguished from default.
7. User config Backend defaults to opencode even when unset; ambiguity about whether backend is user intent.
8. Tier defaults can be overridden by user config default_tier even when user intended per-skill defaults.
9. needs:\* labels and --mcp are handled in multiple places (daemon vs spawn path), risking divergence.
10. Different code paths (orch spawn vs orch work) set model/backend differently due to extra default_model injection.

## Migration Plan (Incremental)

1. Add config/userconfig LoadWithMeta helpers (no behavior change).
2. Implement ResolvedSpawnSettings and update spawn pipeline to use it.
3. Update DetermineSpawnBackend/ResolveAndValidateModel to use explicitness-aware inputs.
4. Add diagnostics: log resolved settings + source in SPAWN_CONTEXT.md.
5. Add tests for precedence table and explicitness handling.

## Open Questions

- Should project config provide a per-skill default model, or stay global?
- Should infra escape hatch ever auto-apply if user config explicitly sets backend?

## Verification (planned)

- Unit tests for precedence ordering and explicitness metadata.
- Regression tests for the 10 bug classes.
