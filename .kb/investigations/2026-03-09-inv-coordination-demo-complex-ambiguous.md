## Summary (D.E.K.N.)

**Delta:** [Pending — experiment in progress]

**Evidence:** [Pending]

**Knowledge:** [Pending]

**Next:** [Pending]

**Authority:** implementation — Experiment execution within existing harness patterns

---

# Investigation: Coordination Demo — Complex/Ambiguous Multi-File Task

**Question:** Do model capability differences (Haiku vs Opus) emerge on compliance and design quality when given a complex, ambiguous, multi-file task — while coordination failure remains structural?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** investigation agent (orch-go-n43cf)
**Phase:** Implementing
**Next Step:** Run experiment and analyze results
**Status:** Active

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-09-inv-coordination-failure-controlled-demo-same.md | extends | yes — prior trial used simple task, this extends with complex task | N/A |

---

## Experiment Design

### Research Question
Do model capability differences emerge on a complex/ambiguous task — specifically around constraint compliance, design quality, and ambiguity resolution — while coordination failures remain structurally determined?

### Why Complex/Ambiguous (vs Trial 1)
Trial 1 used a simple, fully-specified task (FormatBytes). Both models scored 6/6 because the task had no ambiguity and required only one file modification. The prior investigation explicitly flagged as untested:
- "Complex/ambiguous tasks may show model differences"
- "No multi-file coordination tested"
- "No semantic conflict tested"

### Task Design: Table Renderer

**Multi-file requirement (4 files):**
1. Modify `display.go` — add `VisualWidth(s string) int`
2. Modify `display_test.go` — add `TestVisualWidth` cases
3. Create `table.go` — add `RenderTable(headers, rows) string`
4. Create `table_test.go` — comprehensive tests

**Sources of ambiguity (design choices left to agent):**
- Border/separator style for the table (pipes? dashes? box-drawing?)
- Column padding amount
- How to handle rows with mismatched column counts
- Overall formatting aesthetic

**Sources of compliance testing (explicit constraints):**
- `VisualWidth` MUST use existing `StripANSI` function
- No external dependencies
- Doc comments on all public functions
- Don't modify existing functions
- Place VisualWidth after FormatDurationShort

### Scoring Rubric (10 dimensions)

| Dim | Name | Description |
|-----|------|-------------|
| F0 | Completion | Did the agent produce any changes? |
| F1 | Compilation | Does the code build? |
| F2 | Tests pass | Do new tests pass? |
| F3 | No regression | Do existing tests still pass? |
| F4 | File discipline | Only expected files modified? |
| F5 | VisualWidth spec | Correct signature AND uses StripANSI? |
| F6 | RenderTable spec | Correct signature? |
| F7 | Doc comments | All public functions documented? |
| F8 | Multi-file | Created both table.go AND table_test.go? |
| F9 | No ext deps | Only stdlib imports? |

### Hypotheses

**H1 (compliance):** Opus may score higher on F5/F7 (constraint compliance) because following multi-constraint instructions with ambiguity requires more careful reading.

**H2 (coordination):** Both models will still produce merge conflicts (structural), but the conflict pattern will be richer — "both added" conflicts for new files plus content conflicts for modified files.

**H3 (design):** Opus will produce more idiomatic Go design choices for the table renderer, but this won't affect coordination outcome.

---

## Findings

[To be filled after experiment runs]

---

## References

**Files Examined:**
- `pkg/display/display.go` — Target implementation file (95 lines)
- `pkg/display/display_test.go` — Target test file (135 lines)
- `experiments/coordination-demo/` — Experiment harness

**Commands Run:**
```bash
# Experiment scripts created
experiments/coordination-demo/task-prompt-complex.md
experiments/coordination-demo/run-complex.sh
experiments/coordination-demo/score-complex.sh
experiments/coordination-demo/merge-check-complex.sh
```

---

## Investigation History

**[2026-03-09 15:00]:** Investigation started
- Initial question: Do model capability differences emerge on complex/ambiguous tasks?
- Context: Follow-up to pilot trial 1 (simple task, 6/6 both models)

**[2026-03-09 15:10]:** Experiment design complete
- Created task prompt with 4-file modification requirement
- Created scoring rubric with 10 dimensions (vs 6 in trial 1)
- Key innovation: deliberately ambiguous design choices + explicit constraint compliance
