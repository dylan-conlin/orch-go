<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** No unified `orch stats` command exists, but substantial data already being collected via events.jsonl (4145+ events across 27 event types) and scattered analytical commands (history, hotspot, retries, reconcile).

**Evidence:** events.jsonl tracks spawns (1522), completions (1452), abandonments (77), daemon activity (420), session metrics. Existing commands: `orch history` (skill usage), `orch hotspot` (fix density), `orch retries` (failure patterns), `orch reconcile` (zombies).

**Knowledge:** The system is data-rich but insight-poor - events are captured but not aggregated for decision support. Orchestrator needs historical patterns, not just current state.

**Next:** Recommend creating `orch stats` command that aggregates events.jsonl to surface: completion rates, session durations, skill effectiveness, friction indicators.

---

# Investigation: Orch Stats Command Exploration

**Question:** Do we have an orch stats command? Should we? If so, what should it include?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** design-session spawn
**Phase:** Complete
**Next Step:** Create implementation issue for orch stats command
**Status:** Complete

---

## Findings

### Finding 1: Rich Event Data Already Being Captured

**Evidence:** events.jsonl contains 4145+ events across 27 distinct event types:
- `session.spawned`: 1522 events (full spawn history)
- `agent.completed`: 1452 events (completion tracking)
- `agent.abandoned`: 77 events (failure tracking)
- `daemon.spawn`: 420 events (autonomous spawning)
- `agent.wait.complete`/`agent.wait.timeout`: 42/11 events (wait patterns)
- `session.orchestrator.started`/`.ended`: 10/9 events (orchestrator sessions)
- `spawn.telemetry`: 317 events (model, skill, gap analysis)

**Source:** `~/.orch/events.jsonl`, event types defined in `pkg/events/logger.go:13-24`

**Significance:** The raw data for comprehensive stats already exists. The gap is aggregation and surfacing.

---

### Finding 2: Analytical Commands Are Fragmented

**Evidence:** Existing commands provide partial views:
- `orch history` - Skill usage from workspace markers (not events)
- `orch hotspot` - Fix commit density and investigation clusters
- `orch retries` - Failed attempt patterns per issue
- `orch reconcile` - Zombie in_progress issues
- `orch status` - Current swarm state only (no historical)

**Source:** 
- `cmd/orch/history.go` - Workspace-based skill analysis
- `cmd/orch/hotspot.go` - Git history + kb reflect analysis
- `cmd/orch/retries_cmd.go` - Beads comment pattern matching
- `cmd/orch/reconcile.go` - Cross-references beads vs sessions
- `cmd/orch/status_cmd.go` - Real-time agent state

**Significance:** Each command serves a purpose but there's no unified view. An orchestrator must run multiple commands to understand system health.

---

### Finding 3: Critical Metrics Not Currently Surfaced

**Evidence:** Today's orchestrator reflection revealed unmet observability needs:
1. **Completion rate** - What % of spawns complete successfully?
2. **Session duration** - How long do agents typically run?
3. **Frame collapse frequency** - How often do agents run out of context?
4. **Skill effectiveness** - Which skills have best completion rates?
5. **Daemon vs manual** - Are autonomous spawns more effective?

From events.jsonl: completion rate = 1452/1522 = 95.4%, but this isn't surfaced anywhere.
Abandonment rate = 77/1522 = 5.1% - also not visible.

**Source:** User request in SPAWN_CONTEXT.md, event analysis

**Significance:** These metrics would inform orchestrator decisions: which skills need improvement, when to intervene, system health trends.

---

### Finding 4: Event Data Structure Supports Rich Analysis

**Evidence:** spawn.telemetry events capture:
- `skill`, `model`, `spawn_mode` (headless/tmux/inline)
- `gap_context_quality`, `gap_has_gaps` (context analysis)
- `beads_id`, `workspace` (tracking)
- `no_track`, `skip_artifact_check` (spawn options)

agent.completed events include:
- `beads_id`, `reason`, `forced` flag
- Duration can be computed from spawn→complete timestamps

**Source:** Recent events.jsonl entries, spawn_cmd.go event logging

**Significance:** The event structure already supports the analysis we need. Implementation is mainly aggregation logic.

---

## Synthesis

**Key Insights:**

1. **Data-Rich, Insight-Poor** - The orchestration system captures extensive telemetry but doesn't aggregate it for decision support. Events.jsonl is a goldmine that nobody reads.

2. **Fragmentation Tax** - Orchestrators must run 4+ commands to understand system health. This creates friction and leads to overlooked patterns.

3. **Missing Temporal View** - Current tooling shows snapshots (status) or narrow slices (retries, zombies). No command shows trends over time.

**Answer to Investigation Question:**

**Do we have an orch stats command?** No. No command aggregates events.jsonl data.

