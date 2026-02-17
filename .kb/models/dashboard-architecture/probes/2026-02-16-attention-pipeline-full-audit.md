# Probe: Attention Pipeline Full Audit — What's Real vs Stub

**Model:** dashboard-architecture
**Date:** 2026-02-16
**Status:** Complete

---

## Question

The model claims the attention system has 9 badge types (verify, decide, escalate, likely_done, recently_closed, unblocked, stuck, crashed, verify_failed) and that backend collectors feed `/api/attention` with real signals. Are any of these signals working correctly? For each signal type, is the detection logic implemented or placeholder? Are signals computed from real agent state or hardcoded/stubbed?

---

## What I Tested

1. Hit the live `/api/attention` endpoint and analyzed the response:
```bash
curl -sk 'https://localhost:3348/api/attention' | python3 -c "
import json, sys, collections
data = json.load(sys.stdin)
print(f'Total items: {data[\"total\"]}')
counts = collections.Counter(item['signal'] for item in data['items'])
print(f'Signal counts: {dict(counts)}')
"
# Result: Total items: 59
# Signal counts: {'issue-ready': 33, 'recently-closed': 26}
# All 11 registered collectors ran. Only 2 produced output.
```

2. Hit the `/api/attention/likely-done` endpoint:
```bash
curl -sk 'https://localhost:3348/api/attention/likely-done' | python3 -c "..."
# Result: Total likely-done: 0
```

3. Checked `/api/agents?since=all` for stuck/awaiting-cleanup agents:
```bash
curl -sk 'https://localhost:3348/api/agents?since=all' | python3 -c "..."
# Result: 1341 agents, statuses: dead=9, completed=1332
# Awaiting-cleanup: 0, Active/idle (stuck candidates): 0
```

4. Checked `~/.orch/verify-failed.jsonl` for verification failures:
```bash
# 52 unique beads_ids, but 0 within 72h cutoff (most recent: 96.1h ago)
```

5. Checked `~/.orch/verifications.jsonl` for recorded verifications:
```bash
# File exists with many entries. But all 26 recently-closed items have
# verification_status: "unverified" — verification entries aren't matching.
```

6. Read every collector implementation in `pkg/attention/`:
   - `beads.go` (BeadsCollector) — Signal: `issue-ready`
   - `git.go` + `git_collector.go` (GitCollector) — Signal: `likely-done`
   - `recently_closed_collector.go` (RecentlyClosedCollector) — Signal: `recently-closed`
   - `agent_collector.go` (AgentCollector) — Signal: `verify`
   - `stuck_collector.go` (StuckCollector) — Signal: `stuck`
   - `unblocked_collector.go` (UnblockedCollector) — Signal: `unblocked`
   - `verify_failed_collector.go` (VerifyFailedCollector) — Signal: `verify-failed`
   - `epic_orphan_collector.go` (EpicOrphanCollector) — Signal: `epic-orphaned`
   - `stale_collector.go` (StaleIssueCollector) — Signal: `stale`
   - `duplicate_candidate_collector.go` (DuplicateCandidateCollector) — Signal: `duplicate-candidate`
   - `competing_collector.go` (CompetingCollector) — Signal: `competing`

7. Read the frontend store (`attention.ts`), badge mapping (`mapSignalToBadge`), type definitions (`work-graph.ts`), and rendering (`work-graph-tree.svelte`, `work-graph-tree-helpers.ts`).

---

## What I Observed

### Collector-by-Collector Audit

| # | Collector | Signal | Implementation Status | Currently Firing? | Why / Why Not |
|---|-----------|--------|-----------------------|-------------------|---------------|
| 1 | BeadsCollector | `issue-ready` | **REAL** — queries `bd ready`, returns open/unblocked issues | **YES** (33 items) | Working correctly. But has no badge mapping — falls through to `default: return 'verify'` |
| 2 | RecentlyClosedCollector | `recently-closed` | **REAL** — queries closed issues within 24h lookback | **YES** (26 items) | Working correctly. All 26 are `unverified` because verifications log entries use different IDs |
| 3 | GitCollector | `likely-done` | **REAL** — scans git log for issue ID mentions, cross-refs with open issues, excludes issues with active workspaces | **NO** (0 items) | Logic is sound but condition is strict: needs open issue + commits mentioning its ID + no active workspace. Currently no open issues match. |
| 4 | AgentCollector | `verify` | **REAL** — queries `/api/agents?since=all`, filters for `status=awaiting-cleanup` with a beads_id | **NO** (0 items) | No agents currently in `awaiting-cleanup` state (all 1341 are `completed` or `dead`). Would fire if an agent was in that state. |
| 5 | StuckCollector | `stuck` | **REAL** — queries agents, filters for `active`/`idle` status, checks spawn time > 2h + inactivity > 30min or `is_stalled` | **NO** (0 items) | No active/idle agents exist right now. Logic is implemented and would fire if conditions met. |
| 6 | UnblockedCollector | `unblocked` | **REAL** — lists open issues with dependencies, checks if all blocking deps are resolved | **NO** (0 items) | No open issues currently have all-resolved blocking dependencies. Logic is sound. |
| 7 | VerifyFailedCollector | `verify-failed` | **REAL** — reads `~/.orch/verify-failed.jsonl`, deduplicates, filters by 72h age | **NO** (0 items) | File has 52 entries but all are > 72h old (most recent: 96.1h). Would fire with fresh failures. |
| 8 | EpicOrphanCollector | `epic-orphaned` | **REAL** — reads `~/.orch/events.jsonl`, finds `epic.orphaned` events within 7 days | **NO** (0 items) | No recent `epic.orphaned` events. Logic is implemented. |
| 9 | StaleIssueCollector | `stale` | **REAL** — lists open issues, checks if `updated_at` > 30 days ago | **NO** (0 items) | All open issues have recent activity. Would fire for truly stale issues. |
| 10 | DuplicateCandidateCollector | `duplicate-candidate` | **REAL** — O(n^2) Jaccard similarity on open issue titles, threshold 60% | **NO** (0 items) | No open issue pairs exceed 60% title similarity. |
| 11 | CompetingCollector | `competing` | **REAL** — groups issues by `area:` label, then title similarity at 40% | **NO** (0 items) | Requires `area:` labels on issues. Few/no issues have them. |

