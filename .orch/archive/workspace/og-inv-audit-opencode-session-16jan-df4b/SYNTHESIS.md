# Session Synthesis

**Agent:** og-inv-audit-opencode-session-16jan-df4b
**Issue:** orch-go-4zc2s
**Duration:** 2026-01-16 13:28 → 2026-01-16 14:05
**Outcome:** success

---

## TLDR

Audited OpenCode session start injection mechanisms and found it's architecturally leaner than Claude Code: ~4KB direct injection vs ~25KB. Key difference is that OpenCode loads orchestrator skill via config.instructions (file reference) rather than direct injection, and uses cleaner worker detection (ORCH_WORKER env var + SPAWN_CONTEXT.md presence).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-16-inv-audit-opencode-session-start-injection.md` - Comprehensive audit of OpenCode plugins and comparison with Claude Code hooks

### Files Modified
- None (investigation only)

### Commits
- (pending) `investigation: audit OpenCode session start injection`

---

## Evidence (What Was Observed)

### Plugin Discovery
- 4 global plugins at `~/.config/opencode/plugin/`: session-resume.js, guarded-files.ts, session-compaction.ts, friction-capture.ts
- 4 project plugins at `~/Documents/personal/orch-cli/.opencode/plugin/`: session-context.ts, agentlog-inject.ts, usage-warning.ts, bd-close-gate.ts
- Plugins auto-discovered via glob pattern `{plugin,plugins}/*.{ts,js}` in config directories

### Session Start Injection Sizes
```bash
# Session resume injection
orch session resume --for-injection | wc -c
# Output: 4,120 bytes (~1,030 tokens)

# Orchestrator skill (loaded via instructions, not injected)
wc -c ~/.claude/skills/meta/orchestrator/SKILL.md
# Output: 86,451 bytes (~21,613 tokens)
```

### Key Finding: Instructions vs Injection
- **Instructions:** File paths added to config.instructions array, loaded at config time
- **Injection:** Content pushed via client.session.prompt with noReply: true

OpenCode's session-context.ts uses instructions (file reference), not injection:
```typescript
// session-context.ts:103-106
config.instructions.push(skillPath)  // Adds file path, not content
```

### Worker Detection Mechanisms
| System | Mechanism | Notes |
|--------|-----------|-------|
| OpenCode | ORCH_WORKER env var | Simple boolean |
| OpenCode | SPAWN_CONTEXT.md presence | File-based detection |
| Claude Code | CLAUDE_CONTEXT env var | Values: worker/orchestrator/meta-orchestrator |

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-16-inv-audit-opencode-session-start-injection.md` - Full audit with comparison tables

### Decisions Made
- No new decisions - this is an audit investigation

### Constraints Discovered
- OpenCode plugins are project-scoped when in `.opencode/plugin/` (only load for that project)
- ORCH_WORKER env var must be set at server level, not per-session (plugins see server env)
- session-resume.js works around this by checking for SPAWN_CONTEXT.md file presence instead

### Key Insight
OpenCode's architecture separates "instructions" (file references) from "injection" (runtime content push):
- Instructions: Loaded once at config time, included in system context
- Injection: Pushed into session after creation, appears as conversation message

This explains why OpenCode "feels" lighter - the orchestrator skill is loaded as a file reference, not pushed as runtime content.

### Externalized via `kb`
- (none) - Investigation findings documented in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with comparison tables)
- [x] Tests passing (N/A - investigation only)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-4zc2s`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does the instructions mechanism affect context budgeting differently from injection?
- Should orch-cli's project plugins be ported to orch-go?
- Could Claude Code adopt a similar instructions vs injection separation?

**Areas worth exploring further:**
- Probe 3: SPAWN_CONTEXT.md audit (what content for worker vs orchestrator?)
- Probe 4: Usage analysis (which injected content is actually referenced?)

**What remains unclear:**
- Whether instructions and injection have different context budget implications
- Performance difference between config-time loading vs runtime injection

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-audit-opencode-session-16jan-df4b/`
**Investigation:** `.kb/investigations/2026-01-16-inv-audit-opencode-session-start-injection.md`
**Beads:** `bd show orch-go-4zc2s`
