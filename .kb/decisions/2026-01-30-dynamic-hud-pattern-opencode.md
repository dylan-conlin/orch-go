# Decision: Dynamic HUD Pattern for OpenCode

**Date:** 2026-01-30
**Status:** Accepted
**Context:** Synthesized from investigation 2026-01-27-inv-design-exploration-dynamic-hud-pattern.md

## Summary

Use OpenCode's `experimental.chat.system.transform` hook for dynamic HUD (heads-up display) injection. Single-tier implementation via plugin (`orch-hud.ts`) provides per-turn system prompt injection with sessionID support. Hook receives mutable `system: string[]` array on every LLM call, enabling context injection without fork modification. No XML tags needed - markdown formatting achieves same semantic goal.

## The Problem

Agents need real-time contextual information without manual queries:

**Orchestrators need:**
- Spawn state (active agents, completion signals)
- Backlog health (triage:ready count, blocked items)
- Current focus

**Workers need:**
- Beads issue context (ID, phase, deliverables)
- Constraints (from kb quick)
- Usage/budget warnings

Current state: Agents must run `orch frontier`, `bd show`, `kb constraints` manually to get this context. Information scattered across commands instead of surfaced automatically.

Pain point: Session amnesia - agents don't know their own state without explicit queries. Meta-orchestrators don't see completion signals without checking each agent.

## The Decision

### Use experimental.chat.system.transform Hook

**Primary injection mechanism:** `experimental.chat.system.transform` hook

**How it works:**
- Hook fires on every LLM call (llm.ts:73)
- Receives mutable `system: string[]` array
- Receives `sessionID` (post-upstream-sync, commit 4752c8315)
- Can modify system prompt per-turn
- Allows reading external state (beads, spawn registry) per call

**Why this hook (not others):**
- `session.created`: Fires once only, not per-turn
- `tool.execute.before`: Per-tool, not per-turn
- `experimental.session.compacting`: Only during compaction (infrequent)
- `experimental.chat.system.transform`: Per-turn, has sessionID, can inject dynamically

**Implementation:** Single plugin file `~/.config/opencode/plugin/orch-hud.ts`

### HUD Content by Role

**Orchestrator HUD:**
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

*Last updated: per-turn*
---
```

**Worker HUD:**
```markdown
---
## Agent HUD (Auto-Updated)

### Current Work
- **Issue:** orch-go-12345
- **Phase:** Implementation
- **Deliverable:** SYNTHESIS.md with recommendation

### Active Constraints
- Test changes before committing
- Never push to remote (orchestrator handles that)

*Last updated: per-turn*
---
```

### Markdown Formatting (Not XML Tags)

**Rejected:** XML-style tags like `<teammate-message>` or `<orch-hud>`

**Chosen:** Markdown sections with `## Agent HUD` header

**Why markdown:**
- No infrastructure changes required
- Agents parse plain text equally well
- Visual distinction through formatting
- Simpler implementation
- Already proven in session-compaction.ts plugin

Claude Code's `<teammate-message>` tags are infrastructure-generated during context assembly, not stored. Replicating requires injection at system prompt assembly stage - which `experimental.chat.system.transform` provides, but markdown is simpler.

### No Fork Modification Required

`experimental.chat.system.transform` hook exists upstream (post Jan 27 sync, commit 4752c8315) with sessionID support. No OpenCode fork changes needed.

**Supplementary hooks available (for edge cases):**
- `experimental.session.compacting`: Preserve HUD through compaction
- `tool.execute.before`: Contextual warnings (already used: slow-find-warn.ts)

## Why This Design

### Principle: Plugin-First, Fork-Later

From OpenCode integration guide: Prefer plugin solutions over fork modifications. Plugins:
- Zero fork maintenance burden
- Iterate without rebuilding OpenCode
- User-installable without code changes

Fork modifications only when plugin limitations proven problematic in practice.

### Proven Pattern: session-compaction.ts

Production plugin demonstrates viable architecture:
- Gathers dynamic context (tier, beads ID, phase, constraints)
- Builds structured markdown
- Injects via compaction hook

Same pattern applies to HUD: gather state, format markdown, inject via hook.

### Key Insight: Per-Turn Updates Enable Fresh Context

HUD state changes infrequently (phase transitions, new agents spawned, work completed) but needs to be current when agent makes decisions.

Per-turn injection via `experimental.chat.system.transform`:
- State read on each LLM call
- Fresh beads/orch data every turn
- No manual refresh needed
- Automatic updates without polling

