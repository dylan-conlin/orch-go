# Design: Coaching Metrics Redesign

**Date:** 2026-02-14
**Type:** Architecture Design
**Phase:** Complete
**Status:** Complete
**Issue:** orch-go-0pc
**Scope:** Plugin metrics, orch stats integration, session-start auto-surfacing

---

## Problem Statement

The coaching plugin (1831 lines, `plugins/coaching.ts`) tracks 8 orchestrator metrics. Real-world data (1022 metrics collected) shows noise:
- `action_ratio` (381 events) and `analysis_paralysis` (355 events) dominate but generate false positives (workers legitimately read more than they write, bash sequences are common)
- `compensation_pattern` (79 events) fires frequently on keyword overlap without being actionable
- `context_ratio` (41 events) is a ratio that doesn't drive behavior change
- `frame_collapse` (13 events) and `circular_pattern` (7 events) are high-signal, low-noise
- `behavioral_variation` (145 events) has value but needs tuning
- `context_usage` (1 event) is the only worker metric recorded so far

Additionally, `orch stats` provides operational metrics (spawns, completions, verification) but no behavioral health visibility. And there's no mechanism to auto-surface system health when an orchestrator session starts.

### Success Criteria

1. Orchestrator metrics: 4 high-signal metrics (down from 8)
2. Stats integration: `orch stats` shows coaching summary section
3. Session start: Orchestrator automatically sees last-24h health summary
4. No regressions in worker health tracking

### Constraints

- Plugins cannot see LLM response text (fundamental OpenCode constraint)
- Plugin hooks available: `tool.execute.after`, `event` (for `session.created`), `experimental.chat.messages.transform`, `config`
- Coaching.ts is at 1831 lines (above 1500-line accretion boundary)
- Worker metrics just started working Feb 14 - don't touch them
- Dashboard (`web/src/lib/stores/coaching.ts`) references current metric shapes

---

## Fork 1: How to Detect the 3 New Orchestrator Metrics

### Fork 1a: `spawn_without_context` — KILLED (Post-Synthesis)

**What it would detect:** Orchestrator running `orch spawn` without first running `kb context`.

**Why killed:** `orch spawn` already runs `kb context` automatically via `pkg/spawn/kbcontext.go`. The infrastructure extracts keywords from the task description, runs tiered search (local → global), and injects results into SPAWN_CONTEXT.md. The orchestrator skill's instruction to manually run `kb context` before every spawn is a holdover from before this automation existed.

**Action:** Remove manual `kb context` instruction from orchestrator skill. The real quality gate is the automated context score shown in spawn output (e.g., "Context: ✓ 90/100 (excellent)"). If context is low, the orchestrator should improve the task description, not manually run `kb context`.

**Result:** 4 orchestrator metrics (not 5): frame_collapse, completion_backlog, behavioral_variation, circular_pattern.

---

### Fork 1b: `completion_backlog`

**What it detects:** Agents at Phase:Complete for >10 minutes without `orch complete` being run.

**Options:**

- **A: Plugin-side detection via tool calls.** Plugin monitors for bash commands containing `bd show` or `orch status` and parses output for Phase:Complete agents. Fragile — depends on orchestrator running specific commands.
- **B: Plugin-side detection via periodic polling.** Plugin periodically shells out to check agent status. Violates the "plugins observe, don't act" pattern and adds latency.
- **C: Go-side detection in `orch stats` or `orch serve`.** The Go backend already has agent status logic in `serve_agents.go`. Add a check there and expose via API. Plugin reads from API when checking behavioral health.
- **D: Plugin `event` hook + session metadata.** Listen to `session.status` events for idle sessions, cross-reference with workspace Phase files.

**Substrate says:**
- Model: "Plugins cannot see LLM response text" — but they CAN see tool call outputs. However, plugins shouldn't be parsing `bd show` output.
- Model: "Observation coupled to intervention" is already a known anti-pattern.
- Principle: Compose over monolith — the Go backend already knows agent status.

**RECOMMENDATION:** Option C — Go-side detection. Add `completion_backlog` detection to `orch serve` (specifically the `/api/agents` handler or a new `/api/coaching/health` endpoint). The Go backend already reads workspace Phase comments and knows agent timestamps. Write a metric to `coaching-metrics.jsonl` from the Go side, making it available to both `orch stats` and the dashboard.

This is architecturally cleaner: the Go backend has the data, the plugin has the detection pattern. Don't force the plugin to discover what the Go backend already knows.

