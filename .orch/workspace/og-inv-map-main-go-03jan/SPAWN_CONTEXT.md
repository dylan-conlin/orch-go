TASK: Map main.go command dependencies and recommend split strategy - analyze the 4823 line god object, identify command groupings, find shared state/helpers, map import dependencies, recommend file/package boundaries that avoid circular imports. Produce actionable refactoring plan.

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "map"

### Prior Decisions
- kb reflect uses single command with --type flag for four reflection modes
  - Reason: Consistent with kb pattern, most discoverable, extensible. Maps directly to signals from ws4z investigations.
- kb context uses keyword matching, not semantic understanding - 'how would the system recommend...' questions require orchestrator synthesis
  - Reason: Tested kb context with various query formats: keyword queries (swarm, dashboard) returned results, but semantic queries (swarm map sorting, how should dashboard present agents) returned nothing. The pattern reveals desire for semantic query answering that would require LLM-based RAG.
- Dashboard account name lookup uses email reverse-mapping from accounts.yaml
  - Reason: Provides meaningful account identifier (personal/work) instead of ambiguous email prefix
- bd exec.Command calls in orch-go are concurrency-safe
  - Reason: Beads daemon serializes database access via Unix socket, GetCommentsBatch has semaphore limiting to 10 concurrent
- RPC-first fallback pattern for beads operations
  - Reason: Sequential RPC calls through single connection are simpler than concurrent CLI subprocess spawning; daemon handles serialization so semaphore pattern is unnecessary
- Skill selection judgment preserved at issue creation not labeling
  - Reason: Daemon uses InferSkill(issueType) only; skill:* labels not implemented. Type→skill mapping is reliable. Batch-labeling doesn't sacrifice judgment.
- What artifact type should capture evolving landscape maps (entities, positioning, relationships over time)? Investigations are point-in-time, guides are procedures, decisions are choices. Maps/landscapes are a distinct knowledge type - tracking a domain, not answering a question.

### Related Investigations
- SSE-Based Completion Detection and Notifications
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-sse-based-completion-detection.md
- Implement Synthesis Card Display in Swarm Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-synthesis-card-display-swarm.md
- Port model flexibility and arbitrage to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-investigate-model-flexibility-arbitrage-orch.md
- Migrate orch-go from tmux to HTTP API
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-migrate-orch-go-tmux-http.md
- Agent Registry for Persistent Tracking
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-agent-registry-persistent.md
- Daemon Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-daemon-command.md
- Add work command to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-work-command.md
- Scaffold beads-ui v2 (Bun + SvelteKit 5 + shadcn-svelte)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-scaffold-beads-ui-v2-bun.md
- Update Investigation Skill to Use D.E.K.N. Summary
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-update-investigation-skill-use-summary.md
- Design: kb reflect Command Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md
- Registry Usage Audit in orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md
- Deep Post-Mortem on 24 Hours of Development Chaos
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md
- Design: Self-Reflection Protocol Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md
- Enhance Tmuxinator Config Generation with Port Registry
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-enhance-tmuxinator-config-generation-use.md
- Failure Mode Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md
- [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md
- Phase 3 - Evaluate spawn session_id capture without registry
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md
- Synthesis: Registry Evolution and Orch Identity
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md
- Dashboard Shows 0 Agents Despite API Returning 209
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-debug-dashboard-shows-0-agents-despite-api-returning-209.md
- Add Concurrency Control to Daemon
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-add-concurrency-control-daemon-worker.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.




📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

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
2. **SET UP investigation file:** Run `kb create investigation map-main-go-command-dependencies` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-map-main-go-command-dependencies.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-map-main-go-03jan/SYNTHESIS.md
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
<!-- Checksum: 68f6b8415809 -->
<!-- Source: .skillc -->
<!-- To modify: edit files in .skillc, then run: skillc build -->
<!-- Last compiled: 2025-12-30 10:08:22 -->

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
3. Try things, observe what happens (add findings progressively)
4. **Run a test to validate your hypothesis**
5. Fill conclusion only if you tested
6. Final commit

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


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.
