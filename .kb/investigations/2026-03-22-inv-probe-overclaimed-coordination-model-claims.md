## Summary (D.E.K.N.)

**Delta:** All 3 overclaimed coordination model claims should be SCOPED (narrowed) rather than confirmed or contradicted — the evidence supports the findings but the language exceeds the evidence scope.

**Evidence:** Analytical probes against existing 139-trial dataset + literature mapping against 4 established coordination taxonomies (Malone & Crowston 1994, Mintzberg 1979, MAST, distributed systems).

**Knowledge:** The coordination model's experimental findings are sound but its generalization language exceeds its evidence tier. Key insight: "task complexity" was the wrong variable (task structure matters); messaging fails due to a specific false merge model, not fundamentally; the four primitives are domain-specific to multi-agent SE.

**Next:** No implementation needed. Model.md already updated with scoped language. Three open questions added for future experimental work.

**Authority:** implementation - Scoping existing claims within model maintenance, no cross-boundary impact.

---

# Investigation: Probe Overclaimed Coordination Model Claims

**Question:** Do three overclaimed coordination model claims survive targeted probing, or should they be narrowed?

**Started:** 2026-03-22
**Updated:** 2026-03-22
**Owner:** orch-go-dotqm
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** coordination

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/coordination/probes/2026-03-22-external-framework-validation.md | extends | yes | none |
| .kb/models/coordination/probes/2026-03-22-probe-falsify-align-as-substrate.md | extends | yes | none |

---

## Findings

### Finding 1: Claim 4 — "Task complexity" is the wrong variable; task STRUCTURE matters

**Evidence:** Both tested task families (simple: FormatBytes+FormatRate; complex: VisualWidth+FormatTable) share identical structural coordination properties: additive, same file, same gravitational insertion point (line 94, after FormatDurationShort). They differ only in implementation difficulty. The anticipatory placement experiment already showed task-type sensitivity: 20% success for simple vs 100% for complex with a different mechanism.

**Source:** `experiments/coordination-demo/redesign/results/20260310-174045/` — all 80 trials. Messaging plan files from 20 trials showing identical insertion points across both task types.

**Significance:** The claim holds for additive tasks with shared gravitational points but the title "task-complexity-independent" overclaims because it implies generality across structurally different task types. Scoped to "additive task complexities."

---

### Finding 2: Claim 6 — Messaging fails due to a specific false merge model, not "fundamentally"

**Evidence:** 18/20 messaging agents wrote accurate plans and correctly identified the other agent's work. All concluded "no conflicts expected" while choosing identical insertion points. The false model: agents believe same-point additions with different function names merge cleanly. The gate experiment (20 trials) confirmed: even explicit self-checking doesn't fix the false model — agents perform the check, conclude "no conflict," and don't move.

**Source:** Plan files from `experiments/coordination-demo/redesign/results/20260310-174045/messaging/*/trial-*/messages/plan-*.txt`. Gate outputs from `results/20260322-124035/gate/*/trial-*/agent-*/stdout.log`.

**Significance:** The failure mechanism is diagnostic: agents have wrong models of git merge, not that communication can't coordinate. This means messaging could potentially work with merge-mechanics education, multi-round negotiation, or tool-augmented conflict detection. "Fundamentally flawed" overclaims from a specific and narrow failure mechanism.

---

### Finding 3: Claim 9 — Four primitives are domain-specific to multi-agent SE, not universal

**Evidence:** Mapped against 4 established taxonomies:
- Malone & Crowston (1994): 4/5 dependency types map cleanly. Gap: task decomposition (upstream of Route)
- Mintzberg (1979): 5 mechanisms are complementary (describe "how" vs primitives' "what"). Gap: Throttle has no Mintzberg equivalent
- Distributed systems: 5/7 concerns map cleanly. Gaps: fault tolerance/recovery, consistency model choice
- MAST: 14/14 failure modes map (LLM-specific confirmation)

Missing primitives: decomposition (pre-Route), recovery (post-failure), meta-coordination (choosing strategy).

**Source:** Web search for Malone & Crowston coordination theory, Mintzberg five coordination mechanisms, MAST paper. Cross-referenced with distributed systems literature.

**Significance:** The four primitives are a well-structured taxonomy for their domain. The generality claim "to any multi-agent system" is the problem, not the primitives. Scoped to "multi-agent SE with merge-based integration."

---

## Structured Uncertainty

**What's tested:**

- Claim 4 scoping: both task families share gravitational-convergence structure (verified: diff hunk headers all at @@ -92 @@)
- Claim 6 false merge model: agents believe same-point additions merge cleanly (verified: 18/20 plan files contain this error)
- Claim 9 mapping coverage: 4 taxonomies examined with structured comparison (verified: web search + literature analysis)

**What's untested:**

- Whether modification/refactoring tasks show different coordination patterns (predicted: yes, but no experiment run)
- Whether git-merge-aware messaging would change messaging condition results (predicted: plausible, but untested)
- Whether recovery/decomposition primitives should be added to the model (analysis suggests yes, but no experiment)

**What would change this:**

- Running a modification-task experiment where messaging produces 0% conflict rate would confirm Claim 4 scoping
- Running a merge-educated messaging experiment where agents change insertion points would confirm Claim 6 scoping
- Finding a multi-agent SE coordination failure that doesn't map to any of the four primitives + recovery + decomposition would extend Claim 9 further

---

## References

**Files Examined:**
- `.kb/models/coordination/model.md` — parent model, all claims
- `experiments/coordination-demo/redesign/results/20260310-174045/` — 4-condition experiment (N=80)
- `experiments/coordination-demo/redesign/results/20260322-124035/` — gate experiment (N=20)
- `experiments/coordination-demo/redesign/results/20260322-162206/` — anticipatory placement (N=20)
- `experiments/coordination-demo/redesign/prompts/*.md` — task prompts
- `pkg/display/display.go` — target file for experiments

**Probe Files Created:**
- `.kb/models/coordination/probes/2026-03-22-probe-claim4-task-type-scope.md`
- `.kb/models/coordination/probes/2026-03-22-probe-claim6-messaging-scope.md`
- `.kb/models/coordination/probes/2026-03-22-probe-claim9-primitives-generality.md`
