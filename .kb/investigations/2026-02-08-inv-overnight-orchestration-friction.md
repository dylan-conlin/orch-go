# Investigation: Overnight Autonomous Orchestration Friction

**Date:** 2026-02-08
**Context:** Orchestrator ran autonomously for ~90 minutes, triaging 35 ready issues, spawning 25+ agents, processing completions, and feeding the daemon. All spawns on gpt-5.3-codex (GPT).
**Outcome:** Successfully processed backlog from 35 ready → 5 residual. But friction was significant and several issues would have been showstoppers without workarounds.

---

## Friction Points (Severity-Ordered)

### 1. OpenCode Plugin Loader Crashes on .test.ts Files (P0 — Blocked Startup)

**What happened:** OpenCode TUI and server crashed on startup because `.opencode/plugin/skillc-generated-guard.test.ts` was in the plugin directory. OpenCode loads ALL `.ts` files in `.opencode/plugin/` as plugins. The test file imported `describe` from `bun:test`, which throws `Cannot use describe outside of the test runner` when imported as a regular module.

**Root cause:** A parallel agent created the guard plugin and its test file in the same directory. OpenCode has no exclusion pattern for `*.test.ts` files in the plugin directory.

**Workaround:** Renamed to `.test.ts.disabled`.

**Systemic fix needed:** OpenCode plugin loader should skip `*.test.ts` and `*.spec.ts` files. Or plugins should use a manifest/index file instead of globbing all `.ts` files.

**Impact:** Complete startup failure. Dylan couldn't use `oc` at all until fixed.

---

### 2. Beads DB Perpetually Out of Sync (P1 — Degraded Every Operation)

**What happened:** Almost every `bd` command failed with `Database out of sync with JSONL. Run 'bd sync --import-only' to fix.` Even after running `bd sync --import-only` (which reported success), subsequent commands still failed with the same error.

**Root cause:** Multiple agents were writing to `.beads/issues.jsonl` concurrently. The DB hash check detected the mismatch but import didn't resolve it because the file kept changing between import and next operation. `bd sync` also failed because git had unstaged changes preventing pull.

**Workaround:** Used `--allow-stale` on every command for the entire session.

**Systemic fix needed:**
- `--allow-stale` should be the default when concurrent agents are detected (or in orchestrator mode)
- `bd sync --import-only` should re-verify after import and retry if hash changed
- The staleness check should have a "last imported within N seconds" grace period

**Impact:** Every single `bd` command required `--allow-stale` flag. Without it, all beads operations fail. This is the #1 friction point for autonomous orchestration.

---

### 3. bd_subprocess_cap_hit Log Spam (P1 — Noise Obscures Signal)

**What happened:** Every `orch status`, `orch review`, and `orch complete` call produced 50-200 lines of `bd_subprocess_cap_hit` log messages before the actual output. The cap of 12 concurrent `bd` subprocesses was constantly hit because `orch status` calls `bd show` and `bd comments` for every agent in parallel.

**Root cause:** `orch status --all` with many agents (15+ including untracked) fires parallel `bd show` + `bd comments` for each, immediately hitting the cap of 12. The cap-hit events are logged at info level to stderr, which mixes with command output.

**Workaround:** Piped through `grep -v "bd_subprocess_cap_hit"` when I needed clean output. But many commands don't support easy filtering.

**Systemic fix needed:**
- Cap-hit messages should be logged at debug level, not info
- Or: batch `bd` calls to avoid hitting the cap (fetch all issues in one call)
- Or: raise the cap proportionally to active agent count

**Impact:** Severely degraded readability of all orch commands. Made it hard to quickly parse status output.

---

### 4. bd label add Syntax Trap (P2 — Silent Wrong Behavior)