**Implementation:**
```go
// In serve_agents.go or a new coaching health check:
// When serving /api/agents, check for agents where:
// 1. Phase == "Complete" (from workspace Phase comment)
// 2. completedAt + 10min < now
// Write metric to coaching-metrics.jsonl
```

**Trade-off:** This breaks the "plugin is the only metric writer" pattern. But the alternative (plugin polling agent status) is worse — it creates a circular dependency where the plugin calls the API that calls the plugin.

**Open question for orchestrator:** Is the Go backend writing to coaching-metrics.jsonl acceptable, or should completion_backlog live in a separate metrics file? Recommend same file for unified aggregation.

---

### Fork 1c: `direct_implementation`

**What it detects:** Orchestrator editing code files (Edit/Write) outside the allowed orchestration paths (.kb/, .orch/, CLAUDE.md, AGENTS.md).

**Substrate says:**
- Model: This is EXACTLY what `frame_collapse` already detects. The `isCodeFile()` function (line 476) does this check.
- Current `frame_collapse` fires on edit/write to any code file extension (.go, .ts, .svelte, etc.) outside orchestration paths.

**RECOMMENDATION:** `direct_implementation` IS `frame_collapse` with a renamed metric and expanded allowlist. The existing `isCodeFile()` function already excludes `.orch/`, `.kb/`, `.beads/`, `skills/`, `plugins/`, `claude.md`, `skill.md`, `readme.md`, `spawn_context.md`, `synthesis.md`.

**Changes needed:**
1. Rename `frame_collapse` metric_type to `direct_implementation` (or keep both for backwards compat)
2. Add `agents.md` to the orchestration paths allowlist in `isCodeFile()`
3. Keep the tiered injection (1st warning, 3+ strong warning)

**Trade-off:** Renaming breaks dashboard references and historical metric querying. Recommend: keep emitting as `frame_collapse` but ALSO emit as `direct_implementation`, then deprecate `frame_collapse` in next cycle.

Actually, simpler: The issue says KEEP `frame_collapse`. Re-reading the requirements: `direct_implementation` is Edit/Write outside .kb/.orch/CLAUDE.md/AGENTS.md. `frame_collapse` is orchestrator editing CODE files. These are the same thing. **Recommend treating `direct_implementation` as a refinement of `frame_collapse` rather than a separate metric.** Update the allowlist on `frame_collapse` and keep the name.

---

## Fork 2: Removal Strategy for Killed Metrics

**Decision:** Should killed metrics be fully removed from coaching.ts or just disabled?

**Metrics to kill:** `action_ratio`, `analysis_paralysis`, `compensation_pattern`, `context_ratio`, `dylan_signal_prefix`, `premise_skipping`

**Options:**

- **A: Full removal.** Delete all detection code, injection messages, state tracking, and formatMetricForCoach handling for killed metrics.
- **B: Disable via flag.** Add `DISABLED_METRICS` set, skip detection but keep code.
- **C: Remove detection + injection, keep metric type definitions.** Allows historical data to be read but stops generating new data.

**Substrate says:**
- Principle: "Avoid backwards-compatibility hacks" — if it's unused, delete it.
- Constraint: coaching.ts is at 1831 lines (above 1500-line accretion boundary). Removing 6 metrics should drop ~400-600 lines.
- Dashboard references: `serve_coaching.go` aggregates `action_ratio` and `analysis_paralysis` in `aggregateMetrics()`. Dashboard coaching store doesn't reference specific metric types.

**RECOMMENDATION:** Option A — full removal. Delete all code for the 6 killed metrics. This is the right time to reduce coaching.ts below the accretion boundary. The historical data in `coaching-metrics.jsonl` will still contain old metrics, but `serve_coaching.go` and `orch stats` should simply ignore unknown metric types.

