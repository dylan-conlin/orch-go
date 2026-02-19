# Probe: Config-to-Spawn Override Audit

**Model:** daemon-autonomous-operation
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Does the daemon spawn path honor user config (backend/default_model), or are there override points that can silently change backend/model selection?

---

## What I Tested

```bash
rg -n "DetermineSpawnBackend|ResolveAndValidateModel|default_model|Backend" pkg/orch/extraction.go cmd/orch/spawn_cmd.go pkg/userconfig/userconfig.go pkg/config/config.go pkg/model/model.go
```

---

## What I Observed

- User config loads from `~/.orch/config.yaml` and defaults Backend to `opencode` when missing; errors return nil config (`pkg/userconfig/userconfig.go:138-166`).
- Model resolution uses `userconfig.Load()` for aliases/default_model, but ignores load errors (`pkg/orch/extraction.go:537-553`).
- Backend selection in `DetermineSpawnBackend` uses project config first, then user config, then infra detection/default (`pkg/orch/extraction.go:684-776`).
- Project config defaults `SpawnMode` to `opencode`, so any existing `.orch/config.yaml` will override user config backend even when `spawn_mode` is unset (`pkg/config/config.go:90-107`).
- Daemon spawn path shells out to `orch work`, which loads `default_model` into `spawnModel` when empty (`cmd/orch/spawn_cmd.go:331-338`), then uses the same backend determination path (`cmd/orch/spawn_cmd.go:425-444`).

---

## Model Impact

- [x] **Extends** model with: spawn path silently ignores malformed user config or implicit project defaults, which can override user backend/model choices without warning.

---

## Notes

[Any additional context, caveats, or follow-up questions]
