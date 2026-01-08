# Session Synthesis

**Agent:** og-feat-investigate-opencode-plugin-08jan-c3e5
**Issue:** orch-go-n5h2g.1
**Duration:** 2026-01-08 ~10:00 → 2026-01-08 ~11:30
**Outcome:** success

---

## TLDR

Investigated OpenCode's plugin system to understand its full capabilities for principle mechanization. Found a mature, powerful plugin API with 20+ hooks spanning the full session lifecycle. Designed 3 high-value plugins (Coherence Detector, Provenance Tracker, Task Spawn Analyzer) that can observe agent behavior and enforce principles through context injection and blocking gates.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Comprehensive investigation of OpenCode plugin system with hook analysis, existing plugin patterns, ecosystem review, and 3 plugin designs

### Files Modified
- None

### Commits
- (pending commit of investigation file)

---

## Evidence (What Was Observed)

- Plugin system provides full SDK client access via `client` parameter (verified: `opencode/packages/plugin/src/index.ts:26-33`)
- 20+ hook types available covering session lifecycle, tool execution, messages, permissions (verified: `opencode/packages/plugin/src/index.ts:146-216`)
- Hooks run sequentially via `await fn(input, output)` - no parallel execution (verified: `opencode/packages/opencode/src/plugin/index.ts:68-83`)
- `tool.execute.before` can block operations by throwing (verified: `bd-close-gate.ts` uses this pattern)
- Context injection works via `client.session.prompt({ noReply: true })` (verified: 5 existing plugins use this)
- Dylan already has 7 functioning plugins demonstrating all key patterns

### Tools Examined
- `opencode/packages/opencode/src/plugin/index.ts` - Core plugin loader and trigger
- `opencode/packages/plugin/src/index.ts` - Plugin types (Hooks, PluginInput)
- `opencode/packages/web/src/content/docs/plugins.mdx` - Official documentation
- `~/.config/opencode/plugin/*.ts` - 7 existing plugins

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Complete hook inventory, plugin patterns, and 3 plugin designs

### Decisions Made
- Observation-first approach: Build observation plugins before gates to understand behavior patterns
- Three mechanization patterns: (1) blocking gates via throw, (2) context injection via noReply prompt, (3) observation logging

### Constraints Discovered
- Hooks run sequentially - heavy plugins could slow operations
- No timeout protection - plugins must be well-behaved
- Worker detection requires checking multiple signals (ORCH_WORKER env, SPAWN_CONTEXT.md, path)

### Externalized via `kn`
- None yet - recommend post-completion: `kn decide "observation-first for plugin development" --reason "generates data to inform which gates are needed"`

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, plugin designs ready for implementation)

### If Close
- [x] All deliverables complete (investigation file with hook analysis, ecosystem review, 3 plugin designs)
- [x] Investigation file has findings, synthesis, implementation recommendations
- [x] Ready for `orch complete orch-go-n5h2g.1`

### Recommended Follow-up Work
**Issue:** Implement Coherence Detector Plugin
**Skill:** feature-impl
**Context:**
```
First of 3 observation plugins from investigation orch-go-n5h2g.1. 
Uses tool.execute.after hook on edit tool to track file edit counts per session.
Injects context warning when same file edited 3+ times.
See: .kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does compaction handle injected context from plugins? (noReply:true messages)
- What's the memory footprint of tracking state in plugins across many sessions?
- Could `opencode-skillful` pattern reduce skill loading overhead?

**Areas worth exploring further:**
- Performance benchmarking of multiple concurrent plugins
- Error recovery patterns when plugins fail mid-session

**What remains unclear:**
- Whether experimental hooks will remain stable in future versions
- Community plugin quality/maintenance status

---

## Session Metadata

**Skill:** feature-impl (investigation phase)
**Model:** Claude (via OpenCode)
**Workspace:** `.orch/workspace/og-feat-investigate-opencode-plugin-08jan-c3e5/`
**Investigation:** `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md`
**Beads:** `bd show orch-go-n5h2g.1`
