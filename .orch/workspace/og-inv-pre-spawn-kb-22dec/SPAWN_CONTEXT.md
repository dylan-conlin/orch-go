TASK: pre-spawn kb context check now surfaces too much irrelevant cross-repo content (price-watch, dotfiles, etc). investigate filtering strategies: query relevance scoring, repo allowlist, category prioritization. recommend approach that preserves cross-repo signal while reducing noise

## PRIOR KNOWLEDGE (from kb context)

**Query:** "pre"

### Constraints (MUST respect)
- [orch-knowledge] Beads tracks agent lifecycle, not WORKSPACE.md
  - Reason: WORKSPACE.md deprecated for lifecycle tracking - beads is now the work tracker
- [orch-cli] exclude_files parameter in git validation doesn't prevent all parallel completion races
  - Reason: Code at git_utils.py:268-292 implements exclusion filtering but TOCTOU race still occurs when multiple agents complete within seconds. Validation runs before beads sync completes, creating race window where one agent's beads changes appear as uncommitted in another agent's validation.
- [dotfiles] tmux-fingers must only activate on trigger key (prefix+F), never override fundamental bindings
  - Reason: Intercepting all input locks terminal with no escape hatch
- [dotfiles] macOS date -j needs TZ=UTC for UTC timestamps
  - Reason: Without TZ=UTC, date -j interprets ISO timestamps as local time, causing incorrect calculations
- [kn] Decisions require reason with min 20 chars
  - Reason: Friction prevents low-value entries, forces intentionality
- [orch-go] Agents must not spawn more than 3 iterations without human review
  - Reason: Prevents runaway iteration loops like 12 tmux fallback tests in 9 minutes
- [orch-go] orch init must be idempotent - safe to run multiple times
  - Reason: Prevents accidental overwrites and enables 'run init to update' pattern
- [orch-go] Beads cross-repo contamination can create orphaned FK references
  - Reason: bd-* prefixed dependencies were found in orch-go database from separate beads repo
- [orch-go] Template ownership: kb-cli owns artifact templates (investigation/decision/guide), orch-go owns spawn-time templates (SYNTHESIS/SPAWN_CONTEXT/FAILURE_REPORT)
  - Reason: Prevents drift by establishing clear domains. See .kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md

### Prior Decisions
- [orch-knowledge] kn integrates via: (1) no CLI wrapping - call kn directly, (2) smart auto-inject in orch spawn, (3) per-project .kn/ only
  - Reason: Keep tools focused (Unix philosophy), auto-inject prevents missing critical knowledge, per-project is simpler and most knowledge is project-specific
- [orch-knowledge] Unified spawnable sessions: single PreToolUse gate calibrated by skill type (orchestrator/architect: hard, investigation: soft, feature-impl: none)
  - Reason: Synthesizes 4 converged investigations; respects Gate Over Remind principle; knowledge density varies predictably by session type
- [orch-knowledge] PreToolUse hook gates Phase: Complete on kn entry existence
  - Reason: Implements Gate Over Remind principle. Hook checks CLAUDE_CONTEXT=worker, CLAUDE_DELIVERABLES contains investigation, and queries .kn/entries.jsonl for recent entries. Escape hatch via SKIP_KN_GATE=1.
- [orch-knowledge] Knowledge capture gates are calibrated by skill type
  - Reason: architect/orchestrator get hard gates (must have kn entries), investigation gets soft gate (warning), feature-impl/etc get no gate. Implemented in pre-complete-kn-check.py hook.
- [orch-knowledge] Hard gate validation successful - PreToolUse hook blocks Phase: Complete without kn entries
  - Reason: Tested by attempting to write Phase: Complete to WORKSPACE.md without any prior kn entries. Gate fired correctly, blocked the operation, and provided helpful guidance.
- [orch-knowledge] Soft gate validation successful - PreToolUse hook warns but allows completion for investigation skill
  - Reason: Tested by writing Status: Complete to investigation file without kn entries. Hook printed warning to stderr but allowed operation. Contrast with hard gate (architect/orchestrator) which blocks.
- [orch-knowledge] Spawnable orchestrator knowledge gate validated - PreToolUse hook correctly blocks bd close for orchestrator skill without kn entries
  - Reason: Tested by spawning orchestrator skill as worker, attempting bd close. Hook detected skill type from SPAWN_CONTEXT.md, verified no kn entries, and blocked with helpful guidance message.
- [orch-knowledge] AI-first CLI ≠ JSON-first - LLM agents read prose well, JSON is for scripts/pipelines
  - Reason: Observed in real orchestration sessions: Claude interprets human-readable output and follows actionable error messages naturally
- [orch-knowledge] Knowledge archives grow through aggressive externalization, not planning
  - Reason: Analysis of .kb shows 490 artifacts in 6 weeks - growth is emergent from use, not designed upfront. Investigations beget decisions which beget more investigations. This organic growth pattern produces better coverage than pre-planned taxonomy.
- [orch-knowledge] Preventive validation (bd create warning) beats corrective audit skill for beads issue quality
  - Reason: Investigation found 79% of issues meet standards; 21% gap comes from hasty creation during intense sessions, not systematic failure. Only 4% truly not spawn-ready. Audit skill treats symptoms; creation-time validation addresses root cause.
- [orch-knowledge] Cross-repo beads duplicates should be closed in favor of the repo where implementation belongs
  - Reason: Found ok-wol7 (orch-knowledge with target:orch-cli) duplicating orch-cli-cux. Target labels don't prevent duplication - better to have single issue in correct repo.
- [orch-cli] Per-tool JSONL error logs with unified reader is preferred approach for cross-CLI error aggregation
  - Reason: Maintains repo independence, allows independent error taxonomy evolution, simpler than shared file (no concurrent write coordination). Option B evaluated against Option A (shared file) and Option C (wrapper capture).
- [orch-cli] Hybrid SDK integration approach for orch-cli
  - Reason: SDK provides programmatic control (hooks, session persistence, budget limits) while tmux provides visual multiplexing. Hybrid approach preserves Dylan's workflow while enabling new features. Validated SDK works with native OAuth.
- [beads-ui-svelte] dual-logging architecture separates event logs from error analytics
  - Reason: OrchLogger handles comprehensive event logging for debugging; ErrorLogger focuses on error pattern detection. Separation prevents pollution and enables specialized analytics.
- [price-watch] AI data view should use hierarchical JSON with statistical context
  - Reason: AI needs structured data with pre-calculated stats, volume weights, and time series to identify patterns Jim might miss - flat structures lose aggregation context and require AI to rebuild baselines
- [price-watch] Keep browser automation for SendCutSend scraper
  - Reason: Direct API possible but browser automation preferable: (1) async file processing requires polling, (2) reCAPTCHA/Cloudflare protections benefit from browser context, (3) current scraper works reliably, (4) marginal ~30sec savings doesn't justify rework risk
- [price-watch] Standalone SvelteKit for comparison view rewrite
  - Reason: Better dev experience (HMR), server routes eliminate CORS complexity, preserves SvelteKit full capabilities
- [orch-go] Registry respawn workflow uses slot reuse pattern
  - Reason: Preserves single-entry-per-ID invariant while enabling abandon→respawn lifecycle
- [orch-go] Registry updates must happen before beads close in orch complete
  - Reason: Prevents inconsistent state where beads shows closed but registry shows active
- [orch-go] Use build/orch for serve daemon
  - Reason: Prevents SIGKILL during make install
- [orch-go] Implement 3-tier guardrail system: preflight checks, completion gates, daily reconciliation
  - Reason: Post-mortem showed 115 commits in 24h with 7 missing guardrails enabling runaway automation
- [orch-go] Port allocation should use ranges by purpose (vite: 5173-5199, api: 3333-3399)
  - Reason: Prevents conflicts and makes purpose clear from port number
- [kb-cli] kb as the Knowledge Hub (Context + Link + Promote)
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/decisions/2025-12-13-kb-as-knowledge-hub.md
- [orch-knowledge] Feature Coordination Skill Creation
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-01-14-feature-coordination-skill-creation.md
- [orch-knowledge] Systematic Memory File Management for Orchestrator
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-03-how-should-the-orchestrator-systematically.md
- [orch-knowledge] Passive Agent Monitoring Implementation
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-03-how-should-we-implement-passive.md
- [orch-knowledge] Architect Agent Lifecycle: Continuous vs On-Demand
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-04-architect-lifecycle-continuous-vs-ondemand.md
- [orch-knowledge] Orchestrator Role Boundaries and Overload Management
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-04-orchestrator-role-boundaries.md
- [orch-knowledge] Orchestrator Work Prioritization Strategy (ROADMAP vs TODO.org)
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-04-what-should-the-orchestrators-relationship.md
- [orch-knowledge] Orchestrator Delegation Enforcement Mechanism
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-05-orchestrator-delegation-enforcement.md
- [orch-knowledge] Evolution Path to Safe Autonomous Triage
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-08-autonomous-triage-evolution-path.md
- [orch-knowledge] BACKLOG/ROADMAP Structure: Merge vs Two-File Separation
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-08-backlog-roadmap-merge-decision.md
- [orch-knowledge] Orchestrator Implementation Boundary: Delegate All vs Hybrid Model
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-08-orchestrator-implementation-boundary-final.md
- [orch-knowledge] Technical Fix vs. Behavioral Fix: Decision Framework
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-08-technical-fix-vs-behavioral-fix.md
- [orch-knowledge] Main-Branch-Only Git Workflow
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-12-main-branch-only-workflow.md
- [orch-knowledge] ADR: Self-Contained CLAUDE.md with Template Build System
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-14-orchestrator-restructuring-template-build-system.md
- [orch-knowledge] Session Amnesia as Foundational Design Constraint
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-14-session-amnesia-foundational-constraint.md
- [orch-knowledge] Deprecate Coordination Journal
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-15-deprecate-coordination-journal.md
- [orch-knowledge] Directive Guidance with Transparency Principle
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-15-directive-guidance-transparency-principle.md
- [orch-knowledge] Global Orchestration Knowledge Distribution Pattern
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-15-global-orchestration-knowledge-distribution.md
- [orch-knowledge] Orchestrator Authority Boundaries (Decide vs Escalate)
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-15-orchestrator-authority-boundaries.md
- [orch-knowledge] Skills Location Policy (CDD vs Dylan-Specific)
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-15-skills-location-policy.md
- [orch-knowledge] System-Level Amnesia as Design Constraint
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-15-system-amnesia-as-design-constraint.md
- [orch-knowledge] Orchestration Input Model: CLI vs Emacs
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-16-orchestration-input-model-cli-vs-emacs.md
- [orch-knowledge] Strategic Dogfooding for Meta-Orchestration
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-16-strategic-dogfooding.md
- [orch-knowledge] Agent Registry: File-Based Storage with fcntl Locking vs Database
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-17-agent-registry-file-based-vs-database.md
- [orch-knowledge] Project-Scoped Orchestrator Views: orch status Default Behavior
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-17-project-scoped-orchestrator-views-orch-status.md
- [orch-knowledge] Skill Reorganization & Taxonomy
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-18-skill-reorganization-taxonomy.md
- [orch-knowledge] ROADMAP Format - Markdown for Open Source Compatibility
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-19-roadmap-format-markdown-for-open-source.md
- [orch-knowledge] SPAWN_CONTEXT.md as First-Class Orchestrator Artifact
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-19-spawn-context-first-class-orchestrator-artifact.md
- [orch-knowledge] Track Workspace Files in Git
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-19-track-workspaces-in-git.md
- [orch-knowledge] Communication Intent Taxonomy for Orchestrator Interactions
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-20-communication-intent-taxonomy.md
- [orch-knowledge] Distribution Directory Naming Clarification
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-21-distribution-directory-naming.md
- [orch-knowledge] Action Plan: Orchestrator Instruction Optimization
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-21-instruction-optimization-action-plan.md
- [orch-knowledge] Investigation Agent Workspace Elimination
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-21-investigation-workspace-elimination.md
- [orch-knowledge] Orchestrator CLAUDE.md Size Limit Resolution
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-21-orchestrator-claude-md-size-limit-resolution.md
- [orch-knowledge] Orchestrator Instruction Synchronization via Explicit Commands
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-21-orchestrator-instruction-synchronization.md
- [orch-knowledge] Remove Redundant Architecture Context Injection
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-21-remove-architecture-context-injection.md
- [orch-knowledge] Codex Backend AGENTS.md Resolution
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-22-codex-backend-skip-agents-md.md
- [orch-knowledge] Formalize Orchestrator Dogfooding Practice
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-22-formalize-orchestrator-dogfooding-practice.md
- [orch-knowledge] Phase vs Status Field Separation
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-22-phase-status-field-separation.md
- [orch-knowledge] Hybrid Skill Architecture - Interactive vs Spawned Contexts
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-22-skill-system-hybrid-architecture.md
- [orch-knowledge] Archive Speculative ROADMAP Items
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-24-archive-speculative-roadmap-items.md
- [orch-knowledge] Simplify Investigation System
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-25-simplify-investigation-system.md
- [orch-knowledge] Orchestrator Autonomy Pattern
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-26-orchestrator-autonomy-pattern.md
- [orch-knowledge] Skill Architecture Consolidation (22 → 9)
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-26-skill-architecture-consolidation.md
- [orch-knowledge] Feature List Format Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-27-feature-list-format-design.md
- [orch-knowledge] Decision Lifecycle Management System
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-decision-lifecycle-management.md
- [orch-knowledge] Evidence Hierarchy Principle for Investigations
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-evidence-hierarchy-principle.md
- [orch-knowledge] Evolve by Distinction as Named Principle
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-evolve-by-distinction.md
- [orch-knowledge] Human Control Plane (Dashboard)
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-human-control-plane-dashboard.md
- [orch-knowledge] Keep Investigation Subdirectory Structure (Do Not Consolidate)
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-investigation-directory-organization.md
- [orch-knowledge] Pending Work Surfacing via Enhanced SessionStart
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-pending-work-surfacing.md
- [orch-knowledge] Rigidity vs Configurability in Orchestration System
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-rigidity-vs-configurability.md
- [orch-knowledge] Verify "NOT DONE" Claims in Self-Review Checklists
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-28-verify-not-done-claims-in-self-review.md
- [orch-knowledge] Artifact Organization - orch-cli / orch-knowledge Split
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-30-artifact-orch-cli-knowledge-split.md
- [orch-knowledge] Knowledge Integration as Orchestrator Core Responsibility
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-30-knowledge-integration-orchestrator-responsibility.md
- [orch-knowledge] orch-cli Template Architecture
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-30-orch-cli-template-architecture.md
- [orch-knowledge] Orchestrator Delegates All Investigations
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-04-orchestrator-delegates-all-investigations.md
- [orch-knowledge] bd work Delegates to orch spawn via Subprocess
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-07-bd-work-delegation-pattern.md
- [orch-knowledge] Beads Owns Lifecycle, Orch Owns Spawn Mechanics
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-07-beads-orch-responsibility-split.md
- [orch-knowledge] Knowledge Capture via PreToolUse Gate
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-07-knowledge-capture-pretooluse-gate.md
- [orch-knowledge] Minimal orch-cli - Spawn Mechanics Only
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-07-orch-cli-minimal-scope.md
- [orch-knowledge] Orchestrators Are Spawnable (Not Separate Session Concept)
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-07-orchestrator-session-explicit-lifecycle.md
- [orch-knowledge] Task Tool vs orch spawn - Context-Dependent Rule
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-07-task-tool-vs-orch-spawn-context-rule.md
- [orch-knowledge] Global KB Guides Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-12-global-kb-guides-design.md
- [orch-knowledge] Skillc Architecture and Principles
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/2025-12-21-skillc-architecture-and-principles.md
- [orch-knowledge] Superpowers Framework Independence
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/20251026-superpowers-independence.md
- [orch-knowledge] Agent Delegation Reduces Orchestrator Context Usage
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/20251102-delegation-reduces-context.md
- [orch-knowledge] Verification as Primary Orchestrator Responsibility
  - See: /Users/dylanconlin/orch-knowledge/.kb/decisions/20251103-verification-primary-orchestrator-responsibility.md
- [orch-cli] Five Concerns Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/decisions/2025-12-01-five-concerns-architecture.md
- [orch-cli] orch-cli Layered Architecture with Beads as Memory Layer
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/decisions/2025-12-01-orch-cli-architecture-layered-separation.md
- [orch-cli] Eliminate WORKSPACE.md from orch-cli
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/decisions/2025-12-06-eliminate-workspace-md.md
- [orch-cli] Go + OpenCode Agent Management
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/decisions/2025-12-18-sdk-based-agent-management.md
- [price-watch] ADR-005: SendCutSend Material Mapping Approach
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/decisions/2025-11-07-scs-material-mapping-approach.md
- [price-watch] Keep Unavailable Materials as Catalog Probes
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/decisions/2025-11-21-keep-unavailable-materials-probe-strategy.md
- [price-watch] Hide Forgot Password UI
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/decisions/2025-12-02-hide-forgot-password-ui.md
- [price-watch] OshCut Thickness Tolerance Strategy
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/decisions/2025-12-04-oshcut-thickness-tolerance-strategy.md
- [scs-slack] Monthly Slack Export Delivery Method
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/scs-slack/.kb/decisions/2025-12-19-slack-export-delivery-method.md
- [orch-go] Single-Agent Review Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-21-single-agent-review-command.md
- [skillc] Skillc Multi-Level Context Model
  - See: /Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-multi-level-context-model.md
- [orch-knowledge] Soft gate stderr output
- [dotfiles] yabai -m rule --add app!="^(Emacs|Ghostty|Firefox)$" space=3
- [dotfiles] tmux-fingers configuration with key binding overrides
- [price-watch] Fabworks extractStandardShippingCost using :has-text('Standard') selector
- [orch-go] Deleting orphaned beads dependencies with bd-* prefix

