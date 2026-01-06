## Summary (D.E.K.N.)

**Delta:** Synthesized 11 tmux investigations into an authoritative guide - tmux is a mature, opt-in interactive mode with proven concurrent spawning, session ID resolution, and TUI detection.

**Evidence:** Analyzed investigations from Dec 20-23 2025 covering: architecture decisions, concurrent spawn testing (alpha/beta/gamma/delta/epsilon/zeta), session ID resolution fixes, attach mode, fallback mechanisms, and SIGKILL debugging.

**Knowledge:** Tmux mode coexists with headless (default) and inline modes; key learnings: fire-and-forget scales to 6+ concurrent agents, session ID capture is unreliable (use API lookup instead), launchd KeepAlive can cause SIGKILL on binary updates.

**Next:** Guide created at `.kb/guides/tmux-spawn-guide.md` - archive superseded investigations.

---

# Investigation: Synthesize 11 Tmux Investigations

**Question:** What patterns and authoritative guidance can be extracted from the 11 tmux-related investigations?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-feat-synthesize-tmux-investigations-06jan agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** The following investigations are now consolidated:
- 2025-12-20-inv-migrate-orch-go-tmux-http.md
- 2025-12-20-inv-tmux-concurrent-delta.md
- 2025-12-20-inv-tmux-concurrent-epsilon.md
- 2025-12-20-inv-tmux-concurrent-zeta.md
- 2025-12-21-debug-orch-send-fails-silently-tmux.md
- 2025-12-21-inv-add-tmux-fallback-orch-status.md
- 2025-12-21-inv-add-tmux-flag-orch-spawn.md
- 2025-12-21-inv-implement-attach-mode-tmux-spawn.md
- 2025-12-21-inv-tmux-spawn-killed.md
- 2025-12-22-debug-orch-send-fails-silently-tmux.md
- archived/2025-12-23-inv-test-tmux-spawn.md

---

## Findings

### Finding 1: Tmux is now opt-in, HTTP API is default

**Evidence:** Investigation `2025-12-20-inv-migrate-orch-go-tmux-http.md` documented that `spawnTmux` flag defaults to `false` (main.go:82). The default spawn path is `runSpawnHeadless()` which uses HTTP API. Tmux requires explicit `--tmux` flag.

**Source:** cmd/orch/main.go:644-656, confirmed in 2025-12-21-inv-add-tmux-flag-orch-spawn.md

**Significance:** The architecture decision to make headless the default was validated. Tmux mode is for interactive monitoring only, not for automation/daemon use.

---

### Finding 2: Fire-and-forget spawn scales to 6+ concurrent agents

**Evidence:** Three concurrent test investigations (delta, epsilon, zeta) confirmed:
- Delta checkin at 08:24:51, Beta at 08:24:14, Alpha at 08:23:55
- Epsilon observed 18+ concurrent sessions, 53 workspaces, 29 processes
- Zeta confirmed window 12 allocation in workers-orch-go session
- No workspace conflicts or race conditions detected

**Source:** 
- 2025-12-20-inv-tmux-concurrent-delta.md
- 2025-12-20-inv-tmux-concurrent-epsilon.md
- 2025-12-20-inv-tmux-concurrent-zeta.md

**Significance:** Fire-and-forget pattern is production-ready. Workspace isolation via unique naming prevents conflicts. Upper concurrency limit not tested but >6 is confirmed.

---

### Finding 3: Session ID capture from tmux spawns is unreliable

**Evidence:** Only 1 of 100+ workspaces had `.session_id` file. `FindRecentSessionWithRetry` often fails due to timing, and failure is silently ignored. This created `orch send` failures.

**Source:** 
- 2025-12-21-debug-orch-send-fails-silently-tmux.md (Finding 2)
- 2025-12-22-debug-orch-send-fails-silently-tmux.md

**Significance:** Session ID lookup must use fallback chain: workspace files → OpenCode API sessions by title → tmux windows by beads ID. Never assume session ID is available.

---

### Finding 4: OpenCode attach mode enables dual TUI/API access

**Evidence:** Investigation documented that standalone mode (`opencode {project_dir}`) creates ephemeral server invisible to main server. Attach mode (`opencode attach {server_url}`) connects to shared server, enabling API access while showing TUI.

**Source:** 2025-12-21-inv-implement-attach-mode-tmux-spawn.md

**Significance:** `runSpawnTmux` now uses `BuildOpencodeAttachCommand` for proper dual access. This enables `orch tail`, `orch send`, and `orch question` to work with tmux-spawned agents.

---

### Finding 5: Tmux fallback provides resilience for status/tail/question

**Evidence:** Added `ListWorkersSessions` to pkg/tmux, updated `runTail` with tmux fallback (capture-pane), updated `runStatus` with tmux discovery. When OpenCode API fails, commands still work via tmux.

**Source:** 2025-12-21-inv-add-tmux-fallback-orch-status.md

**Significance:** Tmux windows are source of truth for active agents. Registry provides metadata (Beads ID, Skill) that can be reconciled with window names.

---

### Finding 6: Two SIGKILL root causes identified and fixed

