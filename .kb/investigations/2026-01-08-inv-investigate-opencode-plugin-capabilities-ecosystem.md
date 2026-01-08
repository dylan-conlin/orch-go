<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode's plugin system is powerful and mature - plugins receive full SDK client access, Bun shell ($), and can hook into 20+ event types including tool execution, session lifecycle, and compaction.

**Evidence:** Source code analysis of opencode/packages/opencode/src/plugin/index.ts, opencode/packages/plugin/src/index.ts, and 7 existing plugins in ~/.config/opencode/plugin/. All hooks are async and run sequentially. Plugins can throw to block operations.

**Knowledge:** Three key capabilities: (1) tool.execute.before can block/modify tool calls by throwing, (2) session events enable context injection via client.session.prompt with noReply:true, (3) config hook enables dynamic instruction injection. Dylan already has 7 plugins demonstrating these patterns.

**Next:** Implement 2-3 high-value plugins for principle mechanization: "Coherence Over Patches" detector, "Gate Over Remind" enforcement via beads, and "Provenance" tracking for claims without evidence.

**Promote to Decision:** recommend-no - Tactical implementation of existing capability, not architectural choice.

---

# Investigation: OpenCode Plugin Capabilities and Ecosystem

**Question:** What are the full capabilities of OpenCode's plugin system? What data is available at each hook? What principles can be mechanized?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Worker agent (orch-go-n5h2g.1)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Plugin API provides rich context and full system access

**Evidence:** Plugin function receives:
- `client`: Full OpenCode SDK client (session.prompt, session.list, app.log, find.*, file.*, etc.)
- `$`: Bun shell API for executing commands
- `project`: Current project information
- `directory`: Working directory
- `worktree`: Git worktree path
- `serverUrl`: OpenCode server URL

**Source:** 
- `opencode/packages/plugin/src/index.ts:26-33` (PluginInput type)
- `opencode/packages/opencode/src/plugin/index.ts:24-31` (input construction)

**Significance:** Plugins have nearly unlimited capability - they can read/write files (via $), make HTTP requests (via client or fetch), access kb/kn (via $`kb ...`), and interact with the full OpenCode API. This means any principle that can be expressed as a check at a specific point in the workflow can be mechanized.

---

### Finding 2: Hook types span the full session lifecycle

**Evidence:** Available hooks from `Hooks` interface:

**Session Lifecycle:**
- `session.created`, `session.updated`, `session.deleted`
- `session.idle`, `session.status`, `session.error`
- `session.compacted`, `session.diff`
- `experimental.session.compacting` (pre-compaction)

**Tool Execution:**
- `tool.execute.before` - receives `{tool, sessionID, callID}` + `{args}` (can throw to block)
- `tool.execute.after` - receives same input + `{title, output, metadata}`

**Message/Chat:**
- `message.updated`, `message.removed`
- `message.part.updated`, `message.part.removed`
- `chat.message` (new message), `chat.params` (LLM params)
- `experimental.chat.messages.transform`, `experimental.chat.system.transform`

**Other:**
- `config` - modify config at load time
- `event` - catch-all for any event
- `permission.ask` - intercept permission requests
- `file.edited`, `file.watcher.updated`
- `command.executed`

**Source:**
- `opencode/packages/plugin/src/index.ts:146-216` (Hooks interface)
- `opencode/packages/web/src/content/docs/plugins.mdx:143-205` (documentation)

**Significance:** Every meaningful system transition has a hook. Session start is perfect for context injection. Tool execute hooks are perfect for gates. Session idle is perfect for friction capture prompts.

---

### Finding 3: Hooks run sequentially, errors can block operations

**Evidence:** From `Plugin.trigger`:
```typescript
for (const hook of await state().then((x) => x.hooks)) {
  const fn = hook[name]
  if (!fn) continue
  await fn(input, output)  // Sequential await
}
```

For events, plugins receive but cannot block:
```typescript
Bus.subscribeAll(async (input) => {
  for (const hook of hooks) {
    hook["event"]?.({ event: input })  // No await/try-catch visible
  }
})
```

**Source:**
- `opencode/packages/opencode/src/plugin/index.ts:68-83` (trigger function)
- `opencode/packages/opencode/src/plugin/index.ts:96-103` (event subscription)

**Significance:** 
- `tool.execute.before` can throw to block tool execution (used by bd-close-gate.ts and guarded-files.ts)
- Event handlers run without waiting - failures don't block the event
- Hooks run in plugin load order (global config → project config → global dir → project dir)

---

### Finding 4: Existing plugins demonstrate key patterns

**Evidence:** Dylan's 7 existing plugins cover major use cases:

