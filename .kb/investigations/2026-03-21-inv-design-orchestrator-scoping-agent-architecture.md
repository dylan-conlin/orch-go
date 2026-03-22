## Summary (D.E.K.N.)

**Delta:** Orchestrator should become a scoping-only agent that enriches issues before daemon execution. The architecture has 7 design forks; 5 are navigable from substrate, 2 require Dylan's judgment.

**Evidence:** 69% of daemon routing uses coarsest type-based inference (probe orch-go-j4ej7). 0 reworks across 1,102 completions means no quality signal enters the learning loop. Orchestrator currently has `orch spawn` in its tool space, creating two competing execution paths.

**Knowledge:** Separating judgment (orchestrator) from execution (daemon) eliminates the competing-executor problem. The orchestrator's value is enrichment: adding `skill:*` labels, structured descriptions, and `area:*` labels lifts routing from 69% type-fallback to label-based precision. Comprehension-gating means completion review is the throughput bottleneck by design, not by accident.

**Next:** Promote to decision if Dylan accepts. Then implementation issues for: (1) orchestrator skill rewrite, (2) enrichment protocol, (3) comprehension queue, (4) bypass measurement.

**Authority:** strategic - Reshapes the actor model (orchestrator role, daemon responsibility boundary, human throughput constraint). Irreversible once implemented.

---

# Investigation: Design Orchestrator Scoping Agent Architecture

**Question:** How should the orchestrator-as-scoping-agent architecture work? What does the orchestrator do, what does the daemon do, how does enrichment flow into routing, and how does comprehension-gating prevent the pipeline from outrunning the human?

**Started:** 2026-03-21
**Updated:** 2026-03-21
**Owner:** architect (orch-go-hlklh)
**Phase:** Complete
**Next Step:** Promote to decision if accepted
**Status:** Complete
**Model:** orchestrator-session-lifecycle

**Patches-Decision:** N/A (new architectural direction)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| probe: issue quality baseline (orch-go-j4ej7) | foundational | Yes — verified 69% type-fallback rate against events.jsonl | None |
| probe: human feedback channel structural disuse (CV-07) | extends | Yes — verified 0 reworks, 37% auto-completed | None |
| thread: comprehension-gated throughput | extends | Yes — read thread content | None |
| guide: daemon.md | context | Yes — read current daemon OODA loop | None |
| guide: spawn.md | context | Yes — read spawn flow and context generation | None |
| orchestrator SKILL.md | context | Yes — read current tool space and role definition | None |

---

## Findings

### Finding 1: The Competing Executor Problem

**Evidence:** The orchestrator SKILL.md defines two execution paths:
1. **Release path** (primary): `bd create "task" --type X -l triage:ready` → daemon picks up
2. **Direct spawn** (exception): `orch spawn SKILL "task"` → orchestrator spawns directly

The "exception" path is used for urgent work, custom context, or daemon-down scenarios. But having two execution paths means the orchestrator maintains execution concerns (skill selection, model selection, spawn flags) that duplicate daemon logic.

**Source:** Orchestrator SKILL.md lines 62-66 (fast path table), spawn.md lines 580-590 (daemon vs manual spawn table), work_cmd.go lines 154-238 (daemon spawn path)

**Significance:** If orchestrator becomes judgment-only, the "exception" path must either (a) be removed, (b) restricted to Dylan-only bypass, or (c) preserved but measured. This is Fork 1.

---

### Finding 2: Issue Enrichment Is the Leverage Point for Routing Accuracy

**Evidence:** The probe (orch-go-j4ej7) found:
- 69% of inference falls to type-based fallback (475/682 unique issues)
- Only 12% use `skill:*` labels (83/682) — the highest-confidence signal
- Only 2% of the 4,036 issue corpus have skill labels
- 34% have NO description at all (orch-go: 48%)
- The description heuristic fires for only 5% of inferences

The daemon's inference pipeline has 4 tiers (label → title → description → type), but the richer tiers are starved of data. The orchestrator creating issues via `bd create "fix X" --type bug -l triage:ready` provides only type — no skill label, no structured description, no area label.

**Source:** probe 2026-03-21-probe-issue-quality-baseline-inference-honesty.md, skill_inference.go lines 221-258 (InferSkillFromIssue)

