TASK: Synthesize spawn investigations

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "synthesize"

### Prior Decisions
- Investigations already do semantic query answering (kb context → read → synthesize). The 'recommend' pattern desire is for SYNCHRONOUS/INLINE answers without spawn overhead, not new capability.
  - Reason: Dylan's reframe: every investigation spawn does exactly this. The friction is 30s+ spawn latency vs 5s inline answer. Consider 'kb ask' for mini-investigations without artifact overhead.
- Dashboard gets lightweight acknowledgment actions (approve, reject, mark reviewed, priority bump); orchestrator keeps reasoning actions (spawn, synthesize, scope)
  - Reason: Control separation: if it requires judgment, orchestrator. If it's confirmation of something already decided, dashboard. Reduces context switching without blurring the thinking/seeing boundary.
- Accumulated investigations should be synthesized into guides when 10+ exist on a topic
  - Reason: Guides provide single authoritative reference vs scattered investigations
- Synthesize investigations when 10+ accumulate on a topic
  - Reason: Per kb context pattern: accumulated investigations should be synthesized into guides when 10+ exist on a topic. Guides provide single authoritative reference vs scattered investigations.
- Synthesize 10+ investigations into single guide
  - Reason: Guides provide single authoritative reference vs scattered investigations; synthesis threshold of 10+ validated as useful trigger for consolidation
- Synthesize investigations into guides when 10+ exist on a topic
  - Reason: Guides provide single authoritative reference vs scattered investigations; 44 dashboard investigations proved this threshold is overdue
- Daemon guide is the single authoritative reference
  - Reason: Synthesized 31 investigations into one guide; prevents future agents from re-investigating solved problems
- Synthesize investigations into guides at 10+ threshold
  - Reason: 17 beads investigations consolidated into single authoritative guide; provides entry point with evidence trail via cross-references
- Synthesize investigations when 17+ exist on a topic into a guide
  - Reason: 17 agent investigations were successfully consolidated into single authoritative reference; guides provide single source of truth vs scattered investigations
- CLI guide at .kb/guides/cli.md is authoritative reference
  - Reason: Synthesized 16 scattered investigations into single reference; future CLI questions start here, not with new investigations
- OpenCode guide at .kb/guides/opencode.md is authoritative reference
  - Reason: Synthesized 16 investigations into single guide; future agents should read guide before spawning OpenCode investigations
- Synthesize investigations into guides when 10+ exist on a topic
  - Reason: 15 headless investigations successfully consolidated into single authoritative guide, reducing future agent research time
- Strategic Orchestrator Model: orchestrator's three jobs are COMPREHEND, TRIAGE, SYNTHESIZE (not DELEGATE)
  - Reason: Understanding is orchestrator work. Daemon handles coordination (spawning). Orchestrator builds mental models.
- Minimal Artifact Taxonomy for Amnesia-Resilient Orchestration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md
- Synthesis is Strategic Orchestrator Work, Not Spawnable Work
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md

