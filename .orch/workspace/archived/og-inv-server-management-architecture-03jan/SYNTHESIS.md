# Session Synthesis

**Agent:** og-inv-server-management-architecture-03jan
**Issue:** orch-go-n1vv
**Duration:** 2026-01-03 ~13:55 → ~14:50
**Outcome:** success

---

## TLDR

Investigated server management architecture confusion across tmuxinator, orch servers, and launchd. Found that the three-layer architecture is intentional and sound: launchd manages persistent infrastructure (daemon, API, web UI), tmuxinator manages per-project dev servers, and `orch servers` is a CLI wrapper. Vite process pileup is caused by orphaned child processes when launchd restarts the web service without killing children.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-03-inv-server-management-architecture-confusion-tmuxinator.md` - Full investigation documenting the architecture

### Files Modified
- None

### Commits
- `9302ed89` - investigation: server-management-architecture - checkpoint

---

## Evidence (What Was Observed)

- Three launchd services running: `com.orch.daemon`, `com.orch-go.serve`, `com.orch-go.web` (verified via `launchctl list | grep orch`)
- Daemon is actively polling with 60s interval, showing verbose debug output in `~/.orch/daemon.log`
- Multiple vite processes (5) running, some with PPID=1 indicating orphaned processes from launchd restarts
- 34 tmuxinator configs exist in `~/.tmuxinator/workers-*.yml` for various projects
- `orch servers` command wraps tmuxinator, doesn't duplicate it (verified in `cmd/orch/servers.go:231-260`)

### Tests Run
```bash
# Verify launchd services
launchctl list | grep orch
# OUTPUT: 3 services running

# Check process parentage of vite processes
ps -p 59843 -o pid,ppid,start,comm,args
# OUTPUT: PPID=1, confirming orphaned process

# Verify daemon status
launchctl print gui/$(id -u)/com.orch.daemon
# OUTPUT: state = running, working correctly
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-03-inv-server-management-architecture-confusion-tmuxinator.md` - Complete architecture documentation

### Decisions Made
- Architecture is intentional - The three-layer separation (launchd, tmuxinator, orch servers) is by design, not confusion
- Documentation needed - The confusion arises from lack of explicit documentation, not design flaw

### Constraints Discovered
- Launchd process cleanup - When launchd restarts a service, child processes (like vite) become orphaned (PPID=1) unless explicitly handled
- Tmux session lifecycle - `workers-{project}` sessions are project-scoped and independent of launchd services

### Externalized via `kn`
- No new kn entries created (findings captured in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation question fully answered
- [x] Investigation file has complete D.E.K.N. summary
- [x] Ready for `orch complete orch-go-n1vv`

### Follow-up Recommendations (Optional)

**Issue 1:** Add architecture documentation to CLAUDE.md or docs
**Skill:** feature-impl
**Context:** Document the three-layer server management architecture (launchd, tmuxinator, orch servers) in a discoverable location.

**Issue 2:** Fix vite process cleanup on launchd restart
**Skill:** systematic-debugging
**Context:** Add `<key>AbandonProcessGroup</key><false/>` to com.orch-go.web.plist to kill child processes on restart, or create wrapper script with signal trapping.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the web UI dev server use tmuxinator instead of launchd for consistency?
- Are there other orphaned processes beyond vite that should be cleaned up?
- Does the 143 restarts of com.orch-go.web indicate instability or normal launchd behavior?

**Areas worth exploring further:**
- Process supervision with proper cleanup hooks across all launchd services
- Whether `orch servers` should manage infrastructure servers too (unifying the CLI)

**What remains unclear:**
- Whether the vite processes on different ports are intentional (some on 5188, 5189)
- The relationship between com.orch-go.web (launchd) and tmuxinator workers-orch-go

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-20250514
**Workspace:** `.orch/workspace/og-inv-server-management-architecture-03jan/`
**Investigation:** `.kb/investigations/2026-01-03-inv-server-management-architecture-confusion-tmuxinator.md`
**Beads:** `bd show orch-go-n1vv`
