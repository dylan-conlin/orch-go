<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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
**Phase:** Investigating
**Next Step:** Read all new investigations since Jan 6
**Status:** In Progress

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

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
