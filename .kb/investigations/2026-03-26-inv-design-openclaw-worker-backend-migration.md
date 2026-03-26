## Summary (D.E.K.N.)

**Delta:** The migration from OpenCode to OpenClaw as worker backend has a clean 4-phase path that replaces ~5,800 LoC (pkg/opencode/) and 48 importing files with a ~300 LoC WebSocket client, because the backend boundary sits at a single interface (`backends.Backend`) that already decouples execution technology from orchestration logic.

**Evidence:** Source analysis of the current Backend interface (pkg/spawn/backends/backend.go), all 48 files importing pkg/opencode/, the OpenClaw API surface (114+ WebSocket RPC methods from prior investigation), and the existing claude backend (pkg/spawn/claude.go) which proves the system already works without OpenCode.

**Knowledge:** The migration is not a backend swap — it's a three-layer refactoring: (1) abstract session operations behind an interface, (2) implement that interface for OpenClaw, (3) delete OpenCode. The claude backend serves as both fallback during migration and proof that the system works without OpenCode. The hardest part is not adding OpenClaw — it's untangling the 48 files that casually import opencode types for status display, token counting, and transcript access.

**Next:** Phase 1 (SessionClient interface extraction) can begin immediately as a no-behavior-change refactoring. Phase 2 (OpenClaw client) is gated on OpenClaw gateway running locally. Phase 3 (backend selection) is a config change. Phase 4 (deletion) is gated on stability.

**Authority:** architectural - This is a cross-component structural change affecting spawn, daemon, completion, and dashboard systems. Strategic decisions (whether to do it at all, when to drop OpenCode) were already made in the thread.

---

# Investigation: Design OpenClaw Worker Backend Migration

**Question:** If orch-go keeps its methodology and orchestrator conversation model, what is the cleanest way to replace the OpenCode worker path with an OpenClaw-backed execution layer?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-y4k6w
**Phase:** Complete
**Next Step:** None (design ready for implementation scheduling)
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md | extends | yes | none — confirms OpenClaw has no structural coordination (orch-go still needed) |
| .kb/investigations/2026-03-23-inv-investigate-openclaw-external-api-surface.md | extends | yes | none — provides the API mapping this design consumes |
| .kb/models/opencode-fork/probes/2026-03-24-probe-fork-necessity-assessment.md | extends | yes | none — confirms fork is maintenance weight on secondary backend |
| .kb/threads/2026-03-24-openclaw-migration-from-claude-code.md | implements | yes | none — this design realizes the decisions recorded there |

---

## Findings

### Finding 1: The backend boundary already exists at the right abstraction level

**Evidence:** `pkg/spawn/backends/backend.go` defines:
```go
type Backend interface {
    Name() string
    Spawn(ctx context.Context, req *SpawnRequest) (*Result, error)
}
```

The `SpawnRequest` carries `Config`, `ServerURL`, `MinimalPrompt`, `BeadsID`, `SkillName`, `Task`, `Attach`. The `Result` returns `SessionID`, `SpawnMode`, `TmuxInfo`, `RetryAttempts`.

Three backends implement this: `InlineBackend` (opencode CLI subprocess), `HeadlessBackend` (opencode HTTP API), `TmuxBackend` (opencode TUI in tmux window). The `claude` path bypasses all three via `SpawnClaude()` in `pkg/spawn/claude.go`.

**Source:** `pkg/spawn/backends/backend.go:1-89`, `pkg/spawn/claude.go:53-209`

**Significance:** Adding an OpenClaw backend means implementing this same interface. The spawn pipeline (`pkg/orch/spawn_pipeline.go`) doesn't care which backend runs — it only touches the `Backend` interface. This is the clean seam.

However, `SpawnRequest.ServerURL` is OpenCode-specific. It should become a generic endpoint field or be resolved inside the backend itself from config.

---

### Finding 2: OpenCode dependency extends far beyond spawn — 48 files, 6 usage categories

