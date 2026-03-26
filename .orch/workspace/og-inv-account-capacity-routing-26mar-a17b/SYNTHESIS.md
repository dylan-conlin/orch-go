# Session Synthesis

**Agent:** og-inv-account-capacity-routing-26mar-a17b
**Issue:** orch-go-jqkvm
**Duration:** 2026-03-26T09:24:05-07:00 -> 2026-03-26T09:55:00-07:00
**Outcome:** success

---

## Plain-Language Summary

I traced how `pkg/spawn/resolve.go` picks a Claude account today and found that it no longer does a simple work-first-then-spillover check. Instead, it scores every account with capacity data by how much usable headroom it has after factoring in both the 5-hour limit, the 7-day limit, and the account tier multiplier, then falls back to default account rules only when capacity data is unavailable. I also updated the knowledge base pages that still described the older threshold-based routing story so future agents read the current behavior instead of stale docs.

## TLDR

The account router now chooses the account with the highest tier-weighted effective headroom, defined as `min(FiveHourRemaining*tier, SevenDayRemaining*tier)`, with weighted 5-hour headroom and then name as tie-breakers. I documented that behavior in the investigation and refreshed the key KB models that still claimed primary/spillover threshold routing.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-inv-account-capacity-routing-26mar-a17b/VERIFICATION_SPEC.yaml` - Verification contract for the investigation.
- `.orch/workspace/og-inv-account-capacity-routing-26mar-a17b/SYNTHESIS.md` - Session synthesis for orchestrator review.
- `.orch/workspace/og-inv-account-capacity-routing-26mar-a17b/BRIEF.md` - Dylan-facing comprehension brief.

### Files Modified
- `.kb/investigations/2026-03-26-inv-account-capacity-routing-work-pkg.md` - Recorded findings, evidence, recommendations, and uncertainty.
- `.kb/models/spawn-architecture/model.md` - Replaced stale primary/spillover wording with effective-headroom routing details.
- `.kb/models/model-access-spawn-paths/model.md` - Updated account-routing summary and cascade to match the current resolver.
- `.kb/models/orchestration-cost-economics/model.md` - Updated economic routing description to reflect tier-weighted scoring.
- `.kb/models/orchestration-cost-economics/probes/2026-02-26-probe-account-distribution-wiring-trace.md` - Added a staleness note pointing readers to the new investigation.

### Commits
- None yet

---

## Evidence (What Was Observed)

- `resolveAccount()` computes `fiveHourAbs`, `weeklyAbs`, and `effectiveHeadroom := min(fiveHourAbs, weeklyAbs)` for each account with capacity data, then sorts by effective headroom, 5-hour headroom, and name (`pkg/spawn/resolve.go:537`, `pkg/spawn/resolve.go:562`, `pkg/spawn/resolve.go:572`).
- Account roles are only used when no heuristic routing is possible; with a `CapacityFetcher`, the loop scores all accounts together and ignores `Role` for ranking (`pkg/spawn/resolve.go:525`, `pkg/spawn/resolve.go:550`).
- The current test suite explicitly covers tier weighting, weekly exhaustion, CLI override, and nil-capacity fail-open behavior (`pkg/spawn/resolve_test.go:1335`, `pkg/spawn/resolve_test.go:1471`, `pkg/spawn/resolve_test.go:1200`, `pkg/spawn/resolve_test.go:1229`).
- Existing KB artifacts still described a `>20%` primary/spillover heuristic before this session (`.kb/models/model-access-spawn-paths/model.md:107`, `.kb/models/orchestration-cost-economics/probes/2026-02-26-probe-account-distribution-wiring-trace.md:54`).

### Tests Run
```bash
# Targeted routing regression coverage
go test ./pkg/spawn -run "AccountHeuristic" -count=1
# PASS: ok github.com/dylan-conlin/orch-go/pkg/spawn 0.429s
```

---

## Verification Contract

- See `.orch/workspace/og-inv-account-capacity-routing-26mar-a17b/VERIFICATION_SPEC.yaml`.
- Key outcome: targeted `AccountHeuristic` tests passed while the updated docs were aligned to the live resolver logic.

---

## Architectural Choices

No architectural choices - task was within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-account-capacity-routing-work-pkg.md` - Current routing explanation with code-backed findings.

### Decisions Made
- Documentation should describe account routing as effective-headroom scoring rather than primary/spillover threshold switching.

### Constraints Discovered
- Historical KB probes can drift behind implementation and need explicit staleness notes when they remain useful as historical context.

### Externalized via `kb quick`
- `kb quick decide "Account routing documentation should describe resolveAccount as effective-headroom scoring, not primary/spillover threshold switching" --reason "pkg/spawn/resolve.go and AccountHeuristic tests show current routing is min(FiveHourRemaining*tier, SevenDayRemaining*tier) with role only used in fallback paths"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-jqkvm`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should dashboard/account telemetry surface effective headroom directly so humans do not have to mentally combine tier and limit percentages?

**Areas worth exploring further:**
- A broader KB sweep for other stale references to the retired threshold heuristic.

**What remains unclear:**
- Exactly when the implementation switched from threshold routing to effective-headroom scoring.

---

## Friction

No friction - smooth session.

---

## Session Metadata

**Skill:** investigation
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-inv-account-capacity-routing-26mar-a17b/`
**Investigation:** `.kb/investigations/2026-03-26-inv-account-capacity-routing-work-pkg.md`
**Beads:** `bd show orch-go-jqkvm`
