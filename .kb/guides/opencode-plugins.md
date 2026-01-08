# OpenCode Plugin System Guide

**Purpose:** Authoritative reference for building OpenCode plugins as the bridge between orchestration intelligence and execution reality. Read this before writing or debugging plugins.

**Last verified:** 2026-01-08

**Synthesized from:** Investigation `2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` + 8 production plugins in `~/.config/opencode/plugin/`

**Companion guide:** `.kb/guides/opencode.md` covers HTTP API integration; this covers the plugin system.

---

## Overview

OpenCode plugins are the mechanism for **mechanizing principles** - turning "remember to X" into "cannot proceed without X". They bridge the gap between knowing what should happen and ensuring it happens.

This guide covers:
- Three plugin patterns (Observation, Gates, Context Injection)
- Hook selection (which of 20+ hooks for which purpose)
- Worker vs orchestrator detection
- State management across sessions
- Testing approaches
- Common pitfalls (from production experience)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      OpenCode Server                             │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Plugin System                          │   │
│  │  ┌────────────────────────────────────────────────────┐  │   │
│  │  │  Plugin 1  │  Plugin 2  │  Plugin 3  │  ...        │  │   │
│  │  └────────────────────────────────────────────────────┘  │   │
│  │                         │                                 │   │
│  │                    Sequential                             │   │
│  │                    Execution                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                         │                                        │
│                         ▼                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     Hooks                                 │   │
│  │  tool.execute.before/after  │  session.created/idle     │   │
│  │  config                     │  event (catch-all)        │   │
│  │  experimental.session.compacting                        │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │         Agent Session          │
              │  (receives injected context,   │
              │   blocked commands, etc.)      │
              └───────────────────────────────┘
```

**Key insight:** Plugins run sequentially (not in parallel). Errors in `tool.execute.before` can block operations. Event handlers are fire-and-forget.

---

## Plugin Input

Every plugin receives these inputs:

```typescript
type PluginInput = {
  client: ReturnType<typeof createOpencodeClient>  // Full SDK client
  project: Project                                  // Current project info
  directory: string                                 // Working directory
  worktree: string                                  // Git worktree path
  serverUrl: URL                                    // OpenCode server URL
  $: BunShell                                       // Bun shell for commands
}
```

**The `client` provides:**
- `client.session.prompt()` - Inject context into sessions
- `client.session.list()` - List active sessions
- `client.app.log()` - Log to OpenCode's logging system
- `client.find.*`, `client.file.*` - File operations

**The `$` (Bun shell) provides:**
- Execute any shell command: `await $`orch usage --json`.quiet()`
- Access to `kb`, `kn`, `bd`, `orch` commands
- File I/O via shell commands

---

## The Three Plugin Patterns

### Pattern 1: Gates (Blocking)

**Purpose:** Prevent specific actions from completing.

**Hook:** `tool.execute.before`

**Mechanism:** Throw an error to block the operation.

**Example:** `bd-close-gate.ts` - Prevents workers from running `bd close`

```typescript
"tool.execute.before": async (input, output) => {
  if (input.tool !== "bash") return
  
  const command = output.args?.command as string
  if (shouldBlockCommand(command)) {
    throw new Error(getBlockedMessage())  // Blocks execution
  }
}
```

**When to use:**
- Hard rules that should never be violated
- Commands that bypass verification processes
- Operations that are dangerous in certain contexts

**Production examples:**
- Block `bd close` in worker context (orchestrator handles closure)
- Block `git push --force` to main branch
- Block editing auto-generated files directly

---

### Pattern 2: Context Injection (Guiding)

**Purpose:** Surface relevant information without blocking.

**Hook:** `tool.execute.before`, `event` (session.created), `event` (session.idle)

**Mechanism:** `client.session.prompt({ noReply: true, parts: [...] })`

**Example:** `guarded-files.ts` - Shows protocol when editing guarded files

```typescript
"tool.execute.before": async (input, output) => {
  if (input.tool !== "edit") return
  
  const filePath = output.args?.filePath as string
  const protocol = await getGuardedFileProtocol(filePath)
  
  if (protocol) {
    await client.session.prompt({
      path: { id: input.sessionID },
      body: {
        noReply: true,  // Don't wait for response
        parts: [{
          type: "text",
          text: `${protocol}\n\n---\n*File: ${filePath}*`
        }]
      }
    })
  }
}
```

**When to use:**
- Soft guidance (agent can proceed but should be aware)
- Surfacing protocols and modification rules
- Warning about constraints or usage limits
- Preserving context during compaction

**Production examples:**
- Usage warnings at session start (`usage-warning.ts`)
- Friction capture prompts on session idle (`friction-capture.ts`)
- Protocol injection for guarded files (`guarded-files.ts`)
- Context preservation during compaction (`session-compaction.ts`)

---

### Pattern 3: Observation (Learning)

**Purpose:** Track behavior patterns for analysis.

**Hook:** `tool.execute.before` + `tool.execute.after`, `event`

**Mechanism:** Log to file, update in-memory state, call external services.

**Example:** `action-log.ts` - Tracks tool outcomes for pattern detection

```typescript
const pendingArgs = new Map<string, any>()

