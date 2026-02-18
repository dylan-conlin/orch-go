# Probe: Headless Codex Provider Initialization Scoping

**Model:** opencode-fork
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Do headless (API-created) OpenCode sessions initialize the Codex provider the same way as TUI sessions, or is provider setup skipped/failed for headless spawns? Is Codex auth initialization per-project or per-server? Does a project `.env` file affect OpenAI provider loading?

---

## What I Tested

Traced the complete code path from HTTP request through provider initialization in the OpenCode fork source (`~/Documents/personal/opencode/packages/opencode/src/`):

1. **HTTP middleware** (`server/server.ts:196-213`): Every request goes through `Instance.provide({directory, init: InstanceBootstrap, fn: next})` where directory comes from `x-opencode-directory` header or `process.cwd()`.

2. **Instance state scoping** (`project/instance.ts`, `project/state.ts`): `Instance.state(init)` creates per-directory singletons via `State.create(() => Instance.directory, init)`. The State cache keys on `(directory, init_function_reference)`.

3. **Auth module** (`auth/index.ts:37`): `const filepath = path.join(Global.Path.data, "auth.json")` — reads from `~/.local/share/opencode/auth.json`. **GLOBAL**, not per-directory. Confirmed auth.json has `openai: {type: "oauth", expires: 1772304513051, access_len: 1879}`.

4. **Env module** (`env/index.ts:4-8`): `Instance.state(() => ({ ...process.env }))` — copies server's `process.env` at instance creation time. Does NOT load `.env` files from the project directory.

5. **Provider.state** (`provider/provider.ts:704-982`): Per-directory singleton Promise. Initialization reads from:
   - `Config.get()` — merges global + project `opencode.json` (per-directory via `Instance.state`)
   - `ModelsDev.get()` — global models.dev cache
   - `Env.all()` — server `process.env` copy
   - `Auth.all()` — global auth.json
   - `Plugin.list()` → hooks including CodexAuthPlugin

6. **Codex plugin loader** (`plugin/codex.ts:351-416`): Called from Provider.state at line 884:
   ```
   plugin.auth.loader(() => Auth.get("openai"), database["openai"])
   ```
   The loader checks `auth.type !== "oauth"` → returns `{}` if not OAuth. If OAuth, returns `{apiKey: OAUTH_DUMMY_KEY, fetch: customFetch}` where customFetch handles OAuth bearer tokens and URL rewriting to `https://chatgpt.com/backend-api/codex/responses`.

7. **Plugin loading** (`plugin/index.ts:22,42-46`): CodexAuthPlugin is in `INTERNAL_PLUGINS` array — loaded unconditionally for every directory, not per-project.

8. **Checked project configs**:
   - orch-go `.opencode/opencode.json`: `{"$schema": "https://opencode.ai/config.json"}` (empty)
   - price-watch `opencode.json`: schema + instructions + MCP (no provider config)
   - Global `~/.config/opencode/opencode.jsonc`: Has `"openai"` provider with model configs and options
   - price-watch `.env`: Contains DB creds, Redis, Rails, R2, SCS OAuth — **no OPENAI_API_KEY**

9. **State caching and rejection** (`project/state.ts:12-29`): `State.create` caches the init() result including rejected Promises. No retry mechanism. A transient failure during Provider.state initialization would permanently fail for that directory until Instance eviction or server restart.

---

## What I Observed

### Auth Scoping: GLOBAL, not per-project

The `Auth` module reads from a single global file (`~/.local/share/opencode/auth.json`). The `Auth.get("openai")` call in the plugin section of Provider.state returns the same OAuth entry regardless of which directory the Instance is scoped to.

### Provider State: Per-directory but deterministically identical

`Provider.state` is scoped per-directory via `Instance.state()`. However, all inputs to the initialization are either global or deterministically derived:
- Auth: global auth.json
- Models.dev: global cache
- Env: server process.env (same for all directories)
- Config: global + project merged (but neither project has OpenAI-specific config)

Result: Provider.state for orch-go and price-watch would produce **identical** OpenAI provider entries because:
- Same global config provides `openai.options` and model definitions
- Same auth.json provides OAuth tokens
- Neither project has `OPENAI_API_KEY` in env or project config
- CodexAuthPlugin is loaded as an internal plugin unconditionally