**Significance:** If the orchestrator actively enriches issues before releasing them — adding `skill:*` labels, `area:*` labels, structured descriptions — the daemon's higher-confidence inference tiers actually get exercised. This transforms the orchestrator's role: instead of sometimes executing work, it always judges and scopes work.

---

### Finding 3: Comprehension Is Non-Delegatable and Currently Unqueued

**Evidence:** The orchestrator SKILL.md says: "Synthesis is comprehension, not reporting. Workers produce atoms; you compose meaning." And: "Understanding happens through engagement. You can't spawn 'understand this' — do it yourself."

But the completion system treats review as a throughput challenge. The daemon auto-completes 37% of work (406/1,102) with `--force` bypassing all interactive verification. The `review done` batch path closes agents with minimal review. There is no queue that tracks "reviewed but not yet comprehended" vs "comprehended."

The thread (2026-03-21) articulates the tension: "If the orchestrator needs synthesis to scope the next issue well, then comprehension IS the rate limiter."

**Source:** Orchestrator SKILL.md lines 32-43, probe 2026-03-20 human feedback channel lines 70-97 (daemon auto-complete), thread 2026-03-21 comprehension-gated throughput

**Significance:** The pipeline currently has no mechanism to slow down when comprehension lags. If scoping quality depends on prior comprehension, the system should track a comprehension queue — work that's mechanically complete but not yet understood. This queue becomes the throttle: daemon won't spawn more work while the comprehension queue exceeds a threshold.

---

## Synthesis

### Fork Analysis

**Fork 1: Does `orch spawn` stay in orchestrator's tool space?**

| Option | Pros | Cons |
|--------|------|------|
| A. Remove entirely | Clean separation, no competing executors | No escape hatch for urgent/custom work |
| B. Keep for Dylan only | Human retains bypass, orchestrator is judgment-only | Need to distinguish Dylan-via-orchestrator from orchestrator-autonomous |
| C. Keep but measure | Track bypass rate, let data decide | Doesn't solve the competing-executor problem |

**Recommendation: B — Keep `orch spawn` for Dylan only, remove from orchestrator tool space.**

**Substrate:** "Premise Before Solution" principle says validate direction before designing. The orchestrator having spawn creates a gravitational pull toward execution. Removing spawn from the orchestrator while preserving it for Dylan preserves the escape hatch without contaminating the scoping role.

**Implementation:** Orchestrator SKILL.md tool space changes from "orch spawn/complete/status/review" to "bd create/label/show, orch complete/status/review, kb context." The orchestrator can enrich and release, but cannot execute. Dylan can still run `orch spawn` directly in the terminal.

---

**Fork 2: Does the orchestrator specify `skill:*` label, or does daemon always infer?**

| Option | Pros | Cons |
|--------|------|------|
| A. Orchestrator always specifies skill label | 100% precise routing, no inference gap | Orchestrator must make skill judgment for every issue |
| B. Orchestrator adds skill label when type-based inference would be wrong | Targeted enrichment, orchestrator only intervenes when it matters | Orchestrator must know what daemon would infer |
| C. Daemon always infers, orchestrator enriches only description/labels | Simpler orchestrator, daemon fully autonomous | 69% stays at type-fallback |

**Recommendation: B — Orchestrator adds `skill:*` label when the type-based default would be wrong.**

**Substrate:** The daemon already checks `skill:*` labels first (InferSkillFromIssue line 233). The issue quality probe showed labels are the highest-confidence signal but only 12% of issues have them. The orchestrator doesn't need to label everything — only the cases where `task` → `feature-impl` is wrong (e.g., a task that needs `architect`).

**Enrichment protocol:** When creating or triaging an issue, the orchestrator evaluates: "Would the daemon's type-based inference be correct for this issue?" If no → add `skill:*` label. If yes → no label needed. Additionally, always add `area:*` labels for context injection.

---

**Fork 3: How does custom spawn context work when orchestrator can't spawn?**

| Option | Pros | Cons |
|--------|------|------|
| A. Issue description becomes ORIENTATION_FRAME (current `orch work` behavior) | Already implemented, no changes needed | Limited to flat text |
| B. Structured description with sections (Context, Constraints, Success Criteria) | Rich context for agents, better scoping | Requires description template |
| C. Orchestrator adds FRAME comments after creation | Separates issue tracking from context enrichment | More steps for orchestrator |

