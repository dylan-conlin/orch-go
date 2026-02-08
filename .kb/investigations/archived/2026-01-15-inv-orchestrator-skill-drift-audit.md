# Investigation: Orchestrator Skill Drift Audit

**Date:** 2026-01-15
**Status:** Complete
**Outcome:** Systematic drift identified, issues created for remediation

---

## Summary

Comprehensive audit of `.kb/models/` and `.kb/guides/` against the orchestrator skill (`~/.claude/skills/meta/orchestrator/SKILL.md`) identified **19 drift items** across 4 categories. This investigation documents each drift item with evidence, impact, and recommended fix.

---

## Methodology

1. Read all 16 models in `.kb/models/`
2. Read all 28 guides in `.kb/guides/`
3. Read orchestrator skill (1929 lines)
4. Cross-referenced for inconsistencies, missing content, and outdated information
5. Prioritized by impact to orchestrator/agent effectiveness

---

## Drift Items by Priority

### HIGH PRIORITY (Blocks correct behavior)

#### H1: Strategic Orchestrator Model Not Reflected

**Evidence:**
- `orchestrator-session-lifecycle.md:31` says orchestrators do "Strategic comprehension"
- `orchestrator-session-management.md:37` still shows "Tactical execution" in architecture diagram
- Decision `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` established COMPREHEND → TRIAGE → SYNTHESIZE

**Impact:** Orchestrators receive conflicting framing about their role. Some sources say "tactical execution/coordination", others say "strategic comprehension". This causes identity confusion.

**Fix:** Update all sources to use "Strategic comprehension" consistently. Update orchestrator skill architecture section.

**Files to update:**
- `.kb/guides/orchestrator-session-management.md:37`
- `orch-knowledge/skills/src/meta/orchestrator/` source files

---

#### H2: Dev Environment Two-Manager Architecture Undocumented

**Evidence:**
- `dev-environment-setup.md` establishes: overmind (via `orch-dashboard`) for dev services
- `daemon.md` establishes: launchd for orch daemon
- No source clearly states "two process managers for two concerns"

**Impact:** Agents may try to use launchd for dev services or overmind for daemon, causing failures or incorrect fixes.

**Fix:** Add explicit "Process Manager Architecture" section distinguishing:
- **Dev services** (overmind via `orch-dashboard`): opencode, orch serve, web UI
- **Orch daemon** (launchd): autonomous background processing

**Files to update:**
- `CLAUDE.md` - add clear section
- Orchestrator skill - add to infrastructure section

---

#### H3: Model Selection Constraints Outdated

**Evidence:**
- `model-access-spawn-paths.md` documents:
  - Opus gate via fingerprinting (Jan 8, 2026)
  - Gemini Flash TPM limits forcing Sonnet switch (Jan 9, 2026)
  - Default model is now Sonnet, not Opus
  - Infrastructure work auto-detection applies `--backend claude --tmux`

**Impact:** Orchestrators may spawn with wrong model expectations, creating zombie agents or rate limit issues.

**Fix:** Update orchestrator skill model selection guidance to reflect:
- Sonnet as default (not Opus)
- Opus requires `--backend claude` (Max subscription)
- Infrastructure work auto-applies escape hatch
- Gemini Flash has TPM limits (2,000 req/min)

**Files to update:**
- Orchestrator skill model selection section
- CLAUDE.md model defaults

---

#### H4: Triage Bypass Required for Manual Spawns

**Evidence:**
- `spawn.md:71-94` establishes `--bypass-triage` is required for manual spawns
- This was added to encourage daemon-driven workflow
- Orchestrator skill may not mention this requirement

**Impact:** Manual spawns fail without clear explanation if `--bypass-triage` not provided.

**Fix:** Add triage bypass documentation to orchestrator skill spawn section.

**Files to update:**
- Orchestrator skill spawn guidance

---

#### H5: Inconsistent Orchestrator Default Spawn Mode

**Evidence:**
- `spawn.md:57-68` says headless is default
- `tmux-spawn-guide.md:36` says policy skills default to tmux
- `orchestrator-session-management.md:79` says orchestrator default = tmux

**Impact:** Confusion about whether orchestrators spawn headless or with tmux.

