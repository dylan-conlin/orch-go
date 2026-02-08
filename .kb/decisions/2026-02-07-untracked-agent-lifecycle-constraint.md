# Decision: Untracked Agent Lifecycle - Consistent Filtering Constraint

**Date:** 2026-02-07
**Status:** Accepted
**Authority:** Architectural (cross-component: spawn, status, complete, abandon, daemon, dashboard)

---

## Context

Untracked agents (spawned with `--no-track`) use synthetic beads IDs like `{project}-untracked-{timestamp}` that don't exist in the beads database. This creates a parallel lifecycle that bypasses beads tracking entirely.

Four separate investigations (Dec 2025 - Jan 2026) revealed bugs where codepaths didn't handle this pattern consistently:

| Investigation | Bug | Resolution |
|---------------|-----|------------|
| 2025-12-26 | Untracked agents lingered 30 min in `orch status` | Expected behavior (auto-cleanup via idle timeout) |
| 2026-01-04 | `orch abandon`/`orch complete` failed for untracked | Added `isUntrackedBeadsID()` check, skip beads ops |
| 2026-01-20 | Untracked sessions counted against concurrency limit | Added filter in `checkConcurrencyLimit` (daemon already correct) |
| 2026-01-27 | Dashboard didn't show untracked agents | Expected behavior (project filter by design) |

The recurring pattern: every new codepath that touches beads IDs must remember to handle the `-untracked-` case, or it breaks silently.

## Decision

**Constraint:** Every codepath that processes beads IDs MUST call `isUntrackedBeadsID()` early and handle the untracked case explicitly before any beads database operations.

**Specific rules:**

1. **Filter early** — Check `isUntrackedBeadsID(beadsID)` before any `bd` commands, RPC calls, or database lookups. The daemon's `DefaultActiveCount` pattern is the reference:
   ```go
   beadsID := extractBeadsIDFromSessionTitle(s.Title)
   if beadsID == "" || isUntrackedBeadsID(beadsID) {
       continue
   }
   ```

2. **Skip, don't fail** — Untracked agents should be silently skipped in beads operations (status lookup, issue closure, completion events), not cause errors.

3. **Cleanup via timeout** — Untracked agents auto-cleanup after 30 min idle. No explicit completion mechanism exists by design.

4. **Project name is `{project}-untracked`** — The `extractProjectFromBeadsID()` function produces a distinct project name (e.g., `orch-go-untracked`), which is intentionally NOT in `included_projects`. This means dashboard excludes them by default.

5. **Concurrency exclusion** — Untracked sessions MUST NOT count against the concurrency limit. They're ephemeral test/ad-hoc spawns.

## Consequences

**Positive:**
- Consistent behavior across all codepaths (spawn, status, complete, abandon, daemon, dashboard)
- Untracked agents are truly lightweight — no beads overhead, no concurrency impact
- Bugs caught early by following the constraint

**Negative:**
- No completion tracking for untracked agents (can't distinguish "finished" from "stuck")
- 30 min lingering in status is a UX annoyance (but acceptable trade-off)
- Each new codepath is a potential regression if developer forgets the check

**Mitigation for regressions:** Code review checklist item — any function that calls `bd show/close/comment` or uses beads IDs must handle untracked case.

## Synthesized From

- `.kb/investigations/2025-12-26-inv-investigate-untracked-agents-lingering-orch.md`
- `.kb/investigations/2026-01-04-inv-untracked-agents-cleanup-path-problem.md`
- `.kb/investigations/2026-01-20-inv-untracked-sessions-count-against-concurrency.md`
- `.kb/investigations/2026-01-27-inv-investigate-untracked-agents-no-track.md`