**Evidence:** Exit code 137 had two causes:
1. **Stale binary** - `~/bin/orch` didn't have latest code; fixed by post-commit hook for auto-rebuild
2. **launchd KeepAlive** - `orch serve` daemon using `~/bin/orch` with KeepAlive caused SIGKILL when binary replaced; fixed by using `build/orch` for daemon

**Source:** 2025-12-21-inv-tmux-spawn-killed.md

**Significance:** Prevention measures: git post-commit hook, `orch version --source` for staleness detection, separate daemon binary path.

---

### Finding 7: OpenCode TUI readiness detection works correctly

**Evidence:** `IsOpenCodeReady` function checks for:
- Prompt box (`┃`) AND
- Either agent selector OR command hints

All test cases passed: empty pane (false), shell only (false), loading (false), ready with prompt box + agent (true).

**Source:** archived/2025-12-23-inv-test-tmux-spawn.md (Finding 3)

**Significance:** TUI readiness detection is reliable. Integration tests confirmed full workflow works.

---

## Synthesis

**Key Insights:**

1. **Three spawn modes coexist with different purposes**
   - **Headless (default):** Daemon/automation use, HTTP API only, no TUI overhead
   - **Tmux (`--tmux`):** Interactive monitoring, TUI visible in tmux window, API via attach mode
   - **Inline (`--inline`):** Blocking with TUI, for debugging or quick tasks

2. **Session ID resolution requires fallback chain** - Direct session ID capture fails often for tmux spawns. The proven pattern is: try workspace `.session_id` → search OpenCode API by title → search tmux windows by beads ID → fall back to tmux capture-pane.

3. **Fire-and-forget enables true parallelism** - By not blocking on spawn confirmation, orchestrator can dispatch multiple agents simultaneously. Workspace isolation prevents conflicts. Monitoring happens via SSE events or status polling.

4. **Tmux is source of truth for active interactive agents** - When API/registry are out of sync, tmux windows provide ground truth. Window names contain beads ID for matching.

5. **Binary version mismatch is a recurring trap** - Multiple investigations hit issues from stale binaries. Prevention measures (post-commit hooks, version command) are essential.

**Answer to Investigation Question:**

The 11 tmux investigations represent a complete evolution from "should we add tmux mode?" to "tmux mode is production-ready with known patterns and pitfalls." Key outcomes:
- Architecture settled on tmux as opt-in interactive mode
- Concurrent spawning validated to 6+ agents
- Session ID resolution pattern established (fallback chain)
- Attach mode enables dual TUI/API access
- Multiple failure modes debugged and fixed (SIGKILL, silent send failures)
- Comprehensive testing confirmed reliability

---

## Structured Uncertainty

**What's tested:**

- ✅ Concurrent spawn scales to 6+ agents (validated via alpha/beta/gamma/delta/epsilon/zeta tests)
- ✅ Attach mode provides dual TUI/API access (unit tests pass, integration verified)
- ✅ Session ID fallback chain works (resolveSessionID tested with various inputs)
- ✅ TUI readiness detection is accurate (IsOpenCodeReady tested with multiple scenarios)
- ✅ Tmux fallback for status/tail/question works (smoke-tested)

**What's untested:**

- ⚠️ Maximum concurrency limit (tested up to 18+ sessions, upper bound unknown)
- ⚠️ Resource consumption patterns at scale (memory/CPU per spawn not measured)
- ⚠️ Long-term stability under sustained concurrent load (point-in-time tests only)
- ⚠️ Performance with >50 concurrent spawns

**What would change this:**

- Finding would be wrong if concurrent spawns >20 cause resource exhaustion
- Finding would be wrong if attach mode fails in newer OpenCode versions
- Finding would be wrong if tmux window limits cause spawn failures

---

## Implementation Recommendations

### Recommended Approach ⭐

**Create authoritative tmux guide** - Consolidate learnings into `.kb/guides/tmux-spawn-guide.md`

**Why this approach:**
- Single authoritative reference vs 11 scattered investigations
- Matches "10+ investigations → guide" synthesis pattern from kb context
- Provides quick answers for future tmux questions

**Trade-offs accepted:**
- Original investigations preserved for deep-dive/historical context
- Guide is summary, not exhaustive documentation

**Implementation sequence:**
1. Create guide with architecture overview, usage patterns, troubleshooting
2. Archive or mark superseded investigations
3. Update orchestrator skill to reference guide

---

## References

**Files Examined:**
- 11 tmux-related investigations in `.kb/investigations/`
- pkg/tmux/tmux.go - tmux package implementation
- cmd/orch/main.go - spawn command implementations

**Related Artifacts:**
- **Guide:** `.kb/guides/tmux-spawn-guide.md` (created by this investigation)
- **Decision:** kn-34d52f - "orch-go tmux spawn is fire-and-forget - no session ID capture"

---

## Investigation History

**2026-01-06:** Investigation started
- Initial question: What patterns can be extracted from 11 tmux investigations?
- Context: Topic accumulated 11 investigations per `kb synthesize` recommendation

**2026-01-06:** All 11 investigations read and analyzed
- Identified 7 key findings across architecture, concurrency, session resolution, attach mode, fallback, debugging, and testing

**2026-01-06:** Investigation completed
- Status: Complete
- Key outcome: Created authoritative guide, identified patterns, no contradictions found
