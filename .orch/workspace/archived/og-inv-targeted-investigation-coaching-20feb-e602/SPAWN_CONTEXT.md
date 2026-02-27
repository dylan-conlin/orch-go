TASK: Targeted investigation: How does the coaching plugin inject content into agent conversations?

Read .opencode/plugins/coaching.ts and trace the injectHealthSignal function:
1. What mechanism does it use to inject content? (system prompt modification, new message, tool response modification, etc.)
2. Where would injected content appear in the database? (message table, part table, or nowhere?)
3. Does the plugin have any runtime logging we can check?
4. Is there a test or way to verify the plugin is actually firing during tool.execute.after?

Also check: Does the tool.execute.after hook actually fire for the gpt-5.2-codex model/agent type? Maybe it only fires for certain configurations.

Deliverable: Write findings to SYNTHESIS.md in your workspace.


SPAWN TIER: light

⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.



## CONFIG RESOLUTION

- Backend: opencode (source: cli-flag)
- Model: openai/gpt-5.2-codex (source: cli-flag)
- Tier: light (source: cli-flag)
- Spawn Mode: headless (source: default)
- MCP: none (source: default)
- Mode: tdd (source: default)
- Validation: tests (source: default)




## PRIOR KNOWLEDGE (from kb context)

**Query:** "targeted investigation how"

### Constraints (MUST respect)
- Registry is caching layer, not source of truth - all data exists in OpenCode/tmux/beads
  - Reason: Investigation found all registry data can be derived from primary sources
- orch status counts ALL workers-* tmux windows as active
  - Reason: Discovered during phantom agent investigation - status inflated by persistent windows
- D.E.K.N. 'Next:' field must be updated when marking Status: Complete
  - Reason: Prevents stale investigations that mislead future agents
- LLM guidance compliance requires signal balance - overwhelming counter-patterns (56:13 ratio) drowns specific exceptions
  - Reason: Investigation found orchestrator skill has 4:1 ask-vs-act signal ratio causing autonomy guidance to fail
- Orchestrator is AI, not Dylan - Dylan interacts with AI orchestrators who spawn/complete agents
  - Reason: Investigation 2025-12-25-design-orchestrator-completion-lifecycle-two incorrectly framed Dylan as the actor who spawns agents. The actor model is: Dylan ↔ AI Orchestrator ↔ Worker Agents. Mental model sync flows: Agent→Orchestrator (synthesis) and Orchestrator→Dylan (conversation).
- Investigation filenames are generated from task description at spawn time
  - Reason: generateSlug() runs at spawn before agent starts - cannot reflect findings in initial filename
- Ask 'should we' before 'how do we' for strategic direction changes
  - Reason: Epic orch-go-erdw was created assuming skills-as-value was correct direction. Architect review revealed the premise was wrong - current separation is intentional design. Wasted work avoided by validating premise before execution.

### Prior Decisions
- OpenCode ListSessions WITH x-opencode-directory header returns disk sessions, WITHOUT returns in-memory
  - Reason: Finding from investigation - explains 2 vs 238 session count discrepancy
- Investigations live in .kb/ not workspaces
  - Reason: kb context discoverability essential; SYNTHESIS.md bridges via investigation_path pointer
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- orch-go is primary CLI, orch-cli (Python) is reference/fallback
  - Reason: Go provides better primitives (single binary, OpenCode HTTP client, goroutines); Python taught requirements through 27k lines and 200+ investigations
- D.E.K.N. is universal handoff structure
  - Reason: Delta/Evidence/Knowledge/Next enables 30-second context transfer between Claude instances - proven across SYNTHESIS.md, investigations, and session handoffs
- kb reflect uses single command with --type flag for four reflection modes
  - Reason: Consistent with kb pattern, most discoverable, extensible. Maps directly to signals from ws4z investigations.
- Self-reflection is signal-triggered not time-scheduled
  - Reason: Density thresholds (3+ investigations) produce actionable signals; time intervals (weekly review) produce noise. Per ws4z.8 investigation.
- Dual-mode architecture (tmux for visual, HTTP for programmatic) is the correct design
  - Reason: Investigation confirmed each mode serves distinct, irreplaceable needs
- Confidence scores in investigations are noise, not signal
  - Reason: Nov 2025 Codex case: 5 investigations at High/Very High confidence all reached wrong conclusions. Current distribution: 90% cluster at 85-95%, providing no discriminative value. Recommendation: Replace with structured uncertainty (tested/untested enumeration).
