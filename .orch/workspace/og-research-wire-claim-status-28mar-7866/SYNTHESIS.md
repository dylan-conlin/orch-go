# Session Synthesis

**Agent:** og-research-wire-claim-status-28mar-7866
**Issue:** orch-go-og67s
**Duration:** 2026-03-28
**Outcome:** success

---

## Plain-Language Summary

Wired claim status into `orch orient` so that every session start now shows which models have untested claims and whether any claims were recently contradicted. Before this change, orient showed individual "Knowledge Edges" (tensions, stale claims) but had no aggregate view of claim coverage across models. Now it shows a per-model summary like "architectural-enforcement: 6/9 confirmed, 3 untested core" alongside existing edge types. This closes the visibility loop in the research cycle: orient shows gaps, the orchestrator can trigger research, and the next orient shows updated status.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `pkg/claims/claims.go` — Added `ModelClaimStatus` and `RecentDisconfirmation` types; `CollectClaimStatus()`, `CollectRecentDisconfirmations()`, and `FormatClaimSurface()` functions (~120 lines)
- `pkg/claims/claims_test.go` — Added 9 test functions covering new functionality
- `cmd/orch/orient_cmd.go` — Updated `collectClaimEdges` to wire claim status summaries and disconfirmations into orient output

---

## Evidence (What Was Observed)

- The claims.yaml infrastructure already tracked all needed data (confidence, evidence with verdicts and dates). The gap was purely aggregation and formatting.
- 9 models have claims.yaml files; 2 currently have untested claims (architectural-enforcement: 3 core, harness-engineering: 1 core)
- No recent disconfirmations exist in current data (no `contradicts` evidence within 7 days)
- Pre-existing test failures in daemon periodic tests are unrelated (beads/briefs directory not found in test environment)

### Tests Run
```bash
go test ./pkg/claims/ -v -run "TestCollect|TestFormat"
# PASS: 12 tests (including 9 new)

go test ./pkg/claims/ ./pkg/orient/ -count=1
# ok pkg/claims  0.186s
# ok pkg/orient  0.345s

go build ./cmd/orch/
# Build successful, no errors
```

---

## Architectural Choices

### FormatClaimSurface as unified formatter vs extending FormatEdges
- **What I chose:** New `FormatClaimSurface` function that combines all three data sources (statuses, disconfirmations, edges) into one "Knowledge Edges" section
- **What I rejected:** Modifying `FormatEdges` to accept additional parameters, or having orient_cmd.go concatenate multiple formatted strings
- **Why:** A single formatter produces a coherent section with clear sub-headers. FormatEdges remains available for backward compatibility.
- **Risk accepted:** FormatEdges is now unused by orient (only used by any external callers). If no external callers exist, it could be removed.

### Deferred pending probes detection
- **What I chose:** Omit "claims with pending probes" from V1
- **What I rejected:** Cross-referencing beads issues or events.jsonl for in-flight probes
- **Why:** No clean data source links spawned probes to specific claim IDs. Adding this would require either beads queries (shell-out) or event parsing with uncertain matching. The two implemented features (untested summary + disconfirmations) provide the core value.
- **Risk accepted:** Users don't see which claims already have research in progress, potentially causing duplicate spawns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-28-inv-wire-claim-status-into-orient.md` — Implementation investigation

### Decisions Made
- Only show models with untested claims (not all models) to keep output focused
- Sort models by core untested count descending for priority surfacing
- Use 7-day window for "recently disconfirmed" (matches existing spawn keyword window)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (12 tests, 0 failures)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-og67s`

---

## Unexplored Questions

- **Pending probes detection**: How to link spawned probe agents to specific claim IDs? Could use beads labels like `claim:HE-08` on probe issues, but this requires convention adoption.
- **Stale confirmed claims with recent probes**: The staleness threshold (30 days) may cause confirmed claims to cycle between "confirmed" and "stale" — should orient suppress recently-probed stale claims?

---

## Friction

No friction — smooth session. Claims infrastructure was well-designed for extension.

---

## Session Metadata

**Skill:** research
**Workspace:** `.orch/workspace/og-research-wire-claim-status-28mar-7866/`
**Investigation:** `.kb/investigations/2026-03-28-inv-wire-claim-status-into-orient.md`
**Beads:** `bd show orch-go-og67s`
