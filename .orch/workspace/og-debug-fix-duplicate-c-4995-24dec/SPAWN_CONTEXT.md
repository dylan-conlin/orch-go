TASK: Fix duplicate 'Clear' button selector ambiguity in tests

🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-mhec.2 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Run: `bd comment orch-go-mhec.2 "Phase: Complete - [1-2 sentence summary of deliverables]"`
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

## PRIOR KNOWLEDGE (from kn)

*Relevant knowledge discovered. CONSTRAINTS must be respected.*

```
CONSTRAINTS (1):
  kn-8de6a0 Beads cross-repo contamination can create orphaned FK references
    reason: bd-* prefixed dependencies were found in orch-go database from separate beads repo

DECISIONS (5):
  kn-741ba1 Multi-repo hydration requires healthy database
    reason: Orphaned dependencies in kb-cli blocked all database operations including multi-repo sync. Fix with 'bd doctor --fix' before attempting multi-repo setup.
  kn-605d3b Beads multi-repo hydration works correctly in v0.33.2
    reason: Config disconnect bug fixed in commit 634c0b93. Prior kn entry about 'buggy v0.29.0' is superseded.
  kn-64dddf bd multi-repo config is YAML-only, database config is legacy
    reason: Fix commit 634c0b93 moved repos config from database to YAML. GetMultiRepoConfig() reads YAML only. Stale binary causes silent failure.
  kn-19f87a Parse workspace date from name suffix (DDmon format) for completed agent updated_at, with file modification time as fallback
    reason: Enables proper sorting in archive section where completed agents lacked timestamps
  kn-d620d7 Dashboard account name lookup uses email reverse-mapping from accounts.yaml
    reason: Provides meaningful account identifier (personal/work) instead of ambiguous email prefix

ATTEMPTS (2):
  kn-e24c88 debugging Insufficient Balance error when orch usage showed 99% remaining
    failed: was checking wrong thing - the OpenCode SERVER version matters, not CLI. Dev server (0.0.0-dev from Dec 12) had stale auth. Fix: restart server with current version.
  kn-0509d9 Deleting orphaned beads dependencies with bd-* prefix
    failed: Cannot restore - issues belong to different repo
```

*If you discover new constraints, decisions, or failed approaches, record them:*
- `kn constrain "<rule>" --reason "<why>"`
- `kn decide "<what>" --reason "<why>"`
- `kn tried "<what>" --failed "<why>"`

## PRIOR INVESTIGATIONS (from kb)

*Relevant investigations and decisions discovered. Review for context.*

### CLI orch spawn Command Implementation
- **Path:** `.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _2. **Workspace naming follows existing conventions** - `og-{skill-prefix}-{task-slug}-{date}` patter..._
  - _All tests pass. Template generation validated against Python patterns. Skill loading tested with tem..._

### SSE Event Monitoring Client
- **Path:** `.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _**Source:** Build errors showed redeclarations, fixed by proper separation_
  - _1. **Standard SSE protocol** - OpenCode follows standard SSE format, making parsing straightforward ..._

### Fix comment ID parsing - Comment.ID type mismatch
- **Path:** `.kb/investigations/2025-12-19-inv-fix-comment-id-parsing-comment.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _# Investigation: Fix comment ID parsing - Comment.ID type mismatch_

### Fix SSE parsing - event type inside JSON data
- **Path:** `.kb/investigations/2025-12-19-inv-fix-sse-parsing-event-type.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _**TLDR:** Question: Why does SSE parsing fail to detect event types? Answer: OpenCode SSE events inc..._
  - _# Investigation: Fix SSE parsing - event type inside JSON data_

### Legacy Artifacts Synthesis Protocol Alignment
- **Path:** `.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _| og-fix | 4 | 0 | 0% |_
  - _for type in feat inv debug work arch research fix explore; do_

*If these investigations are relevant, read the full files for detailed context.*


## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-mhec.2**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-mhec.2 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-mhec.2 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-mhec.2 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-mhec.2 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-mhec.2 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-mhec.2`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## ADDITIONAL CONTEXT

BEADS ISSUE: orch-go-mhec.2

Issue Description:
## Problem

Playwright test `filtering.spec.ts:85` fails because there are 2 elements matching 'Clear' button when filters are active:
1. The filter bar's Clear button
2. The 'Clear filters' link in the empty state

## Evidence

Test failure:
```
Error: strict mode violation: getByRole('button', { name: 'Clear' }) resolved to 2 elements:
1) <button class="text-xs text-muted-foreground...">Clear</button>
2) <button class="...mt-1 h-auto p-0 text-xs">Clear filters</button>
```

## Fix

Option A: Update test to use more specific selector:
```ts
const clearButton = page.getByTestId('filter-bar').getByRole('button', { name: 'Clear' });
```

Option B: Add data-testid to the Clear button in +page.svelte:
```svelte
<button onclick={clearFilters} class="..." data-testid="clear-filters-button">
```




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
<!-- Checksum: d7a3b7403bb7 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/systematic-debugging/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2025-12-24 09:37:07 -->


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
2. Check `agentlog errors --limit 10` - error may already be captured
3. Phase 1: Root cause investigation (understand WHAT and WHY)
4. Phase 2: Pattern analysis (working vs broken differences)
5. Phase 3: Hypothesis testing (form and test specific theory)
6. Phase 4: Implementation (failing test, fix root cause, verify)
7. Document and complete

**Red flag:** If thinking "quick fix for now" → STOP, return to Phase 1.


## Error Visibility (BEFORE Phase 1)

Check if errors have already been logged before investigating:

```bash
agentlog errors --limit 10   # View recent errors
agentlog tail                # Watch real-time (if debugging running process)
```

**If agentlog shows relevant errors:**
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
- [ ] **Tests passing** - Including reproduction test
- [ ] **Smoke-test passed** - Actual failing scenario now works
- [ ] **Self-review passed** - Pattern scope, no debug code, no workarounds
- [ ] **Discovered work reviewed** - Tracked or noted "No discoveries"
- [ ] **Git clean** - `git status` shows "nothing to commit"
- [ ] **Reported** - `bd comment <beads-id> "Phase: Complete - [summary]"`

**If ANY unchecked, work is NOT complete.**

### After All Criteria Met

```bash
bd comment <beads-id> "Phase: Complete - Root cause: [X], Fix: [Y], Tests passing"
bd close <beads-id> --reason "Root cause: [X], Fix: [Y]"
```

Then call `/exit` to close agent session.

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


ADDITIONAL DELIVERABLES:
- workspace:  (REQUIRED)

WORKSPACE DIR: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-debug-fix-duplicate-c-4995-24dec
(Use `bd comment orch-go-mhec.2` for progress tracking)


VERIFICATION REQUIRED:
- [ ] Root cause documented in investigation file
- [ ] If fix implemented: Verify fix works
  - [ ] Run automated tests if available
  - [ ] OR document manual verification steps and complete them
- [ ] No regression introduced (existing functionality still works)

IMPORTANT: Ensure these requirements are met before reporting Phase: Complete via `bd comment`.

CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md
- CDD: ~/orch-knowledge/docs/cdd-essentials.md
- Process guide: ~/.claude/skills/worker/systematic-debugging/SKILL.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. `bd comment orch-go-mhec.2 "Phase: Complete - [1-2 sentence summary]"`
2. `/exit`

⚠️ Your work is NOT complete until you run both commands.