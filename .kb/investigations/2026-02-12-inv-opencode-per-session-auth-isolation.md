## Summary (D.E.K.N.)

**Delta:** OpenCode cannot support per-session auth isolation — auth is global to the server process via a shared auth.json, with no per-session override in the API. Three viable alternatives exist with different trade-off profiles.

**Evidence:** Traced auth flow through OpenCode source: `Auth.get()` reads `~/.local/share/opencode/auth.json` on every API call (no cache), provider state cached per-instance, session create API accepts no auth parameters. Anthropic plugin's custom `fetch` wrapper re-reads auth.json on every request.

**Knowledge:** The auth.json re-read-on-every-call behavior means sequential switching "works" but creates a TOCTOU race when orchestrator and workers share the same server. Two-server approach (via `XDG_DATA_HOME`) provides clean isolation. Claude CLI backend (`CLAUDE_CONFIG_DIR`) already provides per-process isolation today.

**Next:** Architectural decision needed — recommend two-server approach for OpenCode-backed workers, with Claude CLI as the proven fallback for immediate use.

**Authority:** architectural — Cross-component decision affecting spawn pipeline, server management, and multi-account workflow.

---

# Investigation: OpenCode Per-Session Auth Isolation

**Question:** Can OpenCode support per-session auth isolation so orchestrator (personal Max 5x) and workers (work Max 20x) use different accounts simultaneously?

**Started:** 2026-02-12
**Updated:** 2026-02-12
**Owner:** orch-go-49029
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/current-model-stack/probes/2026-02-12-split-orchestrator-worker-account-quota.md | extends | yes | No conflicts — probe correctly identified Claude CLI as clean path and OpenCode as having race risk. This investigation deepens the OpenCode analysis. |

---

## Findings

### Finding 1: Auth.get() reads from disk on every call — no in-memory cache

**Evidence:** `packages/opencode/src/auth/index.ts:39-46`:
```typescript
export async function get(providerID: string) {
  const auth = await all()
  return auth[providerID]
}
export async function all(): Promise<Record<string, Info>> {
  const file = Bun.file(filepath)
  const data = await file.json().catch(() => ({}) as Record<string, unknown>)
  // ...parse and return
}
```
Every call to `Auth.get()` opens and parses `auth.json` from disk. No caching layer. The filepath is `Global.Path.data + "/auth.json"` which resolves to `~/.local/share/opencode/auth.json`.

**Source:** `packages/opencode/src/auth/index.ts:37-56`, `packages/opencode/src/global/index.ts:8`

**Significance:** This means auth.json is effectively a global mutable variable. Any process writing to it affects all subsequent reads by any session in the same server. This is the root of the race condition.

---

### Finding 2: Anthropic auth plugin re-reads auth on EVERY API request

**Evidence:** The `opencode-anthropic-auth` plugin (`~/.cache/opencode/node_modules/opencode-anthropic-auth/index.mjs:104-106`) provides a custom `fetch` wrapper:
```javascript
async fetch(input, init) {
  const auth = await getAuth();  // calls Auth.get("anthropic") → reads auth.json
  if (auth.type !== "oauth") return fetch(input, init);
  // ... uses auth.access token for request
}
```
The `getAuth` function is a closure over `() => Auth.get(providerID)` passed by provider.ts:877.

**Source:** `~/.cache/opencode/node_modules/opencode-anthropic-auth/index.mjs:84-306`, `packages/opencode/src/provider/provider.ts:877`

**Significance:** Even after provider initialization, every Anthropic API call reads fresh auth from disk. If `maybeSwitchSpawnAccount()` writes work-account tokens to auth.json, the orchestrator's next API call will use those tokens — breaking the orchestrator's session auth.

---

### Finding 3: Provider state is cached per-instance, but auth flows through plugin fetch wrappers

**Evidence:** Provider state initialization (`provider.ts:696-975`) calls `Auth.all()` once during init to set `provider.key`. This is cached via `Instance.state()` → `State.create()` (keyed by project directory). However, for Anthropic OAuth, the plugin sets `apiKey: ""` and provides a custom `fetch` wrapper that bypasses the cached key entirely — it calls `getAuth()` on every request to get the OAuth bearer token.

