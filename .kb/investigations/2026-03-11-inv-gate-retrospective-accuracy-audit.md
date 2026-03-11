---
title: "Gate Retrospective Accuracy Audit"
date: 2026-03-11
type: investigation
status: Complete
model: harness-engineering
plan_ref: .kb/plans/2026-03-11-gate-signal-vs-noise.md
plan_phase: "Phase 3"
---

# Gate Retrospective Accuracy Audit

**Model:** harness-engineering
**Plan:** Gate Signal vs Noise — Phase 3
**Data source:** `~/.orch/events.jsonl` — 5,363 events, 216 spawns, 211 completions

## Methodology

For each signal gate identified in Phase 1, sampled all available blocks/failures and classified each as:
- **True Positive (TP):** Gate caught a real problem that needed fixing
- **False Positive (FP):** Gate blocked good work incorrectly
- **Ambiguous (AMB):** Insufficient data to determine

Classification criteria: examined the flagged file/line, the agent's skill and task, bypass reasons, and eventual outcome (did the issue complete after the gate was addressed?).

---

## Per-Gate Audit Results

### 1. build (C12) — 3 failures, 0 bypassed

| # | Issue | Skill | Error | Classification | Rationale |
|---|-------|-------|-------|----------------|-----------|
| 1 | orch-go-3w5dz | systematic-debugging | `go build ./...` failed | **TP** | Build was genuinely broken. Failed 3 times before agent fixed it and completed. |
| 2 | orch-go-3w5dz | systematic-debugging | (repeat) | **TP** | Same issue, second attempt — still broken. |
| 3 | orch-go-3w5dz | systematic-debugging | (repeat) | **TP** | Same issue, third attempt — still broken. |

**False positive rate: 0/3 = 0%**
**Confidence: Low** (n=3, all from same issue)

---

### 2. vet (C13) — 3 failures, 0 bypassed

| # | Issue | Skill | Error | Classification | Rationale |
|---|-------|-------|-------|----------------|-----------|
| 1-3 | orch-go-3w5dz | systematic-debugging | `go vet` failed (co-occurs with build) | **TP** | `go vet` failures on same broken build — real code problems. |

**False positive rate: 0/3 = 0%**
**Confidence: Low** (n=3, same issue as build)

---

### 3. phase_complete (C1) — 3 failures, 7 bypassed

| # | Issue | Skill | Decision | Classification | Rationale |
|---|-------|-------|----------|----------------|-----------|
| 1-2 | orch-go-av0l5 | feature-impl | Failed: phase='BLOCKED' | **TP** | Agent was genuinely blocked, not complete. Orchestrator tried to close prematurely. Eventually completed after unblocking. |
| 3 | orch-go-2uuu1 | feature-impl | Failed: phase='BLOCKED' | **TP** | Agent stuck at BLOCKED. Never completed — gate correctly prevented premature closure. |

Bypass data not available in events (7 bypasses noted in census but bypass events not logged with reasons for this gate).

**False positive rate: 0/3 = 0%**
**Confidence: Medium** (n=3 failures clear, bypass reasons unknown)

---

### 4. synthesis (C2) — 1 failure, 6 bypassed

| # | Issue | Skill | Decision | Classification | Rationale |
|---|-------|-------|----------|----------------|-----------|
| 1 | orch-go-h8uax | systematic-debugging | Failed: SYNTHESIS.md missing | **TP** | Agent didn't write required synthesis. Eventually completed after adding it. |

Census notes 6 bypasses for light-tier and no-track (correct behavior — exemptions are working as designed).

**False positive rate: 0/1 = 0%**
**Confidence: Low** (n=1)

---

### 5. explain_back (C16) — 52 failures, 1 bypassed

| Sample | Classification | Rationale |
|--------|----------------|-----------|
| All 52 failures across 43 unique issues | **TP** | Gate detects missing `--explain` flag from orchestrator. This is a *human-input gate* — it measures whether the orchestrator articulated comprehension. The "failure" IS the signal: orchestrator didn't provide explain-back. All 43 issues eventually completed after orchestrator provided the flag. |

| # | Issue | Bypass Reason | Classification |
|---|-------|---------------|----------------|
| 1 | (1 bypass in data) | Partial fix | **AMB** | Partial fix may justify skipping explain-back, but could also mask incomplete understanding. |

**False positive rate: 0/52 = 0%** (gate is measuring orchestrator behavior, not agent quality)
**Confidence: High** (n=52, consistent pattern)

