TASK: /api/beads endpoint takes 6.5s - profile and fix

The /api/beads endpoint takes 6.5 seconds to respond.

## Context
During the Jan 7 dashboard performance investigation, we fixed /api/agents (32s → 627ms) but found other slow endpoints:
- /api/beads: 6.5s
- /api/beads/ready: 5.2s

## What to investigate
1. Profile the endpoint to find where time is spent
2. Check if it's fetching all issues when it could use caching
3. Check if bd list is being called multiple times
4. Look for O(n) patterns similar to /api/agents

## Reference
- .kb/investigations/2026-01-07-inv-dashboard-api-agents-performance-synthesis.md
- cmd/orch/serve.go (beads endpoint handler)

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "api"

### Constraints (MUST respect)
- OpenCode x-opencode-directory header returns ALL disk sessions, not just matching ones
  - Reason: API behavior is counterintuitive - without header returns in-memory only
- Stall detection: session.status=busy for >5min without message.part events indicates hung Claude API call
  - Reason: SSE monitoring pattern - healthy sessions emit regular message.part.updated events
- HTTP clients for external APIs should always have timeouts
  - Reason: http.DefaultClient has no timeout which can cause indefinite hangs when servers are unresponsive or in redirect loops
- SvelteKit stores using browser APIs must use init() pattern with onMount
  - Reason: SSR initializes module at server-side where window is undefined; store instance persists through hydration, so initialization code must be called explicitly in onMount
- Dashboard SSE connections can exhaust HTTP/1.1 browser connection pool (6 per origin)
  - Reason: Two SSE endpoints (/api/events, /api/agentlog) consume long-lived connections. When combined with slow API responses, fetch requests queue as pending indefinitely. Permanent fix needed: HTTP/2, multiplexed SSE, or WebSocket.

### Prior Decisions
- Port allocation should use ranges by purpose (vite: 5173-5199, api: 3333-3399)
  - Reason: Prevents conflicts and makes purpose clear from port number
- Session ID resolution pattern
  - Reason: Commands that need to find agents should use resolveSessionID or the runTail pattern: workspace files first, then API lookup, then tmux fallback
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- CreateSession API now accepts model parameter for headless spawns
  - Reason: Headless mode was missing model selection capability - added Model field to CreateSessionRequest struct and threaded through CreateSession function to achieve parity with inline/tmux modes
- Real-time UI updates via client-side SSE parsing
  - Reason: Parsing SSE events client-side (rather than polling API) provides instant updates without backend changes and scales better with many agents
- orch servers uses project-grouped output not service-grouped
  - Reason: Showing ports grouped by project (e.g., 'web:5173, api:3333') is more intuitive than separate rows per service because developers think in terms of projects not individual services
- orch servers open targets web port only
  - Reason: Opening both web and api ports in browser is noisy; web port (vite/dev server) is the primary entry point for local development
- Use x-opencode-env-ORCH_WORKER header for headless spawns
  - Reason: OpenCode HTTP API doesn't support env vars in request body, but follows x-opencode-* header pattern like x-opencode-directory, so we use x-opencode-env-ORCH_WORKER: 1 header when creating sessions via CreateSession API call
- ORCH_WORKER=1 set via environment inheritance on OpenCode server start and cmd.Env for direct spawns
  - Reason: Headless spawns use HTTP API where env can't be passed directly, but agents inherit env from server process. For inline/tmux, use cmd.Env on exec.Cmd.
- Use CLI subprocess for headless spawns
  - Reason: OpenCode HTTP API ignores model parameter; CLI (opencode run --model) is the only way to specify models
- Dashboard beads stats use bd stats --json API call
  - Reason: Provides comprehensive issue statistics with ready/blocked/open counts in single call
- Dashboard panel additions follow pattern: API endpoint in serve.go -> Svelte store -> page.svelte integration
  - Reason: Established during focus/beads/servers panel additions Dec 24
- Focus drift API uses CheckDrift from focus package
  - Reason: Reuses existing pkg/focus CheckDrift() rather than duplicating drift detection logic
- Keep beads as external dependency with abstraction layer
  - Reason: 7-command interface surface is narrow; dependency-first design (ready queue, dep graph) has no equivalent in alternatives; Phase 3 abstraction addresses API stability risk at low cost
- Use direct fetch in page.evaluate to test API performance
  - Reason: Bypasses SSE connection requirement while still testing the mock API response time
- 5 second debounce for is_processing state
  - Reason: Prevents gold border flashing between rapid tool calls. Agent switches busy/idle every ~100-500ms during active work.
- html-to-markdown v2 requires explicit plugin registration (base + commonmark) and WithDomain is a ConvertOptionFunc for ConvertString, not a converter option
  - Reason: v2 API breaking change from v1
- Agent registry removal complete - all state from OpenCode API + beads
  - Reason: Phase 4 complete: removed registry package, documentation cleaned up. No agent-registry.json file needed.
- Find a therapist - structural external perspective for Dylan's level
  - Reason: Perspective is Structural principle applied to Dylan's level reveals gap. Friends/family independently recommended. AI system pointing to same conclusion. History of AI-induced psychosis + continued intensive AI work = structural safeguard needed, not just ad-hoc.
- API endpoint handlers share same data types as CLI
  - Reason: ChangelogResult struct already had JSON tags, no new response type needed - enables code reuse

