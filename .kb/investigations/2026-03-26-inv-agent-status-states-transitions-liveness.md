<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The phase-based liveness machine has three runtime states (`active`, `completed`, `dead`) driven by four mutually exclusive conditions: latest non-complete phase comment, latest complete phase comment, grace-period-without-phase, and timeout-without-phase.

**Evidence:** `pkg/verify/liveness.go` encodes the four condition branches, `pkg/verify/liveness_test.go` exercises each branch and the 5-minute boundary, and `go test ./pkg/verify -run 'TestVerifyLiveness|TestLivenessResult_Warning|TestVerifyLivenessGracePeriod'` passed.

**Knowledge:** Liveness intentionally treats any reported non-complete phase as active regardless of age, while `abandon` layers a separate 30-minute recency guard on top of the same phase data.

**Next:** Close after review; the state machine is documented and no code change is required.

**Authority:** implementation - This session documents existing behavior and validation without proposing cross-component changes.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Agent Status States Transitions Liveness

**Question:** What agent liveness states exist in `pkg/verify/liveness.go`, what exact inputs trigger each state, and how do downstream callers interpret those results?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: `VerifyLiveness` is a 3-state machine with 4 entry conditions

**Evidence:** `VerifyLiveness` first parses the latest `Phase:` comment. If a phase exists and equals `Complete` case-insensitively, it returns `completed` with reason `phase_complete`. If a phase exists but is not complete, it returns `active` with reason `phase_reported`. If no phase exists and `Now - SpawnTime` is less than 5 minutes, it returns `active` with reason `recently_spawned`. Otherwise it returns `dead` with reason `no_phase_reported`.

**Source:** `pkg/verify/liveness.go:13`, `pkg/verify/liveness.go:20`, `pkg/verify/liveness.go:28`, `pkg/verify/liveness.go:102`

**Significance:** This defines the complete state machine: three statuses, four transition conditions, and no other hidden branches.

---

### Finding 2: The latest matching phase comment wins, and phase text is loosely normalized

**Evidence:** `ParsePhaseFromComments` scans comments in order and overwrites `latestPhase` each time a comment matches `Phase:\s*(\w+)`. That means later phase comments replace earlier ones. The regex is case-insensitive and supports summaries after hyphen, en-dash, or em-dash. `Complete` is treated case-insensitively by `VerifyLiveness`, but every other phase label is preserved as written and still maps to `active` as long as it is not `Complete`.

**Source:** `pkg/verify/beads_api.go:18`, `pkg/verify/beads_api.go:81`, `pkg/verify/liveness.go:105`, `pkg/verify/check_test.go:11`

**Significance:** The machine does not enumerate Planning vs Implementing vs Testing states; it collapses all non-complete phases into the same liveness state and relies on the latest comment only.

---

### Finding 3: Downstream commands use the same liveness result differently

**Evidence:** `orch complete` passes comments plus workspace spawn time into `VerifyLiveness`; if the result is `active`, it only warns and asks for confirmation. `orch abandon` calls `VerifyLiveness` without spawn time, so only explicit phase comments can make the session look active there; it blocks immediately only for `recently_spawned` when spawn time is available, then separately applies a 30-minute `checkPhaseRecency` rule to non-complete phases. Tests confirm stale non-complete phases remain `active` inside liveness itself, and the exact 5-minute boundary flips from `active` to `dead` because the grace-period comparison is strict `< 5 min`.

**Source:** `cmd/orch/complete_verification.go:271`, `cmd/orch/abandon_cmd.go:255`, `pkg/verify/liveness_test.go:77`, `pkg/verify/liveness_test.go:193`

**Significance:** The core machine is intentionally simple; freshness semantics beyond the initial grace period are delegated to callers, especially abandon.

---

## Synthesis

**Key Insights:**

1. **Liveness is coarse by design** - The system distinguishes only running, finished, and silent agents; detailed worker phases are metadata for warnings, not additional machine states.

