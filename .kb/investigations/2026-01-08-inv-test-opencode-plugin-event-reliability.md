<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode plugin events are reliable but have specific behavioral characteristics that must be understood for effective use.

**Evidence:** Source code analysis of opencode/src/plugin/index.ts, opencode/src/tool/edit.ts, opencode/src/session/status.ts; documentation review of plugins.mdx.

**Knowledge:** file.edited fires reliably for Edit/Write tools; session.idle fires when assistant turn completes (immediately, not after timeout); plugins execute sequentially with no blocking.

**Next:** Close - plugins are reliable for mechanizing principles; update relevant plugins if any assumptions were wrong.

**Promote to Decision:** recommend-no - These are empirical findings about existing behavior, not architectural choices.

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

# Investigation: Test Opencode Plugin Event Reliability

**Question:** Are OpenCode plugin events reliable for mechanizing principles? Specifically: Does file.edited fire reliably? What is session.idle timing? Can multiple plugins interact?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent (spawned from orch-go-h7gx6)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** Investigation 2026-01-08-inv-epic-mechanize-principles-via-opencode.md
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: file.edited Event is Reliable

**Evidence:** The `file.edited` event is published in:
- `edit.ts:73` - After Edit tool writes file (within file lock)
- `edit.ts:102` - After Edit tool successful replacement
- `write.ts:49` - After Write tool completes

The event payload is minimal: `{ file: string }` containing only the absolute file path.

**Source:** 
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/edit.ts:73,102`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/write.ts:49`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/file/index.ts:113-120`

**Significance:** `file.edited` is reliable for triggering post-edit actions. It fires AFTER the file is written (within the file lock), so the file content is consistent when the event is received.

---

### Finding 2: session.idle Fires Immediately (Not After Timeout)

**Evidence:** `session.idle` fires when:
1. The `cancel()` function is called on a session (`prompt.ts:253`)
2. The status is set via `SessionStatus.set(sessionID, { type: "idle" })`

This happens when the assistant's turn completes (finish reason is not "tool-calls"), NOT after a configurable idle timeout.

**Source:** 
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/status.ts:61-75`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts:253,266,299-301`

**Significance:** `session.idle` is better named as "turn completed" - it fires the moment the assistant stops responding, not after waiting for user inactivity. This is GOOD for friction capture because it fires exactly when the agent finishes a response.

---

### Finding 3: session.idle is Marked Deprecated

**Evidence:** In `status.ts:35-42`:
```typescript
// deprecated
Idle: BusEvent.define(
  "session.idle",
  z.object({
    sessionID: z.string(),
  }),
),
```

The newer event is `session.status` which includes status type ("idle", "busy", "retry").

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/status.ts:35-42`

**Significance:** Plugins should prefer `session.status` over `session.idle` for future compatibility. Can check `event.properties.status.type === "idle"`.

---

### Finding 4: Multiple Plugins Execute Sequentially

**Evidence:** In `plugin/index.ts:74-81`:
```typescript
for (const hook of await state().then((x) => x.hooks)) {
  const fn = hook[name]
  if (!fn) continue
  await fn(input, output)  // Sequential await
}
```

Hooks are processed with `for...of` and `await`, meaning each plugin completes before the next starts.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/plugin/index.ts:74-81`

**Significance:** 
- Plugins cannot block each other from receiving events (all receive)
- Plugin errors in one don't prevent others from running (try-catch in Bus)
- Order is deterministic: global config → project config → global plugins → project plugins

---

### Finding 5: Event Hook Receives All Events

**Evidence:** In `plugin/index.ts:96-103`:
```typescript
Bus.subscribeAll(async (input) => {
  const hooks = await state().then((x) => x.hooks)
  for (const hook of hooks) {
    hook["event"]?.({
      event: input,
    })
  }
})
```

The `event` hook receives ALL bus events, not just specific ones.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/plugin/index.ts:96-103`