### Related Investigations
- [kb-cli] Auto-Rebuild Binary After Code Changes
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-08-auto-rebuild-binary-after-code.md
- [kb-cli] Project Registry Not Used in Global Search
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-12-debug-integrate-project-registry-global-search.md
- [kb-cli] Search Output Improvements
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-12-inv-search-output-improvements-limit-summary.md
- [kb-cli] Global Guides Publish and Guides Commands
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-12-simple-global-guides-publish-guides-commands.md
- [kb-cli] AI-Native Knowledge System - Comprehensive Design Document
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-13-design-ai-native-knowledge-system-comprehensive.md
- [kb-cli] AI-Native Knowledge Management Architecture
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-13-design-native-knowledge-management-architecture.md
- [kb-cli] Add kb context Command for Unified Knowledge Discovery
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-14-inv-add-context-command-unified-knowledge.md
- [kb-cli] Add Link Command to Connect Artifacts
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-14-inv-add-link-command-connect-artifacts.md
- [kb-cli] Add Promote Command Convert Entries
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-14-inv-add-promote-command-convert-entries.md
- [kb-cli] Add Init Command
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-19-inv-add-init-command.md
- [kb-cli] kb reflect --type open Implementation
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-21-inv-implement-kb-reflect-type-open.md
- [kb-cli] kb chronicle Command - Temporal Narrative View
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-21-inv-kb-chronicle-command-temporal-narrative.md
- [kb-cli] kb reflect MVP - Two Modes
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/2025-12-21-inv-kb-reflect-mvp-two-modes.md
- [kb-cli] Fix Create Investigation Slash Parsing Bug
  - See: /Users/dylanconlin/Documents/personal/kb-cli/.kb/investigations/simple/2025-12-07-fix-create-investigation-slash-parsing.md
