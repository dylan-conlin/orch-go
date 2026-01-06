TASK: Extend skill.yaml schema with spawn_requires section

🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-erdw.1 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Run: `bd comment orch-go-erdw.1 "Phase: Complete - [1-2 sentence summary of deliverables]"`
2. Run: `/exit` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.


CONTEXT: [See task description]

ARCHITECTURE CONTEXT:
- **Orchestration Pattern:** Per-project orchestrators (Architecture B)
  - Multiple `.orch/` directories across projects (meta-orchestration, price-watch, context-driven-dev, etc.)
  - Each project has independent orchestration context
  - Dylan switches contexts via `cd` - not managing all projects from one instance
  - When in `/project-name/`, you ARE that project's orchestrator
- **Key Architectural Constraints:**
  - Projects are architecturally independent (loose coupling)
  - Cross-project dependencies = exception, not rule
  - Shared concerns extracted to libraries, not coordinated via meta-orchestrator

⚠️ **META-ORCHESTRATION TEMPLATE SYSTEM** (Critical if working on meta-orchestration):

**IF task involves these files/patterns:**
- .orch/CLAUDE.md updates
- Orchestrator guidance changes
- Pattern/workflow documentation
- Any file with <!-- ORCH-TEMPLATE: ... --> markers

**THEN you MUST understand the template build system:**

