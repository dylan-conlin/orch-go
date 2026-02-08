## TLDR

The current single-source SQLite design answers the right problem but over-rotates on architecture relative to current bottlenecks. Recommendation: revise before full rollout by narrowing scope to command-write state + explicit reconciliation, treat dashboard/daemon as primary consumers, and define hard migration gates to avoid indefinite dual-write drift.

## Summary (D.E.K.N.)

**Delta:** Evaluating the 7 review questions shows the design should not proceed unchanged; it needs a narrower, explicitly staged architecture.

**Evidence:** Primary-source code review shows session-list filtering already reduced the 5.8s cliff path, `orch phase` is not implemented yet, SPAWN context still instructs `bd comment`, and state writes are currently non-fatal.

**Knowledge:** The winning distinction is runtime convenience vs authoritative truth: SQLite can be a fast runtime projection, but without strict reconciliation and ownership rules it will recreate registry-style drift.

**Next:** Revise key forks (phase ingestion, failure/reconciliation model, migration gates) before promoting this to implementation.

**Authority:** architectural - This crosses orch/beads/OpenCode boundaries and changes source-of-truth semantics.

---

# Investigation: Evaluate Single Source Agent State

**Question:** Is the proposed single-source SQLite architecture sufficient as designed, when stress-tested against 7 critical review questions, or should it be revised first?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Architect Agent (orch-go-21372)
**Phase:** Complete
**Next Step:** Revise design doc with constrained v1 scope and explicit migration gates
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-02-06-design-single-source-agent-state.md` | extends | Yes | Partial - migration/adoption assumptions too optimistic |
| `.kb/investigations/2026-02-06-inv-agent-state-field-level-audit.md` | confirms | Yes | None |
| `.kb/investigations/2026-02-06-inv-opencode-fork-audit-session-lifecycle-integration.md` | confirms | Yes | None |
| `.kb/investigations/2026-02-04-inv-agents-own-declaration-via-bd.md` | deepens | Yes | None |
| `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` | confirms | Yes | Risk of repeating drift if SQLite authority is premature |

---

## What I tried

1. Read the proposed design and all listed prerequisite artifacts.
2. Verified claims against primary sources in current code paths (`cmd/orch/*`, `pkg/state/*`, `pkg/spawn/context.go`).
3. Checked implementation readiness assumptions (phase command existence, template adoption, read-path migration).

## What I observed

- The new state package exists and is partially integrated on command write paths, but the status read path still performs multi-source reconciliation.
- `orch phase` is referenced in design intent but no command file currently exists.
- Spawn templates still direct agents to `bd comment "Phase: ..."` as the active protocol.
- Current state DB writes are non-fatal in `spawn`, `complete`, and `abandon`, which is safe for cache semantics but unsafe for declared authority semantics.

## Test performed

- `go test ./pkg/state/...` -> `ok github.com/dylan-conlin/orch-go/pkg/state 0.116s`

---

## Findings

### Finding 1: Q1 and Q6 - the design targets a real pain but is oversized for current validated bottlenecks

**Evidence:** `status_cmd` already uses server-side session filtering (`?start`) with inline note that unfiltered 89-session list was 5.8s and filtered path is <10ms; distributed-join costs still remain, but the original cliff is partially mitigated in the common path.

**Source:** `cmd/orch/status_cmd.go:199`, `cmd/orch/status_cmd.go:202`, `cmd/orch/status_cmd.go:211`, `.kb/investigations/2026-02-06-inv-agent-state-field-level-audit.md:294`

**Significance:** The case for a full always-on materializer is weaker for CLI reads alone; the strongest justification is continuous-read surfaces (dashboard/daemon), not one-off `orch status` calls.

---

### Finding 2: Q2 and Q3 - operational risk and drift risk are still under-specified for "single source" semantics

**Evidence:** State writes are explicitly non-fatal in spawn/complete/abandon command paths; no materializer/reconciliation ownership loop exists in current primary code; registry drift history already shows that "best effort writes + eventual assumptions" fails over time.

**Source:** `cmd/orch/spawn_cmd.go:885`, `cmd/orch/complete_cmd.go:1133`, `cmd/orch/abandon_cmd.go:420`, `pkg/state/integration.go:13`, `.kb/decisions/2026-01-12-registry-is-spawn-cache.md:14`

**Significance:** This is correct behavior if SQLite is treated as a cache/projection. It is insufficient if SQLite is declared authority without hard reconciliation contracts.

---

### Finding 3: Q5 - adoption path for `orch phase` is not implementation-ready

**Evidence:** No phase command implementation file exists under `cmd/orch/*phase*.go`; current generated spawn context content still instructs `bd comment` for phase transitions.

**Source:** `cmd/orch/*phase*.go` (no matches), `pkg/spawn/context.go:189`, `pkg/spawn/context.go:317`

**Significance:** The design's runtime authority model assumes an agent behavior switch that has not landed in command surface or templates; dual protocol period likely extends, increasing divergence risk.

---

### Finding 4: Q4 and Q7 - migration control and conflict authority are underspecified

**Evidence:** Design proposes phased dual-write + shadow-read, but current implementation does not include shadow discrepancy policy, bailout thresholds, or a hard end condition for dual-write. The proposed runtime/audit split is directionally consistent with prior orthogonal-dimensions model but lacks tie-break rules when systems disagree.

**Source:** `.kb/investigations/2026-02-06-design-single-source-agent-state.md:223`, `.kb/investigations/2026-02-06-design-single-source-agent-state.md:230`, `.kb/investigations/2026-02-04-inv-agents-own-declaration-via-bd.md:70`

**Significance:** Without explicit discrepancy budgets, owner escalation, and terminal migration criteria, the system risks living in permanent dual-write ambiguity.

---

## Synthesis

### Per-question assessment (7 critical questions)

1. **Over-architecting past simpler fix?** -> **Needs revision.** Keep the SQLite direction, but v1 should prioritize immutable + command-written fields and defer full event materialization until dashboard/daemon benefits are quantified.
2. **Operational surface area too large?** -> **Needs revision.** Define operator, failure modes, and degraded-read behavior before adding always-on SSE + backfill + polling loops.
3. **Repeating registry drift pattern?** -> **Needs revision.** Single-source claims require mandatory reconciliation plus strict "authoritative field ownership" map; otherwise it is a new registry.
4. **Dual-write timeline realistic?** -> **Needs revision.** Add explicit bailout and forcing functions (deadline + measurable exit criteria) to avoid indefinite shadow mode.
5. **Will `orch phase` be adopted?** -> **Insufficient today.** Command and template migration are not landed; default protocol remains `bd comment`.
6. **UX improvement vs complexity?** -> **Needs reframing.** Primary beneficiary appears to be continuous readers (serve/dashboard/daemon); CLI speedup alone no longer justifies full architecture.
7. **Beads-audit vs runtime separation clean?** -> **Conceptually yes, operationally incomplete.** Separation matches orthogonal-state model, but conflict resolution policy must be explicit.

### Recommended architectural posture

**Proceed, but revise specific forks before implementation-scale rollout.**

Adopt a constrained **Projection-First** plan:

- **Phase A (low-risk):** SQLite as projection of immutable + command-owned fields (`spawn`, `complete`, `abandon`) with periodic read-reconciliation job.
- **Phase B (measured):** Add phase/runtime ingestion only after `orch phase` exists and template migration reaches target adoption.
- **Phase C (gated):** Promote SQLite to primary runtime read source only when shadow discrepancy SLO is met for a fixed window and bail-out path is tested.

---

## Structured Uncertainty

**What's tested:**

- ✅ State DB package compiles and tests pass (`go test ./pkg/state/...`).
- ✅ Current status command still uses multi-source query path (code inspection).
- ✅ No `orch phase` command implementation currently exists (file search).

**What's untested:**

- ⚠️ End-to-end discrepancy rates under real dual-write traffic (no shadow telemetry yet).
- ⚠️ Operational burden of SSE+backfill+polling over long-running daemon uptime.
- ⚠️ User-visible gain split between CLI and dashboard under production-like load.

**What would change this:**

- If shadow-read shows <0.1% discrepancy for 2 weeks with actionable auto-reconcile, faster cutover is justified.
- If dashboard/daemon latency remains bottlenecked after Phase A projection, deeper materialization is justified.
- If `orch phase` reaches near-universal usage via enforced templates/gates, phase authority can move cleanly.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Revise design to constrained projection-first rollout | architectural | Cross-component semantics and migration policy |
| Add explicit discrepancy SLO + bailout policy before dual-write | architectural | Governs system trust and cutover safety |
| Keep beads canonical for closure while SQLite owns runtime projection | architectural | Boundary decision across two state systems |

### Recommended Approach ⭐

**Revise and proceed with a constrained Projection-First architecture** rather than full single-source materializer in v1.

**Why this approach:**
- Retains the strategic direction (fast local state) without repeating registry drift conditions.
- Aligns with evidence hierarchy by matching authority claims to currently implemented guarantees.
- Preserves graceful degradation while reducing operational surface area.

**Trade-offs accepted:**
- Slower path to complete single-source semantics.
- Temporary continued use of multi-source reads for some fields.
- Additional reconciliation instrumentation work before cutover.

**Implementation sequence:**
1. Add explicit field ownership + discrepancy policy (authoritative map, tie-breaks, escalation).
2. Land `orch phase` command + template migration + adoption telemetry.
3. Run bounded shadow-read with pre-defined exit/bail criteria; only then cut over runtime reads.

### Alternative Approaches Considered

**Option B: Proceed exactly as designed now**
- **Pros:** Faster conceptual convergence.
- **Cons:** High operational and drift risk before adoption/reconciliation primitives are proven.
- **When to use instead:** If immediate dashboard latency is a hard blocker and operator capacity is available.

**Option C: Abandon SQLite effort and optimize existing joins only**
- **Pros:** Lowest near-term complexity.
- **Cons:** Leaves architectural complexity distributed and fragile; does not reduce long-term reconciliation burden.
- **When to use instead:** If project priority shifts away from dashboard/daemon runtime quality.

**Rationale for recommendation:** Option A captures most durable value while respecting current implementation reality and prior drift lessons.

---

## References

**Files Examined:**
- `.kb/investigations/2026-02-06-design-single-source-agent-state.md` - baseline design and migration proposal
- `.kb/investigations/2026-02-06-inv-agent-state-field-level-audit.md` - field mutability and cost map
- `.kb/investigations/2026-02-06-inv-opencode-fork-audit-session-lifecycle-integration.md` - integration option analysis
- `.kb/investigations/2026-02-04-inv-agents-own-declaration-via-bd.md` - orthogonal state model
- `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` - prior drift precedent
- `cmd/orch/status_cmd.go` - current hot path behavior and filtering
- `cmd/orch/spawn_cmd.go` - state write integration semantics
- `cmd/orch/complete_cmd.go` - completion write integration semantics
- `cmd/orch/abandon_cmd.go` - abandonment write integration semantics
- `pkg/state/db.go` and `pkg/state/integration.go` - SQLite schema/behavior
- `pkg/spawn/context.go` - active phase-reporting template instructions

**Commands Run:**
```bash
bd show orch-go-21372
go test ./pkg/state/...
```

---

## Investigation History

**2026-02-06 13:41:** Investigation initialized and linked to issue
- Created artifact from template and reported path to beads.

**2026-02-06 13:50:** Substrate and primary-source review complete
- Read all required artifacts and verified key claims against code.

**2026-02-06 14:05:** Evaluation completed
- Outcome: recommend revising forks before full single-source rollout.
