# Config-to-Spawn Override Audit

Date: 2026-02-18

## Scope
- Trace user config (`~/.orch/config.yaml`) and project config (`.orch/config.yaml`) through model/backend selection to spawn dispatch.
- Identify every override point where user backend/default_model can be ignored or changed.

## Findings

### BROKEN
- None found that override explicit user config (`backend: opencode`, `default_model: codex`) in the current priority chain.

### RISKY
- Project config defaults override user backend even when `spawn_mode` is unset.
  - Evidence: `config.ApplyDefaults()` forces `SpawnMode = "opencode"` when empty (`pkg/config/config.go:90-94`). `DetermineSpawnBackend` treats any project config `SpawnMode` as explicit and prioritizes it above user config (`pkg/orch/extraction.go:733-736`).
  - Impact: If a project has `.orch/config.yaml` for unrelated settings and the user sets `backend: claude` in `~/.orch/config.yaml`, the project default silently overrides to `opencode`.
  - Proposed fix: Track whether `spawn_mode` was explicitly set in the project config (e.g., add `SpawnModeSet bool` during unmarshal), and only prefer project config when explicitly set.

- Malformed user config silently disables default_model and backend preferences.
  - Evidence: `userconfig.Load()` returns error on YAML parse failures (`pkg/userconfig/userconfig.go:138-152`), but `ResolveAndValidateModel` and `DetermineSpawnBackend` ignore load errors (`pkg/orch/extraction.go:537-548`, `pkg/orch/extraction.go:692-695`).
  - Impact: A malformed `~/.orch/config.yaml` makes spawns fall back to hardcoded defaults without warning.
  - Proposed fix: Surface a warning when `userconfig.Load()` fails in spawn pipeline, and fall back explicitly with a logged message (or block spawn on parse errors if config exists).

- User `default_model` does not count as an explicit model for backend selection.
  - Evidence: `ResolveAndValidateModel` uses `cfg.DefaultModel` when `spawnModel` is empty (`pkg/orch/extraction.go:545-548`), but `DetermineSpawnBackend` checks `explicitModel := spawnModel != ""` (`pkg/orch/extraction.go:703`). Manual spawns pass the raw `spawnModel` (often empty) into `DetermineSpawnBackend` (`cmd/orch/spawn_cmd.go:425-444`).
  - Impact: If the task is classified as infrastructure work and the user only set `default_model` (no `backend`), infra detection can force `claude`, which ignores `default_model` and may conflict with codex requirements.
  - Proposed fix: Pass the effective model spec into `DetermineSpawnBackend`, or set `spawnModel` from user config in manual spawns (mirroring `runWork`).

- Infrastructure detection can force `claude` when no explicit backend/model/config is present.
  - Evidence: `DetermineSpawnBackend` forces `claude` when no explicit flags or config and `isInfrastructureWork` is true (`pkg/orch/extraction.go:747-753`), using a broad keyword list (`pkg/orch/extraction.go:1746-1772`).
  - Impact: Tasks with keywords like `dashboard`, `skillc`, or `spawn template` may flip backend unexpectedly when user config is missing or malformed.
  - Proposed fix: Keep as-is but narrow keywords or require explicit opt-in flag for infra detection; at minimum, log when infra detection overrides a default.

### OK
- User `backend` is respected when explicitly set in `~/.orch/config.yaml`.
  - Evidence: `DetermineSpawnBackend` treats user config backend as explicit (`userCfgExplicit`) and prefers it over infra detection (`pkg/orch/extraction.go:692-746`).

- User `default_model` is applied for model resolution (spawns without `--model`).
  - Evidence: `ResolveAndValidateModel` uses `cfg.DefaultModel` when `modelFlag` is empty (`pkg/orch/extraction.go:545-548`).

- Daemon spawns (`orch work`) explicitly load `default_model` into `spawnModel` before spawning.
  - Evidence: `runWork` sets `spawnModel` from user config (`cmd/orch/spawn_cmd.go:331-338`). Daemon calls `orch work` (`pkg/daemon/issue_adapter.go:348-355`).

- Codex aliases exist in model resolution.
  - Evidence: `Aliases` includes `codex`, `codex-mini`, etc. (`pkg/model/model.go:60-65`).

- Spawn templates do not override backend/model.
  - Evidence: `SpawnContextTemplate` is informational only; backend selection occurs before context write (`pkg/spawn/context.go:53-170`, `pkg/orch/extraction.go:684-776`).

- Account switching logic does not alter backend selection.
  - Evidence: `CheckAndAutoSwitchAccount` only touches account credentials and does not update backend (`pkg/orch/extraction.go:211-277`). It is not invoked in the spawn pipeline.
