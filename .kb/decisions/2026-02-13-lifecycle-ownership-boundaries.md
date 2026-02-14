# Decision: Lifecycle Ownership Boundaries (Own / Accept / Lobby)

**Date:** 2026-02-13
**Status:** Superseded
**Superseded-By:** `.kb/decisions/2026-02-14-lifecycle-ownership-own-accept-build.md`
**Context:** Orch has accumulated ~8,800 lines of lifecycle management code across clean (7 flags), complete (11 gates), status (Priority Cascade), and registry — largely compensating for OpenCode's missing session management features. Three distinct ghost types (phantom, ghost, orphan) exist for one root cause: sessions never expire.

## Decision

### The Core Reframe: State vs Infrastructure

The four-layer state model conflates two distinct concerns:

| Type | Layers | Purpose | Lifecycle |
|------|--------|---------|-----------|
| **State** | Beads comments, workspace files | What work was done, what phase it's in | Persistent, orch-controlled |
| **Infrastructure** | OpenCode sessions, tmux windows | Execution resources | Transient, externally-controlled |

Treating infrastructure as state creates a reconciliation burden. Orch should manage state and *use* infrastructure, not reconcile them.

### Three-Bucket Ownership Model

**BUCKET 1: OWN (orch's domain, can't delegate)**
- Verification gates (all 11 in `orch complete`)
- Phase tracking (beads comments as canonical state)
- Workspace lifecycle (creation, archival, tier-based requirements)
- Skill integration (spawn context generation)
- Dual backend abstraction (OpenCode vs Claude CLI)
- Beads integration (issue creation, closure, dependency tracking)

**BUCKET 2: ACCEPT (external constraints, work within them)**
- OpenCode sessions persist indefinitely → periodic cleanup is reality
- No session state API → SSE for completion detection remains necessary
- No session metadata → workspace files as metadata store
- Dual backend complexity → escape hatch is architecturally justified

**BUCKET 3: LOBBY UPSTREAM (would reduce orch burden if OpenCode added)**
- Session TTL with configurable expiry → eliminates ghost/phantom/orphan classes
- Session metadata API (store beads_id, workspace, tier with session) → eliminates registry
- Session state HTTP endpoint (busy/idle queryable) → simplifies status polling
- Completion callback webhook → simplifies completion detection

### Implementation Plan

**Phase 1: Simplify clean** (1-2 days)
- Merge phantom/ghost/orphan into single concept ("stale sessions")
- Reduce 7 clean flags to 3: `--workspaces` (archive old), `--sessions` (purge stale), `--all`
- Delete ghost/phantom/orphan distinction in code

**Phase 2: Eliminate registry** (2-3 days)
- Validate workspace `.session_id` files serve all lookup needs
- Update status command to derive from OpenCode + beads (no registry)
- Update abandon command to use workspace files for session ID lookup
- Delete `pkg/registry/`

**Phase 3: File upstream issues** (1 day)
- Session TTL: `opencode/opencode` GitHub issue with use case
- Session Metadata API: issue with orchestration use case
- Session State HTTP endpoint: issue explaining SSE-only limitation

**Phase 4: Formalize state model** (1 day)
- Update `.kb/models/agent-lifecycle-state-model.md` to distinguish state from infrastructure
- Document three-bucket ownership model

## Consequences

**Positive:**
- ~40% lifecycle code reduction (~3,400 lines) once simplification + registry elimination complete
- No phantom/ghost/orphan zoo — one concept: "stale infrastructure"
- Clean becomes intuitive (3 flags vs 7)
- Registry elimination removes a reconciliation point

**Negative:**
- Still own session cleanup until upstream adds TTLs (months/quarters)
- SSE completion detection stays until upstream adds state API
- Registry removal needs careful validation of edge cases (abandon, cross-project)

**Risks:**
- OpenCode may reject upstream proposals (sessions-are-forever-by-design)
- Workspace files may not cover all registry lookup edge cases (need validation)

## Evidence

- Investigation: `.kb/investigations/2026-02-13-inv-evaluate-lifecycle-management-orch-own.md`
- Prior decision: `.kb/decisions/2026-01-12-registry-is-spawn-cache.md` (registry already demoted)
- Prior decision: `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` (event + periodic cleanup)
- Prior decision: `.kb/decisions/2026-01-14-infrastructure-complexity-justified.md` (complexity justified but redistributable)
- Code audit: 8,800 lines across clean_cmd.go, complete_cmd.go, serve_agents.go, registry, opencode client
