TASK: Rate limit account switch kills in-flight agents: When account hits rate limit and orchestrator runs 'orch account switch', agents can't recover. Need: 1) proactive rate limit monitoring (warn at 80%, pause spawning at 90%), 2) orch clean --preserve-orchestrator flag to protect meta sessions. Focus on the clean flag first as immediate mitigation.

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "rate"

### Constraints (MUST respect)
- Auto-generated skills require template edits
  - Reason: Direct edits to SKILL.md will be overwritten by build process - must edit src/SKILL.md.template and src/phases/*.md
- Beads cross-repo contamination can create orphaned FK references
  - Reason: bd-* prefixed dependencies were found in orch-go database from separate beads repo
- Untracked spawns (--no-track) generate placeholder beads IDs that fail bd comment commands
  - Reason: Beads IDs like orch-go-untracked-* don't exist in database, causing bd comment to fail with 'issue not found' - this is expected behavior, not a bug
- Verification cannot rely solely on AI-generated explanations
  - Reason: AI explanations increase trust even when wrong; experts can rationalize AI outputs creating validation loops (Nate Jones article)
- Investigation filenames are generated from task description at spawn time
  - Reason: generateSlug() runs at spawn before agent starts - cannot reflect findings in initial filename
- Ask 'should we' before 'how do we' for strategic direction changes
  - Reason: Epic orch-go-erdw was created assuming skills-as-value was correct direction. Architect review revealed the premise was wrong - current separation is intentional design. Wasted work avoided by validating premise before execution.
- Separate user observations from hypotheses before spawning
  - Reason: Evidence Hierarchy: observations are primary evidence, hypotheses are secondary claims to verify. User saying 'hydration is slow' contains observation (slow) and hypothesis (hydration) that must be parsed.

### Prior Decisions
- kb search and rg are complementary tools
  - Reason: kb searches knowledge artifacts (.kb/), rg searches code - agents should use both strategically based on query type
- Headless Swarm = batch execution + rate-limit management across accounts
  - Reason: User clarified scope: focus on concurrent agent spawning with capacity management, not distributed architecture or multi-model routing
- SPAWN_CONTEXT.md is 100% redundant - generated from beads + kb context + skill + template
  - Reason: Investigation confirmed all content exists elsewhere and can be regenerated at spawn time
- orch servers uses project-grouped output not service-grouped
  - Reason: Showing ports grouped by project (e.g., 'web:5173, api:3333') is more intuitive than separate rows per service because developers think in terms of projects not individual services
- Tmux session existence is correct abstraction for 'orch servers' status - separate infrastructure from project servers
  - Reason: orch serve is persistent monitoring infrastructure, not ephemeral project server. Tmux lifecycle matches dev server lifecycle. Mixing them creates semantic confusion and false status reporting.
- Skills performing well after Dec 20-23 changes
  - Reason: SYNTHESIS rates high (~80%), progressive disclosure reduced feature-impl 77% without quality loss, light-tier spawn system working correctly
- SHOULD-HOW-EXECUTE sequence for strategic questions
  - Reason: Epic orch-go-erdw was created from 'how do we' without testing premise. Architect found premise wrong. Wasted work avoided if 'should we' asked first.
- Visual verification gate is already integrated into VerifyCompletionFull
  - Reason: Found that VerifyVisualVerificationForCompletion is called in check.go:394-403, blocking completion when web/ files modified without evidence
- Generate gap reasons from context
  - Reason: Gap events contain skill, task, and occurrence data that can be formatted into meaningful kn entry reasons
- Systems form around foundational constraints through accumulated compensations - same mechanism as evolution
  - Reason: Observed in orch system: amnesia constraint generated externalization infrastructure, agreement constraint generated governance layer. Not designed top-down, shaped by pressure over time.
- generateReasonFromGaps must ensure 20+ char minimum for kn compatibility
  - Reason: The kn command requires --reason to be at least 20 characters. When gap events lack skill/task metadata, the base reason was too short.
- Use native Go RPC client for beads integration instead of CLI subprocess
  - Reason: Eliminates subprocess overhead, provides type safety, enables proper error handling. Socket at .beads/bd.sock. See .kb/investigations/2025-12-25-inv-design-beads-integration-strategy-orch.md
- Detection accelerates pressure, prevention relieves it - audits and health checks are detection (surfaces failures faster), not compensation (prevents failures from occurring)
  - Reason: Pressure Over Compensation should reject prevention, not detection. orch doctor is detection - it makes failures visible immediately rather than waiting for downstream breakage. The failure still happens, the learning signal is preserved, but latency is reduced.
- Auto-switch already implemented in account.AutoSwitchIfNeeded and integrated at spawn time in runSpawnWithSkill
  - Reason: Found existing implementation during investigation - checkAndAutoSwitchAccount at main.go:975 calls AutoSwitchIfNeeded which handles the 80% threshold check and account switching
- Feature-impl light tier is intentional, review tooling needs update
  - Reason: Tier system deliberately separates knowledge-producing skills (full) from implementation-focused skills (light). Review tooling only scans SYNTHESIS.md, creating blind spot for light tier completions.
- 5-tier escalation model integrated into daemon ProcessCompletion
  - Reason: Enables auto-completion for ~60% of routine work (None/Info/Review levels) while blocking completions requiring human decision (Block level for visual approval, Failed level for verification errors)
- orch stats command should aggregate events.jsonl
  - Reason: Rich event data exists (4145+ events, 27 types) but isn't surfaced. Completion rate 95.4% is calculable but invisible. Need unified observability.
- Periodic reflection uses poll loop cycle counter, not separate goroutine
  - Reason: Simpler implementation, poll loop already runs every minute, no coordination issues
- Cache infrastructure in separate file
  - Reason: 470+ lines of caching logic (beadsCache, workspaceCache, globalWorkspaceCacheType) is a cohesive unit that belongs in serve_agents_cache.go, separate from HTTP handlers
- Phase 2 daemon.go extraction follows Phase 1 pattern
  - Reason: Extracted active_count.go, issue_adapter.go, completion_processing.go using same pattern as Phase 1 (rate_limiter.go, skill_inference.go, issue_queue.go). daemon.go reduced from 1162 to 610 lines.

### Related Investigations
- CLI orch complete Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-complete-command.md
- CLI orch spawn Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md
- CLI Project Scaffolding and Build Setup
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-project-scaffolding-build.md
- OpenCode Client Package Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-opencode-session-management.md
- SSE Event Monitoring Client
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md
- Desktop Notifications on Completion
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-desktop-notifications-completion.md
- Fix SSE parsing - event type inside JSON data
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-fix-sse-parsing-event-type.md
- Set beads issue status to in_progress on spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-set-beads-issue-status-progress.md
- Update README with current CLI commands and usage
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-update-readme-current-cli-commands.md
- Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- Synthesis Protocol Design for Agent Handoffs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md
- Add Missing Spawn Flags
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-missing-spawn-flags-no.md
- Add Usage/Capacity Tracking to Account Package
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-usage-capacity-tracking-account.md
- Add Wait Command to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-wait-command-orch.md
- Automate Knowledge Sync using Cobra Doc Gen
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-automate-knowledge-sync-using-cobra.md
- KB Search vs Grep Benchmark
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md
- Compare orch-cli (Python) vs orch-go Features
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md
- Enhance status command with swarm progress
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-status-command-swarm-progress.md
- Expose Strategic Alignment Commands
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-expose-strategic-alignment-commands-focus.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.




## REPRODUCTION (BUG FIX)

🐛 **This is a bug fix issue.** The fix is verified when the reproduction steps no longer produce the bug.

**Original Reproduction:**
**Failure mode discovered 2026-01-06:**

Root cause: Agents are tied to the account that spawned them. When that account hits limits, those agents can't continue. No mechanism to detect, migrate, pause, or resume.

Impact: Complete loss of work context - had to restart meta-orchestrator from scratch.

Potential solutions:
1. Proactive rate limit monitoring (warn at 80%, pause spawning at 90%)
2. Agent session persistence that survives account switches  
3. Graceful degradation - queue new work instead of failing
4. orch clean --preserve-orchestrator flag to protect meta sessions

**Verification Requirement:**
Before marking Phase: Complete, you MUST:
1. Attempt to reproduce the original bug using the steps above
2. Confirm the bug NO LONGER reproduces after your fix
3. Report verification via: `bd comment orch-go-iz74x "Reproduction verified: [describe test performed]"`

⚠️ A bug fix is only complete when the original reproduction steps pass.


🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-iz74x "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-iz74x "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
1. Surface it first: `bd comment orch-go-iz74x "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation rate-limit-account-switch-kills` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-rate-limit-account-switch-kills.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-iz74x "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-debug-rate-limit-account-06jan-cf51/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-iz74x**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-iz74x "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-iz74x "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-iz74x "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-iz74x "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-iz74x "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-iz74x`.

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
<!-- Checksum: b78f2c8d8ff8 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/systematic-debugging/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-06 10:38:49 -->


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

**USE:** Playwright MCP - lighter weight, focused on automation

**AVOID:** browser-use MCP - causes context explosion (screenshots, full DOM)

**Decision flow:**
1. Need visual verification? → `snap`
2. Need browser automation (clicking, typing)? → Playwright MCP
3. Need visual + automation? → Consider if truly necessary


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
- **web:** http://localhost:5188
- **api:** http://localhost:3348

**Quick commands:**
- Start servers: `orch servers start orch-go`
- Stop servers: `orch servers stop orch-go`
- Open in browser: `orch servers open orch-go`



🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-iz74x "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.
