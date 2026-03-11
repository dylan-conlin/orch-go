---
title: "Gate Census and Signal/Noise Classification"
date: 2026-03-11
type: investigation
status: Complete
model: harness-engineering
plan_ref: .kb/plans/2026-03-11-gate-signal-vs-noise.md
plan_phase: "Phase 1"
---

# Gate Census and Signal/Noise Classification

**Model:** harness-engineering
**Plan:** Gate Signal vs Noise — Phase 1

## Summary

Complete inventory of every gate in the system, classified as signal (catching real problems), noise (forcing routine bypasses), or unknown (insufficient data).

**Data source:** `orch stats` output from 216 spawns, 211 completions, 5,336 events (last 7 days as of 2026-03-11).

---

## Gate Inventory

### A. Spawn Gates (pre-spawn checks)

| # | Gate | File | Type | Bypass/Fail Count | Bypass Rate | Classification | Evidence |
|---|------|------|------|-------------------|-------------|----------------|----------|
| S1 | **triage** | `pkg/spawn/gates/triage.go` | BLOCKING | 53 bypassed / 216 spawns | 24.5% | **SIGNAL** | Enforces daemon-driven workflow. Bypass reasons are substantive (urgent items, release gates, Dylan-requested). 79.6% of spawns are daemon-driven. |
| S2 | **hotspot** | `pkg/spawn/gates/hotspot.go` | BLOCKING (CRITICAL files) | 72 bypassed / 216 spawns | 33.3% | **NOISE** | 72 bypasses with **zero reasons recorded**. Unmeasurable gate — no data on whether bypasses were justified. |
| S3 | **verification** | `pkg/spawn/gates/verification.go` | BLOCKING | 24 bypassed / 216 spawns | 11.1% | **NOISE** | All 24 bypasses have identical reason: "testing independent parallel work". Single repeated override — gate is a speed bump, not a filter. |
| S4 | **concurrency** | `pkg/spawn/gates/concurrency.go` | BLOCKING | 0 in stats | 0% | **UNKNOWN** | No bypass/block events in current window. Default 5 agent limit. Fails open on infra errors. |
| S5 | **ratelimit** | `pkg/spawn/gates/ratelimit.go` | BLOCKING (95%+) / WARNING (80%+) | 0 in stats | 0% | **UNKNOWN** | No rate limit events in current window. May not be triggering with current account rotation. |
| S6 | **question** | `pkg/spawn/gates/question.go` | ADVISORY (never blocks) | N/A | N/A | **SIGNAL** | Warning-only gate. Alerts to open questions in dependency chain. No bypass needed since it doesn't block. |
| S7 | **agreements** | `pkg/spawn/gates/agreements.go` | ADVISORY (never blocks) | N/A | N/A | **UNKNOWN** | Warning-only. No data on how often it warns or whether warnings change behavior. |
| S8 | **governance** | `pkg/orch/governance.go` | ADVISORY (spawn-time warning) | N/A | N/A | **SIGNAL** | Warns about governance-protected files before worker starts. Prevents wasted sessions (workers would be blocked by hooks at runtime). |

#### Daemon-Specific Spawn Gates (pkg/daemon/spawn_gate.go pipeline)

| # | Gate | Type | Evidence |
|---|------|------|----------|
| D1 | **spawn-tracker** (L1) | BLOCKING (fail-open) | In-memory TTL cache. Prevents immediate re-spawn of same issue. Infrastructure gate, no stats. |
| D2 | **session-dedup** (L2) | BLOCKING (fail-open) | Checks for existing OpenCode/tmux session. Infrastructure gate. |
| D3 | **title-dedup-memory** (L3) | BLOCKING (fail-open) | In-memory title dedup. Prevents duplicate spawns for same-titled issues. |
| D4 | **title-dedup-beads** (L4) | BLOCKING (fail-open) | Checks beads DB for in_progress duplicates. |
| D5 | **fresh-status** (L5) | BLOCKING (fail-open) | TOCTOU guard — re-fetches issue status to catch races. |
| D6 | **spawn-count advisory** | ADVISORY | Warns on thrashing (spawn count >= 3). |
| D7 | **architect-escalation** | ROUTING (Layer 2) | Routes impl skills to architect for hotspot areas. No stats on frequency. |