"tool.execute.before": async (input, output) => {
  // Store args for correlation in after hook
  if (TRACKED_TOOLS.includes(input.tool)) {
    pendingArgs.set(input.callID, output.args)
  }
},

"tool.execute.after": async (input, output) => {
  const args = pendingArgs.get(input.callID)
  pendingArgs.delete(input.callID)
  
  const event: ActionEvent = {
    timestamp: new Date().toISOString(),
    tool: input.tool,
    target: extractTarget(input.tool, args),
    outcome: determineOutcome(input.tool, output.output),
    session_id: input.sessionID
  }
  
  logAction(event)  // Write to ~/.orch/action-log.jsonl
}
```

**When to use:**
- Pattern detection (repeated failures, thrashing)
- Metrics collection
- Building evidence for future gates
- Understanding agent behavior

**Production examples:**
- Action logging for futile exploration detection (`action-log.ts`)
- Session tracking for orchestrator management (`orchestrator-session.ts`)

---

## Hook Selection Guide

### Session Lifecycle Hooks

| Hook | When Fired | Data Available | Use Case |
|------|------------|----------------|----------|
| `session.created` | New session starts | `sessionID` | Context injection, session setup |
| `session.idle` | Session stops working | `sessionID` | Friction capture, cleanup |
| `session.updated` | Session state changes | `sessionID`, state | State monitoring |
| `session.error` | Error occurs | `sessionID`, error | Error handling |
| `session.compacted` | After compaction | `sessionID` | Post-compaction actions |
| `session.deleted` | Session removed | `sessionID` | Cleanup |

### Tool Execution Hooks

| Hook | When Fired | Data Available | Can Block? |
|------|------------|----------------|------------|
| `tool.execute.before` | Before tool runs | `tool`, `sessionID`, `callID`, `args` | Yes (throw) |
| `tool.execute.after` | After tool completes | `tool`, `sessionID`, `callID`, `title`, `output`, `metadata` | No |

**Critical:** `before` has `output.args`, `after` has `output.output`. Use `callID` to correlate.

### Configuration Hooks

| Hook | When Fired | Data Available | Use Case |
|------|------------|----------------|----------|
| `config` | At plugin load | Mutable `config` object | Add instructions, modify settings |

**Example:** Inject orchestrator skill into instructions

```typescript
config: async (config) => {
  if (!config.instructions) config.instructions = []
  config.instructions.push("~/.claude/skills/meta/orchestrator/SKILL.md")
}
```

### Experimental Hooks

| Hook | Status | Use Case |
|------|--------|----------|
| `experimental.session.compacting` | Experimental | Inject context to survive compaction |
| `experimental.chat.messages.transform` | Experimental | Modify message history |
| `experimental.chat.system.transform` | Experimental | Modify system prompt |

**Warning:** Experimental hooks may change or be removed.

---

## Worker vs Orchestrator Detection

Plugins often need different behavior for workers vs orchestrators:

```typescript
async function isWorker(directory: string): Promise<boolean> {
  // Method 1: Environment variable (most reliable)
  if (process.env.ORCH_WORKER === "1") {
    return true
  }
  
  // Method 2: SPAWN_CONTEXT.md exists (workers have this)
  const spawnContextPath = join(directory, "SPAWN_CONTEXT.md")
  if (await exists(spawnContextPath)) {
    return true
  }
  
  // Method 3: Path contains .orch/workspace/
  if (directory.includes(".orch/workspace/")) {
    return true
  }
  
  return false
}
```

**Detection priority:**
1. `ORCH_WORKER=1` env var (set by `orch spawn`)
2. `SPAWN_CONTEXT.md` in working directory
3. Path contains `.orch/workspace/`

**When to use:**
- `bd-close-gate.ts` - Only blocks workers (orchestrators can close issues)
- `orchestrator-session.ts` - Only injects skill for orchestrators (skips workers)
- `usage-warning.ts` - Applies to both (everyone needs to know limits)

---

## State Management

### In-Memory State (Session-Scoped)

Use Maps/Sets for state that should persist within a session:

```typescript
// Track files we've already warned about
const warnedFiles = new Set<string>()

