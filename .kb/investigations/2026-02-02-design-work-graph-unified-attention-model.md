# Design: Work Graph as Unified Attention Model

**Date:** 2026-02-02
**Status:** Active
**Type:** design

**Question:** Should Work Graph become the orchestrator's unified attention surface, synthesizing all signals about what needs attention?

**Delta:** Work Graph currently displays issue state but doesn't synthesize attention signals. The proposal is to evolve it from "issue viewer with extras" to "orchestrator's attention surface" - computing what needs attention by reconciling issues, commits, workspaces, agents, and artifacts.

---

## The Core Insight

The orchestrator has one job: Move work from "needs doing" to "done" while maintaining system coherence.

Everything we've built serves this:

| Component | What it *should* tell orchestrator |
|-----------|-----------------------------------|
| Work Graph | "Here's all the work and how it relates" |
| WIP Section | "Here's what's actively being worked" |
| Completion verification | "Here's what's ready to close" |
| Artifact feed | "Here's knowledge that was produced" |

**The problem:** These are separate views that don't talk to each other:
- Work Graph shows issues but doesn't know about commits
- Completion verification knows about commits but only runs on `orch complete`
- WIP shows running agents but not "done but not closed"
- Artifact feed shows knowledge but not "knowledge that needs action"

**The gap:** No unified "attention model" synthesizes all signals into "what should the orchestrator do next?"

---

## The Triggering Example

Issues stay open after work is done because commits happen outside the spawn flow:
- Issue created via `bd create`
- Work done interactively (not via `orch spawn`)
- Commits land referencing the issue
- Nothing triggers `bd close` because there's no workspace/agent/completion flow
- Issue stays open forever

Work Graph should surface this: "Issue X has commits referencing it but no workspace - likely done, needs verification."

---

## The Proposed Mental Model Shift

**Current:** Work Graph = "issue viewer with extras"

**Proposed:** Work Graph = "orchestrator's attention surface"

Work Graph shouldn't just *display* state, it should *compute* attention priority by synthesizing:

| Signal | Source | Attention implication |
|--------|--------|----------------------|
| Issue open, no workspace, has commits | git + beads | "Likely done, needs verification" |
| Issue open, workspace exists, Phase: Complete | beads comments | "Ready for orch complete" |
| Issue open, agent stuck >2h | registry + time | "Needs intervention" |
| Investigation with recommendation | artifact scan | "Needs decision" |
| Issue blocked, blocker closed | beads graph | "Unblocked, ready to spawn" |
| Issue in WIP, agent idle >30m | SSE + time | "May need attention" |
| Queued issue, daemon paused | daemon status | "Won't auto-spawn, manual action needed" |

---

## Open Questions

### 1. What are ALL the attention signals?

We listed some above. Is the list complete? What else does an orchestrator actually look at to decide "what next"?

Consider:
- Priority signals (P0 vs P2)
- Age signals (stale issues)
- Dependency signals (blocked/unblocked)
- Capacity signals (too many agents running)
- Knowledge signals (unanswered questions, pending decisions)
- External signals (CI failures, deploy status)

### 2. Where do these signals live today?

| Signal | Current source | Accessible to Work Graph? |
|--------|---------------|---------------------------|
| Issue status | beads | Yes (existing API) |
| Issue relationships | beads graph | Yes (existing API) |
| Commits mentioning issues | git log | No - needs new API |
| Workspace existence | .orch/workspace/ | Partially (registry) |
| Agent status | OpenCode + registry | Yes (existing API) |
| Phase: Complete | beads comments | Yes (existing API) |
| Artifact recommendations | .kb/ scan | Yes (existing API) |
| Daemon status | daemon state file | Yes (existing API) |

**Gap:** Git commit analysis is not currently exposed.

### 3. What's the attention priority model?

Not all signals are equal. Proposed hierarchy (high to low):

1. **Failures** - Agent crashed, verification failed, CI broken
2. **Stuck** - Agent stuck >2h, issue blocked for days
3. **Ready for action** - Phase: Complete, ready to spawn, needs decision
4. **Informational** - Knowledge produced, progress made

Should this be configurable? User-adjustable weights?

### 4. How does this change the orchestrator's workflow?

If Work Graph becomes the attention surface:
- Does `orch status` become redundant? (Agent status is in WIP)
- Does `orch review` become redundant? (Completions surfaced in Work Graph)
- Does `bd ready` become redundant? (Ready issues surfaced with context)

Or do CLI commands remain for scripting/automation while Work Graph is the human interface?

### 5. What's the interaction model?

Once Work Graph surfaces "this needs attention," what does the orchestrator *do*?

Options:
- **View only:** Work Graph shows what needs attention, orchestrator acts via terminal
- **Action buttons:** Click to complete, click to spawn, click to close
- **Hybrid:** Common actions have buttons, complex actions go to terminal

Consideration: Action buttons increase coupling and maintenance burden.

### 6. Does this subsume the daemon's job?

Daemon currently decides "what to spawn next" based on `bd ready` + labels.

If Work Graph is the attention surface:
- Is daemon just "auto-act on certain attention signals"?
- Should daemon's logic be visible in Work Graph? ("Daemon would spawn this, but capacity full")
- Should Work Graph be able to trigger daemon actions?

---

## Relationship to Existing Designs

### Phase 2 (Lifecycle Tracking)
Phase 2 designed DeliverableChecklist, AttemptHistory, IssueSidePanel - these are about understanding *why* an issue needs attention (history, context). They complement the attention model.

### Phase 3 (Artifact Feed)
Already implemented. Surfaces knowledge artifacts. The "needs decision" section is an attention signal. This fits naturally into the unified model.

### Completion Verification Model
Currently runs at `orch complete` time. Could this become continuous? Work Graph could show verification status in real-time rather than only at completion.

---

## Next Steps

1. Architect session to pressure-test this framing
2. Enumerate complete attention signal list
3. Design the reconciliation layer (how signals become attention items)
4. Decide on interaction model (view-only vs action buttons)
5. Prototype attention priority in existing Work Graph

---

## References

- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md` - Triggering investigation
- `.kb/models/completion-verification.md` - Current verification architecture
- `.kb/models/completion-lifecycle.md` - Agent completion lifecycle
- `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md` - Original Work Graph design
- `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md` - Phase 2 design
- `.kb/investigations/2026-01-31-design-work-graph-phase3-artifact-feed.md` - Phase 3 design