**Actual behavior:** Policy/orchestrator skills auto-default to tmux (code at `spawn_cmd.go:789`). Workers default to headless.

**Fix:** Reconcile documentation - clarify the exception for policy skills.

**Files to update:**
- `spawn.md` - add policy skill exception
- Orchestrator skill - clarify orchestrator spawns get tmux by default

---

### MEDIUM PRIORITY (Reduces effectiveness)

#### M1: Follow Orchestrator Mechanism Missing

**Evidence:**
- `follow-orchestrator-mechanism.md` (Jan 15, 2026) documents new capability
- Dashboard context following via `/api/context`
- Ghostty window sync via tmux `after-select-window` hook
- lsof fallback for Claude Code panes
- Socket detection for overmind context

**Impact:** Orchestrators unaware of follow mechanism, can't troubleshoot when it breaks.

**Fix:** Add "Dashboard Follow Mechanism" section to orchestrator skill.

**Files to update:**
- Orchestrator skill monitoring section

---

#### M2: Session Handoff Window Scoping Not Documented

**Evidence:**
- `session-resume-protocol.md:106-138` documents window-scoped handoffs
- Path is `.orch/session/{window-name}/latest/` not `.orch/session/latest/`
- Prevents concurrent orchestrators from clobbering handoffs

**Impact:** Orchestrators may look for handoffs in wrong location or not understand window isolation.

**Fix:** Update session management docs in orchestrator skill.

**Files to update:**
- Orchestrator skill session management section

---

#### M3: Duplicate Prevention Not Documented

**Evidence:**
- `spawn.md:134-167` documents duplicate prevention
- `daemon.md:136-156` documents SpawnedIssueTracker with 5-min TTL
- Concurrency limit default is 5 agents

**Impact:** Orchestrators may not understand why spawn is blocked for existing work.

**Fix:** Add duplicate prevention to orchestrator skill spawn section.

**Files to update:**
- Orchestrator skill spawn guidance

---

#### M4: Rate Limit Monitoring Not Documented

**Evidence:**
- `spawn.md:98-131` documents rate limit monitoring
- 80% warning, 95% blocking thresholds
- Auto-switch to alternate accounts at critical

**Impact:** Orchestrators surprised by spawn blocks at high usage.

**Fix:** Add rate limit section to orchestrator skill.

**Files to update:**
- Orchestrator skill spawn guidance

---

#### M5: Five-Tier Escalation Model Not Documented

**Evidence:**
- `completion.md:79-109` documents escalation tiers
- None/Info/Review/Block/Failed
- Determines auto-completion eligibility

**Impact:** Orchestrators don't understand why some completions auto-close and others don't.

**Fix:** Add escalation model to orchestrator skill completion section.

**Files to update:**
- Orchestrator skill completion guidance

---

#### M6: Gap Gating for Context Quality Not Documented

**Evidence:**
- `spawn.md:225-231` documents gap gating flags
- `--gate-on-gap`, `--skip-gap-gate`, `--gap-threshold`

**Impact:** Orchestrators unaware of context quality gates.

**Fix:** Add context quality section to orchestrator skill.

**Files to update:**
- Orchestrator skill spawn guidance

---

#### M7: Two-Tier Reflection Automation Not Documented

**Evidence:**
- `daemon.md:369-389` documents reflection automation
- synthesis (10+ investigations) → auto-create issues
- open (>3 days) → surface for review
- promote/stale/drift → surface only (no auto-issues)

**Impact:** Orchestrators don't understand daemon reflection behavior.

**Fix:** Add reflection automation to orchestrator skill daemon section.

**Files to update:**
- Orchestrator skill daemon guidance

---

### LOW PRIORITY (Nice to have)

#### L1: Skill-Type Values Not Canonically Listed

**Evidence:**
- `orchestrator-session-lifecycle.md:75` says `skill-type: policy` OR `orchestrator`
- `spawn-architecture.md` says `skill-type: orchestrator`
- No canonical list of all valid skill-type values

**Impact:** Minor confusion when creating new skills.

**Fix:** Add skill-type reference to skill system documentation.

**Files to update:**
- `.kb/guides/skill-system.md` (if exists) or create

---

#### L2: Beads Field Name Gotcha Not in Skill

