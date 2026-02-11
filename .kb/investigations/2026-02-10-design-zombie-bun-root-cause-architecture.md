## Summary (D.E.K.N.)

**Delta:** Zombie bun processes recur because three independent failure modes compound: (1) orphan detector pattern `"run --attach"` doesn't match current tmux command format `opencode attach`, (2) process ledger is never populated for tmux or headless spawns, (3) OpenCode's `Session.remove()` deletes storage but doesn't kill the attached bun process.

**Evidence:** Verified all three gaps against source code and live system — process-ledger.jsonl is 0 bytes, `orphans.go:45` literal doesn't match `opencode attach` process args, OpenCode `session/index.ts:353` remove function only deletes storage.

**Knowledge:** The problem is an integration mismatch at the orch-OpenCode boundary: orch owns session lifecycle but OpenCode owns process lifecycle, and neither signals the other at termination time. Prior investigations correctly identified the pattern but misattributed it to "headless spawns" — headless spawns post-Feb-9 are HTTP-only and don't create bun processes. The actual zombie source is tmux-spawned `opencode attach` processes.

**Next:** Implement two-track fix: (1) orch-side: replace command-format matching with session-aware process detection, (2) OpenCode-side: add process termination to Session.remove(). Both are needed for defense in depth.

**Authority:** architectural - Crosses orch process detection, orch spawn, OpenCode session lifecycle, and OpenCode fork changes.

---

# Investigation: Why Zombie Bun Processes Keep Recurring

**Question:** Why do zombie bun processes keep recurring (3 times: Jan 31=26, Feb 7=13, Feb 10=34) despite multiple fix attempts? Is the problem solvable within orch alone, or does the architecture need to change at the orch-OpenCode boundary? What would a fix look like that CANNOT regress?

**Started:** 2026-02-10
**Updated:** 2026-02-10
**Owner:** architect worker (orch-go-21520)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Defect Class:** integration-mismatch

**Patches-Decision:** `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md`
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-02-08-inv-design-process-lifecycle-cleanup-prevent.md` | deepens | Yes — verified findings 1-3 against current code | Finding 3 partially incorrect: "headless spawn starts an OpenCode subprocess" is no longer true post-Feb-9 |
| `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` | extends | Yes — two-tier principle is sound but both tiers are now non-functional | N/A |
| `.kb/models/system-reliability-feb2026.md` | confirms | Yes — C2 constraint (process creation requires cleanup) not yet enforced | N/A |

---

## Findings

### Finding 1: The orphan detector's command format filter is stale — doesn't match current tmux-spawned processes

**Evidence:** `orphans.go:45` checks `strings.Contains(line, "run --attach")`. But tmux spawns now use `BuildOpencodeAttachCommand` which generates `opencode attach <url> --dir <project>`. At the OS level, this becomes `bun run --conditions=browser ./src/index.ts attach http://...` — the literal `"run --attach"` never appears because `run` and `attach` aren't adjacent.

The format change happened when tmux spawns switched from `BuildSpawnCommand` (`opencode run --attach <url>`) to `BuildOpencodeAttachCommand` (`opencode attach <url>`). The orphan detector was never updated.

**Source:** `pkg/process/orphans.go:45`, `pkg/tmux/tmux.go:225`, `pkg/opencode/client.go:178-197`

**Significance:** Tier 2 orphan detection is completely blind to all current tmux-spawned bun processes. This is the immediate cause of zombie accumulation for tmux-spawned agents.

---

### Finding 2: The process ledger is never populated for tmux or current headless spawns

**Evidence:** The ledger write at `spawn_execute.go:218` is gated on `result.cmd != nil`. Three paths skip this:

1. **Headless spawns (post-Feb-9):** `startHeadlessSession` now uses HTTP API only (`CreateSession` + `sendHeadlessPrompt`). Returns `headlessSpawnResult{SessionID: sessionID}` with `cmd: nil`. Confirmed: headless switched to HTTP-only in commit `6890004b` (Feb 9).

2. **Tmux spawns:** `runSpawnTmux` sends `opencode attach ...` to a tmux window via `tmux.SendKeys`. No `exec.Cmd` exists — the bun process is started by tmux's shell, not by orch. PID is never known to orch.

3. **Confirmed empty:** `~/.orch/process-ledger.jsonl` is 0 bytes on the live system.

**Source:** `cmd/orch/spawn_execute.go:218-239`, `cmd/orch/spawn_execute.go:361-383`, `cmd/orch/spawn_execute.go:434-510`

