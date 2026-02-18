# Session Synthesis

**Agent:** og-debug-fix-explicit-model-17feb-2c9d
**Issue:** orch-go-1020
**Outcome:** success

---

## Plain-Language Summary

The infrastructure escape hatch in `DetermineSpawnBackend` was silently overriding the backend to "claude" whenever it detected infrastructure-related keywords in the task — even when the user had explicitly passed `--model codex`. Since codex runs on the opencode backend, forcing claude made the explicit model choice meaningless. The fix adds `--model` flag awareness: when the user explicitly chooses a model, the escape hatch becomes advisory (warning only) instead of overriding, allowing the config backend (opencode) to be used. The escape hatch still fires when no explicit flags are set.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- `go test ./pkg/orch/ -run TestDetermineSpawnBackend` — 5 tests, all pass
- `go vet ./pkg/orch/` — clean
- `go build ./cmd/orch/` — clean

---

## Delta (What Changed)

### Files Modified
- `pkg/orch/extraction.go` - Added `explicitModel` tracking in `DetermineSpawnBackend`; when `--model` is explicitly set, infrastructure detection becomes advisory (warning only) instead of overriding the backend. Updated priority comment.
- `pkg/orch/extraction_test.go` - Added `TestDetermineSpawnBackend_ExplicitModelPreventsInfraOverride` and `TestDetermineSpawnBackend_ExplicitModelAndBackend` tests.

---

## Evidence (What Was Observed)

- `DetermineSpawnBackend` at `pkg/orch/extraction.go:612` already handled explicit `--backend` flag correctly (line 621: `explicitBackend := backendFlag != ""`)
- The `spawnModel` parameter was passed to the function but NEVER used in the function body — it was dead code
- When `--model codex` was set without `--backend`, the function fell through to `isInfrastructureWork()` which matched on keywords like "spawn_cmd.go", "opencode", etc., forcing backend to "claude"
- The claude backend (`SpawnClaude` at `pkg/spawn/claude.go:55`) launches `claude --dangerously-skip-permissions` without passing `--model`, so any non-Claude model is silently ignored
- User's config: `spawn_mode: opencode`, `default_model: codex` — both overridden by infrastructure detection for all orch-go work

### Tests Run
```bash
go test ./pkg/orch/ -run TestDetermineSpawnBackend -v
# 5 tests, all PASS

go vet ./pkg/orch/ ./cmd/orch/
# clean

go build ./cmd/orch/
# clean
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Treat explicit `--model` the same as explicit `--backend` for escape hatch bypass — model choice implies backend requirements
- Infrastructure detection stays active for advisory warnings even when flags are explicit

### Constraints Discovered
- `spawnModel` parameter was already passed to `DetermineSpawnBackend` but unused — this enabled the fix without changing function signatures

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-1020`

---

## Unexplored Questions

- The `spawnOpus` flag (`--opus`) is declared and registered but never passed to `DetermineSpawnBackend` — it's dead code since the function signature change. May warrant cleanup.
- The `isInfrastructureWork` function is duplicated in both `cmd/orch/spawn_cmd.go` and `pkg/orch/extraction.go` — the cmd version is only used by tests.

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-fix-explicit-model-17feb-2c9d/`
**Beads:** `bd show orch-go-1020`
