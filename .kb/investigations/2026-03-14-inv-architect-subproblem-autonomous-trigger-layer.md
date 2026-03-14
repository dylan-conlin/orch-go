## Summary (D.E.K.N.)

**Delta:** The autonomous trigger layer should extend the Sense phase with a `PatternSignal` alongside gates and issues, using 7 detectors as periodic tasks that feed pattern signals into the OODA cycle — Option B (issue creation pipeline) over Option A (OODA extension) or Option C (new phase), because issue creation provides natural dedup, compliance, and audit trail via existing beads infrastructure.

**Evidence:** Analyzed 16 existing periodic tasks in scheduler.go, 3 existing auto-create patterns (synthesis_auto_create.go, knowledge_health.go, agreement_check.go), the OODA cycle in ooda.go, and 24 threads in .kb/threads/ — all follow the same Detect→Dedup→Create→MarkRun pattern.

**Knowledge:** The daemon's existing periodic task infrastructure is mature enough that adding new detectors is trivial (register + IsDue + MarkRun). The hard problem is not detection — it's creation/removal asymmetry: the system can create issues faster than agents can close them, leading to queue bloat. A global autonomous budget (max issues/day) with per-detector caps is the primary constraint needed.

**Next:** Implement in 3 phases: (1) PatternDetector interface + 3 detectors (recurring bugs, investigation orphans, thread staleness), (2) 4 more detectors + budget enforcement, (3) retirement/expiry mechanism for auto-created issues.

**Authority:** architectural — Cross-component design affecting daemon OODA cycle, issue creation pipeline, compliance system, and beads integration.

---

# Investigation: Autonomous Trigger Layer Design

**Question:** How should the daemon detect patterns in the knowledge base, codebase, and issue history, then autonomously create investigation/probe/synthesis work?

**Started:** 2026-03-14
**Updated:** 2026-03-14
**Owner:** architect (spawned by orchestrator)
**Phase:** Complete
**Next Step:** Implementation via 3-phase plan
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Compliance-coordination bifurcation (subproblems 1-3) | extends | Yes - read ooda.go, compliance.go, coordination.go | None |
| `.kb/decisions/2026-02-07-orchestrator-reflection-session-protocol.md` | constrains | Yes - reflection triggers use event+time-floor hybrid | None |
| `.kb/decisions/2026-02-14-model-staleness-detection.md` | pattern source | Yes - three-component Detect→Annotate→Queue | None |
| `.kb/threads/2026-03-12-creation-removal-asymmetry-adding-local.md` | constrains | Yes - creation/removal asymmetry is structural | None |

---

## Findings

### Finding 1: Three existing auto-create patterns establish the template

**Evidence:** The daemon already has three patterns that detect conditions and auto-create beads issues:

1. **synthesis_auto_create.go** (263 lines): Detects investigation clusters (5+ investigations on same topic, no model exists) → creates `triage:ready` issue with `daemon:synthesis` label. Dedup via label-based search. Uses `SynthesisAutoCreateService` interface for testability.

2. **knowledge_health.go** (167 lines): Detects when active kb quick entries exceed threshold → creates `triage:review` issue with `area:knowledge` label. Dedup via title substring matching in open issues.

3. **agreement_check.go**: Detects failing agreements → creates `triage:review` issues for error-severity failures. Dedup via label-based search.

All three follow identical structure:
```
ShouldRun*() → RunPeriodic*() → [check threshold] → [dedup] → [create issue] → MarkRun()
```

**Source:** `pkg/daemon/synthesis_auto_create.go:64-153`, `pkg/daemon/knowledge_health.go:92-117`, `pkg/daemon/scheduler.go:1-134`

**Significance:** The autonomous trigger layer doesn't require new infrastructure — it's an extension of an established pattern. Each new detector is ~100-200 lines following the same template. The `SynthesisAutoCreateService` interface pattern is the gold standard for testability.

---

### Finding 2: The OODA cycle has a clean extension point in Sense

**Evidence:** The `SenseResult` struct (ooda.go:14-22) currently collects two signals:
- `GateSignal` (compliance gates)
- `Issues` (ready queue from beads)

