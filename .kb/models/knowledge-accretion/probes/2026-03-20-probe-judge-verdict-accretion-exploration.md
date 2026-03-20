# Probe: Judge Verdict — Knowledge Accretion Exploration (4-Probe Evaluation)

**Model:** knowledge-accretion
**Date:** 2026-03-20
**Status:** Complete
**claim:** KA-05, KA-10, substrate-generalization, CV-07
**verdict:** extends

---

## Question

Four parallel probes investigated the knowledge accretion problem from different angles: (1) prompt context as accreting substrate, (2) intervention effectiveness audit, (3) governance infrastructure self-accretion, (4) human feedback channel structural disuse. Do these sub-findings hold up under cross-examination? Where do they converge, contradict, or leave gaps?

---

## What I Tested

Read all 4 probe files and verified key claims against primary sources:

```bash
# Verified CLAUDE.md line count (probe 1 claims 753)
wc -l CLAUDE.md  # 753 — confirmed

# Verified default model contradiction (probe 1)
grep -n 'DefaultModel\|defaultModel' pkg/model/model.go  # claude-sonnet-4-5-20250929
# CLAUDE.md line ~267: "google/gemini-3-flash-preview"
# CLAUDE.md line ~561: "Opus (Max subscription)"
# Three different values — confirmed

# Verified probe-to-model merge gate is blocking (probe 2)
grep 'result.Passed = false' pkg/verify/probe_model_merge.go  # line 97 — confirmed

# Verified accretion precommit is advisory (probe 2)
grep 'advisory\|never block' pkg/verify/accretion_precommit.go
# "Files over 1500 lines emit advisory warnings and events. Never blocks." — confirmed

# Verified event counts (probe 4)
grep -c '"agent.completed"' ~/.orch/events.jsonl    # 1113 (slightly higher than probe's 1102)
grep -c '"agent.reworked"' ~/.orch/events.jsonl      # 0 — confirmed
grep -c '"agent.abandoned"' ~/.orch/events.jsonl     # 11 — confirmed

# Verified governance infrastructure sizes (probe 3)
find pkg/verify -name "*.go" ! -name "*_test.go" -exec cat {} + | wc -l   # 9242 — confirmed
find pkg/opencode -name "*.go" ! -name "*_test.go" -exec cat {} + | wc -l # 2396 — confirmed
find pkg/daemon -name "*.go" ! -name "*_test.go" -exec cat {} + | wc -l   # 17104 — confirmed

# Verified daemon periodic task count (probe 3)
grep 'RunPeriodic' cmd/orch/daemon_periodic.go | grep -v '//' | grep -v 'func ' | wc -l  # 26 — confirmed

# Verified rework command exists (probe 4)
wc -l cmd/orch/rework_cmd.go  # 355 — confirmed
grep 'agent.reworked' pkg/events/logger.go  # event type exists — confirmed
```

Every quantitative claim I spot-checked matched primary data exactly (or within trivial margin for the completion count, which grew by 11 during the investigation window).

---

## What I Observed

### Per-Finding Verdicts

#### Probe 1: Prompt Context as Accreting Substrate

| Dimension | Score (1-5) | Rationale |
|-----------|-------------|-----------|
| **Evidence Quality** | 5 | All primary data. Git log growth trajectory, manual content classification, SPAWN_CONTEXT size distribution across 129 workspaces. The three-way default model contradiction is a verified concrete example. |
| **Internal Consistency** | 5 | No contradictions within the probe. Growth rates, content type ratios, and stacking calculations are internally coherent. |
| **Model Alignment** | 5 | Directly extends the substrate generalization table with a new confirmed substrate. Tests the five conditions and finds all met. |
| **Actionability** | 4 | Proposed gates (section TTL, contradiction detection, size budget, relevance filtering) are concrete and implementable. Docked 1 point because relevance filtering requires semantic understanding that's hard to automate. |
| **Novelty** | 5 | The model's substrate table explicitly did not cover prompt context. The "read amplification" property (every token consumed by every agent) is a genuinely new insight — code accretion has localized blast radius, prompt accretion has total blast radius. |

**Overall Verdict: ACCEPTED**

