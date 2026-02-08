---
date: 2026-01-29
status: Complete
tags: [opencode, tui, agents, mode]
---

# OpenCode TUI Mode Switch Investigation

**Question:** Dylan reports OpenCode TUI shows a mode indicator that cycles via Tab: build → orchestrator → plan. What does this actually do?

**Status:** Complete

## Summary

The "mode" shown in OpenCode TUI is **not a separate mode system** - it's simply displaying which **agent** was used for that message. The Tab key cycles between available agents (build, plan, and any custom agents), and each agent has different permissions and prompts.

**Key Finding:** The term "mode" is a **deprecated field** in the AssistantMessage schema that stores the agent name for backward compatibility. The actual functionality is agent selection, not mode switching.

## Evidence

### 1. Mode Field is Deprecated Agent Name

**Source:** `packages/opencode/src/session/message-v2.ts:375-378`
```typescript
/**
 * @deprecated
 */
mode: z.string(),
```

**Source:** `packages/opencode/src/session/prompt.ts:530`
```typescript
mode: agent.name,  // mode is set to agent name
```

**Significance:** The "mode" field exists only for backward compatibility - it's the same as the agent name.

### 2. Tab Key Cycles Agents (Not Modes)

**Source:** `packages/opencode/src/config/config.ts:703-704`
```typescript
agent_cycle: z.string().optional().default("tab").describe("Next agent"),
agent_cycle_reverse: z.string().optional().default("shift+tab").describe("Previous agent"),
```

**Source:** `packages/opencode/src/cli/cmd/tui/app.tsx:398-404`
```typescript
{
  title: "Agent cycle",
  value: "agent.cycle",
  keybind: "agent_cycle",
  category: "Agent",
  hidden: true,
  onSelect: () => {
    local.agent.move(1)
  },
},
```

**Source:** `packages/opencode/src/cli/cmd/tui/context/local.tsx:68-76`
```typescript
move(direction: 1 | -1) {
  batch(() => {
    let next = agents().findIndex((x) => x.name === agentStore.current) + direction
    if (next < 0) next = agents().length - 1
    if (next >= agents().length) next = 0
    const value = agents()[next]
    setAgentStore("current", value.name)
  })
}
```

**Significance:** Tab key increments through the agents array (wrapping around). It's standard array cycling, not mode switching.

### 3. Available Agents (What Tab Cycles Through)

**Source:** `packages/opencode/src/cli/cmd/tui/context/local.tsx:37`
```typescript
const agents = createMemo(() => sync.data.agent.filter((x) => x.mode !== "subagent" && !x.hidden))
```

**Source:** `packages/opencode/src/agent/agent.ts:74-111`

Built-in agents:
- **build** (mode: "primary") - Default agent, executes tools with standard permissions
- **plan** (mode: "primary") - Disallows edit tools except for plan files
- **general** (mode: "subagent") - Not shown in Tab cycle
- **explore** (mode: "subagent") - Not shown in Tab cycle
- **compaction/title/summary** (mode: "primary", hidden: true) - Not shown in Tab cycle

**Significance:** Tab cycles only through primary visible agents. By default, that's `build` and `plan`. There is no "orchestrator" agent in the default agent list - Dylan may have configured a custom agent.

### 4. TUI Display of "Mode"

**Source:** `packages/opencode/src/cli/cmd/tui/routes/session/index.tsx:1280`
```typescript
<span style={{ fg: theme.text }}>{Locale.titlecase(props.message.mode)}</span>
```

**Source:** `packages/opencode/src/cli/cmd/tui/component/tips.tsx:54`
```
"Press {highlight}Tab{/highlight} to cycle between Build and Plan agents",
```

**Significance:** The TUI displays `message.mode` (which is `agent.name`) in the message header. The tips explicitly say "cycle between agents", not "modes".

### 5. Agent Differences (What Actually Changes)

**Source:** `packages/opencode/src/agent/agent.ts:74-111`

Each agent has:
- **Different permissions** - `build` allows most tools, `plan` denies edit tools
- **Optional custom prompt** - Can override system prompt
- **Optional model override** - Can force specific model
- **Optional temperature/topP** - Can tune generation parameters

**Example - Plan agent restrictions:**
```typescript
plan: {
  permission: PermissionNext.merge(
    defaults,
    PermissionNext.fromConfig({
      question: "allow",
      plan_exit: "allow",
      external_directory: {
        [path.join(Global.Path.data, "plans", "*")]: "allow",
      },
      edit: {
        "*": "deny",
        [path.join(".opencode", "plans", "*.md")]: "allow",
      },
    }),
  ),
}
```

