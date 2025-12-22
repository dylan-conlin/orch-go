TASK: Fix orch account add OAuth flow. Current bug: uses local callback server but Anthropic only allows their own callback URL. Fix: (1) Change redirect_uri to https://console.anthropic.com/oauth/code/callback, (2) Remove local callback server, (3) Prompt user to paste authorization code, (4) Exchange code for tokens. Reference: opencode-anthropic-auth plugin at /tmp/auth-check/package/index.mjs shows working flow.

## PRIOR KNOWLEDGE (from kb context)

**Query:** "orch"

### Constraints (MUST respect)
- orch tail tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: Dual-dependency failure causes fallback to fail when both are stale/missing
- orch-go DefaultModel should be Opus (claude-opus-4-5-20251101), not Gemini
  - Reason: Orchestrator guidance expects Opus for complex work, current Gemini default conflicts with operational practice
- orch-go agent state exists in four layers (OpenCode memory, OpenCode disk, registry, tmux)
  - Reason: Each layer has independent lifecycle - cleanup must touch all layers or ghosts accumulate
- orch complete must verify SYNTHESIS.md exists and is not placeholder before closing issue
  - Reason: 70% of agents completed without synthesis in 24h chaos period
- orch init must be idempotent - safe to run multiple times
  - Reason: Prevents accidental overwrites and enables 'run init to update' pattern
- orch status can show phantom agents (tmux windows where OpenCode exited)
  - Reason: No reconciliation between tmux liveness and OpenCode session state
- Beads cross-repo contamination can create orphaned FK references
  - Reason: bd-* prefixed dependencies were found in orch-go database from separate beads repo
- Template ownership: kb-cli owns artifact templates (investigation/decision/guide), orch-go owns spawn-time templates (SYNTHESIS/SPAWN_CONTEXT/FAILURE_REPORT)
  - Reason: Prevents drift by establishing clear domains. See .kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md
- orch status counts ALL workers-* tmux windows as active
  - Reason: Discovered during phantom agent investigation - status inflated by persistent windows
- skillc cannot compile SKILL.md templates without template expansion feature
  - Reason: orch-knowledge skills use SKILL-TEMPLATE markers that require regex substitution not concatenation
- orch-knowledge repo is at ~/orch-knowledge (not ~/Documents/personal/orch-knowledge)
  - Reason: Agents kept failing to find it when given relative path. Skill sources live at ~/orch-knowledge/skills/src/worker/{skill}/.skillc/

### Prior Decisions
- orch-go tmux spawn is fire-and-forget - no session ID capture
  - Reason: opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring.
- Registry updates must happen before beads close in orch complete
  - Reason: Prevents inconsistent state where beads shows closed but registry shows active
- orch spawn context delivery is reliable
  - Reason: Verified that SPAWN_CONTEXT.md is correctly populated and accessible by the agent
- orch-go CLI independence
  - Reason: CLI commands connect directly to OpenCode (4096), not orch serve (3333)
- Use build/orch for serve daemon
  - Reason: Prevents SIGKILL during make install
- Orchestrator sessions need SESSION_HANDOFF.md
  - Reason: Session amnesia applies to orchestrator work; skillc pattern provides mature template
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- Multi-agent synthesis relies on workspace isolation + SYNTHESIS.md + orch review
  - Reason: 100 commits, 52 synthesis files, 0 conflicts validates current architecture
- orch-go is primary CLI, orch-cli (Python) is reference/fallback
  - Reason: Go provides better primitives (single binary, OpenCode HTTP client, goroutines); Python taught requirements through 27k lines and 200+ investigations
- Reflection value comes from orchestrator review + follow-up, not execution-time process changes
  - Reason: Evidence: post-synthesis reflection with Dylan created orch-go-ws4z epic (6 children) from captured questions
- Beads OSS: Clean Slate over Fork
  - Reason: Local features (ai-help, health, tree) not used by orch ecosystem. Drop rather than maintain.
- skillc and orch build skills are complementary, not competing
  - Reason: skillc compiles project-local .skillc/ to CLAUDE.md; orch build skills compiles templated skills to ~/.claude/skills/. Different purposes, both needed.
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- Pre-spawn kb context should filter to orch ecosystem repos
  - Reason: 33% of global results are noise from unrelated repos (price-watch, dotfiles). Filtering preserves cross-repo signal while eliminating noise.
