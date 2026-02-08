<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** claude-sneakpeek enables native multi-agent features by patching Claude Code's cli.js to bypass statsig feature flags, while orch-go orchestrates from outside using external tools (beads, registry, skills).

**Evidence:** Verified swarm-mode-patch.ts (line 79-107) patches gate function from `return!1` to `return!0`; orch-go's registry.go and context.go show external orchestration via OpenCode API and spawn context injection.

**Knowledge:** Two fundamentally different philosophies - "unlock what's inside" (sneakpeek) vs "build on top" (orch-go). Sneakpeek depends on Claude Code's internal implementations surviving updates; orch-go is provider-agnostic but adds complexity. Both solve multi-agent coordination but are complementary rather than competing.

**Next:** Consider monitoring sneakpeek's swarm mode maturity - if Anthropic stabilizes native multi-agent, could simplify orch-go's spawn system. No immediate action needed.

**Promote to Decision:** recommend-no - Informational comparison, not architectural choice

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

# Investigation: Claude Sneakpeek Comparison

**Question:** How does claude-sneakpeek approach multi-agent orchestration compared to orch-go? What problems does it solve and what are the architectural differences?

**Started:** 2026-01-24
**Updated:** 2026-01-24
**Owner:** og-inv-clone-https-github-24jan-7b63
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Sneakpeek uses binary patching to enable hidden Claude Code features

**Evidence:** The `swarm-mode-patch.ts` file (lines 79-107) shows how sneakpeek patches Claude Code's cli.js:
```typescript
// Pattern: function XX(){if(Yz(process.env.CLAUDE_CODE_AGENT_SWARMS))return!1;return xK("tengu_brass_pebble",!1)}
// Changes gate function from return!1 (disabled) to return!0 (enabled)
const patched = content.replace(gate.fullMatch, `function ${gate.fnName}(){return!0}`);
```

This bypasses the `tengu_brass_pebble` statsig feature flag to enable native swarm mode, TeammateTool, delegate mode, and teammate coordination.

**Source:** `~/Documents/personal/claude-sneakpeek/src/core/variant-builder/swarm-mode-patch.ts:79-107`

**Significance:** Fundamentally different from orch-go's approach. Sneakpeek enables features Anthropic has already built but not released publicly. This means the multi-agent implementation quality depends on Anthropic, but is fragile to Claude Code updates that might change function signatures.

---

### Finding 2: Sneakpeek creates isolated Claude Code "variants" with separate configs

**Evidence:** Each variant lives in `~/.claude-sneakpeek/<variant>/` with:
- `npm/` - Patched Claude Code installation
- `config/` - Isolated settings, sessions, MCP servers, API keys
- `tweakcc/` - Theme and prompt modifications
- `tasks/<team>/` - JSON task storage per team

Wrapper scripts set `CLAUDE_CONFIG_DIR` to isolate state completely from the main Claude Code install.

**Source:** `DESIGN.md:12-28`, `src/core/wrapper.ts:262-394`

**Significance:** This variant approach allows running multiple Claude Code instances with different configurations simultaneously. orch-go achieves similar isolation through workspaces and spawn contexts, but manages this externally rather than through environment variables.

---

### Finding 3: Sneakpeek's team mode uses file-based JSON task storage

**Evidence:** Task store (`src/core/tasks/store.ts`) manages tasks as individual JSON files:
```typescript
export function saveTask(tasksDir: string, task: Task): void {
  fs.mkdirSync(tasksDir, { recursive: true });
  const taskPath = path.join(tasksDir, `${task.id}.json`);
  writeJson(taskPath, task);
}
```
Tasks are per-team, per-variant, with team names derived dynamically from git repo folder names.

**Source:** `src/core/tasks/store.ts:49-53`, `src/core/tasks/types.ts:1-20`

**Significance:** Simpler than orch-go's beads system (SQLite + git sync) but less feature-rich. No dependency tracking, no cross-project awareness, no git-based history. Good for basic multi-agent coordination; orch-go's beads provides richer project management.

---

### Finding 4: Sneakpeek supports multiple LLM providers

**Evidence:** Supports Z.ai, MiniMax, OpenRouter, ccrouter (local models), and "mirror" (pure Anthropic API):
- Provider templates define env vars, blocked tools, splash art, and model mappings
- Z.ai variants block WebSearch/WebFetch (prefer zai-cli)
- MiniMax variants seed MCP server for coding-plan

**Source:** `src/providers/index.ts` (referenced in AGENTS.md:114-120), `src/brands/zai.ts`, `src/brands/minimax.ts`