- [orch-knowledge] Audit All Skills for .orch/.kb Path Mismatches
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-06-audit-all-skills-orch-path.md
- [orch-knowledge] .orch → .kb Migration Completion
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-06-complete-orch-migration-across-all.md
- [orch-knowledge] E2E Test - kb create Minimal Design Workflow
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-06-e2e-test-create-minimal-design.md
- [orch-knowledge] Migrate .orch → .kb for Remaining Projects
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-06-migrate-orch-remaining-projects.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-06-orchestrator-role-patterns.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-06-test-injection-explore-how-orchestrator.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-06-test-template-variable-rendering-verification.md
- [orch-knowledge] Orch Worker Lifecycle Friction Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-architect-analyze-orch-worker-lifecycle.md
- [orch-knowledge] How OpenCode Bypasses Anthropic's OAuth Credential Restriction
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-deep-investigation-how-opencode-bypass.md
- [orch-knowledge] Strategic Beads Orchestrator Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-design-strategic-beads-orchestrator-patterns.md
- [orch-knowledge] Fix kb create investigation slash bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-fix-create-investigation-slash-bug.md
- [orch-knowledge] Flatten Investigation Directory Implementation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-flatten-investigation-directory-configurable-path.md
- [orch-knowledge] Strange Terminal Escape Sequences in Claude Code Input
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-investigate-strange-terminal-escape-sequences.md
- [orch-knowledge] Investigation Directory Flattening Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-investigation-directory-flattening-design.md
- [orch-knowledge] Orch-Beads Integration - Guidance Location
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-orch-beads-integration-where-beads.md
- [orch-knowledge] Orchestrator Prompt Amplification Mechanisms
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-orchestrator-prompt-amplification-how-orchestrator.md
- [orch-knowledge] Path B Deep Dive - Beads as Lifecycle Owner
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-path-deep-dive-beads-lifecycle.md
- [orch-knowledge] Spawn Verification Test
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-test-spawn-verification.md
- [orch-knowledge] Update Orchestrator Skill with 5 Strategic Moments
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-07-update-orchestrator-skill-strategic-moments.md
- [orch-knowledge] Spawned Orchestrators Not Interactive Despite Request
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-debug-spawned-orchestrators-not-interactive.md
- [orch-knowledge] Deep Dive into Orch Ecosystem Principles and Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-design-deep-dive-orch-ecosystem.md
- [orch-knowledge] AI-First CLI Philosophy and kn Tightening Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-design-deep-exploration-orch-ecosystem-philosophy.md
- [orch-knowledge] Auto-name Orchestrator tmux Windows
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-auto-name-orchestrator-tmux-windows.md
- [orch-knowledge] Soft Gate Warning Not Visible in Claude Code
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-debug-soft-gate.md
- [orch-knowledge] Document Missing Orch Commands in Orchestrator Skill
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-document-missing-orch-commands-orchestrator.md
- [orch-knowledge] Document orch clean --stale in Orchestrator Skill
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-document-orch-clean-stale-orchestrator.md
- [orch-knowledge] Document orch work command in orchestrator skill
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-document-orch-work-command.md
- [orch-knowledge] Find All References to 'orch session' Command
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-find-all-references-orch-session.md
- [orch-knowledge] Knowledge Capture Gate for Architect Workers
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-implement-knowledge-capture-gate-architect.md
- [orch-knowledge] Implement Knowledge Capture Gate for Orchestrator Sessions
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-implement-knowledge-capture-gate-orchestrator.md
- [orch-knowledge] PreToolUse Hook for Knowledge Capture Gate
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-pretooluse-hook-pre-complete-check.md
- [orch-knowledge] Verify Orchestrator Skill Not Auto-Loading
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-quick-test-verify-orchestrator-skill.md
- [orch-knowledge] Remove Superseded orch session References
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-remove-superseded-orch-session-references.md
- [orch-knowledge] Soft Gate Warning Visibility with Edit Tool
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-retest-soft-gate-update-tool.md
- [orch-knowledge] Spawned Orchestrator Identity Confusion
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-spawned-orchestrator-identity-confusion.md
- [orch-knowledge] No-Gate Validation for Feature-Impl Skill Type
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-test-gate-validation.md
- [orch-knowledge] Soft Gate Validation for Investigation Skill
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-test-soft-gate-validation.md
- [orch-knowledge] Update Orchestrator Skill - Document --auto-track Flag
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-update-orchestrator-skill-document-auto.md
- [orch-knowledge] Update Orchestrator Skill - Document Beads Labels in SPAWN_CONTEXT
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-update-orchestrator-skill-document-beads.md
- [orch-knowledge] Update Orchestrator Skill with Session Lifecycle
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-update-orchestrator-skill-session-lifecycle.md
- [orch-knowledge] Worker Implemented Partial Scope Without Flagging Deviation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-08-inv-worker-implemented-partial-scope-without.md
- [orch-knowledge] orch end Command Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-09-design-add-orch-end-command-clean.md
- [orch-knowledge] Context Engineering Papers vs Orch Ecosystem
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-09-design-context-engineering-papers-orch-comparison.md
- [orch-knowledge] Why Agents Skip Completion Protocol Despite SESSION COMPLETE Block
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-09-inv-deep-investigation-r2fo-agent-completion.md
- [orch-knowledge] Knowledge Placement Decision Framework
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-09-inv-design-knowledge-placement-decision-framework.md
- [orch-knowledge] Rules for AI-First CLIs
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-09-inv-design-rules-first-clis-extract.md
- [orch-knowledge] Deep Analysis of .kb Directory - Knowledge Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-10-design-deep-analysis-kb-directory-patterns.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-10-design-first-cli-design-rules-interactive.md
- [orch-knowledge] Integration Wiring as Explicit Phase in Project Planning
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-10-design-integration-wiring-explicit-phase.md
- [orch-knowledge] Knowledge Artifact Taxonomy
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-10-design-knowledge-artifact-taxonomy.md
- [orch-knowledge] Window Picker for Orchestrator/Worker Switching
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-10-design-window-picker-orchestrator-worker-switching.md
- [orch-knowledge] Add --mcp Flag Documentation to Orchestrator Skill
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-10-inv-add-mcp-flag-documentation-orchestrator.md
- [orch-knowledge] Add SessionStart Hook for Agentlog Context Injection
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-10-inv-add-sessionstart-hook-agentlog-context.md
- [orch-knowledge] Beads Backlog Quality Gap Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-11-inv-beads-backlog-quality-gap-analysis.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-11-inv-design-emacs-following-active-orchestrator.md
- [orch-knowledge] Claude Code Installation Issues - Aider Venv PATH Pollution
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-debug-claude-code-aider-venv-path-pollution.md
- [orch-knowledge] Design: Meta-Orchestration for Cross-Project Coordination
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-design-meta-orchestration-cross-project-coordination.md
- [orch-knowledge] Audit Beads Backlogs Across orch-knowledge and orch-cli
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-inv-audit-beads-backlogs-across-orch.md
- [orch-knowledge] Design snap CLI - Screenshot Utility for AI Agents
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-inv-design-snap-cli-screenshot-utility.md
- [orch-knowledge] Test OpenCode Backend API
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-inv-test-opencode-backend-v2.md
- [orch-knowledge] Test OpenCode Backend
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-inv-test-opencode-backend.md
- [orch-knowledge] Update Knowledge Files for Option C Architecture Decision
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-inv-update-knowledge-option-c.md
- [orch-knowledge] Workers Misusing kb create investigation --type Flag
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-inv-workers-misusing-create-investigation-type.md
- [orch-knowledge] Research: Claude Max Weekly Limit API - Programmatic Usage Checking
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-12-research-claude-max-weekly-limit-api.md
- [orch-knowledge] Add orch build --opencode validation command
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-13-inv-add-orch-build-opencode-validation.md
- [orch-knowledge] Skill/Hook System Migration from Claude Code to OpenCode
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-13-inv-investigate-how-skill-hook-system.md
- [orch-knowledge] Port Remaining Hooks to OpenCode Plugins
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-13-inv-port-remaining-hooks-opencode-plugins.md
- [orch-knowledge] Reconcile session-context.ts with session-start.sh behavior
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-13-inv-reconcile-session-context-session-start.md
- [orch-knowledge] Orchestrator Skill Audit - Drift, Redundancy, Obsolete Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-14-audit-orchestrator-skill-drift-redundancy.md
- [orch-knowledge] Orchestrator Skill Phase 4 - Extract Reference Material
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-14-simple-orchestrator-skill-phase-extract-reference.md
- [orch-knowledge] Orchestrator Skill Refactoring - Phase 2+3
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-14-simple-orchestrator-skill-refactoring-phase-consolidate.md
- [orch-knowledge] Design Investigation: Feature Creation Workflow
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-15-design-feature-creation-workflow.md
- [orch-knowledge] Add Design Triage Section to Orchestrator Skill
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-15-inv-add-design-triage-section-orchestrator.md
- [orch-knowledge] Design Investigation: Issue Quality System
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-15-inv-design-issue-quality-system.md
- [orch-knowledge] Migrate Orchestrator Skill to kb context
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-15-inv-migrate-orchestrator-skill-from-context.md
- [orch-knowledge] Where kb link Fits in Orchestration Workflow
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-15-inv-where-link-fit-orchestration-workflow.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-16-inv-design-flexible-issue-creation-patterns.md
- [orch-knowledge] Install opencode-skills Plugin for Portable Skill Support
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-16-inv-install-opencode-skills-plugin-portable.md
- [orch-knowledge] Add Design-Session to Orchestrator Skill Selection Guide
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-17-inv-add-design-session-orchestrator-skill.md
- [orch-knowledge] Opportunities to Expand triage:ready Pattern Across Skills
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-17-inv-investigate-opportunities-expand-triage-ready.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-17-inv-test-auto-create-session.md
- [orch-knowledge] Raw Tmux Fallback for orch spawn
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-17-inv-test-raw-tmux-fallback.md
- [orch-knowledge] Box-Drawing Characters for Workflow Diagrams
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-17-simple-explore-box-drawing-characters-workflow.md
- [orch-knowledge] Add E2E Tests for Hooks and Context Loading
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-add-e2e-tests-hooks-context.md
- [orch-knowledge] Adding Friction Point for Model Selection When Spawning Agents
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-add-friction-point-model-selection.md
- [orch-knowledge] Add ai-help Command to bd CLI
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-add-help-command-cli.md
- [orch-knowledge] Add Scope Enumeration Checkpoint to Feature-Impl
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-add-scope-enumeration-checkpoint-feature.md
- [orch-knowledge] Agent Gets Stuck Waiting for Emacs
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-agent-gets-stuck-waiting-emacs.md
- [orch-knowledge] Beads CLI Add Warning for Short Descriptions
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-beads-cli-add-warning-short.md
- [orch-knowledge] Create Issue Quality Skill for Portable Standards
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-create-issue-quality-skill-portable.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-detect-new-cli-commands-not.md
- [orch-knowledge] Flatten Spawn Template Scope Sections
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-flatten-spawn-template-scope-sections.md
- [orch-knowledge] orch spawn hangs when run in background mode
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-orch-spawn-hangs-when-run.md
- [orch-knowledge] Port Remaining Claude Code Hooks to OpenCode
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-port-remaining-claude-code-hooks.md
- [orch-knowledge] Smoke Test Buffering Fix for Background Mode Spawn
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-smoke-test-buffering-fix.md
- [orch-knowledge] Buffering Smoke Test - Investigation Workflow Verification
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-test-investigation-buffering-smoke-test.md
- [orch-knowledge] Test Spawn Workflow
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-test-spawn.md
- [orch-knowledge] Update Orchestrator Skill - Auto-Track Default
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-update-orchestrator-skill-auto-track.md
- [orch-knowledge] Verify reliability-testing Handles Discovered Issues with Triage Labels
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-18-inv-verify-reliability-testing-handles-discovered.md
- [orch-knowledge] Add Orch Ecosystem Architecture Documentation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-19-inv-add-orch-ecosystem-architecture-doc.md
- [orch-knowledge] orch build skills fails to deploy to OpenCode
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-19-inv-orch-build-skills-fails-deploy.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-19-inv-say-hello.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-19-inv-test-cross-project-complete-say.md
- [orch-knowledge] Test Spawn Integration with Beads Tracking
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-19-inv-test-spawn-integration-beads-tracking.md
- [orch-knowledge] Update Orchestrator Skill - Remove orch work
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-19-inv-update-orchestrator-skill-remove-orch.md
- [orch-knowledge] Sync Knowledge with orch-go (Headless Swarm & Synthesis Protocol)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/2025-12-20-inv-sync-knowledge-orch-go-headless.md
- [orch-knowledge] Investigation Templates
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/INDEX.md
- [orch-knowledge] Codex Agents Hang on Spawn - AGENTS.md Orchestrator Guidance Confusion
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/agent-failures/2025-11-22-codex-agents-hang-on-spawn-agents-md-orchestrator-guidance.md
- [orch-knowledge] Agent Failure Investigation: Codex Ignores .orch Directory
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/agent-failures/2025-11-22-codex-dotfile-ignore-root-cause.md
- [orch-knowledge] Agent Failure Investigation: Codex Execution Loop / Spawn Hang
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/agent-failures/2025-11-22-codex-execution-loop-spawn-hang.md
- [orch-knowledge] Codex Hangs on All Prompts - AGENTS.md Context Overload
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/agent-failures/2025-11-22-codex-hangs-on-all-prompts-agents-md-context-overload.md
- [orch-knowledge] Why Agent Forgot to Commit Changes (Recurring Pattern)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/agent-failures/2025-11-24-agent-forgot-commits-recurring.md
- [orch-knowledge] Agent Failures Investigations (Frozen/Legacy)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/agent-failures/README.md
- [orch-knowledge] CDD Privacy Audit for Public Release
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-01-cdd-privacy-audit.md
- [orch-knowledge] Doom Emacs CDD Integration Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-02-doom-cdd-audit-direct.md
- [orch-knowledge] Empirical Audit of Session Amnesia Resilience
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-14-empirical-audit-amnesia-resilience.md
- [orch-knowledge] Architecture Audit: Meta-Orchestration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-architecture-audit.md
- [orch-knowledge] Why Codebase Audit Agents Terminated Prematurely
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-investigate-why-codebase-audit-agents.md
- [orch-knowledge] Orchestrator CLAUDE.md Template System Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-orchestrator-template-audit.md
- [orch-knowledge] Organizational Drift Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-organizational-audit.md
- [orch-knowledge] Performance Audit: Meta-Orchestration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-performance-audit.md
- [orch-knowledge] Quick Audit Scan - Meta-Orchestration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-quick-audit-scan.md
- [orch-knowledge] Comprehensive Security Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-security-audit.md
- [orch-knowledge] Comprehensive Tests Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-15-tests-audit.md
- [orch-knowledge] Documentation Audit - Recent Code Changes vs Docs
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-19-documentation-audit.md
- [orch-knowledge] Audit: Orchestrator Template Marker Compliance (Final Report)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-19-orchestrator-template-marker-audit-final.md
- [orch-knowledge] Codebase Audit Investigation: Orch Command Coverage in Orchestrator Memory
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-21-audit-orch-command-coverage-orchestrator.md
- [orch-knowledge] Codebase Audit Investigation: Orchestrator Instruction Value
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-21-systematic-analysis-orchestrator-instruction-value.md
- [orch-knowledge] Codebase Audit Investigation: Root Directory Cleanup
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-22-audit-clean-root-directory-users.md
- [orch-knowledge] Workspace Audit: 2025-11-22 Completed Workspaces
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-22-audit-completed-2025-workspaces-unfinished.md
- [orch-knowledge] Codebase Audit Investigation: Documentation Completeness
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-22-audit-orch-cli-commands-features.md
- [orch-knowledge] Codebase Audit Investigation: Worker Skills - Investigation Template Workflow Implementation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-24-audit-all-worker-skills-that.md
- [orch-knowledge] Deep Analysis: Orchestrator Instruction Templates
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-24-deep-analysis-orchestrator-instruction-templates.md
- [orch-knowledge] Codebase Audit Investigation: Architecture
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-26-architecture-audit.md
- [orch-knowledge] Codebase Audit Investigation: Architecture (Follow-up)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-27-architecture-audit-god-objects-coupling.md
- [orch-knowledge] Organizational Drift Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-27-organizational-drift-audit.md
- [orch-knowledge] Performance Audit: Spawn Timing, Caching, I/O Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-27-performance-audit-spawn-timing-caching.md
- [orch-knowledge] Codebase Security Audit Investigation: Nov 27 vs Nov 15 Comparison
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-27-security-audit.md
- [orch-knowledge] Codebase Audit Investigation: Test Quality
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/audits/2025-11-27-tests-quality-comprehensive-audit.md
- [orch-knowledge] Design Investigation: Discovery Work in features.json
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-28-discovery-work-in-features-json.md
- [orch-knowledge] Hookify Adoption Decision
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-29-hookify-adoption-decision.md
- [orch-knowledge] Design: Investigation Recommendation Surfacing
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-29-investigation-recommendation-surfacing.md
- [orch-knowledge] Design Investigation: Should Orch CLI Be Open Sourced?
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-29-open-source-orch-cli-decision.md
- [orch-knowledge] Design Investigation: orch Future Direction - What to Adopt from VC
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-29-orch-future-direction-vc-adoption.md
- [orch-knowledge] Was the Orchestrator Template Build System Over-Engineering?
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-29-orchestrator-template-build-system-evaluation.md
- [orch-knowledge] Artifact Organization After orch-cli Extraction
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-artifact-organization-orch-cli-knowledge-split.md
- [orch-knowledge] Full Day Synthesis: November 29-30, 2025
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-full-day-synthesis-nov29-30.md
- [orch-knowledge] orch-cli Template Architecture
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-orch-cli-template-architecture.md
- [orch-knowledge] Policy vs Procedure Skill Distinction
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-policy-vs-procedure-skill-distinction.md
- [orch-knowledge] Post-Mortem: Orchestration Session 30 November 2025
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-session-post-mortem-30nov.md
- [orch-knowledge] Post-Mortem: Third Orchestration Session Analysis (30 November 2025)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-session-post-mortem-third.md
- [orch-knowledge] Session Synthesis: November 30, 2025 Orchestration Improvements Sprint
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-session-synthesis-nov30-improvements.md
- [orch-knowledge] Skills as Executable Natural Language
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-11-30-skills-as-executable-natural-language.md
- [orch-knowledge] Cross-Repo Beads Visibility for Orch Ecosystem
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-05-cross-repo-beads-visibility.md
- [orch-knowledge] Design: Skill Benchmark Project
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-05-skill-benchmark-project-design.md
- [orch-knowledge] Design Investigation: All Orch Spawns Require Beads Tracking?
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-06-all-orch-spawns-beads-tracking.md
- [orch-knowledge] Deep Exploration of Orch Ecosystem for Rewrite Decision
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-06-deep-exploration-orch-ecosystem-rewrite.md
- [orch-knowledge] Design Investigation: Ad-Hoc Spawns Without Beads Issues
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-ad-hoc-spawns-without-beads-issues.md
- [orch-knowledge] bd work Interface Design - Skills, Context Injection, tmux Integration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-bd-work-interface-skills-context.md
- [orch-knowledge] Cross-Repo Beads Spawning Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-cross-repo-beads-spawning-design.md
- [orch-knowledge] Knowledge Capture Gates Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-design-implement-knowledge-capture-gates.md
- [orch-knowledge] Orchestrator Session Lifecycle and Spawn Mechanism
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-design-orchestrator-session-lifecycle-spawn.md
- [orch-knowledge] Meta-Orchestration Principles Refinement
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-discuss-potentially-refine-meta-orchestration.md
- [orch-knowledge] Minimal orch-cli After Beads-Native Agents
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-minimal-orch-cli-after-beads.md
- [orch-knowledge] orch spawn Commit Check Friction
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-orch-spawn-commit-check-friction.md
- [orch-knowledge] SPAWN_CONTEXT Storage Location in Beads-Native Model
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-spawn-context-storage-location-design.md
- [orch-knowledge] Task Tool vs tmux Agents - When to Use Each
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-task-tool-tmux-agents-when.md
- [orch-knowledge] Unified Spawnable Sessions with Knowledge Gates
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-unified-spawnable-sessions-knowledge-gates.md
- [orch-knowledge] bd work Implementation Strategy Given Beads Team Rejection
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-07-work-implementation-strategy-beads-team.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/design/2025-12-08-brainstorm-ui-orchestration.md
- [orch-knowledge] Data Collection Systems Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-04-data-collection-analysis.md
- [orch-knowledge] Quick-Spawn Failure Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-07-quick-spawn-failure-analysis.md
- [orch-knowledge] Meta-Orchestration Investigations Migration to Unified-KB
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-08-kb-migration-analysis.md
- [orch-knowledge] Strategic Analysis: Meta-Orchestration Project
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-08-meta-orchestration-strategic-analysis.md
- [orch-knowledge] Research: uv for Python Dependency Management in orch CLI
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-13-uv-migration-analysis.md
- [orch-knowledge] Research: Claude Code CLI Alternatives (OpenAI Codex, Google Gemini)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-18-evaluate-claude-code-alternatives-openai-codex-google.md
- [orch-knowledge] Evaluate Whether orch learn Command Automation Is Needed
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-18-evaluate-whether-orch-learn-command.md
- [orch-knowledge] AI Agent Orchestration Frameworks: Competitive Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-19-agent-orchestration-frameworks-competitive-analysis.md
- [orch-knowledge] Gemini CLI Integration Architecture and Feasibility
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-19-gemini-cli-integration-architecture-feasibility.md
- [orch-knowledge] Feasibility Investigation: Orch CLI Codex Integration Technical Architecture
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-21-orch-cli-codex-integration-technical-architecture.md
- [orch-knowledge] Research: Workspace Status Field Design (Phase + Status)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-21-workspace-status-field-design.md
- [orch-knowledge] Feasibility Investigation: Orch inbox feature design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-22-brainstorm-orch-inbox-feature-design.md
- [orch-knowledge] Feasibility Investigation: Skill System - Embedded vs Autonomous Discovery
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-22-skill-system-embedded-vs-autonomous.md
- [orch-knowledge] Research: Amp vs Claude Code CLI Comparison
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/2025-11-30-amp-vs-claude-code-comparison.md
- [orch-knowledge] Feasibility Investigations (Frozen/Legacy)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/feasibility/README.md
- [orch-knowledge] Evolution of Meta-Orchestration System
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-25-investigate-how-meta-orchestration-system.md
- [orch-knowledge] Why Investigation Agents Add Extra Sections Not In Template
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-25-why-investigation-agents-sometimes-add.md
- [orch-knowledge] Why orch complete reports "ROADMAP item not found" for existing items
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-25-why-orch-complete-sometimes-report.md
- [orch-knowledge] Why orch spawn sometimes takes 5-10 seconds before agent window appears
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-25-why-orch-spawn-sometimes-take.md
- [orch-knowledge] CLI Bloat Reduction Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-audit-orch-cli-bloat-reduction.md
- [orch-knowledge] Evaluate 'Codebase Expert' Agent Pattern for Crystallization
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-evaluate-codebase-expert-agent-pattern.md
- [orch-knowledge] Explore OpenCode as Potential Orch Backend
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-explore-opencode-potential-orch-backend.md
- [orch-knowledge] Missing Investigation Template in orch create-investigation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-fix-missing-investigation-template-orch.md
- [orch-knowledge] Fix orch dashboard crash: Rich markup error
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-fix-orch-dashboard-crash-rich-markup.md
- [orch-knowledge] Investigate Large CLAUDE.md Performance Warnings
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-investigate-large-claude-performance-warnings.md
- [orch-knowledge] Shell CWD Reset Messages Investigation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-investigate-shell-cwd-reset-messages.md
- [orch-knowledge] OpenCode Architecture
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-opencode-architecture.md
- [orch-knowledge] Design: OpenCode as Orch Backend
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-opencode-orch-backend-design.md
- [orch-knowledge] Review Investigations for Proto-Decisions to Promote
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-review-all-investigations-orch-investigations.md
- [orch-knowledge] ~/.claude/CLAUDE.md Being Overwritten by Orchestrator Context
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-root-cause-analysis-claude-claude.md
- [orch-knowledge] Root Cause: .orch/CLAUDE.md Exceeded Claude Code 40k Warning
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-root-cause-orch-claude-exceeded.md
- [orch-knowledge] Skill Consolidation - SPAWN_CONTEXT Embedding Test
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-26-test-skill-consolidation-read-your.md
- [orch-knowledge] Anthropic Long-Running Agents - Strategic Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-anthropic-long-running-agents-strategic-analysis.md
- [orch-knowledge] Bug: orch complete fails when ROADMAP.org missing
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-bug-orch-complete-fails-when-roadmap-missing.md
- [orch-knowledge] Investigation Thrashing Detector Evaluation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-evaluate-investigation-thrashing-detector-investigate.md
- [orch-knowledge] Final E2E Stash Test
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-final-e2e-stash-test.md
- [orch-knowledge] Agent Session Termination Bug - Agents Not Calling /exit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-fix-agent-session-termination-bug.md
- [orch-knowledge] Fix orch clean crash when agent window is None
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-fix-orch-clean-crash-when.md
- [orch-knowledge] Fix Registry Re-Animation Race
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-fix-registry-animation-race.md
- [orch-knowledge] Workspace Verification Race Condition
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-fix-workspace-verification-race.md
- [orch-knowledge] Stash Test - List Files
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-27-stash-test-just-list-files.md
- [orch-knowledge] AskUserQuestion Tool Empirical Testing
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-askuserquestion-empirical-testing.md
- [orch-knowledge] Artifact Audit: Unfinished Business from Nov 27-28, 2025
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-audit-all-artifacts-created-last.md
- [orch-knowledge] Audit: Frontmatter Usage Across Orchestration System
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-audit-frontmatter-usage-across-orchestration.md
- [orch-knowledge] Pre-Commit Hook Audit: Missing Checks
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-audit-pre-commit-hook-missing.md
- [orch-knowledge] Research: Claude Code AskUserQuestion Tool Documentation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-claude-code-askuserquestion-tool.md
- [orch-knowledge] Status Field Inconsistency in Investigations
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-deep-dive-investigation-status-field.md
- [orch-knowledge] Claude Code Repository Resources for Meta-Orchestration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-investigate-claude-code-repo-resources.md
- [orch-knowledge] Web Dashboard Showing Stale Cached JS Despite Rebuild
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-28-web-dashboard-stale-cache-issue.md
- [orch-knowledge] Feature-Dev 7-Phase vs Feature-Impl 4-Phase Comparison
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-compare-feature-dev-phase-discovery.md
- [orch-knowledge] Context Continuity Gap Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-context-continuity-gap-analysis-project.md
- [orch-knowledge] Beads vs Orch Comparison
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-explore-compare-beads-orch-what.md
- [orch-knowledge] Multi-Agent Parallel Review for Codebase Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-explore-multi-agent-parallel-review.md
- [orch-knowledge] Fix: find_investigation_file should use primary_artifact from agent registry
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-fix-find-investigation-file-use.md
- [orch-knowledge] Investigation recommendations surfacing in daemon instead of CLI
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-fix-investigation-recommendations-surfacing-daemon.md
- [orch-knowledge] How many decisions were created in the last 7 days?
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-quick-test-how-many-decisions.md
- [orch-knowledge] List Most Recent Workspaces Created Today
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-quick-test-list-most-recent-workspaces-created-today.md
- [orch-knowledge] Test Coverage Analysis for Orch CLI
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-quick-test-what-current-test-coverage-orch-cli-scope.md
- [orch-knowledge] Code-Review Plugin Confidence Scoring Pattern Study
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-study-code-review-plugin-confidence.md
- [orch-knowledge] [Topic]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-29-test-spawn-reliability-verify-timeout.md
- [orch-knowledge] Comprehensive Audit of Orch CLI Commands
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-30-comprehensive-audit-orch-cli-commands.md
- [orch-knowledge] Markdown Field Extraction: Frontmatter Audit
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-30-markdown-field-extraction-frontmatter-audit.md
- [orch-knowledge] Spawn Context Drift Review
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-30-review-spawn-context-drift-skills.md
- [orch-knowledge] What is the Current State of Skill Testing Infrastructure?
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-11-30-what-current-state-skill-testing.md
- [orch-knowledge] Side-by-Side Comparison: browser-use MCP vs playwright MCP
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-02-side-side-comparison-browser-use.md
- [orch-knowledge] Beads Grouping Features Integration into Orch Workflows
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-05-beads-grouping-features-integration-into.md
- [orch-knowledge] Brainstorming Skill Integration Pattern
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-05-brainstorming-skill-integration-pattern-how.md
- [orch-knowledge] tmux/ghostty Configuration Sprawl Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-05-investigate-tmux-ghostty-configuration-sprawl.md
- [orch-knowledge] orch complete verification fails - investigation file not found
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-05-orch-complete-verification-fails-investigation.md
- [orch-knowledge] Price-Watch Post-Mortem Analysis (Dec 5, 2025)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-05-price-watch-post-mortem.md
- [orch-knowledge] Proposal: Automated Knowledge Sync between orch-go and orch-knowledge
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/simple/2025-12-20-sync-automation-proposal.md
- [orch-knowledge] CDD Component Inventory
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-01-cdd-component-inventory.md
- [orch-knowledge] CDD Core vs Optional Component Classification
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-01-cdd-core-vs-optional-classification.md
- [orch-knowledge] OAuth Token Authentication Failures
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-01-oauth-token-auth.md
- [orch-knowledge] Superpowers Architectural Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-01-superpowers-architectural-patterns.md
- [orch-knowledge] Why Agents Forgot to Commit Changes
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-02-agents-forgot-commits.md
- [orch-knowledge] Context Usage - Orchestration vs Direct Execution
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-02-context-usage-orchestration-vs-direct.md
- [orch-knowledge] Emacs Orchestrator Integration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-02-emacs-orchestrator-integration.md
- [orch-knowledge] gh Repo Defaulting to SendAssist
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-03-gh-repo-defaulting-root-cause.md
- [orch-knowledge] Prompt-Based Stop Hooks for Agent Validation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-03-prompt-based-stop-hooks.md
- [orch-knowledge] Tmux Window Rename Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-03-tmux-rename-bug.md
- [orch-knowledge] Recommendation Patterns in Meta-Orchestration System
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-analyze-recommendation-patterns-propose-improvements.md
- [orch-knowledge] Fix orch status phase parser
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-fix-orch-status-phase-parser.md
- [orch-knowledge] implement-feature Skill Context Detection
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-mode-autonomous-add-context-detection.md
- [orch-knowledge] Window Reuse Registry Tracking
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-mode-autonomous-detect-when-window.md
- [orch-knowledge] Infra-Flag Append Pattern with ROADMAP Structure
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-mode-autonomous-verify-untriaged-section.md
- [orch-knowledge] orch clean Only Removed 1 of 5 Completed Agents
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-orch-clean-only-removed-completed.md
- [orch-knowledge] Should `orch` Be Project-Aware?
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-orch-project-awareness.md
- [orch-knowledge] Orch Command Registry Mechanism
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-orch-registry-mechanism.md
- [orch-knowledge] orch send doesn't actually send messages
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-orch-send-doesn-actually-send.md
- [orch-knowledge] ROADMAP Update Process Breakdown
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-08-roadmap-update-process-breakdown.md
- [orch-knowledge] alt-o Binding Application Filtering
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-09-alt-binding-needs-switch-between.md
- [orch-knowledge] Fix Failing Orch Tests
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-09-fix-failing-orch-tests.md
- [orch-knowledge] Hephaestus Project - Learnings for Orch/CDD Approach
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-09-hephaestus-learnings.md
- [orch-knowledge] Research: Task Runners for JavaScript Projects
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-09-task-runners-javascript.md
- [orch-knowledge] Research: Tmux Vertico/Consult-Style Window Switching
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-09-tmux-vertico-style-window-switching.md
- [orch-knowledge] yabai alt-o binding cycle behavior
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-09-yabai-alt-binding-cycle-only.md
- [orch-knowledge] orch clean Window Closure Failure
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-fix-orch-clean-failing-close.md
- [orch-knowledge] orch clean Incorrectly Treating window_closed as Completed
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-fix-orch-clean-incorrectly-treating.md
- [orch-knowledge] Fix orch spawn to support resuming existing workspaces
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-fix-orch-spawn-support-resuming.md
- [orch-knowledge] Fix Registry Reconciliation Using Window ID
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-fix-registry-reconciliation-using-window.md
- [orch-knowledge] Fix Tmux Window Number Mismatch
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-fix-tmux-window-number-mismatch.md
- [orch-knowledge] Workspace Creation Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-fix-workspace-creation-bug-where.md
- [orch-knowledge] CLAUDE.md Drift from Orch Command Features
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-investigate-claude-drift-from-orch.md
- [orch-knowledge] Workspace Name Underscore Handling Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-mode-autonomous-begin_example-orch-spawn.md
- [orch-knowledge] orch question Multi-Line Extraction Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-mode-autonomous-testing-revealed-when.md
- [orch-knowledge] ROADMAP Drift - Completed Work Marked as TODO
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-mode-interactive-work-done-without.md
- [orch-knowledge] Resume Functionality Test
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-10-test-task.md
- [orch-knowledge] Fix Failing Tests in test_spawn.py
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-11-fix-failing-tests-test-spawn.md
- [orch-knowledge] orch clean Incomplete Cleanup Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-11-mode-autonomous.md
- [orch-knowledge] orch logs Not Showing Today's Logs
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-11-orch-logs-not-showing-today.md
- [orch-knowledge] Fix Tmux Window Number Mismatch (Root Cause: Window Renumbering)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-11-tmux-window-renumbering-fix.md
- [orch-knowledge] Why Spawn Command Executed 4x Creating Duplicate Windows
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-12-follow-investigation-why-spawn-command-execute-4x.md
- [orch-knowledge] Window Number Prediction and Registry Tracking
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-12-mode-autonomous-window-number-prediction.md
- [orch-knowledge] orch tail shows wrong window
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-12-orch-tail-shows-wrong-window.md
- [orch-knowledge] Recurring tmux Window Tracking Issues
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-12-tmux-window-tracking-issues.md
- [orch-knowledge] Knowledge Artifact Location After Unified-KB Migration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-12-verify-knowledge-artifacts-being-created.md
- [orch-knowledge] Fix 11 Failing Tests
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-13-fix-failing-tests-132-passing.md
- [orch-knowledge] Why Worker Agents Don't Run Tests After Changes
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-13-why-worker-agents-not-running-tests.md
- [orch-knowledge] Brainstorming Skill Auto-Spawns Workers
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-bug-brainstorming-skill-auto-spawns.md
- [orch-knowledge] implement-feature Skill Missing Design Doc Detection
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-bug-implement-feature-skill-doesn.md
- [orch-knowledge] Orchestrator tmux Session Disappears on Last Agent Completion
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-bug-when-orchestrator-tmux-session.md
- [orch-knowledge] Dashboard returns 404 on /agents endpoint
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-dashboard-returns-404-agents-endpoint.md
- [orch-knowledge] 'Habit' Pattern Across Claude Code Skills, CDD, and Orch System
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-explore-habit-pattern-across-claude.md
- [orch-knowledge] Fix "No module named web" Error in orch serve
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-fix-module-named-web-error.md
- [orch-knowledge] click.prompt() Blocking in Non-Interactive Spawn
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-mode-autonomous.md
- [orch-knowledge] What Can We Borrow from Organizational Knowledge Management?
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-org-km-patterns-vs-orch-system.md
- [orch-knowledge] Programmatic Context Monitoring for Claude Code Sessions
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-programmatic-context-monitoring.md
- [orch-knowledge] Session Amnesia as Foundational Design Constraint
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-14-session-amnesia-philosophical-implications.md
- [orch-knowledge] Agent Spawn Session Termination Pattern
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-agent-spawn-session-termination.md
- [orch-knowledge] Orchestrator-Human Interaction Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-analyze-orchestrator-human-interaction-patterns.md
- [orch-knowledge] Fix orch history showing 0 duration for completed agents
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-fix-orch-history-showing-duration.md
- [orch-knowledge] Fix `orch lint --all` Recursive Scanning Performance Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-fix-orch-lint-all-recursive.md
- [orch-knowledge] Fix orch spawn --from-roadmap Task Extraction
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-fix-orch-spawn-from-roadmap.md
- [orch-knowledge] Root Cause of 2-Minute Session Termination Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-investigate-root-cause-minute-session.md
- [orch-knowledge] Spawn Prompt Delivery Failure
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-investigate-spawn-prompt-delivery-failure.md
- [orch-knowledge] llm vs orch - CLI Architecture Comparison
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-llm-vs-orch-comparison.md
- [orch-knowledge] Orch Project Registry Global Search Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-orch-project-registry-global-search-bug.md
- [orch-knowledge] GitHub Search for 2-Minute Session Termination Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-search-github-existing-issues-related.md
- [orch-knowledge] Systematic Debugging of 2-Minute Session Termination Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-15-systematic-debug-minute-session.md
- [orch-knowledge] Agent Spawn Validation Requirements
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-add-agent-spawn-validation-document.md
- [orch-knowledge] Alfred Text Expansion Setup and Programmatic Management
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-alfred-text-expansion-setup-programmatic-management.md
- [orch-knowledge] WebSocket Patterns for Live Dashboard Updates
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-explore-websocket-patterns-live-dashboard.md
- [orch-knowledge] Fix Claude Wrapper Prompt Delivery Bug
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-fix-claude-wrapper-prompt-delivery.md
- [orch-knowledge] Interactive Spawn Implementation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-interactive-spawn-implementation.md
- [orch-knowledge] Agent Registry Reconciliation Timing
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-investigate-agent-registry-reconciliation-timing.md
- [orch-knowledge] Orch CLI & Data Structures for UI Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-orch-cli-data-structures-for-ui.md
- [orch-knowledge] ROADMAP.org System Evaluation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-roadmap-system-evaluation.md
- [orch-knowledge] WebSocket Patterns for Vue Dashboard Improvements
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-websocket-patterns-vue-dashboard-improvements.md
- [orch-knowledge] Workspace Population Pattern Violation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-16-workspace-population-pattern-violation.md
- [orch-knowledge] Agent Registry Race Condition
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-17-agent-registry-race-condition-concurrent.md
- [orch-knowledge] Autonomous Verification Workflow Integration with orch status
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-17-autonomous-verification-orch-status-integration.md
- [orch-knowledge] --from-roadmap --resume Failure
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-17-investigate-from-roadmap-resume-failure.md
- [orch-knowledge] orch status Not Showing All Agents
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-17-orch-status-not-showing-any.md
- [orch-knowledge] Fragmentation Across Implementation Skills and Multi-Phase Validation Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-18-analyze-fragmentation-across-implementation-skills.md
- [orch-knowledge] Fix All Failing Tests
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-18-fix-all-failing-tests-test.md
- [orch-knowledge] Fix Remaining Failing Tests (Continuation)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-18-fix-remaining-failing-tests-continuation.md
- [orch-knowledge] orch send Causing Orchestrator Tmux Window Copy Mode
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-18-investigate-orch-send-causing-orchestrator.md
- [orch-knowledge] Code Orchestration & Delegation Tools: Competitive Landscape
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-code-orchestration-delegation-tools-competitive.md
- [orch-knowledge] Competitive Landscape Synthesis: Meta-Orchestration Strategic Positioning
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-competitive-landscape-synthesis.md
- [orch-knowledge] Dual-Format ROADMAP Config Support
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-dual-format-roadmap-config-support.md
- [orch-knowledge] Registry Re-Animation Race - Phantom Agents from Concurrent Operations
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-fix-registry-animation-race-phantom.md
- [orch-knowledge] Gemini 3.0 Analysis Synthesis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-gemini-analysis-synthesis.md
- [orch-knowledge] Ghost Template Bug - SPAWN_PROMPT.md Not Loaded
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-ghost-template-bug-spawn-prompt-not-loaded.md
- [orch-knowledge] Why Agents Don't Exit Automatically After Marking Phase: Complete
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-investigate-why-agents-don-exit.md
- [orch-knowledge] ROADMAP Markdown Structure and Parser Design
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-roadmap-markdown-structure-and-parser-design.md
- [orch-knowledge] Template Build System - PROJECT-SPECIFIC Marker Support
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-template-build-system-project-specific-markers.md
- [orch-knowledge] Worker Skill Invocation Failure
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-19-worker-skill-invocation-failure.md
- [orch-knowledge] Catastrophic Failure Modes
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-catastrophic-failure-modes.md
- [orch-knowledge] Communication Intent Taxonomy - Sequential vs. Alternatives
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-communication-intent-taxonomy-sequential-vs-alternatives.md
- [orch-knowledge] Org-mode vs Markdown for Workspace Files
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-compare-org-mode-markdown-workspace.md
- [orch-knowledge] Context Loading Patterns and Agent Splitting Strategy
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-context-loading-patterns-and-agent-splitting.md
- [orch-knowledge] Global CLAUDE.md Build System Integration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-global-claude-md-build-system-integration.md
- [orch-knowledge] Research: Internet Reaction to Gemini 3.0
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-internet-reaction-gemini.md
- [orch-knowledge] SessionEnd Hook Verification
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-sessionend-hook-verification.md
- [orch-knowledge] Skill Content Embedding vs Progressive Disclosure
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-skill-content-embedding-progressive-disclosure.md
- [orch-knowledge] Spawn Prompt Template Disconnect
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-spawn-prompt-template-disconnect.md
- [orch-knowledge] Orchestration System Fixes Verification
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-test-both-fixes-skill-content.md
- [orch-knowledge] Hook-Based Lifecycle (Phase: Complete Trigger)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-20-test-hook-based-lifecycle-write.md
- [orch-knowledge] Architecture Context Injection Redundancy
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-21-architecture-context-injection-redundancy-recursive.md
- [orch-knowledge] CDD Essentials to Template Migration Mapping
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-21-cdd-essentials-to-template-migration-mapping.md
- [orch-knowledge] Backend selection (lines 1167-1173)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-21-how-backend-adapter-pattern-work.md
- [orch-knowledge] Synthesis Workflow Integration Requirements
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-21-synthesis-workflow-integration-requirements.md
- [orch-knowledge] Template Marker Synchronization Gap in Orchestrator CLAUDE.md Files
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-21-template-marker-synchronization-gap.md
- [orch-knowledge] Test ClaudeBackend Integration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-21-test-claude-backend.md
- [orch-knowledge] Codex CLI Documentation Sources and Sync Feasibility
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-codex-cli-documentation-sources.md
- [orch-knowledge] Codex Integration & Multi-Agent Context Projection
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-codex-integration-architecture.md
- [orch-knowledge] Config System and Dual-Format ROADMAP Support
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-config-system-dual-format-roadmap.md
- [orch-knowledge] Debug Codex CLI Hang - Implementation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-debug-why-codex-cli-hangs.md
- [orch-knowledge] Dylan's tmux/Ghostty/yabai Window Setup
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-dylan-tmux-ghostty-yabai-window-setup.md
- [orch-knowledge] Feature-Impl Skill Structure Analysis
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-feature-impl-skill-structure-analysis.md
- [orch-knowledge] Strategic Hook Opportunities in Orchestration System
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-hook-opportunities-claude-code.md
- [orch-knowledge] Multi-Agent Memory Architecture Comparison
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-multi-agent-memory-architecture-comparison.md
- [orch-knowledge] [Investigation Title]
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-orch-directory-structure-and-purpose.md
- [orch-knowledge] orch patterns --trending Feature Verification
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-orch-patterns-trending-feature-verification.md
- [orch-knowledge] PROJECT-SPECIFIC Section Misuse in Meta-Orchestration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-project-specific-section-misuse-cleanup.md
- [orch-knowledge] Claude Code Spawn Mechanism Test
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-quick-test-read-readme-confirm.md
- [orch-knowledge] ROADMAP.org Current Phase Status
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-roadmap-current-phase-status.md
- [orch-knowledge] Sketchybar auto-update not firing
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-sketchybar-auto-update-not-firing.md
- [orch-knowledge] Orchestrator Template Size Reduction Strategy
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-template-size-reduction-analysis.md
- [orch-knowledge] Test Codex Backend E2E Integration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-test-codex-backend-e2e-integration.md
- [orch-knowledge] Test sketchybar auto-update in new session
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-test-sketchybar-auto-update-new-session.md
- [orch-knowledge] Test sketchybar updates - quick smoke test
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-test-sketchybar-updates-quick-smoke.md
- [orch-knowledge] Spawn Process Verification
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-test-spawn-verify-agent-creation.md
- [orch-knowledge] Workspace Template and Feedback Integration
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-22-workspace-template-and-feedback-integration.md
- [orch-knowledge] Codex Hang Epistemic Debt Case Study
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-document-codex-hang-epistemic-debt.md
- [orch-knowledge] Registry fcntl Locking Mechanism
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-how-registry-fcntl-locking-mechanism.md
- [orch-knowledge] Phase 1 Review - Self-Coordinating Investigation Workflow
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-review-phase-self-coordinating-investigation.md
- [orch-knowledge] Template Variable Substitution Mechanism
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-test-template-variable-substitution-work.md
- [orch-knowledge] Verify Self-Coordinating Investigation Workflow
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-verify-self-coordinating-investigation-workflow.md
- [orch-knowledge] Gemini CLI Usage Limits for Paid Tiers
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-23-what-specific-usage-limits-gemini.md
- [orch-knowledge] Critical Bug - Skill Content Embedding Failure
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-24-critical-bug-skill-content-embedding.md
- [orch-knowledge] Templates That Benefit from Dedicated Orch CLI Commands
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-24-identify-templates-that-benefit-from.md
- [orch-knowledge] Orch Create-Investigation CLI Command - Template and CLI Patterns
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-24-orch-create-inv-cli-command-template-patterns.md
- [orch-knowledge] Test Verification - Skill Content Embedding
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-24-test-verification-confirm-skill-content.md
- [orch-knowledge] Skill System Architecture Consolidation
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-26-skill-system-architecture-consolidation.md
- [orch-knowledge] VC vs orch - Two Philosophies of AI-Assisted Development
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/2025-11-29-vc-vs-orch-philosophical-comparison.md
- [orch-knowledge] Systems Investigations (Frozen/Legacy)
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/README.md
- [orch-knowledge] Discussion: Template Marker Synchronization Approach
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/gemini-build-discussion.md
- [orch-knowledge] implement-feature Skill Architecture Mismatch
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/systems/implement-feature-skill-mismatch.md
- [orch-knowledge] yegge-blogpost-beads-orchestration.md
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/yegge-blogpost-beads-orchestration.md
- [orch-knowledge] yegge-vibe-coding.md
  - See: /Users/dylanconlin/orch-knowledge/.kb/investigations/yegge-vibe-coding.md
