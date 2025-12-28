# Session Synthesis

**Agent:** og-inv-action-logging-integration-28dec
**Issue:** orch-go-vp6g
**Duration:** 2025-12-28 → 2025-12-28
**Outcome:** success

---

## TLDR

Investigated action logging integration points for detecting repeated futile agent actions. Found that OpenCode's `tool.execute.after` plugin hook is the integration point - a TypeScript plugin can observe tool outcomes and write to the existing `~/.orch/action-log.jsonl` infrastructure.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-action-logging-integration-points-agent.md` - Complete investigation with findings, recommendations, and implementation details

### Files Modified
- None

### Commits
- (pending - investigation file ready for commit)

---

## Evidence (What Was Observed)

- OpenCode v1.0.182 has plugin system with `tool.execute.before` and `tool.execute.after` hooks (source: https://opencode.ai/docs/plugins/)
- Existing plugins at `~/.config/opencode/plugin/` demonstrate the hook pattern (e.g., `bd-close-gate.ts`)
- `pkg/action/action.go` has complete Logger infrastructure with `LogEmpty`, `LogError` methods
- `cmd/orch/patterns.go` already collects action patterns and displays them
- The gap is: no data source writes to `action-log.jsonl` currently

### Verification
- Confirmed OpenCode version: `opencode --version` → 1.0.182
- Read existing plugins to verify hook patterns work
- Verified `orch patterns` command exists and handles action patterns

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-action-logging-integration-points-agent.md` - Full integration point analysis

### Decisions Made
- Integration via OpenCode plugin (not orch-go hooks or SSE monitoring) because it's the designed extension point
- Focus on investigative tools (Read/Glob/Grep) with non-success outcomes to match the "sharp concept"
- Plugin writes directly to `action-log.jsonl` (not shell out to orch command)

### Constraints Discovered
- SSE events don't include tool output - only status changes
- Post-session transcript parsing is too late for pattern detection
- Glass MCP is not the right model - it controls tools directly, but agent sessions use OpenCode

### Integration Point Summary
| Approach | Feasibility | Why/Why Not |
|----------|-------------|-------------|
| OpenCode `tool.execute.after` | ✅ Best | Designed for this, has tool output |
| SSE Event Monitoring | ❌ No | Missing tool output data |
| Post-Session Parsing | ⚠️ Possible | Too late, complex |
| Hooks into orch-go | ❌ No | Wrong layer - orch-go spawns, doesn't see tool calls |

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, ready for implementation)

### If Close
- [x] All deliverables complete (investigation file with findings)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-vp6g`

### Follow-Up Work Needed

**Issue to Create:** "Implement action-logger OpenCode plugin"
**Skill:** feature-impl
**Context:**
```
Create ~/.config/opencode/plugin/action-logger.ts that:
1. Hooks tool.execute.after
2. Filters for Read/Glob/Grep with empty/error outcomes
3. Writes ActionEvent to ~/.orch/action-log.jsonl

See .kb/investigations/2025-12-28-inv-action-logging-integration-points-agent.md for details.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is the exact structure of `tool.execute.after` input/output? (Needs live testing)
- How to determine "empty" outcome varies by tool - Read returns error, Glob/Grep return empty array
- Should patterns trigger automatic `kn tried` entries?

**Areas worth exploring further:**
- Whether pattern surfacing should happen at session start (inject via plugin)
- Performance impact of logging every tool call

**What remains unclear:**
- Whether plugin has access to session ID and workspace context
- Best error handling strategy for plugin write failures

---

## Session Metadata

**Skill:** investigation
**Model:** (spawned agent)
**Workspace:** `.orch/workspace/og-inv-action-logging-integration-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-action-logging-integration-points-agent.md`
**Beads:** `bd show orch-go-vp6g`