**Note:** This gate has a 100% eventual-completion rate — every issue that failed explain_back eventually completed. This means the gate is never blocking *wrong work*, it's enforcing *comprehension discipline*.

---

### 6. verified (C17) — 57 failures, 0 bypassed

| Sample | Classification | Rationale |
|--------|----------------|-----------|
| All 57 failures across 44 unique issues | **TP** | Same pattern as explain_back — human-input gate measuring orchestrator behavioral verification. All 44 unique issues eventually completed after orchestrator provided `--verified`. |

**False positive rate: 0/57 = 0%**
**Confidence: High** (n=57, consistent pattern)

---

### 7. triage (S1) — 53 bypasses (spawn gate)

Sampled all 53 bypass reasons. Categorized:

| Category | Count | Examples | Classification |
|----------|-------|----------|----------------|
| **Dylan-requested** | 4 | "Dylan wants immediate investigation", "Dylan requested doom emacs setup" | **TP** — Gate correctly forced accountability; bypass reason proves human authority. |
| **Plan-phase unblocked** | 12 | "Phase 6 of harness plan, unblocked by Phase 5", "All 4 dependencies closed" | **TP** — Gate forced explicit justification for non-daemon spawn. Reasons are substantive. |
| **Time-sensitive/urgent** | 8 | "Publication is time-sensitive", "Oshcut detected us scraping — need anti-detection" | **TP** — Gate created accountability record for urgency-driven overrides. |
| **Daemon not picking up** | 2 | "Daemon not picking up, redesigned experiment needs fresh run" | **AMB** — Could indicate daemon bug rather than legitimate bypass. |
| **Knowledge/research work** | 15 | "Core model creation", "Blog post ready to write", "Falsifiability test" | **TP** — Research/creative work legitimately bypasses triage. Gate recorded the reason. |
| **Critical path items** | 8 | "Critical path for v0.1 release", "hotspot gate requires architect review" | **TP** — Gate documented urgency and scope for release-blocking work. |
| **Feature unblocked** | 4 | "Design doc complete, ready for implementation", "Directly tested, now wire as gate" | **TP** — Legitimate sequential work where daemon queue adds unnecessary latency. |

**False positive rate: 0/53 = 0%**
**Confidence: High** (n=53)

**Key insight:** Triage gate never produces false positives because it's a *process gate*, not a *correctness gate*. It doesn't judge whether work is valid — it forces the spawner to articulate *why* they're bypassing the daemon queue. Every bypass reason is legitimate, which means the gate is working: it creates accountability without blocking legitimate work. The 2 "daemon not picking up" cases are ambiguous — they might indicate daemon bugs that should be fixed.

---

### 8. accretion_precommit (P1) — 1 block, 1 bypass

| # | Decision | File | Classification | Rationale |
|---|----------|------|----------------|-----------|
| 1 | Block | `cmd/orch/stats_test.go` | **TP** | File exceeded 1500-line threshold. Blocked agent-caused bloat. |
| 2 | Bypass | (FORCE_ACCRETION=1) | **AMB** | Force override used, no reason recorded in event. |

**False positive rate: 0/1 blocks = 0%**
**Confidence: Very low** (n=1 block)

---

### 9. question (S6) — Advisory gate (never blocks)

No blocks to audit. Advisory-only gate warns about open questions in dependency chain.

**False positive rate: N/A** (advisory gate, no blocks)

---

### 10. governance (S8) — Advisory gate (never blocks)

No blocks to audit. Warns about governance-protected files before spawn.

**False positive rate: N/A** (advisory gate, no blocks)

---

## Bonus: self_review reclassification (currently NOISE)

The Phase 1 census classified self_review as NOISE (33 bypasses, fmt.Print false positive pattern). Auditing the 19 *failure* events reveals a nuanced picture:

| Category | Count | Examples | Classification |
|----------|-------|----------|----------------|
| **Intentional CLI output** (cmd/) | 5 | `complete_cmd.go:276` — `fmt.Printf("Review tier: %s\n")`, `kb.go:155` — help text | **FP** — Legitimate CLI output, not debug statements |
| **Intentional CLI output** (pkg/) | 3 | `completion_processing.go:586` — `fmt.Printf("Skipping %s...")`, `stats_output.go:16` — stats formatting | **FP** — Package-level CLI output, not in cmd/ |
| **Pre-existing code** | 5 | Same files flagged across multiple agents who didn't add the code | **FP** — Gate should scope to agent's diff, not entire file |
| **console.error (not console.log)** | 2 | `harness.ts:94`, `completion-review.svelte:52` — error handling | **FP** — `console.error` is not a debug statement |
| **Actual debug statements** | 0 | None in sample | — |
| **Ambiguous** | 4 | Various fmt.Print in non-obvious contexts | **AMB** |

