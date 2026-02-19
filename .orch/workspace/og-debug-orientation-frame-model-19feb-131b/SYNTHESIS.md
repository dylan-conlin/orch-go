# Session Synthesis

**Agent:** og-debug-orientation-frame-model-19feb-131b
**Issue:** orch-go-1115
**Outcome:** success

---

## Plain-Language Summary

Model drift issues were being created in the wrong beads project because the `createBeadsIssue` function used the daemon's current working directory to find the beads socket, rather than the project directory where the stale model lives. When the daemon ran from a different project (e.g., price-watch), all model drift issues landed there instead of in orch-go. The fix threads the model's project directory (already available from `projectDirFromModelPath`) through the candidate, group, and issue-creation structs so that `beads.FindSocketPath` and the CLI fallback both target the correct project.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/model_drift_reflection.go` - Added `ProjectDir` field to `modelDriftCandidate`, `modelDriftGroup`, and `ModelDriftIssueCreateArgs` structs. Threaded project dir from metadata through to `createBeadsIssue`. Updated `createBeadsIssue` to use `FindSocketPath(dir)` and `WithCwd(dir)` for RPC, and `FallbackCreateInDir` for CLI fallback.
- `pkg/beads/client.go` - Added `FallbackCreateInDir` function that accepts an explicit directory parameter, overriding DefaultDir and process cwd for `bd create` commands.

---

## Evidence (What Was Observed)

- `createBeadsIssue` at model_drift_reflection.go:345 called `beads.FindSocketPath("")` with empty dir
- `FindSocketPath("")` falls back to `os.Getwd()` when `DefaultDir` is unset (client.go:151)
- Daemon code in daemon.go never sets `beads.DefaultDir`
- Model path contains correct project via `projectDirFromModelPath` (e.g., `/Users/.../orch-go` from `.kb/models/...`) but this was never passed to issue creation
- Staleness events file is global (`~/.orch/model-staleness-events.jsonl`) so events from all projects are mixed together

### Tests Run
```bash
go build ./cmd/orch/        # PASS
go vet ./cmd/orch/           # PASS
go vet ./pkg/beads/          # PASS
go vet ./pkg/daemon/         # PASS
go test ./pkg/daemon/ -v     # PASS - all tests including ModelDrift
go test ./pkg/beads/ -v      # PASS - all tests
```

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-1115`

---

## Unexplored Questions

- The `ListIssuesWithLabel` call in model drift backpressure also uses daemon cwd, meaning backpressure checks may query the wrong project. However, once issues are created in the correct project, future daemon runs from that project will see them correctly. This is a secondary concern.
- The `listIssuesWithLabelCLI` and `ListReadyIssues` functions in issue_adapter.go also don't set `cmd.Dir`. Any daemon feature that creates or queries issues inherits the process cwd. This is a broader pattern worth addressing (discovered work).

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-orientation-frame-model-19feb-131b/`
**Beads:** `bd show orch-go-1115`