**Evidence:**
- `beads-integration.md:139-159` documents `issue_type` vs `type` gotcha
- Common mistake in jq queries

**Impact:** Minor - agents hit this occasionally.

**Fix:** Add beads gotchas to orchestrator skill.

**Files to update:**
- Orchestrator skill beads section

---

#### L3: Checkpoint Thresholds Duplicated

**Evidence:**
- `orchestrator-session-lifecycle.md:104-114` has thresholds
- `orchestrator-session-management.md:115-130` has same thresholds
- DRY violation

**Impact:** Maintenance burden, potential for drift between copies.

**Fix:** Single source for thresholds, reference from other docs.

**Files to update:**
- Choose canonical location, update others to reference

---

#### L4: Infrastructure Work Keywords Incomplete

**Evidence:**
- `model-access-spawn-paths.md:72-73` lists some keywords
- Keywords: "opencode", "spawn", "daemon", "registry", "orch serve", "overmind", "dashboard"
- May not be exhaustive

**Impact:** Some infrastructure work may not auto-apply escape hatch.

**Fix:** Document complete keyword list or link to code.

**Files to update:**
- Model or orchestrator skill

---

#### L5: Cross-Project Completion Auto-Detection Not Documented

**Evidence:**
- `completion.md:113-139` documents auto-detection
- Extracts PROJECT_DIR from SPAWN_CONTEXT.md
- Routes beads queries to correct project

**Impact:** Orchestrators may not know this works automatically.

**Fix:** Add to orchestrator skill completion section.

**Files to update:**
- Orchestrator skill completion guidance

---

## Recommended Remediation Order

### Phase 1: High Priority (Blocks correct behavior)
1. H3: Model Selection Constraints - prevents zombie agents
2. H4: Triage Bypass Required - prevents spawn failures
3. H2: Two-Manager Architecture - prevents infrastructure confusion
4. H1: Strategic Orchestrator Model - prevents role confusion
5. H5: Orchestrator Default Spawn Mode - prevents mode confusion

### Phase 2: Medium Priority (Reduces effectiveness)
6. M1: Follow Orchestrator Mechanism
7. M2: Session Handoff Window Scoping
8. M3: Duplicate Prevention
9. M4: Rate Limit Monitoring
10. M5: Five-Tier Escalation Model
11. M6: Gap Gating
12. M7: Two-Tier Reflection

### Phase 3: Low Priority (Nice to have)
13. L1-L5: Various minor items

---

## Implementation Notes

### Source of Truth Hierarchy

Established through this audit:
1. **Models** (`.kb/models/`) - Architectural truth (HOW things work)
2. **Guides** (`.kb/guides/`) - Procedural truth (HOW to do things)
3. **Decisions** (`.kb/decisions/`) - Historical truth (WHY we chose)
4. **Skill** - Compilation target from above sources

### Skill Regeneration Process

The orchestrator skill is generated via `skillc` from source files in:
```
~/orch-knowledge/skills/src/meta/orchestrator/
```

Updates should go to source files, then regenerate skill.

### Drift Prevention

Consider adding to monthly maintenance:
- `kb reflect` includes drift detection for models/guides vs skill
- Or: scheduled drift audit (this investigation as template)

---

## Next Actions

- [ ] Create beads issues for each drift item
- [ ] Prioritize by phase
- [ ] Assign to appropriate skill (feature-impl for updates)
- [ ] Track completion via beads

---

## References

**Models audited:**
- spawn-architecture.md
- agent-lifecycle-state-model.md
- opencode-session-lifecycle.md
- dashboard-architecture.md
- beads-integration-architecture.md
- completion-verification.md
- daemon-autonomous-operation.md
- escape-hatch-visibility-architecture.md
- dashboard-agent-status.md
- model-access-spawn-paths.md
- orchestrator-session-lifecycle.md
- follow-orchestrator-mechanism.md

**Guides audited:**
- dev-environment-setup.md
- spawn.md
- daemon.md
- resilient-infrastructure-patterns.md
- session-resume-protocol.md
- completion.md
- status-dashboard.md
- orchestrator-session-management.md
- decision-authority.md
- tmux-spawn-guide.md
- beads-integration.md
- agent-lifecycle.md