- kb-cli owns artifact templates (investigation, decision, guide, research)
  - Reason: Consolidation complete - skill templates/ directories removed, kb-cli hardcoded updated with D.E.K.N.

### Models (synthesized understanding)
- Probe: Are completion gates catching defects or generating bypass noise after targeted skips?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-09-friction-bypass-analysis-post-targeted-skips.md
  - Recent Probes:
    - 2026-02-19-probe-glass-removal-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-glass-removal-verification.md
    - 2026-02-19-probe-accretion-enforcement-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-accretion-enforcement-gap-analysis.md
    - 2026-02-19-probe-coupling-hotspot-detection-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md
    - 2026-02-18-probe-entropy-spiral-fix-commit-relevance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
- Phase 3 Review: Model Pattern Analysis (N=5)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE3_REVIEW.md
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
  - Summary:
    **At N=11, the model pattern shows exceptional consistency and proven utility.** All 11 models converged on the 6-section structure without enforcement. The enable/constrain query works across every domain tested. Most significantly: **the models that emerged reveal your cognitive investment priorities** - hot paths (spawn, agent, dashboard), strategic understanding (orchestrator, daemon), and owned complexity (completion, beads integration).
    
    **Key finding:** High investigation count + model existence = **friction that refused to resolve**. The absence of models for external dependencies (kb, tmux) despite high investigation counts reveals clear ownership boundaries.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Models
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/README.md
- kb reflect Cluster Hygiene
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/kb-reflect-cluster-hygiene.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-14.
    Changed files: .kb/investigations/archived/, .kb/investigations/synthesized/, pkg/spawn/config.go.
    Verify model claims about these files against current code.
  - Summary:
    `kb reflect --type synthesis` clusters investigations by lexical similarity. This is a useful discovery signal, but it is not automatically a valid synthesis boundary. Effective triage requires a second step: classify each cluster by semantic cohesion, verify key claims against current code/behavior, then route to one of three dispositions: **converge** (create/update decision/model), **split** (separate mixed lineages), or **demote** (use probe/quick artifact instead of full investigation).
    
    ---
  - Critical Invariants:
    1. **Lexical cluster != conceptual model**
    2. **Code/test evidence outranks archived claims**
    3. **One canonical decision/model per mechanism**
    4. **Redundant investigations must point to canonical artifact**
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Lexical collision
    
    `feature` clusters can mix tiering behavior, cross-repo implementation tasks, and decision-gate debugging. Treating them as one topic creates noisy synthesis.
    
    ### Failure Mode 2: Time-drifted conclusions
    
    Investigation findings can become stale after code changes (for example, fail-open to fail-closed gate behavior). Re-validation is required during consolidation.
    
    ### Failure Mode 3: Artifact overuse
    
    Quick fact lookups become full investigations, increasing maintenance burden without adding durable understanding.
    
    ### Failure Mode 4: Incomplete closure metadata
    
    Archived files without clear `Superseded-By` pointers remain discoverable but ambiguous, causing repeated re-triage.
    
    ### Failure Mode 5: Scans archived/synthesized directories
    
    kb reflect scans `.kb/investigations/archived/` and `.kb/investigations/synthesized/` directories, creating false positives for already-processed clusters.
    
    **Evidence:** Extract synthesis (Jan 17) moved 14 investigations to `synthesized/code-extraction-patterns/`, but kb reflect still reported "13 investigations need synthesis" by scanning the synthesized directory.
    
    **Impact:** Agents repeatedly triage and investigate the same clusters thinking they need synthesis, when they've already been consolidated into guides.
    
    **Workaround:** Manually verify guide completeness and ignore kb reflect output for topics with synthesized/ or archived/ directories.
    
    **Fix needed:** kb-cli synthesis detection should exclude `.kb/investigations/archived/` and `.kb/investigations/synthesized/` from scanning.
    
    **Source:** `2026-01-17-inv-synthesize-extract-investigation-cluster-13.md:59-69`, `2026-02-14-inv-synthesize-synthesize-investigations-10-synthesis.md`
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Completion Verification Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-14.
    Changed files: pkg/verify/check.go, pkg/verify/visual.go, .beads/issues.jsonl.
    Deleted files: pkg/verify/phase.go, pkg/verify/evidence.go, cmd/orch/complete.go.
    Verify model claims about these files against current code.
  - Summary:
    Completion verification operates through **three independent gates** (Phase, Evidence, Approval) that check different aspects of "done". Phase gate verifies agent claims completion, Evidence gate requires visual/test proof in beads comments, Approval gate (UI changes only) requires human sign-off. Verification is **tier-aware**: light tier checks Phase + commits, full tier adds SYNTHESIS.md, orchestrator tier checks SESSION_HANDOFF.md instead. The **5-tier escalation model** surfaces knowledge-producing work (investigation/architect/research) for mandatory orchestrator review before auto-closing. Cross-project detection uses SPAWN_CONTEXT.md to determine which directory to verify in. **Targeted bypasses** (`--skip-{gate} "reason"`) replace blanket `--force`, allowing specific gates to be skipped while others still run.
    
    ---
  - Why This Fails:
    ### 1. Evidence Gate False Positive
    
    
    **What happens:** Agent passes Evidence gate without actual visual verification.
    
    **Root cause:** Agent generates screenshot placeholder text ("Screenshot attached") without actually attaching screenshot. Evidence gate searches for keyword "screenshot", finds it, passes.
    
    **Why detection is hard:** Text-based keyword matching can't distinguish placeholder from actual proof.
    
    **Fix:** Approval gate for UI changes. Even if Evidence passes, human must verify via --approve.
    
    **Why this matters:** False positive on Evidence gate means broken UI ships thinking it's verified.
    
    ### 2. Approval Gate Bypass
    
    **What happens:** Non-UI changes accidentally avoid approval gate.
    
    **Root cause:** File path detection (`modifiedWebFiles()`) misclassifies files. `web-utils/` not under `web/`, approval skipped.
    
    **Why detection is hard:** File structure varies across projects. Heuristics (path contains "web") can miss edge cases.
    
    **Fix:** Explicit skill-based detection. `feature-impl` with UI flag requires approval, regardless of file paths.
    
    **Future:** Skill manifest declares "requires_ui_approval: true".
    
    ### 3. Cross-Project Verification Wrong Directory
    
    **What happens:** Verification runs in wrong directory, checks wrong tests, reports false failure.
    
    **Root cause:** `SPAWN_CONTEXT.md` missing PROJECT_DIR, fallback uses workspace location (orch-go), but agent worked in orch-cli.
    
    **Why detection is hard:** Workspace location != work location. No guaranteed signal of where work happened.
    
    **Fix:** `orch spawn --workdir` explicitly sets PROJECT_DIR in SPAWN_CONTEXT.md. Verification reads it.
    
    **Prevention:** Make --workdir mandatory for cross-project spawns, fail spawn if missing.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-19-probe-glass-removal-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-glass-removal-verification.md
    - 2026-02-19-probe-accretion-enforcement-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-accretion-enforcement-gap-analysis.md
    - 2026-02-19-probe-coupling-hotspot-detection-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md
    - 2026-02-18-probe-entropy-spiral-fix-commit-relevance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
