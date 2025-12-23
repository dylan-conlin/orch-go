# Session Synthesis

**Agent:** og-inv-find-command-performance-23dec
**Issue:** orch-go-tymf
**Duration:** 2025-12-23 (session start) → 2025-12-23 (session end)
**Outcome:** success

---

## TLDR

Investigated whether to install fd, create wrapper scripts, or enhance agent guidance for better file search performance. Found that OpenCode's built-in glob/grep tools already use ripgrep (5-37x faster than find), making additional tooling redundant. Recommendation: maintain status quo.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-find-command-performance-evaluate-alternatives.md` - Comprehensive investigation with performance tests and recommendations
- `.orch/workspace/og-inv-find-command-performance-23dec/SYNTHESIS.md` - This synthesis document

### Files Modified
- None (investigation only, no code changes)

### Commits
- `00dec80` - Add investigation: find command performance evaluation

---

## Evidence (What Was Observed)

- OpenCode documentation confirms glob/grep tools use ripgrep internally (https://opencode.ai/docs/tools/)
- Performance tests in orch-go project: find (0.062s) vs fd (0.012s) vs ripgrep (0.011s) - 5-5.6x faster
- Performance tests from home directory: find (1.415s) vs fd (0.038s) - 37x faster
- Gitignore respect: find found 10 node_modules dirs, fd found 0 (correctly ignored)
- Content search: grep (0.091s) vs ripgrep (0.007s) - 13x faster
- Agents already receive guidance: "File search: Use Glob (NOT find or ls)" in spawn context

### Tests Run
```bash
# File search performance comparison (project directory)
time find . -name "*.go" -type f      # 0.062s (88 files)
time fd -e go -t f .                  # 0.012s (88 files)
time rg --files --glob "*.go"         # 0.011s (88 files)

# File search performance comparison (home directory)
time find ~ -name "*.go" | head -20   # 1.415s total
time fd -e go . ~ | head -20          # 0.038s total

# Gitignore behavior test
find . -name "node_modules" -type d   # Found 10 directories
fd node_modules -t d                  # Found 0 (respects .gitignore)

# Content search performance comparison
time grep -r "func.*Error" --include="*.go" .  # 0.091s (21 matches)
time rg "func.*Error" --type go                # 0.007s (21 matches)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-find-command-performance-evaluate-alternatives.md` - Documents performance findings and recommendation

### Decisions Made
- **Do not install fd** - OpenCode's glob tool already uses ripgrep, which is equivalently fast
- **Do not create wrapper scripts** - Would add complexity without solving root cause
- **Maintain status quo** - Existing tools and guidance are optimal
- **Monitor only if complaints recur** - If find performance is reported again, first verify if agents are using bash find vs Glob tool

### Constraints Discovered
- None - this was a performance evaluation, not constraint discovery

### Externalized via `kn`
- `kn decide "Use OpenCode's built-in glob/grep tools instead of installing fd" --reason "glob/grep already use ripgrep internally (5-37x faster than find). Installing fd would be redundant and doesn't solve root cause (agents using wrong tool)"` - Created decision kn-4bf1b1

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created and committed)
- [x] Tests passing (N/A for investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-tymf`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How often do agents actually use bash find vs Glob tool in practice? (Would require session audit)
- Are there legitimate cases where bash find is necessary despite performance cost? (Edge cases unknown)
- Could OpenCode add telemetry to track tool usage patterns? (Would enable data-driven guidance improvements)

**Areas worth exploring further:**
- Session audit to measure Glob vs bash find usage rates
- A/B test of stronger guidance (constraints vs suggestions) if compliance is low

**What remains unclear:**
- Whether the existing "Use Glob (NOT find)" guidance is effective enough in practice
- If there are edge cases where bash find is the right tool despite performance cost

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-find-command-performance-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-find-command-performance-evaluate-alternatives.md`
**Beads:** `bd show orch-go-tymf`