**Significance:** orch-go is Claude Max centric (requires OAuth token management, account switching for rate limits). Sneakpeek's provider flexibility could be useful for teams wanting to use alternative backends. Different optimization goals.

---

### Finding 5: orch-go orchestrates from outside Claude Code

**Evidence:** orch-go's architecture:
- `pkg/spawn/context.go` generates SPAWN_CONTEXT.md with task instructions, skill content, beads tracking
- `pkg/registry/registry.go` tracks agents externally (agent ID, session ID, beads ID, timestamps)
- Uses OpenCode API for session management or direct Claude CLI spawning
- Beads issues provide persistent task tracking across sessions

orch-go doesn't modify Claude Code at all - it coordinates multiple vanilla Claude Code instances.

**Source:** `pkg/spawn/context.go:54-356`, `pkg/registry/registry.go:43-69`, `CLAUDE.md` architecture diagram

**Significance:** More complex but provider-independent. Works with stock Claude Code. The "external orchestration" approach means orch-go can evolve independently of Claude Code releases, but requires maintaining its own state management layer.

---

## Synthesis

**Key Insights:**

1. **"Inside vs Outside" Philosophy** - Sneakpeek modifies Claude Code's internals to unlock hidden features (Finding 1), while orch-go builds orchestration externally without touching Claude Code (Finding 5). Both achieve multi-agent coordination but with opposite integration strategies.

2. **Trade-off: Simplicity vs Resilience** - Sneakpeek's approach is simpler (enable existing code) but fragile to Claude Code updates. orch-go's approach is more complex (external state management, spawn contexts, beads) but survives Claude Code changes. Sneakpeek depends on Anthropic's implementation; orch-go owns its implementation.

3. **Different Provider Strategies** - Sneakpeek targets teams wanting alternative LLM providers (Finding 4). orch-go targets Claude Max power users needing multi-account management and cross-project orchestration. They serve different user personas.

**Answer to Investigation Question:**

claude-sneakpeek and orch-go solve multi-agent orchestration through fundamentally different approaches:

**Sneakpeek** patches Claude Code's cli.js to bypass `tengu_brass_pebble` statsig flag, enabling native `TeammateTool`, `TaskCreate/Get/Update/List`, and swarm mode that Anthropic built but hasn't publicly released. It creates isolated "variants" with separate configs, uses simple file-based JSON task storage, and supports multiple LLM providers.

**orch-go** builds orchestration externally: spawning vanilla Claude Code processes, injecting context via SPAWN_CONTEXT.md, tracking agents in an external registry, and managing work through beads issues (SQLite + git sync). It's Claude Max centric with OAuth token management and multi-account support.

The key philosophical difference is **"unlock what exists" vs "build on top"**. Sneakpeek bets that Anthropic's native implementation is good and will stabilize; orch-go builds a provider-agnostic layer that owns its implementation. They're complementary rather than competing - sneakpeek could simplify orch-go's spawn system if native swarm mode matures, while orch-go's beads/registry/skill system provides richer project management than sneakpeek's JSON tasks.

---

## Structured Uncertainty

**What's tested:**

- ✅ Cloned repo successfully and examined source code (verified: `git clone` succeeded, read TypeScript files)
- ✅ Swarm mode patching mechanism (verified: read swarm-mode-patch.ts, regex patterns match documented behavior)
- ✅ Task storage structure (verified: read store.ts, types.ts - JSON file-based per-team tasks)
- ✅ Wrapper script implementation (verified: read wrapper.ts - sets CLAUDE_CONFIG_DIR, handles splash art, team mode env vars)

**What's untested:**

- ⚠️ Actually running sneakpeek to verify swarm mode works (not installed/tested)
- ⚠️ Performance comparison between native swarm mode and orch-go spawn (not benchmarked)
- ⚠️ How well native TeammateTool handles complex orchestration patterns (not evaluated)
- ⚠️ Stability of sneakpeek across Claude Code version updates (no longitudinal data)

**What would change this:**

- Finding would change if Anthropic publicly enables swarm mode (sneakpeek patching would become unnecessary)
- Finding would change if native task storage gained beads-like features (dependency tracking, git sync)
- Finding would change if orch-go adopted native swarm mode as spawn backend

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Monitor and Evaluate** - No immediate changes to orch-go. Monitor sneakpeek's swarm mode maturity as a potential future simplification path.

**Why this approach:**
- orch-go's external orchestration is battle-tested and provider-agnostic
- Sneakpeek's patching approach is fragile to Claude Code updates
- Native swarm mode is still feature-flagged (not stable API surface)
- Beads provides richer task management than sneakpeek's JSON storage

