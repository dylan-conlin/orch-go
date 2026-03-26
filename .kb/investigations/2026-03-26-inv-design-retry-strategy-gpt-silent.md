## TLDR

orch should add a bounded, OpenCode-only retry path for zero-token silent deaths, but not treat every zero-token completion as retryable success. The retry trigger needs an explicit failure fingerprint plus telemetry because current lifecycle code treats persisted idle sessions as alive and therefore misses the failure completely.

## Summary (D.E.K.N.)

**Delta:** Zero-token GPT-5.4 silent deaths should trigger one immediate backend-scoped retry only when orch can prove the session accepted work, produced no tokens, and ended without Phase: Complete or landed artifacts.

**Evidence:** The benchmark recorded a GPT-5.4 investigation that died with `tokens: {input: 0, output: 0}` on attempt 1 then succeeded unchanged on retry, while current code treats `busy -> idle` as success, treats persisted sessions as alive, and hides zero-token stats as `-` in status output.

**Knowledge:** The problem is not "retry or no retry" but missing failure classification; without a classifier orch either misses recoverable deaths or risks duplicate-action loops and premature destruction.

**Next:** Implement the failure classifier, bounded retry handoff, and observability plan captured below before routing GPT-5.4 reasoning skills automatically.

**Authority:** architectural - The fix crosses spawn, session-state interpretation, daemon recovery, and operator visibility boundaries.

---

# Investigation: Design Retry Strategy Gpt Silent

**Question:** Should orch auto-retry GPT-5.4 / OpenCode sessions that terminate with zero tokens, and if so what failure fingerprint and retry budget should govern that behavior?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-7iw5a
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## What I tried

1. Read the current retry, session-status, stall-detection, and orphan-recovery code paths.
2. Read the GPT-5.4 routing and benchmark investigations plus the OpenCode session lifecycle model.
3. Compared where orch already retries transient failures versus where zero-token deaths currently fall through.

## What I observed

1. Spawn retry exists only around session creation / prompt submission, not around post-prompt empty execution.
2. OpenCode idle sessions persist on disk, and lifecycle orphan recovery currently asks only whether the session exists, not whether it is still processing.
3. Status output suppresses zero-token evidence, so the failure is both under-classified and under-observed.

## Test performed

