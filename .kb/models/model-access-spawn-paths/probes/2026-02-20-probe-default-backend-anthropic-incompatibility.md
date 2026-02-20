# Probe: Default Backend + Anthropic Model Incompatibility

**Model:** model-access-spawn-paths
**Date:** 2026-02-20
**Status:** Complete

---

## Question

Testing if the resolve_test.go failures (4 pre-existing) are caused by the incompatibility between:
1. Default backend = opencode
2. Default model = Anthropic sonnet
3. Anthropic models blocked on OpenCode backend (without allow_anthropic_opencode override)

---

## What I Tested

```bash
# 1. Ran the failing tests to see error messages
cd /Users/dylanconlin/Documents/personal/orch-go && go test -v ./pkg/spawn/... -run TestResolve 2>&1 | head -60

# Output showed 6 failures, all with same error:
# resolve_test.go:102: Resolve() error = backend opencode does not support provider anthropic (set allow_anthropic_opencode: true to override)

# 2. Examined resolve.go to trace the resolution path
# Lines 255-259: Default backend was changed to claude
# Lines 385-396: modelBackendRequirement now returns BackendClaude for Anthropic

# 3. Examined resolve_test.go to see test expectations
# Tests expected default backend = opencode (old behavior)
# Tests didn't account for opencode + anthropic incompatibility

# 4. Ran all tests after code was already fixed
cd /Users/dylanconlin/Documents/personal/orch-go && go test -v ./pkg/spawn/... -run TestResolve 2>&1

# All 27 tests pass (including 4 that were failing)
```

---

## What I Observed

The fix was already applied in resolve.go and resolve_test.go before I started:

**resolve.go changes:**
1. Default backend changed from `opencode` to `claude` (lines 255-259)
2. `modelBackendRequirement()` now returns `BackendClaude` for Anthropic models (lines 393-395)

**resolve_test.go changes:**
1. `TestResolve_PrecedenceLayers/default_backend`: Expects `BackendClaude` now (lines 97-112)
2. `TestResolve_BugClass05_ProjectDefaultFlashNotExplicit`: Added user config with gpt-4o as fallback (lines 240-262)
3. `TestResolve_BugClass06_InfraEscapeHatchDoesNotOverrideExplicitBackend`: Added gpt-4o user config (lines 264-283)
4. `TestResolve_BugClass11_ProjectConfigModelOverridesUserDefaultModel`: Added explicit opencode backend (lines 366-388)
5. `TestResolve_BugClass11c_UserDefaultModelFallbackWhenNoProjectConfig`: Added explicit opencode backend (lines 536-555)

**Root cause:** Anthropic banned subscription OAuth in third-party tools (Feb 19 2026), making OpenCode + Anthropic models a dead path without explicit override. Tests assumed old behavior where opencode was default backend and sonnet was usable on it.

---

## Model Impact

- [x] **Confirms** invariant: Anthropic models are blocked on OpenCode backend by default
- [x] **Extends** model with: Default backend is now claude (not opencode) to match default model's provider requirement. This aligns with the kb-2d62ef decision documented in the code.

---

## Notes

- The fix was already in place when I started investigation
- All 27 resolve tests now pass
- The code change creates a coherent default: claude backend + sonnet model (both Anthropic)
- Tests that need to verify opencode behavior now explicitly set `input.CLI.Backend = BackendOpenCode` and use non-Anthropic models
