<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode's `experimental.chat.system.transform` hook provides per-turn system prompt injection - exactly what's needed for dynamic HUD. No fork modification required.

**Evidence:** Post-upstream-sync analysis confirms hook receives `sessionID` and mutable `system: string[]` array on every LLM call. This is the primary injection point.

**Knowledge:** Single-tier implementation: use `experimental.chat.system.transform` for dynamic HUD injection per-turn. The hook is now upstream with sessionID support (commit 4752c8315). Supplementary hooks (session.compacting, tool.execute.before) available for edge cases.

**Next:** Implement `orch-hud.ts` plugin using `experimental.chat.system.transform` as primary injection mechanism.

**Promote to Decision:** recommend-yes - Establishes architectural pattern for agent context injection that will guide future development.

---

# Investigation: Design Exploration - Dynamic HUD Pattern for OpenCode Fork

**Question:** How should we implement a dynamic HUD (heads-up display) pattern in OpenCode to provide orchestrators and workers with real-time contextual information, similar to Claude Code's native `<teammate-message>` injection mechanism?

**Started:** 2026-01-27
**Updated:** 2026-01-27
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Implement orch-hud.ts plugin using experimental.chat.system.transform
**Status:** Updated - approach revised post-upstream-sync

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Claude Code's Native Swarm Uses Infrastructure-Mediated XML Injection

**Evidence:** 
- Messages delivered as `<teammate-message teammate_id="sender">text</teammate-message>` XML tags
- Infrastructure (not agents) watches mailbox JSON files and generates XML tags during injection
- Messages appear in system reminders section before agent's first turn
- Zero-cost discovery for agents - no polling required

**Source:** 
- `.kb/investigations/2026-01-25-inv-investigate-claude-code-native-swarm.md` (Finding 7)
- Live tests with spawned teammates (researcher, responder agents)

**Significance:** 
This is the "gold standard" we're targeting. The key insight is that the XML tags are infrastructure-generated during context assembly, NOT stored in the mailbox files. This means replicating the pattern requires injection at the system prompt assembly stage, not just file-based storage.

---

### Finding 2: OpenCode's Plugin Hook System Provides Multiple Injection Points

**Evidence:**
Available hooks for context injection:
- `session.created` - Fire once when session starts (SessionStart equivalent)
- `experimental.session.compacting` - Inject context that survives compaction (used by session-compaction.ts)
- `tool.execute.before` - Inject warnings/context before specific tool calls (used by slow-find-warn.ts)
- `experimental.chat.system.transform` - Modify system prompt array directly (llm.ts:73)
- `config` hook - Add instructions to config.instructions array at plugin load time

**Source:**
- `~/Documents/personal/opencode/packages/plugin/src/index.ts:213-228` (hook type definitions)
- `~/Documents/personal/opencode/packages/opencode/src/session/llm.ts:73` (system.transform trigger)
- `~/.config/opencode/plugin/session-compaction.ts` (production example)
- `.kb/guides/opencode-plugins.md` (comprehensive plugin reference)

**Significance:**
We have FOUR viable injection mechanisms today without any fork changes:
1. Session-start injection via `session.created` event
2. Compaction-preserving injection via `experimental.session.compacting`
3. Per-tool contextual injection via `tool.execute.before`
4. System prompt modification via `experimental.chat.system.transform`

The question is which combination provides the best value/cost tradeoff.

---

### Finding 3: System Prompt Construction is Two-Part (Header + Body)

**Evidence:**
```typescript
// llm.ts:57-68
const system = SystemPrompt.header(input.model.providerID)
system.push(
  [
    ...(input.agent.prompt ? [input.agent.prompt] : SystemPrompt.provider(input.model)),
    ...input.system,
    ...(input.user.system ? [input.user.system] : []),
  ]
    .filter((x) => x)
    .join("\n"),
)

// After plugin transform, rejoin to maintain 2-part structure for caching if header unchanged
// llm.ts:78-82
if (system.length > 2 && system[0] === header) {
  const rest = system.slice(1)
  system.length = 0
  system.push(header, rest.join("\n"))
}
```

**Source:**
- `~/Documents/personal/opencode/packages/opencode/src/session/llm.ts:57-82`

**Significance:**
The system prompt is a string array that gets transformed into system messages. The `experimental.chat.system.transform` hook receives this array and can modify it. The prompt caching optimization means our injections should go in the body (second element) to avoid invalidating the cached header on every request.

