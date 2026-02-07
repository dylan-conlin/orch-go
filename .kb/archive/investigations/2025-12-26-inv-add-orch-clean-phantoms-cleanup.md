<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The `orch clean --phantoms` flag is already fully implemented and working in the codebase.

**Evidence:** `orch clean --help` shows the --phantoms flag; `orch clean --phantoms --dry-run` successfully scans for phantom windows; tests pass (`TestCleanWorkspaceBased`, `TestCleanPreservesInProgressWorkspaces`).

**Knowledge:** The feature was implemented in `cmd/orch/main.go:3581-3673` via `cleanPhantomWindows()` function, which finds tmux windows with beads IDs but no active OpenCode session.

**Next:** Close - no work needed, feature already exists and is working.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Add Orch Clean Phantoms Cleanup

**Question:** Does `orch clean --phantoms` need to be implemented to cleanup phantom agents?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: --phantoms flag already registered in cleanCmd

**Evidence:** The flag is registered at `cmd/orch/main.go:3187`:
```go
cleanCmd.Flags().BoolVar(&cleanPhantoms, "phantoms", false, "Close all phantom tmux windows (stale agent windows)")
```

**Source:** `cmd/orch/main.go:3187`

**Significance:** The CLI interface for phantom cleanup already exists and is properly wired to the command.

---

### Finding 2: cleanPhantomWindows function fully implemented

**Evidence:** The function `cleanPhantomWindows()` at lines 3581-3673 implements the full phantom detection and cleanup logic:
- Gets all OpenCode sessions and builds a map of recently active beads IDs
- Scans all workers tmux sessions for windows with beads IDs
- Identifies phantom windows (beads ID in window name but no active OpenCode session)
- Closes phantom windows (or reports them in dry-run mode)

**Source:** `cmd/orch/main.go:3581-3673`

**Significance:** The core functionality is complete and handles both detection and cleanup properly.

---

### Finding 3: Command help shows --phantoms as available option

**Evidence:** Running `orch clean --help` shows:
```
--phantoms          Close all phantom tmux windows (stale agent windows)
```

**Source:** `orch clean --help` output

**Significance:** The feature is user-accessible and documented in the CLI.

---

### Finding 4: Dry-run test confirms feature works

**Evidence:** Running `orch clean --phantoms --dry-run` outputs:
```
Scanning for phantom tmux windows...
  Found 11 active OpenCode sessions
  No phantom windows found
```

**Source:** Command output from test run

**Significance:** The feature properly scans for phantoms and reports status.

---

## Synthesis

**Key Insights:**

1. **Feature already complete** - The `--phantoms` flag and its implementation (`cleanPhantomWindows()`) are already in the codebase and working.

2. **Implementation is robust** - The function properly handles active session detection, phantom identification, and both dry-run and real cleanup modes.

3. **Tests pass** - Related tests (`TestCleanWorkspaceBased`, `TestCleanPreservesInProgressWorkspaces`) verify the clean command behavior.

**Answer to Investigation Question:**

No implementation work is needed. The `orch clean --phantoms` feature is already fully implemented at `cmd/orch/main.go:3581-3673`. The feature:
- Is registered as a CLI flag
- Properly detects phantom windows (tmux windows with beads ID but no active OpenCode session)
- Supports both dry-run and actual cleanup modes
- Is documented in `--help` output

---

## Structured Uncertainty

**What's tested:**

- ✅ CLI flag registration works (verified: `orch clean --help` shows `--phantoms`)
- ✅ Dry-run mode works (verified: ran `orch clean --phantoms --dry-run`)
- ✅ Unit tests pass (verified: `go test ./cmd/orch/... -run "Clean"`)

**What's untested:**

- ⚠️ Actual phantom cleanup (no phantom windows existed during test)
- ⚠️ Edge cases with malformed window names

**What would change this:**

- Finding would be wrong if `orch clean --phantoms` fails when phantom windows exist
- Finding would be wrong if the flag was removed in a recent commit

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Close this issue - no work needed** - The feature is already fully implemented.

**Why this approach:**
- Feature exists and works (verified via CLI help and dry-run)
- Tests pass
- No implementation gaps found

**Success criteria:**
- ✅ `orch clean --phantoms` command exists (verified)
- ✅ Dry-run mode works (verified)
- ✅ Tests pass (verified)

---

## References

**Files Examined:**
- `cmd/orch/main.go:3144-3190` - cleanCmd definition and flag registration
- `cmd/orch/main.go:3341-3474` - runClean function
- `cmd/orch/main.go:3581-3673` - cleanPhantomWindows function
- `cmd/orch/clean_test.go` - Tests for clean command

**Commands Run:**
```bash
# Check current clean command options
orch clean --help

# Test phantom cleanup in dry-run mode
orch clean --phantoms --dry-run

# Run clean-related tests
go test ./cmd/orch/... -run "Clean" -v
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-19-inv-cli-orch-status-command.md` - Related status command investigation

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Does `orch clean --phantoms` need to be implemented?
- Context: Task spawned to add phantom cleanup feature

**2025-12-26:** Feature found to already exist
- Examined main.go, found cleanPhantomWindows() fully implemented
- Verified via CLI help and dry-run test

**2025-12-26:** Investigation completed
- Status: Complete
- Key outcome: No work needed - feature already fully implemented