| Plugin | Hook Used | Pattern |
|--------|-----------|---------|
| `action-log.ts` | `tool.execute.before/after` | Logging/observability |
| `guarded-files.ts` | `tool.execute.before` | Context injection before edit |
| `friction-capture.ts` | `event (session.idle)` | Prompt injection at lifecycle point |
| `orchestrator-session.ts` | `config` + `event` | Config modification + session start |
| `usage-warning.ts` | `event (session.created)` | Context injection at session start |
| `bd-close-gate.ts` | `tool.execute.before` | Command blocking (throws) |
| `agentlog-inject.ts` | `event (session.created)` | Context injection at session start |

Key patterns:
1. **Blocking gate**: `tool.execute.before` + `throw new Error(message)` - used for bd-close-gate
2. **Context injection**: `client.session.prompt({ noReply: true, parts: [...] })` - used for warnings/guides
3. **Config modification**: `config` hook + modify `config.instructions` array - used for skill injection
4. **Deduplication**: Map/Set + callID to prevent duplicate processing
5. **Worker detection**: Check `ORCH_WORKER` env, `SPAWN_CONTEXT.md`, or path for `.orch/workspace/`

**Source:**
- `~/.config/opencode/plugin/*.ts` (all 7 plugins)
- `~/.config/opencode/lib/*.ts` (shared helpers)

**Significance:** The patterns are proven. New plugins for principle mechanization can follow these templates directly.

---

### Finding 5: Community ecosystem shows additional patterns

**Evidence:** Notable community plugins:
- `opencode-dynamic-context-pruning` - Token optimization via message transform
- `opencode-shell-strategy` - Shell command interception/modification
- `opencode-supermemory` - Persistent memory via external service
- `opencode-skillful` - Lazy skill loading on demand
- `opencode-scheduler` - Scheduled jobs via launchd/systemd

**Source:** https://opencode.ai/docs/ecosystem#plugins

**Significance:** Community has solved similar problems. Worth reviewing `opencode-skillful` for lazy loading pattern (could reduce session start overhead).

---

### Finding 6: Tool data available at each phase

**Evidence:** For `tool.execute.before`:
```typescript
input: { tool: string, sessionID: string, callID: string }
output: { args: any }  // Can modify args before execution
```

For `tool.execute.after`:
```typescript
input: { tool: string, sessionID: string, callID: string }
output: { title: string, output: string, metadata: any }
```

Tool args by tool type:
- `read`: `{ filePath: string }`
- `edit`: `{ filePath: string, oldString: string, newString: string, replaceAll?: boolean }`
- `bash`: `{ command: string, workdir?: string, timeout?: number }`
- `glob`: `{ pattern: string, path?: string }`
- `grep`: `{ pattern: string, path?: string, include?: string }`
- `task`: `{ prompt: string, description: string, subagent_type: string }`

**Source:**
- `opencode/packages/plugin/src/index.ts:174-185` (hook types)
- `opencode/packages/opencode/src/session/prompt.ts:369-418` (trigger calls)
- Existing plugins demonstrate arg access

**Significance:** All tool args are accessible. A plugin could analyze patterns like:
- Sequential failed glob/grep searches (futile exploration)
- Edits to same file 3+ times (thrashing)
- Read of decision files before architectural changes (or lack thereof)

---

## Synthesis

**Key Insights:**

1. **Full system access enables any mechanization** - Plugins can run shell commands, access kb/kn, call the API, modify args, block operations, and inject context. Any principle that can be expressed as "when X happens, do Y" can be mechanized.

2. **Three primary mechanization patterns emerge**:
   - **Gates (blocking)**: `tool.execute.before` + `throw` - for hard rules like "workers can't bd close"
   - **Context injection (guiding)**: `client.session.prompt({ noReply: true })` - for soft guidance like "you're about to edit a guarded file"
   - **Observation (learning)**: Event handlers + logging - for pattern detection like "futile exploration"

3. **Worker/orchestrator distinction is solved** - Multiple signals: `ORCH_WORKER` env, `SPAWN_CONTEXT.md` existence, path containing `.orch/workspace/`. Plugins can apply different rules per context.

4. **Performance is acceptable but sequential** - Hooks run one-by-one. Heavy plugins could slow operations. Keep hooks fast; defer heavy work to async/background.

**Answer to Investigation Question:**

OpenCode's plugin system provides comprehensive capabilities for principle mechanization:

1. **Hook data availability**: Tool hooks provide full args (before) and results (after). Session events provide session ID for API calls. Config hook provides mutable config object.

2. **Plugin API beyond hooks**: Full SDK client, Bun shell ($), project/worktree paths. Plugins can do anything: file I/O, HTTP, kb/kn commands, API calls.

3. **Performance characteristics**: Sequential hook execution, async/await throughout. Throwing blocks operations. No timeout protection - plugins must be well-behaved.

4. **Ecosystem patterns**: Established patterns for blocking gates, context injection, and observability. Community plugins show more advanced patterns.

