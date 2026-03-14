## Summary (D.E.K.N.)

**Delta:** Designed a three-tier decision protocol that classifies 19 daemon decision types by reversibility and blast radius, modulates tiers through the existing compliance dial, and uses events.jsonl as the decision log with no new local state.

**Evidence:** Analysis of coordination.go (binary routing), compliance.go (4-level dial with DeriveX pattern), autoadjust.go (safety asymmetry), verification_tracker.go (pause-after-N precedent), events/logger.go (25+ extensible event types), and daemon_loop.go (signal/status machinery).

**Knowledge:** The daemon already makes ~15 distinct decision types but only classifies 2 (completion routing, compliance downgrades). The compliance dial provides the modulation axis. The VerificationTracker provides the escalation precedent. Events.jsonl provides the decision log without violating No Local Agent State.

**Next:** Implement in 6 phases starting with types+logging (zero behavior change), then Tier 2 veto, Tier 3 pending, dashboard, removal mechanism, learning feedback loop.

**Authority:** architectural — Cross-component design touching daemon, events, daemonconfig, and dashboard with multiple valid approaches evaluated.

---

# Investigation: Architect Subproblem — Decision Protocol Design (Three-Tier Autonomy)

**Question:** How should the daemon classify the decisions it makes into three tiers of autonomy (autonomous, propose-and-act, genuine decision), and how should each tier's escalation path work asynchronously?

**Started:** 2026-03-14
**Updated:** 2026-03-14
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None — ready for implementation decomposition
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Subproblem 1 (Autonomous Trigger Layer) | extends | Template only, pending | None — trigger layer produces decisions, this protocol classifies them |
| `.kb/guides/decision-authority.md` | substrate | Yes — read directly | None — agent authority guide maps cleanly to daemon decision authority |
| `~/.kb/principles.md` | substrate | Yes — Verification Bottleneck, Asymmetric Velocity | None — principles guide tier thresholds |

---

## Findings

### Finding 1: The Daemon Already Makes ~15 Distinct Decision Types Without Classification

**Evidence:** Traced through coordination.go, compliance.go, daemon_loop.go, and periodic task code. The daemon makes decisions in 6 categories:

| Category | Decisions | Current Classification |
|----------|-----------|----------------------|
| **Spawn** | Select issue, infer skill/model, route extraction, architect escalate | Implicit Tier 1 (just does it) |
| **Completion** | Auto-complete light, auto-complete full, label for review | Binary: auto vs review (coordination.go:150-174) |
| **Knowledge** | Create synthesis issue, model drift issue, agreement check issue | Implicit Tier 1 (auto-creates) |
| **Lifecycle** | Reset orphan, resume stuck agent, flag phase timeout | Implicit Tier 1 (auto-acts) |
| **Compliance** | Downgrade compliance level | Implicit Tier 2 (acts but logged) |
| **Work Graph** | Detect duplicates, surface removal candidates | Advisory only (logs, no action) |

**Source:** `pkg/daemon/coordination.go:150-174` (RouteCompletion), `pkg/daemon/ooda.go:110-180` (Decide phase), `cmd/orch/daemon_loop.go:360-486` (spawn cycle)

**Significance:** The decision taxonomy already exists implicitly. The gap is that all decisions are treated as either "just do it" or "label for human review" — no middle ground, no veto window, no classification by risk.

---

### Finding 2: The Compliance Dial Provides the Modulation Axis

**Evidence:** `pkg/daemonconfig/compliance.go` defines 4 levels with per-skill/model/combo resolution. The pattern of `DeriveX(ComplianceLevel)` functions (lines 96-148) is the established way to compute behavioral parameters from compliance level. Five such functions already exist:

- `DeriveVerificationThreshold` — how many auto-completions before pause
- `DeriveInvariantThreshold` — how many violations before pause
- `DeriveArchitectEscalationEnabled` — whether to escalate to architect
- `DeriveSynthesisRequired` — whether SYNTHESIS.md is required
- `DerivePhaseEnforcement` — "required" vs "advisory"

**Source:** `pkg/daemonconfig/compliance.go:96-148`