**Evidence:** 48 non-test files outside `pkg/opencode/` import the package. They break into distinct usage categories:

| Category | Files | What They Use | Migration Impact |
|----------|-------|---------------|------------------|
| **Spawn (3 backends)** | 3 | CreateSession, SendMessage, BuildSpawnCommand | Replace with OpenClaw client |
| **Session queries** | ~19 | ListSessions, GetSession | Replace with OpenClaw SessionsList |
| **Transcript/messages** | ~8 | GetMessages, ExtractRecentText | Replace with OpenClaw GetSessionMessages |
| **Status/monitoring** | ~7 | GetSessionStatusByID, Monitor (SSE), IsReachable | Replace with AgentWait or WebSocket events |
| **Types (passive)** | ~26 | Session, Message, TokenStats structs | Need equivalent types or interface |
| **Client construction** | ~all | NewClient(serverURL) | Replace with new client |

The type dependency is the stickiest. 26 files reference `opencode.Session`, `opencode.Message`, `opencode.TokenStats` as data structures for display, caching, and export — not for API calls. These need either:
- (a) A backend-agnostic type package (`pkg/session/types.go`)
- (b) OpenClaw-specific types replacing them
- (c) Conversion at the boundary

**Source:** `grep -r '"github.com/dylan-conlin/orch-go/pkg/opencode"' --include="*.go"` across the full codebase

**Significance:** The migration is not just "swap the spawn backend." It's untangling a type dependency that permeates the status dashboard, daemon, cleanup, and completion systems. The spawn interface is clean; the type leakage is the real work.

---

### Finding 3: The claude backend proves the system works without OpenCode

**Evidence:** `pkg/spawn/claude.go:SpawnClaude()` launches agents via tmux + `claude` CLI with zero OpenCode involvement:
- Creates tmux window directly
- Builds launch command with `BuildClaudeLaunchCommand()`
- Pipes SPAWN_CONTEXT.md to `claude --dangerously-skip-permissions`
- Returns `tmux.SpawnResult` (no session ID, no OpenCode session)

When using claude backend:
- Completion detection: beads `Phase: Complete` comments (via `bd comment`)
- Liveness: tmux window existence
- Status: tmux pane capture
- No SSE, no HTTP API polling, no session metadata

The claude backend is the existence proof that orch-go's orchestration layer (beads, skills, completion, workspace) doesn't inherently need OpenCode.

**Source:** `pkg/spawn/claude.go:162-209`, `pkg/verify/backend.go:86-108` (tmux verification path)

**Significance:** During migration, claude backend is not just a "fallback" — it's the stable ground. Any new OpenClaw backend only needs to be *better* than claude (headless operation, multi-model routing, programmatic session control), not replace something irreplaceable.

---

### Finding 4: OpenClaw's API surface maps 1:1 to orch-go's current operations, with improvements

**Evidence:** (From prior investigation `.kb/investigations/2026-03-23-inv-investigate-openclaw-external-api-surface.md`)

| orch-go Operation | OpenCode Method | OpenClaw Equivalent | Improvement |
|-------------------|-----------------|---------------------|-------------|
| Create session + send prompt | HTTP POST `/session` + POST `/session/{id}/prompt_async` (2 calls) | WebSocket `agent` (1 call) | Combined operation, idempotency key |
| Monitor progress | SSE stream parsing (fragile, noted in CLAUDE.md gotchas) | `agent.wait` polling or WebSocket events | No SSE parsing needed |
| Check status | HTTP GET | `sessions.list` or `health` | Same |
| Get transcript | HTTP GET `/session/{id}/message` | `getSessionMessages` | Same |
| Kill/cleanup | HTTP DELETE | `sessions.abort` + `sessions.delete` | Explicit abort |
| Inject context | File-based SPAWN_CONTEXT.md | `extraSystemPrompt` param | No file IO needed |
| Session metadata | Fork-specific PATCH `/session/{id}` | Session params at creation | No fork needed |

