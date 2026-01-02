TASK: Fix Anthropic API key requirement bug during Gemini spawns

🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-6ju "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Run: `bd comment orch-go-6ju "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
ATTEMPTS (1):
  kn-e24c88 debugging Insufficient Balance error when orch usage showed 99% remaining
    failed: was checking wrong thing - the OpenCode SERVER version matters, not CLI. Dev server (0.0.0-dev from Dec 12) had stale auth. Fix: restart server with current version.
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

### Test spawn integration with real OpenCode server
- **Path:** `.kb/investigations/2025-12-19-inv-test-spawn-integration-real-opencode.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _- **Possible fix:** Read session ID from stdout stream, then detach/kill process or run in backgroun..._

*If these investigations are relevant, read the full files for detailed context.*

## ACTIVE SERVICES

*Running services on common dev ports. Use these for API calls/testing.*

- :3306 → mysqld (PID 2004) - likely dev server
- :3334 → bun (PID 64375) - likely dev server
- :4096 → bun (PID 54316) - likely dev server
- :5000 → ControlCe (PID 706) - macOS Control Center (can ignore)
- :5173 → node (PID 64379) - likely dev server
- :5432 → postgres (PID 1580) - likely dev server
- :8765 → Python (PID 1427) - likely API server


## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-6ju**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-6ju "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-6ju "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-6ju "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-6ju "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-6ju "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-6ju`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## ADDITIONAL CONTEXT

BEADS ISSUE: orch-go-6ju

Issue Description:
Agents are incorrectly prompting for an Anthropic API key during spawn when Gemini is configured via API key, bypassing the expected OAuth flow.




## SKILL GUIDANCE (issue-creation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: issue-creation
skill-type: procedure
audience: worker
spawnable: true
category: workflow
description: Transform symptoms into rich beads issues with Problem-Solution-Evidence structure. Investigates root cause before creating issue - the issue IS the deliverable, not an investigation file.
parameters:
- name: symptom
  description: The observed problem or behavior (e.g., "polling seems broken", "login fails intermittently")
  type: string
  required: true
allowed-tools:
- Read
- Glob
- Grep
- Bash
- Write
- Edit
deliverables:
- type: beads-issue
  required: true
  description: "Rich beads issue with P-S-E structure, file references, and appropriate triage label"
