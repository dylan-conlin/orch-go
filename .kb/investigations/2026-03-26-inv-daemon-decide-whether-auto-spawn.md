## TLDR

The daemon only auto-spawns work after two layers pass: cycle-level spawn gates and per-issue compliance/coordination checks. If an item survives that path it is spawned directly, sometimes rerouted to extraction or architect; deferral to the orchestrator mainly happens after completion, where default review-tier work is labeled `daemon:ready-review` and persistent verification failures escalate to `triage:review`.

## Summary (D.E.K.N.)

**Delta:** I traced the daemon's triage tree from queue polling through spawn routing and the separate completion path that hands work back to the orchestrator.

**Evidence:** `CheckPreSpawnGates`, `CheckIssueCompliance`, `ShouldDeferTestIssue`, `Decide`, `RouteIssueForSpawn`, `RouteCompletion`, and `ProcessCompletion` all align with targeted daemon tests that passed in `go test ./pkg/daemon -run 'Test(CheckPreSpawnGates|CheckIssueCompliance|Decide|ShouldDeferTestIssue|CheckArchitectEscalation|RouteCompletion|ProcessCompletion)'`.

**Knowledge:** The daemon's "spawn vs orchestrator" split is not a single branch; spawn decisions happen before agent launch, while orchestrator deferral is mostly a completion-routing outcome.

**Next:** Close this investigation and use it as the explain-back reference for daemon triage behavior.

**Authority:** implementation - This is a codepath-mapping investigation inside existing daemon behavior, not a new architectural decision.

---

# Investigation: Daemon Decide Whether Auto Spawn

**Question:** How does `pkg/daemon/` decide whether a beads issue gets auto-spawned versus deferred for orchestrator review?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## What I Tried

- Read the daemon's OODA loop, compliance checks, routing helpers, spawn execution path, and completion routing code.
- Cross-checked behavior against unit tests in `pkg/daemon/` that exercise gates, filtering, deferral, escalation, and completion routing.
- Ran a focused `go test` command against the relevant decision-tree tests.

## Findings

### Finding 1: The daemon can refuse to spawn before it looks at any specific issue

**Evidence:** `CheckPreSpawnGates()` short-circuits the cycle if verification is paused, completion processing is unhealthy, the comprehension queue is full, or the hourly rate limit is hit. `Decide()` immediately returns `Blocked=true` when `GateSignal.Allowed` is false.

**Source:** `pkg/daemon/compliance.go:25`, `pkg/daemon/ooda.go:116`, `pkg/daemon/compliance_test.go:17`

**Significance:** "Defer to orchestrator" starts at the cycle level: if these gates fail, nothing is auto-spawned regardless of queue contents.

---

### Finding 2: Issue-level triage is a filter chain plus one coordination deferral

**Evidence:** `Decide()` iterates prioritized issues and first applies `ShouldDeferTestIssue()` so same-project test tasks wait behind active implementation siblings. It then applies `CheckIssueCompliance()`, which rejects items in the skip set, recently spawned items, missing types, non-spawnable types, `blocked` or `in_progress` issues, items already labeled with daemon completion labels, issues missing the required spawn label, and issues blocked by dependencies.

**Source:** `pkg/daemon/ooda.go:149`, `pkg/daemon/sibling_sequencing.go:60`, `pkg/daemon/compliance.go:109`, `pkg/daemon/sibling_sequencing_test.go:49`

**Significance:** Before spawning, the daemon's practical rule is "take the first prioritized issue that is not deferred and passes compliance." Everything else stays in beads for a later cycle or human intervention.

---

### Finding 3: Surviving issues are routed, not handed to the orchestrator, unless the daemon later hits completion review

**Evidence:** Once an issue is selected, `InferSkillFromIssue()` chooses the skill by label, title, description, then type fallback. `RouteIssueForSpawn()` may replace the issue with an extraction task for critical hotspots or escalate implementation work to `architect` for hotspot areas when compliance permits; explicit `skill:*` labels suppress architect escalation. If routing succeeds, `spawnIssue()` marks the issue `in_progress` before calling `SpawnWork()`.

**Source:** `pkg/daemon/skill_inference.go:214`, `pkg/daemon/coordination.go:37`, `pkg/daemon/architect_escalation.go:78`, `pkg/daemon/spawn_execution.go:57`, `pkg/daemon/architect_escalation_test.go:115`

**Significance:** For open work, the daemon does not usually "defer to orchestrator" after selection; it either spawns the selected issue, spawns replacement extraction work, or blocks and retries later.

---

### Finding 4: The main orchestrator handoff happens after agent completion

**Evidence:** `ProcessCompletion()` first verifies the finished workspace, then `RouteCompletion()` sends `effort:small` to `auto-complete-light`, `review-tier=auto` or `review-tier=scan` to full auto-complete, and everything else to `label-ready-review`. That default path adds `daemon:ready-review` for orchestrator review. If verification keeps failing and the retry budget is exhausted, the daemon labels the issue `daemon:verification-failed`; a periodic escalation later adds `triage:review` for human attention.

**Source:** `pkg/daemon/coordination.go:147`, `pkg/daemon/completion_processing.go:355`, `pkg/daemon/verification_failed_escalation.go:64`, `pkg/daemon/coordination_test.go:11`

**Significance:** The answer to "auto-spawn vs orchestrator" is two-stage: spawning is autonomous when pre-spawn gates and issue checks pass, but completed work usually returns to orchestrator review unless its tier explicitly allows daemon auto-completion.