**Significance:** Tier 1 ledger-backed cleanup is a no-op because the ledger has no entries. The ledger was designed for the old headless CLI path (pre-Feb-9) which no longer exists.

---

### Finding 3: OpenCode's Session.remove() deletes storage but does not terminate the attached bun process

**Evidence:** When `orch complete` calls `client.DeleteSession(sessionID)`, OpenCode's `Session.remove()` (opencode/src/session/index.ts:353) removes children, unshares, deletes messages/parts/storage, and publishes `Event.Deleted`. It does NOT:
- Signal or kill any attached bun processes
- Send any termination signal to `opencode attach` TUI clients
- Terminate any running agent work

Meanwhile, `orch complete`'s `deleteSessionAndProcess` (complete_cleanup.go:96) tries to terminate the process via `spawn.ReadProcessID(workspacePath)`, but this file is never written for tmux spawns.

**Source:** `opencode/packages/opencode/src/session/index.ts:353-374`, `cmd/orch/complete_cleanup.go:90-118`

**Significance:** When an agent completes and orch calls DeleteSession, the OpenCode session is removed from storage but the bun process in the tmux window continues running indefinitely. This is the lifecycle gap at the orch-OpenCode boundary.

---

### Finding 4: The HANDOFF diagnosis misattributes the zombie source to "headless spawns"

**Evidence:** The Feb 10 HANDOFF states "Both tiers of orphan detection are blind to headless-spawned processes (the PRIMARY spawn path)." This was correct pre-Feb-9 when headless spawns created bun subprocesses via `BuildSpawnCommand`. But commit `6890004b` (Feb 9) switched headless to HTTP-only — no bun subprocess is created.

Post-Feb-9, the zombie source is:
- **Tmux-spawned agents** (`opencode attach` → bun process) — these are orchestrators, manual spawns, `--tmux` flag spawns
- **Residual pre-Feb-9 headless spawns** — from the 12+ days of uptime (system up since Jan 29)

The default spawn path for daemon-driven workers is headless, which post-Feb-9 is HTTP-only and cannot produce zombies.

**Source:** Commit `6890004b`, `cmd/orch/spawn_pipeline.go:738-739`, `cmd/orch/spawn_execute.go:361-383`

**Significance:** Future fixes must target tmux-spawned processes, not headless. The Feb 9 HTTP migration accidentally solved the headless zombie path but introduced a new gap: headless spawns now have zero process tracking (no PID, no ledger entry, no ability to terminate).

---

### Finding 5: What was actually implemented vs. left undone from prior recommendations

**Evidence:**

**Implemented:**
- ✅ Process ledger (`pkg/process/ledger.go`) — full JSONL-backed ledger with `Sweep`, `SweepWithKill`, `Reconcile` operations
- ✅ Daemon orphan reaper (`pkg/daemon/orphan_reaper.go`) — two-tier: ledger-backed Tier 1 + title-matching Tier 2
- ✅ `orch complete` cleanup — deletes session, attempts PID termination, removes ledger entry, closes tmux window
- ✅ `orch clean --processes` — manual orphan cleanup command
- ✅ LRU/TTL instance eviction in OpenCode fork

**Left undone:**
- ❌ Startup sweep at `orch serve` launch — never implemented
- ❌ Ledger population for tmux spawns — never implemented (PID unknown)
- ❌ Orphan detection pattern update for `opencode attach` format — never updated
- ❌ OpenCode process termination on session delete — never implemented
- ❌ C1/C2 automated enforcement (from unbounded resource constraints decision) — never shipped

**Source:** Comparison of `.kb/investigations/2026-02-08-inv-design-process-lifecycle-cleanup-prevent.md` recommendations against current codebase.

**Significance:** The infrastructure for cleanup exists but is not wired to the actual process creation paths. The gap is not missing code but misaligned code: detection matches a format that no longer exists, the ledger tracks a spawn path that was removed.

---

## Synthesis

**Key Insights:**

1. **Three independent failure modes compound** — Any one of: (a) stale detection pattern, (b) empty ledger, (c) no session→process termination coupling would allow zombies. All three are active simultaneously, which is why partial fixes haven't helped.

2. **The Feb 9 HTTP migration was an accidental partial fix** — By removing bun subprocesses from the headless path, it eliminated the largest zombie source. But it left tmux spawns completely unmanaged and removed the only path that populated the process ledger.

