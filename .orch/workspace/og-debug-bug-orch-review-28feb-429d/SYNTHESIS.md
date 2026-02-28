# Session Synthesis

**Agent:** og-debug-bug-orch-review-28feb-429d
**Issue:** orch-go-ym0m
**Outcome:** success

---

## Plain-Language Summary

`orch review triage` was returning all 28 open issues instead of only those labeled `triage:review` (which was zero). The root cause was that `CLIClient.List()` in `pkg/beads/cli_client.go` did not pass the `Labels` or `LabelsAny` fields to the `bd list` CLI command. Since the beads daemon socket doesn't exist in this environment, the RPC path always fails and falls through to the CLI fallback — which was issuing `bd list --json --status open --limit 0` without any `-l` flag, returning all open issues unfiltered.

The fix adds `-l` and `--label-any` flag passthrough to `CLIClient.List()`. This also fixes any other callers passing Labels through the CLI client (e.g., `pkg/orch/extraction.go` which queries by `skill:architect` label).

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## TLDR

Fixed `orch review triage` showing all open issues instead of only `triage:review`-labeled ones, caused by `CLIClient.List()` not passing label filters to the `bd list` CLI command.

---

## Delta (What Changed)

### Files Modified
- `pkg/beads/cli_client.go` - Added Labels and LabelsAny flag passthrough to List() method
- `pkg/beads/cli_client_test.go` - Added TestCLIClient_ListArgsBuilding with 6 subtests covering label filtering

---

## Evidence (What Was Observed)

- `bd list --status=open -l triage:review --json` returns `[]` (0 items) — correct
- `bd list --status=open --json --limit 0` returns 28 items — all open issues, no label filter
- `orch review triage --non-interactive` showed 28 items before fix, 0 items after fix
- `.beads/bd.sock` does not exist, confirming RPC path always fails and CLI fallback is used

### Tests Run
```bash
go test ./pkg/beads/ -v
# 57 tests passed, 10 skipped (no daemon socket), 0 failed

go vet ./pkg/beads/ ./cmd/orch/
# No issues
```

---

## Architectural Choices

No architectural choices — task was within existing patterns. The fix adds the missing flag passthrough that the RPC path already handled via JSON serialization.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- When beads daemon is not running (no bd.sock), ALL beads queries fall through to CLI client — any filtering not implemented in CLIClient.List() is silently ignored, returning unfiltered results

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Smoke test passes (orch review triage shows 0 items)
- [x] Ready for `orch complete orch-go-ym0m`

---

## Unexplored Questions

- `pkg/orch/extraction.go:461` also queries with `Labels: []string{"skill:architect"}` via CLI client — this was silently broken too and is now fixed by the same change. Worth verifying extraction behavior in a separate session.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-bug-orch-review-28feb-429d/`
**Beads:** `bd show orch-go-ym0m`