The SDK itself is also cached per hash of options (`provider.ts:1011-1013`), but the `fetch` wrapper is part of those options, so the per-request auth read is baked into the cached SDK.

**Source:** `packages/opencode/src/provider/provider.ts:846-975`, `packages/opencode/src/project/state.ts:12-29`

**Significance:** There are two auth paths: (1) API key providers cache the key at init, (2) OAuth providers (Anthropic, Codex) re-read on every request via custom fetch. Anthropic Max uses path (2), making it vulnerable to auth.json mutations.

---

### Finding 4: Global.Path.data is scoped by XDG_DATA_HOME — viable for two-server isolation

**Evidence:**
```typescript
// global/index.ts:8
const data = path.join(xdgData!, app)  // xdgData = $XDG_DATA_HOME or ~/.local/share
```

Verified that `XDG_DATA_HOME` override works:
```bash
$ XDG_DATA_HOME=/tmp/opencode-test-data bun -e 'import { xdgData } from "xdg-basedir"; console.log(xdgData)'
/tmp/opencode-test-data
```

`opencode serve` accepts `--port` flag, allowing a second server on a different port.

**Source:** `packages/opencode/src/global/index.ts:1-8`, verified via `bun -e` test

**Significance:** Running `XDG_DATA_HOME=~/.local/share/opencode-work opencode serve --port 4097` would create a second OpenCode server with its own `auth.json` at `~/.local/share/opencode-work/opencode/auth.json`. Workers could be routed to port 4097 via `--server http://127.0.0.1:4097`.

---

### Finding 5: Session create API has no auth override capability

**Evidence:** The session create endpoint (`server/routes/session.ts:185-208`) accepts:
```typescript
z.object({
  parentID: Identifier.schema("session").optional(),
  title: z.string().optional(),
  permission: Info.shape.permission,
}).optional()
```

No auth, provider, or account fields. The session inherits whatever auth the server's `Auth.get()` returns at API call time.

**Source:** `packages/opencode/src/server/routes/session.ts:140-156`, `packages/opencode/src/session/index.ts:140-156`

**Significance:** There is no way to pass auth tokens per-session through the HTTP API. Per-session isolation would require OpenCode source changes.

---

### Finding 6: OPENCODE_CONFIG_DIR does NOT scope auth.json

**Evidence:** `OPENCODE_CONFIG_DIR` affects:
- Config file loading (`config/config.ts:143-145`)
- AGENTS.md instruction loading (`session/instruction.ts:21-22`)
- Relative instruction resolution (`session/instruction.ts:35-41`)

It does NOT affect `Global.Path.data` (where auth.json lives). `Global.Path.data` is controlled only by `XDG_DATA_HOME`.

**Source:** `packages/opencode/src/flag/flag.ts:76-78`, `packages/opencode/src/config/config.ts:143-151`, `packages/opencode/src/global/index.ts:8`

**Significance:** `OPENCODE_CONFIG_DIR` cannot be used to scope auth per-instance. This eliminates one potential approach.

---

## Synthesis

**Key Insights:**

1. **Auth.json is a global shared mutable resource** — All sessions in an OpenCode server read from the same auth.json file on every Anthropic API call. There is no per-session, per-instance, or per-provider scoping mechanism.

2. **Two clean isolation boundaries exist today** — (a) `CLAUDE_CONFIG_DIR` for Claude CLI backend (per-process keychain, proven working), (b) `XDG_DATA_HOME` for OpenCode server (per-process data dir, gives independent auth.json, not yet wired).

3. **Sequential auth.json switching is a TOCTOU race** — `maybeSwitchSpawnAccount()` writes to auth.json before spawn, but the orchestrator's ongoing sessions will read the new auth on their next API call. Serialization alone doesn't fix this because the orchestrator makes API calls concurrently with workers.

**Answer to Investigation Question:**

No, OpenCode cannot support per-session auth isolation. Auth is global to the server process. Three alternatives exist:

