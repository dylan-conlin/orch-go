# Session Synthesis

**Agent:** og-arch-design-action-batching-27dec
**Issue:** orch-go-7872
**Duration:** 2025-12-27
**Outcome:** success

---

## TLDR

Designed action batching layer for Playwright MCP. Recommend a single `browser_batch` tool that accepts an array of actions, executes sequentially with stop-on-error, and returns consolidated results with per-action status. Achieves 3-5x round-trip reduction for common workflows while maintaining structured error handling and MCP compliance.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-design-action-batching-layer-playwright.md` - Full architect investigation with design recommendation

### Files Modified
- `.orch/features.json` - Added feat-019 for Playwright MCP action batching wrapper

### Commits
- (pending) architect: design action batching layer for Playwright MCP

---

## Evidence (What Was Observed)

- MCP protocol uses request-response (JSON-RPC 2.0) - each tool call is a round-trip
- Playwright MCP has ~30 individual tools (browser_click, browser_type, etc.)
- Login workflow example: 5 tool calls = 5 round-trips when pattern is predictable
- `browser_run_code` tool exists but requires JS generation (unreliable for LLMs)
- MCP tools support `inputSchema` for parameters and `outputSchema` for structured responses
- Playwright MCP already has structured result handling pattern

### Research Conducted
- Reviewed MCP protocol specification (tools, calling, responses)
- Analyzed Playwright MCP repository structure and tool list
- Reviewed meta-orchestration principles for alignment

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-design-action-batching-layer-playwright.md` - Full design document with 5 approaches evaluated

### Decisions Made
- **Single batch tool approach** over proxy layer, DSL, or JS code generation
  - Rationale: Clean MCP integration, uses existing tool schemas, additive change
- **Wrapper MCP server first** before upstream PR
  - Rationale: Faster iteration, no external dependency on acceptance
- **Action safety classification** for batchable vs non-batchable tools
  - Rationale: Some tools (snapshot, dialog handling) require LLM decision-making

### Constraints Discovered
- MCP protocol doesn't have streaming tools - batch result is all-or-nothing
- Some actions (browser_snapshot) shouldn't be batched - used for LLM decisions
- Safety limits needed: max 20 actions, 5s per-action timeout

### Externalized via `kn`
- Not needed - design is comprehensive in investigation artifact

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Feature list reviewed and updated (feat-019 added)
- [x] Ready for `orch complete orch-go-7872`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orch-go have built-in MCP server capabilities? Currently focuses on orchestration
- Could future MCP streaming tools change the batching design?
- How would parallel action execution change safety constraints?

**Areas worth exploring further:**
- Named workflow templates (Approach 5) once common patterns emerge
- Upstream contribution to microsoft/playwright-mcp

**What remains unclear:**
- Performance characteristics of large batches (>10 actions)
- How well LLMs handle nested JSON for batch construction in practice

*(Straightforward design session - primary decision was clear)*

---

## Session Metadata

**Skill:** architect
**Model:** claude
**Workspace:** `.orch/workspace/og-arch-design-action-batching-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-design-action-batching-layer-playwright.md`
**Beads:** `bd show orch-go-7872`
