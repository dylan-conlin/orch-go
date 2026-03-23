## Summary (D.E.K.N.)

**Delta:** OpenClaw exposes a comprehensive WebSocket RPC API (114+ methods) and supplementary HTTP endpoints that provide complete programmatic agent control — session create, message send, wait-for-completion, and session delete — meaning orch-go CAN drive OpenClaw agents externally, with a richer API surface than OpenCode.

**Evidence:** Direct examination of `~/Documents/personal/clawdbot/src/gateway/server-methods/agent.ts` (lines 197-870), `src/gateway/server-methods/sessions.ts` (lines 625-800), `src/plugins/runtime/types.ts` (lines 8-66), and `src/agents/cli-backends.ts` (lines 40-108) confirms: WebSocket `agent` method accepts message/provider/model/sessionKey/idempotencyKey params; `agent.wait` polls for completion with timeout; `sessions.create`/`sessions.send`/`sessions.list` provide full session lifecycle control; HTTP endpoints at `/v1/chat/completions` and `/hooks/agent` provide REST alternatives.

**Knowledge:** Three viable architectures exist, ranked by implementation cost and capability: (A) orch-go drives OpenClaw via WebSocket RPC — direct replacement for OpenCode driving, richest control surface; (B) orch-go drives OpenClaw via HTTP hooks — simpler, stateless, but less control; (C) coordination plugin inside OpenClaw replaces orch-go entirely — most powerful but highest migration cost.

**Next:** Strategic decision for Dylan — recommend Option A (orch-go drives OpenClaw via WebSocket) as the migration path, since it preserves orch-go's daemon/coordination/completion layer while gaining OpenClaw's multi-model routing, subagent hierarchy, and channel delivery.

**Authority:** strategic - Choosing OpenClaw as execution backend is an irreversible platform commitment affecting orch-go's architecture.

---

# Investigation: OpenClaw External API Surface — Can orch-go Drive OpenClaw Agents Programmatically?

**Question:** Does OpenClaw expose a programmatic API surface sufficient for orch-go to create sessions, send prompts, monitor progress, and retrieve results — the same operations it performs against OpenCode today?

**Started:** 2026-03-23
**Updated:** 2026-03-23
**Owner:** orch-go-hqsgm
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md | extends | yes | none — confirms OpenClaw's gateway architecture; this investigation drills into API specifics |
| .kb/investigations/2026-03-23-inv-investigate-orch-go-coordination-primitives-port.md | extends | yes | none — plugin SDK findings complement external API findings (internal vs external control) |

---

## Findings

### Finding 1: OpenClaw Gateway exposes 114+ WebSocket RPC methods for programmatic control

**Evidence:** The gateway server at `src/gateway/server-methods/` registers handlers for a JSON-RPC-style WebSocket protocol. Key methods for agent orchestration:

| Method | Purpose | orch-go Equivalent |
|--------|---------|-------------------|
| `agent` | Send message to agent (creates session if needed) | `opencode.SendPrompt()` |
| `agent.wait` | Poll for agent run completion with timeout | SSE monitoring |
| `sessions.create` | Create new session with explicit key | Session creation |
| `sessions.send` | Send message to existing session | `opencode.SendPrompt()` |
| `sessions.list` | List all sessions | Dashboard queries |
| `sessions.delete` | Delete session | Session cleanup |
| `sessions.abort` | Abort running session | Kill/cleanup |
| `agents.list` | List configured agents | N/A (new capability) |
| `health` | Gateway health check | `opencode.IsReachable()` |

The `agent` method (line 198-299 of `agent.ts`) accepts:
```typescript
{
  message: string;
  agentId?: string;
  provider?: string;        // model provider override
  model?: string;            // model override
  sessionKey?: string;       // explicit session targeting
  idempotencyKey: string;    // dedup support
  extraSystemPrompt?: string; // context injection!
  lane?: string;             // routing lane
  deliver?: boolean;         // channel delivery
  timeout?: number;
}
```

