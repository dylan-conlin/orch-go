# Probe: Backend Resolution Architecture Drift (67 stale spawns, 13 commits)

**Model:** model-access-spawn-paths
**Date:** 2026-02-20
**Status:** Complete

---

## Question

The model-access-spawn-paths model was last updated 2026-01-12. Its staleness warning flagged `pkg/spawn/config.go` as deleted and `~/.claude/skills/meta/orchestrator/SKILL.md` as deleted. Do the model's claims about backend selection priority, infrastructure detection, file locations, and critical invariants still hold?

---

## What I Tested

### 1. File existence — falsified staleness claims

```bash
ls -la pkg/spawn/config.go
# EXISTS — 457 lines, last modified recently
# Staleness detector incorrectly flagged as deleted

ls -la ~/.claude/skills/meta/orchestrator/SKILL.md
# EXISTS — 35 KB, recompiled 2026-02-18 14:13:33
# Staleness detector incorrectly flagged as deleted
```

Both files exist. The staleness detector's "deleted files" claim was wrong.

### 2. Backend selection priority — location and structure changed

```bash
rg 'selectBackend' --type go
# NOT FOUND in any Go file in the project
```

Model claims `pkg/spawn/config.go:selectBackend()`. This function does not exist. Backend selection is now in `pkg/spawn/resolve.go:resolveBackend()`.

**Model's claimed priority (4 levels):**
```
1. Explicit --backend flag
2. Auto-apply for infrastructure work (keywords detected)
3. Model-based auto-selection
4. Default: opencode
```

**Actual priority (6 levels, from resolve.go:resolveBackend):**
```
1. CLI --backend flag
2. Model-derived requirement (openai/google/deepseek → opencode)
3. Project config spawn_mode
4. User config backend
5. Infrastructure heuristic → claude (advisory when overridden)
6. Default: opencode
```

Infrastructure detection dropped from priority 2 to priority 5 and became **advisory** when higher-priority settings are present.

### 3. Infrastructure detection — location changed

```bash
rg 'detectInfrastructureWork' --type go
# NOT FOUND in any Go file

rg 'func isInfrastructureWork' --type go
# Found: pkg/orch/extraction.go:1842
```

Model claims `pkg/spawn/config.go:detectInfrastructureWork()`. This function is now `isInfrastructureWork()` in `pkg/orch/extraction.go:1842`. The keywords list has expanded significantly:

**Model's keywords:** "opencode", "spawn", "daemon", "registry", "orch serve", "overmind", "dashboard"

**Actual keywords (from extraction.go:1844-1869):** "opencode", "orch-go", "pkg/spawn", "pkg/opencode", "pkg/verify", "pkg/state", "cmd/orch", "spawn_cmd.go", "serve.go", "status.go", "main.go", "dashboard", "agent-card", "agents.ts", "daemon.ts", "skillc", "skill.yaml", "SPAWN_CONTEXT", "spawn system", "spawn logic", "spawn template", "orchestration infrastructure", "orchestration system"

8 keywords grew to 22, and several model-claimed keywords ("spawn", "daemon", "registry") are not in the current list (replaced by more specific patterns).

### 4. Critical Invariant: Opus only via Claude CLI backend

```bash
go test ./pkg/spawn/ -run TestResolve_AnthropicModelBlockedOnOpenCodeByDefault -v
# PASS — Anthropic models on opencode return error

go test ./pkg/spawn/ -run TestResolve_AnthropicModelAllowedWithUserConfigOverride -v
# PASS — Can be overridden with allow_anthropic_opencode: true
```

Model claims "Opus only accessible via Claude CLI backend." This is **partially correct** — Anthropic models are blocked on opencode *by default*, but a new `allow_anthropic_opencode: true` user config override exists. The model doesn't know about this escape valve.

### 5. Critical Invariant: --backend claude implies tmux

```bash
go test ./pkg/spawn/ -run TestResolve_BugClass13 -v
# PASS: ClaudeBackendImpliesTmuxSpawnMode
# PASS: ExplicitHeadlessOverridesClaudeBackend
# PASS: ExplicitTmuxWithClaudeBackendStaysExplicit
# PASS: InfraEscapeHatchAlsoImpliesTmux
```

New behavior not in model: `--backend claude` now automatically implies `--tmux` spawn mode (commit `0b7192aef`). This is a derived setting unless explicitly overridden by `--headless`.

### 6. Flash models now blocked

