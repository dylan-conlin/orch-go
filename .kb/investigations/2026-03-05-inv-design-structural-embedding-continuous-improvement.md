## Summary (D.E.K.N.)

**Delta:** 5 of 8 feedback loop commands are already structurally embedded (3 in daemon periodic, 2 in orient) but results don't flow between layers; the remaining 3 (stats summaries, drift, friction aggregation) need homes, and the solved bridge pattern is daemon-status.json.

**Evidence:** Code analysis of orient_cmd.go (8 data sources), daemon_periodic.go (9 periodic tasks), complete_pipeline.go (7 advisory types), and daemon status.go (snapshot bridge pattern).

**Knowledge:** The gap isn't "commands don't run" — it's "results don't surface where decisions happen." Daemon runs reflect/health/drift periodically but results stay in daemon logs. Orient reads events.jsonl but not daemon snapshots. The fix is connecting existing producers to existing consumers via the daemon-status.json bridge.

**Next:** Implement in 4 phases — Phase 1 (orient health summary) has highest impact per effort.

**Authority:** architectural — Cross-component design affecting orient, debrief, daemon, and completion pipeline interactions.

---

# Investigation: Design Structural Embedding for Continuous Improvement Feedback Loops

**Question:** Where should each feedback loop command (stats, reflect, hotspot, backlog cull, triage review, drift, changelog, bd stats) be structurally embedded so it gets used without relying on someone remembering to run it?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-pcfwd)
**Phase:** Complete
**Next Step:** None — ready for implementation
**Status:** Complete

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-05-inv-design-friction-capture-protocol-worker.md | extends | Yes — reviewed capture format | None — friction capture defines the capture point; this design defines the aggregation/surfacing layer |

---

## Findings

### Finding 1: Most commands are already running — results just don't surface

**Evidence:** The daemon already runs 9 periodic tasks including reflection (1h), model drift (4h), knowledge health (2h), cleanup (6h), recovery (5m), orphan detection (30m), phase timeout (5m), question detection (5m), and agreement check (30m). Orient already collects throughput, ready issues, plans, threads, model freshness, and focus. The completion pipeline already runs 7+ advisories.

**Source:** `cmd/orch/daemon_periodic.go:27-92`, `cmd/orch/orient_cmd.go:60-116`, `cmd/orch/complete_pipeline.go:199-420`

**Significance:** The problem is not "commands don't run" — it's "results don't flow to decision points." The daemon runs kb reflect every hour but orient never reads the results. Orient computes throughput but doesn't include completion rate or abandonment rate from `orch stats`. The commands themselves work; the wiring between layers is missing.

---

### Finding 2: daemon-status.json is the solved bridge pattern

**Evidence:** The daemon already writes `~/.orch/daemon-status.json` every poll cycle with snapshots from periodic tasks (knowledge health, phase timeout, question detection, agreement check). `orch status` and `orch serve` both read this file. The pattern is: daemon computes → writes snapshot → consumers read snapshot.

**Source:** `pkg/daemon/status.go:12-63` (DaemonStatus struct), `cmd/orch/daemon.go:838-890` (WriteStatusFile calls)

**Significance:** No new bridge mechanism needed. Extend the existing DaemonStatus struct with new snapshot fields for reflection results, beads health, and friction accumulation. Orient reads the file; debrief reads the file. Zero new infrastructure.

---

### Finding 3: Orient has 8 data sources — adding 6 more would cause bloat

**Evidence:** Orient currently collects: (1) throughput, (2) previous session, (3) ready issues with KB context, (4) active plans, (5) active threads, (6) relevant models, (7) stale models, (8) focus. The formatted output is already substantial.

**Source:** `cmd/orch/orient_cmd.go:60-116`, `pkg/orient/orient.go` (OrientationData struct)

**Significance:** Adding stats+reflect+drift+changelog+friction+beads as separate sections would make orient unusable. The solution is aggregation: ONE "System Health" section that summarizes all signals in a compact format. Individual signals remain available via their standalone commands for drill-down.

---

### Finding 4: Debrief is the natural home for session-retrospective feedback

**Evidence:** Debrief already auto-populates "What Happened" from events.jsonl and includes quality checks. It runs at session end when the orchestrator has context on what the session accomplished. Drift analysis ("how aligned was this session with focus?") and friction summary ("what friction did agents report?") are session-retrospective questions.

**Source:** `cmd/orch/debrief_cmd.go:81-168`, `pkg/debrief/debrief.go:1-50` (DebriefData struct)

**Significance:** Debrief currently captures WHAT happened but not HOW WELL it went. Adding drift and friction sections completes the feedback loop: orient says "here's where we are" → session work → debrief says "here's how it went and what to improve."

---

### Finding 5: Only hotspot enforcement benefits from threshold pressure

