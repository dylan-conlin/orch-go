# Session Synthesis

**Agent:** og-feat-drift-prevention-auto-08jan-c18c
**Issue:** orch-go-edcy
**Duration:** 2026-01-08 → 2026-01-08
**Outcome:** success

---

## TLDR

Implemented automatic CLI command doc debt tracking: new commands detected during `orch complete` are persisted to `~/.orch/doc-debt.json`, surfaced via `orch doctor --docs`, and managed via `orch docs` subcommands.

---

## Delta (What Changed)

### Files Created
- `cmd/orch/docs_cmd.go` - New `orch docs` command with `list`, `mark`, and `unmark` subcommands
- `pkg/userconfig/docdebt_test.go` - Unit tests for doc debt tracking

### Files Modified
- `pkg/userconfig/userconfig.go` - Added DocDebt types, LoadDocDebt, SaveDocDebt, and related methods
- `cmd/orch/complete_cmd.go` - Integrated doc debt tracking into new command detection flow
- `cmd/orch/doctor.go` - Added `--docs` flag and runDocDebtCheck function

### Commits
- To be committed: feat: add CLI command doc debt tracking

---

## Evidence (What Was Observed)

- Existing `detectNewCLICommands()` in complete_cmd.go:758-826 provides reliable detection of new CLI commands
- Config drift pattern in doctor.go:885-1020 establishes proven design for drift detection (separate flag, report struct, clear output)
- ~/.orch/ directory already contains multiple JSON state files (gap-tracker.json, session.json, etc.) - doc-debt.json follows this pattern
- All unit tests pass for new DocDebt functionality

### Tests Run
```bash
# Unit tests for doc debt
go test ./pkg/userconfig/... -run TestDocDebt -v
# PASS: 4/4 tests

# Full test suite
go test ./... -short
# PASS: All except pre-existing beads integration test
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-drift-prevention-auto-track-cli.md` - Investigation documenting findings and design decisions

### Decisions Made
- Passive tracking over blocking: Advisory approach preferred to avoid interrupting workflows
- JSON state file over YAML: Follows existing pattern (gap-tracker.json, session.json)
- Doctor integration over kb reflect: Keeps drift detection centralized in orch, kb reflect is for CLAUDE.md constraints

### Constraints Discovered
- Doc debt state must be separate from config.yaml (config is settings, doc-debt.json is state)
- Detection happens during `orch complete` only (not continuous scanning)

### Externalized via `kn`
- `kn decide "passive doc debt tracking over blocking" --reason "Blocking completion would interrupt legitimate workflows without proportional benefit"` - recommended

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-edcy`

### Future Enhancements (Optional)
- Weekly digest via daemon could surface doc debt periodically
- Auto-detection of doc updates (scan SKILL.md and commands-reference.md for command mentions)
- Integration with orch changelog to track when docs are updated

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should doc debt be project-specific or global? Currently global at ~/.orch/doc-debt.json
- Could we auto-detect documentation by parsing the skill/reference files?

**Areas worth exploring further:**
- Daemon integration for periodic doc debt summaries
- Cross-repo doc debt tracking (skillc produces SKILL.md in different repo)

**What remains unclear:**
- How often is doc debt actually reviewed in practice? May need UX iteration.

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude (Opus)
**Workspace:** `.orch/workspace/og-feat-drift-prevention-auto-08jan-c18c/`
**Investigation:** `.kb/investigations/2026-01-08-inv-drift-prevention-auto-track-cli.md`
**Beads:** `bd show orch-go-edcy`
