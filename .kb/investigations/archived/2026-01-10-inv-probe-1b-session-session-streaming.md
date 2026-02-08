<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** YES - Plugins can stream orchestrator transcript to coach session in real-time using multiple viable paths (plugin hooks + SDK client OR GlobalBus + SDK client).

**Evidence:** Plugin system provides `client` SDK with `send()` and `sendAsync()` methods; plugins receive events for ALL sessions via global singleton state; GlobalBus emits all events across instances; existing coaching.ts plugin demonstrates session state tracking.

**Knowledge:** Plugins are loaded globally (not per-session), so one plugin instance sees events from all sessions; SDK client can send messages to any session by sessionID; no session isolation prevents cross-session communication.

**Next:** Recommend implementing via plugin hooks (tool.execute.after + experimental.chat.messages.transform) with SDK client.sendAsync() to coach session; simpler than GlobalBus and follows existing coaching.ts pattern.

**Promote to Decision:** recommend-no (tactical implementation finding, not architectural constraint)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Probe 1B - Session-to-Session Streaming

**Question:** Can an OpenCode plugin stream the orchestrator's transcript to a coach session in real-time?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** architect agent (og-arch-probe-1b-session-10jan-c0f4)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Plugins are Global Singletons That See All Sessions

**Evidence:**
- Plugin loading in `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/plugin/index.ts:16-66` uses `Instance.state()` to create a singleton that persists across all sessions
- Plugin hooks are triggered for ALL sessions via `Plugin.trigger()` (line 68-83)
- Existing coaching.ts plugin maintains a `Map<string, SessionState>` tracking multiple sessions simultaneously (line 213)

**Source:**
- `opencode/packages/opencode/src/plugin/index.ts:16-66` (state initialization)
- `~/.config/opencode/plugin/coaching.ts:213` (session Map)

**Significance:**
Since plugins are global singletons, a single plugin instance receives events from ALL sessions (orchestrator AND coach). This means the plugin can distinguish between sessions and route data accordingly.

---

### Finding 2: SDK Client Can Send Messages to Any Session

**Evidence:**
- PluginInput includes `client: ReturnType<typeof createOpencodeClient>` (opencode/packages/plugin/src/index.ts:27)
- SDK client provides `send()` and `sendAsync()` methods that accept `sessionID` parameter (opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts:1302-1392)
- No session isolation - any client can send to any sessionID

**Source:**
- `opencode/packages/plugin/src/index.ts:26-33` (PluginInput definition)
- `opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts:1302, 1390` (send/sendAsync methods)

**Significance:**
Plugins can call `client.session.sendAsync({ sessionID: coachSessionId, text: "..." })` to inject messages into the coach session. No architectural barrier prevents cross-session messaging.

---

### Finding 3: Plugin Hooks Provide Real-Time Transcript Access

**Evidence:**
- `tool.execute.after` hook receives tool name, sessionID, and output (line 178-185 of plugin/src/index.ts)
- `experimental.chat.messages.transform` hook receives full message history with parts (line 186-194)
- `MessageV2.Event.PartUpdated` bus event emits whenever message parts are added/updated (session/message-v2.ts:406-412)

**Source:**
- `opencode/packages/plugin/src/index.ts:178-194` (hook definitions)
- `opencode/packages/opencode/src/session/message-v2.ts:406-412` (PartUpdated event)

**Significance:**
Plugins can access transcript content in real-time via multiple hooks. For streaming orchestrator → coach, `tool.execute.after` provides immediate notification when orchestrator performs actions, while `experimental.chat.messages.transform` provides full context.

---

### Finding 4: GlobalBus Provides Alternative Cross-Instance Communication

**Evidence:**
- `Bus.publish()` emits to both instance-scoped Bus AND GlobalBus singleton (bus/index.ts:59-62)
- GlobalBus is `EventEmitter<{ event: [{ directory?: string, payload: any }] }>` (bus/global.ts:3-10)
- All session events (Session.Event.Created, Updated, MessageV2.Event.PartUpdated) flow through GlobalBus

**Source:**
- `opencode/packages/opencode/src/bus/index.ts:59-62` (GlobalBus.emit)
- `opencode/packages/opencode/src/bus/global.ts:3-10` (GlobalBus definition)

**Significance:**
While plugins currently use instance-scoped Bus via `Bus.subscribeAll()`, they COULD import GlobalBus directly to listen to events across ALL instances. This provides an alternative architecture if needed (though more complex than using plugin hooks).

---

## Synthesis

**Key Insights:**