**Source:** `~/Documents/personal/clawdbot/src/gateway/server-methods/agent.ts:197-299`, `sessions.ts:625-800`

**Significance:** This is a **superset** of what orch-go currently uses from OpenCode. The `extraSystemPrompt` field is particularly valuable — it enables skill/context injection without file manipulation. The `idempotencyKey` provides built-in dedup that orch-go currently handles manually.

---

### Finding 2: OpenClaw runs headless as a pure API server — no UI or messaging channel required

**Evidence:** The gateway can start in headless mode:
```bash
openclaw gateway --allow-unconfigured --port 3000 --bind lan
```

This is how Fly.io deployment works (`fly.toml`):
```
app = "node dist/index.js gateway --allow-unconfigured --port 3000 --bind lan"
```

Docker compose confirms gateway is the only required service:
```yaml
services:
  openclaw-gateway:
    # runs standalone, no messaging channels needed
```

Authentication is token-based:
```bash
OPENCLAW_GATEWAY_TOKEN=<token>
# or OPENCLAW_GATEWAY_PASSWORD=<password>
```

**Source:** `~/Documents/personal/clawdbot/fly.toml`, `docker-compose.yml`, `.env.example`

**Significance:** OpenClaw can replace OpenCode as the execution layer without requiring Slack/Discord/Telegram setup. It runs as a local daemon on a configurable port, just like OpenCode runs on port 4096.

---

### Finding 3: The `agent.wait` method provides synchronous completion polling — replacing SSE monitoring

**Evidence:** `agent.wait` (line 783-869 of `agent.ts`):
```typescript
"agent.wait": async ({ params, respond, context }) => {
  const runId = (p.runId ?? "").trim();
  const timeoutMs = typeof p.timeoutMs === "number" ? Math.max(0, Math.floor(p.timeoutMs)) : 30_000;
  // ... polls lifecycle + dedupe for terminal state ...
  respond(true, {
    runId,
    status: snapshot.status,  // "ok" | "error" | "timeout"
    startedAt: snapshot.startedAt,
    endedAt: snapshot.endedAt,
    error: snapshot.error,
  });
}
```

Workflow: `agent` → returns `{runId}` → `agent.wait({runId, timeoutMs})` → returns terminal status.

This replaces orch-go's SSE-based monitoring (`pkg/opencode/sse.go`) with a simpler request/response pattern.

**Source:** `~/Documents/personal/clawdbot/src/gateway/server-methods/agent.ts:783-869`

**Significance:** SSE parsing is one of orch-go's most fragile integrations (noted in CLAUDE.md gotchas: "Event type is inside JSON data, not `event:` prefix"). `agent.wait` eliminates this entirely.

---

### Finding 4: OpenClaw delegates to Claude Code CLI as a backend — same delegation model as orch-go

**Evidence:** `cli-backends.ts` (lines 40-70) defines the default Claude CLI backend:
```typescript
const DEFAULT_CLAUDE_BACKEND: CliBackendConfig = {
  command: "claude",
  args: ["-p", "--output-format", "json", "--permission-mode", "bypassPermissions"],
  resumeArgs: ["-p", "--output-format", "json", "--permission-mode", "bypassPermissions", "--resume", "{sessionId}"],
  output: "json",
  modelArg: "--model",
  sessionMode: "always",
};
```

Also supports Codex (lines 72-108):
```typescript
const DEFAULT_CODEX_BACKEND: CliBackendConfig = {
  command: "codex",
  args: ["exec", "--json", "--color", "never", "--sandbox", "workspace-write"],
};
```

The `cli-runner.ts` (line 53) function `runCliAgent()` builds a system prompt, resolves workspace, and spawns the CLI backend as a subprocess via `process/supervisor/`.

**Source:** `~/Documents/personal/clawdbot/src/agents/cli-backends.ts:40-108`, `cli-runner.ts:53-200`

**Significance:** OpenClaw's execution model is **structurally identical** to what orch-go does: shell out to `claude -p` with JSON output, manage sessions via `--session-id`/`--resume`. The difference is OpenClaw adds watchdog monitoring, failover, and multi-backend routing.