- orch complete auto-closes tmux window after successful verification
  - Reason: Complete means done - window goes away, beads closes, workspace remains. Prevents phantom accumulation (41 windows today). Debugging escape hatch: don't complete until ready to close.
- orch status shows PHASE and TASK columns from beads data
  - Reason: Makes output actionable - users can immediately see what each agent is doing
- Template ownership split by domain
  - Reason: kb-cli owns knowledge artifacts (investigation, decision, guide, research); orch-go owns orchestration artifacts (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT, SESSION_HANDOFF)
- orch init delegates .kb/ creation to kb init command
  - Reason: Ensures consistency with kb's own initialization logic and avoids duplicating directory structure knowledge
- ECOSYSTEM.md location is ~/.orch/ not ~/.claude/
  - Reason: Keeps orchestration docs with orchestration state; ~/.claude/ is for Claude-specific config
- Beads OSS Relationship - Clean Slate
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md

### Related Investigations
- CLI orch complete Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-complete-command.md
- CLI orch spawn Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md
- CLI Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-status-command.md
- CLI Project Scaffolding and Build Setup
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-project-scaffolding-build.md
- OpenCode Client Package Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-opencode-session-management.md
- SSE Event Monitoring Client
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md
- Desktop Notifications on Completion
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-desktop-notifications-completion.md
- Fix comment ID parsing - Comment.ID type mismatch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-fix-comment-id-parsing-comment.md
- Set beads issue status to in_progress on spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-set-beads-issue-status-progress.md
- Update README with current CLI commands and usage
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-update-readme-current-cli-commands.md
- OpenCode POC - Spawn Session Via Go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-simple-opencode-poc-spawn-session-via.md
- Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- Synthesis Protocol Design for Agent Handoffs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md
- Add /api/agentlog endpoint to serve.go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-api-agentlog-endpoint-serve.md
- Add CLI Commands for Focus, Drift, and Next
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-cli-commands-focus-drift.md
- Add --dry-run flag to daemon run command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-dry-run-flag-daemon.md
- Add Missing Spawn Flags
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-missing-spawn-flags-no.md
- Add orch review command for batch completion workflow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-orch-review-command-batch.md
- Add Usage/Capacity Tracking to Account Package
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-usage-capacity-tracking-account.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.



🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-u1nt "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-u1nt "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
2. **SET UP investigation file:** Run `kb create investigation fix-orch-account-add-oauth` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-fix-orch-account-add-oauth.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-u1nt "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-debug-fix-orch-account-22dec/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-u1nt**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-u1nt "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-u1nt "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-u1nt "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-u1nt "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-u1nt "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-u1nt`.

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
audience: worker
spawnable: true
category: debugging
description: Use when encountering any bug, test failure, or unexpected behavior,
  before proposing fixes - four-phase framework (root cause investigation, pattern
  analysis, hypothesis testing, implementation) that ensures understanding before
  attempting solutions
deliverables:
- type: investigation
  path: "{project}/.kb/investigations/{date}-debug-{slug}.md"
  required: false
  description: Investigation file (recommended for complex bugs, optional for simple/quick fixes)
- type: commit
  required: false
  description: Fix commit (if root cause identified and fixed)
verification:
  requirements: |
    - [ ] Root cause documented in investigation file
    - [ ] If fix implemented: Verify fix works
      - [ ] Run automated tests if available
      - [ ] AND smoke-test end-to-end (run actual failing scenario)
      - [ ] Document smoke-test in completion comment
    - [ ] No regression introduced
  test_command: null  # Agent specifies via bd comment
  timeout: 300
  required: true
---

# Debugging Process

## Overview

You've been assigned to debug an issue using a systematic approach.

Random fixes waste time and create new bugs. Quick patches mask underlying issues.

**Core principle:** ALWAYS find root cause before attempting fixes. Symptom fixes are failure.

**Violating the letter of this process is violating the spirit of debugging.**

## The Iron Law

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

If you haven't completed Phase 1, you cannot propose fixes.

## When to Use

Use for ANY technical issue:
- Test failures
- Bugs in production
- Unexpected behavior
- Performance problems
- Build failures
- Integration issues

**Use this ESPECIALLY when:**
- Under time pressure (emergencies make guessing tempting)
- "Just one quick fix" seems obvious
- You've already tried multiple fixes
- Previous fix didn't work
- You don't fully understand the issue

**Don't skip when:**
- Issue seems simple (simple bugs have root causes too)
- You're in a hurry (rushing guarantees rework)
- Manager wants it fixed NOW (systematic is faster than thrashing)

## Quick Reference

**Core workflow:**
1. Check debugging patterns (identify if specialized technique applies)
2. Check agentlog errors (`agentlog errors --limit 10`) - may already have the error captured
3. Phase 1: Root cause investigation (understand WHAT and WHY)
4. Phase 2: Pattern analysis (identify differences between working and broken)
5. Phase 3: Hypothesis and testing (form and test specific hypothesis)
6. Phase 4: Implementation (create failing test, fix root cause, verify)
7. Document findings in investigation file (hypothesis + evidence)
8. Complete checklist (root cause identified, tests passing, clean git tree)

**Common pattern:** NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST. Always complete all four phases sequentially. If you find yourself proposing solutions before understanding the root cause, STOP and return to Phase 1.

**Red flag check:** If thinking "quick fix for now" or "just try X" → STOP, return to Phase 1.

---

## Common Debugging Patterns

Before starting Phase 1, identify which pattern matches your debugging situation. Load the appropriate technique for specialized guidance:

### Pattern: Deep Call Stack Errors
**Symptoms:** Error occurs deep in execution, origin unclear, long call chain, data corruption propagated through multiple layers

**Technique:** Read [techniques/root-cause-tracing.md](techniques/root-cause-tracing.md)
- Systematic backward tracing through 5-step process
- Stack trace instrumentation strategies (`new Error().stack`)
- Finding test pollution with bisection
- Real-world example: empty projectDir traced through 5 levels

**When to load:** Error message shows deep call stack, or you can't identify where bad data originated

---

### Pattern: Timing-Dependent Failures
**Symptoms:** Flaky tests, race conditions, arbitrary timeouts, inconsistent pass/fail, "works locally but fails in CI"

**Technique:** Read [techniques/condition-based-waiting.md](techniques/condition-based-waiting.md)
- Replace timeouts with condition polling
- Generic `waitFor()` implementation with proper timeout
- Deterministic async test patterns
- When arbitrary timeout IS correct (documented exceptions)

**When to load:** Tests fail intermittently, or you see `setTimeout()` used for synchronization

---

### Pattern: Invalid Data Deep in System
**Symptoms:** Bad data causes failures far from source, validation missing at boundaries, error handling added reactively

**Technique:** Read [techniques/defense-in-depth.md](techniques/defense-in-depth.md)
- 4-layer validation strategy (entry, business logic, environment, debug)
- Make bugs structurally impossible
- Fail-fast at every boundary
- Real-world example: same projectDir bug caught at 4 different layers

**When to load:** You've fixed the immediate error but want to prevent similar bugs, or fixing symptom deep in stack without addressing root validation issue

---

**If no pattern matches:** Continue with Phase 1 below. You may identify a pattern during investigation.

---

## Visual Debugging Tools

### snap - Screenshot CLI (Recommended for Visual Verification)

**Use snap for visual state verification during debugging:**

```bash
# Capture current screen state
snap                          # Returns file path

# Capture specific window
snap list --json              # Find window IDs
snap window "Firefox"         # Capture by app name