**Impact on dependent code:**
1. `serve_coaching.go:263-281` — Remove `action_ratio` and `analysis_paralysis` threshold checks from `aggregateMetrics()`. Replace with new metric thresholds.
2. `plugins/coaching.ts` — Remove ~600 lines of detection code for 6 metrics.
3. `web/src/lib/stores/coaching.ts` — No changes needed (doesn't reference specific metric types, just `overall_status`).
4. `coaching-metrics.jsonl` — Historical data preserved. New aggregation ignores old types.

**Estimated line reduction:** ~400-600 lines from coaching.ts, bringing it to ~1200-1400 (below accretion boundary).

---

## Fork 3: How Should `orch stats` Read Coaching Metrics?

**Options:**

- **A: New function in stats_cmd.go.** Read coaching-metrics.jsonl directly, aggregate by metric_type for the --days window.
- **B: Shared reader with serve_coaching.go.** Extract `readCoachingMetrics()` to a shared package (e.g., `pkg/coaching/reader.go`).
- **C: Call serve_coaching.go's handler internally.** Stats command calls the same aggregation logic.

**Substrate says:**
- `serve_coaching.go` already has `readCoachingMetrics()` and `aggregateMetrics()`.
- `stats_cmd.go` has its own JSONL reader pattern (`parseEvents()`).
- Both read from `~/.orch/` directory.
- `stats_cmd.go` is 1112 lines (below accretion boundary).

**RECOMMENDATION:** Option B — extract shared reader. Create `pkg/coaching/metrics.go` with:
- `ReadMetrics(path string, limit int) ([]Metric, error)` — shared JSONL reader
- `AggregateByType(metrics []Metric, since time.Time) map[string]MetricSummary` — aggregate by metric_type with counts, latest value, avg value
- `FormatTextSummary(summary map[string]MetricSummary) string` — text output for stats

Both `serve_coaching.go` (for API) and `stats_cmd.go` (for CLI) import the shared package.

**Stats output design:**
```
🧠 BEHAVIORAL HEALTH (coaching metrics)
  Orchestrator:
    frame_collapse:        2 events (last: 2h ago)
    completion_backlog:    1 event  (last: 15m ago) ⚠️
    behavioral_variation:  3 events
    circular_pattern:      0 events ✅
  Workers:
    tool_failure_rate:     3 events across 2 sessions
    context_usage:         1 event
    (4 worker metrics tracked, 2 sessions active)
```

**Trade-off:** Creating a new package adds a file but reduces duplication between stats and serve_coaching. Worth it.

---

## Fork 4: Session-Start Auto-Surfacing Mechanism

**Decision:** How to automatically show health summary when orchestrator session starts.

**Options:**

- **A: Plugin `event` hook — listen for `session.created`.** When a session is created, inject a health summary message via `client.session.prompt({ noReply: true })`.
- **B: Plugin `config` hook.** Inject health summary into `config.instructions` at plugin initialization. Problem: runs once at server start, not per-session.
- **C: First-tool-call detection in `tool.execute.after`.** When a new session's first tool call arrives, inject health summary. Uses existing hook infrastructure.
- **D: OpenCode `experimental.chat.system.transform` hook.** Inject health summary into the system prompt for each session. Problem: fires on every LLM call, not just session start.
- **E: External mechanism — orchestrator skill instruction.** Add "run `orch stats --days 1` at session start" to orchestrator skill. No plugin changes needed.

**Substrate says:**
- Model: "Observation coupled to intervention" is a known anti-pattern. But this is a one-time injection, not continuous.
- The Explore agent confirmed `session.created` event IS available via the `event` hook.
- Plugin already has access to `client.session.prompt()` for injection.
- Constraint: `session.created` event fires for ALL sessions (workers too), not just orchestrator sessions.

**RECOMMENDATION:** Option A — listen for `session.created` via the `event` hook. When fired:

1. Check if session is a worker (via metadata) — if so, skip.
2. Read last 24h of coaching-metrics.jsonl (shared reader from Fork 3's `pkg/coaching/`).
3. Generate a compact health summary.
4. Inject via `client.session.prompt({ sessionID, noReply: true })`.

**Implementation sketch:**
```typescript
event: async ({ event }) => {
  if (event.type === "session.created") {
    const sessionInfo = event.properties?.info;
    const sessionId = sessionInfo?.id;
    if (!sessionId) return;

    // Skip workers (check metadata.role)
    if (sessionInfo?.metadata?.role === 'worker') return;

    // Read recent metrics and generate summary
    const summary = generateHealthSummary();
    if (summary) {
      await client.session.prompt({
        sessionID: sessionId,
        prompt: summary,
        noReply: true,
      });
    }
  }
}
```

**Critical consideration:** The health summary should include:
- Last 24h coaching metric counts by type
- Completion backlog (agents at Phase:Complete)
- Agent spawn/completion rates (from events.jsonl or `orch stats --days 1 --json`)

**Problem:** Plugin reads coaching-metrics.jsonl directly (TypeScript). But `orch stats` reads events.jsonl (Go). For the session-start summary to include operational stats, either:
1. Plugin shells out to `orch stats --days 1 --json` and parses the output
2. Plugin only shows coaching metrics, operational stats come separately
3. Plugin calls the `orch serve` API endpoint for combined data

**Recommend Option 1** — shell out to `orch stats --days 1 --json`. The plugin already reads files; shelling out to orch stats gets all the data in one call. Use child_process.execSync or similar.

**Trade-off:** Shell-out adds ~500ms latency to session start. Acceptable for a one-time injection.

**Fallback (Option E):** If the plugin approach is too complex or fragile, the simplest fallback is adding an instruction to the orchestrator skill: "At session start, run `orch stats --days 1` to review system health." This requires zero code changes but relies on the orchestrator following the instruction.

---

## Fork 5: Dashboard Coaching UI Updates

**Decision:** Should the dashboard coaching UI be updated to match new metrics?

**Substrate says:**
- Dashboard coaching store (`web/src/lib/stores/coaching.ts`) only references `overall_status`, `status_message`, `worker_health`. It does NOT reference specific orchestrator metric types.
- `serve_coaching.go` hardcodes `action_ratio` and `analysis_paralysis` thresholds in `aggregateMetrics()`.

**RECOMMENDATION:** Update `serve_coaching.go` to use new metric types for health calculation. Dashboard Svelte code needs no changes — it already just shows the aggregated status.

**Changes in `serve_coaching.go`:**
```go
// Replace action_ratio/analysis_paralysis threshold checks with:
// frame_collapse: any events = warning
// completion_backlog: any events = warning
// behavioral_variation: 5+ events = warning
// circular_pattern: any events = poor
```

---

## Implementation Plan

### Phase 1: Shared Metrics Reader (Foundation)

**Files:** Create `pkg/coaching/metrics.go`, `pkg/coaching/metrics_test.go`

**What:**
- Extract `readCoachingMetrics()` from `serve_coaching.go` into shared package
- Add `AggregateByType()` function for stats integration
- Add `FormatTextSummary()` for CLI output
- Add `ReadMetricsSince(path string, since time.Time) ([]Metric, error)` with time filtering

**Why first:** Both stats integration (Phase 2) and dashboard update (Phase 4) depend on this.

### Phase 2: Stats Integration

**Files:** Modify `cmd/orch/stats_cmd.go`, update `cmd/orch/serve_coaching.go` to use shared reader

**What:**
- Import `pkg/coaching` in stats_cmd.go
- Add `CoachingStats` field to `StatsReport` struct
- Read coaching-metrics.jsonl in `aggregateStats()`
- Add "BEHAVIORAL HEALTH" section to `outputStatsText()`
- Add JSON coaching field to `outputStatsJSON()`
- Update `serve_coaching.go` to import shared reader instead of inline code

### Phase 3: Plugin Metric Redesign

**Files:** Modify `plugins/coaching.ts` (and deploy to `.opencode/plugin/coaching.ts`)

**What (removals):**
- Remove `action_ratio` detection in `flushMetrics()` (lines 593-625)
- Remove `action_ratio` injection (lines 645-649, 676-683)
- Remove `analysis_paralysis` detection in `flushMetrics()` (lines 627-657)
- Remove `analysis_paralysis` injection (lines 652-657, 684-691)
- Remove `compensation_pattern` detection (lines 1449-1483)
- Remove `context_ratio` metric from `flushMetrics()` (lines 592-605)
- Remove `dylan_signal_prefix` detection (lines 1398-1417)
- Remove `premise_skipping` detection (lines 1485-1539)
- Remove `injectCoachingMessage` cases: `action_ratio`, `analysis_paralysis`, `premise_skipping`, `premise_skipping_strong` (lines 676-745)
- Remove `formatMetricForCoach` cases for removed metrics (lines 855-894)
- Remove `DylanPatternState` interface and all related state tracking
- Remove `detectSignalPrefix()`, `detectPriorityUncertainty()`, `detectPremiseSkipping()`, `extractKeywordsSimple()`, `detectCompensation()` functions
- Clean up `SessionState` interface to remove `dylan` field

**What (additions):**
- Update `frame_collapse` allowlist: add `agents.md` to `isCodeFile()` orchestration paths

**What (tuning):**
- `behavioral_variation` threshold: consider raising from 3 to 5 consecutive variations

**Estimated impact:** Remove ~500 lines, add ~50 lines. Net: coaching.ts drops from ~1831 to ~1380 lines.

### Phase 4: Health Calculation Update

**Files:** Modify `cmd/orch/serve_coaching.go`

**What:**
- Update `aggregateMetrics()` to use new metric types for health status
- Replace `action_ratio`/`analysis_paralysis` checks with `frame_collapse`, `completion_backlog`, `behavioral_variation`, `circular_pattern`

### Phase 5: Session-Start Auto-Surfacing

**Files:** Modify `plugins/coaching.ts`

**What:**
- Add `event` handler for `session.created`
- On non-worker session creation, shell out to `orch stats --days 1 --json`
- Parse JSON output and format compact health summary
- Inject via `client.session.prompt({ noReply: true })`

**Dependencies:** Phase 2 must be complete (orch stats needs coaching section for the summary to be useful)

### Phase 6: Completion Backlog Detection (Go-side)

**Files:** New logic in `cmd/orch/serve_agents.go` or `cmd/orch/serve_coaching.go`

**What:**
- When serving `/api/agents` or on a periodic check, detect agents where Phase == "Complete" and completedAt + 10min < now
- Write `completion_backlog` metric to coaching-metrics.jsonl
- Exposed automatically via existing coaching aggregation

**Dependencies:** Phase 1 (shared writer/reader)

---

## Risks and Open Questions

### Risk 1: coaching.ts Accretion (1831 lines)
After Phase 3 removals (~500 lines removed, ~50 added), the file drops to ~1380 lines. This is safely below the 1500-line boundary. But if Phase 5 (session-start) adds significant code, it could creep back up. **Mitigation:** Consider extracting worker health tracking to a separate module during Phase 3.

### Risk 2: Go-side Metric Writing (Phase 6)
The completion_backlog metric would be the first metric written by the Go backend rather than the TypeScript plugin. This creates a second writer to `coaching-metrics.jsonl`. **Mitigation:** JSONL is append-only and safe for concurrent writers. The shared reader doesn't care who wrote the line.

### Risk 3: Session-Start Shell-Out Latency (Phase 5)
Shelling out to `orch stats --days 1 --json` from the plugin adds latency to session creation. **Mitigation:** Run asynchronously — use `child_process.exec` (async), not `execSync`. The health summary arrives shortly after session start, not blocking it.

### Risk 4: `direct_implementation` vs `frame_collapse` Confusion
The issue lists `direct_implementation` as a new metric, but it's functionally identical to `frame_collapse`. **Recommendation:** Keep `frame_collapse` name, update its allowlist. Document that `direct_implementation` was merged into `frame_collapse` rather than creating a duplicate.

### Open Question 1: Behavioral Variation Threshold
Current threshold is 3 consecutive variations in the same semantic group. With 145 events in the data, this may be too sensitive. **Recommend:** Raise to 5 as part of Phase 3 tuning, then observe.

### Open Question 2: Completion Backlog Polling Frequency
How often should the Go backend check for completion backlog? Options: on every `/api/agents` request (piggyback), on a timer (every 60s), or only when `orch stats` is run. **Recommend:** Piggyback on `/api/agents` which is already polled by the dashboard every 30s.

---

## Recommended Phasing Summary

| Phase | Effort | Dependencies | Priority |
|-------|--------|-------------|----------|
| 1. Shared Metrics Reader | Small (1h) | None | P0 — foundation |
| 2. Stats Integration | Medium (2h) | Phase 1 | P0 — core deliverable |
| 3. Plugin Metric Redesign | Medium (2-3h) | None | P0 — core deliverable |
| 4. Health Calculation Update | Small (30m) | Phase 3 | P1 — follow Phase 3 |
| 5. Session-Start Surfacing | Medium (1-2h) | Phase 2 | P1 — can ship separately |
| 6. Completion Backlog | Small (1h) | Phase 1 | P2 — separate from main redesign |

**Phases 1-4 are the core work** (~5-6h total, 2-3 agents).
**Phase 5 and 6 can ship independently** as follow-up.

**Recommended spawn strategy:**
- Agent 1: Phase 1 + Phase 2 (Go backend: shared reader + stats integration)
- Agent 2: Phase 3 + Phase 4 (Plugin redesign + serve_coaching update)
- Agent 3 (follow-up): Phase 5 (session-start surfacing)
- Agent 4 (follow-up): Phase 6 (completion backlog detection)

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves recurring coaching noise issues
- Future agents modifying coaching metrics should see this

**Suggested blocks keywords:**
- coaching metrics
- coaching plugin
- behavioral health
- orch stats
