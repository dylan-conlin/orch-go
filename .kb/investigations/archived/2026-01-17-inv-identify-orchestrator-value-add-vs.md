<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator judgment matters for synthesis, goal refinement, frame correction, hotspot detection, and triage decisions (the 20%); routing execution is already automated by daemon (the 80%), and "routing overhead" is workflow debt from triage discipline gaps and spawn reliability issues, not necessary orchestrator function.

**Evidence:** Strategic Orchestrator Model decision establishes orchestrator role as comprehension; daemon model documents complete automation of poll-spawn-complete cycle; prior investigations found 26% daemon utilization despite full automation; daemon log shows spawn reliability failures; 27 triage:ready issues queued showing triage discipline gap.

**Knowledge:** The question reframes from "what routing can daemon handle" (already handles 80%) to "how to reduce workflow friction so daemon autonomy is actually used" (fix spawn reliability, strengthen triage discipline, clarify exception criteria).

**Next:** Create issues for: (1) investigate spawn reliability "Failed to extract session ID" errors, (2) implement Proactive Hygiene Checkpoint in orchestrator skill, (3) add daemon utilization metric/alert, (4) document manual spawn exception criteria.

**Promote to Decision:** recommend-no - Findings consolidate existing decisions (Strategic Orchestrator Model, Synthesis is Strategic Work) and point to implementation improvements, not new architectural choices.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Identify Orchestrator Value Add Vs

**Question:** When does orchestrator judgment actually matter vs when is it just routing? If daemon could handle 80% of spawns correctly, orchestrator should focus on the 20%. Informs: daemon autonomy expansion, orchestrator focus areas.

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-identify-orchestrator-value-17jan-95d9
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Strategic Orchestrator Model Redefines the Division of Labor

**Evidence:** Decision document "Strategic Orchestrator Model" (2026-01-07) established that orchestrator's job is **comprehension**, not coordination. Coordination is the daemon's job. The work division:

| Work Type | Who Does It | Why |
|-----------|-------------|-----|
| Investigation (discovering facts) | Worker agent | Requires codebase exploration |
| Implementation (writing code) | Worker agent | Requires file editing |
| Synthesis (combining findings) | Strategic orchestrator | Requires cross-agent context |
| Understanding (building models) | Strategic orchestrator | Requires engagement, not delegation |
| Coordination (what to spawn when) | Daemon | Already automated |

**Source:** 
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md`
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md`
- `.kb/models/daemon-autonomous-operation.md`

**Significance:** This reframes the question. Orchestrators aren't meant to do routing at all - that's already automated. The question is whether orchestrators are spending time on comprehension/synthesis (high value) or getting pulled into tactical dispatch (low value).

---

### Finding 2: High-Value Activities Require Strategic Judgment, Not Routing

**Evidence:** Investigation "Interactive Orchestrators Compensation Pattern" (2026-01-06) identified three legitimate orchestrator functions that daemon cannot replicate:

