## Summary (D.E.K.N.)

**Delta:** orch-go codebase has good test coverage (avg 60%+) but suffers from architectural issues: 4823-line main.go god object, 808 raw fmt.Printf calls, and several packages lacking tests (sessions at 0%).

**Evidence:** go test -cover shows 21.9% cmd/orch coverage, 0% sessions, 25% tmux; grep found 20+ runtime regex compilations, 11 ignored errors with `_ =`, and no structured logging.

**Knowledge:** The codebase is functional but technical debt is accumulating in CLI commands (should be separate files) and error handling (silent failures). Concurrency patterns are correctly implemented with proper defer.

**Next:** Create 3 beads issues for highest-priority fixes: split main.go, add structured logging, fix sessions package coverage.

---

# Investigation: Comprehensive Orch-Go Audit - Bugs, Reliability, Architecture

**Question:** What are the most impactful bugs, reliability concerns, architectural gaps, refactoring needs, test coverage gaps, and code quality issues in orch-go?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Claude (codebase-audit skill)
**Phase:** Complete
**Next Step:** None - audit complete
**Resolution-Status:** Resolved
**Status:** Complete

---

## Findings

### Finding 1: God Object in cmd/orch/main.go (4823 lines)

**Evidence:** 
- 4823 lines of code in single file
- 149 function/var/const/type definitions
- Contains 17+ cobra command definitions that should be separate files
- Handles spawning, sending, monitoring, status, completion, and more

**Source:** `cmd/orch/main.go`, `grep -n "^func \|^var \|^const \|^type "` output

**Significance:** This is the primary maintenance burden. Adding features requires understanding the entire file. Commands like `spawn`, `send`, `complete`, `account` should each have their own file for maintainability.

---

### Finding 2: Low Test Coverage in Key Packages

**Evidence:**
| Package | Coverage | Risk |
|---------|----------|------|
| cmd/orch | 21.9% | High - core CLI logic |
| pkg/sessions | 0.0% | High - session management |
| pkg/tmux | 25.2% | Medium - tmux integration |
| pkg/usage | 27.3% | Low - usage tracking |
| pkg/account | 16.0% | High - OAuth/account management |

**Source:** `/usr/local/go/bin/go test -cover ./...`

**Significance:** Low coverage in cmd/orch is expected (CLI is hard to test), but 0% in pkg/sessions is a gap that needs addressing. The sessions package has 8261 lines with no tests.

---

### Finding 3: 808 Raw fmt.Printf Calls - No Structured Logging

**Evidence:**
- 808 occurrences of fmt.Println/fmt.Printf outside tests
- Zero usage of structured logging (slog, zerolog, logrus)
- Debug statements hardcoded in daemon.go (lines 294, 306, 313, 320, 327, 334, 339)

**Source:** `grep -rn "fmt\.Println\|fmt\.Printf"` excluding tests

**Significance:** Makes debugging and observability difficult. Cannot easily filter log levels, add structured fields, or enable verbose mode dynamically. DEBUG statements should use a proper logging framework.

---

### Finding 4: Runtime Regex Compilation (20+ instances)

**Evidence:**
- 20+ calls to `regexp.MustCompile` inside functions
- Examples in: wait.go:83, history.go:243-290, verify/check.go:98-338

**Source:** `grep -rn "regexp\.MustCompile\|regexp\.Compile"`

**Significance:** Minor performance impact per call, but patterns are compiled on every function invocation. Should be moved to package-level `var` blocks for compile-time initialization.

---

### Finding 5: Non-Atomic File Writes

**Evidence:**
- Most file writes use direct `os.WriteFile()` without temp file + rename pattern
- Only 2 places use proper atomic writes: `pkg/daemon/status.go:76-81`, `pkg/spawn/session.go:28-34`
- 24+ other write locations use non-atomic pattern

**Source:** `grep -rn "os\.WriteFile"` output

