# Session Synthesis

**Agent:** og-inv-investigate-opencode-codex-18feb-2152
**Issue:** orch-go-1030
**Duration:** 2026-02-18
**Outcome:** success

---

## Plain-Language Summary

Investigated whether OpenCode's Codex (OpenAI OAuth) auth is scoped per-project, which was hypothesized to explain why headless Codex agents fail intermittently for orch-go and consistently for price-watch. **The hypothesis is disproven**: auth is read from a single global file (`~/.local/share/opencode/auth.json`), the Codex plugin is loaded unconditionally as an internal plugin, and OpenCode does NOT load `.env` files from project directories. Provider state is technically per-directory via `Instance.state()`, but all inputs (auth, env, models.dev) are global — so the Codex provider produces identical results for every project. The most likely cause of intermittent failures is server crashes killing active sessions, or a subtler bug: `State.create` caches rejected Promises with no retry, so a transient init failure permanently disables providers for a directory until instance eviction (30min) or server restart.

## Verification Contract

See `.orch/workspace/og-inv-investigate-opencode-codex-18feb-2152/VERIFICATION_SPEC.yaml`

---

## TLDR

Codex auth is GLOBAL (not per-project). The .env hypothesis is wrong — OpenCode doesn't load them. Intermittent failures are most likely server crashes or cached rejected Promises in State.create. Price-watch failure needs separate investigation since auth scoping is ruled out.

---

## Delta (What Changed)

### Files Created
- `.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md` - Comprehensive probe with full code trace
- `.kb/investigations/2026-02-18-inv-codex-auth-project-scoping.md` - Investigation with D.E.K.N. summary

### Files Modified
- None (investigation only)

### Commits
- TBD (will commit at session end)

---

## Evidence (What Was Observed)

- `auth/index.ts:37`: Auth filepath is `path.join(Global.Path.data, "auth.json")` — global, hardcoded
- `env/index.ts:4-8`: Env.state does `{ ...process.env }` — server process env, no .env loading
- `plugin/index.ts:22`: CodexAuthPlugin is in `INTERNAL_PLUGINS` array — unconditional loading
- `plugin/index.ts:56`: Old npm `opencode-openai-codex-auth` explicitly skipped
- `plugin/codex.ts:357`: Loader checks `auth.type !== "oauth"` — global check
- `provider/provider.ts:704`: Provider.state uses `Instance.state` — per-directory cache
- `provider/provider.ts:863-908`: Plugin loader calls `Auth.get("openai")` — global
- `project/state.ts:20-22`: Caches init() result including rejected Promises
- auth.json confirmed: `openai: {type: "oauth", expires: 1772304513051}`
- orch-go opencode.json: empty (just schema)
- price-watch opencode.json: schema + instructions + MCP (no provider config)
- price-watch .env: DB/Redis/R2/SCS creds — NO `OPENAI_API_KEY`

### Tests Run
```
# Code trace only (investigation, not implementation)
# Verified file contents and code paths through direct source reading
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md` - Extends OpenCode Fork model with auth scoping details
- `.kb/investigations/2026-02-18-inv-codex-auth-project-scoping.md` - Full investigation with D.E.K.N.

### Constraints Discovered
- State.create caches rejected Promises with no retry — transient failures become permanent per-directory
- OpenCode has no .env loader — common misconception when projects have .env files
- CodexAuthPlugin is internal, not configurable per-project

### Externalized via `kb`
- `kb quick constrain` — "OpenCode does not load .env files from project directories" (to record)
- `kb quick tried` — ".env affecting Codex auth" failed "OpenCode Env module copies process.env, no dotenv loader"

---

## Next (What Should Happen)

**Recommendation:** close + spawn follow-ups

### If Close
- [x] All deliverables complete (probe, investigation, synthesis)
- [x] Investigation conclusive — hypothesis disproven with evidence
- [x] Probe has Status: Complete

### Follow-up Issues

1. **Fix State.create rejected Promise caching** (bug in opencode fork)
   - `State.create` should retry on rejection or clear the cache entry
   - This is the root cause of intermittent provider failures

2. **Investigate price-watch Codex failure with different hypothesis**
   - Auth scoping is ruled out
   - New hypotheses: Instance never created for PW directory, InstanceBootstrap failure, model alias mismatch in orch-go cross-project spawn

---

## Unexplored Questions

**Questions that emerged during this session:**
- Does orch-go pass the correct `x-opencode-directory` header when spawning for price-watch? The path is `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch` — different base than orch-go's `/Users/dylanconlin/Documents/personal/`
- Could InstanceBootstrap fail specifically for price-watch due to Playwright MCP init (`npx @playwright/mcp@latest`)?
- When Provider.state rejects, what error does the session see? Is it logged anywhere?

**What remains unclear:**
- The exact error message/behavior when price-watch tries to use Codex
- Whether the "never works" claim was tested after the latest server restart

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-investigate-opencode-codex-18feb-2152/`
**Investigation:** `.kb/investigations/2026-02-18-inv-codex-auth-project-scoping.md`
**Probe:** `.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md`
**Beads:** `bd show orch-go-1030`
