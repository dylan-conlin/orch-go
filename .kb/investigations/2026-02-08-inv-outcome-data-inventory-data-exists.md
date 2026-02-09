## Summary (D.E.K.N.)

**Delta:** The system already has enough raw data to compute outcome health (completion, abandonment, timing, and knowledge-pipeline flow), but the data is split across incompatible stores and key joins are heuristic.

**Evidence:** Measured from primary artifacts: `.beads/issues.jsonl` (3088 issues), `.orch/workspace/*/.spawn_time` + `.beads_id` (83 usable spawn-to-close durations), `~/.orch/events.jsonl` (552 `agent.abandoned` events), `~/.orch/stability.jsonl` (176 snapshots, 14 interventions), and live API/CLI outputs (`orch stats --json`, `orch frontier --json`, `orch status --json`).

**Knowledge:** Beads issue status alone under-reports abandonment, timing is measurable but lacks attempt IDs, and investigation promotion can be measured today via citation scans but remains mostly manual and weak for non-archived work.

**Next:** Implement lightweight instrumentation upgrades: explicit attempt IDs across spawn/complete/abandon, explicit abandonment state/reason in beads, and first-class investigation lineage links.

**Authority:** architectural - Changes cross state stores (`beads`, `events`, workspace metadata, and API surface) and require orchestrator-level schema/API choices.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Outcome Data Inventory Data Exists

**Question:** What data already exists to measure system-level success, what can be measured today without new code, and what requires new instrumentation?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** OpenCode worker (orch-go-21508)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation                                                   | Relationship | Verified | Conflicts                                                                                    |
| --------------------------------------------------------------- | ------------ | -------- | -------------------------------------------------------------------------------------------- |
| `.kb/investigations/2026-01-17-inv-spawn-to-value-ratio.md`     | extends      | Yes      | No direct conflict; this inventory validates and extends measurable data sources             |
| `.kb/investigations/2026-01-06-inv-orch-stats-command-if-so.md` | confirms     | Yes      | Confirms metrics source (`~/.orch/events.jsonl`) while exposing additional data quality gaps |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Completion ratio is directly measurable from beads issues, with skill-level breakdown possible via inferred skill mapping.

**Evidence:** Current issue inventory (`.beads/issues.jsonl`) contains 3088 issues with 3046 closed-success, 23 open, and 19 abandoned (abandoned inferred via `close_reason` contains "abandon"). Closed vs (open+abandoned) rate = 98.64%. By inferred skill: `feature-impl` 1939/1966 (98.63%), `systematic-debugging` 411/420 (97.86%), `investigation` 29/33 (87.88%), `architect` 6/7 (85.71%), `kb-reflect` 585/585 (100%).

**Source:** `bd list --json --all --limit 0 --allow-stale`; `.beads/issues.jsonl`; `pkg/daemon/skill_inference.go:26` (issue_type->skill inference).

**Significance:** We can report a system-level completion metric today without schema changes, but abandonment is only weakly encoded in issue close reasons (not first-class status).

---

### Finding 2: Spawn-to-close timing distribution is measurable from workspace metadata, but join quality is imperfect.

**Evidence:** In `.orch/workspace/`, 103 workspace directories exist; 101 include `.spawn_time`; 101 include `.beads_id`; 93 IDs resolve to issues; 83 have both spawn timestamp and issue `closed_at` for duration calculation. Distribution (minutes): p50=5.63, p90=34.26, p95=84.35, max=824.07, negatives=3. Data quality gaps: 8 workspaces point to IDs absent from project issues (`orch-go-untracked-*` and cross-project ID `pw-8947`), and long-tail outliers indicate retries/re-spawns sharing one beads ID.

**Source:** `.orch/workspace/*/.spawn_time`; `.orch/workspace/*/.beads_id`; `.beads/issues.jsonl`; duration scripts executed during this investigation.

**Significance:** Timing telemetry exists now and supports baseline distributions, but attempt-level accuracy is limited by missing explicit attempt identifiers.

---

### Finding 3: Investigation-to-model/decision/guide pipeline is measurable by citation scan, and currently weak for active (non-archived) investigations.

**Evidence:** Across `.kb/investigations/` there are 816 markdown investigations. References by downstream artifacts: models=73, decisions=94, guides=74, any downstream=189, unreferenced=627. For non-archived investigations only: total=75, referenced by models=0, any downstream=7, unreferenced=68.

**Source:** `.kb/investigations/**/*.md`; `.kb/models/**/*.md`; `.kb/decisions/**/*.md`; `.kb/guides/**/*.md`; citation scan script (filename-match based).

**Significance:** The intended investigation->model/decision/guide pipeline is measurable today, and current numbers show low conversion for recent work.

---

### Finding 4: Abandonment patterns are richly observable in events/stability logs, but weakly represented in beads issue state.

