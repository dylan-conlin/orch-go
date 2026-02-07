---
linked_issues:
  - orch-go-4zc2s
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode injects ~4KB at session start vs Claude Code's ~25KB; orchestrator skill loaded via instructions array (not direct injection), with explicit worker detection (ORCH_WORKER env var).

**Evidence:** Examined 8 OpenCode plugins across global (~/.config/opencode/plugin/) and project (.opencode/plugin/) locations; only session-resume.js injects content at session.created event; session-context.ts adds orchestrator skill to config.instructions (file reference, not immediate content).

**Knowledge:** OpenCode's architecture separates "instructions" (file references loaded at config time) from "injection" (content pushed via client.session.prompt); this is fundamentally different from Claude Code's hook-based immediate injection.

**Next:** Update epic with findings; consider harmonizing Claude Code hooks to use similar role-detection pattern (CLAUDE_CONTEXT exists but underutilized).

**Promote to Decision:** recommend-no - This is audit findings, not an architectural decision. Epic should synthesize findings from all probes.

---

# Investigation: Audit OpenCode Session Start Injection

**Question:** What plugins/hooks run when OpenCode spawns an agent, and how does this compare to Claude Code's SessionStart hooks?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-inv-audit-opencode-session-16jan-df4b
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Epic:** `.orch/epics/context-injection-architecture.md` (Probe 2)

---

## Findings

### Finding 1: OpenCode Plugin Discovery Architecture

**Evidence:** OpenCode auto-discovers plugins via glob pattern `{plugin,plugins}/*.{ts,js}` in three locations:

| Location | Scope | Example |
|----------|-------|---------|
| `~/.config/opencode/plugin/` | Global | session-resume.js, guarded-files.ts |
| `~/.opencode/plugin/` | Global | (none currently) |
| `.opencode/plugin/` in project | Project-specific | session-context.ts, agentlog-inject.ts |

Plugins are concatenated via `mergeConfigConcatArrays()` - global and project plugins both load.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/config/config.ts:320-333` (loadPlugin function), lines 91-134 (plugin aggregation)

**Significance:** Unlike Claude Code hooks (registered in settings.json), OpenCode plugins are auto-discovered from directories. No explicit registration needed.

---

### Finding 2: Plugin Event Types and Trigger Timing

**Evidence:** OpenCode plugins can hook into multiple events. Categorized by when they fire:

**Session Start (session.created):**
| Plugin | Location | Purpose |
|--------|----------|---------|
| session-resume.js | Global | Inject session handoff (~4KB) |
| agentlog-inject.ts | orch-cli project | Inject recent errors (conditional) |
| usage-warning.ts | orch-cli project | Inject usage warning (conditional) |

**Config Phase (before session):**
| Plugin | Location | Purpose |
|--------|----------|---------|
| session-context.ts | orch-cli project | Add orchestrator skill to instructions |

**Other Events (NOT session start):**
| Plugin | Event | Purpose |
|--------|-------|---------|
| guarded-files.ts | tool.execute.before (Edit) | Protocol warning |
| friction-capture.ts | session.idle | Friction capture prompt |
| session-compaction.ts | experimental.session.compacting | Context preservation |
| bd-close-gate.ts | tool.execute.before (Bash) | Block bd close |

**Source:** Plugin source code analysis at `~/.config/opencode/plugin/` and `~/Documents/personal/orch-cli/.opencode/plugin/`

**Significance:** Only 3-4 plugins fire at session start vs Claude Code's 7 hooks. Most OpenCode plugins fire on tool/idle/compaction events.

---

### Finding 3: Session Start Injection Size Matrix

**Evidence:** Tested actual injection content:

| Plugin | Trigger | Condition | Output Size | Est. Tokens |
|--------|---------|-----------|-------------|-------------|
| session-resume.js | session.created | Handoff exists + NOT worker | ~4,120 bytes | ~1,030 |
| agentlog-inject.ts | session.created | .agentlog/ has errors | Variable (~200-2000) | ~50-500 |
| usage-warning.ts | session.created | Usage >80% | ~200 bytes | ~50 |
| session-context.ts | config | Orch project + NOT worker | 86,451 bytes (skill file) | ~21,613 |

**Key insight:** session-context.ts doesn't inject at session.created - it adds the skill **file path** to `config.instructions` during the config phase. This is loaded as a file reference, similar to how `~/.claude/CLAUDE.md` works in opencode.jsonc's "instructions" array.

**Source:**
- `orch session resume --for-injection | wc -c` = 4,120 bytes
- `wc -c ~/.claude/skills/meta/orchestrator/SKILL.md` = 86,451 bytes
- Plugin source code inspection

**Significance:** Direct session-start injection is ~4KB max (without errors/warnings). The orchestrator skill (~86KB) loads via instructions mechanism, not direct injection.

---

### Finding 4: Worker Detection Mechanisms

**Evidence:** Two worker detection patterns exist:

**OpenCode (session-context.ts:92-95):**
```typescript
// Check if this is a worker agent (per-session check)
if (process.env.ORCH_WORKER) {
  console.log("[session-context] Skipping - worker agent detected")
  return
}
```

**OpenCode (session-resume.js:55-83):**
```javascript
// Check for SPAWN_CONTEXT.md in workspace subdirectory (reliable indicator for workers)
const spawnContextPath = path.join(workspaceDir, entry.name, 'SPAWN_CONTEXT.md')
// ... if found, skip injection
```

**Comparison with Claude Code (load-orchestration-context.py:436-447):**
```python
def is_spawned_agent():
    ctx = os.environ.get('CLAUDE_CONTEXT', '')
    return ctx in ('worker', 'orchestrator', 'meta-orchestrator')