- [.doom.d] Doom Emacs Configuration Audit
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/2025-11-02-doom-config-audit.md
- [.doom.d] ROADMAP.org Existence and Setup
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/2025-11-16-check-roadmap-org-exists-set.md
- [.doom.d] JSON Decode Error in orch-status Auto-Update
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/2025-11-16-fix-json-decode-error-orch.md
- [.doom.d] Code Quality Audit - config/orch-config.el
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/audits/2025-11-30-orch-config-code-audit.md
- [.doom.d] AI-Assisted Emacs Development Workflow
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/design/2025-11-30-ai-assisted-emacs-dev-workflow.md
- [.doom.d] Bug .doom.d-dw5 - Cursor Reset NOT Fixed
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/simple/2025-11-30-bug-doom-dw5-not-fixed.md
- [.doom.d] Debug Two Related Auto-Update Issues
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/simple/2025-11-30-debug-two-related-auto-update.md
- [.doom.d] Tab to Collapse/Expand Agents is Broken
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/simple/2025-11-30-orch-config-tab-collapse-expand.md
- [.doom.d] org-refile Hang with UTF8 Entities Message
  - See: /Users/dylanconlin/.doom.d/.kb/investigations/systems/2025-11-21-debug-org-refile-hang-utf8.md
- [snap] Design snap CLI - Screenshot Utility for AI Agents
  - See: /Users/dylanconlin/Documents/personal/snap/.kb/investigations/2025-12-12-inv-design-snap-cli-screenshot-utility.md
- [snap] Implement snap list (window discovery)
  - See: /Users/dylanconlin/Documents/personal/snap/.kb/investigations/2025-12-12-inv-implement-snap-list-window-discovery.md
- [beads] Add source_repo to bd list --json
  - See: /Users/dylanconlin/Documents/personal/beads/.kb/investigations/2025-12-21-inv-add-source-repo-field-bd.md
- [beads] bd list --json source_repo returns null
  - See: /Users/dylanconlin/Documents/personal/beads/.kb/investigations/2025-12-21-inv-bd-list-json-not-returning.md
- [beads] Fix bd repo add to write YAML
  - See: /Users/dylanconlin/Documents/personal/beads/.kb/investigations/2025-12-21-inv-fix-bd-repo-add-write.md
- [agentlog] Add Rails/Turbo TypeScript Snippet
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-add-rails-turbo-typescript-snippet.md
- [agentlog] agentlog init --install Implementation
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-agentlog-init-actually-install-snippets.md
- [agentlog] Create Go Snippet Tests
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-create-go-snippet.md
- [agentlog] Create Python Snippet
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-create-python-snippet.md
- [agentlog] Create Ruby/Rails Snippet
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-create-ruby-rails-snippet.md
- [agentlog] Create Rust Snippet
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-create-rust-snippet.md
- [agentlog] Create TypeScript Snippet
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-create-typescript-snippet.md
- [agentlog] End-to-End CLI Testing
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-end-end-cli-testing.md
- [agentlog] Implement Doctor Command
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-implement-doctor-command.md
- [agentlog] Implement 'errors' Command
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-implement-errors-command.md
- [agentlog] Implement Init Command
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-implement-init-command.md
- [agentlog] Implement prime command
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-implement-prime-command.md
- [agentlog] Implement Tail Command
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-implement-tail-command.md
- [agentlog] orch-cli and beads-ui-svelte Integration Patterns for agentlog
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-research-orch-cli-beads-svelte.md
- [agentlog] Task: Scaffold Go CLI with Cobra
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-scaffold-cli-cobra.md
- [agentlog] Validate beads-ui-svelte Integration Compatibility
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-validate-beads-svelte-integration.md
- [agentlog] Write JSONL Schema Spec
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-10-inv-write-jsonl-schema-spec.md
- [agentlog] Add --path Flag for Monorepo/Subdir Support
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-11-inv-add-path-flag-monorepo-subdir.md
- [agentlog] Auto-detect Node.js vs Browser TypeScript
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-11-inv-auto-detect-node-browser-typescript.md
- [agentlog] Node.js Snippet Auto-Update .gitignore
  - See: /Users/dylanconlin/Documents/personal/agentlog/.kb/investigations/2025-12-11-inv-node-snippet-auto-update-gitignore.md
