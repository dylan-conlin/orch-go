TASK: Quick test: List the files in .opencode/plugin/ and summarize what each plugin does in 1 sentence each. This is a verification test - complete quickly.


SPAWN TIER: light

⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "quick test list"

### Constraints (MUST respect)
- Agents must not spawn more than 3 iterations without human review
  - Reason: Prevents runaway iteration loops like 12 tmux fallback tests in 9 minutes
- External integrations require manual smoke test before Phase: Complete
  - Reason: OAuth feature shipped with passing tests but failed real-world use. Tests couldn't cover actual OAuth flow with Anthropic.
- Worker agents must test spawn functionality via unit tests and code review, not end-to-end spawning
  - Reason: Prevents recursive spawn testing incidents while still enabling verification

### Prior Decisions
- Refactor shell-out functions for testability
  - Reason: Extracting command construction into Build*Command functions allows unit testing of logic without side effects
- Temporal density and repeated constraints are highest value reflection signals
  - Reason: Low noise, high actionability - tested against real kn/kb data
- Constraint validity tested by implementation, not age
  - Reason: Constraints become outdated when code contradicts them (implementation supersession), not when time passes. Test with rg before assuming constraint holds.
- beads output format uses ' - ' separator for title
  - Reason: All bd commands (list, ready) use this format: {beads-id} [{priority}] [{type}] {status} ... - {title}
- When spawned for cross-repo work, verify work completion status before starting
  - Reason: Task orch-go-oo1f: spawned in orch-go for work in orch-knowledge. Template was already retired (commit 7430185) before agent fully engaged. Quick verification could have saved agent context.
- Tmux is the default spawn mode in orch-go, not headless
  - Reason: Testing and code inspection confirmed tmux is default (main.go:1042), CLAUDE.md documentation was incorrect
- Spawn system verified functional for basic use cases
  - Reason: Test spawn successfully created workspace, loaded context, created investigation file via kb CLI
- kb context uses keyword matching, not semantic understanding - 'how would the system recommend...' questions require orchestrator synthesis
  - Reason: Tested kb context with various query formats: keyword queries (swarm, dashboard) returned results, but semantic queries (swarm map sorting, how should dashboard present agents) returned nothing. The pattern reveals desire for semantic query answering that would require LLM-based RAG.
- UI changes in web/ require visual verification via Playwright MCP screenshot
  - Reason: Tests passing does not guarantee UI renders correctly. Visual evidence ensures feature works as intended.
- SHOULD-HOW-EXECUTE sequence for strategic questions
  - Reason: Epic orch-go-erdw was created from 'how do we' without testing premise. Architect found premise wrong. Wasted work avoided if 'should we' asked first.

### Models (synthesized understanding)
- Completion Verification Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification.md
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
- Beads Integration Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
- Escape Hatch Visibility Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/escape-hatch-visibility-architecture.md
- Phase 3 Review: Model Pattern Analysis (N=5)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE3_REVIEW.md
- Models
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/README.md
- OpenCode Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-session-lifecycle.md
- {Title}
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/TEMPLATE.md
- Agent Completion Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-lifecycle.md
- Daemon Autonomous Operation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation.md

### Guides (procedural knowledge)
- Completion Gates
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion-gates.md
- Code Extraction Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/code-extraction-patterns.md
- API Development Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/api-development.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Tmux Spawn Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/tmux-spawn-guide.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Beads Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/beads-integration.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- OpenCode Plugin System Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode-plugins.md
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md

### Related Investigations
- Test Verify Daemon Skip Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-test-verify-daemon-skip-functionality.md
- Test Orch Go Directory
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-22-inv-test-orch-go-directory.md
- Test Fix Nested Skillc
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-test-fix-nested-skillc.md
- Test Hotspot Warning Cmd Orch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-test-hotspot-warning-cmd-orch.md
- Fix Testbuildopencodeattachcommand Test Expects Attach
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-fix-testbuildopencodeattachcommand-test-expects-attach.md
- tmux concurrent zeta - 6th concurrent spawn test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-zeta.md
- Test Coaching Plugin Pattern Detection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-16-inv-orch-go-investigation-test-coaching.md
- Test Spawn Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-test-spawn-functionality.md
- Harden Flaky Time Based Tests
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-harden-flaky-time-based-tests.md
- Final Test Installed Binary
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-final-test-installed-binary.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.


