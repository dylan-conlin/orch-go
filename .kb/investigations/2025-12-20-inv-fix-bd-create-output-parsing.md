**TLDR:** Question: Why does bd create output parsing capture 'open' instead of the issue ID? Answer: Parser splits entire multi-line output by spaces and takes last word, which is "open" from "Status: open" on the last line, instead of parsing the issue ID from the first line. Very High confidence (99%) - reproduced issue and confirmed actual output format.

---

# Investigation: Fix bd create output parsing

**Question:** Why does the createBeadsIssue function capture 'open' instead of the actual issue ID when parsing `bd create` output?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Claude (systematic-debugging)
**Phase:** Complete
**Next Step:** Implement fix
**Status:** Complete
**Confidence:** Very High (99%)

---

## Findings

### Finding 1: Actual bd create output is multi-line with status on last line

**Evidence:** Running `bd create "test issue for parsing"` produces:
```
✓ Created issue: orch-go-5z9
  Title: test issue for parsing
  Priority: P2
  Status: open
```

**Source:** cmd/orch/main.go:386-391, manual test via `bd create "test issue for parsing"`

**Significance:** The output has 4 lines, with "open" being the last word on the last line. Current parser splits all text by spaces and takes the last word, which captures "open" instead of "orch-go-5z9".

---

### Finding 2: Parser assumes single-line output format

**Evidence:** Code at cmd/orch/main.go:386-391:
```go
// Parse issue ID from output (expected format: "Created issue: proj-123")
outputStr := strings.TrimSpace(string(output))
parts := strings.Split(outputStr, " ")
if len(parts) > 0 {
    // Take the last word which should be the issue ID
    return parts[len(parts)-1], nil
}
```

**Source:** cmd/orch/main.go:386-391

**Significance:** Comment says "expected format: 'Created issue: proj-123'" suggesting single-line assumption, but actual output is multi-line. Splitting by spaces across ALL lines causes the last word to be "open" from "Status: open".

---

### Finding 3: Issue ID is on first line after "Created issue:"

**Evidence:** In the actual output, the issue ID "orch-go-5z9" appears on the first line immediately after "Created issue: "

**Source:** Manual test output

**Significance:** Fix should parse only the first line and extract the issue ID that follows "Created issue: "

---

## Synthesis

**Key Insights:**

1. **Output format mismatch** - The parser was written assuming single-line output "Created issue: proj-123" but actual `bd create` outputs 4 lines with metadata, ending with "Status: open"

2. **Naive string splitting** - The current approach of splitting all output by spaces and taking the last word works for single-line output but fails catastrophically with multi-line output

3. **First line contains the ID** - The issue ID is consistently on the first line in format "✓ Created issue: <issue-id>", making first-line parsing the correct approach

**Answer to Investigation Question:**

The parser captures 'open' instead of the issue ID because it splits the entire multi-line output by spaces and takes the last word. Since `bd create` outputs end with "Status: open", the last word is "open" rather than the issue ID which appears on the first line. The fix is to parse only the first line and extract the text after "Created issue: ".

---

## Confidence Assessment

**Current Confidence:** Very High (99%)

**Why this level?**

Root cause clearly identified through reproduction, fix tested via both unit tests and smoke test. Only tiny uncertainty around edge cases in unusual bd create output formats.

**What's certain:**

- ✅ Root cause identified: parser splits entire multi-line output, captures "open" from last line
- ✅ Fix verified with unit tests covering multiple output formats  
- ✅ Smoke test passed: real bd create command correctly parsed
- ✅ No regressions introduced (existing code paths preserved)

**What's uncertain:**

- ⚠️ Future bd create output format changes (mitigated by robust first-line parsing)
- ⚠️ Edge case: bd create failing with different error format (handled with meaningful error message)

**What would increase confidence to 100%:**

- Long-term monitoring in production usage
- Testing with bd CLI from different versions (if applicable)

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**First-line parsing with strings.Split by newline** - Parse only the first line of output, split by spaces, and extract the issue ID after "issue:"

**Why this approach:**
- Directly addresses root cause by parsing only the first line where the issue ID is located
- Robust to additional metadata lines being added in future `bd create` output
- Simple implementation using standard library (strings.Split, strings.Fields)

**Trade-offs accepted:**
- Assumes first line always contains "Created issue: <id>" format
- Doesn't validate the checkmark symbol (✓) which could change

**Implementation sequence:**
1. Split output by newline to get first line
2. Parse first line for "Created issue:" pattern
3. Extract the issue ID (word after "issue:")

### Alternative Approaches Considered

**Option B: Regex parsing**
- **Pros:** More precise, could validate entire format
- **Cons:** Overkill for simple parsing, harder to maintain
- **When to use instead:** If format becomes more complex or validation needed

**Option C: Expecting bd create to output only the ID**
- **Pros:** Simplest parsing (no parsing needed)
- **Cons:** Requires changing bd CLI, may not be feasible
- **When to use instead:** If we control bd and want to add a --quiet flag

**Rationale for recommendation:** Option A directly fixes the bug with minimal code changes, is robust to future output additions, and uses only standard library functions.

---

### Implementation Details

**What to implement first:**
- Update createBeadsIssue function at cmd/orch/main.go:386-391
- Change parsing logic to: split by newline, parse first line only
- Extract issue ID from first line after "issue:"

**Things to watch out for:**
- ⚠️ Edge case: What if bd create fails and output format is different?
- ⚠️ Need to handle both "Created issue: <id>" and potential "✓ Created issue: <id>" formats
- ⚠️ Empty output or unexpected format should return meaningful error

**Areas needing further investigation:**
- None - root cause is clear and fix is straightforward

**Success criteria:**
- ✅ Parser correctly extracts issue ID from multi-line bd create output
- ✅ Test with actual bd create command and verify issue ID is captured
- ✅ Existing orch spawn functionality works with corrected parsing

---

## References

**Files Examined:**
- cmd/orch/main.go:375-395 - createBeadsIssue function with broken parsing logic
- cmd/orch/main_test.go - Created unit tests for parsing logic

**Commands Run:**
```bash
# Test actual bd create output format
bd create "test issue for parsing"

# Run unit tests
go test -v ./cmd/orch/

# Build binary
make build
```

**External Documentation:**
- None

**Related Artifacts:**
- **Workspace:** .orch/workspace/og-debug-fix-bd-create-20dec/SPAWN_CONTEXT.md
- **Beads Issue:** orch-go-c4r

---

## Investigation History

**2025-12-20 10:00:** Investigation started
- Initial question: Why does bd create output parsing capture 'open' instead of the issue ID?
- Context: Spawned via systematic-debugging skill to fix parsing bug

**2025-12-20 10:05:** Root cause identified
- Ran actual bd create command and observed multi-line output format
- Confirmed parser splits entire output by spaces and takes last word "open"
- Issue ID is on first line in format "✓ Created issue: <id>"

**2025-12-20 10:15:** Fix implemented and tested
- Updated parsing logic to parse only first line
- Created unit tests covering multiple output formats
- All tests passing

**2025-12-20 10:20:** Smoke test completed
- Verified fix works end-to-end with real bd create command
- Successfully parsed "orch-go-1ld" from multi-line output
- Confirmed no regression

**2025-12-20 10:25:** Investigation completed
- Final confidence: Very High (99%)
- Status: Complete
- Key outcome: Parser now correctly extracts issue ID from first line, fixing bug where "open" was captured instead
