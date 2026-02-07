<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created comprehensive guide for OpenCode plugin system as orchestration-execution bridge, synthesizing 8 production plugins and prior investigation into authoritative reference.

**Evidence:** Analyzed all 8 plugins in ~/.config/opencode/plugin/, plugin SDK types in opencode/packages/plugin/src/index.ts, and investigation findings - identified three core patterns (Gates, Context Injection, Observation) with production examples.

**Knowledge:** Plugin patterns map directly to principle mechanization: Gates for hard rules (throw to block), Context Injection for soft guidance (noReply: true), Observation for pattern detection. Worker/orchestrator detection uses three signals: ORCH_WORKER env, SPAWN_CONTEXT.md, path pattern.

**Next:** Close - guide created at .kb/guides/opencode-plugins.md, ready for use by agents building new plugins.

**Promote to Decision:** recommend-no - Synthesis work producing guide, not architectural choice.

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

# Investigation: Create OpenCode Plugins Guide

**Question:** How to synthesize existing investigation and 8 production plugins into an authoritative guide for the OpenCode plugin system?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Worker agent (orch-go-s90si)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Three distinct plugin patterns emerged from production analysis

**Evidence:** Analyzed 8 production plugins, identified clear pattern separation:
- **Gates (blocking):** `bd-close-gate.ts` uses `tool.execute.before` + throw to block commands
- **Context Injection:** `guarded-files.ts`, `friction-capture.ts`, `usage-warning.ts` use `client.session.prompt({ noReply: true })` 
- **Observation:** `action-log.ts` tracks outcomes without blocking using before/after correlation

**Source:** `~/.config/opencode/plugin/*.ts` (8 files), investigation `2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md`

**Significance:** Clean pattern taxonomy enables agents to choose correct approach when building new plugins.

---

### Finding 2: Hook data differs between before and after phases

**Evidence:** 
- `tool.execute.before` receives: `input: { tool, sessionID, callID }`, `output: { args }`
- `tool.execute.after` receives: `input: { tool, sessionID, callID }`, `output: { title, output, metadata }`
- Production plugins use Map with `callID` as key to correlate: store args in before, retrieve in after

**Source:** `opencode/packages/plugin/src/index.ts:174-185`, `~/.config/opencode/plugin/action-log.ts:307-349`

**Significance:** Common pitfall - agents expect args in after hook. Guide explicitly documents this pattern.

---

### Finding 3: Worker vs orchestrator detection has three reliable signals

**Evidence:** `orchestrator-session.ts` shows detection hierarchy:
1. `process.env.ORCH_WORKER === "1"` (set by orch spawn)
2. `SPAWN_CONTEXT.md` exists in working directory
3. Path contains `.orch/workspace/`

**Source:** `~/.config/opencode/plugin/orchestrator-session.ts:76-100`

**Significance:** Consistent detection pattern across plugins enables different behavior for workers vs orchestrators.

---

## Synthesis

**Key Insights:**

1. **Plugin patterns map to principle mechanization modes** - Gates enforce hard rules (must not violate), Context Injection provides soft guidance (should consider), Observation enables learning (track for analysis). This taxonomy helps agents choose the right pattern.

2. **State management is non-trivial** - Plugins load multiple times (global + project), hooks split data across phases. Production plugins solve this with callID correlation and file-based deduplication locks.

3. **Guide structure follows opencode.md style** - Reference guide format with architecture diagram, quick reference tables, and "What Lives Where" sections matches existing guide style.

**Answer to Investigation Question:**

Guide created at `.kb/guides/opencode-plugins.md` covering:
- Three plugin patterns with production examples
- Hook selection guide (20+ hooks mapped to use cases)  
- Worker vs orchestrator detection (three signals)
- State management across sessions (callID correlation, file locks)
- Testing approaches (debug logging, load verification)
- Six common pitfalls with solutions

The guide synthesizes the investigation's 6 findings with analysis of all 8 production plugins, providing single authoritative reference for plugin development.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 8 production plugins analyzed (verified: read each file completely)
- ✅ Plugin SDK types documented from source (verified: read index.ts)
- ✅ Hook data structures documented (verified: traced through SDK types)

**What's untested:**

- ⚠️ Guide completeness for novel use cases (not validated against new plugin creation)
- ⚠️ Testing approaches section (documented from patterns, not personally tested)
- ⚠️ Some experimental hooks may have changed (only verified against current source)

**What would change this:**

- Finding would be wrong if OpenCode plugin API changed significantly
- Guide would be incomplete if production plugins use undocumented patterns
- Pitfalls section would need updates if new failure modes emerge

---

## Implementation Recommendations

N/A - This was synthesis work, not investigation of a problem requiring implementation.

**Deliverable:** `.kb/guides/opencode-plugins.md` - Guide created covering all required topics.

---

## References

**Files Examined:**
- `~/.config/opencode/plugin/*.ts` (8 plugins) - Production plugins for pattern analysis
- `~/.config/opencode/lib/*.ts` (3 files) - Shared helper libraries
- `opencode/packages/plugin/src/index.ts` - Plugin SDK type definitions
- `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Prior investigation
- `.kb/guides/opencode.md` - Reference for guide style

**Commands Run:**
```bash
# List plugins
glob *.ts ~/.config/opencode/plugin

# Create investigation file
kb create investigation create-kb-guides-opencode-plugins
```

**External Documentation:**
- https://opencode.ai/docs/plugins - Official plugin documentation

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Source investigation being synthesized
- **Guide:** `.kb/guides/opencode.md` - Companion guide covering HTTP API

---

## Investigation History

**2026-01-08 09:00:** Investigation started
- Initial question: Synthesize investigation + 8 plugins into authoritative guide
- Context: Part of kb guide creation for plugin system

**2026-01-08 09:15:** Analysis complete
- Identified three core patterns: Gates, Context Injection, Observation
- Mapped hook selection to use cases
- Documented common pitfalls from production plugins

**2026-01-08 09:30:** Investigation completed
- Status: Complete
- Key outcome: Guide created at `.kb/guides/opencode-plugins.md`