---

### Finding 5: Three HTTP API surfaces provide REST alternatives to WebSocket

**Evidence:** The gateway serves HTTP endpoints on the same port as WebSocket:

1. **OpenAI-compatible API** (`POST /v1/chat/completions`): Full OpenAI Chat Completions format with streaming SSE. Auth via Bearer token. Config: `gateway.http.endpoints.chatCompletions.enabled`.

2. **Webhook/Hook API** (`POST /hooks/agent`): Dispatch agent commands via HTTP POST. Supports idempotency keys.

3. **Tools Invoke API** (`POST /tools/invoke`): Call individual tools directly without full agent turns.

4. **Session Kill** (`POST /api/sessions/kill`): Force-terminate sessions.

5. **Session History** (`GET /api/sessions/history/:sessionKey`): Export session transcripts.

**Source:** `~/Documents/personal/clawdbot/src/gateway/server-http.ts:1-80`, `openai-http.ts:1-60`, `tools-invoke-http.ts`

**Significance:** orch-go could use HTTP hooks for fire-and-forget spawns (simpler integration) while using WebSocket for session management and monitoring (richer control).

---

### Finding 6: Plugin runtime exposes subagent control for in-process coordination

**Evidence:** `PluginRuntime.subagent` (from `src/plugins/runtime/types.ts:54-66`):
```typescript
subagent: {
  run: (params: SubagentRunParams) => Promise<SubagentRunResult>;
  waitForRun: (params: SubagentWaitParams) => Promise<SubagentWaitResult>;
  getSessionMessages: (params) => Promise<SubagentGetSessionMessagesResult>;
  deleteSession: (params: SubagentDeleteSessionParams) => Promise<void>;
};
```

`SubagentRunParams` accepts: `sessionKey`, `message`, `provider`, `model`, `extraSystemPrompt`, `lane`, `deliver`, `idempotencyKey`.

This is only available in-process (during gateway request handling), not from external API calls.

**Source:** `~/Documents/personal/clawdbot/src/plugins/runtime/types.ts:8-66`

**Significance:** If orch-go coordination logic moves INTO an OpenClaw plugin (Option C), it gets direct subagent spawning without WebSocket overhead. But this requires reimplementing daemon + completion + beads integration in TypeScript.

---

## Synthesis

**Key Insights:**

1. **API surface is a superset of OpenCode** — OpenClaw provides everything orch-go currently uses from OpenCode (session create, prompt send, status monitoring, session delete) PLUS additional capabilities: `agent.wait` polling, `extraSystemPrompt` injection, multi-model routing, idempotency, and subagent hierarchy.

2. **Two integration paths, not three** — While three architectures are theoretically possible, Option B (HTTP-only) and Option A (WebSocket RPC) are really the same path at different depths. Start with HTTP hooks for spawning, add WebSocket for monitoring. Option C (coordination plugin) is a separate strategic decision about replacing orch-go, not enhancing it.

3. **The delegation model is identical** — Both OpenCode and OpenClaw shell out to `claude -p --output-format json`. OpenClaw adds a supervision layer (watchdogs, failover, multi-backend) but the fundamental execution is the same CLI invocation. This means migrating from OpenCode to OpenClaw is an API client swap, not an architectural change.

**Answer to Investigation Question:**

Yes, orch-go can drive OpenClaw agents programmatically with a richer API than it currently has with OpenCode. The migration path is:

| orch-go Operation | OpenCode Method | OpenClaw Equivalent |
|-------------------|-----------------|---------------------|
| Create session + send prompt | HTTP POST to OpenCode API | WebSocket `agent` or HTTP `POST /hooks/agent` |
| Monitor progress | SSE stream parsing | WebSocket `agent.wait` (polling) or session events |
| Check status | HTTP GET | WebSocket `sessions.list` or HTTP health endpoints |
| Kill/cleanup | HTTP DELETE | WebSocket `sessions.abort`/`sessions.delete` |
| Inject context | File-based (SPAWN_CONTEXT.md) | `extraSystemPrompt` param (no file needed) |