**Evidence:** `~/.orch/events.jsonl` has 552 `agent.abandoned` events (476 for `orch-go-*`), with 252 missing `reason` (224 missing reason in `orch-go` subset). Repeated-abandon pattern is measurable: 226 distinct `orch-go` beads IDs abandoned; 144 IDs abandoned >=2 times; top repeats include `orch-go-21029` (8), `orch-go-21129` (8), `orch-go-21398` (8). In beads data, only 19 issues have abandon markers in `close_reason` and only 1 is explicit `Auto-abandoned`. Comments show 24 `DEAD SESSION` messages and 29 comments containing "abandon".

**Source:** `~/.orch/events.jsonl`; `.beads/issues.jsonl`; SQLite `comments` table in `.beads/beads.db`; `~/.orch/stability.jsonl` (13 `agent_abandoned` interventions).

**Significance:** For abandonment metrics, events are the authoritative source; beads issue state alone substantially undercounts and lacks reason coverage.

---

### Finding 5: Observability surface is broad and already production-usable, but lacks a single standards-based metrics endpoint.

**Evidence:** CLI observability outputs are available and structured: `orch stats --json`, `orch frontier --json`, `orch status --json`, `orch stability --json`. API endpoints for observability respond successfully (`/api/daemon`, `/api/operator-health`, `/api/frontier`, `/api/errors`, `/api/attention`, `/api/kb-health`, `/api/usage`, `/api/usage/cost`), while `/api/metrics` returns 404. `cmd/orch/serve.go` documents numerous JSON endpoints but no Prometheus-style `/metrics` route.

**Source:** `orch` command outputs; HTTPS calls to `https://127.0.0.1:3348/api/...`; `cmd/orch/serve.go:85`.

**Significance:** Most success metrics can be computed from existing JSON feeds now, but external monitoring/alerting integration is harder without a canonical metrics endpoint.

---

## Synthesis

**Key Insights:**

1. **Outcome metrics are possible now** - Completion, abandonment, and timing can be computed immediately from existing artifacts without schema migrations.

2. **Primary-source split creates metric drift risk** - `beads`, workspace metadata, and `events.jsonl` each encode partial truth, so cross-source joins are required and currently heuristic.

3. **Knowledge pipeline throughput is currently low for active work** - Citation scans show most non-archived investigations are not yet referenced by models/decisions/guides.

**Answer to Investigation Question:**

The data needed to define system-level success metrics already exists, but it is fragmented. Today we can measure: issue-level completion ratio by inferred skill (Finding 1), spawn-to-close duration distributions (Finding 2), investigation downstream-reference rates (Finding 3), and abandonment volume/reasons/retries (Finding 4), plus live health via CLI/API observability endpoints (Finding 5). The main limitations are semantic gaps (abandonment not first-class in beads status), join ambiguity (multiple attempts per beads ID), and missing fields in event payloads (many abandonments without reason).

---

## Structured Uncertainty

**What's tested:**

- ✅ Completion and skill outcome rates are measurable from current issue artifacts (verified by parsing `.beads/issues.jsonl` and `bd list --json --allow-stale`).
- ✅ Spawn-to-close timing distribution is measurable from workspace metadata joined to issue `closed_at` timestamps (verified over `.orch/workspace/*`).
- ✅ Abandonment and health observability is live today via CLI/API/logs (verified with `orch stats/frontier/status/stability` and HTTPS `/api/*` calls).

**What's untested:**

- ⚠️ Whether all historical `.orch/workspace-archive` entries are consistent with current `.orch/workspace` timing patterns (not sampled in this run).
- ⚠️ Whether filename-based citation scanning undercounts references that omit explicit investigation filenames.
- ⚠️ Whether `/api/metrics` omission is intentional long-term or temporary implementation gap.

**What would change this:**

- If beads introduces explicit abandonment status/reason fields and backfills history, current abandonment undercount conclusion should be revised.
- If attempt IDs are added and linked at spawn/close time, current timing-noise caveats should materially shrink.
- If downstream artifacts begin storing explicit investigation IDs, citation-based pipeline metrics should be replaced by direct graph queries.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation                                                                            | Authority     | Rationale                                                                |
| ----------------------------------------------------------------------------------------- | ------------- | ------------------------------------------------------------------------ |
| Add first-class attempt and abandonment linkage across issue, event, and workspace layers | architectural | Requires coordinated schema/event/API changes across multiple components |

**Authority Levels:**

- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"

- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Unified Outcome Ledger (minimal fields first)** - Add stable IDs and explicit outcome fields so success metrics can be queried without heuristic joins.

**Why this approach:**

- Removes heuristic joins between workspace metadata, events, and beads state.
- Makes abandonment measurable as first-class state with reason coverage.
- Enables robust timing and retry analytics with attempt-level identity.

**Trade-offs accepted:**

- Defers full historical backfill; start with forward-only instrumentation.
- Adds modest write-path complexity in spawn/complete/abandon flows.

**Implementation sequence:**

1. Add `attempt_id` at spawn and persist to workspace (`.attempt_id`) plus `events.jsonl` spawn/complete/abandon payloads.
2. Add explicit abandonment fields (`outcome=abandoned`, `abandon_reason`) in beads close/reopen/comment workflow.
3. Add one read model/API endpoint that returns normalized outcomes (success/open/abandoned, durations, attempt count) for dashboard and metrics consumers.

