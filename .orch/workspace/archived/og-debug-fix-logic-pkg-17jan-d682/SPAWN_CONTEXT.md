TASK: Fix logic in pkg/registry or spawn_cmd.go that counts idle/completed sessions toward the concurrency limit. Justification: Tactical fix for a specific counting logic error identified during health check (95 idle agents counted as active) (orch-go-tonum)


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "logic pkg registry"

### Constraints (MUST respect)
- tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: If both are stale/missing, fallback fails despite window existing
- tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: Both paths needed for resilience; if both stale/missing, fallback fails despite window existing
- Tmux fallback requires at least one valid path: current registry window_id OR beads ID in window name format [beads-id]
  - Reason: Dual dependency failure causes fallback to fail even when window exists (discovered iteration 5, confirmed iteration 10)
- orch tail tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: Dual-dependency failure causes fallback to fail when both are stale/missing
- orch-go agent state exists in four layers (OpenCode memory, OpenCode disk, registry, tmux)
  - Reason: Each layer has independent lifecycle - cleanup must touch all layers or ghosts accumulate
- Registry is caching layer, not source of truth - all data exists in OpenCode/tmux/beads
  - Reason: Investigation found all registry data can be derived from primary sources

### Prior Decisions
- Registry respawn workflow uses slot reuse pattern
  - Reason: Preserves single-entry-per-ID invariant while enabling abandon→respawn lifecycle
- Registry updates must happen before beads close in orch complete
  - Reason: Prevents inconsistent state where beads shows closed but registry shows active
- Refactor shell-out functions for testability
  - Reason: Extracting command construction into Build*Command functions allows unit testing of logic without side effects
- Session_id stored in workspace file not registry
  - Reason: Co-locates data with workspace, single writer, no lock contention
- Post-registry lifecycle uses 4 state sources: OpenCode sessions, tmux windows, beads issues, workspaces
  - Reason: Registry removed due to false positive completion detection; derived lookups replace central state
- Use inline lineage metadata for project extraction, not centralized registry
  - Reason: Session Amnesia principle requires self-describing artifacts; centralized registry creates fragile external dependency
- orch init delegates .kb/ creation to kb init command
  - Reason: Ensures consistency with kb's own initialization logic and avoids duplicating directory structure knowledge
- Use tmux-centric CLI commands for cross-project server management
  - Reason: Leverages existing port registry and tmuxinator infrastructure, fits developer workflow, delivers immediate value with minimal code (~200 lines)
- Focus drift API uses CheckDrift from focus package
  - Reason: Reuses existing pkg/focus CheckDrift() rather than duplicating drift detection logic
- Skills own domain behavior, spawn owns orchestration infrastructure
  - Reason: Architect review found current separation is correct design, not fragmentation. Skills containing beads/phase logic would reduce portability and violate Compose Over Monolith. See .kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md
- Registry is Spawn-Time Metadata Cache
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-12-registry-is-spawn-cache.md

### Models (synthesized understanding)
- Agent Lifecycle State Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model.md
- Orchestrator Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle.md
- Dashboard Agent Status Calculation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-agent-status.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md
- Agent Completion Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-lifecycle.md
- Beads Integration Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
- Code Extraction Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/extract-patterns.md
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
- Phase 3 Review: Model Pattern Analysis (N=5)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE3_REVIEW.md
- Completion Verification Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification.md

### Guides (procedural knowledge)
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Agent Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/agent-lifecycle.md
- Beads Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/beads-integration.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Code Extraction Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/code-extraction-patterns.md
- API Development Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/api-development.md
- Dashboard Architecture Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard-architecture.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md

### Related Investigations
- Registry Abandonment Workflow Validate Simple
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md
- Registry Usage Audit in orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md
- Synthesize Registry Investigations 11 Synthesis
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-15-inv-synthesize-registry-investigations-11-synthesis.md
- Synthesis: Registry Evolution and Orch Identity
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md
- Agents Being Marked Completed in Registry Prematurely
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md
- orch complete registry status update
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-orch-complete-closes-beads-issue.md
- Phase 3 - Evaluate spawn session_id capture without registry
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md
- Investigate Registry Population Failures Root
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-investigate-registry-population-failures-root.md
- Session Registry Doesn't Update When Orchestrator Workspaces Are Archived
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-session-registry-doesnt-update-orchestrator.md

### Failed Attempts (DO NOT repeat)
- orch clean to remove ghost sessions automatically

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.




## REPRODUCTION (BUG FIX)

🐛 **This is a bug fix issue.** The fix is verified when the reproduction steps no longer produce the bug.

