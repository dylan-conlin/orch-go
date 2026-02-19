TASK: Staleness audit of the orchestrator skill against current CLI reality.

ORIENTATION_FRAME: Orchestrators are using flags that don't exist (--opus), which means the orchestrator skill is teaching agents wrong things. This is the most-loaded skill — staleness here propagates everywhere.

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go
SESSION SCOPE: Medium

DIMENSION: organizational (documentation/skill accuracy)

WHAT TO AUDIT:
The orchestrator skill lives in the .skillc directory structure. The compiled output is what agents actually see. Audit the SOURCE files against current CLI reality.

1. Find the orchestrator skill source files — look in .skillc/src/meta/orchestrator/ or similar
2. Cross-reference EVERY CLI command/flag reference in the skill against actual CLI help output
3. Check: orch spawn --help, orch complete --help, orch status --help, orch review --help, orch monitor --help, bd --help, kb --help
4. Identify ALL stale references — not just model selection

KNOWN STALE (confirmed):
- '--opus' flag (doesn't exist — replaced by --backend claude|opencode)
- Model Selection table in Section 3 (sonnet vs opus guidance is entirely wrong)
- Likely: spawn mode descriptions may be stale

DELIVERABLE:
- Investigation file listing every stale reference found
- For each: the stale text, what's actually true now, and proposed replacement text
- Organize by severity: actively harmful (teaches wrong flags) vs misleading vs cosmetic

WHAT NOT TO DO:
- Do NOT rewrite the skill files
- Do NOT make changes — this is audit only
- Do NOT audit non-orchestrator skills

VERIFICATION: The investigation should be checkable by running the CLI help commands yourself.


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "staleness audit orchestrator"

### Constraints (MUST respect)
- orch-go DefaultModel should be Opus (claude-opus-4-5-20251101), not Gemini
  - Reason: Orchestrator guidance expects Opus for complex work, current Gemini default conflicts with operational practice
- D.E.K.N. 'Next:' field must be updated when marking Status: Complete
  - Reason: Prevents stale investigations that mislead future agents
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- GetAccountCapacity token comparison can fail when tokens drift between auth.json and accounts.yaml
  - Reason: External token rotation (by OpenCode) causes isActiveAccount check to fail, leading to auth.json being left with stale tokens after rotation
- LLM guidance compliance requires signal balance - overwhelming counter-patterns (56:13 ratio) drowns specific exceptions
  - Reason: Investigation found orchestrator skill has 4:1 ask-vs-act signal ratio causing autonomy guidance to fail
- Dashboard must be fully usable at 666px width (half MacBook Pro screen). No horizontal scrolling. All critical info visible without scrolling.
  - Reason: Primary workflow is orchestrator CLI + dashboard side-by-side on MacBook Pro. Minimum width constraint - should expand gracefully on larger displays.
- Orchestrator is AI, not Dylan - Dylan interacts with AI orchestrators who spawn/complete agents
  - Reason: Investigation 2025-12-25-design-orchestrator-completion-lifecycle-two incorrectly framed Dylan as the actor who spawns agents. The actor model is: Dylan ↔ AI Orchestrator ↔ Worker Agents. Mental model sync flows: Agent→Orchestrator (synthesis) and Orchestrator→Dylan (conversation).

### Prior Decisions
- Orchestrator sessions need SESSION_HANDOFF.md
  - Reason: Session amnesia applies to orchestrator work; skillc pattern provides mature template
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- Reflection value comes from orchestrator review + follow-up, not execution-time process changes
  - Reason: Evidence: post-synthesis reflection with Dylan created orch-go-ws4z epic (6 children) from captured questions
- Template ownership split by domain
  - Reason: kb-cli owns knowledge artifacts (investigation, decision, guide, research); orch-go owns orchestration artifacts (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT, SESSION_HANDOFF)
- ECOSYSTEM.md location is ~/.orch/ not ~/.claude/
  - Reason: Keeps orchestration docs with orchestration state; ~/.claude/ is for Claude-specific config
- Tiered spawn protocol uses .tier file in workspace for orch complete
  - Reason: Allows VerifyCompletion to read tier from workspace and skip SYNTHESIS.md requirement for light-tier spawns without requiring orchestrator to pass tier explicitly
- Domain-based template ownership: kb-cli owns artifact templates (investigation, decision, guide), orch-go owns spawn-time templates (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT)
  - Reason: Validated during stale template retirement - skill templates directories are orphaned when kb create provides templates
- bd multi-repo config is YAML-only, database config is legacy
  - Reason: Fix commit 634c0b93 moved repos config from database to YAML. GetMultiRepoConfig() reads YAML only. Stale binary causes silent failure.
- Default spawn mode is headless with --tmux opt-in
  - Reason: Aligns implementation with documentation (CLAUDE.md, orchestrator skill), reduces TUI overhead for automation, tmux still available via explicit flag
- Binary staleness should be prevented with make install
  - Reason: Project tracks ./orch binary in git, but build process creates build/orch. Users should run 'make install' to sync ~/bin/orch rather than manually copying to root directory.

### Models (synthesized understanding)
- Orchestrator Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-12.
    Changed files: pkg/spawn/orchestrator_context.go, pkg/session/registry.go, cmd/orch/complete_cmd.go, pkg/verify/check.go, cmd/orch/session.go.
    Deleted files: ~/.orch/sessions.json.
    Verify model claims about these files against current code.
  - Summary:
    Orchestrator sessions operate in a **three-tier hierarchy** (meta-orchestrator → orchestrator → worker) where each level is completed by the level above. Orchestrators produce **SESSION_HANDOFF.md** (not SYNTHESIS.md) and **wait** for completion (not /exit). They track via **session registry** (not beads) because orchestrators manage conversations, not work items. Frame collapse occurs when orchestrators drop levels and do work below their station - detected externally, not self-diagnosed. Checkpoint discipline uses duration thresholds (2h/3h/4h) as a proxy for context exhaustion.
    
    ---
  - Why This Fails:
    ### 1. Frame Collapse (Orchestrator → Worker)
    
    **What happens:** Orchestrator drops into worker-level implementation (editing code, debugging, investigating).
    
    **Root cause:** Vague goals → exploration mode → investigation → debugging. **Framing cues override skill instructions.**
    
    **Why detection is hard:** Orchestrators can't self-diagnose frame collapse. The frame defines what's visible, so from inside the collapsed frame, the behavior feels appropriate.
    
    **Detection signals:**
    - Edit tool usage on code files (not orchestration artifacts)
    - Time spent >15 minutes on direct fixes
    - SESSION_HANDOFF.md shows "Manual fixes" sections
    - Post-mortem reveals work that should have been spawned
    
    **NOT the fix:** Adding more ABSOLUTE DELEGATION RULE warnings. The agent already knows. The problem is framing, not awareness.
    
    **Prevention:**
    1. Provide specific goals with action verbs, concrete deliverables, success criteria
    2. Use WHICH vs HOW test: meta decides WHICH focus, orchestrator decides HOW to execute
    3. Frame collapse check in SESSION_HANDOFF.md template
    4. Potential: OpenCode plugin tracking Edit usage on code vs artifacts
    
    **Trigger pattern:** Failure-to-implementation. After agents fail, orchestrator tries to "just fix it" instead of trying different spawn strategy.
    
    ### 2. Self-Termination Attempts
    
    **What happens:** Spawned orchestrator tries to run `orch session end` or `/exit` instead of waiting for completion.
    
    **Root cause:** ORCHESTRATOR_CONTEXT.md template contradicted the hierarchical completion model (told orchestrator to self-terminate).
    
    **Why it's wrong:** Breaks the "completed by level above" invariant. Orchestrator can't verify its own work from meta perspective.
    
    **Fix:** Template updated Jan 2026 to instruct "write SESSION_HANDOFF.md and WAIT".
    
    ### 3. Session Registry Drift
    
    **What happens:** `~/.orch/sessions.json` shows status "active" for completed sessions.
    
    **Root cause:** `orch complete` didn't update session registry status (only closed beads issues or removed registry entries).
    
    **Why it matters:** Stale active sessions accumulate, `orch status` shows ghost sessions, registry becomes unreliable.
    
    **Fix:** `orch complete` now updates status to "completed", `orch abandon` updates to "abandoned" - sessions preserved for history, not removed.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-17-orchestrator-skill-injection-path-trace
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
    - 2026-02-16-orchestrator-skill-orientation-redesign
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md
    - 2026-02-15-orchestrator-skill-deployment-sync
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-15-orchestrator-skill-deployment-sync.md
- Probe: Orchestrator Skill Orientation Redesign
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md
  - Recent Probes:
    - 2026-02-17-orchestrator-skill-injection-path-trace
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
    - 2026-02-16-orchestrator-skill-orientation-redesign
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md
    - 2026-02-15-orchestrator-skill-deployment-sync
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-15-orchestrator-skill-deployment-sync.md
- Probe: Orchestrator Skill Injection Path Trace
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
  - Recent Probes:
    - 2026-02-17-orchestrator-skill-injection-path-trace
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
    - 2026-02-16-orchestrator-skill-orientation-redesign
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md
    - 2026-02-15-orchestrator-skill-deployment-sync
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-15-orchestrator-skill-deployment-sync.md
- Probe: Spawn-Time Staleness Detection Behavioral Verification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-time-staleness-detection-behavioral-verification.md
  - Recent Probes:
    - 2026-02-15-spawn-workflow-mechanics-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-workflow-mechanics-analysis.md
    - 2026-02-15-spawn-time-staleness-detection-behavioral-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-time-staleness-detection-behavioral-verification.md
- Follow Orchestrator Mechanism
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/follow-orchestrator-mechanism.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-15.
    Changed files: pkg/tmux/follower.go, pkg/tmux/tmux.go.
    Deleted files: ~/.tmux.conf.local, ~/.local/bin/sync-workers-session.sh.
    Verify model claims about these files against current code.
  - Summary:
    The "follow orchestrator" mechanism keeps the dashboard and workers Ghostty window synchronized with the orchestrator's current project context. Two independent systems work together: the **dashboard polls `/api/context`** to filter agents by project, and the **tmux `after-select-window` hook** switches the workers Ghostty to the matching `workers-{project}` session. Both rely on detecting the orchestrator pane's working directory, with an lsof fallback for when `#{pane_current_path}` is empty (e.g., running Claude Code).
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Orchestration Cost Economics
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-28.
    Deleted files: ~/.anthropic/, .kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md, pkg/spawn/backend.go.
    Verify model claims about these files against current code.
  - Summary:
    Agent orchestration cost is driven by three factors: **model pricing** (10-100x variance), **access restrictions** (fingerprinting, OAuth blocking), and **visibility** (lack of tracking caused $402 surprise spend). The Jan 2026 cost crisis revealed that headless spawning without cost visibility leads to runaway spend. DeepSeek V3 at $0.25/$0.38/MTok is now a **viable primary option** after testing confirmed stable function calling (contradicting earlier "unstable" documentation).
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Probe: Verifiability-First Closure Audit — Did Claimed Work Actually Land?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-15-verifiability-first-closure-audit.md
  - Recent Probes:
    - 2026-02-18-probe-entropy-spiral-fix-commit-relevance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
    - 2026-02-16-daemon-completion-loop-bypasses-verification-gates
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-16-daemon-completion-loop-bypasses-verification-gates.md
    - 2026-02-16-probe-three-code-paths-verification-state
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md
    - 2026-02-15-verificationtracker-backlog-count-mismatch
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-15-verificationtracker-backlog-count-mismatch.md
- Probe: Orchestrator Skill Deployment Sync
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-15-orchestrator-skill-deployment-sync.md
  - Recent Probes:
    - 2026-02-17-orchestrator-skill-injection-path-trace
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
    - 2026-02-16-orchestrator-skill-orientation-redesign
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md
    - 2026-02-15-orchestrator-skill-deployment-sync
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-15-orchestrator-skill-deployment-sync.md
- Probe: Attention Pipeline Full Audit — What's Real vs Stub
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-attention-pipeline-full-audit.md
  - Summary:
    **REAL and implemented (all 11 collectors):** Every collector has genuine detection logic reading real data (beads API, git log, agents API, JSONL files). None are stubs or return hardcoded data.
    
    **Currently firing:** Only 2 of 11 (BeadsCollector, RecentlyClosedCollector).
    
    **Not firing due to current state:** 6 of 11 (GitCollector, StuckCollector, UnblockedCollector, VerifyFailedCollector, EpicOrphanCollector, StaleIssueCollector). These would fire under the right conditions.
    
    **Cannot ever fire (badge types with no collector):** 3 badge types — `decide`, `escalate`, `crashed`. These exist in the frontend type system and badge config but have no backend collector that would ever emit them.
    
    **Net effect on dashboard:** 75% of open issues show false "Awaiting verification" badges due to `issue-ready` signals falling through to the default case. Zero legitimate badges are currently displayed.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-17-knowledge-tree-duplicate-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-17-knowledge-tree-duplicate-fix.md
    - 2026-02-16-knowledge-tree-tab-persistence
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-knowledge-tree-tab-persistence.md
    - 2026-02-16-attention-badge-verify-noise-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise-fix.md
    - 2026-02-16-agents-api-phase-field-missing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-agents-api-phase-field-missing.md
    - 2026-02-16-knowledge-tree-ssr-window-check
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-knowledge-tree-ssr-window-check.md
- Probe: Entropy Spiral Fix Commit Relevance Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
  - Recent Probes:
    - 2026-02-18-probe-entropy-spiral-fix-commit-relevance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
    - 2026-02-16-daemon-completion-loop-bypasses-verification-gates
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-16-daemon-completion-loop-bypasses-verification-gates.md
    - 2026-02-16-probe-three-code-paths-verification-state
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md
    - 2026-02-15-verificationtracker-backlog-count-mismatch
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-15-verificationtracker-backlog-count-mismatch.md

### Guides (procedural knowledge)
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md
- Completion Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Worker Patterns Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/worker-patterns.md
- Workspace Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/workspace-lifecycle.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- OpenCode Plugin System Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode-plugins.md
- Synthesis Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/synthesis-workflow.md

### Related Investigations
- Identify Orchestrator Value Add Vs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md
- Analyze Orchestrator Session Management Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md
- Meta-Orchestrator Architecture for Spawnable Orchestrator Sessions
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md
- Diagnose Orchestrator Skill 18% Completion Rate
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md
- Orchestrator Session Lifecycle Without Beads Tracking
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-05-inv-design-orchestrator-session-lifecycle-without.md
- Orchestrator Skill as Spawnable Agent Gap
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md
- Orchestrator Skill Drift Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md
- Design Solution for Model Artifact Staleness
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md
- Orchestrator Completion Lifecycle Design
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-25-design-orchestrator-completion-lifecycle-two.md

### Failed Attempts (DO NOT repeat)
- debugging Insufficient Balance error when orch usage showed 99% remaining
- Researching Foreman, Overmind, and Nx for polyrepo server management

### Open Questions
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.






🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-1036 "Phase: Planning - [brief description]"`
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
2. Run: `bd comment orch-go-1036 "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.



VERIFICATION REQUIREMENTS (ORCH COMPLETE):
Your work is verified in two human gates before closing:
- Gate 1 (explain-back): orchestrator must explain what was built and why.
- Gate 2 (behavioral, Tier 1 only): orchestrator confirms behavior is verified.
Provide clear Phase: Complete summary and VERIFICATION_SPEC.yaml evidence to support both gates.


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
1. Surface it first: `bd comment orch-go-1036 "CONSTRAINT: [what constraint] - [why considering workaround]"`
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


2. **SET UP probe file:** This is confirmatory work against an existing model.
   - Model content was injected in PRIOR KNOWLEDGE section above
   - Create probe file in model's probes/ directory
   - Use probe template structure: Question, What I Tested, What I Observed, Model Impact
   - Your probe should confirm, contradict, or extend the model's claims

   - **IMPORTANT:** After creating probe file, report the path via:
     `bd comment orch-go-1036 "probe_path: /path/to/probe.md"`



3. **UPDATE probe file** as you work:
   - Question: What model claim are you testing?
   - What I Tested: Actual command/code run (not just code review)
   - What I Observed: Actual output/behavior
   - Model Impact: Confirms/contradicts/extends which invariant

4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]


