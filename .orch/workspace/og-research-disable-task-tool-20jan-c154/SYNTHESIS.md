# Session Synthesis

**Agent:** og-research-disable-task-tool-20jan-c154
**Issue:** ad-hoc spawn (no beads tracking)
**Duration:** 2026-01-20 → 2026-01-20
**Outcome:** success

---

## TLDR

Researched how to disable OpenCode's Task tool for orchestrator agents. Found that OpenCode has built-in `permission.task` configuration that can disable Task tool globally or selectively via `opencode.json`. No code changes needed - can configure at project or global level.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-20-research-disable-task-tool-opencode-orchestrator.md` - Research findings on disabling Task tool

### Files Modified
- None (research-only session)

### Commits
- Will commit investigation file after creating SYNTHESIS.md

---

## Evidence (What Was Observed)

- OpenCode configuration schema includes `task: PermissionRule.optional()` at `config.ts:449`
- Task tool implementation checks permissions via `PermissionNext.evaluate("task", a.name, caller.permission)` at `task.ts:29`
- Comprehensive tests in `permission-task.test.ts` show permission patterns: `"*": "deny"`, `"orchestrator-*": "allow"`, etc.
- Child sessions automatically have task permission denied: `{ permission: "task", pattern: "*", action: "deny" }` at `task.ts:80-83`
- OpenCode documentation at https://opencode.ai/docs/agents#task-permissions confirms configuration options

### Tests Run
```bash
# Searched OpenCode source for Task tool implementation
find ~/Documents/personal/opencode/packages -name "*.ts" -type f | xargs grep -l "task" -i | head -10

# Examined configuration schema
grep -n "permission" ~/Documents/personal/opencode/packages/opencode/src/config/config.ts | head -20
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-20-research-disable-task-tool-opencode-orchestrator.md` - Complete research findings with D.E.K.N. summary

### Decisions Made
- Recommended using built-in OpenCode permission system over custom plugins or skill modifications
- Recommended starting with global disable (`"task": "deny"`) then refining to selective if needed

### Constraints Discovered
- Configuration changes require OpenCode restart to take effect
- Need to know exact agent names for selective disabling (e.g., "orchestrator", "meta-orchestrator")
- Last matching rule wins in permission evaluation (important for wildcard ordering)

### Externalized via `kb`
- `kb quick decide "Use OpenCode permission.task configuration to disable Task tool" --reason "Built-in feature, no custom code, supports granular control"`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** Implement Task tool disable configuration in orch-go
**Skill:** feature-impl
**Context:**
Create `.opencode/opencode.json` in orch-go with `permission.task` configuration to disable Task tool for orchestrators. Start with global disable (`{ "permission": { "task": "deny" } }`) then test and refine to selective if needed. Document configuration.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What exact agent names are used in orch-go orchestration system? (orchestrator, meta-orchestrator, etc.)
- Should other agents (feature-impl, investigation, etc.) also have Task tool disabled?
- What impact on existing workflows that might legitimately need Task tool?

**Areas worth exploring further:**
- OpenCode plugin system for more advanced tool interception if needed
- Configuration validation tools for OpenCode

**What remains unclear:**
- How orchestrator agents are identified in OpenCode configuration (exact names)
- Whether there are any legitimate use cases for orchestrators to use Task tool

---

## Session Metadata

**Skill:** research
**Model:** anthropic/claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-research-disable-task-tool-20jan-c154/`
**Investigation:** `.kb/investigations/2026-01-20-research-disable-task-tool-opencode-orchestrator.md`
**Beads:** ad-hoc spawn (no beads tracking)