```

**Source:** Plugin source files compared with Claude Code hook at `~/.orch/hooks/load-orchestration-context.py`

**Significance:** OpenCode uses ORCH_WORKER env var (simpler) and SPAWN_CONTEXT.md presence (file-based). Claude Code uses CLAUDE_CONTEXT env var with more values. Both achieve role-aware injection but with different mechanisms.

---

### Finding 5: Comparison Table - OpenCode vs Claude Code Session Start

**Evidence:** Full comparison matrix:

| Aspect | Claude Code | OpenCode |
|--------|-------------|----------|
| **Hook/Plugin Count** | 7 SessionStart hooks | 3-4 session.created plugins |
| **Worst-Case Injection** | ~25K tokens | ~1K tokens (direct) |
| **Orchestrator Skill** | Hook injection (93KB) | Instructions array (86KB file ref) |
| **Session Resume** | Always (~4KB) | Conditional on worker status (~4KB) |
| **Beads Guidance** | Always (~740 tokens) | Not injected (in SPAWN_CONTEXT) |
| **Worker Detection** | CLAUDE_CONTEXT (partial use) | ORCH_WORKER + SPAWN_CONTEXT.md |
| **Error Context** | Hook (~0 usually) | Plugin (~0 usually) |
| **Usage Warnings** | Hook (~0 usually) | Plugin (~0 usually) |

**Hook-to-Plugin Mapping:**

| Claude Code Hook | OpenCode Equivalent | Notes |
|------------------|---------------------|-------|
| session-start.sh | session-resume.js | Both inject handoff |
| load-orchestration-context.py | session-context.ts | Orchestrator skill loading |
| bd prime | (none) | OpenCode relies on SPAWN_CONTEXT |
| inject-orch-patterns.sh | (none) | Not ported |
| agentlog-inject.sh | agentlog-inject.ts | Same purpose |
| usage-warning.sh | usage-warning.ts | Same purpose |
| reflect-suggestions-hook.py | (none) | Not ported |

**Source:** Cross-comparison of all hooks and plugins

**Significance:** OpenCode's architecture is significantly leaner at session start. The key difference is that the orchestrator skill is loaded via **instructions** (config-time file loading) rather than **injection** (runtime content push).

---

### Finding 6: The Instructions vs Injection Distinction

**Evidence:** OpenCode has two mechanisms for providing context:

1. **Instructions (config.instructions array):**
   - File paths loaded at config initialization
   - Content read from files and included in system context
   - Similar to `"instructions": ["~/.claude/CLAUDE.md"]` in opencode.jsonc
   - The session-context plugin adds orchestrator skill this way

2. **Injection (client.session.prompt with noReply: true):**
   - Content pushed directly into session after creation
   - Appears as a message in the conversation
   - Used by session-resume, agentlog-inject, usage-warning

```typescript
// Instructions (session-context.ts:103-106)
config.instructions.push(skillPath)  // File reference

