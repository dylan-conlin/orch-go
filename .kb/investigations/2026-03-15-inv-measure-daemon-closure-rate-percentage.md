## Summary (D.E.K.N.)

**Delta:** 84% of daemon-spawned agents achieve a correctly closed loop (verified or KB artifact); 28.7% produce KB artifacts — investigation and architect skills lead at 63% and 47% KB rates.

**Evidence:** Queried all 14,936 events in events.jsonl, cross-referenced 307 daemon.spawn events with agent.completed, accretion.delta, and session.auto_completed events.

**Knowledge:** Feature-impl and systematic-debugging produce KB artifacts only 25-28% of the time, which is expected since their primary output is code, not knowledge. The 21 never-completed agents (6.8%) represent the real loss — dead spawns that consumed resources without closure.

**Next:** Close. The 84% closure rate is a baseline metric. Consider tracking never-completed agents as a daemon health signal.

**Authority:** implementation - Pure measurement, no architectural decisions needed.

---

# Investigation: Measure Daemon Closure Rate Percentage

**Question:** What percentage of daemon-produced completions end in a KB artifact or correctly closed loop? What's the breakdown by work type?

**Started:** 2026-03-15
**Updated:** 2026-03-15
**Owner:** orch-go-magh6
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** harness-engineering

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-11-design-measurement-surface-harness-falsification.md | extends | yes (measurement surface exists in events.jsonl) | - |
| .kb/models/harness-engineering/probes/2026-03-11-probe-measurement-surface-design-falsification.md | extends | yes (measurement surface design was feasible) | - |

---

## Findings

### Finding 1: Headline Closure Rate — 84.0%

**Evidence:** Of 307 daemon-spawned agents, 258 achieved a correctly closed loop (defined as verification_passed=true OR produced a .kb/ artifact in accretion.delta). Breakdown:

| Category | Count | % of Spawned |
|----------|-------|-------------|
| KB + Verified | 88 | 28.7% |
| Verified only | 170 | 55.4% |
| KB only | 0 | 0.0% |
| Just closed (no verification, no KB) | 28 | 9.1% |
| Never completed | 21 | 6.8% |

**Source:** `~/.orch/events.jsonl` — cross-referenced daemon.spawn (307), agent.completed (270 matching daemon spawns), session.auto_completed (196 matching), accretion.delta (88 with .kb/ files).

**Significance:** The system closes 84% of daemon-spawned work correctly. The 28 "just closed" agents (9.1%) completed but without formal verification — these are primarily cross-project agents (toolshed, pw, kb-cli) where verification may work differently. The 21 never-completed (6.8%) are dead spawns.

---

### Finding 2: KB Artifact Rate Varies by Skill

**Evidence:**

| Skill | Spawned | Completed | KB Art | Verified | KB% | Close% |
|-------|---------|-----------|--------|----------|-----|--------|
| feature-impl | 185 | 177 | 44 | 151 | 25% | 82% |
| systematic-debugging | 68 | 58 | 16 | 56 | 28% | 82% |
| investigation | 33 | 30 | 19 | 30 | 63% | 91% |
| architect | 17 | 17 | 8 | 17 | 47% | 100% |
| research | 4 | 4 | 1 | 4 | 25% | 100% |

**Source:** Same events.jsonl query, grouped by daemon.spawn skill field.

**Significance:** Investigation (63%) and architect (47%) skills produce KB artifacts at high rates — expected since their primary output is knowledge. Feature-impl (25%) and systematic-debugging (28%) produce KB artifacts secondarily (discovered work, investigations found during implementation). Architect has 100% closure rate — no daemon-spawned architect has ever failed to complete.

---

### Finding 3: KB Artifact Type Distribution

**Evidence:** Of the 88 daemon agents producing KB artifacts, the breakdown by artifact type:

| KB Subcategory | Unique Agents |
|---------------|---------------|
| .kb/investigations/ | 60 |
| .kb/models/ (non-probe) | 43 |
| .kb/probes/ | 26 |
| .kb/decisions/ | 19 |
| .kb/guides/ | 18 |
| .kb/sessions/ | 15 |
| .kb/global/ | 14 |
| .kb/quick/ | 13 |
| .kb/threads/ | 10 |
| .kb/plans/ | 8 |
| .kb/synthesis/ | 2 |
| .kb/publications/ | 1 |

(Agents often produce multiple artifact types — a single investigation agent may create an investigation file + probe + model update.)

**Source:** accretion.delta file_deltas paths, filtered to .kb/ prefix with lines_added > 0.

**Significance:** Investigations are the dominant KB output (60 agents), followed by model updates (43) and probes (26). The 13 agents producing .kb/quick/ artifacts represent ad-hoc knowledge capture during work. The probe-to-model pipeline is active (26 probes, 43 model updates).

---

### Finding 4: The 28 "Just Closed" Agents

**Evidence:** These 28 agents completed (reported Phase: Complete) but lacked both verification_passed=true in agent.completed events and any KB artifact in accretion.delta. Breakdown:
- 18 are cross-project agents (toolshed-*, pw-*, kb-cli-*) — verification infrastructure may differ
- 10 are orch-go agents that completed before verification was instrumented or had edge-case closure paths

