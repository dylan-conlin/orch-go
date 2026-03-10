# Coordination Demo — Complex/Ambiguous Task Results

**Date:** 2026-03-09
**Baseline commit:** 3a3f863a2
**Task:** Add table renderer to `pkg/display` — `VisualWidth` + `RenderTable` across 4 files
**Task type:** Complex, multi-file, deliberately ambiguous design choices

## Scoring Summary

| Metric | Haiku | Opus |
|--------|-------|------|
| F0: Completion | 1/1 | 1/1 |
| F1: Compilation | 1/1 | 1/1 |
| F2: New Tests Pass | 1/1 | 1/1 |
| F3: No Regression | 1/1 | 1/1 |
| F4: File Discipline | 1/1 | 1/1 |
| F5: VisualWidth Spec (uses StripANSI) | 1/1 | 1/1 |
| F6: RenderTable Spec | 1/1 | 1/1 |
| F7: Doc Comments | 1/1 | 1/1 |
| F8: Multi-file (table.go + table_test.go) | 1/1 | 1/1 |
| F9: No External Deps | 1/1 | 1/1 |
| **Total** | **10/10** | **10/10** |
| **Duration** | **65s** | **88s** |

## Coordination Failure

**Merge result: CONFLICT (4 files)**

| File | Conflict Type | Reason |
|------|--------------|--------|
| `display.go` | content | Both added VisualWidth at line 95, different implementations |
| `display_test.go` | content | Both added TestVisualWidth at line 135, different test cases |
| `table.go` | add/add | Both created new file with different content |
| `table_test.go` | add/add | Both created new file with different content |

**New conflict type vs Trial 1:** `add/add` conflicts appear when both agents create the same new file. Trial 1 only had `content` conflicts. The multi-file task doubles the conflict surface (4 files vs 2 files).

## Implementation Comparison

### VisualWidth — The Capability Signal

| Aspect | Haiku | Opus |
|--------|-------|------|
| Implementation | `len(StripANSI(s))` (1 line) | `for range stripped { count++ }` (rune loop) |
| Unicode handling | **Byte count** — incorrect for multi-byte chars | **Rune count** — correct for Unicode |
| `VisualWidth("日本語")` | Returns **9** (bytes) ❌ | Returns **3** (runes) ✅ |
| Unicode test cases | 0 | 2 (`"日本語"`, `"\x1b[31m日本語\x1b[0m"`) |
| Lines of code | 3 | 7 |

**This is a genuine capability difference.** Opus anticipated the Unicode edge case unprompted. The task spec said nothing about Unicode — Opus independently identified that `len()` is wrong for non-ASCII text and implemented rune counting.

Note: Neither is fully correct for CJK (which are typically "full-width"/2-column characters in terminals), but Opus is meaningfully closer.

### Table Renderer — Design Divergence

| Aspect | Haiku | Opus |
|--------|-------|------|
| Column separator | 2 spaces (`"  "`) | Pipe with spaces (`" \| "`) |
| Header separator | Dashes only (`"----"`) | Dashes with plus (`"---+-"`) |
| Extra columns | **Expands table** (appends new columns) | **Ignores extras** (truncates to header width) |
| Trailing newline | Strips (`TrimSuffix`) | Includes in output |
| Builder variable | `result` | `b` |
| Helper function signatures | Identical names | Identical names |
| Helper function implementations | `strings.Join(parts, "  ")` | Loop with `b.WriteString` |

### Test Quality — Opus's Alignment Test

| Aspect | Haiku | Opus |
|--------|-------|------|
| VisualWidth tests | 7 cases | 8 cases (incl. 2 Unicode) |
| Table tests | 10 tests | 9 tests |
| ANSI alignment method | Checks separator width exists | **Verifies vertical position alignment** |
| Extra column assertion | Asserts extras ARE rendered | Asserts extras are NOT rendered |
| Auto-sizing test | No | Yes (verifies separator width ≥ content) |

Opus's `TestRenderTable_ANSIAlignment` is qualitatively stronger:
```go
// Opus: verifies actual column alignment positions
taskAPos := strings.Index(lines[2], "Task A")
taskBPos := strings.Index(lines[3], "Task B")
if taskAPos != taskBPos { ... }
```

vs Haiku:
```go
// Haiku: only checks separator exists
if !strings.Contains(separatorLine, "----") { ... }
```

### Completion Messages

| Aspect | Haiku | Opus |
|--------|-------|------|
| Output | "✅ Completion reported to beads" | File locations, test counts, all passing |
| Detail level | Generic success | Specific (file:line, counts) |

## Key Findings

### Finding 1: Both Models Achieve Perfect Automated Scores (10/10)

Binary compliance scoring cannot distinguish between the implementations. Both models follow all explicit constraints correctly.

### Finding 2: Capability Differences Emerge in Ambiguity Resolution

Opus anticipated Unicode edge cases unprompted, producing a more correct `VisualWidth` implementation. Haiku's byte-counting implementation is subtly wrong for non-ASCII text but passes all its own tests (because Haiku doesn't test Unicode).

**This is the pattern:** capability differences appear not in constraint compliance (both pass), but in **anticipating edge cases the spec didn't mention**.

### Finding 3: Coordination Failure Remains 100% Structural

Despite different implementations, approaches, and even different conflict types (content vs add/add), the merge conflict rate is 100%. All 4 files conflict. The structural nature extends from same-position-insertion (Trial 1) to new-file-creation (Trial 2).

### Finding 4: Semantic Conflict in Design Choices

Even if a hypothetical merge resolved text conflicts, the merged code would have a **semantic conflict**: Haiku and Opus handle extra columns differently (expand vs ignore). Tests from one would fail against the other's implementation. This is a new failure mode not seen in Trial 1.

### Finding 5: Speed Advantage Persists

Haiku: 65s, Opus: 88s. Haiku is 26% faster (vs 22% in Trial 1). The speed advantage is consistent even on more complex tasks.