# JSON output for programmatic use
snap --json                   # Returns {"path": "/path/to/screenshot.png"}
```

**When to use snap:**
- Verifying UI state matches expectations
- Documenting visual bugs for investigation file
- Smoke-testing UI changes before claiming complete
- Capturing error dialogs or visual glitches

**Advantages over MCP-based approaches:**
- Zero context cost (returns file path, not image data)
- Fast (native macOS screencapture)
- No browser automation complexity

### Browser Automation (When snap isn't enough)

**Post-mortem learning:** browser-use MCP causes context explosion. Use playwright MCP instead.

**USE: Playwright MCP**
- Lighter weight, focused on automation
- Doesn't blow up context with screenshots/DOM dumps
- Suitable for: clicking, typing, navigating, checking elements

**AVOID: browser-use MCP**
- Causes context explosion (screenshots, full DOM, etc.)
- Will consume most of your context budget
- Only use if explicitly required AND you understand the cost

**IDEAL (if available): Console bridge**
- Project-specific mechanism to surface browser console to agent
- Zero context cost for passive monitoring
- Check if project has this before reaching for MCPs

### Decision flow:
1. Need to verify visual state? → Use `snap`
2. Does project have a console bridge? → Use it for errors
3. Need browser automation (clicking, typing)? → Use playwright MCP
4. Need visual inspection + automation? → Consider if truly necessary, then browser-use as last resort

---

## Investigation File (Optional for Simple Bugs)

Investigation files are **recommended** for complex bugs but **optional** for simple/quick fixes.

### When to Create Investigation File

**Create investigation file when:**
- Bug requires multi-step root cause analysis
- Multiple hypotheses need testing
- Findings should be preserved for future reference
- Pattern may recur (want synthesis workflow)

**Skip investigation file when:**
- Bug is obvious and localized (e.g., typo, wrong variable)
- Fix can be completed in <15 minutes
- Root cause is immediately clear from error message
- Commit message can fully document the fix

### Create Investigation Template (if needed)

```bash
# Create investigation using kb CLI command
# Update SLUG based on your debugging topic
# Use debug/ prefix for debugging investigations
kb create investigation "debug/topic-in-kebab-case"
```

**After creating the template:**
1. Fill Question field with specific bug description from SPAWN_CONTEXT
2. Update metadata (Started date set automatically, verify Status)
3. Document findings progressively during Phase 1-4 (don't wait until end)
4. Update Confidence and Resolution-Status as you make progress

**Important:**
- The `kb create investigation` command auto-detects project directory and creates the investigation in the appropriate subdirectory.
- The investigation file includes Resolution-Status field (Unresolved/Resolved/Recurring/Synthesized/Mitigated) which is critical for the synthesis workflow. Always fill this field when completing the investigation.

### Commits-Only Completion

If you skip the investigation file, ensure your commits are well-documented:
- Use descriptive commit messages explaining the root cause
- Include "why" not just "what" in commit messages
- Example: `fix: handle null session in auth middleware - was causing silent failures when Redis connection dropped`

**Proceed to Error Visibility check, then Phase 1 below.**

---

## Error Visibility (BEFORE Phase 1)

**CRITICAL:** Before starting investigation, check if errors have already been logged. The `agentlog` tool captures errors from agent sessions - these may contain the exact error you need to debug.

### Check Logged Errors

```bash
# View recent errors (most useful starting point)
agentlog errors --limit 10

