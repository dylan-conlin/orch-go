# Session Synthesis

**Agent:** we-inv-analyze-orchestration-session-16jan-ecbe
**Issue:** (ad-hoc, no-track)
**Duration:** 2026-01-16 ~30min
**Outcome:** success

---

## TLDR

Analyzed why orchestration session hit context limit prematurely. Found that **67% of the 56K-token context was consumed by setup** - primarily the embedded orchestrator skill (2,385 lines / ~28K tokens). Recommended tiered skill approach (core vs full) and tool output pagination.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-16-inv-analyze-orchestration-session-hit-context.md` - Complete analysis of context consumption in orchestrator sessions

### Files Modified
- None

### Commits
- (pending) Will commit investigation file

---

## Evidence (What Was Observed)

- Orchestrator skill spans lines 315-2700 of transcript (~2,385 lines / ~28K tokens)
- `orch status` produced 437 lines listing 182 agents in one unbounded output
- Session handoff + startup hooks consumed 220 lines before skill content even loaded
- User made only ~4 requests before exporting at line 4657
- CLI help text printed 3x (~90 lines) when `orch complete` failed with different flags

### Tests Run
```bash
# Line count analysis
wc -l oddly-short-orch-session-full.txt
# 4662 lines

# Token count from file read error
# "File content (56414 tokens) exceeds maximum"
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-16-inv-analyze-orchestration-session-hit-context.md` - Complete context consumption analysis

### Decisions Made
- Recommend tiered skill approach over lazy loading because skill is 50% of setup context
- Recommend pagination as complementary fix (addresses 9% of context)

### Constraints Discovered
- **Skill embedding budget constraint**: Skills designed for reference are not optimized for embedding
- **Tool output budget constraint**: Unbounded listings consume context proportional to system scale (182 agents = 437 lines)
- **Startup accumulation constraint**: Multiple systems (beads hooks, session handoffs, kb context, skill embedding) each add "reasonable" content that compounds to 67%

### Externalized via `kn`
- Not applicable (no kn available in this project)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Status:** Complete`
- [x] SYNTHESIS.md created
- [ ] Commit investigation file

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What's the actual token budget needed for a productive orchestrator session? (Need to analyze successful sessions)
- Which orchestrator skill sections are actually referenced during typical sessions? (Usage analysis)
- Can we auto-generate "core" from "full" based on usage patterns? (May be over-engineering)

**Areas worth exploring further:**
- Measure impact of skill reduction on orchestrator effectiveness
- Analyze token budget across multiple successful/failed sessions
- Consider dynamic skill loading based on detected needs

**What remains unclear:**
- Whether tiered skills will degrade orchestrator decision quality
- The optimal size for "core" skill content (estimated 500-800 lines, not validated)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/we-inv-analyze-orchestration-session-16jan-ecbe/`
**Investigation:** `.kb/investigations/2026-01-16-inv-analyze-orchestration-session-hit-context.md`
**Beads:** (ad-hoc, no tracking)
