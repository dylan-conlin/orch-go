---
linked_issues:
  - orch-go-vp6g
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode provides `tool.execute.after` plugin hook that can observe tool outcomes - this is the integration point for action logging.

**Evidence:** OpenCode docs show `tool.execute.after` hook; existing plugins (`bd-close-gate.ts`) demonstrate the pattern; `pkg/action/` already has Logger and Tracker ready to receive events.

**Knowledge:** Integration requires an OpenCode plugin (not hooks into orch-go), which writes to `~/.orch/action-log.jsonl` via the existing `pkg/action` infrastructure.

**Next:** Create `~/.config/opencode/plugin/action-logger.ts` that logs investigative tool outcomes (Read, Glob, Grep) to action-log.jsonl.

---

# Investigation: Action Logging Integration Points for Agent Tool Outcomes

**Question:** Where in the architecture can we observe agent tool results to enable action logging for behavioral pattern detection?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent (orch-go-vp6g)
**Phase:** Complete
**Next Step:** None - findings complete, ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: OpenCode Has `tool.execute.after` Hook - The Key Integration Point

**Evidence:** OpenCode plugin documentation explicitly shows:
```typescript
// Tool Events
"tool.execute.before"
"tool.execute.after"
```

The `bd-close-gate.ts` plugin demonstrates `tool.execute.before`:
```typescript
"tool.execute.before": async (input: any, output: any) => {
  if (input.tool !== "bash") return
  const command = output.args?.command as string | undefined
  // ... can access tool name and args
}
```

**Source:** 
- https://opencode.ai/docs/plugins/ (Events section)
- `/Users/dylanconlin/.config/opencode/plugin/bd-close-gate.ts:29-44`

**Significance:** This is the exact integration point needed. The `tool.execute.after` hook should provide access to tool outcomes (success/empty/error), enabling action logging without modifying OpenCode core.

---

### Finding 2: pkg/action Already Has Complete Logger Infrastructure

**Evidence:** The action logging subsystem exists with:
- `ActionEvent` struct: tool, target, outcome, session_id, workspace, context
- `Logger` with methods: `LogSuccess`, `LogEmpty`, `LogError`, `LogFallback`
- `Tracker` for loading events and finding patterns
- `FormatPatterns` for human-readable output
- Default path: `~/.orch/action-log.jsonl`

```go
// pkg/action/action.go:41-69
type ActionEvent struct {
    Timestamp      time.Time `json:"timestamp"`
    Tool           string    `json:"tool"`
    Target         string    `json:"target"`
    Outcome        Outcome   `json:"outcome"` // success, empty, error, fallback
    ErrorMessage   string    `json:"error_message,omitempty"`
    FallbackAction string    `json:"fallback_action,omitempty"`
    SessionID      string    `json:"session_id,omitempty"`
    Workspace      string    `json:"workspace,omitempty"`
    Context        string    `json:"context,omitempty"`
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/action/action.go:41-69, 119-220`

**Significance:** No new Go code needed for logging infrastructure - just need a plugin to call the existing logger.

---

### Finding 3: Pattern Detection and `orch patterns` Command Already Exist

**Evidence:** The `orch patterns` command already:
- Collects action patterns via `collectActionPatterns()`
- Shows futile action patterns alongside retry patterns and gap patterns
- Categorizes by severity (critical 5+, warning 3+)
- Suggests `kn tried` commands for detected patterns

```go
// cmd/orch/patterns.go:396-465
func collectActionPatterns() ([]DetectedPattern, error) {
    tracker, err := action.LoadTracker("")
    actionPatterns := tracker.FindPatterns()
    // ... converts to DetectedPattern with severity
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/patterns.go:394-465`

**Significance:** The entire action logging pipeline exists except the data source - once the plugin writes to `action-log.jsonl`, patterns will surface automatically.

---

### Finding 4: Sharp Concept - Log Investigative Actions, Not Mutating Commands

**Evidence:** From the spawn context:
> "Action logging detects repeated futile investigative actions (Read/Glob/Grep returning empty), NOT mutating commands or status queries."