**Significance:** Adding `ClassifyDecision(class, level)` follows the exact same pattern. The compliance dial already modulates autonomy — this design just extends it to decision classification.

---

### Finding 3: Safety Asymmetry is an Established Pattern

**Evidence:** `pkg/daemonconfig/autoadjust.go` implements compliance auto-downgrade with:
- Only downgrades (less strict), never upgrades (line 29)
- One step at a time: strict -> standard -> relaxed -> autonomous (line 56)
- Requires 10+ samples and 80%+ success rate (lines 6-12)
- Conservative threshold prevents overreaction to small samples

**Source:** `pkg/daemonconfig/autoadjust.go:6-13` (constants), lines 30-68 (SuggestDowngrades)

**Significance:** Same safety asymmetry applies to decision tier evolution: the system can learn to increase autonomy (demote Tier 2 to Tier 1), but should never self-promote caution. Exception: high veto/rejection rate should auto-promote (safety mechanism).

---

### Finding 4: Events.jsonl is the Right Decision Log

**Evidence:** The events system (`pkg/events/logger.go`) already has 25+ event types. Each event is a self-contained JSON line with type, timestamp, session_id, and data map. The `LearningStore` (`pkg/events/learning.go`) computes aggregates from events.jsonl without local state cache.

**Source:** `pkg/events/logger.go:14-73` (event types), `pkg/events/learning.go:44-167` (ComputeLearning)

**Significance:** Adding `decision.*` event types follows established pattern. The learning store can be extended for per-decision-class acceptance rates. Respects No Local Agent State constraint.

---

### Finding 5: The VerificationTracker Provides the Escalation Precedent

**Evidence:** `pkg/daemon/verification_tracker.go` implements a "do N things autonomously, then pause for human input" pattern:
- Counts completions since last human verification (line 62-86)
- Pauses daemon when threshold reached (line 80)
- Resumes via signal file `~/.orch/daemon-resume.signal` (line 183-190)
- Seeded from backlog to survive restarts (line 125-136)

The signal file mechanism (`checkDaemonSignals` in `daemon_loop.go:488-514`) provides async human-to-daemon communication without requiring a session.

**Source:** `pkg/daemon/verification_tracker.go:46-56`, `cmd/orch/daemon_loop.go:488-514`

**Significance:** Tier 2 veto and Tier 3 pending use the same signal-file pattern. Tier 3 also uses beads issues for richer approve/reject responses.

---

### Finding 6: Work Graph Detects But Doesn't Act on Removal Candidates

**Evidence:** `daemon_loop.go:710-731` (`runWorkGraphAnalysis`) computes the work graph each cycle, detects title duplicates and removal candidates, but only logs them.

**Source:** `cmd/orch/daemon_loop.go:710-731`

**Significance:** The work graph already computes the signals that removal decisions would consume. The decision protocol upgrades "log and move on" to "classify, act at appropriate tier, track outcome."

---

## Synthesis

**Key Insights:**

1. **The classification function is a matrix, not a switch** — Decision tier = f(decision_class, compliance_level). The compliance dial modulates the base tier, giving 4 different behavior profiles from one design. This avoids the N*M problem of configuring each decision individually.

2. **Optimistic execution with veto is the right middle ground** — Tier 2 acts immediately and offers a veto window. This matches the daemon's async nature: Dylan may not respond for hours, so blocking on approval would make Tier 2 equivalent to Tier 3. The risk is acceptable because all Tier 2 actions are reversible by definition.

3. **Soft-delete makes all removals inherently safe** — The creation/removal asymmetry dissolves when removals are label-based (not deletion). Every removal becomes at most Tier 2, because undoing it is a single label change.

**Answer to Investigation Question:**

The daemon should classify decisions using a `ClassifyDecision(class, compliance)` function that maps 19 decision types to base tiers (T1/T2/T3), then adjusts by compliance level:
- **strict** promotes all tiers one level (more cautious)
- **standard** uses base tiers
- **relaxed** demotes one level (more autonomous)
- **autonomous** makes everything T1 except base-T3 which becomes T2