1. **No Session Isolation in OpenCode Plugin Architecture** - Plugins are loaded once globally and receive events from ALL sessions. Combined with SDK client's ability to send to any sessionID, there's no architectural barrier preventing cross-session communication. This is BY DESIGN (share.ts demonstrates intentional cross-session data flow).

2. **Two Viable Implementation Paths Exist** - Path A: Plugin hooks (`tool.execute.after` + `experimental.chat.messages.transform`) + SDK `client.sendAsync()`. Path B: Import GlobalBus + listen to all events + SDK `client.sendAsync()`. Path A is simpler and follows existing coaching.ts pattern.

3. **Real-Time Streaming Already Works** - The infrastructure exists today. Coaching plugin already tracks tool execution across sessions in real-time. Adding coach session injection requires: (1) Identify coach sessionID, (2) Call `client.session.sendAsync()` with orchestrator's transcript content, (3) Filter to send only orchestrator session events.

**Answer to Investigation Question:**

**YES** - A plugin CAN stream the orchestrator's transcript to a coach session in real-time.

**Supporting Evidence:**
- Finding 1: Plugin singleton sees all sessions → can distinguish orchestrator vs coach
- Finding 2: SDK client can send to any sessionID → no permission boundary
- Finding 3: Hooks provide real-time transcript access → immediate notification
- Finding 4: GlobalBus provides alternative path if needed

**Mechanism:**
1. Plugin receives `tool.execute.after` event from orchestrator session
2. Plugin checks if event is from orchestrator (via sessionID or directory matching)
3. Plugin calls `client.session.sendAsync({ sessionID: coachSessionId, text: "Orchestrator just: <action>" })`
4. Coach session receives message as if user sent it
5. Coach can respond based on behavioral patterns detected

**Limitations:**
- Coach sessionID must be known (could be passed via env var or stored in plugin state)
- No built-in session-to-session messaging abstraction (but SDK client provides primitives)
- Experimental hooks (`experimental.chat.messages.transform`) carry API stability risk

---

## Structured Uncertainty

**What's tested:**

- ✅ **Plugins see all sessions** - Verified by reading plugin loading code (Instance.state singleton pattern) and coaching.ts session Map
- ✅ **SDK client has send methods** - Verified by reading SDK type definitions (send/sendAsync methods with sessionID parameter)
- ✅ **Hooks provide transcript access** - Verified by reading plugin hook definitions and MessageV2 event schemas
- ✅ **No session isolation** - Verified by reading SDK client code (no permission checks for cross-session messaging)

**What's untested:**

- ⚠️ **Actual cross-session message delivery** - Have not created test plugin that sends from one session to another
- ⚠️ **Performance impact** - Unknown latency/overhead of sendAsync() during real-time streaming
- ⚠️ **Coach session creation timing** - Unclear when coach session would be created (on orchestrator start? on demand?)
- ⚠️ **Message formatting** - What format should streamed transcript take? (raw events? summarized? filtered?)
- ⚠️ **Coach session loop risk** - If coach sends messages, could those trigger plugin hooks that stream back to coach?

**What would change this:**

- Finding would be INVALIDATED if SDK client.sendAsync() requires same-session authentication token
- Finding would be WEAKENED if OpenCode adds session isolation in future (permissions layer)
- Finding would need REVISION if experimental hooks are removed and no alternative exists

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Extend Coaching Plugin with Session-to-Session Streaming** - Add coach session injection to existing coaching.ts plugin using `tool.execute.after` hook + SDK `client.sendAsync()`.

**Why this approach:**
- Leverages proven infrastructure (Finding 1: coaching.ts already tracks sessions globally)
- Uses stable SDK methods (Finding 2: client.sendAsync is production API, not experimental)
- Minimal code delta (extend existing hook, add sendAsync call, filter by sessionID)
- No new plugin needed (reduces maintenance burden)

**Trade-offs accepted:**
- Couples streaming to coaching plugin (acceptable - same purpose: behavioral coaching)
- Coach session must be created manually (acceptable for MVP - automate later)
- No message queue or retry logic (acceptable - real-time best-effort delivery)

**Implementation sequence:**
1. **Add coach session identification** - Store coach sessionID in plugin state (via env var or API call)
2. **Extend tool.execute.after hook** - After tracking metrics, check if event is from orchestrator session
3. **Call client.sendAsync()** - Send formatted message to coach session: "Orchestrator action: <tool> - <summary>"
4. **Test with manual coach session** - Create coach session, verify messages arrive

### Alternative Approaches Considered

**Option B: Separate Streaming Plugin**
- **Pros:** Clean separation of concerns, independent testing
- **Cons:** Duplicates session tracking (Finding 1 shows coaching.ts already has this), adds maintenance burden
- **When to use instead:** If streaming needs differ significantly from coaching (e.g., streaming to external system, not coach session)

