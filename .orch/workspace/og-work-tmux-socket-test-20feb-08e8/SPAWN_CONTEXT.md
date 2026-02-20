TASK: tmux socket test from inside overmind


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## CONFIG RESOLUTION

- Backend: claude (source: derived (model-compatibility))
- Model: anthropic/claude-opus-4-5-20251101 (source: user-config)
- Tier: full (source: heuristic (skill-default))
- Spawn Mode: tmux (source: cli-flag)
- MCP: none (source: default)
- Mode: tdd (source: default)
- Validation: tests (source: default)




## PRIOR KNOWLEDGE (from kb context)

**Query:** "tmux socket test"

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
- Agents must not spawn more than 3 iterations without human review
  - Reason: Prevents runaway iteration loops like 12 tmux fallback tests in 9 minutes
- orch status can show phantom agents (tmux windows where OpenCode exited)
  - Reason: No reconciliation between tmux liveness and OpenCode session state
- orch status counts ALL workers-* tmux windows as active
  - Reason: Discovered during phantom agent investigation - status inflated by persistent windows
- External integrations require manual smoke test before Phase: Complete
  - Reason: OAuth feature shipped with passing tests but failed real-world use. Tests couldn't cover actual OAuth flow with Anthropic.
- Worker agents must test spawn functionality via unit tests and code review, not end-to-end spawning
  - Reason: Prevents recursive spawn testing incidents while still enabling verification

### Prior Decisions
- orch-go tmux spawn is fire-and-forget - no session ID capture
  - Reason: opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring.
- Use IsOpenCodeReady for tmux TUI detection
  - Reason: Reliably detects when OpenCode TUI is ready for input in a tmux pane, avoiding race conditions.
- Refactor shell-out functions for testability
  - Reason: Extracting command construction into Build*Command functions allows unit testing of logic without side effects
- Iteration 8 tmux fallback verification successful
  - Reason: After attach mode implementation (7ca8438), all three fallback mechanisms (tail, question, status) continue to work correctly with no regressions
- Temporal density and repeated constraints are highest value reflection signals
  - Reason: Low noise, high actionability - tested against real kn/kb data
- Constraint validity tested by implementation, not age
  - Reason: Constraints become outdated when code contradicts them (implementation supersession), not when time passes. Test with rg before assuming constraint holds.
- Session ID resolution pattern
  - Reason: Commands that need to find agents should use resolveSessionID or the runTail pattern: workspace files first, then API lookup, then tmux fallback
- Dual-mode architecture (tmux for visual, HTTP for programmatic) is the correct design
  - Reason: Investigation confirmed each mode serves distinct, irreplaceable needs
- Post-registry lifecycle uses 4 state sources: OpenCode sessions, tmux windows, beads issues, workspaces
  - Reason: Registry removed due to false positive completion detection; derived lookups replace central state
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring

### Models (synthesized understanding)
- Probe: Dashboard Blind to Claude CLI Tmux Agents
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-17-dashboard-blind-to-tmux-agents.md
  - Recent Probes:
    - 2026-02-20-model-drift-major-restructuring
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-20-model-drift-major-restructuring.md
    - 2026-02-19-agents-api-closed-issues-filter
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-19-agents-api-closed-issues-filter.md
    - 2026-02-18-filter-unspawned-issues
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-18-filter-unspawned-issues.md
    - 2026-02-18-cross-project-visibility-cache-context
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-18-cross-project-visibility-cache-context.md
    - 2026-02-18-session-status-empty-phantoms
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-18-session-status-empty-phantoms.md
- Probe: Tmux Readiness Timeout Root Cause
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-18-tmux-readiness-timeout.md
  - Recent Probes:
    - 2026-02-20-probe-skill-content-template-fix-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-skill-content-template-fix-verification.md
    - 2026-02-20-skill-content-template-processing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-skill-content-template-processing.md
    - 2026-02-20-probe-authority-section-dedup
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-authority-section-dedup.md
    - 2026-02-20-spawn-bloat-analysis-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-spawn-bloat-analysis-gap.md
    - 2026-02-20-probe-orientation-frame-dedup-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-orientation-frame-dedup-verification.md