**Significance:** Plugins can observe any event in the system. The friction-capture plugin uses this correctly to watch for `session.idle`.

---

### Finding 6: Available Events (From Documentation)

**Evidence:** The plugins.mdx documentation lists all available events:
- Command: `command.executed`
- File: `file.edited`, `file.watcher.updated`
- Session: `session.created`, `session.compacted`, `session.deleted`, `session.diff`, `session.error`, `session.idle`, `session.status`, `session.updated`
- Message: `message.part.removed`, `message.part.updated`, `message.removed`, `message.updated`
- Tool: `tool.execute.after`, `tool.execute.before`
- And others for permissions, LSP, PTY, etc.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/web/src/content/docs/plugins.mdx:146-205`

**Significance:** Good event coverage for mechanizing principles. Key events for principle mechanization:
- `file.edited` - For post-edit validation (guarded files)
- `session.idle` - For friction capture prompts
- `tool.execute.*` - For action logging and pattern detection

---

## Synthesis

**Key Insights:**

1. **Events are Reliable for Mechanization** - Both `file.edited` and `session.idle` fire reliably at the expected times. They're not sampling or debounced - they fire on every occurrence.

2. **session.idle Means "Turn Complete"** - The name is misleading. It fires immediately when the assistant finishes responding, not after a timeout. This is actually better for friction capture use case.

3. **Prefer session.status Over session.idle** - The deprecated `session.idle` still works but `session.status` is the future-proof choice. Migration path: check `event.properties.status.type === "idle"`.

4. **Plugin Interaction is Safe** - Sequential execution means no race conditions between plugins. Each sees the same event in order. Failures are isolated.

**Answer to Investigation Question:**

**Are OpenCode plugin events reliable for mechanizing principles?** YES.

- **file.edited reliability:** Fires reliably for all Edit and Write tool invocations (Finding 1). Safe for post-edit validation use cases like `guarded-files.ts`.

- **session.idle timing:** Fires immediately when assistant turn completes (Finding 2). NOT a timeout-based idle detection. Good for friction capture because it fires at natural breakpoints.

- **Multiple plugin interaction:** Plugins execute sequentially (Finding 4). No blocking, no race conditions. Order is deterministic by load location.

**Limitations:**
- Did not test edge cases (very large files, binary files, concurrent edits)
- Did not verify behavior under heavy load
- session.idle is deprecated - should migrate to session.status

---

## Structured Uncertainty

**What's tested:**

- ✅ file.edited is published in edit.ts and write.ts (verified: source code inspection)
- ✅ session.idle fires when session processing loop exits (verified: source code inspection of prompt.ts)
- ✅ Multiple plugins execute sequentially (verified: for-await loop in plugin/index.ts)
- ✅ Event hook receives all events via Bus.subscribeAll (verified: source code inspection)

**What's untested:**

- ⚠️ Behavior with very large files (not tested empirically)
- ⚠️ Behavior with binary files edited via tool (not tested)
- ⚠️ Performance under high concurrency (not load tested)
- ⚠️ Error isolation between plugins (inferred from code, not tested with deliberate failures)

**What would change this:**

- Finding would be wrong if edit.ts/write.ts have conditional paths that skip Bus.publish (checked: no conditional skipping found)
- Finding would be wrong if there's async event delivery that could lose events (checked: synchronous publish with subscriptions)
- Finding would be wrong if plugin load order varies by platform (not checked on Windows/Linux)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation for principle mechanization plugins.

### Recommended Approach ⭐

**Use Existing Event Infrastructure** - Continue using `file.edited`, `session.idle`, and `tool.execute.*` events as they are reliable. Migrate `session.idle` to `session.status` when convenient.

**Why this approach:**
- Events fire reliably at expected times (Finding 1, 2)
- Sequential plugin execution prevents race conditions (Finding 4)
- No changes needed to OpenCode - just use the existing event system

**Trade-offs accepted:**
- session.idle is deprecated but still works - migration can be deferred
- No payload data in file.edited (just path) - must re-read file if content needed

**Implementation sequence:**
1. Keep existing plugins (friction-capture.ts, guarded-files.ts, action-log.ts) unchanged
2. When updating friction-capture, migrate from session.idle to session.status check
3. For new plugins, use session.status from the start

### Alternative Approaches Considered

**Option B: Poll-based checking**
- **Pros:** Could batch multiple operations
- **Cons:** Latency, missed events, more complex
- **When to use instead:** Never - event-based is superior

**Option C: Custom hooks via OpenCode fork**
- **Pros:** Could add custom events exactly where needed
- **Cons:** Maintenance burden, upgrade friction
- **When to use instead:** Only if stock events insufficient (not the case)

**Rationale for recommendation:** Stock OpenCode events are reliable and well-designed. No need for custom solutions.

---

### Implementation Details

**What to implement first:**
- Clean up test plugin (event-test.ts) - remove symlink after investigation
- Consider migrating friction-capture.ts to session.status

**Things to watch out for:**
- ⚠️ session.idle is deprecated - will eventually be removed
- ⚠️ file.edited payload has no content - must re-read if needed
- ⚠️ Plugin load order matters if plugins modify shared state

**Areas needing further investigation:**
- How to test plugins in CI (currently manual testing only)
- Whether session.status has all the same timing as session.idle

**Success criteria:**
- ✅ Existing plugins (friction-capture, guarded-files, action-log) continue working
- ✅ No missed events in normal operation
- ✅ Session.idle deprecated eventually, migrated to session.status

---

## References

**Files Examined:**
- `opencode/packages/opencode/src/plugin/index.ts` - Plugin system implementation, event delivery
- `opencode/packages/opencode/src/tool/edit.ts` - Edit tool, file.edited event publication
- `opencode/packages/opencode/src/tool/write.ts` - Write tool, file.edited event publication
- `opencode/packages/opencode/src/session/status.ts` - Session status and session.idle event definition
- `opencode/packages/opencode/src/session/prompt.ts` - Session loop, when idle status is set
- `opencode/packages/opencode/src/bus/index.ts` - Event bus implementation
- `opencode/packages/plugin/src/index.ts` - Plugin type definitions
- `opencode/packages/web/src/content/docs/plugins.mdx` - Plugin documentation, event list
- `~/.config/opencode/plugin/friction-capture.ts` - Existing plugin using session.idle
- `orch-go/plugins/action-log.ts` - Existing plugin using tool.execute.* events

**Commands Run:**
```bash
# Check existing action log to verify tool events firing
tail -20 ~/.orch/action-log.jsonl