- [orch-cli] Fix orch init references deprecated command
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-11-30-fix-orch-init-references-deprecated.md
- [orch-cli] What is 2+2?
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-11-30-what-answer-briefly-complete-immediately.md
- [orch-cli] Browser Use Session Persistence
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-01-browser-use-session-persistence.md
- [orch-cli] OpenCode Backend Implementation Investigation
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-01-investigate-opencode-backend-implementation-orch.md
- [orch-cli] Claude Code Native Session Management vs tmux
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-01-investigate-whether-orch-cli-use.md
- [orch-cli] Research: Agent Mail MCP Server
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-02-agent-mail-mcp.md
- [orch-cli] orch spawn --issue fails with project not found
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-02-fix-orch-spawn-issue-fails.md
- [orch-cli] Research: Go vs Python vs Rust for CLI Tools
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-02-go-python-rust-cli-comparison.md
- [orch-cli] Implement Tail Command Support for OpenCode Backend
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-02-implement-tail-command-support-opencode.md
- [orch-cli] orch create-investigation "Template not found" Error
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-02-orch-create-investigation-fails-template.md
- [orch-cli] Agent Mail Integration Test
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-02-test-agent-mail-integration-register.md
- [orch-cli] Agent Mail Messaging System Test
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-02-test-agent-mail-messaging-just.md
- [orch-cli] Why Agents Skip Completion Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-04-agents-skip-completion-protocol.md
- [orch-cli] Audit: SPAWN_CONTEXT.md Quality (Dec 2-4 Workers)
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-04-audit-spawn-context-quality.md
- [orch-cli] Leveraging Claude Code --agent Flag for Skill-Based Spawning
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-05-claude-agent-flag-for-skills.md
- [orch-cli] Duplicate Prefixes in Tmux Window Names
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-05-fix-duplicate-prefixes-tmux-window.md
- [orch-cli] Where WORKSPACE.md Files Are Still Being Created
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-05-investigate-where-workspace-files-still.md
- [orch-cli] Agent Registry Removal - Use Beads as Single Source of Truth
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-agent-registry-removal-remove-registry.md
- [orch-cli] Eliminate WORKSPACE.md from orch-cli
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-eliminate-workspace-entirely-from-orch.md
- [orch-cli] primary_artifact set for non-required deliverables
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-fix-primary-artifact-set-non.md
- [orch-cli] orch check false positive for missing WORKSPACE.md
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-orch-check-false-positive-workspace.md
- [orch-cli] Parse Areas Needing Investigation in orch complete
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-orch-complete-parse-areas-needing.md
- [orch-cli] orch complete verification filename mismatch
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-orch-complete-verification-filename-mismatch.md
- [orch-cli] orch output show issue titles
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-orch-output-show-issue-titles.md
- [orch-cli] pattern_detection.py Removal - Dead Code Cleanup
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-pattern-detection-removal-delete-orch.md
- [orch-cli] Single Coordination Mechanism - Converge on Beads Comments
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-single-coordination-mechanism-converge-beads.md
- [orch-cli] spawn.py Consolidation Strategy
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-spawn-consolidation-reduce-from-2188.md
- [orch-cli] Unify Path Conventions to .kb/
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-unify-path-conventions-remove-orch.md
- [orch-cli] Notes-Based Agent Metadata Storage
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-06-update-beads-integration-write-agent.md
- [orch-cli] Orchestrator Window Stability During Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-07-test-confirm-orchestrator-window-stays.md
- [orch-cli] Real-time UI Updates in beads-ui
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-07-test-real-time-updates.md
- [orch-cli] Verify Client Switching Works After Fix
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-07-test-verify-client-switching-works.md
- [orch-cli] Per-Project Workers Sessions Verification
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-07-test-verify-per-project-workers.md
- [orch-cli] Add load_kb_context() for spawn prompt enrichment
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-add-load-context-spawn-prompt.md
- [orch-cli] Audit Beads Issues for Underspecified Migration Paths
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-audit-beads-issues-underspecified-migration.md
- [orch-cli] Enumerate .orch/ References - Migrate vs Keep
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-enumerate-orch-references-migrate-keep.md
- [orch-cli] When Can orch complete Be Removed?
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-investigate-when-orch-complete-removed.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-orch-clean-stale-clean-agents.md
- [orch-cli] orch lint --issues: Validate Beads Issues
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-orch-lint-issues-validate-beads.md
- [orch-cli] Pass Beads Labels to SPAWN_CONTEXT
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-pass-beads-labels-spawn-context.md
- [orch-cli] Removing .orch/workspace/ Directory
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-removing-orch-workspace-directory.md
- [orch-cli] Show Convergence Status in orch status
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-show-convergence-status-orch-status.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-test-auto-track-creates-beads.md
- [orch-cli] Spawn Context Switch Prevention
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-08-inv-test-spawn-context-switch-prevention.md
- [orch-cli] Fix orch clean bug - save() got unexpected keyword argument 'skip_merge'
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-debug-fix-orch-clean-bug-save.md
- [orch-cli] Git Validation Race on .beads/ During Parallel Completions
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-debug-git-validation-race-beads-during.md
- [orch-cli] File Search Keyword Extraction
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-debug-investigation-file-search-keyword-extraction.md
- [orch-cli] Phase Field Detection Too Fragile
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-debug-investigation-phase-field-detection.md
- [orch-cli] orch complete Beads ID Lookup and Investigation Filename Bugs
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-debug-orch-complete-fix-beads-lookup.md
- [orch-cli] Validate repo consistency before closing beads issues
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-debug-validate-repo-consistency-before-closing.md
- [orch-cli] Add Tool/Model Fields to Skill Frontmatter Schema
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-add-allowed-tools-disallowed-tools.md
- [orch-cli] Add Local Error Logging with Analytics
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-add-local-error-logging-analytics.md
- [orch-cli] Generate .claude/agents/ Files at Spawn Time
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-generate-claude-agents-files-spawn.md
- [orch-cli] Feature: orch complete auto-send /exit
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-orch-complete-auto-send-exit.md
- [orch-cli] orch complete Error Patterns (110 failures/day)
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-orch-complete-error-patterns-110.md
- [orch-cli] orch complete --force should trust commits over phase status
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-orch-complete-force-trust-commits.md
- [orch-cli] Parallel orch complete Support for Beads
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-orch-complete-support-parallel-completions.md
- [orch-cli] orch end Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-orch-end-command-clean-session.md
- [orch-cli] orch end race condition - /exit via tmux arrives during tool execution
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-orch-end-race-condition-exit.md
- [orch-cli] orch status default to active-only
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-orch-status-default-active-only.md
- [orch-cli] Remove Interactive Prompts from orch CLI
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-remove-interactive-prompts-from-orch.md
- [orch-cli] SessionStart Error Hook Verification
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-09-inv-test-verify-sessionstart-hook-shows.md
- [orch-cli] Fix --mcp flag: add -- separator before prompt
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-debug-fix-mcp-flag-add-separator.md
- [orch-cli] Fix --mcp flag to write config to temp file
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-debug-fix-mcp-flag-write-config.md
- [orch-cli] Orchestration Architecture - Is AI-native CLI Optimal?
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-design-orchestration-architecture-native-cli-optimal.md
- [orch-cli] Add agentlog context injection to spawn_prompt.py
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-inv-add-agentlog-context-injection-spawn.md
- [orch-cli] Add agentlog_integration.py wrapper
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-inv-add-agentlog-integration-wrapper.md
- [orch-cli] Agent-to-Orchestrator Completion Notification
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-inv-agent-orchestrator-completion-notification.md
- [orch-cli] Claude Agent SDK Integration Possibilities for orch-cli
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-inv-claude-agent-sdk-integration-possibilities.md
- [orch-cli] Unified Error Aggregation Across CLI Tools
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-10-inv-unified-error-aggregation-across-orch.md
- [orch-cli] Verify BUILTIN_MCP_SERVERS Fix
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-11-debug-fix-builtin-mcp-servers-wrong.md
- [orch-cli] Workers Not Receiving Playwright MCP When Spawned with --mcp Flag
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-11-debug-workers-not-receiving-playwright-mcp.md
- [orch-cli] Bug Lifecycle Architecture Design
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-11-inv-architect-bug-lifecycle-design-error.md
- [orch-cli] Bug Handling Analysis in orch-cli
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-11-inv-bug-handling-analysis-orch-cli.md
- [orch-cli] Fix --project flag with full path creates invalid tmuxinator config
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-11-inv-fix-project-flag-full-path.md
- [orch-cli] Optimize orch spawn output for Claude Code display
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-11-inv-optimize-orch-spawn-output-claude.md
- [orch-cli] Workspace Name Collision Across Projects
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-11-inv-workspace-name-collision-across-projects.md
- [orch-cli] OpenCode Compatibility Audit for orch-cli and orch-knowledge
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-audit-opencode-compatibility-orch-cli.md
- [orch-cli] Fix OpenCode spawn automation - Enter keypress not submitting
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-debug-fix-opencode-spawn-automation-enter.md
- [orch-cli] Fix Registry Merge Logic Bug
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-debug-fix-registry-merge-logic-bug.md
- [orch-cli] OpenCode Spawn Flow - Sessions Not Working
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-debug-opencode-spawn-flow-sessions.md
- [orch-cli] orch spawn hangs with long multi-line task prompts
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-debug-orch-spawn-hangs-long-multi.md
- [orch-cli] Debug 'session not found' error during OpenCode spawn
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-debug-session-not-found-error.md
- [orch-cli] Session Not Found Error in OpenCode Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-debug-session-not-found-opencode.md
- [orch-cli] Add Focus Integration to Work Daemon
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-add-focus-integration-work-daemon.md
- [orch-cli] Audit Hooks for OpenCode Compatibility
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-audit-hooks-opencode-compatibility.md
- [orch-cli] Create OpenCode Plugin for bd close Gate
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-create-opencode-plugin-block-close.md
- [orch-cli] Create OpenCode Plugin for Session Context Loading
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-create-opencode-plugin-session-context.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-daemon-use-projects-registry-instead.md
- [orch-cli] Issue Refinement Stage (draft → ready) for Daemon Intake
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-design-issue-refinement-stage-draft.md
- [orch-cli] Document OpenCode Plugin Setup for orch-cli
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-document-opencode-plugin-setup-orch.md
- [orch-cli] Work Daemon Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-implement-work-daemon-autonomous-beads.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-model-test-verify-defaults-opus.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-orch-usage-check-claude-max.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-test-config.md
- [orch-cli] Test Spawn Timing Fix
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-test-spawn-timing-fix.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-inv-verify-spawn-fix-see-this.md
- [orch-cli] Quick Test Verification
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-12-simple-quick-test-verification-complete-immediately.md
- [orch-cli] orch tail fails for OpenCode agents - 'no session_id' error
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-13-debug-orch-tail-fails-opencode-agents.md
- [orch-cli] Redesigning Issue Creation in orch Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-13-design-redesign-issue-creation-process-orch.md
- [orch-cli] Beads Issue Management Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-13-investigation-beads-issue-management-patterns.md
- [orch-cli] Orchestrator Skill Loads for OpenCode Workers
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-14-debug-orchestrator-skill-loads-workers-hook.md
- [orch-cli] Add orch review command for batched completion synthesis
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-14-inv-add-orch-review-command-batched.md
- [orch-cli] Add Proper Background Daemonization with launchd/systemd Integration
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-14-inv-add-proper-background-daemonization-launchd.md
- [orch-cli] Daemon Auto-Completion Boundaries
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-14-inv-daemon-auto-completion-boundaries.md
- [orch-cli] Playwright MCP Context Window Blowout
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-14-inv-playwright-mcp-context-window-blowout.md
- [orch-cli] Port Playwright MCP Features to OpenCode Backend
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-14-inv-port-playwright-mcp-features-opencode.md
- [orch-cli] Programmatic Export Trigger in OpenCode
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-16-inv-investigate-opencode-export-programmatic-trigger.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-16-inv-quick-test-verify-session-transcript.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-16-inv-test-mcp-config-generation.md
- [orch-cli] Test Playwright MCP Settings
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-16-inv-test-playwright-mcp-settings-navigate.md
- [orch-cli] Verify Playwright MCP Tools Availability
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-16-inv-test-verify-playwright-mcp-tools.md
- [orch-cli] Cleanup .orch/CLAUDE.md References in orch-knowledge
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-cleanup-orch-knowledge-claude-md-refs.md
- [orch-cli] OpenCode MCP Config Write Location
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-debug-opencode-mcp-config-write-workspace.md
- [orch-cli] Auto-initialize .orch/ in repos without setup
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-design-auto-initialize-orch-repos.md
- [orch-cli] Design: Fix Stale Agents with Phase: Unknown
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-design-epic-fix-stale-agents-phase.md
- [orch-cli] Design: Future of the Orchestration Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-design-future-orchestration-ecosystem.md
- [orch-cli] Remove .orch/CLAUDE.md References from orch-cli
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-inv-cleanup-remove-all-orch-claude.md
- [orch-cli] Design: Solution for Stale Agents with Phase: Unknown
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-inv-design-solution-stale-agents-phase.md
- [orch-cli] Fix Placeholder Substitution in SPAWN_CONTEXT Critical Instructions
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-inv-fix-placeholder-substitution-spawn-context.md
- [orch-cli] Make --auto-track default for orch spawn
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-inv-make-auto-track-default-orch.md
- [orch-cli] orch spawn auto-create workers session
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-inv-orch-spawn-auto-create-workers.md
- [orch-cli] orch spawn auto-initialize minimal .orch/ when missing
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-17-inv-orch-spawn-auto-initialize-minimal.md
- [orch-cli] Deep Audit of orch-cli Codebase
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-audit-deep-audit-orch-cli-codebase.md
- [orch-cli] Zero Fragile Code (ZFC) Audit Across Orch Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-audit-zero-fragile-code-zfc-audit.md
- [orch-cli] Fix OpenCode Agent Completion Cleanup
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-debug-fix-opencode-agent-completion-cleanup.md
- [orch-cli] Fix Transcript Export for OpenCode Backend
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-debug-fix-transcript-export-opencode-using.md
- [orch-cli] KeyError project_dir in orch complete after registry slimming
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-debug-orch-complete-broken-after-registry.md
- [orch-cli] orch friction command API error
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-debug-orch-friction-command-api-error.md
- [orch-cli] Agent Monitoring Scalability Design
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-design-agent-monitoring-scalability.md
- [orch-cli] Design: ctx - Unified Context Assembler
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-design-ctx-unified-context-assembler.md
- [orch-cli] Epic: Simplify .orch/ Initialization and Cleanup Legacy References
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-design-epic-simplify-orch-initialization-cleanup.md
- [orch-cli] Add Browser Cleanup Instruction to SPAWN_CONTEXT
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-add-browser-cleanup-instruction-spawn.md
- [orch-cli] Add Transcript Formatter for OpenCode JSON Exports
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-add-transcript-formatter-convert-opencode.md
- [orch-cli] Add Warning When Spawning Without Beads Tracking
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-add-warning-when-spawning-without.md
- [orch-cli] Bug Lifecycle Error Analysis from Price-Watch
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-bug-lifecycle-design-examine-actual.md
- [orch-cli] Consolidate Error Logging to Agentlog Schema
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-consolidate-error-logging-agentlog-schema.md
- [orch-cli] Critical Review of ctx Unified Context Assembler Design
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-deep-exploration-honest-review-ctx.md
- [orch-cli] Design orch friction command for system introspection
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-design-orch-friction-command-system.md
- [orch-cli] Flip OAuth Token Priority to OpenCode First
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-flip-oauth-token-priority-opencode.md
- [orch-cli] Implement orch friction command
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-implement-orch-friction-command.md
- [orch-cli] Implement src/orch/ask.py - ZFC Model Call Utility
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-implement-src-orch-ask-zfc.md
- [orch-cli] Migrate orch-cli from ~/.orch/errors.jsonl to .agentlog/errors.jsonl
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-migrate-orch-cli-from-orch.md
- [orch-cli] Multi-Account Usage Checking Design
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-multi-account-usage-checking-design.md
- [orch-cli] OpenCode Export Produces Corrupted JSON
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-opencode-export-produces-corrupted-json.md
- [orch-cli] orch spawn --auto-track uses full task as beads title, exceeds 500 char limit
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-orch-spawn-auto-track-uses.md
- [orch-cli] Phase 4 Registry Cleanup and Verification
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-phase-registry-cleanup-verification.md
- [orch-cli] Phase 3 - Strip Registry Writes to tmux-only
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-phase-strip-registry-writes-tmux.md
- [orch-cli] Phase 2 - Update Read Paths to Prefer Beads
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-phase-update-read-paths-prefer.md
- [orch-cli] Redesign orch review as synthesis command
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-redesign-orch-review-synthesis-command.md
- [orch-cli] Why OpenCode Feels Faster Than Claude Code
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-inv-why-opencode-feel-faster-than.md
- [orch-cli] Multi-Account Usage Checking Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-18-simple-multi-account-usage-checking-implement.md
- [orch-cli] Investigation Deliverable Verification Fails to Find Existing Files
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-investigation-deliverable-verification-fails-find.md
- [orch-cli] OpenCode Spawn Reads Wrong SPAWN_CONTEXT.md Path First
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-opencode-spawn-reads-wrong-spawn.md
- [orch-cli] orch abandon doesn't close beads issue
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-orch-abandon-doesn-close-beads.md
- [orch-cli] orch clean doesn't remove abandoned agents from status
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-orch-clean-doesn-remove-abandoned.md
- [orch-cli] orch complete crashes with KeyError when agent missing project_dir
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-orch-complete-crashes-keyerror-when.md
- [orch-cli] Orch Ecosystem Architecture Map
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-orch-ecosystem-architecture-map-full.md
- [orch-cli] orch status --global shows stale phase data
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-orch-status-global-shows-stale.md
- [orch-cli] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-test-verification-fix-check-primary.md
- [orch-cli] Why Daemon Reports 19 Active Agents
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/2025-12-19-inv-why-daemon-reports-active-agents.md
- [orch-cli] Fix orch build global Path Detection
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/design/2025-11-30-orch-build-global-path-fix.md
- [orch-cli] orch-cli's Strategic Role in the AI Agent Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/design/2025-12-01-orch-cli-role-in-agent-ecosystem.md
- [orch-cli] kb Project Architecture Design
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/design/2025-12-03-kb-project-architecture.md
- [orch-cli] Post-Mortem: Worker Double-Tracking (Workspace + Beads)
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/design/2025-12-04-post-mortem-worker-double-tracking.md
- [orch-cli] Per-Project Workers Sessions Design
  - See: /Users/dylanconlin/Documents/personal/orch-cli/.kb/investigations/simple/2025-12-07-per-project-workers-sessions-design.md
- [beads-ui-svelte] IssuesList.svelte SSR and @const Fix Verification
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-debug-fix-issueslist-svelte-verification.md
- [beads-ui-svelte] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-board-view-blocked-ready-progress.md
- [beads-ui-svelte] Command Palette (Cmd+K) Implementation
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-command-palette-cmd.md
- [beads-ui-svelte] Create Issue Dialog Implementation
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-create-issue-dialog.md
- [beads-ui-svelte] Dependency Indicators in List View
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-dependency-indicators-list-view.md
- [beads-ui-svelte] Issue Detail View with Inline Editing
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-issue-detail-view-inline-editing.md
- [beads-ui-svelte] Issues List View with Vim Navigation
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-issues-list-view-vim-navigation.md
- [beads-ui-svelte] Link Investigations to Beads Issues
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-link-investigations-beads-issues.md
- [beads-ui-svelte] Wire WebSocket Client to Main Page
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-09-inv-wire-websocket-client-main-page.md
- [beads-ui-svelte] Fix Broken Vim j/k Keybindings in Issues List
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-debug-fix-broken-vim-keybindings-issues.md
- [beads-ui-svelte] Lines Toggle Adds Padding But No Connector Lines
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-debug-lines-toggle-adds-padding-doesn.md
- [beads-ui-svelte] Optimal Sort Patterns for Beads UI Issues List
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-design-beads-sort-patterns-optimal.md
- [beads-ui-svelte] Console Log Bridge Implementation
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-add-console-log-bridge-frontend.md
- [beads-ui-svelte] Add DependencyGraph Component to UI
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-add-dependencygraph-component.md
- [beads-ui-svelte] Show Closed Toggle Implementation
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-add-show-closed-toggle-issues.md
- [beads-ui-svelte] Analyze orch-cli Error Logging Patterns
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-analyze-orch-cli-error-logging.md
- [beads-ui-svelte] Audit beads-ui-svelte for Unwired Components
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-audit-beads-svelte-unwired-components.md
- [beads-ui-svelte] Enhance Console Bridge to Catch Uncaught Errors
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-enhance-console-bridge-catch-uncaught.md
- [beads-ui-svelte] Epic Child Dependency Model Confusion
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-epic-child-dependency-model-confusion.md
- [beads-ui-svelte] Epic Inline Positioning
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-epic-inline-positioning-appear-before.md
- [beads-ui-svelte] Beads Data Model Mental Model
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-explore-beads-data-model.md
- [beads-ui-svelte] WebSocket Auto-Reconnection Implementation
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-implement-websocket-auto-reconnection-beadsclient.md
- [beads-ui-svelte] Project Header Reactive Stats Update
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-update-project-header-when-beads.md
- [beads-ui-svelte] E2E Test Today's Features with Playwright MCP
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-use-playwright-mcp-verify-new.md
- [beads-ui-svelte] Why Do Dev Servers Need Frequent Restarts?
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-why-dev-servers-need-frequent.md
- [beads-ui-svelte] Wire CommandPalette Actions to Real Functionality
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-wire-commandpalette-actions-real-functionality.md
- [beads-ui-svelte] Wire Epics Page to WebSocket
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-wire-epics-page-websocket-replace.md
- [beads-ui-svelte] Wire IssueDetail Component to Main Page
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-10-inv-wire-issuedetail-component-main-page.md
- [beads-ui-svelte] tmux-follower CPU spike from rapid project switching
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-11-debug-tmux-follower-cpu-spike-rapid.md
- [beads-ui-svelte] Make Beads IDs Click-to-Copy
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-11-inv-make-beads-ids-click-copy.md
- [beads-ui-svelte] Fix Inverted Blocks/Blocked-by Badge Labels
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-12-debug-fix-inverted-blocks-blocked-badge.md
- [beads-ui-svelte] Add Comments Display to IssueDetail Component
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-12-inv-add-comments-display-issuedetail-component.md
- [beads-ui-svelte] Add Demo Beads Dataset
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-12-inv-add-demo-beads-dataset-showcasing.md
- [beads-ui-svelte] Add Toggle Switch Between Live Project and Demo Dataset
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-12-inv-add-toggle-switch-between-live.md
- [beads-ui-svelte] Responsive IssueDetail Side Panel Layout
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-13-inv-responsive-issuedetail-side-panel-layout.md
- [beads-ui-svelte] Add Dark/Light Theme Toggle
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-16-inv-add-dark-light-theme-toggle.md
- [beads-ui-svelte] Add Proper Navbar Component
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-16-inv-add-proper-navbar-component.md
- [beads-ui-svelte] Optimize Dark Theme Readability
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-16-inv-optimize-dark-theme-readability-use.md
- [beads-ui-svelte] Issue Detail Panel Split-Pane Conversion
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-17-simple-issue-detail-panel-split-pane-conversion.md
- [beads-ui-svelte] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-21-inv-add-multi-repo-filtering-ui.md
- [beads-ui-svelte] Add Multi-Repo Support to beads-ui-svelte
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-21-inv-add-multi-repo-support-beads.md
- [beads-ui-svelte] Does beads-ui-svelte support multi-repo hydration?
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-21-inv-does-beads-ui-svelte-support-multi-repo-hydration.md
- [beads-ui-svelte] Verify beads-ui Multi-Repo Filtering
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/2025-12-21-inv-verify-beads-ui-multi-repo.md
- [beads-ui-svelte] Beads UI Svelte - Data Model, CLI, and UI Analysis
  - See: /Users/dylanconlin/Documents/personal/beads-ui-svelte/.kb/investigations/simple/2025-12-09-beads-svelte-exploration-beads-source.md
