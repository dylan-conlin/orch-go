## Summary (D.E.K.N.)

**Delta:** Implemented auto-detection of new CLI commands in `orch complete` - when agents add new cobra commands to cmd/orch/*.go, the system now surfaces a prompt recommending skill documentation updates.

**Evidence:** Code compiles successfully, unit tests pass (TestNewCLICommandContentDetection, TestDetectNewCLICommandsGitStatusParsing), and integration follows existing hasGoChangesInRecentCommits pattern.

**Knowledge:** New CLI commands are identifiable by: (1) being Added (not Modified) in git, (2) located in cmd/orch/*.go (not _test.go), (3) containing both `cobra.Command{` and `rootCmd.AddCommand(`.

**Next:** Close - implementation complete. Watch for the prompt during next completion that includes new command files.

---

# Investigation: Auto Detect New CLI Commands

**Question:** How can we automatically detect when agents add new CLI commands and prompt for skill documentation updates?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent og-feat-auto-detect-new-26dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing Pattern for Go Change Detection

**Evidence:** `hasGoChangesInRecentCommits` function at main.go:3017 provides a template for detecting file changes using `git diff --name-only HEAD~5..HEAD`.

**Source:** cmd/orch/main.go:3017-3049

**Significance:** Established pattern for git-based file change detection that can be extended for new file detection.

---

### Finding 2: Cobra Command File Patterns

**Evidence:** New CLI commands follow consistent patterns:
- Files in cmd/orch/*.go contain `var xxxCmd = &cobra.Command{` declarations
- Commands are registered via `rootCmd.AddCommand(xxxCmd)` in init() functions
- Example files: doctor.go (line 20, 44), fetchmd.go (line 27, 58)

**Source:** cmd/orch/doctor.go:20-44, cmd/orch/fetchmd.go:27-58

**Significance:** Two-part detection is reliable: file must contain both cobra.Command definition AND AddCommand registration.

---

### Finding 3: Git Status Distinguishes Added vs Modified

**Evidence:** `git diff --name-status` provides status codes: A (Added), M (Modified), D (Deleted). This distinguishes truly new files from modifications to existing commands.

**Source:** Git documentation, tested manually with `git diff --name-status HEAD~10..HEAD`

**Significance:** Using --name-status instead of --name-only enables precise detection of only NEW command files, avoiding false positives from command modifications.

---

## Synthesis

**Key Insights:**

1. **Detection requires git + content analysis** - Git tells us which files are new, content analysis confirms they're cobra commands.

2. **Integration point is post-auto-rebuild** - The natural place to surface the prompt is immediately after the auto-rebuild that was triggered by Go changes.

3. **Prompt is informational, not blocking** - The prompt recommends documentation updates but doesn't gate completion.

**Answer to Investigation Question:**

Detection is implemented via `detectNewCLICommands(projectDir string) []string` which:
1. Uses `git diff --name-status HEAD~5..HEAD` to find Added files
2. Filters to cmd/orch/*.go (excluding _test.go)
3. Reads file content to verify cobra.Command + rootCmd.AddCommand patterns
4. Returns list of new command filenames

When detected, a prominent box is displayed recommending skill documentation updates.

---

## Structured Uncertainty

**What's tested:**

- ✅ Content detection logic correctly identifies cobra commands (TestNewCLICommandContentDetection - 5 cases)
- ✅ Git status parsing correctly identifies added files (TestDetectNewCLICommandsGitStatusParsing - 6 cases)
- ✅ Code compiles without errors

**What's untested:**

- ⚠️ End-to-end flow (requires actual git history with added command file)
- ⚠️ Display formatting on different terminal widths

**What would change this:**

- If cobra command registration patterns change (e.g., subcommand-only files)
- If git diff output format changes in future git versions

---

## Implementation Recommendations

### Recommended Approach ⭐

**Detect in orch complete** - Check for new CLI commands after auto-rebuild and display informational prompt.

**Why this approach:**
- Natural integration point (already running post-commit checks)
- Non-blocking (doesn't gate completion)
- Timely (surfaces immediately when relevant)

**Trade-offs accepted:**
- Only detects at completion time (not during implementation)
- Requires git history (won't work with squashed commits)

**Implementation sequence:**
1. Add `detectNewCLICommands()` function - uses git + content analysis
2. Integrate into `runComplete()` after auto-rebuild block
3. Add unit tests for detection logic

### Alternative Approaches Considered

**Option B: SYNTHESIS.md template guidance**
- **Pros:** Zero code changes, agents self-report
- **Cons:** Relies on agent compliance, easily forgotten
- **When to use instead:** If detection logic proves unreliable

**Option C: Post-completion hook**
- **Pros:** Cleaner separation of concerns
- **Cons:** Hook infrastructure doesn't exist yet, more complex
- **When to use instead:** If more post-completion actions are needed

**Rationale for recommendation:** Option A is simplest to implement, integrates with existing patterns, and provides immediate value without new infrastructure.

---

## References

**Files Examined:**
- cmd/orch/main.go - hasGoChangesInRecentCommits pattern, runComplete integration point
- cmd/orch/doctor.go - cobra command file pattern example
- cmd/orch/fetchmd.go - cobra command file pattern example

**Files Modified:**
- cmd/orch/main.go:3017-3118 - Added detectNewCLICommands function and integration
- cmd/orch/main_test.go:1553-1690 - Added unit tests

**Commands Run:**
```bash
# Verify compilation
go build ./cmd/orch/

# Run unit tests
go test ./cmd/orch/... -run "TestNewCLICommandContentDetection" -v
go test ./cmd/orch/... -run "TestDetectNewCLICommandsGitStatusParsing" -v

# Check recent cmd/orch changes
git diff --name-status HEAD~10..HEAD | grep cmd/orch
```

---

## Investigation History

**2025-12-26 10:30:** Investigation started
- Initial question: How to auto-detect new CLI commands for skill documentation
- Context: orch doctor and orch fetch-md were added without skill updates

**2025-12-26 10:45:** Implementation complete
- Added detectNewCLICommands function
- Integrated into orch complete flow
- Added unit tests

**2025-12-26 11:00:** Investigation completed
- Status: Complete
- Key outcome: Auto-detection of new CLI commands implemented and tested