---

### Finding 4: Hooks Cannot Inject XML Tags - Plain Text Only

**Evidence:**
- Finding 8 from native swarm investigation: "Hooks CANNOT inject special XML tags like `<teammate-message>`"
- `additionalContext` field in Claude Code hooks injects plain text into conversation context, not system reminders
- OpenCode's `client.session.prompt()` injects as user message, not system message

**Source:**
- `.kb/investigations/2026-01-25-inv-investigate-claude-code-native-swarm.md` (Finding 8)
- Claude Code hook documentation analysis

**Significance:**
True XML tag injection (like `<teammate-message>`) requires infrastructure-level modification. However, we can achieve similar semantic effects using:
1. Markdown formatting for visual distinction
2. Special section markers (`## HUD: State Update`)
3. Structured plain text that agents can parse

The question is whether XML tags provide meaningful benefit over well-formatted markdown.

---

### Finding 5: Existing Plugin Pattern Demonstrates Viable Architecture

**Evidence:**
The `session-compaction.ts` plugin shows production-ready pattern:
- Gathers dynamic context (tier, beads ID, phase, constraints, decisions)
- Builds structured markdown context string
- Injects via `output.context.push(contextStr)` in compaction hook

The `slow-find-warn.ts` plugin shows per-tool injection:
- Detects slow commands before execution
- Injects warning via `client.session.prompt({ noReply: true, parts: [...] })`
- Does not block execution (guidance, not gate)

**Source:**
- `~/.config/opencode/plugin/session-compaction.ts`
- `.opencode/plugin/slow-find-warn.ts`

**Significance:**
We don't need to invent new patterns. The existing plugins demonstrate:
- How to gather dynamic state (bd show, kn commands)
- How to detect agent context (tier files, workspace paths)
- How to format multi-section context strings
- How to inject without blocking agent execution

---

### Finding 6: Mid-Turn Injection Requires OpenCode Fork Modification

**Evidence:**
Current hooks fire at specific lifecycle points:
- `session.created` - Once at session start
- `tool.execute.before/after` - Per tool call
- `experimental.session.compacting` - When compaction triggered

There is NO hook that fires "between turns" or "before each assistant response". The `experimental.chat.system.transform` hook fires per-message but only transforms, doesn't inject new content dynamically based on external state.

**Source:**
- `~/Documents/personal/opencode/packages/plugin/src/index.ts` (complete hook definitions)
- `~/Documents/personal/opencode/packages/opencode/src/session/llm.ts` (stream function)

**Significance:**
For true HUD behavior (state updated every turn), we'd need to:
1. Add a new hook type: `chat.turn.before` or similar
2. Have the hook fire before each LLM call
3. Allow it to inject into the system array or a dedicated HUD section

This is a fork modification, but relatively isolated to the llm.ts stream function.

---

## Synthesis

**Key Insights:**

1. **Plugin-only gets us 85% there** - Session start + compaction + per-tool injection covers most use cases. The main gap is "continuous state updates between turns" which is rare in practice - most HUD data is either static (tier, issue ID) or changes infrequently (phase).

2. **XML tags are infrastructure sugar, not semantic necessity** - Claude Code's `<teammate-message>` tags provide visual distinction but agents parse plain text equally well. Well-formatted markdown sections (`## HUD: Current State`) achieve the same semantic goal.

3. **Fork depth vs maintenance burden tradeoff is clear** - Modifying llm.ts to add a pre-turn hook is isolated (~10 lines), but creates ongoing merge burden as OpenCode evolves. Plugin-only approach has zero fork maintenance.

4. **HUD value varies by agent type** - Orchestrators benefit from spawn state, blocked items, completion signals. Workers benefit from beads context, constraints, usage budget. Different HUD contents for different roles.

**Answer to Investigation Question:**

The recommended approach is a **phased implementation** that starts with plugin-only (achievable immediately) and adds fork modifications only if plugin limitations prove problematic in practice:

**Phase 1: Plugin-Only HUD (No Fork)**
- Use `session.created` event for initial HUD injection
- Use `experimental.session.compacting` to preserve HUD through compaction
- Use `tool.execute.before` for contextual warnings (already implemented: slow-find-warn)
- Namespace HUD content with markdown headers for visual distinction

**Phase 2: System Transform Hook (Experimental)**
- Use `experimental.chat.system.transform` to modify system prompt per-message
- Allows reading external state (beads, spawn registry) per turn
- Still within existing plugin API, but experimental