**Evidence:** Hotspot spawn gates (Layer 1) already block implementation spawns on CRITICAL files. This works because the consequence is directly proportional to the risk — spawning on a bloated file without architect review causes accretion. Other feedback signals (stale reflect data, drift from focus, high abandonment rate) are informational — blocking on them would be disproportionate ceremony.

**Source:** `cmd/orch/hotspot.go`, CLAUDE.md "Accretion Boundaries" section, `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md`

**Significance:** Strategy 3 (threshold pressure) should not expand beyond hotspots. The principle "Gate Over Remind" applies when the cost of ignoring the signal is high and immediate. For feedback loops, the cost of ignoring is gradual (knowledge rot, drift) — advisory surfacing is proportionate, blocking is not.

---

## Synthesis

**Key Insights:**

1. **The gap is wiring, not capability** — Most feedback commands already run. The issue is that daemon periodic results (reflect, health, drift) don't flow to orient/debrief where decisions happen. The daemon-status.json pattern solves this.

2. **Aggregation prevents bloat** — Instead of 6 new sections in orient, one "System Health" summary section with per-signal status indicators keeps orient compact while surfacing all signals. Drill-down available via standalone commands.

3. **Orient = forward-looking, Debrief = backward-looking** — This natural split determines embedding: orient gets health summary + changelog (what do I need to know?), debrief gets drift + friction (how did this session go?). Daemon computes both and caches results.

**Answer to Investigation Question:**

The embedding map is:

| Command | Embed In | Mechanism | Already Done? |
|---------|----------|-----------|---------------|
| orch stats (throughput) | Orient → Health Summary | Enhance existing throughput with rates | Partial — throughput exists, rates missing |
| kb reflect | Orient → Health Summary | Read daemon-status.json reflection snapshot | Daemon runs it; orient doesn't read it |
| orch hotspot | Completion advisory + Spawn gate | Already embedded | Yes |
| orch backlog cull | Orient → Health Summary | Count stale issues from bd | Claimed embedded but actually manual |
| orch review triage | Orient → Health Summary | Count triage:review issues | Claimed embedded but actually manual |
| orch drift | Debrief → Focus Alignment | Compare session events to focus.json | Not done |
| orch changelog | Orient → Changes Since Last Session | Shell to orch changelog --since <last-debrief-date> | Not done |
| bd stats | Daemon periodic + Orient → Health Summary | New daemon periodic task, write to status.json | Not done |
| Friction data | Debrief → Session Friction + Daemon accumulation | Parse friction comments from completions | Capture designed (orch-go-sw58d), aggregation not |

---

## Structured Uncertainty

**What's tested:**

- ✅ daemon-status.json bridge pattern works (verified: status.go, 6 existing snapshot types flow from daemon → serve/status)
- ✅ Orient data collection pattern works (verified: orient_cmd.go collects from 8 sources, adds to OrientationData)
- ✅ Debrief auto-population pattern works (verified: debrief_cmd.go auto-populates from events.jsonl)
- ✅ Daemon periodic task framework is extensible (verified: 9 tasks follow identical ShouldRun/RunPeriodic pattern)

**What's untested:**

- ⚠️ Health summary rendering — compact enough for orchestrator consumption (needs prototype)
- ⚠️ Changelog in orient — performance impact of cross-repo git scanning at session start
- ⚠️ Friction accumulation daemon task — detection of recurring patterns from sparse comments
- ⚠️ Optimal health thresholds (abandonment rate > 20% = yellow is a hypothesis)

**What would change this:**

- If orient runtime exceeds 5 seconds, changelog must be async (daemon-cached, not orient-computed)
- If friction comments are too unstructured for pattern detection, accumulation task should just count categories
- If daemon-status.json grows too large (>100KB), snapshot pruning or separate files needed

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Health Summary section in orient | architectural | Cross-component: orient reads daemon snapshots, new data contract |
| Debrief drift + friction sections | architectural | Cross-component: debrief reads events + beads comments |
| New daemon periodic tasks | implementation | Follows established pattern, stays within daemon scope |
| No new threshold pressure | architectural | Policy decision about blocking vs advisory |

### Recommended Approach ⭐

**Layered Embedding via Daemon-Status Bridge** — Connect existing feedback producers to existing consumers using the daemon-status.json snapshot pattern, with a compact Health Summary aggregation in orient.

**Why this approach:**
- Zero new infrastructure — extends proven patterns (daemon snapshots, orient data collection)
- Bloat-resistant — one summary section instead of N separate sections
- Progressive — each phase is independently useful
- Principle: Session Amnesia — health status persists across sessions via daemon-status.json

