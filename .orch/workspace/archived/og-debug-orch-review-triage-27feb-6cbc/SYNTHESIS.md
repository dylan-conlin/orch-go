# Session Synthesis

**Agent:** og-debug-orch-review-triage-27feb-6cbc
**Issue:** orch-go-qku1
**Outcome:** success

---

## Plain-Language Summary

The `orch review triage` promote-to-ready flow was broken because `CLIClient.RemoveLabel` called `bd unlabel`, a command that doesn't exist in the bd CLI. The correct command is `bd label remove`. This one-line fix (plus making `AddLabel` explicit with `bd label add` instead of relying on implicit default behavior) unblocks the triage review workflow. All 5 stuck triage:review items can now be promoted.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace directory.

Key outcomes:
- `bd unlabel` → `bd label remove` (fixes the crash)
- `bd label <id> <label>` → `bd label add <id> <label>` (explicit subcommand)
- All tests pass: `go test ./pkg/beads/ -run TestCLI` and `go test ./cmd/orch/ -run TestTriage`

---

## TLDR

Fixed CLIClient.RemoveLabel calling non-existent `bd unlabel` command (correct: `bd label remove`). Also made CLIClient.AddLabel explicit (`bd label add` instead of implicit). This was preventing triage:review items from being promoted to triage:ready.

---

## Delta (What Changed)

### Files Modified
- `pkg/beads/cli_client.go` - Fixed RemoveLabel to use `bd label remove`, AddLabel to use `bd label add`
- `pkg/beads/cli_client_test.go` - Added test verifying correct command construction for both label operations

### Commits
- (pending) fix: CLIClient label commands use correct bd subcommands

---

## Evidence (What Was Observed)

- `bd unlabel` returns "unknown command" error (confirmed via shell)
- `bd label remove <id> <label>` works correctly (confirmed via shell)
- `bd label <id> <label>` works via implicit add (undocumented but functional)
- `bd label add <id> <label>` is the explicit documented form
- `FallbackRemoveLabel` uses `bd update --remove-label` which is correct (different code path, not affected)
- RPC client path (`Client.RemoveLabel`) uses daemon protocol, not affected by this bug

### Tests Run
```bash
go test ./pkg/beads/ -run TestCLI -v -count=1
# PASS: 4 tests (TestCLIClient_bdCommand, TestCLIClient_ImplementsBeadsClient, TestCLIClient_LabelCommands/AddLabel, TestCLIClient_LabelCommands/RemoveLabel)

go test ./cmd/orch/ -run TestTriage -v -count=1
# PASS: TestTriageItemFromIssue

go vet ./cmd/orch/ ./pkg/beads/
# Clean
```

### Smoke Test
```bash
# Created test issue orch-go-sr11 with triage:review label
# Confirmed bd label remove orch-go-sr11 triage:review works
# Confirmed bd label add orch-go-sr11 triage:ready works
# Confirmed labels updated correctly via bd label list
# Closed test issue after verification
```

---

## Architectural Choices

No architectural choices — task was a straightforward bug fix within existing patterns.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `bd label` requires explicit subcommand (`add`/`remove`/`list`/`list-all`). Using `bd label <id> <label>` without subcommand happens to work as implicit add, but this is undocumented behavior.
- `bd comment` is deprecated — replacement is `bd comments add`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-qku1`

---

## Unexplored Questions

- `bd comment` deprecation: all callers in orch-go (beads client AddComment, review_triage deferTriageItem, etc.) should be audited against current bd CLI syntax. The `bd comment` command still works but shows deprecation warning.
- Whether the beads RPC daemon's `label_remove` operation works correctly (not tested since daemon wasn't running, but the RPC client code looks correct)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-orch-review-triage-27feb-6cbc/`
**Beads:** `bd show orch-go-qku1`
