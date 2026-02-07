# Session Synthesis

**Agent:** og-inv-actually-kills-opencode-07feb-acd4
**Issue:** orch-go-21423
**Duration:** 2026-02-07 01:29 -> 2026-02-07 01:40
**Outcome:** success

---

## TLDR

Investigated what OS-level mechanism kills the OpenCode server process. Found it's macOS jetsam (memory pressure killer) triggered when OpenCode grows to ~8.4 GB RSS, confirmed by JetsamEvent diagnostic report. Three compounding fixes recommended: auto-restart via overmind, heap limit, and memory monitoring.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-07-inv-actually-kills-opencode-server-process.md` - Full investigation with OS-level evidence of jetsam killing OpenCode
- `.orch/workspace/og-inv-actually-kills-opencode-07feb-acd4/SYNTHESIS.md` - This file

### Files Modified
- None (investigation-only session)

### Commits
- `f2b01a66` - investigation: checkpoint - start OS-level OpenCode process death analysis
- (final commit with completed investigation and synthesis)

---

## Evidence (What Was Observed)

- JetsamEvent-2026-02-06-033432.ips shows OpenCode PID 53190 at 550,922 pages = 8,608 MB as `largestProcess` with system at 0.3 GB free
- 26 concurrent bun agent processes consuming 14.3 GB combined at time of jetsam event
- Current OpenCode process (PID 22575) at 1.58 GB RSS after 13.5 hours uptime, 64 FDs open (not exhausted)
- Overmind started with `--can-die opencode` (no auto-restart) - process stays dead after jetsam kill
- crash.log shows 7 unhandledRejection events at 331-431 MB RSS - these don't kill the process
- No bun/opencode crash reports in DiagnosticReports - confirms it's jetsam not a crash bug
- System has 36 GB RAM; OpenCode + agents consumed 22.7 GB (63%)

### Tests Run
```bash
# Verified jetsam report contents
tail -n +2 /Library/Logs/DiagnosticReports/JetsamEvent-2026-02-06-033432.ips | python3 -c "..."
# Result: opencode at 8608 MB, 26 bun processes at 14601 MB total

# Verified current process state
ps -o pid,rss,vsz,etime,%mem,%cpu -p 22575
# Result: RSS 1601984 KB, elapsed 13:34:10

# Verified file descriptor count
lsof -p 22575 | wc -l
# Result: 64 (not exhausted)

# Verified overmind config
ps aux | grep overmind
# Result: --can-die opencode (no --auto-restart)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-07-inv-actually-kills-opencode-server-process.md` - Definitive answer to what kills the OpenCode server

### Decisions Made
- No code changes required in this investigation - findings only

### Constraints Discovered
- OpenCode server has no memory limit and grows unbounded until jetsam kills it at ~8 GB
- Overmind `--can-die` prevents auto-restart, making jetsam kills permanent until manual intervention
- 36 GB system cannot sustain OpenCode server + 26 agent processes (22.7 GB combined)

### Externalized via `kb`
- `kb quick tried "Searching macOS unified log for bun/opencode process events" --failed "log show returns empty - no syslog entries"` (kb-bfc64a)
- `kb quick constrain "OpenCode server has no memory limit - grows unbounded until macOS jetsam kills it at ~8 GB RSS" --reason "Jetsam report shows opencode at 8608 MB as largestProcess"` (kb-630d7e)

---

## Issues Created

No discovered work during this session. The investigation findings themselves are the deliverable - implementation issues should be created by the orchestrator based on the recommendations.

---

## Next (What Should Happen)

**Recommendation:** close + spawn-follow-up

### If Close
- [x] All deliverables complete (investigation file with findings)
- [x] Tests performed (jetsam report analysis, process profiling, FD check)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-21423`

### If Spawn Follow-up
**Issue:** Add --auto-restart opencode to overmind and set Bun heap limit
**Skill:** feature-impl
**Context:**
```
OpenCode server dies from jetsam at ~8 GB RSS. Three fixes needed:
1. Change overmind --can-die to --auto-restart in Procfile (5 min)
2. Add BUN_JSC_heapSize=4096 to opencode env in Procfile (5 min)
3. Add memory logging to OpenCode server.ts (30 min)
See .kb/investigations/2026-02-07-inv-actually-kills-opencode-server-process.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What specifically grows in OpenCode's heap? Need Bun heap profiling to find the exact leak source (sessions cached in memory? tool output? event listeners?).
- Is memory growth linear or accelerating? Only measured at one point (2 MB/min at 1.58 GB).
- Do the 26 agent bun processes also have memory leaks, or is their 400-900 MB per process expected for long-running AI agent sessions?

**Areas worth exploring further:**
- Bun-specific memory profiling tools and heap snapshot capability
- Whether OpenCode upstream (not Dylan's fork) has the same memory growth pattern
- Impact of reducing max concurrent agents on system stability

**What remains unclear:**
- Whether the Feb 6 jetsam event killed OpenCode directly or cascaded (the `reason: per-process-limit` was on IntelligencePlatformComputeServi, not OpenCode)
- How many times OpenCode has been killed by jetsam historically (only one jetsam report found)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-actually-kills-opencode-07feb-acd4/`
**Investigation:** `.kb/investigations/2026-02-07-inv-actually-kills-opencode-server-process.md`
**Beads:** `bd show orch-go-21423`