This is the strongest of the four probes. The 92% reference / 8% directive finding is particularly powerful — it quantifies the signal-to-noise problem in a way the model hasn't previously captured. The contributor analysis (62% agent-committed, 35% automated sync) explains the accretion mechanism with primary evidence.

---

#### Probe 2: Intervention Effectiveness Audit

| Dimension | Score (1-5) | Rationale |
|-----------|-------------|-----------|
| **Evidence Quality** | 5 | Systematic audit of all 31 interventions against source code, event data, and decision history. Every status claim (implemented/removed/not-implemented) verified against code. |
| **Internal Consistency** | 4 | Mostly consistent. One minor tension: claims "every blocking gate followed designed→measured→found-inert→downgraded" but the model-stub precommit gate and probe-to-model merge gate are blocking and NOT downgraded. The claim should be "every *advisory-convertible* blocking gate" — structurally unbypassable gates don't follow this arc. Probe acknowledges this but the summary headline overstates. |
| **Model Alignment** | 5 | Directly tests the model's intervention taxonomy and finds the model's gate deficit table is stale. The contradiction (model says "zero hard knowledge gates" but two exist) is a genuine model correction. |
| **Actionability** | 5 | The effectiveness hierarchy (structural attractors > signaling > blocking > advisory > metrics-only) is immediately actionable: stop building advisory gates, invest in structural attractors and daemon-triggered responses. |
| **Novelty** | 4 | The 13% effectiveness rate is novel quantification. The effectiveness hierarchy partially extends the model's attractor taxonomy (Section 2). Docked 1 point because the "blocking gates get bypassed" observation was already documented in `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` — the probe synthesizes known findings rather than discovering new ones. |

**Overall Verdict: ACCEPTED**

The quantitative scorecard (31 interventions, 4 effective) is the probe's signature contribution. It transforms the model from "here are interventions we could try" to "here are interventions we tried and here's what worked." The gate lifecycle arc is a valuable pattern even if individual observations were known.

---

#### Probe 3: Governance Infrastructure Self-Accretion

| Dimension | Score (1-5) | Rationale |
|-----------|-------------|-----------|
| **Evidence Quality** | 4 | Line counts and file categorization are verified against primary data (I spot-checked pkg/verify, pkg/opencode, pkg/daemon — all match exactly). Docked 1 point for the categorization methodology: the 5-category taxonomy (Core, Governance, Measurement, Knowledge, Meta/Infrastructure) required judgment calls. Some packages straddle categories (e.g., `pkg/coaching/` could be Core or Governance; `pkg/events/` is infrastructure used by everything). The probe doesn't document its categorization decisions transparently. |
| **Internal Consistency** | 4 | The "35% of codebase is governance/measurement/knowledge" claim groups three categories. The conclusion "the cure is becoming the disease" depends on treating all three as "accretion management" — but Knowledge Management (5%) includes `kb` commands, `thread`, `review` which are also core product features, not just accretion management. The probe doesn't distinguish "measuring accretion" from "managing knowledge as a product feature." Governance alone (11%) is a much smaller number. |
| **Model Alignment** | 5 | Directly tests KA-10 ("anti-accretion mechanisms can themselves accrete"). The 18%→23% growth acceleration is the strongest evidence KA-10 has received. |
| **Actionability** | 3 | The probe identifies the problem but proposes no remediation. "The cure is becoming the disease" is diagnostic but doesn't say what to do about it. Should governance be pruned? Which of the 22 daemon tasks should be removed? The probe stops at measurement. |
| **Novelty** | 4 | The quantification (35%, 85% of daemon tasks, 76% of events) is new. The self-referential observation ("this probe is itself an instance of the pattern") is intellectually honest. Docked 1 point because KA-10 already predicted this; the probe confirms rather than extends. |

**Overall Verdict: ACCEPTED (with reservations)**

The categorization methodology needs more transparency. The "35%" headline conflates governance (11%), measurement (19%), and knowledge management (5%) in a way that overstates the problem. Governance-only accretion (11%) is concerning but not crisis-level. The 85% daemon task ratio is more compelling evidence because it's less dependent on categorization judgment.

---

