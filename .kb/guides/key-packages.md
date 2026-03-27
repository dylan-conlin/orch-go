# Key Packages

**Purpose:** Descriptions of key orch-go packages — their responsibilities, main types, and APIs.

**Last verified:** 2026-03-27

---

## cmd/orch/main.go (Entry Point)

- Uses Cobra framework for CLI structure
- Global `--server` flag for OpenCode URL
- 70+ subcommands registered across cmd/orch/*.go (see `.kb/guides/cli.md` for full reference)

## pkg/opencode/ (OpenCode Client)

- `Client` struct with HTTP methods for OpenCode REST API
- `ListSessions()`, `GetSession()`, `CreateSession()`, `GetMessages()`
- `SSEClient` for real-time event streaming
- Session status polling for completion detection

## pkg/model/ (Model Resolution)

- `Resolve(spec)` maps aliases to full provider/model format
- Aliases: `opus`, `sonnet`, `haiku` (Anthropic), `flash`, `pro` (Gemini), `gpt-5.4`, `codex-latest` (OpenAI)
- Code fallback: `anthropic/claude-sonnet-4-5-20250929`; effective default is Opus via project config
- Helper methods: `IsAnthropicModel()`, `IsOpenAI()`, `IsReasoningModel()`, `ProviderName()`, `ModelFamily()`, `String()`

## pkg/account/ (Account Management)

- `LoadConfig()` reads `~/.orch/accounts.yaml`
- `Switch(name)` refreshes OAuth tokens and updates OpenCode auth
- Token sources: OpenCode auth file, macOS Keychain

## pkg/spawn/ (Spawn Context)

- `SpawnConfig` struct with all spawn parameters
- `GenerateContext()` creates SPAWN_CONTEXT.md content
- Embeds skill content, task description, beads issue context
- Spawn gates in `pkg/spawn/gates/` (governance-protected)
- OPSEC enforcement (`opsec.go`): proxy health check, env injection, settings merge/unmerge for network sandboxing

## pkg/verify/ (Completion Verification)

- `Check()` validates agent work before closing (governance-protected)
- Verifies: Phase Complete, deliverables exist, commits present
- `Update()` closes beads issue with completion reason
- Pre-commit checks: `accretion_precommit.go`, `model_stub_precommit.go`, `duplication_precommit.go` (power `orch precommit`)
- `consequence_sensor.go`: validates architect outputs declare how gate effects will be observed
- `plan_hydration.go`: hydrates multi-phase architect plans into implementation issues
- `synthesis_parser.go`: parses SYNTHESIS.md to extract phases (`ExtractPhases`), TLDRs, and structured sections
- `synthesis_quality.go`: computes 6 quality signals from parsed Synthesis — structural_completeness, evidence_specificity, model_connection, connective_reasoning, tension_quality, insight_vs_report. `ComputeSynthesisQuality()` returns `SynthesisQuality` with signal count and `MeetsThreshold()` for advisory gates

## pkg/daemon/ + pkg/daemonconfig/ (Daemon)

- OODA poll cycle: Sense → Orient → Decide → Act
- `ComplianceConfig` for per-spawn resolution
- Allocation profiles for skill-aware slot scoring
- Verification retry: retries failed verifications with backoff
- Agreement checks: cross-validates daemon decisions
- Beads health monitoring with circuit breaker
- Phase timeout detection and escalation
- Artifact sync: documentation drift detection
- Phantom spawn detection: post-spawn workspace verification (`workspace_verify.go`)
- Shared interfaces (`interfaces.go`): `WorkspaceVerifier`, `IssueUpdater`
- Spawn execution pipeline (`spawn_execution.go`): dedup gates → pool slot → status update → spawn → workspace verify
- Cycle cache (`cycle_cache.go`): shares `GetActiveAgents()` across periodic tasks to avoid redundant queries
- Audit selection: random quality audits weighted toward auto-completed work
- Capacity polling: account capacity cache for `orch status`
- Comprehension queue (`comprehension_queue.go`): two-state lifecycle (unread/processed), spawn throttling, quality signal parsing (`ParseBriefSignals`, `ParseBriefSignalCount`), queue ordering by signal count (`OrderBriefsBySignals`), brief feedback tracking (`RecordBriefFeedback`)
- Resume signal (`resume_signal.go`): file-based daemon resume trigger
- Shutdown budget (`shutdown_budget.go`): explicit time budgets for shutdown phases (4s total, launchd 5s - 1s safety)
- Launchd log dedup (`log.go`): detects stdout redirect to avoid double logging under launchd

## pkg/compose/ (Brief Composition)

- `Compose(briefsDir, threadsDir)` clusters briefs by keyword overlap, matches to threads
- `NameClusters(clusters)` assigns distinctive names using inverse cluster frequency scoring
- `WriteDigest(digest, digestsDir)` writes digest markdown to `.kb/digests/`
- Keyword extraction, TF-IDF-style scoring, Jaccard similarity clustering
- Thread matching uses title keyword overlap (weighted 2x) + body overlap; requires `MinThreadTitleOverlap` (1) and `MinThreadMatchScore` (3)
- Constants: `MinKeywordOverlap` (4), `MaxDocumentFrequency` (0.15)
- `ThreadInfo.TitleKeywords` — keywords from thread title only, used for focused matching
- CLI: `orch compose` (cmd/orch/compose_cmd.go)

## pkg/dupdetect/ (Duplicate Detection)

- Detects duplicate spawns during completion pipeline
- Allowlist for known false positives

## pkg/orient/ (Daemon Orient Phase)

- Measurement feedback loop
- Work graph for daemon prioritization
- Git-based ground-truth metrics (merge rate, net code impact)
- Model trust scores with decay tracking
- Thread-first orient: leads with active thread state, then evidence/operations

## pkg/orch/ (Spawn Pipeline & Completion)

- Spawn pipeline orchestration: preflight checks, backend routing, mode selection
- Completion pipeline logic (extracted from cmd/orch)
- Governance checks and spawn gate integration
- Backend abstraction: Claude CLI (tmux) vs OpenCode (headless)
- Spawn type definitions, inference, and beads integration
- Loop controller (`loop.go`): `--loop` spawn mode iteration cycle (wait → eval → rework)

## pkg/plan/ (Coordination Plans)

- Plan types for multi-phase coordination
- Beads status overlay integration

## pkg/routing/ (Issue Routing)

- Skill inference from issue type, title, description, and labels (`InferSkillForIssue`)
- Area label inference from file path keywords (`InferAreaFromText`)
- `EnrichRoutingLabels()` auto-applies `skill:` and `area:` labels at issue creation
- Preserves explicit labels — only infers when prefix not already present

## pkg/claims/ (Claim Tracking)

- Machine-readable claim index (`claims.yaml`) for KB models
- Tension-cluster detection (cross-model claim convergence)
- Claim lifecycle states (hypothesis → tested → confirmed/refuted)
- Drives daemon probe generation and orient surfacing
- `ExtractClaimRef()` extracts claim IDs from probe content (frontmatter `claim:` field or Model Impact verdict lines)

## pkg/beads/ (Beads Issue Tracking)

- `BeadsClient` interface decouples from `bd` CLI
- Issue CRUD, comment management, label operations
- Mock support for testing without `bd` binary

## pkg/events/ (Event System)

- Structured event types: `session.spawned`, `agent.completed`, `agent.rejected`, `spawn.gate_decision`, `daemon.architect_escalation`, `spawn.model_route`, `daemon.thin_issue_detected`, `kb.context.timeout`, etc.
- `events.jsonl` append-only log for telemetry
- Event enrichment fields for beads close hook
- Specialized event emitters: `kb_context_timeout.go`, `model_route.go`, `thin_issue.go`, `decision.go`, `command.go`

## pkg/discovery/ (Agent Discovery)

- Backend-aware agent query interface
- Prevents multi-backend blindness (Class 2 defects)
- Unified view across Claude CLI (tmux) and OpenCode backends

## pkg/attention/ (Work Graph Monitoring)

- Composable attention signal architecture
- Attention signals for daemon work prioritization
- Signal types: model contradictions, hotspot acceleration, knowledge decay

## pkg/hook/ (Hook Testing & Tracing)

- Hook configuration reader from `~/.claude/settings.json`
- Matcher resolution, trace viewing
- Simulation of hook invocations outside Claude Code

## pkg/control/ (Control Plane)

- Control plane immutability via macOS `chflags uchg`
- Lock/unlock/status for governance files
- Deny rule validation in settings.json

## pkg/artifactsync/ (Artifact Sync)

- Change-scope classification at completion time
- Drift event logging to `artifact-drift.jsonl`
- Manifest management for tracked documentation artifacts

## pkg/tmux/ (Tmux Backend)

- Tmux session and window management for agent spawning
- Tmux follower polling for orchestrator output
- Window targeting by workspace name

## pkg/skills/ (Skill System)

- Skill discovery and loading from `~/.claude/skills/`
- Section filtering for skill content injection
- Skill metadata parsing

## pkg/kbmetrics/ (KB Health Metrics)

- Claims-per-model extraction
- Knowledge base health analysis
- Quality scoring for spawn context
- Evidence-tier classification (`drift.go`): classifies claim annotations into tiers (assumed → validated)
- Tier-drift and scope-drift detection: flags overclaim language that exceeds declared evidence tier
- Provenance tracking for KB claim sources

## pkg/debrief/ (Session Debriefs)

- Session debrief generation and auto-population
- Durable artifacts at `.kb/sessions/YYYY-MM-DD-debrief.md`
- Cross-session trend tracking

## pkg/entropy/ (Codebase Health)

- Growth trend analysis, duplication detection
- Structural health scoring (Harness Layer 3)
- Entropy spiral condition detection

## pkg/focus/ (Priority Tracking)

- North star tracking for multi-project prioritization
- Current priority goal storage for work selection guidance

## pkg/identity/ (Project Resolution)

- Issue ID prefix to project directory mapping
- Cross-project identity resolution

## pkg/group/ (Project Groups)

- Project group resolution
- Collections of related projects sharing KB context scope

## pkg/decisions/ (Decision Lifecycle)

- Enforcement type classification
- Staleness detection for uncited decisions
- Decision budget cap enforcement

## pkg/config/ (Project Config)

- Project-level configuration (`orch.yaml`)
- Backend selection, model defaults, spawn settings
- `OpsecConfig`: network sandbox settings (sandbox enable, proxy port, blocked domains)

## pkg/scaling/ (Scaling Utilities)

- Numeric and string scaling utilities for N>2 agent experiments
- `Normalize()`, `Clamp()`, `Wrap()` helpers

## pkg/userconfig/ (User Config)

- User-level configuration (`~/.orch/config.yaml`)
- Notification preferences, account defaults

## pkg/agent/ (Agent Utilities)

- Agent filtering logic for determining "active" agents
- Workspace scanning, manifest parsing, agent metadata

## pkg/session/ (Orchestrator Session State)

- Orchestrator session state management (start, status, end, resume)
- Session goal tracking, time tracking, label management

## pkg/sessions/ (Session History)

- Search and listing for OpenCode session history
- Walks OpenCode disk storage, fetches message content for searching

## pkg/action/ (Action Outcome Logging)

- Action outcome logging and behavioral pattern detection
- Tracks tool invocations and outcomes for pattern surfacing

## pkg/health/ (System Health Monitoring)

- Time-series health monitoring for `orch doctor`
- Tracks system invariants, alerts on threshold crossings

## pkg/checkpoint/ (Verification Checkpoints)

- Verification checkpoint tracking infrastructure
- Phase 1 of verifiability-first mechanical enforcement

## pkg/completion/ (Completion Validation)

- Completion artifact validation (COMPLETION.yaml parsing)
- Required field enforcement per work type

## pkg/coaching/ (Coaching Plugin)

- Agent coaching and guidance infrastructure

## pkg/thread/ (Living Threads)

- Living threads — multi-session accumulating knowledge artifacts
- Captures insight as it forms for comprehension
- Lifecycle states: `forming`, `active`, `converged`, `subsumed` (backward-compatible with `open`/`resolved`)
- Relational frontmatter: `spawned_from`, `spawned`, `active_work`, `resolved_by`
- `BackPropagateCompletion()`: on `orch complete`, moves beads IDs from `active_work` to `resolved_by`
- `CreateWithParent()`: child thread creation with bidirectional parent/child links
- `LinkWork()`: connect beads issues to threads (used by `spawn --thread`)
- `thread resolve` prompts for resolved-to target (model/decision/brief) when not provided via `--to` flag

## pkg/port/ (Port Allocation)

- Port allocation registry for orch-go projects
- Allocate, list, release, tmuxinator integration

## pkg/workspace/ (Workspace Operations)

- Shared workspace scanning and manifest operations

## pkg/service/ (Service Monitoring)

- Service monitoring for overmind-managed processes
- PID tracking, crash detection, restart events

## pkg/claudemd/ (CLAUDE.md Generation)

- CLAUDE.md template generation for different project types

## pkg/timeline/ (Session Timeline)

- Session timeline extraction and grouping

## pkg/urltomd/ (URL to Markdown)

- URL-to-Markdown conversion using headless Chrome (chromedp)

## pkg/beadsutil/ (Beads Utilities)

- Shared beads ID parsing, extraction, and resolution utilities

## pkg/question/ (Question Extraction)

- Extraction of pending questions from agent output

## pkg/display/ (Output Formatting)

- Shared output formatting: string truncation, ID abbreviation, ANSI helpers

## pkg/graph/ (Work Graph)

- Graph data structures for work dependencies

## pkg/activity/ (Activity Feed)

- Activity feed persistence for agent workspaces
- Exports session activity to ACTIVITY.json for archival

## pkg/notify/ (Desktop Notifications)

- Desktop notification functionality (configurable via `~/.orch/config.yaml`)

## pkg/state/ (Agent State Reconciliation)

- Agent state reconciliation across multiple sources

## pkg/advisor/ (Model Recommendation)

- Model recommendation using live API data (OpenRouter)

## pkg/kbgate/ (KB Quality Gates)

- Adversarial quality gates for knowledge base artifacts
- Challenge artifact generation with severity codes (ENDOGENOUS_EVIDENCE, VOCABULARY_INFLATION, etc.)
- Publication readiness validation
- Claim quality enforcement

## pkg/tree/ (Knowledge Lineage Tree)

- Parses investigation files and builds knowledge lineage graph
- Node types: investigations, guides, models, decisions
- Relationship extraction from Prior-Work tables and Synthesized-From headers
- Powers `orch tree` visualization

## pkg/bench/ (Benchmark Engine)

- Benchmark execution engine composing spawn/wait/eval/rework primitives
- Scenario configuration, model resolution, result aggregation
- Report generation with pass/fail/rework metrics

## pkg/execution/ (Backend Abstraction)

- Backend-agnostic types and interfaces for session management
- Decouples orchestration from specific backends (OpenCode, OpenClaw, Claude CLI)
- `SessionClient` interface, `OpenCodeAdapter` implementation

## pkg/openclaw/ (OpenClaw Client)

- WebSocket JSON-RPC client for the OpenClaw gateway
- Agent dispatch, completion polling, session management, health checks

## pkg/certs/ (TLS Certificates)

- Static TLS certificate and key files for OPSEC proxy
- Not a Go package — contains `cert.pem` and `key.pem`
