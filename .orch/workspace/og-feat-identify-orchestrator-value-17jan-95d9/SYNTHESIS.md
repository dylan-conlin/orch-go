# Session Synthesis

**Agent:** og-feat-identify-orchestrator-value-17jan-95d9
**Issue:** orch-go-4tven.6
**Duration:** 2026-01-17 10:44 → 2026-01-17 11:50
**Outcome:** success

---

## TLDR

Investigated when orchestrator judgment matters vs routing overhead. Found: orchestrator judgment matters for synthesis, goal refinement, frame correction, hotspot detection, and triage decisions (the 20%); routing execution is already automated by daemon (the 80%); "routing overhead" is workflow debt from triage discipline gaps and spawn reliability issues, not a necessary orchestrator function.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` - Complete investigation with 5 findings, synthesis, and recommendations
- `.orch/workspace/og-feat-identify-orchestrator-value-17jan-95d9/SYNTHESIS.md` - This synthesis document

### Files Modified
None (investigation only, no code changes)

### Commits
- Pending: investigation file and SYNTHESIS.md

---

## Evidence (What Was Observed)

**From Strategic Orchestrator Model decision (2026-01-07):**
- Orchestrator role defined as **comprehension**, not coordination
- Coordination is daemon's job (already automated)
- Work division: Investigation/Implementation → Worker | Synthesis/Understanding → Orchestrator | Coordination → Daemon

**From Daemon Autonomous Operation model:**
- Daemon fully automates poll-spawn-complete cycle (every 60s)
- Skill inference from issue type: bug→systematic-debugging, feature→feature-impl, task→investigation
- Capacity management via WorkerPool with reconciliation against OpenCode
- Cross-project operation across multiple repos

**From Interactive Orchestrators investigation (2026-01-06):**
- Found 3 legitimate orchestrator functions: goal refinement, frame correction, synthesis
- Daemon underutilization (26%) is SEPARATE from orchestrator value
- Interactive orchestrators are NOT compensation for daemon gaps

**From current system state:**
```bash
# Daemon running
launchctl list | grep orch
# Output: 42350	0	com.orch.daemon

# 27 issues ready for daemon spawning
bd list -l triage:ready --limit 0
# Output: 27

# Daemon log shows spawn failures
tail -50 ~/.orch/daemon.log
# Found: "Headless spawn failed: Failed to extract session ID"
```

**From spawn pattern analysis:**
- Prior investigation found 26% daemon utilization vs 74% manual spawns
- Manual spawns not because daemon can't route, but due to triage discipline gaps and spawn reliability issues
- Exception cases (design-session 100% manual, investigation 90% manual) inherently need orchestrator context

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` - Answers question about orchestrator value-add vs routing overhead

### Key Insights

1. **Orchestrators Aren't Meant to Route - That's Already Automated** - Strategic Orchestrator Model and Daemon Autonomous Operation show routing is fully automated. Orchestrators doing routing work is a symptom of workflow problems, not a necessary function.

2. **The 20% Requiring Judgment is Strategic Work, Not Dispatch** - Goal refinement, frame correction, synthesis, hotspot detection, epic readiness evaluation. These require cross-agent context, reasoning about patterns, and understanding that spans multiple investigations. Can't be automated.

3. **Triage is the Judgment Bottleneck, Not the Routing** - Orchestrator judgment happens at triage time: Is issue type correct? Is scope clear? Is this a hotspot area? Once labeled `triage:ready`, daemon handles everything. Goal is faster triage cycles, not smarter routing.

4. **Low Daemon Utilization ≠ Orchestrators Should Do Routing** - 74% manual spawns doesn't mean daemon can't handle routing. It means triage discipline is inconsistent, spawn reliability has issues, and some skills inherently need orchestrator context.

### Constraints Discovered
- Triage decisions (type correctness, scope clarity, hotspot detection) require judgment that daemon can't replicate
- Synthesis requires cross-agent context and reasoning - can't be delegated to spawned agents
- Some skills (design-session, investigation) inherently need orchestrator context for goal refinement

### Recommendations for Daemon Autonomy Expansion
1. **Fix spawn reliability first** - Investigate "Failed to extract session ID" errors (foundational: daemon must work reliably)
2. **Strengthen triage checkpoints** - Implement Proactive Hygiene Checkpoint design from 2026-01-09 investigation
3. **Add daemon utilization metric** - Track daemon spawn % vs manual spawn % to surface triage discipline gaps
4. **Document exception criteria** - Clarify when manual spawn is legitimate (urgent, complex, interactive) vs workflow workaround

### Recommendations for Orchestrator Focus Areas
- **Synthesis** - Combining findings from multiple agents into coherent understanding (core competency, not spawnable)
- **Triage judgment** - Type correctness, scope clarity, hotspot detection (can't be automated)
- **Goal refinement** - Converting Dylan's vague intent to actionable goals (requires conversation)
- **Frame correction** - Catching when orchestrator drops into tactical mode (requires external perspective)

---

## Next (What Should Happen)

**Recommendation:** close with follow-up issues

### If Close
- [x] All deliverables complete (investigation file written, SYNTHESIS.md created)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-4tven.6`

### Spawn Follow-up Issues

**Issue 1: Investigate daemon spawn reliability failures**
**Skill:** systematic-debugging
**Context:**
Daemon log shows "Headless spawn failed: Failed to extract session ID" errors. This prevents daemon from reliably spawning agents, forcing orchestrators to work around via manual spawn. Debug root cause and fix spawn reliability to enable daemon autonomy.

**Issue 2: Implement Proactive Hygiene Checkpoint in orchestrator skill**
**Skill:** feature-impl
**Context:**
Investigation 2026-01-09 designed Proactive Hygiene Checkpoint for triage discipline (session start trigger, multi-trigger guidance). Implementation needed in orchestrator skill to reduce triage:ready queue buildup and increase daemon utilization.

**Issue 3: Add daemon utilization metric and alerting**
**Skill:** feature-impl
**Context:**
Track daemon spawn % vs manual spawn % from events.jsonl. Surface metric in `orch stats` or dashboard. Add alert when daemon utilization drops below threshold (e.g., <40%) to catch triage discipline gaps early.

**Issue 4: Document manual spawn exception criteria**
**Skill:** investigation
**Context:**
Clarify when manual spawn is actually needed (urgent work, complex judgment, interactive synthesis with design-session) vs workflow workaround (daemon reliability issues, triage friction). Add guidance to orchestrator skill.

---

## Unexplored Questions

**Questions that emerged during this session:**
- What percentage of manual spawns are legitimate exceptions vs workflow workarounds? (Need spawn event analysis with reason codes)
- Can triage type-checking be partially automated? (Heuristics: "fix", "bug" in title → probably bug type)
- How much time do orchestrators actually spend on triage vs synthesis? (Time tracking would inform priority)
- What synthesis opportunities are missed due to triage overhead? (Qualitative: ask Dylan)
- What other daemon spawn failure modes exist beyond "Failed to extract session ID"?

**Areas worth exploring further:**
- Fuzzy matching for triage deduplication (bd duplicates only catches exact matches)
- Triage forcing functions if guidance alone doesn't improve discipline (gates, reminders)
- Epic consolidation heuristics (how to identify orphan issues that should be epic children)
- Near-duplicate issue detection at creation time (prevent triage burden)

---

## Session Metadata

**Skill:** feature-impl
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-feat-identify-orchestrator-value-17jan-95d9/`
**Investigation:** `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md`
**Beads:** `bd show orch-go-4tven.6`
