**TLDR:** Question: Implement Go CLI orch complete command with Phase verification and beads integration. Answer: Successfully implemented `pkg/verify` for phase parsing and `orch complete` command with --force and --reason flags. High confidence (90%) - all tests pass, validated against beads CLI integration.

---

# Investigation: CLI orch complete Command Implementation

**Question:** How to implement the orch complete command in Go with Phase: Complete verification and beads issue closure?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: bd CLI provides JSON output for programmatic access

**Evidence:** The `bd` CLI supports `--json` flag for structured output:
- `bd show <id> --json` returns issue details as JSON array
- `bd comments <id> --json` returns comments as JSON array (or null if none)
- `bd close <id> --reason <text>` closes issues with a reason

**Source:** `bd --help`, `bd comments --help`, `bd close --help`

**Significance:** Enables reliable parsing of beads data without complex text parsing. The JSON output is stable and machine-readable.

---

### Finding 2: Phase status pattern follows "Phase: <status> - <summary>" convention

**Evidence:** From Python reference `src/orch/complete.py` and spawn protocols, agents report phase via:
- `bd comment <id> "Phase: Planning - description"`
- `bd comment <id> "Phase: Implementing - description"`
- `bd comment <id> "Phase: Complete - summary of deliverables"`

The pattern supports multiple dash types (hyphen, en-dash, em-dash) for robustness.

**Source:** `orch-cli/src/orch/complete.py:39-43`, SPAWN_CONTEXT.md templates

**Significance:** Pattern matching can reliably extract phase status from comment content. Using regex with case-insensitive matching handles variations.

---

### Finding 3: Verification workflow follows verify-then-close pattern

**Evidence:** From Python reference, the completion workflow:
1. Verify issue exists and is not closed
2. Check for "Phase: Complete" in comments
3. Close issue with reason (uses summary if available)
4. Log completion event

**Source:** `orch-cli/src/orch/complete.py:close_beads_issue`, `complete_agent_work`

**Significance:** Go implementation follows same pattern for consistency with existing orchestration workflows.

---

## Synthesis

**Key Insights:**

1. **bd CLI is the interface to beads** - Rather than direct database access, the complete command shells out to `bd` for all beads operations. This maintains loose coupling.

2. **Phase parsing is comment-based** - The latest Phase: comment determines the agent's current state. Multiple phases over time are expected (Planning -> Implementing -> Complete).

3. **Force flag bypasses verification** - When --force is used, we trust the orchestrator's judgment that the agent's work is complete even without Phase: Complete.

**Answer to Investigation Question:**

The orch complete command is implemented with:
- `pkg/verify/check.go` - Phase parsing, verification, and beads CLI integration
- `cmd/orch/main.go` - Complete command with --force and --reason flags

The command verifies Phase: Complete in beads comments before closing the issue. All components have tests.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

All tests pass. Implementation matches Python reference patterns. bd CLI integration verified with real beads installation.

**What's certain:**

- ✅ Phase parsing correctly handles various formats (hyphen, en-dash, em-dash)
- ✅ bd CLI JSON output parsing works for comments and issues
- ✅ Complete command integrates with existing CLI structure
- ✅ --force flag bypasses phase verification as expected

**What's uncertain:**

- ⚠️ Cross-repo beads operations not implemented (db_path parameter not ported)
- ⚠️ Session cleanup (tmux window closing) not implemented (deferred to phase 2)
- ⚠️ Transcript export not implemented (deferred to phase 2)

**What would increase confidence to Very High (95%+):**

- End-to-end test with actual agent completing a task
- Test with bd CLI failure scenarios
- Integration with orch spawn to verify full lifecycle

---

## Implementation Recommendations

**Purpose:** Implementation is complete. These are notes for future enhancements.

### Delivered Components

1. **pkg/verify/check.go** - Core verification logic:
   - `GetComments()` - Fetch comments from bd CLI
   - `ParsePhaseFromComments()` - Extract latest phase status
   - `VerifyCompletion()` - Full verification check
   - `CloseIssue()` - Close beads issue with reason
   - `GetIssue()` - Fetch issue details

2. **pkg/verify/check_test.go** - Comprehensive tests for phase parsing

3. **cmd/orch/main.go** - Complete command with flags:
   - `--force` / `-f` - Skip phase verification
   - `--reason` / `-r` - Custom close reason

### Future Enhancements

- Add cross-repo beads support (db_path parameter)
- Add session cleanup (tmux window closing)
- Add transcript export
- Add investigation file path verification

---

## References

**Files Examined:**
- `orch-cli/src/orch/complete.py` - Python completion logic
- `orch-cli/src/orch/end_commands.py` - End session workflow
- `.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md` - Spawn implementation reference

**Commands Run:**
```bash
# Test bd CLI interface
bd show orch-go-ph1.5 --json
bd comments orch-go-ph1.5 --json
bd close --help

# Build and test
make build
go test ./pkg/verify/... -v
./build/orch-go complete --help
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md` - Spawn command implementation
- **Investigation:** `.kb/investigations/2025-12-19-inv-cli-orch-status-command.md` - Status command implementation

---

## Investigation History

**2025-12-19:** Investigation started
- Initial question: Implement orch complete command in Go
- Context: Part of orch-go Phase 1.5

**2025-12-19:** Implementation complete
- Created pkg/verify with phase parsing and beads integration
- Added complete command to cmd/orch/main.go
- Final confidence: High (90%)
- Status: Complete
- Key outcome: orch complete command implemented with Phase verification and beads integration
