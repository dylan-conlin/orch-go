<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Designed explicit completion lifecycle for two orchestration modes (Active/Triage) with work-type-specific requirements.

**Evidence:** Analyzed existing code (verify.go, review.go, SYNTHESIS.md template), orchestrator skill, prior investigations (synthesis-protocol, single-agent-review).

**Knowledge:** Completion depth should vary by work type and orchestration mode - not one-size-fits-all. Mental model sync is the critical gap.

**Next:** Add completion lifecycle section to orchestrator skill with mode/work-type matrix.

**Confidence:** High (85%) - Design addresses observed gaps; needs validation with real orchestration sessions.

---

# Investigation: Orchestrator Completion Lifecycle Design

**Question:** How should the orchestrator complete agents in two distinct modes (Active Orchestration vs Triage Orchestration), and what lifecycle steps are required for different work types?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Architect agent (og-arch-design-orchestrator-completion-25dec)
**Phase:** Complete
**Next Step:** Update orchestrator skill with completion lifecycle section
**Status:** Complete
**Confidence:** High (85%)

---

## Problem Framing

### The Core Tension

Two process modes have emerged in orchestration practice:

1. **Active Orchestration:** Dylan spawns agents directly, monitors them, completes them in real-time. High engagement, immediate synthesis.

2. **Triage Orchestration:** Daemon spawns agents overnight, orchestrator synthesizes in batch next morning. Lower engagement, deferred synthesis.

These modes have different completion needs, but the current skill and tooling treat them identically.

### What "Completion" Actually Means

Completion is more than `orch complete <id>`:
- **Verification:** Did the agent produce what was expected?
- **Synthesis:** What changed? What was learned?
- **Mental Model Sync:** Does Dylan's understanding of the system match reality?
- **Follow-up Triage:** What new work emerged? What needs attention?

### Success Criteria

1. Orchestrator knows explicitly which completion workflow to use based on mode and work type
2. Dylan's mental model stays synchronized with system changes
3. Quality control is proportional to risk (UI changes need more verification than docs)
4. Follow-up work is systematically captured (not lost in synthesis)

---

## Findings

### Finding 1: Current completion flow is verification-focused, not synthesis-focused

**Evidence:** The `runComplete()` function in `cmd/orch/main.go:2434-2596` performs:
- Phase verification via beads comments
- Liveness check (is agent still running?)
- Issue closure
- Tmux window cleanup
- Auto-rebuild for Go changes

But it does NOT:
- Surface SYNTHESIS.md content to orchestrator
- Prompt for mental model sync
- Extract follow-up issues
- Vary behavior by work type

**Source:** `cmd/orch/main.go:2434-2596`

**Significance:** The tooling focuses on "is it safe to close?" not "what do I need to understand?" The synthesis is produced by agents (SYNTHESIS.md) but not consumed systematically by orchestrator.

---

### Finding 2: `orch review` already has synthesis card display but it's separate from complete

**Evidence:** The `review.go` file implements:
- Single-agent review (`orch review <id>`) with full synthesis display
- Batch review (`orch review`) for project-level overview
- Synthesis card parsing and display

The prior decision record (`.kb/decisions/2025-12-21-single-agent-review-command.md`) recommended Option A: `--preview` flag on `orch complete`.

**Source:** `cmd/orch/review.go:216-258`, `.kb/decisions/2025-12-21-single-agent-review-command.md`

**Significance:** The building blocks exist. The gap is integration - review should be the default path for certain work types, not an opt-in flag.

---

### Finding 3: Different work types have fundamentally different completion needs

**Evidence:** Examining skill categories and their outputs:

| Work Type | Primary Output | Mental Model Impact | Verification Need |
|-----------|---------------|---------------------|-------------------|
| **Bug fix** | Code change | Low (behavior restored) | Tests pass, regression prevented |
| **UI feature** | Visible behavior | High (user experience changed) | Visual verification, browser smoke test |
| **Investigation** | Knowledge artifact | Medium (understanding updated) | Findings reviewed, conclusions validated |
| **Architecture** | Decision + recommendations | High (system direction changed) | Trade-offs understood, decision approved |
| **Refactor** | Code change (no behavior) | Low (same behavior) | Tests pass, no regression |

**Source:** Analysis of skill outputs from orchestrator skill and spawn tier definitions in `spawn.TierLight`/`spawn.TierFull`

**Significance:** A bug fix can be completed with "tests pass" verification, but an architecture decision requires Dylan to understand and approve the recommendation. One-size-fits-all completion is wrong.