**Original Reproduction:**
'orch spawn' blocks with 'concurrency limit reached' even when 0 agents are in running state. Likely an error in pkg/registry counting logic.

**Verification Requirement:**
Before marking Phase: Complete, you MUST:
1. Attempt to reproduce the original bug using the steps above
2. Confirm the bug NO LONGER reproduces after your fix
3. Report verification via: `bd comment orch-go-tonum "Reproduction verified: [describe test performed]"`

⚠️ A bug fix is only complete when the original reproduction steps pass.


🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-tonum "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-tonum "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
1. Surface it first: `bd comment orch-go-tonum "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation fix-logic-pkg-registry-spawn` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-fix-logic-pkg-registry-spawn.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-tonum "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-debug-fix-logic-pkg-17jan-d682/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-tonum**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-tonum "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-tonum "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-tonum "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-tonum "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-tonum "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-tonum`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (systematic-debugging)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: systematic-debugging
skill-type: procedure
description: Use when encountering any bug, test failure, or unexpected behavior, before proposing fixes - four-phase framework (root cause investigation, pattern analysis, hypothesis testing, implementation) that ensures understanding before attempting solutions
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 210caf2e1407 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/systematic-debugging/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-17 01:46:54 -->


## Summary

Four-phase debugging framework: Root Cause → Pattern Analysis → Hypothesis Testing → Implementation. Core principle: understand before fixing.

---

# Systematic Debugging

## Summary

Four-phase debugging framework: Root Cause → Pattern Analysis → Hypothesis Testing → Implementation. Core principle: understand before fixing.

---

## The Iron Law

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

If you haven't completed Phase 1, you cannot propose fixes.

---

## When to Use

Use for ANY technical issue: test failures, production bugs, unexpected behavior, performance problems, build failures, integration issues.

**Use ESPECIALLY when:**
- Under time pressure (emergencies make guessing tempting)
- "Just one quick fix" seems obvious
- Previous fixes didn't work
- You don't fully understand the issue

---

## Quick Reference

1. Check if specialized technique applies (see patterns section)
2. Check console/logs for errors - error may already be captured
3. Phase 1: Root cause investigation (understand WHAT and WHY)
4. Phase 2: Pattern analysis (working vs broken differences)
5. Phase 3: Hypothesis testing (form and test specific theory)
6. Phase 4: Implementation (failing test, fix root cause, verify)
7. Document and complete

**Red flag:** If thinking "quick fix for now" → STOP, return to Phase 1.


## Error Visibility (BEFORE Phase 1)

Check if errors have already been logged before investigating:

```bash
# Check project-specific error logs
tail -50 *.log 2>/dev/null
# Check build/test output
make test 2>&1 | tail -30
# Check runtime logs (if applicable)
docker logs <container> --tail 50 2>/dev/null
```

**If logs show relevant errors:**
1. Copy error details to investigation file
2. Use as starting point for Phase 1
3. You may already have root cause evidence

**If empty or unhelpful:** Proceed to Phase 1.

---

## Common Debugging Patterns

Before starting Phase 1, identify if a specialized technique applies:

| Pattern | Symptoms | Technique |
|---------|----------|-----------|
| **Deep call stack** | Error deep in execution, origin unclear, data corruption propagated | [techniques/root-cause-tracing.md](techniques/root-cause-tracing.md) |
| **Timing-dependent** | Flaky tests, race conditions, arbitrary timeouts, "works locally fails in CI" | [techniques/condition-based-waiting.md](techniques/condition-based-waiting.md) |
| **Invalid data propagation** | Bad data causes failures far from source, missing validation | [techniques/defense-in-depth.md](techniques/defense-in-depth.md) |

Load the appropriate technique for specialized guidance.


## The Four Phases

Complete each phase before proceeding to next.

### Phase 1: Root Cause Investigation

**Goal:** Understand WHAT and WHY

**Load:** [phases/phase1-root-cause.md](phases/phase1-root-cause.md)

Key activities:
- Read error messages carefully (stack traces completely)
- Reproduce consistently
- Check recent changes AND pattern recognition (whack-a-mole detection)
- In multi-component systems: add diagnostic instrumentation before fixing
- Trace data flow to source

**Success criteria:** You understand root cause, not just symptoms

---

### Phase 2: Pattern Analysis

**Goal:** Identify differences between working and broken

**Load:** [phases/phase2-pattern-analysis.md](phases/phase2-pattern-analysis.md)

Key activities:
- Find working examples in same codebase
- Read reference implementations COMPLETELY (don't skim)
- List every difference, however small
- Understand dependencies and assumptions

**Success criteria:** You know what's different and why it matters

---

### Phase 3: Hypothesis and Testing

**Goal:** Form and test specific hypothesis

**Load:** [phases/phase3-hypothesis-testing.md](phases/phase3-hypothesis-testing.md)

Key activities:
- Form single hypothesis: "I think X is the root cause because Y"
- Test minimally (one variable at a time)
- Verify before continuing - didn't work? Form NEW hypothesis, don't add more fixes

**Success criteria:** Hypothesis confirmed or new hypothesis formed

---

### Phase 4: Implementation

**Goal:** Fix root cause, not symptom

**Load:** [phases/phase4-implementation.md](phases/phase4-implementation.md)

Key activities:
- Create failing test case
- Implement single fix
- **Smoke-test end-to-end** (critical - see below)
- If 3+ fixes failed: question architecture

**Success criteria:** Bug resolved, tests pass, smoke-test confirms real fix

---

## Smoke-Test Requirement

**Before claiming fix is complete, you MUST:**
1. Run the actual failing scenario that triggered debugging
2. Verify expected behavior now occurs
3. Document smoke-test in completion comment

**Valid:** "Bug: CLI crashes on --mcp" → Run `orch spawn --mcp`, verify no crash
**Invalid:** "Unit tests pass" (necessary but not sufficient)

**If cannot smoke-test:** Document WHY in completion comment.


## Visual Debugging Tools

### snap - Screenshot CLI (Recommended)

```bash
snap                          # Capture screen, returns file path
snap list --json              # Find window IDs
snap window "Firefox"         # Capture by app name
snap --json                   # JSON output: {"path": "/path/to/screenshot.png"}
```

**Use for:** Verifying UI state, documenting visual bugs, smoke-testing UI changes.

**Advantage:** Zero context cost (returns file path, not image data).

### Browser Automation

**USE:** Glass MCP - connects to your actual Chrome tabs via DevTools Protocol

**FALLBACK:** Playwright MCP - for headless/CI scenarios

**AVOID:** browser-use MCP - causes context explosion (screenshots, full DOM)

**Decision flow:**
1. Need visual verification? → `snap` (zero context cost)
2. Need browser automation (clicking, typing, DOM inspection)? → Glass MCP (spawned with --mcp glass)
3. Need headless/CI testing? → Playwright MCP
4. Need DevTools console errors? → Glass MCP (glass_page_state tool)

**Glass advantages:**
- Connects to your actual Chrome (not headless)
- Auto-check DevTools console errors
- Inspect live DOM state
- CLI commands for validation gates (glass assert)


## Investigation File (Optional for Simple Bugs)

Investigation files are **recommended** for complex bugs but **optional** for simple fixes.

### When to Create

**Create when:**
- Multi-step root cause analysis needed
- Multiple hypotheses to test
- Findings should be preserved
- Pattern may recur (for synthesis)

**Skip when:**
- Bug is obvious and localized (typo, wrong variable)
- Fix completes in <15 minutes
- Root cause immediately clear from error
- Commit message can fully document fix

### Create Template (if needed)

```bash
kb create investigation "debug/topic-in-kebab-case"
```

**After creating:**
1. Fill Question field with specific bug description
2. Document findings progressively during Phases 1-4
3. Update Confidence and Resolution-Status as you progress
4. Set Resolution-Status when complete (Resolved/Mitigated/Recurring)

### Commits-Only Completion

If skipping investigation file, ensure descriptive commits:
- Include "why" not just "what"
- Example: `fix: handle null session in auth middleware - was causing silent failures when Redis connection dropped`


## Self-Review (Mandatory)

After implementing fix, perform self-review before completion.

### Pattern Scope Verification

**If bug was a pattern that could exist elsewhere:**

```bash
# Check for pattern occurrences
rg "bug_pattern"                    # Should be 0 or documented exceptions
rg "range\(len\(" --type py         # Off-by-one example
rg "timeout.*=.*[0-9]" --type py    # Hardcoded timeout example
```

**Skip if:** Bug was truly one-off (typo, unique logic error).

### Debugging-Specific Checks

| Check | If Failed |
|-------|-----------|
| Root cause addressed (not symptom) | Return to Phase 1 |
| No debug code left (console.log, print) | Remove before commit |
| No temporary workarounds ("TODO: fix properly") | Complete the fix |
| Regression test exists | Add test |
| Investigation documented | Update file |

### Standard Checks

- [ ] No hardcoded secrets
- [ ] No injection vulnerabilities
- [ ] Conventional commit format (`fix:`, `test:`)
- [ ] Atomic commits

### Discovered Work

If you found related bugs, tech debt, or systemic issues:

```bash
bd create "description" --type bug    # or --type task
bd label <id> triage:ready           # or triage:review
```

**Note "No discovered work" in completion if nothing found.**

### Report via Beads

```bash
# If issues found and fixed:
bd comment <beads-id> "Self-review: Fixed [issue summary]"