#### Probe 4: Human Feedback Channel Structural Disuse

| Dimension | Score (1-5) | Rationale |
|-----------|-------------|-----------|
| **Evidence Quality** | 5 | Primary data throughout: event counts from events.jsonl (verified), source code read of rework/abandon/complete commands with line-number citations, all 11 abandon reasons examined and categorized. The friction audit (8 steps for rework vs 1 step for re-spawn) is concrete and verifiable. |
| **Internal Consistency** | 5 | Claims are mutually reinforcing without contradiction. The friction asymmetry (rework=hard, complete=easy) explains the observed data (0 reworks, 1113 completions) perfectly. |
| **Model Alignment** | 4 | Tests the completion-verification model's §7, not the knowledge-accretion model directly. Extends accretion understanding through a secondary mechanism: without negative feedback, the learning loop can't distinguish good from bad work, so accretion of low-quality artifacts goes unchecked. The connection is real but indirect. |
| **Actionability** | 5 | Five concrete recommendations with clear implementation paths: `orch reject`, review UX integration, stats separation, signal taxonomy, model staleness fix. These are immediately implementable. |
| **Novelty** | 5 | The "false ground truth problem" (100% success rate because failure paths don't exist) is a novel insight that connects to the measurement-honesty model. The distinction between operational abandons (all 11) and quality abandons (0) hasn't been made before. |

**Overall Verdict: ACCEPTED**

This is the most actionable of the four probes. The 0-rework finding is a striking data point, and the structural explanation (friction asymmetry + no reject verb + daemon auto-complete bypass) is convincing. The model staleness correction (§7 says infrastructure is missing; infrastructure exists but is unusable) is a genuine contribution.

---

### Contested Findings

#### 1. "35% of codebase is accretion management" (Probe 3) vs effectiveness hierarchy (Probe 2)

Probe 3 frames the governance infrastructure as self-accretion (the cure becoming the disease). Probe 2 shows that 4 of the 31 interventions actually work. These findings are in tension: if only 13% of interventions work, the infrastructure supporting the ineffective 87% is dead weight — but probe 3 counts all governance infrastructure equally without weighting by effectiveness. The honest framing: ~13% of governance infrastructure contributes to effective interventions; the rest is measurement/monitoring that doesn't drive action (probe 2's "metrics-only" category).

**Resolution:** Both probes are correct in their observations. The synthesis is: governance infrastructure accreted (probe 3), most of it is ineffective (probe 2), and the specific infrastructure that works (structural attractors, daemon signal cascades) is a small fraction of the total. The 35% number is real but misleadingly presented as uniformly "accretion management" when much of it is inert measurement.

#### 2. "Zero hard knowledge gates" (model claim) vs "Two hard gates exist" (Probe 2)

Probe 2 finds two hard knowledge gates (probe-to-model merge, model-stub precommit) that the model's gate deficit table didn't account for. But the model was already updated (I can see "Correction (2026-03-20)" in the model text), so this was a same-day fix. Not a deep contradiction but a documentation staleness issue — the very kind of accretion artifact that probe 1 identifies.

#### 3. Prompt context accretion (Probe 1) vs daemon governance overhead (Probe 3)

Probe 1 shows CLAUDE.md growing 8x in 91 days with 35% automated commits. Probe 3 shows the daemon running 22 governance tasks. These are connected: the daemon's governance tasks generate artifacts (events, findings, extractions) that feed back into SPAWN_CONTEXT.md (via KB growth) and potentially into CLAUDE.md (via artifact sync). Neither probe traces this feedback loop explicitly.

---

### Coverage Gaps

**What the 4 probes together still don't cover:**

1. **Skill accretion dynamics.** Probe 1 measures skill file sizes but doesn't analyze growth trajectories or whether skill content degrades (the skill-content-transfer model's "mechanical staleness" failure mode). The ux-audit skill at 2,827 lines is flagged but not investigated.

2. **Agent performance degradation from context bloat.** Probe 1 shows 2,400+ lines of injected context per agent but doesn't measure whether this actually degrades agent performance. Do agents with heavier context windows produce worse work? Are completion rates different for agents with 2,400 vs 1,200 lines of context? This is the key missing experiment.

