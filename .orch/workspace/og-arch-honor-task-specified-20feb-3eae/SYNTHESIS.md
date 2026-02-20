# Session Synthesis

**Agent:** og-arch-honor-task-specified-20feb-3eae
**Issue:** orch-go-1138
**Outcome:** success

---

## Plain-Language Summary

The spawn template was hardcoding "SESSION SCOPE: Medium" on every spawn, even when the orchestrator explicitly set a different scope (like "Small") in the task description. This caused agents to see contradictory instructions — "Small" from the task and "Medium" from the template. The fix parses scope from the task description and also adds a `--scope` flag, then uses the actual scope value in the template with scope-appropriate guidance (Small gets "text edits only", Large gets "checkpoint after 3 hours", Medium stays as before). Backward compatible — tasks without scope still default to Medium.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 11 automated tests covering scope parsing, resolution priority, and template output for all three scope levels.

---

## Delta (What Changed)

### Files Modified
- `pkg/spawn/config.go` — Added `Scope` field to Config struct, added scope constants (ScopeSmall/ScopeMedium/ScopeLarge)
- `pkg/spawn/context.go` — Added `ParseScopeFromTask()`, `ResolveScope()`, scope-conditional template, `Scope` field to contextData
- `pkg/spawn/context_test.go` — Added 3 test functions: TestParseScopeFromTask (7 cases), TestResolveScope (4 cases), TestGenerateContext_SessionScope (4 cases)
- `pkg/orch/extraction.go` — Added `Scope` to SpawnContext struct and BuildSpawnConfig, deduped parseSessionScope to use spawn.ParseScopeFromTask
- `cmd/orch/spawn_cmd.go` — Added `--scope` flag, threaded scope through SpawnContext

### Files Created
- `.kb/models/spawn-architecture/probes/2026-02-20-probe-session-scope-template-honor.md`

### Tests Run
```bash
go test ./pkg/spawn/ -run "TestParseScopeFromTask|TestResolveScope|TestGenerateContext_SessionScope" -v
# PASS: 11/11 tests passing (0.013s)
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Scope resolution priority: explicit `--scope` flag > parsed from task > default "medium" (mirrors tier resolution pattern)
- Deduped `parseSessionScope` from extraction.go — canonical implementation now in `spawn.ParseScopeFromTask`

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing (11/11)
- [x] Probe file created with complete findings
- [x] Ready for `orch complete orch-go-1138`

No discovered work.