- Verified repository root with `pwd`.
- Created this investigation with `kb create investigation design-retry-strategy-gpt-silent --orphan`.
- Queried prior art with `bd show orch-go-7iw5a` and `kb context "retry strategy gpt silent death openai opencode"`.
- Read primary code and artifact sources cited below to confirm current behavior.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` | extends | yes | none |
| `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` | extends | yes | none |
| `.kb/models/opencode-session-lifecycle/model.md` | extends | yes | model notes predate some SQLite details, but session-status semantics still match code |

---

## Findings

### Finding 1: The observed GPT-5.4 failure is transient, recoverable, and currently indistinguishable from normal idle completion

**Evidence:** The March 26 benchmark recorded one GPT-5.4 investigation failure on first attempt, annotated as a "silent zero-token termination," then the identical task succeeded on rerun in about two minutes. The benchmark explicitly notes `tokens: {input: 0, output: 0}` and classifies the rerun as successful.

**Source:** `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:136`, `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:156`

**Significance:** This is evidence for a retryable class of failure, but only a narrow one: the task itself was viable, and the failure signature was "accepted spawn, no work produced, second attempt succeeds" rather than a deterministic prompt or auth error.

---

### Finding 2: Current spawn safeguards stop at prompt acceptance, not execution viability

**Evidence:** `Retry` wraps only create-session / send-prompt failures. `VerifySessionAfterPrompt` checks that the session exists in the target directory and listens briefly for `session.error`, but it does not verify token production or successful progress after the first few seconds. `WaitForSessionIdle` treats `busy -> idle` as success without checking whether any message or token output exists.

**Source:** `pkg/spawn/errors.go:196`, `pkg/orch/spawn_modes.go:183`, `pkg/orch/spawn_modes.go:308`, `pkg/opencode/client.go:237`, `pkg/opencode/client.go:631`

**Significance:** Zero-token silent death sits in an unowned gap between "spawn failed" and "agent completed." The existing retry mechanism cannot catch it because nothing in the post-prompt execution path classifies it as failure.

**Defect class exposure:** Class 7 (Premature Destruction) if we kill/retry too aggressively on partial work; Class 6 (Duplicate Action) if we retry after the agent actually made hidden progress.

---

### Finding 3: Lifecycle recovery currently misses this failure because persisted idle sessions still count as alive

**Evidence:** Discovery correctly interprets missing OpenCode liveness as idle and maps explicit `idle` status to `session_idle`. But lifecycle orphan detection does not use session status; it calls `SessionExists`, and that method is documented to return true for any persisted session, not just actively running ones. Therefore a session that silently dies into persisted idle state is not considered an orphan and will not be force-abandoned for respawn.

**Source:** `pkg/discovery/discovery.go:463`, `pkg/agent/lifecycle_impl.go:426`, `pkg/opencode/client.go:256`

**Significance:** The current automatic retry path is structurally pointed at the wrong signal. Even if zero-token death is recoverable, orch will not naturally reach the existing orphan-respawn machinery because the session still exists.

**Defect class exposure:** Class 5 (Contradictory Authority Signals) because session existence says "alive" while session status says "idle / done" and beads phase says "not complete."

---

### Finding 4: Operator visibility currently hides the evidence needed to judge retry safety

**Evidence:** Token stats are fetched for status and completion telemetry, but compact status renders zero-token sessions as `-`. Stall detection keys off unchanged token totals only while an agent is still processing, so it never explains a fast `busy -> idle` empty exit. Completion telemetry records tokens only after successful close flows, which is too late for first-failure diagnosis.

**Source:** `cmd/orch/status_test.go:500`, `cmd/orch/status_cmd.go:339`, `pkg/daemon/stall_tracker.go:17`, `cmd/orch/complete_postlifecycle.go:435`

**Significance:** Even a correct retry classifier will be hard to trust without surfacing the exact failure fingerprint. Provenance requires a visible trail: phase, session status transition, token counts, retry count, and final outcome.

**Defect class exposure:** Class 2 (Multi-Backend Blindness) if we generalize OpenCode-only zero-token semantics to Claude/tmux agents that do not share the same status and token API.

---

## Synthesis

**Key Insights:**

1. **The right unit is a failure fingerprint, not a token threshold.** Zero tokens alone are insufficient because they can represent no messages, hidden partial work, or observability gaps; the retry decision needs a compound predicate tied to OpenCode session semantics.

2. **The current system has a semantic hole between spawn and lifecycle.** Spawn retries transport errors, lifecycle retries dead workers, but zero-token `busy -> idle` deaths are neither, so the system silently accepts them.

3. **Retry must be bounded and evidence-preserving.** The benchmark's successful rerun justifies one automatic retry, but the defect taxonomy warns against loops and duplicate work if the classifier is loose.

**Answer to Investigation Question:**

Yes, orch should auto-retry GPT-5.4 silent deaths, but only for an OpenCode-specific "empty execution" fingerprint rather than any generic zero-token termination. The recommended fingerprint is: the session accepted the prompt, transitioned through `busy` or `retry`, ended `idle` within a short execution window, produced zero total tokens and no meaningful assistant text, reported no `session.error`, has no Phase: Complete comment, and has no landed artifacts. When that fingerprint matches, orch should issue exactly one immediate retry against the same issue, log the classification, and escalate to human review on the second failure instead of looping.

---

## Structured Uncertainty

**What's tested:**

- ✅ A real GPT-5.4 investigation silently died once and then succeeded unchanged on rerun (verified: benchmark investigation artifact).
- ✅ Current spawn code retries transport/setup failures but not post-prompt empty execution (verified: read `pkg/spawn/errors.go`, `pkg/orch/spawn_modes.go`, `pkg/opencode/client.go`).
- ✅ Lifecycle orphan detection uses persisted session existence rather than active session status (verified: read `pkg/agent/lifecycle_impl.go` and `pkg/opencode/client.go`).

**What's untested:**

- ⚠️ Exact prevalence of the zero-token fingerprint beyond the single benchmark example.
- ⚠️ Whether some OpenCode sessions can emit zero tokens while still writing useful assistant text or tool side effects.
- ⚠️ Whether architect/debugging workloads show the same retryability profile as investigation work.

**What would change this:**

- If replay tests show zero-token sessions can still produce landed artifacts, the classifier must require artifact checks before retry.
- If larger benchmark runs show repeated second-attempt failures, auto-retry should downgrade to direct escalation rather than remain default.
- If OpenCode exposes a stronger terminal error signal for this class, the design should switch from heuristic token inference to server-native classification.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add an OpenCode-only empty-execution classifier and one-shot retry path | architectural | Crosses spawn, lifecycle, and daemon boundaries and must avoid duplicate work |
| Add telemetry + UI surfacing for retry fingerprint and outcome | architectural | Observability contract spans status, events, and completion review |
| Keep GPT-5.4 reasoning skills on manual / limited routing until retry path is validated | strategic | Affects production routing and trust posture |

### Recommended Approach ⭐

**Fingerprinted One-Shot Retry** - Treat zero-token termination as retryable only when a compound OpenCode failure fingerprint proves "accepted work, produced nothing, exited early."

**Why this approach:**
- It matches the only observed real-world failure, which succeeded unchanged on rerun.
- It respects defect classes 6 and 7 by requiring evidence before retrying and by limiting retry count to one.
- It keeps backend semantics honest instead of imposing an OpenCode token heuristic on tmux / Claude workers.

**Trade-offs accepted:**
- We accept a slightly slower completion path because orch must inspect session outcome before deciding retry vs escalation.
- We defer broad multi-model routing until the classifier is validated on more than one benchmark incident.

**Implementation sequence:**
1. Add a terminal-outcome classifier for OpenCode sessions that records status transition, token stats, assistant output presence, and artifact state (`orch-go-o9k80`).
2. Use that classifier to route a matched empty-execution failure into one immediate retry with retry metadata attached to the issue / workspace / events stream (`orch-go-phbzy`).
3. Surface the classification and retry outcome in status / review so operators can distinguish transient recovery from deterministic failures (`orch-go-bfzfc`), then validate the full behavior end-to-end (`orch-go-6k6o8`).

### Alternative Approaches Considered

**Option B: Never auto-retry zero-token terminations**
- **Pros:** Zero risk of duplicate action; simplest behavior.
- **Cons:** Leaves a proven recoverable failure class to manual intervention and keeps GPT-5.4 reasoning skills artificially underpowered.
- **When to use instead:** If subsequent evidence shows the fingerprint is noisy or partial work is common.

**Option C: Auto-retry any zero-token or missing-phase session**
- **Pros:** Maximum recovery automation.
- **Cons:** Too loose; likely to create loops, hide real regressions, and mis-handle backend differences.
- **When to use instead:** Only if OpenCode later emits a canonical terminal reason that makes the classifier precise.

**Rationale for recommendation:** Option A is the only path that converts the observed transient into automation without violating provenance or lifecycle safety.

---

### Implementation Details

**What to implement first:**
- A small `OpenCodeTerminalOutcome` classifier object derived from session status, messages, token stats, and workspace artifact state.
- A one-shot retry budget scoped to `empty_execution` failures, separate from spawn transport retry and verification retry budgets.
- Event / status fields that make the retry reason visible (`empty_execution`, `retry_attempt=1`, `recovered=true|false`).

**Things to watch out for:**
- ⚠️ Do not infer failure from zero tokens alone; require prompt acceptance plus no assistant output plus no landed artifacts.
- ⚠️ Keep this backend-scoped; Claude/tmux agents do not share OpenCode token/status semantics.
- ⚠️ Avoid reusing orphan detection's `SessionExists` check for this path because persisted idle sessions are the problem.

**Areas needing further investigation:**
- Capture 5-10 more GPT-5.4 investigation / architect runs to estimate true empty-execution frequency.
- Determine whether OpenCode can expose a stronger server-side termination reason than current client heuristics.
- Decide whether empty-execution retry should attach a beads label for later analytics or stay in event telemetry only.

**Success criteria:**
- ✅ A simulated or replayed empty-execution session is retried once automatically and the second attempt is visible in events/status.
- ✅ A repeated empty-execution failure escalates instead of looping.
- ✅ Status / review surfaces the exact fingerprint and retry outcome so a human can verify why orch acted.

---

## References

**Files Examined:**
- `pkg/spawn/errors.go` - existing transport-level retry behavior
- `pkg/orch/spawn_modes.go` - headless spawn verification path after prompt submission
- `pkg/opencode/client.go` - post-prompt verification, session existence, idle wait semantics
- `pkg/agent/lifecycle_impl.go` - orphan detection and retry eligibility logic
- `pkg/discovery/discovery.go` - backend-aware status mapping for OpenCode sessions
- `cmd/orch/status_cmd.go` - token fetching and stall/status presentation
- `pkg/daemon/stall_tracker.go` - stalled-session semantics
- `cmd/orch/complete_postlifecycle.go` - token telemetry capture timing
- `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - benchmark evidence for the failure and rerun success
- `.kb/models/opencode-session-lifecycle/model.md` - session lifecycle constraints and status semantics