The Go client (`pkg/opencode/client.go`) would need a new sibling `pkg/openclaw/client.go` implementing WebSocket connection + JSON-RPC framing. Estimated ~300-400 lines of Go.

---

## Structured Uncertainty

**What's tested:**

- ✅ WebSocket RPC methods exist with documented type signatures (verified: read `agent.ts`, `sessions.ts`, `types.ts` source)
- ✅ `agent` method accepts message, provider, model, sessionKey, extraSystemPrompt params (verified: read type definitions at `agent.ts:211-244`)
- ✅ `agent.wait` returns terminal status with runId/status/error fields (verified: read implementation at `agent.ts:783-869`)
- ✅ Headless operation supported (verified: fly.toml and docker-compose.yml both run gateway-only mode)
- ✅ Claude CLI delegation uses same `claude -p --output-format json` pattern (verified: `cli-backends.ts:40-70`)

**What's untested:**

- ⚠️ WebSocket authentication flow — read the auth code but didn't connect to a running instance
- ⚠️ `agent.wait` behavior under concurrent requests — code handles dedup but no load test
- ⚠️ Whether `extraSystemPrompt` fully replaces SPAWN_CONTEXT.md file injection — may have size limits
- ⚠️ Performance characteristics — WebSocket vs HTTP latency for orch-go spawn patterns
- ⚠️ Whether OpenClaw gateway can run alongside OpenCode on same machine without port conflicts

**What would change this:**

- If `agent.wait` doesn't work reliably for long-running sessions (>30min), SSE-style streaming would still be needed
- If `extraSystemPrompt` has a low character limit, file-based context injection would still be required
- If WebSocket connection management adds more complexity than SSE parsing, the "simpler" claim inverts

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Build `pkg/openclaw/client.go` WebSocket client | architectural | New package, cross-boundary dependency on external system |
| Choose OpenClaw as execution backend | strategic | Irreversible platform commitment, affects all agent infrastructure |
| Use `extraSystemPrompt` for context injection | implementation | Tactical optimization within existing patterns |

### Recommended Approach: orch-go drives OpenClaw via WebSocket RPC (Option A)

**Why this approach:**
- Preserves orch-go's daemon, coordination, completion, and beads layers entirely
- Replaces the most fragile integration point (OpenCode SSE parsing) with cleaner `agent.wait` polling
- Gains multi-model routing, failover, and supervision for free
- `extraSystemPrompt` eliminates SPAWN_CONTEXT.md file writing for context injection

**Trade-offs accepted:**
- WebSocket client is more complex than HTTP client (connection lifecycle, reconnection)
- OpenClaw must run as a local daemon (same as OpenCode today)
- Two-process architecture (orch-go + OpenClaw) vs single-process if using plugin approach

**Implementation sequence:**
1. **Build `pkg/openclaw/client.go`** — WebSocket connection, auth, JSON-RPC framing, `agent`/`agent.wait`/`sessions.list` methods (~300 LoC)
2. **Add spawn backend** — New spawn backend in `pkg/spawn/` that uses OpenClaw client instead of OpenCode client
3. **Migrate monitoring** — Replace SSE-based status polling with `agent.wait` loop
4. **Context injection** — Use `extraSystemPrompt` for skill/context injection instead of file writing

### Alternative Approaches Considered

**Option B: HTTP-only integration (hooks + OpenAI compat API)**
- **Pros:** Simpler client (HTTP POST), no WebSocket lifecycle management
- **Cons:** No session monitoring, no completion waiting, fire-and-forget only
- **When to use instead:** If orch-go only needs to spawn and doesn't care about completion status

**Option C: Coordination plugin inside OpenClaw (replaces orch-go)**
- **Pros:** Most powerful — in-process subagent control, no IPC overhead, reaches OpenClaw's user base
- **Cons:** Requires reimplementing daemon, completion, beads, and skill system in TypeScript; abandons Go codebase
- **When to use instead:** If Dylan decides orch-go's coordination model should be distributed as an OpenClaw plugin rather than maintained as a standalone tool

