# Self-Measurement of an Agent Orchestration System

*March 16, 2026*

---

## 1. What This System Is

This is a report from a system that measures itself.

**orch-go** is a multi-agent orchestration system built by one developer (Dylan Conlin) over 3 months. It coordinates AI coding agents — spawning them, constraining their work through structural enforcement ("gates"), verifying their output, and tracking the whole lifecycle through event telemetry.

The system currently runs Claude Code agents via tmux windows, managed by an autonomous daemon. In the 8-day period covered by this report (March 8-16, 2026), it processed:

- **449 spawns** (agent sessions started)
- **666 completions** (agent work verified and closed)
- **9 abandonments** (2% — agents that got stuck and were killed)
- **15,037 events** logged to `events.jsonl`

312 of those 449 spawns — **69.5%** — were initiated by the daemon without human intervention. One person, orchestrating dozens of concurrent AI agents, with the system itself deciding what to work on next.

The question this report asks: **do the enforcement mechanisms actually work, or are they theater?**

---

## 2. What We Instrumented, and Why

On March 11, 2026, we ran a measurement audit and discovered something uncomfortable: the enforcement infrastructure we'd built over 3 months was largely unmeasured.

- The duplication detector ran on every completion. Nobody knew it cost **111 seconds per run** until we added pipeline timing.
- The accretion delta tracker was supposed to measure file growth. It covered **4.7% of completions** — a path filter bug had silently blinded it for weeks.
- Spawn gates existed for hotspot files, rate limiting, and concurrency. They emitted **zero events**. We knew gates existed but not how often they fired, what they blocked, or whether blocks were correct.
- 52% of `agent.completed` events lacked skill and outcome fields. We were measuring survivors, not decisions.

This is the insight that reframed everything: **enforcement without measurement is theological.** You believe the gate works. You have no evidence.

So we instrumented. Between March 11-16, we added:

- `spawn.gate_decision` events at every gate evaluation (allow, block, bypass)
- Pipeline timing for every completion step
- Enriched completion events with skill, outcome, and duration fields
- Accretion delta coverage fix (4.7% → ~100%)
- A `harness audit` command that surfaces anomalies automatically

The data in this report comes from those instruments.

---

## 3. The Numbers

### 3.1 Agent Lifecycle

| Metric | Value | Period |
|--------|-------|--------|
| Total spawns | 449 | Mar 8-16 (8 days) |
| Total completions | 666 | Mar 8-16 |
| Abandonments | 9 (2.0%) | Mar 8-16 |
| Daemon-initiated spawns | 312 (69.5%) | Mar 8-16 |
| Phase timeouts detected | 149 | Mar 8-16 |
| Duplication detections | 89 | Mar 8-16 |
| Recovery events | 40 | Mar 8-16 |
| Auto-completions (daemon) | 243 | Mar 8-16 |
| Commits | 1,626 | 6 weeks |
| Fix:feat ratio | 0.62 | 28 days |

Completions exceed spawns because the completion count includes agents spawned before the measurement window. The daemon auto-completed 243 sessions — 36.5% of all completions were closed without human involvement.

The 149 phase timeouts represent agents that stopped reporting progress. Some recovered (40 recovery events), some were abandoned (9), most were eventually completed by the daemon's auto-complete logic.

### 3.2 Skill Distribution

| Skill | Spawns | % |
|-------|--------|---|
| feature-impl | 266 | 59.2% |
| systematic-debugging | 80 | 17.8% |
| investigation | 54 | 12.0% |
| architect | 21 | 4.7% |
| capture-knowledge | 7 | 1.6% |
| exploration-orchestrator | 7 | 1.6% |
| research | 6 | 1.3% |
| other | 8 | 1.8% |

Feature implementation dominates. This is expected — the system is actively being built. The 0.62 fix:feat ratio (256 fix commits to 414 feat commits over 28 days) suggests the system generates roughly 6 bugs for every 10 features. Whether that's acceptable depends on context; for a solo developer with 40+ concurrent agents, it's at least not catastrophic.

### 3.3 Gate Audit (30 days, 449 spawns)

This is the uncomfortable part.

| Gate | Invocations | Blocks | Bypasses | Allows | Fire Rate | Coverage |
|------|-------------|--------|----------|--------|-----------|----------|
| triage | 246 | 0 | 98 | 148 | 39.8% | 54.8% |
| verification | 245 | 0 | 0 | 245 | 0.0% | 54.6% |
| hotspot | 244 | 0 | 0 | 244 | 0.0% | 54.3% |
| concurrency | 81 | 0 | 0 | 81 | 0.0% | 18.0% |
| ratelimit | 81 | 0 | 0 | 81 | 0.0% | 18.0% |
| accretion_precommit | 43 | 2 | 2 | 39 | 9.3% | 9.6% |
| drain | 1 | 0 | 0 | 1 | 0.0% | 0.2% |
| governance | 1 | 0 | 0 | 1 | 0.0% | 0.2% |

