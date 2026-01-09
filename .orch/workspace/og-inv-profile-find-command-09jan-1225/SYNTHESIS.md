# Session Synthesis

**Agent:** og-inv-profile-find-command-09jan-1225
**Issue:** orch-go-uo8qv
**Duration:** 2026-01-09 15:13 → 2026-01-09 23:45
**Outcome:** success

---

## TLDR

Profiled find vs fd performance and found fd is 4-92x faster, with guidance-first implementation recommended to reduce agent file search time from 30+ seconds to <1 second.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-09-inv-profile-find-command-performance-explore.md` - Complete investigation with performance benchmarks and implementation recommendations

### Files Modified
- None (investigation only, no code changes)

### Commits
- `dc5768b6` - investigation: profile-find-command-performance-explore - checkpoint
- `3161148d` - investigation: profile-find-command-performance-explore - completed

---

## Evidence (What Was Observed)

### Performance Benchmarks

**Small tree (orch-go project):**
- find: 0.102s | fd: 0.022s → 4.6x faster
- Pattern: `find . -name "*.go"` vs `fd -e go`

**Medium tree (with node_modules):**
- find: 0.124s | fd: 0.030s → 4.1x faster  
- Pattern: Finding 5502 .js files

**Large tree (Documents/personal):**
- find: 11.675s | fd: 0.126s → 92x faster
- Pattern: Finding .go files across multiple repos

**Gitignore awareness:**
- find found 2831 .md files (includes node_modules, .git)
- fd found 46 .md files (respects .gitignore automatically)

### Compatibility Testing

- fd --glob mode produces same results as find -name patterns
- Output format is identical (newline-separated paths)
- Both work with downstream pipelines (wc, xargs, grep)

### Tests Run
```bash
# Performance comparison
time find . -name "*.go" -type f 2>/dev/null | wc -l
time /opt/homebrew/bin/fd -e go | wc -l

# Glob mode compatibility
find . -name 'serve*.go' -type f
/opt/homebrew/bin/fd --glob 'serve*.go'
# → same results

# Installation
/opt/homebrew/bin/brew install fd
# → fd 10.3.0 installed successfully
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-09-inv-profile-find-command-performance-explore.md` - Complete performance analysis with 6 findings

### Decisions Made
- **Decision 1:** Recommend guidance-first approach over translation layer because it's simpler, more transparent, and teaches agents the better tool
- **Decision 2:** Use --glob mode for find compatibility (easier than regex mode for agents)
- **Decision 3:** Follow existing PATH pattern (symlink to ~/.bun/bin)

### Constraints Discovered
- fd not in agent PATH by default (requires symlink per CLAUDE.md pattern)
- fd has different binary name on Debian (fdfind) - need alias
- fd output order differs from find (sorted vs traversal order) - usually doesn't matter
- fd skips hidden files by default (need -H flag for .env, .git, etc.)

### Key Insights
1. **Root cause:** 30+ second delays occur when find scans large directory trees without exclusions
2. **Why fd wins:** Parallel execution + automatic .gitignore respect + optimized traversal
3. **Implementation simplicity:** --glob mode makes translation straightforward if needed
4. **Adoption path:** Guidance-first is lowest complexity, can add translation if needed

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with findings and recommendations)
- [x] Investigation file has `**Phase:** Complete`
- [x] Tested performance claims with actual benchmarks
- [x] Ready for `orch complete orch-go-uo8qv`

**Implementation steps for orchestrator:**
1. Create symlink: `ln -sf /opt/homebrew/bin/fd ~/.bun/bin/fd`
2. Add fd guidance to SPAWN_CONTEXT.md template (before DELIVERABLES section)
3. Update ~/.claude/CLAUDE.md with fd recommendation
4. Test with one agent session and measure improvement

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does fd perform on network filesystems (NFS, SMB)? - might be slower than local
- What's agent comprehension rate after adding guidance? - need to measure adoption
- Should we also recommend ripgrep (rg) for content search? - similar pattern, separate issue
- Does fd work well in Docker containers / CI environments? - deployment consideration

**Areas worth exploring further:**
- Translation layer implementation if guidance approach fails
- Automatic detection of "slow" find patterns worth translating
- Performance comparison on Linux (only tested macOS)
- Agent analytics to track find→fd adoption rate

**What remains unclear:**
- Whether agents will adopt fd consistently with guidance alone (need to measure)
- Edge cases where find is actually needed (rare, but might exist)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus (via OpenCode)
**Workspace:** `.orch/workspace/og-inv-profile-find-command-09jan-1225/`
**Investigation:** `.kb/investigations/2026-01-09-inv-profile-find-command-performance-explore.md`
**Beads:** `bd show orch-go-uo8qv`
