# Probe: Orchestration Cost Economics Model Drift Audit

**Model:** orchestration-cost-economics
**Date:** 2026-02-20
**Status:** Complete

---

## Question

The model references 3 deleted artifacts (`pkg/spawn/backend.go`, `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md`, `~/.anthropic/`) and has stale spawn path economics. Has the cost/path model shifted, and do the model's claims still hold against current code?

---

## What I Tested

### Test 1: Verify deleted file references

```bash
ls pkg/spawn/backend.go  # FILE MISSING
ls .kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md  # FILE MISSING
ls ~/.anthropic/  # DIRECTORY MISSING
ls ~/.local/share/opencode/auth.json  # EXISTS (2.4 KB)
```

### Test 2: Current spawn architecture files

```bash
ls pkg/spawn/*.go  # 37 files — no backend.go
# Key files: resolve.go, claude.go, opencode_mcp.go, atomic.go, config.go
```

Examined `pkg/spawn/resolve.go` — the dual backend is now `BackendClaude = "claude"` and `BackendOpenCode = "opencode"` constants (lines 26-27), resolved via centralized `Resolve()` function with 7-level precedence: CLI > beads labels > project config > user config > heuristics > defaults.

### Test 3: Default model in code

```go
// pkg/model/model.go:19-22
var DefaultModel = ModelSpec{
    Provider: "anthropic",
    ModelID:  "claude-sonnet-4-5-20250929",
}
```

### Test 4: Flash model ban

```go
// pkg/spawn/resolve.go:392-396
func validateModel(resolvedModel model.ModelSpec) error {
    if resolvedModel.Provider == "google" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "flash") {
        return fmt.Errorf("flash models are not supported for agent work")
    }
    return nil
}
```

### Test 5: Model alias ecosystem expansion

```go
// pkg/model/model.go — Aliases map now includes:
// - Anthropic: opus, sonnet, haiku (unchanged)
// - Google: flash, pro (flash banned at validation)
// - OpenAI: gpt, gpt4o, gpt-4o, gpt-5, gpt5-latest, gpt-5-mini, o3, o3-mini
// - Codex: codex, codex-mini, codex-max, codex-latest, codex-5.1, codex-5.2
// - DeepSeek: deepseek, deepseek-chat, deepseek-r1, reasoning
```

### Test 6: Backend compatibility enforcement

```go
// pkg/spawn/resolve.go:372-383
// OpenCode backend + Anthropic model → ERROR (unless allow_anthropic_opencode: true)
// Claude backend + non-Anthropic model → ERROR
// OpenCode backend + Opus → WARNING
```

### Test 7: Infrastructure escape hatch

```go
// pkg/spawn/resolve.go:237-238
if input.InfrastructureDetected {
    return ResolvedSetting{Value: BackendClaude, Source: SourceHeuristic, Detail: "infra-escape-hatch"}, warnings, nil
}
```

### Test 8: Build verification

```bash
go build ./cmd/orch/  # Success (no errors)
```

---

## What I Observed

### Deleted References (3 stale artifacts)

| Model Reference | Current Status | Impact |
|----------------|----------------|--------|
| `pkg/spawn/backend.go` (line 366) | **DELETED** — replaced by `resolve.go` + `claude.go` + `opencode_mcp.go` | HARMFUL: model directs readers to nonexistent file |
| `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` (line 260, 365) | **DELETED** — decision absorbed into code patterns | MISLEADING: reference exists but file doesn't |
| `~/.anthropic/` (line 363) | **DELETED** — auth is now at `~/.local/share/opencode/auth.json` | HARMFUL: model claims wrong auth location |

### Stale Claims in Spawn Path Economics (Section: "Spawn Path Economics")

1. **"Primary path uses Sonnet, DeepSeek, Gemini via OpenCode API"** — STALE. Flash is explicitly banned. Default is now Sonnet via Claude backend (Max subscription), not OpenCode API. The primary economic path has inverted.

2. **"Economic Decision Tree" asks "Is cost the primary constraint? → YES: Use DeepSeek V3"** — PARTIALLY STALE. Still technically valid but the decision tree doesn't reflect that Anthropic models on OpenCode backend are now blocked by default (`allow_anthropic_opencode: true` required to override).

