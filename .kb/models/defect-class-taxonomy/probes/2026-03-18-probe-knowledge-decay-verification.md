# Probe: Defect Class Taxonomy — Knowledge Decay Verification

**Date:** 2026-03-18
**Model:** defect-class-taxonomy
**Trigger:** 999d since last probe (model created 2026-03-03, never probed)
**Method:** Classified 29 fix commits since model creation + verified structural prevention claims against codebase

---

## Claim Verification

### Claim 1: Seven defect classes cover the majority of fix commits

**Verdict: CONFIRMED**

Classified 29 fix commits (2026-03-03 to 2026-03-18). All 29 fit one of the 7 classes:

| Class | Count | % |
|-------|-------|---|
| 1 (Filter Amnesia) | 6 | 21% |
| 3 (Stale Artifact Accumulation) | 6 | 21% |
| 4 (Cross-Project Boundary Bleed) | 5 | 17% |
| 5 (Contradictory Authority Signals) | 4 | 14% |
| 0 (Scope Expansion) | 2 | 7% |
| 2 (Multi-Backend Blindness) | 2 | 7% |
| 6 (Duplicate Action) | 2 | 7% |
| 7 (Premature Destruction) | 2 | 7% |

This addresses the model's open question about what percentage of fixes fit the taxonomy. In this 15-day window: 100% (29/29). Some classifications are arguable but none required a new class.

---

### Claim 2: Dependency graph — upstream classes enable downstream

**Verdict: CONFIRMED with nuance**

The model predicts Class 3 (stale artifacts) enables Class 0 (scope expansion). In recent commits, Class 3 and Class 0 remain active but the cross-project expansion (Class 4) has overtaken Class 0 as the primary "expansion vector." This is consistent with the model's note that Class 4 "multiplies exposure for scope expansion."

Class 4 is now the dominant expansion driver, not Class 3→0. The dependency graph should be updated to reflect this observed shift.

---

### Claim 3: Structural prevention measures

| Prevention | Model Claim | Current State | Verdict |
|------------|-------------|---------------|---------|
| Allowlist scanner pattern (Class 0) | Three-layer defense proposed | Not implemented as systematic pattern | UNIMPLEMENTED |
| Canonical filter functions (Class 1) | Shared filters replace per-consumer reimplementation | Partially adopted (`FilterActiveIssues`, `CheckIssueCompliance`) but not systematic | PARTIAL |
| Backend-aware query (Class 2) | `DiscoverLiveAgents()` is "embryo" | Matured: queries both OpenCode + tmux, deduplicates, widely adopted | EVOLVED — no longer embryo |
| Eliminate `beads.DefaultDir` (Class 4a) | Global state variable to eliminate | No instances found in codebase — fully eliminated | RESOLVED |
| Single canonical status (Class 5) | `ListUnverifiedWork()` is "embryo" | Two canonical functions with explicit precedence: `ListUnverifiedWork()` (verification) + `determineAgentStatus()` (completion priority cascade) | EVOLVED — dual specialization, no oscillation |
| Idempotency keys (Class 6) | Recommended structural fix | Not implemented — still uses layered TTL approach (7 patches) | UNIMPLEMENTED |
| Liveness verification (Class 7) | Check 2+ signals before destruction | Pattern followed in practice but not enforced | INFORMAL |

---

### Claim 4: Multi-backend blindness will stabilize post-transition

**Verdict: TRENDING CONFIRMED**

Only 2 of 29 fixes (7%) were Class 2, down from 15+ historical instances. The `DiscoverLiveAgents()` function now covers the primary discovery path. The `doctor_defect_scan.go` automated scanner checks for Class 2 violations with a 60+ function allowlist. This suggests the class is stabilizing as the model's open question predicted was possible.

---

### Claim 5: Class 5 oscillation pattern

**Verdict: RESOLVED**

The model documents fix oscillation (Dec 28 → Jan 6 → Jan 8) where each fix was correct for its trigger but wrong for the opposite case. The current `determineAgentStatus()` function implements a 6-level priority cascade that replaces the oscillating signal precedence. No evidence of oscillation in recent commits — 4 Class 5 fixes in the window are measurement/filtering issues, not status derivation oscillation.

---

## New Finding: Automated Defect Scanning

`cmd/orch/doctor_defect_scan.go` implements automated scanning for Classes 2 and 5, integrated into `orch doctor`. This is not mentioned in the model but represents significant structural progress — continuous monitoring for the two classes that had the most architectural investment.

---

## New Finding: Cross-Project (Class 4) is the rising class

5 of 29 fixes (17%) are Class 4 — driven by daemon multi-project expansion. Sub-patterns 4b (CWD assumption) and 4c (missing project context) are the active instances. Sub-pattern 4a (beads.DefaultDir global) is fully resolved.

---

## Summary

The defect class taxonomy remains **accurate and useful** after 15 days. All 7 classes are still producing instances. The structural fixes have progressed unevenly:

- **Mature:** Classes 2 (backend-aware query), 4a (global elimination), 5 (canonical status)
- **Partial:** Class 1 (some shared filters), Class 7 (informal practice)
- **Unimplemented:** Classes 0 (allowlist scanner), 6 (idempotency keys)

**Model updates needed:**
1. Mark `DiscoverLiveAgents()` as mature, not "embryo"
2. Mark `determineAgentStatus()` as the Class 5 canonical fix, not just `ListUnverifiedWork()`
3. Mark Class 4a (`beads.DefaultDir`) as resolved
4. Note Class 4 as the rising expansion vector, overtaking the Class 3→0 chain
5. Add `doctor_defect_scan.go` to references as automated scanning infrastructure
6. Update Class 2 instance rate trend (declining, 7% of recent fixes)