### Constraint: Token Budget <500 Tokens

HUD should stay under 500 tokens to leave room for actual work. This requires:
- Priority-based content (most important first)
- Compact formatting
- Limit to top 5 items per category

### Lesson: Infrastructure-Mediated Injection Works

Claude Code's native swarm investigation showed that XML tags are infrastructure-generated, not stored. The `experimental.chat.system.transform` hook provides the same injection point - modifying system prompt before LLM sees it.

## Trade-offs

**Accepted:**
- Markdown formatting instead of XML tags
- Experimental API may change in OpenCode updates
- Per-turn state queries add latency (mitigated by caching)
- No true mid-turn updates (only per-LLM-call)

**Rejected:**
- Immediate fork modification: Delays value delivery, adds maintenance burden
- Session-start only injection: State can drift over long sessions
- Client-side polling: Wastes tokens, agent-initiated instead of infrastructure-provided

## Constraints

1. **Token budget <500 tokens** - HUD content must stay compact
2. **experimental.chat.system.transform hook is experimental** - May change in future OpenCode versions
3. **SessionID required for context lookup** - Hook must receive sessionID (now available post-upstream-sync)
4. **Graceful degradation** - If `bd` or `orch` commands fail, show minimal HUD or skip

## Implementation Notes

**File to create:**
- `~/.config/opencode/plugin/orch-hud.ts`

**HUD element priority:**

| Priority | Element | Role | Source |
|----------|---------|------|--------|
| 1 | Active agents + status | Orchestrator | `orch frontier` |
| 2 | Beads issue context | Worker | `.beads_id` + `bd show` |
| 3 | Current phase | Worker | bd comments |
| 4 | Blocked items count | Orchestrator | `bd list --blocked` |
| 5 | Active constraints | Both | `kb constraints` |
| 6 | Usage/budget warning | Both | `orch usage` |
| 7 | Completion signals | Orchestrator | Agent state changes |
| 8 | Triage:ready count | Orchestrator | `bd list -l triage:ready` |

**Implementation sequence:**

**Phase 1: Core HUD Plugin**
- Create `orch-hud.ts` with role detection (ORCH_WORKER env, workspace paths)
- Implement `experimental.chat.system.transform` handler
- Build role-appropriate HUD markdown
- Inject into system array

**Phase 2: Role-Specific Content**
- Orchestrator: Gather spawn state via `orch frontier`
- Worker: Gather beads context via `bd show`
- Format as markdown sections
- Limit to top 5 items per category

**Phase 3: Testing**
- Spawn orchestrator, verify HUD appears
- Spawn worker, verify beads context shown
- Check token count (<500)
- Verify no noticeable latency

**Phase 4: Compaction Integration (Optional)**
- Add `experimental.session.compacting` handler
- Preserve critical HUD elements through compaction
- Re-gather dynamic state after compaction

**Things to watch out for:**
- Performance: Running `bd` and `orch` commands per-turn adds latency (consider caching)
- Token overhead: Must stay under 500 tokens
- State staleness: Acceptable with per-turn updates
- Plugin load order: Use single file to avoid conflicts

**Success Criteria:**
- Orchestrator sees spawn state without running `orch frontier` manually
- Workers see beads context without checking `.beads_id` file
- HUD appears on every turn
- Token overhead <500 tokens
- No noticeable latency increase

## References

**Investigation:**
- `.kb/investigations/2026-01-27-inv-design-exploration-dynamic-hud-pattern.md` - HUD design exploration
- `.kb/investigations/2026-01-25-inv-investigate-claude-code-native-swarm.md` - Native swarm XML injection mechanism

**Files:**
- `opencode/packages/opencode/src/session/llm.ts:73` - system.transform hook trigger
- `opencode/packages/plugin/src/index.ts:213-228` - Hook type definitions
- `~/.config/opencode/plugin/session-compaction.ts` - Production plugin example
- `.opencode/plugin/slow-find-warn.ts` - Per-tool injection example

**Models:**
- `.kb/models/context-injection.md` - Context injection architecture
- `.kb/models/opencode-session-lifecycle.md` - Session lifecycle

**Guides:**
- `.kb/guides/opencode-plugins.md` - Plugin system reference
- `.kb/guides/opencode.md` - OpenCode integration guide

**Principles:**
- Plugin-First, Fork-Later - Minimize fork maintenance
- Infrastructure Over Instruction - Automatic context delivery vs manual queries
- Surfacing Over Browsing - Surface decision points automatically
