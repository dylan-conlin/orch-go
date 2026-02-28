# Probe: Stalled Agent Failure Pattern Audit

**Model:** agent-lifecycle-state-model
**Date:** 2026-02-28
**Status:** Complete

---

## Question

Does the agent-lifecycle-state-model's "Failure Mode 3: Agent Went Idle But Not Complete" accurately capture all stall failure modes? Are there additional failure categories not covered by the model's 4 failure modes?

---

## What I Tested

Scanned all 1655 archived workspaces and 131 active workspaces for agents that lack SYNTHESIS.md, cross-referenced with beads phase comments and AGENT_MANIFEST.json metadata. Categorized every no-SYNTHESIS agent by: last reported phase, skill type, model, beads issue status.

```bash
# Total counts
find .orch/workspace/archived/og-*/ -maxdepth 1 -name "SYNTHESIS.md" | wc -l  # 717 completed
ls -d .orch/workspace/archived/og-*/ | wc -l                                   # 1655 total

# Phase categorization for 441 manifested agents without SYNTHESIS.md
for dir in .orch/workspace/archived/og-*/; do
  [[ ! -f "$dir/SYNTHESIS.md" && -f "$dir/.beads_id" && -f "$dir/AGENT_MANIFEST.json" ]] || continue
  beads_id=$(cat "$dir/.beads_id")
  last_phase=$(bd show "$beads_id" | grep "Phase:" | tail -1)
  # ... categorized into: Phase:Complete/no-phase/stalled-in-phase
done

# Model stall rates (archived agents with manifests)
# Opus: 100 stall / 224 total = 44.6%
# Sonnet: 183 / 268 = 68.3% (inflated by pre-protocol era)
# GPT-5.2-codex: 83 / 123 = 67.5%
# GPT-4o: 14 / 16 = 87.5%
```

---

## What I Observed

### Quantitative Summary

| Category | Count | Description |
|----------|-------|-------------|
| Pre-protocol era (no manifest) | 497 | Before manifest/phase tracking system existed |
| Phase: Complete, no SYNTHESIS.md | 194 | Completed but didn't write SYNTHESIS |
| No phase reported at all | 228 | Spawned but never reported any phase |
| Stalled in non-terminal phase | 19 | True stalls — stuck in Planning/Implementing/QUESTION/BLOCKED/Exploration |

### True Phase Stalls (19 agents) — Breakdown

| Stall Phase | Count | Models | Pattern |
|-------------|-------|--------|---------|
| Implementing | 5 | GPT-4o (1), GPT-5.2-codex (3), Codex (1) | Agent entered implementing, never completed |
| QUESTION | 5 | GPT-5.2-codex (5), all same issue (orch-go-fq5) | No answer delivery mechanism |
| Planning | 4 | GPT-5.2-codex (3), Codex (1) | Agent planned but never started |
| BLOCKED | 1 | GPT-5.2-codex | Concurrency limit blocked spawn |
| Exploration | 1 | Opus (orch-go-nn43) | Prior art confusion |

### Key Patterns Observed

1. **Duplicate Spawn Storms**: 5 agents for same issue (orch-go-dr0u) within 2 minutes. 10 retries on another slug. Daemon dedup cache is in-memory, doesn't survive restarts.

2. **Non-Anthropic Model Protocol Failures**: 15 of 19 true stalls are GPT-4o or GPT-5.2-codex. These models don't reliably follow phase reporting, SYNTHESIS creation, or completion protocol.

3. **QUESTION Deadlock**: 5 agents stalled on Phase: QUESTION with no mechanism to deliver answers to headless agents.

4. **SYNTHESIS Compliance Gap**: 194 agents (44% of manifested stalls) actually completed successfully but didn't create SYNTHESIS.md. These are protocol compliance failures, not stalls.

5. **Silent Failures**: 228 agents with manifests never reported any phase — pre-phase-reporting skill versions or agents that crashed on startup.

---

## Model Impact

- [x] **Extends** model with: 5 failure modes not covered by existing Failure Mode 3

The model's Failure Mode 3 ("Agent Went Idle But Not Complete") describes a single pattern: session exhausts context, stops responding, no Phase: Complete written. This is accurate but incomplete.

**New failure modes discovered:**

1. **QUESTION Deadlock** — Agent explicitly signals it needs input via Phase: QUESTION, but the orchestration system has no mechanism to deliver answers to headless agents. This is a structural gap, not a session exhaustion issue.

2. **Prior Art Confusion** — Agent discovers prior work that overlaps its task, gets confused about remaining scope, and stalls in Exploration. Example: orch-go-nn43 found 2 prior agents already implemented query enrichment, couldn't determine what was left to do.

3. **Concurrency Ceiling Stall** — Agent needs to spawn sub-tasks but hits concurrency limits. Reports BLOCKED, waits indefinitely for slot availability that never comes.

4. **Duplicate Spawn Storm** — Same issue spawned 5-10x due to in-memory dedup cache not surviving daemon restarts. Creates 4-9 wasted agent-sessions per occurrence. Not an agent failure per se, but a systemic cause of workspace pollution.

5. **SYNTHESIS Compliance Gap** — Agent completes work (Phase: Complete) but doesn't write SYNTHESIS.md. This makes the agent appear stalled when it's actually done. 194 instances found.

6. **Model Protocol Incompatibility** — Non-Anthropic models (GPT-4o, GPT-5.2-codex) fail to follow the worker-base skill protocol reliably. 87.5% stall rate for GPT-4o vs 44.6% for Opus. This isn't about capability — it's about instruction-following fidelity for multi-step protocols.

**Model invariant confirmed:** "Phase: Complete is agent's declaration — Only agent can reach this, not orchestrator" — The 194 agents that did report Phase: Complete but have no SYNTHESIS.md confirm this invariant is working. The problem is downstream: SYNTHESIS enforcement is missing.

---

## Notes

- The 56.6% "stall rate" is misleading. The true stall rate for agents with manifests is: 19 true stalls / 441 manifested no-SYNTHESIS agents = 4.3%. Most of the "stalls" are protocol compliance gaps or pre-protocol era agents.
- Model stall rates are also misleading: Sonnet's 68.3% is inflated by 133 agents from Feb 14-17 before phase reporting was in the skill. The true Sonnet stall rate for agents that could have reported phases is much lower.
- The orch-go-nn43 case represents a systemic problem: the orchestrator spawns work without checking if prior agents already completed overlapping work. This wastes expensive Opus tokens on re-exploration.
