TASK: Implement orch servers command for centralized server management across projects.

**Commands to implement (in order):**
1. `orch servers` / `orch servers list` - Show all projects with port allocations and running status
2. `orch servers start <project>` - Start servers via tmuxinator
3. `orch servers stop <project>` - Stop servers pane
4. `orch servers attach <project>` - Switch to servers pane
5. `orch servers open <project>` - Open in browser
6. `orch servers status` - Summary view

**Key implementation details:**
- Use pkg/port to read port registry
- Use pkg/tmux for tmux operations
- Check actual port binding with lsof -i :{port}
- Tmuxinator configs at ~/.tmuxinator/workers-{project}.yml
- Servers window is always window 1 in workers-{project} session

**Reference:** .kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md

Start with 'orch servers list' as foundation, then build other commands on top.

SPAWN TIER: light

⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.



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
- Untracked spawns (--no-track) generate placeholder beads IDs that fail bd comment commands
  - Reason: Beads IDs like orch-go-untracked-* don't exist in database, causing bd comment to fail with 'issue not found' - this is expected behavior, not a bug

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
- orch-init-claudemd-already-implemented
  - Reason: orch init already creates CLAUDE.md by default with auto-detection - implemented in og-feat-implement-orch-init-21dec

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
1. Report via `bd comment orch-go-qvcq "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Run: `bd comment orch-go-qvcq "Phase: Complete - [1-2 sentence summary of deliverables]"`
2. Run: `/exit` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.

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
2. **SET UP investigation file:** Run `kb create investigation implement-orch-servers-command-centralized` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-implement-orch-servers-command-centralized.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-qvcq "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)
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

## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-qvcq**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-qvcq "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-qvcq "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-qvcq "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-qvcq "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-qvcq "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-qvcq`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## SKILL GUIDANCE (feature-impl)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: feature-impl
skill-type: procedure
audience: worker
spawnable: true
category: implementation
description: Unified feature implementation with configurable phases (investigation, clarifying-questions, design, implementation, validation, integration). Replaces test-driven-development, surgical-change, and feature-coordination skills. Use for any feature work with phases/mode/validation configured by orchestrator.

deliverables:
  investigation:
    required: false
    description: "Investigation file (.kb/investigations/YYYY-MM-DD-inv-{topic}.md). Required when investigation phase included in configuration."

  design:
    required: false
    description: "Design document (.kb/decisions/ or docs/plans/). Required when design phase included in configuration."

  tests:
    required: false
    description: "Test files (location varies by project). Required when mode=tdd or validation=tests."

  implementation:
    required: false
    description: "Source code changes. Required when implementation phase included in configuration."

  validation-evidence:
    required: false
    description: "Validation evidence (smoke test results, test output, screenshots). Required when validation != none. Report via bd comment."

verification:
  requirements:
    - "All configured phases completed (investigation findings, design docs, implementation, validation evidence as applicable)"
    - "Tests pass OR validation evidence documented via bd comment (automated tests, smoke test, or multi-phase validation)"
    - "Implementation matches design (if design phase used)"
    - "No regressions introduced (existing functionality still works)"
    - "All deliverables committed and reported via bd comment"
---

<!-- AUTO-GENERATED by skillc -->
<!-- Source: skills/src/worker/feature-impl/.skillc -->
<!-- To modify: edit files in skills/src/worker/feature-impl/.skillc, then run: skillc build -->
<!-- Last compiled: 2025-12-22 14:41:01 -->


# Feature Implementation (Unified Framework)

**For orchestrators:** Spawn via `orch spawn feature-impl "task" --phases "..." --mode ... --validation ...`

**For workers:** You've been spawned to implement a feature using a phased approach with specific configuration.

---

## Your Configuration

**Read from SPAWN_CONTEXT.md** to understand your configuration:

- **Phases:** Which phases you'll proceed through (e.g., `investigation,clarifying-questions,design,implementation,validation`)
- **Current Phase:** Determined by your progress (start with first configured phase)
- **Implementation Mode:** `tdd` or `direct` (only relevant if implementation phase included)
- **Validation Level:** `none`, `tests`, `smoke-test`, or `multi-phase` (only relevant if validation phase included)

**Example configuration:**
```
Phases: design, implementation, validation
Mode: tdd
Validation: smoke-test
```

---

## Deliverables

| Configuration | Required |
|---------------|----------|
| investigation phase | Investigation file |
| design phase | Design document |
| implementation phase | Source code |
| mode=tdd | Tests (write first) |
| validation=tests | Tests |
| validation=smoke-test | Validation evidence via bd comment |
| validation=multi-phase | Phase checkpoints via bd comment |

---

## Workflow

Proceed through phases sequentially per your configuration.

**Phases:** Investigation → Clarifying Questions → Design → Implementation (TDD/direct) → Validation → Self-Review → Integration

