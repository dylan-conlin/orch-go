# Session Synthesis

**Agent:** og-feat-identify-orchestrator-value-17jan-e0b7
**Issue:** orch-go-4tven.6
**Duration:** 2026-01-17 (resumed completed investigation)
**Outcome:** success

---

## TLDR

Investigation complete: Orchestrator judgment matters for synthesis, goal refinement, frame correction, hotspot detection, and triage decisions (the 20%); routing execution is already automated by daemon (the 80%); "routing overhead" is workflow debt from triage discipline gaps and spawn reliability issues, not necessary orchestrator function.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` - Complete investigation analyzing orchestrator value-add vs routing overhead (already existed from prior agent session)

### Files Modified
- None (investigation was already complete)

### Commits
- None required (investigation already committed)

---

## Evidence (What Was Observed)

### Investigation Findings (5 Key Discoveries)

**Finding 1: Strategic Orchestrator Model Redefines Division of Labor**
- Decision document (2026-01-07) establishes orchestrator role as **comprehension**, not coordination
- Coordination is daemon's job (already automated)
- Work division: Investigation/Implementation → Workers | Synthesis/Understanding → Orchestrators | Coordination → Daemon
- Source: `.kb/decisions/2026-01-07-strategic-orchestrator-model.md`

**Finding 2: High-Value Activities Require Strategic Judgment**
- Three irreplaceable orchestrator functions: goal refinement, frame correction, synthesis
- Additional strategic activities: hotspot detection, issue type correction, follow-up extraction, epic readiness evaluation
- These are categorically different from queue processing - require reasoning and cross-agent context
- Source: `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md`

**Finding 3: Daemon Already Automates Routing**
- Poll-spawn-complete cycle runs every 60s: poll beads → filter triage:ready → infer skill from type → spawn → monitor completion → close
- Skill inference mapping: bug→systematic-debugging, feature→feature-impl, task→investigation, epic→architect
- Current status: Daemon running (PID 42350), 27 triage:ready issues queued
- Source: `.kb/models/daemon-autonomous-operation.md`, daemon source code

**Finding 4: Triage is the Judgment Bottleneck**
- Triage labels control daemon autonomy: triage:ready (daemon spawns immediately), triage:review (orchestrator judgment needed), no label (daemon skips)
- Orchestrator judges: type correctness, scope clarity, hotspot detection, dependency blocking
- Investigation (2026-01-09) found: "Triage requires judgment (can't be command-driven automation)"
- Current gap: 27 triage:ready issues queued, but orchestrator skill says "triage is part of hygiene checkpoint" (discipline inconsistent)
- Source: Orchestrator skill "Triage Protocol", `.kb/investigations/2026-01-09-inv-add-proactive-triage-workflow-orchestrators.md`

**Finding 5: Daemon Utilization Gap is Separate from Orchestrator Value**
- Prior investigation found 26% daemon utilization (74% manual spawns)
- But: Interactive orchestrators serve legitimate functions (goal refinement, frame correction, synthesis)
- Underutilization causes: triage discipline gaps, bypass-triage friction, spawn failures, exception cases
- Daemon log shows "Failed to extract session ID" errors (spawn reliability issues)
- Source: `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md`, daemon logs

### Tests Run
```bash
# Verified daemon running
launchctl list | grep orch
# Result: com.orch.daemon running (PID 42350)

# Checked triage:ready queue depth
bd list -l triage:ready --limit 0
# Result: 27 issues available for daemon spawning