### Related Investigations
- Compare orch-cli (Python) vs orch-go Features
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md
- Gemini 2.0 and Model Arbitrage Alternatives (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-research-gemini-model-arbitrage-alternatives.md
- Scope Out Headless Swarm Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md
- Design: kb reflect Command Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md
- Design: Minimal Artifact Set Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-minimal-artifact-taxonomy.md
- Chronicle Artifact Type Design
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md
- Design: Self-Reflection Protocol Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md
- Multi-Agent Synthesis and Conflict Detection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-multi-agent-synthesis-when-multiple.md
- Orchestrator Session Boundaries
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md
- Reflection Checkpoint Pattern for Agent Sessions
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md
- Temporal Signals for Autonomous Reflection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-temporal-signals-autonomous-reflection.md
- Tracing Confidence Score History and Effectiveness
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-trace-confidence-score-effectiveness.md
- Workers Stall During Build Phase
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-workers-stall-during-build-phase.md
- Investigate Git Branching Strategies Swarm
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-investigate-git-branching-strategies-swarm.md
- Full Ecosystem Audit - Scope and Simplify
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md
- Emerging Pattern - "How Would the System Recommend..."
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-investigate-emerging-pattern-how-would.md
- Orchestrator Completion Lifecycle Design
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-25-design-orchestrator-completion-lifecycle-two.md
- Design Kb Ask Command for Inline Mini-Investigations
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-25-inv-design-kb-ask-command-inline.md
- Fix Orch Learn Act Generate
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-25-inv-fix-orch-learn-act-generate.md
- Strategic Question-Asking Sequence
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.





🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-bdfgu "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-bdfgu "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
1. Surface it first: `bd comment orch-go-bdfgu "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation synthesize-spawn-investigations` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-synthesize-spawn-investigations.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-bdfgu "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-synthesize-spawn-investigations-07jan-c804/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-bdfgu**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-bdfgu "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-bdfgu "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-bdfgu "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-bdfgu "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-bdfgu "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-bdfgu`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (investigation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: investigation
skill-type: procedure
description: Record what you tried, what you observed, and whether you tested. Key discipline - you cannot conclude without testing.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: a1ea3997ce46 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-07 14:44:12 -->


<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation file with findings -->
<!-- /SKILL-CONSTRAINTS -->
## Summary

**Purpose:** Answer a question by testing, not by reasoning.

---

# Investigation Skill

**Purpose:** Answer a question by testing, not by reasoning.

## The One Rule

**You cannot conclude without testing.**

If you didn't run a test, you don't get to fill the Conclusion section.

## Evidence Hierarchy

**Artifacts are claims, not evidence.**

| Source Type | Examples | Treatment |
|-------------|----------|-----------|
| **Primary** (authoritative) | Actual code, test output, observed behavior | This IS the evidence |
| **Secondary** (claims to verify) | Workspaces, investigations, decisions | Hypotheses to test |

When an artifact says "X is not implemented," that's a hypothesis—not a finding to report. Search the codebase before concluding.

**The failure mode:** An agent reads a workspace claiming "feature X NOT DONE" and reports that as a finding without checking if feature X actually exists in the code.


## Workflow

1. Create investigation file: `kb create investigation {slug}`
2. **IMMEDIATE CHECKPOINT (before ANY exploration):**
   - Fill in your **Question** from SPAWN_CONTEXT
   - Add first finding: `### Finding 1: Starting approach` with your planned first step
   - **Commit immediately:** `git add .kb/investigations/*.md && git commit -m "investigation: {slug} - checkpoint"`
   - This ensures if you die, there's a trail of what you were attempting
3. Try things, observe what happens (add findings progressively)
4. **Run a test to validate your hypothesis**
5. Fill conclusion only if you tested
6. Final commit

**Why the immediate checkpoint?** Agents can die from API errors, context limits, or crashes. Without a checkpoint, you leave only an empty template with no record of what was attempted.

## Error Recovery

**If you encounter a fatal error during exploration:**

1. **Before doing anything else**, add a finding to your investigation file:
   ```markdown
   ### Finding N: ERROR - [brief description]
   
   **Error:** [Full error message]
   
   **Context:** [What you were attempting when error occurred]
   
   **Significance:** [Why this blocks progress or what it reveals]
   ```

2. Commit immediately: `git add .kb/investigations/*.md && git commit -m "investigation: {slug} - error encountered"`

3. Report via beads: `bd comment <beads-id> "ERROR: [error summary] - see investigation file"`

4. If error is recoverable, continue. If fatal, the investigation file now has a record of what happened.

**Example errors to document:**
- API rate limits or size limits (e.g., "100 PDF pages max")
- Tool failures or missing dependencies
- Context exhaustion warnings
- External service unavailability

## D.E.K.N. Summary

**Every investigation file starts with a D.E.K.N. summary block at the top.** This enables 30-second handoff to fresh Claude.

| Section | Purpose | Example |
|---------|---------|---------|
| **Delta** | What was discovered/answered | "Test-running guidance is missing from spawn prompts" |
| **Evidence** | Primary evidence supporting conclusion | "Searched 5 agent sessions - none ran tests" |
| **Knowledge** | What was learned (insights, constraints) | "Agents follow documentation literally" |
| **Next** | Recommended action | "Add test-running instruction to template" |

**Fill D.E.K.N. at the END of your investigation, before marking Complete.**


## Template

The template enforces the discipline. Use `kb create investigation {slug}` to create.

```markdown
## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered]
**Evidence:** [Primary evidence supporting conclusion]
**Knowledge:** [What was learned]
**Next:** [Recommended action]
**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Flag for orchestrator/human review

---

# Investigation: [Topic]

**Question:** [What are you trying to figure out?]
**Status:** Active | Complete

## Findings
[Evidence gathered]

## Test performed
**Test:** [What you did to validate]
**Result:** [What happened]

## Conclusion
[Only fill if you tested]
```

## Common Failures

**"Logical verification" is not a test.**

Wrong:
```markdown
## Test performed
**Test:** Reviewed the code logic
**Result:** The implementation looks correct
```

Right:
```markdown
## Test performed
**Test:** Ran `time orch spawn investigation "test"` 5 times
**Result:** Average 6.2s, breakdown: 70ms orch overhead, 5.5s Claude startup
```

**Speculation is not a conclusion.**

Wrong:
```markdown
## Conclusion
Based on the code structure, the issue is likely X.
```

Right:
```markdown
## Conclusion
The test confirmed X is the cause. When I changed Y, the behavior changed to Z.
```

## When Not to Use

- **Fixing bugs** → Use `systematic-debugging`
- **Trivial questions** → Just answer them
- **Documentation** → Use `capture-knowledge`


## Self-Review (Mandatory)

Before completing, verify investigation quality:

### Scope Verification

**Did you scope the problem with rg before concluding?**

| Check | How | If Failed |
|-------|-----|-----------|
| **Problem scoped** | Ran `rg` to find all occurrences of the pattern being investigated | Run now, update findings |
| **Scope documented** | Investigation states "Found X occurrences in Y files" | Add concrete numbers |
| **Broader patterns checked** | Searched for variations/related patterns | Document what else exists |

**Examples:**
```bash
# Investigating "how does auth work?"
rg "authenticate|authorize|jwt|token" --type py -l  # Scope: which files touch auth

# Investigating "why does X fail?"
rg "error.*X|X.*error" --type py  # Find all error handling for X

# Investigating "where is config loaded?"
rg "config|settings|env" --type py -l  # Scope the config surface area
```

**Why this matters:** Investigations that don't scope the problem often miss the full picture. "I found one place that does X" is less useful than "X happens in 3 files: A, B, C."

---

### Investigation-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Real test performed** | Not "reviewed code" or "analyzed logic" | Go back and test |
| **Conclusion from evidence** | Based on test results, not speculation | Rewrite conclusion |
| **Question answered** | Original question has clear answer | Complete the investigation |
| **Reproducible** | Someone else could follow your steps | Add detail |

### Self-Review Checklist

- [ ] **Test is real** - Ran actual command/code, not just "reviewed"
- [ ] **Evidence concrete** - Specific outputs, not "it seems to work"
- [ ] **Conclusion factual** - Based on observed results, not inference
- [ ] **No speculation** - Removed "probably", "likely", "should" from conclusion
- [ ] **Question answered** - Investigation addresses the original question
- [ ] **File complete** - All sections filled (not "N/A" or "None")
- [ ] **D.E.K.N. filled** - Replaced placeholders in Summary section (Delta, Evidence, Knowledge, Next)
- [ ] **NOT DONE claims verified** - If claiming something is incomplete, searched actual files/code to confirm (not just artifact claims)

### Discovered Work Check

*During this investigation, did you discover any of the following?*

| Type | Examples | Action |
|------|----------|--------|
| **Bugs** | Broken functionality, edge cases that fail | `bd create "description" --type bug` |
| **Technical debt** | Workarounds, code that needs refactoring | `bd create "description" --type task` |
| **Enhancement ideas** | Better approaches, missing features | `bd create "description" --type feature` |
| **Documentation gaps** | Missing/outdated docs | Note in completion summary |

*When creating issues for discovered work, apply triage labels:*

| Confidence | Label | When to use |
|------------|-------|-------------|
| High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Lower | `triage:review` | Uncertain scope, needs orchestrator input |

Example:
```bash
bd create "Bug: edge case in validation" --type bug
bd label <issue-id> triage:ready  # or triage:review
```

**Checklist:**
- [ ] **Reviewed for discoveries** - Checked investigation for patterns, bugs, or ideas beyond original scope
- [ ] **Tracked if applicable** - Created beads issues for actionable items (or noted "No discoveries")
- [ ] **Included in summary** - Completion comment mentions discovered items (if any)

**If no discoveries:** Note "No discovered work items" in completion comment. This is common and acceptable.

**Why this matters:** Investigations often reveal issues beyond the original question. Beads issues ensure these discoveries surface in SessionStart context rather than getting buried in investigation files.

### Document in Investigation File

At the end of your investigation file, add:

```markdown
## Self-Review

- [ ] Real test performed (not code review)
- [ ] Conclusion from evidence (not speculation)
- [ ] Question answered
- [ ] File complete

**Self-Review Status:** PASSED / FAILED
```

**Only proceed to commit after self-review passes.**


---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kn decide` | `kn decide "Use Redis for sessions" --reason "Need distributed state"` |
| Tried something that failed | `kn tried` | `kn tried "SQLite for sessions" --failed "Race conditions"` |
| Discovered a constraint | `kn constrain` | `kn constrain "API requires idempotency" --reason "Retry logic"` |
| Found an open question | `kn question` | `kn question "Should we rate-limit per-user or per-IP?"` |

**Quick checklist:**
- [ ] Reflected on session: What did I learn that the next agent should know?
- [ ] Externalized at least one item via `kn` command

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."

---

## Completion

Before marking complete:

1. Self-review passed (see above)
2. **Leave it Better:** At least one `kn` command run OR noted as not applicable
3. `## Test performed` has a real test (not "reviewed code" or "analyzed logic")
4. `## Conclusion` is based on test results
5. D.E.K.N. summary filled (Delta, Evidence, Knowledge, Next, **Promote to Decision**)
6. **Decision promotion flag:** Set to `recommend-yes`, `recommend-no`, or `unclear` - orchestrator/human makes final call
7. Link artifact to beads issue: `kb link <investigation-file> --issue <beads-id>`
8. Report via beads: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"` (report phase FIRST - before commit)
9. `git add` and `git commit` the investigation file
10. Run `/exit` to close session

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Note:** `bd close` is removed from agent responsibilities - only the orchestrator closes issues via `orch complete`.

---

**Remember:** The old investigation system produced confident wrong conclusions. The fix is simple: test before concluding.






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-bdfgu "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.