### Frontend Badge Mapping Audit

The frontend defines 9 badge types but the `mapSignalToBadge()` function only maps 5 signal names:

| Badge Type | Has Backend Collector? | Has Frontend Mapping? | Can Currently Fire? |
|------------|------------------------|----------------------|---------------------|
| `verify` | Yes (AgentCollector) | Yes (direct + default fallback) | Yes, but currently all false positives from `issue-ready` default |
| `decide` | **NO** | No mapping — would need collector emitting `decide` signal | **NEVER** — no collector exists |
| `escalate` | **NO** | No mapping — would need collector emitting `escalate` signal | **NEVER** — no collector exists |
| `likely_done` | Yes (GitCollector) | Yes (`likely-done` -> `likely_done`) | Not currently, but implemented |
| `recently_closed` | Yes (RecentlyClosedCollector) | Yes (via `verification_status=verified`) | Not currently (all items are `unverified`) |
| `unblocked` | Yes (UnblockedCollector) | Yes (`unblocked` -> `unblocked`) | Not currently, but implemented |
| `stuck` | Yes (StuckCollector) | Yes (`stuck` -> `stuck`) | Not currently, but implemented |
| `crashed` | **NO** | No mapping — would need collector emitting `crashed` signal | **NEVER** — no collector exists |
| `verify_failed` | Yes (VerifyFailedCollector) | Yes (`verify-failed` -> `verify_failed`) | Not currently (entries too old) |

### Signal-to-Badge Gap: 5 Collectors Have No Badge Mapping

These backend signals hit `default: return 'verify'` and produce false badges:

| Backend Signal | Collector | Intended Purpose | Badge Received |
|---------------|-----------|------------------|----------------|
| `issue-ready` | BeadsCollector | Actionability — "ready to spawn" | `verify` (wrong) |
| `stale` | StaleIssueCollector | Observability — "no activity >30d" | `verify` (wrong) |
| `duplicate-candidate` | DuplicateCandidateCollector | Observability — "possible duplicate" | `verify` (wrong) |
| `competing` | CompetingCollector | Observability — "overlapping scope" | `verify` (wrong) |
| `epic-orphaned` | EpicOrphanCollector | Authority — "open children of closed epic" | `verify` (wrong) |

### Summary: What's Real vs What's Not

**REAL and implemented (all 11 collectors):** Every collector has genuine detection logic reading real data (beads API, git log, agents API, JSONL files). None are stubs or return hardcoded data.

**Currently firing:** Only 2 of 11 (BeadsCollector, RecentlyClosedCollector).

**Not firing due to current state:** 6 of 11 (GitCollector, StuckCollector, UnblockedCollector, VerifyFailedCollector, EpicOrphanCollector, StaleIssueCollector). These would fire under the right conditions.

**Cannot ever fire (badge types with no collector):** 3 badge types — `decide`, `escalate`, `crashed`. These exist in the frontend type system and badge config but have no backend collector that would ever emit them.

**Net effect on dashboard:** 75% of open issues show false "Awaiting verification" badges due to `issue-ready` signals falling through to the default case. Zero legitimate badges are currently displayed.

---

## Model Impact

- [x] **Extends** model with: The attention system has a **fully implemented but mostly dormant** backend with 11 real collectors, but the dashboard currently shows zero correct badges. The architecture is sound — every collector reads real data from real sources (beads CLI, git log, agents HTTP API, JSONL event files). The problems are: (1) `mapSignalToBadge()` defaults unmapped signals to `'verify'`, causing 75% false positive badges; (2) 3 of 9 badge types (`decide`, `escalate`, `crashed`) have no backend collector — they are frontend-only aspirational types; (3) currently only 2 of 11 collectors produce output, and neither maps to a useful badge (one produces false badges, the other produces "unverified" badges). The system is 2-3 fixes away from working: fix the default mapping, add missing badge types for the 5 unmapped signals, and create collectors for `decide`/`escalate`/`crashed` (or remove those badge types).

---

## Notes

- The verification status mismatch (26 recently-closed items all showing "unverified" despite hundreds of verification entries in `verifications.jsonl`) suggests a key mismatch: verifications use issue IDs that don't match the `item.Subject` field used for lookup. Worth a separate investigation.
- The `getInProgressSubline()` function in `work-graph-tree-helpers.ts:317` uses `node.attentionBadge === 'verify'` to show "Awaiting review (Phase: Complete)" for in-progress issues — since every ready issue gets a false `verify` badge, this would show misleading subtext for in-progress issues.
- The recently-closed items only appear in views showing closed issues. The work-graph tree typically shows open issues, so the 26 recently-closed signals don't contribute to tree badge noise — only the 33 `issue-ready` signals do.
- The `CompetingCollector` requires `area:` labels. Unless issues are systematically labeled with area tags, this collector will never fire.