- Probe: Daemon Test Fail-Fast Fix
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-test-fail-fast-fix.md
  - Recent Probes:
    - 2026-02-19-probe-daemon-go-extraction-completeness
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-go-extraction-completeness.md
    - 2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md
    - 2026-02-19-probe-config-surface-area-extraction
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-config-surface-area-extraction.md
    - 2026-02-18-probe-project-config-spawn-mode-explicitness
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-18-probe-project-config-spawn-mode-explicitness.md
    - 2026-02-18-probe-daemon-cleanup-after-pkg-cleanup-deletion
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-18-probe-daemon-cleanup-after-pkg-cleanup-deletion.md
- Probe: Test Suite Health — New Failures in Daemon Extraction
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-16-test-suite-health-new-failures.md
  - Recent Probes:
    - 2026-02-19-probe-daemon-go-extraction-completeness
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-go-extraction-completeness.md
    - 2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md
    - 2026-02-19-probe-config-surface-area-extraction
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-config-surface-area-extraction.md
    - 2026-02-18-probe-project-config-spawn-mode-explicitness
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-18-probe-project-config-spawn-mode-explicitness.md
    - 2026-02-18-probe-daemon-cleanup-after-pkg-cleanup-deletion
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-18-probe-daemon-cleanup-after-pkg-cleanup-deletion.md
- Worker Detection Stress Test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-stress-test.md
  - Recent Probes:
    - 2026-02-14-worker-detection-stress-test
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-stress-test.md
    - 2026-02-14-worker-detection-header-implementation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-header-implementation.md
    - 2026-02-14-metrics-redesign-architecture-validation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-metrics-redesign-architecture-validation.md
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
- Escape Hatch Visibility Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/escape-hatch-visibility-architecture.md
  - Summary:
    **Core insight:** The architectural choice of dual-window Ghostty setup isn't just "nice to have" - it's a **required component** of escape-hatch spawning architecture.
    
    ```
    Critical Infrastructure Work
      → Requires Escape Hatch (independence + visibility + capability)
        → Visibility Requires --tmux Flag
          → --tmux Requires Dual-Window Setup
            → Dual-Window Requires Auto-Switch Hook
    ```
    
    Remove any link in this chain and the visibility criterion fails.
    
    ---
    
    **Primary Evidence (Verify These):**
    - `pkg/spawn/backend.go` - Backend selection logic (--backend claude flag handling)
    - `pkg/spawn/spawn.go` - Spawn mode routing (headless vs tmux)
    - `~/.tmux.conf.local:58-61` - Auto-switch hook configuration
    - `~/.local/bin/sync-workers-session.sh` - Workers session auto-switching script
    - `cmd/orch/spawn.go` - Spawn command with escape-hatch flags
    - Dashboard code showing headless agent monitoring as alternative
  - Your findings should confirm, contradict, or extend the claims above.
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
  - Summary:
    **At N=11, the model pattern shows exceptional consistency and proven utility.** All 11 models converged on the 6-section structure without enforcement. The enable/constrain query works across every domain tested. Most significantly: **the models that emerged reveal your cognitive investment priorities** - hot paths (spawn, agent, dashboard), strategic understanding (orchestrator, daemon), and owned complexity (completion, beads integration).
    
    **Key finding:** High investigation count + model existence = **friction that refused to resolve**. The absence of models for external dependencies (kb, tmux) despite high investigation counts reveals clear ownership boundaries.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Agent Lifecycle State Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model.md
  - Summary:
    Agent state exists across **four independent layers** (tmux windows, OpenCode in-memory, OpenCode on-disk, beads comments). These layers fall into two distinct categories: **state layers** (beads, workspace files) that represent what work was done, and **infrastructure layers** (OpenCode sessions, tmux windows) that represent transient execution resources. The dashboard reconciles these via a **Priority Cascade**: check beads issue status first (highest authority), then Phase comments, then SYNTHESIS.md existence, then session status. Agents are discovered via a **two-lane architecture**: tracked work (beads-first via `queryTrackedAgents`) and untracked sessions (OpenCode session list). Status can appear "wrong" at the dashboard level while being "correct" at each individual layer - this is a measurement artifact from combining multiple sources of truth.
    
    ---
  - Critical Invariants:
    1. **Phase: Complete is agent's declaration** - Only agent can reach this, not orchestrator
    2. **Beads issue closed = canonical completion** - All status queries defer to beads
    3. **Session existence ≠ agent still working** - Sessions persist indefinitely
    4. **Status checks don't mutate state** - `determineAgentStatus()` is a pure function, no side effects
    5. **Multiple sources must be reconciled** - No single source has complete truth; query engine joins with reason codes
    6. **Tmux windows are UI layer only** - Not authoritative for state
    7. **No persistent lifecycle caches** - Only in-memory, process-local caches with short TTLs allowed. Disk-backed state (registry, sessions.json, state.db) is structurally prohibited by architecture lint tests
    8. **Silent failures must be visible** - Every missing field gets an explicit reason code, never empty metadata
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Dashboard Shows "Active" When Agent is Done
    
    **Symptom:** Dashboard shows agent as active, but `bd show <id>` says status=closed
    
    **Root cause:** Dashboard caching or SSE lag - hasn't received beads update yet
    
    **Why it happens:**
    
    - Agent reaches Phase: Complete
    - `orch complete` closes beads issue
    - Beads issue status = closed
    - Dashboard hasn't refreshed or polled beads yet
    - Dashboard still shows cached "active" state
    
    **Fix:** Refresh dashboard browser tab (forces beads query)
    
    **NOT the fix:** Deleting OpenCode session (treats symptom, not cause)
    
    ### Failure Mode 2: Completed Agents Showing Wrong Status
    
    **Symptom:** Agent completed work but dashboard shows unexpected status
    
    **Root cause:** Completion signals exist but session is dead, creating ambiguity
    
    **How the Priority Cascade handles this (current):**
    
    - If beads issue closed → "completed" (Priority 1, regardless of session state)
    - If Phase: Complete + session dead → "awaiting-cleanup" (Priority 2, needs orch complete)
    - If Phase: Complete + session alive → "completed" (Priority 3)
    - If SYNTHESIS.md exists + session dead → "awaiting-cleanup" (Priority 4)
    - If SYNTHESIS.md exists + session alive → "completed" (Priority 5)
    
    **Fix (Jan 8, refined Feb 2026):** Priority Cascade puts beads/Phase check before session existence check. The `awaiting-cleanup` status (added Feb 2026) distinguishes completed-but-orphaned agents from truly dead agents.
    
    ### Failure Mode 3: Agent Went Idle But Not Complete
    
    **Symptom:** Session status is "idle" but no `Phase: Complete` comment
    
    **Root cause:** Agent ran out of context, crashed, or didn't follow completion protocol
    
    **Why it happens:**
    
    - Session exhausts context (150k tokens)
    - Agent stops responding
    - SSE event: `session.status = idle`
    - No `Phase: Complete` was ever written
    - Dashboard shows "idle" or "waiting"
    
    **This is expected behavior.** Session idle ≠ work complete. Only agents that explicitly run `bd comment <id> "Phase: Complete"` are considered done.
    
    **Fix:** Check workspace for what agent accomplished, then either:
    
    - `orch complete <id> --force` if work is done
    - `orch abandon <id>` if work is incomplete
    
    ### Failure Mode 4: Cross-Project Agents Not Visible
    
    **Symptom:** Agent spawned with `--workdir /other/project` doesn't appear in dashboard
    
    **Root cause:** Dashboard only scans current project's `.orch/workspace/` directory
    
    **Why it happens:**
    
    - Workspace created in `/other/project/.orch/workspace/`
    -
    ... [truncated]
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-20-model-drift-major-restructuring
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-20-model-drift-major-restructuring.md
    - 2026-02-19-agents-api-closed-issues-filter
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-19-agents-api-closed-issues-filter.md
    - 2026-02-18-filter-unspawned-issues
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-18-filter-unspawned-issues.md
    - 2026-02-18-cross-project-visibility-cache-context
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-18-cross-project-visibility-cache-context.md
    - 2026-02-18-session-status-empty-phantoms
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-18-session-status-empty-phantoms.md
- Probe: Default Backend + Anthropic Model Incompatibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-probe-default-backend-anthropic-incompatibility.md
  - Recent Probes:
    - 2026-02-20-model-aware-backend-routing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-model-aware-backend-routing.md
    - 2026-02-20-probe-default-backend-anthropic-incompatibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-probe-default-backend-anthropic-incompatibility.md
    - 2026-02-20-backend-resolution-architecture-drift
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-backend-resolution-architecture-drift.md
    - 2026-02-19-probe-opencode-mcp-wiring
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-19-probe-opencode-mcp-wiring.md

