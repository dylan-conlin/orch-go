# Session Synthesis

**Agent:** og-inv-token-limit-explosion-23dec
**Issue:** orch-go-4gb9
**Duration:** 2025-12-23 15:00 → 2025-12-23 18:00
**Outcome:** success

---

## TLDR

Investigated why spawn orch-go-iq2h hit 207k tokens (exceeding 200k limit). Found two primary bloat sources: KB context explosion (60k+ tokens from broad queries) and double skill loading (37k tokens from auto-loading orchestrator skill in worker sessions). Recommendations: set ORCH_WORKER=1, add KB token limits, add pre-spawn warnings.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md` - Complete investigation with 7 findings, synthesis, and implementation recommendations

### Files Modified
- None (investigation only, no code changes)

### Commits
- `283867a` - investigation: token limit explosion analysis

---

## Evidence (What Was Observed)

- **KB context bloat:** Found og-inv-pre-spawn-kb-22dec with 2,838 total lines, 2,437 of which were KB context (86%) from query "pre"
- **Double skill loading:** session-context plugin (session-context.js:59) auto-loads orchestrator skill (1,251 lines) for all orch projects
- **Token statistics:** OpenCode stats showed 142.8K avg tokens/session, 1.4B cache reads (25x input tokens)
- **File sizes measured:**
  - Global CLAUDE.md: 51 lines
  - Project CLAUDE.md: 229 lines  
  - Orchestrator skill: 1,251 lines
  - feature-impl skill: 400 lines
- **Failed spawn:** orch-go-iq2h closed with "token limit exceeded (207k > 200k)"

### Tests Run
```bash
# File size measurements
wc -l ~/.claude/CLAUDE.md  # 51 lines
wc -l orch-go/CLAUDE.md    # 229 lines
wc -l orchestrator/SKILL.md  # 1,251 lines
wc -l feature-impl/SKILL.md  # 400 lines

# Find largest spawn contexts
find .orch/workspace -name "SPAWN_CONTEXT.md" -exec wc -l {} \; | sort -rn | head -10
# Found: 2,838 lines (og-inv-pre-spawn-kb-22dec)

# KB context testing
kb context "model"  # Returns ~80 lines of cross-repo matches
kb context "pre"    # Even broader, triggers global search

# Token statistics
opencode stats --project "" --days 7
# Avg tokens/session: 142.8K
# Cache read: 1.4B tokens
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md` - Token bloat analysis

### Decisions Made
- N/A (investigation, no decisions)

### Constraints Discovered
- KB context can inject 60k+ tokens from broad queries due to cross-repo matching
- Orchestrator skill unnecessarily loaded in worker sessions wastes 37k tokens
- OpenCode average session uses 142.8K tokens, leaving little buffer before 200k limit

### Externalized via `kn`
- `kn constrain "Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading" --reason "...37k tokens wasted..."` - kn-d54b4f

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created and committed)
- [x] Tests passing (N/A for investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-4gb9`

**Implementation tasks (for follow-up):**
1. Set ORCH_WORKER=1 in runSpawnHeadless (cmd/orch/main.go:1147)
2. Add KB context token limit in pkg/spawn/kbcontext.go (MaxKBContextTokens ~20k)
3. Add pre-spawn token estimate/warning before CreateSession

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What exactly is in OpenCode's base system prompt? (estimated 40k-60k tokens but not measured directly)
- Does OpenCode index the entire project directory or just load specific files from instructions?
- Can we get accurate token counts from OpenCode API before sending requests to Claude?

**Areas worth exploring further:**
- Relevance scoring for KB context matches (weight by recency, project match, category)
- Dynamic KB context limits based on skill type (investigation needs more context, feature-impl needs less)
- Token-aware model selection (auto-switch to Gemini 2.5 Pro when approaching Claude limits)

**What remains unclear:**
- Whether the 207k failure happened on first message or accumulated over multiple exchanges
- Whether OpenCode's file context contributes significantly beyond explicit instructions

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-token-limit-explosion-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md`
**Beads:** `bd show orch-go-4gb9`
