## Summary (D.E.K.N.)

**Delta:** Orch-go and clawdbot have complementary architectures that enable high-value integration through clawdbot's webhook hooks system calling orch's HTTP API, with Discord as the command/notification surface.

**Evidence:** Orch-go exposes 50+ REST endpoints on `orch serve` (port 3348) with JSON responses. Clawdbot has a webhook hooks system (`/hooks` endpoint with bearer auth), Discord actions API (sendMessage, react, threads), cron/heartbeat system, and native slash commands - all extensible via plugins.

**Knowledge:** The highest-value integration is status notifications (orch → Discord) and remote spawn triggers (Discord → orch). Both are achievable with existing APIs on both sides. Clawdbot's hooks system is the natural integration point - no changes needed to clawdbot core.

**Next:** Implement Phase 1 (orch → Discord notifications via clawdbot hooks/Discord API) as a Go package in orch-go. Decision needed on whether to build as orch plugin, daemon extension, or standalone bridge service.

**Authority:** strategic - This is a cross-project integration decision involving two separate systems, resource commitment for ongoing maintenance, and establishes a new communication channel for the orchestration system.

---

# Investigation: Orch/Clawdbot Integration Possibilities

**Question:** What capabilities would an orch/clawdbot integration enable, what's the integration surface, how feasible is each capability, and what approach should we take?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None - escalate to orchestrator/Dylan for strategic decision
**Status:** Complete

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/guides/cli.md` | extends | Yes - verified API structure | None |
| `.kb/guides/daemon.md` | extends | Yes - verified daemon architecture | None |
| `.kb/guides/spawn.md` | extends | Yes - verified spawn mechanics | None |

---

## Findings

### Finding 1: Orch-go has a comprehensive HTTP API ideal for external integration

**Evidence:** `orch serve` (port 3348, HTTPS with TLS) exposes 50+ endpoints covering the entire orchestration lifecycle:

| Category | Key Endpoints | Data Available |
|----------|---------------|----------------|
| Agent monitoring | `/api/agents`, `/api/events` (SSE) | Active agents, status, phase, tokens, runtime |
| Work queue | `/api/beads/ready`, `/api/frontier` | Ready issues, blocked work, stuck agents |
| Lifecycle | `/api/attention`, `/api/pending-reviews` | Completion verification, review needs |
| System health | `/api/usage`, `/api/daemon`, `/health` | Usage %, daemon status, capacity |
| Knowledge | `/api/kb-health`, `/api/decisions` | KB hygiene signals, decision center |

All endpoints return JSON with CORS headers. The `/api/events` endpoint provides SSE streaming for real-time agent activity.

**Source:** `cmd/orch/serve.go:358-530` - Full mux registration of all endpoints

**Significance:** Orch-go is already a web service. Any integration only needs to make HTTP calls to existing endpoints - no new API needs to be built on the orch side.

---

### Finding 2: Clawdbot's webhook hooks system provides the inbound integration point

**Evidence:** Clawdbot exposes a webhook endpoint at `/hooks` (configurable) with bearer token authentication:

```yaml
# clawdbot config
hooks:
  enabled: true
  token: "secret-token"
  path: "/hooks"
