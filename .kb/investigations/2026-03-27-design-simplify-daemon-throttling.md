## Summary (D.E.K.N.)

**Delta:** The verification pause and comprehension queue gates measure the same thing ("you're not keeping up") but use separate state that diverges, and the comprehension gate is broken for label-ready-review items due to a headless completion race.

**Evidence:** Code audit of all three completion routes shows headless completion's TransitionToProcessed removes comprehension:unread before the daemon's next poll cycle, making the comprehension gate invisible for the most common work type. Meanwhile, verification tracker uses stale in-memory state requiring ongoing patches (ResyncWithBacklog).

**Knowledge:** Collapsing to a single gate backed by durable beads labels (comprehension:unread) eliminates both the stale-state problem and the dual-gate confusion. The critical enabling change: gate TransitionToProcessed on interactive-only (not headless), so comprehension:unread survives until human review.

**Next:** Implementation — 3 issues created: core gate collapse, dashboard/API surface update, resume mechanism.

**Authority:** architectural — Cross-component change (daemon, completion lifecycle, dashboard API, hooks) with multiple valid approaches

---

# Investigation: Design — Simplify Daemon Throttling

**Question:** Should verification pause and comprehension queue be collapsed into a single gate, and if so, how?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** worker (orch-go-5e02e)
**Phase:** Complete
**Next Step:** None — implementation issues created
**Status:** Complete
**Model:** completion-verification

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-26-inv-daemon-verification-pause-disagrees-orch.md | extends | yes — ResyncWithBacklog was a patch on the stale-state problem; this design eliminates the problem entirely | - |
| .kb/models/completion-verification/probes/2026-03-26-probe-verification-signal-bypass-from-non-human-paths.md | extends | yes — signal gating was correct but created conditions where tracker goes stale; single gate makes this irrelevant | - |

---

## Findings

### Finding 1: The two gates track different subsets with different state backends

**Evidence:**
- Verification tracker: in-memory counter + seenIDs map, only counts `label-ready-review` completions. Reset only by interactive `orch complete` (not headless, not orchestrator). State stored in RAM with signal files for cross-process communication.
- Comprehension queue: beads labels (`comprehension:unread`), counts ALL completions (auto-complete-light, auto-complete, label-ready-review). Cleared by ANY `orch complete` invocation (including headless and orchestrator).

**Source:** `pkg/daemon/verification_tracker.go:62-86` (RecordCompletion), `pkg/daemon/coordination.go:279` (only label-ready-review calls recordUnverifiedCompletion), `pkg/daemon/coordination.go:233,250,282` (all three routes call addComprehensionUnread), `cmd/orch/complete_lifecycle.go:203-219` (TransitionToProcessed runs for all, WriteVerificationSignal gated on interactive only)

**Significance:** The gates have fundamentally different scopes and lifetimes. This makes it hard to reason about when either will fire and creates the "two gates, one pipeline" confusion.

---

### Finding 2: Comprehension gate is broken for label-ready-review items (headless completion race)

**Evidence:** In the label-ready-review path:
1. `addComprehensionUnread(beadsID)` — adds `comprehension:unread` label
2. `fireHeadlessCompletion(beadsID)` — fires async goroutine running `orch complete --headless`
3. Headless completion calls `completeLifecycleActions()` → `TransitionToProcessed()` → removes `comprehension:unread`

The headless goroutine typically completes within seconds. By the next daemon poll cycle (15s), `comprehension:unread` is already gone. The comprehension gate never sees label-ready-review items.

For auto-complete-light, the race goes the other direction (CompleteLight runs BEFORE addComprehensionUnread), leaving both labels present. This was already documented in `.kb/briefs/orch-go-sfzdv.md`.

**Source:** `pkg/daemon/coordination.go:280-287` (labelReadyReview sequence), `cmd/orch/complete_lifecycle.go:203-207` (TransitionToProcessed runs for headless), `pkg/daemon/coordination.go:217-235` (auto-complete-light order)

**Significance:** The comprehension gate effectively only throttles auto-complete-light agents where the ordering accidentally preserves the label. For the most common non-trivial work (label-ready-review), only the verification tracker provides a real gate.

