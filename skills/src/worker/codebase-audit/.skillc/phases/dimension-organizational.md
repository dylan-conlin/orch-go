# Codebase Audit: Organizational Drift

**TLDR:** Systematic investigation of organizational drift - ROADMAP hygiene, artifact coherence, template consistency, process adherence. Produces prioritized recommendations with system amnesia root cause analysis.

**When to use:** Dylan says "audit organizational drift", "check ROADMAP hygiene", "find documentation drift", or when you suspect accumulated organizational debt.

**Output:** Investigation file with drift patterns, evidence, system amnesia analysis, and actionable fixes.

---

## Quick Reference

### Focus Areas

1. **ROADMAP Drift** - Completed work marked TODO, missing tasks, stale priorities
2. **Documentation Drift** - Reference docs vs operational templates out of sync
3. **Template Drift** - Workspace templates vs actual workspaces inconsistent
4. **State Duplication** - Same info in multiple places falling out of sync
5. **Context Boundary Leaks** - Manual sync points across contexts (code ↔ docs ↔ tracking)

### Process (4 Phases)

1. **Pattern Search** (15-30 min) - Use automated tools to find drift candidates
2. **Evidence Collection** (30-60 min) - Validate patterns, gather concrete examples
3. **System Amnesia Analysis** (15-30 min) - Identify which coherence principles violated
4. **Documentation** (30 min) - Write investigation with recommendations and fixes

### Key Deliverable

Investigation file at `.kb/investigations/YYYY-MM-DD-audit-organizational-drift.md` with:
- **Status:** Complete
- **Root Cause:** Drift patterns with system amnesia analysis
- **Recommendations:** Prioritized fixes (forcing functions, automation, validation)

---

## Detailed Workflow

### Phase 1: Pattern Search (15-30 minutes)

**Use automated tools to find drift candidates:**

#### ROADMAP Drift Patterns

```bash
# Compare ROADMAP entries against recent git commits
cd ~/meta-orchestration
git log --oneline --since="30 days ago" | rg "feat:|fix:" | head -20
# Manually compare against docs/ROADMAP.org TODO items

# Find DONE items without completion metadata
rg "^\*\* DONE" docs/ROADMAP.org -A 5 | rg -v "CLOSED:|:Completed:"

# Find completed agents not in ROADMAP
orch history | rg "Completed" | head -10
# Check if these appear in ROADMAP
```

#### Template Drift Patterns

```bash
# Find workspaces missing new template fields
rg "^# Workspace:" .orch/workspace/ -l | while read ws; do
  grep -q "Session Scope" "$ws" || echo "MISSING SESSION SCOPE: $ws"
  grep -q "Checkpoint Strategy" "$ws" || echo "MISSING CHECKPOINT STRATEGY: $ws"
done

# Compare workspace template against actual workspaces
diff -u ~/.orch/templates/workspace/WORKSPACE.md \
        .orch/workspace/latest-workspace/WORKSPACE.md | head -50
```

#### Documentation Drift Patterns

```bash
# Find orch commands in code but not in operational templates
rg "def (spawn|check|status|complete|resume|send)" tools/orch/cli.py -o | \
  cut -d' ' -f2 | while read cmd; do
    grep -q "$cmd" ~/.orch/templates/orchestrator/orch-commands.md || \
      echo "MISSING IN TEMPLATE: $cmd"
  done

# Find features documented but not in reference docs
rg "orch \w+" ~/.orch/templates/orchestrator/ -o | sort -u > /tmp/template_cmds
rg "^###? orch" tools/README.md -o | sort -u > /tmp/readme_cmds
comm -23 /tmp/template_cmds /tmp/readme_cmds
```

#### Manual Sync Points (Fragile Patterns)

```bash
# Find "remember to" or "don't forget" instructions
rg "remember to|don't forget|make sure to update" docs/ --type md -i

# Find TODO comments about updating related files
rg "TODO.*update|FIXME.*sync" --type py --type md -C 2
```

#### State Duplication

```bash
# Find status/phase duplicated across systems
rg "status.*=.*(active|completed|paused)" --type py -l | \
  xargs rg "Phase.*=.*(Active|Complete|Paused)" -l

# Find completion timestamps in multiple places
rg "completed_at|completion_time|finished_at" --type py --type json
```

**Document all search commands in investigation file** (reproducibility)

---

### Phase 2: Evidence Collection (30-60 minutes)

**For each pattern found, gather concrete evidence:**

#### Evidence Standards

**For ROADMAP drift:**
- Specific ROADMAP entry + corresponding git commit showing drift
- Date completed vs date still showing as TODO
- Count of drift instances (how pervasive?)
- User impact (does this affect planning/prioritization?)