These flow to Orient→Decide→Act unchanged. A third signal type (`PatternSignals []PatternSignal`) would naturally extend Sense without modifying Orient/Decide/Act — because pattern detectors create issues (which appear in the next cycle's `Issues` list), not spawn decisions.

However, this extension point is unnecessary if pattern detectors create issues directly (Option B). The issues will appear in the ready queue on the next poll cycle, flowing through the existing OODA pipeline without modification.

**Source:** `pkg/daemon/ooda.go:14-43`, `pkg/daemon/daemon_periodic.go:31-143`

**Significance:** Two architectural options diverge here:
- **Option A** (extend Sense): Pattern signals flow through OODA → adds complexity to all 4 phases
- **Option B** (periodic tasks create issues): Detectors are periodic tasks that create beads issues → zero OODA modification needed

Option B is strictly better — it follows the existing synthesis_auto_create.go pattern and requires no OODA changes. Principle: "coherence over patches" — don't modify working abstractions when the existing architecture already supports the new capability.

---

### Finding 3: Thread staleness is a filesystem scan, not a beads query

**Evidence:** 24 threads exist in `.kb/threads/`, all with frontmatter:
```yaml
title: "..."
status: open
created: 2026-03-12
updated: 2026-03-12
resolved_to: ""
```

Threads are markdown files with a `status` field and date metadata. Staleness detection is a filesystem scan:
1. Glob `.kb/threads/*.md`
2. Parse frontmatter for `status: open` and `updated` date
3. Flag threads where `now - updated > threshold` (e.g., 7 days) and no probes exist

This follows the exact same pattern as plan_staleness.go (scan `.kb/plans/`, parse frontmatter, check progress).

**Source:** `.kb/threads/2026-03-12-creation-removal-asymmetry-adding-local.md:1-7`, `pkg/daemon/plan_staleness.go:97-153`

**Significance:** Thread staleness is the simplest detector to implement — pure filesystem scan, no git history or beads queries needed.

---

### Finding 4: Recurring bug detection requires beads query + package clustering

**Evidence:** The daemon's `IssueQuerier` interface already supports `ListIssuesWithLabel(label)` and `ListReadyIssues()`. To detect "3+ bugs in same package":

1. Query all closed bugs from recent window: `bd list --status=closed --type=bug --since=30d`
2. Extract file paths from titles/descriptions (using existing `extractFileReferences` from workgraph.go:247-257)
3. Cluster by Go package (directory prefix)
4. Flag packages with 3+ bugs

The file path extraction already exists (`filePathPattern` in workgraph.go:63), and Jaccard tokenization exists for title similarity. This detector builds on existing infrastructure.

**Source:** `pkg/daemon/workgraph.go:63-64,247-257`, `pkg/daemon/interfaces.go` (IssueQuerier)

**Significance:** Most detectors can compose from existing building blocks. The recurring bug detector needs one new beads query method (`ListClosedBugsInWindow(duration)`) but reuses file extraction and clustering logic.

---

### Finding 5: Creation/removal asymmetry demands an autonomous budget

**Evidence:** The thread at `.kb/threads/2026-03-12-creation-removal-asymmetry-adding-local.md` identifies a structural principle: "Adding is a single-agent action. Removing requires coordinating with unknown dependents." In the orch-go context: 15 gates added, 1 removed. Feature flags accumulate (73% never removed per FlagShark).

Applied to autonomous triggers: if 7 detectors each create issues at their natural rate, the ready queue will grow unboundedly. The current daemon has 20 spawns/hour rate limit, but no limit on *issue creation*. An investigation that takes 2 hours but creates 5 follow-up issues amplifies the queue by 5x per depth.

The existing `SynthesisAutoCreateThreshold` (default 5) is a per-detector threshold but there's no global budget across all autonomous issue creation.

**Source:** `.kb/threads/2026-03-12-creation-removal-asymmetry-adding-local.md`, `pkg/daemon/daemon.go:57` (RateLimiter), `pkg/daemon/synthesis_auto_create.go:77-79`

**Significance:** The autonomous budget is the single most important design constraint. Without it, the system creates work faster than it retires work. This is the "Layer -1 that removes gates that have become ceremony" from the thread.

---

### Finding 6: Hotspot acceleration requires git log, which is expensive

**Evidence:** Detecting "file gained N lines in M days" requires:
```bash
git log --since="30 days ago" --numstat -- pkg/daemon/
```
This shells out to git and parses numstat output. For a project with 65+ daemon files and 30 days of history, this could produce substantial output. The existing `HotspotChecker` interface checks current file size (`wc -l`) but doesn't track growth rate.

Growth rate detection adds a temporal dimension that the current snapshot-based checks don't have. Options:
- **Full git log**: Expensive but accurate (~2-5s per scan)
- **Diff against known baseline**: Store last-check line counts, diff against current
- **Piggyback on reflection**: `kb reflect` already runs git-based analysis

**Source:** `pkg/daemon/hotspot.go` (HotspotChecker interface), `pkg/daemon/hotspot_checker.go`

**Significance:** Hotspot acceleration is the only detector that requires git history. It should run on a long interval (6-24h) and is a Phase 2 candidate.

---

### Finding 7: Model contradiction detection already exists partially

**Evidence:** `kb reflect --type model-drift` (invoked in `model_drift_reflection.go:39`) already detects models whose `code_refs` have changed since `Last Updated`. The decision at `.kb/decisions/2026-02-14-model-staleness-detection.md` established a three-component pattern: Detect → Annotate → Queue.

What's missing: probes that *contradict* a model's claims (vs probes that exist for *stale* models). Contradiction detection requires:
1. Scan `.kb/models/*/probes/*.md` for probes with `contradicts` in their title/content
2. Check if the parent model was updated after the probe date
3. If not → unresolved contradiction

This is a filesystem scan similar to thread staleness, not a git query.

**Source:** `pkg/daemon/model_drift_reflection.go:1-74`, `.kb/decisions/2026-02-14-model-staleness-detection.md`

**Significance:** Model contradiction detection extends the existing model drift infrastructure with a complementary signal. It should be a separate periodic task (not bundled with model drift) to keep concerns separated.

---

## Synthesis

**Key Insights:**

1. **The autonomous trigger layer is not a new subsystem — it's a catalog of periodic tasks following an established pattern.** The daemon already has 16+ periodic tasks, 3 of which auto-create issues. Adding 7 more detectors is architectural extension, not architectural change. Each detector is ~100-200 lines, follows the `ShouldRun*/RunPeriodic*/MarkRun` template, and uses a service interface for testability.

2. **Option B (issue creation pipeline) is strictly superior to Option A (OODA extension) or Option C (new phase).** Pattern detectors create beads issues with `daemon:trigger` label + detector-specific sublabel. These issues appear in the ready queue on the next cycle, flowing through existing OODA → Orient → Decide → Act without modification. This gives us free compliance, prioritization, dedup, and audit trail.

3. **The autonomous budget is the hard problem, not detection.** Creation/removal asymmetry means the system can generate issues faster than agents close them. A global daily budget (e.g., 5 autonomous issues/day) with per-detector caps (e.g., 2/detector/day) prevents queue bloat. Retirement: auto-created issues not spawned within N days get auto-closed with `daemon:expired` label.

**Answer to Investigation Question:**

The daemon should detect patterns via 7 periodic tasks (registered in PeriodicScheduler), each implementing a `PatternDetector` interface. Detected patterns create beads issues with `daemon:trigger` label, which flow through the existing OODA pipeline. A global autonomous budget constrains creation rate, and an expiry mechanism retires stale auto-created issues.

---

## Structured Uncertainty

**What's tested:**

- ✅ Existing periodic task pattern works for auto-creating issues (verified: synthesis_auto_create.go creates and deduplicates issues successfully)
- ✅ PeriodicScheduler supports adding new tasks trivially (verified: 16 tasks already registered, `Register(name, enabled, interval)` API)
- ✅ File path extraction and title similarity exist in workgraph.go (verified: code read)
- ✅ Thread files have parseable frontmatter with status/updated dates (verified: read 24 thread files)
- ✅ Model drift detection already exists as periodic task (verified: model_drift_reflection.go)

**What's untested:**

- ⚠️ Global autonomous budget prevents queue bloat (hypothesis — need to observe actual creation rates)
- ⚠️ 7 detectors running on different intervals won't cause performance issues (hypothesis — each is lightweight, but cumulative shell-outs need measurement)
- ⚠️ Expiry mechanism for stale auto-created issues works without side effects (hypothesis — need to verify no valid issues get expired)
- ⚠️ Git log parsing for hotspot acceleration completes in reasonable time (hypothesis — not benchmarked)
- ⚠️ Compliance dial integration for autonomous triggers (hypothesis — should respect autonomous level for trigger frequency)

**What would change this:**

- If queue growth rate exceeds completion rate by 3x+, the budget needs to be lower or detectors need smarter prioritization
- If git log parsing takes >10s, hotspot acceleration should use a diff-against-baseline approach instead
- If the daemon's 60s poll cycle can't accommodate the additional periodic task checks, tasks need staggering

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Option B: Pattern detectors as periodic tasks creating beads issues | architectural | Cross-component: affects daemon, beads, scheduler, and compliance. Multiple valid approaches evaluated. |
| Global autonomous budget (5 issues/day) | architectural | Resource commitment with cross-system impact. Rate affects queue depth and agent throughput. |
| PatternDetector interface design | implementation | Standard interface pattern, reversible, follows existing codebase conventions. |
| Expiry mechanism for auto-created issues | architectural | Affects beads issue lifecycle, requires coordination with completion pipeline. |
| 7-detector catalog | implementation | Each detector is independent, reversible, and follows established patterns. |

### Recommended Approach: Pattern Detectors as Periodic Issue Creators (Option B)

**Summary:** Each pattern detector is a periodic task that scans for a condition, deduplicates against existing issues, and creates `triage:ready` beads issues when patterns are found. These issues flow through the normal OODA pipeline on the next cycle.

**Why this approach:**
- Zero modification to OODA cycle (Sense/Orient/Decide/Act unchanged)
- Free compliance enforcement (auto-created issues pass through same gates as human-created issues)
- Natural audit trail (beads issues are permanent records of detected patterns)
- Established pattern (synthesis_auto_create.go is the proven template)
- Testable (service interface pattern enables unit testing without beads/filesystem)

**Trade-offs accepted:**
- One-cycle delay between detection and spawn (acceptable: 60s latency for patterns that evolve over days)
- Issues accumulate in beads (mitigated by autonomous budget + expiry)
- No real-time pattern response (acceptable: patterns are slow-moving signals, not alerts)

### Alternative Approaches Considered

**Option A: Extend Sense phase with PatternSignals**
- **Pros:** Tighter OODA integration, patterns visible in every phase
- **Cons:** Requires modifying SenseResult, OrientResult, SpawnDecision, and Act — touching 4 proven structs for a feature that doesn't need real-time response. Defect class exposure: Class 0 (Scope Expansion) — scanner widens, consumer assumptions break.
- **When to use instead:** If patterns needed sub-cycle response time

**Option C: New 'Cognition' phase between Orient and Decide**
- **Pros:** Clean conceptual separation, patterns don't pollute existing phases
- **Cons:** Breaks the OODA acronym (OOCDA?), adds a phase to every cycle even when no patterns detected, requires threading pattern context through to Decide. Violates "coherence over patches" — adds structural complexity for a feature that's better served by existing periodic task infrastructure.
- **When to use instead:** If pattern detection requires access to Orient's prioritized issue list (it doesn't)