# Watch errors in real-time (if debugging a running process)
agentlog tail
```

**Why check agentlog first:**
- Errors may already be captured with full context
- Stack traces and error messages are preserved
- Avoids re-running failing code just to see the error
- Saves time by surfacing known issues immediately

**If agentlog shows relevant errors:**
1. Copy the error details to your investigation file
2. Use them as your starting point for Phase 1
3. You may already have root cause evidence

**If agentlog is empty or unhelpful:**
- Proceed to Phase 1 normally
- You'll gather error evidence through reproduction

---

## The Four Phases

You MUST complete each phase before proceeding to the next.

### Phase 1: Root Cause Investigation
**Goal:** Understand WHAT and WHY

**Load:** Read [phases/phase1-root-cause.md](phases/phase1-root-cause.md)

Key activities:
- Read error messages carefully
- Reproduce consistently
- Check recent changes
- Gather evidence in multi-component systems
- Trace data flow

**Success criteria:** You understand the root cause, not just symptoms

---

### Phase 2: Pattern Analysis
**Goal:** Identify differences between working and broken

**Load:** Read [phases/phase2-pattern-analysis.md](phases/phase2-pattern-analysis.md)

Key activities:
- Find working examples
- Compare against references
- Identify differences
- Understand dependencies

**Success criteria:** You know what's different and why it matters

---

### Phase 3: Hypothesis and Testing
**Goal:** Form and test specific hypothesis

**Load:** Read [phases/phase3-hypothesis-testing.md](phases/phase3-hypothesis-testing.md)

Key activities:
- Form single hypothesis
- Test minimally (one variable)
- Verify before continuing
- When you don't know, say so

**Success criteria:** Hypothesis confirmed or new hypothesis formed

---

### Phase 4: Implementation
**Goal:** Fix root cause, not symptom

**Load:** Read [phases/phase4-implementation.md](phases/phase4-implementation.md)

Key activities:
- Create failing test case
- Implement single fix
- **Smoke-test end-to-end** (see below)
- If 3+ fixes failed: Question architecture

**Success criteria:** Bug resolved, tests pass, **smoke-test confirms fix works in real usage**

#### Smoke-Test Requirement (CRITICAL)

**Post-mortem learning:** The --mcp bug took 4 agents because none tested end-to-end before completing. Code compiled, unit tests passed, but the actual feature was still broken.

**Before claiming fix is complete, you MUST:**
1. Run the actual failing scenario that triggered this debugging
2. Verify the expected behavior now occurs
3. Document the smoke-test in your completion comment

**Examples of valid smoke-tests:**
- Bug: "CLI crashes on --mcp flag" → Run `orch spawn --mcp` and verify no crash
- Bug: "Login fails silently" → Actually log in and verify success message
- Bug: "Export produces empty file" → Run export and verify file has content

**NOT valid smoke-tests:**
- "Unit tests pass" (necessary but not sufficient)
- "Code compiles" (meaningless)
- "I reviewed the code and it looks right" (not a test)

**If you cannot smoke-test:** Document WHY in your completion comment (e.g., "requires production credentials"). This is acceptable but must be explicit.

---

## Quick Reference

| Phase | Key Activities | Success Criteria | Load |
|-------|---------------|------------------|------|
| **1. Root Cause** | Read errors, reproduce, check changes, gather evidence | Understand WHAT and WHY | [phase1-root-cause.md](phases/phase1-root-cause.md) |
| **2. Pattern** | Find working examples, compare | Identify differences | [phase2-pattern-analysis.md](phases/phase2-pattern-analysis.md) |
| **3. Hypothesis** | Form theory, test minimally | Confirmed or new hypothesis | [phase3-hypothesis-testing.md](phases/phase3-hypothesis-testing.md) |
| **4. Implementation** | Create test, fix, verify | Bug resolved, tests pass | [phase4-implementation.md](phases/phase4-implementation.md) |

---

## Red Flags - STOP and Follow Process

If you catch yourself thinking:
- "Quick fix for now, investigate later"
- "Just try changing X and see if it works"
- "Add multiple changes, run tests"
- "Skip the test, I'll manually verify"
- "It's probably X, let me fix that"
- "I don't fully understand but this might work"
- "Pattern says X but I'll adapt it differently"
- "Here are the main problems: [lists fixes without investigation]"
- Proposing solutions before tracing data flow
- **"One more fix attempt" (when already tried 2+)**
- **Each fix reveals new problem in different place**
- **"We fixed this same TYPE of issue before" (check git history for whack-a-mole pattern)**
- **"Just bump this timeout/retry/limit again"** (may be symptom-fixing)

**ALL of these mean: STOP. Return to Phase 1.**

---

## Fast-Path Alternative (Obvious/Localized Failures)

If the failure is clearly localized and trivial (e.g., import path error,
undefined name, single failing test with an obvious cause), consider the
`quick-debugging` skill instead. It provides a lightweight loop with strict
escalation back to this skill if the first attempt fails or the issue proves
non-trivial.

Use `quick-debugging` when:
- Scope is ≤ 2 files and ≤ 1 hour of work
- Error points directly at the fix location (import/name/signature mismatch)
- No architectural uncertainty or broad system behavior involved

Otherwise, continue with the full systematic process below.

**If 3+ fixes failed:** Question the architecture (see Phase 4)

---

## Integration with Other Skills

**This skill requires using:**
- **test-driven-development** - REQUIRED for creating failing test case (see Phase 4)

**Specialized techniques (via progressive disclosure):**
- See "Common Debugging Patterns" section above for when to load:
  - `techniques/root-cause-tracing.md` - Deep call stack errors
  - `techniques/condition-based-waiting.md` - Timing-dependent failures
  - `techniques/defense-in-depth.md` - Invalid data propagation

**Verification discipline:**
- Always run tests/verification after implementing fix
- Confirm actual output before claiming success
- Evidence before assertions (verification rules should be in CLAUDE.md, not a skill)

---

## Real-World Impact

From debugging sessions:
- Systematic approach: 15-30 minutes to fix
- Random fixes approach: 2-3 hours of thrashing
- First-time fix rate: 95% vs 40%
- New bugs introduced: Near zero vs common

---

## Self-Review (Mandatory)

**After implementing fix, perform self-review before completion.**

This is a mandatory quality gate. Report findings via `bd comment`.

### Self-Review Checklist

#### 0. Pattern Scope Verification

**If the bug was a pattern that could exist elsewhere in the codebase:**

| Check | How | If Failed |
|-------|-----|-----------|
| **Pattern scoped** | Ran `rg "bug_pattern"` to find all occurrences | Run now, document count |
| **All instances fixed** | Run `rg "bug_pattern"` now - should be 0 or documented exceptions | Fix remaining instances |
| **Similar patterns checked** | Searched for variations of the bug pattern | Document any found |

**Examples:**
```bash
# Off-by-one error in loop bounds
rg "range\(len\(" --type py  # Check for similar anti-pattern

