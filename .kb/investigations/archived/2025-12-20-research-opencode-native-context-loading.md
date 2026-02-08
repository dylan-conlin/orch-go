**TLDR:** OpenCode's native configuration (`opencode.json`) is strictly static and limited to file paths in the `instructions` array. Dynamic context injection (like active agents or knowledge entries) requires the plugin system, which can be improved by using the `experimental.chat.system.transform` hook for transparent system prompt modification. High confidence (90%) based on SDK type analysis and existing plugin patterns.

---

# Investigation: OpenCode Native Context Loading

**Question:** Does OpenCode have native features for auto-loading dynamic context (like skills or system prompts) at session start, and how do they compare to the current plugin approach?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Native `instructions` are static
OpenCode supports an `instructions` array in `opencode.json` (project) and `~/.config/opencode/opencode.jsonc` (global).

**Evidence:** 
- `~/.config/opencode/opencode.jsonc` contains:
  ```json
  "instructions": ["~/.claude/CLAUDE.md"]
  ```
- SDK types define `instructions` as `Array<string>`.

**Source:** `~/.config/opencode/opencode.jsonc`, `@opencode-ai/sdk/dist/gen/types.gen.d.ts`

**Significance:** This is the native way to load "always-on" instructions, but it is limited to static file paths. It does not support command execution or dynamic content natively in the JSON.

---

### Finding 2: Plugin system is the current "hook" mechanism
The `experimental.hook` field in `opencode.json` supports `file_edited` and `session_completed`, but `session_started` is not present in current SDK types.

**Evidence:**
- `experimental.hook` type definition:
  ```typescript
  hook?: {
    file_edited?: { [key: string]: Array<{ command: Array<string>, ... }> };
    session_completed?: Array<{ command: Array<string>, ... }>;
  };
  ```
- Current orchestration context is loaded via `SessionContextPlugin` listening for `session.created`.

**Source:** `@opencode-ai/sdk/dist/gen/types.gen.d.ts`, `~/.config/opencode/plugin/session-context.ts`

**Significance:** The plugin system has replaced or superseded older experimental hooks for session startup logic.

---

### Finding 3: Transparent context injection via `system.transform`
The SDK provides an `experimental.chat.system.transform` hook for plugins.

**Evidence:**
- SDK type definition:
  ```typescript
  "experimental.chat.system.transform"?: (input: {}, output: { system: string[]; }) => Promise<void>;
  ```

**Source:** `@opencode-ai/plugin/dist/index.d.ts`

**Significance:** This hook allows plugins to dynamically append to the system prompt array before it's sent to the LLM. This is more "native" and transparent than the current approach of injecting a visible user message with `noReply: true`.

---

## Synthesis

**Key Insights:**

1. **Static vs. Dynamic Split** - OpenCode's native configuration is designed for static assets (files, settings). Dynamic behavior (running commands, conditional injection) is intentionally delegated to the plugin system.
2. **Global Configuration exists** - Global instructions can be set in `~/.config/opencode/opencode.jsonc` without a plugin, but they remain static.
3. **Plugin Transparency Gap** - The current `session-context.ts` plugin uses a "hacky" injection method (sending a message) because it's easy to implement, but the SDK supports a cleaner `system.transform` hook that integrates directly with the system prompt.

**Answer to Investigation Question:**
OpenCode does **not** have native features for **dynamic** context injection in its JSON configuration. The `instructions` array is the native way to load static files globally or per-project. For orchestration patterns that require dynamic data (like `orch status`), the plugin system is the correct and only supported mechanism. The current plugin approach is valid but could be improved by using the `experimental.chat.system.transform` hook for better transparency.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
I have examined the SDK type definitions, the global configuration files, and the existing plugin implementations. The lack of dynamic fields in the JSON schema and the presence of specific plugin hooks for system prompt transformation strongly indicate the intended architecture.

**What's certain:**
- ✅ `instructions` array is for static file paths.
- ✅ `~` expansion is supported in paths.
- ✅ `experimental.hook` does not currently support `session_started` in the public SDK types.
- ✅ Plugins are the primary way to handle dynamic session startup logic.

**What's uncertain:**
- ⚠️ Whether `opencode.json` might support undocumented environment variable expansion in the future.
- ⚠️ The exact reason why `session_started` was removed from `experimental.hook` (likely in favor of the more powerful plugin system).

---

## Implementation Recommendations

### Recommended Approach ⭐
**Improve Plugin Transparency** - Refactor `session-context.ts` to use the `experimental.chat.system.transform` hook instead of `client.session.prompt`.

**Why this approach:**
- Context becomes part of the system prompt, not a visible message.
- No "bleeding" into the TUI or session history.
- More robust integration with the LLM's core instructions.

**Implementation sequence:**
1. Update `SessionContextPlugin` to return the `experimental.chat.system.transform` hook.
2. Move context building logic into this hook.
3. Append the built context to the `output.system` array.

---

## References

**Files Examined:**
- `~/.config/opencode/opencode.jsonc` - Global configuration
- `~/.config/opencode/plugin/session-context.ts` - Current plugin implementation
- `@opencode-ai/sdk/dist/gen/types.gen.d.ts` - SDK configuration types
- `@opencode-ai/plugin/dist/index.d.ts` - Plugin hook definitions

**Commands Run:**
- `opencode --help`
- `ls -R ~/.config/opencode`
- `grep -r "instructions" ~/.config/opencode`