**Recommendation: A+C hybrid — Description is the primary context, FRAME comments add orchestrator judgment.**

**Substrate:** This is already how `orch work` operates (work_cmd.go lines 196-213). Issue description → ORIENTATION_FRAME, plus FRAME beads comments appended if they exist. The current mechanism works. The orchestrator's job is to write rich descriptions and add FRAME comments when the issue needs strategic context beyond the description.

**No structural change needed** — the mechanism exists. The behavioral change is the orchestrator consistently writing good descriptions instead of terse one-liners.

---

**Fork 4: How does comprehension-gating work mechanically?**

| Option | Pros | Cons |
|--------|------|------|
| A. Comprehension queue with daemon throttle | Explicit pressure, daemon respects human pace | Requires new queue infrastructure |
| B. Spawn rate limiter keyed to completion review backlog | Reuses existing rate limiter pattern | Conflates mechanical review with comprehension |
| C. Advisory-only: dashboard shows comprehension debt, no enforcement | Low friction, information-only | No structural pressure (same problem as advisory accretion gates) |

**Recommendation: A — Explicit comprehension queue with daemon throttle.**

**Substrate:** The accretion gates decision (2026-03-17) showed that advisory-only gates have 100% bypass rate. So option C won't work. The comprehension queue needs to be structural.

**Design:**
- After daemon auto-completes an agent, the issue enters a `comprehension:pending` state (label, not beads status)
- The orchestrator reviews completed work and removes `comprehension:pending` when synthesis is done
- If `comprehension:pending` count exceeds a threshold (configurable, default 5), daemon pauses spawning new work
- This creates backpressure: too much uncomprehended work → pipeline stalls → orchestrator catches up → pipeline resumes

**Why label-based, not status-based:** The beads issue status should remain `closed` after completion. The comprehension state is orthogonal to the issue lifecycle — work is done, but understanding hasn't happened. A label captures this cleanly.

**Counter-argument addressed:** "Idle agents with good scoping > busy agents with blind scoping." The thread articulates this well. If the orchestrator can't scope well because it hasn't understood prior work, spawning more work is negative-value (creates more uncomprehended results, compounds the comprehension debt).

---

**Fork 5: Should the daemon auto-complete, or should all completions route through orchestrator?**

| Option | Pros | Cons |
|--------|------|------|
| A. Daemon auto-completes as today (37% auto-completed) | Fast capacity reclaim, no bottleneck | No quality signal, comprehension bypassed |
| B. All completions route through orchestrator | Every completion gets comprehension | Orchestrator becomes bottleneck, overnight work stalls |
| C. Daemon mechanically completes (reclaim slot), queues for orchestrator comprehension | Fast capacity reclaim AND comprehension happens | Two-phase completion |

**Recommendation: C — Two-phase completion: daemon completes mechanically, queues for comprehension.**

**Substrate:** The thread explicitly called out this separation: "Completion review does two things: (1) mechanical verification (reclaim daemon slot) and (2) synthesis (compose meaning for Dylan). These are in tension."

**Design:**
- Phase 1 (daemon): Verify Phase: Complete reported, run mechanical checks, close beads issue, reclaim slot. Add `comprehension:pending` label.
- Phase 2 (orchestrator): Read SYNTHESIS.md, extract key learnings, connect to threads, remove `comprehension:pending`. This is when Three-Layer Reconnection happens.

**Why not defer Phase 2 indefinitely:** The thread warns: "777 orphaned investigations prove that deferred synthesis becomes permanent deferral." The comprehension queue threshold prevents this — when the queue is full, the pipeline stalls until the orchestrator catches up.

---

**Fork 6: Does orchestrator have `orch complete`, or is that daemon-only too?**

**Recommendation: Orchestrator keeps `orch complete`.**

The orchestrator's comprehension phase requires reviewing SYNTHESIS.md and running Three-Layer Reconnection. `orch complete` is how the orchestrator triggers this review for non-auto-completed agents (Block/Failed escalation). This is comprehension, not execution.

---

**Fork 7: Bypass escape hatch — who gets it, how is it measured?**

**Recommendation: Dylan (not orchestrator) gets `orch spawn` as bypass. Measure bypass rate.**

