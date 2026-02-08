# Spawn System Evolution

**Type:** Changelog
**Parent:** `.kb/models/model-access-spawn-paths.md`
**Purpose:** Historical timeline of spawn system changes. For current state, see `current-model-stack.md`.

---

## Timeline

**Jan 8, 2026:** Opus auth gate discovered
- Attempted to spawn with `--model opus`
- Received auth rejection: "This credential is only authorized for use with Claude Code"
- Created zombie agents (tracked but never ran)
- Investigated fingerprinting mechanism (TLS, HTTP/2, headers)

**Jan 8, 2026:** Header injection attempt failed
- Injected all known Claude Code headers into OpenCode provider
- Opus still rejected
- Gemini spawns started hanging (header conflicts)
- Abandoned spoofing approach

**Jan 9, 2026:** Gemini Flash TPM limits hit
- Single investigation agent hitting 2,000 req/min limit
- Tool-heavy spawns (35+ calls/sec) trigger rate limiting
- OpenCode retry logic causes 3-30s delays per request
- Forced immediate switch to Sonnet for reliability

**Jan 9, 2026:** Community Opus workarounds research
- Discovered community had found tool name fingerprinting mechanism
- Official plugin (0.0.7) released, then re-blocked within 6 hours
- 474+ GitHub comments documenting cat-and-mouse game
- Strategic decision: abandon workarounds, accept Sonnet/Gemini split

**Jan 10, 2026:** Dual spawn architecture emerged
- Implemented `--backend` flag for explicit path selection
- Documented escape hatch pattern in CLAUDE.md
- Opus becomes Max-subscription-only model
- Primary path remains OpenCode API with Sonnet/Flash

**Jan 10, 2026:** Backend flag bug fixed
- Decision doc examples used `--mode`, code used `--backend`
- Flag was being ignored
- Fixed naming, verified priority order

**Jan 11, 2026:** Infrastructure work auto-detection added
- Keyword-based detection (opencode, spawn, daemon, registry, etc.)
- Auto-applies `--backend claude --tmux` at priority 2.5
- Prevents agents from killing themselves
- Makes escape hatch invisible for common cases

**Jan 12, 2026:** Cost tracking gap identified
- No visibility into Sonnet spend since Jan 9 switch
- Dashboard shows Max usage but not API token usage
- Strategic decisions blocked without cost data
- Investigation documented requirements for tracking implementation

**Jan 12, 2026:** Model created from synthesis
- Recognized constraint has system-wide ripple effects
- Escape hatch pattern now embedded in spawn priority logic
- Cost/quality/reliability tradeoffs explicit
- Strategic questions about model usage surfaced

**Jan 18, 2026:** Claude CLI recommended as primary path
- Discovered API costs: $402 in ~2 weeks, ramping to $70-80/day
- Projected spend: $2,100-2,400/mo vs $200/mo flat Max subscription
- Decision: Use Claude CLI + Opus as primary (via config, not code change)
- Code default unchanged (`opencode`), but config overrides it
- See: `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md`

**Jan 19, 2026:** DeepSeek V3 function calling confirmed
- Tested: 3 minutes, 62K tokens, successful completion with tool calls
- Cost: $0.25/$0.38/MTok (viable cost-effective API option)
- Works despite "unstable" warning in DeepSeek docs
- See: `.kb/investigations/2026-01-19-inv-test-deepseek-v3-function-calling.md`

**Jan 20, 2026:** Docker backend added as third spawn path
- Provides Statsig fingerprint isolation for rate limit escape hatch
- Architecture: Host tmux window runs `docker run ... claude` (NOT nested tmux)
- Uses `~/.claude-docker/` for fresh fingerprint per spawn
- Same lifecycle commands (status, complete, abandon) via tmux
- No dashboard visibility (escape hatch philosophy - use tmux)
- See: `.kb/investigations/archived/2026-01-20-inv-design-claude-docker-backend-integration.md`

**Jan 21, 2026:** GPT-5.2 tested and deemed unsuitable for orchestration
- ChatGPT Pro subscription ($200/mo) tested as potential escape hatch
- Five critical anti-patterns: reactive gates, role collapse, excessive deliberation, no failure adaptation, literal instruction handling
- Decision: Claude Opus 4.5 exclusive for orchestration
- GPT-5.2 may work for constrained worker tasks only
- See: `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md`

**Jan 22, 2026:** Model updated to reflect current state
- Removed hardcoded defaults; model now points to config for actual values
- Backend selection mechanism documented (priority cascade)
- GPT-5.2 orchestration constraint added
- DeepSeek V3 added as viable API model

**Jan 24, 2026:** GPT-5.2 tested and deemed unsuitable for workers
- Tested gpt-5.2 and gpt-5.2-codex via OpenCode as workers
- gpt-5.2: Completed task, then hallucinated 565-line quantum physics simulation
- gpt-5.2-codex: Went idle repeatedly, couldn't navigate 662-line spawn context
- Root cause: Spawn context is "Claude dialect" - evolved to fit Claude's instruction-following
- Decision: Accept Claude-specific orchestration, don't invest in multi-model paths
- See: `.kb/decisions/2026-01-24-claude-specific-orchestration-accepted.md`