3. **Model pricing table lists Gemini Flash as active option** — CONTRADICTED by code. Flash is banned for agent work since the `validateModel()` gate.

### New Architecture Not Covered

1. **Centralized `ResolvedSpawnSettings` system** with provenance tracking (source attribution for every setting). This is the biggest architectural change — the model has no awareness of it.

2. **`allow_anthropic_opencode` override flag** — a user config option that relaxes the Anthropic-on-OpenCode block. Not mentioned in model.

3. **OpenAI/Codex as first-class provider** — the model treats OpenAI as a theoretical alternative ("Codex CLI available"). In code, OpenAI has 12 model aliases and is a supported `opencode` backend provider with `modelBackendRequirement()` routing.

4. **Project-level and user-level model config** — Models can be configured per-project (`.orch/config.yaml`) and per-user (`~/.orch/user.yaml`), with config aliases taking precedence over built-in aliases. Not in model.

5. **Claude backend auto-implies tmux** (`resolve.go:185-187`) — when `backend=claude` and no explicit spawn mode, tmux is derived automatically. Model mentions this behavior but attributes it to the deleted `backend.go`.

### Claims That Still Hold

1. **Infrastructure escape hatch pattern** — CONFIRMED. `resolveBackend()` auto-selects `BackendClaude` when `InfrastructureDetected` is true. Exactly as model describes.

2. **Opus requires Claude backend** — CONFIRMED. `warnOnNonOptimalCombo()` warns on opus+opencode, and `validateModelCompatibility()` blocks Anthropic-on-OpenCode by default.

3. **Claude Max $200/mo flat rate advantage** — the economic argument still holds, but the model doesn't reflect that this is now the *default* path (Sonnet via Max), not just the escape hatch.

4. **Fingerprinting blocks third-party tools** — CONFIRMED by the `validateModelCompatibility()` gate which blocks Anthropic models on OpenCode backend.

5. **DeepSeek V3 as viable option** — still in aliases as `deepseek-chat`, routed to OpenCode backend.

---

## Model Impact

- [x] **Contradicts** claims: 3 deleted file references, Flash listed as active option, primary path described as "OpenCode API" when it's now "Claude backend + Max subscription"
- [x] **Extends** model with: OpenAI/Codex as first-class provider (12 aliases), centralized `ResolvedSpawnSettings` with provenance, `allow_anthropic_opencode` override, project/user model config, Flash ban gate

**Verdict: CONTRADICTS + EXTENDS**

The model's core economic insight (Max subscription beats API pricing) is more true than ever — it's now the *default* path, not just an optimization. But the model's description of *how* the system implements this has drifted substantially. The dual spawn architecture has been refactored from a monolithic `backend.go` into a multi-file resolver system (`resolve.go`, `claude.go`, `opencode_mcp.go`), and the provider ecosystem expanded from 3 (Anthropic/Google/DeepSeek) to 4+ (+ OpenAI/Codex).

### Recommended Model Updates

| Section | Action | Priority |
|---------|--------|----------|
| "Spawn Path Economics" table | Update primary path to "Claude backend + Max subscription (default)" | HIGH |
| "Primary Evidence" references | Remove `pkg/spawn/backend.go`, `~/.anthropic/`, stale decision path | HIGH |
| "Model Pricing Comparison" | Mark Flash as "BANNED for agent work" or remove row | HIGH |
| "Alternatives Evaluated" | Promote OpenAI/Codex from "backup" to "first-class provider" | MEDIUM |
| New section: "Config Resolution" | Document `ResolvedSpawnSettings` provenance system | MEDIUM |
| "Economic Decision Tree" | Add `allow_anthropic_opencode` branch, update default path | MEDIUM |
| "Cost Visibility Gap" | Check if still unimplemented or if any tracking landed | LOW |

---

## Notes

- The model was last updated 2026-01-28, and substantial refactoring happened in Feb 2026 (the `ResolvedSpawnSettings` system, Flash ban, OpenAI expansion).
- The `she-llac.com` credit formula section may also be stale but couldn't verify (external dependency).
- The `allow_anthropic_opencode` flag suggests the hard Anthropic-on-OpenCode block has a documented escape hatch, which softens constraint C1 slightly.