3. **Command-format-based detection is structurally fragile** — The orphan detector broke when `opencode run --attach` changed to `opencode attach`. Any fix based on string-matching process args will regress on the next format change.

4. **The prior recommendations were architecturally sound but targeted the wrong layer** — "Lineage-backed two-tier cleanup" is the right design, but the recommendations assumed headless spawns create processes (no longer true) and didn't account for the tmux path where orch never knows the PID.

**Answer to Investigation Question:**

**Is the problem solvable within orch alone?** Two of three failure modes are fixable within orch (detection pattern, ledger population). The third (session delete → process termination) ideally requires an OpenCode change, but orch can work around it by killing the tmux window (which kills the bun process) during `orch complete`.

**What would a fix look like that CANNOT regress?** The fix must be session-based, not command-format-based. Instead of matching bun processes by command args, the system should:
1. Use OpenCode sessions API as the source of truth for what's "active"
2. Discover bun processes by project directory path (stable across format changes)
3. Cross-reference: any bun process for a project directory without a matching active session = orphan

This design is invariant to command format changes because it never inspects command args beyond identifying the process as bun + project directory.

---

## Structured Uncertainty

**What's tested:**

- ✅ Process ledger is empty (`~/.orch/process-ledger.jsonl` = 0 bytes, verified on live system)
- ✅ `orphans.go:45` literal `"run --attach"` doesn't match `opencode attach` format (code inspection + current `ps` output showing `bun run --conditions=browser ./src/index.ts /path` with no `--attach`)
- ✅ OpenCode `Session.remove()` does NOT kill bun processes (code inspection of opencode/src/session/index.ts:353-374)
- ✅ Headless spawns post-Feb-9 are HTTP-only, `startHeadlessSession` returns `cmd: nil` (code inspection + commit 6890004b)
- ✅ Tmux spawns use `SendKeys` not `exec.Command`, PID never captured (code inspection of spawn_execute.go:434-510)

**What's untested:**

- ⚠️ Whether `opencode attach` bun processes detect session deletion via SSE/Bus events and self-terminate (not tested — requires spawning an agent and deleting its session while monitoring the process)
- ⚠️ Whether OpenCode server restart properly terminates all attached bun processes (not tested — requires simulating server restart with active agents)
- ⚠️ The exact breakdown of Feb 10's 34 zombies between old-headless vs tmux-spawned (system was rebooted, evidence lost)

**What would change this:**

- If `opencode attach` does detect session deletion and self-terminates, then Finding 3 is less severe (only matters when OpenCode server restarts, not when sessions complete normally)
- If OpenCode exposes PID metadata per session via API, the detection problem becomes trivial — query API, get PIDs, cross-reference with live processes
- If daemon starts using tmux-mode spawns instead of headless, the zombie problem would return at scale

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Replace command-format detection with session-aware detection | architectural | Changes detection contract across daemon, orphan reaper, and process package |
| Add `orch complete` tmux-window kill as process termination | implementation | Single-component change, reversible, already has tmux cleanup in complete_cleanup.go |
| OpenCode: add process termination to Session.remove() | architectural | Cross-boundary change to the OpenCode fork |
| Startup sweep at orch serve / daemon init | architectural | Adds new lifecycle phase to both serve and daemon |

### Recommended Approach ⭐

**Session-aware process detection with defense-in-depth termination** — Replace command-format matching with session-ID-based cross-referencing, and add process termination at both `orch complete` (immediate) and OpenCode session deletion (boundary fix).

**Why this approach:**
- Session IDs are stable across command format changes (non-regressive)
- Two-layer termination (orch + OpenCode) provides defense in depth
- Leverages existing infrastructure (OpenCode sessions API, state DB, tmux cleanup)

**Trade-offs accepted:**
- Requires OpenCode fork change (but fork is already maintained)
- Session-aware detection requires OpenCode server to be running (falls back to broader matching when server is down)

**Implementation sequence:**

1. **Fix immediate gap — update `FindAgentProcesses` detection** (orch-only, 1h)
   - Replace `"run --attach"` filter with broader detection: match bun processes containing `"./src/index.ts"` (the OpenCode entrypoint, stable across all spawn formats)
   - This immediately makes Tier 2 detection functional again
   - Why first: stops the bleeding — daemon reaper starts finding orphans

