## Summary (D.E.K.N.)

**Delta:** Claude Code v2.0-2.1 introduces major features relevant to orch: MCP tool search (85% token reduction), skills hot-reload, hooks in skill frontmatter, background agents with Ctrl+B, status line JSON API, and session teleportation to claude.ai/code.

**Evidence:** Official changelog (1096+ commits in v2.1.0), Anthropic announcements, documentation at code.claude.com/docs.

**Knowledge:** Subagents now properly inherit parent model; status line receives rich JSON context (model, context window %, cost); skills can define scoped hooks in frontmatter; MCP tool search is default (no opt-in needed).

**Next:** Update orch spawn context to leverage status line JSON API for orchestrator context injection; evaluate skill frontmatter hooks for worker coordination.

**Promote to Decision:** recommend-no (informational research, no architectural decision needed)

---

# Investigation: Recent Claude Code Updates Relevant to Orch Ecosystem

**Question:** What Claude Code features and breaking changes since v2.0 are relevant to the orch orchestration system?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** og-research-research-recent-claude-22jan-0b6c
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: MCP Tool Search - 85% Token Reduction

**Evidence:** MCP Tool Search shipped in v2.1.7, now enabled by default. Lazy-loads tool definitions on-demand instead of upfront. Anthropic benchmarks: 77K tokens → 8.7K tokens with 50+ MCP tools.

**Source:**
- https://venturebeat.com/orchestration/claude-code-just-got-updated-with-one-of-the-most-requested-user-features
- Changelog v2.1.7: "MCP tool search auto mode enabled by default"

**Significance:** For orch ecosystem, this means agents with many MCP servers no longer consume context on tool definitions. Accuracy improvement: Opus 4.5 accuracy jumped from 79.5% to 88.1% on MCP evaluations.

---

### Finding 2: Status Line JSON API with Rich Context

**Evidence:** Status line commands receive structured JSON via stdin containing:
- `model.id` and `model.display_name` (current model)
- `context_window.used_percentage` and `remaining_percentage`
- `cost.total_cost_usd`, `total_duration_ms`, `total_lines_added/removed`
- `workspace.current_dir` and `project_dir`
- `session_id` and `transcript_path`

**Source:**
- https://code.claude.com/docs/en/statusline
- Verified JSON schema in documentation

**Significance:** Orch can inject orchestration context (agent ID, issue ID, spawn skill) into the status line, giving spawned agents visibility into their orchestration context. This is a clean integration point.

---

### Finding 3: Skills Hot-Reload and Frontmatter Hooks

**Evidence:** v2.1.0 introduced:
- Automatic skill hot-reload (no session restart needed)
- `context: fork` for isolated sub-agent contexts
- Hooks in skill frontmatter (`PreToolUse`, `PostToolUse`, `Stop`)
- `disable-model-invocation: true` to prevent Claude auto-invoking skills
- `user-invocable: false` to hide from user slash commands

**Source:**
- https://www.threads.com/@boris_cherny/post/DTOyRyBD018 (v2.1.0 announcement)
- https://code.claude.com/docs/en/skills

**Significance:** Orch skills can now define scoped hooks without global settings.json pollution. Hot-reload enables skill development without restarting agents. Forked context prevents worker skills from polluting orchestrator context.

---

### Finding 4: Background Agents and Unified Ctrl+B

**Evidence:** v2.0.64 introduced async agents/bash with message passing. v2.1.0 added unified Ctrl+B for backgrounding both agents and shell commands. Background tasks continue even after main agent completes.

**Source:**
- Changelog v2.0.64: "Asynchronous agents and bash commands with message passing"
- https://www.threads.com/@boris_cherny/post/DR9aGzQkSP1 (background agents demo)

**Significance:** Orch can leverage native background agents for long-running tasks (monitoring, builds) without blocking orchestrator. Environment variable `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS` available to disable if needed.

---

### Finding 5: Session Teleportation to claude.ai/code

**Evidence:** v2.1.0 added `/teleport` and `/remote-env` commands. Sessions can be moved from CLI to claude.ai/code web interface. Requires GitHub connection and same claude.ai user.

**Source:**
- Changelog v2.1.0: "Added /teleport and /remote-env commands"
- https://code.claude.com/docs/en/claude-code-on-the-web

**Significance:** For orch ecosystem, teleportation is one-way (CLI → web only). Not useful for current orch spawn architecture which uses headless sessions. May be useful for debugging stuck agents by teleporting to web.

---

