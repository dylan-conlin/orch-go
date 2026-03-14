# Session Synthesis

**Agent:** og-debug-fix-daemon-beads-13mar-4747
**Issue:** orch-go-q6y6j
**Outcome:** success

---

## Plain-Language Summary

Fixed the daemon's beads socket resolution to use explicit project directories instead of relying on `os.Getwd()`. When the daemon runs from launchd, the working directory is wrong (e.g., `/`), causing `FindSocketPath("")` to find a different project's beads socket. This meant the daemon saw issues from wrong projects and missed orch-go issues entirely. The fix propagates the project registry's `CurrentDir()` through all daemon beads query paths: issue listing, status checks, epic children, label queries, title dedup, and status updates.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for behavioral verification.

Key outcomes:
- `defaultIssueQuerier` methods all route through `*ForProject` variants when registry is set
- `ListReadyIssuesMultiProject` no longer falls back to `ListReadyIssues()` (which used `FindSocketPath("")`)
- `spawnIssue` resolves `statusProjectDir` from registry for local-project issues
- `TitleDedupBeadsGate` uses project-aware `FindInProgressByTitleForProject` when registry is set
- 4 new unit tests, all existing tests pass (15.4s)

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/interfaces.go` — Added `currentDir` field to `defaultIssueQuerier` and `defaultIssueUpdater`; all methods route to `*ForProject` variants when `currentDir` is set
- `pkg/daemon/issue_selection.go` — `resolveIssueQuerier()` now sets `currentDir` from `ProjectRegistry.CurrentDir()`
- `pkg/daemon/spawn_execution.go` — `spawnIssue()` resolves `statusProjectDir` from registry for local-project issues; `buildSpawnPipeline()` wires project-aware `FindInProgressByTitleForProject` into `TitleDedupBeadsGate`; rollback also uses resolved project dir
- `pkg/daemon/issue_adapter.go` — Added `ListEpicChildrenForProject()` and `FindInProgressByTitleForProject()`; refactored `ListEpicChildren` and `FindInProgressByTitle` to delegate to new ForProject variants; removed broken zero-issues fallback in `ListReadyIssuesMultiProject`
- `pkg/daemon/daemon_test.go` — 4 new tests: currentDir propagation, no-registry fallback, no-fallback-to-FindSocketPath, and spawn-uses-registry-dir

---

## Evidence (What Was Observed)

- `FindSocketPath("")` (client.go:68-75) falls back to `os.Getwd()` when dir is empty — confirmed root cause
- 6 functions in `issue_adapter.go` used `FindSocketPath("")`: `ListReadyIssues`, `ListEpicChildren`, `ListIssuesWithLabel`, `GetBeadsIssueStatus`, `FindInProgressByTitle`, `UpdateBeadsStatus`
- The daemon already had project-aware variants for most operations (`*ForProject` functions) but didn't use them for local-project issues
- `ListReadyIssuesMultiProject` had a fallback at line 451 that called `ListReadyIssues()` when zero issues returned — this would find wrong-project issues from launchd

### Tests Run
```bash
go test ./pkg/daemon/... -count=1 -timeout 120s
# ok  github.com/dylan-conlin/orch-go/pkg/daemon  15.439s

go build ./...
# (clean build, no errors)
```

---

## Architectural Choices

### Route through `currentDir` field vs. passing projectDir as parameter to IssueQuerier methods
- **What I chose:** Added `currentDir` field to `defaultIssueQuerier`, set lazily from registry in `resolveIssueQuerier()`
- **What I rejected:** Changing the `IssueQuerier` interface to accept `projectDir` on every method
- **Why:** Interface change would break all mock implementations in tests (50+ sites). The `currentDir` field achieves the same result without interface changes, and the lazy update pattern already exists for `registry`.
- **Risk accepted:** If `currentDir` becomes stale (registry refreshed), the querier uses the old dir. Low risk — `currentDir` is the daemon's own project dir which doesn't change.

### Remove `ListReadyIssuesMultiProject` zero-issues fallback vs. fix the fallback
- **What I chose:** Removed the fallback entirely — 0 issues is a valid result
- **What I rejected:** Making the fallback also project-aware
- **Why:** The fallback was designed for "registry has projects but none have beads sockets." In practice, this is better handled by returning empty and letting the daemon poll again next cycle, rather than querying with wrong CWD.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Prior knowledge constraint confirmed: "FindSocketPath must receive explicit projectDir when caller has cross-project context" — this session found the exact 6 violation sites and fixed them

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing (15.4s, 0 failures)
- [x] Ready for `orch complete orch-go-q6y6j`

---

## Unexplored Questions

- `active_count.go`, `cleanup.go`, `artifact_sync_default.go`, `synthesis_auto_create.go` also use `FindSocketPath("")` — these are secondary daemon utility paths, not part of the issue query pipeline. Lower priority but same pattern.
- `countUnverifiedWithoutFiltering` in `issue_adapter.go` also uses `FindSocketPath("")` — only reached as fallback-of-fallback path when `verify.CountUnverifiedWork()` fails

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-debug-fix-daemon-beads-13mar-4747/`
**Beads:** `bd show orch-go-q6y6j`
