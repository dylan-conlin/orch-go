## Summary (D.E.K.N.)

**Delta:** Plans need exactly one new command (`orch plan hydrate`) and one completion gate to bridge prose plans to beads execution. The gap is narrow: infrastructure for both halves exists, they just aren't connected.

**Evidence:** 9 plans in `.kb/plans/`, 0 have `**Beads:**` fields populated. `gate-signal-vs-noise` plan: 4 phases, 0 beads issues. `orch plan show` already parses phases and overlays beads status — but there's nothing to overlay. `complete_architect.go` creates single issues, not decompositions. Architect skill Phase 5d documents manual decomposition but it's not enforced.

**Knowledge:** The Mar 5 investigation already designed the three-part decomposition (graph in beads, narrative in .kb/plans/, binding in plan references). The plan artifact format is designed and in use. `orch plan show/status` already reads plans and queries beads. The missing piece is the bridge: plan phases → beads issues.

**Next:** Implement `orch plan hydrate <slug>` (create beads issues from plan phases, write IDs back to plan) + advisory gate in completion pipeline (warn when architect produces multi-phase work without beads issues).

**Authority:** architectural — touches completion pipeline, adds new command, establishes plan-beads binding pattern

---

# Design: Plan-Beads Integration Gap

**Question:** How to bridge plans (prose in `.kb/plans/`) to beads (execution state in `.beads/`) so orchestrator frame collapse doesn't occur?

**Started:** 2026-03-11
**Owner:** architect (orch-go-865v3)
**Phase:** Complete
**Status:** Complete

---

## The Problem

Plans and beads are two halves of the same coordination problem, currently disconnected:

```
Plans (.kb/plans/)              Beads (.beads/)
├── Phasing rationale           ├── Work items
├── Blocking logic              ├── Dependencies
├── Strategic narrative         ├── Status tracking
├── Decision points             ├── Phase comments
└── Cross-project awareness     └── Label-based discovery

         NO CONNECTION
```

**Evidence of the gap:**

1. `gate-signal-vs-noise` plan: 4 phases with clear deliverables, exit criteria, and dependencies. Zero beads issues created. When it's time to execute Phase 1, the orchestrator must either do it itself (frame collapse) or create issues ad-hoc (losing phasing rationale).

2. All 9 existing plans: None have populated `**Beads:**` fields in their phase sections. Plans are prose-only.

3. `complete_architect.go`: Creates a single implementation issue from architect synthesis. Doesn't handle decomposition into multiple phase-aligned issues with dependencies.

**Frame collapse pattern:**
```
Orchestrator creates plan with 4 phases
    ↓
Time to execute Phase 1
    ↓
No beads issues exist → no daemon pickup
    ↓
Orchestrator spawns agents directly (losing phasing context)
    OR orchestrator does the work itself (frame collapse)
    ↓
Phases 2-4 have no beads representation → sequence lost
```

---

## What Already Exists

| Component | Status | Gap |
|-----------|--------|-----|
| Plan artifact format | Working (9 plans exist) | No `**Beads:**` fields populated |
| `orch plan show` | Working (parses phases, overlays beads status) | Nothing to overlay |
| `orch plan status` | Working (summary view) | Shows phases without beads binding |
| `orch plan create` | Working (delegates to `kb create plan`) | Creates plan but no issues |
| `complete_architect.go` | Working (single issue creation) | No decomposition support |
| Plan phase parser | Working (`parsePlanContent`, `PlanPhase` struct) | `BeadsIDs` field always empty |
| Beads dependency model | Working (`bd dep add`, blocking semantics) | Not connected to plan phases |
| Architect skill Phase 5d | Documented (manual decomposition procedure) | Not enforced or automated |

**Key insight:** The infrastructure on both sides is mature. This is a wiring problem, not a design problem.

---

## Findings

### Finding 1: The Bridge Is One Command — `orch plan hydrate`

**What it does:** Reads a plan's phases, creates beads issues for each phase, adds inter-phase dependencies, and writes the beads IDs back into the plan file.

```bash
orch plan hydrate gate-signal-vs-noise
```

**Before:**
```markdown
### Phase 1: Gate census and classification
**Goal:** Enumerate every gate, classify as signal/noise/unknown
**Beads:**
**Depends on:** Nothing
```

**After:**
```markdown
### Phase 1: Gate census and classification
**Goal:** Enumerate every gate, classify as signal/noise/unknown
**Beads:** orch-go-a1b2
**Depends on:** Nothing
```

**Implementation sketch:**