2. **Fix `orch complete` — ensure bun process dies** (orch-only, 1h)
   - After deleting the OpenCode session, kill the tmux window BEFORE archiving workspace
   - The tmux window kill sends SIGHUP to the bun process, which terminates it
   - `cleanupTmuxWindow` already exists in `complete_cleanup.go:81` — verify it runs before session deletion, not after
   - Why second: fixes the most common zombie creation path (completed agents)

3. **Add startup sweep** (orch-only, 2h)
   - At `orch serve` startup and daemon initialization, sweep all bun+opencode processes against active sessions
   - Reconcile: kill any bun process whose project directory doesn't match an active session
   - Why third: closes the restart-window gap where zombies accumulate between tier-2 reaper cycles

4. **OpenCode boundary fix** (fork change, 2h)
   - Add process tracking to OpenCode's session lifecycle: when `opencode attach` connects, record the attached process info
   - On `Session.remove()`, signal attached processes to terminate
   - Why fourth: provides the structural fix that prevents the category, but orch-side fixes provide immediate relief

### Alternative Approaches Considered

**Option B: Pure ledger-based approach (populate ledger for all spawn paths)**
- **Pros:** Keeps current architecture, adds missing data
- **Cons:** Tmux spawns' PIDs are fundamentally unknowable at spawn time (process is started by tmux shell, not orch). Would require post-spawn PID discovery, which is racy.
- **When to use instead:** If OpenCode exposes per-session PID metadata via API

**Option C: Kill all bun processes on every reaper cycle**
- **Pros:** Simple, guaranteed to clear zombies
- **Cons:** Would kill legitimate running agents. Unacceptable false-positive rate.
- **When to use instead:** Never in production; only as emergency manual cleanup

**Rationale for recommendation:** Option A (session-aware detection) is the only approach that is (a) non-regressive to format changes, (b) doesn't require solving the tmux-PID-discovery problem, and (c) provides defense in depth through two termination layers.

---

### Implementation Details

**What to implement first:**
- Update `FindAgentProcesses` in `pkg/process/orphans.go` to match `"./src/index.ts"` instead of `"run --attach"`
- Verify `cleanupTmuxWindow` ordering in `orch complete` relative to session deletion

**Things to watch out for:**
- ⚠️ Don't kill the OpenCode server process itself — it also matches `bun` + `./src/index.ts`. Distinguish by: server has `serve --port` in args; agents have project directory paths
- ⚠️ Cross-project contamination: only kill bun processes whose directory matches orch-managed project directories
- ⚠️ Startup sweep race: new spawns during sweep could be falsely classified as orphans. Use a grace period (skip processes started within last 30s)

**Areas needing further investigation:**
- Whether `opencode attach` detects session deletion via SSE events (if yes, the boundary fix may be partially redundant)
- Whether OpenCode's instance eviction (LRU/TTL) terminates associated bun processes when evicting idle instances
- The exact process tree when OpenCode server handles headless sessions — are there hidden subprocess spawns?

**Success criteria:**
- ✅ Zero zombie bun processes after 7 days of mixed headless + tmux workload
- ✅ `orch clean --processes` reports 0 orphans on healthy system
- ✅ Daemon orphan reaper successfully identifies and kills stale tmux-spawned bun processes
- ✅ `orch complete` leaves zero residual bun processes for the completed agent

---

## Decision Fork Analysis

### Fork 1: Detection Strategy

**Options:**
- A: Fix string match (`"run --attach"` → `"./src/index.ts"`) — covers current formats
- B: Session-aware detection (query OpenCode API, cross-reference with ps) — format-independent
- C: Both A+B (A as fallback when server is down, B as primary)

**Substrate says:**
- Principle: "Coherence over patches" — if 5+ fixes hit the same area, redesign not patch
- Decision: Two-tier cleanup requires both tiers to function

**Recommendation:** Option C — Use session-aware detection (B) as primary, with broader process detection (A) as fallback when OpenCode server is unavailable. This provides both immediate fix and structural non-regression.

### Fork 2: Where Termination Happens

**Options:**
- A: orch-only (kill tmux window + broader detection)
- B: OpenCode-only (Session.remove kills process)
- C: Both (defense in depth)

**Substrate says:**
- Principle: "Escape hatches" — critical paths need independent secondary paths
- Model: system-reliability-feb2026 — two-tier cleanup is required, not either/or

**Recommendation:** Option C — orch terminates the process it can reach (tmux window), OpenCode terminates the process it owns (attached bun). Neither depends on the other succeeding.

