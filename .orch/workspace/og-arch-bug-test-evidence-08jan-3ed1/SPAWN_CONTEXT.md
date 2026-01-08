TASK: Bug: test_evidence gate triggers on markdown-only changes

Gate should skip when only .md files modified. Markdown docs don't need tests.

Fix: In pkg/verify/test_evidence.go, filter file list to exclude .md before checking if code was modified.

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "bug"

### Constraints (MUST respect)
- Untracked spawns (--no-track) generate placeholder beads IDs that fail bd comment commands
  - Reason: Beads IDs like orch-go-untracked-* don't exist in database, causing bd comment to fail with 'issue not found' - this is expected behavior, not a bug
- UI features require browser verification before marking complete
  - Reason: Post-mortem og-work-dashboard-two-modes-27dec: SSR hydration bug passed build/typecheck but failed in browser; agent documented 'not browser-tested' but had no gate to force validation
- High patch density in a single area (5+ fix commits, 10+ conditions, duplicate logic) signals missing coherent model - spawn architect before more patches
  - Reason: Dashboard status logic accumulated 10+ conditions from incremental agent patches without anyone stepping back to design. Each patch was locally correct but globally incoherent. Discovered Jan 4 2026 after weeks of completion bugs.
- force flags should not bypass substantive gates only procedural checks
  - Reason: Bug fix orch-go-599i showed --force was bypassing repro verification. Substantive gates require explicit bypass flags with reasons.
- UI features require browser verification before marking complete
  - Reason: Post-mortem og-work-dashboard-two-modes-27dec: SSR hydration bug passed build/typecheck but failed in browser; agent documented 'not browser-tested' but had no gate to force validation
- High patch density in a single area (5+ fix commits, 10+ conditions, duplicate logic) signals missing coherent model - spawn architect before more patches
  - Reason: Dashboard status logic accumulated 10+ conditions from incremental agent patches without anyone stepping back to design. Each patch was locally correct but globally incoherent. Discovered Jan 4 2026 after weeks of completion bugs.
- Untracked spawns (--no-track) generate placeholder beads IDs that fail bd comment commands
  - Reason: Beads IDs like orch-go-untracked-* don't exist in database, causing bd comment to fail with 'issue not found' - this is expected behavior, not a bug

### Prior Decisions
- Cross-project epics use Option A: epic in primary repo, ad-hoc spawns with --no-track in secondary repos, manual bd close with commit refs
  - Reason: Only working pattern today. Beads multi-repo hydration is read-only aggregation, bd repo commands are buggy.
- orch complete auto-closes tmux window after successful verification
  - Reason: Complete means done - window goes away, beads closes, workspace remains. Prevents phantom accumulation (41 windows today). Debugging escape hatch: don't complete until ready to close.
- Beads multi-repo hydration works correctly in v0.33.2
  - Reason: Config disconnect bug fixed in commit 634c0b93. Prior kn entry about 'buggy v0.29.0' is superseded.
- Dashboard should use progressive disclosure (Active/Recent/Archive sections) for session management
  - Reason: Balances operational visibility (active work always visible) with historical debugging (expand sections as needed) and UI clarity (collapsed sections reduce clutter). Only approach that satisfies all three user contexts: development focus, debugging history, and health monitoring.
- System resource visibility not needed in orchestrator
  - Reason: External monitoring (sketchybar) already provides CPU/memory visibility; high resource usage in orch processes indicates bugs to fix, not normal states to monitor
- snap CLI for debugging capture, Playwright MCP for web validation
  - Reason: Web validation requires browser control (navigate, wait, interact) before capture - snap only provides capture
- Config-as-code via generation + drift detection for external config
  - Reason: Prevents invisible config bugs like plist flag drift by making expected config explicit and detectable. Extends ~/.orch/config.yaml, adds orch config generate and orch doctor --config.
- reflect_issues config default false
  - Reason: The flag that caused the 2-day plist bug - explicit default prevents invisible drift
- System resource visibility not needed in orchestrator
  - Reason: External monitoring (sketchybar) already provides CPU/memory visibility; high resource usage in orch processes indicates bugs to fix, not normal states to monitor
- Cross-project epics use Option A: epic in primary repo, ad-hoc spawns with --no-track in secondary repos, manual bd close with commit refs
  - Reason: Only working pattern today. Beads multi-repo hydration is read-only aggregation, bd repo commands are buggy.
- Dashboard should use progressive disclosure (Active/Recent/Archive sections) for session management
  - Reason: Balances operational visibility (active work always visible) with historical debugging (expand sections as needed) and UI clarity (collapsed sections reduce clutter). Only approach that satisfies all three user contexts: development focus, debugging history, and health monitoring.
- snap CLI for debugging capture, Playwright MCP for web validation
  - Reason: Web validation requires browser control (navigate, wait, interact) before capture - snap only provides capture
- orch complete auto-closes tmux window after successful verification
  - Reason: Complete means done - window goes away, beads closes, workspace remains. Prevents phantom accumulation (41 windows today). Debugging escape hatch: don't complete until ready to close.
- Beads multi-repo hydration works correctly in v0.33.2
  - Reason: Config disconnect bug fixed in commit 634c0b93. Prior kn entry about 'buggy v0.29.0' is superseded.
- Beads OSS Relationship - Clean Slate
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md
- Orchestrator System Resource Visibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md
- Structured Logging for orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-03-structured-logging-orch-go.md
- Synthesis is Strategic Orchestrator Work, Not Spawnable Work
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md
- Observation Infrastructure Principle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-08-observation-infrastructure-principle.md
- debugging Insufficient Balance error when orch usage showed 99% remaining

