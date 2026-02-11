# Session Handoff — 2026-02-10

## What Happened This Session

Dylan switched from OpenCode TUI to Claude Code directly because the orchestration system is too fragile (5 OpenCode crashes in recent days, ghost completions, model routing bugs, zombie agents).

## Work Completed

### 1. GateCommitEvidence (Phase 4 — DONE, not yet committed)

Added a new Tier 1 (core) verification gate that blocks `orch complete` when the agent branch has zero commits. This prevents ghost completions — the root cause of 22 issues being closed with no work landed.

**Files changed:**
- `pkg/verify/check.go` — Added `GateCommitEvidence` constant, registered as core gate, added skip flag case
- `cmd/orch/complete_gates.go` — Added `checkCommitEvidence()` function (runs `git rev-list --count` on agent branch vs merge-base), added `commitEvidenceResult` type, `readBranchNameForCommitGate()` helper, `joinGateErrors()` helper
- `cmd/orch/complete_verify.go` — Added `CommitEvidence` field to `SkipConfig`, wired into `hasAnySkip()`, `skippedGates()`, `shouldSkipGate()`, `getSkipConfig()`
- `cmd/orch/complete_cmd.go` — Added `completeSkipCommitEvidence` flag, `--skip-commit-evidence` CLI flag, updated help text
- `cmd/orch/complete_gates_test.go` — 4 tests: nil target, no branch, zero commits (fails), with commits (passes)

**Build:** Clean. **Vet:** Clean. **Tests:** All pass.
**Binary installed:** `make install` done, live at `~/bin/orch`.

### 2. GPT Model Routing Investigation (DIAGNOSED, not fixed)

**Root cause found — two separate issues:**

1. **Tmux spawns can't pass model.** The tmux path (`runSpawnTmux`) types the prompt via `tmux.SendKeysLiteral`. No mechanism to tell the TUI which model to use — it defaults to opus. This is why spawns with `--tmux` always run on opus regardless of config.

2. **Headless spawns DO pass the model correctly** via `sendHeadlessPrompt()` at `cmd/orch/spawn_execute.go:385-396`. The model string is split into `providerID`/`modelID` and sent in the `prompt_async` payload. However, `Session.create` in OpenCode (`opencode/packages/opencode/src/session/index.ts:140-155`) does NOT accept a `model` field in its zod schema — the model field orch sends on session creation is silently dropped. Model only takes effect on the first message.

3. **gpt-5.3-codex requires OAuth** (from Dylan's notes). The OpenAI OAuth entry in `~/.local/share/opencode/auth.json` has disappeared before. Without it, requests fall through to `OPENAI_API_KEY` (Platform API) where gpt-5.3-codex doesn't exist. Previous orchestrator confirmed OAuth was restored, but crashes may wipe it again.

**Bottom line for GPT spawns:** Headless should work (model passed on prompt). Tmux will NOT work (model not passable). The E2E test needs to be headless.

## What's Still Open

### From Dylan's 5-Problem List
1. ~~Rate limiter bug~~ — Fixed (commit `a3b2569e`)
2. ~~Broken test~~ — Fixed (commit `dc8a7cd9`)
3. ~~Zombie agents~~ — Cleaned up
4. **Stale worktrees** — 175 dirs in `/tmp/orch-*`, 2 in `.orch/worktrees/`. Low priority cleanup.
5. **GateCommitEvidence** — Code done, needs commit + E2E validation

### E2E Pipeline Test (Phase 3 — NOT DONE)
The E2E test was attempted twice but didn't complete:
- First attempt: wrong model (opus via tmux) — abandoned `orch-go-21516`
- Second attempt: tried to reuse issue that already had Phase: Complete
- Need to spawn fresh, headless, let it complete, then run `orch complete`

### From Dylan's Notes (not yet addressed)
- Dashboard spawns showing no title (line 109 of DYLANS_THOUGHTS.org)
- Need diagnostic/firefighting mode for orchestrator skill (line 111)
- "Setting a 30 minute reap timer that destroys my primary UI" (line 154) — careless agent behavior

## System State at Handoff
- **Build:** Clean
- **Git:** Uncommitted changes in `pkg/verify/check.go`, `cmd/orch/complete_gates.go`, `cmd/orch/complete_verify.go`, `cmd/orch/complete_cmd.go`, `cmd/orch/complete_gates_test.go` (the commit evidence gate)
- **Dashboard/OpenCode/Daemon:** All running
- **Account:** 10% used (6d 17h until reset)
- **Swarm:** 0 active agents
- **Config:** `~/.orch/config.yaml` has `default_model: openai/gpt-5.3-codex` and all skill_models set to GPT

## Recommended Next Steps
1. **Commit the gate code** — it's tested and ready
2. **Run E2E test headless** — `orch spawn --bypass-triage feature-impl "trivial task"` (no --tmux)
3. **Verify GPT model** — check `orch status` shows GPT not opus after headless spawn
4. **Run `orch complete`** on the E2E agent — verify commit evidence gate passes, cherry-pick lands
5. **If tmux GPT spawns matter:** fix requires either (a) passing model via `opencode run --model` CLI flag, or (b) a config file the TUI reads per-session