1. Run: `/exit` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.



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
2. **SET UP investigation file:** Run `kb create investigation quick-test-list-files-opencode` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-quick-test-list-files-opencode.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. ⚡ SYNTHESIS.md is NOT required (light tier spawn).


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



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
<!-- Checksum: b17aa8f7412f -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-17 01:46:54 -->

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

- **Primary** (authoritative): Actual code, test output, observed behavior → This IS the evidence
- **Secondary** (claims to verify): Workspaces, investigations, decisions → Hypotheses to test

When an artifact says "X is not implemented," that's a hypothesis—search the codebase before concluding.

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for evidence hierarchy examples and common failures.

## Workflow

1. **Create file:** `kb create investigation {slug}`
2. **IMMEDIATE CHECKPOINT:** Fill Question, add Finding 1 ("Starting approach"), commit immediately
3. **TEST-FIRST GATE:** "What's the simplest test I can run right now?" (60-second rule)
4. Try things, observe what happens (add findings progressively)
5. **Run a test to validate your hypothesis**
6. Fill conclusion only if you tested
7. Final commit

**Why checkpoint immediately?** Agents can die from API errors, context limits, or crashes. Without a checkpoint, no record of what was attempted.

**Reference:** See `~/.claude/skills/worker/investigation/reference/error-recovery.md` for handling fatal errors during exploration.

## D.E.K.N. Summary

**Every investigation file starts with a D.E.K.N. summary block.** Enables 30-second handoff to fresh Claude.

- **Delta:** What was discovered/answered
- **Evidence:** Primary evidence supporting conclusion
- **Knowledge:** What was learned (insights, constraints)
- **Next:** Recommended action

**Fill D.E.K.N. at the END, before marking Complete.**

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for D.E.K.N. examples.

## Template

Use `kb create investigation {slug}` to create. Required sections:
- **D.E.K.N. Summary** (Delta, Evidence, Knowledge, Next)
- **Question** and **Status**
- **Findings** (add progressively)
- **Test performed** (not "reviewed code" - actual test)
- **Conclusion** (only if you tested)

**Reference:** See `~/.claude/skills/worker/investigation/reference/template.md` for full structure and `reference/examples.md` for common failures.

## When Not to Use

- **Fixing bugs** → Use `systematic-debugging`
- **Trivial questions** → Just answer them
- **Documentation** → Use `capture-knowledge`

## Self-Review (Mandatory)

Before completing, verify investigation quality:

### Self-Review Checklist

- [ ] **Test is real** - Ran actual command/code, not just "reviewed"
- [ ] **Evidence concrete** - Specific outputs, not "it seems to work"
- [ ] **Conclusion factual** - Based on observed results, not inference
- [ ] **No speculation** - Removed "probably", "likely", "should" from conclusion
- [ ] **Question answered** - Investigation addresses the original question
- [ ] **File complete** - All sections filled (not "N/A" or "None")
- [ ] **D.E.K.N. filled** - Replaced placeholders in Summary section
- [ ] **Scope verified** - Ran `rg` to find all occurrences before concluding
- [ ] **NOT DONE claims verified** - If claiming incomplete, searched actual code

### Discovered Work

If you found bugs, tech debt, or enhancement ideas during investigation:
- Create beads issues: `bd create "description" --type bug|task|feature`
- Apply label: `bd label <id> triage:ready` or `triage:review`

**If no discoveries:** Note "No discovered work items" in completion comment.

**Reference:** See `~/.claude/skills/worker/investigation/reference/self-review-guide.md` for scope verification examples and discovered work procedures.

**Only proceed to commit after self-review passes.**

---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**
- `kb quick decide "X" --reason "Y"` (made a choice)
- `kb quick tried "X" --failed "Y"` (something failed)
- `kb quick constrain "X" --reason "Y"` (found a constraint)
- `kb quick question "X"` (open question)

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."

**Reference:** See `~/.claude/skills/worker/investigation/reference/leave-it-better.md` for command examples.

---

## Completion

1. Self-review passed
2. Leave it Better completed (or noted why N/A)
3. D.E.K.N. summary filled (with **Promote to Decision** flag)
4. Report: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"` (FIRST - before commit)
5. Commit: `git add && git commit`
6. Exit: `/exit`

**Why report before commit?** If agent dies after commit but before reporting, orchestrator cannot detect completion.

---

**Remember:** Test before concluding.



---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.



1. `/exit`

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.


⚠️ Your work is NOT complete until you run these commands.