Key improvements:
- `agent.wait` replaces SSE parsing (the single most fragile integration in orch-go)
- `extraSystemPrompt` eliminates file-based context injection for headless spawns
- `idempotencyKey` provides built-in dedup (orch-go handles this manually today)
- Multi-model routing is built in (no model→backend resolution needed)

**Source:** Prior investigation, OpenClaw source at `~/Documents/personal/clawdbot/src/gateway/server-methods/agent.ts`

**Significance:** The OpenClaw migration is not a lateral move — it's an upgrade. Every orch-go operation maps to an OpenClaw equivalent, and several become simpler.

---

### Finding 5: Three Claude Code-specific assumptions pervade the worker plumbing

**Evidence:** Searching the codebase for assumptions that couple to specific backends:

**Assumption 1: Session IDs are OpenCode UUIDs**
- `spawn.WriteSessionID()` / `spawn.ReadSessionID()` write `.session_id` to workspace
- `verify/backend.go` reads session ID to fetch OpenCode transcripts
- Dashboard server caches sessions by OpenCode session ID
- These IDs are meaningless for claude backend (no session ID exists) and would be different format for OpenClaw

**Assumption 2: Completion detection has two paths (opencode transcript vs tmux capture)**
- `verify/backend.go:VerifyBackendDeliverables()` switches on backend type:
  - `opencode/headless` → check OpenCode transcript for "Phase: Complete"
  - `claude/tmux` → check tmux pane capture for "Phase: Complete"
- Neither path uses the authoritative source: beads comments via `bd`
- An OpenClaw path would need a third branch, OR the function should use beads as the single source

**Assumption 3: The daemon's CompletionService depends on OpenCode SSE**
- `pkg/daemon/completion.go:CompletionService` holds an `*opencode.Monitor` field
- It tracks headless sessions via SSE events for completion detection
- `StallTracker` takes `*opencode.TokenStats` directly
- The daemon cannot detect OpenClaw session completion without a new monitoring path

**Source:** `pkg/spawn/session.go`, `pkg/verify/backend.go:23-108`, `pkg/daemon/completion.go:26-49`, `pkg/daemon/stall_tracker.go:43-60`

**Significance:** These three assumptions define the migration's critical path. Session IDs need abstraction (or beads ID becomes the handle). Completion detection should converge on beads (the authority). Daemon monitoring needs an OpenClaw-aware path.

---

### Finding 6: Deletion inventory — what can go when OpenClaw becomes the worker substrate

**Evidence:** Complete inventory of OpenCode-dependent code:

**Can delete immediately when OpenCode backend is removed:**
| Component | Lines | Files | What it does |
|-----------|-------|-------|--------------|
| `pkg/opencode/` (production) | 2,396 | 9 | HTTP client, SSE, CLI output processing, types |
| `pkg/opencode/` (tests) | 3,425 | 4 | Tests for above |
| `pkg/spawn/backends/headless.go` | 123 | 1 | OpenCode HTTP API spawn |
| `pkg/spawn/backends/inline.go` | 75 | 1 | OpenCode CLI subprocess spawn |
| `pkg/spawn/backends/tmux.go` (opencode parts) | ~80 | 1 | Pre-creation via OpenCode API, opencode attach |
| `pkg/spawn/opencode_mcp.go` + test | ~200 | 2 | MCP config for OpenCode |
| `pkg/tmux/spawn_opencode.go` | ~50 | 1 | OpenCode TUI tmux helpers |
| **Subtotal** | **~6,350** | **19** | |

