# Task: Add FormatBytes to pkg/display

## Instructions

Add a `FormatBytes` function to `pkg/display/display.go` and write comprehensive tests in `pkg/display/display_test.go`.

### Requirements

1. **Function signature:** `func FormatBytes(bytes int64) string`
2. **Behavior:**
   - Format byte counts into human-readable strings
   - Use binary units: B, KiB, MiB, GiB, TiB
   - Show 1 decimal place for non-byte units (e.g., "1.5 MiB")
   - Handle negative values by prefixing with "-"
   - Handle zero: return "0 B"
3. **Examples:**
   - `FormatBytes(0)` → `"0 B"`
   - `FormatBytes(512)` → `"512 B"`
   - `FormatBytes(1024)` → `"1.0 KiB"`
   - `FormatBytes(1536)` → `"1.5 KiB"`
   - `FormatBytes(1048576)` → `"1.0 MiB"`
   - `FormatBytes(-1024)` → `"-1.0 KiB"`
4. **Tests:** Add `TestFormatBytes` to `pkg/display/display_test.go` covering:
   - Zero, small bytes, exact boundaries (1024, 1048576)
   - Negative values
   - Large values (GiB, TiB)
   - Values between boundaries

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run TestFormatBytes
```

Commit your changes when tests pass.

## IMPORTANT: Agent Coordination Protocol

Another agent is SIMULTANEOUSLY working on this same codebase implementing a different feature.

Their task: # Task: Add FormatRate to pkg/display

## Instructions

Add a `FormatRate` function to `pkg/display/display.go` and write comprehensive tests in `pkg/display/display_test.go`.

### Requirements

1. **Function signature:** `func FormatRate(bytesPerSec float64) string`
2. **Behavior:**
   - Format transfer rates into human-readable strings
   - Use binary units with "/s" suffix: B/s, KiB/s, MiB/s, GiB/s
   - Show 1 decimal place for non-byte-per-sec units (e.g., "1.5 MiB/s")
   - Handle zero: return "0 B/s"
   - Handle negative values by prefixing with "-"
3. **Examples:**
   - `FormatRate(0)` → `"0 B/s"`
   - `FormatRate(512)` → `"512 B/s"`
   - `FormatRate(1024)` → `"1.0 KiB/s"`
   - `FormatRate(1536)` → `"1.5 KiB/s"`
   - `FormatRate(1048576)` → `"1.0 MiB/s"`
   - `FormatRate(-1024)` → `"-1.0 KiB/s"`
4. **Tests:** Add `TestFormatRate` to `pkg/display/display_test.go` covering:
   - Zero, small values, exact boundaries (1024, 1048576)
   - Negative values
   - Large values (GiB/s)
   - Fractional values

### Constraints

- Do NOT modify any existing functions
- Do NOT add new dependencies
- Follow existing code style (see existing functions for patterns)

### Verification

After implementing, run:
```bash
go test ./pkg/display/ -v -run TestFormatRate
```

Commit your changes when tests pass.


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

You have access to a shared coordination directory: /tmp/coord-msg-mergedu-9-1077

1. BEFORE writing any code, create your implementation plan:
   - Write to: /tmp/coord-msg-mergedu-9-1077/plan-a.txt
   - Include: which files you'll modify, where you'll insert code (after which function), what function names you'll add
   - IMPORTANT: State the EXACT line region you plan to use (e.g., 'inserting at lines 95-130')

2. AFTER writing your plan, check for the other agent's plan:
   - Read: /tmp/coord-msg-mergedu-9-1077/plan-b.txt
   - If it exists, review it and check for OVERLAPPING LINE REGIONS
   - If your insertion regions overlap or are adjacent (within 3 lines), you MUST change YOUR insertion point
   - Remember: different function names at the same location STILL CONFLICT in git

3. After implementing, write a summary:
   - Write to: /tmp/coord-msg-mergedu-9-1077/done-a.txt
   - Include: what you implemented, the EXACT line numbers of your insertion

### Goal: Your changes must merge cleanly with the other agent's changes. Two insertions at the same line position WILL NOT merge cleanly regardless of content.