All 28 reported "Phase: Complete" with substantive completion messages — none appear to be abandoned or failed work.

**Source:** Manual review of completion reasons for all 28 agents.

**Significance:** These are not failures — they're completions that didn't produce evidence visible in orch-go's events.jsonl. Cross-project agents close via their own project's beads, and early orch-go agents predate verification instrumentation.

---

### Finding 5: The 21 Never-Completed Agents

**Evidence:** 21 daemon-spawned agents have no agent.completed or session.auto_completed event:
- 10 systematic-debugging
- 8 feature-impl
- 3 investigation (including this current session orch-go-magh6)

These include agents that were superseded (e.g., trigger layer Phase 1 → replaced by Phase 2), blocked agents, and agents that died from API/context errors.

**Source:** daemon.spawn events with no matching completion event.

**Significance:** 6.8% dead spawn rate is the primary inefficiency. These represent wasted compute — the daemon spawned work that never finished. This is a more actionable metric than KB artifact rate for daemon health.

---

## Synthesis

**Key Insights:**

1. **84% correctly closed loop** — The headline number. 258 of 307 daemon-spawned agents either passed verification or produced KB artifacts. This is the daemon's closure rate.

2. **KB artifact production is skill-dependent, not a universal quality signal** — Investigation (63%) and architect (47%) produce KB by design. Feature-impl (25%) produces KB incidentally. A feature-impl agent that writes code, passes tests, and closes its issue has "correctly closed the loop" even without a KB artifact.

3. **The real loss is dead spawns (6.8%), not missing KB artifacts** — The 21 never-completed agents consumed daemon slots and compute without producing any value. Tracking this as a daemon health metric would be more actionable than KB artifact rate.

**Answer to Investigation Question:**

**84.0% of daemon-produced completions end in a correctly closed loop** (verified or KB artifact). **28.7% specifically produce KB artifacts.** By work type: investigation leads at 63% KB rate, architect at 47%, feature-impl and systematic-debugging at 25-28%. The remaining 16% splits into "completed but unverified" (9.1%, mostly cross-project) and "never completed" (6.8%, dead spawns).

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon closure rate of 84.0% (verified: queried all 14,936 events, cross-referenced 307 daemon.spawn with agent.completed/auto_completed/accretion.delta)
- ✅ KB artifact rate of 28.7% (verified: counted accretion.delta events with .kb/ file paths for daemon-spawned agents)
- ✅ Skill breakdown (verified: grouped by daemon.spawn skill field, verified totals sum correctly)

**What's untested:**

- ⚠️ Whether "just closed" cross-project agents have verification in their own project's events.jsonl (would require reading pw/toolshed/kb-cli events)
- ⚠️ Whether dead spawns are recoverable or represent permanent loss (would need to check if issues were re-spawned)
- ⚠️ Quality of KB artifacts produced (counted existence, not whether content was substantive)

**What would change this:**

- If cross-project verification events exist in other projects' events.jsonl, the "just closed" count would shrink and closure rate would increase
- If dead spawns were re-spawned under new beads IDs, the effective closure rate is higher than measured
- If KB artifact quality is low (template stubs, minimal content), the 28.7% KB rate overstates knowledge production

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Track dead spawn rate as daemon health metric | implementation | Adds measurement within existing event infrastructure |

### Recommended Approach: Track Dead Spawn Rate

**Track dead spawn rate as daemon health metric** - Emit a daemon health event that includes never-completed spawn count as a percentage.

**Why this approach:**
- Dead spawns (6.8%) are the most actionable inefficiency signal
- Already have daemon.beads_health event infrastructure
- More actionable than KB artifact rate (which varies by skill design)

**Trade-offs accepted:**
- Not tracking KB artifact quality (just existence)
- Cross-project agents remain opaque

**Implementation sequence:**
1. Add never-completed count to daemon periodic health check
2. Surface in `orch harness report` as "spawn completion rate"

---

## References

**Files Examined:**
- `~/.orch/events.jsonl` - All 14,936 events queried for daemon.spawn, agent.completed, session.auto_completed, daemon.complete, accretion.delta

**Commands Run:**
```bash
# Count events by type
python3 -c "import json; ..." # grouped all event types

# Cross-reference daemon spawns with completions
python3 -c "..." # matched 307 daemon.spawn beads_ids against completion events

# Analyze KB artifact subcategories
python3 -c "..." # classified .kb/ file paths from accretion.delta events
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-03-11-design-measurement-surface-harness-falsification.md - Measurement surface design
- **Probe:** .kb/models/harness-engineering/probes/2026-03-11-probe-measurement-surface-design-falsification.md - Measurement feasibility

---

## Investigation History

**2026-03-15:** Investigation started
- Initial question: What percentage of daemon-produced completions end in a KB artifact or correctly closed loop?
- Context: Spawned by daemon as measurement task for harness engineering model

**2026-03-15:** Data collection complete
- Queried 14,936 events, identified 307 daemon-spawned agents
- Cross-referenced with 5 event types for comprehensive closure analysis

**2026-03-15:** Investigation completed
- Status: Complete
- Key outcome: 84.0% daemon closure rate; 28.7% KB artifact rate; 6.8% dead spawn rate
