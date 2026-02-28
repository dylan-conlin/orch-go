# Design: Backend-Agnostic Session Contract

**Date:** 2026-02-14
**Phase:** Complete
**Status:** Complete
**Beads:** orch-go-he8
**Type:** Architecture Design
**Related Models:** agent-lifecycle-state-model.md, spawn-architecture.md, opencode-session-lifecycle.md
**Related Decisions:** 2026-02-14-lifecycle-ownership-own-accept-build.md, 2026-02-13-lifecycle-ownership-boundaries.md

---

## Design Question

What session contract should orch-go use across both OpenCode (HTTP API, SQLite sessions) and Claude CLI (subprocess, tmux, no session object) backends, to enable removing ~800 lines of `.session_id` cross-reference code?

## Success Criteria

1. Single contract works for both backends without backend-specific branching in consumers
2. Enables removal of individual dotfiles (.beads_id, .tier, .spawn_time, .spawn_mode) that duplicate AGENT_MANIFEST.json
3. Preserves graceful degradation (Claude CLI agents remain fully manageable)
4. Aligns with the state vs infrastructure distinction from lifecycle ownership decision
5. Doesn't over-engineer (no abstract interface for 2 backends)

## Constraints

- **Local-First** (principle): Files on disk, versionable, inspectable with Unix tools
- **Self-Describing Artifacts** (principle): The contract file must contain its own operating instructions
- **Session Amnesia** (principle): Next Claude must be able to resume by finding these files
- **Share Patterns Not Tools** (principle): File format is the contract, not shared code
- **Escape Hatches** (principle): Claude CLI must work independently of OpenCode
- **Lifecycle Ownership** (decision): State layers (orch-owned) vs infrastructure layers (orch-uses)

---

## Problem Framing

### Current State

orch-go has **two spawn backends** with different session models:

| Aspect | OpenCode (headless/inline/tmux) | Claude CLI |
|--------|--------------------------------|------------|
| Session object | OpenCode Session (SQLite, HTTP API) | None |
| Session ID | `ses_abc...` (written to .session_id) | Not applicable |
| Status query | OpenCode API (`GET /session/status`) | tmux pane capture |
| Completion detection | SSE events or API poll | tmux output scan |
| Transcript | API message history | tmux scrollback |
| Persistence | OpenCode SQLite DB | tmux session (volatile) |

Both backends write **identical workspace state files**:
- `.beads_id` — issue tracking ID
- `.tier` — "light" or "full"
- `.spawn_time` — Unix nanoseconds
- `.spawn_mode` — "headless", "inline", "tmux", or "claude"
- `AGENT_MANIFEST.json` — all above plus workspace_name, skill, project_dir, git_baseline, model

The Claude CLI backend does NOT write `.session_id` because no OpenCode session exists.

### The Redundancy Problem

AGENT_MANIFEST.json already contains every field stored in the individual dotfiles:

| Dotfile | AGENT_MANIFEST.json field | Redundant? |
|---------|--------------------------|-----------|
| `.beads_id` | `beads_id` | Yes |
| `.tier` | `tier` | Yes |
| `.spawn_time` | `spawn_time` | Yes |
| `.spawn_mode` | `spawn_mode` | Yes |
| `.session_id` | Not present | No (infrastructure reference) |

The ~800 lines of cross-reference code exist because consumers read individual dotfiles, cross-reference with OpenCode sessions, and reconcile. If consumers read AGENT_MANIFEST.json instead, the cross-reference collapses.

### What Consumers Actually Need

| Consumer | Fields Required | Currently Reads |
|----------|----------------|-----------------|
| `orch status` | beads_id, spawn_mode (for routing) | .session_id + .beads_id + OpenCode API + tmux |
| `orch complete` | beads_id, tier, spawn_mode | .beads_id + .tier + .spawn_mode + .session_id |
| `orch abandon` | beads_id, spawn_time, session_id (transcript export) | .beads_id + .spawn_time + .session_id |
| `orch clean` | spawn_time, tier, beads_id | .spawn_time + .tier + .beads_id |
| `state/reconcile` | session_id (OpenCode check), beads_id | .session_id + .beads_id |
| `verify/backend` | spawn_mode (routing) | .spawn_mode |
| `daemon` | beads status | bd CLI (not workspace files) |

Key observations:
- **Every consumer needs beads_id** — it's the universal lookup key
- **session_id is only needed for OpenCode-specific operations** (transcript fetch, liveness check)
- **spawn_mode is needed for routing** (which infrastructure to query)
- **All state fields are in AGENT_MANIFEST.json**

