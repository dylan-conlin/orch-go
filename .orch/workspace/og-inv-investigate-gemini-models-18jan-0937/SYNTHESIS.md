# Session Synthesis

**Agent:** og-inv-investigate-gemini-models-18jan-0937
**Issue:** orch-go-nsy7w
**Duration:** 2026-01-18 ~14:00 → 2026-01-18 ~14:45
**Outcome:** success

---

## TLDR

Investigated why Gemini models output text mimicking tool calls instead of invoking tools. **Root cause:** The `gemini.txt` system prompt in OpenCode contains 14 examples showing tool calls as `[tool_call: ...]` text patterns, which the model mimics instead of using the actual function calling API.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-inv-investigate-gemini-text-mimicking-tool-calls.md` - Full investigation with findings, root cause, and recommendations
- `.orch/workspace/og-inv-investigate-gemini-models-18jan-0937/SYNTHESIS.md` - This synthesis

### Files Modified
- None (investigation only, no code changes)

### Commits
- (pending) Investigation file

---

## Evidence (What Was Observed)

- **14 instances of `[tool_call: ...]`** found in `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/gemini.txt` at lines 77, 82, 89, 90, 94, 107, 109, 122, 124, 126, 128, 135, 138, 146
- **Zero instances** in `anthropic.txt` - Anthropic models use prose descriptions instead
- Bug report showed `[tool_call: edit{filePath:<ctrl46>/path<ctrl46>...}]` - the `<ctrl46>` escaping confirms model is outputting text, not calling API
- System prompt selection at `system.ts:30` routes `gemini-*` models to the problematic prompt
- Gemini API documentation confirms proper function calling returns structured JSON, not text

### Tests Run
```bash
# Found 14 occurrences in gemini.txt
grep "tool_call:" gemini.txt
# lines 77, 82, 89, 90, 94, 107, 109, 122, 124, 126, 128, 135, 138, 146

# Found 0 occurrences in anthropic.txt
grep "tool_call:" anthropic.txt
# (no results)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-inv-investigate-gemini-text-mimicking-tool-calls.md` - Documents root cause and fix recommendations

### Decisions Made
- Root cause is **prompt engineering issue**, not API or rate limit problem
- Fix requires modifying `gemini.txt` in OpenCode to remove/replace the `[tool_call: ...]` patterns

### Constraints Discovered
- **Prompt examples train LLMs** - Showing tool calls as text teaches models to output text instead of using API
- Provider-specific prompts require provider-specific testing

### Externalized via `kb`
- `kb quick constrain "Prompt examples showing tool calls as text (like [tool_call: ...]) will train LLMs to output text instead of using function calling API" --reason "..."` - kb-b7198e

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, fix is in OpenCode repo not orch-go)

### If Close
- [x] All deliverables complete (investigation file documenting root cause)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-nsy7w`

### Implementation (for OpenCode maintainers or fork)
**Location:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/gemini.txt`
**Action:** Replace 14 `[tool_call: ...]` instances with prose like "Use the read tool to examine the file"

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How often does this bug occur in production? (no metrics available)
- Are specific Gemini model versions (gemini-3-flash vs gemini-2.5-pro) more susceptible?
- Does removing the examples degrade Gemini's tool usage comprehension?

**Areas worth exploring further:**
- Testing the fix with multiple Gemini models
- Checking if `beast.txt`, `codex.txt`, or other prompts have similar issues

**What remains unclear:**
- Whether there are other prompt elements contributing to the behavior
- Impact on Gemini's tool usage understanding after removing examples

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-gemini-models-18jan-0937/`
**Investigation:** `.kb/investigations/2026-01-18-inv-investigate-gemini-text-mimicking-tool-calls.md`
**Beads:** `bd show orch-go-nsy7w`
