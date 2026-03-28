# Session Synthesis

**Agent:** og-arch-architect-sketchybar-orch-28mar-1ef1
**Issue:** orch-go-ngzhu
**Duration:** 2026-03-28 09:34 → 2026-03-28 10:05
**Outcome:** success

---

## Plain-Language Summary

The sketchybar widget has had four separate reliability bugs over the past week — all different symptoms (stale "0", wrong count, unreliable logs) but all sharing the same root cause: the widget faithfully displayed whatever daemon-status.json said, and daemon-status.json kept getting it wrong. The instinct is to add more data sources or health checks to the widget, but that would be treating the display as the problem when it was the data source.

The good news: the data source is already fixed. The throttle collapse (orch-go-ziyvv) removed the in-memory verification counter and made the daemon source comprehension counts from beads labels directly. The mtime-based liveness check catches daemon deaths. The bd fallback provides live data when the daemon is dead. The integration test catches Go/bash health signal drift. Taken together, these fixes — made individually across 4 prior sessions — constitute the structural redesign this investigation was looking for. The widget doesn't need changes; it needed the daemon to stop lying to it.

---

## TLDR

Investigated whether the sketchybar widget needs architectural changes to fix recurring reliability failures. Found the architecture is already correct — all failures traced to daemon data quality problems that have been systematically fixed. Two cleanup items identified: dead verification code in bash script and missing contract documentation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-28-design-architect-sketchybar-orch-widget-recurring.md` — Architectural review finding current design is sound

### Files Modified
- None — this was a review/validation session, no code changes needed

### Commits
- Investigation + synthesis artifacts

---

## Evidence (What Was Observed)

- Daemon comprehension count flows: `CheckComprehensionThrottle()` → `BeadsComprehensionQuerier.CountPending()` → `bd list --label comprehension:unread` → daemon-status.json (daemon_loop.go:634-644). This is beads-sourced, not in-memory.
- VerificationTracker fully removed — `computeVerification()` returns static green (health_signals.go:109-116), comment at daemon_loop.go:536 confirms removal
- Widget mtime check (orch_status.sh:94-115): 120s → yellow, 600s → red/dead, triggers bd fallback
- Integration test covers 14 cases across 4 test functions (sketchybar_integration_test.go)
- Dead code: bash script checks `.verification.is_paused` (line 136-144) which no longer exists in daemon-status.json

### Tests Run
```bash
# Verified integration test exists and covers health parity
# (test reads both Go health computation and extracted bash script, compares outputs)
go test ./pkg/daemon/ -run TestSketchybar -v  # 14 test cases across 4 functions
```

---

## Architectural Choices

### Validate existing architecture vs. redesign
- **What I chose:** Validated that the current poll-file architecture is correct
- **What I rejected:** Adding redundant data sources (always-on beads query, daemon API polling, PID liveness check)
- **Why:** The recurring failures were daemon data quality problems, not widget architecture problems. Adding redundancy to the widget treats the display as the problem. The daemon-side fixes (throttle collapse, mtime liveness, bd fallback) address root causes.
- **Risk accepted:** If a new daemon regression introduces stale data in a field that doesn't have mtime-level detection, the widget will show wrong data again. Mitigated by integration test and the general principle that daemon-status.json data should come from authoritative sources.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification specification.

Key outcomes verified:
- Widget architecture is structurally sound (no changes needed)
- All four recurring failure modes have existing fixes
- Integration test provides ongoing drift detection
- Two cleanup items identified and documented

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-28-design-architect-sketchybar-orch-widget-recurring.md` — Comprehensive architectural review

### Decisions Made
- The widget architecture (poll-file + event provider + conditional bd fallback) is the correct design — no changes needed
- The daemon-widget contract is: daemon sources from beads → writes daemon-status.json atomically → widget reads with mtime validation → falls back to beads when daemon dead

### Constraints Discovered
- bd fallback fires on LIVENESS_LEVEL != green (not just dead) — means 500ms cost during daemon stalls (mtime 120-600s), not just deaths
- Integration test's `buildHealthComputeScript()` must stay manually in sync with actual orch_status.sh — no automated drift detection

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation with architectural recommendation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-ngzhu`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Whether the bd fallback firing during daemon yellow state (mtime 120-600s) is desirable or just a side effect — it adds cost during transient stalls
- Whether `computeVerification()` should be removed entirely from health_signals.go rather than being a static-green stub — it occupies a signal slot without providing information

**Areas worth exploring further:**
- The integration test could be extended to cover the mtime-triggers-bd-fallback path end-to-end
- A comment in orch_status.sh pointing to the integration test would help future editors maintain parity

**What remains unclear:**
- Whether ~40s max lag (30s daemon poll + 10s widget poll) causes practical confusion during rapid orch complete sessions

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-architect-sketchybar-orch-28mar-1ef1/`
**Investigation:** `.kb/investigations/2026-03-28-design-architect-sketchybar-orch-widget-recurring.md`
**Beads:** `bd show orch-go-ngzhu`