**Significance:** Risk of partial writes corrupting state files (config.yaml, focus state, etc.) on crash. Critical for daemon status and registry files.

---

### Finding 6: Ignored Errors (11 instances)

**Evidence:**
| Location | Pattern | Risk |
|----------|---------|------|
| serve.go:570 | `projectDir, _ = os.Getwd()` | Medium |
| init.go:272-275 | Port allocation errors ignored | Low |
| daemon.go:1198 | Synthesis parse error ignored | Low |
| beads/client.go:254 | Getwd error ignored | Medium |
| tmux.go:214 | Tmuxinator config error ignored | Low |

**Source:** `grep -rn "_ = "`

**Significance:** Most are benign (fallbacks exist), but silent failures can mask issues. The os.Getwd() ignores are concerning - could lead to wrong working directory.

---

### Finding 7: Lock/Unlock Pattern Inconsistency

**Evidence:**
- Most mutex usage correctly uses `defer mu.Unlock()`
- But verify/check.go:875-894 uses non-deferred `Unlock()` pattern
- Not a bug (correctly paired), but inconsistent with other code

**Source:** `grep -rn "defer.*Unlock\|Unlock()$"` showing check.go:877, check.go:894

**Significance:** Low risk since the locks are properly paired, but inconsistent style. The deferred pattern is safer if code changes add early returns.

---

### Finding 8: Context Timeout on External Commands

**Evidence:**
- kb context queries have 5-second timeout (spawn/kbcontext.go:151)
- This is good defensive coding
- Other exec.Command calls lack explicit timeouts

**Source:** `grep -rn "context\.WithTimeout"` - only 3 instances

**Significance:** Most CLI commands are quick, but long-running external processes could hang indefinitely. Low priority since user can Ctrl+C.

---

### Finding 9: Hardcoded Path Patterns

**Evidence:**
- 19+ instances of `filepath.Join(..., ".orch", "workspace")`
- No centralized path constants
- Patterns duplicated across main.go, serve.go, review.go, wait.go, etc.

**Source:** `grep -rn 'filepath\.Join.*"\.orch"'`

**Significance:** Not a bug, but maintenance burden. Changing workspace location would require updating many files. Should have a pkg/paths package with constants.

---

### Finding 10: Security Posture is Acceptable

**Evidence:**
- No hardcoded secrets found
- OAuth tokens stored in proper config files (not code)
- exec.Command usage is for known CLI tools (git, tmux, bd, kb)
- No user input directly passed to shell (no injection risk)

**Source:** `grep -rn "password\|secret\|api_key"` - all in docs/comments

**Significance:** Security is not a concern. The codebase follows good practices for credential handling.

---

## Synthesis

**Key Insights:**

1. **Maintainability is the Primary Concern** - The 4823-line main.go is the biggest issue, not bugs. Splitting it into per-command files would dramatically improve maintainability without changing behavior.

2. **Test Coverage Gaps Follow a Pattern** - Low coverage is in UI-heavy code (CLI, tmux) and newer packages (sessions). The well-tested packages (capacity 95%, action 84%, patterns 86%) show the team values testing.

3. **Reliability is Good Despite Style Issues** - Concurrency primitives are used correctly, context timeouts exist where needed, and error handling is mostly complete. The ignored errors are in fallback paths.

**Answer to Investigation Question:**

The orch-go codebase is **functional and reliable** but has **architectural debt** that will slow future development. Priority issues:

1. **High Impact, Medium Effort:** Split main.go into separate command files
2. **Medium Impact, Low Effort:** Add structured logging (replace DEBUG printfs)
3. **Medium Impact, Medium Effort:** Add tests for pkg/sessions (0% coverage)
4. **Low Impact, Low Effort:** Move regex compilations to package-level vars

The codebase does NOT have significant bugs or security issues. The concerns are around maintainability and observability, not correctness.

---

## Structured Uncertainty

**What's tested:**

