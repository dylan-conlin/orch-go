# Session Synthesis

**Agent:** og-debug-daemon-fails-spawn-25feb-c1d2
**Issue:** orch-go-1230
**Outcome:** success

---

## Plain-Language Summary

The daemon's cross-project spawn was silently failing because `orch work --workdir` didn't redirect beads database lookups to the target project. When the daemon found a toolshed issue via multi-project polling and called `orch work --workdir /path/to/toolshed toolshed-164`, the `runWork()` function tried to look up the issue in orch-go's local `.beads/` database (where it doesn't exist) before ever consulting the `--workdir` flag. This caused `verify.GetIssue()` to return "issue not found" and the spawn to fail silently — 103 consecutive times for all cross-project issues.

The fix sets `beads.DefaultDir` from `--workdir` at the start of `runWork()`, so all downstream beads lookups (GetIssue, FindSocketPath, skill inference) target the correct project's database. Additionally, `daemon preview` and `daemon dry-run` now initialize the ProjectRegistry so cross-project issues are visible in diagnostic output.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification commands and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/spawn_cmd.go` - Set `beads.DefaultDir` from `spawnWorkdir` at start of `runWork()`, restored on return
- `cmd/orch/daemon.go` - Initialize `ProjectRegistry` in `runDaemonDryRun()` and `runDaemonPreview()` for cross-project visibility

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- `bd show toolshed-164` from orch-go directory: `Error: no issue found matching "toolshed-164"` (root cause confirmed)
- `bd show toolshed-164` from toolshed directory: returns issue correctly
- `beads.FindSocketPath("")` uses CWD or `beads.DefaultDir` — setting DefaultDir redirects all downstream lookups
- `verify.GetIssue()` at spawn_cmd.go:396 is the first beads call in `runWork()`, runs BEFORE `spawnWorkdir` is consulted
- `daemon preview` without registry showed only orch-go issues; with registry shows toolshed, specs-platform, bd, orch-cli, opencode

### Tests Run
```bash
go test ./cmd/orch/ ./pkg/daemon/ -count=1 -timeout 60s
# ok  github.com/dylan-conlin/orch-go/cmd/orch  6.279s
# ok  github.com/dylan-conlin/orch-go/pkg/daemon  6.640s

go vet ./cmd/orch/
# (clean)
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `beads.DefaultDir` is a global variable — safe for CLI commands (separate processes) but would need mutex for concurrent use
- `CheckBlockingDependencies` in daemon's `NextIssueExcluding` is fail-open for cross-project issues (can't check deps in another project's beads DB without passing projectDir) — acceptable but creates a blind spot

### Decisions Made
- Fix at the earliest possible point (`runWork` entry) rather than creating project-aware variants of every downstream function — minimizes blast radius
- Use `defer` to restore `beads.DefaultDir` after `runWork` returns — clean restoration pattern

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-1230`

---

## Unexplored Questions

- `CheckBlockingDependencies` should accept `projectDir` for cross-project dependency checking — currently fail-open, which means the daemon can't detect if a cross-project issue is blocked
- Content-aware dedup (`FindInProgressByTitle`) only checks the local beads database — a cross-project issue with a duplicate title in another project wouldn't be caught

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-daemon-fails-spawn-25feb-c1d2/`
**Beads:** `bd show orch-go-1230`
