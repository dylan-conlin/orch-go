<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Claude Code 2.1.x (January 2026) shipped MCP lazy loading, skill hot-reload, forked skill context, and hooks in skill frontmatter - all directly impacting orch spawn/skill architecture.

**Evidence:** Researched official changelog, release notes, and documentation from GitHub, Anthropic, and tech media.

**Knowledge:** Key orch-relevant changes: (1) MCP Tool Search saves 85% context, (2) Skills can fork to sub-agent with `context: fork`, (3) Hooks can be scoped to skill/agent lifecycle via frontmatter, (4) Subagents recover from permission denial, (5) SDK renamed to Claude Agent SDK.

**Next:** Update orch skills to leverage `context: fork` and frontmatter hooks where appropriate. Consider MCP tool search implications for spawn context budget.

**Promote to Decision:** recommend-no - Informational research, no architectural decision needed yet

---

# Investigation: Research Recent Claude Code Updates

**Question:** What recent Claude Code updates (January 2026) are relevant to the orch ecosystem's agent orchestration, spawn system, and skill management?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Dylan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: MCP Tool Search (Lazy Loading) - January 15, 2026

**Evidence:** Claude Code 2.1.7 shipped MCP Tool Search, which lazy-loads MCP tool definitions instead of preloading them into context. Activates automatically when tool descriptions exceed 10% of context window. Internal benchmarks show reduction from ~77K tokens to ~8.7K tokens (85% reduction).

