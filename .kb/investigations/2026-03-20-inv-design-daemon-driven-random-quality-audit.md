## Summary (D.E.K.N.)

**Delta:** A 3-layer daemon-driven random quality audit system closes the broken feedback loop (0 negative signals / 1,113 completions) by structurally embedding audit selection, automated deep review, and `orch reject` integration into the daemon's existing periodic task infrastructure.

**Evidence:** Existing `orch audit select` (audit_cmd.go) provides crypto-random selection but isn't wired to daemon; `orch reject` (reject_cmd.go, orch-go-c51kt) provides 1-step rejection but lacks automated trigger; `ComputeLearning()` (learning.go) tracks per-skill metrics but has no `RejectedCount` field or `agent.rejected` event handler — learning loop is structurally blind to rejections.

**Knowledge:** Structural > signaling hierarchy demands: audit selection embedded in daemon scheduler (structural, unavoidable), deep review spawned as investigation agent (structural, automated), and rejection fed back through existing `orch reject` pipeline (structural, event-emitting). The CV model's "no agent-judgment gates" constraint applies to synchronous completion gates but NOT to asynchronous post-hoc audits by independent agents — randomness + temporal separation break the closed-loop concern.

**Next:** Implementation in 3 phases: (1) daemon periodic audit selection + `RejectedCount` in learning.go, (2) audit agent skill with structured review protocol, (3) daemon audit-result-to-reject pipeline.

**Authority:** architectural — Crosses daemon, completion, learning, and skill system boundaries; requires orchestrator synthesis across 4 subsystems.

---

# Investigation: Design Daemon-Driven Random Quality Audit

**Question:** How should a daemon-driven random quality audit system be designed to close the broken feedback loop (0 negative signals / 1,113 completions), following the structural > signaling effectiveness hierarchy?

**Started:** 2026-03-20
**Updated:** 2026-03-20
**Owner:** architect (orch-go-f2ynp)
**Phase:** Complete
**Next Step:** Implementation issues created for 3-phase build-out
**Status:** Complete
**Model:** completion-verification