```

The hooks system supports two dispatch modes:
1. **Wake hooks** - Inject system events into the main agent session
2. **Agent hooks** - Spawn isolated agent turns with specific messages, delivering responses to channels

Hook payloads can specify:
- `message` - What to tell the agent
- `channel` - Where to deliver the response (discord, telegram, etc.)
- `to` - Specific target (channel ID, user ID)
- `deliver` - Whether to send the response to a channel
- `model`, `thinking`, `timeoutSeconds` - Agent execution params

**Source:** `clawdbot/src/gateway/hooks.ts:1-44`, `clawdbot/src/gateway/server/hooks.ts:1-116`

**Significance:** This is the natural integration point. Orch can POST to clawdbot's hooks endpoint to trigger messages in Discord channels without needing Discord bot tokens directly. Clawdbot handles all Discord API complexity.

---

### Finding 3: Clawdbot's Discord actions provide rich notification capabilities

**Evidence:** The Discord channel plugin exposes a comprehensive actions API:

| Action | Use Case for Orch |
|--------|-------------------|
| `sendMessage` | Post status updates, completion notifications |
| `react` | React to spawn request messages with ✅/⚠️/🚀 |
| `threadCreate` | Create per-agent threads for progress tracking |
| `threadReply` | Post phase updates within agent threads |
| `poll` | Run polls for triage decisions ("Should we prioritize X?") |
| `setPresence` | Show bot status reflecting swarm state ("Working on 3 tasks") |
| `channelCreate` | Auto-create project-specific channels |

Discord messages support up to 2000 characters, threads provide organization, and reactions enable quick acknowledgment.

**Source:** `clawdbot/skills/discord/SKILL.md:1-579`, `clawdbot/extensions/discord/src/channel.ts:33-304`

**Significance:** These actions enable rich, conversational orchestration UX rather than just plain-text notifications.

---

### Finding 4: Clawdbot's cron and native commands enable Discord → Orch triggers

**Evidence:**

**Cron system:** Clawdbot has a built-in cron system that can run periodic tasks:
- `every` schedules (poll orch status every N minutes)
- `at` schedules (one-shot tasks)
- Isolated agent execution with configurable models and thinking levels
- Results delivered to specific channels

**Native slash commands:** Discord slash commands are registered via `createDiscordNativeCommand()` and can trigger custom logic. These are discoverable in Discord's UI (type `/` to see available commands).

**Hooks (inbound):** External systems can call clawdbot's `/hooks/agent` endpoint to trigger agent actions. The reverse direction (orch calling clawdbot) works via the hooks system.

**Source:** `clawdbot/src/gateway/server/hooks.ts:32-105`, `clawdbot/src/discord/monitor/native-command.ts`, `clawdbot/src/cron/service/timer.ts`

**Significance:** These mechanisms enable the reverse direction - Discord users triggering orch operations. A cron job could poll `orch serve` for status and post summaries. Native commands could trigger spawns.

---

## Synthesis

**Key Insights:**

1. **Both systems are already HTTP services** - Orch runs on port 3348 (HTTPS), clawdbot gateway runs with webhook support. Integration is HTTP calls between local services. No new protocols needed.

2. **Clawdbot is the right "mouth" for Discord** - Rather than orch-go implementing Discord API directly (which would duplicate clawdbot's mature Discord integration), orch should use clawdbot as an intermediary. Clawdbot handles rate limiting, message chunking, threading, and all Discord protocol complexity.

3. **The hook system is the natural seam** - Clawdbot's hooks provide a clean API boundary. Orch doesn't need to know about Discord internals - it just sends "deliver this message to #orchestration channel" via the hooks endpoint.

4. **Value scales with direction** - Orch → Discord (notifications) has immediate, clear value with low complexity. Discord → orch (commands) is higher complexity but enables remote orchestration.

**Answer to Investigation Question:**

An orch/clawdbot integration is both feasible and valuable. The integration enables:
1. **Status notifications** - Agent completions, blockers, capacity alerts posted to Discord
2. **Remote triggering** - Spawn agents, triage issues, approve work from Discord
3. **Team visibility** - Swarm status dashboards in Discord threads
4. **Conversational orchestration** - Natural language commands to the orchestration system

The integration surface is clean: orch's HTTP API (50+ endpoints) on one side, clawdbot's webhook hooks + Discord actions on the other. Both sides have existing, stable APIs. Feasibility is high for notifications (days of work), moderate for remote commands (1-2 weeks).

---

## Capability Matrix

| Capability | Description | Direction | Feasibility | Value | Orch API Exists? | Clawdbot API Exists? |
|-----------|-------------|-----------|-------------|-------|-------------------|----------------------|
| **Completion notifications** | Post to Discord when agents finish work | orch → Discord | ✅ Easy | 🔴 High | `/api/events` (SSE), `/api/attention` | hooks/agent → sendMessage |
| **Blocker alerts** | Notify when agents are blocked/stuck | orch → Discord | ✅ Easy | 🔴 High | `/api/frontier` (stuck agents) | hooks/agent → sendMessage |
| **Capacity warnings** | Alert when usage hits thresholds | orch → Discord | ✅ Easy | 🟡 Medium | `/api/usage` | hooks/agent → sendMessage |
| **Swarm status summary** | Periodic status digest in Discord | orch → Discord | ✅ Easy | 🟡 Medium | `/api/agents`, `/api/beads` | hooks/agent → sendMessage |
| **Spawn from Discord** | Trigger agent spawns via messages | Discord → orch | 🟡 Medium | 🔴 High | CLI: `orch spawn` | native commands or hook |
| **Triage from Discord** | Label issues as triage:ready from Discord | Discord → orch | 🟡 Medium | 🟡 Medium | CLI: `bd label` | native commands |
| **Approve from Discord** | Approve agent work from Discord | Discord → orch | 🟡 Medium | 🟡 Medium | `/api/approve` (POST) | native commands |
| **Bot presence** | Bot status reflects swarm state | orch → Discord | ✅ Easy | 🟢 Low | `/api/agents` | setPresence action |
| **Conversational orch** | Natural language orchestration | Bidirectional | 🔴 Hard | 🔴 High | Full CLI/API | clawdbot agent session |
| **Investigation threads** | Auto-create Discord threads per agent | orch → Discord | 🟡 Medium | 🟡 Medium | `/api/agents` | threadCreate + threadReply |

---

## Structured Uncertainty

**What's tested:**

- ✅ Orch serve exposes JSON API on port 3348 with 50+ endpoints (verified: read `cmd/orch/serve.go:358-530`)
- ✅ Clawdbot has webhook hooks system with bearer auth (verified: read `src/gateway/hooks.ts`, `src/gateway/server/hooks.ts`)
- ✅ Discord actions include sendMessage, react, threadCreate, setPresence (verified: read `skills/discord/SKILL.md`)
- ✅ Both systems run locally on same machine (verified: spawn context confirms local dev setup)

**What's untested:**

- ⚠️ Actual HTTP call from orch-go to clawdbot hooks endpoint (not performed - would require running clawdbot gateway)
- ⚠️ Discord message delivery latency through the hooks → cron → Discord pipeline
- ⚠️ Whether clawdbot's hooks can handle the volume of events from an active swarm (e.g., 5 agents completing in rapid succession)
- ⚠️ Whether SSE streaming from orch's `/api/events` can be consumed reliably by a Go client for real-time notifications

**What would change this:**

- If clawdbot's hooks system has rate limits or timeouts that prevent rapid-fire notifications
- If Dylan decides orch should be self-contained (no external dependencies for core orchestration)
- If clawdbot's Discord channel plugin changes its action API significantly
- If a simpler approach (e.g., direct Discord webhook URLs) provides sufficient functionality

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Build orch → Discord notification bridge | strategic | Cross-project integration, establishes ongoing dependency between orch-go and clawdbot |
| Phase 1: Notification-only integration | architectural | Determines integration architecture and pattern for all future capabilities |
| Discord channel config in orch config | implementation | Config key placement, no architectural impact |

### Fork 1: Integration Architecture

**Options:**
- A: **Orch daemon extension** - Add Discord notification to daemon's poll loop
- B: **Standalone bridge service** - Separate process polling orch and posting to Discord
- C: **Orch plugin/hook in clawdbot** - Clawdbot polls orch via cron jobs

**Substrate says:**
- Decision: "orch-go CLI independence" - CLI connects directly to OpenCode, not through orch serve
- Constraint: "Registry is caching layer, not source of truth"
- Principle: "Prefer industry-standard tools over custom solutions"

**RECOMMENDATION:** Option A (daemon extension) because:
- Daemon already has the poll loop, capacity awareness, and completion detection
- Adding a "notify on completion" step to `CompletionOnce()` is natural
- No new process to manage or monitor
- Aligns with existing daemon architecture

**Trade-off accepted:** Couples notification delivery to daemon lifecycle (if daemon is down, no notifications)

### Fork 2: Discord Communication Path

**Options:**
- A: **Via clawdbot hooks** - POST to clawdbot's `/hooks/agent` endpoint
- B: **Direct Discord API** - Use Discord webhook URLs or bot token directly from orch-go
- C: **Direct Discord webhooks** - Use Discord's built-in webhook URLs (no bot needed)

**RECOMMENDATION:** Option C (Direct Discord webhooks) for Phase 1 because:
- Zero dependency on clawdbot running
- Discord webhook URLs are simple POST endpoints with JSON
- No authentication complexity beyond the webhook URL itself
- Works even when clawdbot is down or not deployed
- Webhook URLs support embeds (rich formatting), images, and up to 2000 chars

Then **Option A (clawdbot hooks)** for Phase 2 when richer capabilities are needed (spawn from Discord, conversational orchestration).

**Trade-off accepted:** Discord webhooks can't receive messages (send-only). Phase 2 adds bidirectional capabilities.

### Recommended Approach ⭐

**Phased integration, starting with Discord webhooks for notifications, graduating to clawdbot hooks for bidirectional orchestration**

**Why this approach:**
- Phase 1 delivers immediate value with minimal complexity (1-2 days of work)
- No dependency on clawdbot being running for basic notifications
- Natural graduation path - add capabilities incrementally
- Each phase is independently valuable

**Trade-offs accepted:**
- Phase 1 is send-only (no commands from Discord)
- Discord webhook URLs must be manually configured
- Richer features (threads, reactions) require Phase 2

**Implementation sequence:**

**Phase 1: Orch → Discord Notifications (Discord Webhooks)**
1. Add `discord_webhook_url` to `~/.orch/config.yaml`
2. Create `pkg/discord/webhook.go` - simple POST to Discord webhook
3. Add notification hooks to daemon's `CompletionOnce()` and blocker detection
4. Format messages as Discord embeds (color-coded: green=complete, red=blocked, yellow=warning)

**Phase 2: Orch → Discord Notifications (via Clawdbot)**
1. Add `clawdbot_hooks_url` and `clawdbot_hooks_token` to config
2. Create `pkg/clawdbot/client.go` - HTTP client for clawdbot hooks API
3. Enable richer notifications: threads per agent, reactions on completion, bot presence
4. Use clawdbot's agent system for formatting intelligence

**Phase 3: Discord → Orch Commands (via Clawdbot)**
1. Create clawdbot skill or native commands for orch operations
2. Implement `orch spawn`, `orch status`, `bd label triage:ready` from Discord
3. Add approval workflow from Discord reactions
4. Natural language command parsing via clawdbot's agent

### Alternative Approaches Considered

**Option B: Build everything through clawdbot**
- **Pros:** Richer Discord features from day 1, leverages existing bot infrastructure
- **Cons:** Hard dependency on clawdbot running, more complex initial setup, cross-repo coordination needed
- **When to use instead:** If clawdbot is always running on the same machine and team visibility is the primary driver

**Option C: Skip clawdbot, build Discord bot in orch-go**
- **Pros:** Self-contained, no external dependencies
- **Cons:** Duplicates clawdbot's Discord complexity, Go Discord libraries less mature than TypeScript, requires bot token management
- **When to use instead:** If orch-go needs to work in environments where clawdbot isn't available

**Rationale for recommendation:** Phased approach maximizes value delivered per unit of effort while keeping the door open for richer capabilities. Discord webhooks for Phase 1 are literally ~100 lines of Go code.

---

### Implementation Details

**What to implement first:**
- `pkg/discord/webhook.go` - Discord webhook client (POST JSON to webhook URL)
- Config schema update in `~/.orch/config.yaml` for `discord.webhook_url`
- Integration into daemon's `CompletionOnce()` for completion notifications

**Things to watch out for:**
- ⚠️ Discord webhook rate limits (30 requests/minute per webhook)
- ⚠️ Message formatting - Discord embeds have specific field limits (title: 256, description: 4096, fields: 25)
- ⚠️ Webhook URLs contain secrets - store securely, never log
- ⚠️ Notification fatigue - consider batching completions or adding configurable filters

**Areas needing further investigation:**
- Whether Discord threads can be managed via webhooks (limited - may need bot token for threads)
- Optimal notification format for mobile Discord (embeds vs plain text)
- Whether to add notification preferences per-agent-skill (e.g., only notify on architect completions)

**Success criteria:**
- ✅ Agent completion posts a message to configured Discord channel
- ✅ Blocker/stuck agent detection posts an alert
- ✅ Messages include beads ID, skill, runtime, and summary
- ✅ Notification can be disabled via config toggle

---

## References

**Files Examined:**
- `cmd/orch/serve.go:310-560` - orch serve HTTP API registration (50+ endpoints)
- `pkg/daemon/daemon.go` - Daemon package structure
- `.kb/guides/cli.md` - CLI command reference
- `.kb/guides/daemon.md` - Daemon architecture and completion detection
- `.kb/guides/spawn.md` - Spawn mechanics and backend architecture
- `.kb/guides/status.md` - Status command data sources
- `clawdbot/extensions/discord/index.ts` - Discord plugin registration
- `clawdbot/extensions/discord/src/channel.ts` - Discord channel plugin (422 lines)
- `clawdbot/extensions/discord/src/runtime.ts` - Runtime injection
- `clawdbot/src/gateway/hooks.ts` - Webhook hooks configuration
- `clawdbot/src/gateway/server/hooks.ts` - Hook dispatch (wake + agent)
- `clawdbot/src/gateway/server-http.ts` - Gateway HTTP server
- `clawdbot/skills/discord/SKILL.md` - Discord actions reference
- `clawdbot/src/discord/monitor/native-command.ts` - Native slash commands
- `clawdbot/AGENTS.md` - Clawdbot project structure and conventions

**Commands Run:**
```bash
# Pull latest code
git pull  # orch-go (had unstaged changes)
git pull  # clawdbot (updated to 875324e7c)