---

### Finding 4: Mental model sync is the critical gap

**Evidence:** Current SYNTHESIS.md template has sections for:
- Delta (what changed) ✓
- Evidence (what was observed) ✓
- Knowledge (what was learned) ✓
- Next (what should happen) ✓

But there's no prompt asking: "Did this change Dylan's mental model of the system?"

Examples where mental model sync matters:
- Agent discovers a constraint Dylan didn't know existed
- Agent implements feature differently than Dylan expected
- Agent's investigation reveals architecture Dylan misunderstood
- Agent makes a design decision Dylan needs to know about

**Source:** `.orch/templates/SYNTHESIS.md`

**Significance:** Without explicit mental model sync, Dylan can become disconnected from the system state. This is especially true in Triage mode where he didn't watch the work happen.

---

### Finding 5: Follow-up work often gets lost in synthesis

**Evidence:** SYNTHESIS.md Next section supports:
- close
- spawn-follow-up (with issue/skill/context)
- escalate
- resume

But the orchestrator doesn't systematically:
- Extract follow-up issues
- Create beads issues automatically
- Present options for discovered work

Unexplored Questions section captures this conceptually but it's passive (agent writes it, orchestrator may not read it).

**Source:** `.orch/templates/SYNTHESIS.md`, `pkg/verify/review.go:33-35`

**Significance:** Work discovered during implementation (the "iceberg effect") needs a forcing function to be captured. Otherwise it falls through the cracks.

---

## Synthesis

### Key Insights

1. **Two-mode completion is real and should be explicit** - Active and Triage modes have different time pressures, context availability, and attention budgets. Design for both.

2. **Work type determines completion depth** - Bug fixes need "did it work?" verification. Architecture decisions need "do I understand this?" synthesis. UI features need "does it look right?" validation.

3. **Mental model sync is the underserved need** - The system captures what changed but doesn't prompt for whether understanding was updated. This is especially critical after Triage mode batches.

4. **Follow-up triage needs a forcing function** - Agents discover issues. Without explicit capture at completion time, they get lost.

### Completion Lifecycle Framework

**Phase 1: Verification (Automated)**
- Phase: Complete reported via beads
- Constraints satisfied (from SPAWN_CONTEXT.md)
- Deliverables exist (SYNTHESIS.md, investigation, commits)
- Tests pass (if applicable)

**Phase 2: Synthesis Review (Mode-Dependent)**
- Active Mode: Quick scan of TLDR + recommendation
- Triage Mode: Full review of SYNTHESIS.md sections

**Phase 3: Mental Model Sync (Work-Type Dependent)**
- UI Feature: Browser verification, visual confirmation
- Architecture: Decision understood, recommendation approved
- Investigation: Conclusions internalized, constraints noted
- Bug Fix/Refactor: Skip (low mental model impact)

**Phase 4: Follow-up Triage (Always)**
- Extract Unexplored Questions
- Create beads issues for discovered work
- Decide: close vs spawn-follow-up vs escalate

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The design is grounded in concrete evidence from existing code and artifacts. The two-mode distinction is observable in practice. The work-type matrix is based on skill categorizations that already exist.

**What's certain:**

- ✅ Two orchestration modes exist (Active vs Triage) with different needs
- ✅ Current completion flow is verification-focused, not synthesis-focused
- ✅ Work types have different mental model impact (investigation ≠ bug fix)
- ✅ Follow-up work extraction is passive, needs forcing function

**What's uncertain:**

- ⚠️ Optimal automation level - should follow-up issue creation be automatic or prompted?
- ⚠️ Whether mental model sync needs explicit prompts or just better SYNTHESIS.md sections
- ⚠️ How to handle partial completions (agent did 80% of expected work)

**What would increase confidence to Very High (95%+):**

- Validate framework with 2 weeks of real orchestration sessions
- Test both Active and Triage completion flows
- Measure if follow-up work capture rate improves

---

## Implementation Recommendations

### Recommended Approach ⭐: Mode-Aware Completion Lifecycle

Add explicit completion lifecycle section to orchestrator skill that:
1. Distinguishes Active vs Triage mode
2. Provides work-type-specific checklists
3. Forces mental model sync for high-impact work
4. Prompts for follow-up extraction

**Why this approach:**
- Leverages existing infrastructure (SYNTHESIS.md, `orch review`)
- Minimal tooling changes (skill update, not code)
- Immediately usable via orchestrator reading the skill

