<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Proactive hygiene workflow should be a session start checkpoint (Option A) with multi-trigger guidance, using existing bd commands (duplicates, stale, epic).

**Evidence:** bd CLI has all needed primitives (verified via --help), session start protocol already exists and is interactive (found in orchestrator skill), existing "triage" references are all reactive (12 grep matches, all about triage:ready/review labels).

**Knowledge:** Infrastructure exists, guidance missing - this is documentation problem not tooling problem; "triage" term is overloaded (reactive per-issue vs proactive hygiene).

**Next:** Implement by adding "Proactive Hygiene Checkpoint" section to orchestrator skill with workflow table and multi-trigger guidance.

**Promote to Decision:** recommend-no (tactical workflow documentation, not architectural pattern)

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

# Investigation: Add Proactive Triage Workflow Orchestrators

**Question:** Should proactive triage (dedup/consolidate/stale/reprioritize) be a session start checkpoint (Option A) or standalone command (Option B), and what workflow should orchestrators follow?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Worker Agent (og-feat-add-proactive-triage-09jan-2fb9)
**Phase:** Complete
**Next Step:** None - proceed to implementation
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Beads CLI already provides all hygiene primitives

**Evidence:** The bd CLI has commands for all proactive hygiene tasks:
- `bd duplicates` - Finds exact duplicates by content hash, can auto-merge with `--auto-merge`
- `bd stale --days N` - Shows issues not updated in N days (default 30)
- `bd epic status` - Shows epic completion progress
- `bd epic close-eligible` - Closes epics where all children complete
- `bd list` with rich filters - Can find orphan issues, priority mismatches, label-based grouping

**Source:** `bd --help`, `bd duplicates --help`, `bd stale --help`, `bd epic --help`

**Significance:** No new commands needed - the primitives exist. The gap is WHEN to run them (triggers) and HOW to combine them (workflow).

---

### Finding 2: Session start protocol is already interactive and guidance-driven