### Related Investigations
- CLI orch spawn Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md
- Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- Automate Knowledge Sync using Cobra Doc Gen
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-automate-knowledge-sync-using-cobra.md
- KB Search vs Grep Benchmark
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md
- Compare orch-cli (Python) vs orch-go Features
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md
- Expose Strategic Alignment Commands
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-expose-strategic-alignment-commands-focus.md
- Fix bd create output parsing
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-fix-bd-create-output-parsing.md
- Daemon Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-daemon-command.md
- Add Tail Command for Agent Debugging
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-tail-command.md
- Add work command to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-work-command.md
- Gemini 2.0 and Model Arbitrage Alternatives (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-research-gemini-model-arbitrage-alternatives.md
- Tmux Concurrent Epsilon Spawn Capability
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-tmux-concurrent-epsilon.md
- tmux concurrent zeta - 6th concurrent spawn test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-tmux-concurrent-zeta.md
- Update All Worker Skills with 'Leave it Better' Phase
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-update-all-worker-skills-include.md
- Research: DeepSeek and Llama Model Comparison for Arbitrage (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-research-deepseek-llama-arbitrage-comparison.md
- orch send fails silently for tmux-based agents
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-debug-orch-send-fails-silently-tmux.md
- Design: kb reflect Command Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md
- Design: Minimal Artifact Set Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-minimal-artifact-taxonomy.md
- Add tmux fallback for orch status and tail
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-add-tmux-fallback-orch-status.md
- Agents Being Marked Completed in Registry Prematurely
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.




## REPRODUCTION (BUG FIX)

🐛 **This is a bug fix issue.** The fix is verified when the reproduction steps no longer produce the bug.

**Original Reproduction:**
Gate should skip when only .md files modified. Markdown docs don't need tests.

Fix: In pkg/verify/test_evidence.go, filter file list to exclude .md before checking if code was modified.

**Verification Requirement:**
Before marking Phase: Complete, you MUST:
1. Attempt to reproduce the original bug using the steps above
2. Confirm the bug NO LONGER reproduces after your fix
3. Report verification via: `bd comment orch-go-vsdz3 "Reproduction verified: [describe test performed]"`

⚠️ A bug fix is only complete when the original reproduction steps pass.


🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-vsdz3 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-vsdz3 "Phase: Complete - [1-2 sentence summary of deliverables]"`
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

**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Surface it first: `bd comment orch-go-vsdz3 "CONSTRAINT: [what constraint] - [why considering workaround]"`
2. Wait for orchestrator acknowledgment before proceeding
3. The accountability is a feature, not a cost

This applies to:
- System constraints discovered during work (e.g., API limits, tool limitations)
- Architectural patterns that seem inconvenient for your task
- Process requirements that feel like overhead
- Prior decisions (from `kb context`) that conflict with your approach

**Why:** Working around constraints without surfacing them:
- Prevents the system from learning about recurring friction
- Bypasses stakeholders who should know about the limitation
- Creates hidden technical debt

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation bug-test-evidence-gate-triggers` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-bug-test-evidence-gate-triggers.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-vsdz3 "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-arch-bug-test-evidence-08jan-3ed1/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-vsdz3**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-vsdz3 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-vsdz3 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-vsdz3 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-vsdz3 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-vsdz3 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-vsdz3`.

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
description: Strategic design skill for deciding what should exist. Use when design reasoning exceeds quick orchestrator chat. Produces investigations (with recommendations) that can be promoted to decisions. Distinct from investigation (understand what exists) - architect is for shaping the system.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 8dc3d30410b6 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/architect/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/architect/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/architect/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-08 09:49:47 -->


## Summary

**Purpose:** Shape the system through strategic design decisions.

---

# Architect Skill

**Purpose:** Shape the system through strategic design decisions.

---

## Foundational Guidance

**Before making design recommendations, review:** `~/.kb/principles.md`

Key principles for architects:
- **Premise before solution** - "Should we X?" before "How do we X?" Validate direction before designing
- **Evolve by distinction** - When problems recur, ask "what are we conflating?"
- **Coherence over patches** - If 5+ fixes hit the same area, recommend redesign not another patch
- **Evidence hierarchy** - Code is truth; artifacts are hypotheses to verify
- **Session amnesia** - Will this help the next Claude resume?

**Strategic principles:**
- **Perspective is structural** - Hierarchy exists for perspective, not authority. When recommending org/level changes, ensure each level provides viewpoint the level below can't have
- **Escalation is information flow** - When you recommend escalation paths, frame them as "information reaching the right vantage point" not "asking permission"

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

- **Use natural conversation with recommendation** (question tool as fallback)
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

**Use the question tool only if:**
- Dylan seems overwhelmed by options
- Need to force explicit choice (prevent vague "maybe both")
- Structured comparison would clarify decision

**question tool interface:**
```json
{
  "questions": [{
    "question": "Complete question text",
    "header": "Short label (max 12 chars)",
    "options": [
      {"label": "Option (1-5 words)", "description": "Explanation"}
    ]
  }]
}
```
- Make recommended option first with "(Recommended)" in label
- Users can always select "Other" for custom input

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

**Fallback: question tool**
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

## Completion Criteria

Before marking complete:

- [ ] All 4 phases completed
- [ ] Self-review passed
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
2. `bd comment orch-go-vsdz3 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.