---

## What I Observed

- The daemon's spawn path is organized as `Sense -> Orient -> Decide -> Act`, which makes the triage tree explicit rather than implicit across helpers.
- The queue is intentionally broader than the spawnable set: `Sense()` still reads ready issues even when a gate blocks, and `Decide()` narrows that list.
- The clearest orchestrator handoff in current code is not before spawning but after completion, via `daemon:ready-review` and eventually `triage:review`.

## Test Performed

```bash
go test ./pkg/daemon -run 'Test(CheckPreSpawnGates|CheckIssueCompliance|Decide|ShouldDeferTestIssue|CheckArchitectEscalation|RouteCompletion|ProcessCompletion)'
```

**Result:** PASS (`ok   github.com/dylan-conlin/orch-go/pkg/daemon 0.423s`)

---

## Synthesis

**Key Insights:**

1. **Spawnability is layered** - A queue item is not spawnable just because it is "ready"; it must survive cycle gates, coordination deferral, and issue compliance.
2. **Routing is still autonomous** - Hotspot extraction and architect escalation change what gets spawned, but they are still daemon-side spawn outcomes rather than orchestrator deferrals.
3. **Orchestrator review is the default completion sink** - The daemon is conservative about closing work on its own, so most normal-tier finished agents end up waiting for orchestrator explain-back.

**Answer to Investigation Question:**

The daemon auto-spawns an issue only if the poll-cycle gates allow spawning, the issue survives test-sibling deferral and compliance filtering, and routing succeeds without a spawn failure. At that point it marks the issue `in_progress` and launches work directly, possibly after swapping in extraction work or escalating the skill to `architect`. The daemon defers to the orchestrator mainly after completion: unless the finished agent is `effort:small`, `review-tier=auto`, or `review-tier=scan`, the daemon labels it `daemon:ready-review` instead of closing it, and repeated verification failures escalate further to `triage:review`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Pre-spawn gating and issue-compliance branches are covered by targeted unit tests and match the current source.
- ✅ Test-sibling deferral is covered by unit tests and is invoked directly from `Decide()`.
- ✅ Completion routing defaults to orchestrator review unless an auto-complete tier applies.

**What's untested:**

- ⚠️ I did not run a live daemon cycle against a real beads queue.
- ⚠️ I did not simulate cross-project spawning or cross-project completion routing end to end.
- ⚠️ I did not exercise UI/status-display code that presents this decision tree to humans.

**What would change this:**

- A live daemon run that contradicted the tested branch order would change the conclusion.
- Hidden spawn-side effects outside `pkg/daemon/` would change the conclusion if they mutate labels or status before `Decide()` runs.
- A different completion tier source than `verify.ReadReviewTierFromWorkspace()` would change the handoff conclusion.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| No code change; use this investigation to explain current daemon triage behavior | implementation | The task was to map current behavior, and the code already matches the traced decision tree |

### Recommended Approach ⭐

**Close as documentation** - Keep this as a knowledge artifact rather than turning it into a behavior change.

**Why this approach:**
- The requested deliverable was understanding, not a fix.
- The current code and tests are internally consistent.
- The investigation already gives the orchestrator an explain-back path.

**Trade-offs accepted:**
- No additional dashboard visualization of the decision tree.
- No runtime trace logging beyond what already exists.

**Implementation sequence:**
1. Preserve this investigation as the canonical trace.
2. Use it during `orch complete` explain-back.
3. Only open a follow-up if humans need a more visible status surface.

---

## References

**Files Examined:**
- `pkg/daemon/compliance.go` - Pre-spawn gates and issue compliance filters.
- `pkg/daemon/ooda.go` - Sense/Orient/Decide/Act decision sequence.
- `pkg/daemon/sibling_sequencing.go` - Test-issue deferral rule.
- `pkg/daemon/skill_inference.go` - Skill inference priority order.
- `pkg/daemon/coordination.go` - Spawn routing and completion routing.
- `pkg/daemon/architect_escalation.go` - Hotspot-based architect escalation.
- `pkg/daemon/spawn_execution.go` - Final spawn execution and status mutation.
- `pkg/daemon/completion_processing.go` - Completion verification and ready-review labeling.
- `pkg/daemon/verification_failed_escalation.go` - Human-review escalation after retry exhaustion.

**Commands Run:**
```bash
# Verify project location
pwd

# Create investigation artifact
kb create investigation daemon-decide-whether-auto-spawn --orphan

# Run targeted daemon decision tree tests
go test ./pkg/daemon -run 'Test(CheckPreSpawnGates|CheckIssueCompliance|Decide|ShouldDeferTestIssue|CheckArchitectEscalation|RouteCompletion|ProcessCompletion)'
```

**Related Artifacts:**
- `/.orch/workspace/og-inv-daemon-decide-whether-26mar-ccf5/SPAWN_CONTEXT.md` - Session protocol and deliverable requirements.

---

## Investigation History

**2026-03-26 00:00:** Investigation started
- Initial question: How daemon triage decides spawn versus orchestrator handoff.
- Context: Requested trace through `pkg/daemon/` decision logic.

**2026-03-26 00:00:** Codepath mapping completed
- Read the gating, filtering, routing, spawn, and completion code paths and matched them against relevant tests.

**2026-03-26 00:00:** Investigation completed
- Status: Complete
- Key outcome: The daemon auto-spawns only after layered gates pass, while orchestrator deferral is primarily a completion-routing behavior.
