<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [To be filled after experiment completes]

**Evidence:** [To be filled after experiment completes]

**Knowledge:** [To be filled after experiment completes]

**Next:** [To be filled after experiment completes]

**Authority:** implementation - Experiment design and execution within investigation scope

---

# Investigation: Coordination Failure Controlled Demo — Haiku vs Opus

**Question:** Do higher-capability models (Opus) produce fewer coordination failures than lower-capability models (Haiku) when given the same task on a shared codebase?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** investigation agent (orch-go-qrfhe)
**Phase:** Investigating
**Next Step:** Collect and score experiment results
**Status:** In Progress

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/daemon-autonomous-operation/model.md (GPT stall rates) | extends | yes | N/A — that data covers GPT, this covers Haiku |
| .kb/investigations/2026-03-05-inv-post-mortem-price-watch-orchestration.md | extends | pending | N/A |

**Relationship types:** extends — this investigation adds Haiku vs Opus comparison data to existing model performance knowledge.

---

## Experiment Design

### Research Question
Do higher-capability models (Opus) produce fewer coordination failures than lower-capability models (Haiku) when given identical coding tasks on the same codebase?

### Methodology

1. **Baseline:** Both agents start from the same git commit (bde5d5898)
2. **Isolation:** Each agent works in an independent git worktree
3. **Task:** Identical — add `FormatBytes(bytes int64) string` to `pkg/display/display.go` with tests
4. **Models:** claude-haiku-4-5-20251001 vs claude-opus-4-5-20251101
5. **Measurement:** 6-dimension scoring rubric (see below)
6. **Merge test:** After both complete, attempt to merge changes to detect coordination failures

### Scoring Rubric (6 dimensions)

| Dimension | Description | Score |
|-----------|-------------|-------|
| F0: Completion | Did the agent produce any changes? | 0-1 |
| F1: Compilation | Does the code compile? | 0-1 |
| F2: New Tests | Do the new tests pass? | 0-1 |
| F3: No Regression | Do existing tests still pass? | 0-1 |
| F4: File Discipline | Only expected files modified? | 0-1 |
| F5: Spec Match | Function matches specified signature? | 0-1 |

Total: 0-6 per trial

### Coordination Failure Categories

When both agents' changes are merged:
- **Clean merge:** No conflicts, merged code passes all tests
- **Git conflict:** Merge fails due to textual conflicts
- **Semantic conflict:** Merge succeeds but tests fail (incompatible changes)

### Task Specification

Add `FormatBytes(bytes int64) string` to `pkg/display/display.go`:
- Binary units: B, KiB, MiB, GiB, TiB
- 1 decimal place for non-byte units
- Handle negative values (prefix "-")
- Handle zero ("0 B")
- Comprehensive tests in `display_test.go`

### Controls

- Same baseline commit
- Same task prompt (1,192 bytes)
- Same permissions (`--dangerously-skip-permissions`)
- Independent worktrees (no shared state)
- No time limit (measure natural completion time)

---

## Findings

### Finding 1: Experiment Infrastructure Created

**Evidence:** Created experiment automation in `experiments/coordination-demo/`:
- `task-prompt.md` — Standardized task specification
- `run.sh` — Automated experiment runner (creates worktrees, runs agents, collects results)
- `score.sh` — 6-dimension scoring script
- `merge-check.sh` — Merge conflict detection

**Source:** `experiments/coordination-demo/*.sh`

**Significance:** Reproducible experiment infrastructure. Can be re-run with `./run.sh --trials N` for statistical significance.

---

### Finding 2: Pilot Run Executed

**Evidence:** [To be filled after agents complete]

**Source:** `experiments/coordination-demo/results/pilot-20260309-134852/`

**Significance:** First quantitative Haiku vs Opus comparison on this codebase.

---

### Finding 3: [To be filled]

**Evidence:** [To be filled]

**Source:** [To be filled]

**Significance:** [To be filled]

---

## Synthesis

**Key Insights:**

1. [To be filled after results]

2. [To be filled after results]

3. [To be filled after results]

**Answer to Investigation Question:**

[To be filled after results]

---

## Structured Uncertainty

**What's tested:**

- ✅ [To be filled after experiment]

**What's untested:**

- ⚠️ Single trial — not statistically significant (need 5+ trials per model)
- ⚠️ Only one task type tested — results may not generalize to other task complexities
- ⚠️ No protocol adherence measured (beads comments, phase reports) — headless mode bypasses protocol

**What would change this:**

- Multiple trials would establish confidence intervals
- Multiple task types (simple → complex) would test generalization
- Spawn via `orch spawn` instead of `claude -p` would test protocol adherence

---

## Implementation Recommendations

[To be filled after results]

---

## References

**Files Examined:**
- `pkg/display/display.go` — Target file for experiment task (95 lines, 6 functions)
- `pkg/display/display_test.go` — Target test file (135 lines, 6 test functions)
- `~/.orch/events.jsonl` — Historical agent events (1,145 events, no model tracking)
- `.kb/models/daemon-autonomous-operation/model.md` — Prior model performance data

**Commands Run:**
```bash
# Create worktrees
git worktree add -b coord-demo-haiku /tmp/coord-demo-haiku bde5d58982
git worktree add -b coord-demo-opus /tmp/coord-demo-opus bde5d58982

# Run agents (parallel)
claude --model claude-haiku-4-5-20251001 --dangerously-skip-permissions -p "$(cat task-prompt.md)"
claude --model claude-opus-4-5-20251101 --dangerously-skip-permissions -p "$(cat task-prompt.md)"
```

**Related Artifacts:**
- **Experiment scripts:** `experiments/coordination-demo/`
- **Results:** `experiments/coordination-demo/results/pilot-20260309-134852/`

---

## Investigation History

**[2026-03-09 13:41]:** Investigation started
- Initial question: Do Haiku and Opus have different coordination failure rates?
- Context: Harness publication demo — need quantitative model comparison data

**[2026-03-09 13:48]:** Experiment design complete, pilot run started
- Created 6-dimension scoring rubric
- Launched Haiku and Opus agents in parallel worktrees
- Task: FormatBytes function implementation

**[2026-03-09 HH:MM]:** [To be filled - results collected]