### Guides (procedural knowledge)
- Tmux Spawn Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/tmux-spawn-guide.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Completion Gates
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion-gates.md
- Worker Patterns Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/worker-patterns.md
- Code Extraction Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/code-extraction-patterns.md
- Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/status.md
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Headless Spawn Mode Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/headless.md
- Two-Tier Sensing Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/two-tier-sensing-pattern.md

### Related Investigations
- Fix Tmux Socket Path Orch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-15-inv-fix-tmux-socket-path-orch.md
- tmux concurrent zeta - 6th concurrent spawn test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-zeta.md
- Test Headless Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-22-inv-test-headless-spawn.md
- Test Verify Daemon Skip Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-test-verify-daemon-skip-functionality.md
- Test Spawn Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-test-spawn-functionality.md
- Test Orch Go Directory
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-22-inv-test-orch-go-directory.md
- Test Fix Nested Skillc
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-test-fix-nested-skillc.md
- Tmux Concurrent Epsilon Spawn Capability
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-epsilon.md
- Test Hotspot Warning Cmd Orch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-test-hotspot-warning-cmd-orch.md
- Test Tmux Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-test-tmux-spawn.md

### Failed Attempts (DO NOT repeat)
- orch tail on tmux agent
- orch clean to remove ghost sessions automatically

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.






📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — the workspace often has unrelated changes (.autorebuild.lock, .beads/, build/).
Stage ONLY the specific files you created or modified for your task, by name.


