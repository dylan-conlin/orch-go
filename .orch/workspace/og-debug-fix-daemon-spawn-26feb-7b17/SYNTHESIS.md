# Session Synthesis

**Agent:** og-debug-fix-daemon-spawn-26feb-7b17
**Issue:** orch-go-1264
**Outcome:** success

---

## Plain-Language Summary

Separated the beads issue title from its description in the daemon spawn prompt construction. Previously, `orch work` combined `issue.Title + "\n\n" + issue.Description` into a single TASK field, which meant long descriptions (especially those starting with ORIENTATION_FRAME: or FRAME:) would pollute workspace names and make the TASK field unwieldy. Now, TASK contains only the issue title (concise, drives workspace name slug), and the issue description appears in a separate `ORIENTATION_FRAME:` section in SPAWN_CONTEXT.md. This matches how manual `orch spawn` works—short task, separate context.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and outcomes.

Key outcomes:
- `go test ./pkg/spawn/ -run TestGenerateContext_OrientationFrame` — 3/3 pass
- `go test ./cmd/orch/ ./pkg/spawn/ ./pkg/orch/` — all pass
- `go vet ./cmd/orch/ ./pkg/spawn/ ./pkg/orch/` — clean

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/spawn_cmd.go` — `runWork()` now sets `task = issue.Title` (was `title + "\n\n" + description`); description stored in `spawnOrientationFrame` module var; threaded into `SpawnContext.OrientationFrame`
- `pkg/orch/extraction.go` — Added `OrientationFrame` field to `SpawnContext` struct; threaded through `BuildSpawnConfig`; FRAME comment now uses OrientationFrame content (falls back to Task for manual spawns)
- `pkg/spawn/config.go` — Added `OrientationFrame string` field to `Config` struct
- `pkg/spawn/context.go` — Added `OrientationFrame` to `contextData` struct; added `ORIENTATION_FRAME:` section to `SpawnContextTemplate` (conditional, only rendered when non-empty)
- `pkg/spawn/context_test.go` — Added `TestGenerateContext_OrientationFrame` with 3 subtests

---

## Evidence (What Was Observed)

- The existing `og-arch-orientation-frame-dylan-20feb-9c0f` workspace confirmed the pattern: its SPAWN_CONTEXT.md started with `TASK: ORIENTATION_FRAME: Dylan and the orchestrator can't communicate...` — full description text in the TASK field
- The `runWork` function was the only code path that combined title + description into task
- The FRAME beads comment (extraction.go:982) was recording cfg.Task (the combined text); now it records OrientationFrame preferentially
- Manual `orch spawn` is unaffected because `spawnOrientationFrame` defaults to empty string

### Tests Run
```bash
go test ./cmd/orch/ ./pkg/spawn/ ./pkg/orch/ -count=1
# ok cmd/orch 2.809s
# ok pkg/spawn 0.493s
# ok pkg/orch 0.012s
```

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing (3 new + all existing)
- [x] No discovered work

---
