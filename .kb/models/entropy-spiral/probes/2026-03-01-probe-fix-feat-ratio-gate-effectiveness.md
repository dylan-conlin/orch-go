# Probe: Fix:Feat Ratio Measurement — Do the Gates Work?

**Model:** entropy-spiral
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The entropy-spiral model claims: "During spirals, unverified velocity had negative value (0.96:1 fix:feat ratio). 48 gates now exist to prevent this." Has the fix:feat ratio improved significantly post-gates? Healthy target: less than 0.3:1.

---

## What I Tested

Measured fix:feat ratio from git log on master branch for the 30-day post-rollback period (Feb 12 - Mar 1, 2026). Compared to the spiral-era ratio from the preserved `entropy-spiral-feb2026` branch (Jan 18 - Feb 12, 2026).

```bash
# Post-rollback: classify all 967 commits by conventional-commit prefix
git log --oneline --since="2026-01-30" --until="2026-03-02" --first-parent master | awk '...'

# Spiral-era baseline from preserved branch
git log --format="%s" entropy-spiral-feb2026 --since="2026-01-18" --until="2026-02-12" | python3 '...'

# Weekly trend analysis
git log --format="%aI %s" ... | python3 (weekly bucketing)

# Fix commit classification (infrastructure vs agent vs general)
git log --format="%s" ... | grep "^fix:" | python3 (pattern matching on gate/spawn/daemon/verification keywords)

# Velocity analysis (daily commit counts, threshold violations)
git log --format="%aI" ... | python3 (daily counts, distribution)
```

---

## What I Observed

### 1. Overall Fix:Feat Ratio (barely improved)

| Period | Total Commits | Fixes | Features | Fix:Feat Ratio |
|--------|--------------|-------|----------|----------------|
| Spiral era (Jan 18 - Feb 12) | 1,144 | 204 | 209 | **0.98:1** |
| Post-rollback (Feb 12 - Mar 1) | 967 | 175 | 198 | **0.88:1** |

Improvement: 0.98 → 0.88 (10% reduction). Target was 0.30:1. Current ratio is **2.9x the target**.

### 2. Weekly Trend (worsening)

| Week | Total | Code | Fix | Feat | Fix:Feat | Notes |
|------|-------|------|-----|------|----------|-------|
| W07 (Feb 12-15) | 214 | 142 | 21 | 49 | **0.43:1** | Post-rollback honeymoon |
| W08 (Feb 16-22) | 280 | 205 | 54 | 74 | **0.73:1** | Degrading |
| W09 (Feb 24-Mar 1) | 473 | 239 | 100 | 75 | **1.33:1** | Worse than spiral era |

The ratio regressed from 0.43:1 (good) to 1.33:1 (worse than the spiral era) within 3 weeks.

### 3. Velocity (not gated)

| Metric | Value | Spiral-Era Comparison |
|--------|-------|-----------------------|
| Avg commits/day | 60.4 | 44/day during spiral |
| Median commits/day | 62 | — |
| Max commits/day | 151 | — |
| Days exceeding 45-commit threshold | 11/16 (69%) | 45/day was cited as "can't verify" |

Velocity is **37% higher** than during the spiral. The "verification bottleneck" constraint (system cannot change faster than human can verify) is being violated daily.

### 4. Fix Commit Classification (the gates fix themselves)

| Fix Category | Count | % of Fixes |
|--------------|-------|------------|
| Infrastructure (gates/spawn/daemon/verification) | 72 | 41.9% |
| Agent lifecycle (agent/tmux/session/worker) | 31 | 18.0% |
| General fixes | 69 | 40.1% |

**60% of all fixes are fixing the infrastructure or agent system** — the very systems the gates are designed to protect. The gates themselves have become a source of fix churn.

### 5. Worst Days (fix:feat > 2.0)

| Date | Total | Fix | Feat | Ratio | Pattern |
|------|-------|-----|------|-------|---------|
| Mar 1 | 64 | 19 | 2 | **9.50:1** | Almost all fix work |
| Feb 25 | 48 | 13 | 5 | **2.60:1** | Gate maintenance day |
| Feb 17 | 42 | 11 | 4 | **2.75:1** | Agent lifecycle fixes |

### 6. Noise Volume Growing

| Week | Total Commits | Noise (bd sync, knowledge, probes, docs, etc.) | Noise % |
|------|--------------|------------------------------------------------|---------|
| W07 | 214 | 72 | 34% |
| W08 | 280 | 75 | 27% |
| W09 | 473 | 234 | **49%** |

Nearly half of W09 commits are non-code artifacts. The system is generating increasingly more meta-work.

---

## Model Impact

- [ ] **Confirms** invariant
- [ ] **Contradicts** invariant
- [x] **Extends** model with: Gates prevent catastrophic spirals (no rollback needed in 3 weeks) but introduce their own fix:feat overhead via infrastructure maintenance. The ratio hasn't improved because gates replace spiral-induced churn with maintenance-induced churn. Two findings extend the model:

### Extension 1: Gate Maintenance Cost

The 48 gates generate their own fix churn. 42% of all fixes are infrastructure fixes (gates, spawn, daemon, verification). Adding more gates increases the fix:feat denominator proportionally. The model correctly identifies the spiral mechanism but doesn't account for the steady-state maintenance cost of the gates themselves as a persistent source of fixes.

### Extension 2: Velocity Not Gated

The model's Verification Bottleneck Principle ("system cannot change faster than human can verify") is not being enforced. At 60.4 commits/day (37% above spiral velocity), verification bandwidth is clearly exceeded. The 48 gates are **detection/recovery** mechanisms, not **prevention** mechanisms — they don't limit velocity, they attempt to catch problems after they're created. The model distinguishes prevention from detection but the current implementation is weighted heavily toward detection.

### What the Gates DO Prevent

Despite the poor ratio, no catastrophic spiral requiring rollback has occurred in 3 weeks post-rollback. W07 achieved 0.43:1 — proving that the system CAN operate at healthy ratios. The degradation from W07→W09 suggests the gates slow the descent but don't prevent it. The absence of a full rollback may be the gates' real value — they convert catastrophic spirals into gradual degradation.

---

## Notes

### Why the 0.3:1 Target May Be Wrong

The target assumes a mature, stable system. orch-go is actively developing 48 gates, a dashboard, agent lifecycle management, and knowledge infrastructure simultaneously. Some fix:feat pressure is inherent to building infrastructure that monitors itself. A more nuanced target might be:
- Infrastructure fix:feat: tolerate higher ratio (system is young)
- Application fix:feat: should be < 0.3:1 (the actual signal)

### Implications for Model Update

The entropy-spiral model should distinguish:
1. **Spiral-mode fixes** — agents fixing other agents' mistakes (what the 0.96:1 measured)
2. **Maintenance-mode fixes** — infrastructure upkeep (42% of current fixes)
3. **Application fixes** — actual bug fixes in features (40% of current fixes)

Only category 1 indicates a spiral. The current ratio (0.88:1) conflates all three, making it a noisy signal for spiral detection.

### Raw Numbers for Posterity

```
Post-rollback period: Feb 12 - Mar 1, 2026 (16 active days)
Total commits: 967
fix: 175 (18.1%)
feat/add/implement: 198 (20.5%)
refactor: 29 (3.0%)
chore/cleanup: 45 (4.7%)
bd sync: 153 (15.8%)
probe: 35 (3.6%)
investigation: 26 (2.7%)
docs: 55 (5.7%)
other: 220 (22.7%)
```