```go
// From pkg/spawn/resolve.go:392-396
func validateModel(resolvedModel model.ModelSpec) error {
    if resolvedModel.Provider == "google" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "flash") {
        return fmt.Errorf("flash models are not supported for agent work")
    }
    return nil
}
```

Model discusses Flash TPM limits as a constraint requiring workarounds (Tier 3 application, etc.). Flash is now **completely blocked** at the resolve layer — no workaround available or needed.

### 7. Default model

```go
// From pkg/model/model.go:19-22
var DefaultModel = ModelSpec{
    Provider: "anthropic",
    ModelID:  "claude-sonnet-4-5-20250929",
}
```

Model claims default was "claude-sonnet-4-5 (default since Jan 9, was gemini-3-flash before TPM limits)." The Sonnet default is confirmed. Flash is no longer even an option.

---

## What I Observed

| Model Claim | Current Reality | Verdict |
|---|---|---|
| `pkg/spawn/config.go` deleted | EXISTS (457 lines) | False alarm from staleness detector |
| `~/.claude/skills/meta/orchestrator/SKILL.md` deleted | EXISTS (35KB, recompiled Feb 18) | False alarm from staleness detector |
| `config.go:selectBackend()` | Does not exist; replaced by `resolve.go:resolveBackend()` | Contradicts |
| `config.go:detectInfrastructureWork()` | Does not exist; now `extraction.go:isInfrastructureWork()` | Contradicts |
| 4-level backend priority | 6-level priority with infra detection at level 5 (advisory) | Contradicts |
| Infrastructure keywords (8) | Expanded to 22, different specific keywords | Extends |
| Opus only via Claude CLI | Mostly true, but `allow_anthropic_opencode` override exists | Extends |
| Escape hatch provides true independence | Still true | Confirms |
| Flash has TPM limits (workaround: Tier 3) | Flash now completely blocked at resolve layer | Contradicts |
| Cost tracking gap | Not verified in this probe (out of scope) | — |
| Default: Sonnet | Confirmed: `anthropic/claude-sonnet-4-5-20250929` | Confirms |
| Infrastructure work kills itself failure mode | Still valid and addressed by auto-detection | Confirms |
| Zombie agent failure mode | Still valid pattern | Confirms |

---

## Model Impact

- [x] **Contradicts** claims:
  - Backend selection function location and priority chain completely restructured
  - Infrastructure detection moved to different package with expanded keyword set and now advisory
  - Flash models are blocked, not just TPM-limited
  - File deletion claims from staleness detector were false

- [x] **Extends** model with:
  - `pkg/spawn/resolve.go` — new centralized resolver with 6-level precedence and provenance tracking
  - `allow_anthropic_opencode: true` user config — escape valve for Anthropic on OpenCode
  - `--backend claude` → automatic tmux implication
  - Project config and user config as new precedence layers in backend selection
  - ResolvedSpawnSettings with per-setting source tracking (SettingSource enum)

- [x] **Confirms** invariants:
  - Escape hatch independence (Claude CLI ≠ OpenCode server)
  - Infrastructure work auto-detection concept (though location and implementation changed)
  - Opus access requiring Claude CLI backend (with new override escape valve)
  - Default model is Sonnet
  - Zombie agent and infrastructure self-kill failure modes

---

## Recommended Model Updates

1. **Primary Evidence section** — update all file references:
   - `pkg/spawn/config.go:selectBackend()` → `pkg/spawn/resolve.go:resolveBackend()`
   - `pkg/spawn/config.go:detectInfrastructureWork()` → `pkg/orch/extraction.go:isInfrastructureWork()`
   - Add `pkg/spawn/resolve.go:Resolve()` as central entry point

2. **Backend Selection Priority** — rewrite to 6-level chain with advisory infrastructure detection

3. **Constraints section** — update:
   - Constraint 5 (Flash TPM) → Flash is now blocked entirely
   - Add Constraint: Anthropic models blocked on OpenCode by default (with override)
   - Add: `--backend claude` implies tmux

4. **Infrastructure keywords** — update to current 22-keyword set

5. **Constraint 6 (Community workarounds)** — may be outdated (Jan 2026 timeframe)

---

## Notes

The model's conceptual framework (dual spawn architecture, escape hatch pattern, infrastructure auto-detection) remains sound. The staleness is primarily in implementation details: function names, file locations, precedence chain structure, and the Flash blocking change. The biggest conceptual shift is infrastructure detection becoming **advisory** rather than overriding — explicit user settings now always win.