**Evidence:** The orchestrator skill has a "Session Start Protocol" section that guides orchestrators through an interactive workflow:
1. Surface fires (what's broken/blocking)
2. Surface nagging (what's on your mind)
3. Propose threads (suggest priorities)
4. Confirm focus

This is already a checkpoint pattern where orchestrator orients before diving into work.

**Source:** `~/.claude/skills/meta/orchestrator/SKILL.md` (Session Start Protocol section), beads issue orch-go-ngtyj Option A

**Significance:** Adding proactive triage as a checkpoint at session start fits the existing pattern. Session start is already when orchestrators orient and prioritize.

---

### Finding 3: Existing triage references are all reactive (issue-level)

**Evidence:** Searching for "triage" in orchestrator skill shows only references to `triage:ready` and `triage:review` labels, which are about reactive decisions on individual issues:
- "Should this issue be spawned?" (triage:ready)
- "Does this need review first?" (triage:review)

No guidance exists for proactive backlog hygiene (dedup, consolidate, close stale, reprioritize).

**Source:** `grep -n "triage" ~/.claude/skills/meta/orchestrator/SKILL.md` (12 matches, all about labels)

**Significance:** The term "triage" is overloaded - existing usage is reactive (per-issue spawning decisions), new workflow is proactive (cross-issue hygiene). Need clear naming to distinguish.

---

### Finding 4: Multiple trigger points needed, not just session start

**Evidence:** The beads issue orch-go-ngtyj identifies four triggers:
1. Session start (orient before diving in)
2. Before spawning batch work (avoid spawning dupes)
3. When backlog feels noisy (friction signal)
4. Weekly rhythm (prevent drift)

The prior investigation on capacity-aware labeling (2025-12-27-inv-capacity-utilization-workflow) also mentions checking for ready work before batch-labeling.

**Source:** Beads issue orch-go-ngtyj description, prior investigation 2025-12-27-inv-capacity-utilization-workflow-orchestrator-proactively.md

**Significance:** Session start is the PRIMARY trigger, but not the ONLY trigger. Need guidance for ongoing hygiene, not just session boundaries.

---

## Synthesis

**Key Insights:**

1. **Infrastructure exists, guidance missing** - All the primitives for proactive hygiene exist in bd CLI (duplicates, stale, epic management). The gap is systematic guidance on WHEN and HOW to use them. This is a documentation/workflow problem, not a tooling problem.

2. **Session start is natural primary trigger** - The existing session start protocol is already interactive and guidance-driven. Adding proactive hygiene as a checkpoint fits naturally: "Orient on priorities AND clean backlog before diving in."

3. **Need clear naming to avoid confusion** - "Triage" is overloaded (reactive per-issue vs proactive hygiene). Use "Proactive Hygiene Workflow" or "Backlog Health Checkpoint" to distinguish from existing triage:ready/review labeling.

4. **Multiple triggers, not just session boundaries** - Session start is primary but not exclusive. Orchestrators also need triggers for: before batch-labeling (avoid spawning dupes), when backlog feels noisy (friction signal), weekly rhythm (prevent drift).

**Answer to Investigation Question:**

Use **Option A (session start checkpoint)** as the PRIMARY trigger, with guidance for additional triggers. Specifically:

1. Add "Proactive Hygiene Checkpoint" section to orchestrator skill's Session Start Protocol
2. Document the workflow: duplicates → stale → epics → reprioritize
3. Include guidance for secondary triggers (before batch-labeling, weekly, friction-driven)
4. Use existing bd commands (no new CLI commands needed)

This approach:
- Leverages existing session start pattern (already interactive)
- Uses existing bd commands (duplicates, stale, epic)
- Provides systematic triggers (not just ad-hoc)
- Preserves orchestrator judgment (can't be automated)

**Why not Option B (standalone `orch triage` command):**
- Session start already exists and is natural trigger point
- Triage requires judgment (can't be command-driven automation)
- Adding command doesn't solve "when to run it" problem
- Orchestrators need guidance, not another tool

---

## Structured Uncertainty

**What's tested:**

- ✅ bd duplicates exists and can auto-merge (verified: `bd duplicates --help` shows --auto-merge flag)
- ✅ bd stale exists with configurable days threshold (verified: `bd stale --help` shows --days flag)
- ✅ bd epic commands exist for status and close-eligible (verified: `bd epic --help`)
- ✅ Session start protocol exists in orchestrator skill (verified: grep found "Session Start Protocol")

**What's untested:**

- ⚠️ Whether orchestrators will actually follow proactive hygiene checkpoint (behavioral)
- ⚠️ Whether the suggested workflow order (duplicates → stale → epics → reprioritize) is optimal
- ⚠️ Whether weekly trigger is frequent enough to prevent drift (may need data on backlog growth rate)
- ⚠️ Whether "bd duplicates" catches near-duplicates or only exact matches (content hash suggests exact only)

**What would change this:**

- If orchestrators skip hygiene checkpoint due to time pressure, need more forcing function (command gate?)
- If exact-match duplicates are rare but near-duplicates common, may need fuzzy matching tool
- If backlog grows >50 issues/week, weekly rhythm insufficient (need more frequent triggers)
- If session start adds too much friction, may need to make hygiene checkpoint optional/skippable

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Session Start Checkpoint + Multi-Trigger Guidance** - Add "Proactive Hygiene Checkpoint" to session start protocol with guidance for additional triggers throughout work sessions.

**Why this approach:**
- Leverages existing session start protocol (Finding 2 - already interactive)
- Uses existing bd commands (Finding 1 - primitives exist)
- Addresses multiple triggers (Finding 4 - not just session boundaries)
- Preserves judgment (hygiene needs human decision-making, can't be automated)
- Zero code changes needed (documentation-only change)

**Trade-offs accepted:**
- Relies on orchestrator discipline (no forcing function)
- May add friction to session start (but necessary for backlog health)
- Doesn't solve near-duplicate detection (bd duplicates is exact-match only)

**Implementation sequence:**
1. Add "Proactive Hygiene Checkpoint" section to orchestrator skill after Session Start Protocol
2. Document workflow: duplicates → stale → epic-cleanup → reprioritize
3. Add multi-trigger guidance table (session start, before batch-label, weekly, friction-driven)
4. Update Session Start Protocol section to reference hygiene checkpoint

### Alternative Approaches Considered

**Option B: Standalone `orch triage` command**
- **Pros:** Single command to run all hygiene checks, could automate some decisions
- **Cons:** Doesn't solve "when to run it" problem (Finding 4), adds CLI complexity, removes judgment from workflow
- **When to use instead:** If hygiene workflow becomes standardized enough to automate decisions

**Option C: Automated hygiene via daemon**
- **Pros:** Zero orchestrator overhead, runs continuously
- **Cons:** Loses human judgment (dedup/consolidate require understanding), may make incorrect decisions
- **When to use instead:** If exact-match deduplication proves sufficient and safe to auto-merge

**Rationale for recommendation:** Session start checkpoint is natural integration point (Finding 2) that uses existing primitives (Finding 1) without adding CLI complexity. Guidance-based approach preserves orchestrator judgment while providing systematic triggers (Finding 4).

---

### Implementation Details

**What to implement first:**
1. Draft "Proactive Hygiene Checkpoint" section with workflow table
2. Add multi-trigger guidance (session start, before batch-label, weekly, friction)
3. Update Session Start Protocol to reference hygiene checkpoint
4. Test workflow by running through it manually once

**Things to watch out for:**
- ⚠️ bd duplicates only catches exact matches (content hash) - won't find near-duplicates
- ⚠️ Session start checkpoint may feel like overhead initially - emphasize time savings
- ⚠️ Weekly trigger is vague - may need calendar integration or reminder mechanism
- ⚠️ "When backlog feels noisy" is subjective - provide concrete signals (>30 open, >10 P1, etc.)

**Areas needing further investigation:**
- Near-duplicate detection - do we need fuzzy matching or is exact-match sufficient?
- Optimal hygiene frequency - is weekly enough or do we need more frequent checks?
- Forcing functions - if orchestrators skip checkpoint, do we need gates/reminders?
- Epic consolidation heuristics - how to identify orphan issues that should be epic children?

**Success criteria:**
- ✅ Orchestrator skill has clear Proactive Hygiene Checkpoint section with workflow
- ✅ Multi-trigger guidance table exists with concrete triggers
- ✅ Workflow can be followed without ambiguity (specific bd commands listed)
- ✅ Orchestrators report reduced duplicate spawns and stale issue accumulation

---

## References

**Files Examined:**
- ~/.claude/skills/meta/orchestrator/SKILL.md - Searched for existing triage guidance and session start protocol
- beads issue orch-go-ngtyj - Original feature request with Options A and B

**Commands Run:**
```bash
# Check bd hygiene commands
bd --help
bd duplicates --help
bd stale --help
bd epic --help
bd list --help

# Search for existing triage guidance
grep -n "triage" ~/.claude/skills/meta/orchestrator/SKILL.md

# Check orch session capabilities
orch session --help
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-27-inv-capacity-utilization-workflow-orchestrator-proactively.md - Capacity-aware batch labeling workflow (reactive triage)
- **Investigation:** .kb/investigations/2025-12-27-inv-add-triage-batch-workflow-section.md - Implementation of batch labeling guidance
- **Issue:** orch-go-ngtyj - Original feature request

---

## Investigation History

**2026-01-09 14:43:** Investigation started
- Initial question: Should proactive triage be session start checkpoint (A) or standalone command (B)?
- Context: Beads issue orch-go-ngtyj requesting proactive backlog hygiene workflow

**2026-01-09 14:50:** Key finding - primitives exist
- bd duplicates, stale, epic commands provide all needed functionality
- Gap is workflow guidance, not tooling

**2026-01-09 14:55:** Investigation completed
- Status: Complete
- Key outcome: Recommend Option A (session start checkpoint) with multi-trigger guidance, using existing bd commands
