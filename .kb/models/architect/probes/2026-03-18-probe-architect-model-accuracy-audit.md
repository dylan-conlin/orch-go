# Probe: Architect Model Accuracy Audit

**Date:** 2026-03-18
**Parent Model:** architect
**Trigger:** Knowledge decay — model had no prior probes

---

## Purpose

Verify whether the architect model's core claims still hold against current codebase state.

---

## Findings

### Claim 1: Decomposition produces higher-value fixes

**Status: STILL SUPPORTED** — The structural pattern holds. The 4 source investigations all demonstrate decomposition value, and no counter-evidence found. This is a general principle, not a codebase-specific claim.

### Claim 2: Phased plans ordered by value-per-effort produce better adoption

**Status: OUTDATED** — The model states "accretion layers 0-1 shipped, layers 2-3 not started." In reality, **all 4 layers shipped:**

| Layer | Model Claim | Actual State |
|-------|-------------|-------------|
| 0 - Spawn gates | Shipped | Shipped (`pkg/spawn/gates/hotspot.go`, 91 lines) — **now advisory** |
| 1 - Completion gates | Shipped | Shipped (`pkg/verify/accretion.go`, 296 lines) — **now advisory** |
| 2 - Coaching plugin | Not started | **Shipped** (`pkg/daemon/architect_escalation.go`, 136 lines) — daemon escalation routing, not coaching detection |
| 3 - CLAUDE.md boundaries | Not started | **Shipped** — CLAUDE.md documents accretion thresholds and enforcement |

Critical update: All gates converted from blocking to advisory per decision `2026-03-17-accretion-gates-advisory-not-blocking.md` after 100% bypass rate over 2-week measurement. The model's framing of "enforcement" no longer matches reality — gates signal, they don't block.

The claim itself (phased ordering → better adoption) is still supported by the evidence: high-value phases shipped first and are still live.

### Claim 3: Investigation→architect→implementation sequence prevents architectural violations

**Status: STILL HYPOTHESIS** — Model correctly marks this as "enforced by spawn gate infrastructure, not experimentally validated." No new experimental evidence found. The architect escalation routing exists (`pkg/daemon/architect_escalation.go`) and routes feature-impl/systematic-debugging to architect for hotspot files, but there's no before/after comparison data.

### Evidence Claim: 6-layer dedup gauntlet → CAS-like collapse to ~60 lines

**Status: PARTIALLY CONTRADICTED** — The gauntlet was refactored but not collapsed to ~60 lines. Current architecture:
- `pkg/daemon/spawn_gate.go` (312 lines) — 5 composable gates in SpawnPipeline
- `pkg/daemon/spawn_execution.go` (277 lines) — execution with beads primary + memory secondary dedup
- Primary dedup uses beads status update to "in_progress" (persistent, CAS-like)
- Architecture is cleaner and testable, but 589 total lines, not ~60

### Evidence Claim: Coaching plugin for accretion

**Status: FALSE** — Coaching plugin (`plugins/coaching.ts`) exists but does NOT detect accretion. It detects frame collapse, user message extraction, and premise-skipping. No accretion coaching metrics found.

### Implication: Architect is a coordination skill, not a planning skill

**Status: STILL SUPPORTED** — The daemon escalation routing and architect investigations all produce implementation issues (beads), not code. The pattern holds.

---

## Summary of Model Accuracy

| Area | Accuracy |
|------|----------|
| Core claims (decomposition, phasing) | Supported but stale |
| Implementation status | **Outdated** — all 4 layers shipped, all advisory |
| Dedup pipeline | **Partially contradicted** — refactored but not collapsed to 60 lines |
| Coaching for accretion | **False** — coaching exists but not for accretion |
| Verification levels | Confirmed |
| Architect routing | Confirmed |
| Structural principles | Still sound |

---

## Recommended Model Updates

1. Update "layers 0-1 shipped, 2-3 not started" → all 4 layers shipped, all advisory
2. Add the advisory-not-blocking decision as key evidence
3. Correct dedup pipeline claim: refactored to composable gates, not collapsed to 60 lines
4. Remove coaching plugin reference from accretion context (coaching doesn't detect accretion)
5. Add probe reference
