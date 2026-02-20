# Probe: SESSION SCOPE Template Honor

**Status:** Complete
**Date:** 2026-02-20
**Model:** spawn-architecture
**Issue:** orch-go-1138

## Question

Does the spawn template honor task-specified SESSION SCOPE, or does it always inject "Medium" regardless of what the task description specifies?

## What I Tested

1. Read the `SpawnContextTemplate` in `pkg/spawn/context.go` — line 228 hardcodes `SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])`
2. The `contextData` struct had no `Scope` field
3. The `Config` struct had no `Scope` field
4. Despite `parseSessionScope()` existing in `extraction.go` for tier inference, the parsed scope was never threaded to the template

### Fix Applied

1. Added `Scope` field to `Config` (config.go) and `contextData` (context.go)
2. Added `ParseScopeFromTask()` and `ResolveScope()` to `pkg/spawn/context.go`
3. Updated `SpawnContextTemplate` to use scope-conditional output (Small/Medium/Large with appropriate guidance)
4. Added `--scope` flag to spawn command
5. Deduplicated `parseSessionScope` from extraction.go → now calls `spawn.ParseScopeFromTask`
6. Threaded scope through `SpawnContext` and `BuildSpawnConfig` in extraction.go

### Test Results

```bash
go test ./pkg/spawn/ -run "TestParseScopeFromTask|TestResolveScope|TestGenerateContext_SessionScope" -v
# 11 tests, all PASS

go test ./pkg/spawn/ -run "TestParseScopeFromTask" -v
# 7 tests: parses small/medium/large, case insensitive, no scope, empty task, multiline

go test ./pkg/spawn/ -run "TestResolveScope" -v
# 4 tests: explicit priority, task parse, default medium, case normalization

go test ./pkg/spawn/ -run "TestGenerateContext_SessionScope" -v
# 4 tests: small/large/parsed-from-task/default-medium template output
```

## What I Observed

- **Before fix:** Template always emitted `SESSION SCOPE: Medium` regardless of task content
- **After fix:** Template emits scope matching the task description or explicit `--scope` flag
- **Scope resolution priority:** explicit `--scope` flag > parsed from task > default "medium"
- **Backward compatible:** Tasks without scope specification still get "Medium" (no behavior change)

## Model Impact

**Extends** Spawn Architecture model:

- **New invariant:** SESSION SCOPE in template reflects actual task scope (parsed or explicit)
- **New spawn parameter:** `--scope` flag (small/medium/large)
- **Scope resolution:** Same regex pattern as tier inference, now canonical in `pkg/spawn`
- **Dedup:** `parseSessionScope` in extraction.go now delegates to `spawn.ParseScopeFromTask`

**Confirms** Critical Invariant 1 (workspace name = kebab-case task description) — scope doesn't affect workspace naming, only template output.