**Requires refactoring before deletion (type dependencies):**
| Component | Files | What needs to change |
|-----------|-------|---------------------|
| `cmd/orch/serve_agents_*.go` | 7 | Dashboard server uses opencode.Session/Message types |
| `cmd/orch/status_*.go` | 3 | Status display uses opencode types |
| `cmd/orch/sessions*.go` | 3 | Session listing uses opencode.ListSessions |
| `cmd/orch/tokens.go` | 1 | Token counting uses opencode.AggregateTokens |
| `cmd/orch/clean_*.go` | 3 | Cleanup uses opencode.ListSessions + DeleteSession |
| `cmd/orch/tail_cmd.go` | 1 | Transcript uses opencode.GetMessages |
| `cmd/orch/wait.go` | 1 | Wait uses opencode session discovery |
| `pkg/daemon/*.go` | 4 | Completion service, stall tracker, cleanup, recovery |
| `pkg/discovery/discovery.go` | 1 | Session discovery uses opencode.ListSessions |
| `pkg/sessions/sessions.go` | 1 | Session abstraction wraps opencode.ListSessions |
| `pkg/activity/export.go` | 1 | Activity export uses opencode types |
| `pkg/verify/backend.go` | 1 | Completion verification uses opencode transcript |
| `pkg/state/reconcile.go` | 1 | State reconciliation uses opencode sessions |
| **Subtotal** | **~28** | Require type migration or backend-agnostic interface |

**External deletion (outside orch-go):**
- OpenCode fork (`~/Documents/personal/opencode`) — stop maintaining, 975 commits behind
- `orch-dashboard` OpenCode service entry — remove from Procfile
- OpenCode port (4096) — no longer needed

**Total deletable:** ~6,350 LoC immediate + ~28 files refactored = significant simplification

**Source:** Full codebase analysis via grep, wc, and file examination

**Significance:** The deletion payoff is substantial: ~6,350 lines of direct OpenCode code, elimination of fork maintenance, and simplification of the dashboard service topology. But the ~28 files with type dependencies mean the cleanup is phased, not atomic.

---

## Synthesis

**Key Insights:**

1. **The migration has four natural phases, each independently valuable** — (1) Extract session interface, (2) Implement OpenClaw client, (3) Wire into backend selection, (4) Delete OpenCode. Each phase delivers value (cleaner abstractions → new capability → flexibility → simplification) and each has its own rollback point.

2. **The stickiest dependency is types, not API calls** — 26 files reference `opencode.Session`, `opencode.Message`, `opencode.TokenStats` as data structures. The API calls (session CRUD, message sending) are concentrated in ~15 files and cleanly replaceable. But the type leakage into the dashboard, status, export, and daemon systems means we need a backend-agnostic type layer *before* we can delete pkg/opencode/.

3. **Beads should become the single completion authority** — Today, completion verification has three paths (opencode transcript, tmux capture, beads comments) that sometimes disagree. The migration is an opportunity to converge on beads `Phase: Complete` comments as the single source. This eliminates the need for transcript access in completion verification entirely.

4. **The claude backend is the stability anchor** — It already works, has no OpenCode dependency, and handles all current production spawns. The migration adds OpenClaw as a *better* option for headless/automated spawns, not as a replacement for something that's broken.

**Answer to Investigation Question:**

The cleanest migration path has four phases with clear seams:

**Phase 1: SessionClient Interface (no behavior change)**
Extract a `pkg/execution/session.go` interface:
```go
type SessionClient interface {
    CreateAndPrompt(ctx context.Context, req SessionRequest) (SessionHandle, error)
    WaitForCompletion(ctx context.Context, handle SessionHandle, timeout time.Duration) (CompletionStatus, error)
    ListSessions(ctx context.Context, directory string) ([]SessionInfo, error)
    GetMessages(ctx context.Context, handle SessionHandle) ([]Message, error)
    DeleteSession(ctx context.Context, handle SessionHandle) error
    IsReachable(ctx context.Context) bool
}
```
Wrap `pkg/opencode/Client` to implement this interface. Update the ~48 importing files to use the interface. No behavior change — pure refactoring.

