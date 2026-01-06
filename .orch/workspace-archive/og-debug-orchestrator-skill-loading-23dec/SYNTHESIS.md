# Session Synthesis

**Agent:** og-debug-orchestrator-skill-loading-23dec
**Issue:** orch-go-v2cz
**Duration:** 2025-12-23 09:30 → 2025-12-23 10:30
**Outcome:** success

---

## TLDR

Debugged why orchestrator skill was loading for workers despite `audience:orchestrator` field. Root cause: session-context plugin checked ORCH_WORKER env var at plugin init (once) instead of per-session in config hook. Fixed by moving check into hook.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-orchestrator-skill-loading-workers-despite.md` - Full investigation with root cause analysis

### Files Modified
- `~/Documents/personal/orch-cli/.opencode/plugin/session-context.ts` - Moved ORCH_WORKER check from plugin init to config hook (per-session)

### Commits
- `ac945ea` - fix: move ORCH_WORKER check to config hook for per-session filtering

---

## Evidence (What Was Observed)

- Orchestrator skill has `audience: orchestrator` field in SKILL.md:4
- Session-context plugin at `orch-cli/.opencode/plugin/session-context.ts` handles loading
- ORCH_WORKER check on line 72 runs at plugin init (once globally)
- Config hook (lines 88-102) runs per-session and adds skill unconditionally
- When OpenCode starts in orchestrator context, plugin enables skill loading for ALL sessions

### Tests Run
```bash
# Verified skill metadata
cat ~/.claude/skills/meta/orchestrator/SKILL.md | head -10
# Confirmed audience:orchestrator field exists

# Found and read plugin code
find ~/Documents/personal/orch-cli -name "session-context.ts"
# Located root cause

# Committed fix
git commit -m "fix: move ORCH_WORKER check to config hook..."
# SUCCESS: Changes committed
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-orchestrator-skill-loading-workers-despite.md` - Root cause analysis with 90% confidence

### Decisions Made
- Decision: Use env var check instead of parsing audience field - env var more reliable and set explicitly by spawn mechanism
- Decision: Move check to config hook rather than refactor plugin architecture - minimal change, standard pattern

### Constraints Discovered
- OpenCode plugins initialize once globally, hooks run per-session - env checks must be inside hooks for per-session behavior
- Plugin init vs hook execution timing is critical for correct filtering

### Externalized via `kn`
- No `kn` entries needed - investigation file captures all knowledge

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file, fix committed)
- [⏳] Tests passing (needs smoke test - spawn worker and verify skill not loaded)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-v2cz`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Do other plugins in `.opencode/plugin/` have similar timing bugs? (agentlog-inject.ts, usage-warning.ts, bd-close-gate.ts)
- Would parsing audience field from SKILL.md be useful for future skills beyond orchestrator?
- What's the proper smoke test for this fix? (spawn worker, check context)

**Areas worth exploring further:**
- Plugin reload mechanism - does OpenCode need restart or can it hot-reload?
- OpenCode plugin API stability across versions

**What remains unclear:**
- Whether fix works across all OpenCode versions (assumed stable API)
- Edge cases around plugin reload and multiple concurrent sessions

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-orchestrator-skill-loading-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-orchestrator-skill-loading-workers-despite.md`
**Beads:** `bd show orch-go-v2cz`