1. **CREATE SYNTHESIS.md** in your workspace with:
   - Plain-Language Summary (what was built/found)
   - Delta (what changed)
2. **COMMIT YOUR WORK:**
   ```bash
   git add <files you changed>
   git commit -m "feat: [brief description of changes]"
   ```
3. Run: `/exit` to close the agent session


⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Worker rule: Commit your work, call `/exit`. Don't push.




CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go

SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours


AUTHORITY:
Authority delegation rules are provided via skill guidance (worker-base skill).
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

2. [Task-specific deliverables]


3. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-tmux-socket-test-20feb-08e8/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Track progress via beads comments. Call /exit to close agent session when done.




## SKILL GUIDANCE (hello)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: hello
skill-type: procedure
audience: worker
spawnable: true
category: test
description: Simple test skill that prints hello and exits
allowed-tools:
- Bash
deliverables: []
verification:
  requirements: |
    - [ ] Agent printed "Hello from orch-go!"
  test_command: null
  required: true
---

# Hello Skill

**Purpose:** Test the spawn system with a trivial task.

## Directive

When spawned with this skill, print the following message exactly:

```
Hello from orch-go!
```

Then exit immediately by running `/exit`.

## Notes

- This is a minimal test skill to verify spawn functionality.
- No other actions are required.
- The verification is simply that the message was printed.

---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — stage ONLY your task files by name.



1. Create SYNTHESIS.md in your workspace
2. **COMMIT YOUR WORK:** `git add <files you changed> && git commit -m "feat: [description]"`
3. `/exit`



⛔ **NEVER run `git push`** - Workers commit locally only.
⚠️ Your work is NOT complete until Phase: Complete is reported (or /exit for --no-track).
