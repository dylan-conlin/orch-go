<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The session.created event DOES include the directory property at `event.properties.info.directory`, enabling directory-based worker detection.

**Evidence:** Verified in OpenCode source: Session.Info schema (session/index.ts:39-84) includes `directory: z.string()`, and hook handler (hook/index.ts:67) uses `payload.properties.info.directory`.

**Knowledge:** Worker detection can happen at session.created time by checking if `info.directory` contains `.orch/workspace/`. Additionally, metadata.role exists in the schema but may not be reliably set.

**Next:** Proceed to implement directory-based worker detection in coaching.ts event hook using `event.properties.info.directory`.

**Promote to Decision:** recommend-no (tactical implementation of existing recommendation)

---

# Investigation: Verify session.created Event Has Directory Property

**Question:** Does the session.created event payload include the directory property needed for directory-based worker detection?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** orch-go-4fwje
**Phase:** Complete
**Next Step:** None - verified, implementation can proceed
**Status:** Complete

<!-- Lineage -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Session.Info Schema Includes Directory

**Evidence:** The Session.Info zod schema at `packages/opencode/src/session/index.ts:39-84` includes:

```typescript
export const Info = z
  .object({
    id: Identifier.schema("session"),
    projectID: z.string(),
    directory: z.string(),  // <-- DIRECTORY IS INCLUDED
    parentID: Identifier.schema("session").optional(),
    // ... other fields
    metadata: z
      .object({
        role: z.enum(["orchestrator", "meta-orchestrator", "worker"]).optional(),
      })
      .optional(),
  })
```

**Source:** `~/Documents/personal/opencode/packages/opencode/src/session/index.ts:39-84`

**Significance:** The directory field is a required string in the schema, meaning it will always be present in session.created events.

---

### Finding 2: Event.Created Includes Full Info Object

**Evidence:** The session.created event is defined at `session/index.ts:96-102`:

```typescript
export const Event = {
  Created: BusEvent.define(
    "session.created",
    z.object({
      info: Info,  // <-- Full Info object including directory
    }),
  ),
}
```

**Source:** `~/Documents/personal/opencode/packages/opencode/src/session/index.ts:96-102`

**Significance:** The event payload structure is `event.properties.info.*`, not `event.properties.*` directly.

---

### Finding 3: Hook Handler Confirms Access Path

**Evidence:** The existing hook handler at `hook/index.ts:57-68` demonstrates the correct access pattern:

```typescript
Bus.subscribe(Session.Event.Created, async (payload) => {
  log.info("session.created event received", { sessionID: payload.properties.info.id })
  // ...
  await run(hook, {
    OPENCODE_SESSION_ID: payload.properties.info.id,
    OPENCODE_SESSION_DIRECTORY: payload.properties.info.directory,  // <-- Used here
  })
})
```

**Source:** `~/Documents/personal/opencode/packages/opencode/src/hook/index.ts:57-68`

**Significance:** OpenCode itself uses `payload.properties.info.directory`, confirming the access path works.

---

### Finding 4: metadata.role Also Available But Optional

**Evidence:** The schema shows metadata.role exists:

```typescript
metadata: z
  .object({
    role: z.enum(["orchestrator", "meta-orchestrator", "worker"]).optional(),
  })
  .optional(),
```

**Source:** `~/Documents/personal/opencode/packages/opencode/src/session/index.ts:75-79`

**Significance:** metadata.role could be used as a fallback detection signal when available, but prior investigation notes it's not reliably set. Directory-based detection is more reliable.

---

## Synthesis

**Key Insights:**

1. **Directory is always available** - Unlike metadata.role which is optional and unreliable, directory is a required field in Session.Info, guaranteeing availability at session creation time.

2. **Correct access path is `event.properties.info.directory`** - The orchestrator-session.ts plugin currently uses `(event as any).properties?.sessionID` which is incorrect. The correct path is `event.properties.info.id` for session ID and `event.properties.info.directory` for directory.

3. **No OpenCode changes needed** - The directory is already exposed in the event; we just need to use it in the coaching plugin.

**Answer to Investigation Question:**

YES, the session.created event DOES include the directory property. It is accessible at `event.properties.info.directory` (not `event.properties.directory`). This confirms that directory-based worker detection can be implemented by checking if this path contains `.orch/workspace/` at session creation time.

---

## Structured Uncertainty

**What's tested:**

- ✅ **Session.Info schema includes `directory: z.string()`** (verified: read session/index.ts:43)
- ✅ **Event.Created includes full Info object** (verified: read session/index.ts:96-102)
- ✅ **Hook handler uses `payload.properties.info.directory`** (verified: read hook/index.ts:67)

**What's untested:**

- ⚠️ **Runtime event actually has directory populated** (schema says required, but not runtime tested)
- ⚠️ **Directory path format for workers** (assume `.orch/workspace/` based on spawn behavior)

**What would change this:**

- Finding would be wrong if OpenCode modified event emission to exclude info fields
- Finding would be wrong if directory is set to different value than working directory

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add event hook to coaching.ts** - Handle session.created events and mark sessions as workers when directory contains `.orch/workspace/`.

**Why this approach:**
- Uses signal available BEFORE any tool calls (Finding 1, 2)
- Correct access path now verified (Finding 3)
- No upstream changes needed (Finding 3)

**Trade-offs accepted:**
- Relies on directory path convention (stable, all workers use .orch/workspace/)
- TypeScript cast needed for event properties (OpenCode plugin types don't export event details)

**Implementation sequence:**
1. Add `event` hook to coaching plugin that handles `session.created`
2. Extract `info.directory` and `info.id` from `event.properties`
3. If directory contains `.orch/workspace/`, add sessionId to workerSessions cache
4. Existing tool-argument detection remains as backup

### Implementation Details

**Corrected implementation:**

```typescript
event: async ({ event }) => {
  if (event.type !== "session.created") return

  // Correct access path: event.properties.info.* (not event.properties.*)
  const info = (event as any).properties?.info
  if (!info) return

  const sessionId = info.id
  const directory = info.directory

  if (!sessionId) return

  // Early worker detection via directory path
  if (directory && directory.includes(".orch/workspace/")) {
    workerSessions.set(sessionId, true)
    log(`Worker detected (session.created, directory): ${sessionId}`)
  }
}
```

**Things to watch out for:**
- ⚠️ Access path is `event.properties.info.*` not `event.properties.*`
- ⚠️ Need TypeScript cast `(event as any)` due to untyped event properties

**Success criteria:**
- ✅ No coaching alerts fire on worker sessions
- ✅ Worker detection happens before first tool call
- ✅ Orchestrator sessions continue to receive coaching

---

## References

**Files Examined:**
- `~/Documents/personal/opencode/packages/opencode/src/session/index.ts:39-102` - Session.Info schema and Event definitions
- `~/Documents/personal/opencode/packages/opencode/src/hook/index.ts:57-68` - Hook handler showing usage
- `plugins/coaching.ts:1310-1877` - Current coaching plugin implementation
- `plugins/orchestrator-session.ts:244-281` - Current session.created handling (wrong access path)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-23-inv-review-coaching-plugin-worker-detection.md` - Prior investigation recommending this approach

---

## Investigation History

**2026-01-23:** Investigation started
- Initial question: Does session.created event have directory property?
- Context: Needed to verify before implementing directory-based worker detection

**2026-01-23:** Found Session.Info schema in OpenCode
- Confirmed directory is required string field in schema
- Confirmed event.properties.info.* is correct access path

**2026-01-23:** Investigation completed
- Status: Complete
- Key outcome: Session.created event DOES have directory at `event.properties.info.directory`