- Daemon Autonomous Operation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-12.
    Changed files: pkg/daemon/daemon.go, pkg/daemon/skill_inference.go, pkg/daemon/completion_processing.go, pkg/daemon/spawn_tracker.go, cmd/orch/daemon.go.
    Verify model claims about these files against current code.
  - Summary:
    The daemon is an **autonomous agent spawner** that operates in a **poll-spawn-complete cycle**: polls beads for `triage:ready` issues, infers skill from issue type, spawns within capacity limits, monitors for `Phase: Complete`, verifies and closes. The daemon operates **independently of orchestrators** - orchestrators triage (label issues ready), daemon spawns (batch processing), orchestrators synthesize (review completed work). Skill inference uses **issue type** (not labels) to map task→investigation, bug→systematic-debugging, etc. Capacity management uses **WorkerPool** with reconciliation against OpenCode to free stale slots.
    
    ---
  - Why This Fails:
    ### 1. Capacity Starvation
    
    **What happens:** Pool shows MaxAgents active, but `orch status` shows fewer actual agents running.
    
    **Root cause:** Spawn failures don't release slots. Agent spawned, counted against pool, but spawn fails (bad skill name, missing context, etc.) - slot never released.
    
    **Why detection is hard:** Pool only knows about attempts, not outcomes. No feedback loop from spawn failure to pool.
    
    **Fix:** Reconciliation with OpenCode. Query actual sessions, release slots for non-existent agents.
    
    **Prevention:** Spawn tracking with retry limits (`pkg/daemon/spawn_tracker.go`).
    
    ### 2. Duplicate Spawns
    
    **What happens:** Same issue spawned multiple times by daemon on consecutive polls.
    
    **Root cause:** Spawn latency. Issue labeled `triage:ready` at poll N, daemon spawns, but spawn hasn't transitioned issue to `in_progress` by poll N+1. Daemon sees same issue still ready, spawns again.
    
    **Why detection is hard:** Race condition between poll interval (60s) and spawn transition time (variable).
    
    **Fix:** Spawn deduplication via tracking. Track spawned beads IDs in memory, skip on subsequent polls until status confirms transition.
    
    **Source:** `pkg/daemon/spawn_tracker.go`
    
    ### 3. Skill Inference Mismatch
    
    **What happens:** Daemon spawns wrong skill for issue type.
    
    **Root cause:** Issue type doesn't match actual work needed. User creates `task` for implementation work (should be `feature`), daemon infers `investigation`.
    
    **Why detection is hard:** Type is set at creation, can't be changed by daemon. Daemon trusts type.
    
    **Fix:** Manual override. Orchestrator updates issue type before labeling `triage:ready`, or spawns manually with correct skill.
    
    **Prevention:** Better issue creation prompts, type validation, skill override via labels (future).
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-20-probe-daemon-once-dedup-shared-spawn-path
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-20-probe-daemon-once-dedup-shared-spawn-path.md
    - 2026-02-20-probe-daemon-model-inference-from-skill
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-20-probe-daemon-model-inference-from-skill.md
    - 2026-02-19-probe-daemon-go-extraction-completeness
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-go-extraction-completeness.md
    - 2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md
    - 2026-02-19-probe-config-surface-area-extraction
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-config-surface-area-extraction.md
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
- Beads Integration Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-12.
    Changed files: pkg/beads/client.go, .beads/issues.jsonl.
    Deleted files: pkg/beads/fallback.go, pkg/beads/lifecycle.go, ~/.beads/daemon.sock, cmd/orch/spawn.go.
    Verify model claims about these files against current code.
  - Summary:
    Beads integration uses **RPC-first with CLI fallback** pattern: try native Go RPC client (fast, no process spawn), fall back to CLI subprocess if daemon unavailable. The integration operates at **three points in agent lifecycle**: spawn (create issue), work (report phase via comments), complete (close with reason). **Auto-tracking** creates issues automatically unless `--no-track` flag set. The RPC client lives in **pkg/beads** (never shell out with `exec.Command` directly) and provides 10x performance improvement over CLI (single RPC call vs subprocess spawn + JSON parse).
    
    ---
  - Why This Fails:
    ### 1. RPC Client Unavailable
    
    **What happens:** RPC calls fail, client falls back to CLI, performance degrades.
    
    **Root cause:** Beads daemon not running. RPC socket `~/.beads/daemon.sock` doesn't exist.
    
    **Why detection is hard:** Fallback is silent. No warning that RPC failed. User sees slow performance, doesn't know why.
    
    **Fix:** Start beads daemon: `bd daemon start` or ensure launchd starts it on boot.
    
    **Detection:** Log RPC failures, surface in `orch doctor` health check.
    
    ### 2. Beads ID Not Found
    
    **What happens:** `orch complete orch-go-abc1` fails with "issue not found".
    
    **Root cause:** Cross-project spawn. Issue created in orch-knowledge, but trying to complete from orch-go. Beads scoped to current directory's `.beads/`.
    
    **Why detection is hard:** Beads ID looks valid (correct format), but doesn't exist in current project's `.beads/issues.jsonl`.
    
    **Fix:** `cd` into correct project before completion, or use `--workdir` flag.
    
    **Prevention:** `orch complete` should detect project from workspace, auto-cd.
    
    ### 3. Auto-Tracking Creates Duplicates
    
    **What happens:** `orch spawn` creates issue, but issue already exists for same work.
    
    **Root cause:** User creates issue manually, then spawns with auto-tracking. Both create issue.
    
    **Why detection is hard:** No deduplication. Beads doesn't check if similar issue exists.
    
    **Fix:** Use `--issue <id>` flag to reference existing issue instead of auto-creating.
    
    **Prevention:** Better UX: `orch spawn` could check for related issues and prompt user.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-17-bd-sync-precommit-hook-deadlock
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-17-bd-sync-precommit-hook-deadlock.md
    - 2026-02-09-bd-sync-safe-timeout-retry-for-hash-mismatch
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-09-bd-sync-safe-timeout-retry-for-hash-mismatch.md
    - 2026-02-09-bd-sync-safe-post-sync-readiness-check
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-09-bd-sync-safe-post-sync-readiness-check.md
    - 2026-02-08-synthesis-dedup-parse-error-fail-closed
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-08-synthesis-dedup-parse-error-fail-closed.md
- Spawn Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture.md
  - Summary:
    Spawn evolved through 5 phases from basic CLI integration to daemon-driven automation with triage friction. The architecture creates a workspace with SPAWN_CONTEXT.md embedding skill content + task description + kb context, then launches a session via two-phase atomic spawn with rollback. Spawn settings are resolved via `pkg/spawn/resolve.go` with 6-level precedence and per-setting provenance tracking. The tier system (light/full) determines whether SYNTHESIS.md is required at completion. Triage friction (`--bypass-triage` flag) intentionally makes manual spawns harder to encourage daemon-driven workflow.
    
    ---
  - Critical Invariants:
    1. **Workspace name = kebab-case task description** - Used for tmux window, directory name, session title
    2. **Beads ID required for phase reporting** - `--no-track` creates untracked IDs that can't report to beads
    3. **KB context uses --global flag** - Cross-repo constraints are essential
    4. **Skill content stripped for --no-track** - Beads instructions removed when not tracking
    5. **Session scoping is per-project** - `orch send` only works within same directory hash
    6. **Token estimation at 4 chars/token** - Warning at 100k, error at 150k
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Cross-Project Spawn Sets Wrong Session Directory
    
    **Symptom:** `orch spawn --workdir /other/project` creates session with orchestrator's directory
    
    **Root cause:** Session directory is set from spawn caller's CWD, not `--workdir` target
    
    **Why it happens:**
    - OpenCode infers directory from process CWD
    - `--workdir` changes agent's working directory, not spawn process CWD
    - Session gets orchestrator directory, beads issue in orchestrator project
    
    **Impact:**
    - Sessions unfindable via directory filtering
    - Cross-project work tracking is split
    
    **Fix needed:** Pass explicit directory to OpenCode session creation
    
    ### Failure Mode 2: Token Limit Exceeded on Spawn
    
    **Symptom:** Spawn fails with "context too large" error
    
    **Root cause:** SPAWN_CONTEXT.md exceeds 150k token limit
    
    **Why it happens:**
    - Skill content (~10-40k tokens)
    - KB context can be large (30-50k tokens)
    - Task description minimal
    - Estimation: 4 chars/token
    
    **Fix (Dec 22):** Warning at 100k tokens, hard error at 150k with guidance
    
    ### Failure Mode 3: Daemon Spawns Blocked Issues
    
    **Symptom:** Daemon spawns issue that has blockers
    
    **Root cause:** Dependency checking missing in triage workflow
    
    **Why it happens:**
    - `bd ready` returns issues without blockers
    - Daemon spawns from `triage:ready` label (doesn't check dependencies)
    - Race condition: issue labeled before dependencies checked
    
    **Fix (Jan 3):** Dependency gating with `--force` override flag
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-20-probe-session-scope-template-honor
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-session-scope-template-honor.md
    - 2026-02-20-probe-progressive-skill-disclosure-design
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-progressive-skill-disclosure-design.md
    - 2026-02-20-probe-skill-content-template-fix-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-skill-content-template-fix-verification.md
    - 2026-02-20-skill-content-template-processing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-skill-content-template-processing.md
    - 2026-02-20-probe-authority-section-dedup
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-authority-section-dedup.md

### Guides (procedural knowledge)
- How Spawn Works
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawn.md
- Synthesis Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/synthesis-workflow.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Two-Tier Sensing Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/two-tier-sensing-pattern.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Beads Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/beads-integration.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- OpenCode Integration Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode.md
- Code Extraction Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/code-extraction-patterns.md

### Related Investigations
- Investigate Skills Produce Investigation Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-investigate-skills-produce-investigation-artifacts.md
- Synthesize Synthesize Investigations 10 Synthesis
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-synthesize-synthesize-investigations-10-synthesis.md
- Emerging Pattern - "How Would the System Recommend..."
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-investigate-emerging-pattern-how-would.md
- Synthesize Synthesis Investigations (26 Total)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-synthesis-investigations-26-synthesis.md
- Post-Synthesis Investigation Archival Workflow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-design-post-synthesis-investigation-archival.md
- Synthesis of 15 Skill Investigations (Dec 2025 - Jan 2026)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-synthesize-skill-investigations-15-synthesis.md
- Synthesis of 10 Completion Investigations
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-completion-investigations-10-synthesis.md
- Clean Up 10 Empty Investigation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-18-inv-clean-up-10-empty-investigation.md
- Synthesize Session Investigations (10)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-session-investigations-10-synthesis.md
- Synthesize Status Investigations (12)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-status-investigations-12-synthesis.md

### Failed Attempts (DO NOT repeat)
- Researching Foreman, Overmind, and Nx for polyrepo server management

### Open Questions
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.






📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — the workspace often has unrelated changes (.autorebuild.lock, .beads/, build/).
Stage ONLY the specific files you created or modified for your task, by name.


1. **COMMIT YOUR WORK:**
   ```bash
   git add <files you changed>
   git commit -m "feat: [brief description of changes]"
   ```
2. Run: `/exit` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.


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


2. **SET UP probe file:** This is confirmatory work against an existing model.
   - Model content was injected in PRIOR KNOWLEDGE section above
   - Create probe file in model's probes/ directory
   - Use probe template structure: Question, What I Tested, What I Observed, Model Impact
   - Your probe should confirm, contradict, or extend the model's claims



3. **UPDATE probe file** as you work:
   - Question: What model claim are you testing?
   - What I Tested: Actual command/code run (not just code review)
   - What I Observed: Actual output/behavior
   - Model Impact: Confirms/contradicts/extends which invariant

4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]


