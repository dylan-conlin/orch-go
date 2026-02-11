# Session Handoff — 2026-02-10 (Session 2: Claude Code)

## What Happened This Session

Dylan switched to Claude Code directly (bypassing OpenCode TUI). This session committed the GateCommitEvidence work from the previous session, fixed GPT model routing, created issues from DYLANS_THOUGHTS.org, and attempted to get the daemon running with reliability-focused work.

## Work Completed (All Pushed to Remote)

### 1. GateCommitEvidence — COMMITTED & PUSHED ✅
- Commit `4d9825a0` — Tier 1 core gate blocks `orch complete` when agent branch has zero commits
- Commit `cbc27351` — Session cleanup safety (--preserve-orchestrator defaults true, 1-hour safety floor)
- Commit `964e9f25` — Gate tier documentation (from E2E agent on Sonnet)
- E2E validated: Sonnet agent spawned → completed → commit evidence gate passed → cherry-pick landed

### 2. GateCommitEvidence Caught a Ghost Completion ✅
- GPT-5.3-codex test agent ran 120K tokens on a "one-line comment" task and committed NOTHING
- Gate correctly blocked completion with: "agent branch has 0 commits (ghost completion: no work to land)"
- Proves the gate works exactly as intended

### 3. GPT OAuth — Partially Fixed, Then Discovered Deeper Problem
**What was fixed:**
- Installed `opencode-openai-codex-auth@4.4.0` plugin (was missing from node_modules)
- Fixed Procfile: stripped `OPENAI_API_KEY` and removed invalid `BUN_JSC_heapSize` (commit `db58c09c`)
- Dylan re-authenticated via `opencode auth login` — OAuth tokens present in auth.json

**What we discovered:**
- The OpenAI OAuth plugin (`opencode-openai-codex-auth`) only works in OpenCode TUI mode, NOT in `opencode serve` (headless) mode
- In serve mode, the plugin's `auth.loader` hook doesn't activate — OpenCode falls back to needing `OPENAI_API_KEY` env var
- With API key stripped (for safety), NO OpenAI models work in headless mode
- With API key present, models work BUT bill to Platform API (credit card), not ChatGPT Pro subscription
- The "successful" GPT spawn earlier in the session was secretly using the Platform API key (before we stripped it)

**Resolution:**
- Switched all models to Anthropic (Claude Max OAuth, confirmed working in serve mode, $200/mo flat)
- Config updated: `~/.orch/config.yaml` now uses `anthropic/claude-sonnet-4-20250514` (default) and `anthropic/claude-opus-4-5-20251101` (complex tasks)
- GPT config preserved as comments for when plugin issue is fixed
- Issue `orch-go-mkq3y` remains open for the deeper plugin-in-serve-mode fix

### 4. Issues Created from DYLANS_THOUGHTS.org (7 new)
| Issue | What | Priority |
|-------|------|----------|
| `orch-go-gaqy0` | Daemon slot observability in work graph | P2 |
| `orch-go-ik05a` | New spawns show no title in work graph | P2 |
| `orch-go-tqmkr` | Orchestrator diagnostic/firefighting mode | P1 |
| `orch-go-n1kpb` | 30-min reap timer destroyed UI (partial fix landed) | P1 |
| `orch-go-ucf63` | Work graph o/i keys regression | P2 |
| `orch-go-6v2ta` | OpenCode server crash pattern (6 crashes now) | P1 |
| `orch-go-cmdfh` | Audit ghost completion work loss | P2 |

### 5. Other Issues Created This Session
| Issue | What | Priority |
|-------|------|----------|
| `orch-go-mkq3y` | ChatGPT Pro OAuth broken (partially fixed, deeper issue found) | P1 |
| `orch-go-4bo36` | API key billing safety gate | P1 |
| `orch-go-irpii` | `orch wait` hangs forever | P1 |

### 6. Knowledge Captured
- Constraint `kb-7c5604`: OpenCode server must start with OPENAI_API_KEY unset
- Investigation: `.kb/investigations/2026-02-10-inv-gpt-codex-oauth-setup.md` (from Opus agent)

## What's Still Open / Not Done

### Daemon Not Started Yet
- Config switched to Anthropic models ✅
- Issues labeled `triage:ready`: `orch-go-irpii`, `orch-go-4bo36`, `orch-go-ugla4`, `orch-go-nweny`, `orch-go-tye87`, `orch-go-k6lvv`
- Daemon was being started when session ran out of context
- **Next session should just start daemon:** `orch daemon run`

### GPT OAuth in Serve Mode (orch-go-mkq3y)
- Plugin works in TUI but not in `opencode serve`
- Root cause: plugin's `auth` hook doesn't activate in serve mode
- The plugin IS installed in both `~/.config/opencode/node_modules/` and `~/.cache/opencode/node_modules/`
- Need to investigate OpenCode source to understand why plugins load differently in serve vs TUI
- `oco` function in `~/.zshrc` (line 788) was the existing workaround for TUI: `env -u OPENAI_API_KEY "$OPENCODE_BIN" "$@"`

### OpenCode Crash Pattern (orch-go-6v2ta)
- Now 6 crashes. Latest crash happened during this session while testing.
- overmind has `--auto-restart opencode` but it's not reliably restarting
- Server was manually restarted: `env -u ANTHROPIC_API_KEY -u OPENAI_API_KEY ~/.bun/bin/opencode serve --port 4096`

### API Key Billing Safety (orch-go-4bo36)
- OPENAI_API_KEY is still in Dylan's environment (source unknown — not in .zshrc, possibly from another app)
- Procfile strips it from OpenCode server ✅
- But if someone adds it back or starts server without Procfile, credit card gets billed
- Need a proper gate in `orch spawn` that detects API key vs OAuth auth method

## System State at Handoff

- **Git:** Clean working tree, all pushed to remote (master at `db58c09c` or later with bd sync)
- **Build:** Clean
- **OpenCode:** Was manually started (may have crashed again). Restart with: `env -u ANTHROPIC_API_KEY -u OPENAI_API_KEY ~/.bun/bin/opencode serve --port 4096`
- **Dashboard:** Running via overmind
- **Daemon:** NOT running — needs `orch daemon run`
- **Account:** ~11% used (6d 15h until reset)
- **Config:** `~/.orch/config.yaml` set to Anthropic models (Sonnet default, Opus for complex)
- **Swarm:** 0 active agents (old test sessions may still exist in OpenCode, harmless)

## Recommended Next Steps (Priority Order)

1. **Verify OpenCode is running:** `curl -s http://127.0.0.1:4096/session | head -1` — if not, restart per above
2. **Start daemon:** `orch daemon run` — will pick up 6 `triage:ready` reliability issues on Anthropic models
3. **Monitor and complete agents** as they finish — `orch status`, then `orch complete <id>`
4. **Push any beads changes:** `./scripts/bd-sync-safe.sh && git push`

## Key Discoveries This Session

1. **`oco` function exists** (`~/.zshrc:788`) — strips OPENAI_API_KEY for TUI mode. The Procfile equivalent was missing.
2. **BUN_JSC_heapSize is invalid** in current Bun — was silently crashing OpenCode on restart via overmind.
3. **OpenCode plugin auth hooks don't work in serve mode** — this is the fundamental blocker for GPT-via-OAuth headless spawns.
4. **GateCommitEvidence works** — caught a real ghost completion on its first day in production.
