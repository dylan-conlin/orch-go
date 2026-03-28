# Probe: Issue Creation Dedup — Upstream vs Spawn-Time Effectiveness

**Model:** Architect Skill Effectiveness
**Date:** 2026-03-27
**Status:** Complete

---

## Question

The architect model claims "Architect output should be implementation issues, not code" — but does the skill's issue creation path check whether work is already committed before creating follow-ups? The CommitDedupGate catches duplicates at spawn-time (daemon level), but if the architect never checks git history before `bd create`, the duplicate issue exists in beads regardless of whether it gets spawned. Does moving dedup upstream to issue-creation-time reduce duplicate issue noise?

**Model claims being tested:**

> Architect output should be implementation issues, not code. All 4 investigations produced prioritized implementation plans that became beads issues.

> Defect Class 6: Duplicate Action — Same action performed multiple times, no dedup. Prevention: Idempotency keys.

---

## What I Tested

1. Examined current architect skill template (SKILL.md.template) for any pre-creation dedup guidance
2. Examined worker-base discovered-work.md for any pre-creation dedup guidance
3. Reviewed CommitDedupGate in pkg/daemon/prior_art_dedup.go to understand spawn-time dedup
4. Reviewed the duplicate extraction provenance trace (probe 2026-02-16) for root cause analysis
5. Designed and implemented skill-level pre-creation dedup check

---

## What I Observed

### Current state: No pre-creation dedup in any skill

- The architect SKILL.md.template Phase 5d instructs `bd create` for component/integration/implementation issues with zero dedup guidance
- Worker-base discovered-work.md instructs `bd create` for bugs/tasks/features with zero dedup guidance
- ALL dedup happens at spawn-time in the daemon's SpawnPipeline (7 gates: L1-L7)
- CommitDedupGate (L6) specifically catches "architect created follow-up for work already done by another agent" — but by then the issue exists in beads

### The gap

The current architecture has a **create-then-filter** pattern:
1. Architect creates issue via `bd create` (no dedup)
2. Issue enters beads with `triage:ready` status
3. Daemon picks up issue, runs SpawnPipeline
4. CommitDedupGate (L6) rejects if commits exist → issue sits in beads forever as zombie

The upstream fix is **filter-then-create**:
1. Before `bd create`, check if recent commits address the proposed work
2. Before `bd create`, check if an open issue already covers the same scope
3. Only create if no prior art found

### Evidence: Duplicate extraction provenance trace

The 2026-02-16 probe documented 9+ duplicate extraction issues created because the daemon's extraction logic had no convergence check. While that was a daemon-specific bug, the same pattern applies to architect follow-ups: the issue creation path has no check for "is this work already done?"

---

## Model Impact

- [ ] **Extends** Claim 2 (phased implementation): The phasing principle should extend to issue creation — phased dedup means catching duplicates at the earliest possible point (creation-time), not just at spawn-time.

- [ ] **Extends** with new architectural insight: Dedup is a **layered concern** that should exist at multiple levels: (1) issue-creation-time in skills, (2) spawn-time in daemon gates, (3) execution-time via session dedup. The current architecture only has layers 2 and 3.

- [ ] **Confirms** Defect Class 6 (Duplicate Action) applicability: The architect skill's `bd create` path is exposed to Class 6 with no prevention pattern. Adding a pre-creation git log check implements the "idempotency key" prevention pattern at the skill level.
