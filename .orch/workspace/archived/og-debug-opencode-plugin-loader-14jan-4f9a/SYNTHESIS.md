# Synthesis: OpenCode Plugin Loader Crash Investigation

**Agent:** systematic-debugging  
**Issue:** orch-go-p54r4  
**Date:** 2026-01-14  
**Duration:** ~1.5 hours

---

## TLDR (30-Second Summary)

**Problem:** OpenCode plugin loader crashed with "fn3 is not a function" error, HTTP 500 on /health endpoint.

**Root Cause:** session-resume.js plugin used v1 API format (object export with `on_session_created` hook) instead of v2 API (function export returning Hooks object with standardized hooks).

**Fix:** Migrated session-resume.js to v2 API format; verified TypeScript plugins already v2-compatible; restored all plugins to active directory.

**Result:** Server loads all plugins without errors; crash resolved.

---

## What Was Done

### Investigation (Phase 1-2)
1. Examined error location: src/plugin/index.ts:57:28 - `fn(input)` call on non-function object
2. Analyzed plugin loader code - iterates ALL exports, expects functions
3. Checked session-resume.js - exports object directly (v1 format)
4. Reviewed OpenCode v2 plugin API - requires function exports returning Hooks
5. Verified TypeScript plugins (friction-capture.ts, session-compaction.ts, guarded-files.ts) already v2-compatible

### Implementation (Phase 3-4)
1. Migrated session-resume.js:
   - Changed from `export default { ... }` to `export default async ({ client, directory }) => { return { ... } }`
   - Replaced `on_session_created` hook with `event` hook
   - Added event type filtering: `if (event.type !== 'session.created') return`
   - Used OpenCode SDK client for message injection: `client.session.prompt(...)`
2. Restored TypeScript plugins from backup to active directory
3. Tested server restart - no errors, all plugins load successfully

### Verification
- ✅ Server starts without crashes
- ✅ No "fn3 is not a function" errors in logs
- ✅ All 4 plugins loaded (session-resume.js + 3 TypeScript plugins)
- ⚠️ Runtime behavior not fully tested (requires session handoff files)

---

## Key Findings

### Finding 1: API Version Incompatibility
- **What:** session-resume.js used v1 object export format incompatible with v2 plugin loader
- **Why it matters:** Loader assumes ALL exports are functions, crashes when encountering objects
- **Evidence:** Plugin loader code (index.ts:54-59) calls `fn(input)` without type checking

### Finding 2: Hook Name Changed
- **What:** v1 used custom hook `on_session_created`, v2 uses standardized `event` hook
- **Why it matters:** Even with correct export format, hook wouldn't fire without migration
- **Evidence:** Hooks interface (packages/plugin/src/index.ts:146-216) defines valid hooks

### Finding 3: TypeScript Plugins Already Migrated
- **What:** All TypeScript plugins already used v2 API format
- **Why it matters:** Only session-resume.js needed migration, not a widespread issue
- **Evidence:** Checked friction-capture.ts, session-compaction.ts, guarded-files.ts - all export functions

---

## Decisions Made

### Decision 1: Migrate in Place Rather Than Rewrite
- **Choice:** Update existing session-resume.js file structure
- **Why:** Preserves functionality, minimal changes, follows existing TypeScript plugin patterns
- **Trade-off:** Kept JavaScript (didn't convert to TypeScript) for faster fix

### Decision 2: Event-Based Session Detection
- **Choice:** Use `event` hook with `event.type === 'session.created'` filtering
- **Why:** Only v2-compatible approach; matches OpenCode event system
- **Alternative considered:** Custom hook name (not supported in v2)

---

## Technical Details

### Original Code (v1 API)
```javascript
export default {
  name: 'session-resume',
  on_session_created: async (context) => {
    const { cwd, sendSystemMessage } = context;
    // ... implementation
  }
}
```

### Migrated Code (v2 API)
```javascript
export default async ({ client, directory }) => {
  return {
    event: async ({ event }) => {
      if (event.type !== 'session.created') return;
      const sessionID = event.properties?.info?.id;
      // ... implementation using client.session.prompt(...)
    }
  }
}
```

### Key Changes
1. **Export:** Object → Function accepting PluginInput
2. **Hook:** `on_session_created` → `event` with type filtering
3. **Message injection:** `sendSystemMessage` → `client.session.prompt`
4. **Duplicate prevention:** Added `injectedSessions` Set to track processed sessions

---

## Lessons Learned

### 1. Plugin API Migration Pattern
- OpenCode v2 requires strict function exports (not objects)
- Plugin loader doesn't validate export types before calling
- Error message "fn3 is not a function" indicates third export in iteration

### 2. Event-Based Hook System
- v2 uses generic `event` hook for all events
- Plugins filter by `event.type` instead of hook-specific callbacks
- Event payload structure: `{ type: "session.created", properties: { info: Session } }`

### 3. Mixed Plugin Formats
- TypeScript plugins tend to follow type system (caught v2 API early)
- JavaScript plugins can lag behind API changes
- Check ALL plugin files during API migrations, not just TypeScript

---

## Follow-Up Actions

### Immediate
- [x] Migrated session-resume.js to v2 API
- [x] Restored TypeScript plugins to active directory
- [x] Verified server loads without errors

### Deferred
- [ ] Full end-to-end test of session resume functionality (requires handoff file setup)
- [ ] Consider converting session-resume.js to TypeScript for type safety
- [ ] Document plugin API migration pattern for future reference

### Not Needed
- ~~Migrate TypeScript plugins~~ (already v2-compatible)
- ~~Fix plugin loader~~ (working as designed for v2 API)

---

## Files Modified

1. `~/.config/opencode/plugin/session-resume.js` - Migrated to v2 API (102 lines)
2. `~/.config/opencode/plugin/friction-capture.ts` - Restored from backup
3. `~/.config/opencode/plugin/session-compaction.ts` - Restored from backup
4. `~/.config/opencode/plugin/guarded-files.ts` - Restored from backup

---

## References

### Investigation File
- `.kb/investigations/2026-01-14-inv-opencode-plugin-loader-crashes-fn3.md`

### Code Examined
- `~/Documents/personal/opencode/packages/opencode/src/plugin/index.ts` (plugin loader)
- `~/Documents/personal/opencode/packages/plugin/src/index.ts` (Plugin and Hooks types)
- `~/Documents/personal/opencode/packages/plugin/src/example.ts` (v2 plugin example)

### External Documentation
- OpenCode plugin API (inferred from types, no official v2 docs found)

---

## Success Criteria

✅ **Primary:** Server starts without "fn3 is not a function" error  
✅ **Primary:** All plugins load successfully  
⚠️ **Secondary:** Session resume functionality works (not fully tested)

**Overall Status:** SUCCESS - Crash resolved, plugins restored, server operational.
