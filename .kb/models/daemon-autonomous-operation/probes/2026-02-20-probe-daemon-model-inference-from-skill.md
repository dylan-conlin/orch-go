# Probe: Daemon Model Inference from Skill Type

**Status:** Complete
**Date:** 2026-02-20
**Model:** daemon-autonomous-operation
**Issue:** orch-go-1148

## Question

Can the daemon infer an appropriate model from the skill type during auto-spawn, and does the inferred model correctly flow through to `orch work --model` and the resolve pipeline's model-aware backend routing?

## What I Tested

1. Added `InferModelFromSkill()` function mapping skill→model alias in `pkg/daemon/skill_inference.go`
2. Added `Model` field to `OnceResult` and `PreviewResult`
3. Changed `spawnFunc` signature to `func(beadsID, model string) error`
4. Updated `SpawnWork` to pass `--model` to `orch work` in `pkg/daemon/issue_adapter.go`
5. Added `--model` flag to `workCmd` in `cmd/orch/spawn_cmd.go`
6. Updated `orch daemon preview`, `orch daemon once`, and daemon run loop output to show inferred model
7. Ran `go test ./pkg/daemon/ -count=1` (6.389s, all pass)
8. Ran `go test ./cmd/orch/ -count=1` (3.325s, all pass)
9. Ran `go build ./cmd/orch/ && go vet ./cmd/orch/ && go vet ./pkg/daemon/` (clean)

## What I Observed

- **Model inference mapping works correctly**: opus for deep-reasoning skills (systematic-debugging, investigation, architect, codebase-audit, research), sonnet for implementation skills (feature-impl, issue-creation) and as default
- **Extraction override works**: When extraction gate replaces skill with "feature-impl", model is also re-inferred to match
- **Model flows through entire path**: daemon Once() → spawnFunc(beadsID, model) → SpawnWork → `orch work --model <model> <beadsID>` → resolve pipeline → model-aware backend routing
- **All 23 test closures updated** for new spawnFunc signature, all existing tests pass
- **Preview output enhanced**: Shows both "Inferred skill" and "Inferred model" lines

## Model Impact

**Extends daemon-autonomous-operation model:**
- New capability: model inference alongside skill inference
- Mapping: `skillModelMapping` map in skill_inference.go (configurable)
- Default: "sonnet" via `DefaultSkillModel` constant
- Feeds into model-aware backend routing (kb-2d62ef): opus→claude backend, sonnet→claude backend (both Anthropic)

**Confirms existing invariant:**
- Skill inference uses issue type to map task→feature-impl, bug→architect, etc. (unchanged)
- Model inference is layered on top of skill inference, not replacing it
