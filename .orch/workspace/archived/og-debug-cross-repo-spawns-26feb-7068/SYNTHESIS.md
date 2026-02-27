# Session Synthesis

**Agent:** og-debug-cross-repo-spawns-26feb-7068
**Issue:** orch-go-1152
**Outcome:** success

---

## Plain-Language Summary

When the orchestrator spawns a probe agent with `--workdir` pointing to one repo but the model being probed lives in a different repo, probe artifacts were silently landing in the wrong repo's `.kb/` directory. This fix adds three layers of protection: (1) spawn-time detection that warns the orchestrator when a cross-repo model situation exists, (2) explicit instructions in SPAWN_CONTEXT.md telling the agent to use the absolute path from the model's `See:` reference instead of creating probes relative to their workdir, and (3) a post-completion check in `orch complete` that flags when a reported probe_path or investigation_path is outside the agent's project directory, alerting the orchestrator that manual integration may be needed.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expectations.

---

## TLDR

Cross-repo spawns now detect when the primary model lives in a different repo than `--workdir`, warn the orchestrator at spawn time, instruct the agent explicitly about where to create probes, and flag cross-repo deliverables at completion time.

---

## Delta (What Changed)

### Files Modified
- `pkg/spawn/kbcontext.go` - Added `DetectCrossRepoModel()` function and `CrossRepoModelDir` field to `KBContextFormatResult`
- `pkg/spawn/config.go` - Added `CrossRepoModelDir` field to `Config` struct
- `pkg/spawn/context.go` - Added `CrossRepoModelDir` to `contextData`, added cross-repo warning in SPAWN_CONTEXT.md template
- `pkg/orch/extraction.go` - Threaded `CrossRepoModelDir` through `SpawnContext`, `GatherSpawnContext()`, and `BuildSpawnConfig()`
- `cmd/orch/spawn_cmd.go` - Added stderr warning when cross-repo model detected, wired `CrossRepoModelDir` into SpawnContext
- `cmd/orch/rework_cmd.go` - Updated `GatherSpawnContext()` call for new return value
- `pkg/verify/beads_api.go` - Added `ParseProbePathFromComments()`, `CheckCrossRepoDeliverable()`
- `pkg/verify/check.go` - Added cross-repo deliverable warning in `VerifyCompletionFullWithComments()`
- `pkg/spawn/kbcontext_test.go` - Added `TestDetectCrossRepoModel`
- `pkg/spawn/context_test.go` - Added `TestGenerateContext_CrossRepoModelWarning`
- `pkg/verify/check_test.go` - Added `TestParseProbePathFromComments`, `TestCheckCrossRepoDeliverable`

---

## Evidence (What Was Observed)

- Root cause confirmed: `ProjectDir` (from `--workdir`) is used as the single source of truth for all deliverable paths, but `PrimaryModelPath` (from kb context) can reference a model in a different repo
- The SPAWN_CONTEXT.md template said "Create probe file in model's probes/ directory" without specifying the absolute path, relying on agents to derive it from the `See:` reference
- Detection approach uses git repo root detection (`.git` directory walking) with `.kb/` directory fallback

### Tests Run
```bash
go test ./pkg/spawn/ -run "TestDetectCrossRepoModel|TestGenerateContext_CrossRepoModelWarning" -v
# PASS: 6/6 tests

go test ./pkg/verify/ -run "TestParseProbePathFromComments|TestCheckCrossRepoDeliverable" -v
# PASS: 7/7 tests

go test ./pkg/spawn/ ./pkg/verify/ ./cmd/orch/
# PASS: all 3 packages
```

---

## Architectural Choices

### Detection via git root walking vs path prefix comparison
- **What I chose:** Git root detection (walk up to find `.git`) with `.kb/` fallback
- **What I rejected:** Simple path prefix comparison (would miss repos in nested directories)
- **Why:** Git root gives the actual repo boundary, which is the semantically correct unit
- **Risk accepted:** Slightly slower due to filesystem stat calls, but only runs once per spawn

### Warning-only at completion vs blocking gate
- **What I chose:** Informational warning in `VerifyCompletionFullWithComments()`
- **What I rejected:** Blocking gate that fails completion
- **Why:** Cross-repo deliverables aren't necessarily wrong — the agent may have correctly followed the absolute path. Blocking would create false positives.
- **Risk accepted:** Orchestrator must manually check the warning

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The `os.Stat` check for `.git` handles both regular git directories and git worktrees (where `.git` is a file, not a directory)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (10/10 new tests, all existing tests pass)
- [x] Ready for `orch complete orch-go-1152`

---

## Unexplored Questions

- Should the orchestrator auto-create a follow-up issue in the model's repo when a cross-repo deliverable is detected? (Currently manual integration)
- When agents receive the cross-repo warning, do they reliably follow the absolute path? Would need to observe actual agent behavior with this new template section.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-cross-repo-spawns-26feb-7068/`
**Beads:** `bd show orch-go-1152`