---

## Exploration: Three Options Evaluated

### Fork 1: Contract File Format

**Options:**
- A: Keep individual dotfiles as contract
- B: SESSION_STATE.json (new file mimicking OpenCode session shape)
- C: AGENT_MANIFEST.json (existing file, already universal)

**Substrate says:**
- Principle (Local-First): Files on disk — all three options satisfy this
- Principle (Self-Describing Artifacts): JSON with clear schema > scattered dotfiles
- Model (Agent Lifecycle): AGENT_MANIFEST.json already written by all backends
- Decision (Lifecycle Ownership): State should be orch-owned — AGENT_MANIFEST is

**Recommendation:** Option C (AGENT_MANIFEST.json). It already exists, is already written by all backends, already contains all state fields. Creating SESSION_STATE.json would duplicate it. Keeping dotfiles preserves redundancy.

**Trade-off accepted:** Consumers need to parse JSON instead of reading simple text files. Acceptable because `ReadAgentManifest()` already exists and is O(1) per workspace.

### Fork 2: Where Should session_id Live?

**Options:**
- A: Add session_id to AGENT_MANIFEST.json
- B: Keep session_id as separate .session_id file
- C: Look up session_id from OpenCode API at query time (no local storage)

**Substrate says:**
- Model (Agent Lifecycle): session_id is an infrastructure reference, not state
- Model (Spawn Architecture): .session_id is written post-spawn by backends, sometimes after retries
- Decision (Lifecycle Ownership): Infrastructure handles are "Accept" bucket — work with them

**Recommendation:** Option B (keep .session_id separate). Rationale:
1. session_id is an **infrastructure handle**, not spawn-time state — it doesn't belong in the immutable manifest
2. session_id is written **after** AGENT_MANIFEST (sometimes after retries) — adding it would require re-writing the manifest
3. For Claude CLI, session_id will never exist — having it absent from a separate file is cleaner than having `"session_id": ""` in the manifest
4. The `pkg/spawn/session.go` Read/WriteSessionID functions are clean and atomic

**Trade-off accepted:** Two files to manage per workspace (AGENT_MANIFEST.json + .session_id) instead of one. Acceptable because they serve different purposes with different write timings.

### Fork 3: Backward Compatibility Strategy

**Options:**
- A: Hard cutover — delete dotfiles, break old workspaces
- B: Read AGENT_MANIFEST first, fall back to dotfiles, stop writing dotfiles
- C: Read AGENT_MANIFEST first, fall back to dotfiles, keep writing both (temporary)

**Substrate says:**
- Principle (Graceful Degradation): Core functionality works without optional layers
- Principle (Session Amnesia): Old workspaces must remain readable
- Practical: Existing workspaces in .orch/workspace/ have dotfiles but AGENT_MANIFEST.json is newer — some may not have it

**Recommendation:** Option C (dual-write + read-with-fallback) as a transition, then Option B once all active workspaces have AGENT_MANIFEST.json.

Phase 1: `ReadAgentState()` reads AGENT_MANIFEST.json first, falls back to individual dotfiles
Phase 2: After 2 weeks (all active workspaces refreshed), stop writing individual dotfiles except .session_id
Phase 3: Remove dotfile read fallback after archiving old workspaces

### Fork 4: Consumer Migration Pattern

**Options:**
- A: Each consumer reads AGENT_MANIFEST directly (scattered)
- B: Single `ReadAgentState()` function returns typed struct (centralized)

**Substrate says:**
- Principle (Compose Over Monolith): Each command does one thing well
- Current pattern: `spawn.ReadAgentManifest()` already exists and returns `*AgentManifest`
- Practical: 7 consumers read different subsets of fields

**Recommendation:** Option B. `ReadAgentManifest()` already exists. Consumers call it once, destructure what they need. This is already the pattern — just not universally adopted. The migration is: replace `os.ReadFile(filepath.Join(workspace, ".beads_id"))` with `manifest, _ := spawn.ReadAgentManifest(workspace); beadsID := manifest.BeadsID`.

### Fork 5: How Does Status Query Work for Claude CLI Agents?

**Options:**
- A: Skip OpenCode liveness check entirely when spawn_mode="claude"
- B: Check tmux window liveness as the infrastructure signal
- C: Both (check spawn_mode, route to appropriate infrastructure check)

