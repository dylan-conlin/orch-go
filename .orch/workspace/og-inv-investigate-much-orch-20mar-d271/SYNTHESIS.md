# Session Synthesis

**Agent:** og-inv-investigate-much-orch-20mar-d271
**Issue:** ad-hoc (no beads tracking)
**Duration:** 2026-03-20
**Outcome:** success

---

## TLDR

35% of orch-go's 138K non-test Go lines (48,187 lines) exist to detect, measure, and govern accretion — and this percentage is accelerating (18% → 23% of files in March alone). The daemon runs 22 governance periodic tasks vs 4 core tasks (85% governance). This empirically confirms claim KA-10: the accretion management infrastructure is itself accreting, meeting all five conditions from KA-02. The cure is becoming the disease.

---

## Plain-Language Summary

I categorized every Go file in orch-go by purpose: core agent orchestration, governance/gates, measurement/stats, knowledge management, and infrastructure. The result: about a third of the codebase exists not to orchestrate agents, but to watch, measure, and govern the orchestration. The daemon — supposed to be an autonomous agent manager — spends 85% of its periodic cycles on meta-work like reflection, drift detection, and friction accumulation. This governance share was stable at 18% for months but jumped to 23% in March, growing faster than the core system. This is the exact pattern the knowledge-accretion model (KA-10) predicts: anti-accretion mechanisms that lack their own coordination mechanism will accrete just like the substrates they govern.

---

## Delta (What Changed)

### Files Created
- `.kb/models/knowledge-accretion/probes/2026-03-20-probe-governance-infrastructure-self-accretion.md` — Probe confirming KA-10
- `.kb/investigations/2026-03-20-inv-investigate-accretion-management-overhead.md` — Full investigation with D.E.K.N.

### Files Modified
- `.kb/models/knowledge-accretion/model.md` — Added probe reference, updated "Observed in" to include governance self-accretion
- `.kb/models/knowledge-accretion/claims.yaml` — Added new evidence to KA-10, updated last_validated date

### Commits
- (pending)

---

## Evidence (What Was Observed)

- **35% of codebase is governance/measurement/KB**: 48,187 of 138,050 non-test Go lines
- **Core is 43%**: 59,242 lines (spawn, complete, daemon, agent lifecycle)
- **Daemon is 85% governance**: 22 of 26 periodic tasks are governance/measurement
- **Events are 76% governance**: 64 of 84 event types are governance/measurement
- **Governance accelerating**: 18% → 23% of files in 20 days (March 2026)
- **pkg/verify/ (9,242 lines) is 4x pkg/opencode/ (2,396 lines)**: verification infrastructure dwarfs the API client

### Tests Run
```bash
# Line counts per category
for f in cmd/orch/*.go; do wc -l < "$f"; done | sort -rn
find pkg/$d -name "*.go" -not -name "*_test.go" -exec cat {} + | wc -l

# Periodic task enumeration
grep 'RunPeriodic' cmd/orch/daemon_periodic.go  # → 26 tasks

# Event type count
grep -E '^\| `[a-z]' CLAUDE.md | wc -l  # → 84

# File growth timeline
git ls-tree -r --name-only "$commit" -- cmd/orch/ pkg/  # at 7 dates
```

---

## Architectural Choices

No architectural choices — task was pure investigation/measurement.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/knowledge-accretion/probes/2026-03-20-probe-governance-infrastructure-self-accretion.md` — Probe confirming KA-10 with quantitative evidence

### Constraints Discovered
- The governance layer itself satisfies all five accretion conditions (KA-02) — no meta-governance mechanism exists
- The March acceleration (18%→23%) suggests governance is not at equilibrium

---

## Verification Contract

See probe file for methodology. Key outcomes verified:
- Line counts reproducible via `wc -l` on all files
- Periodic task count reproducible via `grep RunPeriodic`
- File growth timeline reproducible via `git ls-tree` at dated commits

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete (probe, investigation, model update, claims update)
- [x] Investigation file has Status: Complete
- [x] Ready for review

**Strategic question for Dylan:** Is 35% governance the right equilibrium for orch-go, or should governance be consolidated/pruned? The answer depends on whether orch-go is primarily a production tool or a research platform for studying accretion.

---

## Unexplored Questions

- **Is the governance effective?** 35% overhead is only a problem if the governance doesn't work. The 100% accretion gate bypass rate suggests some of it isn't working, but this probe didn't measure effectiveness broadly.
- **What's the governance equilibrium?** Is there a natural ceiling, or will governance approach 50% and beyond?
- **Does the governance layer slow the daemon?** 26 periodic tasks per cycle may have runtime cost (related to the behavioral accretion probe from earlier today).

---

## Friction

No friction — smooth session.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-investigate-much-orch-20mar-d271/`
**Investigation:** `.kb/investigations/2026-03-20-inv-investigate-accretion-management-overhead.md`