**Patches-Decision:** N/A (new capability)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/completion-verification/probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md` | extends | Yes — verified event counts (1,113 completions, 0 reworks, 11 operational abandons) | None |
| `.kb/models/knowledge-accretion/probes/2026-03-20-probe-judge-verdict-accretion-exploration.md` | extends | Yes — verified coverage gap #2 (agent performance degradation measurement) | None |
| `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md` | informs | Read — "no agent-judgment gates" in completion pipeline | Design navigates this: audit is post-hoc, not synchronous gate |

---

## Findings

### Finding 1: Existing Infrastructure is Nearly Complete But Disconnected

**Evidence:** Three components exist independently, covering 80% of the needed pipeline:

| Component | Location | What It Does | What's Missing |
|-----------|----------|-------------|----------------|
| `orch audit select` | `cmd/orch/audit_cmd.go` | Crypto-random selection of 2 completions/week, labels `audit:deep-review` | Not wired to daemon; no automated review spawning |
| `orch reject` | `cmd/orch/reject_cmd.go` | 1-step rejection: reopen + label + `agent.rejected` event | No automated trigger; only manual invocation |
| `ComputeLearning()` | `pkg/events/learning.go` | Per-skill metrics: spawn/complete/abandon/rework counts, success rates | Does NOT handle `agent.rejected` events; no `RejectedCount` |
| Daemon scheduler | `pkg/daemon/scheduler.go` | 27 periodic tasks with named intervals, `IsDue`/`MarkRun` API | No `TaskAuditSelect` registered |

**Source:**
- `cmd/orch/audit_cmd.go:135-175` — `runAuditSelect()` with crypto-rand Fisher-Yates shuffle
- `cmd/orch/reject_cmd.go:55-149` — `runReject()` with event emission
- `pkg/events/learning.go:96-172` — `computeLearningFiltered()` switch statement: handles `AgentReworked` but NOT `AgentRejected`
- `pkg/daemon/scheduler.go:7-34` — 27 task constants, no audit task

**Significance:** The design is primarily wiring — connecting existing components rather than building from scratch. The critical gap is the `ComputeLearning()` blind spot: even if `orch reject` is used manually, the daemon learning system will never see the signal.

---

### Finding 2: The "No Agent-Judgment Gate" Constraint Has Narrow Scope

**Evidence:** The CV model's constraint (§ "Why No Agent-Judgment Gates?") states:

> "Agent reviewing agent code is a closed loop — same model family, same blind spots, no provenance chain."

This was decided for **synchronous completion gates** where the reviewing agent is blocking the completion pipeline. Three properties distinguish a random audit from a completion gate:

| Property | Completion Gate | Random Audit |
|----------|----------------|--------------|
| **Timing** | Synchronous (blocks completion) | Asynchronous (days after completion) |
| **Selection** | Every completion | Unpredictable subset (N%) |
| **Agent relationship** | Same session or follow-on | Independent agent, different session, different context |
| **Consequence of false positive** | Blocks valid completion | Rejects completed work (can be overridden) |
| **Provenance** | Opinion masquerading as gate | Investigation artifact with evidence |

The CV model also notes: "cross-model opinions are still opinions, not executable evidence. The correct application is cross-model review as **advisory signal**." Random audit findings are advisory — they trigger `orch reject` which reopens the issue for human review, not automatic deletion.

**Source:**
- `.kb/models/completion-verification/model.md:377-388` — "Why No Agent-Judgment Gates?" section
- `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md` — original decision

**Significance:** The design is architecturally consistent with the CV model's constraints because it operates in a fundamentally different context (post-hoc, asynchronous, advisory). The audit agent's verdict triggers `orch reject`, which reopens the issue — a human (the orchestrator) can override by simply completing it again.

---

### Finding 3: The Effectiveness Hierarchy Demands Structural Embedding

**Evidence:** From the intervention effectiveness audit (probe 2, judge verdict):

> Effectiveness hierarchy: **structural attractors > signaling > blocking > advisory > metrics-only**
> 4 of 31 interventions work (13%). The ones that work are structural.

The current `orch audit select` is at the "advisory" tier — a launchd job sends a notification, then someone manually reviews. 0 audit results have been acted on (the `orch audit install` launchd plist exists but there's no evidence of regular use).

To be effective, the audit system must be **structural**: embedded in the daemon loop, spawning review agents automatically, and feeding results into `orch reject` without requiring manual intervention.

**Source:**
- `.kb/models/knowledge-accretion/probes/2026-03-20-probe-judge-verdict-accretion-exploration.md:87-88` — effectiveness hierarchy
- `cmd/orch/audit_cmd.go:177-257` — launchd installation (advisory mechanism)

**Significance:** The design must automate the full loop: selection → review → rejection. Any manual step in the middle (e.g., "audit labels issues, human reviews later") will have the same 0% action rate as the current advisory approach.

---

### Finding 4: Learning System Gap — RejectedCount Not Tracked

**Evidence:** `pkg/events/learning.go` defines `SkillLearning` with these counters:

```go
SpawnCount, TotalCompletions, SuccessCount, ForcedCount,
AbandonedCount, ReworkCount, VerificationFailures, VerificationBypasses
```

The `computeLearningFiltered()` switch handles: `SessionSpawned`, `AgentCompleted`, `AgentAbandonedTelemetry`, `SpawnGateDecision`, `AgentReworked`, `VerificationFailed`, `VerificationBypassed`.

**Missing:** `EventTypeAgentRejected` is not in the switch. The `SkillLearning` struct has no `RejectedCount` field.

**Consequence:** Even after `orch reject` ships (orch-go-c51kt), the daemon's `ComputeLearning()` will silently ignore rejection events. Success rates remain 100%. The learning loop stays broken.

**Source:** `pkg/events/learning.go:96-172` — switch statement missing `EventTypeAgentRejected` case

**Significance:** This is a prerequisite fix. Without it, the entire audit → reject → learn pipeline has a broken final link.

---

### Finding 5: Auto-Completed Work is the Highest-Value Audit Target

**Evidence:** From the human feedback probe:
- 37% of completions (406/1,102) are daemon auto-completed via `orch complete --force`
- These bypass ALL interactive verification gates (explain-back, behavioral)
- They enter the learning loop indistinguishable from human-verified work

Auto-completed work has the lowest verification rigor and highest probability of containing undetected quality issues.

**Source:**
- `.kb/models/completion-verification/probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md:82-87`
- `pkg/daemon/auto_complete.go` — `--force` flag bypasses interactive gates

**Significance:** The audit selection should weight toward auto-completed work, not sample uniformly from all completions. A 2/week random sample from the full pool might never audit auto-completed work if manual completions outnumber them.

---

## Synthesis

**Key Insights:**

1. **The pipeline is 80% built** — `orch audit select` (selection), `orch reject` (feedback), `ComputeLearning` (aggregation) exist. The work is wiring them together through the daemon's periodic task infrastructure, plus fixing `learning.go`'s blind spot.

2. **Post-hoc audits are architecturally distinct from completion gates** — The CV model's "no agent-judgment gates" constraint doesn't apply because audits are asynchronous, randomly sampled, and produce advisory verdicts (not blocking gates). The audit agent's output triggers `orch reject` which is reversible.

3. **Auto-completed work should be oversampled** — 37% of completions skip all human verification. Random uniform sampling would audit these proportionally, but they deserve disproportionate scrutiny because they've received the least verification.

4. **The learning loop needs `RejectedCount` before anything else** — Without this fix in `learning.go`, the entire feedback pipeline is broken at the aggregation layer. This is the structural prerequisite.

**Answer to Investigation Question:**

The daemon-driven random quality audit system should be a 3-layer structural pipeline embedded in the daemon's existing periodic task infrastructure:

**Layer 1 — Selection** (daemon periodic task): Weekly random selection of N completions, weighted toward auto-completed work, using existing `recentClosedIssues()` + `cryptoRandSelection()` infrastructure from `audit_cmd.go`.

**Layer 2 — Deep Review** (spawned audit agent): An investigation-skill agent examining issue intent match, test coverage of changes, and obvious problems. Produces a structured verdict (PASS/FAIL with category and evidence).

**Layer 3 — Feedback** (daemon result processing): On FAIL verdict, daemon calls `orch reject` with the audit agent's findings. This emits `agent.rejected`, which the learning system (after `RejectedCount` fix) aggregates into per-skill quality rates.

---

## Structured Uncertainty

**What's tested:**

- Existing `orch audit select` correctly uses crypto/rand for unpredictable selection (verified: read `audit_cmd.go:115-133`)
- `orch reject` correctly emits `agent.rejected` events with skill/category metadata (verified: read `reject_cmd.go:129-137`)
- `ComputeLearning()` does NOT handle `agent.rejected` events (verified: grep for `AgentRejected` in `learning.go` returns 0 matches)
- Daemon scheduler supports adding new periodic tasks via `Register()` (verified: read `scheduler.go:59-67`)
- 37% of completions are auto-completed (verified: event counts in probe)

**What's untested:**

- Whether audit agent can reliably determine issue-intent match (requires building and measuring the audit skill)
- Whether 2/week sample rate produces actionable signal vs noise (requires running for 4+ weeks)
- Whether audit-triggered rejections will be overridden by orchestrators (creating a new "rubber stamp" pattern)
- Performance impact of spawning audit agents on daemon capacity (consumes 1 slot per audit)

**What would change this:**

- If audit agents produce >50% false positive rate, the system becomes noise and should be abandoned in favor of purely structural checks (test execution, build, static analysis)
- If the "no agent-judgment gates" constraint is interpreted to include post-hoc asynchronous audits, the audit agent approach is invalid — fall back to purely mechanical checks
- If `orch reject` is never implemented (orch-go-c51kt doesn't ship), Layer 3 has no feedback path

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `RejectedCount` to `learning.go` | implementation | Tactical fix inside `pkg/events`, no cross-boundary impact |
| Add `TaskAuditSelect` daemon periodic task | implementation | Follows 27 existing periodic task patterns exactly |
| Design audit agent skill | architectural | New skill crossing daemon, spawn, and completion subsystems |
| Wire daemon audit results to `orch reject` | architectural | Cross-subsystem pipeline: daemon → reject → learning |
| Auto-complete oversampling strategy | architectural | Changes daemon completion classification semantics |

### Recommended Approach: 3-Phase Structural Audit Pipeline

**Phase 1: Learning Loop Fix + Daemon Selection** (implementation authority)

Wire the prerequisite infrastructure:

1. **Add `RejectedCount` to `SkillLearning` struct** in `pkg/events/learning.go`:
   - Add `RejectedCount int` field
   - Add `EventTypeAgentRejected` case in `computeLearningFiltered()` switch
   - Adjust success rate: `SuccessRate = SuccessCount / (TotalCompletions + AbandonedCount)` stays the same (rejections are a subset of completions), but add `RejectionRate = RejectedCount / TotalCompletions`
   - This makes rejection data visible to `SuggestDowngrades()` in `daemonconfig`

2. **Add `TaskAuditSelect` periodic task** in `pkg/daemon/scheduler.go` and a new `periodic_audit.go`:
   - Register with 7-day interval (configurable via config.yaml `audit.interval`)
   - Reuse `recentClosedIssues()` and `cryptoRandSelection()` from `audit_cmd.go` (extract to `pkg/audit/` shared package)
   - Default: select 2 completions per cycle
   - Selection weighting: 60% from auto-completed pool, 40% from all completions
   - Label selected issues `audit:deep-review`
   - Emit `audit.selected` event with beads IDs

3. **Add `audit.select_count` and `audit.interval` to daemon config** in `pkg/daemonconfig/`:
   - `select_count: 2` (default)
   - `interval: 168h` (7 days, default)
   - `auto_complete_weight: 0.6` (fraction of selections from auto-completed pool)

**Phase 2: Audit Agent Skill** (architectural authority)

Create a lightweight investigation skill specialized for quality audit:

1. **Create `skills/src/worker/quality-audit/` skill**:
   - Input: beads ID of completed issue
   - Agent reads: original issue description, SYNTHESIS.md, git diff, test output, AGENT_MANIFEST.json
   - Structured review checklist:
     - **Intent Match**: Does the SYNTHESIS.md description match the original issue? Do git changes address the asked-for behavior?
     - **Test Coverage**: Do tests exist for changed files? Run `go test ./... -count=1` on affected packages. Are test names related to changed behavior?
     - **Obvious Problems**: Introduced bugs visible in diff? Architectural violations? Unreasonable file growth?
     - **Verification Depth**: What verification level was assigned? What gates were bypassed? Was this auto-completed?
   - Output: structured verdict file `AUDIT_VERDICT.md` in workspace:
     ```
     verdict: PASS | FAIL
     category: quality | scope | approach | stale  (only if FAIL)
     confidence: high | medium | low
     reason: <1-2 sentence explanation>
     evidence: <specific file/line references>
     ```

2. **Skill routing**: Daemon spawns audit agents with `investigation` skill type (or new `quality-audit` skill if created), Opus model (deep reasoning category in daemon's model inference table).

3. **Capacity consideration**: Audit spawns should NOT count against the daemon's main capacity pool — they're infrastructure work, not issue-driven work. Add `audit_capacity: 1` to daemon config (separate pool) or use `--extra-capacity` flag.

**Phase 3: Verdict-to-Reject Pipeline** (architectural authority)

Wire audit results back to the feedback loop:

1. **Daemon monitors audit completions**: When an `audit:deep-review` labeled issue's audit agent reports `Phase: Complete`, daemon reads `AUDIT_VERDICT.md`.

2. **On FAIL verdict**:
   - Daemon calls `orch reject <original-beads-id> "<reason>" --category <category>`
   - This reopens the original issue, adds `rejected` + `triage:ready` labels, emits `agent.rejected`
   - Learning loop picks up the signal via Phase 1's `RejectedCount` addition
   - Emit `audit.failed` event with audit beads ID + original beads ID for traceability

3. **On PASS verdict**:
   - Remove `audit:deep-review` label from original issue
   - Emit `audit.passed` event
   - This is positive confirmation — it means verified-quality work exists in the system

4. **On LOW confidence verdict**: Surface for orchestrator review instead of auto-rejecting. Add `audit:needs-review` label. This prevents audit false positives from auto-rejecting good work.

**Why this approach:**
- Follows structural > signaling hierarchy — all three layers are embedded in daemon (structural, unavoidable)
- Reuses 80% existing infrastructure (audit_cmd.go selection, reject_cmd.go feedback, learning.go aggregation)
- Respects CV model's "no agent-judgment gates" by operating post-hoc and asynchronously
- Weighted auto-complete sampling targets the highest-risk completions
- LOW confidence gate prevents audit false positives from creating noise

**Trade-offs accepted:**
- Audit agents consume capacity (1 slot per audit, ~2/week) — acceptable given 5-slot default pool
- Audit verdicts are agent judgment, not provenance — mitigated by LOW confidence gate and reversibility of `orch reject`
- 7-day cycle means quality issues detected late — acceptable for v1; can decrease interval if audit precision is high

### Alternative Approaches Considered

**Option B: Purely Structural Checks (No Audit Agent)**
- **Pros:** No agent capacity cost; no agent-judgment concern; deterministic
- **Cons:** Cannot assess intent match (the highest-value check); limited to mechanical verification (test existence, build, file size) which completion gates already do
- **When to use instead:** If audit agent false positive rate exceeds 50%, fall back to this

**Option C: Human-in-the-Loop Audit (Notification Only)**
- **Pros:** Human judgment is authoritative; no false positive risk
- **Cons:** Current `orch audit install` launchd approach is exactly this — and it has 0% action rate. Advisory mechanisms fail (effectiveness hierarchy evidence)
- **When to use instead:** Never — this is the current system that already doesn't work

**Option D: Daemon Performs Lightweight Checks, Spawns Agent Only for Suspicious**
- **Pros:** Efficient — only deep-reviews suspicious work
- **Cons:** Adds two-tier complexity; "suspicious" heuristics would need tuning; defeats the randomness principle (selection becomes predictable based on heuristics)
- **When to use instead:** If capacity is severely constrained and 2 audit agents/week is too expensive

**Rationale for recommendation:** Option A maximizes structural effectiveness while reusing existing infrastructure. The learning loop fix (Phase 1) is a prerequisite regardless of approach. The audit agent (Phase 2) is the only approach that can assess semantic intent match. The verdict pipeline (Phase 3) is the only approach that feeds signal back automatically.

---

### Implementation Details

**What to implement first:**
- `RejectedCount` in `learning.go` — prerequisite for the entire feedback loop; blocks nothing, enables everything
- `TaskAuditSelect` daemon periodic task — low risk, follows 27 existing patterns exactly
- Extract `recentClosedIssues()` and `cryptoRandSelection()` to `pkg/audit/` for shared use

**Things to watch out for:**
- Audit agent must NOT use `bd close` on the original issue — it only reads and produces a verdict
- Auto-complete oversampling requires distinguishing auto-completed from manually completed issues in beads (may need new label or event query)
- If `orch reject` isn't deployed yet (orch-go-c51kt), Phase 3 blocks — design Phase 1 and 2 to be independently valuable (selection + review without auto-rejection)
- Defect class exposure: **Class 5 (Contradictory Authority Signals)** — audit verdict disagrees with completion gate. Mitigation: LOW confidence gate, and audit is advisory (triggers reject which is reversible)
- Defect class exposure: **Class 6 (Duplicate Action)** — same completion audited twice. Mitigation: `audit:deep-review` label prevents re-selection (already in `recentClosedIssues()` filter)

**Areas needing further investigation:**
- Audit agent precision/recall after first 10 audits — measure false positive rate before expanding
- Whether `SuggestDowngrades()` in `daemonconfig` should also consider `RejectionRate` for upgrade (tightening compliance) decisions
- Whether audit should also check knowledge-producing skills (investigations, architect designs) or only implementation skills

**Success criteria:**
- `RejectedCount > 0` in learning stats after first audit cycle — proves the feedback loop is closed
- Audit agent produces verdicts with HIGH confidence in >70% of cases — proves semantic review is feasible
- At least 1 legitimate FAIL verdict in first month — proves the system detects actual quality issues (currently 0 negative signals)
- No manual steps required between selection and rejection — proves structural embedding works

---

## Design Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│  DAEMON PERIODIC LOOP                                               │
│                                                                     │
│  TaskAuditSelect (weekly, configurable)                             │
│    │                                                                │
│    ├─ Query: recent closed issues (7-day window)                    │
│    ├─ Filter: exclude already-audited (audit:deep-review label)     │
│    ├─ Weight: 60% from auto-completed pool, 40% all                │
│    ├─ Select: crypto-rand Fisher-Yates, pick N (default 2)         │
│    ├─ Label: audit:deep-review                                      │
│    └─ Spawn: investigation agent per selected issue                 │
│         │                                                           │
│         ▼                                                           │
│  ┌──────────────────────────────────┐                               │
│  │  AUDIT AGENT (investigation)     │                               │
│  │                                  │                               │
│  │  Reads:                          │                               │
│  │  - Original issue description    │                               │
│  │  - SYNTHESIS.md                  │                               │
│  │  - Git diff (workspace)          │                               │
│  │  - Test output (re-run tests)    │                               │
│  │  - AGENT_MANIFEST.json           │                               │
│  │                                  │                               │
│  │  Checks:                         │                               │
│  │  1. Intent match (issue↔work)    │                               │
│  │  2. Test coverage of changes     │                               │
│  │  3. Obvious problems in diff     │                               │
│  │  4. Verification depth review    │                               │
│  │                                  │                               │
│  │  Produces: AUDIT_VERDICT.md      │                               │
│  │  (verdict, category, confidence) │                               │
│  └──────────────┬───────────────────┘                               │
│                 │                                                    │
│                 ▼                                                    │
│  Daemon reads verdict on audit agent completion                     │
│    │                                                                │
│    ├─ PASS → remove audit:deep-review, emit audit.passed            │
│    ├─ FAIL (high confidence) → orch reject <id> "<reason>"          │
│    │    └─ agent.rejected event → learning.go RejectedCount++       │
│    └─ FAIL (low confidence) → label audit:needs-review              │
│                                                                     │
│  ┌──────────────────────────────────┐                               │
│  │  LEARNING LOOP                   │                               │
│  │                                  │                               │
│  │  ComputeLearning() now tracks:   │                               │
│  │  - RejectedCount per skill       │                               │
│  │  - RejectionRate per skill       │                               │
│  │  - Category breakdown            │                               │
│  │                                  │                               │
│  │  Feeds: SuggestDowngrades()      │                               │
│  │  (compliance adjustment)         │                               │
│  └──────────────────────────────────┘                               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## References

**Files Examined:**
- `cmd/orch/audit_cmd.go` — Existing audit selection infrastructure (crypto-rand, launchd install)
- `cmd/orch/reject_cmd.go` — Rejection verb with `agent.rejected` event emission
- `cmd/orch/daemon_periodic.go` — 27 periodic tasks, handler pattern
- `pkg/daemon/scheduler.go` — Periodic scheduler with `Register`/`IsDue`/`MarkRun` API
- `pkg/daemon/periodic_learning.go` — Learning refresh periodic task
- `pkg/events/learning.go` — `ComputeLearning()`, `SkillLearning` struct, switch statement gap
- `pkg/events/logger.go` — `AgentRejectedData`, `LogAgentRejected()`, event type constants
- `.kb/models/completion-verification/model.md` — CV model §7, §"Why No Agent-Judgment Gates?"
- `.kb/models/completion-verification/probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md` — 0 reworks, 37% auto-completed, friction asymmetry
- `.kb/models/knowledge-accretion/probes/2026-03-20-probe-judge-verdict-accretion-exploration.md` — Coverage gap #2 (agent performance degradation), effectiveness hierarchy

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md` — No agent-judgment gates (this design navigates it)
- **Issue:** orch-go-c51kt — `orch reject` command (prerequisite for Phase 3)
- **Probe:** `.kb/models/completion-verification/probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md` — Primary evidence for the broken feedback loop