### Related Investigations
- CLI Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-status-command.md
- SSE Event Monitoring Client
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md
- Desktop Notifications on Completion
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-desktop-notifications-completion.md
- OpenCode POC - Spawn Session Via Go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-simple-opencode-poc-spawn-session-via.md
- Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- Add /api/agentlog endpoint to serve.go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-api-agentlog-endpoint-serve.md
- Add Usage/Capacity Tracking to Account Package
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-usage-capacity-tracking-account.md
- Enhance status command with swarm progress
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-status-command-swarm-progress.md
- Finalize Native Implementation for orch send
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-finalize-native-implementation-orch-send.md
- Implement Account Switch with OAuth Token Refresh
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-account-switch-oauth-token.md
- Implement Headless Spawn Mode
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-headless-spawn-mode-add.md
- SSE-Based Completion Detection and Notifications
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-sse-based-completion-detection.md
- Implement Synthesis Card Display in Swarm Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-synthesis-card-display-swarm.md
- Port model flexibility and arbitrage to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-investigate-model-flexibility-arbitrage-orch.md
- Migrate orch-go from tmux to HTTP API
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-migrate-orch-go-tmux-http.md
- Daemon Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-daemon-command.md
- Add Tail Command for Agent Debugging
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-tail-command.md
- POC Port Python Standalone + API Discovery to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-poc-port-python-standalone-api.md
- Refactor orch tail to use OpenCode API
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-refactor-orch-tail-use-opencode.md
- Research: Claude 4.5 and Claude Max Pricing (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-research-claude-claude-max-pricing.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.





🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-fwae6 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-fwae6 "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
1. Surface it first: `bd comment orch-go-fwae6 "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation api-beads-endpoint-takes-5s` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-api-beads-endpoint-takes-5s.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-fwae6 "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-api-beads-endpoint-07jan-3012/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-fwae6**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-fwae6 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-fwae6 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-fwae6 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-fwae6 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-fwae6 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-fwae6`.

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
description: Unified feature implementation with configurable phases (investigation, clarifying-questions, design, implementation, validation, integration). Replaces test-driven-development, surgical-change, and feature-coordination skills. Use for any feature work with phases/mode/validation configured by orchestrator.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 8997c0a4e566 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/feature-impl/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-07 09:43:35 -->


## Summary

**For orchestrators:** Spawn via `orch spawn feature-impl "task" --phases "..." --mode ... --validation ...`

---

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

**⚠️ UI Visual Verification (MANDATORY if web/ files modified):**

Before completing, run: `git diff --name-only | grep "^web/"`

If ANY files returned → Visual verification is REQUIRED:
1. Rebuild server: `make install` then restart via `orch servers`
2. Capture screenshot via Playwright MCP (`browser_take_screenshot` tool)
3. Document evidence: `bd comment <beads-id> "Visual verification: [description]"`

**⛔ Cannot mark Phase: Complete without visual evidence for web/ changes.**

**When validation fails:**
1. Check logs for runtime errors (test output, project logs)
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

### Original Symptom Validation (For Bug Fixes)
- [ ] Re-ran the EXACT command/scenario from original issue (same flags, same mode)
- [ ] Documented actual result (not an estimate) via `bd comment`
- [ ] If testing different mode/flag than original, explicitly justified why

**⚠️ Scope Redefinition Warning:** Agents can claim "fix complete" by testing a different scenario (e.g., `--json` flag when issue showed bare command). The fix is only verified when the original failing scenario passes.

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
- [ ] Created beads issues with `triage:review` label (default - lets orchestrator review before daemon spawns)

### Original Symptom Validation (For Bug Fixes)

⚠️ **This gate is MANDATORY for bug fixes.** Skip only for pure features/refactoring.

**Purpose:** Prevent "scope redefinition" - fixing a different problem than the original symptom.

**Before marking complete:**
1. **Re-run the original failing command** from the issue
   - Not a similar command - the EXACT command (same flags, same mode)
   - Example: If issue shows `time orch status # 1:25.67`, run `time orch status` (not `time orch status --json`)
2. **Document the actual result** in a beads comment:
   ```bash
   bd comment <beads-id> "Original symptom validation: [command] → [result]"
   ```
3. **Compare against original evidence** - is the symptom resolved?

**⚠️ Scope Redefinition Warning:**
If your fix validates against a DIFFERENT command/mode than the original issue:
- Example: Original issue shows text mode slow, you're testing JSON mode
- Example: Original issue shows `--verbose` flag, you're testing without it

This is a RED FLAG. Either:
- Re-test with the original command and document result
- OR explicitly justify why the different command is a valid proxy (with beads comment)

**Checklist:**
- [ ] Re-ran exact original command from issue
- [ ] Documented actual timing/behavior via `bd comment`
- [ ] Result matches expected fix (not an estimate like "~10s")
- [ ] If testing different mode/flags: justified why via `bd comment`

**If issues found:** Fix immediately, commit, re-verify.

**If validation skipped:** Document why in completion comment (e.g., "Original symptom validation: N/A - pure refactoring, no bug fix")

**Reference:** `.kb/investigations/2025-12-29-inv-root-cause-analysis-agent-orch.md` - Root cause analysis showing why this gate matters.

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
- [ ] **If web/ modified:** Visual verification completed with `bd comment` evidence
- [ ] Final status: `bd comment <beads-id> "Phase: Complete - [summary]"`

**⚠️ If web/ files modified without visual verification → completion will be REJECTED.**

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
2. `bd comment orch-go-fwae6 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.
