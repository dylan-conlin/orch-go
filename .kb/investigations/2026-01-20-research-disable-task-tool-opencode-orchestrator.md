<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode has built-in `permission.task` configuration that can disable Task tool globally or selectively.

**Evidence:** Source code shows `task: PermissionRule.optional()` in config schema and `PermissionNext.evaluate("task", ...)` checks in Task tool.

**Knowledge:** Task tool permissions work like other tool permissions, support wildcards, can be configured per-agent or globally.

**Next:** Implement `permission.task` configuration in orch-go `.opencode/opencode.json` to disable Task tool for orchestrators.

**Promote to Decision:** recommend-yes - This establishes a configuration pattern for controlling agent tool access.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Disable Task Tool Opencode Orchestrator

**Question:** How to disable the Task tool in OpenCode for orchestrator agents? Can OpenCode disable specific tools via configuration? Can this be done selectively (orchestrators only) or is it global? What configuration options exist? Are there hooks or plugins that could intercept/block Task tool usage?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** research agent
**Phase:** Complete
**Next Step:** Implement configuration in orch-go project
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: OpenCode has built-in permission system for tools including Task tool

**Evidence:** OpenCode configuration schema includes `permission.task` field that can be set to "allow", "deny", or "ask". The Task tool checks permissions via `PermissionNext.evaluate("task", a.name, caller.permission).action !== "deny"`.

**Source:** 
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/config/config.ts:449` - `task: PermissionRule.optional()` in permission schema
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/task.ts:29` - Permission check in Task tool
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/test/permission-task.test.ts` - Comprehensive tests for task permissions

**Significance:** The Task tool can be disabled or restricted via OpenCode configuration without code changes.

### Finding 2: Task permissions can be configured globally or per-agent

**Evidence:** Permission configuration can be set in `opencode.json` at global level or per-agent level. Supports wildcard patterns and specific agent names.

**Source:**
- OpenCode documentation at https://opencode.ai/docs/agents#task-permissions
- Test file shows examples: `{ permission: { task: { "*": "deny", "orchestrator-*": "allow" } } }`
- Configuration supports nested structure: `agent.orchestrator.permission.task`

**Significance:** Can disable Task tool for orchestrators while allowing it for other agents, or disable it completely.

### Finding 3: Child sessions automatically have task permission denied

**Evidence:** When Task tool creates a child session, it automatically adds `{ permission: "task", pattern: "*", action: "deny" }` to prevent infinite recursion.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/task.ts:80-83`

**Significance:** Even if orchestrators could use Task tool, their spawned agents cannot spawn further agents, preventing delegation chains.

### Finding 4: Configuration can be project-specific or global