# If passed:
bd comment <beads-id> "Self-review passed - ready for completion"
```


## Fix-Verify-Fix Cycle (Atomic Debugging)

**Fix + Verify = One Unit of Work**

Do NOT:
- Implement fix → claim complete → wait for new spawn if fails
- "Fix is done, verification is a separate task"

DO:
- Implement fix → verify immediately → if fails, iterate
- Only claim complete when smoke-test passes

### When to Iterate vs Escalate

**Keep iterating if:**
- Verification reveals related issue in same area
- Fix was incomplete but direction correct
- You understand why it failed

**Escalate if:**
- 3+ fix attempts failed (questioning architecture needed)
- Root cause was misidentified (return to Phase 1)
- Issue outside your scope/authority

### Reporting During Iteration

```bash
bd comment <beads-id> "Fix attempt 1: [what tried] - Result: [pass/fail + why]"
bd comment <beads-id> "Fix attempt 2: [refined approach] - Result: [pass/fail]"
# Only when actually working:
bd comment <beads-id> "Phase: Complete - Fix verified via [smoke-test description]"
```

---

## Red Flags - STOP and Follow Process

If you catch yourself thinking:
- "Quick fix for now, investigate later"
- "Just try changing X and see if it works"
- "Skip the test, I'll manually verify"
- "I don't fully understand but this might work"
- "One more fix attempt" (when already tried 2+)
- Each fix reveals new problem in different place

**ALL mean: STOP. Return to Phase 1.**


## Completion Criteria

Before marking complete, verify ALL:

- [ ] **Root cause identified** - Documented in investigation OR commit message
- [ ] **Fix implemented** - Addresses root cause, not symptoms
- [ ] **Tests passing** - Including reproduction test, with **actual test output documented**
- [ ] **Smoke-test passed** - Actual failing scenario now works
- [ ] **Self-review passed** - Pattern scope, no debug code, no workarounds
- [ ] **Discovered work reviewed** - Tracked or noted "No discoveries"
- [ ] **Phase reported with test evidence** - `bd comment <beads-id> "Phase: Complete - Tests: <cmd> - <output>"` (BEFORE final commit)
- [ ] **Git clean** - `git status` shows "nothing to commit"

**If ANY unchecked, work is NOT complete.**

### After All Criteria Met (in this EXACT order)

```bash
# 1. Report phase FIRST (before commit) - prevents agent death race condition
# Include ACTUAL test output, not just "tests passing"
bd comment <beads-id> "Phase: Complete - Root cause: [X], Fix: [Y], Tests: go test ./... - 23 passed, 0 failed"

