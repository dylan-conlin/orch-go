# Session Synthesis

**Agent:** og-arch-probe-1b-session-10jan-c0f4
**Issue:** orch-go-pp9it
**Duration:** 2026-01-10 (session start) → 2026-01-10 (completing)
**Outcome:** success

---

## TLDR

Investigated whether OpenCode plugins can stream orchestrator transcript to coach session in real-time. **Answer: YES** - Multiple viable paths exist using plugin hooks + SDK client or GlobalBus + SDK client.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-10-inv-probe-1b-session-session-streaming.md` - Complete investigation with findings, synthesis, and implementation recommendations

### Files Modified
- None (investigation-only session)

### Commits
- Pending: Will commit investigation file before completion

---

## Evidence (What Was Observed)

### Finding 1: Plugins Are Global Singletons
- Source: `opencode/packages/opencode/src/plugin/index.ts:16-66`
- Plugins use `Instance.state()` to create singleton that persists across all sessions
- Plugin hooks triggered for ALL sessions via `Plugin.trigger()`
- Existing coaching.ts maintains `Map<string, SessionState>` tracking multiple sessions

### Finding 2: SDK Client Enables Cross-Session Messaging
- Source: `opencode/packages/plugin/src/index.ts:26-33`, `opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts:1302, 1390`
- PluginInput includes `client` SDK with `send()` and `sendAsync()` methods
- Methods accept `sessionID` parameter - no session isolation
- Any client can send to any sessionID

### Finding 3: Real-Time Transcript Access Available
- Source: `opencode/packages/plugin/src/index.ts:178-194`, `opencode/packages/opencode/src/session/message-v2.ts:406-412`
- `tool.execute.after` hook receives tool name, sessionID, output
- `experimental.chat.messages.transform` hook receives full message history
- `MessageV2.Event.PartUpdated` bus event emits on message part updates

### Finding 4: GlobalBus Provides Alternative Architecture
- Source: `opencode/packages/opencode/src/bus/index.ts:59-62`, `opencode/packages/opencode/src/bus/global.ts:3-10`
- `Bus.publish()` emits to both instance-scoped Bus AND GlobalBus singleton
- All session events flow through GlobalBus
- Plugins could import GlobalBus directly for cross-instance listening

### Tests Run
```bash
# No runtime tests performed - investigation via code reading
# Verified architecture via source code analysis
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-10-inv-probe-1b-session-session-streaming.md` - Technical feasibility investigation for session-to-session streaming

### Decisions Made
- **Decision 1:** Recommend Path A (plugin hooks + SDK client) over Path B (GlobalBus) because simpler and follows existing coaching.ts pattern
- **Decision 2:** Extend existing coaching.ts plugin rather than create new plugin (leverages existing session tracking infrastructure)

### Constraints Discovered
- **Coach session ID must be known** - Plugin needs to identify coach session (via env var or API call)
- **Infinite loop risk** - If coach sends messages, those trigger hooks; need filtering to prevent coach → coach streaming
- **No message queue** - Real-time best-effort delivery only (acceptable for MVP)

### Externalized via `kb`
- Not applicable - investigation findings documented in investigation file, not kb quick entries

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created and filled)
- [x] Investigation file has `**Phase:** Complete`
- [x] Synthesis document created
- [x] Ready for `orch complete orch-go-pp9it`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- **Coach session auto-creation** - Should plugin create coach session on demand? Or require manual setup?
- **Message persistence** - Should streamed messages persist in coach session history? Or be ephemeral?
- **Bidirectional communication** - Can coach respond to orchestrator? (Would require reverse streaming)
- **Multi-orchestrator support** - If Dylan runs multiple orchestrator sessions, should they all stream to same coach? Or separate coaches?
- **Rate limiting strategy** - If orchestrator performs 100 tool calls/min, how to prevent coach session overwhelm?

**Areas worth exploring further:**
- Test plugin creation - Validate cross-session messaging works in practice (create test plugin that sends from session A to B)
- Performance benchmarking - Measure sendAsync() latency during high-frequency tool execution
- Coach UX design - What message format is most useful for pattern detection?

**What remains unclear:**
- sendAsync() error handling behavior - Does it throw or return error on invalid sessionID?
- Coach session lifecycle management - How to detect coach session closure?
- Message filtering granularity - Should all tool calls stream, or only specific categories?

---

## Session Metadata

**Skill:** architect
**Model:** sonnet (default)
**Workspace:** `.orch/workspace/og-arch-probe-1b-session-10jan-c0f4/`
**Investigation:** `.kb/investigations/2026-01-10-inv-probe-1b-session-session-streaming.md`
**Beads:** `bd show orch-go-pp9it`