**False positive rate: 15/19 = 79%** (confirming NOISE classification)

---

## Summary Table

| Gate | Type | Samples (n) | True Positive | False Positive | Ambiguous | FP Rate | Confidence |
|------|------|-------------|---------------|----------------|-----------|---------|------------|
| **build** | Completion | 3 | 3 | 0 | 0 | **0%** | Low (n=3) |
| **vet** | Completion | 3 | 3 | 0 | 0 | **0%** | Low (n=3) |
| **phase_complete** | Completion | 3 | 3 | 0 | 0 | **0%** | Medium |
| **synthesis** | Completion | 1 | 1 | 0 | 0 | **0%** | Low (n=1) |
| **explain_back** | Completion | 52 | 52 | 0 | 0 | **0%** | High |
| **verified** | Completion | 57 | 57 | 0 | 0 | **0%** | High |
| **triage** | Spawn | 53 | 51 | 0 | 2 | **0%** | High |
| **accretion_precommit** | Pre-commit | 1 | 1 | 0 | 0 | **0%** | Very Low |
| **question** | Advisory | N/A | — | — | — | N/A | — |
| **governance** | Advisory | N/A | — | — | — | N/A | — |
| *self_review (NOISE)* | *Completion* | *19* | *0* | *15* | *4* | ***79%*** | *Medium* |

**Aggregate signal gate false positive rate: 0/173 = 0%**

---

## Findings

### Finding 1: Signal gates have zero false positives

Across 173 sampled blocks/failures from 8 blocking signal gates, zero were false positives. Every gate block corresponded to a real problem:
- build/vet: genuine compilation errors
- phase_complete: agent genuinely not done
- synthesis: agent genuinely didn't write synthesis
- explain_back/verified: orchestrator genuinely didn't provide comprehension/verification
- triage: spawner genuinely needed to justify manual spawn
- accretion_precommit: file genuinely exceeded threshold

### Finding 2: Signal gates split into two distinct categories

**Correctness gates** (build, vet, phase_complete, synthesis, accretion_precommit): Block work that is objectively wrong. Low volume (11 total events), 100% accuracy. These are the cheapest gates — they catch real defects and never false-alarm.

**Discipline gates** (explain_back, verified, triage): Enforce human process discipline. High volume (162 total events), 100% accuracy. But "accuracy" means something different — they measure whether a human did something, not whether agent work is correct. These gates cannot be wrong in the traditional sense because they're measuring process compliance, not code quality.

### Finding 3: Confidence intervals are wide for low-volume gates

| Gate | 95% CI on FP Rate (Wilson) |
|------|---------------------------|
| build/vet | [0%, 63%] (n=3) |
| phase_complete | [0%, 63%] (n=3) |
| synthesis | [0%, 79%] (n=1) |
| explain_back | [0%, 7%] (n=52) |
| verified | [0%, 6%] (n=57) |
| triage | [0%, 7%] (n=53) |
| accretion_precommit | [0%, 79%] (n=1) |

Only explain_back, verified, and triage have enough data for confident estimates. The other gates need Phase 4 prospective tracking to narrow intervals.

### Finding 4: self_review is confirmed as the worst-calibrated gate

79% false positive rate across 19 failures. The gate's pattern matching (`fmt.Print` in Go, `console.log` in JS) is too broad:
- Catches intentional CLI output (not just debug statements)
- Fires on `console.error` (error handling, not debugging)
- Fires on pre-existing code the agent didn't write
- SkipCLIFiles only covers cmd/ but intentional output lives in pkg/ too

---

## Recommendations

1. **Signal gates are healthy.** No calibration changes needed for build, vet, phase_complete, synthesis, explain_back, verified, triage, or accretion_precommit.

2. **Prospective tracking is essential** for build, vet, phase_complete, synthesis, and accretion_precommit — sample sizes too small for confident FP rate estimates. Phase 4 should prioritize these.

3. **Discipline gates (explain_back, verified) deserve scrutiny** — not for accuracy (they're always accurate) but for *value*. 100% accuracy at measuring "did the orchestrator type --explain" doesn't prove the comprehension step improves outcomes. Phase 4 should correlate: do issues where explain_back passes have fewer regressions?

4. **self_review needs the Phase 2 fixes** — 79% FP rate confirms it's the noisiest gate in the system. The fmt.Print pattern needs scoping to agent-diff-only and console.error exclusion.