# Explore orch-go API surface
glob cmd/orch/serve*.go  # Found 42 serve-related files
grep "mux.HandleFunc" cmd/orch/serve.go  # Found 50+ registered endpoints

# Explore clawdbot Discord integration
glob extensions/discord/**/*.ts  # 3 core files
grep "hook|webhook" src/gateway/*.ts  # Found hooks system
grep "native.*command" src/discord/**/*.ts  # Found native slash commands
```

---

## Investigation History

**2026-02-06 16:xx:** Investigation started
- Initial question: What capabilities would an orch/clawdbot integration enable?
- Context: Spawned by orchestrator to assess integration possibilities between orch-go (agent orchestration) and clawdbot (Discord/messaging bot)

**2026-02-06 16:xx:** Codebase analysis complete
- Analyzed orch-go: 50+ REST API endpoints, SSE events, CLI commands, daemon with poll loop
- Analyzed clawdbot: Discord channel plugin, webhook hooks system, cron, native commands
- Both are local HTTP services, making integration straightforward

**2026-02-06 16:xx:** Investigation completed
- Status: Complete
- Key outcome: Phased integration recommended - Phase 1 (Discord webhooks, ~100 LOC) for notifications, Phase 2 (clawdbot hooks) for richer features, Phase 3 (Discord commands) for bidirectional orchestration
