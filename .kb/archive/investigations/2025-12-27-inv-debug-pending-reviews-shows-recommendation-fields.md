<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Fixed `parseActionItems` to skip indented lines and distinguish `* ` (bullet) from `**` (bold markdown).

**Evidence:** Before fix: 8 items returned for 3 actual recommendations (indented metadata lines captured). After fix: 3 items correctly.

**Knowledge:** Markdown bold (`**Skill:**`) was matched by `strings.HasPrefix(line, "*")`, and indented metadata lines were treated as separate items.

**Next:** Close - fix implemented and tested, all tests pass.

---

# Investigation: Debug Pending Reviews Shows Recommendation Fields

**Question:** Why do recommendation fields like `**Skill:** feature-impl` and `**Context:**` show as separate dismissable items in pending reviews?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** og-debug-pending-reviews-shows-27dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Markdown bold matched as bullet point

**Evidence:** `strings.HasPrefix(line, "*")` returns true for `**Skill:** feature-impl` because the line starts with `*`. This caused markdown bold syntax to be treated as bullet point list items.

**Source:** `pkg/verify/check.go:295` (before fix)

**Significance:** Lines like `**Skill:** feature-impl`, `**Issue:**`, `**Context:**` in Spawn Follow-up sections were being incorrectly captured as action items.

---

### Finding 2: Indented continuation lines not distinguished from main items

**Evidence:** The parsing logic trimmed whitespace before checking list markers, losing indentation information. Lines like `   - Skill: feature-impl` (indented metadata under a numbered item) became `- Skill: feature-impl` after trimming and were captured as separate items.

**Source:** 
- `pkg/verify/check.go:290-296` (original logic)
- `.orch/workspace/og-inv-glass-integration-status-27dec/SYNTHESIS.md` (example with indented lines)

**Significance:** This caused metadata/context lines that were indented under main numbered items to appear as 8 separate items instead of 3 grouped recommendations.

---

### Finding 3: Subsection patterns missing for "Spawn Follow-up" variants

**Evidence:** The code looked for `### Follow-up Work` subsection but the actual template uses `### Spawn Follow-up` and `### If Spawn Follow-up`.

**Source:** 
- `pkg/verify/check.go:274` (regex pattern)
- `.orch/templates/SYNTHESIS.md:103-109` (template uses `### If Spawn Follow-up`)

**Significance:** Spawn Follow-up sections would not be parsed unless they happened to match the older "Follow-up Work" naming convention.

---

## Synthesis

**Key Insights:**

1. **Bullet point detection was too greedy** - Using `HasPrefix(line, "*")` matched both markdown list items (`* item`) and markdown bold (`**bold**`).

2. **Indentation carries semantic meaning** - In markdown nested lists, indented lines are continuation/metadata for the parent item, not separate action items.

3. **Multiple naming conventions exist** - The codebase evolved from `### Follow-up Work` to `### Spawn Follow-up` and `### If Spawn Follow-up`.

**Answer to Investigation Question:**

The recommendation fields showed as separate items due to two bugs in `parseActionItems`:
1. `strings.HasPrefix(line, "*")` matched markdown bold syntax `**Field:**` 
2. Indented lines were not distinguished from main items, so `   - Skill: feature-impl` metadata lines were captured as separate items

Both issues are fixed by: (a) using `"* "` (with space) to match only real bullets, and (b) checking for leading whitespace before trimming to skip indented lines.

---

## Structured Uncertainty

**What's tested:**

- ✅ Fix correctly skips indented lines (verified: new test TestParseSynthesisIndentedContinuationLines passes)
- ✅ Fix correctly ignores `**Field:**` bold syntax (verified: TestParseSynthesisSpawnFollowUpNoFalsePositives passes)
- ✅ Fix still captures non-indented bullets and numbered items (verified: TestParseSynthesisSpawnFollowUpWithActions passes)
- ✅ All existing tests still pass (verified: `go test ./pkg/verify/...` - PASS)

**What's untested:**

- ⚠️ Running server with new binary to verify API output (would require server restart)
- ⚠️ Edge case: tabs vs spaces for indentation (assumed both handled)

**What would change this:**

- Finding would be wrong if markdown files use different indentation conventions (e.g., tabs instead of spaces)
- Finding would be wrong if there are valid use cases for `*item` (no space) as bullet points

---

## Implementation Recommendations

**Purpose:** N/A - Implementation already complete.

### Recommended Approach ⭐

**Fix applied in pkg/verify/check.go** - Two changes to `parseActionItems`:
1. Check for leading whitespace before trimming to skip indented lines
2. Use `"* "` (with space) instead of just `"*"` to distinguish bullets from bold markdown

**Implementation sequence:**
1. ✅ Skip indented lines (check `line[0] == ' ' || line[0] == '\t'` before trimming)
2. ✅ Use `"- "` and `"* "` (with space) for bullet detection
3. ✅ Added pattern matching for `### Spawn Follow-up` and `### If Spawn Follow-up`

---

## References

**Files Modified:**
- `pkg/verify/check.go:261-310` - Updated `extractNextActions` and `parseActionItems` functions
- `pkg/verify/check_test.go` - Added 3 new test cases

**Commands Run:**
```bash
# Verify fix on real synthesis file
go run /tmp/verify_fix.go
# Result: 3 items correctly (down from 8)

# Run tests
go test -run TestParseSynthesis ./pkg/verify/... -v
# PASS: all 8 tests passing
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-inv-glass-integration-status-27dec/SYNTHESIS.md` - Example file that triggered the bug

---

## Investigation History

**2025-12-27 ~09:40:** Investigation started
- Initial question: Why do recommendation fields show as separate items?
- Context: Bug reported in pending reviews section

**2025-12-27 ~10:00:** Root cause identified
- Found two issues: bold syntax matched as bullet, indented lines not distinguished

**2025-12-27 ~10:20:** Investigation completed
- Status: Complete
- Key outcome: Fixed `parseActionItems` to properly handle markdown bold and indented lines