**Phase 2: OpenClaw Client (~300 LoC)**
Implement `SessionClient` for OpenClaw via WebSocket RPC. Methods map:
- `CreateAndPrompt` → WebSocket `agent` (combined create + prompt)
- `WaitForCompletion` → `agent.wait` (repeated with backoff for long tasks)
- `ListSessions` → `sessions.list`
- `GetMessages` → `getSessionMessages`
- `DeleteSession` → `sessions.delete`
- `IsReachable` → `health`

**Phase 3: Backend Selection**
Add `"openclaw"` to `DetermineSpawnBackend()`. Config gains `openclaw_url` and `openclaw_token`. Create `OpenClawBackend` implementing `backends.Backend` using the OpenClaw `SessionClient`.

**Phase 4: Deletion**
Delete `pkg/opencode/`, headless/inline backends, OpenCode-specific tmux paths, fork maintenance.

**Where the backend boundary sits:** At `backends.Backend` for spawn, and at `SessionClient` for everything else (status, transcript, cleanup, daemon monitoring). The orchestration layer (skills, beads, completion workflow, workspaces) sits above both interfaces and is untouched.

**What Claude Code assumptions need removal:**
1. Session IDs as OpenCode UUIDs → abstract to `SessionHandle` (string, could be OpenClaw runId or beads ID)
2. SSE monitoring → abstract to `WaitForCompletion`
3. Fork-specific metadata/TTL → OpenClaw handles session lifecycle natively

**How orchestration primitives are preserved:**
- Beads lifecycle: Unchanged (env vars + `bd comment` from agent)
- Completion workflow: `Phase: Complete` detected via beads (convergence, not new path)
- Workspace system: Unchanged (filesystem-based, backend-independent)
- Skill system: Unchanged (SPAWN_CONTEXT.md or `extraSystemPrompt`)
- Wait command: Uses `SessionClient.WaitForCompletion` instead of polling OpenCode

**How claude CLI fallback stays available:**
- `pkg/spawn/claude.go` is completely independent of OpenCode and OpenClaw
- `DetermineSpawnBackend` keeps `"claude"` as a valid backend
- Infrastructure escape hatch stays (auto-switch to claude for infra work)
- During migration, claude is the default; openclaw is opt-in via `--backend openclaw`

**What can be deleted:**
- `pkg/opencode/` — 5,821 LoC (production + tests)
- Headless + inline backends — ~200 LoC
- OpenCode fork — entire repository (~32 custom commits, 975 behind)
- OpenCode in orch-dashboard — one Procfile entry
- 48 files refactored to use SessionClient interface (code simplified, not deleted)

---

## Structured Uncertainty

**What's tested:**

- ✅ Backend interface (`backends.Backend`) cleanly decouples spawn from execution technology (verified: read interface definition and all three implementations)
- ✅ Claude backend works without any OpenCode dependency (verified: read `pkg/spawn/claude.go`, no opencode import)
- ✅ OpenClaw WebSocket API provides all operations orch-go needs (verified: prior investigation read agent.ts, sessions.ts source)
- ✅ 48 files import pkg/opencode/ with 6 distinct usage categories (verified: grep + file analysis)
- ✅ ~6,350 LoC directly deletable, ~28 files need refactoring (verified: wc + import analysis)

**What's untested:**

- ⚠️ `agent.wait` behavior for long-running sessions (30-60min) — may need repeated polling or event subscription
- ⚠️ `extraSystemPrompt` size limits — SPAWN_CONTEXT.md can be 10-20KB with full skill content
- ⚠️ WebSocket reconnection reliability under gateway restarts
- ⚠️ OpenClaw headless gateway running locally alongside Claude Code (port 18789 vs 4096 — should be fine)
- ⚠️ Token/usage data availability from OpenClaw sessions (stall tracker, token counting)
- ⚠️ Whether the `SessionClient` interface is sufficient for all 48 files' needs without excessive adaptation

**What would change this:**

