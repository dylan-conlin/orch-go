<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Work Graph and Completion Verification evolved as separate systems with different "anchors" - Work Graph anchors on beads issues (what work exists), while Completion Verification anchors on spawn workspaces (proving work is done). This creates a gap where Work Graph can show "open" issues that are actually done (ad-hoc work), and Completion can't verify work that wasn't spawned.

**Evidence:** Read all 3 phase designs, the verification gap investigation, both completion models, and the completion guide. Phase 2's DeliverableChecklist/AttemptHistory track spawn-flow history but don't bridge to ad-hoc work. Verification gap investigation already identified this as architectural.

**Knowledge:** The conceptual gap is "issue lifecycle vs agent lifecycle" - Work Graph observes issues (static), Completion verifies agents (dynamic), but nothing tracks the full journey from issue creation through work completion to knowledge persistence for non-spawned work.

**Next:** Wait for Dylan's input. Three potential directions identified: (1) extend Work Graph to own "likely done" detection, (2) make spawn optional with lightweight verification, (3) accept the gap as a feature boundary. This is a strategic decision about system scope.

**Authority:** strategic - This is about what the orchestration system is FOR (scope boundary) and involves irreversible architectural choices about where verification happens

---

# Investigation: Deep Review of Work Graph and Orchestration Lifecycle

**Question:** What is the cohesive understanding gap between Work Graph's role and the orchestration lifecycle, and what conceptual integration is missing?

**Started:** 2026-02-02
**Updated:** 2026-02-02
**Owner:** Claude (spawned architect)
**Phase:** Synthesizing
**Next Step:** Discuss with Dylan before proposing implementation
**Status:** Active - awaiting discussion

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Work Graph has three distinct phases with different "done" states

**Evidence:** The three phase designs reveal different completion criteria:

| Phase | Focus | "Done" Means |
|-------|-------|--------------|
| **Phase 1** | Structure (tree view) | Issue can be browsed in hierarchy |
| **Phase 2** | Activity (agent overlay) | Agent completed its attempt |
| **Phase 3** | Artifacts | Knowledge artifact exists for review |

Phase 1 is structural (implemented). Phase 2 tracks execution lifecycle (partially implemented - WIP section exists, but DeliverableChecklist/IssueSidePanel/AttemptHistory missing). Phase 3 is about knowledge outputs (implemented).

**Source:**
- `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md`
- `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md`
- `.kb/investigations/2026-01-31-design-work-graph-phase3-artifact-feed.md`

**Significance:** The phases each answer a different question: "What exists?" (Phase 1), "What's happening?" (Phase 2), "What was learned?" (Phase 3). The gap is in connecting these: an issue can be "done" in one sense but not others.

---

### Finding 2: Completion Verification is workspace-centric, not issue-centric

**Evidence:** The completion model documents three independent gates (Phase, Evidence, Approval), but ALL gates require a workspace:

```go
func VerifyCompletionWithTier(workspace string) error {
    tier := readTierFile(workspace)  // Requires workspace
    // ...
}
```

From `completion-verification.md:143-155`:
- Light tier: Phase + commits (from workspace)
- Full tier: Phase + commits + SYNTHESIS.md (from workspace)
- Orchestrator tier: SESSION_HANDOFF.md (from workspace)

The verification gap investigation confirmed: "orch complete <beads-id> → looks up workspace in .orch/workspace/ → Workspace contains .session_id, .tier, SPAWN_CONTEXT.md → Verification gates all read from workspace + beads comments → No verification path exists for work without a workspace."