6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-audit-staleness-audit-orchestrator-18feb-8da1/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your probe file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to probe file
- Add '**Status:** QUESTION - [question]' when needing input



## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-1036**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-1036 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-1036 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-1036 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-1036 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-1036 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-1036`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (codebase-audit)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 2f5753c67dfd -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-17 12:10:22 -->


## Summary

**Purpose:** Common protocols shared by all worker skills. This is inherited by worker skills via dependencies.

---

# Worker Base Patterns

**Purpose:** Common protocols shared by all worker skills. This is inherited by worker skills via dependencies.

**What this provides:**
- Authority delegation (what you can decide vs escalate)
- Hard limits (constitutional constraints that override all authority)
- Constitutional objection protocol (how to raise ethical concerns)
- Beads progress tracking (how to report via bd comment)
- Phase reporting (how to signal transitions)
- Exit/completion protocol (how to properly end a session)

---



## Authority Delegation

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

---



## Hard Limits (Constitutional)

**These limits override ALL authority - orchestrator, user, or otherwise.**

Workers CANNOT do these regardless of instruction:

| Hard Limit | Constitutional Basis |
|------------|---------------------|
| Generate malware, exploits, or attack tools | Claude doesn't create weapons |
| Implement deceptive UI patterns (dark patterns) | Claude doesn't manipulate users |
| Build surveillance without consent disclosure | User autonomy and transparency |
| Intentionally bypass authentication/authorization | System integrity |
| Create content designed to deceive | Honesty as near-constraint |
| Automate harassment or mass targeting | Avoiding harm |
| Implement discriminatory logic | Ethical AI principles |

**When instructed to violate a hard limit:**

1. **Document** - `bd comment <id> "HARD LIMIT: [limit] - Cannot proceed with [specific instruction]"`
2. **Do NOT proceed** - No partial implementation, no "just this once"
3. **Continue other work** - If task has separable components, complete those
4. **Wait for human** - This bypasses orchestrator; only human can review

**Why these are non-negotiable:** Claude's constitution establishes these as near-inviolable constraints. Orchestrators are Claude instances too - they cannot authorize violations. Only human judgment can evaluate edge cases.

**Common false positives (these are usually OK):**
- Security testing tools for authorized pentesting
- Analytics with proper consent disclosure
- Authentication code (building it, not bypassing it)
- Competitive analysis (observation, not deception)

---



## Constitutional Objection Protocol

**Trigger:** You believe an instruction conflicts with constitutional values (safety, ethics, honesty, user wellbeing) but it's not a clear Hard Limit violation.

**This is DIFFERENT from operational escalation:**