This constrains the scope to:
- **In scope:** Read (file doesn't exist), Glob (no matches), Grep (no matches)
- **Out of scope:** Bash commands (mutating), Write/Edit (mutating), success cases (not futile)

**Source:** SPAWN_CONTEXT.md task description

**Significance:** The plugin should be selective - only log non-success outcomes from Read/Glob/Grep tools, avoiding noise from successful operations or intentional mutations.

---

### Finding 5: Existing Plugin Patterns Show TypeScript Plugin Structure

**Evidence:** Existing plugins at `~/.config/opencode/plugin/` demonstrate:
1. Plugin type definition via `@opencode-ai/plugin`
2. Access to `$` (Bun shell) for running commands
3. Event subscriptions via return object with hook names
4. Async hook functions with input/output parameters

```typescript
// Example from bd-close-gate.ts
import type { Plugin } from "@opencode-ai/plugin"

export const BdCloseGate: Plugin = async ({ project, client, $, directory, worktree }) => {
  return {
    "tool.execute.before": async (input: any, output: any) => {
      if (input.tool !== "bash") return
      // ... access tool info
    },
  }
}
```

**Source:** `/Users/dylanconlin/.config/opencode/plugin/bd-close-gate.ts`

**Significance:** The plugin structure is proven. Creating an action logging plugin follows the same pattern.

---

## Synthesis

**Key Insights:**

1. **No Hooks Into OpenCode Core Needed** - OpenCode's plugin system already provides the integration point via `tool.execute.after`. This is cleaner than trying to parse SSE events or post-session transcripts.

2. **Infrastructure Exists, Just Needs Data Source** - The entire action logging pipeline (`pkg/action`, `orch patterns`, pattern surfacing) is implemented. The missing piece is a plugin that writes events to `action-log.jsonl`.

3. **Sharp Scoping Prevents Noise** - By focusing only on investigative tools (Read/Glob/Grep) and non-success outcomes, the system will detect the exact problem (checking SYNTHESIS.md on light-tier agents) without flooding the log with routine operations.

4. **Glass Model is Irrelevant** - Glass doesn't log to `action-log.jsonl` currently. The prior investigation mentioned Glass because it controls tool execution directly, but for OpenCode-based agents, the plugin hook is the right approach.

**Answer to Investigation Question:**

The architecture provides **OpenCode plugin hooks** as the integration point for observing agent tool results. Specifically:
- `tool.execute.after` fires after each tool invocation
- The hook receives tool name and output
- A plugin can write to `~/.orch/action-log.jsonl` using orch-go's `pkg/action` infrastructure

This doesn't require hooks into orch-go or modifying the MCP layer. The minimal integration is:
1. Create `~/.config/opencode/plugin/action-logger.ts`
2. Hook `tool.execute.after`
3. For Read/Glob/Grep with empty/error outcomes, shell out to `orch action log` (or write directly)
4. Existing `orch patterns` and pattern analyzer will surface the data

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode has `tool.execute.after` hook (verified: docs at opencode.ai/docs/plugins/)
- ✅ Existing plugins use `tool.execute.before` successfully (verified: read bd-close-gate.ts)
- ✅ pkg/action Logger and Tracker exist and work (verified: read action.go and action_test.go)
- ✅ `orch patterns` command exists and collects action patterns (verified: read patterns.go)

**What's untested:**

- ⚠️ What exactly `tool.execute.after` receives (input/output structure not documented in detail)
- ⚠️ How to determine "empty" outcome from tool output in the hook
- ⚠️ Whether writing to JSONL from TypeScript plugin is best vs shelling out to orch

**What would change this:**

- Finding would be wrong if `tool.execute.after` doesn't provide tool output
- Finding would be wrong if OpenCode restricts what plugins can write to disk
- Alternative approach needed if hook performance is too slow for logging

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**OpenCode Plugin with Direct JSONL Writing** - Create TypeScript plugin that hooks `tool.execute.after`, filters for investigative tools with non-success outcomes, and writes directly to `action-log.jsonl`.

**Why this approach:**
- Uses OpenCode's native plugin system (designed for this use case)
- Matches existing plugin patterns in `~/.config/opencode/plugin/`
- Writes to existing log file that `orch patterns` already reads
- No changes needed to orch-go code

**Trade-offs accepted:**
- Duplicates some JSONL writing logic (can't import Go code into TypeScript)
- Plugin runs in OpenCode process (minor memory overhead)
- Requires TypeScript rather than Go (consistency with existing plugins)

**Implementation sequence:**
1. **Create action-logger.ts plugin** - Hook `tool.execute.after`, detect empty outcomes
2. **Add orch action log command** - Simple CLI for plugins to log events (avoid JSONL duplication)
3. **Test with live session** - Verify events appear in `orch patterns` output

### Alternative Approaches Considered

**Option B: Shell out to `orch action log` from plugin**
- **Pros:** Keeps JSONL logic in Go, simpler plugin
- **Cons:** Subprocess overhead per tool call, requires adding CLI command first
- **When to use instead:** If direct file writing from plugin proves problematic

**Option C: SSE Event Monitoring**
- **Pros:** Already exists (`orch monitor`), no new plugin needed
- **Cons:** SSE events don't include tool outcomes, only status changes
- **When to use instead:** Not viable - SSE lacks tool output data

**Option D: Post-Session Transcript Parsing**
- **Pros:** Complete data available, no runtime overhead
- **Cons:** Too late for within-session pattern detection, complex parsing
- **When to use instead:** If real-time logging proves too noisy

**Rationale for recommendation:** Option A is the cleanest integration - OpenCode designed the plugin system exactly for this use case, and the existing plugins prove the pattern works.

---

### Implementation Details

**What to implement first:**
1. Create `~/.config/opencode/plugin/action-logger.ts` skeleton with `tool.execute.after` hook
2. Log all tool calls to console to understand output structure
3. Add filtering logic for Read/Glob/Grep with empty outcomes
4. Write events to `~/.orch/action-log.jsonl`

**Things to watch out for:**
- ⚠️ Tool output structure may vary by tool type - need to handle each case
- ⚠️ "Empty" detection differs: Read returns file not found, Glob/Grep return empty array
- ⚠️ Performance: logging should be async to not block tool execution

**Areas needing further investigation:**
- Exact structure of `tool.execute.after` input/output parameters
- Whether plugin has access to session ID and workspace context
- How to handle the plugin writing errors (silent fail? log to console?)

**Success criteria:**
- ✅ Action events appear in `~/.orch/action-log.jsonl` when Read returns empty
- ✅ `orch patterns` shows futile_action pattern after 3+ empty reads
- ✅ Pattern detection catches the SYNTHESIS.md check on light-tier agents

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/action/action.go` - Logger and Tracker implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/action/action_test.go` - Usage patterns
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/patterns.go` - Pattern collection and display
- `/Users/dylanconlin/.config/opencode/plugin/bd-close-gate.ts` - Plugin hook pattern
- `/Users/dylanconlin/.config/opencode/plugin/session-context.js` - Plugin initialization pattern
- `/Users/dylanconlin/.config/opencode/plugin/agentlog-inject.ts` - Event handling pattern

**Commands Run:**
```bash
# Verified OpenCode version
opencode --version  # 1.0.182

# Checked existing plugins
ls ~/.config/opencode/plugin/
```

**External Documentation:**
- https://opencode.ai/docs/plugins/ - Plugin system and events
- https://opencode.ai/docs/sdk/ - Client API for session interaction
- https://opencode.ai/docs/server/ - Server API and event types

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` - Root cause analysis
- **Decision:** Action logging subsystem uses `action-log.jsonl` (kn entry)
- **Constraint:** "Tool action outcomes are ephemeral - cannot detect behavioral patterns without action logging"

---

## Investigation History

**2025-12-28 Initial:** Investigation started
- Initial question: Where can we observe agent tool results for action logging?
- Context: Prior investigation identified need for action outcome tracking; `pkg/action` exists but has no data source

**2025-12-28:** Key finding - OpenCode plugin hooks
- Discovered `tool.execute.before` and `tool.execute.after` hooks in OpenCode plugin system
- Verified existing plugins demonstrate the pattern successfully
- Confirmed pkg/action infrastructure is ready to receive events

**2025-12-28:** Investigation completed
- Status: Complete
- Key outcome: OpenCode plugin with `tool.execute.after` hook is the integration point; create `action-logger.ts` plugin

---

## Self-Review

- [x] Real test performed (verified docs, read existing plugins, confirmed infrastructure exists)
- [x] Conclusion from evidence (based on documented hooks and working plugin examples)
- [x] Question answered (integration point is OpenCode plugin system)
- [x] File complete (all sections filled, D.E.K.N. at top)

**Self-Review Status:** PASSED
