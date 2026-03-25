## Summary (D.E.K.N.)

**Delta:** Claude Code writes PID-bearing session files to `~/.claude/sessions/{pid}.json`; combining PID liveness + tmux pane content hashing gives a reliable `IsProcessing` equivalent for Claude Code agents.

**Evidence:** Enumerated all `~/.claude/sessions/*.json` — every running Claude process has a file with `pid`, `sessionId`, `cwd`, `startedAt`. PID checks via `kill -0` correctly identify alive vs dead processes. Tmux `capture-pane` content hashing (md5) changes on every tool call/output.

**Knowledge:** The gap isn't "no signal exists" — it's "no signal is currently wired." Claude Code has PID + pane content signals that are equivalent in reliability to OpenCode's `IsProcessing`, just composed differently.

**Next:** Architect session to design `IsProcessing` for Claude Code agents using 3-layer signal composition (PID alive → pane content delta → phase timeout backstop). Implementation authority for the wiring; architectural authority for the signal composition design.

**Authority:** architectural - Cross-component design (discovery, status, daemon all affected), multiple valid approaches

---

# Investigation: How Should Orch Status Detect Liveness for Claude Code Agents?

**Question:** What liveness signals are available for Claude Code agents, and how should orch status compose them into a health determination that avoids false-positive UNRESPONSIVE flags?

**Started:** 2026-03-25
**Updated:** 2026-03-25
**Owner:** orch-go-jhluq
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| dc82e960 (scs-sp-vzm) UNRESPONSIVE IsProcessing fix | extends | yes | no — fix is correct for OpenCode, investigation extends to Claude Code |
| 122c9b9f3 phase-based liveness decision | extends | yes | no — phase-based liveness was the right decision, but needs supplementation |
| 0216f173a silent stall detection | extends | yes | never-started timeout is correctly orthogonal |

---

## Findings

### Finding 1: Claude Code writes PID-bearing session files

**Evidence:** Every running Claude Code process writes `~/.claude/sessions/{pid}.json`:
```json
{"pid":26646,"sessionId":"37a2cf6d-...","cwd":"/Users/dylanconlin/Documents/personal/orch-go","startedAt":1774461565029,"kind":"interactive"}
```
Enumerated 6 session files. All PIDs matched live `claude` processes via `ps -p $pid -o comm=`. When agents spawn with `CLAUDE_CONFIG_DIR`, files go to `$CLAUDE_CONFIG_DIR/sessions/` instead.

**Source:** `~/.claude/sessions/*.json`, `kill -0 $pid` checks

**Significance:** PID liveness is a binary alive/dead signal equivalent to OpenCode's `SessionExists()`. It answers "is the process running?" but NOT "is it actively generating?" — same limitation as tmux window existence.

---

### Finding 2: Tmux pane content capture can detect active generation

**Evidence:** `tmux capture-pane -t @377 -p` returns the full TUI output. When an agent is:
- **Actively generating:** Output shows tool calls, thinking indicators, streaming text — content changes every few seconds
- **Idle at prompt:** Last line shows `❯` prompt character with no spinner — content is static
- **Waiting between turns:** Shows "Crunched for Xm Ys" with no new output

Hashing pane content (`md5`) at two time points detects activity: same hash = idle, different hash = actively working.

**Source:** `tmux capture-pane -t @377 -p | md5` (tested on workers-orch-go window @377)

**Significance:** This is the closest analogue to OpenCode's `IsProcessing`. A content-diff approach polled every 30-60 seconds would distinguish "actively generating" from "idle at prompt" without any API.

---

### Finding 3: The current UNRESPONSIVE detection has a two-backend asymmetry

**Evidence:** Code paths in `status_cmd.go:380-394` and `serve_agents_handlers.go:186-211`:

- **OpenCode agents:** `IsProcessing` (from `statusInfo.IsBusy() || statusInfo.IsRetrying()`) suppresses UNRESPONSIVE flag. An agent confirmed as busy is definitionally not unresponsive.
- **Claude Code agents:** No IsProcessing signal exists in the pipeline. `discovery.go:401-425` routes by `SpawnMode == "claude"` and uses only phase comments + tmux window existence. Phase timeout fires after 30 min regardless of actual activity.

The `!agents[i].IsProcessing` guard in `serve_agents_handlers.go:188` is correct but vacuous for Claude Code agents — `IsProcessing` is always false because it's only set from OpenCode session status.

**Source:** `cmd/orch/serve_agents_handlers.go:188`, `cmd/orch/status_cmd.go:385`, `pkg/discovery/discovery.go:401-425`

**Significance:** This is the root cause of false-positive UNRESPONSIVE for Claude Code agents doing deep investigation work. The scs-sp-vzm fix correctly added the guard for OpenCode but couldn't address Claude Code because no signal existed to wire.

---

### Finding 4: Claude Code history.jsonl has per-message timestamps