// Injection (session-resume.js:111-120)
await client.session.prompt({
  path: { id: sessionID },
  body: { noReply: true, parts: [{ type: 'text', text: content }] }
})  // Direct content
```

**Source:** Plugin source code comparison

**Significance:** This distinction explains why OpenCode "feels" lighter - the orchestrator skill (86KB) is loaded as a file reference in config, not injected at runtime. Both approaches ultimately add to context, but the architecture is different.

---

## Synthesis

**Key Insights:**

1. **OpenCode is architecturally leaner** - Only ~4KB injected at session.created (session resume), compared to Claude Code's potential ~25KB. The orchestrator skill is loaded via config.instructions, not injected.

2. **Role detection is cleaner in OpenCode** - ORCH_WORKER env var and SPAWN_CONTEXT.md file detection are used consistently. Claude Code's CLAUDE_CONTEXT env var exists but is underutilized (only 1 of 7 hooks checks it).

3. **Project-level plugins enable customization** - orch-cli's `.opencode/plugin/` directory adds project-specific functionality (session-context, agentlog, usage warnings). This pattern doesn't exist in Claude Code (all hooks are global).

4. **Plugin timing model is richer** - OpenCode plugins can fire on multiple events (session.created, session.idle, tool.execute.before, config, experimental.session.compacting). Claude Code hooks only have SessionStart/PreToolUse/PostToolUse.

**Answer to Investigation Question:**

When OpenCode spawns an agent:

1. **Config Phase:** session-context.ts adds orchestrator skill path to instructions (if orch project and NOT worker)
2. **Session Created:** session-resume.js injects ~4KB handoff (if exists and NOT worker)
3. **Session Created:** agentlog-inject.ts injects errors (if .agentlog/ has errors)
4. **Session Created:** usage-warning.ts injects warning (if usage >80%)

Total direct injection: **~4-5KB worst-case** (without orchestrator skill file loading)

Compared to Claude Code's **~25KB worst-case** (with orchestrator skill direct injection).

The key architectural difference is that OpenCode loads the orchestrator skill via **instructions** (file reference) rather than **injection** (direct content push). Both approaches add to context budget, but the timing and mechanism differ.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin file locations verified via `ls -la` commands
- ✅ Plugin event types confirmed via source code reading
- ✅ Session resume injection size measured: 4,120 bytes
- ✅ Orchestrator skill size measured: 86,451 bytes
- ✅ Worker detection mechanisms verified in plugin source

**What's untested:**

- ⚠️ Actual token counts (estimated at ~4 chars/token)
- ⚠️ Plugin loading order/timing at runtime
- ⚠️ Whether instructions and injection are additive or have different context budgets
- ⚠️ Performance impact of config-time vs runtime loading

**What would change this:**

- Finding would be wrong if OpenCode has hidden plugins not in standard locations
- Finding would be wrong if instructions array has size limits or deduplication
- Comparison would change if Claude Code hooks were modified to use CLAUDE_CONTEXT consistently

---

## Implementation Recommendations

### Recommended Approach: Harmonize Worker Detection

**Both systems can benefit from consistent role detection:**

1. **Claude Code:** Update hooks to check CLAUDE_CONTEXT consistently (session-start.sh currently doesn't)
2. **OpenCode:** Current approach is cleaner; no changes needed
3. **Cross-system:** Consider standardizing on a single env var (e.g., ORCH_ROLE with values: worker/orchestrator/manual)

**Trade-offs accepted:**
- Maintaining two parallel systems (Claude Code + OpenCode) adds complexity
- Different detection mechanisms (env var vs file presence) have different reliability characteristics

### Alternative Approaches Considered

**Option B: Migrate all injection to spawn context**
- **Pros:** Single source of truth for spawned agents
- **Cons:** Loses dynamic state (errors, usage) that may appear after spawn context generation
- **When to use instead:** If session start latency becomes a problem

**Option C: Disable all hooks/plugins for spawned workers**
- **Pros:** Cleanest separation
- **Cons:** Loses useful conditional context (error warnings, usage warnings)
- **When to use instead:** If context budget is extremely tight

---

## References

**Files Examined:**
- `~/.config/opencode/plugin/session-resume.js` - Global session resume plugin
- `~/.config/opencode/plugin/guarded-files.ts` - Edit-time protocol warning
- `~/.config/opencode/plugin/session-compaction.ts` - Compaction context preservation
- `~/.config/opencode/plugin/friction-capture.ts` - Idle-time friction capture
- `~/Documents/personal/orch-cli/.opencode/plugin/session-context.ts` - Orchestrator skill loading
- `~/Documents/personal/orch-cli/.opencode/plugin/agentlog-inject.ts` - Error context injection
- `~/Documents/personal/orch-cli/.opencode/plugin/usage-warning.ts` - Usage warning injection
- `~/Documents/personal/orch-cli/.opencode/plugin/bd-close-gate.ts` - bd close blocking
- `~/Documents/personal/opencode/packages/opencode/src/config/config.ts` - Plugin loading mechanism
- `~/Documents/personal/opencode/packages/opencode/src/plugin/index.ts` - Plugin execution

**Commands Run:**
```bash
# List global plugins
ls -la ~/.config/opencode/plugin/

# Measure injection sizes
orch session resume --for-injection | wc -c
wc -c ~/.claude/skills/meta/orchestrator/SKILL.md

# Find plugin loading mechanism
grep -r "plugin.*load" ~/Documents/personal/opencode/
```

**Related Artifacts:**
- **Epic:** `.orch/epics/context-injection-architecture.md` - Parent epic
- **Investigation:** `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md` - Claude Code hook audit (Probe 1)

---

## Self-Review

- [x] Real test performed (file size measurements, source code verification)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2026-01-16 13:28:** Investigation started
- Initial question: What does OpenCode inject at session start?
- Context: Probe 2 for context-injection-architecture epic

**2026-01-16 13:35:** Plugin discovery
- Found 4 global plugins at ~/.config/opencode/plugin/
- Found 4 project plugins at ~/Documents/personal/orch-cli/.opencode/plugin/

**2026-01-16 13:45:** Plugin categorization complete
- Identified session.created vs other event plugins
- Discovered instructions vs injection distinction

**2026-01-16 13:55:** Comparison table complete
- OpenCode: ~4KB direct injection at session start
- Claude Code: ~25KB injection at session start
- Key difference: orchestrator skill loaded via instructions vs injection

**2026-01-16 14:00:** Investigation completed
- Status: Complete
- Key outcome: OpenCode is architecturally leaner with ~4KB session start injection vs Claude Code's ~25KB