```go
// cmd/orch/plan_hydrate.go

func hydratePlan(slug string) error {
    // 1. Find and parse the plan
    plan := findPlanBySlug(plans, slug)

    // 2. For each phase without beads issues:
    for i, phase := range plan.Phases {
        if len(phase.BeadsIDs) > 0 {
            continue // already hydrated
        }

        // 3. Create beads issue from phase
        title := fmt.Sprintf("Plan: %s — Phase %d: %s", plan.Title, i+1, phase.Name)
        issue, err := beads.FallbackCreate(title, phase.Goal, "task", 2, labels, "")

        // 4. Track created ID
        createdIDs[i] = issue.ID
    }

    // 5. Add inter-phase dependencies
    for i, phase := range plan.Phases {
        if phase.DependsOn references phase j {
            beads.FallbackDepAdd(createdIDs[i], createdIDs[j])
        }
    }

    // 6. Write beads IDs back into plan file
    updatePlanWithBeadsIDs(planPath, createdIDs)
}
```

**Why a command, not automatic:** Plans are created by humans and architects. Some plans are aspirational/draft. Hydration should be an explicit decision: "this plan is ready to execute, create the work items."

**Labels:** Issues get `triage:ready` (for daemon pickup) + `plan:<slug>` (for plan-level queries).

### Finding 2: Inter-Phase Dependencies Map to Beads Blocking

Plan phases have explicit `**Depends on:**` fields. These map directly to beads blocking dependencies:

```
Phase 2 depends on Phase 1
    ↓
bd dep add <phase-2-issue> <phase-1-issue>   (blocks relationship)
    ↓
bd ready shows Phase 1 issues only
    ↓
Daemon spawns Phase 1 work
    ↓
Phase 1 issues closed → Phase 2 unblocks → daemon spawns Phase 2
```

This is exactly how beads dependencies already work. The `**Depends on:**` field in plans is natural-language; hydration parses it to identify which phase is referenced (by number or name).

**Parsing `Depends on:`:** The field typically says "Phase 1", "Phases 1-3", "Nothing", or "Phase 1 ({beads-id} blocks {beads-id})". Hydration should handle:
- "Nothing" / "none" → no dependencies
- "Phase N" → block on phase N's issues
- "Phases N-M" → block on all issues in phases N through M
- Already has beads IDs → use directly

### Finding 3: Plan-Level Labels Enable Phase-Aware Queries

Adding `plan:<slug>` labels to hydrated issues enables:

```bash
# All issues for a plan
bd list -l plan:gate-signal-vs-noise

# Combined with orch plan show (already works):
orch plan show gate-signal-vs-noise
# Shows phases with live beads status overlay
```

This also enables daemon awareness: the daemon could check if a ready issue belongs to a plan and include plan context in SPAWN_CONTEXT.

### Finding 4: The Completion Gate Is Advisory — "Plan Has Unhydrated Phases"

When an architect produces a multi-phase design (investigation with Implementation Recommendations containing phases), the completion pipeline should check:

```
Architect completes
    ↓
Synthesis has NextActions with 3+ items OR investigation has Phases section
    ↓
Check: does a .kb/plans/ artifact exist referencing this work?
    ↓
Check: does that plan have populated **Beads:** fields?
    ↓
If no plan or no beads: ADVISORY WARNING
    "Architect produced multi-phase design with no plan hydration.
     Run: orch plan hydrate <slug>"
```

**Advisory, not blocking** — per behavioral grammars (start advisory, measure compliance, graduate to blocking). Measured via: count of architect completions with multi-phase output that lack plan hydration.

This extends `complete_architect.go` naturally. Currently it only creates a single implementation issue. With this change:
- Single-component recommendations → single issue (existing behavior)
- Multi-component/multi-phase recommendations → advisory to use `orch plan hydrate`

### Finding 5: Orchestrator Plan Context Injection Prevents Frame Collapse

When the orchestrator session starts, `orch orient` should show active plans with their beads progress. This already partially exists (the investigation from Mar 5 designed it). The key addition: if a plan has hydrated phases, orient shows which phases are ready/blocked/complete:

```
Active Plan: Gate Signal vs Noise (4 phases)
  [x] Phase 1: Gate census (orch-go-a1b2 — closed)
  [~] Phase 2: Fix noise gates (orch-go-c3d4 — in_progress)
  [ ] Phase 3: Retrospective audit (orch-go-e5f6 — blocked by Phase 1)
  [ ] Phase 4: Prospective measurement (blocked by Phases 1-3)
```

This gives the orchestrator enough context to NOT re-derive the plan or do the work itself.

---

## Synthesis

**The gap is narrow.** Both halves of the bridge exist:
- Plans with phased structure, parsed by `plan_cmd.go`
- Beads with dependency mechanics, queried by plan show

**What's missing is the connector:**

1. **`orch plan hydrate <slug>`** — Creates beads issues from plan phases, adds dependencies, writes IDs back to plan. Makes plans executable.

2. **Advisory gate in completion pipeline** — When architect produces multi-phase work without plan hydration, warn. Prevents the "plan exists but has no beads issues" state from persisting.

3. **`orch orient` plan injection** — Show active plans with beads-derived progress in orchestrator session start. Prevents frame collapse by giving the orchestrator the plan context without re-derivation.

**What this does NOT do:**
- Does NOT make beads subsume plan functionality (plans stay in .kb/, beads stays in .beads/)
- Does NOT auto-create plans from architect synthesis (plans are still human/architect decisions)
- Does NOT add cross-project beads dependencies (manual binding in plan narrative, per prior decision)
- Does NOT make plan hydration blocking (advisory first, measure compliance)