**Substrate says:**
- Model (Agent Lifecycle): Priority Cascade checks beads first (highest authority)
- Current code: `state/reconcile.go` already checks tmux windows via `checkTmuxWindow()`
- Model (Dashboard Agent Status): Tmux is "UI layer only" — Low authority

**Recommendation:** Option C. Read spawn_mode from AGENT_MANIFEST.json, route:
- `spawn_mode == "claude"` → check tmux window (infrastructure), skip OpenCode
- `spawn_mode != "claude"` → check OpenCode session (infrastructure), optionally check tmux

This is already what the code does, but implicitly (session_id missing → OpenCode check fails → falls through). Making it explicit via spawn_mode avoids the failed-lookup overhead.

### Fork 6: OpenCode Metadata API Integration (Future)

**Options:**
- A: AGENT_MANIFEST.json replaces OpenCode metadata entirely
- B: Both exist — OpenCode metadata for fast path, AGENT_MANIFEST for local truth
- C: OpenCode metadata replaces AGENT_MANIFEST.json

**Substrate says:**
- Decision (Lifecycle Ownership Build): Phase 5 Step 2 adds metadata API to OpenCode fork
- Principle (Escape Hatches): Claude CLI must work without OpenCode
- Principle (Local-First): Files on disk survive service shutdowns

**Recommendation:** Option B. When the metadata API ships:
- OpenCode backends: write metadata to BOTH OpenCode session AND AGENT_MANIFEST.json
- Claude CLI: write ONLY to AGENT_MANIFEST.json (no OpenCode session exists)
- Status queries: try OpenCode metadata first (in-memory, fast), fall back to AGENT_MANIFEST.json
- This gives the best of both worlds: fast queries via API + reliable local state

---

## Synthesis: The Recommended Contract

### AGENT_MANIFEST.json IS the Session Contract

The answer was already in the codebase. AGENT_MANIFEST.json is the backend-agnostic session contract. It:

1. **Is written by all backends** (OpenCode and Claude CLI) — `spawn.WriteAgentManifest()`
2. **Contains all spawn-time state** — workspace_name, skill, beads_id, project_dir, git_baseline, spawn_time, tier, spawn_mode, model
3. **Is structured JSON** — typed, parseable, self-describing
4. **Is immutable post-spawn** — spawn-time metadata never changes
5. **Has existing read/write functions** — `spawn.ReadAgentManifest()` / `spawn.WriteAgentManifest()`

### What Changes

**Keep separately:**
- `.session_id` — infrastructure handle, written post-spawn, absent for Claude CLI

**Deprecate (read with fallback, stop writing):**
- `.beads_id` — already in AGENT_MANIFEST.json
- `.tier` — already in AGENT_MANIFEST.json
- `.spawn_time` — already in AGENT_MANIFEST.json
- `.spawn_mode` — already in AGENT_MANIFEST.json

**No new files needed.** SESSION_STATE.json is unnecessary — AGENT_MANIFEST.json already IS that file.

### Architecture After Migration

```
.orch/workspace/{agent-name}/
├── AGENT_MANIFEST.json    ← Session contract (all state, both backends)
├── .session_id            ← Infrastructure handle (OpenCode only, optional)
├── SPAWN_CONTEXT.md       ← Agent instructions (immutable)
├── SYNTHESIS.md           ← Agent output (full tier only)
└── [agent outputs]
```

### Consumer Migration (Estimated ~300 lines removed)

Each consumer replaces multiple `os.ReadFile()` + `strings.TrimSpace()` calls with one `spawn.ReadAgentManifest()`:

```go
// Before (typical consumer reads 3-4 files):
beadsData, _ := os.ReadFile(filepath.Join(ws, ".beads_id"))
beadsID := strings.TrimSpace(string(beadsData))
tierData, _ := os.ReadFile(filepath.Join(ws, ".tier"))
tier := strings.TrimSpace(string(tierData))
modeData, _ := os.ReadFile(filepath.Join(ws, ".spawn_mode"))
mode := strings.TrimSpace(string(modeData))

// After (one structured read):
manifest, err := spawn.ReadAgentManifest(ws)
if err != nil {
    // Fallback to dotfiles for backward compat
    manifest = readLegacyDotfiles(ws)
}
beadsID := manifest.BeadsID
tier := manifest.Tier
mode := manifest.SpawnMode
```

### Status Query Routing

