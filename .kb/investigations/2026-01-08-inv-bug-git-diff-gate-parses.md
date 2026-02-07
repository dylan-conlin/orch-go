<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed bug where `isLikelyFilePath` matched event type names like `session.created` as file paths.

**Evidence:** Test suite expanded from 13 to 35 test cases; all pass including new false positive scenarios.

**Knowledge:** File paths should be validated by known file extensions, not just presence of a dot; event type names use semantic dot-separated names that don't match known extensions.

**Next:** Close - fix implemented and tested.

**Promote to Decision:** recommend-no - Tactical bug fix with clear solution, not architectural.

---

# Investigation: Bug Git Diff Gate Parses

**Question:** Why does the git_diff gate incorrectly parse event type names like `session.created` as file paths?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Claude Agent (og-arch-bug-git-diff-08jan-0ea6)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Root cause - naive dot detection

**Evidence:** The original `isLikelyFilePath` function only checked for presence of a dot:
```go
// Must have a file extension
if !strings.Contains(s, ".") {
    return false
}
```

This meant ANY string with a dot was considered a potential file path.

**Source:** `pkg/verify/git_diff.go:80-82` (original code)

**Significance:** Event type names like `session.created`, `agent.spawned` all contain dots, making them false positives.

---

### Finding 2: Known file extensions provide the solution

**Evidence:** Real file paths end with recognizable extensions like `.go`, `.md`, `.yaml`, `.json`, etc. Event type names end with semantic words like `.created`, `.spawned`, `.completed` which are NOT file extensions.

**Source:** Analysis of SYNTHESIS.md files in `.orch/workspace/*/SYNTHESIS.md`

**Significance:** By validating against a list of known file extensions, we can distinguish real files from event type names.

---

### Finding 3: Multiple false positive patterns exist

**Evidence:** Testing revealed several false positive patterns:
- Event types: `session.created`, `agent.spawned`, `task.completed`
- Version numbers: `v0.33.2`, `1.2.3`
- Assignments: `hasCodeChanges=true`

**Source:** `pkg/verify/git_diff_test.go` - expanded test cases

**Significance:** The fix needs to handle all these patterns, not just event types.

---

## Synthesis

**Key Insights:**

1. **Known extensions are the key heuristic** - Real file paths use a finite set of known extensions (.go, .md, .yaml, etc.) while event types use semantic names.

2. **Dotfiles need special handling** - Files starting with `.` (like `.gitignore`, `.env`) don't follow the extension pattern and should always be treated as valid paths.

3. **Version numbers follow a predictable pattern** - Multiple dots with numeric segments (v0.33.2) can be detected and excluded.

**Answer to Investigation Question:**

The git_diff gate incorrectly parsed event type names because `isLikelyFilePath` used a naive heuristic that only checked for the presence of a dot. The fix validates the final extension against a comprehensive list of known file extensions (50+ extensions covering code, config, data, and documentation files). This correctly identifies `session.created` as NOT a file path while preserving detection of real files like `main.go`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Event type names are rejected (verified: `session.created`, `agent.spawned` return false)
- ✅ Version numbers are rejected (verified: `v0.33.2`, `1.2.3` return false)
- ✅ Real file paths are accepted (verified: `main.go`, `pkg/verify/check.go` return true)
- ✅ Dotfiles are accepted (verified: `.gitignore`, `.env`, `.beads/beads.db` return true)
- ✅ Full test suite passes (35 test cases)

**What's untested:**

- ⚠️ Rare file extensions not in the known list (may reject legitimate files)
- ⚠️ Performance with very large SYNTHESIS.md files (not benchmarked)

**What would change this:**

- Finding would be wrong if a legitimate file extension is missing from the known list
- Alternative approach if performance becomes an issue with regex matching

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Known File Extension Validation** - Validate the final extension against a list of known file extensions.

**Why this approach:**
- Clear semantic distinction between files and event types
- Extensible - new extensions can be added to the list
- No regex complexity - simple string matching

**Trade-offs accepted:**
- Must maintain a list of known extensions
- Rare/custom extensions might be missed

**Implementation sequence:**
1. Add `knownFileExtensions` map with 50+ common extensions
2. Modify `isLikelyFilePath` to check extension against map
3. Add special handling for dotfiles (always valid)
4. Add `isVersionNumber` helper for version detection

### Alternative Approaches Considered

**Option B: Path separator requirement (contain / or start with .)**
- **Pros:** Simpler, no extension list needed
- **Cons:** Would reject valid root files like `main.go`
- **When to use instead:** If we only care about files in subdirectories

**Option C: Regex-based event type detection**
- **Pros:** Could catch more patterns
- **Cons:** Harder to maintain, regex complexity
- **When to use instead:** If pattern-based exclusion is needed

**Rationale for recommendation:** Option A (known extensions) balances precision with maintainability and handles all identified false positive cases.

---

### Implementation Details

**What was implemented:**
- `knownFileExtensions` map with 50+ extensions
- Updated `isLikelyFilePath` with extension validation
- `isVersionNumber` helper function
- Expanded test coverage (35 test cases)

**Files Modified:**
- `pkg/verify/git_diff.go` - Core fix
- `pkg/verify/git_diff_test.go` - Test coverage

**Success criteria:**
- ✅ `session.created` returns false (event type)
- ✅ `main.go` returns true (valid file)
- ✅ Full test suite passes

---

## References

**Files Examined:**
- `pkg/verify/git_diff.go` - Main implementation
- `pkg/verify/git_diff_test.go` - Test cases
- `.orch/workspace/*/SYNTHESIS.md` - Example false positives

**Commands Run:**
```bash
# Run tests
go test ./pkg/verify/... -count=1

# Test specific patterns
go test -v ./pkg/verify/... -run TestIsLikelyFilePath
```

---

## Investigation History

**2026-01-08 ~12:00:** Investigation started
- Initial question: Why does git_diff gate parse event type names as file paths?
- Context: Spawned from beads issue orch-go-7lvi2

**2026-01-08 ~12:15:** Root cause identified
- Naive dot detection in `isLikelyFilePath`
- Solution: known file extension validation

**2026-01-08 ~12:30:** Fix implemented and tested
- Status: Complete
- Key outcome: 35 test cases pass, false positives eliminated