### Finding 6: Subagent Model Inheritance Fixed

**Evidence:** v2.1.0 changelog: "Subagents improved to inherit parent's model by default". Historical bug in v1.0.72 where Task tool subagents defaulted to Sonnet instead of inheriting configured model.

**Source:**
- Changelog v2.1.0
- GitHub Issue #5456 (marked resolved)

**Significance:** Orch spawn doesn't use Task tool (disabled in .opencode/opencode.json), but this fix is relevant if Task tool is re-enabled. Model inheritance now works correctly.

---

### Finding 7: Breaking Changes Summary

**Evidence:** Key breaking changes since v2.0:
1. **v2.0.25**: Legacy SDK entrypoint removed; migrate to `@anthropic-ai/claude-agent-sdk`
2. **v2.0.35/v2.0.62**: `ignorePatterns` migrated to deny permissions
3. **v2.0.41**: `AgentOutputTool`/`BashOutputTool` → unified `TaskOutputTool`
4. **v2.0.73**: Removed custom ripgrep configuration
5. **v2.1.0**: SDK minimum zod peer dependency changed to ^4.0.0
6. **v2.1.3**: Slash commands and skills merged (no behavior change)

**Source:** GitHub CHANGELOG.md, official releases

**Significance:**
- SDK rename affects any direct SDK usage (orch uses CLI, not SDK - not affected)
- `ignorePatterns` → deny permissions migration relevant if orch configures project-level ignores
- ripgrep config removal affects agents relying on custom rg settings

---

### Finding 8: Hooks Enhancements

**Evidence:** Hook system expanded significantly:
- v2.0.10: `PreToolUse` can modify tool inputs
- v2.0.43: `SubagentStart` hook, `permissionMode` field for custom agents
- v2.0.45: `PermissionRequest` hook for automatic tool approval/denial
- v2.1.0: Hooks in agent/skill frontmatter (scoped to component lifecycle)
- v2.1.3: Hook timeout extended from 60s to 10 minutes
- v2.1.9: `PreToolUse` can return `additionalContext`

**Source:** Changelog entries, https://code.claude.com/docs/en/hooks

**Significance:** Orch coaching plugins can leverage `PreToolUse` for context injection and tool modification. `PermissionRequest` hook enables automatic approval for specific tools. Frontmatter hooks enable per-skill coaching without global config.

---

### Finding 9: Context Window and Memory Improvements

**Evidence:**
- v2.1.14: Fixed context window calculation regression (was ~65%, now ~98%)
- v2.1.14: Fixed memory leak in long-running sessions
- v2.1.14: Fixed memory issues with parallel subagents
- v2.1.2: Large bash/tool outputs now persisted to disk

**Source:** Changelog v2.1.14, v2.1.2

**Significance:** Long-running orch sessions and parallel agent spawns should be more stable. Memory leaks that may have caused orchestrator crashes are fixed.

---

### Finding 10: New CLI Commands and Settings

**Evidence:** Notable additions:
- `/usage` - view plan limits (v2.0.0)
- `/rewind` - undo code changes (v2.0.0)
- `/config` with search (v2.1.6)
- `/stats` with date filtering (v2.1.6)
- `/teleport`, `/remote-env` (v2.1.0)
- `plansDirectory` setting for custom plan file locations (v2.1.9)
- `CLAUDE_CODE_TMPDIR` for custom temp directories (v2.1.5)
- `CLAUDE_CODE_SHELL` for custom shell (v2.0.65)

**Source:** Changelog entries

**Significance:** `/usage` useful for orch monitoring usage. `plansDirectory` could centralize plan files for multi-agent coordination. Custom tmpdir/shell settings relevant for containerized orch deployments.

---

## Synthesis

**Key Insights:**

1. **Status Line as Orchestration Integration Point** - The status line JSON API provides a clean way to inject orchestration context into spawned agents. The JSON includes model, context window %, cost, workspace info - exactly what an orchestrator needs to monitor.

2. **Skills System Now Production-Ready** - Hot-reload, frontmatter hooks, and context forking make skills a robust mechanism for defining worker behaviors. Orch skills can define their own hooks without polluting global settings.

3. **MCP Efficiency Gains Are Significant** - 85% token reduction for MCP-heavy workflows means orchestrators can run more tools without context pressure. This is automatic (no opt-in).

4. **Background Agent Architecture Aligns with Orch Patterns** - Native background agents with message passing align well with orch's headless spawn model. Ctrl+B unification suggests Anthropic sees this as a core pattern.