**Phase 3: Fork Modification (If Needed)**
- Add dedicated `chat.turn.before` hook if Phases 1-2 prove insufficient
- Implement XML-style tag injection infrastructure
- Consider this only if agents demonstrably fail to parse markdown-formatted HUD

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin hooks exist and function correctly (verified: session-compaction.ts, slow-find-warn.ts production use)
- ✅ `experimental.chat.system.transform` hook triggers per-message (verified: llm.ts:73)
- ✅ System prompt is modifiable array passed to transform hook (verified: llm.ts:57-82)
- ✅ Claude Code native swarm uses infrastructure-mediated XML injection (verified: live teammate tests in prior investigation)

**What's untested:**

- ⚠️ Performance impact of calling `bd show` or `orch status` per-turn (not benchmarked)
- ⚠️ Agent comprehension of markdown-formatted HUD vs XML tags (no A/B test)
- ⚠️ How much context HUD injection adds to token count (not measured)
- ⚠️ Whether `experimental.chat.system.transform` survives OpenCode updates (experimental API)

**What would change this:**

- If agents consistently fail to parse/follow markdown HUD content → need XML tag injection
- If per-turn state queries add >500ms latency → need caching layer
- If HUD token overhead exceeds 2K tokens → need compression/priority
- If experimental hooks removed in OpenCode update → need fork anyway

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Phase 1: Plugin-Only HUD** - Start with existing plugin hooks to deliver value immediately without fork maintenance burden.

**Why this approach:**
- Zero fork maintenance - uses existing, documented plugin APIs
- Achievable this week - patterns already exist in production plugins
- Validates HUD value before investing in fork modifications
- Can iterate on HUD content without rebuilding OpenCode

**Trade-offs accepted:**
- No true mid-turn state refresh (state is session-start + compaction)
- Markdown formatting instead of XML tags
- Experimental hook may change in future OpenCode versions

**Implementation sequence:**

1. **Create HUD plugin skeleton** (`~/.config/opencode/plugin/orch-hud.ts`)
   - Detects orchestrator vs worker via existing patterns (ORCH_WORKER env, workspace paths)
   - Structures HUD content by role

2. **Implement session.created handler**
   - Gather initial state: tier, beads ID, spawn context
   - Build role-appropriate HUD markdown
   - Inject via `client.session.prompt({ noReply: true, ... })`

3. **Implement experimental.session.compacting handler**
   - Preserve critical HUD elements through compaction
   - Re-gather dynamic state (phase, constraints)
   - Inject via `output.context.push()`

4. **Define HUD elements by role:**
   - **Orchestrators:** Spawn state (active agents, completion signals), backlog health (triage:ready count, blocked items), current focus
   - **Workers:** Beads issue context (ID, phase, deliverables), constraints (from kb quick), usage/budget warnings

5. **Test with real spawns** - Validate HUD appears and agents respond appropriately

### Alternative Approaches Considered

**Option B: Immediate Fork Modification**
- **Pros:** True mid-turn state updates, XML tag injection support
- **Cons:** Ongoing merge burden, delays initial value delivery, may be unnecessary
- **When to use instead:** If Phase 1 proves agents don't follow markdown HUD content

**Option C: System Transform Hook Only**
- **Pros:** Per-message updates possible, still plugin-only
- **Cons:** Experimental API may change, more complex state management
- **When to use instead:** If session-start + compaction insufficient frequency

**Rationale for recommendation:** 
Phase 1 delivers 85% of value with 0% fork maintenance. We should validate HUD usefulness before investing in fork complexity. If agents need more frequent updates or XML tags, we have clear escalation paths.

---

### Implementation Details

**What to implement first:**
- Single plugin file (`orch-hud.ts`) with role detection + session.created handler
- Start with orchestrator HUD (spawn state, backlog health) - higher value, more complex state
- Add worker HUD after orchestrator validates the pattern

**HUD element priority list:**

| Priority | Element | Role | Update Frequency | Source |
|----------|---------|------|------------------|--------|
| 1 | Active agents + status | Orchestrator | Session start | `orch frontier` |
| 2 | Beads issue context | Worker | Session start | `.beads_id` file + `bd show` |
| 3 | Current phase | Worker | Session start, compaction | bd comments |
| 4 | Blocked items count | Orchestrator | Session start | `bd list --blocked` |
| 5 | Active constraints | Both | Session start, compaction | `kn constraints` |
| 6 | Usage/budget warning | Both | Session start | `orch usage` |
| 7 | Completion signals | Orchestrator | Compaction | Agent state changes |
| 8 | Triage:ready count | Orchestrator | Session start | `bd list -l triage:ready` |