| Approach | Isolation | Effort | Ready |
|----------|-----------|--------|-------|
| `--backend claude --account work` | Per-process (CLAUDE_CONFIG_DIR) | None | Today |
| Two OpenCode servers (XDG_DATA_HOME) | Per-server (independent auth.json) | Medium (wire up orch-go) | Needs work |
| Sequential auth.json switching | Time-based (TOCTOU race) | None | Working, risky |

---

## Structured Uncertainty

**What's tested:**

- ✅ Auth.get() reads from disk on every call (verified: read source, traced call path)
- ✅ XDG_DATA_HOME override changes xdgData resolution (verified: `bun -e` test)
- ✅ `opencode serve` accepts `--port` flag (verified: `--help` output)
- ✅ OPENCODE_CONFIG_DIR does NOT affect auth.json path (verified: source trace)
- ✅ Session create API has no auth parameter (verified: route handler + zod schema)
- ✅ Claude CLI backend isolation works (verified: prior probe with 3 passing test suites)

**What's untested:**

- ⚠️ Actually running two OpenCode servers simultaneously with different XDG_DATA_HOME (theoretically sound, not end-to-end tested)
- ⚠️ Whether the second server's OAuth token refresh flow works in isolation (should work since refresh is per-auth.json)
- ⚠️ Whether orch-dashboard can manage two servers (would need script changes)

**What would change this:**

- If OpenCode added per-session auth (e.g., auth token in session create body), the two-server approach would be unnecessary
- If Anthropic plugin cached auth in memory, sequential switching would be safer (but still racey at process level)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Use `--backend claude` for split-quota now | implementation | Uses existing proven mechanism, no changes needed |
| Build two-server support for OpenCode workers | architectural | Affects spawn pipeline, server management, dashboard, process reaper |
| Add per-session auth to OpenCode fork | strategic | Major fork divergence, maintenance burden, upstream incompatibility |

### Recommended Approach ⭐

**Two-tier: Claude CLI now, two-server later** — Use `--backend claude --account work` immediately for split-quota Claude workers. Build two-server support as a follow-up for workers that need OpenCode's headless API.

**Why this approach:**
- Claude CLI backend is proven, tested, zero-effort
- Two-server approach provides clean isolation without OpenCode source changes
- Avoids risky auth.json race condition entirely

**Trade-offs accepted:**
- Claude CLI workers get tmux windows (more visible, slightly more overhead)
- Two-server approach needs `orch-dashboard` changes and a second OAuth login

**Implementation sequence:**
1. **Immediate**: Configure default worker account in `.orch/config.yaml` (`spawn_account: work`) and use `--backend claude` for Claude workers needing work account
2. **Follow-up**: Add `worker_server` config key to orch-go that points to a second OpenCode server URL (e.g., `http://127.0.0.1:4097`)
3. **Follow-up**: Extend `orch-dashboard` to start/manage the worker OpenCode server with `XDG_DATA_HOME` override

### Alternative Approaches Considered

**Option B: Sequential auth.json switching (current behavior)**
- **Pros:** Works today with zero changes; race window is small
- **Cons:** TOCTOU race when orchestrator makes Anthropic API calls during worker spawns; unreliable under high concurrency
- **When to use instead:** Acceptable for low-concurrency, manual spawns where orchestrator isn't actively using Claude

**Option C: Add per-session auth to OpenCode fork**
- **Pros:** Perfect isolation within a single server; no extra processes
- **Cons:** Major fork divergence; requires changes to session create API, provider resolution, plugin auth interface; maintenance burden
- **When to use instead:** Only if two-server overhead becomes unacceptable and fork maintenance cost is accepted

**Rationale for recommendation:** Option A provides immediate value (Claude CLI) and a clean upgrade path (two-server) without OpenCode source changes. The race condition in Option B is a real concern given the anthropic plugin's per-request auth reads.

---

### Implementation Details

**What to implement first:**
- Add `orch account add work` to save work account refresh token
- Document the `--backend claude --account work` pattern in CLAUDE.md