**Trade-offs accepted:**
- Relies on orchestrator following the process (not automated)
- Adds cognitive overhead (acceptable for quality)

### Alternative Approaches Considered

**Option B: Automated completion workflows in orch CLI**
- **Pros:** Forces the workflow, can't skip steps
- **Cons:** Significant code changes, may be too rigid
- **When to use instead:** If orchestrator consistently skips mental model sync

**Option C: Work-type-specific completion commands**
- **Pros:** `orch complete-ui`, `orch complete-arch` with tailored flows
- **Cons:** Command proliferation, harder to discover
- **When to use instead:** If mode-aware single command is too complex

**Rationale for recommendation:** Option A (skill update) is highest leverage with lowest risk. It documents the lifecycle explicitly, enabling orchestrators to follow it, while allowing future tooling to enforce it.

---

### Implementation Details

**What to add to orchestrator skill:**

```markdown
## Orchestrator Completion Lifecycle

### Mode Detection

Before completing, identify your mode:

| Mode | Indicators | Attention Budget |
|------|------------|------------------|
| **Active** | You spawned this agent, watched it work | High (immediate synthesis) |
| **Triage** | Daemon spawned, you're reviewing batch | Medium (efficient review) |

### Completion Workflow by Work Type

| Work Type | Verification | Synthesis Depth | Mental Model Sync | Follow-up |
|-----------|--------------|-----------------|-------------------|-----------|
| **Bug Fix** | Tests pass | TLDR only | Skip | Check for related issues |
| **UI Feature** | Browser smoke test | Full SYNTHESIS | Visual confirmation | Screenshot changes |
| **Investigation** | Conclusions in artifact | Full SYNTHESIS | Internalize findings | Create follow-up issues |
| **Architecture** | Decision produced | Full SYNTHESIS + Discussion | Understand trade-offs | Review feature list |
| **Refactor** | Tests pass, no behavior change | TLDR only | Skip | Note any debt discovered |

### Mental Model Sync Questions

For high-impact work, ask yourself:
- Did this reveal something I didn't know?
- Does my understanding of the system need updating?
- Did the agent make a choice I need to remember?
- Should I update CLAUDE.md or create a decision record?

### Follow-up Extraction

For every completion:
1. Check Unexplored Questions section
2. Check Next → spawn-follow-up recommendations
3. Ask: "What did this agent discover that needs tracking?"
4. Create beads issues for discovered work (Tier 1 via `bd create` if cause known)
```

**Things to watch out for:**
- ⚠️ Don't over-engineer - this is guidance, not enforcement
- ⚠️ Mode detection should be quick (seconds, not analysis)
- ⚠️ Mental model sync is for orchestrator, not agents

**Success criteria:**
- ✅ Orchestrator knows which completion workflow to use without thinking
- ✅ Follow-up work is systematically captured
- ✅ Dylan's mental model stays synchronized with system changes
- ✅ Quality control is proportional to work type risk

---

## References

**Files Examined:**
- `cmd/orch/main.go:2434-2596` - runComplete function
- `cmd/orch/review.go` - Single-agent and batch review implementation
- `pkg/verify/check.go` - Verification logic
- `pkg/verify/review.go` - Agent review data structures
- `.orch/templates/SYNTHESIS.md` - Synthesis template structure
- `.kb/decisions/2025-12-21-single-agent-review-command.md` - Prior decision on review

**Commands Run:**
```bash
# Verify project location
pwd  # /Users/dylanconlin/Documents/personal/orch-go

# Create investigation file
kb create investigation design/orchestrator-completion-lifecycle-two
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md` - SYNTHESIS.md design
- **Decision:** `.kb/decisions/2025-12-21-single-agent-review-command.md` - Review command design
- **Workspace:** `.orch/workspace/og-arch-design-orchestrator-completion-25dec/` - This session

---

## Investigation History

**[2025-12-25 10:00]:** Investigation started
- Initial question: How should orchestrator complete agents in two modes (Active vs Triage)?
- Context: Spawned to design explicit completion lifecycle for orchestrator skill

**[2025-12-25 10:15]:** Problem framing complete
- Identified two-mode distinction (Active vs Triage)
- Defined success criteria for completion lifecycle

**[2025-12-25 10:30]:** Findings gathered
- Analyzed runComplete, review.go, SYNTHESIS.md
- Identified mental model sync as critical gap
- Mapped work types to completion needs

**[2025-12-25 10:45]:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Mode-aware completion lifecycle framework designed with work-type-specific requirements
