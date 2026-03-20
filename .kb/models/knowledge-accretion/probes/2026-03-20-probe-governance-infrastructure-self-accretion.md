# Probe: Governance Infrastructure Self-Accretion — Does the Cure Become the Disease?

**Model:** knowledge-accretion
**Date:** 2026-03-20
**Status:** Complete
**claim:** KA-10
**verdict:** confirms

---

## Question

KA-10 claims: "Anti-accretion mechanisms can themselves accrete or create second-order pathologies — cure without meta-coordination shifts accretion to a different dimension."

Is orch-go's governance/measurement infrastructure itself an instance of accretion? Does the code that exists to detect, measure, and manage accretion constitute a significant fraction of the codebase — and is it growing faster than core functionality?

---

## What I Tested

Categorized every non-test `.go` file in `cmd/orch/` and `pkg/` into five categories:

1. **Core** — spawn, complete, status, daemon loop, session, agent lifecycle, beads, opencode client
2. **Governance** — harness, precommit, verification gates, control plane, coaching
3. **Measurement** — stats, hotspot, entropy, health, patterns, doctor, orient, audit, events, claims, modeldrift, findingdedup, dupdetect, artifactsync, digest, daemonconfig
4. **Knowledge Management** — kb commands, review, thread, backlog
5. **Meta/Infrastructure** — serve, config, deploy, servers, hooks, clean, plan, learn, init

Then counted:
- Lines per category across `cmd/orch/` and `pkg/`
- Daemon periodic tasks (26 total) categorized as core vs governance/measurement
- Event types in CLAUDE.md (84 total) categorized as core lifecycle vs governance/measurement
- File count growth over time via `git ls-tree` at 7 date snapshots

```bash
# Category line counts (cmd/orch/ non-test .go files)
for f in <file-list>; do wc -l < "cmd/orch/$f"; done

# pkg/ category line counts
find "pkg/$pkg" -name "*.go" -not -name "*_test.go" -exec cat {} + | wc -l

# Periodic task enumeration
grep 'RunPeriodic' cmd/orch/daemon_periodic.go | grep -v '//' | grep -v 'func '

# File growth over time
git ls-tree -r --name-only "$commit" -- cmd/orch/ pkg/ | grep '\.go$' | grep -v '_test.go'
```

---

## What I Observed

### 1. Codebase Category Breakdown (138,050 non-test Go lines)

| Category | cmd/orch/ | pkg/ | Total | % |
|----------|-----------|------|-------|---|
| Core | 14,567 | 44,675 | 59,242 | 43% |
| Governance | 4,293 | 11,226 | 15,519 | 11% |
| Measurement | 12,736 | 13,714 | 26,450 | 19% |
| Knowledge Mgmt | 5,652 | 566 | 6,218 | 5% |
| Meta/Infra | 17,490 | 10,436 | 27,926 | 20% |

**Governance + Measurement + Knowledge = 48,187 lines (35% of codebase)**

The code that watches, measures, and governs the system is roughly equal to the code that *does* the system's core job (spawn agents, complete work, manage lifecycle).

### 2. Daemon Periodic Tasks: 85% Governance/Measurement

Of 26 daemon periodic tasks, only 4 are core operations:
- **Core (4):** Cleanup, Recovery, OrphanDetection, RegistryRefresh
- **Governance/Measurement (22):** Reflection, ModelDriftReflection, KnowledgeHealth, PhaseTimeout, QuestionDetection, AgreementCheck, BeadsHealth, FrictionAccumulation, ArtifactSync, SynthesisAutoCreate, LearningRefresh, PlanStaleness, ProactiveExtraction, TriggerScan, TriggerExpiry, Digest, InvestigationOrphan, VerificationFailedEscalation, LightweightCleanup, ClaimProbeGeneration, CapacityPoll, TensionClusterScan

The daemon runs 22 meta-tasks for every 4 core tasks. The daemon is primarily a governance engine, not an agent orchestrator.

### 3. Event Types: 76% Governance/Measurement

Of 84 event types:
- **Core lifecycle (20):** session.spawned/completed/ended, agent.completed/abandoned/resumed, daemon.spawn/complete, etc.
- **Governance/measurement (64):** verification.*, spawn.gate_*, daemon.agreement_check, accretion.*, duplication.*, trigger.outcome, artifact.drift, etc.

### 4. Growth Timeline: Governance Accelerating

| Date | Total Files | Gov/Meas Files | Gov/Meas % |
|------|-------------|----------------|------------|
| 2025-12-31 | 100 | 18 | 18% |
| 2026-01-15 | 151 | 29 | 19% |
| 2026-01-31 | 157 | 29 | 18% |
| 2026-02-15 | 207 | 37 | 18% |
| 2026-02-28 | 295 | 52 | 18% |
| 2026-03-10 | 416 | 92 | **22%** |
| 2026-03-20 | 504 | 117 | **23%** |

Governance/measurement was stable at ~18% of codebase from Dec 2025 through Feb 2026. In March 2026, it jumped to 23% — governance grew at **77% rate** (Mar 1-10) vs 41% for total codebase in the same period. The accretion management infrastructure is accreting faster than the system it manages.

### 5. The Recursion

The codebase now contains:
- Code to **detect** accretion (hotspot analysis, entropy measurement, duplication detection)
- Code to **prevent** accretion (spawn gates, precommit checks, verification gates)
- Code to **measure** accretion (stats aggregation, harness audit, gate effectiveness analysis)
- Code to **measure the measurements** (harness gate effectiveness, trigger outcome tracking, decision logging)
- Code to **manage the knowledge about accretion** (claims, probes, model drift, tension clusters, synthesis auto-create)

Each layer exists because the previous layer's behavior needed monitoring. This is textbook second-order accretion.

---

## Model Impact

- [x] **Confirms** claim KA-10: "Anti-accretion mechanisms can themselves accrete"

This is the strongest empirical confirmation KA-10 has received. The numbers:
- 35% of the codebase exists to manage accretion (governance + measurement + knowledge management)
- 85% of daemon periodic tasks are governance/measurement
- 76% of event types are governance/measurement
- Governance share jumped from 18% → 23% in a single month (March 2026), growing faster than the system it governs

The pattern matches exactly what KA-10 predicts: each anti-accretion mechanism (gates, metrics, audits) is itself locally correct and individually justified. But they compose into a governance infrastructure that now rivals the core system in size. The cure is becoming the disease.

**Key nuance:** This doesn't mean the governance is *wrong* — some percentage is necessary. The question is whether 35% represents equilibrium or runaway. The March acceleration (18% → 23%) suggests it's still growing.

---

## Notes

- The `pkg/daemon/` package (17,104 lines) is the largest single package, far exceeding the next largest (`pkg/spawn/` at 12,292). Most of daemon's bulk is periodic task infrastructure for governance/measurement.
- The `pkg/verify/` package (9,242 lines) is larger than `pkg/opencode/` (2,396 lines) — the verification infrastructure is 4x the size of the actual API client that talks to the system being orchestrated.
- This probe itself is an instance of the pattern it describes: an investigation into whether the investigation infrastructure has grown too large.
- The accretion gate was converted from blocking to advisory after a 100% bypass rate (per `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md`). The infrastructure to implement, measure, and eventually disable this gate is still in the codebase.