---

## Investigation History

**2026-03-20 17:00:** Investigation started
- Initial question: How should daemon-driven random quality audit system be designed?
- Context: 0 negative signals / 1,113 completions; orch audit select exists but isn't wired; orch reject being built separately

**2026-03-20 17:10:** Context gathering complete
- Read audit_cmd.go, reject_cmd.go, daemon_periodic.go, scheduler.go, learning.go, CV model, human feedback probe, judge verdict probe
- Key finding: learning.go missing RejectedCount — the final link in the feedback chain is broken

**2026-03-20 17:20:** Fork navigation complete (5 forks)
- Fork 0 (placement): orch-go daemon — confirmed
- Fork 1 (scheduling): daemon periodic vs launchd — daemon periodic (structural > advisory)
- Fork 2 (sample rate): 2/week default, configurable — confirmed by existing audit_cmd constants
- Fork 3 (review mechanism): audit agent vs mechanical checks — audit agent (can assess intent match)
- Fork 4 (agent-judgment concern): post-hoc audit ≠ completion gate — navigated via timing/selection/consequence distinction
- Fork 5 (auto-complete weighting): 60/40 weighted toward auto-completed — highest-risk, lowest-verification pool

**2026-03-20 17:30:** Investigation completed
- Status: Complete
- Key outcome: 3-phase design with implementation-ready details; primary blocker is `RejectedCount` gap in learning.go