**Source:**
- [VentureBeat article](https://venturebeat.com/orchestration/claude-code-just-got-updated-with-one-of-the-most-requested-user-features/)
- [GitHub Changelog](https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md)
- `auto:N` threshold syntax in v2.1.9

**Significance:** Orch-spawned agents will have more context budget available for actual work. MCP-heavy configurations (like orch agents with multiple MCP servers) no longer face context starvation. The `server instructions` field in MCP config is now critical for Tool Search to find relevant tools.

---

### Finding 2: Skill Hot Reload and Forked Context - January 7, 2026

**Evidence:** Claude Code 2.1.0 introduced:
- **Automatic skill hot-reload**: Skills in `~/.claude/skills` or `.claude/skills` activate immediately without restart
- **`context: fork` in skill frontmatter**: Skills can run in a forked sub-agent context with separate conversation history
- **`agent` field in skills**: Specify agent type (Explore, Plan, general-purpose, or custom) when using `context: fork`

**Source:**
- [GitHub Changelog v2.1.0](https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md)
- [Anthropic release notes](https://releasebot.io/updates/anthropic/claude-code)
- [Hooks reference](https://code.claude.com/docs/en/hooks)

**Significance:**
- Hot reload eliminates need to restart sessions when iterating on orch skills
- `context: fork` could replace or complement orch's spawn mechanism for lightweight delegations
- The `agent` field allows orch skills to specify execution context (e.g., Explore vs general-purpose)

---

### Finding 3: Hooks in Skill/Agent Frontmatter - January 7, 2026

**Evidence:** Claude Code 2.1.0 added ability to define PreToolUse, PostToolUse, and Stop hooks directly in skill and agent frontmatter. These hooks are scoped to the component's lifecycle only.

Features:
- Agent-scoped hooks run only during that agent's lifecycle
- `once: true` config for single-execution hooks
- PreToolUse hooks can return `additionalContext` to inject context to the model
- Hooks can control tool execution: allow (bypass permissions), deny, or ask

**Source:**
- [Hooks reference](https://code.claude.com/docs/en/hooks)
- [Eric Buess on X](https://x.com/EricBuess/status/2009073718450889209)
- Changelog v2.1.0 and v2.1.9

**Significance:**
- Orch skills can define lifecycle-scoped hooks without modifying global settings
- `additionalContext` from hooks enables orch's "pain as signal" pattern (coaching plugins)
- Verification agents could use Stop hooks to enforce output formats

---

### Finding 4: Subagent Resilience and Model Inheritance - January 2026

**Evidence:** Claude Code 2.1.x fixed several subagent issues:
- **Permission denial recovery**: Subagents now continue working after permission denial instead of stopping
- **Model inheritance**: Sub-agents correctly inherit parent model during conversation compaction
- **Web search model fix**: Sub-agents use correct model for web searches
- **Memory leak fix**: Long-running sessions no longer leak resources after shell commands
- **Parallel subagent crashes fixed**: Memory issues resolved when running parallel subagents

**Source:**
- Changelog entries for v2.1.3, v2.1.14
- [GitHub releases](https://github.com/anthropics/claude-code/releases)

**Significance:**
- Orch daemon can more reliably run multiple parallel agents
- Permission-related failures won't kill entire agent workflows
- Model specification via orch spawn will be respected throughout session

---

### Finding 5: Session Management Improvements - January 2026

**Evidence:** Multiple session-related improvements:
- **Custom session IDs**: `--session-id` with `--resume`/`--continue` and `--fork-session`
- **Session URL attribution**: Commits and PRs from sessions include source attribution
- **`${CLAUDE_SESSION_ID}` substitution**: Access session ID in skills
- **Session persistence fixes**: Race condition during OAuth token refresh fixed

**Source:**
- [Session Management docs](https://platform.claude.com/docs/en/agent-sdk/sessions)
- Changelog v2.1.0, v2.1.9

**Significance:**
- Orch could use custom session IDs for better tracking/correlation
- Session ID available in skills enables better audit trails
- Session forking could support orch's checkpoint/resume workflows

---

### Finding 6: SDK Rename - Claude Code SDK → Claude Agent SDK

**Evidence:** The Claude Code SDK has been renamed to Claude Agent SDK to reflect broader capabilities beyond coding. Package imports changed:
- TypeScript: `@anthropic-ai/claude-code` → `@anthropic-ai/claude-agent-sdk`
- Python: `from claude_code_sdk` → `from claude_agent_sdk`
- Type rename: `ClaudeCodeOptions` → `ClaudeAgentOptions`

**Source:**
- [Anthropic engineering blog](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)
- [Migration guide](https://platform.claude.com/docs/en/agent-sdk/migration-guide)
- [npm package](https://www.npmjs.com/package/@anthropic-ai/claude-agent-sdk)

**Significance:**
- Any future orch SDK integration should use the new package names
- Signals Anthropic's direction toward general agent orchestration (aligns with orch vision)

---

### Finding 7: Setup Hook Event - January 17, 2026

**Evidence:** New Setup hook event triggered via `--init`, `--init-only`, or `--maintenance` CLI flags. Allows repository setup and maintenance operations to be automated.

**Source:**
- Changelog v2.1.10
- [Releasebot notes](https://releasebot.io/updates/anthropic/claude-code)

**Significance:**
- Orch could leverage Setup hooks for project initialization
- Complements orch's beads hooks for issue-level automation

---

### Finding 8: Additional Relevant Changes

**Evidence:**
- **Thinking mode enabled by default for Opus 4.5**: Extended thinking is now default
- **Wildcard MCP permissions**: `mcp__server__*` syntax for bulk permission management
- **`--tools` flag**: Restrict built-in tools in interactive mode
- **Bash permission wildcards**: Patterns like `npm *`, `* install`, `git * main`
- **Tool hook timeout**: Increased from 60 seconds to 10 minutes
- **`showTurnDuration` setting**: Can hide "Cooked for X" messages
- **`plansDirectory` setting**: Customize where plan files are stored

**Source:** Changelog v2.1.0 through v2.1.15

**Significance:**
- Opus 4.5 thinking mode default affects orch agent behavior/cost
- Wildcard permissions simplify orch permission configuration
- Tool restriction via `--tools` could be useful for constrained orch spawns

---

## Synthesis

**Key Insights:**

1. **Context Budget Liberation** - MCP Tool Search's 85% reduction in context overhead means orch-spawned agents have substantially more working context. This is especially important for agents with multiple MCP integrations.

2. **Native Skill Forking Partially Overlaps with Orch Spawn** - The new `context: fork` capability in skills provides lightweight sub-agent spawning within Claude Code itself. This could complement orch's spawn for simple delegations, though orch's spawn provides richer context (beads integration, skill loading, workspace setup).

3. **Hooks as First-Class Orchestration Primitives** - Frontmatter hooks with `additionalContext` return enable the "pain as signal" architecture orch uses. Skill-scoped hooks mean orch skills can define their own validation/verification without global configuration.

4. **SDK Direction Validates Orch Architecture** - The rename from "Claude Code SDK" to "Claude Agent SDK" signals Anthropic sees this as general agent infrastructure, not just coding. Aligns with orch's multi-agent orchestration vision.

**Answer to Investigation Question:**

The January 2026 Claude Code updates (v2.1.0 through v2.1.15) include several changes directly relevant to orch:

**High Impact:**
- MCP Tool Search (lazy loading) - massive context savings
- Skill hot-reload - faster skill iteration
- Subagent permission denial recovery - more resilient agents
- Parallel subagent memory fixes - better daemon reliability

**Medium Impact:**
- `context: fork` and `agent` field in skills - native lightweight spawning
- Hooks in skill frontmatter - scoped lifecycle hooks
- Session ID improvements - better tracking/correlation

**Low Impact (but worth noting):**
- SDK rename (migration path if using SDK)
- Setup hooks (project initialization)
- Thinking mode default for Opus 4.5

---

## Structured Uncertainty

**What's tested:**

- ✅ MCP Tool Search is enabled by default in Claude Code 2.1.7+ (verified via changelog)
- ✅ Skill hot-reload works without restart (documented in release notes)
- ✅ SDK packages renamed (verified via npm)

**What's untested:**

- ⚠️ How `context: fork` interacts with orch's spawn context generation (not tested)
- ⚠️ Whether frontmatter hooks in orch skills work as expected (not tested)
- ⚠️ Performance impact of MCP Tool Search on orch agent startup time (not benchmarked)

**What would change this:**

- Finding would be wrong if Claude Code reverts any of these features
- If `context: fork` has limitations that make it unsuitable for orch delegations
- If MCP Tool Search has accuracy issues that hurt orch agent effectiveness

---

## Implementation Recommendations

**Purpose:** Bridge from research findings to actionable orch improvements.

### Recommended Approach ⭐

**Incremental Adoption** - Adopt features that align with existing orch patterns, defer those requiring architectural changes.

**Why this approach:**
- MCP Tool Search is automatic - no action needed, just benefit
- Skill hot-reload is automatic - already working
- Subagent fixes are automatic - daemon stability improved

**Trade-offs accepted:**
- Deferring `context: fork` experimentation
- Not immediately migrating to Claude Agent SDK (no current SDK usage)

**Implementation sequence:**
1. **Document in CLAUDE.md** - Note MCP Tool Search implications for spawn context
2. **Consider frontmatter hooks** - Evaluate for orch skills needing lifecycle-scoped behavior
3. **Monitor Opus 4.5 thinking** - Check if extended thinking default affects cost/behavior

### Alternative Approaches Considered

**Option B: Aggressive adoption of context: fork**
- **Pros:** Native lightweight spawning, reduced orch complexity
- **Cons:** Loses beads integration, workspace setup, skill loading that orch provides
- **When to use instead:** For very simple delegations where full orch spawn is overkill

**Option C: SDK migration**
- **Pros:** Future-proofing if orch ever uses SDK
- **Cons:** No current SDK usage, migration effort for nothing
- **When to use instead:** If adding direct SDK integration to orch

---

### Implementation Details

**What to implement first:**
- Add note to orch-go CLAUDE.md about MCP Tool Search
- Review orch skills for frontmatter hook opportunities

**Things to watch out for:**
- ⚠️ `server instructions` field is now critical for MCP Tool Search - ensure orch MCP configs have good instructions
- ⚠️ Thinking mode default for Opus 4.5 may affect token usage

**Areas needing further investigation:**
- How `context: fork` behaves with orch's skill system
- Whether custom session IDs could improve orch tracking

**Success criteria:**
- ✅ Orch agents benefit from MCP Tool Search (automatic)
- ✅ Skill iteration is faster with hot-reload (automatic)
- ✅ Documented understanding of new features for future use

---

## References

**Files Examined:**
- N/A (web research)

**Commands Run:**
```bash
# Web searches for Claude Code updates
# Fetched GitHub changelog
```

**External Documentation:**
- [GitHub Changelog](https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md)
- [GitHub Releases](https://github.com/anthropics/claude-code/releases)
- [Releasebot Notes](https://releasebot.io/updates/anthropic/claude-code)
- [VentureBeat - Claude Code 2.1.0](https://venturebeat.com/orchestration/claude-code-2-1-0-arrives-with-smoother-workflows-and-smarter-agents/)
- [VentureBeat - MCP Tool Search](https://venturebeat.com/orchestration/claude-code-just-got-updated-with-one-of-the-most-requested-user-features/)
- [Hooks Reference](https://code.claude.com/docs/en/hooks)
- [Session Management](https://platform.claude.com/docs/en/agent-sdk/sessions)
- [SDK Migration Guide](https://platform.claude.com/docs/en/agent-sdk/migration-guide)
- [Anthropic Engineering - Agent SDK](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)

**Related Artifacts:**
- **Guide:** `.kb/guides/claude-code-sandbox-architecture.md` - Sandbox constraints
- **Guide:** `.kb/guides/skill-system.md` - Orch skill system
- **Guide:** `.kb/guides/spawn.md` - Orch spawn flow

---

## Investigation History

**2026-01-22 09:00:** Investigation started
- Initial question: What recent Claude Code updates are relevant to orch?
- Context: Keeping orch aligned with Claude Code evolution

**2026-01-22 09:30:** Research completed
- Reviewed changelog, release notes, and documentation
- Key finding: MCP Tool Search, skill hot-reload, frontmatter hooks

**2026-01-22 09:45:** Investigation completed
- Status: Complete
- Key outcome: Identified 8 significant updates with varying orch relevance; most benefits are automatic (MCP Tool Search, hot-reload, subagent fixes)