### Fork 3: Scope of This Fix (Strategic)

**Options:**
- A: Fix detection + termination only (tactical)
- B: Also implement startup sweep + ledger population (comprehensive)
- C: Also add OpenCode boundary changes (holistic)

**Substrate says:**
- Decision: unbounded-resource-consumption-constraints — C2 requires process creation to have matching cleanup
- History: 3 zombie incidents prove tactical fixes don't hold

**Recommendation:** Option C — implement all layers. The pattern of recurrence (Jan 31, Feb 7, Feb 10) proves that partial fixes leave gaps that compound. The effort is modest (~6h total) and the failure mode is severe (2.5GB RAM, system freeze, lost work).

---

## References

**Files Examined:**
- `pkg/process/orphans.go` — Tier 2 orphan detection, verified stale `"run --attach"` filter
- `pkg/process/ledger.go` — Process ledger implementation, verified it's functional but unpopulated
- `pkg/daemon/orphan_reaper.go` — Daemon reaper two-tier logic, verified both tiers are non-functional
- `cmd/orch/spawn_execute.go` — All spawn backends, verified ledger gate and HTTP-only headless
- `cmd/orch/spawn_pipeline.go:708-740` — Dispatch logic, verified default-headless routing
- `cmd/orch/complete_cleanup.go` — Completion cleanup, verified session deletion and PID termination
- `pkg/tmux/tmux.go:210-232` — `BuildOpencodeAttachCommand`, verified `opencode attach` format
- `pkg/opencode/client.go:178-197` — `BuildSpawnCommand`, verified old `run --attach` format
- `opencode/src/session/index.ts:353-374` — Session.remove(), verified no process termination
- `opencode/src/project/instance.ts` — Instance lifecycle, verified eviction is memory-only
- `.orch/HANDOFF.md` — Feb 10 handoff diagnosis, verified partially correct but misattributed source

**Commands Run:**
```bash
# Verify process ledger is empty
cat ~/.orch/process-ledger.jsonl | wc -l  # → 0

# Check current bun processes and parent PIDs
ps -eo pid,ppid,args | grep bun | grep -v grep
# Showed: PID 49621 (bun run --conditions=browser ./src/index.ts /path) with PPID 48585 (zsh)
# No "run --attach" in command args

# Check OpenCode sessions API structure
curl -s http://127.0.0.1:4096/session | python3 -m json.tool
# Returns session metadata (id, title, directory) but no PID information

# Verify headless HTTP-only switch date
git log --oneline --format="%h %ai %s" -- cmd/orch/spawn_execute.go
# 6890004b 2026-02-09 — "headless spawn uses HTTP API"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-14-two-tier-cleanup-pattern.md` — Establishes two-tier cleanup as required pattern
- **Decision:** `.kb/decisions/2026-02-07-unbounded-resource-consumption-constraints.md` — C2 process lifecycle enforcement
- **Investigation:** `.kb/investigations/2026-02-08-inv-design-process-lifecycle-cleanup-prevent.md` — Prior lifecycle design (this investigation deepens and partially corrects it)
- **Model:** `.kb/models/system-reliability-feb2026.md` — System reliability context
- **HANDOFF:** `.orch/HANDOFF.md` — Feb 10 diagnostic (partially correct, this investigation corrects the attribution)

---

## Investigation History

**[2026-02-10 ~17:00]:** Investigation started
- Initial question: Why do zombie bun processes keep recurring despite multiple fix attempts? Is the problem solvable within orch alone?
- Context: 3rd zombie recurrence (34 processes on Feb 10), system freeze requiring reboot, prior investigation recommended lineage-backed cleanup but zombies returned.

**[2026-02-10 ~17:30]:** Core failure modes identified
- Discovered 3 independent failure modes: stale detection pattern, empty ledger, no session→process termination coupling
- Key breakthrough: Feb 9 HTTP migration means headless spawns no longer create bun processes — prior attribution to "headless spawns" is outdated

**[2026-02-10 ~17:45]:** OpenCode boundary analysis completed
- Verified Session.remove() in OpenCode fork does not kill attached processes
- Identified that `opencode attach` format change broke orphan detection
- Confirmed tmux spawns never populate the process ledger

**[2026-02-10 ~18:00]:** Investigation completed
- Status: Complete
- Key outcome: Three independent failure modes, two fixable within orch, one requires OpenCode boundary change. Session-aware detection (not command-format matching) is the non-regressive fix.