**Significance:** Switching agents changes:
- Which tools are available (via permissions)
- Optionally: system prompt, model, temperature
- Does NOT change: base tool set, UI affordances, API routing

### 6. Experimental Plan Mode Flag

**Source:** `packages/opencode/src/session/prompt.ts:1204-1227`

When `Flag.OPENCODE_EXPERIMENTAL_PLAN_MODE` is enabled, switching between `plan` and `build` agents injects synthetic prompts:
- Entering plan mode: "You should begin planning. The plan file will be at {path}."
- Exiting plan mode: "User confirmed to switch to build mode. A plan file exists at {path}. You should execute on the plan."

**Significance:** This is an experimental feature that adds contextual prompts when switching, but still doesn't change the underlying tool availability or API behavior.

## Does Mode Change System Behavior?

### System Prompt / Instructions
**Answer:** Optionally, yes - but only if the agent has a custom `prompt` field configured.
- Default `build` agent: No custom prompt
- `plan` agent: No custom prompt (uses experimental flag injection instead)
- Custom agents: Can define custom prompts

### Tool Availability
**Answer:** Yes - via permissions.
- Each agent has a `permission` ruleset
- Example: `plan` agent denies all `edit` tools except for plan files
- Example: `general` subagent denies `todoread` and `todowrite`

### Temperature / Model Selection
**Answer:** Optionally, yes - if configured per agent.
- Agents can override model via `agent.model` field
- Agents can override temperature/topP
- By default, most agents inherit from user's selected model

### Message Routing
**Answer:** No.
- All agents use the same message sending API
- The `mode` field is just metadata stored on the assistant message

## Is Mode Persisted / Observable?

### Session Metadata
**Answer:** Yes - each assistant message stores the agent name in the `mode` field.

**Source:** `packages/sdk/js/src/v2/gen/types.gen.ts:170-182`
```typescript
export type AssistantMessage = {
  id: string
  sessionID: string
  role: "assistant"
  // ... other fields
  modelID: string
  providerID: string
  mode: string  // stores agent name
  agent: string
  // ...
}
```

### Disk Storage
**Answer:** Yes - messages are persisted to disk with the `mode` field.

### SSE Events
**Answer:** Yes - message events include the full message object with `mode` field.

### Can orch-go observe it via API?
**Answer:** Yes - via OpenCode's ListSessions or SSE message events.

**Tested:** No (investigation scope was code inspection only)

## Dylan's "orchestrator" Agent

Dylan reported seeing "build → orchestrator → plan" in the cycle. This suggests Dylan has a custom agent named "orchestrator" configured in his OpenCode config.

**To verify:** Check Dylan's OpenCode config for custom agents:
- Global: `~/Library/Application Support/opencode/config.json` (doesn't exist)
- Project: `~/Documents/personal/opencode/.opencode/opencode.jsonc` (checked - no custom agents)
- Possibly loaded via plugin or enterprise config

The default agents only include `build` and `plan` as primary visible agents, so "orchestrator" would be custom.

## Recommendations

### For orch-go Integration

If orch-go wants to display which agent was used for a message:
1. **Read from message.agent field** (preferred) or `message.mode` (deprecated)
2. **Both fields contain the same value** (agent name)
3. **No special handling needed** - it's just a string identifier

If orch-go wants to spawn with a specific agent:
1. **Use the `/agent` command in the prompt** - e.g., "/agent plan"
2. **Or set default_agent in config** - requires config file modification
3. **Agent selection happens before message send** - not via API parameter

### For Clarifying UI/UX

The OpenCode TUI uses "mode" terminology in the deprecated field, but the feature is actually **agent selection**. This is misleading.

**Suggested terminology:**
- "mode" → "agent"
- "mode switch" → "agent cycle"
- "build mode" → "build agent"

## Follow-up Questions

1. **Does Dylan have a custom "orchestrator" agent?** - Needs config inspection or asking Dylan directly
2. **Should orch-go display agent names in the dashboard?** - Depends on whether it provides value to users
3. **Should orch-go allow selecting agent at spawn time?** - Would require injecting `/agent X` into the first user message

## Next Steps

**Status:** Investigation complete. No follow-up beads issues needed - this was discovery-only.

If integration is desired, create a separate issue:
- `bd create "Add agent display to orch-go dashboard" --type feature`
- `bd create "Support agent selection in orch spawn" --type feature`
