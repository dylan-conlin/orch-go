# Session Synthesis

**Agent:** og-debug-harness-precommit-accretion-20mar-76d5
**Issue:** orch-go-wzkd7
**Outcome:** success

---

## Plain-Language Summary

The `harness` binary (separate project at `~/Documents/personal/harness/`) was still blocking commits when files exceeded the critical accretion threshold, despite decision 2026-03-17 declaring all accretion gates advisory-only. The orch-go `orch precommit accretion` command was already advisory, but `.git/hooks/pre-commit` also called `harness precommit accretion` which returned a non-zero exit code and the message "commit blocked by accretion gate" when agent-caused bloat pushed a file over the critical threshold. Fixed by changing the harness binary's `CheckStagedAccretionWithResolver()` to always produce warnings (never block), removing the `SKIP_ACCRETION_GATE` bypass env var (no longer needed), and removing the blocking error path from the command handler. All 7 harness packages pass. Binary rebuilt and installed.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for exact verification commands.

Key outcomes:
- `harness precommit accretion` exits 0 (was exit 1 for large files)
- `harness precommit accretion --json` always returns `"passed": true`
- All harness tests pass: `go test ./... -count=1` (7/7 packages)

---

## TLDR

The `harness` binary's precommit accretion gate was still blocking commits (exit 1) for files exceeding the critical threshold. Fixed by making `CheckStagedAccretionWithResolver` advisory-only — agent-caused bloat now goes to WarningFiles instead of BlockedFiles, and the command always exits 0 with warnings.

---

## Delta (What Changed)

### Files Modified (in ~/Documents/personal/harness/)
- `pkg/accretion/precommit.go` — `CheckStagedAccretionWithResolver()` no longer sets `Passed=false` or appends to `BlockedFiles`. Agent-caused bloat goes to `WarningFiles`. `FormatStagedAccretionError()` now returns empty string. `FormatStagedAccretionWarnings()` simplified to two categories (threshold warnings and pre-existing bloat).
- `cmd/harness/precommit_cmd.go` — Removed `SKIP_ACCRETION_GATE=1` bypass logic (not needed when gate never blocks). Removed `!result.Passed` blocking error path. Command always exits 0 with advisory output.
- `pkg/accretion/precommit_test.go` — All tests updated: cases that expected `Passed=false` now expect `Passed=true`. Tests verify `BlockedFiles` is always empty. New test `TestCheckStagedAccretion_CriticalAlwaysAdvisory` replaces `TestCheckStagedAccretion_BlockTakesPrecedenceOverWarning`.
- `cmd/harness/precommit_test.go` — `TestPrecommitAccretionCmd_BlocksLargeFile` → `TestPrecommitAccretionCmd_LargeFileAdvisory`. Removed bypass event test. `TestPrecommitAccretionCmd_RespectsConfigThresholds` now expects advisory pass, not blocking error. New test `TestPrecommitAccretionCmd_JSONAlwaysPassedTrue`.

### Files Updated (in orch-go)
- `harness` (binary) — Rebuilt from harness project

### Commits
- `1d01c5d` (harness repo) — fix: make accretion gate advisory-only, never block (orch-go-wzkd7)

---

## Evidence (What Was Observed)

- The `.git/hooks/pre-commit` hook calls BOTH `orch precommit accretion` (advisory, already fixed) AND `harness precommit accretion` (was blocking)
- Root cause: `harness/pkg/accretion/precommit.go` line 91 set `result.Passed = false` for agent-caused bloat exceeding critical threshold
- `harness/cmd/harness/precommit_cmd.go` line 115 returned `fmt.Errorf("commit blocked by accretion gate")` when `!result.Passed`
- Two accretion checks running in sequence: scripts/pre-commit-exec-start-cleanup.sh calls `orch precommit accretion`, then .git/hooks/pre-commit calls `harness precommit accretion`

### Tests Run
```bash
cd ~/Documents/personal/harness && go test ./... -count=1
# ok github.com/dylan-conlin/harness/cmd/harness       3.984s
# ok github.com/dylan-conlin/harness/pkg/accretion     7.372s
# ok github.com/dylan-conlin/harness/pkg/config        0.609s
# ok github.com/dylan-conlin/harness/pkg/control       0.435s
# ok github.com/dylan-conlin/harness/pkg/events        1.418s
# ok github.com/dylan-conlin/harness/pkg/report        5.671s
# ok github.com/dylan-conlin/harness/pkg/scaffold      0.913s

harness precommit accretion
# pre-commit: accretion gate passed (advisory)
# exit code: 0

harness precommit accretion --json
# {"passed": true, "warnings": null}
# exit code: 0
```

---

## Architectural Choices

### Advisory-only with no bypass flag
- **What I chose:** Removed `SKIP_ACCRETION_GATE=1` entirely rather than keeping it as a no-op
- **What I rejected:** Keeping the env var for backward compatibility
- **Why:** Per decision 2026-03-17, bypass logic is dead code when the gate never blocks. Removing it reduces complexity and avoids agents learning reflexive bypass patterns.
- **Risk accepted:** Any automation relying on `SKIP_ACCRETION_GATE=1` will see the env var silently ignored (same outcome — gate passes), not an error.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The pre-commit hook has a duplicate accretion check: `orch precommit accretion` in the script AND `harness precommit accretion` from the hook shim. Both need to be advisory for the decision to take effect.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing (7/7 packages)
- [x] Binary rebuilt and installed
- [x] Ready for `orch complete orch-go-wzkd7`

---

## Unexplored Questions

- The duplicate accretion check (orch + harness) is redundant — could be consolidated. Not in scope for this bug fix.

---

## Friction

- `ceremony`: Tab-indented Go file Edit tool failures — had to use Write tool for full file rewrite instead of surgical edits. Known CLAUDE.md gotcha.

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-harness-precommit-accretion-20mar-76d5/`
**Beads:** `bd show orch-go-wzkd7`