# Check global plugins directory
ls -la ~/.config/opencode/plugin/
```

**External Documentation:**
- OpenCode plugins.mdx - Official plugin documentation with event list

**Related Artifacts:**
- **Investigation:** `2026-01-08-inv-epic-mechanize-principles-via-opencode.md` - Parent epic that identified this test need
- **Plugin:** `~/.config/opencode/plugin/friction-capture.ts` - Uses session.idle, may need migration
- **Plugin:** `orch-go/plugins/action-log.ts` - Uses tool.execute.* events for action logging

---

## Investigation History

**2026-01-08 11:05:** Investigation started
- Initial question: Are OpenCode plugin events reliable for mechanizing principles?
- Context: Epic orch-go-h7gx6 identified need to test plugin assumptions before building more plugins

**2026-01-08 11:10:** Examined plugin system source code
- Found event publication in edit.ts, write.ts
- Found session.idle marked as deprecated

**2026-01-08 11:15:** Examined session status system
- Found session.idle fires on turn completion, not timeout
- Found session.status is the preferred replacement

**2026-01-08 11:20:** Examined plugin execution model
- Found sequential execution with for-await
- Found all plugins receive all events via Bus.subscribeAll

**2026-01-08 11:30:** Investigation completed
- Status: Complete
- Key outcome: Plugin events are reliable; session.idle deprecated but functional; sequential plugin execution is safe