**Rationale for Option A:** Option A minimizes migration risk while capturing the biggest wins (replacing SSE parsing, gaining multi-model routing). Option C is architecturally superior but is a strategic pivot, not an engineering optimization.

---

### Implementation Details

**What to implement first:**
- `pkg/openclaw/client.go` with `Connect()`, `Agent()`, `AgentWait()`, `SessionsList()` methods
- WebSocket JSON-RPC framing layer (shared between all methods)
- Token-based auth (read from config, inject in connect frame)

**Things to watch out for:**
- ⚠️ WebSocket reconnection — OpenClaw gateway may restart; client needs reconnect logic
- ⚠️ Port conflicts — OpenCode uses 4096, OpenClaw defaults to 18789 — should be fine but verify
- ⚠️ `agent.wait` default timeout is 30s — orch-go agents run for 30-60min; need long-poll or repeated waits
- ⚠️ Auth scopes — `operator.admin` scope needed for model override; verify token config

**Areas needing further investigation:**
- WebSocket event subscription for real-time status (alternative to polling `agent.wait`)
- OpenClaw config file format for workspace/agent identity setup
- How to configure CLI backends when OpenClaw runs alongside Claude Code (shared or separate claude binary)

**Success criteria:**
- ✅ orch-go can spawn an agent via OpenClaw and receive completion status
- ✅ `orch status` shows agent activity from OpenClaw sessions
- ✅ Context injection via `extraSystemPrompt` replaces SPAWN_CONTEXT.md file writing
- ✅ Agent monitoring works without SSE parsing

---

## References

**Files Examined:**
- `~/Documents/personal/clawdbot/src/gateway/server-methods/agent.ts` — Core agent WebSocket RPC handlers
- `~/Documents/personal/clawdbot/src/gateway/server-methods/sessions.ts` — Session CRUD WebSocket handlers
- `~/Documents/personal/clawdbot/src/plugins/runtime/types.ts` — Plugin subagent runtime API
- `~/Documents/personal/clawdbot/src/agents/cli-backends.ts` — Claude/Codex CLI backend config
- `~/Documents/personal/clawdbot/src/agents/cli-runner.ts` — CLI agent execution flow
- `~/Documents/personal/clawdbot/src/gateway/server-http.ts` — HTTP endpoint routing
- `~/Documents/personal/clawdbot/src/gateway/openai-http.ts` — OpenAI-compatible HTTP API
- `~/Documents/personal/clawdbot/fly.toml` — Headless deployment config
- `~/Documents/personal/clawdbot/docker-compose.yml` — Docker headless deployment
- `~/Documents/personal/clawdbot/.env.example` — Environment configuration
- `~/Documents/personal/orch-go/pkg/opencode/client.go` — Current OpenCode client for comparison

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md` — OpenClaw platform overview
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-orch-go-coordination-primitives-port.md` — Coordination plugin feasibility

---

## Investigation History

**2026-03-23 13:56:** Investigation started
- Initial question: Can orch-go drive OpenClaw agents programmatically like it drives OpenCode?
- Context: Evaluating OpenClaw as potential replacement for OpenCode execution layer

**2026-03-23 14:10:** Parallel exploration of gateway, plugin, and headless capabilities
- Three subagents explored gateway API, plugin SDK, and headless deployment in parallel
- 114+ WebSocket methods catalogued, HTTP endpoints mapped

**2026-03-23 14:20:** Source verification of critical claims
- Read `agent.ts`, `sessions.ts`, `cli-backends.ts` directly
- Confirmed `agent` method params, `agent.wait` implementation, CLI backend config
- Mapped orch-go OpenCode operations to OpenClaw equivalents

**2026-03-23 14:30:** Investigation completed
- Status: Complete
- Key outcome: OpenClaw's API surface is a superset of OpenCode's; orch-go can drive it via WebSocket RPC with richer control than current OpenCode integration
