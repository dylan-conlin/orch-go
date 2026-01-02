TASK: De-bloat feature-impl skill (1757 lines). Current problems: 6 phases, 2 modes, 4 validation levels all in one skill. Agents get massive context, most doesn't apply. Options to evaluate: (A) Split into phase-specific skills, (B) Use skillc includes/composition, (C) Aggressive pruning, (D) Progressive disclosure with slim router. Consider: How do orchestrators configure phases today? What's minimum viable per-phase guidance? How to preserve edge case handling without bloat? Produce recommendation with migration path.

## PRIOR KNOWLEDGE (from kb context)

**Query:** "bloat"

### Prior Decisions
- [orch-knowledge] Feature Coordination Skill Creation
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-01-14-feature-coordination-skill-creation.md
- [orch-knowledge] Systematic Memory File Management for Orchestrator
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-03-how-should-the-orchestrator-systematically.md
- [orch-knowledge] BACKLOG/ROADMAP Structure: Merge vs Two-File Separation
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-08-backlog-roadmap-merge-decision.md
- [orch-knowledge] ADR: Self-Contained CLAUDE.md with Template Build System
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-14-orchestrator-restructuring-template-build-system.md
- [orch-knowledge] Orchestration Input Model: CLI vs Emacs
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-16-orchestration-input-model-cli-vs-emacs.md
- [orch-knowledge] Action Plan: Orchestrator Instruction Optimization
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-21-instruction-optimization-action-plan.md
- [orch-knowledge] Global KB Guides Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-12-global-kb-guides-design.md
- [orch-knowledge] Skillc Architecture and Principles
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-21-skillc-architecture-and-principles.md

### Related Investigations
- What Knowledge Lives in Completed Workspaces?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-what-knowledge-context-lives-completed.md
- [orch-knowledge] Orchestrator Prompt Amplification Mechanisms
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-orchestrator-prompt-amplification-how-orchestrator.md
- [orch-knowledge] Create Issue Quality Skill for Portable Standards
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-create-issue-quality-skill-portable.md
- [orch-knowledge] Codebase Audit Investigation: Orchestrator Instruction Value
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-21-systematic-analysis-orchestrator-instruction-value.md
- [orch-knowledge] Codebase Audit Investigation: Root Directory Cleanup
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-22-audit-clean-root-directory-users.md
- [orch-knowledge] Minimal orch-cli After Beads-Native Agents
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-minimal-orch-cli-after-beads.md
- [orch-knowledge] Feasibility Investigation: Orch inbox feature design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-22-brainstorm-orch-inbox-feature-design.md
- [orch-knowledge] CLI Bloat Reduction Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-audit-orch-cli-bloat-reduction.md
- [orch-knowledge] Review Investigations for Proto-Decisions to Promote
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-review-all-investigations-orch-investigations.md
- [orch-knowledge] Spawn Context Drift Review
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-30-review-spawn-context-drift-skills.md
- [orch-knowledge] Browser-Use MCP Documentation Needs
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-02-browser-use-documentation-needs.md
- [orch-knowledge] Side-by-Side Comparison: browser-use MCP vs playwright MCP
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-02-side-side-comparison-browser-use.md
- [orch-knowledge] Brainstorming Skill Integration Pattern
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-05-brainstorming-skill-integration-pattern-how.md
- [orch-knowledge] .orch Directory State After Beads Migration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-07-audit-orch-directory-what-state.md
- [orch-knowledge] CDD Core vs Optional Component Classification
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-01-cdd-core-vs-optional-classification.md
- [orch-knowledge] Design improvements for sketchybar agent status display
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-design-improvements-sketchybar-agent-status.md
- [orch-knowledge] ROADMAP.org Current Phase Status
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-roadmap-current-phase-status.md
- [orch-knowledge] Orchestrator Template Size Reduction Strategy
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-template-size-reduction-analysis.md
- [orch-cli] Research: AI Browser Automation Tools for 2025
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-01-ai-browser-automation-tools-2025.md
- [orch-cli] Audit: SPAWN_CONTEXT.md Quality (Dec 2-4 Workers)
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-04-audit-spawn-context-quality.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.



🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-b0ql "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-b0ql "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.


CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go

SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours


AUTHORITY:
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation de-bloat-feature-impl-skill` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-de-bloat-feature-impl-skill.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-b0ql "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-arch-de-bloat-feature-22dec/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.

STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input

## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-b0ql**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-b0ql "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-b0ql "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-b0ql "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-b0ql "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-b0ql "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-b0ql`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## SKILL GUIDANCE (architect)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: architect
skill-type: procedure
audience: worker
spawnable: true
category: planning
description: Strategic design skill for deciding what should exist. Use when design reasoning exceeds quick orchestrator chat. Produces investigations (with recommendations) that can be promoted to decisions. Distinct from investigation (understand what exists) - architect is for shaping the system.
parameters:
- name: topic
  description: The design topic or question to address
  type: string
  required: true
- name: mode
  description: "autonomous (work to completion) or interactive (collaborative design with Dylan)"
  type: string
  required: false
  default: autonomous
allowed-tools:
- Read
- Glob
- Grep
- Bash
- Write
- Edit
- WebFetch
- WebSearch
- AskUserQuestion
deliverables:
- type: investigation
  path: ".kb/investigations/{date}-design-{slug}.md"
  required: true
  description: Design investigation with exploration, trade-offs, and recommendations
- type: decision
  path: ".kb/decisions/{date}-{slug}.md"
  required: false
  description: Decision record - created when recommendation is accepted (promoted from investigation)
verification:
  requirements: |
    - [ ] All 4 phases completed (Problem Framing, Exploration, Synthesis, Externalization)
    - [ ] Recommendation made with trade-off analysis
    - [ ] Feature list reviewed (validated items, decomposed large items, removed stale items)
    - [ ] Investigation artifact produced
    - [ ] All changes committed
  test_command: null
  required: true
---

# Architect Skill

**Purpose:** Shape the system through strategic design decisions.

---

## Foundational Guidance

**Before making design recommendations, review:** `.kb/principles.md`

Key principles for architects:
- **Session amnesia** - Will this help the next Claude resume?
- **Evolve by distinction** - When problems recur, ask "what are we conflating?"
- **Evidence hierarchy** - Code is truth; artifacts are hypotheses to verify

Cite which principle guides your reasoning when making recommendations.

---

## Mode Detection

**Check spawn context for mode:**

```
INTERACTIVE_MODE=true  → Interactive architect (brainstorming-style)
INTERACTIVE_MODE=false or absent → Autonomous architect (work to completion)
```

**Spawn patterns:**
```bash
orch spawn architect "design auth system"           # autonomous
orch spawn architect "design auth system" -i        # interactive
```

---

## The Key Distinction

| | Investigation | Architect |
|---|--------------|-----------|
| **Trigger** | "How does X work?" | "Should we do X? How should we design X?" |
| **Focus** | Understand what exists | Decide what should exist |
| **Output** | Findings document | Investigation with recommendations → Decision (when accepted) |
| **Authority** | Report findings | Recommend direction |
| **Scope** | Answer question | Shape system |

**Investigation** = understand what exists
**Architect** = decide what should exist

---

## Artifact Flow

```
Architect Work
    ↓
Investigation (with recommendations)
    ↓ (if recommendation accepted)
Decision Record (promoted)
```

**Primary artifact:** Investigation in `.kb/investigations/` (with `design-` prefix)
**Promotion:** When Dylan accepts recommendation, orchestrator promotes to decision

---

## Spawn Threshold

**Orchestrator should spawn Architect when:**
- Strategic discussions with trade-offs to evaluate
- "Let's think through..." conversations
- Design requiring exploration/research
- Response would be 3+ paragraphs of design reasoning