# Checked daemon logs for errors
tail -50 ~/.orch/daemon.log
# Found: "Failed to extract session ID" spawn reliability issues
```

---

## Knowledge (What Was Learned)

### Key Insights from Investigation

1. **Orchestrators Aren't Meant to Route - That's Already Automated**
   - Strategic Orchestrator Model and Daemon Autonomous Operation show routing is fully automated
   - Orchestrators doing routing work is a SYMPTOM of workflow problems, not a necessary function

2. **The 20% Requiring Judgment is Strategic Work, Not Dispatch**
   - Goal refinement, frame correction, synthesis, hotspot detection, epic readiness evaluation
   - These aren't "routing with extra steps" - fundamentally different from queue processing
   - Require cross-agent context, reasoning about patterns, understanding spanning multiple investigations

3. **Triage is the Judgment Bottleneck, Not the Routing**
   - Orchestrator judgment happens at triage: Is type correct? Is scope clear? Is this a hotspot? Are deps resolved?
   - Once labeled triage:ready, daemon handles everything
   - Goal is **faster triage cycles**, not **smarter routing**

4. **Low Daemon Utilization ≠ Orchestrators Should Do Routing**
   - 74% manual spawns doesn't mean daemon can't handle routing
   - Means: triage discipline inconsistent, spawn reliability has issues, some skills inherently need orchestrator context
   - Fix: workflow friction, not routing automation

### Answer to Investigation Question

**Orchestrator judgment matters for:** synthesis, goal refinement, frame correction, hotspot detection, and triage decisions (the 20%)

**Routing execution:** already automated by daemon (the 80%)

**"Routing overhead":** workflow debt, not necessary orchestrator function
- Triage discipline gaps → issues not labeled triage:ready systematically
- Spawn reliability issues → daemon experiencing failures, orchestrators work around via manual spawn
- Exception cases treated as defaults → skills needing orchestrator context used for standard work

### Constraints Discovered
- Triage judgment cannot be automated (requires type correctness evaluation, hotspot detection, scope clarity assessment)
- Synthesis cannot be delegated to spawned agents (requires cross-agent context and strategic comprehension)
- Some skills inherently need orchestrator context (design-session, investigation with complex prompts)

### Externalized via `kb quick`
- Not applicable (investigation artifact itself serves as knowledge externalization)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with D.E.K.N. summary, 5 findings, synthesis, implementation recommendations)
- [x] Investigation file has `Status: Complete`, `Phase: Complete`
- [x] SYNTHESIS.md created in workspace
- [x] Ready for `orch complete orch-go-4tven.6`

### Follow-up Issues to Create (from Investigation Recommendations)

The investigation recommends creating 4 issues:

1. **Investigate spawn reliability "Failed to extract session ID" errors**
   - Type: bug
   - Context: Daemon log shows spawn failures preventing autonomous operation
   - Why: Foundational - daemon must work reliably before expecting orchestrators to use it

2. **Implement Proactive Hygiene Checkpoint in orchestrator skill**
   - Type: feature
   - Context: Investigation 2026-01-09 designed this, needs implementation in orchestrator skill
   - Why: Provides systematic triage triggers, strengthens triage discipline

3. **Add daemon utilization metric/alert**
   - Type: feature
   - Context: Track daemon spawn % vs manual spawn % to surface when triage discipline slips
   - Why: Quick win - makes workflow friction visible

4. **Document manual spawn exception criteria**
   - Type: task
   - Context: When is manual spawn actually needed vs workflow workaround?
   - Why: Clarifies when orchestrator dispatch judgment is legitimate

---

## Unexplored Questions

**From Investigation Structured Uncertainty section:**
- Whether fixing triage discipline would actually increase daemon utilization to target levels (hypothesis, not measured)
- Whether current spawn reliability issues are systematic or transient (single daemon log sample, not trend analysis)
- What percentage of manual spawns are workflow workarounds vs legitimate exceptions (claim based on investigation finding, not current measurement)
- Whether orchestrators actually follow proactive triage checkpoint guidance (behavioral, not verified)
- Whether faster triage cycles would reduce perceived "routing overhead" (proposed improvement, not tested)

**From Investigation "Areas needing further investigation" section:**
- What percentage of manual spawns are actually legitimate exceptions vs workflow workarounds? (Need spawn event analysis with reason codes)
- Can triage type-checking be partially automated? (Heuristics: "fix", "bug" in title → probably bug type)
- What spawn reliability issues exist beyond "Failed to extract session ID"? (Comprehensive daemon failure mode analysis)
- How much time do orchestrators actually spend on triage vs synthesis? (Time tracking would inform priority)
- What synthesis opportunities are missed due to triage overhead? (Qualitative: ask Dylan)

---

## Session Metadata

**Skill:** feature-impl
**Model:** claude-sonnet-4-5-20250514
**Workspace:** `.orch/workspace/og-feat-identify-orchestrator-value-17jan-e0b7/`
**Investigation:** `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md`
**Beads:** `bd show orch-go-4tven.6`