5. **Current usage**: Dylan's 7 plugins already demonstrate all key patterns. New plugins can follow templates.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin trigger mechanism is sequential (verified: read source code at plugin/index.ts:68-83)
- ✅ tool.execute.before can block via throw (verified: bd-close-gate.ts uses this pattern successfully)
- ✅ client.session.prompt with noReply:true injects context (verified: multiple existing plugins use this)
- ✅ Config hook can modify instructions array (verified: orchestrator-session.ts does this)

**What's untested:**

- ⚠️ Performance impact of multiple hooks on each tool call (not benchmarked)
- ⚠️ Error handling when event handlers fail (source suggests fire-and-forget)
- ⚠️ Concurrency behavior when multiple sessions exist (not tested)

**What would change this:**

- Finding would be wrong if hooks had hidden rate limiting or timeouts
- Finding would be wrong if certain operations couldn't be blocked (need to verify each tool type)
- Finding would be wrong if config hook ran too late to affect session

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable plugin implementations for principle mechanization.

### Recommended Approach ⭐

**Start with high-value observation plugins** - Rather than immediately building gates, first build observation plugins to understand agent behavior patterns, then add gates for proven problem areas.

**Why this approach:**
- Observation plugins are low-risk (don't block anything)
- Generate data to inform which gates are actually needed
- Follow the "Friction is Signal" principle - let problems surface naturally

**Trade-offs accepted:**
- Slower to mechanize principles (observation first, then action)
- Requires analysis of observation data before adding gates

**Implementation sequence:**
1. **Coherence detector** - Observe edit patterns to files, flag when same file edited 3+ times
2. **Provenance tracker** - Log when agents make claims, track if evidence was gathered
3. **Gate Over Remind enforcer** - After validation, convert high-value reminders to gates

### Plugin Designs

#### Plugin 1: Coherence Detector (Observation)

**Principle mechanized:** "Coherence Over Patches"

**Hook:** `tool.execute.after` (edit tool)

**Logic:**
```typescript
// Track edits per file per session
const editCounts = new Map<string, number>()

"tool.execute.after": async (input, output) => {
  if (input.tool !== "edit") return
  
  const key = `${input.sessionID}:${output.args?.filePath}`
  const count = (editCounts.get(key) || 0) + 1
  editCounts.set(key, count)
  
  if (count >= 3) {
    // Log to action-log for pattern analysis
    logAction({
      timestamp: new Date().toISOString(),
      tool: "CoherenceAlert",
      target: output.args?.filePath,
      outcome: "warning",
      context: `File edited ${count} times in session. Consider: Is this patching or coherent change?`
    })
    
    // Optionally inject context
    await client.session.prompt({
      path: { id: input.sessionID },
      body: {
        noReply: true,
        parts: [{
          type: "text",
          text: `⚠️ **Coherence Check**: This file has been edited ${count} times. 
The "Coherence Over Patches" principle suggests stopping to consider:
- Is there a deeper issue being patched around?
- Would an architectural change be more appropriate?
- Should this be escalated to the orchestrator?`
        }]
      }
    })
  }
}
```

**Value:** Surfaces thrashing behavior. Data informs when to escalate.

---

#### Plugin 2: Provenance Tracker (Observation)

**Principle mechanized:** "Provenance" - conclusions should trace to evidence outside the conversation

**Hook:** `tool.execute.after` (read, grep, bash) + message analysis

**Logic:**
```typescript
// Track evidence-gathering per session
const evidenceGathered = new Map<string, Set<string>>()

"tool.execute.after": async (input, output) => {
  const evidenceTools = ["read", "grep", "glob", "bash"]
  if (!evidenceTools.includes(input.tool)) return
  
  // Track that evidence was gathered
  const evidence = evidenceGathered.get(input.sessionID) || new Set()
  evidence.add(`${input.tool}:${output.args?.filePath || output.args?.pattern || output.args?.command}`)
  evidenceGathered.set(input.sessionID, evidence)
}

// On session idle, check if conclusions exist without evidence
"event": async ({ event }) => {
  if (event.type !== "session.idle") return
  
  const sessionId = event.properties?.sessionID
  const evidence = evidenceGathered.get(sessionId)
  
  if (!evidence || evidence.size === 0) {
    // Session made conclusions without gathering evidence
    await client.app.log({
      service: "provenance-tracker",
      level: "warn",
      message: "Session concluded without evidence gathering",
      extra: { sessionId }
    })
  }
}
```

**Value:** Surfaces sessions that conclude without evidence. Informs when to add provenance gates.

---

#### Plugin 3: Task Spawn Analyzer (Observation)

**Principle mechanized:** "Perspective is Structural" + "Escalation is Information Flow"

**Hook:** `tool.execute.before/after` (task tool)

**Logic:**
```typescript
// Track task spawns - looking for patterns like:
// - Multiple spawns without synthesis
// - Same task spawned repeatedly (retry loop)
// - Spawns without prior investigation

const taskHistory = new Map<string, Array<{time: number, description: string, type: string}>>()

"tool.execute.before": async (input, output) => {
  if (input.tool !== "task") return
  
  const history = taskHistory.get(input.sessionID) || []
  history.push({
    time: Date.now(),
    description: output.args?.description || "",
    type: output.args?.subagent_type || ""
  })
  taskHistory.set(input.sessionID, history)
  
  // Check for patterns
  const recent = history.slice(-5)
  
  // Same type spawned 3+ times in quick succession
  const sameType = recent.filter(t => t.type === output.args?.subagent_type)
  if (sameType.length >= 3 && (Date.now() - sameType[0].time) < 30 * 60 * 1000) {
    await client.session.prompt({
      path: { id: input.sessionID },
      body: {
        noReply: true,
        parts: [{
          type: "text",
          text: `⚠️ **Spawn Pattern Alert**: You've spawned ${sameType.length} ${output.args?.subagent_type} tasks in the last 30 minutes.

Consider:
- Is this a retry loop that needs a different approach?
- Should you synthesize results from prior spawns first?
- Is there a perspective gap that needs escalation?`
        }]
      }
    })
  }
}
```

**Value:** Detects retry loops and spawn-without-synthesis patterns.

---

### Alternative Approaches Considered

**Option B: Immediate hard gates**
- **Pros:** Instantly enforce principles, no ambiguity
- **Cons:** May block legitimate use cases, doesn't generate learning data, could be frustrating
- **When to use instead:** For known-bad patterns like "workers running bd close"

**Option C: Pure logging without context injection**
- **Pros:** Completely non-intrusive, only observation
- **Cons:** Agents don't get real-time feedback, may not learn from patterns
- **When to use instead:** For metrics/analytics purposes only

**Rationale for recommendation:** Option A (observation + soft guidance) balances learning with immediate value. Agents get feedback without being blocked. Data informs future gates.

---

### Implementation Details

**What to implement first:**
1. Coherence detector - Highest signal-to-noise ratio
2. Action-log improvements - Better pattern detection in existing plugin
3. Provenance tracker - Validates a key epistemic principle

**Things to watch out for:**
- ⚠️ Deduplication - Plugin may load multiple times (use callID/hash dedup like action-log.ts)
- ⚠️ Worker vs orchestrator - Some plugins should only run for workers (check ORCH_WORKER env)
- ⚠️ Session cleanup - Clear Maps/Sets when sessions end to prevent memory leaks
- ⚠️ Async operations - Don't block hooks with slow operations; log and continue

**Areas needing further investigation:**
- Error handling behavior when plugin throws in event handler
- Performance impact of multiple session.prompt calls
- Whether compaction preserves injected context

**Success criteria:**
- ✅ Plugins load without errors
- ✅ Patterns are logged to observable location (action-log.jsonl or separate file)
- ✅ Context injection appears in session without disrupting flow
- ✅ `orch patterns` or similar can surface detected patterns

---

## References

**Files Examined:**
- `opencode/packages/opencode/src/plugin/index.ts` - Core plugin loading and trigger mechanism
- `opencode/packages/plugin/src/index.ts` - Plugin types and Hooks interface
- `opencode/packages/opencode/src/hook/index.ts` - Built-in hooks (file_edited, session_started)
- `opencode/packages/web/src/content/docs/plugins.mdx` - Official plugin documentation
- `~/.config/opencode/plugin/*.ts` - Dylan's 7 existing plugins
- `~/.config/opencode/lib/*.ts` - Shared plugin helpers

**Commands Run:**
```bash
# Search for plugin-related code
glob **/*plugin* /Users/dylanconlin/Documents/personal/opencode

# Find hook trigger points
grep "Plugin\.trigger" /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src

# List existing plugins
glob plugin/* ~/.config/opencode
```

**External Documentation:**
- https://opencode.ai/docs/plugins - Official plugin documentation
- https://opencode.ai/docs/sdk - SDK client documentation
- https://opencode.ai/docs/ecosystem - Community plugins

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-08-observation-infrastructure-principle.md` - Related principle about observability
- **Investigation:** Parent epic `orch-go-n5h2g` - Mechanizing principles with OpenCode plugins

---

## Investigation History

**2026-01-08 10:00:** Investigation started
- Initial question: What are OpenCode plugin capabilities? What principles can be mechanized?
- Context: Part of epic to reduce agent performance gaps through automated principle enforcement

**2026-01-08 10:30:** Core findings complete
- Discovered hook system architecture and data flow
- Mapped all available hook types and their data
- Analyzed existing plugin patterns

**2026-01-08 11:00:** Investigation completed
- Status: Complete
- Key outcome: Plugin system is mature and powerful; three high-value plugins designed for principle mechanization
