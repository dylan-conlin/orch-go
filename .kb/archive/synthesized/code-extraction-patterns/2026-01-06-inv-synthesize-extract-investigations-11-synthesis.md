<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** 10 extraction investigations (one listed file not found) consolidated into authoritative guide at `.kb/guides/code-extraction-patterns.md`.

**Evidence:** Read all 10 available investigations covering serve.go, main.go, and +page.svelte extractions; identified 7 key patterns across Go and Svelte codebases.

**Knowledge:** Extraction pattern is consistent: shared utilities first, then domain-specific files, with tests following handlers. Package-level visibility in Go makes this safe without import changes.

**Next:** Guide created - no further action needed. Use guide for future extraction work.

---

# Investigation: Synthesize Extract Investigations (11 → Guide)

**Question:** What patterns emerge from 11 extraction investigations, and how can they be consolidated into reusable guidance?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Synthesis agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Consistent extraction order - shared utilities first

**Evidence:** Multiple investigations (shared.go, serve_agents.go, status_cmd.go) all follow pattern:
- Phase 1: Extract shared utilities that are used by 2+ domains
- Phase 2+: Extract domain-specific code

**Source:** 
- `2026-01-03-inv-extract-shared-go-utility-functions.md` - 9 utilities extracted first
- `2026-01-03-inv-extract-serve-agents-go-serve.md` - Domain handlers after shared

**Significance:** Extracting shared code first prevents duplication and import complexity.

---

### Finding 2: Package-level visibility eliminates import management

**Evidence:** All Go investigations noted that cross-file function calls work without imports:
- "All files remain in `package main`, so cross-file function calls work without imports"
- "Go's package-level visibility allows splitting code across files without managing imports"

**Source:** Multiple investigations, verified by successful builds

**Significance:** This makes Go extraction safe - no risk of circular imports within the same package.

---

### Finding 3: Parallel agent conflicts require resolution

**Evidence:** Two investigations (`serve_learn.go`, `serve_agents_events.go`) discovered work already completed by parallel agents:
- Duplicate test functions caused build failures
- Resolution involved removing duplicates, not re-doing work

**Source:**
- `2026-01-03-inv-extract-serve-learn-go-serve.md` - "Parallel agent already completed the extraction"
- `2026-01-04-inv-phase-extract-serve-agents-events.md` - "Work Already Completed"

**Significance:** When parallel agents work on related tasks, always check `git log` first.

---

### Finding 4: Significant line reductions achieved

**Evidence:** Total line reductions from 10 investigations:

| Extraction | Lines Reduced |
|------------|---------------|
| serve_agents.go from serve.go | 1106 |
| status_cmd.go from main.go | 1058 |
| clean_cmd.go from main.go | 670 |
| 5 small commands from main.go | 659 |
| serve_agents_cache.go | 430 |
| serve_agents_events.go | 246 |
| StatsBar from +page.svelte | 242 |

**Source:** Line counts from each investigation's DEKN summary

**Significance:** Extraction work has measurable impact on maintainability.

---

### Finding 5: Svelte extraction uses different patterns

**Evidence:** StatsBar extraction used:
- `$bindable` props for two-way parent-child binding
- Direct store imports (no prop drilling)
- Barrel exports via index.ts

**Source:** `2026-01-04-inv-phase-extract-statsbar-component-extract.md`

**Significance:** Svelte patterns differ from Go but serve same goal - cohesive extraction units.

---

## Synthesis

**Key Insights:**

1. **Universal pattern: Shared first, domains second** - Both Go and Svelte extractions follow this order to prevent duplication.

2. **Language-specific safety mechanisms** - Go uses package-level visibility; Svelte uses store imports and bindable props.

3. **Tests follow code** - In Go, test files move with their corresponding handler files.

4. **Parallel agent awareness** - Always check git log when starting extraction work.

**Answer to Investigation Question:**

The 10 extraction investigations reveal a consistent pattern applicable to future work:
1. Identify shared utilities (functions used by 2+ domains)
2. Extract shared utilities to dedicated file
3. Extract domain-specific code in phases
4. Move tests with handlers
5. Target ~300-800 lines per file

This has been documented in `.kb/guides/code-extraction-patterns.md` as the authoritative reference.

---

## Structured Uncertainty

**What's tested:**

- ✅ Pattern identified from 10 real extractions (all builds and tests passed)
- ✅ Guide created with actionable workflow
- ✅ Line count benchmarks from actual extractions

**What's untested:**

- ⚠️ Applying these patterns to non-Go/non-Svelte languages
- ⚠️ Edge cases with complex cross-package dependencies

**What would change this:**

- Future extractions that fail using this pattern would require guide updates
- New languages/frameworks may need different approaches

---

## Implementation Recommendations

**Purpose:** The synthesis is complete - a guide has been created.

### Recommended Approach ⭐

**Use the guide** - `.kb/guides/code-extraction-patterns.md` is now the authoritative reference.

**When to consult:**
- Before starting any extraction work
- When encountering extraction issues
- When onboarding agents to extraction tasks

**Trade-offs accepted:**
- Guide focuses on Go and Svelte (current codebase)
- Future languages may need additions

---

## References

**Investigations Synthesized:**
1. `.kb/investigations/2026-01-03-inv-extract-serve-agents-go-serve.md`
2. `.kb/investigations/2026-01-03-inv-extract-serve-learn-go-serve.md`
3. `.kb/investigations/2026-01-03-inv-extract-serve-system-go-serve.md`
4. `.kb/investigations/2026-01-03-inv-extract-shared-go-utility-functions.md`
5. `.kb/investigations/2026-01-03-inv-extract-status-cmd-go-main.md`
6. `.kb/investigations/2026-01-04-inv-extract-clean-cmd-go-main.md`
7. `.kb/investigations/2026-01-04-inv-extract-small-commands-send-tail.md`
8. `.kb/investigations/2026-01-04-inv-phase-extract-serve-agents-cache.md`
9. `.kb/investigations/2026-01-04-inv-phase-extract-serve-agents-events.md`
10. `.kb/investigations/2026-01-04-inv-phase-extract-statsbar-component-extract.md`

**Note:** `2025-12-26-inv-implement-kb-extract-command-cross.md` was listed but not found.

**Guide Created:**
- `.kb/guides/code-extraction-patterns.md`

---

## Investigation History

**2026-01-06:** Investigation started
- Initial question: Synthesize 11 extraction investigations
- Context: Accumulated investigations benefit from consolidation into guide

**2026-01-06:** Synthesis completed
- Read 10 available investigations (1 not found)
- Identified 5 key findings and 7 patterns
- Created authoritative guide

**2026-01-06:** Investigation completed
- Status: Complete
- Key outcome: Guide created at `.kb/guides/code-extraction-patterns.md`
