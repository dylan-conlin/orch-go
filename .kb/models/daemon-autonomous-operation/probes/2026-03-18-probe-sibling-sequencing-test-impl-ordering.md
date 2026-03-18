# Probe: Sibling Sequencing — Test-Before-Implementation Build Breakage

**Model:** daemon-autonomous-operation
**Date:** 2026-03-18
**Status:** Complete

---

## Question

The daemon model documents a 6-layer spawn dedup pipeline and blocking dependency support, but does it handle ordering within same-project issues? Specifically: when test-writing and implementation issues from the same project are both ready, can the daemon prevent the test agent from being spawned first (and writing tests for types that don't exist yet)?

---

## What I Tested

1. Traced the Decide phase in `ooda.go` — confirmed it evaluates issues independently with no cross-issue coordination.
2. Confirmed `ListReadyIssues()` returns both `open` and `in_progress` issues (comment in `issue_adapter.go:29`), giving the Decide phase visibility into in-progress siblings.
3. Examined the work graph (`workgraph.go`) — it detects file overlaps but only produces advisory `RemovalCandidates`, not enforcement.
4. Reproduced the exact scrape incident: test issue (scrape-9w3) was first in priority order, spawned before implementation siblings (scrape-52p, scrape-gdh), wrote tests for `ghIssue`/`ghPullRequest`/`anthropicRequest` types that didn't exist yet.

```bash
# Verified the gap in Decide
go test ./pkg/daemon/ -run TestDecide_DefersTestIssueSelectsImpl -v
# PASS: implementation sibling selected instead of test issue
```

---

## What I Observed

**Root cause:** The Decide phase loop (`ooda.go:135`) iterates through `PrioritizedIssues` and selects the first issue that passes `CheckIssueCompliance()`. It has no awareness of sibling issues from the same project. A test issue appearing first in priority order gets selected even when implementation siblings haven't been spawned yet.

**Data path:** `ListReadyIssues()` → includes both open + in_progress → `PrioritizedIssues` → Decide iterates independently → test issue selected first → spawned → writes tests for undefined types → build breaks.

**Fix applied:** Added `ShouldDeferTestIssue()` coordination check in Decide before compliance evaluation. Defers test-like issues (heuristic: title/description contains test patterns) when same-project implementation siblings are open or in_progress.

**Test evidence:** 12 unit + integration tests pass. Full daemon suite (27s) passes with no regressions.

---

## Model Impact

- [x] **Extends** model with: Sibling sequencing gap — daemon lacked cross-issue coordination for same-project test vs implementation ordering. The 6-layer dedup pipeline prevents duplicate spawns but doesn't address spawn ordering. New `ShouldDeferTestIssue()` in Decide phase adds a 7th coordination layer: sibling-aware test deferral.
- [x] **Confirms** invariant: `ListReadyIssues` returns both open and in_progress issues, providing the data needed for sibling-aware coordination without new queries (respects No Local Agent State constraint).

---

## Notes

**Limitation:** The heuristic detection (`isTestLikeIssue`) matches title/description patterns. Issues that don't use test-related keywords will not be deferred. This is acceptable — false negatives (test spawned alongside impl) are better than false positives (impl incorrectly deferred).

**Alternative considered:** Beads blocking dependencies between test and implementation issues. The daemon already supports this (`CheckBlockingDependencies`), but it requires the issue creator (architect or orchestrator) to set up deps explicitly. The sibling deferral is an automatic safety net.