**Things to watch out for:**
- ⚠️ Two OpenCode servers would need separate OAuth logins (can't share auth.json)
- ⚠️ The worker server needs its own `XDG_STATE_HOME` too (or state dir override) to avoid SQLite conflicts
- ⚠️ Process reaper (`orch reap`) would need to know about both servers

**Areas needing further investigation:**
- Whether `XDG_STATE_HOME` and `XDG_CACHE_HOME` also need overriding for clean two-server operation
- How to handle the second server's lifecycle in `orch-dashboard`
- Whether the worker server needs a separate `opencode.json` config

**Success criteria:**
- ✅ Orchestrator uses personal Max 5x tokens exclusively
- ✅ Workers use work Max 20x tokens exclusively
- ✅ No cross-contamination of auth tokens between orchestrator and workers
- ✅ Both can operate concurrently without race conditions

---

## References

**Files Examined:**
- `packages/opencode/src/auth/index.ts` - Auth.get() implementation, confirmed disk reads on every call
- `packages/opencode/src/global/index.ts` - Global.Path.data resolution via xdgData
- `packages/opencode/src/provider/provider.ts` - Provider state init, auth loading, SDK creation
- `packages/opencode/src/project/instance.ts` - Instance cache keyed by directory
- `packages/opencode/src/project/state.ts` - State.create() caching mechanism
- `packages/opencode/src/flag/flag.ts` - OPENCODE_CONFIG_DIR dynamic getter
- `packages/opencode/src/config/config.ts` - Config dir resolution (OPENCODE_CONFIG_DIR effect)
- `packages/opencode/src/server/routes/session.ts` - Session create endpoint schema
- `packages/opencode/src/session/index.ts` - Session.create() function signature
- `packages/opencode/src/plugin/index.ts` - Plugin loading, auth loader interface
- `packages/opencode/src/plugin/codex.ts` - Codex auth plugin (OAuth fetch wrapper pattern)
- `~/.cache/opencode/node_modules/opencode-anthropic-auth/index.mjs` - Anthropic auth plugin (custom fetch, per-request token read)
- `cmd/orch/spawn_account_isolation.go` - maybeSwitchSpawnAccount() and resolveSpawnClaudeConfigDir()
- `cmd/orch/spawn_pipeline.go` - Spawn pipeline with serverURL parameterization
- `pkg/opencode/client.go` - Client with configurable ServerURL

**Commands Run:**
```bash
# Verified auth.json location and structure
ls -la ~/.local/share/opencode/auth.json
cat ~/.local/share/opencode/auth.json | python3 -c "..."

# Verified XDG_DATA_HOME scopes xdgData resolution
XDG_DATA_HOME=/tmp/opencode-test-data bun -e 'import { xdgData } from "xdg-basedir"; console.log(xdgData)'
# Output: /tmp/opencode-test-data

# Verified opencode serve accepts --port
~/.bun/bin/opencode serve --help

# Checked running OpenCode server processes
ps aux | grep 'opencode.*serve|bun.*src/index'

# Checked port usage
lsof -i :4096
```

**Related Artifacts:**
- **Probe:** `.kb/models/current-model-stack/probes/2026-02-12-split-orchestrator-worker-account-quota.md` - Prior probe establishing Claude CLI as clean path
- **Model:** `.kb/models/current-model-stack.md` - Multi-Account Access section being extended

---

## Investigation History

**2026-02-12 12:30:** Investigation started
- Initial question: Can OpenCode support per-session auth so orchestrator and workers use different accounts simultaneously?
- Context: Orchestrator uses personal Max 5x, workers need work Max 20x. Current maybeSwitchSpawnAccount() writes globally.

**2026-02-12 13:00:** Source analysis complete
- Auth.get() reads disk on every call, no caching
- Anthropic plugin custom fetch re-reads auth per API request
- Session create API has no auth parameter
- OPENCODE_CONFIG_DIR doesn't affect auth.json path

**2026-02-12 13:15:** Tested XDG_DATA_HOME isolation approach
- Confirmed XDG_DATA_HOME overrides data directory resolution
- Two-server approach is architecturally viable

**2026-02-12 13:20:** Investigation completed
- Status: Complete
- Key outcome: OpenCode cannot do per-session auth; two-server approach via XDG_DATA_HOME is the clean solution, with Claude CLI backend as the immediate fallback.
