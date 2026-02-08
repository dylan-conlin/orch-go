# Design: Action Batching Layer for Playwright MCP

**TLDR:** Recommend adding a single `browser_batch` tool to Playwright MCP that accepts an array of actions and executes them sequentially, returning consolidated results with per-action status. This achieves 3-5x round-trip reduction for common workflows while maintaining structured error handling and MCP compliance.

**Status:** Complete
**Phase:** Complete
**Date:** 2025-12-27
**Issue:** orch-go-7872

---

## Design Question

How should we design an action batching layer for Playwright MCP that allows an agent to submit a sequence of actions that execute until failure or completion, returning consolidated results, to reduce LLM round-trips by 3-5x for common workflows?

---

## Problem Framing

### Current State

Playwright MCP exposes ~30 individual tools (browser_click, browser_type, browser_navigate, etc.). Each tool call requires a complete LLM round-trip.

For a typical login workflow:
```
browser_navigate → wait → browser_type → wait → browser_type → wait → browser_click → wait → browser_snapshot → wait
```
**5 round trips** when the pattern is predictable and could be batched.

### Success Criteria

1. **3-5x reduction in round-trips** for common workflows
2. **Graceful failure handling** - stop at first error, report which action failed
3. **Clear batch semantics** - agents understand what can be batched together
4. **MCP protocol compliance** - works within existing MCP tool/response model
5. **Incremental adoption** - doesn't require rewriting existing Playwright MCP
6. **Observable execution** - can see which actions succeeded before failure

### Constraints