**For documentation drift:**
- Specific inaccuracy (what docs say vs what code does)
- File paths showing divergence
- When drift introduced (git blame to find when docs last updated)
- Impact (who's affected by stale docs - orchestrators, developers, both?)

**For template drift:**
- Specific workspace missing field + template showing field should exist
- Date workspace created vs date template updated
- Migration effort (how many workspaces need updating?)
- Graceful degradation check (does code handle missing fields?)

**For state duplication:**
- Concrete example showing same state in multiple files
- Which is source of truth? (or neither?)
- Instances where states diverged
- Proposed fix (derive, don't duplicate)

**For manual sync points:**
- Specific "remember to" instruction in docs
- Evidence of sync failures (times this was forgotten)
- Automation opportunity (can this be enforced?)

#### Investigation File Structure

```markdown
# Investigation: Organizational Drift Audit

**Date:** YYYY-MM-DD
**Status:** Complete
**Investigator:** Claude (codebase-audit-organizational skill)
**Trigger:** [Dylan's request or suspected drift]

---

## TLDR

**Key findings:** [2-3 sentence summary of major drift patterns]
**Highest priority:** [Top recommendation with ROI]
**Total drift instances:** [Count across all categories]

---

## Scope

**Focus areas:** Organizational drift (ROADMAP, docs, templates, state duplication)
**Boundaries:** [Project-specific or global artifacts?]
**Time spent:** [Actual time for audit]

---

## Findings by Category

### ROADMAP Drift (Priority: High/Medium/Low)

**Pattern:** [Name of drift pattern found]

**Evidence:**
- Instance 1: ROADMAP entry "Task X" marked TODO, git commit abc123 completed 2025-11-10
- Instance 2: [...]
- Total instances: [count]

**Metrics:**
- Tasks completed but not marked DONE: [count]
- Tasks missing completion metadata: [count]
- Average drift age: [days between completion and discovery]

**Impact:** [How this affects planning/orchestration]

**Recommendation:** [Specific fix with automation approach]

**ROI:** [Value gained / time invested]

---

### [Other categories following same structure]

---

## System Amnesia Analysis

**See:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`

**Coherence principles violated:**
- [ ] Single Source of Truth - [Example showing duplication]
- [ ] Automatic Loop Closure - [Example showing manual step]
- [ ] Cross-Boundary Coherence - [Example showing context switch failure]
- [ ] Observable Drift - [Example showing silent drift]
- [ ] Forcing Functions at Creation - [Example showing optional field]

**Common failures observed:**
- [ ] ROADMAP Drift - [X instances, root cause: manual ROADMAP updates]
- [ ] Documentation Drift - [X instances, root cause: template not rebuilt]
- [ ] Template Drift - [X instances, root cause: no migration mechanism]
- [ ] State Duplication - [X instances, root cause: derived state manual]
- [ ] Context Boundary Leaks - [X instances, root cause: no cross-project search]

**Design pattern recommendations:**
- Use "Derive, Don't Duplicate" for [specific case - e.g., registry status from workspace Phase]
- Add "Validation at Boundaries" for [specific workflow - e.g., orch complete checks Phase]
- Implement "Build Systems for Consistency" for [specific docs - e.g., template rebuild automation]
- Add "Forcing Functions" for [specific creation - e.g., ROADMAP requires task-id]

---

## Prioritization

**High Priority (fix now):**
1. [Issue] - Blocking orchestration, high impact, low effort
2. [Issue] - Data loss risk, silent failures

**Medium Priority (schedule soon):**
1. [Issue] - Maintenance burden, moderate effort
2. [Issue] - Developer experience impact

**Low Priority (backlog):**
1. [Issue] - Minor improvement, can defer
2. [Issue] - Nice-to-have, low impact

---

## Recommendations

**Immediate actions (this week):**
- [ ] [Specific task with owner and approach]
  - **Fix:** [What to do]
  - **Automation:** [How to prevent recurrence]
  - **Effort:** [Hours estimated]

**Short-term (this month):**
- [ ] [Planned fix with scope]

**Long-term (next quarter):**
- [ ] [Strategic improvement with ROI]

---

## Reproducibility

**Commands used for pattern search:**
```bash
# Document all grep/rg/find/diff commands used
# This allows re-running audit in future to measure improvement
```

**Metrics baseline:**
- Total ROADMAP entries: [count]
- ROADMAP drift instances: [count]
- Template drift instances: [count]
- Documentation drift instances: [count]
- State duplication instances: [count]
- Manual sync points: [count]

**Re-audit schedule:** 3 months (measure drift reduction)

---

## Related Work

- Decision: `.kb/decisions/2025-11-15-system-amnesia-as-design-constraint.md`
- Checklist: `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- Investigation: [Link to related organizational investigations]

---

## Next Steps

1. **Discuss findings with Dylan** (present prioritization, get approval)
2. **Add high-priority items to ROADMAP** (with effort estimates)
3. **Spawn agents for fixes** (if Dylan approves immediate action)
4. **Schedule re-audit** (3 months to measure improvement)
```

---

### Phase 3: System Amnesia Analysis (15-30 minutes)

**Identify which coherence principles were violated for each drift pattern:**

**Checklist for each finding:**

1. **Single Source of Truth** - Is there duplicate state? Which is authoritative?
2. **Automatic Loop Closure** - Does workflow require manual step to complete?
3. **Cross-Boundary Coherence** - Do updates span contexts (code/docs/tracking)?
4. **Observable Drift** - Was drift silent until manual inspection?
5. **Forcing Functions at Creation** - Could invalid state be created?

**For each violation, propose design pattern:**

| Violation | Pattern | Example Fix |
|-----------|---------|-------------|
| Duplicate state | Derive, Don't Duplicate | Registry status derived from workspace Phase |
| Manual loop closure | Atomic Multi-Context Updates | `orch complete` updates all systems |
| Silent drift | Validation at Boundaries | `orch complete` checks workspace Phase |
| No forcing function | Build Systems for Consistency | Template rebuild on SessionStart hook |

**Root cause categories:**
- **Return trip tax** - Easy to create, hard to remember to update
- **Context switching** - Update happens in different session/context
- **No single source of truth** - Multiple systems maintain same state
- **Manual sync points** - "Remember to" instructions
- **No observability** - Drift accumulates silently

---

### Phase 4: Documentation (30 minutes)

**Write investigation file following template above**

**Critical sections:**
- ✅ TLDR with key findings and top priority
- ✅ Evidence section with concrete examples (file paths, commit shas, counts)
- ✅ System Amnesia Analysis (which principles violated, proposed fixes)
- ✅ Prioritization using ROI framework (impact vs effort)
- ✅ Recommendations with specific, actionable tasks
- ✅ Reproducibility section with commands and baseline metrics

**Present findings to Dylan:**
- "Organizational drift audit complete. Key findings: [TLDR]"
- "Highest priority: [Top item with ROI]"
- "System amnesia root causes: [Top 2-3 principles violated]"
- "Would you like me to add high-priority items to ROADMAP or spawn agents to address them?"

---

## Anti-Patterns to Avoid

**❌ Audit without concrete examples**
- "ROADMAP has drift issues" (vague, not actionable)
✅ **Fix:** "12 tasks completed but marked TODO: Task X (commit abc123, completed 2025-11-10), Task Y (commit def456, completed 2025-11-09), ..."

**❌ No system amnesia analysis**
- Lists drift but doesn't identify root cause or prevention
✅ **Fix:** Map each finding to violated coherence principle, propose forcing function

**❌ No reproducibility**
- Can't re-run audit to measure improvement
✅ **Fix:** Document all commands + baseline metrics

**❌ Recommendations too vague**
- "Fix ROADMAP drift" (what does that mean?)
✅ **Fix:** "Add `orch complete` auto-update: read workspace task-id field, mark ROADMAP entry DONE"

**❌ No prioritization**
- Dylan doesn't know what to fix first
✅ **Fix:** Use ROI framework (impact vs effort matrix)

---

## Related Documentation

- **System amnesia patterns:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- **Investigation template:** `.orch/templates/INVESTIGATION.md`
- **ROADMAP management:** `docs/work-prioritization.md`
- **Template build system:** `.kb/decisions/2025-11-14-orchestrator-restructuring-template-build-system.md`

---

## Example Usage

**Dylan:** "audit organizational drift in meta-orchestration"

**You:**
1. Create investigation file: `.kb/investigations/2025-11-15-organizational-drift-audit.md`
2. Run pattern search commands (ROADMAP drift, template drift, docs drift)
3. Collect evidence (12 ROADMAP drift instances, 5 template drift instances, 3 doc drift instances)
4. System amnesia analysis (violated: Automatic Loop Closure, Observable Drift)
5. Prioritize using ROI framework
6. Write investigation file with recommendations
7. Present: "Audit complete. Found 20 drift instances across 3 categories. Highest priority: Fix `orch complete` to auto-update ROADMAP (violates Automatic Loop Closure - easy fix, high impact). Add to ROADMAP?"

---

*This skill enables systematic, evidence-based organizational drift assessment with system amnesia root cause analysis and actionable recommendations.*
