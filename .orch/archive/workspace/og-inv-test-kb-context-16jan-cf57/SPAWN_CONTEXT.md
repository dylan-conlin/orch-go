TASK: test the new kb context categories

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "test context categories"

### Constraints (MUST respect)
- Agents must not spawn more than 3 iterations without human review
  - Reason: Prevents runaway iteration loops like 12 tmux fallback tests in 9 minutes
- External integrations require manual smoke test before Phase: Complete
  - Reason: OAuth feature shipped with passing tests but failed real-world use. Tests couldn't cover actual OAuth flow with Anthropic.
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- Worker agents must test spawn functionality via unit tests and code review, not end-to-end spawning
  - Reason: Prevents recursive spawn testing incidents while still enabling verification
- kb context command hangs on some queries
  - Reason: Blocks orch spawn from returning, use --skip-artifact-check as workaround

### Prior Decisions
- orch spawn context delivery is reliable
  - Reason: Verified that SPAWN_CONTEXT.md is correctly populated and accessible by the agent
- Refactor shell-out functions for testability
  - Reason: Extracting command construction into Build*Command functions allows unit testing of logic without side effects
- Investigations live in .kb/ not workspaces
  - Reason: kb context discoverability essential; SYNTHESIS.md bridges via investigation_path pointer
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- Failure artifact capture should be layered: automatic events, optional reason flag, manual post-mortems
  - Reason: Balance between context capture and workflow efficiency - most abandons don't need detailed post-mortems
- D.E.K.N. is universal handoff structure
  - Reason: Delta/Evidence/Knowledge/Next enables 30-second context transfer between Claude instances - proven across SYNTHESIS.md, investigations, and session handoffs
- Temporal density and repeated constraints are highest value reflection signals
  - Reason: Low noise, high actionability - tested against real kn/kb data
- Constraint validity tested by implementation, not age
  - Reason: Constraints become outdated when code contradicts them (implementation supersession), not when time passes. Test with rg before assuming constraint holds.
- Pre-spawn kb context should filter to orch ecosystem repos
  - Reason: 33% of global results are noise from unrelated repos (price-watch, dotfiles). Filtering preserves cross-repo signal while eliminating noise.
- SPAWN_CONTEXT.md is 100% redundant - generated from beads + kb context + skill + template
  - Reason: Investigation confirmed all content exists elsewhere and can be regenerated at spawn time
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

### Related Investigations
- Final Test Installed Binary
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-final-test-installed-binary.md
- Test Verify Daemon Skip Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-test-verify-daemon-skip-functionality.md
- Test Fix Nested Skillc
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-test-fix-nested-skillc.md
- Test Orch Go Directory
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-22-inv-test-orch-go-directory.md
- Test Spawn Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-test-spawn-functionality.md
- Test Hotspot Warning Cmd Orch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-test-hotspot-warning-cmd-orch.md
- Pre-Spawn KB Context Noise Filtering
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md
- Harden Flaky Time Based Tests
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-harden-flaky-time-based-tests.md
- Fix Testbuildopencodeattachcommand Test Expects Attach
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-fix-testbuildopencodeattachcommand-test-expects-attach.md
- Test Coaching Plugin Pattern Detection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-16-inv-orch-go-investigation-test-coaching.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.





📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.


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
2. **SET UP investigation file:** Run `kb create investigation test-kb-context-categories` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-test-kb-context-categories.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-test-kb-context-16jan-cf57/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


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
<!-- Checksum: af700a3c0c1d -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-15 14:43:36 -->

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
3. **TOOL EXPERIENCE CHECK (before elaborate investigation):**
   - **Ask Dylan about tools/approaches:** "Have you used [tool] before? What's your experience with [approach]?"
   - **Examples:**
     - "Have you used browser DevTools for debugging this type of issue before?"
     - "What's your debugging workflow for [domain] - quick test first or comprehensive analysis?"
     - "Have you used [specific library/framework] in past projects?"
   - **Why:** System doesn't know Dylan's tool history. Dylan may have simple solutions (DevTools, familiar tools) that avoid elaborate investigation theater.
   - **Reference:** `.kb/investigations/2026-01-09-inv-trust-calibration-meta-pattern.md`
4. **TEST-FIRST GATE (before writing hypotheses):**
   - **Ask yourself:** "What's the simplest test I can run right now?"
   - **60-second rule:** Can I test this in 60 seconds or less?
   - **If YES:** Run the test immediately, document the result
   - **If NO:** Break down the question into smaller testable parts
   - **⚠️ Warning:** Don't dive into documentation or write elaborate hypotheses before attempting a quick test
   - **Example:** Instead of reading 500 lines of SvelteKit docs, open DevTools and check the network tab (30 seconds)
5. Try things, observe what happens (add findings progressively)
6. **Run a test to validate your hypothesis**
7. Fill conclusion only if you tested
8. Final commit

**Why the immediate checkpoint?** Agents can die from API errors, context limits, or crashes. Without a checkpoint, you leave only an empty template with no record of what was attempted.

## Error Recovery

**If you encounter a fatal error during exploration:**

1. **Before doing anything else**, add a finding to your investigation file:
   ```markdown
   ### Finding N: ERROR - [brief description]
   
   **Error:** [Full error message]
   
   **Context:** [What you were attempting when error occurred]
   
   **Significance:** [Why this blocks progress or what it reveals]
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



1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.