- [dotfiles] tmux Line-Copy Mode with Ace-Jump Style Hints
  - See: /Users/dylanconlin/Documents/dotfiles/.kb/investigations/2025-12-11-inv-tmux-line-copy-mode-ace.md
- [dotfiles] tmux Configuration Optimization Investigation
  - See: /Users/dylanconlin/Documents/dotfiles/.kb/investigations/audits/2025-11-02-tmux-optimization.md
- [dotfiles] Docker Desktop Runaway Electron Logging
  - See: /Users/dylanconlin/Documents/dotfiles/.kb/investigations/simple/2025-12-05-docker-desktop-runaway-electron-logging.md
- [dotfiles] Dotfiles Integration Audit: Symlink Coverage
  - See: /Users/dylanconlin/Documents/dotfiles/.kb/investigations/systems/2025-11-23-audit-dotfiles-integration-identify-files.md
- [dotfiles] Comprehensive Keybinding System Analysis
  - See: /Users/dylanconlin/Documents/dotfiles/.kb/investigations/systems/2025-11-23-comprehensive-keybinding-system-analysis.md
- [price-watch] [Investigation Title]
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-05-fix-sendcutsend-scraper-file-upload.md
- [price-watch] Production Reliability Validation Strategy
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-05-investigate-production-reliability-validation-strategy.md
- [price-watch] Recurring BullMQ Job Disappearance After Scraper Restarts
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-05-investigate-recurring-bullmq-job-disappearance.md
- [price-watch] Run 47 Webhook Failure Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-05-why-rails-webhook-return-500.md
- [price-watch] Spawn System Verification
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-07-test-spawn-system-verification.md
- [price-watch] Test Spawn - Verify Orchestration System Working
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-07-test-spawn-verify-orchestration-system.md
- [price-watch] Design Investigation: Scraper Action Visibility
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-08-design-scraper-action-visibility.md
- [price-watch] Complete Trace Retention Strategy - Selective Success Capture
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-08-inv-complete-trace-retention-strategy-selective.md
- [price-watch] Extend Trace Viewer Links to All Cells in Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-08-inv-extend-trace-viewer-links-all.md
- [price-watch] Scraper Action Visibility Implementation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-08-inv-scraper-action-visibility-impl.md
- [price-watch] Spawn Status Display - Verify Agent Shows WORKING Not COMPLETED
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-08-inv-test-spawn-status-display-verify.md
- [price-watch] Fix Comparison View - Filter by Quote Status Instead of Run Status
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-debug-fix-comparison-view-filter-quote.md
- [price-watch] Fix Navbar Showing Dark Theme in Light Mode
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-debug-fix-navbar-showing-dark-theme.md
- [price-watch] Collection Completeness Dashboard Design
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-design-collection-completeness-dashboard-show-gaps.md
- [price-watch] Price Comparison UI Patterns for Cognitive Acceleration
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-design-explore-price-comparison-patterns-cognitive.md
- [price-watch] Light Theme Toggle Implementation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-add-light-theme-toggle-user.md
- [price-watch] Add Tooltip for AVG Column Header
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-add-tooltip-avg-column-header.md
- [price-watch] Collection Completeness Dashboard Show Gaps
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-collection-completeness-dashboard-show-gaps.md
- [price-watch] Comparison View N/A Gaps - Data Exists but Outside Current Period
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-comparison-view-gaps-data-exists.md
- [price-watch] Comparison View UI Improvements
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-comparison-view-hide-avg-column.md
- [price-watch] Comparison View N/A Gaps - Date Range Issue
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-comparison-view-na-date-range.md
- [price-watch] Environment Variable Documentation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-create-env-example.md
- [price-watch] Thickness Variants Handling Deep Dive
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-deep-dive-investigation-thickness-variants.md
- [price-watch] Fix Hidden Backlog - Add purge_bullmq_jobs! to mark_completed!
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-fix-hidden-backlog-add-purge.md
- [price-watch] Gradient Colors Broken in Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-gradient-colors-broken-comparison-view.md
- [price-watch] Hidden Backlog - Collection Runs UI Doesn't Show Pending BullMQ Jobs
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-hidden-backlog-collection-runs.md
- [price-watch] Pricing Adjustment Workflow Improvements
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-interactive-session-develop-pricing-adjustment.md
- [price-watch] N/A Gaps in Comparison View for Specific Part/Material Combinations
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-investigate-gaps-comparison-view-specific.md
- [price-watch] Light Theme Heatmap Colors Contrast Issues
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-light-theme-heatmap-colors-contrast.md
- [price-watch] Remove Polling Methods from ScrapeOrchestrator and BullmqClient
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-remove-polling-methods-from-scrapeorchestrator.md
- [price-watch] Deployment Recovery for Collection Runs
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-09-inv-verify-deployment-recovery-collection-runs.md
- [price-watch] Scraper Pricing - Part Price vs Landed Cost
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-10-inv-determine-scraper-captures-landed-cost.md
- [price-watch] Resolve Uncommitted Gradient Color Changes
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-10-inv-resolve-uncommitted-gradient-color-changes.md
- [price-watch] OshCut Scraper - Does it Capture Landed Price?
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-10-inv-verify-oshcut-scraper-captures-landed.md
- [price-watch] OshCut Shipping Policy and Costs
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-10-research-oshcut-shipping-policy-costs-search.md
- [price-watch] Design Investigation: Better Gap Fill Dispatch Mechanism
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-design-better-gap-fill-dispatch.md
- [price-watch] Auto-login Dev User in Development Environment
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-auto-login-dev-user-development.md
- [price-watch] Bug Handling Analysis in Price-Watch
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-bug-handling-analysis-price-watch.md
- [price-watch] Fix Test Isolation - Scraper Tests Leak Jobs to Real BullMQ
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-fix-test-isolation-scraper-tests.md
- [price-watch] Integrate agentlog into Price Watch Scraper
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-integrate-agentlog-into-price-watch.md
- [price-watch] Optimize Scraper Dockerfile for Faster Render Deploys
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-optimize-scraper-dockerfile-faster-render.md
- [price-watch] OshCut Checkout Flow - Complete End-to-End POC
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-oshcut-checkout-flow-complete-end.md
- [price-watch] OshCut Checkout Flow Implementation Requirements
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-oshcut-implement-checkout-flow-scraping.md
- [price-watch] Persona Selection in Gap Fill Production
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-persona-selection-gap-fill-production.md
- [price-watch] Security Audit - Unprotected Routes in Price Watch
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-protect-bullmq-dashboard-audit-other.md
- [price-watch] OshCut Checkout Flow - Shipping Location and Values
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-verify-oshcut-checkout-flow-where.md
- [price-watch] Wire agentlog logError() in Scraper Catch Blocks
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-inv-wire-agentlog-logerror-scraper-catch.md
- [price-watch] BullMQ Admin Dashboard Unprotected in Production
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-simple-bullmq-admin-dashboard-unprotected.md
- [price-watch] Wire logError() into Scraper Catch Blocks
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-11-simple-debug-wire-logerror-scraper-catch.md
- [price-watch] Build POC JSON Endpoint for Facility Metrics
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-build-poc-json-endpoint-facility.md
- [price-watch] Create Test Script to Verify SendCutSend Slave Database Credentials
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-create-test-script-verify-sendcutsend.md
- [price-watch] JSON Schema for /api/pricing/analysis Endpoint
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-define-json-schema-api-pricing.md
- [price-watch] AI-Optimized Data View for Competitor Pricing Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-design-optimized-data-view-competitor.md
- [price-watch] Facility Metrics Report Integration for AI Pricing Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-explore-whether-facility-metrics-report.md
- [price-watch] Measure Slave DB Replication Lag
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-measure-slave-replication-lag.md
- [price-watch] Optimize Facility Metrics Query Performance
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-optimize-facility-metrics-query-performance.md
- [price-watch] Validate Slave DB Connectivity from Price-Watch Rails Console
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-inv-validate-slave-connectivity-from-price.md
- [price-watch] Playwright Traces Not Showing in Production
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-12-simple-playwright-traces-not-showing-production.md
- [price-watch] agentlog errors.jsonl Empty Despite Errors Occurring
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-debug-agentlog-errors-jsonl-empty-despite.md
- [price-watch] Xometry fs/promises Mock Leak into Trace-Tracker Tests
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-debug-fix-test-isolation-xometry-mock.md
- [price-watch] Fix Xometry Test - captureThumbnail /rails Path
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-debug-fix-xometry-test-capturethumbnail-rails-path.md
- [price-watch] Re-scrape SCS button loses state after page refresh
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-debug-scrape-scs-button-loses-state.md
- [price-watch] Add MaterialRevenueService
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-add-materialrevenueservice-query-slave-day.md
- [price-watch] Current Trace Viewing UI in Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-explore-current-trace-viewing-comparison.md
- [price-watch] Expose Volume Weights in Pricing Analysis API
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-expose-volume-weights-pricing-analysis.md
- [price-watch] Fix Dev Dropdown Z-Index Regression
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-fix-dev-dropdown-index-issue.md
- [price-watch] Fix Fabworks Scraper to Capture Shipping Cost
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-fix-fabworks-scraper-capture-shipping.md
- [price-watch] Fix SendCutSend Scraper Production Method Selector Failure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-fix-sendcutsend-scraper-production-method.md
- [price-watch] Fix STEP Metadata Product Name in CadQuery Generator
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-fix-step-metadata-cadquery-generator.md
- [price-watch] Integrate Time Series Data into Pricing Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-integrate-time-series-data-into.md
- [price-watch] Interactive Audit of Fabworks Scraper Implementation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-interactive-audit-fabworks-scraper-implementation.md
- [price-watch] Comparison View Price Display Pattern
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-investigate-comparison-view-price-display.md
- [price-watch] Query Slave DB for Material-Level Volume/Revenue Data
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-query-slave-material-level-volume.md
- [price-watch] Randomize Part Filenames Per Persona
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-randomize-part-filenames-per-persona.md
- [price-watch] [Investigation Title]
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-rewrite-comparison-view-frontend-using.md
- [price-watch] SendCutSend Scraper Production Method Selector Failure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-sendcutsend-scraper-failing-production-method.md
- [price-watch] Trace Coverage - Save At Least One Per Material Per Company
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-trace-coverage-save-least-one.md
- [price-watch] Using facility_metrics Endpoint for Material Prioritization
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-use-facility-metrics-endpoint-get.md
- [price-watch] Wire Pino Logger Hook to Agentlog
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-wire-pino-logger-hook-automatically.md
- [price-watch] Wire Rails and Sidekiq Error Capture to Agentlog
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-15-inv-wire-rails-sidekiq-error-capture.md
- [price-watch] Architecture Audit - Patterns, Coupling, Rails/Node.js Boundary
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-audit-architecture-audit-patterns-coupling-rails.md
- [price-watch] Comparison View Slow Load - N+1 Query Pattern
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-debug-comparison-view-slow-load-query.md
- [price-watch] application.js TypeError - IIFE Misparse
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-debug-error-application-uncaught-typeerror.md
- [price-watch] Fix abandon! to always clean up pending ScrapeJobs
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-debug-fix-abandon-always-clean-pending.md
- [price-watch] Re-scrape Button Not Working in Expanded View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-debug-fix-scrape-button-not-working.md
- [price-watch] Add Retry Logic to purge_bullmq_jobs!
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-add-retry-logic-purge-bullmq.md
- [price-watch] Add Revenue Badges to Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-add-revenue-badges-comparison-view.md
- [price-watch] Comparison Grid Layout Component with Sticky Headers
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-comparison-grid-layout-component-sticky.md
- [price-watch] Symptom Fix Patterns in Scheduled Jobs and Error Handlers
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-deep-dive-symptom-fix-patterns.md
- [price-watch] Design Investigation: Comparison View Re-scrape UX
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-design-proper-comparison-view-rescrape.md
- [price-watch] Explain New Comparison View Summary Metrics
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-explain-new-comparison-view-metrics.md
- [price-watch] Fix Broken Comparison View Summary Metrics
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-fix-broken-comparison-view-summary.md
- [price-watch] [Investigation Title]
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-fix-test-slowness-stale-connections.md
- [price-watch] Historical Price Explorer for SCS Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-historical-price-explorer-scs-comparison.md
- [price-watch] Lead Time Mode Toggle and Display
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-lead-time-mode-toggle-display.md
- [price-watch] Organizational Audit - Code Structure, Naming Conventions, Documentation Coverage
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-organizational-audit-code-structure-naming.md
- [price-watch] Period-over-Period Delta Indicators with Sparklines
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-period-over-period-delta-indicators.md
- [price-watch] Price Cells with Five-Tier Gradient Coloring
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-price-cells-five-tier-gradient.md
- [price-watch] SendCutSend Direct API Quote Feasibility
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-scs-direct-api-quote-feasibility.md
- [price-watch] SendCutSend Scraper File Lookup Fails After Randomized Upload
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-sendcutsend-scraper-file-lookup-fails.md
- [price-watch] [Investigation Title]
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-systematically-investigate-sendcutsend-vue-client.md
- [price-watch] Tests Slow Performance
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-tests-slow-performance-investigate-why.md
- [price-watch] Triage TODO/FIXME Comments Left by Worker Agents
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-triage-todo-fixme-comments-left.md
- [price-watch] Recent Production Errors in Price-Watch
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-16-inv-use-agentlog-investigate-recent-errors.md
- [price-watch] Continue Test Slowness Fix
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-debug-continue-test-slowness-fix-run.md
- [price-watch] Fix Remaining Scraper Test Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-debug-fix-remaining-scraper-test-failures.md
- [price-watch] Gap Fill Job Network Errors and Database Re-seed
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-debug-gap-fill-job-network-errors.md
- [price-watch] Add Scraping Border CSS Animation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-add-scraping-border-css-animation.md
- [price-watch] R2 Bucket Setup and Credentials Configuration
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-bucket-setup-credentials-configuration.md
- [price-watch] Fix Test Mocking and Type Mismatches
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-fix-test-mocking-missing-credentials.md
- [price-watch] Persistent Storage for Playwright Traces
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-implement-persistent-storage-playwright-traces.md
- [price-watch] OshCut Scraper Design Check Timeout Issue
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-investigate-oshcut-scraper-issue-locally.md
- [price-watch] Migrate Playwright Traces from Render Disk to R2
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-migrate-playwright-traces-from-render.md
- [price-watch] Rails Active Storage Integration with Cloudflare R2
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-rails-active-storage-integration-cloudflare.md
- [price-watch] Scraper R2 Integration for Part File Downloads
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-17-inv-scraper-integration-part-file-downloads.md
- [price-watch] [Investigation Title]
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-add-authentication-handling-sveltekit-frontend.md
- [price-watch] Add data-scs-price Attributes (Verification)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-add-data-scs-price-attributes.md
- [price-watch] Add Pending Scrape Counter to Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-add-pending-scrape-counter-comparison.md
- [price-watch] API Endpoints for Materials/Geometries Browser
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-api-endpoints-materials-geometries-browser.md
- [price-watch] Clone Config Modal and Integration with Collection Runs
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-clone-config-modal-integration-collection.md
- [price-watch] Clone Config and Save YAML API Endpoint
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-clone-config-save-yaml-api.md
- [price-watch] Config Editor UI with Material/Geometry Browser
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-config-editor-material-geometry-browser.md
- [price-watch] Config Validation and Preview API Endpoint
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-config-validation-preview-api-endpoint.md
- [price-watch] Light and Dark Theme Design for Price-Watch
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-design-light-dark-themes-price.md
- [price-watch] Design Session: Collection Run Config Creator UI
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-design-session-collection-run-config.md
- [price-watch] Make Comparison View Toolbar More Vertically Compact
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-make-comparison-view-toolbar-more.md
- [price-watch] SvelteKit Config Editor Port from Rails
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-sveltekit-config-editor-port-rails.md
- [price-watch] Test and Polish Multiple Simultaneous Scrape Handling
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-test-polish-multiple-simultaneous-scrape.md
- [price-watch] SCS Re-scrape Comparison Shows Missing Quantities in BEFORE Row
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-18-inv-why-scs-scrape-comparison-shows.md
- [price-watch] Frontend Architecture AI-Native Alignment
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2025-12-20-inv-analyze-frontend-architecture-frontend-against.md
- [price-watch] Codebase Analysis: Anti-Patterns and Technical Debt
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/audits/2025-11-05-codebase-analysis-antipatterns.md
- [price-watch] Project Structure Audit
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/audits/2025-11-06-project-structure-audit.md
- [price-watch] Audit: Collection Run Configs - Thickness-Variant Part Usage
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/audits/2025-12-04-collection-run-config-thickness-audit.md
- [price-watch] Render API CLI Improvements
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/design/2025-12-03-render-api-cli-improvements.md
- [price-watch] Design Investigation: Push-Based Results Architecture
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/design/2025-12-04-push-based-results-architecture.md
- [price-watch] Design Investigation: Stuck Collection Run Management
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/design/2025-12-04-stuck-collection-run-management.md
- [price-watch] Design Investigation: Collection System Mental Model Shift
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/design/2025-12-05-collection-system-mental-model-shift.md
- [price-watch] Design Investigation: Target Config Approach for Comparison View Alignment
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/design/2025-12-05-target-config-approach.md
- [price-watch] Research: SmartProxy Alternatives for Residential Proxies
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/feasibility/2025-11-12-smartproxy-alternatives-residential-proxies.md
- [price-watch] Concurrent Competitor Scraping Feasibility
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/feasibility/2025-11-17-concurrent-competitor-scraping-feasibility.md
- [price-watch] IPRoyal ISP Proxies vs Web Unblocker for Reliable Scraping
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/feasibility/2025-11-18-iproyal-isp-proxies-vs-web-unblocker.md
- [price-watch] Render.com vs Fly.io for Price-Watch Deployment
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/feasibility/2025-11-19-compare-render-com-fly-price-watch-deployment.md
- [price-watch] Feasibility Investigation: CadQuery for Programmatic STEP Generation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/feasibility/2025-11-21-cadquery-programmatic-step-generation.md
- [price-watch] Existing Scraper Implementations Analysis (OshCut & SendCutSend)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-11-25-analyze-existing-scraper-implementations-oshcut.md
- [price-watch] Competitive Model Spreadsheet Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-11-25-competitive-model-spreadsheet-analysis.md
- [price-watch] SendCutSend Competitors - Definitive List
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-11-26-compile-definitive-sendcutsend-competitors-list.md
- [price-watch] Create Xometry Accounts for Existing Personas
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-11-26-create-xometry-accounts-existing-personas.md
- [price-watch] Beads Tracker Issue Relevance Audit
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-audit-beads-tracker-issue-relevance.md
- [price-watch] UI Audit: Price Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-audit-price-comparison-view.md
- [price-watch] Collection Runs Must Survive Deploys (Resumable/Checkpointing)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-collection-runs-survive-deploys-resumable.md
- [price-watch] Fix Fabworks Scraper: Add Thickness Selection for Steel Materials
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-fabworks-scraper-add-thickness-selection.md
- [price-watch] Fix Missing Thumbnails in Price Quote Comparison View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-fix-missing-thumbnails-price-quote.md
- [price-watch] Scraper POC Process Gap Analysis: Fabworks vs Xometry
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-scraper-poc-process-gap-analysis.md
- [price-watch] Xometry Phase 2 POC: Pre-Implementation Validation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-xometry-phase2-poc-validation.md
- [price-watch] Xometry Scraper Implementation Plan
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-01-xometry-scraper-implementation-plan.md
- [price-watch] Browser-Use MCP Server Availability Check
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-02-check-browser-use-mcp-server.md
- [price-watch] Concurrent Collection Run Race Condition
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-02-concurrent-collection-run-race-condition.md
- [price-watch] Run 18 Incomplete Collection Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-02-run-18-incomplete-collection-analysis.md
- [price-watch] Verify Production Login Works
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-02-verify-production-login-works-navigate.md
- [price-watch] Lead Time Toggle Intermittent Failure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-fix-lead-time-toggle-intermittent.md
- [price-watch] Investigate Rails Test Exit Code
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-investigate-rails-test-exit-code.md
- [price-watch] Normalize SCS Material Availability Semantics
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-normalize-scs-material-availability-semantics.md
- [price-watch] Part Metadata Missing in Production (Recurring)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-part-metadata-production-missing.md
- [price-watch] Part Metadata (Size, Complexity) Missing in Production
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-part-metadata-size-complexity-missing.md
- [price-watch] Persona Credential Migration (Env Vars → Seed Data)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-persona-credential-migration.md
- [price-watch] Production Tooltip Floating (Tippy.js Asset Cache)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-production-tooltip-floating-tippy-asset-cache.md
- [price-watch] Render Deploy Notification Approach
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-03-render-deploy-notification-approach.md
- [price-watch] Auto-Resolve Thickness-Variant Parts in Config Processing
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-04-auto-resolve-thickness-variants.md
- [price-watch] Collection Run Reliability Issues
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-04-collection-run-resume-issues.md
- [price-watch] Fix Failing Rails Tests Blocking
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-04-fix-failing-rails-tests-blocking.md
- [price-watch] Orphaned Sidekiq Jobs Bug
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-04-investigate-orphaned-sidekiq-jobs-bug.md
- [price-watch] OshCut Material Thickness Mapping Near-Matches
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-04-investigate-oshcut-material-thickness-mapping.md
- [price-watch] ⚠️ SUPERSEDED - DO NOT USE
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-04-investigate-oshcut-pricing-scraped-prices.md
- [price-watch] OshCut Config Update Implementation Strategy
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-04-oshcut-config-update-implementation.md
- [price-watch] Comparison View OshCut Quote Requirements
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-05-comparison-view-oshcut-quote-requirements.md
- [price-watch] SendCutSend Scraper File Upload Error
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-05-fix-sendcutsend-scraper-file-upload.md
- [price-watch] Simplify Retry Mechanism
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-05-simplify-retry-mechanism.md
- [price-watch] HRP-188 SCS Price Tweak Verification
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-08-hrp-188-scs-price-tweak-verification.md
- [price-watch] SendCutSend Pricing System Overview
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/simple/2025-12-08-scs-pricing-system-overview.md
- [price-watch] Price Watch Status Investigation - Tuesday Meeting Prep
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-03-price-watch-status-for-meeting.md
- [price-watch] Vertical Slice Completion Criteria for OshCut
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-03-vertical-slice-completion-criteria.md
- [price-watch] Heat Map Analysis Investigation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-04-heat-map-analysis/README.md
- [price-watch] Heat Map Analysis - Jim's Preferred View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-04-heat-map-analysis/investigation.md
- [price-watch] Kenneth Setup Guide Validation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-04-kenneth-setup-validation.md
- [price-watch] Seed Script CSV Handling Behavior
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-04-seed-script-csv-handling.md
- [price-watch] Phase 1.5 Final Demo Validation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-05-phase-1-5-final-validation.md
- [price-watch] Real Setup Validation - Runtime Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-05-real-setup-validation.md
- [price-watch] Database Reset After docker compose down
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-06-database-reset-investigation.md
- [price-watch] Docker Context Violation by yaml-configs-continue Agent
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-06-docker-context-violation.md
- [price-watch] Setup Script Validation Investigation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-06-setup-script-validation.md
- [price-watch] Cleanup and Refactoring Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-07-cleanup-refactoring-analysis.md
- [price-watch] Proxy Connection Issues
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-09-investigate-fix-proxy-connection-issues.md
- [price-watch] Fix Error Classification Logic in Base Scraper
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-fix-error-classification-logic-base.md
- [price-watch] Fix Heat Map Bug - Multiple Materials Display
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-fix-heat-map-bug-display.md
- [price-watch] Fix Heat Map Pending Cells
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-fix-heat-map-pending-cells.md
- [price-watch] OshCut Scraper Login Failure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-fix-oshcut-scraper-login-failure.md
- [price-watch] Recurring Proxy Connection Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-investigate-recurring-proxy-connection-failures.md
- [price-watch] Latest Collection Run Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-latest-collection-run-analysis.md
- [price-watch] Setup Script Updates After Recent Changes
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-setup-script-updates-after-recent-changes.md
- [price-watch] OshCut Deburring Issue Verification
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-verify-oshcut-deburring-issue-still.md
- [price-watch] Why Dark Theme and Bulma Styles Not Showing on Heat Map
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-10-why-dark-theme-bulma-styles.md
- [price-watch] Agent Used Wrong Worktree Docker Environment
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-agent-used-wrong-worktree-docker.md
- [price-watch] Collection Run #10 Failure Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-analyze-failures-collection-run-10.md
- [price-watch] Complete BullMQ Retry Verification
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-complete-bullmq-retry-verification-rebuild.md
- [price-watch] direnv Error - .envrc is Blocked
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-direnv-error-envrc-blocked.md
- [price-watch] Price Watch Project Structure for Agent Coordination
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-explore-price-watch-project-structure.md
- [price-watch] BullMQ Retry Bug - Transient Errors Not Retrying
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-fix-bullmq-retry-bug-transient.md
- [price-watch] Fix Test Suite - Scraper Service Not Starting
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-fix-test-suite-price-watch.md
- [price-watch] Fix Turbo Streams ActionCable Issues
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-fix-turbo-streams-actioncable-issues.md
- [price-watch] Git Workflow Confusion - Worktrees and Branches
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-git-workflow-confusion-worktrees.md
- [price-watch] Heat Map Still Showing Generic Error Messages
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-heat-map-still-showing-generic.md
- [price-watch] Heat Map Timeout Error (collection_run_id=9)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-investigate-timeout-error-heat-map.md
- [price-watch] make logs and docker compose logs -f Produce No Output
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-make-logs-docker-compose-logs.md
- [price-watch] Collection Runs Complete Immediately
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-new-collection-runs-immediately-complete.md
- [price-watch] Quote Failures from Collection Run 2
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-quote-failures-collection-run-21.md
- [price-watch] Quote Failure Reasons Not Displayed on Heat Map
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-quote-failures-collection-run-show.md
- [price-watch] Verify Turbo Streams Auto-Update for Success Rate Column
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-11-verify-turbo-streams-auto-update.md
- [price-watch] Collection Run #22 Failure Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-12-collection-run-22-failure-analysis.md
- [price-watch] Research: Debug-with-Playwright Skill vs New Playwright Tracing
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-12-debug-playwright-skill-vs-mcp-tools.md
- [price-watch] Fix Failing PersonaTest Cooldown Tests
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-12-fix-failing-personatest-cooldown-tests.md
- [price-watch] Playwright Debugging Features in Price Watch
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-12-playwright-debugging-features.md
- [price-watch] Playwright Debugging Options for Dockerized Scrapers
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-12-playwright-debugging-options-dockerized-scrapers.md
- [price-watch] Proxy Reliability History
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-12-proxy-reliability-history.md
- [price-watch] Transient WAF Blocking During OshCut Scraping
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-12-transient-waf-blocking-analysis.md
- [price-watch] Debug Collection Run 42 Failure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-debug-collection-run-failure.md
- [price-watch] Fix Login Failure - User Menu Did Not Appear
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-fix-login-failure-user-menu.md
- [price-watch] Fix OshCut Sign In Button Timeout
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-fix-oshcut-sign-button-timeout.md
- [price-watch] Fix Success Rate Calculation to Exclude Unsupported Materials
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-fix-success-rate-calculation-exclude.md
- [price-watch] Collection Run 45 Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-investigate-collection-run-failures.md
- [price-watch] Existing Scrape Count Tracking and Metrics by Persona/Profile
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-investigate-existing-scrape-count-tracking.md
- [price-watch] Collection Run 41 Proxy Health Check Failure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-investigate-fix-collection-run-failure.md
- [price-watch] Heat Map Retry Results Display Gap
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-investigate-heat-map-retry-results.md
- [price-watch] JavaScript Polling Issues
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-13-js-polling-issues.md
- [price-watch] SendCutSend File Upload Timeout
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-14-fix-scs-scraper-file-upload.md
- [price-watch] Fix SendCutSend Scraper Reliability (9% Success Rate)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-14-fix-sendcutsend-scraper-reliability-success.md
- [price-watch] Collection Run 49 Failure (HTTP 403 Proxy Blocking)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-14-investigate-collection-run-failure-http.md
- [price-watch] Collection Run 47 Timeout Failure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-14-investigate-fix-timeout-failure-collection.md
- [price-watch] Heat Map Display Issue - Multi-Competitor Collection Run
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-14-investigate-heat-map-display-issue.md
- [price-watch] NULL Price Handling for Unavailable SCS Quantities
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-14-investigate-null-price-handling-unavailable.md
- [price-watch] Material Mapping Expansion Strategy (48% → 70%+)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-14-material-mapping-expansion-strategy.md
- [price-watch] SendCutSend Authentication Selector Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-analyze-playwright-trace-identify-correct.md
- [price-watch] Playwright Trace Analysis for OshCut Cold Start Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-analyze-playwright-traces-from-oshcut.md
- [price-watch] Comprehensive Progress Analysis Since Nov 11
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-comprehensive-progress-analysis-since-nov.md
- [price-watch] Collection Runs Index Page - Missing Companies and Config File Data
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-fix-collection-runs-index-page.md
- [price-watch] Fix Iteration 7 Edge Case - 30s Timeout Insufficient
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-fix-iteration-edge-case-where.md
- [price-watch] NoMethodError for automatic_retry_count in Heat Map View
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-fix-nomethoderror-undefined-method-automatic.md
- [price-watch] Collection Run #106 Failure - OshCut Design Check Timeout
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-investigate-collection-run-106-failure.md
- [price-watch] SendCutSend Cloudflare WAF Blocking Public Quotes
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-investigate-sendcutsend-cloudflare-blocking-public.md
- [price-watch] Collection Run #108 Systemic Scraper Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-investigate-systemic-scraper-failures-collection.md
- [price-watch] Why SendCutSend Cookie Clearing Only Works for First Batch
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-investigate-why-sendcutsend-cookie-clearing.md
- [price-watch] IPRoyal Proxy Degradation Blocking SendCutSend Testing
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-iproyal-proxy-degradation-blocking-sendcutsend.md
- [price-watch] OshCut Cold Start Timeout
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-oshcut-cold-start-timeout.md
- [price-watch] SendCutSend Comprehensive Workflow Validation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-sendcutsend-poc-validation-comprehensive-workflow.md
- [price-watch] SendCutSend waitForURL 'load' Event Timeout (100% Failure)
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-sendcutsend-waitforurl-load-event-timeout.md
- [price-watch] SendCutSend Guest vs Authenticated Workflow in Automated Scraper
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-17-test-sendcutsend-guest-workflow-automated.md
- [price-watch] Aggregated Dashboard Production Readiness
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-18-analyze-current-aggregated-dashboard-implementation.md
- [price-watch] Jim's Original Heat Map Design Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-18-analyze-jim-original-heat-map-design.md
- [price-watch] Debug and Fix Comparison View Phase A Issues
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-18-debug-fix-comparison-view-phase.md
- [price-watch] Fix SendCutSend Overlay Interception Causing Click Timeouts
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-18-fix-sendcutsend-overlay-interception-causing.md
- [price-watch] Normalize SendCutSend Material Availability Semantics
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-18-normalize-sendcutsend-material-availability-semantics.md
- [price-watch] SendCutSend Login Redirect - Expects /customer, Gets /parts
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-18-sendcutsend-login-redirect-expects-customer-gets-parts.md
- [price-watch] SendCutSend Scraper Reliability Validation After Recent Fixes
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-18-validate-sendcutsend-scraper-reliability-across.md
- [price-watch] Kenneth's Example Parts Integration Strategy
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-analyze-kenneth-example-parts-integration.md
- [price-watch] Devise Sign-In Page Showing Blank Page
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-debug-why-devise-sign-page.md
- [price-watch] Devise Authentication Current State
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-devise-authentication-current-state.md
- [price-watch] Fix Failing BaseScraper Tests
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-fix-failing-scraper-tests-base.md
- [price-watch] Fix Failing SendCutSend Scraper Tests
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-fix-failing-sendcutsend-scraper-tests.md
- [price-watch] How Styling is Currently Handled Across All Views
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-investigate-how-styling-currently-handled.md
- [price-watch] Unexpected Material Unavailable Results in Collection Run 3
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-investigate-unexpected-material-unavailable-results.md
- [price-watch] Kenneth's Example Parts - URLs and Metadata
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-kenneth-parts-cdn-urls.md
- [price-watch] User Seed Data Structure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-user-seed-data-structure.md
- [price-watch] Validate SendCutSend Material Availability Fix
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-19-validate-sendcutsend-material-availability-fix.md
- [price-watch] Comprehensive Architecture Understanding
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-comprehensive-architecture-understanding.md
- [price-watch] Scraper Development Workflow - When Rebuild vs Restart vs Hot-Reload
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-determine-when-rebuild-restart-hot.md
- [price-watch] Price Watch .orch/ Structure and Taxonomy Migration
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-examine-price-watch-orch-structure.md
- [price-watch] Collection Run #3 Material Mapping Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-fix-missing-material-mappings-top.md
- [price-watch] Collection Run #8 - Duplicate Quote Creation and Incomplete Job Assignment
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-investigate-collection-run-incomplete-job.md
- [price-watch] Collection Run 5 Login Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-investigate-collection-run-login-failures.md
- [price-watch] SendCutSend Dimension Timeout Failures in Collection Run #8
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-investigate-sendcutsend-dimension-timeout-failures.md
- [price-watch] Material Mapping Validation Structure
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-20-material-mapping-validation-structure.md
- [price-watch] Fix Duplicate Key Constraint Violations in price_watch_dev_test.rb
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-fix-duplicate-key-constraint-violations.md
- [price-watch] 100% Material Unavailable for OshCut in Collection Run #11
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-investigate-100-material-unavailable-oshcut.md
- [price-watch] Manual Verification of OshCut N/A Entries from Collection Run 17
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-manually-verify-oshcut-na-entries-collection-run-17.md
- [price-watch] OshCut `/api/materials` Endpoint Analysis
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-oshcut-api-materials-endpoint.md
- [price-watch] OshCut Model Thickness Matching Requirement
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-oshcut-model-thickness-matching-requirement.md
- [price-watch] Synthesis: OshCut Material Availability Constraints
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-synthesis-oshcut-material-availability-constraints.md
- [price-watch] Context Loading Verification
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-test-context-loading-verification.md
- [price-watch] Price Watch Context Loading Verification
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-21-verify-price-watch-context-loading.md
- [price-watch] Why Proxy Timeouts Keep Recurring Despite Multiple Fixes
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-22-why-proxy-timeouts-keep-recurring.md
- [price-watch] BullMQ attemptsMade Not Incrementing
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-24-investigate-bullmq-attemptsmade-not-incrementing.md
- [price-watch] Collection Run #18 Failures
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-24-investigate-fix-collection-run-failures.md
- [price-watch] Why Resolution-Status Field Missing from Investigation File
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-24-investigate-why-resolution-status-field.md
- [price-watch] Heat Map Polling Not Starting on Turbo Navigation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-11-26-fix-heat-map-polling-turbo-navigation.md
- [price-watch] New Collection Run Button Not Working After Turbo Navigation
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-12-04-bug-new-collection-run-button.md
- [price-watch] Fix Polling Controller Overwriting Categorized
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-12-04-fix-polling-controller-overwriting-categorized.md
- [price-watch] BullMQ Job Deduplication Blocking Re-dispatch
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-12-05-fix-bullmq-job-deduplication-blocking.md
- [price-watch] [Investigation Title]
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-12-05-fix-sendcutsend-scraper-file-upload.md
- [price-watch] Handle Invalid Collection Run ID Gracefully
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/systems/2025-12-05-handle-invalid-collection-run-gracefully.md
- [blog] Research: Jesse Vincent's Blog - AI Agent Writing Patterns
  - See: /Users/dylanconlin/Documents/personal/blog/.kb/investigations/2025-12-19-inv-research-jesse-vincent-blog-ai-agents.md
