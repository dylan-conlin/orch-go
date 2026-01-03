<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Recovered 4 of 8 bug fixes from Dec 27-Jan 2 commits via cherry-pick; 4 were too entangled with feature work to cherry-pick cleanly.

**Evidence:** Build and tests pass for all recovered fixes. Complex commits had conflicts touching main.go state machine code.

**Knowledge:** Many "bug fix" commits from the spiral period also contain feature additions; pure cherry-pick is only viable for simple, isolated changes.

**Next:** Orchestrator should note which fixes remain and may need manual extraction later.

---

# Investigation: Recover Priority Bug Fixes

**Question:** Which bug fix commits from Dec 27-Jan 2 can be recovered via cherry-pick without conflicts?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** og-feat-recover-priority-bug-03jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Successfully recovered 4 fixes

**Evidence:** 
- `13f852e8`: fix: filter closed issues from orch review NEEDS_REVIEW output - clean cherry-pick
- `8c9cf054`: fix: suppress plugin output from leaking into OpenCode TUI - minor conflict resolved
- `5447a47f`: fix: patterns package now reads JSONL format correctly - clean cherry-pick
- `0c8fedb8`: fix: standardize on localhost instead of 127.0.0.1 - clean cherry-pick

**Source:** git cherry-pick commands, git log

**Significance:** These 4 fixes were relatively self-contained and could be applied without breaking the build.

---

### Finding 2: 4 fixes were too complex to cherry-pick

**Evidence:**
- `4268e9de`: project filtering action-log - touches many files, conflicts in serve.go, context.go, svelte components
- `fc1c8482`: filter closed issues pending-reviews - entangled with artifact API feature additions
- `155e1771`: filter closed issues architect - entangled with SendPromptWithVerification feature
- `baed7fb1`: HTTP API for headless spawns - conflicts with current main.go struct definitions

**Source:** git cherry-pick --no-commit output showing conflicts

**Significance:** These commits were made during active feature development and are interleaved with other changes, making clean extraction difficult.

---

### Finding 3: Build and tests pass after all recovered changes

**Evidence:**
```
go build ./... - success
go test ./pkg/patterns/... ./cmd/orch/... - all tests pass
```

**Source:** Terminal output

**Significance:** Recovered fixes are safe to use and don't break existing functionality.

---

## Synthesis

**Key Insights:**

1. **Cherry-pick is effective for isolated fixes** - Commits that touched only a few files with no structural changes merged cleanly.

2. **Spiral period commits are often compound** - Many "fix:" commits also contain features, refactoring, or structural changes.

3. **Manual extraction is needed for complex fixes** - The remaining 4 fixes would need manual code extraction rather than cherry-pick.

**Answer to Investigation Question:**

4 of 8 bug fix commits could be recovered via cherry-pick:
- filterClosedIssues function for orch review (13f852e8)
- Plugin output suppression (8c9cf054)
- Patterns JSONL format (5447a47f)
- Localhost standardization (0c8fedb8)

The remaining 4 are too entangled with feature work and would need manual extraction.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes after all changes (go build ./...)
- ✅ Pattern tests pass (go test ./pkg/patterns/...)
- ✅ CLI tests pass (go test ./cmd/orch/...)

**What's untested:**

- ⚠️ End-to-end spawn behavior with recovered changes
- ⚠️ Dashboard UI with localhost changes

**What would change this:**

- If more tests fail in CI, would need to investigate further

---

## Commits Recovered

| Commit | Description | Status |
|--------|-------------|--------|
| 13f852e8 | filter closed issues review | ✅ Recovered |
| 8c9cf054 | suppress plugin output | ✅ Recovered |
| 5447a47f | patterns JSONL format | ✅ Recovered |
| 0c8fedb8 | standardize localhost | ✅ Recovered |
| 4268e9de | project filtering action-log | ❌ Too complex |
| fc1c8482 | filter closed issues pending-reviews | ❌ Too complex |
| 155e1771 | filter closed issues architect | ❌ Too complex |
| baed7fb1 | HTTP API for headless spawns | ❌ Too complex |

---

## Self-Review

- [x] Real test performed (build and tests)
- [x] Conclusion from evidence (based on actual cherry-pick attempts)
- [x] Question answered (listed which could and couldn't be recovered)
- [x] File complete

**Self-Review Status:** PASSED