**5 of 8 gates show zero fires in 30 days.** The harness audit flags them as potentially inert:

```
  ⚠ verification     [zero_fires] 0 fires in 30d — gate may be inert
  ⚠ hotspot          [zero_fires] 0 fires in 30d — gate may be inert
  ⚠ concurrency      [zero_fires] 0 fires in 30d — gate may be inert
  ⚠ ratelimit        [zero_fires] 0 fires in 30d — gate may be inert
  ⚠ drain            [zero_fires] 0 fires in 30d — gate may be inert
```

Three gates also have critically low coverage:

```
  ⚠ concurrency            [low_coverage] 18% coverage (81/449 spawns)
  ⚠ ratelimit              [low_coverage] 18% coverage (81/449 spawns)
  ⚠ accretion_precommit    [low_coverage] 10% coverage (43/449 spawns)
```

The only gate that actually blocks anything is `accretion_precommit` — the pre-commit hook that prevents file growth past 1,500 lines. It fired twice in 30 days. Two blocks. That's the entire enforcement output of 8 gates across 449 spawns.

### 3.4 The Hotspot Gate: A Case Study in Dormancy

The hotspot gate is the most revealing example. It was built to prevent agents from working on critically large files (>1,500 lines). It evaluates on every applicable spawn, costs ~300ms per evaluation, and has blocked exactly zero spawns in 30 days.

Why? **Because it worked.** Previous extraction efforts successfully reduced all files below the 1,500-line threshold. The largest non-test Go file is currently 1,040 lines (`pkg/opencode/client.go`). The gate's triggering condition no longer exists in the codebase.

This creates a philosophical question: is a gate that evaluates but never fires because the problem it was designed to prevent has been solved... a success or a waste? The gate costs ~300ms per spawn — 73 seconds of total compute over 30 days — for zero enforcement value. But removing it means the next time a file crosses 1,500 lines, there's no automatic prevention.

We leave it. But we measure it. And we report the zero honestly.

There's a second problem: 56% of task descriptions don't include file paths. The gate can only match files mentioned explicitly in the task text. A task like "fix the daemon polling logic" would bypass the hotspot gate even if `daemon.go` exceeded 1,500 lines, because the gate doesn't know that "daemon polling logic" lives in `daemon.go`. This is a structural limitation, not a bug — semantic matching is a different complexity class than regex matching.

### 3.5 Duplication Detector: 65% Precision

The duplication detector scans for near-clone functions across the codebase at completion time. It uses AST fingerprinting with an 85% similarity threshold.

A retrospective audit of all 67 detection events (259 match occurrences) revealed:

| Metric | Value |
|--------|-------|
| True positives | 164 (63.3%) |
| False positives | 90 (34.7%) |
| Borderline | 5 (1.9%) |
| **Precision** | **64.6%** |

The false positive breakdown:

| Category | % of FP | Example |
|----------|---------|---------|
| Different semantics | 47% | `parseBeadsIDs ↔ parseBeadsLine` — similar AST, different logic |
| Structural coincidence | 40% | `Logger.Log ↔ WriteCheckpoint` — both write JSONL, different domains |
| Self-match (bug) | 7% | Same function matched against itself |
| Opposite operations | 7% | `CloseIssue ↔ GetComments` — share CLI boilerplate |

A 35% false positive rate means roughly one in three duplication warnings is noise. The largest single source of false positives — `Logger.Log ↔ WriteCheckpoint` (23 occurrences) — isn't even covered by the existing allowlist.

Before this measurement, the assumed false positive rate was 0%. Nobody had checked.

### 3.6 Completion Pipeline

The completion pipeline runs up to 18 gates on every agent's work before closing. The signal/noise classification from a full census:

| Classification | Count | Description |
|----------------|-------|-------------|
| **Signal** | 10 | Gates that catch real problems (build, vet, triage, explain_back, verified, etc.) |
| **Noise** | 5 | Gates that force routine bypasses (hotspot, verification, self_review, architectural_choices, git_diff) |
| **Unknown** | 22+ | Gates with zero data — never triggered in the measurement window |

The self_review gate was the worst offender among noise gates: 33 bypasses, with the dominant false positive being `fmt.Print` statements in intentional CLI output files flagged as "debug statements." It had a **79% false positive rate**. It was subsequently removed.