6. ⚡ SYNTHESIS.md is NOT required (light tier spawn).


STATUS UPDATES:
Update Status: field in your probe file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to probe file
- Add '**Status:** QUESTION - [question]' when needing input




## SKILL GUIDANCE (investigation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 7a8c26d4a43b -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-18 14:13:33 -->

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
bd comment orch-go-untracked-1771631873 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-untracked-1771631873 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-untracked-1771631873 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-untracked-1771631873 "Phase: BLOCKED - Need clarification on API contract"

# Report questions
bd comment orch-go-untracked-1771631873 "Phase: QUESTION - Should we use JWT or session-based auth?"
```

**When to report:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Additional context:**
Use `bd comment` for additional context, findings, or updates:
```bash
bd comment orch-go-untracked-1771631873 "Found performance bottleneck in database query"
bd comment orch-go-untracked-1771631873 "investigation_path: .kb/investigations/2026-02-11-perf-issue.md"
```

**Test Evidence Requirement:**
When reporting Phase: Complete, include test results in the summary:
- Example: `bd comment orch-go-untracked-1771631873 "Phase: Complete - Tests: go test ./... - 47 passed, 0 failed (2.3s)"`
- Example: `bd comment orch-go-untracked-1771631873 "Phase: Complete - Tests: npm test - 23 specs, 0 failures"`
- Example: `bd comment orch-go-untracked-1771631873 "Phase: Complete - Tests: make test - PASS (coverage: 78%)"`

**Why:** `orch complete` validates test evidence in phase comments. Vague claims like "all tests pass" trigger manual verification.

**Never run `bd close`** - Only the orchestrator closes issues via `orch complete`.
- Workers report `Phase: Complete`, orchestrator verifies and closes
- Running `bd close` bypasses verification and breaks tracking

---

## Phase Reporting

**First 3 Actions (Critical):**
Within your first 3 tool calls, you MUST:
1. Report via `bd comment orch-go-untracked-1771631873 "Phase: Planning - [brief description]"`
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

---

## Session Complete Protocol

**When your work is done (all deliverables ready), complete in this EXACT order:**



1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment orch-go-untracked-1771631873 "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
4. Commit any remaining changes (including `VERIFICATION_SPEC.yaml`)
5. Run: `/exit` to close the agent session

**Light Tier:** SYNTHESIS.md is NOT required for this spawn.


**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

**Verification gates (orchestrator-run):**
After you report Phase: Complete, the orchestrator runs `orch complete` with two human gates:

- Gate 1 (explain-back): explain what was built and why.
- Gate 2 (behavioral, Tier 1 only): confirm behavior was verified.
  Make your Phase summary and VERIFICATION_SPEC.yaml clear enough to support both gates.

---

---
name: investigation
skill-type: procedure
description: Record what you tested and observed; default to model-scoped probes when injected model claims are present, otherwise run a full investigation.
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 9c3e89d2e927 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/src/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-18 09:18:39 -->

<!-- SKILL-CONSTRAINTS -->
<!-- optional: .kb/models/*/probes/{date}-*.md | Model probe output when injected model content is present -->
<!-- optional: .kb/investigations/{date}-inv-*.md | Investigation output when no injected model content is present -->
<!-- /SKILL-CONSTRAINTS -->
## Summary

**Purpose:** Answer a question by testing, not by reasoning.

---

# Investigation Skill

**Purpose:** Answer a question by testing, not by reasoning.

## Artifact Mode Selection (Probe Default)

**Before creating any artifact, read SPAWN_CONTEXT.md and detect mode:**

1. Find the `### Models (synthesized understanding)` section
2. Check for injected model-claim markers in model entries:
   - `- Summary:`
   - `- Critical Invariants:` or `- Constraints:`
   - `- Why This Fails:` or `- Failure Modes:`