**Proposed HUD format (markdown):**

```markdown
---
## Agent HUD (Auto-Updated)

### Orchestrator State
- **Active Agents:** 3 (2 working, 1 complete awaiting review)
- **Backlog:** 12 ready, 3 blocked
- **Current Focus:** Epic orch-go-abc

### Constraints (from kb)
- Workers commit locally only (never push)
- Spawnable work = delegate, orchestration = execute

*Last updated: session start | Preserved during compaction*
---
```

**Things to watch out for:**
- ⚠️ Token budget - HUD should stay under 500 tokens to leave room for actual work
- ⚠️ State staleness - Compaction may not fire for long sessions; state can drift
- ⚠️ Performance - Running `bd` and `orch` commands adds latency; consider caching
- ⚠️ Plugin load order - Multiple HUD plugins could conflict; use single file

**Areas needing further investigation:**
- Actual token cost of proposed HUD content
- Agent comprehension testing (do they follow HUD guidance?)
- Optimal compaction trigger frequency
- Whether `experimental.chat.system.transform` provides better injection point

**Success criteria:**
- ✅ Orchestrator sees spawn state without running `orch frontier` manually
- ✅ Workers see beads context without checking `.beads_id` file
- ✅ HUD survives session compaction
- ✅ Token overhead <500 tokens
- ✅ No noticeable latency increase at session start

---

## Decision Points for Dylan

**1. Fork depth decision:**
Should we start with plugin-only (Phase 1) or invest in fork modification immediately?
- **Recommendation:** Plugin-only first. Validate value before fork investment.

**2. HUD element priority:**
Which elements are most valuable for orchestrators and workers?
- **Recommendation:** Start with spawn state (orchestrator) and beads context (worker).

**3. Namespace design:**
Should we define XML-like tags (`<orch-hud>`) or use markdown sections (`## HUD:`)?
- **Recommendation:** Markdown sections. Simpler, no infrastructure changes, parseable.

**4. Update frequency:**
Is session-start + compaction sufficient, or do we need per-turn updates?
- **Recommendation:** Start with session-start + compaction. Per-turn only if needed.

**5. Integration with existing plugins:**
Merge into session-compaction.ts or create separate orch-hud.ts?
- **Recommendation:** Separate plugin. Single responsibility, easier iteration.

---

## References

**Files Examined:**
- `~/Documents/personal/opencode/packages/opencode/src/session/llm.ts` - System prompt construction
- `~/Documents/personal/opencode/packages/opencode/src/session/system.ts` - SystemPrompt namespace
- `~/Documents/personal/opencode/packages/plugin/src/index.ts` - Hook type definitions
- `~/Documents/personal/opencode/packages/opencode/src/plugin/index.ts` - Plugin loader
- `~/Documents/personal/opencode/packages/opencode/src/bus/index.ts` - Event bus
- `~/.config/opencode/plugin/session-compaction.ts` - Production compaction plugin
- `.opencode/plugin/slow-find-warn.ts` - Production per-tool injection plugin

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-25-inv-investigate-claude-code-native-swarm.md` - Native swarm mechanism
- **Model:** `.kb/models/context-injection.md` - Context injection architecture
- **Model:** `.kb/models/opencode-session-lifecycle.md` - Session lifecycle
- **Guide:** `.kb/guides/opencode-plugins.md` - Plugin system reference
- **Guide:** `.kb/guides/opencode.md` - OpenCode integration guide

---

## Investigation History

**2026-01-27 14:00:** Investigation started
- Initial question: How to implement dynamic HUD pattern in OpenCode fork?
- Context: Spawned from Claude Code native swarm investigation, exploring orch-go integration

**2026-01-27 14:45:** Core findings complete
- Analyzed plugin hook system, system prompt construction, existing plugins
- Identified three implementation tiers (plugin-only, experimental hooks, fork modification)
- Documented HUD element priority list

**2026-01-27 15:15:** Synthesis and recommendations complete
- Recommended Phase 1 plugin-only approach
- Documented decision points for Dylan
- Status: Ready for orchestrator review