The completion pipeline's first-try pass rate is **11.8%** (25 of 211). But this is misleading — two "human input" gates (`explain_back` and `verified`) account for 109 of the failures. These gates measure whether the orchestrator performed comprehension steps, not whether the agent's code is correct. Excluding them, the estimated first-try pass rate is 60-70%.

### 3.7 Investigation Orphan Rate

The knowledge base contains **1,213 investigation files**. Of these, **727 have been completed** (Status: Complete or Resolution-Status: Resolved/Synthesized).

That leaves **486 investigations — 40.1%** — that were started but never completed.

The task description cited a 91.3% orphan rate from earlier measurement. The difference reflects either a methodological change (what counts as "orphan") or improvement over time. Either way: between 40% and 91% of investigations started by agents were never concluded. These are artifacts of sessions that ran out of time, got stuck, or were abandoned.

This is the raw throughput cost of autonomous agent work: not every session produces a clean artifact. The system generates investigative debt faster than it can retire it.

### 3.8 Verification Failures

| Metric | Count |
|--------|-------|
| Verification failures | 181 |
| Verification bypasses | 109 |
| Accretion deltas recorded | 185 |

181 verification failures across 666 completions = **27.2% failure rate**. These are agents whose work failed one or more gates on first attempt. The system catches real problems — but the high rate suggests either the gates are too strict, the agents are insufficiently constrained, or both.

### 3.9 Current File Bloat

Despite the gates, 13 files currently exceed 800 lines:

| File | Lines | Status |
|------|-------|--------|
| `pkg/opencode/client.go` | 1,040 | MODERATE |
| `pkg/spawn/learning.go` | 979 | MODERATE |
| `pkg/events/logger.go` | 963 | MODERATE |
| `cmd/orch/stats_aggregation.go` | 959 | MODERATE |
| `cmd/orch/handoff.go` | 898 | MODERATE |
| ... and 8 more | 856-882 | MODERATE |

No files exceed the 1,500-line CRITICAL threshold — which is why the hotspot gate is dormant. But the MODERATE range (800-1,500) is growing. The accretion pre-commit gate warns at 800 lines but doesn't block. Whether warnings without blocks change behavior is unmeasured.

---

## 4. What the Numbers Falsify

### 4.1 "Our gates enforce quality" — Mostly theological

The harness report runs four falsification criteria:

