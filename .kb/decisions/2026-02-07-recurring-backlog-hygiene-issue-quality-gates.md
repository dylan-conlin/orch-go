# Decision: Recurring Backlog Hygiene via Hybrid Quality Gates

**Date:** 2026-02-07
**Status:** Decided
**Context:** orch-go-21452 (agent-created issues with truncated titles and missing descriptions)
**Authority:** Architectural - affects spawn issue creation, completion follow-up behavior, and recurring hygiene operations

---

## Problem

Backlog quality drift recurred in two forms:

1. Auto-created issues had metadata-heavy/truncated titles (`[project] skill: ...`).
2. Auto-created issues omitted descriptions, making them non-actionable for later sessions.

These defects violate amnesia-resilience and create cleanup churn during audits.

---

## Decision

Adopt a **hybrid backlog hygiene model**:

1. **Creation-time quality gates (required, immediate):**
   - Auto-created issues must include a non-empty description.
   - Titles must prioritize task intent over metadata prefixes.
   - Metadata moves to labels/description instead of consuming title budget.

2. **Recurring hygiene checks (scheduled/operational, follow-up):**
   - Add a periodic backlog quality check surface (doctor/audit path) to detect residual or external quality drift.

This combines prevention (at creation) with ongoing detection (recurring hygiene).

---

## Why This Over Alternatives

### Option A: Skill-only fix
- **Pros:** Fast prompt change.
- **Cons:** Not enforceable; misses non-skill issue creation paths.

### Option B: `bd doctor`/CLI extension only
- **Pros:** Strong recurring visibility.
- **Cons:** Detects after damage; does not prevent low-quality creation.

### Option C: Daemon job only
- **Pros:** Autonomous cleanup.
- **Cons:** Added automation complexity and delayed correction.

### Option D: `orch audit` command only
- **Pros:** Explicit operator workflow.
- **Cons:** Manual invocation; still post-facto.

### Option E: **Hybrid (chosen)**
- **Pros:** Prevents bad issues at source and keeps a recurring safety net.
- **Trade-off:** Requires a second follow-up implementation for periodic checks.

---

## Implemented in This Session

1. `cmd/orch/spawn_beads.go`
   - Removed metadata-heavy title format.
   - Added structured description for spawned issues (`Project`, `Skill`, full `Task`).
   - Added `skill:*` label for durable metadata.
   - Applied same behavior to RPC and CLI fallback issue creation paths.

2. `cmd/orch/complete_gates.go`
   - Discovered-work follow-up issue creation now always includes a description.
   - Preserves source context in description (`Follow-up discovered during completion of <beads-id>`).
   - Uses enriched description for area label suggestion.

3. `cmd/orch/spawn_beads_test.go`
   - Added tests ensuring no prefix-style title formatting and full-task preservation in descriptions.

---

## Follow-up Work

Implement recurring backlog quality checks (doctor/audit integration) to flag:
- empty descriptions,
- metadata-prefix title anti-patterns,
- and other actionable quality defects.

---

## References

- `cmd/orch/spawn_beads.go`
- `cmd/orch/complete_gates.go`
- `cmd/orch/spawn_beads_test.go`
- `bd show orch-go-21288` (historical malformed issue example)
- `bd show orch-go-21451` (recent malformed issue example)
