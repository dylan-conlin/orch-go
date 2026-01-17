# Session Synthesis

**Agent:** og-inv-audit-sessionstart-hooks-16jan-b4a3
**Issue:** orch-go-c1y6q
**Duration:** 2026-01-16 12:18 → 2026-01-16 13:05
**Outcome:** success

---

## TLDR

Audited all 7 SessionStart hooks in Claude Code, found total worst-case injection of ~25K tokens with load-orchestration-context.py responsible for 93% (23K tokens); spawn detection exists via CLAUDE_CONTEXT env var but only one hook uses it; beads guidance is duplicated across 3 sources.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md` - Complete audit of 7 SessionStart hooks with size measurements, spawn detection analysis, and overlap findings

### Files Modified
- None (investigation-only work)

### Commits
- (pending) investigation: audit sessionstart hooks for claude code

---

## Evidence (What Was Observed)

- load-orchestration-context.py outputs 93,631 bytes when CLAUDE_CONTEXT is unset (manual session)
- Orchestrator skill at `~/.claude/skills/orchestrator/SKILL.md` is 86,451 bytes
- bd prime outputs 2,961 bytes always
- session-start.sh outputs 4,246 bytes when session resume is available
- reflect-suggestions-hook.py outputs 538 bytes when suggestions exist
- inject-orch-patterns.sh, agentlog-inject.sh, usage-warning.sh output 0 bytes (conditions not met)
- CLAUDE_CONTEXT check exists ONLY in load-orchestration-context.py (lines 436-456)

### Tests Run
```bash
# Tested each hook with simulated input
echo '{"cwd":"/Users/dylanconlin/Documents/personal/orch-go","source":"startup"}' | ~/.orch/hooks/load-orchestration-context.py | wc -c
# Result: 93631 (without CLAUDE_CONTEXT set)

bd prime --full | wc -c
# Result: 2961

wc -c ~/.claude/skills/orchestrator/SKILL.md
# Result: 86451
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md` - Complete hook audit with deliverable table

### Decisions Made
- CLAUDE_CONTEXT env var is the spawn detection mechanism (not a new decision, but confirmed)
- Option A in epic (hooks for manual, spawn context for spawned) is supported by findings

### Constraints Discovered
- Only one hook (load-orchestration-context.py) has spawn detection - all others run regardless
- Beads guidance is duplicated in 3 places: bd prime, SPAWN_CONTEXT.md, orchestrator skill
- inject-orch-patterns.sh depends on a patterns file that doesn't exist (0 bytes)

### Externalized via `kb`
- Investigation file serves as the externalized knowledge for this audit
- Recommending promotion to decision (see investigation D.E.K.N.)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with table)
- [x] Tests performed (hook output sizes measured)
- [x] Investigation file has `**Status:** Complete`
- [ ] Ready for `orch complete orch-go-c1y6q`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Is CLAUDE_CONTEXT reliably set by all spawn paths (opencode, claude --backend, etc.)?
- What's the latency impact of running 7 hooks sequentially?
- Should the orchestrator skill be lazy-loaded on demand rather than at session start?

**Areas worth exploring further:**
- Probe 2: OpenCode plugin audit (next probe in epic)
- Impact of removing bd prime for spawned workers

**What remains unclear:**
- Whether actual token counts match the ~4 chars/token estimate
- Whether hooks execute in parallel or serial

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-audit-sessionstart-hooks-16jan-b4a3/`
**Investigation:** `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md`
**Beads:** `bd show orch-go-c1y6q`