| Type | Examples | Route |
|------|----------|-------|
| **Operational** | "I'm blocked", "Requirements unclear", "Need decision" | → Orchestrator |
| **Constitutional** | "This could harm users", "This feels deceptive", "Ethical concern" | → Human (bypasses orchestrator) |

**Protocol when you have a constitutional concern:**

1. **Identify the value** - Which constitutional principle is at risk? (safety, honesty, user autonomy, avoiding harm)

2. **Document it** - `bd comment <id> "CONSTITUTIONAL CONCERN: [value] - [specific concern]"`

3. **Do NOT proceed** with the concerning component

4. **Continue** with unrelated components if the task is separable

5. **Wait for HUMAN review** - Do not accept orchestrator override on constitutional matters

**Why this bypasses orchestrator:**

Claude's constitution says Claude can refuse unethical instructions regardless of the principal hierarchy. Orchestrators are Claude instances - they cannot authorize constitutional violations any more than you can. Human judgment is required for genuine ethical edge cases.

**Examples:**

| Situation | Response |
|-----------|----------|
| "Add tracking pixel without disclosure" | CONSTITUTIONAL CONCERN: user autonomy - undisclosed tracking |
| "Make the unsubscribe button hard to find" | CONSTITUTIONAL CONCERN: honesty - dark pattern design |
| "Scrape competitor's user data" | CONSTITUTIONAL CONCERN: ethics - unauthorized data collection |
| "Build feature that targets vulnerable users" | CONSTITUTIONAL CONCERN: avoiding harm - exploitation risk |

**When it's NOT a constitutional concern:**
- Technical disagreements about implementation
- Preference for different architecture
- Belief that requirements are suboptimal
- Wanting more context before proceeding

These are operational - escalate to orchestrator normally.

---



## Progress Tracking

**Use `bd comment` for phase transitions and progress updates.**

```bash
# Report progress at phase transitions
bd comment {{.BeadsID}} "Phase: Planning - Analyzing codebase structure"
bd comment {{.BeadsID}} "Phase: Implementing - Adding authentication middleware"
bd comment {{.BeadsID}} "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment {{.BeadsID}} "Phase: BLOCKED - Need clarification on API contract"

# Report questions
bd comment {{.BeadsID}} "Phase: QUESTION - Should we use JWT or session-based auth?"
```

**When to report:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Additional context:**
Use `bd comment` for additional context, findings, or updates:
```bash
bd comment {{.BeadsID}} "Found performance bottleneck in database query"
bd comment {{.BeadsID}} "investigation_path: .kb/investigations/2026-02-11-perf-issue.md"
```

**Test Evidence Requirement:**
When reporting Phase: Complete, include test results in the summary:
- Example: `bd comment {{.BeadsID}} "Phase: Complete - Tests: go test ./... - 47 passed, 0 failed (2.3s)"`
- Example: `bd comment {{.BeadsID}} "Phase: Complete - Tests: npm test - 23 specs, 0 failures"`
- Example: `bd comment {{.BeadsID}} "Phase: Complete - Tests: make test - PASS (coverage: 78%)"`

**Why:** `orch complete` validates test evidence in phase comments. Vague claims like "all tests pass" trigger manual verification.

**Never run `bd close`** - Only the orchestrator closes issues via `orch complete`.
- Workers report `Phase: Complete`, orchestrator verifies and closes
- Running `bd close` bypasses verification and breaks tracking

---



## Phase Reporting

**First 3 Actions (Critical):**
Within your first 3 tool calls, you MUST:
1. Report via `bd comment {{.BeadsID}} "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors phase reporting.

**Status Updates:**
Update Status: field in your workspace/investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed)

**Signal orchestrator when blocked:**
- Add `**Status:** BLOCKED - [reason]` to workspace
- Add `**Status:** QUESTION - [question]` when needing input

---



## Discovered Work (Mandatory)

**Before marking your session complete, review for discovered work.**

During any session, you may encounter:
- **Bugs** - Broken behavior not related to your current task
- **Tech debt** - Code that should be refactored but is out of scope
- **Enhancements** - Ideas for improvements noticed while working
- **Questions** - Strategic unknowns needing orchestrator input

### Checklist

Before completing your session:

- [ ] Reviewed for discovered work (bugs, tech debt, enhancements, questions)
- [ ] Created issues via `bd create` OR noted "No discovered work" in completion comment

### Creating Issues

```bash
# For bugs found
bd create "description of bug" --type bug -l triage:review

# For tech debt or refactoring needs
bd create "description" --type task -l triage:review

# For feature ideas or enhancements
bd create "description" --type feature -l triage:review

# For strategic questions needing decision
bd create "description" --type question -l triage:review
```

### Reporting

In your `Phase: Complete` comment, include either:
- List of issues created: `Created: orch-go-XXXXX, orch-go-YYYYY`
- Or: `No discovered work`

**Why this matters:** Discovered work that isn't tracked gets lost. The next session has no visibility into bugs or opportunities you found. Creating issues ensures nothing falls through the cracks.

### Cross-Repo Issue Handoff

**When you discover an issue that belongs to a different repo**, you cannot create it directly — `bd create` only works in the current project directory, and shell sandboxing prevents `cd` to other repos.

**Instead, output a structured `CROSS_REPO_ISSUE` block** in your beads completion comment or SYNTHESIS.md. The orchestrator will pick this up during completion review and create the issue in the target repo.

**Format:**
```
CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/<target-repo>
  title: "<concise issue title>"
  type: bug|task|feature|question
  priority: 0-4
  description: "<1-3 sentences with context, evidence, and why it matters>"
```

**Rules:**
- Use absolute or `~`-relative paths for `repo`
- Include enough context in `description` for the issue to stand alone (the orchestrator in the other repo won't have your session context)
- One block per issue — multiple issues get multiple blocks
- Report blocks in your `Phase: Complete` comment: `Cross-repo: 1 CROSS_REPO_ISSUE block for price-watch`

**Example:**
```bash
bd comment <beads-id> "Phase: Complete - Implemented token refresh. Cross-repo: 1 CROSS_REPO_ISSUE block below.

CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/price-watch
  title: Fix ScsOauthClient concurrent token refresh
  type: bug
  priority: 2
  description: During orch-go token handling work, discovered price-watch ScsOauthClient has a race condition when multiple goroutines call RefreshToken simultaneously. No mutex protects the shared token state."
```

---



## Session Complete Protocol

**When your work is done (all deliverables ready), complete in this EXACT order:**

{{if eq .Tier "light"}}
1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment {{.BeadsID}} "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
4. Commit any remaining changes (including `VERIFICATION_SPEC.yaml`)
5. Run: `/exit` to close the agent session

**Light Tier:** SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment {{.BeadsID}} "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. Ensure SYNTHESIS.md is created with these required sections:
   - **`Plain-Language Summary`** (REQUIRED): 2-4 sentences in plain language describing what you built/found/decided and why it matters. This is the scaffolding the orchestrator uses during completion review — write it for a human who hasn't read your code. No jargon without explanation. No "implemented X" without saying what X does.
   - **`Verification Contract`**: Link to `VERIFICATION_SPEC.yaml` and key outcomes
4. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
5. Commit all remaining changes (including SYNTHESIS.md and `VERIFICATION_SPEC.yaml`)
6. Run: `/exit` to close the agent session
{{end}}

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

---






---
name: codebase-audit
skill-type: procedure
description: Systematic codebase audit with configurable dimension (security/performance/tests/architecture/organizational/quick)
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 9a2d361ec776 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/codebase-audit/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/src/worker/codebase-audit/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/codebase-audit/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-18 09:18:39 -->


## Summary

**Use when the user says:**
- "Audit [focus area]" (security, performance, tests, architecture, organizational)
- "Run codebase health check"
- "Find [category] issues in the codebase"
- "Quick scan the codebase"

---

# Codebase Audit

<!-- SKILL-TEMPLATE: common-overview -->
<!-- Auto-generated from phases/common-overview.md -->

## When to Use This Skill

**Use when the user says:**
- "Audit [focus area]" (security, performance, tests, architecture, organizational)
- "Run codebase health check"
- "Find [category] issues in the codebase"
- "Quick scan the codebase"

**Auto-detect dimension from context:**
- "Security vulnerabilities" → security dimension
- "Performance bottlenecks" → performance dimension
- "Test coverage" → tests dimension
- "God objects" / "tight coupling" → architecture dimension
- "ROADMAP drift" / "template drift" → organizational dimension
- "Quick health check" → quick dimension

---

## Skill Overview

This skill performs systematic codebase audits with configurable dimensions. Each dimension focuses on a specific area and produces an investigation file with findings, evidence, and actionable recommendations.

**Core workflow:**
1. **Pattern Search** - Automated searches for known issues
2. **Evidence Collection** - Concrete examples with file paths/line numbers
3. **Analysis** - Root cause identification and severity assessment
4. **Documentation** - Investigation file with prioritized recommendations

**Key deliverables:**
- Investigation file at `.kb/investigations/YYYY-MM-DD-audit-{dimension}.md`
- Progress tracked via `bd comment <beads-id> "Phase: [current phase] - [progress details]"`

---

## Evidence Hierarchy

**Artifacts are claims, not evidence.**

| Source Type | Examples | Treatment |
|-------------|----------|-----------|
| **Primary** (authoritative) | Actual code, test output, observed behavior | This IS the evidence |
| **Secondary** (claims to verify) | Workspaces, investigations, decisions | Hypotheses to test |

When an artifact says "X is not implemented," that's a hypothesis—not a finding to report. Search the codebase before concluding.

**The failure mode:** An audit reads a workspace claiming "feature X NOT DONE" and reports that as a finding without checking if feature X actually exists in the code. Always verify artifact claims against primary sources.

---

## Model Awareness (Probe vs Investigation Routing)

**Before creating any artifact, check SPAWN_CONTEXT.md for model-claim markers.**

### Detection

Find the `### Models (synthesized understanding)` section in SPAWN_CONTEXT.md. Look for injected model-claim markers in model entries:
- `- Summary:`
- `- Critical Invariants:` or `- Constraints:`
- `- Why This Fails:` or `- Failure Modes:`

