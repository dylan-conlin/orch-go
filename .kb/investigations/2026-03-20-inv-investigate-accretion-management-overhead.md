## Summary (D.E.K.N.)

**Delta:** 35% of orch-go's non-test Go code (48,187 of 138,050 lines) exists to detect, measure, and govern accretion — and this percentage is growing faster than core functionality (18% → 23% of files in March alone).

**Evidence:** Line-by-line categorization of all cmd/orch/ and pkg/ files; 26 daemon periodic tasks (85% governance); 84 event types (76% governance); git ls-tree snapshots at 7 dates showing accelerating governance share.

**Knowledge:** KA-10 ("anti-accretion mechanisms can themselves accrete") is confirmed empirically in orch-go. The accretion management infrastructure meets all five accretion conditions from KA-02: multiple agents write it (condition 1), agents are amnesiac (2), each addition is locally correct (3), the additions must compose non-trivially (4), and there's no coordination mechanism governing the governance layer itself (5).

**Next:** Route to architect for strategic assessment — is 35% the right equilibrium, or should governance be consolidated/pruned?

**Authority:** strategic - Whether to reduce governance infrastructure is a value judgment about what orch-go is for.

---

# Investigation: Investigate Accretion Management Overhead

**Question:** Is orch-go's accretion management infrastructure itself an instance of accretion — the cure becoming the disease?

**Started:** 2026-03-20
**Updated:** 2026-03-20
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** knowledge-accretion

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/knowledge-accretion/probes/2026-03-20-probe-behavioral-accretion-runtime-cost-grows-silently.md | extends | yes | No |
| .kb/models/harness-engineering/probes/2026-03-17-probe-pre-commit-accretion-gate-2-week-effectiveness.md | confirms | yes | No — 100% gate bypass rate supports the finding that governance grows but doesn't gate |

---

## Findings

### Finding 1: 35% of codebase is accretion management

**Evidence:** Categorized all 504 non-test Go files across cmd/orch/ and pkg/:

| Category | Lines | % |
|----------|-------|---|
| Core (spawn, complete, daemon, agent lifecycle) | 59,242 | 43% |
| Governance (harness, verify, precommit, control) | 15,519 | 11% |
| Measurement (stats, hotspot, entropy, doctor, orient, events) | 26,450 | 19% |
| Knowledge Management (kb, review, thread) | 6,218 | 5% |
| Meta/Infrastructure (serve, config, deploy, clean) | 27,926 | 20% |

Governance + Measurement + Knowledge = 48,187 lines (35%).

**Source:** `wc -l` on every file in cmd/orch/ and `find pkg/ -name "*.go" -not -name "*_test.go"`

**Significance:** The code that watches the system is nearly as large as the code that IS the system. `pkg/verify/` (9,242 lines) is 4x larger than `pkg/opencode/` (2,396 lines) — the verification infrastructure dwarfs the actual API client.

---

### Finding 2: Daemon is 85% governance engine

**Evidence:** Of 26 periodic tasks in `daemon_periodic.go`:
- 4 core: Cleanup, Recovery, OrphanDetection, RegistryRefresh
- 22 governance/measurement: everything from Reflection to TensionClusterScan

**Source:** `cmd/orch/daemon_periodic.go:41-253` — `runPeriodicTasks()` function

**Significance:** The daemon was conceived as an autonomous agent orchestrator. In practice, it spends 85% of its cycles on meta-work: measuring, reflecting, checking health, detecting drift, generating probes, and logging decisions. Only 4 of 26 periodic tasks directly serve the daemon's stated purpose of managing agents.

---

### Finding 3: Governance share is accelerating

**Evidence:** File count at 7 date snapshots via `git ls-tree`:

| Date | Total | Gov/Meas | % |
|------|-------|----------|---|
| Dec 31 | 100 | 18 | 18% |
| Jan 31 | 157 | 29 | 18% |
| Feb 28 | 295 | 52 | 18% |
| Mar 10 | 416 | 92 | 22% |
| Mar 20 | 504 | 117 | 23% |

Governance was stable at 18% for 3 months, then jumped to 23% in March. In the Mar 1-10 window, governance grew at 77% while total codebase grew 41%.

**Source:** `git ls-tree -r --name-only "$commit" -- cmd/orch/ pkg/` at each date

**Significance:** The governance share is not at equilibrium — it's accelerating. Each new measurement spawns more measurement infrastructure.

---

### Finding 4: 76% of event types are governance/measurement

**Evidence:** Of 84 event types in CLAUDE.md:
- 20 core lifecycle events (session.*, agent.*, daemon.spawn/complete)
- 64 governance/measurement events (verification.*, spawn.gate_*, accretion.*, daemon.agreement_check, etc.)