**Design:**
- `orch spawn` works as today when run directly by Dylan in terminal
- Orchestrator SKILL.md removes `orch spawn` from tool space
- When `orch spawn` is used directly (not via daemon), it emits a `spawn.bypass` event
- `orch stats` reports bypass rate: bypass spawns / total spawns
- If bypass rate exceeds 20%, it signals the daemon workflow isn't meeting needs (investigate)

---

### Blocking Questions (Fork 2 and inline work)

**Question 1 (orch-go-24ysj):** Should orchestrator specify `skill:*` labels on issues, or should daemon always infer from enriched metadata?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes:** If orchestrator always labels, daemon inference becomes fallback. If orchestrator never labels, description enrichment must be sufficient. Recommendation B (targeted labeling) is a middle path but requires orchestrator judgment about when type-inference is wrong.

**Question 2 (orch-go-n66ic):** Should `--inline` (interactive work) be human-only, or can orchestrator use it?

- **Authority:** strategic
- **Subtype:** framing
- **What changes:** If human-only, `--inline` is an escape hatch for hands-on debugging. If orchestrator can use it, orchestrator can become a single-threaded worker (against the scoping-only premise). Recommendation: human-only, consistent with removing execution from orchestrator.

---

**Answer to Investigation Question:**

The orchestrator-as-scoping-agent architecture separates judgment from execution:

1. **Orchestrator becomes judgment-only:** Scope, type, label, describe, enrich issues. Remove `orch spawn` from orchestrator tool space. Orchestrator retains `orch complete` for comprehension phase.

2. **Daemon becomes sole executor:** Picks up `triage:ready` issues, infers skill (enriched by orchestrator labels), spawns agents, mechanically completes them. No changes to daemon OODA loop needed.

3. **Enrichment protocol:** Orchestrator adds `skill:*` labels when type-inference would be wrong, adds `area:*` labels always, writes structured descriptions. This lifts routing from 69% type-fallback to label-based precision.

4. **Comprehension-gating:** Two-phase completion: daemon mechanically completes (fast), adds `comprehension:pending` label. Orchestrator does comprehension review (slow). Daemon throttles when `comprehension:pending` exceeds threshold. Pipeline speed = human comprehension rate.

5. **Bypass escape hatch:** Dylan retains `orch spawn` for direct use. Bypass rate measured. Orchestrator cannot spawn.

---

## Structured Uncertainty

**What's tested:**

- ✅ 69% type-fallback rate is real (verified: events.jsonl analysis, probe orch-go-j4ej7)
- ✅ 0 reworks across 1,102 completions means no quality signal (verified: events.jsonl grep)
- ✅ 37% auto-completed by daemon with --force (verified: events.jsonl grep, source code read)
- ✅ `orch work` already uses issue description as ORIENTATION_FRAME (verified: work_cmd.go lines 196-213)
- ✅ Daemon already checks `skill:*` labels before type inference (verified: skill_inference.go line 233)

**What's untested:**

- ⚠️ Comprehension queue threshold of 5 (not benchmarked — may need tuning)
- ⚠️ Whether orchestrator-written descriptions actually improve agent outcomes (no A/B test)
- ⚠️ Whether 20% bypass rate threshold is meaningful (no historical baseline)
- ⚠️ Whether `comprehension:pending` label survives daemon auto-complete (need to verify label persistence through `bd close`)

**What would change this:**

- If comprehension queue creates unacceptable latency (Dylan needs faster throughput), threshold may need to be higher or the gate may need to be advisory
- If orchestrator enrichment doesn't measurably improve routing accuracy, the scoping role may be over-specified
- If bypass rate is consistently >50%, the daemon-first workflow doesn't match actual usage

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Remove orch spawn from orchestrator tool space | architectural | Changes actor model boundary |
| Enrichment protocol (orchestrator labels/describes issues) | architectural | New orchestrator behavioral norm |
| Comprehension queue with daemon throttle | architectural | New infrastructure, cross-component |
| Two-phase completion | architectural | Changes daemon completion flow |
| Bypass measurement | implementation | Adds event logging, within existing patterns |
| Whether to adopt this architecture at all | strategic | Reshapes system roles irreversibly |