# Missing null check before access
rg "\.get\(.*\)\." --type py  # Find similar unguarded access

# Hardcoded timeout that was too short
rg "timeout.*=.*[0-9]" --type py  # Find other hardcoded timeouts
```

**Why this matters:** If you found one instance of a bug pattern, there are often more. Fixing one while leaving others creates inconsistent behavior and future debugging confusion.

**Skip this section if:** Bug was truly one-off (typo, wrong variable name, unique logic error).

---

#### 1. Debugging-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Root cause addressed** | Fix targets root cause, not symptom | Return to Phase 1 |
| **No debug code left** | No console.log, print, debugger statements | Remove before commit |
| **No temporary workarounds** | No "TODO: fix properly later" | Complete the fix |
| **Regression test exists** | Test that would catch this bug if reintroduced | Add test |
| **Investigation documented** | Hypothesis + evidence in investigation file | Update file |

#### 2. Anti-Pattern Detection

- [ ] No god objects introduced (files <300 lines, single responsibility)
- [ ] No tight coupling (components don't directly instantiate dependencies)
- [ ] No magic values (constants named and documented)
- [ ] No incomplete work (no TODO, FIXME, placeholder comments)

#### 3. Security Review

- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No injection vulnerabilities (SQL, command, path traversal)
- [ ] No XSS vulnerabilities (user input escaped)

#### 4. Commit Hygiene

```bash
git log --oneline -5
```

- [ ] Conventional format (`fix:`, `test:`, `refactor:`)
- [ ] Atomic commits (one logical change per commit)
- [ ] No WIP commits in history

#### 5. Test Coverage

- [ ] Reproduction test exists (proves bug is fixed)
- [ ] Edge cases covered
- [ ] Error paths tested

#### 6. Discovered Work Check

*During this debugging, did you discover any of the following?*

| Type | Examples | Action |
|------|----------|--------|
| **Related bugs** | Similar issues, edge cases that still fail | `bd create "description" --type bug` |
| **Technical debt** | Workarounds, code that needs refactoring | `bd create "description" --type task` |
| **Systemic issues** | Architecture problems causing this class of bug | `bd create "description" --type feature` |
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
- [ ] **Reviewed for discoveries** - Checked debugging work for related issues beyond original scope
- [ ] **Tracked if applicable** - Created beads issues for actionable items (or noted "No discoveries")
- [ ] **Included in summary** - Completion comment mentions discovered items (if any)

**If no discoveries:** Note "No discovered work items" in completion comment. This is common and acceptable.

**Why this matters:** Debugging often reveals related bugs and systemic issues. Beads issues ensure these discoveries surface in SessionStart context rather than getting buried in investigation files.

### Report via Beads

**If self-review finds issues:**
1. Fix them before proceeding
2. Report: `bd comment <beads-id> "Self-review: Fixed [issue summary]"`

**If self-review passes:**
- Report: `bd comment <beads-id> "Self-review passed - ready for completion"`

**Checklist summary (verify mentally, report issues only):**
- Debugging-specific: Root cause addressed, no debug code, no workarounds, regression test exists, investigation complete
- Anti-patterns: No god objects, tight coupling, incomplete work
- Security: No hardcoded secrets, no injection vulnerabilities
- Commit hygiene: Conventional format, atomic commits
- Test coverage: Reproduction test, edge cases
- Discovered work: Reviewed for discoveries, tracked or noted "No discoveries"

**Only proceed to Completion Criteria after self-review passes.**

---

## Fix→Verify→Fix Cycle (Atomic Debugging)

**Post-mortem learning:** Multi-round fixes for --mcp required new spawns each time. A single agent doing fix→verify→fix cycles is more efficient than serial spawns.

### The Atomic Unit

**Fix + Verify = One Unit of Work**

Do NOT:
- Implement fix → claim complete → wait for new spawn if it fails
- "Fix is done, verification is a separate task"

DO:
- Implement fix → verify immediately → if fails, iterate
- Continue fix→verify loop until actually working
- Only claim complete when smoke-test passes

### When to Iterate vs Escalate

**Keep iterating if:**
- Verification reveals a related issue in same area
- Fix was incomplete but direction is correct
- You understand why it failed and have a hypothesis

**Escalate (report blocker) if:**
- 3+ fix attempts failed (questioning architecture needed)
- Root cause was misidentified (return to Phase 1)
- Issue is outside your scope/authority
- You need information you don't have access to

### Reporting During Iteration

```bash
# After each fix attempt
bd comment <beads-id> "Fix attempt 1: [what you tried] - Result: [pass/fail + why]"