**Rationale for Option B:** The daemon already has 16 periodic tasks and 3 auto-create patterns. Adding 7 more follows the proven path. The OODA cycle should remain a clean spawn pipeline; pattern detection is orthogonal to spawn decisions.

---

### Core Interface Design

```go
// PatternDetector detects a specific pattern in the environment and produces
// trigger signals that may result in beads issue creation.
// Each detector is registered as a periodic task in the PeriodicScheduler.
type PatternDetector interface {
    // Name returns the detector's unique name (used as scheduler task name).
    Name() string

    // Detect scans for the pattern and returns any triggers found.
    // Must be idempotent — safe to call repeatedly.
    Detect() ([]PatternTrigger, error)
}

// PatternTrigger represents a detected pattern that may warrant investigation.
type PatternTrigger struct {
    // DetectorName identifies which detector produced this trigger.
    DetectorName string

    // Title is the human-readable issue title.
    Title string

    // Description provides context for the spawned agent.
    Description string

    // IssueType is the beads issue type (investigation, task, probe).
    IssueType string

    // Skill is the recommended skill for spawning.
    Skill string

    // Priority is the issue priority (0-4).
    Priority int

    // DedupKey is a stable string for deduplication.
    // If an open issue with this dedup key exists, skip creation.
    DedupKey string

    // Labels are additional beads labels beyond the standard daemon:trigger.
    Labels []string
}

// TriggerBudget controls the rate of autonomous issue creation.
type TriggerBudget struct {
    // MaxPerDay is the global maximum auto-created issues per day.
    MaxPerDay int

    // MaxPerDetector is the per-detector maximum per day.
    MaxPerDetector int

    // CreatedToday tracks issues created today (reset at midnight).
    CreatedToday int

    // PerDetectorToday tracks per-detector counts.
    PerDetectorToday map[string]int

    // LastReset is when counters were last reset.
    LastReset time.Time
}

// CanCreate checks if the budget allows creating an issue from the given detector.
func (b *TriggerBudget) CanCreate(detectorName string) bool {
    b.maybeReset()
    if b.CreatedToday >= b.MaxPerDay {
        return false
    }
    if b.PerDetectorToday[detectorName] >= b.MaxPerDetector {
        return false
    }
    return true
}

// Record records that an issue was created.
func (b *TriggerBudget) Record(detectorName string) {
    b.CreatedToday++
    if b.PerDetectorToday == nil {
        b.PerDetectorToday = make(map[string]int)
    }
    b.PerDetectorToday[detectorName]++
}

// TriggerOrchestrator manages all pattern detectors and coordinates issue creation.
// This is a daemon-level component, NOT local agent state — it queries beads
// and filesystem directly each cycle.
type TriggerOrchestrator struct {
    Detectors []PatternDetector
    Budget    *TriggerBudget
    Service   TriggerService
    Scheduler *PeriodicScheduler
}

// TriggerService provides I/O for the trigger orchestrator (testable interface).
type TriggerService interface {
    // HasOpenTriggerIssue checks if an open issue with the given dedup key exists.
    HasOpenTriggerIssue(dedupKey string) (bool, error)

    // CreateTriggerIssue creates a beads issue from a trigger.
    CreateTriggerIssue(trigger PatternTrigger) (string, error)

    // ExpireStaleIssues closes auto-created issues older than the given age.
    ExpireStaleIssues(maxAge time.Duration) (int, error)
}
```

