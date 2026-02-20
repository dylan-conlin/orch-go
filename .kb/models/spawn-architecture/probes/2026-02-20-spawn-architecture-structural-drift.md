# Probe: Spawn Architecture Structural Drift (46 commits since update)

**Model:** spawn-architecture
**Date:** 2026-02-20
**Status:** Complete

---

## Question

The spawn-architecture model was last updated 2026-01-12 (46 commits ago). Do its structural claims about file locations, component responsibilities, workspace metadata, and spawn lifecycle still hold?

---

## What I Tested

### 1. File existence and line counts

```bash
wc -l cmd/orch/spawn_cmd.go  # Model claims ~800 lines
# Result: 802 lines ✓

wc -l pkg/spawn/context.go  # Model claims ~400 lines
# Result: 1315 lines ✗ (3x growth)

ls pkg/spawn/config.go  # Model claims "SpawnConfig struct and validation"
# Result: Exists, but struct is named Config (not SpawnConfig), no validation logic
```

### 2. Backend selection and resolve architecture

```bash
rg 'selectBackend|detectInfrastructureWork' --type go
# selectBackend: NOT FOUND in pkg/spawn/
# detectInfrastructureWork: NOT FOUND in pkg/spawn/
# These functions were removed/refactored
```

Backend selection now lives in `pkg/spawn/resolve.go:resolveBackend()` with a 6-level precedence chain (not 4-level as the model's related model-access-spawn-paths implied).

### 3. Workspace metadata architecture

```bash
rg 'AgentManifest|AGENT_MANIFEST' pkg/spawn/ --files-with-matches
# Found in: context.go, session.go, atomic.go, atomic_test.go, rework.go, session_test.go, meta_orchestrator_context.go
```

Model claims workspace metadata is 5 dotfiles. Reality: AGENT_MANIFEST.json is now the canonical source, with OpenCode session metadata as primary and dotfiles as legacy fallback. Read path: `ReadAgentManifestWithFallback()` → OpenCode metadata → AGENT_MANIFEST.json → dotfiles.

### 4. Registry removal

```bash
rg 'registry' cmd/orch/spawn_cmd.go -i
# No matches

git log --oneline -- | grep registry
# a9ec5cbf2 feat: remove dead agent registry plumbing
```

Model's state transition diagram includes "Registry entry added (Status: running)" — this no longer exists.

### 5. Atomic spawn architecture

```bash
cat pkg/spawn/atomic.go
# AtomicSpawnPhase1: beads tag + workspace write (with rollback)
# AtomicSpawnPhase2: session ID write + manifest update
```

Spawn is now two-phase atomic with rollback. Model shows linear 5-step flow without atomicity.

### 6. New files in pkg/spawn/ not in model

```bash
ls pkg/spawn/*.go | wc -l
# 37 files
```

Model references 3 files in pkg/spawn/. Now there are 37 Go files. Major additions include:
- `resolve.go` (spawn settings resolution with provenance)
- `atomic.go` (two-phase atomic spawn)
- `claude.go` (Claude CLI backend)
- `session.go` (session management + manifest)
- `ecosystem.go`, `gap.go`, `probes.go`, `rework.go`, `staleness_events.go`, `learning.go`, etc.

### 7. Tests

```bash
go test ./pkg/spawn/ -run TestResolve_BugClass13 -count=1 -v
# PASS: ClaudeBackendImpliesTmuxSpawnMode
# PASS: ExplicitHeadlessOverridesClaudeBackend
# PASS: InfraEscapeHatchAlsoImpliesTmux
```

---

## What I Observed

| Model Claim | Current Reality | Verdict |
|---|---|---|
| `spawn_cmd.go` ~800 lines | 802 lines | Confirms |
| `context.go` ~400 lines | 1315 lines (3.3x growth) | Contradicts |
| `pkg/spawn/config.go` = SpawnConfig + validation | `Config` struct + tier/skill defaults (no validation) | Contradicts |
| Workspace metadata = 5 dotfiles | AGENT_MANIFEST.json + OpenCode metadata + dotfiles (legacy fallback) | Contradicts |
| State transitions include "Registry entry added" | Registry fully removed (commit a9ec5cbf2) | Contradicts |
| Linear 5-step spawn flow | Two-phase atomic spawn with rollback | Contradicts |
| 3 files in pkg/spawn/ referenced | 37 Go files now exist | Extends |
| Tier system (light/full) | Still accurate | Confirms |
| Workspace path format `.orch/workspace/{name}/` | Still accurate | Confirms |
| SPAWN_CONTEXT.md as primary context file | Still accurate (plus ORCHESTRATOR_CONTEXT.md and META_ORCHESTRATOR_CONTEXT.md variants) | Extends |

---

## Model Impact

- [x] **Contradicts** invariants:
  - "Registry entry added (Status: running)" — registry is removed
  - `context.go` ~400 lines — now 1315 lines
  - `config.go` as "SpawnConfig struct and validation" — struct renamed, validation moved to resolve.go
  - Workspace metadata as 5 dotfiles — now AGENT_MANIFEST.json with multi-source fallback
  - Linear spawn flow — now atomic two-phase with rollback

- [x] **Extends** model with:
  - `pkg/spawn/resolve.go` — new centralized settings resolver with provenance tracking (ResolvedSpawnSettings)
  - `pkg/spawn/atomic.go` — two-phase atomic spawn with rollback on failure
  - `pkg/spawn/claude.go` — dedicated Claude CLI backend
  - Context file variants: SPAWN_CONTEXT.md, ORCHESTRATOR_CONTEXT.md, META_ORCHESTRATOR_CONTEXT.md
  - `--backend claude` now implies tmux spawn mode (derived setting)
  - Flash models are now blocked entirely (validateModel returns error)

---

## Recommended Model Updates

1. **Primary Evidence section** needs updated file references:
   - `pkg/spawn/config.go` → Config struct + tier/skill defaults (not validation)
   - Add `pkg/spawn/resolve.go` — settings resolution with provenance
   - Add `pkg/spawn/atomic.go` — two-phase atomic spawn
   - Add `pkg/spawn/claude.go` — Claude CLI backend
   - Update `pkg/spawn/context.go` line count to ~1315

2. **Core Mechanism section** needs:
   - Remove registry from state transitions
   - Add AGENT_MANIFEST.json to workspace metadata
   - Update spawn flow to show atomic phases
   - Add context file variants (orchestrator, meta-orchestrator)

3. **New sections needed:**
   - Resolved settings with provenance (SettingSource enum)
   - Atomic spawn architecture (Phase 1 + Phase 2 with rollback)

---

## Notes

The model is structurally stale — the *concepts* (tier system, workspace creation, SPAWN_CONTEXT.md) are correct but the *implementation details* (file locations, function names, architecture patterns) have drifted significantly. The biggest shift is from ad-hoc dotfiles + registry to AGENT_MANIFEST.json + atomic spawn + resolved settings with provenance.
