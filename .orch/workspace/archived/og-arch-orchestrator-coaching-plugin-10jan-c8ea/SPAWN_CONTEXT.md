TASK: orchestrator coaching plugin technical design - explore OpenCode plugin API, dashboard integration, metrics persistence, session identification. Produce implementation approach with tradeoffs and data model

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



⚠️ Limited context (15/100) - agent may need to discover patterns during work

## PRIOR KNOWLEDGE (from kb context)

**Query:** "orchestrator coaching plugin"

### Related Investigations
- Orchestrator Coaching Plugin Prototype
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-prototype.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.





📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `/exit` to close the agent session



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
1. Document it in your investigation file: "CONSTRAINT: [what constraint] - [why considering workaround]"
2. Include the constraint and your reasoning in SYNTHESIS.md
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
2. **SET UP investigation file:** Run `kb create investigation orchestrator-coaching-plugin-technical-design` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-orchestrator-coaching-plugin-technical-design.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-arch-orchestrator-coaching-plugin-10jan-c8ea/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



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
<!-- Checksum: 400f5dab4f1e -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/architect/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/architect/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/architect/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-10 00:50:38 -->

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

---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.