### Recommended Approach: Orchestrator-as-Scoping-Agent

**Phased implementation** — ship in 4 stages to validate assumptions:

**Phase 1: Enrichment Protocol (behavioral, no code)**
- Update orchestrator SKILL.md: remove `orch spawn` from tool space
- Add enrichment guidance: when to add `skill:*`, `area:*` labels
- Add description quality guidance: what makes a good issue description
- **Validation:** Compare daemon routing accuracy before/after over 50 issues

**Phase 2: Comprehension Queue Infrastructure**
- Add `comprehension:pending` label support
- Modify daemon auto-complete to add label after closing
- Add daemon throttle: check pending count before spawn cycle
- Add `orch comprehension` command: list pending, mark reviewed
- **Validation:** Queue stays bounded, daemon throttles when full

**Phase 3: Two-Phase Completion**
- Modify `orch complete` to explicitly remove `comprehension:pending`
- Add Three-Layer Reconnection as structured prompt in `orch complete`
- Track comprehension time per issue
- **Validation:** Non-zero comprehension events, queue drains during sessions

**Phase 4: Bypass Measurement**
- Add `spawn.bypass` event when `orch spawn` used directly
- Add bypass rate to `orch stats`
- Set up alerting if bypass exceeds threshold
- **Validation:** Bypass rate measurable, stable under 20%

### Alternative Approaches Considered

**Option B: Orchestrator retains spawn but with friction**
- **Pros:** Gradual transition, no hard removal
- **Cons:** Competing executor problem persists. "Friction" didn't work for accretion gates (100% bypass rate)
- **When to use:** If removing spawn from orchestrator creates too much loss of capability

**Option C: Full daemon autonomy (no orchestrator enrichment)**
- **Pros:** Simplest architecture, daemon handles everything
- **Cons:** 69% type-fallback stays. No quality improvement without enrichment.
- **When to use:** If orchestrator enrichment doesn't measurably improve outcomes

**Rationale:** Option A (recommended) addresses the root cause: the orchestrator's value is judgment, not execution. The daemon's weakness is routing accuracy. Connecting them (orchestrator enriches, daemon executes) solves both.

### Things to Watch Out For

- ⚠️ Comprehension queue could stall overnight work if threshold is too low — daemon runs overnight but orchestrator isn't available to drain queue
- ⚠️ Labels on closed issues may have different persistence behavior in beads — verify `comprehension:pending` survives `bd close`
- ⚠️ Phase 1 is behavioral-only: if orchestrator doesn't actually write better descriptions, the architecture doesn't deliver value
- ⚠️ Defect class exposure: Class 5 (Contradictory Authority Signals) — orchestrator labels may conflict with daemon inference. Priority order (label > title > description > type) handles this, but edge cases may arise

### Success Criteria

- ✅ Daemon routing uses `skill:*` labels for >30% of inferences (up from 12%)
- ✅ Comprehension queue stays bounded (never exceeds 2x threshold)
- ✅ Bypass rate stays under 20%
- ✅ Zero orphaned `comprehension:pending` labels older than 48 hours
- ✅ Orchestrator SKILL.md has no spawn commands in tool space

---

## References

**Files Examined:**
- `pkg/daemon/skill_inference.go` — 4-tier inference pipeline, InferSkillFromIssue
- `cmd/orch/work_cmd.go` — daemon spawn path, issue description → ORIENTATION_FRAME
- `~/.claude/skills/meta/orchestrator/SKILL.md` — current orchestrator role and tool space
- `.kb/guides/daemon.md` — daemon OODA loop, skill inference, auto-completion
- `.kb/guides/spawn.md` — spawn flow, gates, context generation
- `.kb/guides/completion.md` — verification architecture, escalation model

**Related Artifacts:**
- **Probe:** `.kb/models/measurement-honesty/probes/2026-03-21-probe-issue-quality-baseline-inference-honesty.md` — 69% type-fallback finding
- **Probe:** `.kb/models/completion-verification/probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md` — 0 reworks, 37% auto-complete
- **Thread:** `.kb/threads/2026-03-21-comprehension-gated-throughput-pipeline-speed.md` — comprehension as rate limiter
- **Question:** orch-go-24ysj — skill specification fork
- **Question:** orch-go-n66ic — inline work fork
