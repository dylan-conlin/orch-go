<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** GitHub Actions workflow created at `.github/workflows/bloat-check.yml` that fails PRs which increase bloat past 800-line threshold.

**Evidence:** Workflow checks modified files, compares with base branch to distinguish "already bloated" from "made worse by this PR", only fails when PR increases bloat.

**Knowledge:** Gate must be passable - can't block PRs that touch already-bloated files without making them worse. Key exclusions: test files, generated code, vendored deps.

**Next:** Commit and close issue. Monitor first few PRs to validate workflow works correctly.

**Promote to Decision:** recommend-no - Tactical CI workflow implementation, design decision already captured in 2026-01-23 investigation.

---

# Investigation: CI Gate Bloat Control Create

**Question:** How to implement GitHub Actions CI gate that fails if modified files exceed 800 lines?

**Started:** 2026-01-24
**Updated:** 2026-01-24
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Gate must distinguish "already bloated" vs "PR made it bloated"

**Evidence:** From design investigation: "Need to distinguish 'file was already bloated' from 'this PR made it bloated'". If we block all PRs that touch bloated files, agents can't fix existing bloat - violates "gate must be passable by the gated party" principle.

**Source:** `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md:223-224`

**Significance:** The workflow must compare current line count with base branch. Only fail if:
- PR causes file to cross 800-line threshold for first time, OR
- PR increases line count of already-bloated file

---

### Finding 2: Test files need exclusion

**Evidence:** Design investigation notes "CI workflow needs to handle test files specially (they're expected to be long)". Test files naturally grow with the codebase and are less harmful when bloated since they're not imported into production code.

**Source:** `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md:221`

**Significance:** Exclude `*_test.go`, `*.test.ts`, `*.spec.ts`, etc. from the check.

---

### Finding 3: No existing GitHub Actions in repo

**Evidence:** `ls -la .github/workflows/` returned "No .github/workflows directory" - this is the first workflow for this repo.

**Source:** Command output during implementation

**Significance:** Need to create `.github/` directory structure. No existing patterns to follow.

---

## Synthesis

**Key Insights:**

1. **Passable gate principle** - The gate must allow agents to fix bloat, not block them from touching bloated files. Comparison with base branch achieves this.

2. **Exclusion patterns** - Test files, generated files, and vendored code are excluded to prevent false positives on legitimate large files.

3. **Warning vs failure** - Pre-existing bloat shows as warning (informational), while PR-caused bloat shows as failure (blocking).

**Answer to Investigation Question:**

Implemented GitHub Actions workflow that:
- Runs on pull_request to master/main
- Checks all modified files that are Go, TypeScript, Svelte, or JavaScript
- Excludes: test files, generated files, vendor dirs, lock files
- Compares line count with base branch
- FAILS only if PR increases bloat or causes new bloat
- WARNS (but passes) for pre-existing bloat

---

## Structured Uncertainty

**What's tested:**

- ✅ Workflow YAML syntax is valid (follows GitHub Actions schema)
- ✅ Exclusion patterns cover common test file patterns
- ✅ Script logic handles missing files (deleted files, new files in base)

**What's untested:**

- ⚠️ Actual workflow execution on GitHub (not tested - requires PR to run)
- ⚠️ Edge case: renamed files (may need additional handling)
- ⚠️ Performance with many modified files (not benchmarked)

**What would change this:**

- If renamed files are double-counted or missed
- If workflow fails on legitimate cases (false positive reports)
- If performance is too slow for large PRs

---

## Implementation Recommendations

**Implemented:** `.github/workflows/bloat-check.yml`

### Key Design Decisions (Intra-task Authority)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Threshold | 800 lines (hard fail) | Per task spec; design investigation established 800 as Context Noise threshold |
| Test files | Excluded | Expected to be longer, less harmful when bloated |
| Already-bloated | Warning only | Gate must be passable - can't block fixes to existing bloat |
| File types | Go, TS, TSX, Svelte, JS, JSX | Primary code files in this codebase |
| Generated code | Excluded (`*.gen.go`, `*_generated.*`) | Not human-maintainable |
| Vendored code | Excluded (`vendor/`, `node_modules/`) | Third-party code |

### Workflow Behavior

**FAIL (blocking):**
- PR causes file to exceed 800 lines for first time
- PR adds lines to already-bloated file

**PASS (with warning):**
- File was already >800 lines, PR didn't increase it
- File was already >800 lines, PR reduced it

**PASS (clean):**
- All modified files under 800 lines

---

## References

**Files Created:**
- `.github/workflows/bloat-check.yml` - CI workflow implementation

**Files Examined:**
- `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md` - Design investigation with recommendations

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md` - Design decisions
- **Model:** `.kb/models/extract-patterns.md` - 800-line threshold rationale

---

## Investigation History

**2026-01-24 12:00:** Investigation started
- Initial question: Implement CI gate for bloat control per design investigation
- Context: Spawned as light tier task to implement workflow

**2026-01-24 12:15:** Implementation complete
- Status: Complete
- Key outcome: GitHub Actions workflow created with base branch comparison for fair bloat detection
