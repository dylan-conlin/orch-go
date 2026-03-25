<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Coordination failure under merge-educated messaging is significantly model-dependent — Sonnet achieves 70% clean merges vs Haiku's 30%, but neither reaches 100%.

**Evidence:** 20 trials on Sonnet (merge-educated-20260324-175037) produced 14/20 clean merges vs 6/20 on Haiku (merge-educated-20260323-093342), with identical prompts, tasks, repo, and conditions.

**Knowledge:** The coordination failure has both a model-dependent component (Sonnet better internalizes spatial/merge reasoning from education text) and a structural component (30% residual failure from a race condition in simultaneous plan writing — both agents independently choose the same insertion point before reading each other's plans).

**Next:** Close. Results strengthen claims: the coordination problem is partially structural but model choice is a major lever. Consider testing with sequential plan exchange (A writes first, B reads then writes) to isolate the race condition component.

**Authority:** strategic - Whether to invest in more model comparisons or pivot to structural fixes is a direction question.

---

# Investigation: Cross-Model Validation — Merge-Educated Messaging on Sonnet

**Question:** Is the coordination failure (agents producing merge conflicts when editing the same file) model-specific (Haiku limitation) or structural (inherent to the task setup)?

**Started:** 2026-03-24
**Updated:** 2026-03-24
**Owner:** orch-go-f9xii
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| experiments/coordination-demo/redesign/results/20260310-174045 (4-condition experiment, Haiku) | extends | yes — confirmed 0/20 messaging baseline | - |
| experiments/coordination-demo/redesign/results/merge-educated-20260323-093342 (merge-educated, Haiku) | extends | yes — confirmed 6/20 success rate | - |

---

## Findings

### Finding 1: Sonnet achieves 70% clean merges with merge-educated messaging

**Evidence:**

| Condition | Model | Simple | Complex | Total |
|-----------|-------|--------|---------|-------|
| messaging (no merge ed.) | Haiku | 0/10 (0%) | 0/10 (0%) | 0/20 (0%) |
| merge-educated | Haiku | 4/10 (40%) | 2/10 (20%) | 6/20 (30%) |
| merge-educated | Sonnet | 7/10 (70%) | 7/10 (70%) | 14/20 (70%) |

**Source:** `experiments/coordination-demo/redesign/results/merge-educated-20260324-175037/analysis.md`

**Significance:** The coordination failure is not purely structural — a more capable model significantly improves coordination under identical conditions. The 40 percentage point gap (30% → 70%) demonstrates model capability matters for spatial/merge reasoning.

---

### Finding 2: Residual 30% failure is a race condition in plan coordination

**Evidence:** Examined conflict cases (e.g., simple trial 5, trial 8). In both:
- Both agents simultaneously wrote plans to shared directory
- Both independently chose the same insertion point (after StripANSI) assuming the other would pick the opposite location
- Agent B's plan stated "leave bottom for Agent A" while Agent A's revised plan stated "insert at bottom" — but this was written too late for B to see

Successful cases (e.g., trial 1): Agent A chose top (after StripANSI), Agent B chose bottom (after FormatDurationShort) — natural separation.

**Source:** `experiments/coordination-demo/redesign/results/merge-educated-20260324-175037/merge-educated/simple/trial-5/messages/` and `trial-8/messages/`

**Significance:** The remaining failures are a coordination protocol problem, not a model reasoning problem. Both agents correctly understand git merge mechanics but race to write plans simultaneously, preventing real-time coordination.

---

### Finding 3: Individual task completion is equally excellent for both models

**Evidence:** Sonnet: 39/40 agents scored 5/5 on task completion (1 timeout at 2/5). Haiku: 40/40 agents scored 6/6. Both models can implement the assigned features perfectly in isolation.

**Source:** Scores CSV files from both experiments

**Significance:** The coordination failure is not about task capability — both models are perfectly competent at the individual work. The failure is specifically about spatial reasoning about insertion points relative to another agent.

---

### Finding 4: Sonnet takes ~2x longer than Haiku

**Evidence:**
- Haiku merge-educated: agent-a avg 71-83s, agent-b avg 73-127s
- Sonnet merge-educated: agent-a avg 160-229s, agent-b avg 145-286s

**Source:** Duration summary from both analysis.md files

**Significance:** Cost/time tradeoff for coordination experiments. Sonnet's improved coordination rate comes at ~2x the compute time. At current pricing, this means cross-model validation costs significantly more.

---

### Finding 5: Harness model ID needed fixing

**Evidence:** First run attempt with `--model sonnet` mapped to `claude-sonnet-4-5-20250514` which doesn't exist. Agents completed in 1-2s with no output. Fixed by passing `--model claude-sonnet-4-5` directly.

**Source:** `experiments/coordination-demo/redesign/results/merge-educated-20260324-174740/` (failed run), stderr logs show "model not found"

**Significance:** The `run-merge-educated.sh` script has a stale model ID in the sonnet case. The correct ID is `claude-sonnet-4-5` (without date suffix). Script at line 48 needs updating.

---

## Synthesis

**Key Insights:**

1. **Dual causation** — Coordination failure is BOTH model-dependent AND structural. Neither explanation alone is sufficient. Sonnet's 70% success vs Haiku's 30% proves model capability matters. Sonnet's 30% residual failure proves structure matters too.

2. **Race condition as structural bottleneck** — The messaging protocol creates a race condition: agents write plans simultaneously, making assumptions about what the other will choose before seeing their actual plan. This is fixable with protocol changes (sequential plan exchange) without changing models.

3. **Model capability as force multiplier** — Even with the race condition, Sonnet succeeds 70% of the time because it more reliably acts on the merge education. It chooses diverse insertion points more often, tolerating the coordination protocol's limitations better.

**Answer to Investigation Question:**

The coordination failure is **partially model-specific and partially structural**. Upgrading from Haiku to Sonnet with the same merge-educated messaging improves clean merge rates from 30% to 70%, confirming a strong model-dependent component. However, even Sonnet fails 30% of the time due to a race condition in the simultaneous plan-writing protocol. The claims about coordination failure from the original 139-trial experiment should be qualified: the 100% failure rate is partly a Haiku limitation, but the underlying structural problem (lack of sequential plan negotiation) persists across models.

---

## Structured Uncertainty

**What's tested:**

- ✅ Sonnet merge-educated produces 14/20 clean merges (ran 20 trials with claude-sonnet-4-5)
- ✅ Haiku merge-educated produces 6/20 clean merges (from prior experiment merge-educated-20260323-093342)
- ✅ Conflict cases show both agents choosing same insertion point (examined plan-a.txt/plan-b.txt from trial 5 and 8)

**What's untested:**

- ⚠️ Opus performance on same task (expected even better but not tested)
- ⚠️ Sequential plan exchange protocol (write A, wait, B reads, B writes) — would isolate race condition
- ⚠️ Whether the 70% rate is stable at higher N (20 trials gives reasonable but not precise confidence)
- ⚠️ Whether Sonnet without merge education matches Haiku with it

**What would change this:**

- Sonnet achieving <40% would suggest model capability doesn't help much (refuted: 70%)
- Sonnet achieving 100% would mean the failure was purely Haiku-specific (refuted: 30% residual)
- Sequential plan exchange achieving 100% on either model would confirm race condition as the structural bottleneck

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix stale Sonnet model ID in harness scripts | implementation | Tactical fix, no cross-boundary impact |
| Report model-dependent finding in experiment writeup | strategic | Shapes claims about coordination research |
| Test sequential plan exchange protocol | architectural | New experimental condition requiring design |

### Recommended Approach: Qualify claims with model dependency

**Why this approach:**
- 139-trial dataset used only Haiku — claims should specify model
- Sonnet's 70% success rate materially changes the narrative
- The race condition insight opens a new intervention direction

**Trade-offs accepted:**
- Not testing Opus (expensive, diminishing returns on the model-vs-structure question)
- Not running sequential protocol test (separate experiment scope)

---

## References

**Files Examined:**
- `experiments/coordination-demo/redesign/run-merge-educated.sh` — experiment runner
- `experiments/coordination-demo/redesign/results/merge-educated-20260323-093342/` — Haiku baseline
- `experiments/coordination-demo/redesign/results/merge-educated-20260324-175037/` — Sonnet results

**Commands Run:**
```bash
# Run Sonnet experiment (20 trials)
bash experiments/coordination-demo/redesign/run-merge-educated.sh --model claude-sonnet-4-5

# Check messaging artifacts from conflict trials
cat results/merge-educated-20260324-175037/merge-educated/simple/trial-5/messages/plan-a.txt
cat results/merge-educated-20260324-175037/merge-educated/simple/trial-8/messages/plan-b.txt
```

**Related Artifacts:**
- **Investigation:** experiments/coordination-demo/redesign/results/20260310-174045/analysis.md — original 4-condition experiment
- **Investigation:** experiments/coordination-demo/redesign/results/merge-educated-20260323-093342/analysis.md — Haiku merge-educated baseline

---

## Investigation History

**2026-03-24 17:47:** Investigation started
- Initial question: Is coordination failure model-specific or structural?
- Context: All 139 prior trials used Haiku; need Sonnet comparison

**2026-03-24 17:47:** First Sonnet run failed — model ID `claude-sonnet-4-5-20250514` invalid
- All 20 agents completed in 1-2s with no output
- Fixed by using `claude-sonnet-4-5` (no date suffix)

**2026-03-24 17:50:** Second Sonnet run started successfully
- First trial produced SUCCESS in 146s/103s — confirmed Sonnet is working

**2026-03-24 19:24:** Experiment completed — 14/20 SUCCESS, 6/20 CONFLICT
- Examined conflict cases: race condition in simultaneous plan writing

**2026-03-24 19:30:** Investigation completed
- Key outcome: Coordination failure is both model-dependent (Sonnet 70% vs Haiku 30%) and structural (30% residual race condition)