**Trade-offs accepted:**
- Health summary is lossy (counts not details) — acceptable because drill-down commands still exist
- Daemon must be running for health data to be fresh — acceptable because daemon is always running in normal operation
- Changelog at orient time may be slow for large repos — mitigate by caching in daemon

**Implementation sequence:**

#### Phase 1: Orient Health Summary (Highest impact, ~200 lines)

Add `collectHealthSummary()` to orient_cmd.go that reads daemon-status.json + computes quick signals:

```go
// New field in OrientationData
type HealthSummary struct {
    OverallStatus    string         `json:"overall_status"`    // green/yellow/red
    Throughput       HealthSignal   `json:"throughput"`
    KBHealth         HealthSignal   `json:"kb_health"`
    Hotspots         HealthSignal   `json:"hotspots"`
    Backlog          HealthSignal   `json:"backlog"`
    FrictionWeekly   HealthSignal   `json:"friction_weekly"`
    BeadsHealth      HealthSignal   `json:"beads_health"`
}

type HealthSignal struct {
    Status  string `json:"status"`  // green/yellow/red
    Summary string `json:"summary"` // "12 completions/8h (85% rate)"
}
```

**Data sources for each signal:**
- Throughput: events.jsonl (already parsed, add completion/abandonment rates)
- KB Health: daemon-status.json → KnowledgeHealth snapshot
- Hotspots: `orch hotspot --json` (cache result, expensive)
- Backlog: `bd list --status=open` filtered to P3/P4 stale
- Friction: events.jsonl friction events (after Phase 2 friction capture lands)
- Beads Health: `bd stats` or daemon-status.json

**Thresholds:**

| Signal | Green | Yellow | Red |
|--------|-------|--------|-----|
| Throughput | Abandonment < 15% | 15-30% | > 30% |
| KB Health | < 30 synthesis opps | 30-60 | > 60 |
| Hotspots | 0 CRITICAL | 1-2 CRITICAL | 3+ CRITICAL |
| Backlog | < 10 stale P3/P4 | 10-20 | > 20 |
| Friction | < 5/week | 5-10/week | > 10/week or same item 3+ |
| Beads Health | < 50 open | 50-100 | > 100 |

**Formatted output:**
```
## System Health [GREEN]
  Throughput: 12 completions/8h (85% completion rate)
  KB: 23 synthesis opportunities (top cluster: context)
  Hotspots: 1 HIGH (session.go)
  Backlog: 8 stale P3/P4 issues
  Friction: 2 reports this week (1 ceremony, 1 tooling)
  Beads: 42 open issues
```

**File targets:** `cmd/orch/orient_cmd.go` (add collectHealthSummary), `pkg/orient/orient.go` (add HealthSummary type)

#### Phase 2: Debrief Enrichment (~150 lines)

Add two new sections to DebriefData:

```go
// New fields in DebriefData
FocusAlignment  *DriftSummary   `json:"focus_alignment,omitempty"`
FrictionSummary []FrictionItem  `json:"friction_summary,omitempty"`
```

**Focus alignment:** Compare session spawn events against focus.json goal. Count aligned vs off-focus spawns. Report percentage.

**Friction summary:** Parse `Friction:` comments from agent.completed events in this session. Categorize by type (bug/gap/ceremony/tooling). List unique items.

**File targets:** `cmd/orch/debrief_cmd.go` (add collection functions), `pkg/debrief/debrief.go` (add types + rendering)

#### Phase 3: New Daemon Periodic Tasks (~200 lines)

Add two new periodic tasks following the established pattern:

**3a. Beads Health Check (interval: 2h)**
```go
func (d *Daemon) RunPeriodicBeadsHealth() *BeadsHealthResult
```
- Run `bd stats` or `bd list --status=open | wc -l`
- Count open issues, stale issues (>14d without activity), priority distribution
- Write BeadsHealthSnapshot to daemon-status.json

**3b. Friction Accumulation (interval: 4h)**
```go
func (d *Daemon) RunPeriodicFrictionAccumulation() *FrictionAccumulationResult
```
- Scan events.jsonl for `friction.*` events in the last 7 days
- Count by category (bug/gap/ceremony/tooling)
- Detect recurring items (same description appearing 3+ times)
- Write FrictionAccumulationSnapshot to daemon-status.json

**File targets:** `pkg/daemon/periodic.go` (add ShouldRun/Run methods), `pkg/daemon/status.go` (add snapshot types), `pkg/daemonconfig/config.go` (add interval fields), `cmd/orch/daemon_periodic.go` (add handler functions)

#### Phase 4: Changelog in Orient (~100 lines, nice-to-have)

Add "Changes Since Last Session" to orient:
- Read last debrief date from `.kb/sessions/`
- Shell out to `orch changelog --since <date> --json`
- Extract notable changes (breaking changes, cross-skill impact)
- Add as compact section between "Previous Session" and "Ready Issues"