```
orch status {agent}
    ↓
Read AGENT_MANIFEST.json → get spawn_mode
    ↓
spawn_mode == "claude"?
    ├── YES → check tmux window (infrastructure)
    └── NO  → check OpenCode session via .session_id (infrastructure)
    ↓
Check beads issue (state, highest authority)
    ↓
Reconcile: state + infrastructure → display status
```

### Alignment with Lifecycle Ownership Decision

| Bucket | Before | After |
|--------|--------|-------|
| **Own** | 6 dotfiles + AGENT_MANIFEST | AGENT_MANIFEST.json (single contract) |
| **Accept** | .session_id for OpenCode | .session_id for OpenCode (unchanged) |
| **Build** | OpenCode metadata API | Writes to both places when available |

---

## Implementation-Ready Checklist

### Required sections
- [x] Problem statement — removing ~800 lines of cross-reference via unified contract
- [x] Approach — AGENT_MANIFEST.json is the contract, deprecate dotfiles
- [x] File targets (for implementation agent):
  - `pkg/spawn/session.go` — add `ReadAgentState()` with fallback
  - `cmd/orch/complete_cmd.go` — migrate to ReadAgentManifest
  - `cmd/orch/clean_cmd.go` — migrate to ReadAgentManifest
  - `cmd/orch/status_cmd.go` — migrate to ReadAgentManifest
  - `cmd/orch/abandon_cmd.go` — migrate to ReadAgentManifest
  - `pkg/state/reconcile.go` — use spawn_mode for routing
  - `pkg/verify/backend.go` — already reads .spawn_mode, migrate
  - `cmd/orch/shared.go` — migrate resolveSessionID
  - `cmd/orch/doctor.go` — migrate workspace scanning
  - `cmd/orch/serve_agents.go` — migrate agent listing
- [x] Acceptance criteria:
  1. All consumers read AGENT_MANIFEST.json via `ReadAgentManifest()` with dotfile fallback
  2. `orch status` shows correct status for both OpenCode and Claude CLI agents
  3. `orch complete` verifies correctly for both backends
  4. `orch clean` archives correctly for both backends
  5. No new files created — AGENT_MANIFEST.json is the contract
  6. Backward compatible — old workspaces with only dotfiles still work
- [x] Out of scope:
  - OpenCode metadata API integration (Phase 5 Step 2 — separate issue)
  - Removing dotfile write path (Phase 2 — after transition period)
  - Abstract SessionStore interface (rejected as over-engineering)

### Trade-offs considered
- SESSION_STATE.json rejected: duplicates AGENT_MANIFEST.json
- Abstract interface rejected: over-engineering for 2 backends
- session_id in AGENT_MANIFEST rejected: infrastructure handle ≠ immutable state

### Phasing
- **Phase 1** (1-2 days): Add `ReadAgentState()` with fallback, migrate highest-impact consumers (status, complete, clean)
- **Phase 2** (1 day): Migrate remaining consumers (doctor, serve_agents, shared)
- **Phase 3** (after 2 weeks): Stop writing individual dotfiles (except .session_id)
- **Phase 4** (after archiving old workspaces): Remove dotfile read fallback

---

## Recommendations

**RECOMMENDED:** Consolidate on AGENT_MANIFEST.json as the backend-agnostic session contract
- **Why:** Already exists, already universal, already structured — this is evolution, not invention
- **Trade-off:** JSON parsing overhead per consumer read (negligible — one file read + unmarshal vs 4-6 file reads)
- **Expected outcome:** ~300 lines of dotfile reading code eliminated, explicit spawn_mode routing replaces implicit fallthrough

**Alternative: SESSION_STATE.json (new file)**
- **Pros:** Clean-slate design, could include runtime state
- **Cons:** Duplicates AGENT_MANIFEST.json, requires writing new file + new read/write functions
- **When to choose:** If AGENT_MANIFEST.json needs to remain strictly immutable and runtime state needs to be tracked locally (e.g., last_activity timestamp). Currently no consumer needs runtime state in workspace files.

**Alternative: Abstract SessionStore interface**
- **Pros:** Clean abstraction, testable, enables future migration
- **Cons:** Over-engineering for 2 backends, adds indirection, filesystem already IS the interface
- **When to choose:** If orch-go needed to support 5+ backends or if the storage mechanism was genuinely uncertain. With 2 known backends and a clear file-based contract, this adds complexity without benefit.

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves recurring session state cross-reference issues
- Future spawn backend additions must follow this contract

**Suggested blocks keywords:**
- session contract
- backend agnostic
- AGENT_MANIFEST
- spawn backend
- workspace state
