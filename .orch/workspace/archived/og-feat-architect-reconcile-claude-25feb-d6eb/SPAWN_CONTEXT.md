TASK: Architect: reconcile Claude Code auto-memory with kb-cli knowledge externalization

Claude Code has a new 'auto memory' feature (MEMORY.md at ~/.claude/projects/<project>/memory/) where the agent writes self-notes that persist across sessions. We already have kb-cli for knowledge externalization (decisions, constraints, tried/failed, investigations, guides, probes). These overlap — both store learned patterns and preferences that persist across sessions. An architect should decide: (1) Should auto-memory be disabled entirely in favor of kb-cli? (2) Should each have a distinct lane (e.g., auto-memory for session-ephemeral context recovery, kb-cli for durable cross-agent knowledge)? (3) Does auto-memory's 200-line startup injection compete with or complement CLAUDE.md instructions? (4) Risk: auto-memory is per-user and invisible to spawned agents — kb-cli artifacts are discoverable by all agents via kb context. Is that a feature or a bug? Reference: the Knowledge Placement table in ~/.claude/CLAUDE.md already defines where different knowledge types go, but doesn't account for auto-memory.


SPAWN TIER: light

⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.



## CONFIG RESOLUTION

- Backend: claude (source: derived (model-requirement))
- Model: anthropic/claude-opus-4-5-20251101 (source: cli-flag)
- Tier: light (source: heuristic (skill-default))
- Spawn Mode: tmux (source: derived (claude-backend-requires-tmux))
- MCP: none (source: default)
- Mode: tdd (source: default)
- Validation: tests (source: default)
- Account: personal (source: heuristic (spillover-activated-5h:86%-7d:90%))




## PRIOR KNOWLEDGE (from kb context)

**Query:** "architect reconcile claude"

### Constraints (MUST respect)
- Stall detection: session.status=busy for >5min without message.part events indicates hung Claude API call
  - Reason: SSE monitoring pattern - healthy sessions emit regular message.part.updated events
- Ask 'should we' before 'how do we' for strategic direction changes
  - Reason: Epic orch-go-erdw was created assuming skills-as-value was correct direction. Architect review revealed the premise was wrong - current separation is intentional design. Wasted work avoided by validating premise before execution.

### Prior Decisions
- Opus default, Gemini escape hatch
  - Reason: 2 Claude Max subscriptions (covered), Gemini is pay-per-token + tier 2 TPM. Escalation: 1) Opus default, 2) account switch, 3) --model flash
- D.E.K.N. is universal handoff structure
  - Reason: Delta/Evidence/Knowledge/Next enables 30-second context transfer between Claude instances - proven across SYNTHESIS.md, investigations, and session handoffs
- Skills own domain behavior, spawn owns orchestration infrastructure
  - Reason: Architect review found current separation is correct design, not fragmentation. Skills containing beads/phase logic would reduce portability and violate Compose Over Monolith. See .kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md
- SHOULD-HOW-EXECUTE sequence for strategic questions
  - Reason: Epic orch-go-erdw was created from 'how do we' without testing premise. Architect found premise wrong. Wasted work avoided if 'should we' asked first.
- Abandon Claude Max OAuth, Use Gemini Flash as Primary Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md

### Models (synthesized understanding)
- Probe: Claude Code Plan Mode vs Feature-Impl Phase Model Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md
  - Recent Probes:
    - 2026-02-24-probe-orch-complete-kills-wrong-tmux-window
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-orch-complete-kills-wrong-tmux-window.md
    - 2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md
    - 2026-02-24-probe-tmux-liveness-two-lane-violation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md
    - 2026-02-24-probe-claude-spawn-dashboard-visibility-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md
    - 2026-02-20-tradeoff-visibility-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-20-tradeoff-visibility-gap-analysis.md