1. **Goal refinement** - Converting vague strategic intent ("improve performance") to actionable orchestrator goals ("reduce orch status latency from 1.2s to <100ms")
2. **Real-time frame correction** - Catching when orchestrator drops into tactical mode (doing spawnable work) and shifting perspective
3. **Synthesis** - Combining worker results into decisions/knowledge (can't spawn "understand this topic")

Additional high-value activities from orchestrator skill:
- **Hotspot detection** - Recognizing when 5+ bug fixes to same area signals systemic issue requiring architect, not more debugging
- **Issue type correction** - When daemon skill inference would be wrong (issue labeled "task" but actually needs feature-impl)
- **Follow-up extraction** - Reading SYNTHESIS.md recommendations from completed agents and deciding what to pursue
- **Epic readiness evaluation** - Determining if understanding is complete enough to spawn work ("can you explain the problem, constraints, and risks?")

**Source:** 
- `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md` lines 118-130
- Orchestrator skill "Strategic-First Orchestration" section
- Orchestrator skill "Orchestrator Core Responsibilities" section

**Significance:** These are categorically different from queue processing. They require reasoning, judgment, and cross-agent context that daemon cannot replicate. This is the 20% that requires orchestrator engagement.

---

### Finding 3: Daemon Already Automates Routing - This is Not Orchestrator Work

**Evidence:** Daemon autonomous operation model documents complete automation of routing mechanics:

**Poll-Spawn-Complete Cycle (runs every 60s):**
1. Reconcile with OpenCode (free stale pool slots)
2. Poll beads: `bd ready --limit 0` (get all ready issues)
3. Filter for `triage:ready` label
4. Infer skill from issue type (bug→systematic-debugging, feature→feature-impl, task→investigation)
5. Spawn within capacity limits (WorkerPool with MaxAgents=5)
6. Monitor for `Phase: Complete` comments (separate completion loop)
7. Verify completion and close issues

**Skill inference mapping:**
- `bug` type → systematic-debugging
- `feature` type → feature-impl  
- `task` type → investigation
- `epic` type → architect
- Fallback → investigation

**Capacity management:**
- WorkerPool tracks active agents by beads ID
- Blocks spawning if at MaxAgents capacity
- Reconciles with OpenCode every poll to free stale slots
- Cross-project operation (polls multiple project directories)

Current status check: `launchctl list | grep orch` shows daemon running (PID 42350), `bd list -l triage:ready --limit 0` shows 27 ready issues available for spawning.

**Source:** 
- `.kb/models/daemon-autonomous-operation.md` lines 1-330
- `pkg/daemon/daemon.go`, `pkg/daemon/skill_inference.go`, `pkg/daemon/pool.go`
- Live verification: daemon running, 27 triage:ready issues queued

**Significance:** **This is the 80%**. Skill inference, spawn execution, completion detection, capacity management - all automated. Orchestrators should NOT be doing routing. If they are, it's a workflow problem, not an automation gap.

---

### Finding 4: Triage is the Judgment Bottleneck - Labels Control Daemon Autonomy

**Evidence:** The triage workflow shows where orchestrator judgment gates daemon autonomy:

**Triage label meanings:**
- `triage:ready` - Confident spawn, daemon spawns immediately on next poll (60s)
- `triage:review` - Needs orchestrator review before spawning
- (no label) - Default, daemon skips

**The flow:**
```
User reports symptom → orch spawn issue-creation "symptom"
                     ↓
Issue created with triage:review (default for uncertainty)
                     ↓
Orchestrator reviews, relabels triage:ready
                     ↓
Daemon auto-spawns on next poll
```

**What orchestrator judges during triage:**
1. **Type correctness** - Does issue type (bug/feature/task/epic) match actual work?
2. **Scope clarity** - Can agent complete without mid-work clarification?
3. **Hotspot detection** - Is this the 5th fix to same area? (Needs architect, not tactical spawn)
4. **Dependency check** - Are blockers resolved? (`bd show <id>` shows no deps)

Investigation "Add Proactive Triage Workflow" (2026-01-09) found: "Triage requires judgment (can't be command-driven automation)" and "preserves orchestrator judgment (can't be automated)".

Current gap: Daemon running with 27 triage:ready issues, but orchestrator skill guidance says "triage is part of hygiene checkpoint" - suggests triage discipline is inconsistent.

**Source:**
- Orchestrator skill "Triage Protocol" section lines 97-166
- `.kb/investigations/2026-01-09-inv-add-proactive-triage-workflow-orchestrators.md` lines 144-150
- Daemon model "Triage Workflow" section
- Live state: `bd list -l triage:ready --limit 0` shows 27 ready

**Significance:** Triage is where orchestrator judgment GATES daemon autonomy. The daemon can't make these judgment calls. But once orchestrator labels `triage:ready`, routing is fully automated. The goal is faster triage cycles, not smarter routing.

---

### Finding 5: Daemon Utilization Gap is Separate from Orchestrator Value

**Evidence:** Prior investigation found daemon utilization at 26% (74% manual spawns). But investigation "Interactive Orchestrators Compensation Pattern" (2026-01-06) concluded: "Daemon underutilization (26% vs target) and interactive orchestrator value are separate questions."

The investigation found interactive orchestrators serve legitimate functions:
- Goal refinement (converting vague intent to actionable goals)
- Frame correction (catching tactical mode drops)
- Synthesis (combining worker results)

These are NOT compensating for daemon gaps. They're categorically different work.

**Daemon underutilization causes (from investigations):**
1. **Triage discipline** - Issues not being labeled triage:ready systematically
2. **Bypass-triage friction** - Manual spawn easier than triage workflow
3. **Spawn failures** - Daemon log shows "Headless spawn failed: Failed to extract session ID" errors
4. **Exception cases** - Design-session (100% manual), investigation (90% manual) inherently need orchestrator context

**Source:**
- `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md` lines 117-132
- `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md` lines 8, 69, 78, 119
- Daemon log: `tail -50 ~/.orch/daemon.log` shows spawn failures
- Live check: 27 triage:ready issues queued but daemon experiencing spawn errors

**Significance:** Low daemon utilization is NOT evidence that daemon can't handle routing. It's evidence of triage workflow friction and spawn reliability issues. Fixing daemon utilization doesn't change what orchestrators should focus on (synthesis, judgment, comprehension).

---

## Synthesis

**Key Insights:**

1. **Orchestrators Aren't Meant to Route - That's Already Automated** - The Strategic Orchestrator Model (Finding 1) and Daemon Autonomous Operation (Finding 3) show routing is fully automated: poll beads, infer skill from type, spawn, monitor completion, close. Orchestrators doing routing work is a SYMPTOM of workflow problems, not a necessary function.

2. **The 20% Requiring Judgment is Strategic Work, Not Dispatch** - Goal refinement, frame correction, synthesis, hotspot detection, epic readiness evaluation (Finding 2). These aren't "routing with extra steps" - they're fundamentally different from queue processing. They require cross-agent context, reasoning about patterns, and understanding that spans multiple investigations.

3. **Triage is the Judgment Bottleneck, Not the Routing** - Orchestrator judgment happens at triage time: Is the issue type correct? Is scope clear? Is this a hotspot area? Are dependencies resolved? (Finding 4). Once labeled `triage:ready`, daemon handles everything. The goal is **faster triage cycles**, not **smarter routing**.

4. **Low Daemon Utilization ≠ Orchestrators Should Do Routing** - 74% manual spawns doesn't mean daemon can't handle routing (Finding 5). It means triage discipline is inconsistent, spawn reliability has issues, and some skills (design-session, investigation) inherently need orchestrator context. Fix the workflow friction, not by having orchestrators do routing manually.

**Answer to Investigation Question:**

**Orchestrator judgment matters for: synthesis, goal refinement, frame correction, hotspot detection, and triage decisions (the 20%).** Routing execution is already automated by the daemon (the 80%). 

The "routing overhead" isn't a necessary orchestrator function - it's workflow debt:
- Triage discipline gaps → issues not labeled `triage:ready` systematically
- Spawn reliability issues → daemon experiencing failures, orchestrators work around via manual spawn
- Exception cases treated as defaults → skills needing orchestrator context (design-session) used for standard work

**Daemon autonomy expansion path:** Fix triage discipline (proactive hygiene checkpoints), fix spawn reliability (investigate "Failed to extract session ID" errors), clarify when manual spawn is actually needed (urgent, complex, interactive synthesis) vs workflow workaround.

**Orchestrator focus areas:** Synthesis (combining findings from multiple agents), triage judgment (is type correct, is scope clear, is this a hotspot), goal refinement (converting Dylan's vague intent to actionable goals), frame correction (catching tactical mode drops). These are not automatable - they require strategic comprehension.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon is running and operational (verified: `launchctl list | grep orch` shows PID 42350)
- ✅ 27 triage:ready issues exist and are available for daemon spawning (verified: `bd list -l triage:ready --limit 0`)
- ✅ Daemon polls every 60s and spawns triage:ready issues (verified: read daemon.go source and daemon model)
- ✅ Daemon infers skill from issue type (verified: skill_inference.go mapping table)
- ✅ Strategic Orchestrator Model decision exists and defines orchestrator role as comprehension, not coordination (verified: read decision document)
- ✅ Prior investigation found daemon utilization at 26% (verified: read investigation document, grep events.jsonl)

**What's untested:**

- ⚠️ Whether fixing triage discipline would actually increase daemon utilization to target levels (hypothesis, not measured)
- ⚠️ Whether current spawn reliability issues are systematic or transient (single daemon log sample, not trend analysis)
- ⚠️ What percentage of manual spawns are workflow workarounds vs legitimate exceptions (claim based on investigation finding, not current measurement)
- ⚠️ Whether orchestrators actually follow proactive triage checkpoint guidance (behavioral, not verified)
- ⚠️ Whether faster triage cycles would reduce perceived "routing overhead" (proposed improvement, not tested)

**What would change this:**

- If daemon skill inference frequently chooses wrong skill → orchestrator routing judgment needed
- If triage decisions (type correctness, scope clarity) could be automated → reduces orchestrator triage burden
- If synthesis could be delegated to spawned agents → reduces orchestrator strategic work
- If orchestrators consistently spend <10% time on triage → "routing overhead" perception is misaligned with reality
- If manual spawn rate stays high after fixing spawn reliability → suggests legitimate need for orchestrator dispatch judgment

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Strengthen Triage Discipline + Fix Spawn Reliability** - Reduce orchestrator routing overhead by making daemon autonomy actually work consistently, freeing orchestrators to focus on synthesis and judgment.

**Why this approach:**
- Daemon already handles 80% of routing mechanics (Finding 3) - leverage existing automation
- Triage is where orchestrator judgment adds value (Finding 4) - type correctness, scope clarity, hotspot detection
- Low daemon utilization is workflow friction, not automation gap (Finding 5) - fix the friction
- Strategic work (synthesis, goal refinement, frame correction) is irreducible (Finding 2) - can't automate this away
- Aligns with Strategic Orchestrator Model (Finding 1) - orchestrators comprehend, daemon coordinates

**Trade-offs accepted:**
- Doesn't eliminate orchestrator triage work (judgment required)
- Doesn't automate synthesis (requires cross-agent context and reasoning)
- Still have exception cases (design-session, urgent items) requiring manual spawn
- Requires behavior change (proactive triage hygiene) which is hard to enforce

**Implementation sequence:**
1. **Fix spawn reliability** - Investigate "Failed to extract session ID" errors in daemon log (foundational: daemon must work reliably)
2. **Strengthen triage checkpoints** - Proactive Hygiene Checkpoint investigation already designed this (2026-01-09), implement in orchestrator skill
3. **Measure daemon utilization** - Add metric/alert for daemon spawn % to surface when triage discipline slips
4. **Document exception criteria** - When is manual spawn actually needed vs workflow workaround? (clarifies when orchestrator dispatch judgment is legitimate)

### Alternative Approaches Considered

**Option B: Automate More of Triage via Heuristics**
- **Pros:** Could auto-label simple cases (bug with repro steps, feature with clear spec), reduce triage burden
- **Cons:** Triage judgment (type correctness, hotspot detection, scope clarity) requires reasoning daemon can't replicate (Finding 4). Automation would miss edge cases (issue labeled "task" but needs feature-impl, 5th fix to same area needs architect not debugging).
- **When to use instead:** If 80%+ of triage decisions are formulaic (type matches work, no hotspots, deps clear). Current evidence suggests judgment is needed.

**Option C: Have Orchestrators Do More Routing Manually**
- **Pros:** Gives orchestrators more control, works around daemon reliability issues
- **Cons:** This is the OPPOSITE of the goal. Findings 1 and 3 show routing is already automated - orchestrators doing routing is workflow debt. Finding 2 shows high-value work is synthesis and judgment, not dispatch.
- **When to use instead:** Never. If daemon can't handle routing, fix the daemon, don't have orchestrators compensate.

**Option D: Eliminate Interactive Orchestrators, Rely Fully on Daemon**
- **Pros:** Simpler model, less human coordination overhead
- **Cons:** Finding 2 shows synthesis, goal refinement, and frame correction require orchestrator engagement. These aren't automatable. Finding 5 investigation explicitly rejected this: "Interactive orchestrators are NOT primarily compensation for daemon gaps."
- **When to use instead:** If synthesis could be automated (workers self-synthesize) and goals are always clear upfront (no refinement needed). Evidence suggests neither is true.

**Rationale for recommendation:** The daemon already automates routing (Finding 3). The problem is workflow friction preventing that automation from being used (Finding 5). Fix the friction (spawn reliability, triage discipline) rather than having orchestrators do routing manually (which is both low-value and already automated). This frees orchestrators for high-value work: synthesis, judgment, comprehension (Findings 1 and 2).

---

### Implementation Details

**What to implement first:**
1. **Investigate spawn reliability** - Daemon log shows "Failed to extract session ID" errors. Create beads issue to debug this (foundational: daemon must work reliably before expecting orchestrators to use it).
2. **Add daemon utilization metric** - Track daemon spawn % vs manual spawn % in events.jsonl. Quick win: surfaces when triage discipline slips.
3. **Implement Proactive Hygiene Checkpoint** - Investigation 2026-01-09 designed this, just needs implementation in orchestrator skill. Provides systematic triage triggers.

**Things to watch out for:**
- ⚠️ "Faster triage" could pressure orchestrators to skip judgment (type correctness, hotspot detection) - emphasize quality over speed
- ⚠️ Daemon reliability must be proven before expecting orchestrators to trust it - spawn failures create manual spawn workarounds
- ⚠️ Exception criteria (when manual spawn is legitimate) must be clear - otherwise "urgent" becomes default rationalization
- ⚠️ Triage discipline is behavioral change - guidance alone may not be enough, may need gates/reminders
- ⚠️ Synthesis time may be invisible to metrics (no spawn events) - don't optimize for daemon % at expense of strategic work

**Areas needing further investigation:**
- What percentage of manual spawns are actually legitimate exceptions vs workflow workarounds? (Need spawn event analysis with reason codes)
- Can triage type-checking be partially automated? (Heuristics: "fix", "bug" in title → probably bug type)
- What spawn reliability issues exist beyond "Failed to extract session ID"? (Comprehensive daemon failure mode analysis)
- How much time do orchestrators actually spend on triage vs synthesis? (Time tracking would inform priority)
- What synthesis opportunities are missed due to triage overhead? (Qualitative: ask Dylan)

**Success criteria:**
- ✅ Daemon spawn % increases from 26% baseline (measure via events.jsonl analysis after 1 week)
- ✅ triage:ready queue stays <10 issues (measure via `bd list -l triage:ready` daily)
- ✅ Spawn reliability failures drop to <5% (measure via daemon log analysis)
- ✅ Orchestrators report less time on dispatch, more on synthesis (qualitative: Dylan feedback)
- ✅ Strategic work artifacts increase (SYNTHESIS.md files created, kb quick entries, decision documents) - measure via git log

---

## References

**Files Examined:**
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Defines orchestrator role as comprehension, not coordination
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Synthesis cannot be delegated to spawned agents
- `.kb/models/daemon-autonomous-operation.md` - Complete model of daemon poll-spawn-complete cycle and automation
- `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md` - Found orchestrators serve legitimate functions (goal refinement, frame correction, synthesis)
- `.kb/investigations/2026-01-09-inv-add-proactive-triage-workflow-orchestrators.md` - Designed proactive hygiene checkpoint for triage discipline
- `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md` - Found 26% daemon utilization, 74% manual spawns
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator skill guidance (triage protocol, strategic-first orchestration)
- `pkg/daemon/daemon.go`, `pkg/daemon/skill_inference.go`, `pkg/daemon/pool.go` - Daemon implementation source

**Commands Run:**
```bash
# Check if daemon is running
launchctl list | grep orch

# Check daemon recent activity and errors
tail -50 ~/.orch/daemon.log

# Count triage:ready issues available for daemon spawning
bd list -l triage:ready --limit 0

# Sample recent spawn patterns (mode: daemon vs manual)
grep "spawn\." ~/.orch/events.jsonl | tail -200 | jq -r '.spawn_mode // "manual"' | sort | uniq -c

# Check beads issue list
bd list --limit 20
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Establishes orchestrator role as comprehension, daemon role as coordination
- **Decision:** `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Synthesis requires orchestrator engagement, not spawnable
- **Model:** `.kb/models/daemon-autonomous-operation.md` - Documents what daemon already automates
- **Investigation:** `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md` - Analyzed orchestrator value vs daemon gaps
- **Investigation:** `.kb/investigations/2026-01-09-inv-add-proactive-triage-workflow-orchestrators.md` - Designed triage discipline improvement
- **Investigation:** `archived/2026-01-10-inv-identify-orchestrator-value-add-vs.md` - Prior attempt at same question (never completed)

---

## Investigation History

**[2026-01-17 10:44]:** Investigation started
- Initial question: When does orchestrator judgment actually matter vs when is it just routing?
- Context: Part of Epic: Model & System Efficiency (orch-go-4tven) to inform daemon autonomy expansion and orchestrator focus areas

**[2026-01-17 11:05]:** Found prior investigation (archived) from 2026-01-10 on same topic
- Never completed (still in template form)
- Confirmed this is genuinely new work

**[2026-01-17 11:15]:** Read Strategic Orchestrator Model decision
- Major finding: Orchestrator role is comprehension, not coordination
- Coordination is daemon's job (already automated)
- Reframes question: Are orchestrators doing comprehension (high value) or dispatch (low value)?

**[2026-01-17 11:25]:** Read daemon model and verified current status
- Daemon already automates 80%: poll, infer skill, spawn, monitor completion, close
- Verified daemon running with 27 triage:ready issues queued
- Found spawn reliability issues in daemon log

**[2026-01-17 11:35]:** Read prior investigations on orchestrator value
- Interactive Orchestrators investigation: Found 3 legitimate functions (goal refinement, frame correction, synthesis)
- Manual spawns investigation: Found 26% daemon utilization vs 74% manual
- Proactive Triage investigation: Designed hygiene checkpoint for triage discipline

**[2026-01-17 11:45]:** Synthesis complete
- Answer: Orchestrator judgment matters for synthesis, goal refinement, frame correction, hotspot detection, triage decisions (the 20%)
- Routing is already automated by daemon (the 80%)
- "Routing overhead" is workflow debt (triage discipline gaps, spawn reliability issues), not necessary orchestrator function
