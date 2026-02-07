TASK: Say hello and exit - testing coaching plugin worker detection



SPAWN TIER: light

⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "say hello exit"

### Constraints (MUST respect)
- orch status can show phantom agents (tmux windows where OpenCode exited)
  - Reason: No reconciliation between tmux liveness and OpenCode session state

### Prior Decisions
- Add Leave it Better after self-review, before completion criteria
  - Reason: Natural position in workflow - self-review validates quality, Leave it Better captures knowledge, completion gates exit

### Models (synthesized understanding)
- Orchestrator Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle.md
- Decidability Graph
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/decidability-graph.md
- Agent Lifecycle State Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model.md
- Escape Hatch Visibility Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/escape-hatch-visibility-architecture.md
- Follow Orchestrator Mechanism
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/follow-orchestrator-mechanism.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md

### Guides (procedural knowledge)
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md
- Recovery Playbooks
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/recovery-playbooks.md
- orch CLI Reference
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/cli.md
- Agent Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/agent-lifecycle.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- Completion Gates
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion-gates.md
- Completion Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion.md
- Decision Authority Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/decision-authority.md
- Development Environment Setup
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dev-environment-setup.md
- OpenCode Integration Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode.md

### Related Investigations
- say hello and exit immediately
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/simple/2025-12-19-say-hello-exit-immediately.md
- Say Hello Exit Immediately
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-say-hello-exit-immediately.md
- CI - Implement Role-Aware Injection in session-start.sh
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-ci-implement-role-aware-injection.md
- Cli Output Not Appearing Orch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-cli-output-not-appearing-orch.md
- Session Resume Discovery Failure Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-13-design-session-resume-discovery-failure.md
- OpenCode Server Keeps Crashing/Dying - Agent Session Loss
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md
- Oc Command Opencode Dev Wrapper
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-oc-command-opencode-dev-wrapper.md
- orch reconcile --fix Doesn't Close Zombie Issues
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-orch-reconcile-fix-doesn-actually.md
- Research: macOS Sequoia Chrome code_sign_clone Disk Space Issue
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-25-inv-research-macos-sequoia-chrome-code-sign-clone.md
- How Do Investigation Files Become Stale and Mislead Agents
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-how-do-investigation-files-become-stale.md

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

2. [Task-specific deliverables from skill guidance]


3. ⚡ SYNTHESIS.md is NOT required (light tier spawn).



Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



## SKILL GUIDANCE (hello)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: hello
skill-type: procedure
audience: worker
spawnable: true
category: test
description: Simple test skill that prints hello and exits
allowed-tools:
- Bash
deliverables: []
verification:
  requirements: |
    - [ ] Agent printed "Hello from orch-go!"
  test_command: null
  required: true
---

# Hello Skill

**Purpose:** Test the spawn system with a trivial task.

## Directive

When spawned with this skill, print the following message exactly:

```
Hello from orch-go!
```

Then exit immediately by running `/exit`.

## Notes

- This is a minimal test skill to verify spawn functionality.
- No other actions are required.
- The verification is simply that the message was printed.

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