// Track pending tool args for correlation
const pendingArgs = new Map<string, any>()
```

**Memory management:** Clear sets/maps when they get too large:

```typescript
if (warnedFiles.size > 500) {
  warnedFiles.clear()
}
```

### Cross-Instance Deduplication

Plugins may load multiple times (global + project). Use file-based locks:

```typescript
const DEDUP_LOCK_DIR = join(homedir(), ".orch", ".action-log-locks")

function isDuplicateEvent(hash: string): boolean {
  try {
    const lockPath = join(DEDUP_LOCK_DIR, hash)
    const fd = openSync(lockPath, "wx")  // Fails if exists
    closeSync(fd)
    return false
  } catch (err) {
    if (err.code === "EEXIST") return true
    return false
  }
}
```

### Persistent State

For state that should survive restarts, write to files:

```typescript
// ~/.orch/action-log.jsonl for JSONL format
appendFileSync(logPath, JSON.stringify(event) + "\n")

// Or use structured files
writeFileSync(statePath, JSON.stringify(state))
```

---

## Testing Approaches

### Manual Testing

1. **Enable debug logging:**
   ```bash
   export ORCH_PLUGIN_DEBUG=1
   ```

2. **Watch plugin output:**
   ```bash
   tail -f ~/.orch/action-log.jsonl
   ```

3. **Test specific hooks:**
   - Create a test session
   - Trigger the hook (e.g., edit a file, run a command)
   - Check for expected behavior

### Plugin Load Verification

```bash
# Check plugin loaded without errors
opencode --help 2>&1 | grep -i "plugin"

# Check for TypeScript errors
bun run ~/.config/opencode/plugin/your-plugin.ts
```

### Integration Testing

1. Start OpenCode with plugins
2. Run test scenarios via API or TUI
3. Verify expected:
   - Context injected
   - Commands blocked
   - Logs written

---

## Common Pitfalls

### 1. Duplicate Plugin Loading

**Problem:** Plugin code runs twice (global + project config).

**Symptoms:** Duplicate log entries, double context injection.

**Solution:** Use callID-based deduplication or file locks:

```typescript
const processedCalls = new Set<string>()

"tool.execute.after": async (input, output) => {
  if (processedCalls.has(input.callID)) return
  processedCalls.add(input.callID)
  // ... rest of handler
}
```

### 2. Blocking Async Operations

**Problem:** Slow async operations in hooks block tool execution.

**Symptoms:** Sluggish agent behavior, timeouts.

**Solution:** Fire-and-forget for non-critical operations:

```typescript
// Bad: Blocks execution
await client.session.prompt({ ... })

// Better: Non-blocking (for non-critical operations)
client.session.prompt({ ... }).catch(err => console.error(err))
```

### 3. Missing Args in After Hook

**Problem:** `tool.execute.after` doesn't have `output.args`.

**Symptoms:** Can't determine what file was edited, what command ran.

**Solution:** Store args from before hook, retrieve in after hook:

```typescript
const pendingArgs = new Map<string, any>()

"tool.execute.before": async (input, output) => {
  pendingArgs.set(input.callID, output.args)
},

"tool.execute.after": async (input, output) => {
  const args = pendingArgs.get(input.callID)
  pendingArgs.delete(input.callID)
  // Now you have both args and output
}
```

### 4. Event Handler Errors Swallowed

**Problem:** Errors in event handlers don't surface.

**Symptoms:** Plugin appears to not work, no error messages.

**Solution:** Explicit error logging:

```typescript
event: async ({ event }) => {
  try {
    // handler code
  } catch (err) {
    console.error("[plugin-name] Event handler error:", err)
  }
}
```

### 5. Wrong Hook for the Job

**Problem:** Using `tool.execute.after` when you need to block.

**Symptoms:** Action completes before you can stop it.

**Solution:** Use the right hook:
- Need to block? → `tool.execute.before` + `throw`
- Need to react after? → `tool.execute.after`
- Need to inject at start? → `event` (session.created)

### 6. Shared Library Confusion

**Problem:** OpenCode plugin loader tries to call lib functions as plugins.

**Symptoms:** Errors about missing hooks, unexpected behavior.

**Solution:** Put shared code in separate `lib/` directory:

```
~/.config/opencode/
├── plugin/
│   ├── my-plugin.ts      # Exports plugin function
│   └── my-other.ts
└── lib/
    ├── helpers.ts        # Shared helpers (not loaded as plugins)
    └── constants.ts