Track progress via `bd comment <beads-id> "Phase: X - details"`.

---

## Step 0: Scope Enumeration (REQUIRED)

**Purpose:** Prevent "Section Blindness" - implementing only part of spawn context.

**Before starting ANY phase work:**

1. **Read ENTIRE SPAWN_CONTEXT** - Don't skim. Don't stop at first section.
2. **Enumerate ALL Requirements** - List every deliverable from ALL sections
3. **Report Scope via Beads:**
   ```bash
   bd comment <beads-id> "Scope: 1. [requirement] 2. [requirement] 3. [requirement] ..."
   ```
4. **Flag Uncertainty** - If unclear what's in scope, ask before proceeding

**Why:** Agents repeatedly implement `## Implementation` while ignoring other sections. Forcing explicit enumeration catches this BEFORE implementation.

**Completion Criteria:**
- [ ] Read full SPAWN_CONTEXT (all sections)
- [ ] Enumerated ALL requirements
- [ ] Reported scope via `bd comment`
- [ ] Flagged any uncertainty

**Once Step 0 complete → Proceed to first configured phase.**

---

## Phase Guidance

### Investigation Phase

**Purpose:** Understand existing system before making changes.

**Deliverables:**
- Investigation file: `.kb/investigations/YYYY-MM-DD-inv-{topic}.md`
- Findings with Evidence-Source-Significance pattern
- Synthesis connecting findings

**Key workflow:**
1. Create investigation template BEFORE exploring (not at end)
2. Add findings progressively as you explore
3. Update synthesis every 3-5 findings
4. Document uncertainty honestly (tested vs untested)

**Completion:** Investigation committed, reported via `bd comment <beads-id> "Phase: Clarifying Questions"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-investigation.md` for detailed workflow, templates, and examples.

---

### Clarifying Questions Phase

**Purpose:** Surface ambiguities BEFORE design work begins.

**Deliverables:**
- Questions documented via `bd comment`
- Answers received from orchestrator
- No remaining ambiguities

**Key workflow:**
1. Review what you know (investigation findings or SPAWN_CONTEXT)
2. Identify gaps: Edge cases? Error handling? Integration? Compatibility? Security?
3. Ask using directive-guidance pattern (state recommendation, ask if matches intent)
4. Block until answers received

**Completion:** All questions answered, reported via `bd comment <beads-id> "Phase: Design"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-clarifying-questions.md` for question categories and patterns.

---

### Design Phase

**Purpose:** Document architectural approach before implementation.

**Deliverables:**
- Design document: `docs/designs/YYYY-MM-DD-{feature}.md`
- Testing strategy
- Architecture decision with trade-off analysis

**Key workflow:**
1. Review investigation findings (if applicable)
2. Determine if design exploration needed (multiple viable approaches → escalate)
3. Create design document using template
4. Get orchestrator approval before implementation

**Completion:** Design approved, committed, reported via `bd comment <beads-id> "Phase: Implementation"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-design.md` for detailed workflow and template.

---

### Implementation Phase (TDD Mode)

**Purpose:** Implement feature using test-driven development.

**When to use:** Feature adds/changes behavior (APIs, business logic, UI).

**The Iron Law:** NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST.

**Key workflow:**
1. **Pre-impl exploration:** Explore codebase with Task tool before coding
2. **TDD Cycle:** RED (write failing test) → GREEN (minimal code to pass) → REFACTOR
3. **UI features:** Mandatory smoke test (tests passing ≠ feature working)
4. **Commit pattern:** Separate test and implementation commits

**Red flags (STOP and restart):**
- Writing code before test
- Test passes immediately
- Rationalizing "just this once"

**Completion:** All tests pass, reported via `bd comment <beads-id> "Phase: Validation"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-implementation-tdd.md` and `reference/tdd-best-practices.md`.

---

### Implementation Phase (Direct Mode)

**Purpose:** Implement non-behavioral changes without TDD overhead.

**When to use:** Refactoring, configuration, documentation, code cleanup.

⚠️ **Critical:** If changing behavior → STOP and switch to TDD mode.

**Key workflow:**
1. **Pre-impl exploration:** Verify change is non-behavioral
2. Run existing tests (establish baseline)
3. Make focused changes (≤2 files, ≤1 hour)
4. Verify no regressions

**Completion:** Tests pass, reported via `bd comment <beads-id> "Phase: Validation"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-implementation-direct.md`.

---

### Validation Phase

**Purpose:** Verify implementation works as intended.

**Validation levels:**

| Level | Workflow |
|-------|----------|
| `none` | Commit, report complete |
| `tests` | Run test suite, verify pass, commit |
| `smoke-test` | Tests + manual verification + evidence capture |
| `multi-phase` | Tests + smoke + STOP for orchestrator approval |