- If `agent.wait` cannot handle sessions >30min, we'd need WebSocket event subscription instead of polling — more complex client
- If `extraSystemPrompt` has a <10KB limit, we'd still need file-based context injection (hybrid approach)
- If OpenClaw's token reporting doesn't include per-session breakdown, stall tracking would need a different signal
- If OpenClaw development velocity makes its API unstable, the interface layer becomes load-bearing (good to have either way)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Phase 1: Extract SessionClient interface | implementation | Pure refactoring within existing patterns, no behavior change |
| Phase 2: Build OpenClaw client | architectural | New package + external dependency, cross-component impact |
| Phase 3: Add openclaw to backend selection | architectural | Cross-component (spawn, daemon, dashboard), config schema change |
| Phase 4: Delete pkg/opencode/ | architectural | Removes major component, irreversible (but git has history) |
| Overall migration decision | strategic | Already made in thread (2026-03-24), this design implements it |

### Recommended Approach ⭐

**Phased Interface-First Migration** — Extract a SessionClient interface, implement it for OpenClaw, wire into backend selection, then delete OpenCode.

**Why this approach:**
- Each phase is independently shippable and reversible
- Phase 1 (interface extraction) improves code quality even if migration is abandoned
- The claude backend provides a stable fallback throughout
- No big-bang cutover — workloads migrate gradually via `--backend openclaw`
- Deletion (Phase 4) only happens after stability is proven

**Trade-offs accepted:**
- Phase 1 is pure refactoring overhead (no new capability)
- Maintaining three backends temporarily (claude, opencode, openclaw) during transition
- WebSocket client adds complexity vs stateless HTTP (but replaces fragile SSE)

**Implementation sequence:**

1. **Phase 1: SessionClient Interface** — Extract `pkg/execution/` with backend-agnostic types and interface. Wrap opencode client. Update imports. ~2-3 implementation sessions.
   - **Rollback:** Revert to direct opencode imports
   - **Verification:** `make test` passes, `orch status` works, `orch spawn` works

2. **Phase 2: OpenClaw Client** — Implement `pkg/openclaw/client.go` with WebSocket JSON-RPC. ~1-2 sessions.
   - **Prerequisite:** OpenClaw gateway running locally (`openclaw gateway --allow-unconfigured --port 18789`)
   - **Rollback:** Delete the package
   - **Verification:** Unit tests connect to local gateway, create session, send prompt, wait for completion

3. **Phase 3: Backend Selection** — Add `"openclaw"` to `DetermineSpawnBackend()`, create `OpenClawBackend`. ~1 session.
   - **Rollback:** Remove openclaw from backend selection
   - **Verification:** `orch spawn --backend openclaw feature-impl "test task"` creates and completes

