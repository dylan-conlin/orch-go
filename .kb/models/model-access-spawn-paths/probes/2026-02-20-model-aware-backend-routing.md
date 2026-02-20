# Probe: Model-Aware Backend Routing Implementation

**Model:** model-access-spawn-paths
**Date:** 2026-02-20
**Status:** Complete
**Decision:** kb-2d62ef

## Question

Can the existing symmetric backend auto-resolve (BugClass14) be generalized into primary routing logic where model provider determines backend, making `--backend` an override rather than a required flag?

## What I Tested

### Test 1: Resolve pipeline tracing for daemon path
Traced `orch work <beadsID>` → `runWork()` → `runSpawnWithSkillInternal()`:
- `runWork` passes `headless=true` to `runSpawnWithSkillInternal` (line spawn_cmd.go:444)
- This sets `CLI.Headless = true` in resolve input (line spawn_cmd.go:544)
- `resolveSpawnMode` returns `{SpawnModeHeadless, SourceCLI}`
- Backend resolves to claude (default is now claude since kb-2d62ef)
- But claude-backend-implies-tmux (resolve.go:199) only triggers when `SpawnMode.Source == SourceDefault`
- Result: SpawnMode stays headless → `DispatchSpawn` routes to `runSpawnHeadless` → fails for Anthropic models

### Test 2: Symmetric auto-resolve scope
The BugClass14 fix (resolve.go:165-170) has condition `result.Backend.Source == SourceProjectConfig`:
- Only auto-switches backend opencode→claude when backend came from project config
- Does NOT apply when backend came from user config, heuristic, or default
- This means user config `backend: opencode` + anthropic model → hard error instead of auto-switch

### Test 3: modelBackendRequirement already exists as provider→backend map
`modelBackendRequirement()` (resolve.go:385-397) correctly maps:
- Anthropic → claude
- OpenAI/Google/DeepSeek → opencode
But it's only called in `resolveBackend()` when `modelSet=true` (CLI model flag set).
When model comes from config/defaults, this mapping is NOT applied.

## What I Observed

Two bugs preventing model-aware routing:

1. **Symmetric auto-resolve too narrow**: Only applies to project-config backends, not user-config/default/heuristic backends
2. **Claude-backend-implies-tmux too narrow**: Only applies when spawn mode was default, not when daemon sets headless=true

These combine to break the daemon path: default backend resolves to claude (correct), but headless spawn mode persists (wrong), causing silent failure.

## Implementation

### Change 1: Generalize symmetric auto-resolve (resolve.go:159-170)
- Remove `result.Backend.Source == SourceProjectConfig` constraint
- Replace with `result.Backend.Source != SourceCLI`
- Model provider determines backend for all non-CLI backend sources
- CLI `--backend` remains the override

### Change 2: Claude backend requires tmux (resolve.go:199)
- Change from `result.SpawnMode.Source == SourceDefault` to checking value
- When backend is claude and spawn mode is headless, override to tmux
- Claude backend physically requires tmux (SpawnClaude creates tmux window + claude CLI)
- Keep explicit CLI spawn modes (inline, tmux) untouched

### Change 3: Update test expectations
- `BugClass13b` test: `--backend claude --headless` should resolve to tmux (not headless)
  because claude backend cannot work with headless mode

## Model Impact

**Extends** invariant 3 ("Anthropic models blocked on OpenCode by default"):
- Now enforced as primary routing logic, not just a blocking rule
- Model provider auto-determines backend when `--backend` not specified
- The dual spawn architecture becomes implicit rather than requiring manual flag selection

**Extends** invariant 1 ("Never spawn OpenCode infrastructure work without --backend claude --tmux"):
- Infrastructure detection is still advisory at priority 5
- But model-aware routing now handles the common case: Anthropic model → claude backend → tmux implied

**Confirms** the daemon autonomous operation model claim that daemon uses `orch work` subprocess:
- No daemon code changes needed - resolve pipeline fixes propagate through `orch work`

## Test Results

```
go test ./pkg/spawn/ -run TestResolve -count=1 → 30 passed, 0 failed (0.008s)
go test ./pkg/daemon/ -count=1 → ok (6.429s)
go test ./pkg/model/ -count=1 → ok (0.005s)
go test ./pkg/orch/ -count=1 → ok (0.010s)
go build ./cmd/orch/ && go vet ./cmd/orch/ → clean
```

### New tests added (BugClass15):
- `user_config_opencode_backend_auto-routes_to_claude_for_anthropic_model`
- `user_config_default_model_codex_auto-routes_to_opencode`
- `daemon_path_headless_with_claude_backend_resolves_to_tmux`
- `project_config_claude_backend_with_non-anthropic_model_auto-routes_backend`

### Existing tests updated:
- `BugClass02`: Now expects backend auto-route (was: model override)
- `BugClass13`: Detail string "implies" → "requires"
- `BugClass13b`: Claude overrides headless to tmux (was: headless wins)
- `BugClass14`: Warning message updated for generalized routing