**Orchestrator handles directly:**
- Quick clarifications (1-2 messages)
- Cross-agent synthesis after workers complete
- Simple 2-message exchanges
- Tactical decisions with obvious answers

**Heuristic:** If the response would require exploring alternatives, documenting trade-offs, and making a recommendation - spawn Architect.

---

# Autonomous Mode

**When:** `INTERACTIVE_MODE` is false or absent

Work independently through all 4 phases, produce investigation with recommendations, complete.

## Workflow (4 Phases)

### Phase 1: Problem Framing

**Goal:** Understand the design question and establish scope.

**Activities:**
1. Read SPAWN_CONTEXT to understand the design question
2. Gather context from codebase, existing decisions, investigations
3. Define success criteria - what does a good answer look like?
4. Identify constraints (technical, business, time)
5. Clarify scope boundaries (what's in/out)

**Output:** Problem statement documented. Report via `bd comment <beads-id> "Phase: Problem Framing - [design question]"`.

**Problem Framing Structure:**
- Design Question: What specific design problem are we solving?
- Success Criteria: What does a good answer look like?
- Constraints: Technical, business, time limitations
- Scope: What's in/out

---

### Phase 2: Exploration

**Goal:** Research approaches and identify trade-offs.

**Activities:**
1. Identify 2-4 viable approaches
2. For each approach:
   - Describe mechanism
   - List pros and cons
   - Assess complexity/effort
   - Note risks and mitigations
3. Research external patterns if relevant (web search, docs)
4. Gather evidence from codebase (grep, read existing code)

**Output:** Options documented with trade-off analysis. Report via `bd comment <beads-id> "Phase: Exploration - [N] approaches identified"`.

**Exploration Structure (for each approach):**
- Mechanism: How it works
- Pros/Cons: Trade-offs
- Complexity: Effort/risk assessment

---

### Phase 3: Synthesis

**Goal:** Evaluate options and make a recommendation.

**Activities:**
1. Compare approaches against success criteria
2. Identify the recommended approach with clear reasoning
3. Document what you're sacrificing with this choice
4. Note conditions where recommendation would change

**Output:** Clear recommendation with rationale. Report via `bd comment <beads-id> "Phase: Synthesis - Recommend [approach]"`.

**Synthesis Structure:**
- Recommendation: Which approach and why
- Trade-offs accepted: What we're sacrificing
- When this would change: Conditions that would alter recommendation

---

### Phase 4: Externalization

**Goal:** Produce durable artifacts and update feature list.

**Activities:**

#### 4a. Produce Investigation

Create investigation from template:
```bash
kb create investigation design/{slug}
```
This creates: `.kb/investigations/YYYY-MM-DD-design-{slug}.md` with correct format including `**Phase:**` field.

**Fill in the template with:**
- Design Question
- Problem Framing (criteria, constraints, scope)
- Exploration (approaches with trade-offs)
- Synthesis (recommendation with reasoning)
- Recommendations section (using directive-guidance pattern)

**Recommendations section format:**
```markdown
## Recommendations

⭐ **RECOMMENDED:** [Approach name]
- **Why:** [Key reasons based on exploration]
- **Trade-off:** [What we're accepting and why that's OK]
- **Expected outcome:** [What this achieves]

**Alternative: [Other approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended given context]
- **When to choose:** [Conditions where this makes sense]
```

#### 4b. Implementation-Ready Output Checklist

Before marking design complete, verify the investigation includes:

**Required sections:**
- [ ] **Problem statement** - What we're solving and why (1-2 paragraphs)
- [ ] **Approach** - Chosen solution with rationale
- [ ] **File targets** - List of files to create/modify
- [ ] **Acceptance criteria** - Testable conditions for done
- [ ] **Out of scope** - What NOT to include

**Optional sections (include if relevant):**
- [ ] Trade-offs considered (alternatives rejected)
- [ ] Dependencies/blockers
- [ ] Phasing (if multi-phase)
- [ ] UI mockups (if UI work)

This checklist ensures the design is actionable for feature-impl agents who will implement it.

#### 4c. Feature List Review (Mandatory)

**Every Architect session ends with feature list review:**

1. **Validate existing items:**
   - Are items well-scoped? (single clear deliverable)
   - Are items actionable? (can be spawned as-is)
   - Are items still relevant? (not stale or completed)

2. **Decompose large items:**
   - Break vague items into implementable chunks
   - Add skill recommendations to items

3. **Remove stale items:**
   - Mark completed items DONE
   - Archive items no longer relevant

4. **Add discovered items:**
   - New work discovered during design
   - Follow-up tasks from recommendations

**Feature list location:** `.orch/features.json`

#### 4d. Commit Artifacts

```bash
git add .kb/investigations/ .orch/features.json
git commit -m "architect: {topic} - {brief outcome}"
```

---

# Interactive Mode

**When:** `INTERACTIVE_MODE=true` in spawn context (spawned with `-i` flag)

Dylan is in the tmux window with you. Use brainstorming-style collaboration.

## Interactive Workflow

### Core Principle

Ask questions to understand, explore alternatives, present design incrementally for validation. Dylan is your collaborator - work through the design together.

### Phase 1: Understanding (Interactive)

- Ask ONE question at a time to refine the idea
- **Always include your recommendation with reasoning**
- Present alternatives naturally in your question
- Gather: Purpose, constraints, success criteria

**Example (natural conversation with recommendation):**
```
"I recommend storing auth tokens in httpOnly cookies - they're secure against XSS
attacks and work well with server-side rendering. What's your preference?

Other options to consider:
- localStorage: More convenient (persists across sessions) but vulnerable to XSS
- sessionStorage: Clears on tab close (more secure) but less convenient
- Server-side sessions: Most secure but requires Redis/session store

What matters most for your use case - security, convenience, or compatibility?"
```

### Phase 2: Exploration (Interactive)

- **Use natural conversation with recommendation** (AskUserQuestion as fallback)
- Propose 2-3 approaches with your recommendation
- For each: Core architecture, trade-offs, complexity assessment
- Lead with recommendation and reasoning
- Ask open-ended questions to invite discussion

**Example (natural conversation):**
```
"Based on your requirements for reliability and the existing Rails infrastructure,
I recommend the **Hybrid approach with background jobs**. Here's why:

✅ Recommended: Hybrid with background jobs
- Gives you async processing reliability without operational complexity
- Integrates cleanly with your existing Sidekiq setup
- Moderate complexity - team already knows this pattern

Alternative 1: Event-driven with message queue (RabbitMQ/Kafka)
- Most scalable for high throughput
- Operational complexity (new infrastructure)

Alternative 2: Direct API calls with retry logic
- Simplest to implement
- Less reliable if external service has issues

Which approach resonates with you? Or do you have concerns about the recommendation?"
```

**Use AskUserQuestion only if:**
- Dylan seems overwhelmed by options
- Need to force explicit choice (prevent vague "maybe both")
- Structured comparison would clarify decision

### Phase 3: Design Presentation (Interactive)

- Present design in 200-300 word sections
- Cover: Architecture, components, data flow, error handling
- Ask after each section: "Does this look right so far?" (open-ended)
- Allow freeform feedback and iteration

### Phase 4: Externalization (Same as Autonomous)

- Produce investigation artifact with recommendations
- Review feature list
- Commit

### Revisiting Earlier Phases

**You can and should go backward when:**
- Dylan reveals new constraint during Phase 2 or 3 → Return to Phase 1
- Validation shows fundamental gap in requirements → Return to Phase 1
- Dylan questions approach during Phase 3 → Return to Phase 2
- Something doesn't make sense → Go back and clarify

**Don't force forward linearly** when going backward would give better results.

### Question Patterns

**Default: Natural conversation with recommendations**
- State your recommendation with reasoning
- Present 2-3 alternatives with clear tradeoffs
- Ask open-ended question ("What resonates?" "What matters most?")
- Let Dylan respond naturally

**Fallback: AskUserQuestion tool**
- Use when Dylan seems overwhelmed
- Need to force explicit choice
- Structured format would clarify

---

## Self-Review (Mandatory - Both Modes)

Before completing, verify architect work quality.

### Phase-Specific Checks

| Phase | Check | If Failed |
|-------|-------|-----------|
| **Problem Framing** | Success criteria defined? | Add criteria |
| **Exploration** | 2+ approaches compared? | Add alternatives |
| **Synthesis** | Clear recommendation with reasoning? | Make decision |
| **Externalization** | Investigation produced? Feature list reviewed? | Complete outputs |

### Self-Review Checklist

#### 1. Problem Framing Quality
- [ ] **Question clear** - Specific design question stated
- [ ] **Criteria defined** - Know what good looks like
- [ ] **Constraints identified** - Technical, business, time
- [ ] **Scope bounded** - In/out clearly stated

#### 2. Exploration Quality
- [ ] **2+ approaches explored** - Not just one option
- [ ] **Trade-offs documented** - Pros/cons for each
- [ ] **Evidence gathered** - Codebase research, external sources
- [ ] **Complexity assessed** - Effort/risk for each approach

#### 3. Synthesis Quality
- [ ] **Recommendation clear** - Not "it depends"
- [ ] **Reasoning explicit** - Why this over alternatives
- [ ] **Trade-offs acknowledged** - What we're sacrificing
- [ ] **Change conditions noted** - When recommendation would change
- [ ] **Principle cited** - Which principle guides this recommendation

#### 4. Externalization Quality
- [ ] **Investigation produced** - In `.kb/investigations/` (with `design-` prefix)
- [ ] **Feature list reviewed** - Validated, decomposed, cleaned
- [ ] **All committed** - Artifacts in git

---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kn decide` | `kn decide "Use event sourcing" --reason "Need full audit trail"` |
| Tried something that failed | `kn tried` | `kn tried "CQRS pattern" --failed "Too complex for current team size"` |
| Discovered a constraint | `kn constrain` | `kn constrain "Must support offline mode" --reason "Field workers without connectivity"` |
| Found an open question | `kn question` | `kn question "Should we use GraphQL or REST for the new API?"` |

**Quick checklist:**
- [ ] Reflected on session: What did I learn that the next agent should know?
- [ ] Externalized at least one item via `kn` command

**If nothing to externalize:** Note in completion comment: "Leave it Better: Design recommendations captured in investigation, no additional knowledge to externalize."

---

## Completion Criteria

Before marking complete:

- [ ] All 4 phases completed
- [ ] Self-review passed
- [ ] **Leave it Better completed** - At least one `kn` command run OR noted as not applicable
- [ ] Clear recommendation made (not "it depends")
- [ ] Investigation produced in `.kb/investigations/` (with `design-` prefix)
- [ ] Investigation file has `**Phase:** Complete` (required for orch complete verification)
- [ ] Feature list reviewed (mandatory for every session)
- [ ] All changes committed to git
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Complete - [recommendation summary]"`
- [ ] Close the beads issue: `bd close <beads-id> --reason "recommendation summary"`
- [ ] Call /exit to close agent session

**If ANY unchecked, architect work is NOT complete.**

---

## Related Skills

- **investigation** - Use when "how does X work?" (understand, not design)
- **research** - Use for external technology comparisons
- **record-decision** - Use when decision is already made, just documenting
- **feature-impl** - Use after Architect produces actionable design

**Note:** For early-stage ideation, use architect with interactive mode (`orch spawn architect -i`). This provides brainstorming-style collaboration with the user present in the tmux window.


---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-b0ql "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`

⚠️ Your work is NOT complete until you run both commands.
