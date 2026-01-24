---
linked_issues:
  - orch-go-qv8cc
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Synthesized 12 new orchestrator investigations (Jan 6-7, 2026) into the session management guide, identifying 5 major new themes: checkpoint discipline, frame collapse detection, stats correlation, dashboard context-following, and interactive orchestrator value.

**Evidence:** Read all 12 investigations, extracted 9 findings, updated guide with new sections for checkpoint thresholds (2h/3h/4h), frame collapse multi-layer detection, coordination skill clarification, and dashboard troubleshooting.

**Knowledge:** Interactive orchestrators are NOT compensation for daemon gaps - they serve goal refinement, synthesis, and frame correction that daemon cannot replicate. Session infrastructure has matured from experimental to operational with proper status updates and checkpoint warnings.

**Next:** Close - guide updated. Follow-up items: workspace-based stats correlation, FindRecentSession title matching fix, coordination skills display separation.

**Promote to Decision:** recommend-no (synthesis consolidates existing decisions, doesn't establish new architectural patterns)

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

# Investigation: Synthesize Orchestrator Investigations

**Question:** What new patterns, findings, and insights have emerged from the 19 orchestrator investigations since the Jan 6, 2026 synthesis, and how should the orchestrator session management guide be updated?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** `.kb/investigations/2026-01-06-inv-synthesize-orchestrator-investigations-28-synthesis.md` (builds on, doesn't replace)
**Superseded-By:** N/A

---

## Findings

### Finding 1: New Theme - Dashboard Context Following

**Evidence:** Two investigations (2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md, 2026-01-07-inv-implement-follow-orchestrator-dashboard-filtering.md) established how dashboard should follow orchestrator's current project context:
- API needs `project_dir` parameter to query correct project's beads
- Cache must be per-project (keyed by directory)
- Frontend passes project_dir from orchestrator context to beads fetch

**Source:** `.kb/investigations/2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md`

**Significance:** Dashboard now supports multi-project orchestration - critical for cross-project visibility.

---

### Finding 2: New Theme - Stats Correlation Bug for Orchestrators

**Evidence:** Two investigations (2026-01-07-design-orch-stats-miscounts-orchestrator-meta.md, 2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md) identified why orchestrator/meta-orchestrator show 0%/16.7% completion:
- Orchestrators are classified as `CoordinationSkill` BY DESIGN - they run until context exhaustion, not complete discrete tasks
- Orchestrator completions use `workspace` identifier, but stats only correlates via `beads_id`
- Fix: Add workspace-based correlation for orchestrator completions

**Source:** `.kb/investigations/2026-01-07-design-orch-stats-miscounts-orchestrator-meta.md`

**Significance:** The low completion rate is NOT a bug - it's by design. Orchestrators are coordination sessions, not tasks.

---

### Finding 3: New Theme - Checkpoint Discipline

**Evidence:** Investigation 2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md implemented session duration warnings:
- 2h = warning, 3h = strong warning, 4h = exceeded threshold
- Visual warnings in `orch session status`
- Graduated urgency matches gradual context degradation

**Source:** `.kb/investigations/2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md`

**Significance:** Addresses the "5h session with partial outcome" problem. Duration-based thresholds are practical proxy for context exhaustion.

---

### Finding 4: New Theme - Frame Collapse Detection

**Evidence:** Investigation 2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md analyzed detection approaches:
- Self-detection unreliable (agent doing worker work has already rationalized it)
- Multi-layer detection needed: skill guidance + OpenCode plugin + SESSION_HANDOFF.md analysis
- Key trigger: failure-to-implementation pattern (after agents fail, orchestrator tries to "just fix it")
- Recommended: Add "Frame Collapse Check" section to SESSION_HANDOFF.md

**Source:** `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md`

**Significance:** External detection required - orchestrators can't see their own frame collapse.

---

### Finding 5: New Theme - Session Registry Status Updates

**Evidence:** Investigation 2026-01-06-inv-session-registry-doesnt-update-orchestrator.md fixed stale sessions:
- `orch complete` was removing sessions instead of updating status to "completed"
- `orch abandon` had NO registry update at all
- Fix: Use `registry.Update()` to set status, preserving history

**Source:** `.kb/investigations/2026-01-06-inv-session-registry-doesnt-update-orchestrator.md`

**Significance:** Session history preserved for tracking and debugging.

---

### Finding 6: New Theme - Interactive vs Spawned Orchestrator Workspaces

**Evidence:** Investigation 2026-01-06-inv-interactive-orchestrator-sessions-don-create.md found:
- Spawned orchestrators get full workspaces with SESSION_HANDOFF.md
- Interactive orchestrators only get `~/.orch/session.json` entry
- Fix: Enhance `orch session start` to create session workspace

**Source:** `.kb/investigations/2026-01-06-inv-interactive-orchestrator-sessions-don-create.md`

**Significance:** Interactive sessions lose context on exit without workspace. Two parallel session models exist.

---

### Finding 7: New Theme - Tmux Session ID Capture

**Evidence:** Investigation 2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md traced why tmux-spawned orchestrators don't capture .session_id:
- Tmux spawns used standalone OpenCode mode (embedded server)
- Fix: Switch to attach mode with `--dir` flag
- Additional issue: `FindRecentSession` matches by title, not workspace

**Source:** `.kb/investigations/2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md`

**Significance:** Session ID capture enables `orch attach` and resume workflows.

---

### Finding 8: Interactive Orchestrators Serve Legitimate Functions

**Evidence:** Investigation 2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md found:
- Daemon utilization is 26% (underutilized, but not broken)
- Interactive orchestrators serve 3 functions daemon CANNOT: (1) goal refinement, (2) real-time frame correction, (3) synthesis
- Interactive orchestrators are NOT compensation for daemon gaps

**Source:** `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md`

**Significance:** Interactive orchestrators and daemon are complementary, not competing. The "meta-orchestrator → Issues → Daemon → Workers" evolution misses the synthesis step.

---

### Finding 9: Skill Updates (Principles + Dashboard Troubleshooting)

**Evidence:** Two investigations documented skill improvements:
- 2026-01-07-inv-add-principles-quick-reference-orchestrator.md: Added Principles Quick Reference section (10 orchestrator-relevant principles)
- 2026-01-07-inv-update-orchestrator-skill-add-dashboard.md: Added Dashboard Troubleshooting protocol

**Source:** `.kb/investigations/2026-01-07-inv-add-principles-quick-reference-orchestrator.md`, `.kb/investigations/2026-01-07-inv-update-orchestrator-skill-add-dashboard.md`

**Significance:** Orchestrator skill evolved with operational improvements - principles for quick scanning, dashboard debugging flow.

---

## Synthesis

**Key Insights:**

1. **Session Infrastructure Matured** - The Jan 6-7 investigations show the orchestrator session system crossing from "experimental" to "operational": session registry now updates status properly (Finding 5), checkpoint discipline enforced via status output (Finding 3), and session ID capture for tmux spawns addressed (Finding 7).

2. **Frame Collapse Detection Remains External** - Multiple findings confirm that orchestrators can't self-detect frame collapse (Finding 4). The solution is multi-layer: skill guidance + handoff sections + plugin potential. The key insight is that detection must happen at BOUNDARIES (session end, handoff review).

3. **Interactive ≠ Compensation** - Finding 8 definitively answers whether interactive orchestrators are "workarounds for daemon gaps" - NO. They serve goal refinement, synthesis, and frame correction that daemon cannot replicate. Daemon underutilization (26%) is a separate issue from orchestrator value.

4. **Dashboard Context-Awareness Added** - Findings 1 and 9 show the dashboard now follows orchestrator's current project context. Per-project caching, API parameters, and reactive frontend combine to support multi-project orchestration.

5. **Orchestrators Are Coordination Sessions, Not Tasks** - Finding 2 reinforces that the 16.7% "completion rate" for orchestrators is BY DESIGN. They're classified as `CoordinationSkill` that runs until context exhaustion. Stats display should separate them from task skills.

**Answer to Investigation Question:**

Since the Jan 6 synthesis, 12 new orchestrator investigations have emerged with these key themes:
1. **Dashboard multi-project support** (context following, per-project cache)
2. **Session infrastructure fixes** (registry updates, workspace creation for interactive sessions)
3. **Checkpoint discipline** (2h/3h/4h thresholds with visual warnings)
4. **Frame collapse detection** (external detection required, multi-layer approach)
5. **Stats correlation** (workspace-based correlation for orchestrators)
6. **Interactive orchestrator value** (NOT compensation - serves synthesis, goal refinement)

The guide should be updated with:
- Checkpoint discipline thresholds and `orch session status` integration
- Frame collapse detection section (multi-layer approach, boundary detection)
- Dashboard troubleshooting quick reference
- Clarification that orchestrator low completion rates are BY DESIGN
- Interactive vs spawned orchestrator workspace differences

---

## Structured Uncertainty

**What's tested:**

- ✅ Read all 12 new orchestrator investigations from Jan 6-7, 2026 (verified: read each file)
- ✅ Guide updated with new sections for checkpoint discipline, frame collapse, dashboard, stats (verified: edit operations)
- ✅ Themes extracted from investigations match content (verified: findings reference specific files)

**What's untested:**

- ⚠️ Whether all 47 orchestrator investigations are now consolidated (only read Jan 6-7 batch, prior synthesis covered earlier)
- ⚠️ Whether the guide is comprehensive enough for all debugging scenarios (needs real-world usage)
- ⚠️ Whether new themes will hold up over time (patterns may evolve)

**What would change this:**

- Finding additional orchestrator investigations not yet synthesized
- Real-world debugging revealing gaps in the guide
- New orchestrator patterns emerging that contradict synthesized themes

---

## Implementation Recommendations

**Purpose:** The investigation deliverable IS the guide update. Additional implementation items identified.

### Recommended Approach ⭐

**Guide-first maintenance with periodic synthesis** - Future orchestrator investigations should update the guide as their primary artifact. Synthesis should happen after every 10-15 new investigations.

**Why this approach:**
- Single source of truth prevents duplicate investigations
- New learnings immediately benefit all agents
- Debugging checklist prevents unnecessary spawns

**Trade-offs accepted:**
- Guide may grow large over time (mitigated by clear sections)
- Needs periodic refresh (mitigated by "last verified" date)

**Implementation sequence:**
1. ✅ Guide updated with 12 new investigation findings
2. Future investigations reference and update the guide
3. Monthly synthesis cadence to consolidate new patterns

### Follow-up Items Identified

| Item | Priority | Source Investigation |
|------|----------|---------------------|
| Add workspace-based correlation to orch stats | Medium | 2026-01-07-design-orch-stats-miscounts-orchestrator-meta.md |
| Add --title to opencode attach or fix FindRecentSession | Medium | 2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md |
| Consider OpenCode plugin for frame collapse detection | Low | 2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md |
| Separate coordination skills from task skills in stats display | Medium | 2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md |

**Success criteria:**
- ✅ Guide updated with checkpoint discipline, frame collapse detection, stats clarification, dashboard context-following
- ✅ History section updated to reflect Jan 7 synthesis
- ✅ Future orchestrators can debug session issues using guide before spawning investigations

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md` - Dashboard context following
- `.kb/investigations/2026-01-07-inv-add-principles-quick-reference-orchestrator.md` - Skill update
- `.kb/investigations/2026-01-07-inv-update-orchestrator-skill-add-dashboard.md` - Skill update
- `.kb/investigations/2026-01-07-design-orch-stats-miscounts-orchestrator-meta.md` - Stats correlation bug
- `.kb/investigations/2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md` - Completion rate analysis
- `.kb/investigations/2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md` - Checkpoint discipline
- `.kb/investigations/2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md` - Session ID capture
- `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md` - Frame collapse detection
- `.kb/investigations/2026-01-06-inv-session-registry-doesnt-update-orchestrator.md` - Registry status fix
- `.kb/investigations/2026-01-06-inv-interactive-orchestrator-sessions-don-create.md` - Interactive workspace gap
- `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md` - Interactive orchestrator value
- `.kb/investigations/2026-01-06-inv-synthesize-orchestrator-investigations-28-synthesis.md` - Prior synthesis

**Commands Run:**
```bash
# Find all orchestrator investigations
glob .kb/investigations/*orchestrator*.md

# Create investigation file
kb create investigation synthesize-orchestrator-investigations
```

**Related Artifacts:**
- **Guide:** `.kb/guides/orchestrator-session-management.md` - Updated with synthesis findings
- **Prior Synthesis:** `.kb/investigations/2026-01-06-inv-synthesize-orchestrator-investigations-28-synthesis.md` - Previous consolidation

---

## Investigation History

**2026-01-07:** Investigation started
- Initial question: What new patterns emerged from Jan 6-7 orchestrator investigations?
- Context: 12 new investigations since prior synthesis

**2026-01-07:** Themes identified
- 9 key findings across: dashboard context, stats correlation, checkpoint discipline, frame collapse detection, session registry, interactive workspaces, skill updates

**2026-01-07:** Guide updated
- Added sections: Checkpoint Discipline, Dashboard Context Following, new Common Problems
- Updated Key Decisions with interactive orchestrator value, checkpoint discipline

**2026-01-07:** Investigation completed
- Status: Complete
- Key outcome: Guide updated with 12 new investigation findings; identified 4 follow-up implementation items