- [blog] Steve Yegge's AI Agent Writing Patterns
  - See: /Users/dylanconlin/Documents/personal/blog/.kb/investigations/2025-12-19-inv-research-steve-yegge-blog-ai-agents.md
- [blog] Research: Anthropic's Public Writing on AI Agent Patterns
  - See: /Users/dylanconlin/Documents/personal/blog/.kb/investigations/2025-12-19-research-anthropic-public-writing-agent-patterns.md
- [scs-slack] Deep Pattern Analysis of Slack Exports for CNC/Machining Channels
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/scs-slack/.kb/investigations/2025-12-19-inv-deep-pattern-analysis-slack-exports.md
- [scs-slack] Design toolshed Unified Services Architecture
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/scs-slack/.kb/investigations/2025-12-19-inv-design-scs-platform-unified-services.md
- [scs-slack] Phil's Monthly Slack Export Delivery Method
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/scs-slack/.kb/investigations/2025-12-19-inv-phil-needs-get-monthly-slack.md
- [scs-slack] Research: Fly.io vs AWS for SvelteKit + Bun Deployment
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/scs-slack/.kb/investigations/2025-12-19-research-flyio-vs-aws-sveltekit-bun-deployment.md
- [scs-slack] Research: SvelteKit as Full-Stack Framework for Internal Tools
  - See: /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/scs-slack/.kb/investigations/2025-12-19-research-sveltekit-fullstack-internal-tools.md