5. **Memory/Stability Improvements Critical for Long-Running Orchestration** - Fixes to memory leaks and parallel subagent issues directly address orchestrator stability concerns.

**Answer to Investigation Question:**

The most orch-relevant updates since v2.0 are:
1. **Status line JSON API** - Integration point for orchestration context
2. **Skills hot-reload and frontmatter hooks** - Enables per-skill coaching
3. **MCP tool search** - Automatic 85% token savings
4. **Background agents** - Native support for orch's spawn pattern
5. **Memory/stability fixes** - Critical for orchestrator reliability

No breaking changes directly affect orch's current architecture (CLI-based spawn, OpenCode API for headless sessions). The SDK rename to Claude Agent SDK is only relevant if orch starts using the SDK directly.

---

## Structured Uncertainty

**What's tested:**

- Status line receives JSON context (verified in official docs)
- MCP tool search is enabled by default (verified in changelog)
- Skills hot-reload works (per changelog and user reports)
- Subagent model inheritance fixed (per changelog)

**What's untested:**

- Status line performance impact in high-frequency update scenarios
- Skill frontmatter hooks interaction with orch's coaching plugins
- Background agent message passing integration with orch monitoring
- Teleportation for debugging stuck agents (not tested in practice)

**What would change this:**

- Anthropic deprecating status line JSON API
- Skills system architectural changes
- MCP tool search opt-in requirement (currently default-on)
- Permission system changes affecting orch spawn context injection

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation for orch ecosystem.

### Recommended Approach: Status Line Integration

**Status Line for Orchestration Context** - Add orchestration context (agent ID, issue ID, skill name) to agent status lines via spawn context generation.

**Why this approach:**
- Status line receives structured JSON with model/context/cost info
- Clean integration point without modifying agent prompts
- Visible to both agent and human operators

**Trade-offs accepted:**
- Requires updating spawn context generation
- Status line script must be deployed to agent environments

**Implementation sequence:**
1. Create status line script that includes orch context from environment variables
2. Update `pkg/spawn/context.go` to include status line configuration
3. Test with tmux spawns for visibility

### Alternative Approaches Considered

**Option B: Skill Frontmatter Hooks for Worker Coordination**
- **Pros:** Per-skill coaching without global config
- **Cons:** Requires skill deployment pipeline updates
- **When to use instead:** When implementing per-worker coaching behaviors

**Option C: Leverage Background Agents for Monitoring**
- **Pros:** Native Claude Code feature, survives orchestrator restart
- **Cons:** Different architecture than current orch spawn
- **When to use instead:** For long-running monitoring tasks

---

### Implementation Details

**What to implement first:**
- Status line script with orch context (quick win, high visibility)
- Evaluate skill frontmatter hooks for coaching plugins

**Things to watch out for:**
- Status line updates at most every 300ms - don't rely on real-time
- Background agents require Max/Pro subscription for cloud execution
- Skill hot-reload may cause brief interruptions

**Areas needing further investigation:**
- How frontmatter hooks interact with existing coaching plugins
- Background agent message passing protocol details
- Teleportation for debugging (low priority, manual use case)

**Success criteria:**
- Spawned agents display orchestration context in status line
- Status line shows context window % for proactive compaction
- Skill updates take effect without agent restart

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` - Project architecture context
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawn.md` - Spawn flow reference

**Commands Run:**
```bash
# Web searches for Claude Code changelog, features, breaking changes
# WebFetch for official documentation
```

**External Documentation:**
- https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md - Official changelog
- https://code.claude.com/docs/en/statusline - Status line documentation
- https://code.claude.com/docs/en/skills - Skills documentation
- https://code.claude.com/docs/en/hooks - Hooks reference

**Related Artifacts:**
- **Guide:** `.kb/guides/spawn.md` - How spawn context generation works
- **Guide:** `.kb/guides/claude-code-sandbox-architecture.md` - Sandbox constraints

---

## Investigation History

**2026-01-22 17:30:** Investigation started
- Initial question: What Claude Code updates since v2.0 are relevant to orch?
- Context: Orch uses Claude Code CLI and OpenCode API for agent orchestration

**2026-01-22 17:45:** Major findings documented
- Researched v2.0.0 through v2.1.15 changelog
- Identified 10 orch-relevant feature categories
- Documented breaking changes

**2026-01-22 18:00:** Investigation completed
- Status: Complete
- Key outcome: Status line JSON API and skills frontmatter hooks are primary integration points