4. **Phase 4: Deletion** — Remove `pkg/opencode/`, headless/inline backends, opencode tmux paths, update orch-dashboard. ~2-3 sessions.
   - **Prerequisite:** OpenClaw backend stable for 1+ week of real usage
   - **Rollback:** `git revert` (but shouldn't need it at this point)
   - **Verification:** `make test` passes, `orch status/spawn/complete` work, no opencode imports remain

### Alternative Approaches Considered

**Option B: Drop OpenCode first, add OpenClaw later (the thread's original order)**
- **Pros:** Immediate simplification, removes maintenance burden now
- **Cons:** Loses headless spawn capability until OpenClaw is added (claude backend is tmux-only); creates a capability gap where automated/daemon spawns break
- **When to use instead:** If daemon-driven spawns are paused anyway and headless isn't needed short-term

**Option C: Direct replacement (no interface layer)**
- **Pros:** Less code, faster initial implementation
- **Cons:** Couples to OpenClaw's API types directly (same mistake as current opencode coupling); no clean rollback path; harder to test
- **When to use instead:** If OpenClaw's API is known to be stable and the codebase will only ever have one execution backend

**Option D: OpenClaw plugin (move coordination INTO OpenClaw)**
- **Pros:** Reaches OpenClaw's 250K users; most powerful integration (in-process subagent control)
- **Cons:** Reimplements daemon, completion, beads, and skill system in TypeScript; abandons Go codebase; ~10x the effort
- **When to use instead:** Strategic pivot to distribution, not engineering optimization

**Rationale for recommendation:** Option A (phased interface-first) minimizes risk at each step, preserves rollback at each phase, and produces a cleaner codebase even if the migration stalls at Phase 1. Options B and C are faster but riskier. Option D is a different project.

---

### Implementation Details

**What to implement first:**

Phase 1 concrete steps:
1. Create `pkg/execution/types.go` with backend-agnostic types:
   ```go
   type SessionInfo struct { ID, Directory, Title string; Created, Updated time.Time; Metadata map[string]string }
   type SessionHandle string
   type CompletionStatus struct { Status string; Error string; Duration time.Duration }
   type Message struct { Role, Content string; Tokens TokenCount }
   type TokenCount struct { Input, Output, Reasoning, Cache int }
   ```
2. Create `pkg/execution/client.go` with `SessionClient` interface
3. Create `pkg/execution/opencode.go` wrapping `pkg/opencode/Client` → `SessionClient`
4. Update highest-impact importers first: `pkg/spawn/backends/`, `pkg/verify/backend.go`, `pkg/daemon/`
5. Update remaining cmd/orch files

**Things to watch out for:**
- ⚠️ **Defect class 2 (Multi-Backend Blindness):** Adding a third backend multiplies test combinations. Mitigation: the interface ensures all backends expose the same contract; test via interface, not implementation.
- ⚠️ **Defect class 3 (Stale Artifact Accumulation):** `.session_id` files written for OpenCode sessions won't exist for OpenClaw. Mitigation: make session ID optional throughout; beads ID is the authoritative handle.
- ⚠️ **Defect class 5 (Contradictory Authority Signals):** Backend selection logic is already complex (4 priority levels). Adding openclaw risks confusion. Mitigation: keep it simple — explicit `--backend openclaw` only, no auto-detection.
- ⚠️ **agent.wait default timeout is 30s** — orch-go agents run 30-60min. Need a polling loop: `while !terminal { agent.wait(timeout: 60s) }`.
- ⚠️ **Dashboard server** (`cmd/orch/serve_*.go`) deeply uses opencode types for agent cards, activity, and caching. This is the hardest refactoring in Phase 1.

**Areas needing further investigation:**
- OpenClaw token reporting format (does it expose per-session input/output/reasoning token counts?)
- OpenClaw session event stream (alternative to polling agent.wait for daemon monitoring)
- Whether `extraSystemPrompt` can replace SPAWN_CONTEXT.md entirely or needs to be supplemented with file injection
- How OpenClaw handles workspace/directory context for Claude CLI agents (the `coding-agent` skill does this, but with its own conventions)

**Success criteria:**
- ✅ `orch spawn --backend openclaw feature-impl "test task"` creates agent, sends prompt, returns handle
- ✅ `orch wait <handle>` blocks until OpenClaw agent completes
- ✅ `orch status` shows both claude and openclaw agents
- ✅ `orch complete <handle>` verifies deliverables from OpenClaw session
- ✅ Daemon can detect OpenClaw agent completion and trigger auto-complete
- ✅ No `pkg/opencode/` imports remain after Phase 4
- ✅ `make test` passes at every phase boundary

---

## Defect Class Exposure

| Defect Class | Exposure | Mitigation |
|-------------|----------|------------|
| Class 2: Multi-Backend Blindness | HIGH — third backend (openclaw) joins claude and opencode | SessionClient interface ensures uniform contract; test all operations through interface |
| Class 3: Stale Artifact Accumulation | MEDIUM — .session_id files, opencode session metadata won't exist for openclaw | Make session ID optional; beads ID as primary handle |
| Class 5: Contradictory Authority Signals | MEDIUM — backend selection gains third option | Explicit opt-in only (`--backend openclaw`); no auto-detection for openclaw |
| Class 7: Premature Destruction | LOW — only during Phase 4 deletion | Gate deletion on stability period (1+ week of real usage) |

---

## References

**Files Examined:**
- `pkg/spawn/backends/backend.go` — Backend interface definition (the clean seam)
- `pkg/spawn/backends/headless.go` — OpenCode HTTP API spawn (to be replaced)
- `pkg/spawn/backends/inline.go` — OpenCode CLI subprocess spawn (to be deleted)
- `pkg/spawn/backends/tmux.go` — OpenCode TUI tmux spawn (to be refactored)
- `pkg/spawn/claude.go` — Claude CLI spawn (the stability anchor)
- `pkg/orch/spawn_backend.go` — Backend selection logic (to gain openclaw option)
- `pkg/opencode/client.go` — OpenCode HTTP client (to be replaced by openclaw client)
- `pkg/opencode/types.go` — OpenCode types (pervasive dependency, needs abstraction)
- `pkg/opencode/sse.go` — SSE streaming (to be replaced by agent.wait)
- `pkg/opencode/monitor.go` — SSE-based session monitoring (daemon dependency)
- `pkg/verify/backend.go` — Completion verification (two paths, should converge on beads)
- `pkg/daemon/completion.go` — Completion service (holds opencode.Monitor, needs interface)
- `pkg/daemon/stall_tracker.go` — Stall detection (uses opencode.TokenStats, needs abstraction)

**Commands Run:**
```bash
# Count opencode package production lines
wc -l pkg/opencode/*.go | grep -v '_test.go' | grep -v total
# → 2,396 LoC across 9 files

# Count opencode package test lines
wc -l pkg/opencode/*_test.go | tail -1
# → 3,425 LoC

# Find all non-test importers of opencode package
grep -r '"github.com/dylan-conlin/orch-go/pkg/opencode"' --include="*.go" | grep -v '_test.go' | grep -v 'pkg/opencode/' | awk -F: '{print $1}' | sort -u
# → 48 files
```

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md` — Strategic decision to migrate
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-openclaw-external-api-surface.md` — OpenClaw API mapping
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md` — OpenClaw platform capabilities
- **Probe:** `.kb/models/opencode-fork/probes/2026-03-24-probe-fork-necessity-assessment.md` — Fork maintenance burden quantified

---

## Investigation History

**2026-03-26 ~09:00:** Investigation started
- Initial question: What is the cleanest way to replace OpenCode worker path with OpenClaw?
- Context: Strategic decision made 2026-03-24 to migrate workers to OpenClaw; this designs the implementation

**2026-03-26 ~09:10:** Parallel exploration of spawn architecture and prior investigations
- Read all 4 prior artifacts (2 investigations, 1 probe, 1 thread)
- Explored full spawn backend architecture (backends/, claude.go, spawn_backend.go)
- Identified Backend interface as the clean spawn seam

**2026-03-26 ~09:25:** Dependency analysis phase
- Counted 48 files importing pkg/opencode/ outside the package
- Classified into 6 usage categories
- Identified type leakage as the stickiest dependency (26 files use opencode types passively)

**2026-03-26 ~09:40:** Fork analysis — 5 design forks identified
- Fork 1: Interface-first vs direct replacement → Interface-first (enables rollback)
- Fork 2: Drop OpenCode first vs add OpenClaw first → Add first (preserves headless capability)
- Fork 3: New type package vs reuse OpenClaw types → New type package (backend-agnostic)
- Fork 4: Completion via transcript vs beads convergence → Beads convergence (single authority)
- Fork 5: Auto-detect openclaw vs explicit opt-in → Explicit opt-in (reduce Class 5 risk)

**2026-03-26 ~10:00:** Investigation completed
- Status: Complete
- Key outcome: 4-phase migration design with concrete seams, rollback points, and deletion inventory
