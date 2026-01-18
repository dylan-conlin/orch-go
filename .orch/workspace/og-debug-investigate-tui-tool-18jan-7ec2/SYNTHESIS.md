# Session Synthesis

**Agent:** og-debug-investigate-tui-tool-18jan-7ec2
**Issue:** orch-go-f5c6b
**Duration:** 2026-01-18 10:36 → 2026-01-18 11:05
**Outcome:** success

---

## TLDR

Investigated TUI tool call display bug showing redundant formatting. Root cause: Gemini models output text mimicking prompt examples before making tool calls, causing duplicate display of tool metadata. Recommended fix: Use `experimental.text.complete` plugin hook to filter `tool_call:` prefixed text.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-inv-investigate-tui-tool-call-display.md` - Complete investigation documenting root cause and fix approach

### Files Modified
- None (investigation-only session)

### Commits
- None yet (investigation file ready to commit)

---

## Evidence (What Was Observed)

- Gemini prompt (`/packages/opencode/src/session/prompt/gemini.txt:77-128`) contains `[tool_call: ...]` examples
- TUI has separate rendering paths for text parts and tool parts (`/packages/opencode/src/cli/cmd/tui/routes/session/index.tsx:1537-1556`)
- Plugin hook `experimental.text.complete` exists at `/packages/opencode/src/session/processor.ts:308-317` for text modification
- Bash tool correctly sets title from `params.description` at `/packages/opencode/src/tool/bash.ts:248`

### Tests Run
```bash
# Code examination only - no functional tests run
# Bug reproduction requires live Gemini session (noted as untested in investigation)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-inv-investigate-tui-tool-call-display.md` - Root cause analysis and fix recommendations

### Decisions Made
- Use plugin hook approach: Text filtering via `experimental.text.complete` because it uses existing infrastructure without core code changes

### Constraints Discovered
- Gemini-specific behavior: Model interprets prompt examples as format to follow for announcing tool calls
- Text and tool parts render independently in TUI - both are displayed when both exist

### Externalized via `kn`
- Not applicable (findings captured in investigation file, no universal constraints or decisions)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Tests passing (investigation-only, no code changes)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-f5c6b`

**Follow-up work (optional):**
- Create text filtering plugin at `~/.opencode/plugin/filter-tool-call-text.ts`
- Test with live Gemini session to verify fix

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does Claude Code have similar text output issues? (Could check anthropic.txt prompt)
- Should gemini.txt prompt be modified to discourage text announcements rather than filtering?

**Areas worth exploring further:**
- Performance impact of plugin hook on large text streams
- Whether filtering could break legitimate text mentioning "tool_call"

**What remains unclear:**
- Exact Gemini behavior without live testing (investigation based on code analysis)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4.5
**Workspace:** `.orch/workspace/og-debug-investigate-tui-tool-18jan-7ec2/`
**Investigation:** `.kb/investigations/2026-01-18-inv-investigate-tui-tool-call-display.md`
**Beads:** `bd show orch-go-f5c6b`