**Template Architecture (3 layers):**
1. **Source:** templates-src/orchestrator/*.md ← EDIT HERE
2. **Distribution:** ~/.orch/templates/orchestrator/*.md (synced via `orch build-global`)
3. **Consumption:** .orch/CLAUDE.md (rebuilt via `orch build --orchestrator`)

**Critical Rules:**
- ❌ NEVER edit .orch/CLAUDE.md sections between `<!-- ORCH-TEMPLATE: ... -->` markers
- ✅ ALWAYS edit source in templates-src/orchestrator/
- ✅ ALWAYS rebuild: `orch build-global && orch build --orchestrator`

**Before editing ANY file:**
```bash
grep "ORCH-TEMPLATE\|Auto-generated" <file>
```

**If file has template markers:**
1. Find source template path in the Auto-generated comment
2. Edit templates-src/orchestrator/[template-name].md
3. Run: `orch build-global` (sync source → distribution)
4. Run: `orch build --orchestrator` (regenerate .orch/CLAUDE.md)
5. Verify changes appear in .orch/CLAUDE.md

**Files that are NOT templates (safe to edit directly):**
- docs/*.md
- tools/orch/*.py
- templates-src/ files (these ARE the source)

**Why this matters:**
- Changes to template-generated sections get SILENTLY OVERWRITTEN on next build
- This is a recurring amnesia bug (see post-mortem: .orch/knowledge/spawning-lessons/2025-11-20-forgot-template-system-context-recurring.md)

**Reference:** .orch/CLAUDE.md lines 77-125 for template system documentation

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

⛔ **NEVER spawn other agents.** Only orchestrators can spawn. If your task involves testing spawn functionality, simulate or mock it - do not actually spawn agents. Recursive spawning exhausts rate limits and creates chaos.

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **REPORT phase via beads:** `bd comment <beads-id> "Phase: Planning - [task description]"`
   - This is your primary progress tracking mechanism
   - Orchestrator monitors via `bd show <beads-id>`
3. **REPORT progress via beads:**
   - Use `bd comment <beads-id>` for phase transitions and milestones
   - Report blockers immediately: `bd comment <beads-id> "BLOCKED: [reason]"`
   - Report questions: `bd comment <beads-id> "QUESTION: [question]"`
4. Report phase transitions via `bd comment <beads-id> "Phase: [phase] - [details]"`
5. [Task-specific deliverables]

STATUS UPDATES (CRITICAL):
Report phase transitions via `bd comment <beads-id>`:
- Phase: Planning
- Phase: Implementing
- Phase: Complete → then call /exit to close agent session

Signal orchestrator when blocked:
- `bd comment <beads-id> "BLOCKED: [reason]"`
- `bd comment <beads-id> "QUESTION: [question]"`

Orchestrator monitors via `bd show <beads-id>` (reads beads comments)

## PRIOR INVESTIGATIONS (from kb)

*Relevant investigations and decisions discovered. Review for context.*

### Replace orch-knowledge with skillc for Skill Management
- **Path:** `.kb/investigations/2025-12-22-inv-epic-replace-orch-knowledge-skillc.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _1. **Extend skillc** to handle SKILL.md-specific patterns (frontmatter preservation or explicit fron..._

### Re-Investigate Skillc vs Orch Build Skills Relationship
- **Path:** `.kb/investigations/2025-12-22-inv-re-investigate-skillc-vs-orch.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _- **Path B (extend skillc):** Add template expansion to skillc. Would need new manifest syntax (`..._

### Should We Evolve Skills to Be Where True Value Resides?
- **Path:** `.kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _> 1. Extend skill.yaml schema with spawn_requires section_

### How Do We Evolve Skills to Be Where True Value Resides?
- **Path:** `.kb/investigations/2025-12-25-inv-epic-question-how-do-we.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _**Next:** Create epic with 4 children: (1) Extend skill.yaml schema for spawn requirements, (2) Migr..._
  - _1. **Extend skill.yaml schema** - Add spawn_requires section (authority, kb_context, servers, beads_..._

*If these investigations are relevant, read the full files for detailed context.*


## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-erdw.1**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-erdw.1 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-erdw.1 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-erdw.1 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-erdw.1 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-erdw.1 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-erdw.1`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## ADDITIONAL CONTEXT

BEADS ISSUE: orch-go-erdw.1

Issue Description:
## Context

Part of Epic: orch-go-erdw (Skill-Manifest-Driven Orchestration)

## Task

Extend the skill.yaml manifest schema to include a spawn_requires section that declares what context/behavior the skill needs at spawn time.

### Proposed Schema Addition

```yaml
spawn_requires:
  authority_level: worker  # worker|elevated|orchestrator
  kb_context: true         # whether to gather kb context before spawn
  beads_tracking: true     # whether to use beads for progress tracking
  servers: auto            # none|auto|required (when to inject server context)
  synthesis: full          # none|light|full (SYNTHESIS.md requirement)
  phase_reporting: true    # whether to inject phase reporting instructions
```

### Files to Modify

- skillc/pkg/compiler/manifest.go - Add SpawnRequires struct and field
- Document schema in skillc README or skill.yaml examples

### Acceptance Criteria

- [ ] New fields parse correctly in skill.yaml
- [ ] Backward compatible with existing skills (missing spawn_requires = defaults)
- [ ] At least one skill updated to use new fields (investigation)
- [ ] skillc build still works with new schema




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
<!-- Checksum: fcf3ce2e311e -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2025-12-24 09:37:07 -->


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
2. Fill in your question
3. Try things, observe what happens
4. **Run a test to validate your hypothesis**
5. Fill conclusion only if you tested
6. Commit

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
5. D.E.K.N. summary filled (Delta, Evidence, Knowledge, Next)
6. `git add` and `git commit` the investigation file
7. Link artifact to beads issue: `kb link <investigation-file> --issue <beads-id>`
8. Report via beads: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"`
9. Close the beads issue: `bd close <beads-id> --reason "conclusion summary"`
10. Run `/exit` to close session

---

**Remember:** The old investigation system produced confident wrong conclusions. The fix is simple: test before concluding.






---


ADDITIONAL DELIVERABLES:
- workspace:  (REQUIRED)

WORKSPACE DIR: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-extend-skill-yaml-25dec
(Use `bd comment orch-go-erdw.1` for progress tracking)


VERIFICATION REQUIRED:
- [ ] Findings documented in investigation file
- [ ] If next-actions identified: Listed in Next-Actions section
- [ ] If "no fix needed": Reason documented

IMPORTANT: Ensure these requirements are met before reporting Phase: Complete via `bd comment`.

CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md
- CDD: ~/orch-knowledge/docs/cdd-essentials.md
- Process guide: ~/.claude/skills/worker/investigation/SKILL.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. `bd comment orch-go-erdw.1 "Phase: Complete - [1-2 sentence summary]"`
2. `/exit`

⚠️ Your work is NOT complete until you run both commands.