**Commands Run:**
```bash
# Verify project path and create investigation
pwd && kb create investigation design-retry-strategy-gpt-silent --orphan

# Inspect issue context
bd show orch-go-7iw5a

# Search knowledge base context
kb context "retry strategy gpt silent death openai opencode"
```

**External Documentation:**
- None - this recommendation is based on local code and artifact evidence.

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - source of the observed GPT-5.4 silent-death benchmark evidence.
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` - prior routing work that made GPT-5.4 available in OpenCode.
- **Plan:** `.kb/plans/2026-03-26-gpt54-empty-execution-retry.md` - phased handoff for the three-component implementation.
- **Workspace:** `.orch/workspace/og-arch-design-retry-strategy-26mar-e992/` - session artifacts for this design pass.

---

## Investigation History

**[2026-03-26 09:55]:** Investigation started
- Initial question: Should orch auto-retry GPT-5.4 silent deaths that end with zero tokens?
- Context: Benchmark evidence showed one investigation task died silently on first attempt but succeeded on rerun.

**[2026-03-26 10:10]:** Current lifecycle gap confirmed
- Found that spawn retries transport/setup failures, while orphan recovery misses persisted idle OpenCode sessions because it checks `SessionExists` instead of active status.

**[2026-03-26 10:20]:** Design recommendation formed
- Recommended a backend-scoped, one-shot retry driven by an explicit empty-execution fingerprint plus telemetry, not a generic zero-token heuristic.

**[2026-03-26 10:27]:** Investigation completed
- Status: Complete
- Key outcome: Created a plan plus four implementation issues for bounded empty-execution retry and verification.