### If markers are present -> Probe Mode (default)

- Create a probe (not a full investigation)
- Use template: `.orch/templates/PROBE.md`
- Write to: `.kb/models/{model-name}/probes/{date}-{slug}.md`
- Required sections:
  - `Question`
  - `What I Tested`
  - `What I Observed`
  - `Model Impact`

### If markers are absent -> Investigation Mode

- Follow standard investigation workflow
- Write to: `.kb/investigations/{date}-inv-{slug}.md`

## The One Rule

**You cannot conclude without testing.**

If you didn't run a test, you don't get to fill the Conclusion section.

## Evidence Hierarchy

**Artifacts are claims, not evidence.**

- **Primary** (authoritative): Actual code, test output, observed behavior → This IS the evidence
- **Secondary** (claims to verify): Workspaces, investigations, decisions → Hypotheses to test

When an artifact says "X is not implemented," that's a hypothesis—search the codebase before concluding.

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for evidence hierarchy examples and common failures.

## Prior Work Acknowledgment

**This section applies in Investigation Mode only (no injected model-claim markers).**

Before creating your investigation file, review prior work from SPAWN_CONTEXT.

1. **Check SPAWN_CONTEXT** for "Related Investigations" section
2. **If prior work exists:**
   - Note which investigations are relevant to your question
   - Plan to verify cited claims AS YOU ENCOUNTER THEM during investigation
