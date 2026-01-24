<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created 10 beads issues from recommend-yes investigations covering skill reduction, synthesis dedup, auth bypass, checkpoint reminders, spawn tracking, investigation promotion tooling, activity persistence, error handling, stalled detection, and screenshot storage.

**Evidence:** Found 11 recommend-yes investigations via grep, read D.E.K.N. summaries to extract recommendations, created issues: orch-go-0iped, orch-go-qu8fj, orch-go-6wxxt, orch-go-b4z4x, orch-go-wq3mz, orch-go-r5l6a, orch-go-v5zow, orch-go-mquh2, orch-go-zzo2z, orch-go-jtok4.

**Knowledge:** Recommend-yes investigations contain actionable architectural patterns worth tracking; investigation Next field provides clear issue descriptions; manual extraction process should be automated via kb reflect tooling.

**Next:** Close this task - all 10 beads issues created with proper context from investigation findings.

**Promote to Decision:** recommend-no (tactical task completion, not architectural)

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

# Investigation: Create Beads Issues 10 Investigations

**Question:** What beads issues should be created for the ~10 investigations flagged with recommend-yes?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** og-feat-create-beads-issues-18jan-622c worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: 11 investigations have recommend-yes flags

**Evidence:** Searched .kb/investigations/*.md with grep for "recommend-yes", found 11 investigations:
1. 2026-01-16-inv-analyze-orchestration-session-hit-context.md - skill size reduction
2. 2026-01-08-inv-synthesize-model-investigations-11-synthesis-work.md - synthesis dedup
3. 2026-01-09-inv-explore-opencode-github-issue-7410.md - Opus 4.5 auth
4. 2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md - checkpoint reminders
5. 2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md - spawn tracking
6. 2026-01-14-inv-meta-failure-decision-documentation-gap.md - investigation promotion
7. 2026-01-07-design-dashboard-activity-feed-persistence.md - activity persistence
8. 2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md - error handling
9. 2026-01-08-inv-design-stalled-agent-detection-agents.md - stalled detection
10. 2026-01-07-design-screenshot-artifact-storage-decision.md - screenshot storage
11. 2026-01-08-inv-kb-cli-fix-reflect-dedup.md - dedup error handling

**Source:** `find .kb/investigations -name "*.md" -type f -exec grep -l "recommend-yes" {} \;`

**Significance:** Each recommend-yes investigation contains actionable architectural patterns worth preserving via beads issues.

---

### Finding 2: D.E.K.N. Next field provides clear issue context

**Evidence:** Each investigation's D.E.K.N. summary contains **Next:** field with specific recommendations:
- "Implement skill size reduction (core vs full variants)"
- "Fix dedup to return true on error"
- "Add kb reflect --type investigation-promotion"

These map directly to beads issue titles and descriptions.

**Source:** Read D.E.K.N. summaries from first 20 lines of each investigation file

**Significance:** Investigation template structure makes issue creation straightforward - Next field becomes issue title, Evidence/Knowledge become issue description.

---

### Finding 3: Created 10 beads issues from recommendations

**Evidence:** Used `bd create` to create 10 issues with proper type (feature/bug), titles from Next field, descriptions including investigation path, context, recommendation, and evidence:

- orch-go-0iped: Implement skill size reduction and compact output flags
- orch-go-qu8fj: Fix synthesis completion recognition to prevent false spawns
- orch-go-6wxxt: Update opencode fork plugin for Opus 4.5 auth bypass  
- orch-go-b4z4x: Add automated checkpoint reminders for orchestrator sessions
- orch-go-wq3mz: Implement status-based spawn dedup to prevent duplicates
- orch-go-r5l6a: Add kb reflect investigation-promotion tooling
- orch-go-v5zow: Implement hybrid SSE + API architecture for activity feed
- orch-go-mquh2: Fix dedup error handling in synthesis issue checks
- orch-go-zzo2z: Implement stalled agent detection UI
- orch-go-jtok4: Implement workspace-scoped screenshot storage

**Source:** `bd create` commands with --type and --description flags

**Significance:** All 10 issues created successfully, now available in beads backlog for daemon or manual spawn.

---

## Synthesis

**Key Insights:**

1. **Investigation template enables systematic issue creation** - The D.E.K.N. structure (especially Next field) makes it straightforward to convert recommend-yes investigations into actionable beads issues. No ambiguity about what to implement.

2. **Architectural patterns cluster into domains** - The 10 issues fall into clear domains: context management (skill size, compact output), reliability (synthesis dedup, spawn tracking, error handling), observability (stalled detection, activity persistence), and infrastructure (auth bypass, screenshot storage, investigation tooling).

3. **Manual extraction should be automated** - This task (finding recommend-yes investigations, reading Next fields, creating issues) is exactly what kb reflect --type investigation-promotion should do. The pattern exists but needs tooling support.

**Answer to Investigation Question:**

Created 10 beads issues from recommend-yes investigations covering:
- Context management: Skill size reduction (orch-go-0iped)
- Reliability: Synthesis dedup (orch-go-qu8fj), spawn tracking (orch-go-wq3mz), error handling (orch-go-mquh2)
- Observability: Stalled detection (orch-go-zzo2z), activity persistence (orch-go-v5zow)
- Infrastructure: Opus auth (orch-go-6wxxt), checkpoint reminders (orch-go-b4z4x), investigation tooling (orch-go-r5l6a), screenshot storage (orch-go-jtok4)

Each issue includes investigation path, context, recommendation, and evidence from original investigation. All issues created with proper type (feature/bug) and ready for daemon or manual spawn.

---

## References

**Files Examined:**
- .kb/investigations/2026-01-16-inv-analyze-orchestration-session-hit-context.md - Skill size reduction recommendation
- .kb/investigations/2026-01-08-inv-synthesize-model-investigations-11-synthesis-work.md - Synthesis dedup issue
- .kb/investigations/2026-01-09-inv-explore-opencode-github-issue-7410.md - Opus 4.5 auth bypass
- .kb/investigations/2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md - Checkpoint discipline
- .kb/investigations/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md - Spawn tracking fix
- .kb/investigations/2026-01-14-inv-meta-failure-decision-documentation-gap.md - Investigation promotion tooling
- .kb/investigations/2026-01-07-design-dashboard-activity-feed-persistence.md - Activity feed architecture
- .kb/investigations/2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md - Dedup error handling
- .kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md - Stalled agent detection
- .kb/investigations/2026-01-07-design-screenshot-artifact-storage-decision.md - Screenshot storage

**Commands Run:**
```bash
# Find all investigations with recommend-yes flag
find .kb/investigations -name "*.md" -type f | xargs grep -l "recommend-yes"

# Extract Next steps from investigations
head -20 <investigation> | grep "Next:"

# Create beads issues (x10)
bd create "title" --type feature --description "context"
```

**Related Artifacts:**
- **Beads Issues:** orch-go-0iped, orch-go-qu8fj, orch-go-6wxxt, orch-go-b4z4x, orch-go-wq3mz, orch-go-r5l6a, orch-go-v5zow, orch-go-mquh2, orch-go-zzo2z, orch-go-jtok4 - Created from this work
- **Investigation:** 2026-01-14-inv-meta-failure-decision-documentation-gap.md - Recommends automating this process via kb reflect

---

## Investigation History

**2026-01-18 11:00:** Investigation started
- Initial question: What beads issues should be created for ~10 investigations flagged recommend-yes?
- Context: Task from orch-go-ere0l to create beads issues for recommend-yes investigations

**2026-01-18 11:15:** Found 11 recommend-yes investigations
- Searched via grep, read D.E.K.N. summaries
- Extracted Next field for each investigation

**2026-01-18 11:30:** Created 10 beads issues
- Used bd create with proper type, title, and description
- All issues created successfully

**2026-01-18 11:45:** Investigation completed
- Status: Complete
- Key outcome: 10 beads issues created from recommend-yes investigations, ready for daemon or manual spawn