### Detector Catalog (7 Detectors)

#### 1. Recurring Bug Areas (`recurring_bugs`)
- **Detection:** Query closed bugs from last 30 days, extract file paths, cluster by Go package, flag packages with 3+ bugs
- **Issue type:** `investigation`
- **Skill:** `investigation`
- **Interval:** 24h
- **DedupKey:** `recurring-bugs:{package-path}`
- **Phase:** 1

#### 2. Investigation Orphan Clusters (`investigation_orphans`)
- **Detection:** Scan `.kb/investigations/` for topics with N+ investigations but no corresponding `.kb/models/{topic}/` directory (extends synthesis_auto_create beyond reflection suggestions)
- **Issue type:** `task`
- **Skill:** `capture-knowledge` (synthesis)
- **Interval:** 24h
- **DedupKey:** `orphan-cluster:{topic-slug}`
- **Phase:** 1

#### 3. Thread Staleness (`thread_staleness`)
- **Detection:** Scan `.kb/threads/*.md`, parse frontmatter, flag `status: open` threads where `now - updated > 7d` and 0 probes reference the thread
- **Issue type:** `probe`
- **Skill:** `investigation`
- **Interval:** 24h
- **DedupKey:** `stale-thread:{thread-slug}`
- **Phase:** 1

#### 4. Model Contradictions (`model_contradictions`)
- **Detection:** Scan `.kb/models/*/probes/*.md` for probes containing "contradicts" (title or content), check if parent model's `Last Updated` date is after probe date — if not, contradiction is unresolved
- **Issue type:** `task`
- **Skill:** `capture-knowledge` (model update)
- **Interval:** 12h
- **DedupKey:** `model-contradiction:{model-slug}:{probe-filename}`
- **Phase:** 2