---

### Finding 3: Verification tracker stale-state is a recurring problem requiring patches

**Evidence:** The ResyncWithBacklog mechanism (added 2026-03-26, investigation orch-go-zem67) was needed because the in-memory tracker seeded at startup never re-checked against actual backlog. Issues closing through non-interactive paths (headless completion, bd close) didn't write verification signals (correctly, per orch-go-hry8a), causing the tracker to stay paused on stale counts.

ResyncWithBacklog calls `verify.ListUnverifiedWork()` on every pause check, reading a JSONL checkpoint file. This is a patch on a design problem: in-memory state that doesn't reflect reality.

**Source:** `pkg/daemon/verification_tracker.go:145-176` (ResyncWithBacklog), `cmd/orch/daemon_loop.go:567-579` (called during pause), `.kb/investigations/2026-03-26-inv-daemon-verification-pause-disagrees-orch.md`

**Significance:** The checkpoint file itself has 206 stale entries. The verification tracker is fighting against its own state model. Switching to beads labels (which are the authoritative state for issue lifecycle) eliminates the category of problem.

---

### Finding 4: Verification tracker has 22-file surface area in the codebase

**Evidence:** VerificationTracker and related types are referenced in: daemon.go (struct field, two constructors), daemon_loop.go (seed, signal check, pause check, status write), daemon_handlers.go (seed in dry-run/preview/once), coordination.go (recordUnverifiedCompletion), compliance.go (gate check), preview.go (pause flag), status.go (VerificationStatusSnapshot), status_display.go (paused display), serve_system_daemon.go (DaemonVerificationStatus API type), serve_attention.go (metadata), verification_tracker.go (core impl), verification_tracker_test.go (890 lines of tests), plus daemonconfig compliance.go (DeriveVerificationThreshold).

**Source:** `grep -r VerificationTracker --include="*.go"` — 22 files

**Significance:** This is substantial removal surface. The implementation must touch all these files, but the changes are mechanical (remove field, remove call, remove type). The comprehension gate already exists in compliance.go Gate 3, so the core gate logic stays.

---

## Synthesis

**Key Insights:**

1. **The comprehension gate was accidentally broken, making verification the only effective gate** — Headless completion removes comprehension:unread before the daemon can count it for label-ready-review items. This means verification was carrying the full throttling load for non-trivial work, with comprehension only catching auto-complete-light items through an ordering accident.

2. **Both gates measure "you're not keeping up" but the implementation divergence creates confusion** — Two thresholds to configure, two state backends to debug, two code paths that don't communicate. The 2026-03-26 ResyncWithBacklog fix was a patch on state divergence. The right fix is one source of truth.

3. **The enabling change is small: gate TransitionToProcessed on interactive-only** — The single change that makes comprehension:unread viable as the sole gate is: don't let headless/automated completions clear the label. This makes comprehension:unread mean "no human has reviewed this" (same semantics the verification tracker was trying to enforce with in-memory state).

**Answer to Investigation Question:**

Yes, the two gates should be collapsed. The single gate should be the comprehension:unread count backed by beads labels, with the critical fix of gating TransitionToProcessed on interactive orch complete only. This eliminates the stale in-memory state problem, the dual-gate confusion, and the broken comprehension race — all at once.

---

## Structured Uncertainty

**What's tested:**

- ✅ Traced all three completion routes (auto-complete-light, auto-complete, label-ready-review) to confirm comprehension label behavior (code audit, coordination.go)
- ✅ Verified headless completion removes comprehension:unread (complete_lifecycle.go:203-207, TransitionToProcessed runs for all paths)
- ✅ Confirmed verification tracker is the only effective gate for label-ready-review (code audit of timing between addComprehensionUnread and fireHeadlessCompletion)
- ✅ Enumerated all 22 files consuming verification tracker state

**What's untested:**

- ⚠️ Whether gating TransitionToProcessed on interactive-only breaks any other consumer of the processed label
- ⚠️ Whether the comprehension:unread count latency (bd list call every poll cycle) is acceptable as the sole gate check
- ⚠️ Whether auto-completed items staying as "unread" creates UX confusion (hook will show higher counts)
- ⚠️ Whether the resume drain mechanism (transition all unread→processed) is fast enough for large backlogs

