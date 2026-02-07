<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode plugin loader crash caused by session-resume.js using v1 API (object export) instead of v2 API (function export).

**Evidence:** Server crashed with "fn3 is not a function" at src/plugin/index.ts:57:28 when loading session-resume.js; after migration to v2 API format, server starts successfully with no errors.

**Knowledge:** OpenCode v2 requires all plugins to export functions that accept PluginInput and return Hooks objects; v1 object exports with custom hook names (like on_session_created) are incompatible; TypeScript plugins already followed v2 API.

**Next:** Restore remaining TypeScript plugins from backup to active directory (they're already v2-compatible); verify server loads all plugins without errors.

**Promote to Decision:** recommend-no - Tactical bug fix, not architectural change (API migration pattern is known)

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

# Investigation: Opencode Plugin Loader Crashes Fn3

**Question:** Why does OpenCode plugin loader crash with "fn3 is not a function" error at src/plugin/index.ts:57:28?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Plugin loader expects functions, but session-resume.js exports an object

**Evidence:**
- OpenCode plugin type definition (packages/plugin/src/index.ts:35): `export type Plugin = (input: PluginInput) => Promise<Hooks>`
- Plugin loader code (packages/opencode/src/plugin/index.ts:54-59) iterates over ALL exports with `Object.entries(mod)` and calls each as `fn(input)`
- session-resume.js exports an object directly: `export default { name: 'session-resume', ... }` (not a function)
- When loader reaches this export, it tries to call the object as a function → "fn3 is not a function"

**Source:**
- ~/.config/opencode/plugin.backup/session-resume.js:15-55
- ~/Documents/personal/opencode/packages/plugin/src/index.ts:35
- ~/Documents/personal/opencode/packages/opencode/src/plugin/index.ts:54-59

**Significance:** This is the root cause. The plugin was written for an older API format that expected object exports, but OpenCode v2 requires function exports that return Hooks objects.

---

### Finding 2: Hook API changed - on_session_created doesn't exist in v2

**Evidence:**
- session-resume.js uses hook: `on_session_created: async (context) => { ... }`
- OpenCode v2 Hooks interface (packages/plugin/src/index.ts:146-216) defines valid hooks: event, config, tool, auth, chat.message, chat.params, permission.ask, tool.execute.before, tool.execute.after, experimental.*
- No `on_session_created` hook exists in the interface
- Correct approach: Use `event` hook and filter for session creation events

**Source:**
- ~/.config/opencode/plugin.backup/session-resume.js:27
- ~/Documents/personal/opencode/packages/plugin/src/index.ts:146-216

**Significance:** Even if we fix the export format, the hook name is wrong. The plugin needs to be migrated to use the v2 event hook system.

---

### Finding 3: Multiple plugins affected - TypeScript plugins already use correct format

**Evidence:**
- friction-capture.ts uses correct format: `export const FrictionCapturePlugin: Plugin = async ({ ... }) => { return { event: ... } }`
- This follows the v2 API: function that returns Hooks object
- Only session-resume.js uses the old object export format
- Other plugins in plugin.backup appear to be symlinks to TypeScript files that likely use correct format

**Source:**
- ~/.config/opencode/plugin.backup/friction-capture.ts:63-117
- ~/.config/opencode/plugin.backup/ directory listing

**Significance:** The issue is isolated to session-resume.js. TypeScript plugins are already compatible with v2 API. Only session-resume.js needs migration.

---

## Synthesis

**Key Insights:**

1. **API Version Mismatch** - session-resume.js was written for OpenCode v1 API which expected object exports with custom hook names. OpenCode v2 requires function exports that return Hooks objects with standardized hook names.

2. **Plugin Loader Assumes All Exports Are Functions** - The loader uses `Object.entries(mod)` to iterate over ALL exports and calls each as a function. When it encounters session-resume.js's default object export, it fails with "fn3 is not a function".

3. **Migration Path is Clear** - friction-capture.ts demonstrates the correct v2 format. The fix requires: (a) wrap the plugin in a function that accepts PluginInput, (b) use the `event` hook instead of `on_session_created`, (c) filter events by type within the event handler.

**Answer to Investigation Question:**

The crash occurs because session-resume.js exports an object directly (`export default { ... }`) instead of a function. The OpenCode v2 plugin loader (src/plugin/index.ts:57) calls `fn(input)` on each export, expecting all exports to be functions. When it tries to call the object as a function, JavaScript throws "fn3 is not a function" (fn3 being the third export processed).

The root issue is API version incompatibility: session-resume.js uses v1 format (object export + `on_session_created` hook) while OpenCode v2 requires function exports that return Hooks objects with standardized hook names.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin loader no longer crashes - OpenCode server started successfully after migration (verified: `ps aux | grep opencode`, server running on port 4096)
- ✅ No "fn3 is not a function" errors in logs (verified: `grep "fn3" ~/.opencode-server.log` returned empty)
- ✅ TypeScript plugins already use v2 API (verified: checked session-compaction.ts, guarded-files.ts, friction-capture.ts - all use `export const XPlugin: Plugin = async ({ ... }) => { ... }` format)
- ✅ Migrated plugin follows v2 API correctly (verified: session-resume.js now exports function that accepts PluginInput and returns Hooks object with event handler)

**What's untested:**

- ⚠️ Session resume functionality (requires SESSION_HANDOFF.md file to exist - not created for this test)
- ⚠️ Plugin execution at runtime (verified loading but not event handling - would require creating actual session)
- ⚠️ Integration with orch session resume command (assumes command works correctly)

**What would change this:**

- Finding would be wrong if server crashes with same error after restart with migrated plugin
- Finding would be wrong if plugin loader rejects the new format
- Finding would be wrong if other plugins also had v1 format (but all TypeScript plugins checked use v2)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Restore plugins from backup after migration** - Move migrated session-resume.js and TypeScript plugins back to active plugin directory.

**Why this approach:**
- TypeScript plugins were already v2-compatible (no migration needed)
- session-resume.js has been migrated to v2 API
- Server verified to load without crashes
- Critical session-resume functionality can be restored

**Trade-offs accepted:**
- Session resume functionality not fully tested (requires handoff file setup)
- Runtime behavior verified through loading, not execution
- Acceptable because: API contract verified, server loads successfully, pattern matches working plugins

**Implementation sequence:**
1. Keep migrated session-resume.js in ~/.config/opencode/plugin/ (already done)
2. Move TypeScript plugins from plugin.backup/ to plugin/ (guarded-files.ts, friction-capture.ts, session-compaction.ts)
3. Test server restart to verify all plugins load without errors
4. Remove or archive plugin.backup/ directory after verification

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