**When validation fails:**
1. Check `agentlog errors` for runtime errors
2. Analyze failure output
3. Return to Implementation, fix, re-validate

**Completion:** Validation evidence documented, reported via `bd comment`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-validation.md` and `reference/validation-examples.md`.

---

### Integration Phase

**Purpose:** Combine multiple validated phases into cohesive feature.

**When to use:** Multi-phase features after individual phases validated.

**Key workflow:**
1. Review all completed phases via beads history
2. Identify integration points (data flow, shared state, API contracts)
3. Write integration tests for cross-phase scenarios
4. E2E verification + regression testing
5. Final smoke test of complete feature

**Completion:** Integration tests pass, reported via `bd comment <beads-id> "Phase: Validation"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-integration.md`.

---

## Self-Review Phase (REQUIRED)

**Purpose:** Quality gate before completion.

**Perform these checks before marking complete:**

### Anti-Pattern Detection
- [ ] No god objects (files >300 lines or multiple concerns)
- [ ] No tight coupling (use dependency injection)
- [ ] No magic values (use named constants)
- [ ] No deep nesting (>3 levels → extract to helpers)
- [ ] No incomplete work (TODOs, placeholders)

### Security Review
- [ ] No hardcoded secrets
- [ ] No injection vulnerabilities (SQL, XSS, command, path traversal)

### Commit Hygiene
- [ ] Conventional format: `type: description`
- [ ] Atomic commits (one logical change each)
- [ ] No WIP commits

### Test Coverage
- [ ] Happy path tested
- [ ] Edge cases covered
- [ ] Error paths tested

### Documentation
- [ ] Public APIs documented
- [ ] No commented-out code or debug statements

### Deliverables
- [ ] All required deliverables exist and complete
- [ ] Deliverables reported via `bd comment`

### Integration Wiring (CRITICAL)
- [ ] New modules imported somewhere
- [ ] New functions called somewhere
- [ ] New routes registered
- [ ] New components rendered
- [ ] No orphaned code

### Demo Data Ban (CRITICAL)
- [ ] No fake identities in production code
- [ ] No placeholder domains (use env vars)
- [ ] No lorem ipsum or magic numbers as data

### Scope Verification (For refactoring/migration)
- [ ] Ran `rg "old_pattern"` - should return 0
- [ ] Ran `rg "new_pattern"` - should match expected count

### Discovered Work
- [ ] Reviewed for discoveries (bugs, tech debt, enhancement ideas)
- [ ] Created beads issues with `triage:ready` or `triage:review` labels

**If issues found:** Fix immediately, commit, re-verify.

**When passed:** `bd comment <beads-id> "Self-review passed - ready for completion"`

---

## Leave it Better (REQUIRED)

**Purpose:** Every session should externalize what you learned.

**Before marking complete, run at least one:**

| What You Learned | Command |
|------------------|---------|
| Made a choice | `kn decide "X" --reason "Y"` |
| Something failed | `kn tried "X" --failed "Y"` |
| Found a constraint | `kn constrain "X" --reason "Y"` |
| Open question | `kn question "X"` |

**If nothing to externalize:** Note in completion comment.

**Completion Criteria:**
- [ ] Reflected on what was learned
- [ ] Ran at least one `kn` command OR noted why nothing to externalize
- [ ] Included "Leave it Better" status in completion comment

---

## Phase Transitions

**After completing each phase:**
1. Report progress: `bd comment <beads-id> "Phase: <new-phase> - <brief summary>"`
2. Output: "Phase complete, moving to next phase"
3. Continue to next phase guidance

---

## Completion Criteria

- [ ] Step 0 completed (scope enumerated)
- [ ] All configured phases completed
- [ ] Self-review passed
- [ ] Leave it Better completed
- [ ] All deliverables created
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Final status: `bd comment <beads-id> "Phase: Complete - [summary]"`

**If ANY unchecked, work is NOT complete.**

**Final step:** After all criteria met:
1. `bd close <beads-id> --reason "summary"`
2. Call /exit to close agent session

---

## Troubleshooting

**Stuck:** Re-read phase guidance, check SPAWN_CONTEXT. If blocked: `bd comment <beads-id> "BLOCKED: [reason]"`

**Unclear requirements:** `bd comment <beads-id> "QUESTION: [question]"` and wait

**Scope changes:** Document change, ask orchestrator via beads comment

---

## Related Skills

- **investigation**, **systematic-debugging**, **architect**, **record-decision**, **code-review**






---


FEATURE-IMPL CONFIGURATION:
Phases: implementation,validation
Mode: tdd
Validation: tests

Follow phase guidance from the feature-impl skill.



CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:

1. `bd comment orch-go-qvcq "Phase: Complete - [1-2 sentence summary]"`
2. `/exit`

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.

⚠️ Your work is NOT complete until you run these commands.
