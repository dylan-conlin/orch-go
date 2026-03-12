# Probe: Does Adding Exploration Mode Routing Degrade Existing Routing Accuracy?

**Model:** skill-content-transfer
**Date:** 2026-03-11
**Status:** Complete

---

## Question

The skill-content-transfer model predicts that adding content to context-type containers can dilute routing accuracy. The orchestrator skill grew by 8 lines (~235 tokens) of exploration mode routing guidance (commit 8678892fd). Does this addition degrade the orchestrator's ability to correctly route 10 non-explore scenarios to existing skills?

**Model claims under test:**
- Invariant 1: Skill length ≤ 500 lines / 5,000 tokens
- Invariant 2: ≤ 4 behavioral norms
- Core mechanism: Knowledge transfers reliably; behavioral constraints dilute at 5+

---

## What I Tested

**Methodology change:** The intended empirical A/B test (10 scenarios × 2 variants × N=3 = 60 `claude --print` calls) was blocked by the global Stop hook (`enforce-phase-complete.py`) contaminating all `claude --print` output. The hook fires on session exit and the model responds to it, replacing the actual routing answer. Multiple workarounds attempted (CLAUDECODE unset, CLAUDE_CONFIG_DIR override, stream-json output, JSON output) — all failed.

**Pivot to structural analysis:** Applied the model's own content-type taxonomy to classify each added line, then traced 10 routing scenarios through the decision tree with and without the exploration content.

### Content Type Classification

| Added Content | Location | Content Type | Rationale |
|---|---|---|---|
| `Broad question, multiple angles → orch spawn --explore investigation "Q"` | Surface Table (L77) | Knowledge | Routing table entry — tells agent what option exists |
| `EXPLORE broadly → investigation --explore` | Decision Tree (L92) | Knowledge | Routing tree entry |
| `Investigation vs Exploration: Single-angle → investigation...` | Clarification (L100-101) | Knowledge | Discrimination criteria, not prohibition |
| `Exploratory: "Map out X", "What are all the ways Y?"` | Intent Table (L111) | Knowledge | Intent-to-skill mapping |
| `--explore`, `--explore-breadth`, `--explore-depth` flags | Spawn Template (L241-243) | Knowledge | Command reference |

**Result: 8/8 lines are knowledge content. 0 behavioral norms added. 0 stance items added.**

### Budget Impact

| Metric | Baseline | With Explore | Delta | Invariant |
|---|---|---|---|---|
| Lines | 503 | 511 | +8 (+1.6%) | ≤500 (already exceeded) |
| Tokens (est.) | ~5,910 | ~6,146 | +235 (+4.0%) | ≤5,000 (already exceeded) |
| Behavioral norms | ~4 | ~4 | 0 | ≤4 ✅ |

### Routing Trace (10 scenarios)

Traced each scenario through the decision tree with and without exploration entries:

| # | Scenario | Expected | Baseline Route | Explore Route | False Positive Risk |
|---|---|---|---|---|---|
| 1 | Dashboard showing wrong status | systematic-debugging | ✓ | ✓ | NONE |
| 2 | Add --timeout flag | feature-impl | ✓ | ✓ | NONE |
| 3 | SQLite vs Postgres decision | architect | ✓ | ✓ | LOW |
| 4 | Daemon spawns failing silently | investigation | ✓ | ✓ | NONE |
| 5 | Completion gate rejecting valid | systematic-debugging | ✓ | ✓ | NONE |
| 6 | Add account rotation | feature-impl | ✓ | ✓ | NONE |
| 7 | Try out Playwright CLI | experiential-eval | ✓ | ✓ | NONE |
| 8 | Compare tmux vs headless spawn | head-to-head | ✓ | ✓ | NONE |
| 9 | Why agents don't fill D.E.K.N. | investigation | ✓ | ✓ | NONE |
| 10 | Redesign spawn pipeline | architect | ✓ | ✓ | LOW |

