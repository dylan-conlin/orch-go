# Session Synthesis

**Agent:** og-arch-simplify-daemon-throttling-27mar-335f
**Issue:** orch-go-5e02e
**Outcome:** success

---

## Plain-Language Summary

The daemon has two gates that both try to answer "is Dylan keeping up with agent output?" — a verification pause (in-memory counter) and a comprehension queue (beads labels). They use different state, different thresholds, and fire on different subsets of completions. The verification tracker goes stale and needed patches (ResyncWithBacklog). Worse, I found the comprehension gate is accidentally broken for the most common work type: headless completion removes the comprehension:unread label before the daemon can count it, making the comprehension gate invisible for label-ready-review agents. The design collapses both into a single gate backed by comprehension:unread labels, with one critical fix: stop headless/automated completions from clearing the label. This makes comprehension:unread mean "no human has reviewed this" — same semantics the verification tracker tried to enforce with fragile in-memory state.

## Verification Contract

See `VERIFICATION_SPEC.yaml`. Key outcome: 3-phase implementation plan with issues created (orch-go-sdpx1, orch-go-4ekc1, orch-go-ziyvv).

---

## TLDR

Collapse verification pause and comprehension queue into single gate (comprehension:unread count), fix the headless completion race that made the comprehension gate invisible for label-ready-review items, remove VerificationTracker and its 22-file surface area.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-27-design-simplify-daemon-throttling.md` — Full design investigation
- `.orch/workspace/og-arch-simplify-daemon-throttling-27mar-335f/VERIFICATION_SPEC.yaml`
- `.orch/workspace/og-arch-simplify-daemon-throttling-27mar-335f/SYNTHESIS.md`
- `.orch/workspace/og-arch-simplify-daemon-throttling-27mar-335f/BRIEF.md`

### Files Modified
- `.kb/models/completion-verification/model.md` — Added planned removal note to VerificationTracker section

---

## Evidence (What Was Observed)

- **Headless completion race (critical):** In label-ready-review path, `addComprehensionUnread()` adds the label, then `fireHeadlessCompletion()` launches an async goroutine that runs `orch complete --headless` → `TransitionToProcessed()` removes the label. By the next poll cycle (15s), the label is gone. Source: `coordination.go:280-287`, `complete_lifecycle.go:203-207`.

- **Verification tracker is the only effective gate for non-trivial work:** Since the comprehension gate can't see label-ready-review items (headless removes the label), only the verification tracker's in-memory counter provides throttling for the most common work type.

- **Stale state is structural, not a bug:** The verification tracker uses in-memory state that only resets via interactive signal files. Non-interactive closures (headless, bd close) correctly don't write signals, but this means the tracker goes stale. ResyncWithBacklog was a patch, not a fix.

- **22-file verification tracker surface:** VerificationTracker is referenced in daemon.go, daemon_loop.go, daemon_handlers.go, coordination.go, compliance.go, preview.go, status.go, status_display.go, serve_system_daemon.go, serve_attention.go, daemonconfig, plus tests.

---

## Architectural Choices

### Single gate backed by beads labels (comprehension:unread) instead of in-memory state
- **What I chose:** Use existing comprehension:unread beads labels as the sole throttle gate
- **What I rejected:** (A) Keep verification tracker and remove comprehension gate, (B) Create new ReviewTracker abstraction
- **Why:** Beads labels are durable, authoritative state. The comprehension infrastructure already exists. The enabling fix (gate TransitionToProcessed on interactive-only) is a 3-line change. Option A perpetuates stale-state problems. Option B adds code when we should remove it.
- **Risk accepted:** Auto-completed items now count toward the gate. This increases the unread count the orchestrator sees. This is intentional — "you're not keeping up" should include all output.

### Gate TransitionToProcessed on interactive-only (not headless/orchestrator)
- **What I chose:** Only clear comprehension:unread when a human runs orch complete
- **What I rejected:** Continue allowing headless completion to clear the label
- **Why:** This makes comprehension:unread mean "no human has reviewed this" — the exact semantics the verification tracker was trying to enforce. Without this, the gate is defeated by the headless completion race.
- **Risk accepted:** Auto-completed items stay as "unread" until interactive orch complete or resume. The orchestrator needs to actively drain these. `orch daemon resume` provides batch drain.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-27-design-simplify-daemon-throttling.md` — Full design with 3-phase implementation plan

### Decisions Made
- Decision: collapse to single comprehension:unread gate because both gates measure the same thing and the dual-state divergence is a structural problem, not a bug to patch

### Constraints Discovered
- The headless completion race makes comprehension:unread invisible for label-ready-review agents. This was likely causing the comprehension gate to appear functional while actually only throttling auto-complete-light agents (where the ordering accident preserves the label).

---

## Next (What Should Happen)

**Recommendation:** close (implementation via spawned issues)

### If Close
- [x] Investigation complete with 3-phase implementation plan
- [x] 3 implementation issues created (orch-go-sdpx1, orch-go-4ekc1, orch-go-ziyvv)
- [x] Dependencies reported via beads comment
- [x] Model updated with planned removal note
- [x] Ready for `orch complete orch-go-5e02e`

**MIGRATION_STATUS:**
```
MIGRATION_STATUS:
  designed: Single comprehension:unread gate replacing verification pause + comprehension queue
  implemented: none (design only)
  deployed: none
  remaining: 3 issues — orch-go-sdpx1 (Phase 1), orch-go-4ekc1 (Phase 2), orch-go-ziyvv (Phase 3)
```

---

## Unexplored Questions

- Whether any dashboard UI components read the `verification` field from daemon-status.json and would break when it disappears
- Whether `bd list --label comprehension:unread` performance degrades as label volume increases (currently fine, but post-change more items will have this label)
- Whether there's a workflow where `comprehension:processed` triggers downstream automation (none found, but didn't exhaustively search)

---

## Friction

Friction: none — smooth session, good kb context coverage

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-simplify-daemon-throttling-27mar-335f/`
**Investigation:** `.kb/investigations/2026-03-27-design-simplify-daemon-throttling.md`
**Beads:** `bd show orch-go-5e02e`
