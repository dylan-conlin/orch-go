# Session Synthesis

**Agent:** og-arch-zombie-bun-processes-10feb-ce39
**Issue:** orch-go-21520
**Duration:** 2026-02-10 ~17:00 → ~18:15
**Outcome:** success

---

## TLDR

Diagnosed why zombie bun processes recur despite 3 prior fix attempts. Root cause is three compounding integration mismatches: (1) orphan detector's `"run --attach"` pattern doesn't match current `opencode attach` command format, (2) process ledger is empty because neither tmux nor headless spawns populate it, (3) OpenCode's Session.remove() deletes storage but never kills the attached bun process. The fix must be session-aware (not command-format-based) to be non-regressive.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-10-design-zombie-bun-root-cause-architecture.md` — Root cause analysis with 5 findings, 3 decision forks, implementation recommendations

### Files Modified
- None (investigation-only session)

### Commits
- Pending (investigation artifact ready to commit)

---

## Evidence (What Was Observed)

- `~/.orch/process-ledger.jsonl` is 0 bytes — Tier 1 ledger-backed detection has zero entries to work with
- `orphans.go:45` literal `"run --attach"` doesn't match current bun process format (`bun run --conditions=browser ./src/index.ts attach http://...`) — Tier 2 title-based detection is blind
- `spawn_execute.go:218` gates ledger write on `result.cmd != nil` — always nil for headless (HTTP API) and tmux (SendKeys) paths
- `spawn_execute.go:361-383` headless path is pure HTTP API since commit `6890004b` (Feb 9) — no bun subprocess created
- OpenCode `session/index.ts:353-374` Session.remove() only deletes storage/messages, publishes Event.Deleted — no process termination
- `ps -eo pid,ppid,args` shows bun processes have PPID=zsh (tmux-spawned), not PPID=opencode-server

### Tests Run
```bash
# Verified process ledger is empty
cat ~/.orch/process-ledger.jsonl | wc -l  # → 0

# Verified bun process command format (no "run --attach" present)
ps -eo pid,ppid,args | grep bun | grep -v grep

# Verified OpenCode sessions API returns no PID metadata
curl -s http://127.0.0.1:4096/session | python3 -m json.tool
```

---

## Verification Contract

- **Spec:** N/A (investigation session — no code changes to verify)
- **Key outcomes:**
  - Root cause identified with 3 independent failure modes
  - Prior investigation's attribution corrected (headless → tmux)
  - 4-step implementation plan produced with decision fork analysis

*(No VERIFICATION_SPEC.yaml needed for pure investigation sessions.)*

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-10-design-zombie-bun-root-cause-architecture.md` — Complete root cause analysis of recurring zombie bun processes

### Decisions Made (Recommendations)
- Detection should be session-aware (OpenCode API), not command-format-based — because format matching is structurally fragile and has already broken once
- Defense-in-depth termination at both orch (tmux window kill) and OpenCode (Session.remove process kill) — because two-tier principle requires both layers
- Implementation should be incremental: fix detection first (immediate relief), then complete lifecycle, then boundary fix

### Constraints Discovered
- Tmux-spawned bun process PIDs are fundamentally unknowable to orch at spawn time — process is started by tmux shell, not by exec.Command
- OpenCode sessions API does not expose PID metadata — detection must use ps cross-referenced with session directory info
- The Feb 9 headless HTTP migration accidentally eliminated the only path that populated the process ledger

### Externalized via `kn`
- N/A (findings externalized in investigation file)

---

## Issues Created

No new issues created. The investigation itself is the deliverable for orch-go-21520. Implementation work should be spawned by orchestrator based on the recommended 4-step sequence.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation with 5 findings, 3 forks, implementation plan)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-21520`

**Orchestrator action needed:** Review recommendations and spawn implementation agents for the 4-step sequence:
1. Fix `FindAgentProcesses` detection pattern (orch-only, ~1h)
2. Fix `orch complete` tmux cleanup ordering (orch-only, ~1h)
3. Add startup sweep to orch serve/daemon (orch-only, ~2h)
4. OpenCode Session.remove() process termination (fork change, ~2h)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does `opencode attach` self-terminate when its session is deleted via SSE event? (If yes, Finding 3 is less severe)
- Does OpenCode server restart send SIGTERM to attached bun processes? (Determines if restart-window zombies are a concern)
- Are there hidden subprocess spawns inside OpenCode server for headless sessions? (Current evidence says no, but not verified with actual headless agent run)

**Areas worth exploring further:**
- Whether OpenCode should expose per-session PID metadata via API (would simplify detection to a single API call)
- Whether overmind's `--auto-restart opencode` properly cleans up before restarting

**What remains unclear:**
- Exact breakdown of the 34 Feb 10 zombies between old-headless (pre-Feb-9) and tmux-spawned (evidence lost to reboot)

---

## Session Metadata

**Skill:** architect
**Model:** opus-4.6
**Workspace:** `.orch/workspace/og-arch-zombie-bun-processes-10feb-ce39/`
**Investigation:** `.kb/investigations/2026-02-10-design-zombie-bun-root-cause-architecture.md`
**Beads:** `bd show orch-go-21520`
