# Synthesis: Daemon Sets Beads but Never Spawns

## Issue
orch-go-1145

## Root Cause
The daemon's spawn pipeline hard-failed when `validateModel()` rejected the project config model (`flash`), instead of falling through to the next precedence level (user config `opus`). This caused 857 consecutive silent spawn failures.

**Configuration causing the bug:**
- Project config (`.orch/config.yaml`): `spawn_mode: opencode`, `opencode.model: flash`
- User config (`~/.orch/config.yaml`): `backend: claude`, `default_model: opus`

**Failure chain:**
1. Daemon polls `bd ready`, finds spawnable issue
2. `Resolve()` picks opencode backend from project config
3. `resolveModel()` picks flash from project config's `opencode.model`
4. `validateModel()` rejects flash (blocked for agent work) → **hard error**
5. Spawn fails, daemon rolls back beads status to open
6. Repeat 857 times

## Fix

### 1. Soft validation fallthrough in `resolveModel()` (pkg/spawn/resolve.go)
Changed `resolveModel()` so that when a project-config or user-config model fails `validateModel()`, it falls through to the next precedence level instead of returning a hard error. Only the final default model level returns an error on validation failure.

### 2. Symmetric backend auto-resolve (pkg/spawn/resolve.go)
Added targeted auto-resolve in `Resolve()`: when backend is opencode (from project config) and the resolved model is anthropic (from a lower precedence level like user config), switch backend to claude. This handles the cascade where project config's opencode model was rejected and the fallback model is anthropic.

Condition is narrowly scoped: only fires when `Backend.Source == SourceProjectConfig && Model.Source != SourceProjectConfig`.

## Tests Added
- `TestResolve_BugClass14_FlashProjectConfigFallsThrough` - Exact daemon config: project opencode+flash, user claude+opus → flash falls through to opus, backend auto-switches to claude
- `TestResolve_BugClass14b_FlashProjectConfigNoUserFallback` - Flash rejected, no user config → falls to DefaultModel (sonnet), backend auto-switches to claude
- `TestResolve_BugClass14c_FlashCLIModelStillErrors` - Explicit `--model flash` still hard-errors (user explicitly chose flash)

## Verification
- All 3 new BugClass14 tests pass
- `go build` and `go vet` pass clean
- `orch daemon preview` now shows spawnable issue (was silently failing before)

## Files Changed
- `pkg/spawn/resolve.go` - resolveModel fallthrough + symmetric auto-resolve
- `pkg/spawn/resolve_test.go` - 3 new BugClass14 tests

## Discovered Work
- 4 pre-existing test failures in resolve_test.go (default opencode + anthropic sonnet incompatibility): TestResolve_PrecedenceLayers/default_backend, beads_label_mcp, BugClass05, BugClass06. These are NOT caused by this change (confirmed via git stash). Separate issue needed.
