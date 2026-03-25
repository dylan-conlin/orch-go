# Task: Add VisualWidth to pkg/display

## Instructions

Add a `VisualWidth` function and a `PadToWidth` function to `pkg/display/display.go`, and write comprehensive tests in `pkg/display/display_test.go`.

### Part 1: VisualWidth

1. **Function signature:** `func VisualWidth(s string) int`
2. **Behavior:** Returns the visual display width of a string, ignoring ANSI escape codes.
3. **Constraint:** You MUST use the existing `StripANSI` function — do NOT reimplement ANSI stripping.
4. **Handle Unicode correctly** — count runes, not bytes.

### Part 2: PadToWidth

1. **Function signature:** `func PadToWidth(s string, width int) string`
2. **Behavior:** Right-pads a string with spaces to reach the target visual width. ANSI codes are preserved but don't count toward width. If the string is already wider than `width`, return it unchanged.

### Tests

Add `TestVisualWidth` and `TestPadToWidth` to `pkg/display/display_test.go` covering:
- Plain ASCII strings
- Strings with ANSI color codes
- Unicode strings (CJK characters, emoji)
- Empty strings
- Edge cases (already at width, wider than target)

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies — standard library only
- All public functions MUST have doc comments
- Follow existing code style

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v
```

All tests (existing and new) must pass. Commit your changes when tests pass.

## IMPORTANT: Agent Coordination Protocol

Another agent is SIMULTANEOUSLY working on this same codebase implementing a different feature.

Their task: # Task: Add FormatTable to pkg/display

## Instructions

Add a `FormatTable` function to `pkg/display/display.go` and write comprehensive tests in `pkg/display/display_test.go`.

### Requirements

1. **Function signature:** `func FormatTable(headers []string, rows [][]string) string`
2. **Behavior:**
   - Render headers and rows as an aligned text table
   - Auto-size each column based on the widest content in that column (including header)
   - Separate the header from data rows with a line of dashes
   - Handle ANSI-colored content correctly — use the existing `StripANSI` function to calculate true visual widths
   - Handle edge cases: empty rows (headers only), rows with fewer columns than headers, nil rows
3. **Design choices (up to you):**
   - Border and separator style (pipes, spaces, etc.)
   - Column padding amount
   - How to handle rows with more columns than headers
4. **Constraint:** You MUST use the existing `StripANSI` function for width calculation — do NOT reimplement ANSI stripping.

### Tests

Add `TestFormatTable` to `pkg/display/display_test.go` covering:
- Basic table (headers + rows)
- Empty table (headers only, no rows)
- ANSI-colored content alignment
- Mismatched column counts
- Single-column table
- Wide content (long strings)

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies — standard library only
- All public functions MUST have doc comments
- Follow existing code style

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v
```

All tests (existing and new) must pass. Commit your changes when tests pass.


### CRITICAL: How Git Merge Actually Works

**You likely have a FALSE mental model of git merge.** Most AI agents believe that if two developers add different functions at the same location in a file, git will merge them cleanly. THIS IS WRONG.

**How git merge ACTUALLY works:**
- Git merges at the TEXT level, not the semantic level
- Git does NOT understand function boundaries, names, or code structure
- When two branches both INSERT text at the SAME line position, git produces a CONFLICT — regardless of what the text contains
- Example: If you add 30 lines after line 94 and the other agent adds 40 lines after line 94, git sees two competing insertions at line 94 and marks it as CONFLICT

**Concrete example of what WILL conflict:**
- Branch A: adds FormatBytes (30 lines) after FormatDurationShort
- Branch B: adds FormatRate (30 lines) after FormatDurationShort
- Result: CONFLICT — git cannot merge these because both modified the same region

**What you MUST do to avoid conflicts:**
- Choose a DIFFERENT insertion point than the other agent — at least 3 lines apart
- For display.go: one agent should insert near the TOP of the file (after StripANSI, before FormatDuration), the other after FormatDurationShort
- For display_test.go: same principle — use DIFFERENT locations
- Do NOT both insert 'after FormatDurationShort' — this WILL conflict even though your function names are different

**After reading the other agent's plan, if you see they are inserting at the same location as you, you MUST move your code to a different location.**

### Coordination Mechanism

You have access to a shared coordination directory: /tmp/coord-msg-mergedu-10-1077

1. BEFORE writing any code, create your implementation plan:
   - Write to: /tmp/coord-msg-mergedu-10-1077/plan-a.txt
   - Include: which files you'll modify, where you'll insert code (after which function), what function names you'll add
   - IMPORTANT: State the EXACT line region you plan to use (e.g., 'inserting at lines 95-130')

2. AFTER writing your plan, check for the other agent's plan:
   - Read: /tmp/coord-msg-mergedu-10-1077/plan-b.txt
   - If it exists, review it and check for OVERLAPPING LINE REGIONS
   - If your insertion regions overlap or are adjacent (within 3 lines), you MUST change YOUR insertion point
   - Remember: different function names at the same location STILL CONFLICT in git

3. After implementing, write a summary:
   - Write to: /tmp/coord-msg-mergedu-10-1077/done-a.txt
   - Include: what you implemented, the EXACT line numbers of your insertion

### Goal: Your changes must merge cleanly with the other agent's changes. Two insertions at the same line position WILL NOT merge cleanly regardless of content.
