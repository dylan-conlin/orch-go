## Summary (D.E.K.N.)

**Delta:** Close the system's broken feedback loop by wiring existing measurement signals to automated structural responses and introducing random quality audit as the missing review step.

**Evidence:** Knowledge accretion exploration (4 probes + judge) found only 4/31 interventions work (all structural/signaling), 35% of codebase is governance (accelerating), 0 negative feedback in 1,113 completions. Two architect designs produced implementation plan.

**Knowledge:** Structural > signaling > blocking > advisory > metrics-only. The system measures well but never closes the loop to action. Random audit bootstraps the quality signal that teaches the system what to audit.

**Next:** Integration verification — confirm the full loop works end-to-end.

---

# Plan: Accretion Response and Quality Audit

**Date:** 2026-03-20
**Status:** Active
**Owner:** Dylan

**Extracted-From:**
- `.kb/investigations/2026-03-20-inv-design-accretion-response-layer-wire.md`
- `.kb/investigations/2026-03-20-inv-design-daemon-driven-random-quality-audit.md`
- Exploration synthesis: `orch-go-y163u`

---

## Objective

The completion pipeline is an open loop: agents complete work, the system records 100% success, nothing is learned. This plan closes the loop via two complementary systems: (1) an accretion response layer that wires existing measurement to automated action, and (2) a random quality audit that introduces the first negative feedback signal. Success = `agent.rejected` events > 0 within 30 days, CLAUDE.md stays under 300 lines, and daemon learning adjusts skill metrics from rejection data.

---

## Phases

### Phase 1: Feedback Infrastructure (COMPLETE)

**Goal:** Build the primitives: reject command, learning integration, CLAUDE.md decomposition
**Deliverables:**
- `orch reject` command — `orch-go-c51kt` (closed)
- `RejectedCount` in learning.go — `orch-go-9t5nv` (closed)
- CLAUDE.md decomposition 753 -> ~250 lines — `orch-go-9iedb` (closed)
- Artifact-sync line budget — `orch-go-52dk8` (closed)
**Exit criteria:** All 4 issues closed
**Status:** COMPLETE (all closed 2026-03-20, light tier, no integration verification)

### Phase 2: Signal Wiring (COMPLETE)

**Goal:** Wire existing measurement signals to automated daemon responses
**Deliverables:**
- Accretion.delta event-driven daemon response — `orch-go-ek7bk` (closed)
- Daemon periodic audit selection task — `orch-go-9g5bp` (closed)
- Verdict-to-reject pipeline — `orch-go-ie81t` (closed)
**Exit criteria:** All 3 issues closed
**Depends on:** Phase 1 (reject command + learning.go fix)
**Status:** COMPLETE (all closed 2026-03-20, light tier, no integration verification)

### Phase 3: Integration Verification (PENDING)

**Goal:** Verify the full closed loop works end-to-end
**Deliverables:**
- Integration test: reject -> learning.go picks up rejected count -> daemon audit selects work -> audit agent evaluates -> verdict triggers reject
- Verify CLAUDE.md stayed under budget after decomposition
- Verify accretion.delta events trigger daemon response (not just periodic scan)
**Exit criteria:** Full loop demonstrated with real or simulated data
**Depends on:** Phase 1 + Phase 2
**Status:** PENDING — needs integration issue

### Phase 4: Calibration (FUTURE)

**Goal:** Tune audit parameters from production data
**Deliverables:**
- Audit rate calibration (what N% is right?)
- Accretion.delta threshold tuning (200 lines across 3 completions — correct?)
- Shift from random to risk-weighted audit selection based on rejection data
**Exit criteria:** 30 days of audit data analyzed, parameters adjusted
**Depends on:** Phase 3 (loop must work first)
**Status:** FUTURE — blocked on Phase 3

---

## Dependency Graph

```
Phase 1: Feedback Infrastructure
  orch-go-c51kt (reject)  ──┐
  orch-go-9t5nv (learning) ─┤
  orch-go-9iedb (CLAUDE.md) │   independent
  orch-go-52dk8 (budget)   ─┘
                              │
Phase 2: Signal Wiring        ▼ depends on Phase 1
  orch-go-ek7bk (delta)   ──┐
  orch-go-9g5bp (selection) ─┤
  orch-go-ie81t (verdict)  ──┘ depends on reject + learning
                              │
Phase 3: Integration           ▼ depends on Phase 1 + 2
  [PENDING ISSUE]             │
                              │
Phase 4: Calibration           ▼ depends on Phase 3 + 30 days data
  [FUTURE]
```

---

## Structured Uncertainty

**What's tested:**
- Effectiveness hierarchy confirmed across 31 interventions (exploration probes)
- `accretion.delta` events exist in completion pipeline (verified in code)
- 0 negative feedback in 1,113 completions (verified in events.jsonl)

**What's untested:**
- Whether agents perform better with smaller CLAUDE.md (key missing experiment)
- Whether `orch reject` will get used (UX friction vs review culture)
- Whether 200-line accretion.delta threshold is correctly calibrated
- Whether audit agent can accurately assess intent match (precision/recall unknown)

**What would change this plan:**
- If audit shows 0 rejections after 30 days, the problem is review culture, not tooling — would need mandatory quality assessment in completion flow
- If CLAUDE.md decomposition degrades agent performance, would need to revert and find a different approach to context budgeting

---

## Success Criteria

- [ ] `agent.rejected` events > 0 within 30 days of deployment
- [ ] CLAUDE.md stays under 300 lines for 30 days
- [ ] At least 1 architect extraction issue created from accretion.delta data
- [ ] Governance code percentage does not increase (measure via line count)
- [ ] Daemon learning shows non-zero RejectedCount for at least 1 skill
