# Model: Coordination

**Created:** 2026-03-09
**Status:** Active
**Source:** Synthesized from 3 investigation(s)

## What This Is

[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]

---

## Core Claims (Testable)

### Claim 1: [Concise claim statement]

[Explanation of the claim. What would you observe if it's true? What would falsify it?]

**Test:** [How to test this claim]

**Status:** Hypothesis

### Claim 2: [Concise claim statement]

[Explanation of the claim.]

**Test:** [How to test this claim]

**Status:** Hypothesis

---

## Implications

[What follows from these claims? How should this model change behavior, design, or decision-making?]

---

## Boundaries

**What this model covers:**
- [Scope item 1]

**What this model does NOT cover:**
- [Exclusion 1]

---

## Evidence

| Date | Source | Finding |
|------|--------|---------|
| 2026-03-09 | Model creation | Initial synthesis from source investigations |

---

## Open Questions

- [Question that further investigation could answer]
- [Question about model boundaries or edge cases]

## Source Investigations

### 2026-03-09-inv-coordination-demo-complex-ambiguous.md

**Delta:** Complex/ambiguous multi-file task reveals capability differences in ambiguity resolution (Opus anticipates Unicode edge cases, produces stronger alignment tests) while coordination failure remains 100% structural — now with two conflict types: content conflicts AND add/add conflicts for new files.
**Evidence:** N=1 experiment: identical 4-file task (VisualWidth + RenderTable) given to Haiku (65s, 10/10) and Opus (88s, 10/10); merge produces CONFLICT in all 4 files; Opus uses rune counting for Unicode while Haiku uses byte length (subtly wrong); Opus tests verify actual column alignment positions while Haiku only checks separator existence.
**Knowledge:** Binary compliance scoring cannot distinguish model capability — both score 10/10. The differentiator is "anticipating edge cases the spec didn't mention" — a capability dimension orthogonal to constraint following. Coordination failure extends to new file creation (add/add conflicts), not just same-position insertion.
**Next:** Close investigation. Findings extend the pilot (Trial 1) with evidence that (a) model capability differences ARE real on complex tasks but are invisible to binary scoring, and (b) coordination failures are structural across all conflict types.

---

### 2026-03-09-inv-coordination-demo-n10-formatbytes.md

**Delta:** At N=10, coordination failure rate is 100% for both Haiku and Opus, confirming the pilot finding that merge conflicts are structural (same insertion points in git), not capability-dependent.
**Evidence:** 20 agent runs (10 per model): both scored 6/6 individually in all trials; all 10 trial pairs produced merge conflicts; Fisher's exact test p=1.0; duration difference not significant (haiku 39.1s vs opus 44.0s, t=1.103, p>0.05).
**Knowledge:** For well-defined, unambiguous tasks, model capability does not affect coordination failure rate. The failure is entirely structural — both models follow the instruction "place after FormatDurationShort" and git cannot auto-merge two different insertions at the same position. Upgrading models will not reduce coordination failures.
**Next:** Close investigation. Data supports harness publication claim that coordination failure is a protocol problem requiring architectural solutions (file-level work assignment, sequential execution), not model upgrades.

---

### 2026-03-09-inv-coordination-failure-controlled-demo-same.md

**Delta:** Coordination failures when two agents implement the same feature are dominated by structural factors (same insertion points), not model capability — both Haiku and Opus scored 6/6 individually but produced 100% merge conflict rate.
**Evidence:** Pilot experiment: identical task (FormatBytes) given to Haiku (49s, 34 test cases) and Opus (63s, 24 test cases) in isolated worktrees; both achieved perfect individual scores; merge produced CONFLICT in both modified files (display.go, display_test.go); both independently generated identical commit messages.
**Knowledge:** Coordination failure is a protocol problem, not a capability problem — even the most capable model cannot avoid conflicts without coordination infrastructure (file-level locking, insertion-point reservation, or sequential execution).
**Next:** Close investigation. Recommend architect review if coordination protocol infrastructure is desired (e.g., file-level work assignment, insertion-point reservation, or pre-merge CI).
