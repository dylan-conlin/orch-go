<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator skill references Claude Code hooks (SessionStart, PreToolUse, PostToolUse, ~/.claude/hooks/) which don't exist in OpenCode; OpenCode uses `.opencode/plugin/` with JS/TS modules and events like `session.created`, `tool.execute.before`, `tool.execute.after`.

**Evidence:** Audited SKILL.md lines 377, 678-679, 1083-1098, 1105-1107, 1486-1491; confirmed OpenCode plugin docs at opencode.ai/docs/plugins; verified existing plugin at .opencode/plugin/action-log.ts uses correct OpenCode patterns.

**Knowledge:** The mapping is: SessionStart → session.created event, PreToolUse → tool.execute.before, PostToolUse → tool.execute.after; location changes from ~/.claude/hooks/ to .opencode/plugin/ (project) or ~/.config/opencode/plugin/ (global).

**Next:** Replace all Claude Code hook references in SKILL.md with OpenCode plugin equivalents.

---

# Investigation: Fix Orchestrator Skill Replace Claude

**Question:** What Claude Code hook references exist in the orchestrator skill and what are their OpenCode plugin equivalents?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Claude Code Hook References in Knowledge Placement Guide

**Evidence:** Lines 1083-1085 and 1093-1098 reference Claude Code hook patterns:
- "Context needed at session start | SessionStart hook"
- "Automated enforcement/blocking | PreToolUse hook"  
- "Automated action after tool runs | PostToolUse hook"
- "I want agents to always know..." → SessionStart hook
- "I want to block agents from..." → PreToolUse hook
- "After agents do X, also do Y..." → PostToolUse hook

**Source:** ~/.claude/skills/meta/orchestrator/SKILL.md:1083-1098

**Significance:** These are the primary references causing agents to implement wrong platform hooks. The guidance table and heuristics use Claude Code terminology.

---

### Finding 2: Hook References in Other Sections

**Evidence:** Additional references found:
- Line 377: "SessionStart hook warns at >80%"
- Lines 678-679: "SessionStart hook, skill update, kn entry"
- Lines 1105-1107: Decision tree mentions "Hook (SessionStart/PreToolUse/PostToolUse)"
- Lines 1486-1489: Session Reflection mentions "New hook needed?"

**Source:** ~/.claude/skills/meta/orchestrator/SKILL.md various lines

**Significance:** Scattered references throughout the skill reinforce Claude Code patterns.

---

### Finding 3: OpenCode Plugin System Mapping

**Evidence:** From OpenCode docs (opencode.ai/docs/plugins) and existing action-log.ts plugin:

| Claude Code | OpenCode Plugin | Location |
|-------------|-----------------|----------|
| SessionStart hook | `session.created` event | `.opencode/plugin/` or `~/.config/opencode/plugin/` |
| PreToolUse hook | `tool.execute.before` event | Same |
| PostToolUse hook | `tool.execute.after` event | Same |
| ~/.claude/hooks/ | `.opencode/plugin/` (project) or `~/.config/opencode/plugin/` (global) | N/A |

**Source:** https://opencode.ai/docs/plugins, /Users/dylanconlin/Documents/personal/orch-go/.opencode/plugin/action-log.ts

**Significance:** Direct 1:1 mapping exists; changes are terminology updates not functionality changes.

---

## Synthesis

**Key Insights:**

1. **Terminology-only change** - The concepts map 1:1; we're updating naming not functionality
2. **Event-based vs file-based** - OpenCode uses JS/TS modules with event handlers vs Claude Code's bash scripts
3. **Location change** - Plugins live in `.opencode/plugin/` not `~/.claude/hooks/`

**Answer to Investigation Question:**

All Claude Code hook references can be directly replaced with OpenCode plugin equivalents:
- SessionStart → session.created event in plugin
- PreToolUse → tool.execute.before event
- PostToolUse → tool.execute.after event
- ~/.claude/hooks/ → .opencode/plugin/ (project) or ~/.config/opencode/plugin/ (global)

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode plugin system works with event handlers (verified: action-log.ts uses tool.execute.before/after)
- ✅ session.created event exists (verified: OpenCode docs list it under Session Events)
- ✅ Plugins can be project-local or global (verified: docs confirm both locations)

**What's untested:**

- ⚠️ Whether all SessionStart use cases translate to session.created (likely yes but not exhaustively verified)

**What would change this:**

- If OpenCode lacks an event that Claude Code hooks provided (unlikely given event list)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Direct terminology replacement** - Replace all Claude Code hook references with OpenCode plugin equivalents throughout SKILL.md

**Why this approach:**
- 1:1 mapping makes changes clear and safe
- Prevents agents from implementing wrong platform
- Maintains the same guidance intent with correct platform terminology

**Implementation sequence:**
1. Replace "SessionStart hook" → "session.created plugin event"
2. Replace "PreToolUse hook" → "tool.execute.before plugin event"
3. Replace "PostToolUse hook" → "tool.execute.after plugin event"
4. Replace "~/.claude/hooks/" → ".opencode/plugin/ (project) or ~/.config/opencode/plugin/ (global)"
5. Add note about OpenCode plugin structure (JS/TS modules with event handlers)

---

## References

**Files Examined:**
- ~/.claude/skills/meta/orchestrator/SKILL.md - Full audit for hook references
- /Users/dylanconlin/Documents/personal/orch-go/.opencode/plugin/action-log.ts - Existing OpenCode plugin example

**External Documentation:**
- https://opencode.ai/docs/plugins - OpenCode plugin documentation

**Related Artifacts:**
- Prior constraint in SPAWN_CONTEXT.md: "Orch uses OpenCode, not Claude Code. Hook system is different"

---

## Investigation History

**2025-12-29 12:30:** Investigation started
- Initial question: What Claude Code hook references need replacing in orchestrator skill?
- Context: Agent implemented wrong platform's hooks because skill references Claude Code patterns

**2025-12-29 12:45:** Investigation completed
- Status: Complete
- Key outcome: 6 locations with hook references identified; all map directly to OpenCode plugin events