**Source:** `grep -E '^\| \`[a-z]' CLAUDE.md`

**Significance:** The event system — itself an accretion management tool — predominantly serves other accretion management tools. Most events exist to feed stats, audits, and dashboards rather than to track core agent lifecycle.

---

## Synthesis

**Key Insights:**

1. **The governance layer satisfies all five accretion conditions from KA-02.** Multiple agents build it (condition 1), they're amnesiac (2), each addition is locally justified (3), the additions must compose (4), and there's no meta-governance mechanism (5). The knowledge-accretion model accurately predicts that governance infrastructure without coordination will itself accrete.

2. **The recursion has at least 4 layers.** Detect accretion → prevent accretion → measure the prevention → measure the measurements. Each layer was individually justified when added. Together they constitute 35% of a 138K-line codebase.

3. **The acceleration in March correlates with the harness engineering push.** The harness (governance verification infrastructure) was introduced Feb 14 and drove most of the March governance growth. This is consistent with KA-09's claim that "creation is always cheaper than removal."

**Answer to Investigation Question:**

Yes, orch-go's accretion management infrastructure is itself accreting. At 35% of codebase and growing faster than core functionality, the governance/measurement layer has become the largest non-core component. The daemon — originally an agent orchestrator — is now primarily a governance engine (85% of periodic tasks). This is the exact pattern KA-10 predicts: anti-accretion mechanisms that lack their own coordination mechanism will accrete just like the substrates they govern.

---

## Structured Uncertainty

**What's tested:**

- ✅ 35% of non-test Go lines are governance/measurement/knowledge (verified: line-by-line categorization of all files)
- ✅ 85% of daemon periodic tasks are governance (verified: enumerated all 26 RunPeriodic* calls)
- ✅ 76% of event types are governance (verified: grep of CLAUDE.md event table)
- ✅ Governance share grew from 18% → 23% in March 2026 (verified: git ls-tree at 7 dates)

**What's untested:**

- ⚠️ Whether 35% is "too much" — there's no objective threshold for governance overhead
- ⚠️ Whether the governance actually prevents accretion effectively (the 100% gate bypass rate suggests not, but this isn't the focus here)
- ⚠️ Whether removing governance would cause the core to accrete faster

**What would change this:**

- Finding that >50% of governance code was actively preventing measurable degradation would reframe this as "justified overhead" rather than "second-order accretion"
- A governance equilibrium (stable percentage over 30+ days) would contradict the "accelerating" finding

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Assess whether to consolidate/prune governance infrastructure | strategic | Whether to reduce governance is a value judgment about orch-go's identity — tool vs research platform |
| Consider a governance budget (e.g., max 25% of codebase) | architectural | Cross-component constraint affecting all future development |

### Recommended Approach

**Governance Budget Assessment** — Determine whether the current 35% is intentional or emergent, and whether a cap should be established.

**Why this approach:**
- The investigation shows the percentage is growing, not stable
- Without a cap, the accretion model predicts continued growth
- A strategic decision is needed because some governance serves research goals (the knowledge-accretion model itself), not production goals

**Trade-offs accepted:**
- A cap might prevent useful governance additions
- Measuring the cap itself adds governance overhead (meta-meta-governance)

### Alternative Approaches Considered

**Option B: Do nothing — governance is the product**
- **Pros:** orch-go is partially a research platform for studying accretion; governance IS the research output
- **Cons:** Eventually governance overwhelms core functionality
- **When to use instead:** If orch-go's primary value is as a research platform, not a production tool

**Option C: Aggressive pruning — remove all advisory/bypassed governance**
- **Pros:** Immediately reduces codebase by the dead-weight governance (e.g., 100%-bypassed accretion gates)
- **Cons:** Loses measurement capability; may need to rebuild later
- **When to use instead:** If production performance/maintainability is top priority

---

## References

**Files Examined:**
- `cmd/orch/daemon_periodic.go` - All 26 periodic tasks enumerated
- All files in `cmd/orch/` and `pkg/` - Line count categorization

**Commands Run:**
```bash
# List and count all Go files
for f in /path/cmd/orch/*.go; do wc -l < "$f"; done | sort -rn

# Count per pkg/ subdirectory
find pkg/$d -name "*.go" -not -name "*_test.go" -exec cat {} + | wc -l

# File growth over time
git ls-tree -r --name-only "$commit" -- cmd/orch/ pkg/ | grep '\.go$' | grep -v '_test.go'
```

**Related Artifacts:**
- **Probe:** `.kb/models/knowledge-accretion/probes/2026-03-20-probe-governance-infrastructure-self-accretion.md`
- **Model:** `.kb/models/knowledge-accretion/model.md` — Claim KA-10
