# Probe: WriteVerificationSignal called from non-human paths defeats comprehension throttle

**Model:** completion-verification
**Date:** 2026-03-26
**Status:** Complete
**claim:** daemon-verification-pause-threshold
**verdict:** extends

---

## Question

Does `WriteVerificationSignal()` fire exclusively from human-initiated paths? The model claims the daemon pauses after N completions without human verification, but is this invariant actually maintained?

---

## What I Tested

Audited all callers of `daemon.WriteVerificationSignal()` across the codebase:

```bash
grep -rn "WriteVerificationSignal" cmd/orch/ pkg/daemon/
```

Found three call sites:
1. `cmd/orch/complete_lifecycle.go:209` — unconditional in `orch complete`
2. `cmd/orch/serve_daemon_actions.go:99` — single issue close via dashboard API
3. `cmd/orch/serve_daemon_actions.go:186` — batch issue close via dashboard API

Cross-referenced with daemon log from 2026-03-26: continuous "Human verification detected - verification counter reset" entries every ~2 minutes, keeping counter at 0/3, while `comprehension:unread` stuck at 4.

---

## What I Observed

All three paths fire `WriteVerificationSignal()` regardless of who the actor is:
- Dashboard API (`serve_daemon_actions.go`) is called by both humans and the orchestrator AI
- `orch complete --headless` (daemon-triggered) fires the signal
- `orch complete` from orchestrator sessions fires the signal
- Result: `completionsSinceVerification` counter never reaches threshold (3)
- Daemon never pauses, comprehension queue never blocks spawning

---

## Model Impact

- [x] **Extends** model with: The verification pause mechanism has a caller-side bypass — `WriteVerificationSignal()` is structurally correct but its callers do not discriminate human from automated invocation. Fixed by removing calls from dashboard API and gating `complete_lifecycle.go` on `!completeHeadless && !target.IsOrchestratorSession`.

---

## Notes

- The fix aligns with the model's documented intent: "Human verification is defined as a manual `orch complete` invocation (not daemon marking ready-for-review)"
- Structural tests added to `verification_tracker_test.go` to prevent regression
- This was the root cause of the March 26 unchecked cascade (34 zero-desc spawns, 4 unread briefs ignored)
