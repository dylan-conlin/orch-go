<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Consolidated session-context.js and orch-session-autostart.ts into single orchestrator-session.ts plugin with shared worker detection.

**Evidence:** Created ~/.config/opencode/plugin/orchestrator-session.ts, deleted both old plugins, verified file structure matches existing plugins.

**Knowledge:** Both plugins had duplicated isWorker() logic; combining them simplifies maintenance and ensures consistent behavior.

**Next:** Close - implementation complete.

---

# Investigation: Consolidate Session Context Plugins

**Question:** How to consolidate session-context.js and orch-session-autostart.ts into a unified orchestrator-session.ts plugin?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage -->
**Supersedes:** .kb/investigations/2025-12-29-inv-create-opencode-plugin-orch-session.md (orch-session-autostart.ts now part of consolidated plugin)

---

## Findings

### Finding 1: session-context.js responsibilities

**Evidence:** The plugin uses config hook to inject orchestrator skill into instructions. It checks ORCH_WORKER env var to skip workers.

**Source:** ~/.config/opencode/plugin/session-context.js (69 lines)

**Significance:** This functionality needs to be preserved in the consolidated plugin. Uses `config` hook.

---

### Finding 2: orch-session-autostart.ts responsibilities

**Evidence:** The plugin uses session.created event to auto-run `orch session start` for orchestrators. Has more comprehensive worker detection (env var + SPAWN_CONTEXT.md + workspace path check).

**Source:** ~/.config/opencode/plugin/orch-session-autostart.ts (129 lines)

**Significance:** This functionality needs to be preserved. The more robust isWorker() logic should be the shared implementation.

---

### Finding 3: Duplicated worker detection logic

**Evidence:** session-context.js only checks `ORCH_WORKER` env var. orch-session-autostart.ts checks env var + SPAWN_CONTEXT.md + .orch/workspace/ path. Both need to know if running as worker.

**Source:** session-context.js:52, orch-session-autostart.ts:41-65

**Significance:** Consolidation allows single shared isWorker() function with comprehensive detection.

---

## Implementation

Created unified `orchestrator-session.ts` with:
1. Shared `isWorker()` function with all three detection methods
2. `config` hook to inject orchestrator skill (from session-context.js)
3. `event` hook for session.created to run `orch session start` (from orch-session-autostart.ts)
4. Consistent logging with `[orchestrator-session]` prefix

Deleted both old plugins:
- ~/.config/opencode/plugin/session-context.js
- ~/.config/opencode/plugin/orch-session-autostart.ts

---

## References

**Files Examined:**
- ~/.config/opencode/plugin/session-context.js - Original config hook plugin
- ~/.config/opencode/plugin/orch-session-autostart.ts - Original event hook plugin
- ~/.opencode/plugin/action-log.ts - Reference for plugin structure

**Files Created:**
- ~/.config/opencode/plugin/orchestrator-session.ts - Unified plugin

**Files Deleted:**
- ~/.config/opencode/plugin/session-context.js
- ~/.config/opencode/plugin/orch-session-autostart.ts

---

## Investigation History

**2025-12-29 13:10:** Investigation started
- Initial question: Consolidate two OpenCode plugins with duplicated logic
- Context: Both plugins target orchestrator sessions with separate worker detection

**2025-12-29 13:14:** Implementation completed
- Created unified orchestrator-session.ts
- Deleted old plugins
- Verified directory state

**2025-12-29 13:16:** Optimization applied
- Moved isWorker() check from each hook to plugin init
- Worker detection now cached, avoiding duplicate async checks
- Aligns with SPAWN_CONTEXT proposed solution pattern
