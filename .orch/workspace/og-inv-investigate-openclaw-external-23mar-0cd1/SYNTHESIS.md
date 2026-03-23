# Session Synthesis

**Agent:** og-inv-investigate-openclaw-external-23mar-0cd1
**Issue:** orch-go-hqsgm
**Duration:** 2026-03-23 13:56 → 2026-03-23 14:30
**Outcome:** success

---

## TLDR

OpenClaw exposes a comprehensive WebSocket RPC API (114+ methods) that provides complete programmatic agent control — session create, prompt send, wait-for-completion, status monitoring, and cleanup. orch-go can drive OpenClaw agents with a richer API surface than it currently has with OpenCode, and the recommended migration path is a WebSocket RPC client (`pkg/openclaw/client.go`) that replaces the SSE-based OpenCode integration.

---

## Plain-Language Summary

I investigated whether orch-go could control OpenClaw agents the same way it currently controls OpenCode agents (creating sessions, sending prompts, monitoring progress, getting results). The answer is yes — and the API is better. OpenClaw's gateway daemon exposes a WebSocket API with methods like `agent` (send a prompt), `agent.wait` (wait for completion), and `sessions.list` (check status). These replace orch-go's most fragile integration — SSE stream parsing — with a simple request/response pattern. OpenClaw also runs headless (no Slack/Discord needed) and delegates to the same `claude -p` CLI that OpenCode uses, so the actual code execution is identical. The practical migration is building a ~300-line Go WebSocket client.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-23-inv-investigate-openclaw-external-api-surface.md` — Full investigation with 6 findings and architecture recommendation

### Files Modified
- None

### Commits
- (pending)

---

## Evidence (What Was Observed)

- OpenClaw WebSocket `agent` method accepts `message`, `provider`, `model`, `sessionKey`, `extraSystemPrompt`, `idempotencyKey` params (source: `clawdbot/src/gateway/server-methods/agent.ts:211-244`)
- `agent.wait` polls for completion and returns `{runId, status, startedAt, endedAt, error}` (source: `agent.ts:783-869`)
- `sessions.create`/`sessions.send`/`sessions.list`/`sessions.delete` provide full session lifecycle (source: `sessions.ts:625-800`)
- Default Claude CLI backend uses identical `claude -p --output-format json --permission-mode bypassPermissions` invocation as orch-go (source: `cli-backends.ts:40-70`)
- Headless deployment confirmed via `fly.toml` and `docker-compose.yml` — no messaging channels required
- HTTP alternatives exist at `/v1/chat/completions`, `/hooks/agent`, and `/api/sessions/kill`
- Plugin runtime exposes `subagent.run()`/`waitForRun()` for in-process control (source: `plugins/runtime/types.ts:54-66`)

### Tests Run
```bash
# Source code examination only — no runtime tests (OpenClaw not running locally)
# Verified claims against source code at ~/Documents/personal/clawdbot/
```

---

## Architectural Choices

### Option A (WebSocket RPC) recommended over Option C (coordination plugin)
- **What I chose:** Recommend orch-go drives OpenClaw via WebSocket as a backend swap
- **What I rejected:** Reimplementing orch-go's coordination model as an OpenClaw TypeScript plugin
- **Why:** Option A preserves orch-go's daemon/coordination/beads layers; Option C requires reimplementing everything in TypeScript — different strategic decision
- **Risk accepted:** WebSocket client is more complex than HTTP client; adds reconnection logic

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-23-inv-investigate-openclaw-external-api-surface.md` — Complete API mapping with 6 findings

### Decisions Made
- Decision 1: Three architectures exist (WebSocket RPC, HTTP hooks, coordination plugin) — Option A recommended because it minimizes migration risk while capturing biggest wins

### Constraints Discovered
- `agent.wait` default timeout is 30s — orch-go agents run 30-60min, need repeated polling or long timeout
- `extraSystemPrompt` may have size limits — need to verify if it can carry full SPAWN_CONTEXT.md content
- WebSocket auth requires `operator.admin` scope for model override capability

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with 6 findings + architecture recommendation)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-hqsgm`

---

## Unexplored Questions

- How does OpenClaw handle long-running agent sessions (>30min) with `agent.wait`? Repeated polling vs WebSocket events?
- Can `extraSystemPrompt` carry ~10KB of context (full SPAWN_CONTEXT.md)?
- What is the WebSocket event subscription model for real-time status updates?
- How to configure OpenClaw's workspace/agent identity for orch-go integration?

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification contract.

---

## Friction

No friction — smooth session. Three parallel subagents explored the codebase efficiently.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-openclaw-external-23mar-0cd1/`
**Investigation:** `.kb/investigations/2026-03-23-inv-investigate-openclaw-external-api-surface.md`
**Beads:** `bd show orch-go-hqsgm`