**Trade-offs accepted:**
- More complexity in orch-go vs potentially simpler native swarm mode
- Maintaining spawn context generation vs delegating to native tools
- Acceptable because stability and features outweigh simplicity gains

**Implementation sequence:**
1. No immediate implementation needed
2. Revisit if Anthropic publicly enables swarm mode (removes feature flag)
3. Evaluate native TeammateTool API for potential spawn backend integration

### Alternative Approaches Considered

**Option B: Adopt sneakpeek's patching approach**
- **Pros:** Simpler, uses native implementations, less code to maintain
- **Cons:** Fragile to updates, requires patching per Claude Code version, loses beads integration
- **When to use instead:** If orch-go complexity becomes unmanageable and native swarm mode stabilizes

**Option C: Hybrid approach - use sneakpeek for spawning, orch-go for tracking**
- **Pros:** Best of both worlds - native multi-agent with external state management
- **Cons:** Two systems to maintain, potential state sync issues
- **When to use instead:** If native task management improves but beads is still needed

**Rationale for recommendation:** orch-go's value is in beads integration, cross-project orchestration, and skill system - not just spawning agents. Sneakpeek optimizes for spawning; orch-go optimizes for project-level coordination. Different goals, complementary not competing.

---

### Implementation Details

**What to implement first:**
- Nothing immediate - this is informational

**Things to watch out for:**
- ⚠️ If Anthropic enables swarm mode publicly, reassess spawn architecture
- ⚠️ `tengu_brass_pebble` flag name may change - would break sneakpeek
- ⚠️ Native task format may evolve, could enable tighter integration

**Areas needing further investigation:**
- How does native TeammateTool handle failure/recovery vs orch-go's abandon/retry?
- What's the native task notification/monitoring mechanism?
- Could orch-go's beads model be implemented as a Claude Code extension?

**Success criteria:**
- ✅ Documented understanding of both approaches (this investigation)
- ✅ Decision framework for when to reconsider (native swarm mode GA)
- ✅ Knowledge preserved for future architectural decisions

---

## References

**Files Examined:**
- `~/Documents/personal/claude-sneakpeek/README.md` - Project overview and install instructions
- `~/Documents/personal/claude-sneakpeek/DESIGN.md` - Architecture and variant structure
- `~/Documents/personal/claude-sneakpeek/AGENTS.md` - Repository guidelines and implementation details
- `~/Documents/personal/claude-sneakpeek/CHANGELOG.md` - Feature history, team mode evolution
- `~/Documents/personal/claude-sneakpeek/src/core/variant-builder/swarm-mode-patch.ts` - Patching mechanism
- `~/Documents/personal/claude-sneakpeek/src/core/tasks/store.ts` - Task storage implementation
- `~/Documents/personal/claude-sneakpeek/src/core/tasks/types.ts` - Task data structure
- `~/Documents/personal/claude-sneakpeek/src/team-pack/index.ts` - Team mode prompt injection
- `~/Documents/personal/claude-sneakpeek/src/core/wrapper.ts` - Variant launcher script
- `~/Documents/personal/claude-sneakpeek/package.json` - Dependencies and version
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` - orch-go context generation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/registry/registry.go` - orch-go agent tracking

**Commands Run:**
```bash
# Clone the repository
git clone https://github.com/mikekelly/claude-sneakpeek ~/Documents/personal/claude-sneakpeek

# Explore source structure
find ~/Documents/personal/claude-sneakpeek/src -type f -name "*.ts" -o -name "*.tsx"
```

**External Documentation:**
- https://github.com/mikekelly/claude-sneakpeek - Source repository
- Demo video: https://x.com/NicerInPerson/status/2014989679796347375 - Swarm mode in action

**Related Artifacts:**
- **Decision:** N/A - This is informational, no decision required
- **Investigation:** N/A - First investigation of sneakpeek
- **Workspace:** `.orch/workspace/og-inv-clone-https-github-24jan-7b63/` - This agent's workspace

---

## Investigation History

**2026-01-24 15:41:** Investigation started
- Initial question: Compare claude-sneakpeek's approach to multi-agent orchestration with orch-go
- Context: Orchestrator wanted to understand alternative approaches in the ecosystem

**2026-01-24 15:45:** Cloned repository, examined documentation
- Understood variant architecture, swarm mode patching, team mode

**2026-01-24 15:50:** Deep dive into source code
- Examined swarm-mode-patch.ts, task storage, wrapper implementation
- Compared with orch-go's spawn/context.go and registry.go

**2026-01-24 16:00:** Investigation completed
- Status: Complete
- Key outcome: Two complementary philosophies - sneakpeek unlocks native features via patching, orch-go builds external orchestration. Different goals, different trade-offs.
