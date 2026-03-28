## Summary (D.E.K.N.)

**Delta:** Wired claim status into `orch orient` — orient now surfaces per-model untested claim counts, recently disconfirmed claims (contradicts evidence in last 7 days), and existing edges in a unified "Knowledge Edges" section.

**Evidence:** All 12 new tests pass. Live output confirmed: 2 models with untested claims surfaced (architectural-enforcement: 3 untested core, harness-engineering: 1 untested core), no recent disconfirmations in current data.

**Knowledge:** The claims.yaml infrastructure already had all the data needed — the gap was aggregation and formatting. CollectClaimStatus, CollectRecentDisconfirmations, and FormatClaimSurface compose cleanly with existing CollectEdges.

**Next:** Pending probes detection (spawned but not complete) deferred — no clean data source exists without beads cross-referencing. Follow-up issue created.

**Authority:** implementation - Extends existing orient data collection within single scope, no architectural changes.

---

# Investigation: Wire Claim Status Into Orient

**Question:** How to surface untested claims at session start via orch orient, extending the Knowledge Edges section?

**Started:** 2026-03-28
**Updated:** 2026-03-28
**Owner:** research agent (orch-go-og67s)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-28-inv-design-research-cycle-autoresearch-style.md` | implements step 4 | yes | None — design specified ~50 lines for orient integration |

---

## Findings

### Finding 1: Claims package already had all building blocks

**Evidence:** `pkg/claims/claims.go` already had `ScanAll`, `CollectEdges`, `FormatEdges`, `Claim.IsStale()`, `Evidence.Verdict` field. The gap was two aggregation functions: one to summarize per-model claim status, one to find recent disconfirmations.

**Source:** `pkg/claims/claims.go:136-158` (ScanAll), `pkg/claims/claims.go:239-296` (CollectEdges)

**Significance:** No new subsystems needed. Three functions added (~120 lines) to compose existing data into the orient output.

---

### Finding 2: Live output shows 2 models with untested claims

**Evidence:** Running `orch orient` against the actual .kb/models/ data shows:
- architectural-enforcement: 6/9 confirmed, 3 untested core
- harness-engineering: 13/14 confirmed, 1 untested core

No recent disconfirmations found (no `contradicts` evidence within 7 days in current data).

**Source:** `go run ./cmd/orch orient | grep -A 30 "Knowledge Edges"`

**Significance:** The aggregation is working correctly against real data. The filtering (only show models with untested claims) keeps the output focused.

---

### Finding 3: Pending probes detection deferred

**Evidence:** The task requested "Claims with pending probes (spawned but not complete)". This requires cross-referencing beads issues with claim IDs. No existing data structure links spawned probes to specific claims — the Evidence struct records completed probes, not in-flight ones. Implementing this would require either:
1. Beads query for open issues with "probe" in title + claim ID parsing
2. Events.jsonl scan for research spawns without completion events

Both add significant complexity for a V1.

**Source:** `pkg/claims/claims.go:59-64` (Evidence struct — only records completed evidence)

**Significance:** Deferred to follow-up issue. The orient integration already provides the core value: seeing what's untested and what was recently contradicted.

---

## Structured Uncertainty

**What's tested:**

- ✅ CollectClaimStatus correctly groups claims by confidence and filters to models with untested claims (verified: 6 test cases)
- ✅ CollectRecentDisconfirmations finds contradicts evidence within day window (verified: 4 test cases)
- ✅ FormatClaimSurface combines all three data sources into unified output (verified: 3 test cases)
- ✅ Live orient output renders correctly against real claims.yaml data (verified: manual test)
- ✅ Existing tests still pass — no regressions (verified: `go test ./pkg/claims/ ./pkg/orient/`)

**What's untested:**

- ⚠️ Pending probes detection (deferred — no data source)
- ⚠️ Performance with large numbers of models/claims (current codebase has 9 claims.yaml files, all small)
- ⚠️ Whether orient claim status actually changes research behavior (requires 30-day measurement)

**What would change this:**

- If pending probes become trackable (e.g., via beads labels linking to claim IDs), the third section could be added
- If models grow beyond ~20 with untested claims, the output may need a summary count instead of per-model listing

---

## References

**Files Modified:**
- `pkg/claims/claims.go` — Added ModelClaimStatus, RecentDisconfirmation types; CollectClaimStatus, CollectRecentDisconfirmations, FormatClaimSurface functions
- `pkg/claims/claims_test.go` — Added 9 test functions covering new functionality
- `cmd/orch/orient_cmd.go` — Updated collectClaimEdges to wire new functions

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-28-inv-design-research-cycle-autoresearch-style.md` — Design reference (step 4 of research cycle)

---

## Investigation History

**2026-03-28:** Investigation started
- Initial question: How to surface untested claims at session start
- Context: Step 3 of research cycle design (orch-go-47ppm)

**2026-03-28:** Implementation complete
- Added 3 functions to claims package + 9 tests
- Wired into orient_cmd.go
- Live output verified
- Pending probes detection deferred
