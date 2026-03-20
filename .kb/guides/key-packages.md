# Key Packages

**Purpose:** Descriptions of key orch-go packages — their responsibilities, main types, and APIs.

**Extracted from:** CLAUDE.md (2026-03-20)

---

## cmd/orch/main.go (Entry Point)

- Uses Cobra framework for CLI structure
- Global `--server` flag for OpenCode URL
- Subcommand groups: `account`, `daemon`, `doctor`, `harness`, `plan`, `control`, `focus`, `hook`, `thread`, `audit`, `backlog`, `settings`, `kb`, `port`, `review`, `patterns`, `session`, `session-history`, `servers`, `learn`, `config`, `docs`, `precommit`, `model`, `logs`, `transcript`, `serve`, `stats`

## pkg/opencode/ (OpenCode Client)

- `Client` struct with HTTP methods for OpenCode REST API
- `ListSessions()`, `GetSession()`, `CreateSession()`, `GetMessages()`
- `SSEClient` for real-time event streaming
- Session status polling for completion detection

## pkg/model/ (Model Resolution)

- `Resolve(spec)` maps aliases to full provider/model format
- Aliases: `opus`, `sonnet`, `haiku` (Anthropic), `flash`, `pro` (Gemini)
- Default: `google/gemini-3-flash-preview` (Opus restricted to Claude Code as of Jan 2026)

## pkg/account/ (Account Management)

- `LoadConfig()` reads `~/.orch/accounts.yaml`
- `Switch(name)` refreshes OAuth tokens and updates OpenCode auth
- Token sources: OpenCode auth file, macOS Keychain

## pkg/spawn/ (Spawn Context)

- `SpawnConfig` struct with all spawn parameters
- `GenerateContext()` creates SPAWN_CONTEXT.md content
- Embeds skill content, task description, beads issue context
- Spawn gates in `pkg/spawn/gates/` (governance-protected)

## pkg/verify/ (Completion Verification)

- `Check()` validates agent work before closing (governance-protected)
- Verifies: Phase Complete, deliverables exist, commits present
- `Update()` closes beads issue with completion reason
- Pre-commit checks: `accretion_precommit.go`, `model_stub_precommit.go`, `duplication_precommit.go` (power `orch precommit`)
- `consequence_sensor.go`: validates architect outputs declare how gate effects will be observed

## pkg/daemon/ + pkg/daemonconfig/ (Daemon)

- OODA poll cycle: Sense → Orient → Decide → Act
- `ComplianceConfig` for per-spawn resolution
- Allocation profiles for skill-aware slot scoring
- Learning Store for per-skill metrics from events.jsonl
- Phase 2 trigger detectors: model contradictions, hotspot acceleration, knowledge decay, skill performance drift
- Per-detector outcome tracking (completed/abandoned rates from beads data)
- Proactive extraction: auto-extracts knowledge from completed agent work
- Verification retry: retries failed verifications with backoff
- Agreement checks: cross-validates daemon decisions
- Synthesis auto-create: generates synthesis artifacts from accumulated findings
- Beads health monitoring with circuit breaker
- Phase timeout detection and escalation
- Investigation orphan cleanup
- Friction accumulator for system improvement signals
- Cycle cache (`cycle_cache.go`): shares `GetActiveAgents()` across periodic tasks to avoid redundant queries

## pkg/digest/ (KB Artifact Digest)

- Produces consumable thinking products from KB artifact changes
- Scans `.kb/threads/`, `.kb/models/`, `.kb/investigations/` for changes
- Packages notable changes as digest files in `~/.orch/digest/`
- Gate logic for filtering low-signal changes

## pkg/dupdetect/ (Duplicate Detection)

- Detects duplicate spawns during completion pipeline
- Allowlist for known false positives

## pkg/orient/ (Daemon Orient Phase)

- Measurement feedback loop
- Work graph for daemon prioritization
- Git-based ground-truth metrics (merge rate, net code impact)
- Model trust scores with decay tracking

## pkg/orch/ (Spawn Pipeline & Completion)

- Spawn pipeline orchestration: preflight checks, backend routing, mode selection
- Completion pipeline logic (extracted from cmd/orch)
- Governance checks and spawn gate integration
- Backend abstraction: Claude CLI (tmux) vs OpenCode (headless)
- Spawn type definitions, inference, and beads integration

## pkg/plan/ (Coordination Plans)

- Plan types for multi-phase coordination
- Beads status overlay integration

## pkg/claims/ (Claim Tracking)

- Machine-readable claim index (`claims.yaml`) for KB models
- Tension-cluster detection (cross-model claim convergence)
- Claim lifecycle states (hypothesis → tested → confirmed/refuted)
- Drives daemon probe generation and orient surfacing
