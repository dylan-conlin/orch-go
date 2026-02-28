# Session Synthesis

**Agent:** og-debug-completion-review-panel-27feb-2819
**Issue:** orch-go-bx3p
**Outcome:** success

---

## Plain-Language Summary

The Completion Review panel was reading from a broken data source (beads label query for `daemon:ready-review`) while the header badge was reading from the daemon's verification counter (checkpoints). These are two completely different systems that happened to measure the same concept — "how many completions need review" — but could diverge. The fix switches the review queue API to use `verify.ListUnverifiedWork()`, which is the same canonical source the daemon seeds its counter from. Now both the header count and the review panel's list come from the same checkpoint-based verification system.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

---

## TLDR

Fixed the Completion Review panel to use `verify.ListUnverifiedWork()` as its data source — the same canonical source that seeds the daemon's verification counter. The panel now shows the actual completions awaiting review with tier and gate status.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve_beads.go` - Changed `getReviewQueueIssues()` → `getReviewQueueItems()` to use `verify.ListUnverifiedWorkWithDir()` instead of beads label query. Updated `ReviewQueueIssueResponse` to include tier/gate fields instead of priority/status/labels.
- `web/src/lib/stores/beads.ts` - Updated `ReviewQueueIssue` interface to match new API response (tier, gate1, gate2).
- `web/src/lib/components/review-queue-section/review-queue-section.svelte` - Updated to display tier labels (T1/T2/T3) and gate status ("needs comprehension", "needs behavioral") instead of priority and labels.

---

## Evidence (What Was Observed)

- **Root cause**: `CLIClient.List()` in `pkg/beads/cli_client.go:126` ignores the `Labels` field from `ListArgs`, so the beads query for `daemon:ready-review` labeled issues was returning ALL in_progress issues (or 0, depending on which client path was used).
- **Two separate tracking systems**: The daemon's `VerificationTracker` counter (in-memory, seeded from checkpoints) and the beads `daemon:ready-review` label (set by `ProcessCompletion`) tracked the same concept through different mechanisms, leading to divergence.
- **Canonical source**: `verify.ListUnverifiedWork()` (in `pkg/verify/unverified.go`) is documented as "the single source of truth for verification state" used by spawn gate, daemon, and review command.

### Tests Run
```bash
go test ./cmd/orch/ -run "Review" -v -count=1
# PASS: 7 tests including TestHandleBeadsReviewQueueWithProjectParam

go test ./pkg/verify/ -run TestListUnverifiedWork -v -count=1
# PASS: 2 tests

go build ./cmd/orch/
# Build succeeds (pre-existing errors in pkg/orch/extraction.go and pkg/agent/lifecycle_impl.go are unrelated)
```

---

## Architectural Choices

### Use checkpoint-based source instead of fixing beads label filtering
- **What I chose:** Switch the review queue API to `verify.ListUnverifiedWork()` (checkpoint-based)
- **What I rejected:** Fixing `CLIClient.List()` to properly filter by labels
- **Why:** The checkpoint-based system is already documented as the canonical source of truth for verification state. Fixing the label filter would still leave two separate tracking mechanisms that could diverge. Using the canonical source eliminates the divergence entirely.
- **Risk accepted:** The review queue items no longer have priority/labels/timestamps fields. Tier and gate status are more actionable for the review use case.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `CLIClient.List()` in `pkg/beads/cli_client.go` silently ignores the `Labels` field in `ListArgs` — it only passes `--status`, `--type`, `--parent`, and `--limit` to `bd list`. This is a latent bug that could affect other label-based queries.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-bx3p`

---

## Unexplored Questions

- **CLIClient.List() label filter bug**: Should `CLIClient.List()` be fixed to pass `--label` flags to `bd list`? This is a broader bug beyond the review queue, but since we've moved the review queue off labels, it's lower priority.
- **daemon:ready-review label obsolescence**: The daemon still adds this label in `ProcessCompletion()`. With the review queue no longer querying it, should the label be removed from the completion flow?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-completion-review-panel-27feb-2819/`
**Beads:** `bd show orch-go-bx3p`