**Should we?** Yes. The data exists, the need is real (evidenced by today's orchestrator reflection revealing gaps), and the implementation is straightforward (parse JSONL, aggregate, display).

**What should it include?**

**Core metrics (from events.jsonl):**
1. **Completion Rate** - spawned → completed / spawned (total and by skill)
2. **Abandonment Rate** - abandoned / spawned (with trend)
3. **Session Duration** - avg, p50, p90 from spawn→complete timestamps
4. **Skill Breakdown** - spawns and completion rates per skill
5. **Spawn Mode Analysis** - headless vs tmux effectiveness
6. **Daemon Health** - daemon.spawn count, auto-completion rate

**Friction indicators:**
- Frame collapse frequency (context exhaustion abandonments)
- Wait timeout ratio (agent.wait.timeout / agent.wait.complete)
- Zombie rate (reconcile needed frequency)
- Retry pattern density (from existing retries command)

**Time windows:**
- Last 24 hours (recent health)
- Last 7 days (week health)
- Last 30 days (trend analysis)

---

## Structured Uncertainty

**What's tested:**

- ✅ events.jsonl exists with 4145+ events (verified: counted events)
- ✅ Event types include completion, abandonment, spawn telemetry (verified: jq group_by)
- ✅ Existing analytical commands don't provide unified stats (verified: reviewed each command)

**What's untested:**

- ⚠️ Session duration calculation accuracy (spawn→complete timestamps may not perfectly align)
- ⚠️ Performance at scale (4k events is fine, 100k+ may need optimization)
- ⚠️ User adoption - will orchestrators actually use stats command?

**What would change this:**

- If events.jsonl is being rotated/pruned, historical analysis would be limited
- If spawn timestamps don't correlate with session starts, duration metrics would be wrong
- If orchestrators prefer dashboard over CLI, stats command should surface to API instead

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Create `orch stats` command that aggregates events.jsonl**

**Why this approach:**
- Builds on existing event infrastructure (no new data collection needed)
- CLI-first fits orchestrator workflow (can be piped, scripted)
- Can later expose via `/api/stats` for dashboard

**Trade-offs accepted:**
- CLI-only initially (dashboard can come later)
- Historical only (not real-time; use `orch status` for that)

**Implementation sequence:**
1. Create `stats_cmd.go` with basic event parsing
2. Add core metrics (completion rate, abandonment rate, spawn count)
3. Add time windowing (--days flag like history/hotspot)
4. Add skill breakdown
5. Add --json output for scripting

### Alternative Approaches Considered

**Option B: Extend orch history**
- **Pros:** Familiar command, some stats exist
- **Cons:** history is workspace-based, not event-based; different data source
- **When to use instead:** If events.jsonl data is unreliable

**Option C: Dashboard-only stats**
- **Pros:** Richer visualization
- **Cons:** Requires browser, not CLI-friendly for orchestrators
- **When to use instead:** For presentation/monitoring scenarios

**Rationale for recommendation:** CLI command matches orchestrator workflow (quick checks while coordinating), events.jsonl provides complete data, and existing commands (history, hotspot) set the pattern.

---

### Implementation Details

**What to implement first:**
- Parse events.jsonl efficiently (stream, don't load all)
- Core metrics: spawn count, completion rate, abandonment rate
- Time window filtering (default: 7 days)

**Things to watch out for:**
- ⚠️ events.jsonl may have schema evolution - handle missing fields gracefully
- ⚠️ Timestamps are Unix seconds, not milliseconds (unlike OpenCode sessions)
- ⚠️ Some events lack session_id (early events, some daemon events)

**Areas needing further investigation:**
- How large does events.jsonl get? May need rotation strategy
- Should stats persist aggregations for performance?

**Success criteria:**
- ✅ `orch stats` shows completion rate, abandonment rate, spawn count
- ✅ `orch stats --days 1` for recent, `--days 30` for trends
- ✅ `orch stats --json` for scripting
- ✅ Skill breakdown shows effectiveness by skill type

---

## References

**Files Examined:**
- `cmd/orch/status_cmd.go` - Current status implementation
- `cmd/orch/history.go` - Workspace-based skill analysis
- `cmd/orch/hotspot.go` - Git analysis pattern
- `cmd/orch/retries_cmd.go` - Beads pattern analysis
- `cmd/orch/reconcile.go` - Zombie detection
- `pkg/events/logger.go` - Event type definitions

**Commands Run:**
```bash
# Count events by type
cat ~/.orch/events.jsonl | jq -s 'group_by(.type) | map({type: .[0].type, count: length})'

# Count total events
wc -l ~/.orch/events.jsonl

# Check recent events
tail -5 ~/.orch/events.jsonl
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md` - Prior decision on visibility needs

---

## Investigation History

**2026-01-06 15:40:** Investigation started
- Initial question: Do we have orch stats? Should we?
- Context: Orchestrator reflection revealed observability gaps

**2026-01-06 15:55:** Context gathering complete
- Found 4145+ events in events.jsonl across 27 types
- Reviewed existing analytical commands (history, hotspot, retries, reconcile)

**2026-01-06 16:10:** Investigation completed
- Status: Complete
- Key outcome: Recommend creating `orch stats` command to aggregate events.jsonl data