**Performance concern:** Cross-repo git scanning can be slow. If >3s, move to daemon periodic (6h interval) and read from cache.

**File targets:** `cmd/orch/orient_cmd.go` (add collectChangelog)

### Alternative Approaches Considered

**Option B: Embed each command as separate orient section**
- **Pros:** Direct, complete data visible
- **Cons:** Orient becomes 14+ sections, wall of text, orchestrator skips most of it. Violates "Gate Over Remind" in reverse — too much data is as bad as too little.
- **When to use instead:** Never for orient. Acceptable for JSON output mode where machine consumers parse what they need.

**Option C: Threshold pressure on all stale signals**
- **Pros:** Forces engagement with feedback loops
- **Cons:** Disproportionate ceremony. Blocking orient on stale reflect data when the orchestrator wants to check agent status is punishing the user. Principle: "Coherence Over Patches" — gates should match the cost of the error they prevent.
- **When to use instead:** Only when ignoring the signal causes immediate, proportional harm (hotspot → accretion). Informational signals → advisory only.

**Rationale for recommendation:** Option A (layered embedding via bridge) respects the existing architecture, follows proven patterns, and addresses the core problem (results don't surface where decisions happen) without over-engineering. It adds ~650 lines across 4 phases, each independently shippable.

---

### Implementation Details

**What to implement first:**
- Phase 1 (orient health summary) delivers the most value per effort
- Can ship without Phase 2-4 and already improves visibility
- Phase 2 (debrief) should follow quickly as it completes the orient↔debrief feedback loop

**Things to watch out for:**
- ⚠️ Daemon-status.json read failure in orient — must fail gracefully (show "daemon: offline" not crash)
- ⚠️ Hotspot computation is expensive (~2-5s) — cache in daemon, not compute in orient
- ⚠️ Class 0 (Scope Expansion) — health summary must have a fixed signal count, not grow unboundedly
- ⚠️ Class 5 (Contradictory Authority) — health thresholds must be defined in ONE place (orient package), not duplicated

**Areas needing further investigation:**
- Optimal health thresholds need calibration against actual data (run `orch stats --days 7` and `bd stats` to establish baselines before hardcoding)
- Friction accumulation pattern detection — may need iteration after friction capture protocol lands (orch-go-sw58d)

**Success criteria:**
- ✅ `orch orient` shows health summary section without any additional flags
- ✅ All 6 health signals populated from existing data sources
- ✅ Orient runtime stays under 5 seconds (no regression from health summary)
- ✅ Debrief shows focus alignment and friction summary
- ✅ Daemon writes beads health and friction accumulation snapshots

---

## References

**Files Examined:**
- `cmd/orch/orient_cmd.go` — Orient command implementation, 8 data sources
- `cmd/orch/debrief_cmd.go` — Debrief command, auto-population from events
- `cmd/orch/complete_pipeline.go` — Completion advisories, 7+ advisory types
- `cmd/orch/daemon_periodic.go` — Periodic task orchestrator, 9 tasks
- `pkg/daemon/daemon.go` — Daemon struct with 15+ tracker fields
- `pkg/daemon/periodic.go` — ShouldRun/RunPeriodic patterns
- `pkg/daemon/status.go` — DaemonStatus struct, snapshot bridge pattern
- `pkg/daemonconfig/config.go` — All daemon periodic intervals
- `cmd/orch/hotspot.go` — 4 hotspot signal types
- `cmd/orch/changelog.go` — Cross-repo commit categorization
- `cmd/orch/backlog_cmd.go` — Stale P3/P4 issue culling
- `cmd/orch/stats_cmd.go` — Event aggregation
- `cmd/orch/status_cmd.go` — Swarm status display
- `.kb/investigations/2026-03-05-inv-design-friction-capture-protocol-worker.md` — Friction capture protocol design

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Hotspot enforcement layers (precedent for proportional gating)
- **Investigation:** `2026-03-05-inv-design-friction-capture-protocol-worker.md` — Friction capture at completion (this design defines the aggregation layer)

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: Where should 8 feedback loop commands be embedded?
- Context: Commands that aren't structurally embedded don't get used; same insight as friction capture protocol.

**2026-03-05:** Exploration complete — 6 forks identified
- Key discovery: 5 of 8 commands already have structural homes (daemon periodic or orient)
- Core gap: daemon results don't flow to orient/debrief via daemon-status.json bridge

**2026-03-05:** Synthesis complete — 4-phase implementation plan
- Recommended: Health Summary pattern in orient, debrief enrichment, 2 new daemon tasks
- No new threshold pressure beyond existing hotspot gates

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: The gap is wiring between existing producers and consumers, not new capabilities. Daemon-status.json is the solved bridge pattern.
