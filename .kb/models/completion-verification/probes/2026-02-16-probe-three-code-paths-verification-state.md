# Probe: Three Code Paths Verification State Divergence

**Date:** 2026-02-16
**Model:** Completion Verification
**Status:** Active

---

## Question

**Claim under test:** The model claims verification operates through "three independent gates" and uses checkpoints to track verification state. The implicit assumption is that all consumers of verification state agree on the definition of "unverified." This probe tests whether three code paths (spawn gate, daemon, review) share a consistent definition.

**Symptom:** After the orch-go-6fer fix (which patched daemon's CountUnverifiedCompletions to filter closed issues), the spawn verification gate still shows 6 unverified items and blocks spawns, while `orch review` shows 0 and daemon preview shows the correct count. The three code paths use different definitions of "unverified."

---

## What I Tested

### Test 1: Code Path Analysis

Traced all three verification state consumers:

**Code Path 1 - Spawn Gate** (`pkg/spawn/gates/verification.go:GetUnverifiedTier1Work`):
- Reads checkpoints → extracts beads IDs → fetches each issue individually via `verify.GetIssue()`
- Filters to **CLOSED** issues only (line 138: `if issue.Status != "closed" { continue }`)
- Checks Tier 1 issues for both gate1 AND gate2 complete
- Returns closed issues that haven't passed both gates

**Code Path 2 - Daemon** (`pkg/daemon/issue_adapter.go:CountUnverifiedCompletions`):
- Reads checkpoints → batch fetches open issues via `verify.ListOpenIssues()`
- Filters to **OPEN** issues only (after orch-go-6fer fix)
- Checks Tier 1 for gate2, Tier 2 for gate1
- Returns count of unverified open issues

**Code Path 3 - Review** (`cmd/orch/review.go:getCompletionsForReview`):
- Scans `.orch/workspace/` for SYNTHESIS.md or light-tier markers
- Uses `filterClosedIssues()` → `verify.ListOpenIssues()` to exclude closed issues
- Only shows **OPEN** issues with workspace directories

### Test 2: Root Cause Identification

The spawn gate has the **opposite** population filter from the other two:
- Spawn gate: checks CLOSED issues → "have you verified work you already finished?"
- Daemon + Review: check OPEN issues → "is there pending work awaiting verification?"

When issues are closed (via `orch review done` or `bd close`):
- Daemon: correctly excludes them (after orch-go-6fer fix)
- Review: correctly excludes them (workspace + open filter)
- Spawn gate: **finds them** as unverified (because it queries closed issues)

### Test 3: Checkpoint File Analysis

All 19 checkpoint entries have `gate1_complete: true`. Only 5 have `gate2_complete: true`. The spawn gate checks both gates, so 14 items would be "unverified" if their issues are closed. Since many of these issues were closed via `orch review done` (which doesn't create gate2 checkpoints), the spawn gate perpetually blocks.

### Test 4: Fix Implementation

Created `pkg/verify/unverified.go` as canonical source of truth:
- `ListUnverifiedWork()` → returns all unverified items (open issues only)
- `CountUnverifiedWork()` → convenience count wrapper
- Uses `ListOpenIssues()` for consistent open-issues filtering
- Uses latest checkpoint per beads ID (append-only file semantics)
- Tier-aware: Tier 1 requires both gates, Tier 2 requires gate1 only

Updated consumers:
- `pkg/spawn/gates/verification.go` → delegates to `verify.ListUnverifiedWork()`
- `pkg/daemon/issue_adapter.go` → delegates to `verify.CountUnverifiedWork()` with legacy fallback

### Test 5: End-to-End Verification

After the fix, tested all three code paths against production data:

```bash
# Spawn gate: no longer blocks
$ orch spawn --bypass-triage investigation "test"
# → Spawned successfully (no verification gate blocked message)

# Daemon: shows 0 unverified
$ orch daemon preview | grep -i verif
# → No verification backlog message

# Review: shows 0 pending
$ orch review
# → "No pending completions"
```

All three code paths now agree: 0 unverified items (all previous items were closed).

---

## What I Observed

### Observation 1: Fundamental semantic disagreement

The spawn gate was designed with a fundamentally different question: "has closed work been verified?" vs. the daemon/review question "is open work awaiting verification?" The former creates an ever-growing backlog (closed issues with incomplete checkpoints accumulate), while the latter naturally resolves as issues are closed.

### Observation 2: Checkpoint file is append-only, never pruned

The checkpoint file at `~/.orch/verification-checkpoints.jsonl` grows indefinitely. Multiple entries can exist for the same beads ID (later entries supersede). No mechanism prunes old entries. This means the spawn gate was doing an expensive O(n) issue lookup for every unique beads ID in the entire checkpoint history.

### Observation 3: The orch-go-6fer fix was a point fix

The orch-go-6fer probe correctly identified the daemon's counting issue but didn't address the spawn gate, which uses a completely separate code path. This confirms the original bug report's diagnosis: "The orch-go-6fer fix only patched one code path."

### Observation 4: Reproduction confirmed and fixed

Before the fix, `orch spawn` blocked with "6 unverified Tier 1 deliverable(s) exist." After the fix, spawn proceeds normally. The 6 items were all closed issues with incomplete gate2 checkpoints — the spawn gate was the only code path that checked closed issues.

---

## Model Impact

**Confirms** model claim: "Checkpoint file is the source of truth for verification state."

**Extends** model: The checkpoint file tracks verification actions, but "unverified" status requires combining checkpoint state WITH issue lifecycle state. The canonical definition is now: "An item is unverified if it has a checkpoint, the issue is still OPEN, and the required gates for its tier are not complete."

**Contradicts** implicit assumption: The model's claim that there are "three independent gates" doesn't capture the issue lifecycle dimension. Verification state is not just about gates — it's about gates AND issue status.

**New invariant:** All consumers of verification state MUST filter to open issues. Verification of closed issues is meaningless (the orchestrator already reviewed and closed them).

**New invariant:** All consumers of verification state MUST use `verify.ListUnverifiedWork()` or `verify.CountUnverifiedWork()` — never implement their own counting logic. This prevents the divergence that caused this bug.

---

**Status:** Complete