3. **If no prior work:** Note "N/A - novel investigation"

**This is acknowledgment, not exhaustive verification.** You verify claims naturally as you encounter them during your investigation, not all upfront.

---

## Workflow

1. **Detect mode from SPAWN_CONTEXT.md** using model-claim markers in `### Models (synthesized understanding)`:
   - `- Summary:`
   - `- Critical Invariants:` or `- Constraints:`
   - `- Why This Fails:` or `- Failure Modes:`
2. **If markers found -> Probe Mode (default)**
   - Pick the most relevant model from the injected models section
   - Create: `.kb/models/{model-name}/probes/{date}-{slug}.md`
   - Use `.orch/templates/PROBE.md`
   - Fill all required sections: Question, What I Tested, What I Observed, Model Impact
3. **If markers not found -> Investigation Mode**
   - Acknowledge prior work: SPAWN_CONTEXT -> "Related Investigations"
   - Create file: `kb create investigation {slug}`
   - IMMEDIATE CHECKPOINT: Fill Question, add Prior-Work table, add Finding 1 ("Starting approach"), commit immediately
4. **TEST-FIRST GATE:** "What's the simplest test I can run right now?" (60-second rule)
5. Try things, observe what happens (add findings/probe evidence progressively)
6. Verify relevant claims as encountered
7. Run a real test to validate your hypothesis
8. Fill conclusion/model impact based on observed evidence only
9. Final commit