#### 5. Hotspot Acceleration (`hotspot_acceleration`)
- **Detection:** `git log --since="30 days ago" --numstat` → aggregate lines added per file → flag files growing >200 lines/month that aren't already in hotspot list
- **Issue type:** `investigation`
- **Skill:** `architect` (extraction review)
- **Interval:** 24h
- **DedupKey:** `hotspot-accel:{file-path}`
- **Phase:** 2

#### 6. Knowledge Decay (`knowledge_decay`)
- **Detection:** Scan `.kb/models/*/model.md` for models where no probes exist with dates within last 30 days (model not probed recently)
- **Issue type:** `probe`
- **Skill:** `investigation` (verification probe)
- **Interval:** 24h
- **DedupKey:** `knowledge-decay:{model-slug}`
- **Phase:** 2

#### 7. Skill Performance Drift (`skill_performance_drift`)
- **Detection:** From `events.ComputeLearning()`, detect skills whose success rate dropped below 50% in the last 7 days when it was previously above 70% (significant degradation)
- **Issue type:** `investigation`
- **Skill:** `investigation`
- **Interval:** 12h
- **DedupKey:** `skill-drift:{skill-name}`
- **Phase:** 2

### Dedup Strategy (Critical for Class 6: Duplicate Action)

Every trigger carries a `DedupKey`. Before creating an issue:
1. Search open issues with label `daemon:trigger` for matching dedup key in description
2. If found → skip (issue already exists for this pattern)
3. If not → check budget → create if budget allows

