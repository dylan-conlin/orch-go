<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created pkg/shell package providing Runner interface, DefaultRunner with timeout/context support, and MockRunner for testing.

**Evidence:** 29 tests passing; all existing tests still pass; package provides Run(), Output(), RunWithStdin(), and Start() methods with functional options for dir, env, and timeout.

**Knowledge:** The 117 exec.Command usages follow common patterns: run with output capture, run with specific working directory, and run with custom environment. These are now abstracted.

**Next:** Migrate existing exec.Command usages to use pkg/shell (follow-up task for incremental adoption).

---

# Investigation: Create Shell Execution Abstraction Pkg

**Question:** How should we abstract the 117 exec.Command usages for better error handling, timeout support, and testability?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Claude (feature-impl agent)
**Phase:** Complete
**Next Step:** None - package created, tests passing
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** .kb/investigations/2025-12-30-audit-orch-go-comprehensive-codebase.md (Issue 2)
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Common exec.Command Patterns

**Evidence:** Analyzed 100+ exec.Command usages across codebase. Found these common patterns:
- Simple execution: `cmd := exec.Command(name, args...); output, err := cmd.Output()`
- With working directory: `cmd.Dir = "/path/to/dir"`
- With environment: `cmd.Env = append(os.Environ(), "VAR=value")`
- With context for timeout: `exec.CommandContext(ctx, name, args...)`
- With stdin: `cmd.Stdin = reader`

**Source:** 
- pkg/beads/cli_client.go:64 - wraps exec.Command with Dir and Env
- pkg/tmux/tmux.go - multiple tmux commands with specific working directories
- pkg/spawn/kbcontext.go:192-194 - uses exec.CommandContext for timeout
- pkg/verify/git_commits.go:83 - simple git log execution

**Significance:** All patterns can be unified under a single Runner interface with functional options.

---

### Finding 2: CLIClient Already Has Good Abstraction Pattern

**Evidence:** pkg/beads/cli_client.go already implements a helper method `bdCommand()` that:
- Creates exec.Command with configured path
- Sets working directory
- Sets environment

**Source:** pkg/beads/cli_client.go:62-74

**Significance:** This pattern validates the approach - a central helper that configures commands consistently. pkg/shell generalizes this to the entire codebase.

---

### Finding 3: Timeout Support Missing in Most Usages

**Evidence:** Most exec.Command usages don't use context-based timeouts. Only pkg/spawn/kbcontext.go uses exec.CommandContext for kb commands.

**Source:** Grep for exec.Command vs exec.CommandContext shows ~95% use plain exec.Command

**Significance:** The new shell.Runner with WithTimeout option enables consistent timeout behavior across all command executions.

---

## Synthesis

**Key Insights:**

1. **Interface-based design enables testing** - The Runner interface allows MockRunner injection, enabling unit tests without actual command execution.

2. **Functional options provide flexibility** - WithDir, WithEnv, WithTimeout can be combined as needed without complex constructor signatures.

3. **Exit code visibility** - ExitError type exposes exit codes and stderr for better error handling, compared to raw exec.ExitError.

**Answer to Investigation Question:**

The shell package provides a clean abstraction via the Runner interface:
- `Run()` - combined stdout/stderr output
- `Output()` - stdout only
- `RunWithStdin()` - with stdin input
- `Start()` - async command execution

DefaultRunner wraps exec.Command with configurable working directory, environment, and timeout. MockRunner enables testing without actual command execution.

---

## Structured Uncertainty

**What's tested:**

- ✅ Run/Output/RunWithStdin methods work (verified: 29 tests passing)
- ✅ WithDir sets working directory (verified: TestDefaultRunner_WithDir)
- ✅ WithEnv sets environment (verified: TestDefaultRunner_WithEnv)
- ✅ WithTimeout enforces timeout (verified: TestDefaultRunner_WithTimeout)
- ✅ ExitError captures exit code (verified: TestDefaultRunner_ExitError)
- ✅ MockRunner records calls (verified: TestMockRunner_RecordsCalls)

**What's untested:**

- ⚠️ Migration of existing exec.Command usages (not in scope - follow-up task)
- ⚠️ Performance impact of abstraction layer (not benchmarked)
- ⚠️ Windows compatibility (tested on macOS only)

**What would change this:**

- If existing code relies on specific exec.Cmd fields not exposed by Runner interface
- If performance overhead is significant for high-frequency command execution

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Incremental Migration** - Migrate exec.Command usages to shell.Runner package-by-package.

**Why this approach:**
- Lower risk - each migration is isolated
- Enables testing improvements as we go
- Doesn't require big-bang refactoring

**Trade-offs accepted:**
- Some time before full migration is complete
- Inconsistency during transition period

**Implementation sequence:**
1. [DONE] Create pkg/shell with Runner, DefaultRunner, MockRunner
2. [NEXT] Migrate pkg/beads/cli_client.go to use shell.Runner
3. [LATER] Migrate pkg/tmux, pkg/verify, etc.

### Alternative Approaches Considered

**Option B: Global exec.Command replacement via linker**
- **Pros:** Automatic, no code changes
- **Cons:** Doesn't improve testability, hacky
- **When to use instead:** Never

**Option C: Do nothing, keep scattered exec.Command**
- **Pros:** No work
- **Cons:** Continues tech debt, hard to test
- **When to use instead:** If resources severely constrained

**Rationale for recommendation:** Incremental migration balances risk, enables testing improvements, and builds momentum.

---

### Implementation Details

**What to implement first:**
- [DONE] pkg/shell package with interface and implementations
- [DONE] Comprehensive tests for the new package

**Things to watch out for:**
- ⚠️ Some usages may need additional exec.Cmd fields (Stdout, Stderr as io.Writer)
- ⚠️ Start() is less common but needed for async processes

**Areas needing further investigation:**
- Which packages would benefit most from MockRunner injection?
- Should we add logging/metrics to DefaultRunner?

**Success criteria:**
- ✅ pkg/shell tests all passing (DONE - 29 tests)
- ✅ All existing tests still passing (DONE)
- ✅ Runner interface covers common patterns (DONE)

---

## References

**Files Examined:**
- pkg/beads/cli_client.go - existing command abstraction pattern
- pkg/tmux/tmux.go - heavy exec.Command usage for tmux
- pkg/spawn/kbcontext.go - uses CommandContext with timeout
- pkg/verify/git_commits.go - simple git log execution

**Commands Run:**
```bash
# Pattern search
grep "exec\.Command" --include="*.go"

# Test execution
go test ./pkg/shell/... -v

# Full test suite
go test ./...
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-30-audit-orch-go-comprehensive-codebase.md - Identified Issue 2

---

## Investigation History

**2026-01-01 00:00:** Investigation started
- Initial question: How to abstract 117 exec.Command usages
- Context: Spawned from comprehensive codebase audit Issue 2

**2026-01-01 00:15:** Pattern analysis completed
- Identified 4 main patterns: basic, with dir, with env, with context
- Designed Runner interface

**2026-01-01 00:30:** Implementation completed
- Created pkg/shell/shell.go with DefaultRunner
- Created pkg/shell/mock.go with MockRunner
- Created pkg/shell/shell_test.go and mock_test.go with 29 tests

**2026-01-01 00:45:** Investigation completed
- Status: Complete
- Key outcome: pkg/shell package created with full test coverage