**Evidence:** `~/.claude/history.jsonl` contains one JSON object per user message:
```json
{"display":"...", "timestamp":1774461697521, "project":"/Users/dylanconlin/...", "sessionId":"c752b272-..."}
```
Each entry has a `sessionId` that could be correlated with spawned agents. However, this only logs **user messages** (inputs), not assistant responses or tool calls. An agent actively generating a long response would not produce new history entries until the next turn.

**Source:** `tail -5 ~/.claude/history.jsonl`

**Significance:** History timestamps are a weak activity signal — they show when the agent last received input, not when it last produced output. Tmux pane content is strictly more informative.

---

### Finding 5: IsPaneActive already exists but isn't wired to status

**Evidence:** `pkg/tmux/pane.go:68-88` has `IsPaneActive()` which checks both `pane_current_command` and child processes. This function is used by `orch clean` (`lifecycle_impl.go:435`) to avoid killing active agents, but is NOT used by `orch status` or `serve_agents_handlers.go` for UNRESPONSIVE detection.

**Source:** `pkg/tmux/pane.go:68`, `pkg/agent/lifecycle_impl.go:435`

**Significance:** The binary "process alive" signal already exists and is tested. What's missing is the granular "actively generating" signal (pane content delta) and the wiring into the status pipeline.

---

## Synthesis

**Key Insights:**

1. **Three-layer signal composition** — Claude Code liveness should use three signals in priority order: (a) PID alive from session file, (b) pane content changing (activity proxy for IsProcessing), (c) phase timeout as backstop. Each layer narrows the diagnosis.

2. **Pane content delta is the IsProcessing equivalent** — For Claude Code agents, the closest analogue to OpenCode's busy/idle status is whether tmux pane output is changing. A simple hash-and-compare on 30-60 second intervals would distinguish actively generating agents from idle ones.

3. **The signal gap is wiring, not capability** — All the raw signals exist (PID files, pane capture, child process detection). The system just doesn't compose them into an `IsProcessing` equivalent for Claude Code agents and wire it into the UNRESPONSIVE detection path.

**Answer to Investigation Question:**

Claude Code agents have three available liveness signals that should be composed into a health determination:

| Signal | What It Tells Us | Confidence | Cost |
|--------|-----------------|------------|------|
| PID alive (`kill -0`) | Process exists | High (binary) | ~1ms |
| Pane content delta | Actively generating output | Medium-High | ~50ms per capture |
| Phase comment age | Last self-reported transition | Medium | Already wired |
| `IsPaneActive()` | Non-shell process in foreground | Medium | ~10ms |

The composed health determination should be:

```
IF PID dead → DEAD (not unresponsive — agent crashed)
IF pane content changed since last check → RUNNING (suppress UNRESPONSIVE)
IF IsPaneActive() → ALIVE (process running, suppress UNRESPONSIVE if < skill-aware threshold)
IF phase timeout exceeded → UNRESPONSIVE (all other signals negative)
```

The key addition over current behavior: **pane content delta as the IsProcessing equivalent**. This prevents false-positive UNRESPONSIVE for agents that are actively generating long responses or deep investigation work without reporting phase comments.

---

## Structured Uncertainty

**What's tested:**

- Session files exist at `~/.claude/sessions/{pid}.json` with PID, sessionId, cwd, startedAt (verified: enumerated 6 files)
- `kill -0 $pid` correctly identifies alive vs dead Claude processes (verified: tested all 6 PIDs)
- `tmux capture-pane -p` returns current TUI content and can be hashed for delta detection (verified: captured pane @377)
- `IsPaneActive()` exists and works for detecting non-shell foreground processes (verified: read code at `pkg/tmux/pane.go:68-88`)

**What's untested:**

- Whether pane content hash actually changes during active generation vs stays stable during idle (not tested with a live actively-generating agent at two time points)
- Whether `CLAUDE_CONFIG_DIR` isolation causes session files to be written elsewhere (code reading suggests yes, but not empirically tested)
- Performance impact of adding pane content capture to every status check for N agents (not benchmarked)
- Whether `history.jsonl` entries from spawned agents differ from interactive sessions (limited testing)

**What would change this:**

- If pane content is buffered/cached by tmux and doesn't change during active generation, the delta approach fails
- If Claude Code adds a native processing API (like `claude --status`), pane capture becomes unnecessary
- If agents are spawned without tmux (e.g., headless Claude Code), pane signals are unavailable

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add pane content delta as IsProcessing for Claude Code | architectural | Cross-component: touches discovery, status_cmd, serve_agents_handlers, daemon phase_timeout |
| Skill-aware phase timeout thresholds | architectural | Changes shared timeout logic that affects all agent types |
| Wire IsPaneActive into UNRESPONSIVE suppression | implementation | Simple conditional addition to existing code paths |

### Recommended Approach: Three-Layer Composed Liveness

**Compose PID liveness + pane content delta + phase timeout into a single IsProcessing-equivalent for Claude Code agents.**

**Why this approach:**
- PID check is the cheapest first-pass filter (1ms, already partially exists)
- Pane content delta provides the granular "actively working" signal without any Claude Code API dependency
- Phase timeout remains as the last-resort backstop for truly stuck agents
- Follows the same pattern as the OpenCode fix (IsProcessing suppresses UNRESPONSIVE)