# 2. Commit any final changes
git add . && git commit -m "fix: [description]"

# 3. Exit
/exit
```

**Test Evidence Requirement:**
- Format: `Tests: <command> - <actual output summary>`
- Good: `Tests: go test ./... - 23 passed, 0 failed`
- Good: `Tests: npm test - 15 specs, 0 failures`
- Bad: `Tests passing` (no command, no numbers)

**Why:** `orch complete` validates test evidence in comments. Vague claims trigger manual verification.

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility.

**Note:** `bd close` is removed from agent responsibilities - only the orchestrator closes issues via `orch complete`.

---

## Fast-Path Alternative

For clearly localized, trivial failures (import path error, undefined name, obvious single-file fix):

Use `quick-debugging` skill instead. It provides lightweight loop with strict escalation back to systematic-debugging if first attempt fails.

**Use quick-debugging when:**
- Scope ≤ 2 files and ≤ 1 hour
- Error points directly at fix location
- No architectural uncertainty

**If 3+ fixes failed:** Question architecture, return to Phase 1.






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md


## LOCAL SERVERS

**Project:** orch-go
**Status:** running

**Ports:**
- **api:** http://localhost:3348
- **web:** http://localhost:5188

**Quick commands:**
- Start servers: `orch servers start orch-go`
- Stop servers: `orch servers stop orch-go`
- Open in browser: `orch servers open orch-go`



🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.



1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-tonum "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.