**Why checkpoint immediately?** Agents can die from API errors, context limits, or crashes. Without a checkpoint, no record of what was attempted.

**Reference:** See `~/.claude/skills/worker/investigation/reference/error-recovery.md` for handling fatal errors during exploration.

## D.E.K.N. Summary

**D.E.K.N. applies to Investigation Mode.** Probes use the probe template's required sections instead.

- **Delta:** What was discovered/answered
- **Evidence:** Primary evidence supporting conclusion
- **Knowledge:** What was learned (insights, constraints)
- **Next:** Recommended action

**Fill D.E.K.N. at the END, before marking Complete.**

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for D.E.K.N. examples.

## Template

Choose template based on SPAWN_CONTEXT mode detection.

### Probe Mode (default when injected model markers are present)

Use `.orch/templates/PROBE.md`. Write to `.kb/models/{model-name}/probes/{date}-{slug}.md`.

Required sections:

- **Question**
- **What I Tested** (actual command/code executed)
- **What I Observed** (concrete output)
- **Model Impact** (confirms | contradicts | extends)

### Investigation Mode (fallback when model markers are absent)

Use `kb create investigation {slug}`. Required sections:

- **D.E.K.N. Summary** (Delta, Evidence, Knowledge, Next)
- **Prior Work** table (entries OR "N/A - novel investigation")
- **Question** and **Status**
- **Findings** (add progressively)
- **Test performed** (not "reviewed code" - actual test)
- **Conclusion** (only if you tested)