### .env Files: NOT loaded by OpenCode

OpenCode does NOT have a dotenv loader. The `Env.state` copies `process.env` (the server process's environment) at instance creation time. Project-level `.env` files have zero effect on provider initialization.

price-watch's `.env` contains database, Redis, Playwright, R2, and SCS OAuth credentials — no `OPENAI_API_KEY`. Even if it did, OpenCode wouldn't read it.

### Codex Plugin Activation Path

The CodexAuthPlugin activates when:
1. `Auth.get("openai")` returns a truthy value (line 871)
2. The auth type is "oauth" (line 357 in codex.ts)
3. Plugin returns `{apiKey: OAUTH_DUMMY_KEY, fetch: customFetch}`

This is a GLOBAL check that doesn't vary by project. The Codex plugin either works for all projects or none.

### Intermittent Failure Mechanism

The 10:12 (dead) vs 10:54 (working) pattern for orch-go is NOT caused by auth scoping. Most likely explanations:

1. **Server crash**: OpenCode server died between 10:12-10:54. Sessions created at 10:12 were killed (0 tokens = model never responded). Fresh server at 10:54 worked normally.

2. **Cached rejected Promise**: If Provider.state initialization failed transiently at 10:12 (e.g., models.dev fetch timeout, auth.json being written), the rejected Promise was cached. All three agents sharing the same directory got the same failure. Server restart cleared the cache.

Evidence favoring explanation 1: Three agents died simultaneously (0 tokens each) — consistent with server crash, not individual auth failures.

### Price-Watch Specific Failure

Based on code analysis, price-watch should have **identical** Codex access to orch-go. The "never works" claim cannot be explained by:
- Auth scoping (global)
- .env files (not loaded)
- Project config (no OpenAI-specific config)
- Provider disabled lists (none configured)

Possible alternative explanations requiring further investigation:
- The `x-opencode-directory` header may not be set correctly when spawning for price-watch (different base path: `/Users/dylanconlin/Documents/work/...` vs `/Users/dylanconlin/Documents/personal/...`)
- Instance for price-watch may never have been initialized on the current server
- Error may be elsewhere in the pipeline (model alias resolution in orch-go, session creation, etc.)

---

## Model Impact

- [x] **Confirms** invariant: "Sessions persist across restarts via disk storage" — Instance.state is NOT persisted, only session data. Provider state must be re-initialized after restart.
- [x] **Extends** model with:
  1. **Auth is GLOBAL, Provider state is PER-DIRECTORY** — but inputs are identical, so output is deterministically identical across projects for Codex specifically.
  2. **No .env loading** — OpenCode Env module copies process.env, doesn't read project .env files. This is a common misconception.
  3. **Rejected Promise caching** — `State.create` caches init() results including rejected Promises with no retry. A transient failure permanently disables a provider for a directory until Instance eviction (30min TTL, 20-instance LRU) or server restart.
  4. **CodexAuthPlugin is an internal plugin** — loaded unconditionally via `INTERNAL_PLUGINS` array, not per-project configuration. The old npm plugin `opencode-openai-codex-auth` is explicitly skipped.

---

## Notes

### The Real Risk: Rejected Promise Caching

The most impactful finding is that `State.create` has no retry mechanism for failed initialization. If `Provider.state` initialization fails (e.g., `Auth.all()` fails because auth.json is being written by `orch account switch`, or `ModelsDev.get()` times out), the rejected Promise is cached per-directory. This means:

- All subsequent requests for that directory will immediately fail
- Only remedy: Instance eviction (30min idle TTL or >20 instances LRU) or server restart
- Multiple concurrent agents for the same directory all share the same fate

This explains the pattern of "all agents at time T fail, agents at time T+42min work" — the Instance was either evicted (TTL) or the server restarted.

### Recommended Follow-ups

1. Add retry logic to `State.create` for async initializers that reject
2. Add logging to Provider.state initialization failures (currently errors may be swallowed)
3. Investigate price-watch failure separately — the auth scoping hypothesis is disproven; look at `x-opencode-directory` header value and Instance initialization logs