### If markers are present → Probe Mode

Your audit findings likely confirm, contradict, or extend an existing model's claims about the system. Route findings to a probe instead of a standalone investigation.

- Pick the most relevant model from the injected models section
- Create: `.kb/models/{model-name}/probes/{date}-{slug}.md`
- Use template: `.orch/templates/PROBE.md`
- Required sections: `Question`, `What I Tested`, `What I Observed`, `Model Impact`
- Focus on how audit findings confirm, contradict, or extend the model's invariants

**Example:** Auditing architecture when a "completion pipeline" model exists → create a probe documenting how the audit's coupling/complexity findings confirm or contradict the model's architectural claims.

### If markers are absent → Investigation Mode

Follow standard investigation file setup below.

---

## Investigation File Setup

**CRITICAL:** Before starting the audit, create investigation file from template. This ensures all findings are documented progressively with proper metadata (including Resolution-Status field for synthesis workflow).

### Create Investigation Template

```bash
# Create investigation using kb CLI command
# Update SLUG based on your audit dimension and topic
# Use audit/ prefix for audit investigations
kb create investigation "audit/dimension-audit-description"
```

**After creating the template:**
1. Fill Question field with specific audit focus from SPAWN_CONTEXT
2. Update metadata (Started date set automatically, verify Status)
3. Document findings progressively during audit (don't wait until end)
4. Update Confidence and Resolution-Status when completing audit

**Important:**
- The `kb create investigation` command auto-detects project directory and creates the investigation in the appropriate subdirectory.
- The investigation file includes Resolution-Status field (Unresolved/Resolved/Recurring/Synthesized/Mitigated) which is critical for the synthesis workflow. Always fill this field when completing the investigation.

**Now proceed with dimension-specific audit guidance below.**

---

## Available Dimensions

### Focused Audits (30-90 min)

**security** - Security vulnerabilities, unsafe patterns, secrets exposure, OWASP compliance
- When: Investigating security risks, penetration test prep, compliance audit
- Output: Security findings with severity ratings (Critical/High/Medium/Low)

**performance** - Performance bottlenecks, N+1 queries, algorithmic complexity, slow operations
- When: App is slow, high resource usage, scaling issues
- Output: Performance findings with profiling data and optimization recommendations

**tests** - Test coverage gaps, flaky tests, missing test types, test quality
- When: Flaky builds, low confidence in tests, missing edge case coverage
- Output: Testing gaps with risk assessment and coverage metrics

**architecture** - Coupling, god objects, missing abstractions, modularity issues
- When: Hard to add features, tight coupling, unclear boundaries
- Output: Architectural issues with refactoring effort estimates

**organizational** - ROADMAP drift, template drift, documentation sync, process violations
- When: Docs out of date, ROADMAP showing completed work as TODO, templates inconsistent
- Output: Organizational drift findings with system amnesia analysis

### Quick Scan (1 hour)

**quick** - Automated pattern search across all focus areas, high-priority issues only
- When: Need rapid health check before major work, onboarding to new codebase
- Output: Top 10 findings across all categories with quick-win recommendations

---

## Common Patterns

**Full audit workflow (2-4 hours):**
1. Run `quick` dimension to identify top issues
2. Run focused dimension for high-priority areas
3. Synthesize findings into single investigation file
4. Prioritize using ROI framework (impact vs effort)

**Targeted audit workflow (30-90 min):**
1. Run single focused dimension (user knows the problem area)
2. Investigation file documents findings
3. Add high-priority items to ROADMAP

<!-- /SKILL-TEMPLATE -->

---

<!-- MODE-SPECIFIC CONTENT -->
<!-- Use --parallel flag for comprehensive multi-agent audits -->

<!-- SKILL-TEMPLATE: mode-parallel -->
<!-- Auto-generated from phases/mode-parallel.md -->

# Parallel Execution Mode

**TLDR:** Use 5 parallel Haiku agents (one per dimension) for 3x faster comprehensive audits. Each agent runs pattern searches and returns JSON findings, which a synthesis agent combines into a prioritized report.

**When to use:** Comprehensive audit needed across multiple dimensions, time-constrained review, full codebase health check before major work.

**Output:** Single investigation file with prioritized findings from all dimensions.

---

## Architecture

```
┌─────────────────┐
│  Orchestrator   │ (spawns all agents in single message)
└────────┬────────┘
         │
    ┌────┴────┬────────┬────────┬────────┐
    ▼         ▼        ▼        ▼        ▼
┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐
│Security│ │Perf   │ │Arch   │ │Tests  │ │Org    │
│ Agent │ │ Agent │ │ Agent │ │ Agent │ │ Agent │
│(Haiku)│ │(Haiku)│ │(Haiku)│ │(Haiku)│ │(Haiku)│
└───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘
    │         │        │        │        │
    └────┬────┴────────┴────────┴────────┘
         │ (JSON findings)
         ▼
┌─────────────────┐
│  Synthesis      │ (Haiku - prioritizes & formats)
│  Agent          │
└────────┬────────┘
         │ (Prioritized report)
         ▼
┌─────────────────┐
│  Final Output   │
└─────────────────┘
```

---

## Key Design Decisions

1. **Haiku for dimension agents** - Pattern searches are IO-bound (grep/glob), not reasoning-heavy. Haiku is 3x faster and cheaper than Sonnet for this workload.

2. **JSON output from dimension agents** - Structured data enables consistent synthesis across agents.

3. **Separate synthesis step** - Keeps dimension agents focused on discovery; synthesis agent handles prioritization logic.

4. **No confidence scoring** - Unlike code-review (which filters false positives), codebase-audit produces objective pattern matches (file exists at N lines = fact, not opinion).

---

## Workflow

### Step 1: Spawn 5 Parallel Dimension Agents

Use a single message with 5 Task tool invocations to spawn all dimension agents concurrently:

```markdown
**For orchestrators:** Spawn parallel audit agents using:

1. Security Agent (Haiku) - Returns JSON with secrets, injection, auth findings
2. Performance Agent (Haiku) - Returns JSON with large files, complexity, N+1 findings
3. Architecture Agent (Haiku) - Returns JSON with god objects, coupling findings
4. Tests Agent (Haiku) - Returns JSON with coverage gaps, flaky test indicators
5. Organizational Agent (Haiku) - Returns JSON with drift patterns, doc sync findings

Each agent prompt should specify:
- Dimension to audit
- Project directory
- JSON output format requirement
- Pattern search commands to run
```

**Example Task tool invocation (5 in one message):**

```
Task 1: "Audit security dimension of PROJECT_DIR. Run pattern searches for secrets, injection, auth issues. Return JSON: {potential_secrets: N, injection_risks: N, auth_issues: N, findings: [...]}"

Task 2: "Audit performance dimension of PROJECT_DIR. Run pattern searches for large files, complexity, N+1. Return JSON: {large_files: [...], complexity_issues: N, findings: [...]}"

Task 3: "Audit architecture dimension of PROJECT_DIR. Run pattern searches for god objects, coupling. Return JSON: {god_objects: [...], coupling_issues: N, findings: [...]}"

Task 4: "Audit tests dimension of PROJECT_DIR. Run pattern searches for coverage gaps, flaky indicators. Return JSON: {coverage_gaps: N, flaky_tests: N, findings: [...]}"

Task 5: "Audit organizational dimension of PROJECT_DIR. Run pattern searches for drift, doc sync. Return JSON: {roadmap_drift: N, template_drift: N, findings: [...]}"
```

### Step 2: Wait for All Agents to Complete

All 5 agents run concurrently. Wait for all Task results to return.

**Expected latency:** ~5-10 seconds (parallel) vs ~15-30 seconds (sequential)

### Step 3: Spawn Synthesis Agent

Once all dimension agent results are available, spawn a synthesis agent:

```markdown
Task: "Synthesize codebase audit findings from 5 dimension agents.

Security findings: {JSON from agent 1}
Performance findings: {JSON from agent 2}
Architecture findings: {JSON from agent 3}
Tests findings: {JSON from agent 4}
Organizational findings: {JSON from agent 5}

Produce prioritized findings:
1. Combine all findings
2. Assign severity (Critical/High/Medium/Low)
3. Sort by ROI (impact vs effort)
4. Return top 20 findings with recommendations"
```

### Step 4: Write Investigation File

Write synthesis output to investigation file:

```bash
# Investigation file location
.kb/investigations/YYYY-MM-DD-audit-comprehensive-parallel.md
```

---

## Expected Benefits

| Metric | Sequential | Parallel | Improvement |
|--------|------------|----------|-------------|
| Wall-clock time | ~15-30 min | ~5-10 min | **3x faster** |
| Token cost | 1x Sonnet | 5x Haiku + 1x Haiku | ~Equal or cheaper |
| Coverage | Single dimension | All dimensions | **Comprehensive** |

---

## Agent Output Format

Each dimension agent returns structured JSON for synthesis:

**Security Agent:**
```json
{
  "dimension": "security",
  "potential_secrets": 20,
  "injection_risks": 3,
  "auth_issues": 0,
  "findings": [
    {"type": "secret", "file": "config.py", "line": 45, "severity": "high", "description": "Hardcoded API key"},
    {"type": "injection", "file": "api.py", "line": 123, "severity": "critical", "description": "SQL injection risk"}
  ]
}
```

**Architecture Agent:**
```json
{
  "dimension": "architecture",
  "god_objects": [
    {"file": "cli.py", "lines": 4031, "methods": 85},
    {"file": "spawn.py", "lines": 2110, "methods": 42}
  ],
  "coupling_issues": 52,
  "findings": [
    {"type": "god_object", "file": "cli.py", "severity": "high", "description": "4031 lines exceeds 300-line threshold"}
  ]
}
```

---

## Synthesis Output Format

The synthesis agent produces a prioritized report:

```markdown
# Comprehensive Audit: [Project Name]

**Date:** YYYY-MM-DD
**Mode:** Parallel (5 dimension agents + synthesis)
**Duration:** X minutes

## Executive Summary

- **Critical findings:** N
- **High priority:** N
- **Medium priority:** N
- **Total findings:** N

## Prioritized Findings (by ROI)

### 1. [CRITICAL] Security: SQL injection in api.py:123
**Dimension:** Security
**Impact:** High (data breach risk)
**Effort:** Low (parameterized queries)
**Recommendation:** Use parameterized queries instead of string formatting

### 2. [HIGH] Architecture: cli.py at 4031 lines
**Dimension:** Architecture
**Impact:** High (maintainability, testing difficulty)
**Effort:** Medium (extract modules)
**Recommendation:** Extract command handlers to separate modules

### 3-20. [Additional findings...]

## Metrics Baseline

| Dimension | Key Metric | Value |
|-----------|------------|-------|
| Security | Potential secrets | 20 |
| Architecture | Files >300 lines | 3 |
| Tests | Coverage gaps | 15 |
| Performance | N+1 queries | 5 |
| Organizational | ROADMAP drift | 8 |

## Next Steps

1. Address critical findings immediately
2. Schedule high-priority fixes this sprint
3. Add medium-priority to backlog
4. Re-audit in 30 days to measure improvement
```

---

## When NOT to Use Parallel Mode

- **Single dimension focus** - If you already know the problem area, use focused audit instead
- **Quick health check** - Use `dimension: quick` for rapid triage without parallel overhead
- **Limited context** - Parallel spawns 6 agents; if context window is constrained, use sequential

---

## Comparison with Sequential Audit

| Aspect | Sequential | Parallel |
|--------|------------|----------|
| **Speed** | 15-30 min | 5-10 min |
| **Token cost** | Lower | Similar (Haiku is cheap) |
| **Depth** | Single dimension deep dive | All dimensions breadth |
| **Use case** | Known problem area | Comprehensive health check |
| **Coordination** | Simple | Requires synthesis step |

---

## Reference

- **Investigation:** `.kb/investigations/simple/2025-11-29-explore-multi-agent-parallel-review.md`
- **Pattern source:** Code-review plugin parallel agent architecture

<!-- /SKILL-TEMPLATE -->

---

<!-- DIMENSION-SPECIFIC CONTENT -->
<!-- The build system will inject the appropriate dimension module here based on spawn configuration -->

<!-- For backward compatibility with old skill names, detect dimension from SPAWN_CONTEXT -->
<!-- If spawned as codebase-audit-security, auto-set dimension=security -->
<!-- If spawned as codebase-audit --dimension performance, use that -->

**Dimension-specific guidance below:**

---

<!-- SKILL-TEMPLATE: dimension-security -->
<!-- Auto-generated from phases/dimension-security.md -->

# Codebase Audit: Security

**TLDR:** Security-focused audit identifying vulnerabilities, unsafe patterns, secrets exposure, and OWASP compliance gaps.

**Status:** STUB - To be fleshed out when needed

**When to use:** Security review needed, penetration test prep, compliance audit, incident investigation

**Output:** Investigation file with security findings rated by severity (Critical/High/Medium/Low) with remediation steps

---

## Focus Areas (To be expanded)

1. **Secrets Exposure** - API keys, passwords, tokens in code/git history
2. **Injection Vulnerabilities** - SQL injection, command injection, XSS
3. **Authentication/Authorization** - Weak auth, missing access controls
4. **Cryptography** - Weak encryption, insecure random, poor key management
5. **Dependencies** - Known vulnerabilities in packages
6. **Input Validation** - Unsafe user input handling
7. **OWASP Top 10** - Compliance with OWASP security standards

---

## Pattern Search Commands (To be expanded)

```bash
# Secrets exposure
rg "password|secret|api_key|token|private_key" --type py --type js -i

# SQL injection
rg "execute\(.*%|\.format\(|f\".*FROM|f\".*WHERE" --type py

# Command injection
rg "subprocess\.call|os\.system|eval\(|exec\(" --type py

# XSS vulnerabilities
rg "innerHTML|dangerouslySetInnerHTML|\.html\(" --type js --type jsx

# Hardcoded credentials
rg "password\s*=\s*['\"]|api_key\s*=\s*['\"]" --type py --type js
```

---

*This skill stub establishes security audit structure. Expand with detailed workflow, severity ratings, and remediation patterns when security audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-performance -->
<!-- Auto-generated from phases/dimension-performance.md -->

# Codebase Audit: Performance

**TLDR:** Performance-focused audit identifying bottlenecks, algorithmic issues, inefficient queries, and optimization opportunities.

**Status:** STUB - To be fleshed out when needed

**When to use:** App is slow, high CPU/memory usage, scaling problems, response time issues

**Output:** Investigation file with performance findings, profiling data, and optimization recommendations with effort estimates

---

## Focus Areas (To be expanded)

1. **Algorithmic Complexity** - O(n²) loops, inefficient algorithms
2. **Database Queries** - N+1 queries, missing indexes, slow queries
3. **Resource Usage** - Memory leaks, excessive allocations
4. **I/O Operations** - Blocking I/O, unnecessary file reads
5. **Caching** - Missing caches, cache invalidation issues
6. **Concurrency** - Poor parallelization, lock contention

---

## Pattern Search Commands (To be expanded)

```bash
# Nested loops (potential O(n²))
rg "for.*:\s*\n.*for.*:" --type py -U

# N+1 query patterns
rg "\.all\(\)|\.filter\(" --type py -C 3 | rg "for.*in"

# Large files (potential complexity issues)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# TODO/FIXME about performance
rg "TODO.*performance|FIXME.*slow|HACK.*optimize" -i

# Blocking I/O in loops
rg "for.*:\s*\n.*open\(|for.*:\s*\n.*requests\." --type py -U
```

---

*This skill stub establishes performance audit structure. Expand with profiling methodology, optimization patterns, and benchmarking when performance audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-tests -->
<!-- Auto-generated from phases/dimension-tests.md -->

# Codebase Audit: Tests

**TLDR:** Testing-focused audit identifying coverage gaps, flaky tests, missing test types, and test quality issues.

**Status:** STUB - To be fleshed out when needed

**When to use:** Flaky CI builds, low confidence in tests, missing edge case coverage, test suite maintenance needed

**Output:** Investigation file with testing gaps, risk assessment, coverage metrics, and test improvement roadmap

---

## Focus Areas (To be expanded)

1. **Coverage Gaps** - Modules without tests, uncovered edge cases
2. **Flaky Tests** - Time-dependent, random, inconsistent results
3. **Missing Test Types** - Unit/integration/e2e gaps
4. **Test Quality** - No assertions, over-mocking, brittle tests
5. **Test Organization** - Poor structure, hard to maintain
6. **Test Performance** - Slow tests, inefficient setup/teardown

---

## Pattern Search Commands (To be expanded)

```bash
# Modules without test files
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort)

# Flaky test indicators (sleep, random, time-based)
rg "sleep|time\.sleep|random\.|datetime\.now" tests/

# Tests without assertions
rg "def test_" tests/ -l | xargs rg "assert" -L

# Large test files (potential god test class)
find tests/ -name "*.py" | xargs wc -l | sort -rn | head -10

# Over-mocking indicators
rg "Mock|patch|MagicMock" tests/ -c | sort -rn | head -10
```

---

*This skill stub establishes testing audit structure. Expand with coverage analysis, flaky test patterns, and test quality metrics when testing audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-architecture -->
<!-- Auto-generated from phases/dimension-architecture.md -->

# Codebase Audit: Architecture

**TLDR:** Architecture-focused audit identifying coupling issues, god objects, missing abstractions, and modularity problems.

**Status:** STUB - To be fleshed out when needed

**When to use:** Hard to add features, tight coupling between modules, unclear boundaries, refactoring needed

**Output:** Investigation file with architectural issues, dependency analysis, and refactoring effort estimates

---

## Focus Areas (To be expanded)

1. **God Objects** - Classes/modules doing too much
2. **Tight Coupling** - Modules depending on too many others
3. **Missing Abstractions** - Repeated patterns not extracted
4. **Circular Dependencies** - Modules importing each other
5. **Poor Modularity** - Unclear boundaries, leaky abstractions
6. **Violation of SOLID Principles** - SRP, OCP, LSP, ISP, DIP violations

---

## Pattern Search Commands (To be expanded)

```bash
# God classes (many methods)
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -10

# Tight coupling (many imports from one module)
rg "^from (\w+) import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn

# Large files (potential god objects)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# Missing abstractions (switch/if-elif chains on type)
rg "if.*isinstance|if.*type\(.*\) ==" --type py -C 3

# Circular dependencies (imports at bottom of file)
rg "^from .* import" --type py | tail -20

# Deep nesting (complexity indicator)
rg "^\s{16,}(if|for|while|def)" --type py
```

---

*This skill stub establishes architecture audit structure. Expand with dependency analysis, refactoring patterns, and SOLID principles when architecture audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-organizational -->
<!-- Auto-generated from phases/dimension-organizational.md -->

# Codebase Audit: Organizational Drift

**TLDR:** Systematic investigation of organizational drift - ROADMAP hygiene, artifact coherence, template consistency, process adherence. Produces prioritized recommendations with system amnesia root cause analysis.

**When to use:** Dylan says "audit organizational drift", "check ROADMAP hygiene", "find documentation drift", or when you suspect accumulated organizational debt.

**Output:** Investigation file with drift patterns, evidence, system amnesia analysis, and actionable fixes.

---

## Quick Reference

### Focus Areas

1. **ROADMAP Drift** - Completed work marked TODO, missing tasks, stale priorities
2. **Documentation Drift** - Reference docs vs operational templates out of sync
3. **Template Drift** - Workspace templates vs actual workspaces inconsistent
4. **State Duplication** - Same info in multiple places falling out of sync
5. **Context Boundary Leaks** - Manual sync points across contexts (code ↔ docs ↔ tracking)

### Process (4 Phases)

1. **Pattern Search** (15-30 min) - Use automated tools to find drift candidates
2. **Evidence Collection** (30-60 min) - Validate patterns, gather concrete examples
3. **System Amnesia Analysis** (15-30 min) - Identify which coherence principles violated
4. **Documentation** (30 min) - Write investigation with recommendations and fixes

### Key Deliverable

Investigation file at `.kb/investigations/YYYY-MM-DD-audit-organizational-drift.md` with:
- **Status:** Complete
- **Root Cause:** Drift patterns with system amnesia analysis
- **Recommendations:** Prioritized fixes (forcing functions, automation, validation)

---

## Detailed Workflow

### Phase 1: Pattern Search (15-30 minutes)

**Use automated tools to find drift candidates:**

#### ROADMAP Drift Patterns

```bash
# Compare ROADMAP entries against recent git commits
cd ~/meta-orchestration
git log --oneline --since="30 days ago" | rg "feat:|fix:" | head -20
# Manually compare against docs/ROADMAP.org TODO items

# Find DONE items without completion metadata
rg "^\*\* DONE" docs/ROADMAP.org -A 5 | rg -v "CLOSED:|:Completed:"

# Find completed agents not in ROADMAP
orch history | rg "Completed" | head -10
# Check if these appear in ROADMAP
```

#### Template Drift Patterns

```bash
# Find workspaces missing new template fields
rg "^# Workspace:" .orch/workspace/ -l | while read ws; do
  grep -q "Session Scope" "$ws" || echo "MISSING SESSION SCOPE: $ws"
  grep -q "Checkpoint Strategy" "$ws" || echo "MISSING CHECKPOINT STRATEGY: $ws"
done

# Compare workspace template against actual workspaces
diff -u ~/.orch/templates/workspace/WORKSPACE.md \
        .orch/workspace/latest-workspace/WORKSPACE.md | head -50
```

#### Documentation Drift Patterns

```bash
# Find orch commands in code but not in operational templates
rg "def (spawn|check|status|complete|resume|send)" tools/orch/cli.py -o | \
  cut -d' ' -f2 | while read cmd; do
    grep -q "$cmd" ~/.orch/templates/orchestrator/orch-commands.md || \
      echo "MISSING IN TEMPLATE: $cmd"
  done

# Find features documented but not in reference docs
rg "orch \w+" ~/.orch/templates/orchestrator/ -o | sort -u > /tmp/template_cmds
rg "^###? orch" tools/README.md -o | sort -u > /tmp/readme_cmds
comm -23 /tmp/template_cmds /tmp/readme_cmds
```

#### Manual Sync Points (Fragile Patterns)

```bash
# Find "remember to" or "don't forget" instructions
rg "remember to|don't forget|make sure to update" docs/ --type md -i

# Find TODO comments about updating related files
rg "TODO.*update|FIXME.*sync" --type py --type md -C 2
```

#### State Duplication

```bash
# Find status/phase duplicated across systems
rg "status.*=.*(active|completed|paused)" --type py -l | \
  xargs rg "Phase.*=.*(Active|Complete|Paused)" -l

# Find completion timestamps in multiple places
rg "completed_at|completion_time|finished_at" --type py --type json
```

**Document all search commands in investigation file** (reproducibility)

---

### Phase 2: Evidence Collection (30-60 minutes)

**For each pattern found, gather concrete evidence:**

#### Evidence Standards

**For ROADMAP drift:**
- Specific ROADMAP entry + corresponding git commit showing drift
- Date completed vs date still showing as TODO
- Count of drift instances (how pervasive?)
- User impact (does this affect planning/prioritization?)

**For documentation drift:**
- Specific inaccuracy (what docs say vs what code does)
- File paths showing divergence
- When drift introduced (git blame to find when docs last updated)
- Impact (who's affected by stale docs - orchestrators, developers, both?)

**For template drift:**
- Specific workspace missing field + template showing field should exist
- Date workspace created vs date template updated
- Migration effort (how many workspaces need updating?)
- Graceful degradation check (does code handle missing fields?)

**For state duplication:**
- Concrete example showing same state in multiple files
- Which is source of truth? (or neither?)
- Instances where states diverged
- Proposed fix (derive, don't duplicate)

**For manual sync points:**
- Specific "remember to" instruction in docs
- Evidence of sync failures (times this was forgotten)
- Automation opportunity (can this be enforced?)

#### Investigation File Structure

```markdown
# Investigation: Organizational Drift Audit

**Date:** YYYY-MM-DD
**Status:** Complete
**Investigator:** Claude (codebase-audit-organizational skill)
**Trigger:** [Dylan's request or suspected drift]

---

## TLDR

**Key findings:** [2-3 sentence summary of major drift patterns]
**Highest priority:** [Top recommendation with ROI]
**Total drift instances:** [Count across all categories]

---

## Scope

**Focus areas:** Organizational drift (ROADMAP, docs, templates, state duplication)
**Boundaries:** [Project-specific or global artifacts?]
**Time spent:** [Actual time for audit]

---

## Findings by Category

### ROADMAP Drift (Priority: High/Medium/Low)

**Pattern:** [Name of drift pattern found]

**Evidence:**
- Instance 1: ROADMAP entry "Task X" marked TODO, git commit abc123 completed 2025-11-10
- Instance 2: [...]
- Total instances: [count]

**Metrics:**
- Tasks completed but not marked DONE: [count]
- Tasks missing completion metadata: [count]
- Average drift age: [days between completion and discovery]

**Impact:** [How this affects planning/orchestration]

**Recommendation:** [Specific fix with automation approach]

**ROI:** [Value gained / time invested]

---

### [Other categories following same structure]

---

## System Amnesia Analysis

**See:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`

**Coherence principles violated:**
- [ ] Single Source of Truth - [Example showing duplication]
- [ ] Automatic Loop Closure - [Example showing manual step]
- [ ] Cross-Boundary Coherence - [Example showing context switch failure]
- [ ] Observable Drift - [Example showing silent drift]
- [ ] Forcing Functions at Creation - [Example showing optional field]

**Common failures observed:**
- [ ] ROADMAP Drift - [X instances, root cause: manual ROADMAP updates]
- [ ] Documentation Drift - [X instances, root cause: template not rebuilt]
- [ ] Template Drift - [X instances, root cause: no migration mechanism]
- [ ] State Duplication - [X instances, root cause: derived state manual]
- [ ] Context Boundary Leaks - [X instances, root cause: no cross-project search]

**Design pattern recommendations:**
- Use "Derive, Don't Duplicate" for [specific case - e.g., registry status from workspace Phase]
- Add "Validation at Boundaries" for [specific workflow - e.g., orch complete checks Phase]
- Implement "Build Systems for Consistency" for [specific docs - e.g., template rebuild automation]
- Add "Forcing Functions" for [specific creation - e.g., ROADMAP requires task-id]

---

## Prioritization

**High Priority (fix now):**
1. [Issue] - Blocking orchestration, high impact, low effort
2. [Issue] - Data loss risk, silent failures

**Medium Priority (schedule soon):**
1. [Issue] - Maintenance burden, moderate effort
2. [Issue] - Developer experience impact

**Low Priority (backlog):**
1. [Issue] - Minor improvement, can defer
2. [Issue] - Nice-to-have, low impact

---

## Recommendations

**Immediate actions (this week):**
- [ ] [Specific task with owner and approach]
  - **Fix:** [What to do]
  - **Automation:** [How to prevent recurrence]
  - **Effort:** [Hours estimated]

**Short-term (this month):**
- [ ] [Planned fix with scope]

**Long-term (next quarter):**
- [ ] [Strategic improvement with ROI]

---

## Reproducibility

**Commands used for pattern search:**
```bash
# Document all grep/rg/find/diff commands used
# This allows re-running audit in future to measure improvement
```

**Metrics baseline:**
- Total ROADMAP entries: [count]
- ROADMAP drift instances: [count]
- Template drift instances: [count]
- Documentation drift instances: [count]
- State duplication instances: [count]
- Manual sync points: [count]

**Re-audit schedule:** 3 months (measure drift reduction)

---

## Related Work

- Decision: `.kb/decisions/2025-11-15-system-amnesia-as-design-constraint.md`
- Checklist: `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- Investigation: [Link to related organizational investigations]

---

## Next Steps

1. **Discuss findings with Dylan** (present prioritization, get approval)
2. **Add high-priority items to ROADMAP** (with effort estimates)
3. **Spawn agents for fixes** (if Dylan approves immediate action)
4. **Schedule re-audit** (3 months to measure improvement)
```

---

### Phase 3: System Amnesia Analysis (15-30 minutes)

**Identify which coherence principles were violated for each drift pattern:**

**Checklist for each finding:**

1. **Single Source of Truth** - Is there duplicate state? Which is authoritative?
2. **Automatic Loop Closure** - Does workflow require manual step to complete?
3. **Cross-Boundary Coherence** - Do updates span contexts (code/docs/tracking)?
4. **Observable Drift** - Was drift silent until manual inspection?
5. **Forcing Functions at Creation** - Could invalid state be created?

**For each violation, propose design pattern:**

| Violation | Pattern | Example Fix |
|-----------|---------|-------------|
| Duplicate state | Derive, Don't Duplicate | Registry status derived from workspace Phase |
| Manual loop closure | Atomic Multi-Context Updates | `orch complete` updates all systems |
| Silent drift | Validation at Boundaries | `orch complete` checks workspace Phase |
| No forcing function | Build Systems for Consistency | Template rebuild on SessionStart hook |

**Root cause categories:**
- **Return trip tax** - Easy to create, hard to remember to update
- **Context switching** - Update happens in different session/context
- **No single source of truth** - Multiple systems maintain same state
- **Manual sync points** - "Remember to" instructions
- **No observability** - Drift accumulates silently

---

### Phase 4: Documentation (30 minutes)

**Write investigation file following template above**

**Critical sections:**
- ✅ TLDR with key findings and top priority
- ✅ Evidence section with concrete examples (file paths, commit shas, counts)
- ✅ System Amnesia Analysis (which principles violated, proposed fixes)
- ✅ Prioritization using ROI framework (impact vs effort)
- ✅ Recommendations with specific, actionable tasks
- ✅ Reproducibility section with commands and baseline metrics

**Present findings to Dylan:**
- "Organizational drift audit complete. Key findings: [TLDR]"
- "Highest priority: [Top item with ROI]"
- "System amnesia root causes: [Top 2-3 principles violated]"
- "Would you like me to add high-priority items to ROADMAP or spawn agents to address them?"

---

## Anti-Patterns to Avoid

**❌ Audit without concrete examples**
- "ROADMAP has drift issues" (vague, not actionable)
✅ **Fix:** "12 tasks completed but marked TODO: Task X (commit abc123, completed 2025-11-10), Task Y (commit def456, completed 2025-11-09), ..."

**❌ No system amnesia analysis**
- Lists drift but doesn't identify root cause or prevention
✅ **Fix:** Map each finding to violated coherence principle, propose forcing function

**❌ No reproducibility**
- Can't re-run audit to measure improvement
✅ **Fix:** Document all commands + baseline metrics

**❌ Recommendations too vague**
- "Fix ROADMAP drift" (what does that mean?)
✅ **Fix:** "Add `orch complete` auto-update: read workspace task-id field, mark ROADMAP entry DONE"

**❌ No prioritization**
- Dylan doesn't know what to fix first
✅ **Fix:** Use ROI framework (impact vs effort matrix)

---

## Related Documentation

- **System amnesia patterns:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- **Investigation template:** `.orch/templates/INVESTIGATION.md`
- **ROADMAP management:** `docs/work-prioritization.md`
- **Template build system:** `.kb/decisions/2025-11-14-orchestrator-restructuring-template-build-system.md`

---

## Example Usage

**Dylan:** "audit organizational drift in meta-orchestration"

**You:**
1. Create investigation file: `.kb/investigations/2025-11-15-organizational-drift-audit.md`
2. Run pattern search commands (ROADMAP drift, template drift, docs drift)
3. Collect evidence (12 ROADMAP drift instances, 5 template drift instances, 3 doc drift instances)
4. System amnesia analysis (violated: Automatic Loop Closure, Observable Drift)
5. Prioritize using ROI framework
6. Write investigation file with recommendations
7. Present: "Audit complete. Found 20 drift instances across 3 categories. Highest priority: Fix `orch complete` to auto-update ROADMAP (violates Automatic Loop Closure - easy fix, high impact). Add to ROADMAP?"

---

*This skill enables systematic, evidence-based organizational drift assessment with system amnesia root cause analysis and actionable recommendations.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-quick -->
<!-- Auto-generated from phases/dimension-quick.md -->

# Codebase Audit: Quick Scan

**TLDR:** 1-hour automated health check across all audit areas. Returns top 10 high-priority findings with quick-win recommendations.

**When to use:** Need rapid health check before major work, onboarding to new codebase, monthly health monitoring, or before deciding which focused audit to run.

**Output:** Investigation file with top findings across all categories, sorted by ROI.

---

## Quick Reference

### Scan Areas (All Categories)

1. **Security** - Secrets, unsafe patterns, SQL injection, XSS
2. **Performance** - Large files, complex functions, N+1 queries
3. **Tests** - Missing tests, coverage gaps, flaky indicators
4. **Architecture** - God objects, tight coupling, missing abstractions
5. **Organizational** - ROADMAP drift, template drift, doc drift

### Process (30-60 minutes)

1. **Automated Scan** (30 min) - Run all pattern search commands
2. **Triage** (15 min) - Filter to top 10 by severity/effort
3. **Document** (15 min) - Write investigation with findings

### Deliverable

Investigation file: `.kb/investigations/YYYY-MM-DD-audit-quick-scan.md`
- Top 10 findings sorted by ROI
- Recommended next steps (which focused audit to run?)

---

## Workflow

### Step 1: Automated Scan (30 minutes)

**Run these commands and capture counts:**

```bash
# Security patterns
echo "=== SECURITY ===" >> /tmp/audit.txt
rg "password|secret|api_key|token" --type py --type js -i | wc -l >> /tmp/audit.txt
rg "eval\(|exec\(|__import__|subprocess\.call" --type py | wc -l >> /tmp/audit.txt

# Performance patterns
echo "=== PERFORMANCE ===" >> /tmp/audit.txt
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -10 >> /tmp/audit.txt
rg "TODO.*performance|FIXME.*slow" -i | wc -l >> /tmp/audit.txt

# Testing patterns
echo "=== TESTS ===" >> /tmp/audit.txt
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort) | wc -l >> /tmp/audit.txt
rg "sleep|time\.sleep|random\." tests/ | wc -l >> /tmp/audit.txt

# Architecture patterns
echo "=== ARCHITECTURE ===" >> /tmp/audit.txt
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -5 >> /tmp/audit.txt
rg "^from|^import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn | head -5 >> /tmp/audit.txt

# Organizational patterns
echo "=== ORGANIZATIONAL ===" >> /tmp/audit.txt
git log --since="30 days ago" --oneline | grep -E "feat:|fix:" | wc -l >> /tmp/audit.txt
rg "remember to|don't forget" docs/ -i | wc -l >> /tmp/audit.txt
```

**Review `/tmp/audit.txt` for high counts indicating issues**

---

### Step 2: Triage (15 minutes)

**From scan results, identify top 10 by severity:**

**Severity matrix:**
- **Critical** - Security vulnerabilities, data loss risk, production blockers
- **High** - Blocking development, significant performance impact, major tech debt
- **Medium** - Maintenance burden, developer experience, moderate risk
- **Low** - Minor improvement, cosmetic, low risk

**Effort estimation:**
- **Quick win** (<4h) - Rename, add docs, simple refactor
- **Medium** (4-16h) - Extract classes, add tests, fix duplication
- **Large** (>16h) - Architectural changes, large-scale refactoring

**Top 10 = Highest severity + Lowest effort (ROI = Severity / Effort)**

---

### Step 3: Document (15 minutes)

**Investigation file structure:**

```markdown
# Investigation: Quick Audit Scan

**Date:** YYYY-MM-DD
**Status:** Complete
**Investigator:** Claude (codebase-audit-quick skill)
**Scan Duration:** [X minutes]

---

## TLDR

**Top 10 findings identified** across security, performance, tests, architecture, organizational

**Recommended next step:** Run focused audit for [category with most high-severity findings]

**Quick wins available:** [Count of findings with <4h effort]

---

## Top 10 Findings (Sorted by ROI)

### 1. [Finding Name] (Severity: Critical/High/Medium, Effort: <4h/4-16h/>16h)

**Category:** Security/Performance/Tests/Architecture/Organizational

**Issue:** [One sentence describing the problem]

**Evidence:** [Quick pointer - file path, line count, or command showing issue]

**Impact:** [Why this matters]

**Quick fix:** [What to do - 1-2 sentences]

**ROI:** High/Medium/Low

---

### 2-10. [Following same structure]

---

## Scan Summary

**Total patterns scanned:** 15+ automated searches

**Findings by category:**
- Security: [count] potential issues
- Performance: [count] potential issues
- Tests: [count] potential issues
- Architecture: [count] potential issues
- Organizational: [count] potential issues

**Baseline metrics:**
- Total files: [count]
- Total lines: [count]
- Largest file: [path] ([lines] lines)
- Test coverage: [X modules without tests]
- ROADMAP drift: [X completed but marked TODO]

---

## Recommended Next Steps

**Immediate actions (quick wins <4h):**
- [ ] [Finding #X] - [Quick fix]

**Focused audits needed:**
- [ ] Run `codebase-audit-[category]` for [specific area with most critical findings]
- [ ] Run `codebase-audit-[category]` for [second priority area]

**Schedule:**
- This week: Address quick wins
- Next week: Run focused audit for [highest priority category]
- Next month: Re-run quick scan to measure improvement

---

## Reproducibility

**Commands to re-run scan:**
See Step 1 automated scan commands above.

**Re-scan schedule:** Monthly (track trend over time)
```

---

## Usage Notes

**When to use quick scan:**
- ✅ Monthly health monitoring
- ✅ Before starting major work (identify risks)
- ✅ Onboarding to unfamiliar codebase
- ✅ Deciding which focused audit to run

**When NOT to use quick scan:**
- ❌ You know the problem area (use focused audit instead)
- ❌ Need deep analysis (quick scan is surface-level)
- ❌ Investigation requires manual code reading

**Follow-up workflow:**
1. Run quick scan
2. Identify category with most critical findings
3. Run focused audit: `codebase-audit-[category]`
4. Address high-priority findings
5. Re-run quick scan in 1 month to measure improvement

---

## Anti-Patterns

**❌ Treating quick scan as comprehensive**
- Quick scan is triage, not deep analysis
✅ **Fix:** Use focused audits for thorough investigation

**❌ No follow-up action**
- Running scan without addressing findings
✅ **Fix:** Always identify at least one quick win to fix immediately

**❌ No baseline tracking**
- Can't measure improvement over time
✅ **Fix:** Re-run monthly, track metrics trend

---

*This skill provides rapid health check across all audit areas, enabling quick triage and informed decision on which focused audit to run next.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: self-review -->
<!-- Auto-generated from phases/self-review.md -->

# Self-Review (Mandatory)

Before completing the audit, verify quality of findings and recommendations.

---

## Audit-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Evidence concrete** | Each finding has file:line reference | Add specific locations |
| **Reproducible** | Pattern searches documented | Add grep/glob commands used |
| **Prioritized** | Recommendations ranked by ROI | Add severity/effort matrix |
| **Actionable** | Each recommendation has clear next step | Make specific |
| **Baseline captured** | Metrics for re-audit comparison | Add counts/percentages |

---

## Self-Review Checklist

### 1. Findings Quality

- [ ] **Each finding has evidence** - Concrete file:line references, not "there are issues"
- [ ] **Pattern searches documented** - grep/glob commands that found issues
- [ ] **False positives filtered** - Reviewed results, removed non-issues
- [ ] **Severity assessed** - Each finding has impact level (critical/high/medium/low)

### 2. Recommendations Quality

- [ ] **Prioritized by ROI** - High impact, low effort items first
- [ ] **Actionable** - Each recommendation specifies what to do
- [ ] **Scoped** - Recommendations are achievable (not "rewrite everything")
- [ ] **Linked to findings** - Each recommendation traces to specific findings

### 3. Documentation Quality

- [ ] **Investigation file complete** - All sections filled
- [ ] **Baseline metrics** - Numbers for future comparison
- [ ] **Reproduction commands** - Someone can re-run the audit
- [ ] **NOT DONE claims verified** - For each 'NOT DONE' or 'NOT IMPLEMENTED' finding, confirmed with file/code search (not just artifact reading)

### 4. Commit Hygiene

- [ ] Conventional format (`audit:` or `chore:`)
- [ ] Investigation file committed

### 5. Discovered Work Check

*Audits typically discover actionable work. Track it in beads so it doesn't get lost.*

| Type | Examples | Action |
|------|----------|--------|
| **Security bugs** | Vulnerabilities, injection risks | `bd create "SECURITY: description" --type bug` |
| **Architecture issues** | God objects, tight coupling, tech debt | `bd create "ARCHITECTURE: description" --type task` |
| **Performance issues** | N+1 queries, missing indexes | `bd create "PERFORMANCE: description" --type bug` |
| **Missing tests** | Coverage gaps, critical paths untested | `bd create "TESTING: description" --type task` |
| **Strategic Unknowns** | Architectural/premise questions discovered | `bd create "description" --type question` |

**Triage labeling for daemon processing:**

After creating issues, apply triage labels based on finding severity:

| Severity | Label | When to use |
|----------|-------|-------------|
| Critical/High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Medium/Low | `triage:review` | Needs orchestrator review before work starts |

Example:
```bash
bd create "SECURITY: SQL injection in api.py:123" --type bug
bd label <issue-id> triage:ready  # Critical severity, clear fix
```

**Why this matters:** Issues labeled `triage:ready` are automatically picked up by the work daemon for autonomous processing. Critical/High severity issues have clear scope and can be worked immediately; Medium/Low issues benefit from orchestrator review first.

**Checklist:**
- [ ] **Reviewed recommendations** - Checked audit recommendations for actionable items
- [ ] **Tracked if applicable** - Created beads issues for high-priority items (or noted "No actionable items")
- [ ] **Included in summary** - Completion comment mentions tracked issues (if any)

**If no actionable items:** Note "No beads issues created - recommendations are informational only" in completion comment.

**Why this matters:** Audits produce recommendations that often require follow-up work. Beads issues ensure these surface in SessionStart context rather than getting buried in audit files.

---

## Report via Beads

**If self-review finds issues:**
1. Fix them before proceeding
2. Report: `bd comment <beads-id> "Self-review: Fixed [issue summary]"`

**If self-review passes:**
- Report: `bd comment <beads-id> "Self-review passed - ready for completion"`

**Checklist summary (verify mentally, report issues only):**
- Findings: Evidence with file:line, pattern searches documented, false positives filtered, severity assessed
- Recommendations: Prioritized by ROI, actionable, scoped, linked to findings
- Documentation: Investigation file complete, baseline metrics, reproduction commands
- Discovered work: Reviewed for actionable items, tracked in beads or noted "No actionable items"

**Only proceed to completion after self-review passes.**

---

## Completion Criteria

Before marking complete:

- [ ] Self-review passed
- [ ] **Leave it Better completed:** At least one `kb quick` command run OR noted as not applicable
- [ ] Investigation file complete with all findings
- [ ] Recommendations prioritized and actionable
- [ ] Baseline metrics documented for re-audit
- [ ] Pattern search commands documented (reproducibility)
- [ ] Discovered work reviewed and tracked (or noted "No actionable items")
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Complete - [findings summary]"`

**If ANY box unchecked, audit is NOT complete.**

---

**After completing all criteria:**

1. Verify all checkboxes marked
2. Report completion: `bd comment <beads-id> "Phase: Complete - Audit findings: [count], Recommendations: [count]"`
3. Call /exit to close agent session

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: leave-it-better -->
<!-- Auto-generated from phases/leave-it-better.md -->

# Leave it Better (Mandatory)

**Purpose:** Every session should leave the codebase, documentation, or knowledge base better than you found it.

**When you're in this phase:** Self-review has passed. Before marking complete, externalize what you learned.

---

## Why This Matters

Knowledge lost to session boundaries is the #1 cause of repeated mistakes and wasted effort. Every session should deposit something into the knowledge base.

**Common examples of lost knowledge:**
- "We tried X but it didn't work because Y" (others will try X again)
- "This works but only if Z is configured this way" (constraint not documented)
- "We chose A over B because..." (decision not recorded)

---

## What to Externalize

**Before marking complete, you MUST externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kb quick decide` | `kb quick decide "Use Redis for sessions" --reason "Need distributed state for horizontal scaling"` |
| Tried something that failed | `kb quick tried` | `kb quick tried "SQLite for sessions" --failed "Race conditions with multiple workers"` |
| Discovered a constraint | `kb quick constrain` | `kb quick constrain "All endpoints must be idempotent" --reason "Retry logic requires safe replay"` |
| Found an open question | `kb quick question` | `kb quick question "Should we rate-limit per-user or per-IP?"` |

---

## Quick Checklist

- [ ] **Reflected on session:** What did I learn that the next agent should know?
- [ ] **Externalized at least one item:** Ran `kb quick decide/tried/constrain/question`
- [ ] **Improved something:** Fixed a typo, clarified docs, added a missing comment, or updated stale info (optional but encouraged)

---

## If Nothing to Externalize

If the work was straightforward implementation with no new learnings:

1. Note in your completion comment: "Leave it Better: No new knowledge to externalize - straightforward implementation"
2. This is acceptable but should be rare

**Common case:** Even "straightforward" work often reveals something worth capturing (edge case, gotcha, or clarification).

---

## Examples

**Good externalization after feature work:**
```bash
kb quick decide "Use optimistic locking for updates" --reason "Prevents lost updates without blocking reads"
kb quick tried "Pessimistic locking" --failed "Caused deadlocks under high concurrency"
```

**Good externalization after debugging:**
```bash
kb quick constrain "Cache invalidation requires explicit call" --reason "TTL alone causes stale reads"
```

**Good externalization after investigation:**
```bash
kb quick question "Is the legacy API still used? Found no callers but unclear if external consumers exist"
```

---

## Completion Criteria (Leave it Better)

- [ ] Reflected on what was learned during the session
- [ ] Ran at least one `kb quick` command OR documented why nothing to externalize
- [ ] Included "Leave it Better" status in completion comment

**Only proceed to final completion after Leave it Better is done.**

<!-- /SKILL-TEMPLATE -->






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
2. `bd comment orch-go-1036 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.