**What would change this:**

- Design would be wrong if there's a consumer that depends on TransitionToProcessed running for headless completions (e.g., a workflow where processed label triggers downstream automation)
- Design would be wrong if beads label queries are too slow to check every 15s poll cycle (currently already happening for Gate 3)
- Design would need adjustment if auto-completed items should NOT count toward the gate (currently they'll count because comprehension:unread stays)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Collapse to single comprehension:unread gate | architectural | Cross-component: daemon, completion lifecycle, dashboard API, hooks |
| Gate TransitionToProcessed on interactive-only | architectural | Changes semantics of comprehension label lifecycle |
| Threshold values per compliance level | implementation | Numerical choice within established compliance framework |

### Recommended Approach ⭐

**Single Review Backlog Gate** — Replace both verification pause and comprehension queue with one gate: comprehension:unread count >= compliance-derived threshold.

**Why this approach:**
- Eliminates the stale in-memory state problem (Finding 3) — beads labels are the authoritative state
- Fixes the broken comprehension gate (Finding 2) — gating TransitionToProcessed makes the label survive until human review
- One threshold, one state backend, one gate to debug (Finding 4 — removes 22-file verification surface)
- Aligns with principle: "evolve by distinction" — we're not conflating two things; these ARE one thing

**Trade-offs accepted:**
- Auto-completed items now count toward the gate (previously invisible). This is intentional — "you're not keeping up" should count all output.
- Higher unread counts in the hook. The orchestrator will see more items. This is accurate, not noise.
- Resume mechanism changes from "reset counter" to "drain labels." Slightly slower for large backlogs but more durable.

**Implementation sequence:**

#### Phase 1: Fix the comprehension label lifecycle (must be first)
1. Gate `TransitionToProcessed()` in `complete_lifecycle.go` on `!completeHeadless && !target.IsOrchestratorSession`
2. Remove `WriteVerificationSignal()` call from complete_lifecycle.go
3. Update comprehension_queue_test.go for new semantics

#### Phase 2: Remove verification tracker and collapse gates
4. Remove Gate 1 (verification pause) from `CheckPreSpawnGates()` in compliance.go
5. Remove `VerificationTracker` field from Daemon struct
6. Remove `recordUnverifiedCompletion()` from coordination.go
7. Remove verification tracker initialization from daemon.go (NewWithConfig, NewWithPool)
8. Remove `seedVerificationTracker()` from daemon_helpers.go
9. Remove `checkVerificationPause()` from daemon_loop.go
10. Simplify `checkDaemonSignals()` — remove verification signal check
11. Add compliance-derived threshold: `DeriveReviewThreshold()` replacing `DeriveVerificationThreshold()`
12. Wire compliance-derived threshold into ComprehensionThreshold in daemon constructors
13. Remove `VerificationPauseThreshold` from daemonconfig.Config

#### Phase 3: Update surfaces and resume mechanism
14. Remove `VerificationStatusSnapshot` from DaemonStatus in status.go
15. Update status_display.go — replace verification pause display with comprehension gate display
16. Update serve_system_daemon.go — replace DaemonVerificationStatus with comprehension gate info
17. Update preview.go — remove VerificationPaused flag
18. Update resume signal handler — drain all comprehension:unread → processed
19. Remove verification_tracker.go and verification_tracker_test.go
20. Remove signal file infrastructure (WriteVerificationSignal, CheckAndClearVerificationSignal, VerificationPath)
21. Keep resume signal infrastructure (WriteResumeSignal, etc.) but change handler behavior
22. Update `.claude/hooks/comprehension-queue-count.sh` messaging

### Alternative Approaches Considered

**Option B: Keep verification tracker, remove comprehension gate**
- **Pros:** Verification tracker is the working gate; less code change
- **Cons:** Keeps the stale in-memory state problem. Doesn't count auto-completed items. Requires ongoing patches like ResyncWithBacklog.
- **When to use instead:** If beads label queries prove too slow for poll-cycle gates

**Option C: Unify into a new ReviewTracker backed by beads labels**
- **Pros:** Clean abstraction, fresh API
- **Cons:** More code to write than reusing existing comprehension infrastructure. The comprehension gate already does what we need.
- **When to use instead:** If the comprehension gate's label semantics (unread/processed/pending) prove inadequate

**Rationale for recommendation:** Option A reuses existing infrastructure (comprehension labels, gate check, hook) and requires primarily removal rather than creation. The one new behavior (gating TransitionToProcessed) is a 3-line change. The principle of "coherence over patches" argues for Option A over Option B's ongoing patch cycle.

---

### Implementation Details

**What to implement first:**
- Phase 1 (fix comprehension label lifecycle) MUST land first — without it, removing the verification tracker leaves no effective gate
- Phase 1 is safe to land independently — it only makes the comprehension gate stricter

**Things to watch out for:**
- ⚠️ Defect Class 1 (Filter Amnesia): Any new completion route added later must call `addComprehensionUnread()`. Document this requirement.
- ⚠️ The `comprehension:processed` label becomes the "ready for Dylan to read" state. Items stay `unread` until interactive orch complete. Ensure the hook messaging is clear about this.
- ⚠️ Auto-completed items now need interactive `orch complete` to clear from the gate. The orchestrator needs a way to batch-drain. `orch daemon resume` provides this (transitions all unread→processed).
- ⚠️ The compliance-derived threshold replaces two separate thresholds. The values (strict=3, standard=8, relaxed=20, autonomous=0) should match the existing verification thresholds since that's the gate that was actually working.
- ⚠️ Defect Class 5 (Contradictory Authority Signals): After collapse, comprehension:unread is the single source of truth for "needs human review." Ensure no other code path interprets this label differently.

**Areas needing further investigation:**
- Performance of `bd list --label comprehension:unread` every 15s poll cycle (currently already happening, likely fine)
- Whether any dashboard UI components depend on the `verification` field in daemon-status.json

**Success criteria:**
- ✅ Daemon pauses after N completions without human review (single threshold, not two)
- ✅ `orch daemon resume` clears the gate and spawning continues
- ✅ Interactive `orch complete` clears individual items from the gate
- ✅ Headless/automated completions do NOT clear items from the gate
- ✅ All existing daemon tests pass (minus removed verification tracker tests)
- ✅ Hook injection shows accurate review backlog count

---

## References

**Files Examined:**
- `pkg/daemon/compliance.go` — Pre-spawn gates (Gate 1: verification, Gate 3: comprehension)
- `pkg/daemon/verification_tracker.go` — Full verification tracker implementation
- `pkg/daemon/comprehension_queue.go` — Comprehension queue implementation and labels
- `pkg/daemon/coordination.go` — Completion routing and label application
- `cmd/orch/complete_lifecycle.go` — TransitionToProcessed and WriteVerificationSignal calls
- `cmd/orch/daemon_loop.go` — Signal handling and pause checking
- `pkg/daemon/daemon.go` — Daemon struct and constructors
- `pkg/daemon/status.go` — DaemonStatus and VerificationStatusSnapshot
- `pkg/daemon/status_display.go` — Status display formatting
- `cmd/orch/serve_system_daemon.go` — Dashboard API verification status
- `pkg/daemonconfig/config.go` — Threshold configuration
- `pkg/daemonconfig/compliance.go` — DeriveVerificationThreshold
- `.claude/hooks/comprehension-queue-count.sh` — Hook injection for orchestrator turns

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-26-inv-daemon-verification-pause-disagrees-orch.md` — Prior fix (ResyncWithBacklog) for the stale-state problem this design eliminates
- **Probe:** `.kb/models/completion-verification/probes/2026-03-26-probe-verification-signal-bypass-from-non-human-paths.md` — Signal gating that created conditions for stale state
- **Brief:** `.kb/briefs/orch-go-sfzdv.md` — Documents the double-label bug from headless completion race
- **Decision:** `.kb/global/decisions/2026-01-04-verification-bottleneck.md` — Verification bottleneck principle (still applies — single gate is still structural enforcement)