```

---

## Plugin File Structure

### Recommended Layout

```
~/.config/opencode/
├── plugin/
│   ├── action-log.ts           # Observation: Track tool outcomes
│   ├── bd-close-gate.ts        # Gate: Block bd close in workers
│   ├── friction-capture.ts     # Context: Prompt for friction on idle
│   ├── guarded-files.ts        # Context: Surface protocols on edit
│   ├── orchestrator-session.ts # Config + Context: Inject skill, start session
│   ├── session-compaction.ts   # Context: Preserve context during compaction
│   └── usage-warning.ts        # Context: Warn on high usage
└── lib/
    ├── bd-close-helpers.ts     # Shared: bd close detection logic
    ├── guarded-files.ts        # Shared: guarded file detection
    └── helpers.ts              # Shared: common utilities
```

### Plugin Template

```typescript
/**
 * Plugin: [Name]
 *
 * [What principle it mechanizes]
 *
 * Triggered by: [hook name]
 * Action: [what it does]
 */

import type { Plugin } from "@opencode-ai/plugin"

const LOG_PREFIX = "[plugin-name]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

export const PluginName: Plugin = async ({
  project,
  client,
  $,
  directory,
  worktree,
}) => {
  log("Plugin initialized")

  return {
    // Add hooks here
  }
}
```

---

## What Lives Where

| Thing | Location | Purpose |
|-------|----------|---------|
| Global plugins | `~/.config/opencode/plugin/*.ts` | User-wide behavior |
| Project plugins | `.opencode/plugin/*.ts` | Project-specific behavior |
| Shared helpers | `~/.config/opencode/lib/*.ts` | Avoid plugin loader confusion |
| Plugin SDK | `@opencode-ai/plugin` | TypeScript types and helpers |
| Action log | `~/.orch/action-log.jsonl` | Tool outcome history |
| Dedup locks | `~/.orch/.action-log-locks/` | Cross-instance deduplication |

---

## Quick Reference

### Pattern Selection

| Goal | Pattern | Hook | Can Block? |
|------|---------|------|------------|
| Block dangerous action | Gate | `tool.execute.before` | Yes |
| Surface information | Context Injection | Various | No |
| Track behavior | Observation | `tool.execute.*` | No |
| Modify config | Config | `config` | N/A |
| Preserve compaction context | Context Injection | `experimental.session.compacting` | No |

### Common Hook Recipes

**Block a command:**
```typescript
"tool.execute.before": async (input, output) => {
  if (input.tool === "bash" && output.args?.command?.includes("dangerous")) {
    throw new Error("Cannot run this command")
  }
}
```

**Inject context on session start:**
```typescript
event: async ({ event }) => {
  if (event.type !== "session.created") return
  await client.session.prompt({
    path: { id: event.properties.sessionID },
    body: { noReply: true, parts: [{ type: "text", text: "Context here" }] }
  })
}
```

**Log tool outcomes:**
```typescript
"tool.execute.after": async (input, output) => {
  appendFileSync(logPath, JSON.stringify({
    tool: input.tool,
    output: output.output?.slice(0, 100)
  }) + "\n")
}
```

---

## References

### Source Code

- **Plugin SDK:** `~/Documents/personal/opencode/packages/plugin/src/index.ts`
- **Plugin loader:** `~/Documents/personal/opencode/packages/opencode/src/plugin/index.ts`
- **Production plugins:** `~/.config/opencode/plugin/*.ts`

### Investigations

- `2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Comprehensive plugin system analysis
- `2026-01-08-observation-infrastructure-principle.md` - Principle behind observation plugins

### External

- **OpenCode plugin docs:** https://opencode.ai/docs/plugins
- **OpenCode SDK docs:** https://opencode.ai/docs/sdk
- **Community plugins:** https://opencode.ai/docs/ecosystem#plugins

---

## History

- **2026-01-08:** Created from synthesis of investigation + 8 production plugins