**Technical:**
- MCP protocol is request-response (JSON-RPC 2.0)
- Playwright MCP already has ~30 individual tools
- Actions have dependencies (can't click element before page loads)
- Some actions require intermediate results
- Error recovery may need human/LLM decision

**From principles:**
- **Session Amnesia** - design must be resumable, state externalized
- **Compose Over Monolith** - prefer composable building blocks
- **Evidence Hierarchy** - execution results are truth, batch spec is hypothesis

### Scope

**In Scope:** Batch submission, execution, failure detection, result consolidation, safe action combinations

**Out of Scope:** Changes to existing tools, visual execution, Playwright test framework, parallel execution

---

## Exploration

### Approach 1: Single Batch Tool (New MCP Tool)

Add a new `browser_batch` tool accepting an array of actions:

```json
{
  "tool": "browser_batch",
  "arguments": {
    "actions": [
      { "tool": "browser_navigate", "params": { "url": "..." } },
      { "tool": "browser_type", "params": { "ref": "...", "text": "..." } },
      { "tool": "browser_click", "params": { "ref": "..." } }
    ],
    "stopOnError": true
  }
}
```

**Pros:** Clean MCP integration, minimal changes, full control
**Cons:** Duplicates schemas, complex nested JSON
**Complexity:** Medium | **Risk:** Low

### Approach 2: Proxy Layer with Action Queue

Proxy MCP server with `batch_start`, normal tool calls (queued), `batch_execute`:

**Pros:** Uses existing schemas, familiar patterns, gradual batching
**Cons:** Proxy architecture, state management, orphaned batches
**Complexity:** High | **Risk:** Medium

### Approach 3: Playwright Code Tool (Exists)

Use existing `browser_run_code` with JavaScript:

```json
{
  "tool": "browser_run_code",
  "arguments": {
    "code": "async (page) => { await page.goto('...'); await page.fill('...'); }"
  }
}
```

**Pros:** Zero implementation, full Playwright API
**Cons:** LLM must generate valid JS, security concerns, opaque errors
**Complexity:** None | **Risk:** High (reliability)

### Approach 4: Action DSL Tool

Custom domain-specific language:

```yaml
- navigate: https://example.com
- fill:
    selector: input[name='user']
    text: admin
- click: button[type='submit']
```

**Pros:** Simple syntax, constrained, readable
**Cons:** New syntax to learn, parser maintenance
**Complexity:** Medium-High | **Risk:** Medium

### Approach 5: Hybrid with Safe Workflow Types

Predefined workflow patterns with strong typing:

```json
{
  "tool": "browser_workflow",
  "arguments": {
    "type": "form_fill",
    "steps": [ ... ],
    "verify": { "text": "Dashboard" }
  }
}
```

**Pros:** Constrained to safe patterns, simple per-type schema
**Cons:** Limited to predefined patterns
**Complexity:** Medium | **Risk:** Low

---

## Synthesis

### Evaluation Summary

| Criterion | Batch Tool | Proxy | Code | DSL | Workflows |
|-----------|-----------|-------|------|-----|-----------|
| Round-trip reduction | ✅ | ✅ | ✅ | ✅ | ✅ |
| Failure handling | ✅ | ⚠️ | ❌ | ✅ | ✅ |
| Clear semantics | ✅ | ⚠️ | ❌ | ⚠️ | ✅ |
| Observable execution | ✅ | ✅ | ❌ | ✅ | ✅ |
| LLM reliability | ⚠️ | ⚠️ | ❌ | ⚠️ | ✅ |

---

## Recommendations

⭐ **RECOMMENDED:** Approach 1 (Single Batch Tool) with Approach 5 enhancements

**Why:**
- Single new tool minimizes MCP surface area
- Uses existing tool schemas as sub-schemas (no duplication)
- Structured actions enable clear failure reporting
- Additive change - doesn't break existing tools
- Aligns with Compose Over Monolith principle

**Trade-off accepted:** LLM must construct nested JSON, but this is well-understood and tool/schema documentation can guide it.

### Detailed Design

#### TypeScript Interfaces

```typescript
interface BatchAction {
  tool: string;                          // existing tool name
  params: Record<string, unknown>;       // existing tool params
  label?: string;                        // for error reporting
  continueOnError?: boolean;             // override stopOnError
}

interface BatchRequest {
  actions: BatchAction[];
  options?: {
    stopOnError?: boolean;               // default: true
    captureIntermediateSnapshots?: boolean;  // default: false
    timeout?: number;                    // per-action timeout
    maxActions?: number;                 // safety limit (default: 20)
  };
}

interface BatchResult {
  completed: number;
  total: number;
  status: 'success' | 'partial' | 'failed';
  results: ActionResult[];
  finalSnapshot?: string;
  failedAt?: {
    index: number;
    action: string;
    error: string;
  };
}

interface ActionResult {
  index: number;
  tool: string;
  label?: string;
  status: 'success' | 'skipped' | 'failed';
  duration: number;
  result?: unknown;
  error?: string;
}
```

#### Action Safety Classification

**Batchable (safe):**
- `browser_navigate`, `browser_click`, `browser_type`
- `browser_fill_form`, `browser_select_option`, `browser_hover`
- `browser_press_key`, `browser_wait_for`

**Batchable with caution:**
- `browser_evaluate`, `browser_drag`

**Not batchable (require immediate response):**
- `browser_snapshot` - often used for LLM decisions
- `browser_file_upload` - may need confirmation
- `browser_handle_dialog` - reactive to dialog state
- `browser_tabs` - changes context

#### Safety Limits

- Max 20 actions per batch (configurable)
- 5-second per-action timeout (configurable)
- 60-second total batch timeout
- No nested batches

### Example Usage

**Login Flow (5 actions → 1 round-trip):**
```json
{
  "tool": "browser_batch",
  "arguments": {
    "actions": [
      { "tool": "browser_navigate", "params": { "url": "https://app.example.com" }},
      { "tool": "browser_fill_form", "params": { 
        "fields": [
          { "ref": "ref_user", "value": "admin" },
          { "ref": "ref_pass", "value": "secret" }
        ]
      }},
      { "tool": "browser_click", "params": { "ref": "ref_submit" }},
      { "tool": "browser_wait_for", "params": { "text": "Welcome" }}
    ]
  }
}
```

**Failure Response:**
```json
{
  "completed": 2,
  "total": 4,
  "status": "partial",
  "results": [
    { "index": 0, "tool": "browser_navigate", "status": "success", "duration": 523 },
    { "index": 1, "tool": "browser_fill_form", "status": "success", "duration": 234 },
    { "index": 2, "tool": "browser_click", "status": "failed", "error": "Element not found" },
    { "index": 3, "tool": "browser_wait_for", "status": "skipped" }
  ],
  "failedAt": { "index": 2, "action": "browser_click", "error": "Element not found" },
  "finalSnapshot": "... accessibility tree at failure point ..."
}
```

### Implementation Options

**Option A:** Add to Playwright MCP directly
- Fork microsoft/playwright-mcp
- Add `browser_batch` tool implementation
- Submit upstream PR

**Option B:** Create wrapper MCP server
- Separate package that wraps Playwright MCP
- Adds batch tool, delegates others
- No upstream dependency

**Recommendation:** Start with Option B for faster iteration, then consider Option A for upstream contribution.

### When This Would Change

- Playwright MCP adds native batching → use that
- MCP protocol adds streaming tools → real-time feedback possible
- Common patterns emerge → named workflows (Approach 5)
- LLMs reliably generate JS → Approach 3 becomes viable

---

## File Targets

For implementation (if proceeding):
- `pkg/mcp/batch/batch.go` - Batch tool implementation
- `pkg/mcp/batch/executor.go` - Sequential action executor
- `pkg/mcp/batch/safety.go` - Action classification and limits
- `pkg/mcp/batch/types.go` - Request/response types

---

## Acceptance Criteria

- [ ] `browser_batch` tool defined with proper MCP schema
- [ ] Sequential execution with stop-on-error behavior
- [ ] Per-action status in response
- [ ] Final snapshot included on completion or failure
- [ ] Safety limits enforced (max actions, timeouts)
- [ ] Integration with existing Playwright MCP tools

---

## Out of Scope

- Parallel action execution
- Visual/screenshot-based workflows
- Playwright test framework integration
- Changes to existing individual tools
- Named workflow types (future enhancement)

---

## Related Artifacts

- **Principles:** Session Amnesia, Compose Over Monolith, Evidence Hierarchy
- **MCP Spec:** https://modelcontextprotocol.io/docs/concepts/tools
- **Playwright MCP:** https://github.com/microsoft/playwright-mcp