3. **Cross-project accretion.** All four probes examine orch-go in isolation. The `.kb/global/` knowledge and skill files are shared across projects. Accretion in shared artifacts would multiply the problem.

4. **Temporal dynamics of knowledge quality.** The probes measure quantity (lines, counts, rates) but not quality over time. Are model claims becoming more accurate? Are investigations getting more precise? Quality degradation from accretion is hypothesized but not measured.

5. **The pruning mechanism.** Probe 1 notes two CLAUDE.md pruning events followed by re-growth. What triggered the pruning? Who did it? Why didn't it stick? Understanding the pruning failure mechanism would inform whether any proposed gate can resist the re-accretion ratchet.

6. **Runtime cost of governance.** Probe 3 counts governance daemon tasks (22 of 26) but doesn't measure their runtime cost. Do the 22 tasks take 85% of daemon cycle time, or are most trivial? CPU time is a different kind of accretion than code lines.

---

### Synthesis Notes

**Key themes that emerge from reading all 4 together:**

**Theme 1: The system's primary feedback loop is broken.** Probe 4 shows zero negative signal entering the learning loop (0 reworks, 0 quality abandons). Probe 2 shows most interventions don't reduce accretion. Probe 3 shows the system responds to this by building more measurement infrastructure. The cycle: problem detected → intervention built → intervention doesn't work → more measurement built to understand why → measurement doesn't drive action → more intervention built. This is the accretion spiral applied to accretion management itself.

**Theme 2: Structural coupling beats behavioral guidance.** Across all four probes, the same pattern: things embedded in structure work, things that rely on agent behavior don't. The model/probe directory system works (structural). Advisory gates don't work (behavioral). Auto-complete works effortlessly (structural path of least resistance). Rework doesn't work (behavioral friction). This aligns with the skill-content-transfer model's finding that attention primers beat action directives.

**Theme 3: The blast radius asymmetry is underappreciated.** Probe 1's key insight — prompt context accretion has total blast radius (every agent, every session) while code accretion has local blast radius (only agents touching that code) — reframes the priority. A 10-line addition to CLAUDE.md costs more system-wide than a 100-line addition to `daemon_periodic.go`. Yet there are no gates on CLAUDE.md and multiple gates on code files.

**Theme 4: Measurement without action is itself accretion.** Probe 2 identifies "metrics-only" as the lowest tier of the effectiveness hierarchy. Probe 3 shows 19% of the codebase is measurement infrastructure. Probe 4 shows the measurements produce false confidence (100% success rate). The measurement infrastructure exists, collects data, but the data doesn't close the loop back to behavioral change. This is the "false ground truth" problem at system scale.

---

## Model Impact

- [x] **Extends** model with: Cross-probe synthesis reveals four meta-themes (broken feedback loop, structural > behavioral, blast radius asymmetry, measurement-as-accretion) that the individual probes don't articulate. The model should incorporate these as cross-cutting observations.

Specific model updates recommended:
1. Add prompt context to the substrate generalization table (from probe 1)
2. Replace "zero hard knowledge gates" with current gate deficit table including 2 hard gates (from probe 2, already partially done)
3. Add intervention effectiveness hierarchy and 13% effectiveness rate (from probe 2)
4. Add KA-10 quantitative confirmation: 35% governance, 85% daemon tasks, 18%→23% acceleration (from probe 3)
5. Cross-reference completion-verification model for the false ground truth / broken feedback loop connection (from probe 4)

---

## Notes

- The completion count increased from 1,102 (probe 4's measurement) to 1,113 during the investigation window — 11 completions in the time it took to analyze them. This illustrates the velocity of the system: accretion continues during accretion analysis.
- All four probes were produced by the same system (Claude agents running in orch-go). The probes themselves are instances of the knowledge accretion pattern: 4 new probe files, each ~150-200 lines, adding to the .kb/ directory. This judge verdict adds a 5th. The meta-recursion is inescapable.
- Probe 3's self-aware note ("this probe itself is an instance of the pattern it describes") deserves elevation: the inability to study accretion without accreting is a fundamental constraint of the system, not just a cute observation.