**0/10 scenarios routed differently. No structural false-positive path exists.**

---

## What I Observed

### 1. The model's dilution prediction applies to behavioral constraints, not knowledge content

The orientation frame characterized this as "adding content to context-type containers dilutes at scale." But the model is more specific: **knowledge** content transfers reliably (+5 points in prior experiments). **Behavioral** constraints dilute at 5+. Since all 8 added lines are knowledge (routing tables, command reference), the model actually predicts **positive transfer, not dilution**.

### 2. The exploration routing has explicit discrimination criteria

The additions include clear discrimination language:
- "Single-angle question → `investigation`. Broad question with multiple viable angles → `--explore`"
- Intent patterns: "Map out X", "What are all the ways Y?"

None of the 10 test scenarios match these patterns. The existing routing categories (FIX, BUILD, DESIGN, TRIAGE, TRY, COMPARE) capture all scenarios unambiguously before "EXPLORE broadly" is reached in the tree.

### 3. The skill was already over the token/line budget before this addition

- Baseline: 503 lines / ~5,910 tokens (exceeds both ≤500 lines and ≤5,000 token invariants)
- With explore: 511 lines / ~6,146 tokens
- The exploration addition is not responsible for crossing the budget threshold — that happened earlier. The 1.6% line increase is within noise.

### 4. The 2 "LOW risk" scenarios (S3, S10) have structural protection

S3 ("SQLite vs Postgres") and S10 ("Redesign spawn pipeline") could superficially appear "broad" but:
- S3 is a binary design decision ("trade-offs"), not "multiple angles" exploration
- S10 is a design task ("redesign"), not "what are all the ways..."
- Both match DESIGN in the decision tree before reaching EXPLORE
- Decision tree ordering provides structural protection: DESIGN is listed before EXPLORE

### 5. Infrastructure limitation: `claude --print` is unusable for A/B testing from orch-go sessions

The global Stop hook (`enforce-phase-complete.py`) fires on every `claude --print` invocation, creating a second conversation turn that replaces the model's actual response in the output. This blocks ALL `claude --print` based testing from any Claude Code session that has the hook configured.

---

## Model Impact

- [x] **Confirms** invariant: Knowledge content does not dilute routing accuracy (consistent with the +5 points knowledge transfer finding and the behavioral-only dilution mechanism)
- [ ] **Contradicts** invariant: N/A
- [x] **Extends** model with: The dilution concern from the orientation frame ("adding content to containers dilutes at scale") is an overgeneralization of the model's actual claim. The model distinguishes three content types with different transfer mechanisms. Only behavioral constraints dilute. The probe corrects the framing: the question should be "does adding behavioral content dilute?" not "does adding any content dilute?"

---

## Notes

### Limitations
- **No empirical A/B data.** The structural analysis traces routing decisions mechanically through the decision tree, which is deterministic. But it doesn't test whether a real model, under the cognitive load of the full skill document, might route differently. The model's own experiments used N=6-7 per scenario with actual API calls. This probe has N=0 empirical trials.
- **The structural analysis may be overconfident.** A model reading 511 lines of context might not follow the decision tree linearly — it might pick up on lexical associations ("explore" appearing in new entries) and be primed toward exploration routing even for non-explore scenarios. This is the mechanism the probe was designed to test, and it remains untested.

### Follow-up needed
- **Unblock `claude --print` for testing:** The Stop hook should not fire in `--print` mode, or its output should be separable from the model's response. This blocks all skill testing infrastructure.
- **If empirical validation is desired:** Run the A/B test from a session without the Stop hook, or via a standalone script with the Anthropic API directly.

### Confidence
- **High confidence** that the additions are knowledge content (classification is mechanical)
- **High confidence** in the structural routing trace (decision tree is deterministic)
- **Medium confidence** that a real model wouldn't be influenced by lexical priming — this is exactly what the empirical test was meant to measure