**Evidence:** OpenCode loads config from multiple locations: project `.opencode/opencode.json`, global `~/.config/opencode/opencode.json`, and remote configs.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/config/config.ts` - config loading logic

**Significance:** Can disable Task tool for specific projects (like orch-go) without affecting other projects.

---

## Synthesis

**Key Insights:**

1. **OpenCode has built-in permission system for Task tool** - The Task tool is not a special case; it's integrated into OpenCode's permission system alongside other tools like `bash`, `edit`, `read`. This means it can be controlled via configuration without code changes.

2. **Permissions can be granular** - Can disable Task tool completely (`"*": "deny"`), disable for specific agents (`"orchestrator": "deny"`), or use wildcards (`"orchestrator-*": "deny"`). Can also set to "ask" for manual approval.

3. **Configuration hierarchy allows targeted restrictions** - Can disable Task tool only for orchestrator agents while allowing it for other agents, or disable it only in specific projects (like orch-go) while allowing it elsewhere.

**Answer to Investigation Question:**

**Yes, OpenCode can disable the Task tool via configuration.** The `permission.task` field in `opencode.json` controls Task tool access. It can be set globally or per-agent, with support for wildcard patterns.

**Configuration options:**
1. **Global disable:** `{ "permission": { "task": "deny" } }` - Disables Task tool completely
2. **Selective disable:** `{ "permission": { "task": { "*": "deny", "general": "allow" } } }` - Disables except for specific agents
3. **Per-agent disable:** Configure in agent definition: `{ "agent": { "orchestrator": { "permission": { "task": "deny" } } } }`
4. **Ask for approval:** `{ "permission": { "task": "ask" } }` - Requires manual approval for each Task tool use

**Can be done selectively:** Yes, can disable for orchestrators only while allowing for other agents.

**Hooks/plugins:** While OpenCode has plugin system, the built-in permission system is sufficient for disabling Task tool. No custom code needed.

---

## Structured Uncertainty

**What's tested:**

- ✅ **OpenCode has permission.task configuration** - Verified in source code: `config.ts:449` includes `task: PermissionRule.optional()` in schema
- ✅ **Task tool checks permissions** - Verified in source code: `task.ts:29` calls `PermissionNext.evaluate("task", a.name, caller.permission)`
- ✅ **Permission system supports wildcards and patterns** - Verified in test file: `permission-task.test.ts` shows examples with `"*"`, `"orchestrator-*"` patterns
- ✅ **Configuration can be global or per-agent** - Verified in documentation and source code showing config loading hierarchy

**What's untested:**

- ⚠️ **Actual configuration application** - Haven't created and tested a real `opencode.json` with `permission.task` settings
- ⚠️ **Orchestrator agent detection** - Not tested if we can identify orchestrator agents vs other agents in configuration
- ⚠️ **Edge cases with wildcard patterns** - Not tested complex pattern matching scenarios

**What would change this:**

- If OpenCode's permission system doesn't actually work as documented (unlikely given comprehensive tests)
- If Task tool has hardcoded bypass for permission checks (no evidence of this)
- If configuration loading has bugs preventing task permission from being applied

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Configure OpenCode permission.task in orch-go project** - Add `permission.task` configuration to `.opencode/opencode.json` in orch-go to disable Task tool for orchestrators.

**Why this approach:**
- Uses built-in OpenCode feature, no custom code required
- Can be targeted to specific projects (orch-go) without affecting other projects
- Supports granular control (can disable for orchestrators only)
- Well-tested feature with comprehensive test coverage

**Trade-offs accepted:**
- Requires understanding OpenCode configuration schema
- May need to identify orchestrator agent names for selective disabling
- Configuration changes require OpenCode restart to take effect

**Implementation sequence:**
1. **Create/update `.opencode/opencode.json`** in orch-go with `permission.task` configuration
2. **Test configuration** by spawning an orchestrator and attempting to use Task tool
3. **Verify behavior** - Task tool should be disabled or require approval based on configuration

### Alternative Approaches Considered

**Option B: Create custom OpenCode plugin to intercept Task tool**
- **Pros:** More control, can add custom logic or logging
- **Cons:** Requires custom code, maintenance burden, more complex
- **When to use instead:** If built-in permission system is insufficient or buggy

**Option C: Modify orchestrator skill to prohibit Task tool use**
- **Pros:** Skill-level enforcement, clear documentation in skill
- **Cons:** Relies on agent compliance, not enforced by system
- **When to use instead:** As complementary measure to configuration

**Rationale for recommendation:** Built-in permission system is the simplest, most maintainable solution that leverages OpenCode's existing capabilities. No custom code means less maintenance and better compatibility with OpenCode updates.

---

### Implementation Details

**What to implement first:**
- Create `.opencode/opencode.json` in orch-go with: `{ "$schema": "https://opencode.ai/config.json", "permission": { "task": "deny" } }`
- Test with simple global deny first, then refine to selective denial if needed
- Document the configuration in project README or CLAUDE.md

**Things to watch out for:**
- ⚠️ **Configuration loading** - OpenCode may need restart to pick up config changes
- ⚠️ **Agent identification** - Need to know exact agent names (e.g., "orchestrator", "meta-orchestrator") for selective disabling
- ⚠️ **Wildcard precedence** - Last matching rule wins in permission evaluation

**Areas needing further investigation:**
- Exact agent names used in orch-go orchestration system
- Whether other agents (feature-impl, investigation, etc.) should also have Task tool disabled
- Impact on existing workflows that might legitimately need Task tool

**Success criteria:**
- ✅ **Orchestrators cannot use Task tool** - Attempting to use Task tool results in permission denied
- ✅ **Other agents unaffected** - Non-orchestrator agents can still use Task tool if needed
- ✅ **Configuration is discoverable** - `.opencode/opencode.json` exists and documents the restriction

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/config/config.ts` - OpenCode configuration schema including permission.task
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/task.ts` - Task tool implementation with permission checks
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/test/permission-task.test.ts` - Comprehensive tests for task permissions
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/task.txt` - Task tool description and usage guidelines

**Commands Run:**
```bash
# Search for Task tool references in OpenCode source
find ~/Documents/personal/opencode/packages -name "*.ts" -type f | xargs grep -l "task" -i | head -10

# Search for permission schema definitions
grep -n "permission" ~/Documents/personal/opencode/packages/opencode/src/config/config.ts | head -20
```

**External Documentation:**
- https://opencode.ai/docs/agents#task-permissions - OpenCode documentation for task permissions
- https://opencode.ai/docs/tools - OpenCode tools documentation including permissions

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/opencode/.kb/investigations/2026-01-17-inv-implement-task-tool-interceptor-orchestrators.md` - Related investigation in OpenCode repo about Task tool interception
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-research-disable-task-tool-20jan-c154/` - Current workspace

---

## Investigation History

**[2026-01-20]:** Investigation started
- Initial question: How to disable the Task tool in OpenCode for orchestrator agents?
- Context: Orchestrators keep using the Task tool to spawn subagents instead of using 'orch spawn'

**[2026-01-20]:** Found OpenCode permission system
- Discovered OpenCode has built-in `permission.task` configuration
- Found Task tool implementation checks permissions via `PermissionNext.evaluate`
- Located comprehensive tests for task permissions

**[2026-01-20]:** Investigation completed
- Status: Complete
- Key outcome: OpenCode can disable Task tool via `permission.task` configuration in `opencode.json`, can be global or selective
