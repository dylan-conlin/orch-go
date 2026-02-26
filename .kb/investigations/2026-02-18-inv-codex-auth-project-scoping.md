# Investigation: Codex Auth Project Scoping

**Date:** 2026-02-18
**Status:** Complete

## D.E.K.N. Summary

- **Delta:** Codex auth is GLOBAL (auth.json), not per-project. Provider state is per-directory but deterministically identical across projects because all inputs (auth, env, models.dev) are global. OpenCode does NOT load .env files. The original hypothesis (per-project auth scoping causes failures) is disproven.
- **Evidence:** Full code trace through server.ts → Instance.provide → State.create → Provider.state → Plugin.list → CodexAuthPlugin.loader → Auth.get. Verified auth.json contents, both project configs, and price-watch .env.
- **Knowledge:** State.create caches rejected Promises with no retry — a transient init failure permanently disables a provider per-directory until eviction/restart. This is the most likely root cause of intermittent headless Codex failures.
- **Next:** (1) Fix State.create to retry rejected Promises. (2) Investigate price-watch failure separately — look at x-opencode-directory header value and Instance init logs, since auth scoping is ruled out.

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| .kb/models/opencode-fork/model.md | extends | yes | None - model claims about fork architecture confirmed |
| .kb/models/opencode-session-lifecycle/model.md | extends | yes | None - session persistence claims confirmed |
| .kb/models/model-access-spawn-paths/model.md | unrelated | N/A | Different topic (Opus access), not Codex |

---

## Question

1. Is Codex plugin initialization per-project or per-server?
2. How does x-opencode-directory header affect provider/auth resolution?
3. Does a .env file in the project directory affect OpenAI provider loading?
4. What happens when a Codex model is requested but the provider isn't loaded?
5. Does server restart explain the 10:12 vs 10:54 pattern?
6. What in price-watch conflicts with Codex OAuth?

---

## Findings

### Finding 1: Auth Module is GLOBAL

`Auth.get()` reads from `~/.local/share/opencode/auth.json` (path hardcoded via `Global.Path.data`). No per-directory scoping. Both projects get the same OAuth tokens.

File: `opencode/packages/opencode/src/auth/index.ts:37`

### Finding 2: Provider State is PER-DIRECTORY but Deterministically Identical

`Provider.state = Instance.state(async () => {...})` creates a per-directory singleton. But initialization reads from:
- `Auth.all()` → global auth.json
- `ModelsDev.get()` → global models cache
- `Env.all()` → server process.env copy (not project .env)
- `Config.get()` → global + project config merged
- `Plugin.list()` → internal plugins (unconditional)

Since neither orch-go nor price-watch has project-specific OpenAI config, the provider initialization is identical.

### Finding 3: CodexAuthPlugin is an Internal Plugin

Loaded unconditionally from `INTERNAL_PLUGINS` array (`plugin/index.ts:22`). Not affected by project config. The old npm package `opencode-openai-codex-auth` is explicitly skipped (`plugin/index.ts:56`).

Activation path:
1. `Auth.get("openai")` returns truthy (global auth.json has OAuth entry)
2. Plugin loader checks `auth.type === "oauth"` (it is)
3. Returns `{apiKey: OAUTH_DUMMY_KEY, fetch: customFetch}`

### Finding 4: .env Files NOT Loaded

`Env.state` does `{ ...process.env }` — shallow copy of server process environment. OpenCode has no dotenv loader. price-watch `.env` contains DB/Redis/R2/SCS creds but no `OPENAI_API_KEY`. Even if it did, OpenCode wouldn't read it.

### Finding 5: Rejected Promise Caching (Root Cause of Intermittent Failures)

`State.create` (`project/state.ts:12-29`):
```typescript
const state = init() // Returns Promise
entries.set(init, { state, dispose })
return state // Caches the Promise, including if rejected
```

No retry logic. If Provider.state's async initializer rejects (auth.json being written, network timeout on models.dev, etc.), the rejected Promise is cached for that directory. All subsequent requests for that directory fail until:
- Instance eviction (30min idle TTL or >20 instances LRU)
- Server restart

This explains: 3 agents at 10:12 all dead (same cached rejected Promise for orch-go directory), 1 agent at 10:54 works (server restarted, fresh cache).

### Finding 6: Price-Watch Cannot Be Explained by Auth Scoping

price-watch should have identical Codex access to orch-go:
- Same global auth.json
- Same CodexAuthPlugin (internal, unconditional)
- No project-specific provider config
- No `OPENAI_API_KEY` in env
- No `disabled_providers` or `enabled_providers` in any config

The "price-watch never works with codex" requires a different investigation. Candidates:
- Instance for price-watch directory was never created (directory never used as x-opencode-directory)
- Model alias resolution in orch-go maps "codex" differently for cross-project spawns
- InstanceBootstrap failure for the price-watch directory (e.g., git worktree detection, Playwright MCP init)

---

## Conclusion

**Codex auth is NOT project-scoped.** The original hypothesis is definitively disproven:

1. Auth is global (auth.json)
2. Provider state is per-directory but produces identical results for Codex
3. .env files are irrelevant (not loaded by OpenCode)
4. CodexAuthPlugin is unconditional (internal plugin)

The intermittent failure pattern is best explained by **server crashes killing sessions** or **rejected Promise caching** after transient initialization failures. The price-watch consistent failure requires separate investigation with a different hypothesis.