### Prior-Work Table Structure

```markdown
## Prior Work

| Investigation                          | Relationship | Verified | Conflicts |
| -------------------------------------- | ------------ | -------- | --------- |
| .kb/investigations/2026-01-26-inv-X.md | extends      | pending  | -         |
| N/A - novel investigation              | -            | -        | -         |
```

**Relationship vocabulary:**

- **Extends:** Adds to prior findings (most common)
- **Confirms:** Validates prior hypothesis with new evidence
- **Contradicts:** Disproves or refines prior conclusion
- **Deepens:** Explores same question at greater depth

**Verified column:** Start with "pending", update to "yes" when you test a cited claim during investigation.

**Conflicts column:** Document contradictions found during verification.

**Reference:** See `~/.claude/skills/worker/investigation/reference/template.md` for full structure and `reference/examples.md` for common failures.

## When Not to Use

- **Fixing bugs** → Use `systematic-debugging`
- **Trivial questions** → Just answer them
- **Documentation** → Use `capture-knowledge`

## Prior Work (Template Independence)

**Applies to Investigation Mode only.** In Probe Mode, use the probe template and skip the Prior-Work table.

**Why this matters:** 701 existing investigations lack the Prior-Work table. The skill must handle both old and new investigations gracefully.

### Creating New Investigations

When creating a new investigation via `kb create investigation`, your file MUST include a Prior-Work table:

```markdown
## Prior Work

| Investigation   | Relationship | Verified | Conflicts     |
| --------------- | ------------ | -------- | ------------- |
| [path or "N/A"] | [type]       | [yes/no] | [description] |

**Relationship types:** extends, confirms, contradicts, deepens
```

**If no prior investigations exist:** Use explicit acknowledgment:

```markdown
## Prior Work

| Investigation             | Relationship | Verified | Conflicts |
| ------------------------- | ------------ | -------- | --------- |
| N/A - novel investigation | -            | -        | -         |
```

### Extending Old Investigations (Graceful Degradation)

If SPAWN_CONTEXT references prior investigations that lack Prior-Work tables:

1. **Do NOT backfill old investigations** - Never modify their structure
2. **Create a new investigation file** with full Prior-Work table
3. **Reference the old investigation** in your Prior-Work table:

```markdown
## Prior Work

| Investigation                                  | Relationship | Verified | Conflicts                                  |
| ---------------------------------------------- | ------------ | -------- | ------------------------------------------ |
| .kb/investigations/2025-11-15-inv-old-topic.md | extends      | yes      | [describe any conflicts with prior claims] |

---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — stage ONLY your task files by name.



1. **COMMIT YOUR WORK:** `git add <files you changed> && git commit -m "feat: [description]"`
2. `/exit`

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.



⛔ **NEVER run `git push`** - Workers commit locally only.
⚠️ Your work is NOT complete until Phase: Complete is reported (or /exit for --no-track).