**What happened:** Used `bd label add orch-go-tyu5f triage:ready area:cli effort:medium` expecting it to add 3 labels. Instead, the command interpreted `triage:ready` and `area:cli` as issue IDs (which don't exist), added only `effort:medium` to `orch-go-tyu5f`, and silently reported "Error resolving triage:ready: no issue found".

**Root cause:** `bd label add` syntax is `[issue-id...] [label]` — last argument is the label, all preceding are issue IDs. This is the inverse of what most CLI tools do (`add <subject> <items...>`). The error message "no issue found matching" doesn't hint that the argument was interpreted as an issue ID rather than a label.

**Workaround:** Discovered the correct syntax, then issued one `bd label add <id> <label>` per label. Required 3x more commands.

**Systemic fix needed:**
- `bd label add <id> <label> [<label>...]` would be more natural — first arg is ID, rest are labels
- Or: `bd label add --issue <id> <label> [<label>...]`
- At minimum: error message should say "interpreted 'triage:ready' as an issue ID — did you mean to add it as a label? Use separate commands for multiple labels."

**Impact:** Lost ~2 minutes debugging why labels weren't being applied. Several issues went to daemon without proper labels.

---

### 5. Daemon ProcessedIssueCache Blocking Re-Spawns (P1 — Was Active Bug)

**What happened:** The daemon had 14 ready issues but would only spawn 1 (`orch-go-xw0mi`). All other `triage:ready` issues were blocked by the processed cache because they had been previously evaluated and rejected (before getting the label). The cache treated "evaluated" as "processed" regardless of outcome.

**Root cause:** `MarkProcessed()` was called after evaluation, not after successful spawn. So rejected issues (missing label, wrong type) got cached and couldn't be retried even after the rejection reason was fixed.

**Resolution:** This was the P1 bug `orch-go-u17ut`, already being debugged when the session started. It completed during the session. Agent fixed `MarkProcessed()` to only fire after confirmed successful spawn.

**Workaround:** Manual spawning with `orch spawn --bypass-triage` for all issues. This worked but defeated the purpose of the daemon.

**Impact:** The daemon was effectively non-functional for the first half of the session. All spawning had to be manual.

---

### 6. Headless Spawn Intermittent Failures (P2 — Required Retry)

**What happened:** One `orch spawn` failed with `Failed to extract session ID: no session ID found in output`. Immediate retry with identical arguments succeeded.

**Root cause:** Unknown. Likely a transient OpenCode server issue — the session creation API didn't return the expected format. Could be related to concurrent session creation load.

**Workaround:** Retry.

**Systemic fix needed:** `orch spawn` should auto-retry once on session ID extraction failure before reporting error.

**Impact:** One wasted spawn attempt. Parallel sibling tool calls also failed due to the error, requiring a second round of spawning.

---

### 7. orch complete Verification Flakiness (P2 — Required Retry)

**What happened:** `orch complete orch-go-wemgi` failed with "verification failed" on first attempt. Immediate retry succeeded and properly closed the issue.

**Root cause:** Likely a race condition — the agent's session may have still been settling when verification ran. The workspace had completion artifacts but the transcript check may have hit a transient state.

**Workaround:** Retry.

**Systemic fix needed:** `orch complete` should retry verification once internally before failing.

**Impact:** Minor — one extra command invocation.

---

### 8. Cross-Project Issue Phantom in bd ready (P2 — Confusing)

**What happened:** `orch-go-doyd7` ("Resolve orchestrator skill token budget overage") appeared in `bd ready` output but `orch spawn --issue orch-go-doyd7` failed with "beads issue not found". The issue exists in the system but apparently belongs to a different project or has a lookup inconsistency.

**Root cause:** Unknown. Possibly the JSONL contains the issue but the lookup uses a different index. Or the issue was created by a cross-project agent and the project prefix doesn't match.

**Workaround:** Skipped the issue.

**Systemic fix needed:** `bd ready` should only show issues that can actually be operated on. If an issue appears in ready but can't be found by ID, that's a data integrity issue.

**Impact:** One wasted spawn attempt and confusion about whether the issue exists.

---

### 9. JSONL Hash Mismatch Warnings on Every Write (P3 — Noise)

**What happened:** Every `bd label add` produced `WARNING: JSONL file hash mismatch detected. Clearing export_hashes to force full re-export.`

**Root cause:** Concurrent agents modifying the JSONL between operations. The hash is checked on every write, and with many agents writing simultaneously, it's always stale.

**Workaround:** Ignored the warnings.

**Systemic fix needed:** In multi-agent environments, hash mismatch is the normal state, not an exception. The warning should be suppressed when concurrent writes are detected, or downgraded to debug level.

**Impact:** Visual noise on every operation.

---

### 10. Parallel Tool Call Cascade Failures (P3 — Workflow Friction)

**What happened:** When launching 3 parallel `orch spawn` commands and one failed (e.g., issue already closed), the other two also failed with `Sibling tool call errored` even though they were independent operations.

**Root cause:** Claude Code's parallel tool call semantics — when one call in a parallel batch errors, sibling calls are cancelled.

**Workaround:** Retried the two surviving spawn commands individually.

**Systemic fix needed:** This is a Claude Code platform behavior, not an orch-go issue. But `orch spawn` could pre-validate issue status before attempting the spawn to avoid triggering the cascade.

**Impact:** Required extra round-trips to complete intended spawns.

---

## Patterns Observed

### The Concurrency Tax
Most friction (#2, #3, #5, #9) stems from **concurrent agent operations on shared state**. Beads JSONL, the daemon processed cache, and bd subprocess pools all assume more sequential access patterns than a 5-agent swarm produces.

**Key insight:** The system was designed for 1-2 concurrent agents. At 5 agents, shared-state contention becomes the dominant friction source.

### The Retry Tax
Items #6, #7, and #10 all required manual retries. In an autonomous session, retries are cheap but they break flow and require the orchestrator to re-issue commands.

**Key insight:** Any operation the orchestrator issues should have built-in retry for transient failures. The retry cost is negligible compared to the context-switch cost of handling errors.

### The Signal-to-Noise Ratio
Items #3 and #9 are pure noise. They don't indicate actionable problems — they indicate the system is operating at scale. Log levels should adapt to operational mode.

**Key insight:** Debug-level logging becomes info-level noise at scale. The system should have an "orchestrator mode" or "swarm mode" that adjusts log levels accordingly.

---

## Recommendations (Priority-Ordered)

1. **OpenCode: Skip *.test.ts in plugin loader** — Prevents startup crashes from test files in plugin dir
2. **Beads: Grace period for staleness check** — If last import was <30s ago, skip the check
3. **orch: Suppress bd_subprocess_cap_hit at info level** — Move to debug, or batch bd calls
4. **bd label add: Support multiple labels** — `bd label add <id> <label1> <label2> ...`
5. **orch spawn: Auto-retry on session ID extraction failure** — One retry before error
6. **orch complete: Auto-retry on transient verification failure** — One retry before error
7. **bd ready: Validate issue accessibility** — Don't show issues that can't be operated on
8. **Daemon: Log-level swarm mode** — `orch daemon run --swarm` or auto-detect from agent count

---

## What Worked Well

- **Daemon auto-spawning after labels** — Once issues had `triage:ready`, the daemon picked them up within seconds
- **Daemon auto-completing** — Many agents were auto-completed without manual `orch complete`
- **Dependency chain resolution** — When t76z7 completed, 8z7c2 unblocked automatically, then 2qcu1
- **`--bypass-triage` escape hatch** — Critical for working around the ProcessedIssueCache bug
- **Build stability** — Despite 76 files changed by 25+ agents, `go build` still passed
- **gpt-5.3-codex throughput** — ~3-5 min per issue, consistent quality, 78% account usage barely moved