The dedup key is embedded in the issue description (not title) as a structured tag:
```
<!-- trigger:dedup:{key} -->
```

This avoids title-based substring matching (fragile) and uses exact string search instead.

### Compliance Integration

Pattern detectors respect the compliance dial:
- **Strict:** Detectors disabled (no autonomous issue creation)
- **Standard:** Detectors enabled with tight budget (3/day global, 1/detector)
- **Relaxed:** Normal budget (5/day global, 2/detector)
- **Autonomous:** Generous budget (10/day global, 3/detector)

### Skill Inference for Auto-Created Issues

Auto-created issues get explicit `skill:X` labels, bypassing the normal 4-level inference chain:
- Recurring bugs → `skill:investigation`
- Orphan clusters → `skill:capture-knowledge`
- Thread staleness → `skill:investigation`
- Model contradictions → `skill:capture-knowledge`
- Hotspot acceleration → `skill:architect`
- Knowledge decay → `skill:investigation`
- Skill performance drift → `skill:investigation`

This is explicit and auditable — no inference ambiguity.

### Retirement/Expiry Mechanism

**Problem:** Auto-created issues that aren't spawned within N days are stale — the pattern may have resolved or been superseded.

**Solution:** New periodic task `trigger_expiry` (interval: 24h):
1. Query open issues with `daemon:trigger` label
2. Check issue `created` date
3. If `now - created > 14 days` → close with comment "Auto-expired: pattern trigger not acted upon within 14 days"
4. Label with `daemon:expired`

This is the "Layer -1 that removes gates" from the creation/removal asymmetry thread.

### Dashboard Integration

New snapshot in daemon status file:
```go
type TriggerSnapshot struct {
    ActiveDetectors  int       `json:"active_detectors"`
    BudgetUsed       int       `json:"budget_used"`
    BudgetMax        int       `json:"budget_max"`
    LastTriggered    time.Time `json:"last_triggered"`
    IssuesCreated    int       `json:"issues_created_today"`
    IssuesExpired    int       `json:"issues_expired_today"`
}
```

---

### Implementation Phases

**Phase 1: Foundation + 3 Detectors** (recommended: 1 implementation session)
1. `PatternDetector` interface, `PatternTrigger` type, `TriggerBudget`
2. `TriggerOrchestrator` with `TriggerService` interface
3. 3 detectors: `recurring_bugs`, `investigation_orphans`, `thread_staleness`
4. Register `TaskTriggerScan` in scheduler
5. Wire into `daemon_periodic.go`
6. Tests using mock `TriggerService`

