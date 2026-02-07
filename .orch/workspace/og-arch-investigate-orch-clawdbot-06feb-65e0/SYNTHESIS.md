# Session Synthesis

**Agent:** og-arch-investigate-orch-clawdbot-06feb-65e0
**Issue:** orch-go-21418
**Duration:** 2026-02-06 → 2026-02-06
**Outcome:** success

---

## TLDR

Investigated integration possibilities between orch-go (agent orchestration) and clawdbot (Discord/messaging bot). Found both systems have clean HTTP APIs making integration straightforward. Recommend a phased approach: Phase 1 uses Discord webhooks for notifications (~100 LOC, 1-2 days), Phase 2 adds clawdbot hooks for richer features, Phase 3 enables bidirectional Discord commands.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-06-inv-investigate-orch-clawdbot-integration-possibilities.md` - Full investigation with capability matrix, feasibility assessment, 3-phase implementation plan, and fork analysis with substrate-based recommendations

### Files Modified
- None (investigation-only session)

### Commits
- (pending commit of investigation and synthesis)

---

## Evidence (What Was Observed)

- Orch-go `serve` command registers 50+ REST API endpoints on port 3348 (HTTPS) covering agents, beads, daemon, usage, attention, frontier, and more (`cmd/orch/serve.go:358-530`)
- Clawdbot has a webhook hooks system at `/hooks` with bearer token auth, supporting wake hooks and isolated agent turns (`src/gateway/hooks.ts`, `src/gateway/server/hooks.ts`)
- Discord channel plugin provides 20+ actions: sendMessage, react, threadCreate, setPresence, polls, etc. (`skills/discord/SKILL.md`)
- Clawdbot supports native Discord slash commands (`src/discord/monitor/native-command.ts`) and cron jobs for periodic tasks
- Both systems run locally on the same machine - integration is localhost HTTP calls
- Clawdbot was successfully pulled (updated from 4027b35 to 875324e) with many new features including Feishu channel, Cloudflare AI Gateway, and voice call improvements

### Tests Run
```bash
# No code changes, so no tests needed
# Investigation-only session
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-06-inv-investigate-orch-clawdbot-integration-possibilities.md` - Complete integration assessment with capability matrix

### Decisions Made
- Recommended: Phased approach (webhooks → clawdbot hooks → bidirectional) because it maximizes value per unit of effort
- Recommended: Discord webhooks for Phase 1 (not clawdbot hooks) because it removes dependency on clawdbot running
- Recommended: Daemon extension (not standalone bridge) because daemon already has the poll loop and completion detection

### Constraints Discovered
- Discord webhook rate limit: 30 requests/minute per webhook URL
- Discord embeds have specific field limits (title: 256 chars, description: 4096 chars, max 25 fields)
- Webhook URLs are send-only - bidirectional communication requires bot token or clawdbot hooks
- Clawdbot hooks require `hooks.enabled: true` and `hooks.token` in config

### Externalized via `kn`
- N/A (recommendations pending Dylan's strategic decision)

---

## Issues Created

No discovered work during this session. This was a pure investigation with no implementation.

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** Should we proceed with orch/clawdbot integration, and which phase to start with?

**Options:**
1. **Phase 1: Discord webhooks for notifications** - ~100 LOC Go, 1-2 days, send-only, zero clawdbot dependency. Delivers completion/blocker/capacity notifications to Discord.
2. **Phase 2: Clawdbot hooks integration** - Richer features (threads, reactions, bot presence), requires clawdbot running, enables better formatting via clawdbot's agent.
3. **Phase 3: Bidirectional orchestration from Discord** - Native slash commands or conversational orchestration, highest complexity, enables remote work management.
4. **Skip integration entirely** - Dashboard + CLI already provide orchestration visibility, Discord adds a second notification surface to maintain.

**Recommendation:** Start with Phase 1. Discord webhooks are trivially simple (~100 lines of Go), provide immediate value for async awareness (know when agents complete without watching the dashboard), and don't create any dependency on clawdbot. If Phase 1 proves valuable, graduate to Phase 2.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could the orch dashboard itself be embedded in Discord via Discord's "Activities" feature?
- Should notification preferences be per-skill (e.g., only notify on architect completions)?
- Could clawdbot's memory system be used to maintain orchestration context across Discord conversations?

**Areas worth exploring further:**
- Discord's thread API limits (webhooks may not support thread management)
- Whether orch's SSE `/api/events` stream can drive real-time Discord notifications vs polling
- Mobile Discord notification behavior (do embeds render well on phone?)

**What remains unclear:**
- Whether Dylan wants notifications on a personal Discord server or a team/project server
- Volume expectations - how many notifications per day would be useful vs noisy
- Whether this should eventually replace email/other notification mechanisms

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-investigate-orch-clawdbot-06feb-65e0/`
**Investigation:** `.kb/investigations/2026-02-06-inv-investigate-orch-clawdbot-integration-possibilities.md`
**Beads:** `bd show orch-go-21418`
