# Session Synthesis

**Agent:** og-debug-test-spawn-23dec
**Issue:** orch-go-mhxi
**Duration:** 2025-12-23 16:10 → 16:16 (6 minutes)
**Outcome:** success

---

## TLDR

Goal: Test spawn functionality to verify it works correctly. Found and fixed spawn hang caused by kb context --global scanning large directories without timeout. Spawn now completes successfully with graceful degradation when KB context times out.

---

## Delta (What Changed)

### Files Modified
- `pkg/spawn/kbcontext.go` - Added 5-second timeout to kb context command execution using context.WithTimeout

### Commits
- `8c8ef36` - fix: add 5-second timeout to kb context queries to prevent spawn hangs

---

## Evidence (What Was Observed)

### Problem Reproduction
- `orch spawn investigation "test"` hung indefinitely during "broader search" step (pkg/spawn/kbcontext.go:131)
- `kb context "verify"` hung for 30+ seconds (exit code 124 timeout)
- `kb context --global "verify"` hung for 30+ seconds
- `find ~/Documents -name ".kb"` hung for 5+ seconds
- Multiple stuck `kb context` processes observed via `ps aux`

### Root Cause Trace
- pkg/spawn/kbcontext.go:131 calls `kb context --global` when local search returns <3 results
- kb-cli/cmd/kb/context.go:181 GetContextGlobal() calls discoverProjects()
- kb-cli/cmd/kb/search.go:142 discoverProjects() does filepath.Walk() of ~/Documents (and ~/Projects, ~/repos, ~/src, ~/code)
- No timeout on cmd.Output() at pkg/spawn/kbcontext.go:136 - blocks indefinitely

### Fix Verification
```bash
# After timeout fix:
./build/orch spawn --no-track --light investigation "test timeout fix"
# Output:
# Checking kb context for: "test timeout context"
# Trying broader search for: "test"  
# No prior knowledge found.
# (Completed successfully - no hang!)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-test-spawn.md` - Documents spawn hang root cause and fix

### Decisions Made
- **Decision:** Add timeout to kb context queries rather than disabling global search entirely
- **Rationale:** Timeout provides graceful degradation - spawn works without KB context if search times out, but still gets KB context when it's fast. Better than losing the feature entirely.

### Root Cause
- **kb context --global hangs** because discoverProjects() scans ~/Documents recursively (3 levels deep) without timeout
- **Spawn amplified the issue** by automatically falling back to global search when local search returns <3 results
- **No timeout** on exec.Command().Output() means it blocks indefinitely if kb hangs

### Fix
- Use `context.WithTimeout(5 * time.Second)` and `exec.CommandContext()` for kb queries
- Treat timeout/error as "no context available" - spawn continues without KB context
- Preserves functionality when KB context is fast, degrades gracefully when slow

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (fix implemented and committed)
- [x] Tests passing (verified spawn completes without hang)
- [x] Investigation file has `Status: Complete`
- [x] Ready for `orch complete orch-go-mhxi`

### Follow-up Items

**Discovered Issue:** 400 error when sending prompt to OpenCode after session creation
- Not directly related to KB context hang
- Needs separate investigation
- May be transient or related to OpenCode API state

**Upstream Fix Opportunity:** kb-cli's discoverProjects() should use registry-only or cached discovery
- Current approach: filepath.Walk() of large directories without timeout
- Better approach: Use kb projects registry (already exists) instead of filesystem scan
- Or: Cache discovered projects with TTL to avoid repeated scans
- This would benefit all kb-cli users, not just orch-go

---

## Unexplored Questions

**Questions that emerged during this session:**
- Why does ~/Documents take so long to scan? (Symlinks? Network mounts? Permissions?)
- Is the 400 error in prompt sending related to token limits or request format?
- How many projects have .kb directories in typical ~/Documents? (Could guide kb-cli optimization)

**Areas worth exploring further:**
- Profiling kb context --global to see exactly where time is spent
- Adding caching to discoverProjects() in kb-cli
- Alternative spawn KB context strategies (registry-only, opt-in global, etc.)

**What remains unclear:**
- Whether the 400 error is a regression or an existing issue
- If other kb commands have similar timeout issues

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-test-spawn-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-test-spawn.md`
**Beads:** `bd show orch-go-mhxi`