**Phase 2: Remaining Detectors + Budget** (1 session)
1. 4 more detectors: `model_contradictions`, `hotspot_acceleration`, `knowledge_decay`, `skill_performance_drift`
2. Compliance dial integration for budget tiers
3. Dashboard snapshot integration
4. Tests for all detectors

**Phase 3: Retirement Mechanism** (1 session)
1. `trigger_expiry` periodic task
2. Auto-close stale daemon:trigger issues
3. `daemon:expired` label
4. Observability: track creation/expiry ratio in events.jsonl

---

### Things to watch out for:
- ⚠️ **Class 6 (Duplicate Action)**: Dedup key search must use exact string match on `<!-- trigger:dedup:{key} -->` tag, not title substring. Title-based dedup is fragile with similar detectors.
- ⚠️ **Class 3 (Stale Artifact Accumulation)**: Without the expiry mechanism (Phase 3), auto-created issues will accumulate. Phase 3 is not optional — it's load-bearing.
- ⚠️ **Class 0 (Scope Expansion)**: Each new detector widens what the daemon "sees." Be conservative with thresholds — false positives create noise that degrades trust in the trigger system.
- ⚠️ **Shell-out cost**: 7 detectors × filesystem/beads queries = significant I/O per interval. Stagger intervals (not all at 24h) to spread load.

### Success criteria:
- ✅ Daemon creates investigation issues for detected patterns without human initiation
- ✅ Queue depth doesn't grow unboundedly (budget enforced, measured in events.jsonl)
- ✅ Auto-created issues get spawned and produce useful findings (>50% completion rate within 14 days)
- ✅ Stale issues get expired automatically (creation/expiry ratio trends toward 1.0)
- ✅ No duplicate issues created for same pattern (dedup key prevents Class 6)

---

## References

**Files Examined:**
- `pkg/daemon/ooda.go` — OODA cycle structure (Sense/Orient/Decide/Act)
- `pkg/daemon/scheduler.go` — PeriodicScheduler with 16 named tasks
- `pkg/daemon/workgraph.go` — Per-cycle dedup/removal signal computation
- `pkg/daemon/knowledge_health.go` — Existing auto-create pattern (knowledge maintenance)
- `pkg/daemon/synthesis_auto_create.go` — Existing auto-create pattern (synthesis issues)
- `pkg/daemon/periodic_learning.go` — Learning refresh + compliance auto-adjust
- `pkg/daemon/plan_staleness.go` — Existing staleness detection pattern
- `pkg/daemon/daemon.go` — Main Daemon struct and loop
- `pkg/daemon/compliance.go` — Compliance dial levels
- `pkg/daemon/coordination.go` — Issue routing
- `pkg/daemon/allocation.go` — Skill-aware scoring
- `cmd/orch/daemon_periodic.go` — Periodic task orchestration from main loop
- `.kb/threads/2026-03-12-creation-removal-asymmetry-adding-local.md` — Creation/removal asymmetry thread
- `.kb/decisions/2026-02-14-model-staleness-detection.md` — Three-component staleness pattern
- `.kb/decisions/2026-02-07-orchestrator-reflection-session-protocol.md` — Reflection triggers

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-14-model-staleness-detection.md` — Detect→Annotate→Queue pattern
- **Thread:** `.kb/threads/2026-03-12-creation-removal-asymmetry-adding-local.md` — Structural constraint on creation

---

## Investigation History

**2026-03-14:** Investigation started
- Initial question: How should the daemon detect patterns and auto-create investigation work?
- Context: Architect subproblem 1 of autonomous trigger layer design

**2026-03-14:** Exploration complete — 7 findings across OODA, periodic tasks, filesystem, and beads
- Read all daemon package files (~65 source files)
- Evaluated 3 architectural options (A: OODA extension, B: issue creation, C: new phase)
- Cataloged 7 pattern detectors with detection logic

**2026-03-14:** Investigation completed
- Status: Complete
- Key outcome: Option B (periodic tasks creating beads issues) is the recommended approach, with 7 detectors implemented in 3 phases, constrained by a global autonomous budget and expiry mechanism