### Alternative Approaches Considered

**Option B: Keep current stores, add post-hoc ETL only**

- **Pros:** No command-path schema changes.
- **Cons:** Preserves join ambiguity and missing abandonment semantics; hard to trust for alerts.
- **When to use instead:** Short-term reporting only, no production gating.

**Option C: Add Prometheus endpoint first (`/api/metrics`) without schema cleanup**

- **Pros:** Faster external monitoring integration.
- **Cons:** Exposes inconsistent underlying semantics, so precision issues remain.
- **When to use instead:** Ops priority is external dashboards/alerts over metric fidelity.

**Rationale for recommendation:** Option A addresses root data-model ambiguity (findings 1-4), while B and C improve visibility but keep core outcome semantics fragmented.

---

### Implementation Details

**What to implement first:**

- Add `attempt_id` and enforce presence in `session.spawned`, `agent.completed`, `agent.abandoned` payloads.
- Add abandon reason enforcement/defaulting to reduce missing reason rate.
- Create one normalized outcome query path (CLI/API) consumed by `orch stats`.

**Things to watch out for:**

- ⚠️ Cross-project/untracked IDs (for example `orch-go-untracked-*`, `pw-*`) can break project-local joins.
- ⚠️ Existing historical data has mixed quality; avoid retroactive assumptions during migration.
- ⚠️ `bd list` staleness checks can fail reads unless sync/flags are handled intentionally.

**Areas needing further investigation:**

- Validate whether archived workspace directories materially change duration distributions.
- Evaluate whether a direct citation graph should replace filename-match pipeline scans.
- Define a durable taxonomy for abandonment reasons (stalled, infra, rate-limit, scope mismatch, manual cleanup).

**Success criteria:**

- ✅ A single query returns completion/open/abandoned counts without heuristic text matching.
- ✅ Timing metrics compute from explicit attempt records with <1% unmatched attempts.
- ✅ Missing abandonment reason rate drops from current ~45% to <5%.

---

## References

**Files Examined:**

- `.beads/issues.jsonl` - Primary issue corpus for status/close_reason metrics.
- `.beads/beads.db` - Comments/events schema and abandonment marker checks.
- `.orch/workspace/*/.spawn_time` and `.orch/workspace/*/.beads_id` - Spawn timing and issue linkage.
- `.kb/investigations/**/*.md` - Investigation population for pipeline counts.
- `.kb/models/**/*.md`, `.kb/decisions/**/*.md`, `.kb/guides/**/*.md` - Downstream citation targets.
- `pkg/daemon/skill_inference.go` - Canonical issue_type->skill mapping used for breakdown.
- `cmd/orch/serve.go` - Enumerated API observability surface.
- `~/.orch/events.jsonl` and `~/.orch/stability.jsonl` - Lifecycle and reliability telemetry.

**Commands Run:**

```bash
# Phase + context start
pwd && orch phase orch-go-21508 Planning "Starting outcome data inventory and source validation"

# Primary observability commands
orch stats --json
orch frontier --json
orch status --json
orch stability --json

# API endpoint checks
curl -k "https://127.0.0.1:3348/api/daemon?project=/Users/dylanconlin/Documents/personal/orch-go"
curl -k "https://127.0.0.1:3348/api/operator-health?project=/Users/dylanconlin/Documents/personal/orch-go"
curl -k "https://127.0.0.1:3348/api/frontier?project=/Users/dylanconlin/Documents/personal/orch-go"
curl -k "https://127.0.0.1:3348/api/metrics?project=/Users/dylanconlin/Documents/personal/orch-go"

# Beads + KB scans and custom inventory scripts
bd list --json --all --limit 0 --allow-stale
sqlite3 .beads/beads.db '.tables'
sqlite3 .beads/beads.db "SELECT COUNT(*) FROM comments WHERE LOWER(text) LIKE '%abandon%';"
python3 [ad-hoc scripts run in this session for counting, joins, and distributions]
```

**External Documentation:**

- None.

**Related Artifacts:**

- **Model:** `.kb/models/daemon-autonomous-operation.md` - Provides expected spawn/skill behavior used in skill mapping interpretation.
- **Model:** `.kb/models/dashboard-agent-status.md` - Provides context for completion/abandonment interpretation and metric caveats.
- **Workspace:** `.orch/workspace/og-inv-outcome-data-inventory-08feb-3d36` - Current worker workspace for this inventory.

---

## Investigation History

**[2026-02-08 23:04 PST]:** Investigation started

- Initial question: What outcome data exists today for success metrics design?
- Context: Spawned from `orch-go-21508` as prerequisite inventory for metrics definition.

**[2026-02-08 23:10 PST]:** Core datasets validated

- Confirmed measurable sources across beads, workspace metadata, events, stability, and live API/CLI outputs.

**[2026-02-08 23:18 PST]:** Investigation completed

- Status: Complete
- Key outcome: Existing data supports immediate baseline metrics, with clear instrumentation priorities to remove semantic/join ambiguity.
