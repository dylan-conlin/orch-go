TASK: Audit fix: commits from entropy-spiral-feb2026 — of the ~161 fix commits, how many address bugs still present on current master? Categorize each as: (1) still-relevant (bug exists on master), (2) irrelevant (code diverged or feature doesn't exist), (3) already-fixed (same fix exists on master). FRAME: Dylan is hitting bugs that were already fixed on the spiral branch but lost in the rollback. We need to know the scale of the problem before deciding recovery strategy.


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "audit commits entropy"

### Constraints (MUST respect)
- Build artifacts should not be committed at project root
  - Reason: Development executables (orch-*, test-*) clutter the directory and waste space (~75MB+)

### Prior Decisions
- Implement 3-tier guardrail system: preflight checks, completion gates, daily reconciliation
  - Reason: Post-mortem showed 115 commits in 24h with 7 missing guardrails enabling runaway automation
- Multi-agent synthesis relies on workspace isolation + SYNTHESIS.md + orch review
  - Reason: 100 commits, 52 synthesis files, 0 conflicts validates current architecture
- Cross-project epics use Option A: epic in primary repo, ad-hoc spawns with --no-track in secondary repos, manual bd close with commit refs
  - Reason: Only working pattern today. Beads multi-repo hydration is read-only aggregation, bd repo commands are buggy.
- When spawned for cross-repo work, verify work completion status before starting
  - Reason: Task orch-go-oo1f: spawned in orch-go for work in orch-knowledge. Template was already retired (commit 7430185) before agent fully engaged. Quick verification could have saved agent context.
- kb-cli investigationTemplate is synced with ~/.kb/templates/INVESTIGATION.md
  - Reason: Updated hardcoded fallback const to match canonical template (234 lines with D.E.K.N., Lineage, Implementation Recommendations). Commit 032ea18 in kb-cli.
- Beads multi-repo hydration works correctly in v0.33.2
  - Reason: Config disconnect bug fixed in commit 634c0b93. Prior kn entry about 'buggy v0.29.0' is superseded.
- bd multi-repo config is YAML-only, database config is legacy
  - Reason: Fix commit 634c0b93 moved repos config from database to YAML. GetMultiRepoConfig() reads YAML only. Stale binary causes silent failure.
- Use phased adversarial verification over post-completion review
  - Reason: Post-completion review doesn't prevent validation loops (Budden case), only detects after agent committed to flawed conclusions
- After agents commit Go changes, orchestrator should auto-rebuild and restart affected services
  - Reason: Manual rebuild/restart is friction. Pattern: detect changed files (cmd/orch/, pkg/) → make install → restart orch serve if running. Could be hook or part of orch complete flow.
- After agents commit Go changes, orchestrator should auto-rebuild and restart affected services
  - Reason: Manual rebuild/restart is friction. Pattern: detect changed files (cmd/orch/, pkg/) -> make install -> restart orch serve if running. Could be hook or part of orch complete flow.

### Models (synthesized understanding)
- Probe: Verifiability-First Closure Audit — Did Claimed Work Actually Land?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-15-verifiability-first-closure-audit.md
  - Recent Probes:
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
    - 2026-02-16-daemon-completion-loop-bypasses-verification-gates
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-16-daemon-completion-loop-bypasses-verification-gates.md
    - 2026-02-16-probe-three-code-paths-verification-state
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md
    - 2026-02-15-verificationtracker-backlog-count-mismatch
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-15-verificationtracker-backlog-count-mismatch.md
    - 2026-02-15-daemon-verification-tracker-checkpoint-enforcement
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-15-daemon-verification-tracker-checkpoint-enforcement.md
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
- Probe: Daemon Warn-and-Continue Anti-Pattern Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-15-daemon-warn-continue-anti-pattern-audit.md
  - Recent Probes:
    - 2026-02-17-daemon-test-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-test-fail-fast-fix.md
    - 2026-02-17-daemon-rollback-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-rollback-fail-fast-fix.md
    - 2026-02-17-daemon-completion-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-completion-fail-fast-fix.md
    - 2026-02-17-extraction-gate-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-extraction-gate-fail-fast-fix.md
    - 2026-02-17-daemon-epic-expansion-fail-fast
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-epic-expansion-fail-fast.md
- Probe: Daemon Config Construction Divergence Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-15-daemon-config-construction-divergence-audit.md
  - Recent Probes:
    - 2026-02-17-daemon-test-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-test-fail-fast-fix.md
    - 2026-02-17-daemon-rollback-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-rollback-fail-fast-fix.md
    - 2026-02-17-daemon-completion-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-completion-fail-fast-fix.md
    - 2026-02-17-extraction-gate-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-extraction-gate-fail-fast-fix.md
    - 2026-02-17-daemon-epic-expansion-fail-fast
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-epic-expansion-fail-fast.md
- OpenCode Fork
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork.md
  - Summary:
    Dylan owns a fork of [sst/opencode](https://github.com/sst/opencode) at `~/Documents/personal/opencode`. The fork is 16 custom commits ahead of upstream (linear forward — all upstream changes included). Custom changes focus on **memory management** (LRU/TTL instance eviction preventing 8.4GB unbounded growth), **SSE cleanup** (idempotent teardown preventing leaked connections), **OAuth stealth mode** (Claude Max access), **ORCH_WORKER header forwarding** (worker session detection), **session metadata** (extensible key-value storage), and **session TTL** (auto-expiry with periodic cleanup). The fork is a TypeScript monorepo (Bun runtime, Hono HTTP framework) with sessions stored in SQLite database (~/.local/share/opencode/opencode.db) using Drizzle ORM with migration-based schema management. Session status (idle/busy/retry) is tracked in-memory only, lost on server restart. A `GET /session/status` endpoint already exists for querying session state.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-18-probe-state-create-rejected-promise-cache
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-state-create-rejected-promise-cache.md
    - 2026-02-18-probe-headless-codex-provider-init
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md
- Probe: Vector #2 — cleanUntrackedDiskSessions Removal and Current State
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md
  - Recent Probes:
    - 2026-02-14-probe-vector2-cleanuntrackedsessions-removal
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md
    - 2026-02-14-probe-vector7-sqlite-migration-json-fallback
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md
- Probe: Control Plane Circuit Breaker Heuristic Calibration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-14-control-plane-heuristic-calibration.md
  - Recent Probes:
    - 2026-02-17-daemon-test-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-test-fail-fast-fix.md
    - 2026-02-17-daemon-rollback-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-rollback-fail-fast-fix.md
    - 2026-02-17-daemon-completion-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-completion-fail-fast-fix.md
    - 2026-02-17-extraction-gate-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-extraction-gate-fail-fast-fix.md
    - 2026-02-17-daemon-epic-expansion-fail-fast
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-epic-expansion-fail-fast.md
- Probe: bd sync pre-commit hook deadlock (second deadlock path)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-17-bd-sync-precommit-hook-deadlock.md
  - Recent Probes:
    - 2026-02-17-bd-sync-precommit-hook-deadlock
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-17-bd-sync-precommit-hook-deadlock.md
    - 2026-02-09-bd-sync-safe-timeout-retry-for-hash-mismatch
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-09-bd-sync-safe-timeout-retry-for-hash-mismatch.md
    - 2026-02-09-bd-sync-safe-post-sync-readiness-check
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-09-bd-sync-safe-post-sync-readiness-check.md
    - 2026-02-08-synthesis-dedup-parse-error-fail-closed
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-08-synthesis-dedup-parse-error-fail-closed.md
- Coaching Plugin
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-14 (worker detection fix verified).
    Changed files: plugins/coaching.ts, cmd/orch/serve_coaching.go, pkg/opencode/client.go, web/src/routes/+page.svelte.
    Deleted files: ~/.orch/coaching-metrics.jsonl.
    Verify model claims about these files against current code.
  - Summary:
    The Coaching Plugin is an OpenCode plugin that provides real-time behavioral feedback to orchestrators and workers through tool usage pattern detection. It implements the "Pain as Signal" architectural pattern: agents should feel friction in real-time rather than learning about it post-hoc.
    
    The plugin hooks `tool.execute.after` to observe tool usage patterns (it cannot see LLM response text—fundamental constraint), detects 8 behavioral patterns using behavioral proxies (action ratio, analysis paralysis, frame collapse, etc.), and injects coaching messages via `client.session.prompt({ noReply: true })`. Metrics persist to `~/.orch/coaching-metrics.jsonl` and are exposed via `/api/coaching` for dashboard visualization.
    
    **Current status (Feb 2026):** Both orchestrator and worker coaching operational. Orchestrator coaching: 1000+ metrics collected. Worker health tracking: verified working Feb 14 — stress test (50+ tool calls) emitted `context_usage` worker metric with zero orchestrator metric leakage. Fix required two opencode fork commits (459a1bfba, 0922edfe7) to: (1) read `x-opencode-env-ORCH_WORKER` header and set `session.metadata.role='worker'`, (2) pass `session.metadata` through `tool.execute.after` plugin hooks. **Note:** Worker detection only works for headless (OpenCode HTTP API) spawns — Claude CLI/tmux spawns bypass the HTTP session creation endpoint and don't get metadata set.
    
    ---
  - Critical Invariants:
    1. **Plugins cannot see LLM response text** - Only tool calls visible, not free-text responses. Fundamental constraint, not fixable.
    2. **Behavioral proxies are the only detection mechanism** - All pattern detection uses tool usage as proxy signals.
    3. **Metrics persist, session state doesn't** - JSONL file survives restarts, in-memory Map is ephemeral.
    4. **Observation coupled to intervention** - Injection only fires from `flushMetrics` within `tool.execute.after` hook.
    5. **Worker detection caching is one-way** - Only cache `true` results (confirmed worker), never cache `false`.
    6. **Two injection mechanisms serve different purposes** - `config.instructions` adds file references at config time (static context like skills), `client.session.prompt(noReply: true)` injects content at runtime (immediate coaching).
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Worker Detection Caching Bug (Jan 10-17)
    
    **Symptom:** Zero worker health metrics in production despite implemented code
    
    **Root cause:** `detectWorkerSession()` cached both `true` AND `false` results, permanently misclassifying workers if ANY tool call happened before a worker-identifying signal
    
    **Why it happens:**
    1. Worker session starts, first tool = `glob` (no detection signal)
    2. `isWorker = false` → cached → function returns `false` forever
    3. Second tool = `read SPAWN_CONTEXT.md` → cached result returned, detection skipped
    4. Worker treated as orchestrator for entire session
    
    **Cascade:**
    ```
    First non-matching tool call → cache false → subsequent detection signals ignored → worker health code never runs
    ```
    
    **Fix (Jan 17):** Only cache `true` results (confirmed worker), never cache `false`. Allow re-evaluation on each tool call until worker confirmed.
    
    **Pattern established:** "Never cache negative results in per-session detection"
    
    ---
    
    ### Failure Mode 2: Invalid Detection Signal (Bash workdir)
    
    **Symptom:** Detection code exists but never fires
    
    **Root cause:** Detection checked for `args.workdir` on bash tool, but bash tool has no `workdir` argument
    
    **Why it happens:**
    - Bash tool args are: `command`, `timeout`, `dangerouslyDisableSandbox`, `run_in_background`
    - No `workdir` argument exists
    - Detection signal `if (tool === "bash" && args?.workdir)` never matches
    
    **Fix (Jan 17):** Removed broken detection signal, restored file-path detection for any `.orch/workspace/` path
    
    ---
    
    ### Failure Mode 3: Observation Coupled to Intervention (Restart Brittleness)
    
    **Symptom:** Coaching messages stop after OpenCode server restart, even though metrics show problems
    
    **Root cause:** Injection is implemented as side effect of metric collection, not as separate concern that can operate independently
    
    **Why it happens:**
    - Metrics are **persistent** (JSONL file, survive restart)
    - Session state is **in-memory** (Map, lost on restart)
    - Injection logic **coupled to metric collection** (only happens via `flushMetrics`)
    - After restart: metrics file shows "poor" status, but no session state exists, so injection never fires
    
    **Cascade:**
    ```
    Server restart → session state lost → flushMetrics not called → injection doesn't fire → coaching stops
    ```
    
    **Architectural fix (not yet implemented):** Separate injection into independent daemon that reads metrics from JSONL and injects via OpenCode API, completely decoupled
    ... [truncated]
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-14-metrics-redesign-architecture-validation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-metrics-redesign-architecture-validation.md
    - 2026-02-14-worker-detection-stress-test
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-stress-test.md
    - 2026-02-14-worker-detection-header-implementation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-header-implementation.md
- Probe: Vector #7 — SQLite Migration Legacy JSON Storage Fallback
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md
  - Recent Probes:
    - 2026-02-14-probe-vector2-cleanuntrackedsessions-removal
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md
    - 2026-02-14-probe-vector7-sqlite-migration-json-fallback
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md

### Guides (procedural knowledge)
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Worker Patterns Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/worker-patterns.md
- Completion Gates
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion-gates.md
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md
- Agent Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/agent-lifecycle.md
- orch CLI Reference
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/cli.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Decision Authority Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/decision-authority.md
- OpenCode Integration Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode.md

### Related Investigations
- Entropy Spiral Postmortem
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-12-inv-entropy-spiral-postmortem.md
- Analyze Commits Between fb0af37f (Dec 27) and 344da9a7 (Jan 2)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-analyze-commits-between-fb0af37f-dec.md
- Entropy Spiral Deep Analysis
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-entropy-spiral-deep-analysis.md
- Entropy Spiral Recovery Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-13-inv-entropy-spiral-recovery-audit.md
- Fix Probe Commit Pipeline Probes
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-fix-probe-commit-pipeline-probes.md
- Audit 800+ Uncategorized Investigations - Archive vs Cluster Recommendations
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-15-audit-uncategorized-investigations-855-archive-vs-cluster.md
- Audit Visual Go Same Since
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-audit-visual-go-same-since.md
- Comprehensive Orch-Go Codebase Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-audit-comprehensive-orch-go-codebase-audit.md
- Recover Dashboard Web UI from Entropy Branch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-13-inv-recover-dashboard-web-ui-entropy.md
- Addendum to Ecosystem Audit - OpenCode's Role
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-addendum-ecosystem-audit-opencode.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.






🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-1032 "Phase: Planning - [brief description]"`
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
2. Run: `bd comment orch-go-1032 "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.



VERIFICATION REQUIREMENTS (ORCH COMPLETE):
Your work is verified in two human gates before closing:
- Gate 1 (explain-back): orchestrator must explain what was built and why.
- Gate 2 (behavioral, Tier 1 only): orchestrator confirms behavior is verified.
Provide clear Phase: Complete summary and VERIFICATION_SPEC.yaml evidence to support both gates.


CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go/

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
1. Surface it first: `bd comment orch-go-1032 "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go/)


2. **SET UP probe file:** This is confirmatory work against an existing model.
   - Model content was injected in PRIOR KNOWLEDGE section above
   - Create probe file in model's probes/ directory
   - Use probe template structure: Question, What I Tested, What I Observed, Model Impact
   - Your probe should confirm, contradict, or extend the model's claims

   - **IMPORTANT:** After creating probe file, report the path via:
     `bd comment orch-go-1032 "probe_path: /path/to/probe.md"`



3. **UPDATE probe file** as you work:
   - Question: What model claim are you testing?
   - What I Tested: Actual command/code run (not just code review)
   - What I Observed: Actual output/behavior
   - Model Impact: Confirms/contradicts/extends which invariant

4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]


6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go//.orch/workspace/og-inv-audit-fix-commits-18feb-0583/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go//.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your probe file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to probe file
- Add '**Status:** QUESTION - [question]' when needing input



## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-1032**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-1032 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-1032 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-1032 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-1032 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-1032 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-1032`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (investigation)

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
```

**Verification happens naturally:** As you explore, you'll encounter claims from prior investigations. Verify them against primary sources (code, test output) when relevant to your question - not all upfront.

### Key Rules

| Situation                       | Action                                                    |
| ------------------------------- | --------------------------------------------------------- |
| New investigation (no priors)   | Add Prior-Work table with "N/A - novel investigation"     |
| New investigation (has priors)  | Add Prior-Work table, cite prior investigations           |
| Extending old investigation     | Create NEW file with Prior-Work table referencing the old |
| Old investigation without table | Leave it alone, don't modify                              |

**Reference:** See `~/.claude/skills/worker/investigation/reference/prior-work-examples.md` for relationship type guidance.


## Self-Review (Mandatory)

Before completing, verify quality for the mode you used:

### Probe Mode Checklist

- [ ] File path is `.kb/models/{model-name}/probes/{date}-{slug}.md`
- [ ] Used `.orch/templates/PROBE.md`
- [ ] All 4 sections present: Question, What I Tested, What I Observed, Model Impact
- [ ] `What I Tested` contains executed command/code (not code review)
- [ ] `What I Observed` includes concrete output
- [ ] Model Impact verdict is explicit: confirms | contradicts | extends

### Investigation Mode Checklist

- [ ] **Prior-Work acknowledged** - Table present with entries OR explicit "N/A - novel investigation"
- [ ] **Cited claims verified** - Any claim referenced from prior work was tested (Verified = "yes")
- [ ] **Test is real** - Ran actual command/code, not just "reviewed"
- [ ] **Evidence concrete** - Specific outputs, not "it seems to work"
- [ ] **Conclusion factual** - Based on observed results, not inference
- [ ] **No speculation** - Removed "probably", "likely", "should" from conclusion
- [ ] **Question answered** - Investigation addresses the original question
- [ ] **File complete** - All sections filled (not "N/A" or "None")
- [ ] **D.E.K.N. filled** - Replaced placeholders in Summary section
- [ ] **Scope verified** - Ran `rg` to find all occurrences before concluding
- [ ] **NOT DONE claims verified** - If claiming incomplete, searched actual code

### Prior-Work Verification

**Applies to Investigation Mode only.**

**Gate:** Your investigation file MUST contain a Prior-Work table.

| Situation                                   | Required Action                                                  |
| ------------------------------------------- | ---------------------------------------------------------------- |
| SPAWN_CONTEXT has "Related Investigations"  | List relevant ones in table, verify claims as you encounter them |
| SPAWN_CONTEXT has no related investigations | Add single row: "N/A - novel investigation"                      |
| You cited prior work without verifying      | Update Verified column, document conflicts found                 |

**This is passable:** You only need to verify claims you actually referenced during your investigation. You do NOT need to exhaustively verify all prior work upfront.

### Discovered Work

If you found bugs, tech debt, or enhancement ideas during investigation:

- Create beads issues: `bd create "description" --type bug|task|feature`
- Apply label: `bd label <id> triage:ready` or `triage:review`

**If no discoveries:** Note "No discovered work items" in completion comment.

**Reference:** See `~/.claude/skills/worker/investigation/reference/self-review-guide.md` for scope verification examples and discovered work procedures.

**Only proceed to commit after self-review passes.**


---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**
- `kb quick decide "X" --reason "Y"` (made a choice)
- `kb quick tried "X" --failed "Y"` (something failed)
- `kb quick constrain "X" --reason "Y"` (found a constraint)
- `kb quick question "X"` (open question)

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."

**Reference:** See `~/.claude/skills/worker/investigation/reference/leave-it-better.md` for command examples.

---

## Completion

1. Self-review passed
2. **Probe vs Investigation requirements met:**
   - Probe Mode: probe file exists in `.kb/models/{model-name}/probes/` with all 4 mandatory sections
   - Investigation Mode: Prior-Work acknowledged and D.E.K.N. summary filled
3. Leave it Better completed (or noted why N/A)
4. Report: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"` (FIRST - before commit)
5. Commit: `git add && git commit`
6. Exit: `/exit`

**Why report before commit?** If agent dies after commit but before reporting, orchestrator cannot detect completion.

---

**Remember:** Test before concluding.






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go//CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.



1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-1032 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.