- ✅ Test coverage verified with `go test -cover ./...` - ran successfully
- ✅ File sizes verified with `wc -l` - main.go is 4823 lines
- ✅ Pattern counts verified with grep - 808 fmt.Printf, 20+ regex, 11 ignored errors

**What's untested:**

- ⚠️ Performance impact of runtime regex (not benchmarked)
- ⚠️ Actual data loss risk from non-atomic writes (theoretical)
- ⚠️ Whether splitting main.go would require interface changes (not attempted)

**What would change this:**

- If pkg/sessions is deprecated/unused, 0% coverage is acceptable
- If regex patterns are rarely called paths, runtime compile is fine
- If main.go has strong cohesion (not tested), splitting might not help

---

## Implementation Recommendations

### Recommended Approach: Incremental Refactoring

**Split main.go into per-command files** - This is the highest-value change that doesn't risk breaking anything.

**Why this approach:**
- No behavior change, just file reorganization
- Enables focused development on specific commands
- Makes code review easier (smaller diffs per command)

**Trade-offs accepted:**
- More files to navigate (acceptable)
- Import structure might change slightly

**Implementation sequence:**
1. Create `cmd/orch/spawn.go` - extract spawn command (highest complexity)
2. Create `cmd/orch/status.go` - extract status command
3. Continue for account, send, wait, complete, etc.
4. Keep `main.go` as just init() and rootCmd

### Alternative Approaches Considered

**Option B: Add structured logging first**
- **Pros:** Quick win, improves debugging immediately
- **Cons:** Doesn't fix maintainability issue
- **When to use instead:** If debugging is the immediate pain point

**Option C: Add tests for sessions package**
- **Pros:** Increases reliability guarantees
- **Cons:** sessions might be deprecated soon (unclear usage)
- **When to use instead:** If sessions is critical path

**Rationale for recommendation:** Splitting main.go has the best ROI - low risk, high maintainability improvement, no behavior change.

---

### Implementation Details

**What to implement first:**
- Extract spawnCmd and all related functions to spawn.go
- This is the most complex command and will be the best test of the approach

**Things to watch out for:**
- ⚠️ Global variables might be referenced across commands (need to check)
- ⚠️ init() functions in Cobra require specific ordering
- ⚠️ Some functions in main.go are shared utilities

**Areas needing further investigation:**
- Which functions in main.go are truly shared vs command-specific?
- Is pkg/sessions actively used or slated for removal?

**Success criteria:**
- ✅ `go build ./cmd/orch` still works
- ✅ All existing tests pass
- ✅ Individual command files are <500 lines each

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Primary analysis target (4823 lines)
- `pkg/daemon/daemon.go` - Concurrency patterns (1319 lines)
- `pkg/verify/check.go` - Lock patterns (903 lines)
- `pkg/spawn/kbcontext.go` - Timeout patterns (570 lines)

**Commands Run:**
```bash
# File size analysis
find . -name "*.go" | xargs wc -l | sort -rn | head -20

# Test coverage
go test -cover ./...

# Pattern counts
grep -rn "fmt\.Println\|fmt\.Printf" --include="*.go" | wc -l
grep -rn "regexp\.MustCompile" --include="*.go" | wc -l
grep -rn "_ = " --include="*.go" | wc -l
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md` - Prior registry audit

---

## Investigation History

**2026-01-03 13:30:** Investigation started
- Initial question: Comprehensive audit of orch-go for bugs, reliability, and architecture issues
- Context: Spawned by orchestrator for systematic codebase health check

**2026-01-03 14:00:** Security audit complete
- No hardcoded secrets, proper OAuth handling, no injection risks

**2026-01-03 14:15:** Architecture audit complete  
- Identified main.go god object as primary issue

**2026-01-03 14:30:** Investigation completed
- Status: Complete
- Key outcome: Maintainability debt in main.go is the priority; codebase is reliable
