## Summary (D.E.K.N.)

**Delta:** GPT-5.3-codex failures are primarily completion-protocol compliance failures after successful implementation/testing, not implementation failures.

**Evidence:** In 3 of 4 analyzed stalled sessions, the agent finished code+validation and then ended with conversational text without `orch phase ... Complete` (and no `/exit`), while one resumed session later completed correctly.

**Knowledge:** The system already has idle recovery, but its resume prompt is generic and current timing/rate-limit behavior is too coarse for phase-aware completion nudges.

**Next:** Implement a layered fix: stronger completion language in spawn context plus phase-aware daemon nudge behavior, then evaluate model-profile tuning.

**Authority:** architectural - This touches spawn template policy, daemon recovery behavior, and model-profile routing across subsystems.

---

# Investigation: Design Orch Handles Model Specific

**Question:** What is the right design for handling model-specific completion-protocol failures (especially GPT-5.3-codex) where agents complete work but fail to run final completion steps, and which combination of options A-E minimizes recurrence with low maintenance cost and future-model portability?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** OpenCode worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md` (referenced in code comments) | deepens | yes | none |
| `.kb/investigations/2026-01-26-inv-design-validation-gate-orch-go.md` | extends | partial | none |

---

## Findings

### Finding 1: Failures cluster at the very end of session lifecycle

**Evidence:**
- `og-feat-extract-shared-http-07feb-3d58` shows completed code + tests, then assistant ends with optional follow-up text and no completion phase report.
- `og-feat-decompose-pkg-spawn-07feb-c0cb` shows decomposition complete and tests passing, but stops without `Phase: Complete`.
- `og-feat-enrich-progress-items-07feb-8d68` shows implementation + validation evidence, then stops with optional follow-up text and no completion phase report.
- `og-feat-decompose-review-go-07feb-671c` eventually did post a `Phase: Complete` comment after a resume nudge.

**Source:**
- `.orch/workspace/og-feat-extract-shared-http-07feb-3d58/SESSION_LOG.md:226`
- `.orch/workspace/og-feat-extract-shared-http-07feb-3d58/SESSION_LOG.md:239`
- `.orch/workspace/og-feat-extract-shared-http-07feb-3d58/SESSION_LOG.md:277`
- `.orch/workspace/og-feat-decompose-pkg-spawn-07feb-c0cb/SESSION_LOG.md:222`
- `.orch/workspace/og-feat-decompose-pkg-spawn-07feb-c0cb/SESSION_LOG.md:232`
- `.orch/workspace/og-feat-enrich-progress-items-07feb-8d68/SESSION_LOG.md:210`
- `.orch/workspace/og-feat-enrich-progress-items-07feb-8d68/SESSION_LOG.md:225`
- `.orch/workspace/og-feat-decompose-review-go-07feb-671c/SESSION_LOG.md:250`
- `bd show orch-go-21430`
- `bd show orch-go-21441`
- `bd show orch-go-21451`

**Significance:** The failure mode is not inability to implement; it is failure to execute final protocol steps under certain model behaviors.

---

### Finding 2: Daemon already has idle recovery, but prompt and timing are not completion-specific

**Evidence:**
- Recovery runs by default every 5 minutes, targets agents idle >10 minutes, and rate-limits to one resume attempt per hour.
- Resume prompt currently says to continue work and report progress, but does not emphasize completion protocol checklist.
- Recovery filters by phase/idle but does not use model- or phase-specific completion nudges.

**Source:**
- `pkg/daemon/daemon_lifecycle.go:229`
- `pkg/daemon/daemon_lifecycle.go:230`
- `pkg/daemon/daemon_lifecycle.go:231`
- `pkg/daemon/daemon_periodic.go:244`
- `pkg/daemon/daemon_periodic.go:306`
- `pkg/daemon/recovery.go:142`

**Significance:** Option B is partially present today, but in a generic form that does not directly target this specific end-of-session failure.

---

### Finding 3: Auto-complete from idle transitions has known false-positive history

**Evidence:**
- CompletionService explicitly documents prior disablement of automatic completion due busy->idle false positives.
- Existing note says agents often go idle briefly during normal operation, causing incorrect completion marking.

**Source:**
- `pkg/opencode/service.go:100`
- `pkg/opencode/service.go:101`
- `pkg/opencode/service.go:102`
- `pkg/opencode/service.go:103`

**Significance:** Option C (daemon auto-complete when idle) risks repeating a previously observed reliability failure unless heavily gated.

---

### Finding 4: Model metadata exists, but spawn context template is not model-aware

**Evidence:**
- Spawn config carries `Model` and writes it to `AGENT_MANIFEST.json`.
- `contextData` passed to SPAWN template has no model field today, so template cannot adapt wording by model behavior class.

**Source:**
- `pkg/spawn/config.go:161`
- `pkg/spawn/context.go:180`
- `pkg/spawn/context.go:35`

**Significance:** Option D can be implemented cleanly, but requires explicit plumbing from config/manifest to template generation or daemon behavior.

---

## Synthesis

**SUBSTRATE:**
- **Principle: Infrastructure Over Instruction** - We should not rely only on prompt wording for protocol-critical behavior.
- **Principle: Gate Over Remind** - Session completion should be driven by enforceable checks and structured nudges, not best-effort reminders.
- **Principle: Provenance** - Completion signals should remain evidence-backed (tests/comments/artifacts), not inferred from weak idle heuristics.
- **Code evidence:** automatic idle-based completion was previously disabled for false positives (`pkg/opencode/service.go:100`).

**Key Insights:**

1. **Instruction-only is insufficient** - Some GPT-5.3-codex sessions still stop after success summaries despite already-strong session-complete instructions.

2. **Current recovery is too generic** - Existing idle recovery should evolve into completion-aware nudging (phase + evidence-aware), not raw "continue work" messages.

3. **Auto-close from idle is wrong default** - Completion should remain explicit and evidence-bearing; daemon can nudge/escalate but should not assert completion by inference.

**Answer to Investigation Question:**

Use a composed strategy: **A + B + D**, reject C as default, reject E as terminal policy.

- **A (stronger instructions)** should be applied immediately for low-cost prompt hardening.
- **B (daemon idle nudge)** should be evolved from current generic recovery into a targeted completion nudge for agents idle in Testing/Validation/Implementing-with-tests-done.
- **D (model-specific variants/profiles)** should be capability-profile based (behavior class), not hardcoded per-model templates.
- **C (auto-complete)** should remain off by default due known false positives; only consider as explicit opt-in with strict evidence gates.
- **E (accept limitation)** is useful only as documentation fallback, not a mitigation strategy.

---

## Structured Uncertainty

**What's tested:**

- ✅ Reproduced the failure pattern via primary artifacts: multiple sessions show implementation/test completion without final phase-complete protocol steps (`SESSION_LOG.md` evidence + beads comments).
- ✅ Verified current daemon recovery behavior and timings in code (interval/threshold/rate limit, generic resume prompt).
- ✅ Verified historical false-positive risk for idle-based auto-complete is already documented in code comments and linked investigation.

**What's untested:**

- ⚠️ Effect size of stronger spawn wording alone for GPT-5.3-codex completion compliance.
- ⚠️ Precision/recall of phase-aware idle nudge detection at 5-10 minute windows.
- ⚠️ Whether capability-profile routing outperforms simple per-model hardcoded variants over time.

**What would change this:**

- If A-only trial materially reduces stuck-completion incidents (<5%), daemon-side behavior may be unnecessary.
- If targeted B nudges still lead to frequent non-completion after 2 attempts, escalation strategy needs redesign.
- If future models display similar failure under different markers, profile taxonomy must shift from model-name to behavior-signals.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Adopt A+B+D layered design; keep C disabled by default | architectural | Requires coordinated changes across spawn template, daemon recovery logic, and model/profile routing behavior. |

### Recommended Approach ⭐

**Layered Completion Reliability (A + B + D)** - Strengthen explicit completion protocol language, add phase-aware idle completion nudges, and support behavior-profile variants rather than per-model forks.

**Why this approach:**
- Addresses both instruction path and infrastructure path (resilient to model drift).
- Reuses existing daemon recovery subsystem instead of adding a parallel mechanism.
- Preserves explicit completion provenance while reducing stuck end-state sessions.

**Trade-offs accepted:**
- More moving pieces than A-only (template + daemon updates).
- Some maintenance overhead for profile mapping and nudge copy iteration.

**Implementation sequence:**
1. **A-now:** Tighten `SPAWN_CONTEXT.md` completion section with explicit non-conversational terminal checklist and failure wording for non-completion.
2. **B-next:** Convert generic recovery prompt into completion-aware nudge when idle and last phase is Testing/Validation/Implementing-with-test-evidence.
3. **D-hardening:** Introduce model behavior profiles (e.g., `strict-complete`, `needs-nudge`) with config-driven mapping; avoid per-model template duplication.

### Alternative Approaches Considered

**Option C: Daemon-side auto-complete for idle-with-commit sessions**
- **Pros:** Maximum autonomy, less manual orchestration overhead.
- **Cons:** Conflicts with known false-positive history; hard to prove correctness from idle/commit signals alone.
- **When to use instead:** Controlled opt-in environments where false positives are acceptable and manual review remains mandatory.

**Option E: Accept and document model limitation**
- **Pros:** Zero implementation cost.
- **Cons:** Leaves recurring operational toil and repeated stuck sessions unresolved.
- **When to use instead:** Temporary fallback while implementing A/B/D.

**Rationale for recommendation:** A+B+D best satisfies reliability and provenance together while minimizing hardcoded model debt.

---

### Implementation Details

**What to implement first:**
- `pkg/spawn/context_template.go` completion block hardening (A).
- `pkg/daemon/recovery.go` resume prompt specialization for completion nudges (B).
- `pkg/daemon/daemon_periodic.go` phase-aware nudge trigger criteria and attempt handling (B).

**Things to watch out for:**
- ⚠️ Avoid resurrecting auto-complete false positives (`pkg/opencode/service.go:100` context).
- ⚠️ Keep nudge logic idempotent and rate-limited to avoid prompt spam.
- ⚠️ Ensure model/profile mapping is data-driven to avoid constant code churn as models rotate.

**Areas needing further investigation:**
- Best signal set for "work done but protocol missing" (phase + tests + diff + last assistant text).
- Whether to route nudges via `orch send` or direct client call for strongest reliability and traceability.
- Metric design: completion-protocol compliance rate by model/profile and by skill type.

**Success criteria:**
- ✅ Reduction in "implemented but no Phase: Complete" incidents for GPT-5.3-codex tasks.
- ✅ Recovery nudges result in completion protocol execution within 1-2 attempts for target phases.
- ✅ No measurable rise in false-positive completion states.

---

## References

**Files Examined:**
- `.orch/workspace/og-feat-extract-shared-http-07feb-3d58/SESSION_LOG.md` - end-of-session behavior and missing completion signal.
- `.orch/workspace/og-feat-decompose-review-go-07feb-671c/SESSION_LOG.md` - resumed case that eventually emitted completion.
- `.orch/workspace/og-feat-decompose-pkg-spawn-07feb-c0cb/SESSION_LOG.md` - implementation done without completion closeout.
- `.orch/workspace/og-feat-enrich-progress-items-07feb-8d68/SESSION_LOG.md` - validation done without completion closeout.
- `pkg/daemon/daemon_periodic.go` - idle recovery trigger and rate limiting.
- `pkg/daemon/recovery.go` - recovery prompt payload.
- `pkg/daemon/daemon_lifecycle.go` - default recovery timings.
- `pkg/opencode/service.go` - disabled auto-complete rationale.
- `pkg/spawn/context_template.go` - current completion protocol wording.
- `pkg/spawn/config.go` and `pkg/spawn/context.go` - model metadata plumbing and template data limits.

**Commands Run:**
```bash
orch phase orch-go-21453 Planning "Analyzing model-specific completion protocol failures and design options"
kb create investigation design-orch-handles-model-specific
bd comment orch-go-21453 "investigation_path: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-07-inv-design-orch-handles-model-specific.md"
bd show orch-go-21430
bd show orch-go-21440
bd show orch-go-21441
bd show orch-go-21451
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md` - prior auto-complete false positive context.
- **Workspace:** `.orch/workspace/og-arch-design-orch-handles-07feb-6253/` - current design workspace.

---

## Investigation History

**2026-02-07 09:49:** Investigation started
- Initial question: Design model-aware handling for completion protocol failures.
- Context: Multiple GPT-5.3-codex workers stalled after implementation in same session.

**2026-02-07 10:05:** Primary evidence validated
- Reviewed four session logs and corresponding beads timelines; confirmed end-of-session protocol failure pattern.

**2026-02-07 10:16:** Design synthesis completed
- Selected A+B+D layered approach, rejected C as default due known false-positive risk.

**2026-02-07 10:18:** Investigation completed
- Status: Complete
- Key outcome: Architectural recommendation defined with implementation sequence and explicit trade-offs.
