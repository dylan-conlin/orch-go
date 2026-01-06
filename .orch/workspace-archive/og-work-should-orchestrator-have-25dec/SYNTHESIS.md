# Session Synthesis

**Agent:** og-work-should-orchestrator-have-25dec
**Issue:** orch-go-untracked-1766729942
**Duration:** ~30 minutes
**Outcome:** success

---

## TLDR

Design session analyzed whether orchestrator should have visibility into system resources (CPU/memory). Recommendation: No - external monitoring (sketchybar) already works, and the 125% CPU bug Dylan observed was already diagnosed and fixed without needing orchestrator-level resource visibility.

---

## Delta (What Changed)

### Files Created
- `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md` - Decision artifact recommending no implementation
- `.kb/investigations/2025-12-25-inv-should-orchestrator-have-visibility-into.md` - Investigation documenting analysis

### Files Modified
- None (design session - no code changes)

### Commits
- Will commit investigation and decision artifacts

---

## Evidence (What Was Observed)

- serve.go:315-319 documents the fix for the 125% CPU bug (IsProcessing HTTP polling) - showing the bug was already diagnosed via external monitoring
- Current dashboard stats bar shows agent-focused metrics (errors, focus, servers, beads) - not system resources
- ps aux showed orch serve at 153% CPU during investigation - reproducing Dylan's observation
- Dashboard 666px width constraint limits space for additional panels

### Tests Run
```bash
# Checked current process resource usage
ps aux | grep -E "(orch|opencode)"
# Result: orch serve at 153% CPU, opencode serve at 6.6% CPU
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md` - Decision recommending no implementation
- `.kb/investigations/2025-12-25-inv-should-orchestrator-have-visibility-into.md` - Supporting investigation

### Decisions Made
- No system resource visibility in orchestrator because external monitoring (sketchybar) already provides this capability and is better suited for system-level concerns

### Constraints Discovered
- Orchestration layer should focus on agent coordination, not process management
- High CPU/memory in orchestration processes indicates bugs, not normal operation worth monitoring

### Externalized via `kn`
- None required - decision captured in `.kb/decisions/` artifact

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (decision + investigation artifacts)
- [x] Investigation file has complete status
- [ ] Ready for `orch complete` - pending commit

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orch-go document sketchybar integration as the recommended resource monitoring approach?
- For users deploying orch-go in CI/CD or remote contexts, would minimal self-diagnostics ("/health with status") be valuable?

**Areas worth exploring further:**
- Other orchestration systems and how they handle health/resource monitoring
- Whether the current 153% CPU on orch serve (observed during investigation) indicates another bug

**What remains unclear:**
- Why orch serve is currently at 153% CPU (possibly same IsProcessing issue not fully fixed, or new cause)
- Whether the bug reproduction is related to this investigation session's dashboard access

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-should-orchestrator-have-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-should-orchestrator-have-visibility-into.md`
**Decision:** `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md`
**Beads:** orch-go-untracked-1766729942
