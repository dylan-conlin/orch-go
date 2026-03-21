# Decision: Lifecycle Ownership Boundaries (Own / Accept / Build)

**Date:** 2026-02-14
**Status:** Accepted
**Enforcement:** context-only
**Supersedes:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`
**Context:** The original decision identified three buckets: Own, Accept, Lobby. The "Lobby" bucket assumed OpenCode was an external dependency requiring upstream issue filing. Discovery that Dylan owns an OpenCode fork (`~/Documents/personal/opencode`) changes "Lobby" to "Build" — these features can be implemented directly.

## What Changed

The original decision's Phase 3 was "File upstream issues." That's now replaced by direct implementation in the fork, with a new Phase 5.

**Phases 1-4 are complete** (implemented 2026-02-13):
- Phase 1: Clean simplified 7→3 flags, ghost/phantom/orphan vocabulary eliminated
- Phase 2: pkg/registry eliminated (529 lines), consumers migrated to workspace files
- Phase 3: ~~File upstream issues~~ → superseded by Phase 5 below
- Phase 4: Lifecycle state model updated with state vs infrastructure distinction

## Decision

### Three-Bucket Ownership Model (Updated)

**BUCKET 1: OWN** — unchanged from original decision.

**BUCKET 2: ACCEPT (narrowed)**
- Dual backend complexity → escape hatch is architecturally justified
- Session status is ephemeral (in-memory only, lost on restart) → acceptable, treat absence as "check session exists" fallback

**BUCKET 3: BUILD IN FORK (was: Lobby Upstream)**
- Session metadata API (store beads_id, workspace, tier with session) → 1-2h in fork
- Session TTL with configurable expiry → 2-4h in fork
- `GET /session/status` already exists → zero OpenCode work, orch-go integration only

### Phase 5: Build in Fork + Integrate in orch-go

**Prerequisite: Rebase fork onto upstream/dev** (384 commits behind as of Feb 14)
- 13 custom commits to cherry-pick back
- Must verify session management code hasn't changed upstream before modifying

**Step 1: Integrate existing session status endpoint** (orch-go only, ~1 day)
- `GET /session/status` already returns `Record<string, SessionStatus>` with idle/busy/retry
- Wire orch-go `pkg/opencode/client.go` to use this instead of SSE-only polling
- Eliminates ~1,400 lines of SSE status parsing

**Step 2: Add session metadata API to fork** (OpenCode 1-2h + orch-go 1 day)
- Add optional `metadata?: Record<string, string>` to Session.Info Zod schema
- Accept metadata in POST /session (create) and PATCH /session/:id (update)
- orch-go: store beads_id, workspace_path, tier, spawn_mode at session creation
- Eliminates ~800 lines of workspace .session_id cross-reference

**Step 3: Add session TTL to fork** (OpenCode 2-4h + orch-go 1 day)
- Add optional `ttl?: number` to Session.Info.time schema
- Add periodic cleanup job (Instance eviction pattern is template)
- Protect sessions with active prompts from deletion
- orch-go: remove remaining cleanup logic for stale sessions
- Eliminates ~1,200 lines of cleanup code

### Fork Constraints

- Keep custom commits minimal and isolated (currently 13, target <20)
- Must periodically rebase onto upstream/dev
- Session metadata schema must be generic (Record<string, string>, not orch-specific fields)
- TTL cleanup must respect active prompts
- Status endpoint caveat: in-memory only, lost on server restart

## Consequences

**Positive:**
- ~3,400 additional lines eliminable from orch-go (on top of Phases 1-2 already done)
- No dependency on upstream project's roadmap — we control the timeline
- Session status endpoint already exists — immediate win with zero OpenCode work
- Metadata API is trivial (1-2h) — Zod schema addition + PATCH handler update

**Negative:**
- Fork maintenance burden — must rebase 13+ custom commits on upstream updates
- Features may conflict with upstream changes to session management
- More fork divergence = harder rebases

**Risks:**
- Upstream may add competing implementations that conflict with ours
- ORCH_WORKER server-side code was already lost in a previous rebase — custom commits can be fragile

## Evidence

- Fork model: `.kb/models/opencode-fork/model.md`
- Fork investigation: `.kb/investigations/2026-02-13-inv-build-model-opencode-fork.md`
- Original decision: `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`
- Phase 1-2 completion: agents orch-go-l1s (clean), orch-go-352 (registry)
- Phase 4 completion: agent orch-go-2a5 (model update)
- `GET /session/status` endpoint confirmed: `packages/opencode/src/session/session.ts`
