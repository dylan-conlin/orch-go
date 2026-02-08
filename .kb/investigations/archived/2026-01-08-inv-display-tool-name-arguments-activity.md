<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard activity feed now displays tool calls like Claude Code TUI: `Bash(git status)` instead of raw `tool` event type.

**Evidence:** Visual verification via screenshot showing `Bash(curl -sk...)`, `Read(/path/to/file...)`, `Glass_screenshot` in activity feed with blue tool names and truncated args.

**Knowledge:** Real-time SSE events contain full tool state (tool name, input args, output) while historical API messages only have basic part data - tool display works for real-time events.

**Next:** Close - feature complete; historical tool names would require OpenCode API enhancement (out of scope).

**Promote to Decision:** recommend-no (implementation detail, not architectural)

---

# Investigation: Display Tool Name Arguments Activity

**Question:** How to display tool name + arguments in activity feed like Claude Code TUI does?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SSE Event Structure Contains Full Tool Data

**Evidence:** Real-time SSE events from `/api/events` include:
```json
{
  "type": "message.part.updated",
  "properties": {
    "part": {
      "type": "tool",
      "tool": "bash",
      "state": {
        "status": "running|completed",
        "input": { "command": "git status", "description": "..." },
        "output": "..."
      }
    }
  }
}
```

**Source:** `curl -sk 'https://localhost:3348/api/events' --max-time 5` - observed live tool calls

**Significance:** All data needed for rich tool display (name, args, status) is available in real-time events.

---

### Finding 2: Historical Messages API Lacks Tool Details

**Evidence:** The OpenCode `MessagePart` struct (`pkg/opencode/types.go:118-125`) only has:
```go
type MessagePart struct {
    ID        string
    SessionID string
    MessageID string
    Type      string // "tool-invocation"
    Text      string
}
```
No `tool` or `state.input` fields - the historical `/api/session/{id}/messages` endpoint cannot provide tool args.

**Source:** `pkg/opencode/types.go:118-125`, `cmd/orch/serve_agents.go:1357-1372`

**Significance:** Tool names/args only work for real-time SSE events, not historical data. This is an OpenCode API limitation.

---

### Finding 3: Frontend Type Definitions Already Support Tool Data

**Evidence:** `web/src/lib/stores/agents.ts:111-136` already defines:
```typescript
interface SSEEvent {
  properties?: {
    part?: {
      type: string;
      tool?: string;
      state?: {
        title?: string;
        input?: unknown;
      };
    };
  };
}
```

**Source:** `web/src/lib/stores/agents.ts:111-136`

**Significance:** No type changes needed - just rendering logic in activity-tab.svelte.

---

## Synthesis

**Key Insights:**

1. **Data availability asymmetry** - Real-time SSE has rich tool data, historical API has minimal part data. This is fundamental to OpenCode's architecture.

2. **Tool-specific arg extraction** - Different tools have different input structures (bash→command, read→filePath, glob→pattern). The helper function handles common patterns.

3. **666px width constraint respected** - Truncation at 60 chars + ellipsis keeps tool calls readable at dashboard minimum width.

**Answer to Investigation Question:**

Display tool names + args by parsing `part.tool` and `part.state.input` from SSE events. Tool name is capitalized and shown in blue monospace. Args are extracted based on tool type (command, filePath, pattern, etc.) and truncated to 60 chars with full text in title tooltip.

---

## Structured Uncertainty

**What's tested:**

- ✅ Real-time tool events display as `ToolName(args)` (verified: screenshot shows Bash, Read, Glass_scroll with args)
- ✅ Truncation works for long args (verified: `curl -sk...` shows truncated with `…`)
- ✅ Tool name shown in blue color (verified: screenshot)

**What's untested:**

- ⚠️ Historical tool events display (known limitation - data not available from API)
- ⚠️ All tool types handled (covered common ones: bash, read, write, glob, grep, webfetch, task)

**What would change this:**

- If OpenCode enhanced `/session/{id}/message` API to include tool details, historical events could also show tool names/args

---

## Implementation Recommendations

### Recommended Approach ⭐

**Helper functions for tool display** - Added to activity-tab.svelte

**Why this approach:**
- Self-contained in the component where it's used
- No changes to data layer needed
- Type-safe with extracted Part type

**Implementation sequence:**
1. `formatToolName(tool)` - Capitalizes first letter
2. `extractToolArg(input)` - Gets most relevant arg for each tool type
3. `truncate(text, maxLen)` - Truncates with ellipsis
4. `formatToolCall(part)` - Combines above for display

---

## References

**Files Examined:**
- `web/src/lib/stores/agents.ts` - SSEEvent type definition
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Rendering logic
- `cmd/orch/serve_agents.go` - Historical messages endpoint
- `pkg/opencode/types.go` - OpenCode MessagePart struct

**Commands Run:**
```bash
# Check live SSE event structure
curl -sk 'https://localhost:3348/api/events' --max-time 5

# Build and restart servers
make install && orch servers restart orch-go
```

---

## Investigation History

**2026-01-08 20:15:** Investigation started
- Initial question: How to display tool name + args like Claude Code TUI
- Context: Dashboard activity feed shows raw event types

**2026-01-08 20:25:** Found SSE event structure with tool data
- Real-time events have tool, state.input with command/filePath/pattern

**2026-01-08 20:30:** Identified historical API limitation
- OpenCode MessagePart struct lacks tool details
- Only real-time events can show tool args

**2026-01-08 20:35:** Investigation completed
- Status: Complete
- Key outcome: Implemented tool display for real-time events; historical data limitation documented