- Probe: Automatic Account Distribution for Claude CLI Spawns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics/probes/2026-02-24-probe-automatic-account-distribution-design.md
  - Recent Probes:
    - 2026-02-24-probe-automatic-account-distribution-design
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics/probes/2026-02-24-probe-automatic-account-distribution-design.md
    - 2026-02-20-model-drift-stale-references-audit
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics/probes/2026-02-20-model-drift-stale-references-audit.md
- Probe: Daemon spawn bypasses user config default_model and Claude spawn has no session tracking
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility.md
  - Recent Probes:
    - 2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility.md
    - 2026-02-21-probe-gpt-model-spawn-e2e-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-21-probe-gpt-model-spawn-e2e-verification.md
    - 2026-02-20-model-aware-backend-routing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-model-aware-backend-routing.md
    - 2026-02-20-probe-default-backend-anthropic-incompatibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-probe-default-backend-anthropic-incompatibility.md
    - 2026-02-20-backend-resolution-architecture-drift
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-backend-resolution-architecture-drift.md
- Probe: orch clean --sessions kills daemon-spawned Claude CLI tmux windows
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-24-probe-orch-clean-sessions-daemon-window-protection.md
  - Recent Probes:
    - 2026-02-24-probe-orch-clean-sessions-daemon-window-protection
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-24-probe-orch-clean-sessions-daemon-window-protection.md
    - 2026-02-14-probe-vector7-sqlite-migration-json-fallback
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md
    - 2026-02-14-probe-vector2-cleanuntrackedsessions-removal
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md
- Probe: Dashboard Blind to Claude CLI Tmux Agents
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-17-dashboard-blind-to-tmux-agents.md
  - Recent Probes:
    - 2026-02-24-probe-orch-complete-kills-wrong-tmux-window
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-orch-complete-kills-wrong-tmux-window.md
    - 2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md
    - 2026-02-24-probe-tmux-liveness-two-lane-violation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md
    - 2026-02-24-probe-claude-spawn-dashboard-visibility-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md
    - 2026-02-20-tradeoff-visibility-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-20-tradeoff-visibility-gap-analysis.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-20.
    Changed files: pkg/spawn/resolve.go, pkg/orch/extraction.go.
    Deleted files: ~/.claude/skills/meta/orchestrator/SKILL.md.
    Verify model claims about these files against current code.
  - Summary:
    Anthropic restricts Opus 4.5 access via fingerprinting that blocks API usage but allows Claude Code CLI with Max subscription. This constraint forced a **dual spawn architecture**: primary path (OpenCode API + Sonnet/Flash, headless, high concurrency) and escape hatch (Claude CLI + Opus, tmux, crash-resistant). The escape hatch exists because critical infrastructure work (fixing the spawn system itself) can't depend on what might fail. Model choice now encodes reliability requirements, not just quality preferences.
    
    ---
  - Critical Invariants:
    1. **Never spawn OpenCode infrastructure work without --backend claude --tmux**
       - Violation: Agent kills itself mid-execution when server restarts
       - Now auto-detected: infrastructure keywords trigger `--backend claude` which implies tmux
    
    2. **Infrastructure detection is advisory, not overriding (changed Feb 2026)**
       - Runs at priority 5 (below CLI, model requirement, project config, user config)
       - When higher-priority setting present, emits warning instead of overriding
       - Ensures explicit user choices are always respected
    
    3. **Anthropic models blocked on OpenCode by default**
       - API requests to Anthropic models on opencode return error
       - Override: `allow_anthropic_opencode: true` in user config (`~/.orch/config.yaml`)
       - Opus specifically requires Claude CLI backend (fingerprinting blocks API)
    
    4. **Escape hatch provides true independence**
       - Claude CLI binary ≠ OpenCode server
       - Tmux session persists across service restarts
       - Different authentication path (Max subscription OAuth)
    
    5. **Flash models are blocked entirely (added Feb 2026)**
       - `validateModel()` returns error for any flash model
       - Supersedes the Gemini Flash TPM limit constraint — no workaround needed
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Zombie Agents (Jan 8, 2026)
    
    **Symptom:** Agents tracked in registry but never actually ran
    
    **Root Cause:** Spawning with `--model opus` before understanding auth gate
    - orch created registry entry
    - OpenCode session created
    - Anthropic rejected API request (fingerprinting)
    - Agent hung in "running" state
    - Consumed concurrency slot without doing work
    
    **Examples:**
    - orch-go-mo0ja, orch-go-pzi2i, orch-go-aoei0, orch-go-gd1gd, orch-go-lwc3o
    
    **Fix:** Never use `--model opus` without `--backend claude`
    
    ### Failure Mode 2: Header Injection Conflicts (Jan 8, 2026)
    
    **Symptom:** Gemini Flash spawns hung after attempting Opus bypass
    
    **Root Cause:** Injected Claude Code headers (`x-app: cli`, `anthropic-version`, etc.) into OpenCode's Anthropic provider
    - Bypassed Opus gate (didn't work)
    - Broke Gemini spawns (headers conflicted with Bun fetch/SDK)
    - System-wide impact from localized change
    
    **Lesson:** Fingerprinting is more sophisticated than headers alone
    
    ### Failure Mode 3: Infrastructure Work Kills Itself
    
    **Symptom:** Agent fixing OpenCode server crashes mid-execution
    
    **Root Cause:** Agent spawned via OpenCode API, agent's fix restarts OpenCode server, agent's session killed
    
    **Solution:** Infrastructure work detection auto-applies `--backend claude --tmux`
    
    **Why auto-detection matters:**
    - Humans forget to add flags manually
    - Task description might not mention "opencode" explicitly
    - Keyword scan catches common patterns
    - Escape hatch becomes invisible safety net
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility.md
    - 2026-02-21-probe-gpt-model-spawn-e2e-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-21-probe-gpt-model-spawn-e2e-verification.md
    - 2026-02-20-model-aware-backend-routing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-model-aware-backend-routing.md
    - 2026-02-20-probe-default-backend-anthropic-incompatibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-probe-default-backend-anthropic-incompatibility.md
    - 2026-02-20-backend-resolution-architecture-drift
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-backend-resolution-architecture-drift.md
- Orchestration Cost Economics
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-20.
    Changed files: pkg/spawn/resolve.go, pkg/model/model.go, cmd/orch/tokens.go.
    Deleted files: ~/.local/share/opencode/auth.json, ~/.anthropic/, pkg/spawn/backend.go.
    Verify model claims about these files against current code.
  - Summary:
    Agent orchestration cost is driven by three factors: **model pricing** (10-100x variance), **access restrictions** (fingerprinting, OAuth blocking), and **visibility** (lack of tracking caused $402 surprise spend). The Jan 2026 cost crisis revealed that headless spawning without cost visibility leads to runaway spend. As of Feb 2026, the **default spawn path is Claude backend + Max subscription** (Sonnet via Claude CLI), making the $200/mo flat rate the primary economic path — not just the escape hatch. The provider ecosystem has expanded to 4 providers (Anthropic, Google, OpenAI/Codex, DeepSeek) with centralized config resolution (`pkg/spawn/resolve.go`) and model-aware backend routing.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-24-probe-automatic-account-distribution-design
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics/probes/2026-02-24-probe-automatic-account-distribution-design.md
    - 2026-02-20-model-drift-stale-references-audit
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics/probes/2026-02-20-model-drift-stale-references-audit.md
- Probe: Orchestrator Skill CLI Staleness Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md
  - Summary:
    | # | Stale Reference | Severity | Location (template line) |
    |---|----------------|----------|--------------------------|
    | 1 | `--opus` flag | HARMFUL | 231-234, 236 |
    | 2 | `orch frontier` | HARMFUL | 320, 621 |
    | 3 | `orch rework` | HARMFUL | 625 |
    | 4 | `orch reflect` | HARMFUL | 625 |
    | 5 | `orch kb archive-old` | HARMFUL | 625 |
    | 6 | `orch clean --stale` | HARMFUL | 649 |
    | 7 | `orch clean --untracked --stale` | HARMFUL | 326 |
    | 8 | Default = "sonnet + headless" | MISLEADING | 231-233 |
    | 9 | "Spawn modes: Default (headless)" | MISLEADING | 227, 544 |
    | 10 | `bd label <id>` (missing subcommand) | MISLEADING | 132, 578 |
    | 11 | Missing --bypass-triage in examples | MISLEADING | 233-234 |
    | 12 | bd comment deprecation | COSMETIC | worker-base (not this template) |
    | 13 | Reference file path | COSMETIC | 631 |
    
    **Total: 7 HARMFUL + 4 MISLEADING + 2 COSMETIC = 13 stale references**
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-25-probe-orchestrator-skill-cross-project-injection-failure
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md
    - 2026-02-24-probe-orchestrator-skill-behavioral-compliance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md
    - 2026-02-18-probe-skillc-pipeline-audit
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-probe-skillc-pipeline-audit.md
    - 2026-02-18-orchestrator-skill-cli-staleness-audit
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md
    - 2026-02-17-orchestrator-skill-injection-path-trace
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
- Probe: Backend-Agnostic Session Contract Design
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-14-backend-agnostic-session-contract.md
  - Recent Probes:
    - 2026-02-24-probe-orch-complete-kills-wrong-tmux-window
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-orch-complete-kills-wrong-tmux-window.md
    - 2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md
    - 2026-02-24-probe-tmux-liveness-two-lane-violation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md
    - 2026-02-24-probe-claude-spawn-dashboard-visibility-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md
    - 2026-02-20-tradeoff-visibility-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-20-tradeoff-visibility-gap-analysis.md
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

### Guides (procedural knowledge)
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- orch CLI Reference
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/cli.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- Resilient Infrastructure Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/resilient-infrastructure-patterns.md
- Completion Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md

### Related Investigations
- Architect: Design Enforcement for Investigation → Architect → Implementation Sequence
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md
- Orchestrator Skill Behavioral Compliance — Why Agents Load the Skill but Revert to Claude Code Defaults
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md
- Evaluate Building API Proxy Layer for Claude Max Account Sharing
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-evaluate-building-api-proxy-layer.md
- Patch Density Architect Escalation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-design-patch-density-architect-escalation.md
- Design: Automatic Account Distribution for Claude CLI Spawns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-24-design-automatic-account-distribution-claude-cli.md
- Architect Design Accretion Gravity Enforcement
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md
- Spike: Claude Code Hooks for Orchestrator Action Enforcement
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md
- Fix Skill Inference Architect Title
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-fix-skill-inference-architect-title.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





## AREA CONTEXT: decisions

## decisions
**Artifacts:** 57 decisions
**Health:** ⚠ not acted on
**Last updated:** 2026-02-25



## HOTSPOT AREA WARNING

⚠️ This task targets files in a **hotspot area** (high churn, complexity, or coupling).

**Hotspot files:**
- `spawn`
- `session`
- `agent`
- `agents`
- `investigations`
- `kb`
- `investigation`
- `context`
- `cli`
- `auto`
- `pattern`
- `sessions`
- `md`
- `probe`
- `claude`
- `feature`
- `project`
- `ui`
- `artifact`
- `knowledge`
- `pre`
- `architect`
- `bug`
- `decision`
- `id`
- `recover`
- `across`
- `artifacts`
- `code`
- `cross`
- `doesn`
- `self`
- `start`
- `type`
- `up`
- `ci`
- `count`
- `injection`
- `patterns`
- `reconcile`
- `recovery`
- `spawned`
- `tab`
- `via`
- `already`
- `constraint`
- `text`
- `spawn`

**Investigation routing:** If your findings affect these files, recommend `architect` follow-up instead of direct `feature-impl`. Hotspot areas require architectural review before implementation changes.



🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. (Allowed) Read this SPAWN_CONTEXT.md file (your first tool call may be this read)
2. Immediately report via `bd comment orch-go-1239 "Phase: Planning - [brief description]"`
3. Read relevant codebase context for your task and begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — the workspace often has unrelated changes (.autorebuild.lock, .beads/, build/).
Stage ONLY the specific files you created or modified for your task, by name.


1. **COMMIT YOUR WORK:**
   ```bash
   git add <files you changed>
   git commit -m "feat: [brief description of changes] (orch-go-1239)"
   ```
2. Run: `bd comment orch-go-1239 "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.


⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Worker rule: Commit your work, report Phase: Complete, call `/exit`. Don't push.

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
Authority delegation rules are provided via skill guidance (worker-base skill).
**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Surface it first: `bd comment orch-go-1239 "CONSTRAINT: [what constraint] - [why considering workaround]"`
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

2. [Task-specific deliverables]


3. ⚡ SYNTHESIS.md is NOT required (light tier spawn).


STATUS UPDATES:
Track progress via beads comments. Call /exit to close agent session when done.



## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-1239**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-1239 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-1239 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-1239 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-1239 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-1239 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-1239`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (feature-impl)

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
bd comment orch-go-1239 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-1239 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-1239 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-1239 "Phase: BLOCKED - Need clarification on API contract"

# Report questions
bd comment orch-go-1239 "Phase: QUESTION - Should we use JWT or session-based auth?"
```

**When to report:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Additional context:**
Use `bd comment` for additional context, findings, or updates:
```bash
bd comment orch-go-1239 "Found performance bottleneck in database query"
bd comment orch-go-1239 "investigation_path: .kb/investigations/2026-02-11-perf-issue.md"
```

**Test Evidence Requirement:**
When reporting Phase: Complete, include test results in the summary:
- Example: `bd comment orch-go-1239 "Phase: Complete - Tests: go test ./... - 47 passed, 0 failed (2.3s)"`
- Example: `bd comment orch-go-1239 "Phase: Complete - Tests: npm test - 23 specs, 0 failures"`
- Example: `bd comment orch-go-1239 "Phase: Complete - Tests: make test - PASS (coverage: 78%)"`

**Why:** `orch complete` validates test evidence in phase comments. Vague claims like "all tests pass" trigger manual verification.

**Never run `bd close`** - Only the orchestrator closes issues via `orch complete`.
- Workers report `Phase: Complete`, orchestrator verifies and closes
- Running `bd close` bypasses verification and breaks tracking

---


## Phase Reporting

**First 3 Actions (Critical):**
Within your first 3 tool calls, you MUST:
1. Report via `bd comment orch-go-1239 "Phase: Planning - [brief description]"`
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



1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment orch-go-1239 "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
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
name: feature-impl
skill-type: procedure
description: Unified feature implementation with configurable phases (investigation, clarifying-questions, design, implementation, validation, integration). Replaces test-driven-development, surgical-change, and feature-coordination skills. Use for any feature work with phases/mode/validation configured by orchestrator.
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 8718d3a6391e -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/skills/src/worker/feature-impl/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-20 15:11:07 -->


## Summary

name: feature-impl
skill-type: procedure
description: Unified feature implementation with configurable phases (investigation, clarifying-questions, design, implementation, validation, integration). Replaces test-driven-development, surgical-change, and feature-coordination skills. Use for any feature work with phases/mode/validation configured by orchestrator.

---

---
name: feature-impl
skill-type: procedure
description: Unified feature implementation with configurable phases (investigation, clarifying-questions, design, implementation, validation, integration). Replaces test-driven-development, surgical-change, and feature-coordination skills. Use for any feature work with phases/mode/validation configured by orchestrator.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 047ddb2689b3 -->
<!-- Source: .skillc -->
<!-- To modify: edit files in .skillc, then run: skillc build -->
<!-- Last compiled: 2026-01-07 14:41:54 -->


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
- **Implementation Mode:** `tdd`, `direct`, or `verification-first` (only relevant if implementation phase included)
- **Validation Level:** `none`, `tests`, `smoke-test`, or `multi-phase` (only relevant if validation phase included)

**Example configuration:**
```
Phases: design, implementation, validation
Mode: tdd
Validation: smoke-test
```

**Mode Selection Guide:**

| Mode | When to Use |
|------|-------------|
| `tdd` | Adding behavior (APIs, business logic, UI) - discover through tests |
| `direct` | Non-behavioral changes (refactoring, config, docs) |
| `verification-first` | Spec exists, multi-agent work, high-risk features - spec is the contract |

---

## Deliverables

| Configuration | Required |
|---------------|----------|
| investigation phase | Investigation file |
| design phase | Design document |
| implementation phase | Source code |
| mode=tdd | Tests (write first) |
| mode=verification-first | Verification spec consumed + tests with AC-xxx traceability + evidence per spec |
| validation=tests | Tests |
| validation=smoke-test | Validation evidence via bd comment |
| validation=multi-phase | Phase checkpoints via bd comment |

---

## Workflow

Proceed through phases sequentially per your configuration.

**Phases:** Investigation → Clarifying Questions → Design → Implementation (TDD/direct) → Validation → Self-Review → Integration

Track progress via `bd comment <beads-id> "Phase: <Phase> - <details>"`.

---

## Step 0: Scope Enumeration (REQUIRED)

**Purpose:** Prevent "Section Blindness" - implementing only part of spawn context.

**Before starting ANY phase work:**

1. **Read ENTIRE SPAWN_CONTEXT** - Don't skim. Don't stop at first section.
2. **Enumerate ALL Requirements** - List every deliverable from ALL sections
3. **Report Scope:**
   ```bash
   bd comment <beads-id> "Phase: Planning - Scope: 1. [requirement] 2. [requirement] 3. [requirement] ..."
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

**Completion:** Investigation committed, reported via `bd comment <beads-id> "Phase: Clarifying Questions - Investigation complete"`

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

**Completion:** All questions answered, reported via `bd comment <beads-id> "Phase: Design - Questions resolved"`

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

**Completion:** Design approved, committed, reported via `bd comment <beads-id> "Phase: Implementation - Design approved"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-design.md` for detailed workflow and template.

---

### Harm Assessment (Pre-Implementation Checkpoint)

**Purpose:** Evaluate feature ethics BEFORE implementation. Distinct from Security Review (code quality) - this is about feature design itself.

**When to run:** Before starting Implementation Phase (TDD or direct).

**Quick Assessment:**

| Question | If YES → |
|----------|----------|
| Could this harm, deceive, or manipulate users? | Document concern |
| Does this collect/transmit unexpected data? | Document concern |
| Could this be weaponized at scale? | Document concern |
| Does this undermine informed consent? | Document concern |
| Disproportionate impact on vulnerable populations? | Document concern |

**If concerns identified:**
1. Document: `bd comment <beads-id> "HARM ASSESSMENT: [concern]"`
2. Check if SPAWN_CONTEXT addresses with safeguards
3. If addressed → Proceed with documented safeguards
4. If NOT addressed → Escalate via Constitutional Objection Protocol (see worker-base)

**If no concerns:** Proceed to Implementation.

**Common false positives (usually OK):**
- Analytics with proper consent disclosure
- Security features (authentication, rate limiting)
- Moderation tools with appeals process
- Personalization with user control

**Completion:** Assessment documented via `bd comment`, proceed to Implementation Phase.

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

**Completion:** All tests pass, reported via `bd comment <beads-id> "Phase: Validation - All tests pass"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-implementation-tdd.md` and `reference/tdd-best-practices.md`.

---


---


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

**Completion:** Integration tests pass, reported via `bd comment <beads-id> "Phase: Validation - Integration tests pass"`

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
- [ ] Reviewed for discoveries (bugs, tech debt, enhancement ideas, strategic questions)
- [ ] Created beads issues with `triage:review` label (default - lets orchestrator review before daemon spawns)
      - Known cause/task: `--type task` or `--type bug`
      - Unknown premise/strategic unknown: `--type question`

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
| Made a choice | `kb quick decide "X" --reason "Y"` |
| Something failed | `kb quick tried "X" --failed "Y"` |
| Found a constraint | `kb quick constrain "X" --reason "Y"` |
| Open question | `kb quick question "X"` |

**If nothing to externalize:** Note in completion comment.

**Completion Criteria:**
- [ ] Reflected on what was learned
- [ ] Ran at least one `kb quick` command OR noted why nothing to externalize
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
- [ ] **If web/ modified:** Visual verification completed with evidence in phase report
- [ ] Final status: `bd comment <beads-id> "Phase: Complete - [summary]"`

**⚠️ If web/ files modified without visual verification → completion will be REJECTED.**

**If ANY unchecked, work is NOT complete.**

**Final step:** After all criteria met:
1. `bd comment <beads-id> "Phase: Complete - [summary]"` (report FIRST)
2. Commit any final changes
3. Call `/exit` to close the agent session

**Note:** Workers do NOT close issues - only the orchestrator closes via `orch complete`.

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
- **api:** http://localhost:3348
- **web:** http://localhost:5188

**Quick commands:**
- Start servers: `orch servers start orch-go`
- Stop servers: `orch servers stop orch-go`
- Open in browser: `orch servers open orch-go`



🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — stage ONLY your task files by name.



1. **COMMIT YOUR WORK:** `git add <files you changed> && git commit -m "feat: [description] (orch-go-1239)"`
2. `bd comment orch-go-1239 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.



⛔ **NEVER run `git push`** - Workers commit locally only.
⚠️ Your work is NOT complete until Phase: Complete is reported (or /exit for --no-track).