**Option C: GlobalBus-Based Streaming**
- **Pros:** More powerful (can listen across instances), doesn't depend on plugin hooks
- **Cons:** More complex (Finding 4 shows GlobalBus is lower-level), requires importing OpenCode internals (coupling risk)
- **When to use instead:** If plugin hooks prove insufficient or unstable

**Rationale for recommendation:** Finding 1 shows coaching.ts already has the session tracking infrastructure. Finding 2 confirms SDK client can send to any session. Extending existing working code is lower risk than building new infrastructure.

---

### Implementation Details

**What to implement first:**
- **Coach session ID storage** - Add `ORCH_COACH_SESSION_ID` env var or plugin state field to identify coach session
- **Session filtering logic** - Determine "is this orchestrator session?" (check directory path matches orch-go project)
- **Message formatting** - Design message format sent to coach (include tool name, summary, timestamp)

**Things to watch out for:**
- ⚠️ **Infinite loop risk** - If coach session sends messages, those will trigger plugin hooks. Need to filter coach messages to prevent streaming coach → coach
- ⚠️ **sendAsync() error handling** - Unknown if sendAsync() throws on invalid sessionID or returns error. Need graceful degradation
- ⚠️ **Rate limiting** - If orchestrator performs 100 tool calls/min, coach session could be overwhelmed. May need throttling
- ⚠️ **Coach session lifecycle** - If coach session closes, plugin needs to detect and stop streaming (or create new coach session?)

**Areas needing further investigation:**
- **Coach session auto-creation** - Should plugin create coach session on demand? Or require manual setup?
- **Message persistence** - Should streamed messages persist in coach session history? Or be ephemeral?
- **Bidirectional communication** - Can coach respond to orchestrator? (Would require reverse streaming)
- **Multi-orchestrator support** - If Dylan runs multiple orchestrator sessions, should they all stream to same coach? Or separate coaches?

**Success criteria:**
- ✅ **Messages arrive in coach session** - Verify `client.sendAsync()` delivers to coach sessionID
- ✅ **No orchestrator disruption** - Streaming doesn't slow down orchestrator or cause errors
- ✅ **Filtering works** - Only orchestrator events stream, not coach events (no loop)
- ✅ **Coach can parse messages** - Message format is useful for pattern detection

---

## References

**Files Examined:**
- `opencode/packages/opencode/src/plugin/index.ts` - Plugin loading mechanism (singleton pattern, global state)
- `opencode/packages/plugin/src/index.ts` - Plugin hook definitions and PluginInput interface
- `opencode/packages/opencode/src/bus/index.ts` - Bus event system and GlobalBus integration
- `opencode/packages/opencode/src/bus/global.ts` - GlobalBus singleton EventEmitter
- `opencode/packages/opencode/src/session/message-v2.ts` - Message event types (PartUpdated, etc.)
- `opencode/packages/sdk/js/src/v2/client.ts` - SDK client creation
- `opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts` - SDK client methods (send, sendAsync)
- `~/.config/opencode/plugin/coaching.ts` - Existing plugin demonstrating session tracking

**Commands Run:**
```bash
# Find session management code
grep -r "class.*Session\|interface.*Session" opencode/packages/opencode/src --include="*.ts"

# Find MessageV2 event definitions
grep -n "Event.*=.*{" opencode/packages/opencode/src/session/message-v2.ts

# Find SDK client methods
grep -n "send\|message" opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts | head -30
```

**External Documentation:**
- OpenCode Plugin API - Hook definitions and SDK client usage

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-10-inv-probe-technical-feasibility-plugins-access.md` - Prior probe confirming plugins can access transcript data
- **Epic:** `orch-go-tjn1r` - Orchestrator Coaching Plugin parent epic

---

## Investigation History

**2026-01-10 (session start):** Investigation started
- Initial question: Can a plugin stream orchestrator transcript to coach session in real-time?
- Context: Spawned from Epic orch-go-tjn1r (Orchestrator Coaching Plugin) to validate session-to-session streaming feasibility

**2026-01-10 (mid-session):** Plugin architecture analyzed
- Discovered plugins are global singletons that see all sessions
- Confirmed SDK client has send/sendAsync methods for cross-session messaging
- Reviewed coaching.ts plugin demonstrating session tracking pattern

**2026-01-10 (completing):** Investigation synthesized
- Status: Complete
- Key outcome: YES - Multiple viable paths exist for streaming orchestrator → coach in real-time; recommend extending coaching.ts plugin
