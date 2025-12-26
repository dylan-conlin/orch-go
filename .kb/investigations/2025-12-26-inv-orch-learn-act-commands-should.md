<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added shell-aware command parsing and validation to `orch learn act` to properly handle quoted strings in generated commands.

**Evidence:** All 45 tests pass including new tests for ParseShellCommand (13 cases), ValidateCommand (17 cases), and command generation validation (4 cases).

**Knowledge:** The original `strings.Fields` broke on quoted strings like `--reason "Used by: investigation. Occurred 5 times"` - shell-style parsing is required for commands with quoted arguments.

**Next:** None - implementation complete with tests.

**Confidence:** High (90%) - Comprehensive test coverage for all command patterns.

---

# Investigation: Orch Learn Act Commands Should Be Tested for Runnability

**Question:** How to ensure `orch learn act` commands are valid and runnable before executing them?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** og-feat-orch-learn-act-26dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: strings.Fields Breaks Quoted Arguments

**Evidence:** The original implementation used `strings.Fields(s.Command)` which splits on whitespace without respecting quotes. A command like:
```
kn decide "auth" --reason "Used by: investigation. Occurred 5 times"
```
Would be split into: `["kn", "decide", "\"auth\"", "--reason", "\"Used", "by:", "investigation.", ...]`

**Source:** `cmd/orch/learn.go:341-347` (original implementation)

**Significance:** Generated commands were unrunnable because arguments with spaces would be incorrectly split.

---

### Finding 2: Four Command Patterns Are Generated

**Evidence:** The `determineSuggestion` function generates four distinct command patterns:
1. `kn decide "query" --reason "reason"` - for no_context gaps
2. `kn constrain "query" --reason "reason"` - for no_constraints gaps  
3. `bd create "title" -d "description"` - for no_decisions gaps
4. `orch spawn investigation "task"` - for default/sparse gaps

**Source:** `pkg/spawn/learning.go:396-420`

**Significance:** All patterns use quoted strings with potentially complex content (colons, commas, periods). All require shell-aware parsing.

---

### Finding 3: Generated Reasons Contain Complex Content

**Evidence:** The `generateReasonFromGaps` function creates reason strings like:
```
"Used by: investigation, feature-impl. Occurred 5 times. Tasks: analyze auth flow"
```

**Source:** `pkg/spawn/learning.go:423-482`

**Significance:** These complex strings with colons, commas, and periods must be preserved as single arguments when passed to the command.

---

## Synthesis

**Key Insights:**

1. **Shell parsing is essential** - Any command with quoted arguments needs proper shell-style parsing that respects quotes as argument delimiters.

2. **Validation prevents runtime errors** - Validating command structure before execution catches malformed commands early.

3. **Test coverage ensures correctness** - Testing that generated commands are valid ensures the full pipeline works end-to-end.

**Answer to Investigation Question:**

Commands should be tested for runnability by:
1. Parsing with shell-aware `ParseShellCommand` instead of `strings.Fields`
2. Validating structure with `ValidateCommand` before execution
3. Testing that `determineSuggestion` always produces valid commands

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Comprehensive test coverage for all command patterns and parsing edge cases. The implementation follows standard shell parsing conventions.

**What's certain:**

- ✅ ParseShellCommand correctly handles double and single quotes
- ✅ All four generated command patterns produce valid, parseable commands
- ✅ ValidateCommand catches common errors (missing args, unterminated quotes)
- ✅ All 45 tests pass including new tests

**What's uncertain:**

- ⚠️ Escaped quotes within strings (not currently needed, but could arise)
- ⚠️ Unicode or special characters in queries/tasks

**What would increase confidence to Very High (95%+):**

- Run actual commands in integration test
- Test with real gap data from production tracker

---

## Implementation Recommendations

**Purpose:** Document what was implemented.

### Recommended Approach ⭐

**Shell-aware parsing with validation** - Added ParseShellCommand and ValidateCommand functions.

**Why this approach:**
- Handles all existing command patterns correctly
- Validates before execution to prevent runtime failures
- Comprehensive test coverage ensures correctness

**Implementation sequence:**
1. Added `ParseShellCommand` function to handle quoted strings (pkg/spawn/learning.go)
2. Added `ValidateCommand` function to check command structure (pkg/spawn/learning.go)
3. Updated `runLearnAct` to use new parsing/validation (cmd/orch/learn.go)
4. Added comprehensive tests (pkg/spawn/learning_test.go)

### What was implemented:

**pkg/spawn/learning.go:**
- `ParseShellCommand(cmdStr string) ([]string, error)` - Shell-style argument parsing
- `ValidateCommand(cmdStr string) error` - Command structure validation
- `validateKnCommand`, `validateBdCommand`, `validateOrchCommand` - Per-tool validation

**cmd/orch/learn.go:**
- Updated `runLearnAct` to call ValidateCommand before running
- Replaced `strings.Fields` with `ParseShellCommand`

**pkg/spawn/learning_test.go:**
- `TestParseShellCommand` - 13 test cases for parsing
- `TestValidateCommand` - 17 test cases for validation  
- `TestDetermineSuggestionGeneratesValidCommands` - 4 test cases for generation

---

## References

**Files Examined:**
- `cmd/orch/learn.go` - CLI command implementation
- `pkg/spawn/learning.go` - Learning system core
- `pkg/spawn/learning_test.go` - Existing tests

**Commands Run:**
```bash
# Run new tests
go test ./pkg/spawn/... -v -run "Parse|Validate|DetermineSuggestion"

# Full test suite
go test ./...

# Build verification
go build ./...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-inv-fix-orch-learn-act-generate.md` - Prior fix for reason generation

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: How to ensure orch learn act commands are valid?
- Context: Commands with quoted strings were failing to execute

**2025-12-26:** Implementation complete
- Added ParseShellCommand and ValidateCommand functions
- Updated runLearnAct to use new parsing
- Added comprehensive tests (34 new test cases)
- Final confidence: High (90%)
- Status: Complete
- Key outcome: orch learn act now properly handles quoted strings in commands