Each tier has a distinct escalation path:
- **T1:** Act + log. Dashboard "Decisions" feed. No notification.
- **T2:** Act + veto timer (1-24h by category). Desktop notification. Signal file for veto.
- **T3:** Block + create `decision:pending` beads issue. Notification. Wait for close.

The decision log uses events.jsonl (new `decision.*` event types) and the learning store feedback loop adjusts tiers based on acceptance rates over time (same safety asymmetry as compliance downgrades).

---

## Structured Uncertainty

**What's tested:**

- ✅ Compliance dial modulation produces correct tier matrix (manual trace through `adjustForCompliance` with all 4 levels × 3 base tiers = 12 cases)
- ✅ All 19 decision types classified with reversibility/blast radius justification (traced through codebase to verify each exists)
- ✅ Events.jsonl pattern confirmed compatible (25+ existing event types, same structure)
- ✅ VerificationTracker signal file pattern confirmed viable for Tier 2 veto mechanism

**What's untested:**

- ⚠️ Veto timer UX — will Dylan actually check within 1-4 hours? (needs real usage data)
- ⚠️ Dashboard decisions panel — design is conceptual, no mockup
- ⚠️ Learning store feedback loop thresholds (90%/30%) — hypothetical, need calibration
- ⚠️ Removal staleness thresholds (30/60 days) — need calibration from actual data

**What would change this:**

- If Dylan never vetos Tier 2 decisions → collapse T1 and T2 into one tier
- If the learning feedback loop adjusts tiers too aggressively → add stability window (max 1 adjustment per week per class)
- If beads issue creation for Tier 3 creates too much noise → switch to dedicated `~/.orch/decisions-pending.jsonl`

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|---|---|---|
| Decision taxonomy + classification function | architectural | Cross-component (daemon, daemonconfig, events) |
| Decision event types in events.jsonl | implementation | Follows established event pattern |
| Tier 2 veto timer mechanism | architectural | New async communication pattern |
| Tier 3 pending issue mechanism | implementation | Uses existing beads pattern |
| Removal mechanism (soft delete) | architectural | Affects knowledge base lifecycle |
| Dashboard decisions panel | implementation | Extends existing serve API |
| Learning store decision tracking | implementation | Extends existing ComputeLearning |

### Recommended Approach: Phased Implementation

**Phased implementation** — Build classification/logging first (value without risk), then add escalation mechanisms incrementally.

**Why this approach:**
- Phase 1 (types + logging) delivers immediate observability with zero behavior change
- Phase 2 (Tier 2 veto) is the highest-value addition (most decisions are T2 at standard compliance)
- Phase 3 (Tier 3 pending) requires beads integration, only needed for rare strategic decisions
- Later phases (dashboard, removal, learning) can be parallelized

**Trade-offs accepted:**
- Tier 2 veto is optimistic-execution — if Dylan vetos, the action has already been done and must be undone. Acceptable because T2 actions are reversible by definition.
- The feedback loop adds complexity to ComputeLearning. Acceptable because the pattern already exists for compliance downgrades.

**Implementation sequence:**

1. **Phase 1: Decision Types + Classification** — Add `DecisionTier`, `DecisionClass` to `pkg/daemonconfig/decision.go`. Add `decision.*` event types to `pkg/events/logger.go`. Wire into existing decision points. Tests: classification matrix, compliance modulation.

2. **Phase 2: Tier 2 Veto Timer** — Add `DecisionTracker` to daemon. Signal file for veto. CLI: `orch daemon veto <id>`. Desktop notification. Tests: timer expiry, veto processing, undo.

3. **Phase 3: Tier 3 Pending Issues** — Create beads issue with `decision:pending` label. Daemon polls for closure. Extract approval/rejection. CLI: `orch daemon approve/reject <id>`.

4. **Phase 4: Dashboard Integration** — `/api/decisions` endpoint. Active veto display with countdown. Pending decisions with approve/reject. Recent decisions feed.

5. **Phase 5: Removal Mechanism** — Periodic staleness detection. Soft-delete for issues, investigations, models. Wire through decision protocol.

6. **Phase 6: Learning Feedback Loop** — Extend ComputeLearning with DecisionLearning. Tier demotion suggestions. Auto-promotion on high rejection rate.

### Alternative Approaches Considered

