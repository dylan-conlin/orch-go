## Summary (D.E.K.N.)

**Delta:** Instrument orch-go to answer: does structural enforcement improve agent quality?

**Evidence:** Measurement gap audit (orch-go-ca0k0) found survivorship bias architecture — 52% of completions lack key fields, gate decisions produce 0 events, advisory findings are print-and-discard.

**Knowledge:** You can't calculate gate accuracy without denominator data. We measure survivors, not decisions.

**Next:** Release Phase 1 issues to daemon (backfill + bypass reason).

---

# Plan: Measurement Instrumentation

**Date:** 2026-03-11
**Status:** Active
**Owner:** Dylan

**Extracted-From:** `.kb/investigations/2026-03-11-inv-investigation-audit-orch-go-ecosystem.md`

---

## Objective

Instrument the orch-go enforcement pipeline so that by Mar 24 (accretion probe checkpoint), we can answer with data: "Do harness gates improve agent quality, or just slow things down?" Success = can query gate accuracy, per-skill quality metrics, and pipeline overhead from events.jsonl.

---

## Substrate Consulted

- **Models:** harness-engineering (hard/soft distinction, gate layering), knowledge-physics (coordination cost observations)
- **Decisions:** Three-layer hotspot enforcement (2026-02-26)
- **Constraints:** events.jsonl format must remain backwards-compatible (additive fields only)
- **Investigation:** orch-go-ca0k0 — 8 findings, 4,411 events analyzed

---

## Phases

### Phase 1: Data Quality (parallelizable)

**Goal:** Make existing events trustworthy
**Issues:**
- `orch-go-nsh33` — Backfill agent.completed fields (52% → 95%+ coverage)
- `orch-go-9o5h2` — Populate hotspot bypass reason field

**Exit criteria:** 95%+ of new completions have skill, outcome, duration fields
**Depends on:** Nothing — start immediately

### Phase 2: Gate Visibility (parallelizable)

**Goal:** Log enforcement decisions, not just outcomes
**Issues:**
- `orch-go-hmfey` — Add spawn.gate_decision event (block/escalate only initially)
- `orch-go-ewmn1` — Add daemon.architect_escalation event

**Exit criteria:** Can query "how many spawns were blocked/escalated this week?"
**Depends on:** Nothing — independent of Phase 1

### Phase 3: Detection Telemetry (parallelizable)

**Goal:** Make advisory findings queryable
**Issues:**
- `orch-go-x71hy` — Add duplication.detected event
- `orch-go-3w5dz` — Diagnose accretion.delta 4.7% coverage

**Exit criteria:** Duplication findings in events.jsonl; accretion coverage >50%
**Depends on:** Nothing — independent of Phase 1-2

### Phase 4: Correlation (blocked on Phase 1-3)

**Goal:** Answer the focus question
**Issues:**
- `orch-go-00r9c` — Gate effectiveness query

**Exit criteria:** Can answer "for agents that hit a gate, was the outcome better or worse?"
**Depends on:** All of Phase 1-3 + 2-4 weeks of data accumulation

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Event schema design | Investigation findings + existing logger.go patterns | Yes |
| Gate volume management | Start blocks-only, expand if needed | Yes |
| Correlation methodology | Needs Phase 1-3 data first | Deferred to Phase 4 |

**Overall readiness:** Ready to execute Phase 1-3 immediately. Phase 4 blocked on data.

---

## Structured Uncertainty

**What's tested:**
- ✅ 52% field coverage gap (parsed 4,411 events)
- ✅ 0 gate decision events (verified in code)
- ✅ Pipeline timing infrastructure works (1 event captured)

**What's untested:**
- ⚠️ Whether 2-4 weeks is enough data for Phase 4 correlation
- ⚠️ Whether gate volume (blocks+escalations only) is manageable
- ⚠️ Root cause of 4.7% accretion.delta coverage

**What would change this plan:**
- If accretion probe (Mar 24) shows clear signal without new instrumentation, Phase 3-4 priority drops
- If orchestrator skill reframe (thread) proceeds first, gate definitions may shift

---

## Success Criteria

- [ ] 95%+ of agent.completed events have skill/outcome/duration
- [ ] Gate decisions (block/escalate) queryable from events.jsonl
- [ ] Can answer: "What % of hotspot-gated spawns were reworked vs non-gated?"
- [ ] Pipeline timing data for 50+ completions
- [ ] Duplication findings queryable (not just printed)