**Source:**
- `.kb/models/completion-verification.md:127-157` - Tier-aware verification
- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md:60-76`

**Significance:** The verification system assumes spawn → workspace → completion. Work that doesn't flow through spawn has no verification path. This is by design (spawn provides context), but creates a conceptual gap.

---

### Finding 3: Two competing "anchors" - issues vs workspaces

**Evidence:** The system has two parallel tracking systems:

| System | Anchor | Tracks | Lifecycle |
|--------|--------|--------|-----------|
| **Beads** | Issue ID | Work that needs doing | Created → Open → In Progress → Closed |
| **Spawn/Workspace** | Workspace name | Execution attempt | Spawn → Active → Complete → Archived |

Work Graph queries beads (issues). Completion Verification queries workspaces (attempts). The correlation is supposed to be:
- Spawn with `--issue` creates workspace tied to beads issue
- `orch complete` closes the beads issue when workspace passes gates

But the coupling is loose:
- Issues can exist without spawns (manual work, deferred ideas)
- Spawns can exist without issues (`--no-track`, ad-hoc debugging)
- Multiple spawns can target the same issue (retries)

**Source:**
- `.kb/models/completion-lifecycle.md:14-23` - The Completion Chain
- `.kb/guides/completion.md:197-227` - Four-Layer Cleanup Model

**Significance:** The two anchors serve different purposes. Issues track "what needs doing" (strategic). Workspaces track "how we're doing it" (tactical). The gap is in the transition: when does "what" become "done" without the "how"?

---

### Finding 4: Phase 2's "lifecycle observability" is spawn-specific

**Evidence:** The Phase 2 design's core concept is "Issue lifecycle as First-Class Observable":

```
The issue is the anchor. Agents come and go. Artifacts accumulate.
The lifecycle is the story of how the issue got (or didn't get) resolved.
```

But the implementation tracks **agent** lifecycle, not **issue** lifecycle:
- AttemptHistory: "How many times was this issue spawned?"
- DeliverableChecklist: "Did the spawned agent produce expected outputs?"
- Health indicators: "Is the current agent healthy?"

None of these answer: "Was this issue resolved by work outside the spawn flow?"

The verification gap investigation explicitly noted: "Phase 2 designs are orthogonal to this gap. All of this assumes spawn-flow work."

**Source:**
- `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md:109-125`
- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md:101-117`

**Significance:** There's a conceptual confusion between "issue lifecycle" (promised) and "agent lifecycle per issue" (implemented). The promise is about issues; the implementation is about spawn attempts.

---

### Finding 5: The three classes of ad-hoc work are real and unaddressed

**Evidence:** The verification gap investigation identified three bypass patterns:

| Class | Example | Current Handling |
|-------|---------|------------------|
| **Interactive claim** | Orchestrator opens session, works on issue directly | No workspace, no tracking |
| **Manual commits** | Human/agent commits outside spawn, issue stays open | Commits exist, issue open |
| **Direct close** | `bd close` without `orch complete` | Hook exists but opt-in |

The verification model documents this as a known constraint but not a solved problem:
> "Cross-project detection uses SPAWN_CONTEXT.md to determine which directory to verify in."

But if there's no SPAWN_CONTEXT.md (no spawn), there's no detection at all.

**Source:**
- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md:121-136`
- `.kb/models/completion-verification.md:209-220` - Cross-Project Verification Wrong Directory

**Significance:** These aren't edge cases. Interactive work is how orchestrators operate. Manual commits happen during pair programming. Direct closes happen when verification feels like friction. The gap is large.

---

### Finding 6: The completion-lifecycle model hints at the answer

**Evidence:** From `completion-lifecycle.md`:

> "The agent completion lifecycle is the transition from **Active Work** to **Knowledge Persistence**."

And:

> "A healthy lifecycle ensures that agent findings are externalized (D.E.K.N.), workspaces are archived, and OpenCode sessions are purged to prevent 'Registry Noise.'"

The key insight: completion isn't just "is this done?" It's "was knowledge captured?" The D.E.K.N. pattern, SYNTHESIS.md, investigation artifacts - these are the actual outputs, not the issue status.

The verification gap's recommendation aligns:
> "Add a new computed status `likely_done` to the graph API response when commits exist that mention the issue ID."

This suggests: the real question is "does evidence of completion exist?" not "did the spawn workflow run?"

**Source:**
- `.kb/models/completion-lifecycle.md:9-14`
- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md:180-199`

**Significance:** The models point toward evidence-based completion detection rather than workflow-based completion verification. This is a potential bridge.

---

## Synthesis

**Key Insights:**

1. **Two systems, two anchors, one gap** - Work Graph observes beads issues (what work exists); Completion Verification validates spawn workspaces (how work was done). The gap is in the middle: "Is this work done?" can only be answered if work flowed through spawn. This was intentional (spawn provides verification context) but creates friction for ad-hoc work.

2. **"Issue lifecycle" vs "agent lifecycle" confusion** - Phase 2's promise was "the issue is the anchor" but the implementation tracks agent attempts per issue. The designs conflate two different things: the story of how an issue got resolved (which may span multiple agents, manual work, and time) vs the health of a currently-running agent (point-in-time observable).

3. **Evidence-based vs workflow-based completion** - The verification gap investigation proposes a shift: instead of "did verification workflow run?", ask "does evidence of completion exist?" Commits, artifacts, and knowledge outputs are evidence. This would let Work Graph surface "likely done" issues even without spawns.

4. **Knowledge persistence is the real deliverable** - The completion-lifecycle model correctly identifies that completion is about knowledge capture (D.E.K.N., SYNTHESIS.md, investigations), not issue status. An issue can be "closed" but fail to capture knowledge. An issue can be "open" but have all knowledge already captured.

5. **The three ad-hoc patterns are design feedback** - Interactive claims, manual commits, and direct closes aren't bugs - they're signals that the spawn workflow has friction. The question isn't "how do we force spawn?" but "what's spawn providing that these bypass?"

**Answer to Investigation Question:**

The cohesive understanding gap is between two different mental models:

**Model A: Workflow-Centric Orchestration**
- Spawn creates tracking context
- Agents work within that context
- Completion verifies against that context
- Issues are tracking artifacts

**Model B: Evidence-Centric Knowledge Management**
- Issues define what knowledge is needed
- Work produces evidence (commits, artifacts, D.E.K.N.)
- Completion detects when evidence exists
- Spawn is one way to produce evidence, not the only way

The current system is Model A. The verification gap investigation proposes elements of Model B. The friction in ad-hoc work happens because Model A requires spawn but Model B only requires evidence.

The missing conceptual integration is: **a unified view of "work completion" that treats spawn-flow and ad-hoc-flow as different evidence sources for the same question: "Does the knowledge exist?"**

---

## Structured Uncertainty

**What's tested:**

- ✅ Phase 1 and Phase 3 fully implemented (verified: audit investigation confirmed)
- ✅ Phase 2 partially implemented - WIP section works (verified: audit investigation)
- ✅ Phase 2 missing DeliverableChecklist/IssueSidePanel/AttemptHistory (verified: grep search)
- ✅ Completion verification requires workspace (verified: read check.go, complete_cmd.go)
- ✅ Three classes of ad-hoc work bypass spawn (verified: identified in gap investigation)

**What's untested:**

- ⚠️ How much ad-hoc work actually happens (no metrics on bypass patterns)
- ⚠️ Whether evidence-based detection would have acceptable false positive rate
- ⚠️ Whether Dylan prefers keeping spawn requirement vs loosening it
- ⚠️ Whether the gap is actually causing problems or just feels incomplete

**What would change this:**

- If Dylan says "spawn is the point, ad-hoc should spawn" → the gap is intentional friction
- If metrics show 90% of work goes through spawn → the gap is minor
- If evidence-based detection has high false positive rate → Model B isn't practical

---

## Blocking Questions (for Dylan)

Before proposing implementation, these strategic questions need answers:

### Q1: Is the spawn requirement a feature or a friction?

**Context:** Spawn provides verification context (workspace, tier, deliverables). Ad-hoc work bypasses this. The question is whether spawn is mandatory-by-design (the point is the context) or optional-by-preference (the point is the evidence).

**Options:**
- **A: Feature** - Spawn is how we ensure quality. Ad-hoc work should spawn.
- **B: Friction** - Evidence is what matters. Spawn is one evidence source.
- **C: Contextual** - Some work needs spawn (agents), some doesn't (human work).

**Why this matters:** This determines whether the gap is a bug to fix or a boundary to accept.

---

### Q2: What is Work Graph's responsibility boundary?

**Context:** Work Graph currently observes beads (issues) and can query workspaces (agents). The verification gap investigation proposes Work Graph own "likely done" detection via git history.

**Options:**
- **A: Observation only** - Work Graph shows what exists, doesn't compute status
- **B: Computed status** - Work Graph computes derived status like "likely done"
- **C: Orchestration helper** - Work Graph assists completion flow (prompts, reconciliation)

**Why this matters:** This determines whether the fix belongs in Work Graph or elsewhere.

---

### Q3: Should Phase 2 track "issue lifecycle" or "agent lifecycle"?

**Context:** The Phase 2 design promises "issue lifecycle as first-class observable" but the implementation tracks agent attempts. True issue lifecycle would include non-spawn work.

**Options:**
- **A: Agent lifecycle** - Phase 2 tracks spawn attempts only (current direction)
- **B: Issue lifecycle** - Phase 2 tracks all work evidence (requires git integration)
- **C: Both** - Agent lifecycle in WIP section, issue lifecycle in side panel

**Why this matters:** This determines the scope of Phase 2's remaining implementation.

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md` - Phase 1 design
- `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md` - Phase 2 design
- `.kb/investigations/2026-01-31-design-work-graph-phase3-artifact-feed.md` - Phase 3 design
- `.kb/investigations/2026-02-02-inv-audit-work-graph-design-docs.md` - Implementation audit
- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md` - Gap analysis
- `.kb/models/completion-verification.md` - Verification architecture
- `.kb/models/completion-lifecycle.md` - Completion lifecycle
- `.kb/guides/completion.md` - Completion workflow

**Commands Run:**
```bash
# Find design docs
glob ".kb/investigations/*work-graph*.md"
glob ".kb/investigations/*phase*.md"

# Find completion models
glob ".kb/models/completion*.md"
```

**Related Artifacts:**
- **Model:** `.kb/models/completion-verification.md` - Three-layer verification
- **Model:** `.kb/models/completion-lifecycle.md` - Lifecycle state transitions
- **Investigation:** `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md` - Prior gap analysis

---

## Investigation History

**2026-02-02 10:00:** Investigation started
- Initial question: What's the cohesive understanding gap between Work Graph and orchestration lifecycle?
- Context: Spawned for deep review of Work Graph design docs

**2026-02-02 10:30:** All design docs read
- Phase 1 (structure), Phase 2 (activity), Phase 3 (artifacts) designs
- Verification gap investigation
- Both completion models
- Completion guide

**2026-02-02 11:00:** Synthesis complete
- Identified two-anchor gap (issues vs workspaces)
- Identified Model A vs Model B framing
- Generated 3 blocking questions for Dylan
- Status: Active - awaiting discussion

---

## Waiting for Dylan

I've synthesized the conceptual gap but am NOT proposing solutions yet per the task: "Wait for Dylan to interact before proposing solutions."

The three blocking questions above capture the strategic decisions needed. Once Dylan provides direction, I can:
- Update this investigation with recommendations
- Create SYNTHESIS.md
- Report Phase: Complete