2. **The only silent-to-active transition without a phase comment is time-based** - An agent starts in an implicit grace-period `active` window and becomes `dead` at exactly 5 minutes unless a phase comment arrives first.

3. **Caller policy matters after liveness classification** - `complete` uses liveness as a soft guard, while `abandon` adds a stricter recency policy for non-complete phases rather than expanding the core state machine.

**Answer to Investigation Question:**

`pkg/verify/liveness.go` defines a 3-state machine: `active`, `completed`, and `dead`. The transitions are: no phase comment plus spawn age under 5 minutes enters `active/recently_spawned`; any latest non-complete `Phase:` comment enters or stays in `active/phase_reported`; any latest `Phase: Complete` comment enters `completed/phase_complete`; and no phase comment at or beyond 5 minutes since spawn enters `dead/no_phase_reported`. There is no transition back out of `completed` inside this function, but because it always reparses the latest comment list, a newer non-complete phase comment would classify the issue as `active` again. Tests in `pkg/verify/liveness_test.go` validate all four branches and the exact 5-minute cutoff.

---

## Structured Uncertainty

**What's tested:**

- ✅ All four liveness branches are exercised by unit tests (`go test ./pkg/verify -run 'TestVerifyLiveness|TestLivenessResult_Warning|TestVerifyLivenessGracePeriod'` → `ok`).
- ✅ The grace period is strict `< 5 min`, so exactly 5 minutes is `dead` (`pkg/verify/liveness_test.go:193`).
- ✅ Non-complete phases remain `active` even when stale in core liveness (`pkg/verify/liveness_test.go:77`).

**What's untested:**

- ⚠️ No end-to-end CLI run was performed to observe the interactive `orch complete` confirmation path.
- ⚠️ No historical comment stream was replayed to test whether malformed later phase comments can mask earlier valid ones in real data.
- ⚠️ Dashboard rendering behavior for these reason codes was not revalidated in this session.

**What would change this:**

- The transition table would change if `VerifyLiveness` gained additional status constants or branch conditions outside the four currently tested paths.
- The statement about reactivation from `completed` would be wrong if upstream comment ordering from beads were not preserved.
- The abandon interpretation would change if callers started passing spawn time consistently or removed `checkPhaseRecency`.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| No implementation change recommended; use this investigation as the reference for current liveness semantics. | implementation | The session clarified existing code and tests without uncovering a defect that requires design work. |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Document current semantics** - Treat `VerifyLiveness` as a small coarse-grained classifier and keep freshness policy in callers unless a future architectural review decides to centralize it.

**Why this approach:**
- The current function is easy to reason about because each status has one reason code path.
- Caller-specific policy stays explicit instead of being hidden inside a more complex shared classifier.
- This matches the accepted decision that phase comments are the canonical liveness source for Claude-backend agents.

**Trade-offs accepted:**
- The system keeps a known false-positive window during the first 5 minutes after spawn.
- That trade-off is already accepted in the decision record and covered by documented downstream behavior.

**Implementation sequence:**
1. Use this investigation as the explanation source for future questions about state meanings and transition rules.
2. If future work changes liveness behavior, update `pkg/verify/liveness.go`, `pkg/verify/liveness_test.go`, and this transition table together.
3. Route any hotspot changes through architect review first, per spawn context guidance.

### Alternative Approaches Considered

**Option B: Fold 30-minute recency into `VerifyLiveness`**
- **Pros:** One place would own both immediate and stale-activity logic.
- **Cons:** It would blur the boundary between coarse liveness classification and abandon-specific policy described in Findings 1 and 3.
- **When to use instead:** If multiple callers need identical stale-phase semantics and architectural review confirms that duplication is causing defects.

**Option C: Add more explicit liveness statuses for Planning/Implementing/Testing**
- **Pros:** Dashboard and CLI messaging could become more specific.
- **Cons:** The current parser and tests intentionally treat those as phase metadata, not separate machine states, so this would be a design change rather than a documentation fix.
- **When to use instead:** If product requirements need stateful workflow reporting rather than simple destructive-command safety checks.