### B. Completion Gates (orch complete verification)

| # | Gate | Const | Level | Failed | Bypassed | AutoSkip | Fail Rate | Classification | Evidence |
|---|------|-------|-------|--------|----------|----------|-----------|----------------|----------|
| C1 | **phase_complete** | `GatePhaseComplete` | V0 | 3 | 7 | 0 | 1.4% | **SIGNAL** | Low fail rate. 7 bypasses, reasons include stop hook bugs preventing Phase: Complete reporting. Legitimate skip for broken infrastructure, not false positives. |
| C2 | **synthesis** | `GateSynthesis` | V2 | 1 | 6 | 0 | 0.5% | **SIGNAL** | Very low fail rate. Bypasses for light-tier and no-track investigations (correct behavior — these shouldn't need synthesis). |
| C3 | **handoff_content** | `GateHandoffContent` | V1 | 0 | 0 | 0 | 0% | **UNKNOWN** | No failures or bypasses in current data. Orchestrator-session gate. |
| C4 | **constraint** | `GateConstraint` | V1 | 0 | 0 | 0 | 0% | **UNKNOWN** | No failures or bypasses. Skill constraint verification. |
| C5 | **phase_gate** | `GatePhaseGate` | V1 | 0 | 0 | 0 | 0% | **UNKNOWN** | No failures or bypasses. Required phase reporting. |
| C6 | **skill_output** | `GateSkillOutput` | V1 | 0 | 0 | 0 | 0% | **UNKNOWN** | No failures or bypasses. Skill output file verification. |
| C7 | **architectural_choices** | `GateArchitecturalChoices` | V1 | 13 | 11 | 0 | 6.2% | **NOISE** | 11 bypasses with varied reasons: "investigation was exploratory", "no code changes", "design doc IS the architectural choices", "straightforward fallback chain". Gate triggers for non-implementation work where architectural choices aren't applicable. |
| C8 | **self_review** | `GateSelfReview` | V1 | 19 | 33 | 0 | 9.0% | **NOISE** | 33 bypasses — the highest bypass count of any gate. Dominant false positive: `fmt.Print` in `stats_output.go` and CLI output files flagged as debug statements. Other: "pre-existing debug statements not from this agent", "cross-repo changes". SkipCLIFiles already added but still triggering. |
| C9 | **probe_model_merge** | `GateProbeModelMerge` | V1 | 0 | 0 | 0 | 0% | **UNKNOWN** | No failures or bypasses in current window. |
| C10 | **test_evidence** | `GateTestEvidence` | V2 | 0 | 0 | 0 | 0% | **UNKNOWN** | No direct bypass data in stats. May be auto-skipped for non-implementation skills. |
| C11 | **git_diff** | `GateGitDiff` | V2 | 11 | 9 | 0 | 5.2% | **NOISE** | 9 bypasses with patterns: "cross-repo changes not visible in git diff" (3x), "partial fix outcome" (2x), "agent blocked by governance" (1x). Gate can't handle cross-repo work — structural false positive. |
| C12 | **build** | `GateBuild` | V2 | 3 | 1 | 0 | 1.4% | **SIGNAL** | Low fail rate, low bypass. 1 bypass for governance-blocked agent. Build failures are real problems. |
| C13 | **vet** | `GateVet` | V2 | 3 | 0 | 0 | 1.4% | **SIGNAL** | 3 failures, 0 bypasses. `go vet` catches real issues. No false positive pattern. |
| C14 | **accretion** | `GateAccretion` | V2 | 0 | 0 | 0 | 0% | **UNKNOWN** | No data in current window. Completion-time accretion check. |
| C15 | **visual_verification** | `GateVisualVerify` | V3 | 0 | 0 | 0 | 0% | **UNKNOWN** | No direct data. May be handled via --approve flag. |
| C16 | **explain_back** | `GateExplainBack` | V3 | 52 | 1 | 0 | 24.6% | **SIGNAL** | 52 failures = orchestrator didn't provide --explain. This is a human-input gate (orchestrator must articulate understanding). High fail rate is expected — it's measuring orchestrator behavior, not agent quality. 1 bypass for partial fix. |
| C17 | **verified** (gate2 behavioral) | Checkpoint | V3+ | 57 | 0 | 0 | 27.0% | **SIGNAL** | 57 failures = orchestrator didn't use --verified. Same pattern as explain_back — human verification gate. High fail rate measures orchestrator discipline, not gate accuracy. |
| C18 | **decision_patch_limit** | `GateDecisionPatchLimit` | V1 | 0 | 0 | 0 | 0% | **UNKNOWN** | No data. |

### C. Pre-commit Gates (git hooks)

| # | Gate | File | Type | Stats Data | Classification | Evidence |
|---|------|------|------|------------|----------------|----------|
| P1 | **accretion_precommit** | `pkg/verify/accretion_precommit.go` | BLOCKING (>1500 lines agent-caused) / WARNING (>800/600) | 1 block, 1 bypass in gate_decision events | **SIGNAL** | Blocks agent-caused file bloat. Pre-existing bloat downgraded to warning (good design). Override: FORCE_ACCRETION=1. |
| P2 | **knowledge** | `pkg/verify/knowledge_precommit.go` | BLOCKING | 0 in stats | **UNKNOWN** | Enforces model coupling for new investigations. Override: FORCE_ORPHAN=1. No data on frequency. |
| P3 | **model-stub** | `pkg/verify/model_stub_precommit.go` | BLOCKING | 0 in stats | **UNKNOWN** | Prevents empty model templates from being committed. Override: FORCE_MODEL_STUB=1. |
| P4 | **duplication** | precommit_cmd.go | ADVISORY | 0 in stats | **UNKNOWN** | Warns on near-clone Go functions (≥85% similarity). Never blocks. |

### D. Publication Gates (kb gate)

| # | Gate | File | Type | Classification | Evidence |
|---|------|------|------|----------------|----------|
| K1 | **publish contract** | `pkg/kbgate/publish.go` | BLOCKING | **UNKNOWN** | Requires challenge_refs, claim_refs, ledger_ref in frontmatter. Domain-specific knowledge pipeline gate. |
| K2 | **challenge artifacts** | `pkg/kbgate/publish.go` | BLOCKING | **UNKNOWN** | Verifies referenced challenge files exist. |
| K3 | **lineage (endogenous evidence)** | `pkg/kbgate/publish.go` | BLOCKING | **UNKNOWN** | Detects self-referential evidence chains. |
| K4 | **banned language** | `pkg/kbgate/publish.go` | BLOCKING | **UNKNOWN** | Blocks novelty-bearing phrases ("physics", "new framework", etc.). |
| K5 | **claim-upgrade signals** | `pkg/kbgate/publish.go` | BLOCKING (downgradable) | **UNKNOWN** | Detects claim upgrades. Can be acknowledged with --acknowledge-claims. |
| K6 | **claim ledger** | `pkg/kbgate/publish.go` | BLOCKING | **UNKNOWN** | Validates claim ledger structure and completeness. |

---

## Classification Summary

| Classification | Count | Gates |
|----------------|-------|-------|
| **SIGNAL** | 10 | triage, question, governance, phase_complete, synthesis, build, vet, explain_back, verified, accretion_precommit |
| **NOISE** | 4 | hotspot (no reasons), verification (identical bypasses), self_review (fmt.Print FP), architectural_choices (wrong scope), git_diff (cross-repo FP) |
| **UNKNOWN** | 22+ | concurrency, ratelimit, agreements, daemon pipeline (D1-D7), most V1 completion gates with 0 data, all publication gates |

*Note: git_diff is counted separately as noise — 5 total noise gates.*

---

## Noise Gate Analysis

### NOISE 1: hotspot (spawn gate) — 72 bypasses, 0 reasons
**False positive pattern:** Gate fires but no reason is recorded, making it impossible to distinguish signal from noise. The bypass itself is not the problem — the lack of measurement is.
**Recommended action:** Add mandatory reason recording for hotspot bypasses. Then reclassify after 2 weeks of data.

### NOISE 2: verification (spawn gate) — 24 bypasses, all identical
**False positive pattern:** All 24 bypasses say "testing independent parallel work". This is a legitimate workflow (parallel agents on independent tasks) that the gate doesn't account for. The gate assumes serial execution.
**Recommended action:** Add auto-bypass for issues with no dependency overlap, or downgrade to advisory when work is demonstrably independent.

### NOISE 3: self_review (completion gate) — 33 bypasses
**False positive pattern:** `fmt.Print` in pkg/ files that are intentional CLI output (stats_output.go, version command). SkipCLIFiles flag exists but only covers cmd/ paths. Also triggers on pre-existing debug statements not added by the agent.
**Recommended action:** Extend SkipCLIFiles to cover pkg/*/output files or add a whitelist pattern for intentional fmt.Print. The baseline-scoped diff check should already handle pre-existing code, so investigate why it's still triggering.

### NOISE 4: architectural_choices (completion gate) — 11 bypasses
**False positive pattern:** Gate requires "Architectural Choices" section in SYNTHESIS.md for skills where no architectural choices were made (investigations, straightforward changes, design docs that ARE the choices).
**Recommended action:** Scope gate to only implementation skills that actually make architectural decisions. Currently fires for V1+ but should be V2+ or skill-filtered.

### NOISE 5: git_diff (completion gate) — 9 bypasses
**False positive pattern:** Cross-repo work (skillc, kb-cli changes) invisible to local git diff. Agent accurately reports files in SYNTHESIS but they're in a different repo.
**Recommended action:** Add cross-repo awareness to git_diff check, or auto-skip when SYNTHESIS mentions cross-repo work.

---

## Signal Gate Evidence

### explain_back (52 failures) and verified (57 failures) — both SIGNAL
These are **human-input gates**, not automated checks. High failure rate is expected and correct — they measure whether the orchestrator performed comprehension and behavioral verification. The "failures" are the orchestrator skipping verification steps, which is exactly what these gates are designed to catch.

### build (3 failures) and vet (3 failures) — both SIGNAL
Low failure rates, real problems caught. No false positive patterns in bypass data.

### triage (53 bypasses) — SIGNAL
Bypass reasons are substantive and varied (release gates, urgency, Dylan-requested). Gate is working as designed — creating accountability for manual spawns.

---

## Aggregate Metrics

- **Total spawn gate bypass rate:** 69.0% (149/216 spawns)
  - Hotspot alone contributes 72/149 = 48.3% of all bypasses
  - Verification contributes 24/149 = 16.1%
  - **If noise gates fixed:** Estimated bypass rate would drop to ~53/216 = 24.5% (triage-only bypasses)

- **Completion gate pass rate first try:** 11.8% (25/211)
  - explain_back (52) + verified (57) = 109 failures from human-input gates
  - **Excluding human-input gates:** Estimated first-try pass rate ~60-70%

---

## Recommendations for Phase 2

1. **Hotspot gate:** Add reason recording (highest impact — 72 unmeasurable bypasses)
2. **self_review:** Fix fmt.Print false positive (33 bypasses, well-understood pattern)
3. **verification gate:** Evaluate whether to add independence detection or downgrade
4. **architectural_choices:** Tighten skill scoping (only implementation skills with code changes)
5. **git_diff:** Add cross-repo detection
6. **Unknown gates:** Most have 0 data because they're new or infrequently triggered. Let Phase 4 prospective tracking resolve these.
