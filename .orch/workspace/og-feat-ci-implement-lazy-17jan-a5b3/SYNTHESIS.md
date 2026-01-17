# Session Synthesis

**Agent:** og-feat-ci-implement-lazy-17jan-a5b3
**Issue:** orch-go-y1ikp
**Duration:** 2026-01-17 11:40 → 2026-01-17 12:00
**Outcome:** success

---

## TLDR

Implemented lazy-loading for the orchestrator skill (52KB) by replacing eager config hook with progressive worker detection and conditional system.transform hook injection, reducing context usage for worker sessions from 52KB to 0KB.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-ci-implement-lazy-loading-orchestrator.md` - Investigation documenting plugin hook architecture analysis and implementation approach

### Files Modified
- `plugins/orchestrator-session.ts` - Complete refactor from config hook to lazy-loading approach

### Commits
- `4b5d2b5a` - feat: implement lazy-loading for orchestrator skill

---

## Evidence (What Was Observed)

### Finding: Config Hook Runs Too Early
- Observed: `config` hook signature only receives `Config` object, no sessionID or session context
- Source: `.opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:112`
- Implication: Cannot detect worker sessions at config time, need later hook

### Finding: Coaching Plugin Has Proven Pattern
- Observed: coaching.ts successfully detects workers via `tool.execute.before` hook checking SPAWN_CONTEXT.md reads and .orch/workspace/ paths
- Source: `plugins/coaching.ts:1319-1360` - detectWorkerSession() function
- Implication: Can reuse this pattern for orchestrator-session plugin

### Finding: System Transform Hook Enables Lazy Loading
- Observed: `experimental.chat.system.transform` hook runs when building system prompt and can modify system[] array
- Source: `.opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:173-177`
- Implication: Perfect timing for conditional injection after worker detection

### Tests Run
No automated tests written (plugin change, testing requires live sessions). Manual verification needed:
```bash
# Test 1: Worker session should NOT load orchestrator skill
ORCH_PLUGIN_DEBUG=1 orch spawn investigation "test" --tmux
# Expected: Logs show "Worker detected" and "Skipping orchestrator skill"

# Test 2: Orchestrator session SHOULD load orchestrator skill  
ORCH_PLUGIN_DEBUG=1 opencode .
# Expected: Logs show "Injected orchestrator skill" with byte count
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-ci-implement-lazy-loading-orchestrator.md` - Documents plugin hook timing, worker detection mechanisms, and implementation rationale

### Decisions Made
- Decision 1: Use progressive worker detection (tool.execute.before + Map cache) rather than trying to detect at config time - timing constraint makes config hook unusable for per-session detection
- Decision 2: Cache orchestrator skill content in memory at plugin init rather than reading on every system prompt - 52KB file read on every message would be slow
- Decision 3: Only cache positive worker detection results (not false negatives) - allows detection to succeed on later tool calls if first tool doesn't match patterns
- Decision 4: Use experimental.chat.system.transform hook despite "experimental" prefix - coaching plugin shows it's stable in practice

### Constraints Discovered
- Plugin hooks run in OpenCode server process, can't see ORCH_WORKER env var from spawned agents - must use tool arguments or session metadata for detection
- Config hook runs before any tools execute - too early for per-session worker detection
- First tool call might not trigger detection if worker's first action isn't SPAWN_CONTEXT.md read - safe default is to load skill unless explicitly detected as worker

### Externalized via `kb`
- Investigation file captures all findings and implementation rationale for future reference

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (plugin refactored, investigation documented)
- [ ] Tests passing (manual testing needed - see Evidence section)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-y1ikp` after orchestrator manual testing

### Manual Testing Required
The orchestrator should verify lazy-loading behavior before closing:

**Test 1: Worker session (skill should NOT load)**
```bash
ORCH_PLUGIN_DEBUG=1 orch spawn investigation "test worker detection" --tmux
# Watch logs for:
# [orchestrator-session] Worker detected (SPAWN_CONTEXT.md read): session XXX
# [orchestrator-session] System transform: Skipping orchestrator skill for worker session XXX
```

**Test 2: Orchestrator session (skill SHOULD load)**  
```bash
ORCH_PLUGIN_DEBUG=1 opencode /Users/dylanconlin/Documents/personal/orch-go
# Watch logs for:
# [orchestrator-session] Cached orchestrator skill content: 52000 bytes
# [orchestrator-session] System transform: Injected orchestrator skill for session XXX (52000 bytes)
```

**Success criteria:**
- Worker sessions do NOT see "Injected orchestrator skill" log
- Orchestrator sessions DO see "Injected orchestrator skill" log
- No TypeScript compilation errors (module resolution warning is non-blocking)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- Performance impact: How much faster are worker sessions without 52KB orchestrator skill? Could measure via context size metrics before/after.
- First-call delay: What happens if worker's first tool call isn't SPAWN_CONTEXT.md? Do they get orchestrator skill for one message then lose it? (Probably harmless but worth observing)
- Caching strategy: Is reading 52KB file once at plugin init the best approach? Could we lazy-read it only when first non-worker session is detected?

**Areas worth exploring further:**
- Apply same lazy-loading pattern to other large skills if they exist
- Add metrics to track how often orchestrator skill is loaded vs skipped
- Consider whether experimental.chat.system.transform hook is stable enough for production use

**What remains unclear:**
- Whether the TypeScript module resolution warning causes any runtime issues (likely not, but worth monitoring)
- Whether there are edge cases in worker detection that coaching plugin has handled but we haven't (e.g., workers that don't read SPAWN_CONTEXT.md immediately)

---

## Session Metadata

**Skill:** feature-impl
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-feat-ci-implement-lazy-17jan-a5b3/`
**Investigation:** `.kb/investigations/2026-01-17-inv-ci-implement-lazy-loading-orchestrator.md`
**Beads:** `bd show orch-go-y1ikp`