**Implementation sequence:**

| Step | What | Where | Depends On |
|------|------|-------|------------|
| 1 | `orch plan hydrate <slug>` | `cmd/orch/plan_hydrate.go` | Nothing |
| 2 | Write-back of beads IDs to plan file | Part of step 1 | Nothing |
| 3 | Advisory gate in architect completion | `cmd/orch/complete_architect.go` | Step 1 exists |
| 4 | Orient plan injection | `pkg/orient/plans.go` | Steps 1-2 (plans need beads IDs) |

Steps 1-2 are one issue. Step 3 is one issue. Step 4 is one issue. Three implementation issues total.

---

## Decision Points

### Decision 1: Should hydration create one issue per phase or one issue per deliverable?

**Context:** A phase like "Fix noise gates" has multiple deliverables (fix self_review, add hotspot reason recording, evaluate verified/explain_back). Should hydration create 1 issue for the phase or 3 issues for the deliverables?

**Options:**
- **A: One issue per phase** — Simple, maps 1:1 to plan structure, daemon spawns one agent per phase.
- **B: One issue per deliverable** — More granular, enables parallel work within a phase, but plan write-back gets complex (multiple IDs per phase).

**Recommendation:** A — one issue per phase. The agent working the phase can create sub-issues if needed. Plan→beads mapping should be 1:1 for simplicity. If a phase needs decomposition, the agent doing it creates the sub-issues (per existing architect decomposition pattern).

**Status:** Recommended, awaiting orchestrator decision.

### Decision 2: Should hydration be idempotent?

**Context:** If `orch plan hydrate` is run twice on the same plan, should it skip already-hydrated phases or error?

**Recommendation:** Idempotent — skip phases that already have `**Beads:**` populated. This is safe because:
- Plan might be partially hydrated (some phases done, new phases added)
- Re-running after adding a phase should only create the new phase's issues
- Beads issue creation already has deduplication

**Status:** Recommended.

### Decision 3: What labels go on hydrated issues?

**Options:**
- **A:** `triage:ready` only — daemon picks up immediately
- **B:** `triage:ready` + `plan:<slug>` — daemon picks up + plan-level query
- **C:** `triage:ready` + `plan:<slug>` + `skill:<inferred>` — full routing

**Recommendation:** B — `triage:ready` + `plan:<slug>`. Skill inference is unreliable from plan phase descriptions and should happen at daemon triage time (existing pattern).

**Status:** Recommended.

---

## Structured Uncertainty

**What's tested:**
- Plan parser correctly extracts phases, goals, dependencies (verified: `plan_cmd.go` + tests)
- Beads dependency model blocks correctly (verified: beads integration guide)
- `orch plan show` overlays beads status on phases (verified: `formatPlanShow`)
- Advisory gates work in completion pipeline (verified: existing completion gate patterns)

**What's untested:**
- Whether `**Depends on:**` field parsing can reliably identify phase references (natural language parsing)
- Whether orchestrators will run `orch plan hydrate` when creating plans (behavioral compliance)
- Whether plan-level labels (`plan:<slug>`) create useful daemon routing or just noise
- Whether one-issue-per-phase is granular enough for daemon-spawned agents

**What would change this design:**
- If orchestrators consistently skip hydration → make it automatic at plan creation time
- If phases are too coarse for agents → switch to one-issue-per-deliverable
- If cross-project plans need hydration → add `--project <dir>` flag to hydrate in other repos

---

## Implementation Issues (to create)

1. **`orch plan hydrate` command** — Parse plan phases, create beads issues with dependencies, write IDs back to plan file. Core bridge.

2. **Architect completion advisory gate** — When architect produces multi-phase design without plan hydration, warn during `orch complete`. Extends `complete_architect.go`.

3. **Orient plan progress injection** — Show active hydrated plans with beads-derived progress in `orch orient` output. Prevents orchestrator frame collapse.

---

## References

**Files examined:**
- `.kb/plans/2026-03-11-gate-signal-vs-noise.md` — evidence of gap (4 phases, 0 beads)
- `.kb/plans/2026-03-10-harness-health-improvement.md` — another example (4 phases, 0 beads)
- `.kb/investigations/2026-03-05-inv-design-orchestrator-coordination-plans-persist.md` — prior design work
- `.kb/decisions/2026-03-10-architect-owns-decomposition-plan-beads-issu.md` — decision on ownership
- `.kb/decisions/2026-02-26-plan-mode-incompatible-with-daemon-spawned-agents.md` — plan mode constraints
- `.kb/global/models/planning-as-decision-navigation.md` — planning model
- `.kb/guides/beads-integration.md` — beads mechanics
- `cmd/orch/plan_cmd.go` — existing plan infrastructure
- `cmd/orch/complete_architect.go` — existing architect completion
- `skills/src/worker/architect/SKILL.md` — architect decomposition procedure