**Rationale for recommendation:** The evidence shows the current design is deliberate and already separated into a small state machine plus caller policy, so documentation is the right output for this investigation.

---

### Implementation Details

**What to implement first:**
- None; no code change is recommended from this investigation.
- If future work touches liveness, start by preserving the four tested branches and the exact 5-minute boundary behavior.
- Review caller behavior in `cmd/orch/complete_verification.go` and `cmd/orch/abandon_cmd.go` before changing shared semantics.

**Things to watch out for:**
- ⚠️ `VerifyLiveness` reparses only the latest matching phase comment, so later comments can supersede earlier `Complete` signals.
- ⚠️ `abandon` currently calls `VerifyLiveness` without `SpawnTime`, so its `recently_spawned` branch depends on whether spawn time is plumbed there.
- ⚠️ Any change in beads comment ordering semantics would change the effective transition history.

**Areas needing further investigation:**
- Whether `abandon` should also pass workspace spawn time consistently instead of relying mostly on `checkPhaseRecency`.
- Whether dashboard/query surfaces should expose reason codes more directly to make grace-period false positives easier to understand.
- Whether caller-specific recency policy now exists in too many places to remain coherent in this hotspot area.

**Success criteria:**
- ✅ A reader can map every status and reason code to a single branch in `pkg/verify/liveness.go`.
- ✅ Unit tests remain the executable proof for all four transition conditions and the boundary case.
- ✅ Future changes can be checked against this transition table to see whether semantics changed intentionally.

---

## References

**Files Examined:**
- `pkg/verify/liveness.go` - Primary implementation of status constants, reason codes, and transition logic.
- `pkg/verify/beads_api.go` - Phase comment parser that determines what input `VerifyLiveness` actually sees.
- `pkg/verify/liveness_test.go` - Unit tests for all liveness branches and warning/grace-period behavior.
- `pkg/verify/check_test.go` - Parser tests showing latest-comment-wins and supported phase formats.
- `cmd/orch/complete_verification.go` - Caller behavior for completion warnings and force overrides.
- `cmd/orch/abandon_cmd.go` - Caller behavior for abandon recency checks layered on top of liveness.
- `.kb/decisions/2026-02-26-phase-based-liveness-over-tmux-as-state.md` - Decision record describing intended liveness rules.

**Commands Run:**
```bash
# Verify working directory
pwd

# Create investigation artifact
kb create investigation agent-status-states-transitions-liveness --orphan

# Run targeted liveness tests
go test ./pkg/verify -run 'TestVerifyLiveness|TestLivenessResult_Warning|TestVerifyLivenessGracePeriod'
```

**External Documentation:**
<!-- All URLs must use markdown hyperlinks: [Display Name](https://url) — never bare URLs or plain text -->
- None.

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-26-phase-based-liveness-over-tmux-as-state.md` - Defines the intended four liveness rules implemented in code.
- **Workspace:** `.orch/workspace/og-inv-agent-status-states-26mar-a5e4/` - Session workspace for this investigation.

---

## Investigation History

**[2026-03-26 09:31]:** Investigation started
- Initial question: What are all agent status states and transitions in the liveness system?
- Context: The orchestrator requested a code-traced explanation of `pkg/verify/liveness.go` and its downstream interpretation.

**[2026-03-26 09:32]:** Core transition rules confirmed
- Read `pkg/verify/liveness.go`, `pkg/verify/beads_api.go`, and parser tests to confirm the 3-state / 4-condition machine and latest-phase-wins behavior.

**[2026-03-26 09:32]:** Verification completed
- Ran `go test ./pkg/verify -run 'TestVerifyLiveness|TestLivenessResult_Warning|TestVerifyLivenessGracePeriod'` and confirmed the 5-minute boundary plus warning behavior.

**[2026-03-26 09:32]:** Investigation completed
- Status: Complete
- Key outcome: Documented the exact liveness states, their transition conditions, and how `complete` and `abandon` layer policy on top.