# If iterating
bd comment <beads-id> "Fix attempt 2: [refined approach] - Result: [pass/fail]"

# Only when actually working
bd comment <beads-id> "Phase: Complete - Fix verified via [smoke-test description]"
```

---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kn decide` | `kn decide "Use optimistic locking" --reason "Prevents lost updates"` |
| Tried something that failed | `kn tried` | `kn tried "Pessimistic locking" --failed "Caused deadlocks"` |
| Discovered a constraint | `kn constrain` | `kn constrain "Cache invalidation requires explicit call" --reason "TTL alone causes stale reads"` |
| Found an open question | `kn question` | `kn question "Is the legacy API still used?"` |

**Quick checklist:**
- [ ] Reflected on session: What did I learn that the next agent should know?
- [ ] Externalized at least one item via `kn` command

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward bug fix, no new knowledge to externalize."

---

## Completion Criteria

**Completion criteria ensure quality delivery and proper handoff to orchestrator.**

Before marking complete, verify ALL of these:

- [ ] **Root cause identified** - Documented in investigation file OR commit message
- [ ] **Fix implemented** - Code changes address root cause (not symptoms)
- [ ] **Tests passing** - All tests green, including reproduction test
- [ ] **Smoke-test passed** - Actual failing scenario now works (see Phase 4)
- [ ] **Discovered work reviewed** - Checked for related bugs/issues, tracked or noted "No discoveries"
- [ ] **Leave it Better completed** - At least one `kn` command run OR noted as not applicable
- [ ] **Git working tree clean** - Run `git status` to verify
  - All in-scope changes committed with meaningful messages
  - If modified files outside scope, document rationale in bd comment
- [ ] **Reported via beads** - `bd comment <beads-id> "Phase: Complete - [resolution summary]"`

**If ANY box is unchecked, work is NOT complete.**

**Note:** Investigation file is optional. For simple bugs, well-documented commits serve as evidence of work. Complex bugs benefit from investigation file for knowledge preservation.

**After completing all criteria above:**

1. Verify all checkboxes are marked
2. Report completion: `bd comment <beads-id> "Phase: Complete - Root cause: [X], Fix: [Y], Tests passing"`
3. Close the beads issue: `bd close <beads-id> --reason "Root cause: [X], Fix: [Y]"`
4. Call /exit to close agent session


---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-u1nt "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`

⚠️ Your work is NOT complete until you run both commands.