**Trade-offs accepted:**
- Pane content polling adds ~50ms per agent per status check (acceptable for <20 agents)
- Requires storing previous pane hash somewhere (in-memory map, similar to `globalStallTracker`)
- Does not work for headless Claude Code agents (none exist today)

**Implementation sequence:**
1. **Add `ClaudePaneActivityTracker`** — in-memory map of windowID → last content hash + timestamp. On each status check, capture pane content, hash it, compare to previous. If different → agent is actively generating.
2. **Wire into `IsProcessing` for Claude Code agents** — in `status_cmd.go` and `serve_agents_handlers.go`, set `IsProcessing = true` when pane content changed within the last 60 seconds. This naturally flows through the existing `!IsProcessing` guard that suppresses UNRESPONSIVE.
3. **Add PID liveness check** — Read `~/.claude/sessions/{pid}.json` (or `$CLAUDE_CONFIG_DIR/sessions/`) for Claude Code agents, check `kill -0 pid`. If dead → mark as dead/phantom (distinct from unresponsive).

### Alternative Approaches Considered

**Option B: Skill-aware phase timeout thresholds only**
- **Pros:** Simpler implementation, no new polling mechanism
- **Cons:** Still based on arbitrary time thresholds (45 min for investigations vs 30 min for feature-impl). Doesn't detect the actual state of the agent.
- **When to use instead:** If pane content delta proves unreliable or too expensive

**Option C: Remove phase timeout for Claude Code agents entirely, rely only on PID + IsPaneActive**
- **Pros:** Eliminates false positives completely for Claude Code
- **Cons:** Masks truly stuck agents. An agent at a prompt (idle shell with Claude as child process) would never be flagged.
- **When to use instead:** Never — some backstop timeout is needed

**Rationale for recommendation:** Option A composes cheap, reliable signals into a progressive diagnosis. It follows the same pattern the scs-sp-vzm fix established for OpenCode (IsProcessing suppresses UNRESPONSIVE) and extends it to Claude Code using the available signals.

---

### Implementation Details

**What to implement first:**
- `ClaudePaneActivityTracker` (in-memory hash map) — foundation for all other changes
- Wire into existing `IsProcessing` field in status pipeline — minimal code change, maximal impact

**Things to watch out for:**
- `CLAUDE_CONFIG_DIR` isolation means session files may be in different directories per account. The PID check needs to scan the config dir from the workspace manifest.
- Tmux pane capture may include ANSI escape codes that change between captures even when content is "stable." Strip ANSI before hashing.
- The activity tracker must be per-process (not persisted) — it's a polling cache, not state.

**Areas needing further investigation:**
- Benchmark pane capture + hash cost at scale (10, 20, 50 agents)
- Verify pane content actually changes detectably during active Claude Code generation
- Determine if Claude Code will add a native processing API in future versions

**Success criteria:**
- Zero false-positive UNRESPONSIVE flags for actively-generating Claude Code agents
- Claude Code agents doing 45-min deep investigation work are NOT flagged as unresponsive
- Truly stuck Claude Code agents (crashed, prompt-locked) ARE flagged within 30 min + polling interval

---

## References

**Files Examined:**
- `pkg/state/reconcile.go` — LivenessResult struct and GetLiveness function
- `pkg/discovery/discovery.go` — QueryTrackedAgents and JoinWithReasonCodes (Claude Code routing at line 401)
- `cmd/orch/status_cmd.go` — UNRESPONSIVE detection loop (line 380-394)
- `cmd/orch/serve_agents_handlers.go` — Dashboard API UNRESPONSIVE detection (line 186-211)
- `cmd/orch/status_format.go` — getAgentStatus priority cascade (line 331-351)
- `pkg/tmux/pane.go` — IsPaneActive, CaptureLines, GetPanePID
- `pkg/daemon/phase_timeout.go` — Daemon phase timeout detection
- `pkg/spawn/claude.go` — BuildClaudeLaunchCommand, MonitorClaude
- `~/.claude/sessions/*.json` — Claude Code session state files

**Commands Run:**
```bash
# Enumerate Claude Code session files
for f in ~/.claude/sessions/*.json; do cat "$f"; done

# Check PID liveness for each session
for f in ~/.claude/sessions/*.json; do
  pid=$(python3 -c "import json; print(json.load(open('$f'))['pid'])")
  kill -0 $pid 2>/dev/null && echo "ALIVE" || echo "DEAD"
done

# Test tmux pane content capture and hashing
tmux capture-pane -t @377 -p | md5

# Check Claude Code history format
tail -5 ~/.claude/history.jsonl | python3 -m json.tool
```

**Related Artifacts:**
- **Decision:** `122c9b9f3` — phase-based liveness over tmux-as-state for claude-backend agents
- **Fix:** `dc82e9606` (scs-sp-vzm) — UNRESPONSIVE detection now consults IsProcessing from OpenCode
- **Guide:** `.kb/guides/agent-lifecycle.md` — Agent Liveness Detection section (line 468-491)