verification:
  requirements: |
    - [ ] Issue has Problem section (what's broken, observed behavior)
    - [ ] Issue has Evidence section (file:line refs, reproduction steps, error messages)
    - [ ] Issue has appropriate type (bug/feature/task)
    - [ ] Issue has triage label (triage:ready or triage:review)
    - [ ] Issue description is >200 characters (rich, not shallow)
  test_command: null
  required: true
---

# Issue Creation Skill

**Purpose:** Transform vague symptoms into high-quality beads issues through targeted investigation.

## The Core Insight

Quality issues come from understanding BEFORE creating, not from validation AFTER.

**Yegge's issues average 609 characters** with Problem-Solution-Evidence structure because someone understood the problem deeply first. Our goal: match that quality through systematic investigation.

## When to Use This Skill

| Use issue-creation | Use bd create directly |
|--------------------|------------------------|
| Symptom reported ("X seems broken") | Obvious, trivial issue |
| Root cause unclear | Already know exactly what's wrong |
| Multiple possible causes | Single clear fix needed |
| Worth 15-30 min investigation | <5 min to document |

**Rule:** If you'd need to investigate before fixing, use this skill.

## Workflow

### Phase 1: Understand the Symptom (5-10 min)

1. **Document the symptom exactly as reported**
   - What behavior was observed?
   - What was expected instead?
   - Any context (when it happens, frequency)?

2. **Reproduce if possible**
   ```bash
   # Try to trigger the symptom
   # Document exact steps and outcome
   ```

3. **Scope the problem area**
   ```bash
   # Find related code
   rg "keyword" --type py -l
   
   # Understand the area
   Read relevant files
   ```

### Phase 2: Investigate Root Cause (10-20 min)

**Goal:** Understand WHY, not just WHERE.

1. **Trace the code path**
   - Start from symptom manifestation
   - Follow the logic backward
   - Document each file and relevant lines

2. **Form hypothesis**
   - What could cause this behavior?
   - What evidence would confirm/deny?

3. **Test hypothesis**
   ```bash
   # Run specific tests, check logs, reproduce with variations
   ```

4. **Document file references**
   - `src/orch/spawn.py:142` - Where X happens
   - `src/orch/registry.py:89` - Related state management

### Phase 3: Create the Issue (5-10 min)

**Use the P-S-E structure:**

Write the full description first, then create with it inline:

```bash
# Create with description inline (NEVER use bd edit - it opens interactive editor)
bd create "Clear title describing the problem" --type bug --description "## Problem

[Full P-S-E content here - see template below]"
```

**Alternative for very long descriptions:**

```bash
# Create issue first
bd create "Clear title describing the problem" --type bug

# Add description via comment (non-interactive)
bd comment <issue-id> "## Problem

[Full P-S-E content here]

## Evidence

[File references and reproduction steps]

## Context

[Impact and related issues]"
```

**Issue Template:**

```markdown
## Problem

[What is broken or wrong. Specific, observable behavior.]

[When/how it manifests. Frequency, conditions.]

## Evidence

[File:line references where the issue exists]
- `src/file.py:123` - [What's wrong here]
- `src/other.py:456` - [Related code]

[Reproduction steps if applicable]
1. Do X
2. Do Y
3. Observe Z (expected: W)

[Error messages, logs, or output]

## Context

[Why this matters. Impact on users/system.]

[Any related issues or prior attempts.]
```

### Phase 4: Apply Labels

**Confidence-based labeling:**

| Label | When to use |
|-------|-------------|
| `triage:ready` | High confidence in diagnosis, clear fix path |
| `triage:review` | Uncertain about root cause or fix approach |

```bash
# High confidence - daemon can auto-spawn
bd label <issue-id> triage:ready

# Lower confidence - human reviews first
bd label <issue-id> triage:review
```

**Additional labels as appropriate:**
- Type: `bug`, `feature`, `task`
- Priority: `P1`, `P2`, `P3` (if clearly emergent)
- Area: `auth`, `ui`, `api` etc. (project-specific)

## Quality Checklist

Before completing, verify:

- [ ] **Problem clear:** Someone unfamiliar could understand what's wrong
- [ ] **Evidence concrete:** File:line references, not just "somewhere in X"
- [ ] **Reproducible:** Steps to trigger (if applicable)
- [ ] **Scoped:** Clear boundaries of what this issue covers
- [ ] **Labeled:** Appropriate triage label applied
- [ ] **>200 chars:** Rich description, not just a title

## Common Failures

**Shallow issue (DON'T):**
```markdown
Title: Fix polling bug
Description: Polling isn't working correctly.
```

**Rich issue (DO):**
```markdown
Title: Polling controller returns stale data after 60s timeout

## Problem

The polling controller in `src/controllers/poll.py` returns 
cached data even after the cache TTL expires. Users see 
outdated information until they hard-refresh.

## Evidence

- `src/controllers/poll.py:89` - Cache check doesn't account for TTL
- `src/cache/redis.py:142` - TTL is set but not used in fetch

Reproduction:
1. Load dashboard
2. Wait 90 seconds (past 60s TTL)
3. Trigger poll refresh
4. Observe: Data unchanged despite backend updates

## Context

Affects all users relying on real-time data. Discovered while 
investigating support ticket #4521.
```

## Completion

When finished:

1. Verify quality checklist passes
2. Note the created issue ID
3. Report via beads:
   ```bash
   bd comment <spawn-issue-id> "Phase: Complete - Created issue <new-issue-id> with P-S-E structure"
   ```
4. Close the spawning issue (if applicable):
   ```bash
   bd close <spawn-issue-id> --reason "Created rich issue <new-issue-id>"
   ```
5. Run `/exit`

## The Issue IS the Deliverable

Unlike investigation skill (produces investigation file), this skill produces a beads issue directly. The investigation happens, but it's internalized - the issue captures the understanding without a separate artifact.

**When to switch to investigation skill instead:**
- Understanding needed beyond single issue (architectural, cross-cutting)
- Knowledge should persist even if issue is closed
- Exploration without clear actionable outcome

---

**Remember:** 15-30 minutes of understanding BEFORE creating the issue produces dramatically better results than trying to add detail AFTER.


---

INVESTIGATION CONFIGURATION:
Type: simple

Create investigation file in .kb/investigations/simple/ subdirectory.
Follow investigation skill guidance for simple investigations.


ADDITIONAL DELIVERABLES:
- beads-issue:  (REQUIRED)

WORKSPACE DIR: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-fix-anthropic-api-key-20dec
(Use `bd comment orch-go-6ju` for progress tracking)


VERIFICATION REQUIRED:
- [ ] Issue has Problem section (what's broken, observed behavior)
- [ ] Issue has Evidence section (file:line refs, reproduction steps, error messages)
- [ ] Issue has appropriate type (bug/feature/task)
- [ ] Issue has triage label (triage:ready or triage:review)
- [ ] Issue description is >200 characters (rich, not shallow)


IMPORTANT: Ensure these requirements are met before reporting Phase: Complete via `bd comment`.

CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md
- CDD: ~/orch-knowledge/docs/cdd-essentials.md
- Process guide: ~/.claude/skills/workflow/issue-creation/SKILL.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. `bd comment orch-go-6ju "Phase: Complete - [1-2 sentence summary]"`
2. `/exit`

⚠️ Your work is NOT complete until you run both commands.