**Option B: Separate decision.jsonl file**
- **Pros:** Clean separation, simpler querying
- **Cons:** Violates No Local Agent State (new state file), ComputeLearning can't consume it, dashboard needs two files
- **When to use instead:** If events.jsonl becomes too large for decision querying

**Option C: All decisions go through beads issues**
- **Pros:** Maximum visibility, beads is existing coordination substrate
- **Cons:** Massive overhead for Tier 1 (~10/hour), beads becomes noisy
- **When to use instead:** If events.jsonl proves insufficient for dashboard

**Rationale:** Events.jsonl extension respects No Local Agent State, follows established patterns, scales to high-frequency Tier 1 decisions without noise.

---

### Implementation Details

**What to implement first:**
- Phase 1 is pure addition (new types, new events, wire into existing decision points)
- No behavior changes, immediate observability value
- Foundation for all subsequent phases

**Things to watch out for:**
- ⚠️ Tier 2 undo mechanics must be idempotent — same veto processed twice should not double-undo
- ⚠️ Decision IDs must be globally unique — use UUIDs (events.jsonl is append-only)
- ⚠️ Veto timer needs to survive daemon restarts — store deadlines in events.jsonl, rehydrate on startup
- ⚠️ Class 6 risk: a Tier 3 decision creating a beads issue could trigger another decision about that issue — need dedup by target
- ⚠️ Class 5 risk: compliance dial and decision tier must derive from single source — `adjustForCompliance` ensures this

**Success criteria:**
- ✅ `orch daemon status` shows decision tier distribution
- ✅ Dashboard "Decisions" feed shows last 20 decisions with tier/outcome
- ✅ Tier 2 veto works end-to-end: daemon acts -> Dylan vetos -> action undone
- ✅ Tier 3 pending: daemon creates issue -> Dylan closes -> daemon executes
- ✅ Learning store tracks decision acceptance rates and suggests tier adjustments

---

## References

**Files Examined:**
- `pkg/daemon/coordination.go` — RouteCompletion (binary routing), RouteIssueForSpawn
- `pkg/daemon/compliance.go` — SpawnGateSignal, CheckIssueCompliance, VerifyCompletionCompliance
- `pkg/daemonconfig/compliance.go` — ComplianceLevel, 5 DeriveX functions
- `pkg/daemonconfig/autoadjust.go` — SuggestDowngrades safety asymmetry pattern
- `pkg/daemonconfig/config.go` — Full Config struct (57 fields)
- `pkg/daemon/daemon.go` — Daemon struct, OODA loop composition
- `pkg/daemon/ooda.go` — Sense/Orient/Decide/Act phases
- `pkg/daemon/verification_tracker.go` — Pause-after-N pattern, signal files
- `pkg/daemon/periodic_learning.go` — Learning refresh + compliance auto-adjust
- `pkg/events/logger.go` — 25+ event types, Logger, event logging
- `pkg/events/learning.go` — LearningStore computation from events.jsonl
- `cmd/orch/daemon_loop.go` — Main loop, signal handling, completion processing, work graph
- `.kb/guides/decision-authority.md` — Agent decision authority framework
- `~/.kb/principles.md` — Verification Bottleneck, Asymmetric Velocity, Safety Asymmetry

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-14-inv-architect-subproblem-autonomous-trigger-layer.md` — Trigger layer that produces decisions for this protocol
- **Guide:** `.kb/guides/decision-authority.md` — Agent-level authority that maps to daemon-level tiers

---

## Investigation History

**2026-03-14:** Investigation started
- Initial question: How to classify daemon decisions into three tiers of autonomy
- Context: Subproblem 3 of autonomous trigger layer architecture

**2026-03-14:** Exploration complete
- Read 14 source files across daemon, daemonconfig, events packages
- Mapped existing decision points, compliance modulation, safety patterns
- Identified 19 distinct decision types in 7 categories

**2026-03-14:** Investigation completed
- Status: Complete
- Key outcome: Three-tier protocol with compliance-modulated classification, events.jsonl decision log, signal-file veto for Tier 2, beads-issue pending for Tier 3, soft-delete removal mechanism, and learning feedback loop for tier adjustment