| Criterion | Verdict | Evidence |
|-----------|---------|----------|
| Gates are ceremony (fire but don't change behavior) | **INSUFFICIENT DATA** | Need 2+ weeks of post-gate accretion data |
| Gates are irrelevant (rarely fire) | **FALSIFIED** | 39.8% fire rate for triage gate; legacy bypass events show 69.5% aggregate |
| Soft harness is inert (removing it changes nothing) | **NOT MEASURABLE** | No controlled experiments exist |
| Framework is anecdotal (only works here) | **NOT MEASURABLE** | Single system, no cross-project deployment |

One criterion falsified (gates do fire). One needs more data. Two are structurally unmeasurable with current infrastructure.

The honest summary: we can prove gates fire. We cannot yet prove they improve outcomes.

### 4.2 "Gates prevent problems" — Two blocks in 30 days

Across 8 spawn gates evaluating 449 spawns, exactly **2 blocks** occurred — both from the accretion pre-commit gate. Every other gate either allowed everything or was bypassed.

This doesn't mean the gates are useless. The triage gate's 98 bypasses all have documented reasons ("urgent," "release gate," "Dylan-requested"). The hotspot gate's zero fires reflect successful prior extraction work. But the claim that gates are actively preventing problems is hard to sustain when the blocking output is 2 events in 30 days.

### 4.3 "The duplication detector catches real issues" — 65% of the time

Before measuring precision, the assumption was 0% false positives. The actual number — 35% false positive rate — means the orchestrator sees roughly 1 false alarm for every 2 real duplications. Over time, this trains the operator to ignore duplication warnings, which is the gate calibration death spiral the harness model predicts.

### 4.4 "Investigation drives knowledge" — 40%+ never conclude

Between 40% and 91% of investigations are orphans, depending on measurement methodology. The system produces investigative artifacts at a rate it cannot close. This is the autonomous equivalent of opening more browser tabs than you'll ever read — except each tab cost real compute.

### 4.5 "The completion pipeline is rigorous" — 22 of 37 gates have zero data

More than half the gates in the system have never triggered in the measurement window. They exist in code. They've been tested manually. But in production, they evaluate zero events. We cannot distinguish "gate that works perfectly and never needs to fire" from "gate that is broken and nobody noticed."

---

## 5. What Remains Unmeasurable

### Soft harness effectiveness

The system uses "soft harness" — instructions in CLAUDE.md, skill documents, SPAWN_CONTEXT.md, and knowledge base entries — to guide agent behavior. Unlike hard gates (which mechanically block), soft harness relies on agents reading and following context.

Measuring whether soft harness works requires controlled experiments: spawn identical tasks with and without each soft harness component, compare outcomes. We haven't done this. Contrastive testing of skill content (265 trials) showed knowledge transfers at +5 lift, stance at +2 to +7, and behavioral constraints dilute to inert at 10+ items. But the actual CLAUDE.md and knowledge base injections have never been tested this way.

The model's own position on soft harness: "probably doesn't work until proven otherwise."

### Cross-system generalizability

Everything in this report is from one system, built by one person. The framework might be a universal truth about multi-agent coordination, or it might be an artifact of this particular codebase's history. We cannot tell without deploying the same measurement infrastructure on a second system.

An independent external review (OpenAI Codex, March 10) assessed the framework as "software architecture + CI/policy enforcement + tech debt management with agent vocabulary." The concepts map to established literature: structural attractors = affordances (Norman, 1988); dilution curves = known prompt-length degradation; architecture doing the work of instruction = Christopher Alexander's Pattern Language (1977). The application to LLM agents is the specific contribution. Whether that application is novel enough to constitute a "framework" versus "good engineering practice applied to agents" is an open question.

### Causal direction

When files stay under 1,500 lines, is it because the hotspot gate exists, because the prior extraction work was effective, because agents naturally write smaller files, or because of some other factor entirely? We observe correlation (gates exist, files are small) but cannot establish causation without the counterfactual (what would have happened without gates).

The accretion pre-commit gate was wired March 10. The checkpoint for causal evidence is March 24 — we need at least 2 weeks of post-gate data to compare against the pre-gate baseline of 6,131 lines/week growth in `cmd/orch/`.

### Gate interaction effects

Individual gates are measured. Whether they compose well — whether the combination of 8 spawn gates and 18 completion gates creates a coherent enforcement surface or an incoherent pile of checks — is unmeasured. We know each gate's fire rate but not whether gates conflict, overlap, or create gaps.

---

## 6. What We Learned

### Measurement changes behavior

Before March 11, the harness was built on conviction: gates are good, more gates are better, enforcement prevents entropy. After March 11, with measurement, the picture changed. Gates that appeared vital turned out dormant. Detectors assumed precise had 35% false positive rates. Coverage assumed at 100% was actually at 4.7%.

The act of measuring didn't change the code. It changed what we knew about the code. That changed what we built next.

### Ceremony vs. enforcement is a real distinction

A gate that evaluates every spawn at 300ms cost and never blocks anything is ceremony — it looks like enforcement but provides none. The system has multiple ceremonial gates. Identifying them requires measurement. Removing them requires judgment: a ceremonial gate might become load-bearing when the environment changes.

### Honest reporting is the product

Most systems that measure themselves do so to validate their assumptions. This report was written to challenge ours. The credibility of any enforcement framework rests not on the gates that fire but on the willingness to report the ones that don't.

Five of eight gates never fired. The duplication detector is 65% precise. Between 40% and 91% of investigations are orphans. 22 of 37 gates have zero production data.

These numbers are the system examining itself. They're uncomfortable. They're also the only kind of numbers worth reporting.

---

## Appendix: Data Sources

| Source | Description |
|--------|-------------|
| `~/.orch/events.jsonl` | 15,037 events, March 8-16, 2026 |
| `orch harness report` | Gate deflection, accretion velocity, falsification verdicts |
| `orch harness audit` | 30-day gate audit with anomaly detection |
| `orch hotspot` | Current file bloat analysis |
| `.kb/models/harness-engineering/model.md` | Theoretical framework and evidence synthesis |
| `.kb/investigations/2026-03-11-*` | Gate census, retrospective accuracy audit, measurement gap audit |
| `.kb/models/harness-engineering/probes/2026-03-13-*` | Duplication precision, hotspot gate cost |
| `git log` | 1,626 commits over 6 weeks, fix:feat ratio calculation |

## Appendix: Falsification Status

| Criterion | Test | Threshold | Current Status |
|-----------|------|-----------|----------------|
| Gates are ceremony | Compare accretion velocity pre/post gate | Post-gate velocity <50% of pre-gate for 2+ weeks | **INSUFFICIENT DATA** — checkpoint Mar 24 |
| Gates are irrelevant | Gate fire rate across all spawns | Fire rate <5% = irrelevant | **FALSIFIED** — 39.8% triage fire rate |
| Soft harness is inert | Controlled A/B removal test | No outcome difference | **NOT MEASURABLE** — no experiments |
| Framework is anecdotal | Deploy on second system | No benefit | **NOT MEASURABLE** — single system |
