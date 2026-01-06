# Session Synthesis

**Agent:** og-work-orch-stats-command-06jan-96d1
**Issue:** orch-go-5a8qs
**Duration:** 2026-01-06 15:40 → 2026-01-06 16:15
**Outcome:** success

---

## TLDR

Investigated whether orch stats command exists and what it should include. No unified stats command exists, but events.jsonl contains 4145+ events across 27 types - rich data that isn't being surfaced. Recommend creating `orch stats` to aggregate completion rates, session durations, skill effectiveness, and friction indicators.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-orch-stats-command-if-so.md` - Full investigation with findings and recommendations

### Files Modified
- None (investigation only)

### Commits
- Investigation file to be committed after synthesis

---

## Evidence (What Was Observed)

- events.jsonl contains 4145 events (verified via `wc -l`)
- 27 distinct event types (verified via `jq group_by(.type)`)
- Key event counts:
  - `session.spawned`: 1522
  - `agent.completed`: 1452 (95.4% completion rate)
  - `agent.abandoned`: 77 (5.1% abandonment rate)
  - `daemon.spawn`: 420 (27.6% daemon-driven)
- Existing analytical commands (history, hotspot, retries, reconcile) don't aggregate this data
- `orch status` shows only current state, no historical patterns

### Commands Run
```bash
# Count events by type
cat ~/.orch/events.jsonl | jq -s 'group_by(.type) | map({type: .[0].type, count: length})'

# Count total events
wc -l ~/.orch/events.jsonl
# Output: 4145

# Check existing commands
grep -E "Use:|Short:" cmd/orch/*.go
```

---

## Knowledge (What Was Learned)

### Key Insight: Data-Rich, Insight-Poor
The orchestration system captures extensive telemetry (events.jsonl) but doesn't aggregate it for decision support. Orchestrators must run multiple commands or analyze raw JSONL to understand system health.

### Constraints Discovered
- events.jsonl timestamps are Unix seconds (not milliseconds like OpenCode sessions)
- Some early events lack session_id field
- events.jsonl may have schema evolution - need graceful handling of missing fields

### What Should `orch stats` Include

**Core metrics:**
1. Completion rate (overall and by skill)
2. Abandonment rate (with trend)
3. Session duration (avg, p50, p90)
4. Skill breakdown (spawns and completion rates per skill)
5. Spawn mode analysis (headless vs tmux effectiveness)
6. Daemon health (spawn count, auto-completion rate)

**Friction indicators:**
- Frame collapse frequency
- Wait timeout ratio
- Zombie rate
- Retry pattern density

**Time windows:**
- `--days 1` - Last 24 hours (recent health)
- `--days 7` - Last week (default)
- `--days 30` - Last 30 days (trends)

---

## Next (What Should Happen)

**Recommendation:** close and create implementation issue

### Summary of Deliverable

This design-session produced:
1. Investigation confirming no stats command exists
2. Analysis of events.jsonl data structure (27 event types, 4145+ events)
3. Specification for what `orch stats` should include
4. Implementation sequence recommendation

### If Implementation Spawned

**Issue:** Create `orch stats` command to aggregate events.jsonl data
**Skill:** feature-impl
**Context:**
```
events.jsonl at ~/.orch/events.jsonl contains 27 event types. 
Parse and aggregate to surface: completion rate, abandonment rate, 
session duration, skill effectiveness. Support --days flag (default 7)
and --json output. See .kb/investigations/2026-01-06-inv-orch-stats-command-if-so.md
for full specification.
```

**Priority:** P2 (helpful but not blocking)

---

## Unexplored Questions

**Questions that emerged during this session:**
- How large does events.jsonl get? May need rotation/pruning strategy
- Should stats be persisted/cached for performance at scale?
- Should stats be exposed via `/api/stats` for dashboard integration?

**Areas worth exploring further:**
- Correlation between skill and session duration (are some skills slower?)
- Predictive signals for frame collapse (can we intervene before context exhaustion?)
- Integration with `orch doctor` for system health checks

---

## Session Metadata

**Skill:** design-session
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-orch-stats-command-06jan-96d1/`
**Investigation:** `.kb/investigations/2026-01-06-inv-orch-stats-command-if-so.md`
**Beads:** `bd show orch-go-5a8qs`
