# Probe: Does ready-queue intake drop unfindable issues before spawn?

**Model:** `.kb/models/beads-integration-architecture.md`
**Date:** 2026-02-08
**Status:** Complete

---

## Question

When `bd ready` (or RPC-ready equivalent) returns an issue ID that cannot be resolved by `show`, does daemon-side intake filter that ID out so downstream spawn paths only see actionable issues?

---

## What I Tested

**Command/Code:**
```bash
go test ./pkg/daemon -run TestFilterAccessibleReadyIssues_DropsIssueNotFound -v
go test ./pkg/daemon
```

**Environment:**
- Repo: `orch-go`
- Branch state: dirty working tree (existing unrelated edits)
- Change under test: `pkg/daemon/issue_adapter.go` accessibility filter in ready intake paths

---

## What I Observed

**Output:**
```text
=== RUN   TestFilterAccessibleReadyIssues_DropsIssueNotFound
2026/02/08 16:46:11 warning: dropping unfindable issue from ready queue: orch-go-b
--- PASS: TestFilterAccessibleReadyIssues_DropsIssueNotFound (0.00s)
PASS
ok   github.com/dylan-conlin/orch-go/pkg/daemon 0.014s

ok   github.com/dylan-conlin/orch-go/pkg/daemon 9.182s
```

**Key observations:**
- Ready intake now explicitly drops IDs that return `beads.ErrIssueNotFound` during accessibility checks.
- Non-not-found errors are treated as transient and kept in queue (prevents false negatives from temporary RPC/CLI failures).
- Package tests pass after introducing the filter.

---

## Model Impact

**Verdict:** extends — Beads ID Not Found

**Details:**
The model already describes ID-not-found as a cross-project failure mode at completion time. This probe adds an earlier mitigation point: ready-queue intake now validates accessibility and drops unfindable issues before spawn attempts. That reduces wasted spawn cycles and turns a late failure into early queue hygiene.

**Confidence:** Medium — Verified by deterministic test coverage and full package test pass; not yet re-observed with a live cross-project phantom ID in current workspace state.