- [beads-ui] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/beads-ui/.kb/investigations/2025-12-06-redesign-list-view-two-line.md
- [beads-ui] Vim Navigation j/k Opens Popups Instead of Highlighting
  - See: /Users/dylanconlin/Documents/personal/beads-ui/.kb/investigations/2025-12-08-debug-fix-vim-navigation-highlight-list.md
- [beads-ui] Vim Navigation Opens Popup Instead of Highlighting
  - See: /Users/dylanconlin/Documents/personal/beads-ui/.kb/investigations/simple/2025-12-08-fix-vim-navigation-highlight-list.md
- [beads-ui] Building a New Beads UI with Cutting-Edge Tech
  - See: /Users/dylanconlin/Documents/personal/beads-ui/.kb/investigations/simple/2025-12-09-design-analyze-rewriting-beads-svelte.md
- [orch-go] CLI orch complete Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-complete-command.md
- [orch-go] CLI orch spawn Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md
- [orch-go] CLI Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-status-command.md
- [orch-go] OpenCode Client Package Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-opencode-session-management.md
- [orch-go] SSE Event Monitoring Client
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md
- [orch-go] Fix comment ID parsing - Comment.ID type mismatch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-fix-comment-id-parsing-comment.md
- [orch-go] Fix SSE parsing - event type inside JSON data
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-fix-sse-parsing-event-type.md
- [orch-go] Set beads issue status to in_progress on spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-set-beads-issue-status-progress.md
- [orch-go] Update README with current CLI commands and usage
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-update-readme-current-cli-commands.md
- [orch-go] Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- [orch-go] Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- [orch-go] Synthesis Protocol Design for Agent Handoffs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-capacity-manager-multi-account.md
- [orch-go] Add --dry-run flag to daemon run command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-dry-run-flag-daemon.md
- [orch-go] Add Missing Spawn Flags
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-missing-spawn-flags-no.md
- [orch-go] Add orch review command for batch completion workflow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-orch-review-command-batch.md
- [orch-go] Add Wait Command to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-wait-command-orch.md
- [orch-go] Automate Knowledge Sync using Cobra Doc Gen
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-automate-knowledge-sync-using-cobra.md
- [orch-go] KB Search vs Grep Benchmark
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md
- [orch-go] Beta Flash Synthesis Protocol Design v3
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-beta-flash-synthesis-protocol-design.md
- [orch-go] Compare orch-cli (Python) vs orch-go Features
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-design-synthesis-protocol-goal-create.md
- [orch-go] Enhance orch review to parse and display SYNTHESIS.md
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-orch-review-parse-display.md
- [orch-go] Enhance status command with swarm progress
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-status-command-swarm-progress.md
- [orch-go] Expose Strategic Alignment Commands
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-expose-strategic-alignment-commands-focus.md
- [orch-go] Finalize Native Implementation for orch send
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-finalize-native-implementation-orch-send.md
- [orch-go] Fix bd create output parsing
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-fix-bd-create-output-parsing.md
- [orch-go] SSE-Based Completion Detection and Notifications
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-sse-based-completion-detection.md
- [orch-go] Make Headless Mode Default for Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-make-headless-mode-default-deprecate.md
- [orch-go] Migrate orch-go from tmux to HTTP API
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-migrate-orch-go-tmux-http.md
- [orch-go] Add Abandon Command to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-abandon-command.md
- [orch-go] Agent Registry for Persistent Tracking
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-agent-registry-persistent.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-clean-command.md
- [orch-go] Daemon Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-daemon-command.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-focus-drift-next.md
- [orch-go] orch-go Add Question Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-question-command.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-resume-command.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-wait-command.md
- [orch-go] Final Sanity Check of orch-go Commands
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-perform-final-sanity-check-orch.md
- [orch-go] Refactoring pkg/registry as Beads Issue State Cache
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-plan-refactoring-pkg-registry-act.md
- [orch-go] POC Port Python Standalone + API Discovery to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-poc-port-python-standalone-api.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-port-comparison.md
- [orch-go] Recursive Research Loop Persona Confusion
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-recursive-research-loop-persona-confusion.md
- [orch-go] Research: Claude 4.5 and Claude Max Pricing (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-research-claude-claude-max-pricing.md
- [orch-go] Scope Out Headless Swarm Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md
- [orch-go] Concurrent tmux spawn test (delta)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-tmux-concurrent-delta.md
- [orch-go] Tmux Concurrent Epsilon Spawn Capability
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-tmux-concurrent-epsilon.md
- [orch-go] tmux concurrent zeta - 6th concurrent spawn test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-tmux-concurrent-zeta.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-wire-beads-ui-v2-orch.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-research-claude-models-late-2025.md
- [orch-go] Model Arbitrage and API vs Max Math (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-research-model-arbitrage-api-vs-max.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-research-model-arbitrage-services.md
- [orch-go] OpenCode Native Context Loading
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-research-opencode-native-context-loading.md
- [orch-go] orch spawn vs Native OpenCode Agents
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-research-orch-spawn-vs-opencode-agents.md
- [orch-go] orch send fails silently for tmux-based agents
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-debug-orch-send-fails-silently-tmux.md
- [orch-go] Deep Pattern Analysis Across Orchestration Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md
- [orch-go] Design: kb reflect Command Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md
- [orch-go] Design: Minimal Artifact Set Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-minimal-artifact-taxonomy.md
- [orch-go] Add Concurrency Limit to orch spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-add-concurrency-limit-orch-spawn.md
- [orch-go] Add tmux fallback for orch status and tail
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-add-tmux-fallback-orch-status.md
- [orch-go] Add --tmux flag to orch spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-add-tmux-flag-orch-spawn.md
- [orch-go] Agents Being Marked Completed in Registry Prematurely
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md
- [orch-go] Agents skip SYNTHESIS.md creation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-agents-skip-synthesis-md-creation.md
- [orch-go] Registry Usage Audit in orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md
- [orch-go] Beads ↔ KB ↔ Workspace Relationship Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-beads-kb-workspace-relationships-how.md
- [orch-go] Beads OSS Relationship - Fork vs Contribute vs Local Patches
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-beads-oss-relationship-fork-vs.md
- [orch-go] Chronicle Artifact Type Design
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md
- [orch-go] Daemon and Hook Integration for kb reflect
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-daemon-hook-integration-kb-reflect.md
- [orch-go] Dashboard Agent Activity Visibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-dashboard-needs-better-agent-activity.md
- [orch-go] Deep Dive into Inter-Agent Communication Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-deep-dive-inter-agent-communication.md
- [orch-go] Deep Post-Mortem on 24 Hours of Development Chaos
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md
- [orch-go] Design: Self-Reflection Protocol Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md
- [orch-go] Single-Agent Review Command Design
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-design-single-agent-review-command.md
- [orch-go] Enhance orch clean with four-layer reconciliation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-enhance-orch-clean-four-layer.md
- [orch-go] Enhance Tmuxinator Config Generation with Port Registry
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-enhance-tmuxinator-config-generation-use.md
- [orch-go] Failure Mode Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md
- [orch-go] Fix BuildSpawnCommand to Pass Model Flag
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-fix-buildspawncommand-pass-model-flag.md
- [orch-go] Fix OAuth Token Revocation in GetAccountCapacity
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-fix-oauth-token-revocation-getaccountcapacity.md
- [orch-go] Fix Session ID Capture Timing in Tmux Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-fix-session-id-capture-timing.md
- [orch-go] Headless Spawn Not Sending Prompts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-headless-spawn-not-sending-prompts.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-implement-failure-report-md-template.md
- [orch-go] Implement orch complete --preview and orchestrator skill update
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-implement-orch-complete-preview-update.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-implement-orch-init-command-project.md
- [orch-go] Port Allocation Registry Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-implement-port-allocation-registry-orch.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-implement-session-handoff-md-template.md
- [orch-go] [Investigation Title]
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-investigate-orch-status-showing-stale.md
- [orch-go] Knowledge Promotion Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md
- [orch-go] Model Handling Conflicts Between orch-go and opencode
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md
- [orch-go] Multi-Agent Synthesis and Conflict Detection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-multi-agent-synthesis-when-multiple.md
- [orch-go] orch complete registry status update
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-orch-complete-closes-beads-issue.md
- [orch-go] Orchestrator Session Boundaries
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md
- [orch-go] Phase 3 - Evaluate spawn session_id capture without registry
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md
- [orch-go] Questioning Inherited Constraints
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-questioning-inherited-constraints-when-how.md
- [orch-go] Reconciliation Should Check Completed Work
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-reconciliation-should-check-completed-work.md
- [orch-go] Reflection Checkpoint Pattern for Agent Sessions
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md
- [orch-go] Registry Abandon Doesn't Remove Agent Entry
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-registry-abandon-doesn-remove-agent.md
- [orch-go] orch init and Project Standardization
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-scope-orch-init-project-standardization.md
- [orch-go] Temporal Signals for Autonomous Reflection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-temporal-signals-autonomous-reflection.md
- [orch-go] orch spawn --tmux getting SIGKILL
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-tmux-spawn-killed.md
- [orch-go] Trace Evolution from orch-cli (Python) to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-trace-evolution-orch-cli-python.md
- [orch-go] Workspace Lifecycle in orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md
- [orch-go] Synthesis: Registry Evolution and Orch Identity
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md
- [orch-go] 40+ Agents Showing as Active in orch status
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md
- [orch-go] Add Liveness Warning to orch complete
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-add-liveness-warning-orch-complete.md
- [orch-go] Audit Orchestration Lifecycle Post-Registry Removal
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md
- [orch-go] Template System Fragmentation Deep Dive
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md
- [orch-go] Knowledge System Support for Project Extraction and Refactoring
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-design-knowledge-system-project-extraction.md
- [orch-go] Replace orch-knowledge with skillc for Skill Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-epic-replace-orch-knowledge-skillc.md
- [orch-go] Fix Pre-Spawn KB Context Check
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-fix-pre-spawn-kb-context.md
- [orch-go] Implement IsLive Function for Agent Liveness Detection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-implement-islive-beadsid-string-function.md
- [orch-go] Skillc vs Orch-Knowledge Skill Build Pipeline
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md
- [orch-go] Spawn Agent with Tmux Flow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-spawn-agent-tmux.md
- [orch-go] Spawn Context Generation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-test-spawn-context.md
- [orch-go] Test Spawn to Verify Pre-Spawn KB Context Check
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-test-spawn-verify-pre-spawn.md
- [orch-go] Tracing Confidence Score History and Effectiveness
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-trace-confidence-score-effectiveness.md
- [orch-go] Update orch status to use IsLive() liveness checks
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-update-orch-status-use-islive.md
- [skill-benchmark] Build Benchmark Harness and Automation Tooling
  - See: /Users/dylanconlin/Documents/personal/skill-benchmark/.kb/investigations/2025-12-06-phase-build-benchmark-harness-automation.md
- [skill-benchmark] Phase 6 - Build Orchestrator Benchmark Scenarios
  - See: /Users/dylanconlin/Documents/personal/skill-benchmark/.kb/investigations/2025-12-06-phase-implementation-build-orchestrator-benchmark.md
- [skill-benchmark] Design Investigation: Orchestrator Benchmark Scenarios
  - See: /Users/dylanconlin/Documents/personal/skill-benchmark/.kb/investigations/design/2025-12-06-orchestrator-benchmark-scenarios.md
- [skillc] AI Context Artifact Patterns
  - See: /Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-21-inv-ai-context-artifact-patterns.md
- [skillc] Define Self-Describing Artifacts Principle
  - See: /Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-21-inv-define-self-describing-artifacts-principle.md
- [skillc] How skillc should handle @import syntax
  - See: /Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-21-inv-explore-how-skillc-should-handle.md
- [skillc] Implement skillc in Go as standalone context compiler
  - See: /Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-21-inv-implement-skillc-go-as-standalone.md
- [skillc] Skillc Multi-Level Context Model
  - See: /Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-21-inv-skillc-multi-level-context-model.md
- [skillc] Why did feature-impl agent (skillc-rys) skip SYNTHESIS.md creation?
  - See: /Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-21-inv-why-did-feature-impl-agent.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.



🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-gcf8 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-gcf8 "Phase: Complete - [1-2 sentence summary of deliverables]"`
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

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation pre-spawn-kb-context-check` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-pre-spawn-kb-context-check.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-gcf8 "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-pre-spawn-kb-22dec/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-gcf8**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-gcf8 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-gcf8 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-gcf8 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-gcf8 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-gcf8 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-gcf8`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## SKILL GUIDANCE (investigation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: investigation
skill-type: procedure
audience: worker
spawnable: true
category: investigation
description: Record what you tried, what you observed, and whether you tested. Key discipline - you cannot conclude without testing.
parameters:
- name: type
  description: Investigation type (simple is the only recommended type)
  type: string
  required: false
  default: simple
allowed-tools:
- Read
- Glob
- Grep
- Bash
- Write
- Edit
deliverables:
- type: investigation
  path: "{project}/.kb/investigations/{date}-inv-{slug}.md"
  required: true
verification:
  requirements: |
    - [ ] Test performed section filled (not "None" or "N/A")
    - [ ] Conclusion based on test results (not speculation)
    - [ ] Investigation file committed
  test_command: null
  required: true
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
2. Fill in your question
3. Try things, observe what happens
4. **Run a test to validate your hypothesis**
5. Fill conclusion only if you tested
6. Commit

## D.E.K.N. Summary

**Every investigation file starts with a D.E.K.N. summary block at the top.** This enables 30-second handoff to fresh Claude.

| Section | Purpose | Example |
|---------|---------|---------|
| **Delta** | What was discovered/answered | "Test-running guidance is missing from spawn prompts" |
| **Evidence** | Primary evidence supporting conclusion | "Searched 5 agent sessions - none ran tests" |
| **Knowledge** | What was learned (insights, constraints) | "Agents follow documentation literally" |
| **Next** | Recommended action | "Add test-running instruction to template" |

**Fill D.E.K.N. at the END of your investigation, before marking Complete.**

## Template

The template enforces the discipline. Use `kb create investigation {slug}` to create.

```markdown
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
```

**Only proceed to commit after self-review passes.**

---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kn decide` | `kn decide "Use Redis for sessions" --reason "Need distributed state"` |
| Tried something that failed | `kn tried` | `kn tried "SQLite for sessions" --failed "Race conditions"` |
| Discovered a constraint | `kn constrain` | `kn constrain "API requires idempotency" --reason "Retry logic"` |
| Found an open question | `kn question` | `kn question "Should we rate-limit per-user or per-IP?"` |

**Quick checklist:**
- [ ] Reflected on session: What did I learn that the next agent should know?
- [ ] Externalized at least one item via `kn` command

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."

---

## Completion

Before marking complete:

1. Self-review passed (see above)
2. **Leave it Better:** At least one `kn` command run OR noted as not applicable
3. `## Test performed` has a real test (not "reviewed code" or "analyzed logic")
4. `## Conclusion` is based on test results
5. D.E.K.N. summary filled (Delta, Evidence, Knowledge, Next)
6. `git add` and `git commit` the investigation file
7. Link artifact to beads issue: `kb link <investigation-file> --issue <beads-id>`
8. Report via beads: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"`
9. Close the beads issue: `bd close <beads-id> --reason "conclusion summary"`
10. Run `/exit` to close session

---

**Remember:** The old investigation system produced confident wrong conclusions. The fix is simple: test before concluding.


---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-gcf8 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`

⚠️ Your work is NOT complete until you run both commands.
