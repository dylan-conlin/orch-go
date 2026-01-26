<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Clawdbot uses the Pi agent runtime (mariozechner/pi-*) with a gateway-centric architecture, featuring standard OAuth/API key auth - no special Opus gate workaround.

**Evidence:** Code analysis of src/agents/, src/providers/, skills/ - uses @mariozechner/pi-ai for OAuth, loadSkillsFromDir from pi-coding-agent, auth-profiles.json for credential storage.

**Knowledge:** Pi agent framework provides skill loading, model auth, and agent runtime; clawdbot wraps this with multi-channel messaging (WhatsApp, Telegram, etc.); fundamentally different architecture than orch-go (CLI wrapper vs. messaging gateway).

**Next:** No action needed - reference investigation for future cross-pollination of ideas (skill format is similar, auth profile rotation is interesting).

**Promote to Decision:** recommend-no - informational investigation, no architectural decision needed

---

# Investigation: Clawdbot Analysis

**Question:** What skills system, OAuth/auth handling, and architecture does clawdbot use, and how does it compare to orch-go?

**Started:** 2026-01-26
**Updated:** 2026-01-26
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Skills System Uses Pi Coding Agent Library

**Evidence:** Skill loading via `loadSkillsFromDir` from `@mariozechner/pi-coding-agent` (workspace.ts:6-8). Skills are SKILL.md files with YAML frontmatter containing `name` and `description`. Optional directories: `scripts/`, `references/`, `assets/`.

**Source:**
- `src/agents/skills/workspace.ts:6-8` - imports loadSkillsFromDir
- `src/agents/skills/workspace.ts:102-175` - loadSkillEntries function
- `skills/coding-agent/SKILL.md` - example skill structure
- `skills/skill-creator/SKILL.md` - skill authoring docs

**Significance:** Skills format is nearly identical to orch-go's. Main difference: clawdbot uses the pi-coding-agent library for loading while orch-go has custom loader. Skill precedence: extra < bundled < managed < workspace. Frontmatter can include `metadata.clawdbot` for emoji, requirements.

---

### Finding 2: Auth Uses Pi AI OAuth + Auth Profiles System

**Evidence:** Auth handled via `@mariozechner/pi-ai` OAuth system (`getOAuthApiKey`, `OAuthCredentials`). Three credential types: `api_key`, `token` (static bearer), `oauth` (refreshable). Stored in `auth-profiles.json`. Supports: Anthropic (API key or OAuth), OpenAI, Google Gemini CLI, GitHub Copilot, Chutes, Qwen Portal, AWS Bedrock.

**Source:**
- `src/agents/auth-profiles/oauth.ts:1` - imports from pi-ai
- `src/agents/auth-profiles/types.ts:5-32` - credential type definitions
- `src/agents/model-auth.ts:246-248` - Anthropic env vars: ANTHROPIC_OAUTH_TOKEN, ANTHROPIC_API_KEY

**Significance:** No special Opus gate workaround. Uses standard Anthropic API keys or OAuth tokens from environment variables. The `pi-ai` library handles OAuth refresh. Significantly different from orch-go's Claude Max-specific OAuth via account switching.

---

### Finding 3: Gateway-Centric Architecture vs. CLI Wrapper

**Evidence:** Clawdbot is a "personal AI assistant" with WebSocket gateway at its core. Multi-channel: WhatsApp (Baileys), Telegram (grammY), Slack (Bolt), Discord, Signal, iMessage, etc. Agents run via gateway sessions with tool streaming.

**Source:**
- `README.md:19-24` - architecture description
- `README.md:171-186` - architecture diagram
- `package.json:163-165` - Pi agent dependencies
- `src/agents/subagent-registry.ts` - session management

**Significance:** Fundamental architecture difference: clawdbot wraps Pi agent runtime in a messaging gateway, while orch-go wraps Claude Code CLI and OpenCode API in a terminal-based orchestration system. Clawdbot is designed for end-users chatting via messaging apps; orch-go is designed for developers orchestrating agent swarms.

---

## Synthesis

**Key Insights:**

1. **Similar skill format, different loading** - Both use SKILL.md with YAML frontmatter, but clawdbot leverages pi-coding-agent library while orch-go has custom loader. Skill migration between systems would be straightforward.

2. **Auth model difference** - Clawdbot uses standard API keys/OAuth via auth profiles that can rotate. Orch-go specifically targets Claude Max subscription with account switching. Clawdbot is more flexible for multi-provider setups.

3. **Architecture divergence** - Clawdbot is a messaging gateway (consumer product), orch-go is an orchestration system (developer tool). Different use cases: "chat with AI via WhatsApp" vs. "spawn agent swarms to do coding work".

**Answer to Investigation Question:**

Clawdbot uses:
- **Skills:** SKILL.md files loaded via pi-coding-agent library, nearly identical format to orch-go
- **Auth:** auth-profiles.json with OAuth/API key/token types, powered by pi-ai library - no Opus gate workaround, just standard Anthropic API auth
- **Architecture:** WebSocket gateway with Pi agent runtime, multi-channel messaging, fundamentally different from orch-go's CLI wrapper approach

**Comparison Table:**

| Aspect | Clawdbot | orch-go |
|--------|----------|---------|
| Core runtime | Pi agent (Node.js) | Claude Code CLI / OpenCode API (Go) |
| Skill format | SKILL.md + frontmatter | SKILL.md + frontmatter (similar) |
| Skill loading | pi-coding-agent library | Custom pkg/skills loader |
| Auth | Auth profiles (OAuth, API key, token) | Claude Max OAuth via accounts |
| Multi-agent | Gateway-managed sessions | Tmux windows + registry |
| Channels | WhatsApp, Telegram, Slack, etc. | Terminal-only (tmux) |
| Use case | Personal AI assistant (consumer) | Agent orchestration (developer) |

---

## Structured Uncertainty

**What's tested:**

- ✅ Skill format uses SKILL.md with YAML frontmatter (verified: read skill files)
- ✅ Auth uses pi-ai OAuth system (verified: read oauth.ts imports)
- ✅ Gateway architecture exists (verified: read README, src/gateway/)

**What's untested:**

- ⚠️ Whether pi-ai OAuth actually handles Opus model access (no live test)
- ⚠️ Performance comparison with orch-go spawn/monitoring
- ⚠️ Whether skill migration between systems actually works smoothly

**What would change this:**

- Discovery that clawdbot has a hidden Claude Max-specific auth path
- Finding that skill format has incompatible edge cases

---

## References

**Files Examined:**
- `README.md` - Architecture overview, features
- `AGENTS.md` - Agent instructions (CLAUDE.md symlinks to this)
- `package.json` - Dependencies (@mariozechner/pi-* packages)
- `src/agents/skills/workspace.ts` - Skill loading implementation
- `src/agents/auth-profiles/*.ts` - Auth profile system
- `src/agents/model-auth.ts` - Model authentication
- `src/providers/github-copilot-auth.ts` - Example OAuth flow
- `skills/coding-agent/SKILL.md` - Example skill
- `skills/skill-creator/SKILL.md` - Skill authoring docs

**Commands Run:**
```bash
# Clone repo
git clone https://github.com/clawdbot/clawdbot ~/Documents/personal/clawdbot

# List structure
ls -laR ~/Documents/personal/clawdbot/skills/
ls -la ~/Documents/personal/clawdbot/src/agents/
```

**External Documentation:**
- https://docs.clawd.bot - Clawdbot documentation
- https://github.com/clawdbot/clawdbot - Source repository

---

## Investigation History

**2026-01-26 12:56:** Investigation started
- Initial question: Analyze clawdbot skills system, auth handling, and architecture
- Context: Requested by orchestrator to understand alternative agent orchestration approach

**2026-01-26 13:05:** Completed codebase analysis
- Examined: skills/, src/agents/, src/providers/, README.md, package.json
- Found: Pi agent runtime, auth profiles system, gateway architecture

**2026-01-26 13:10:** Investigation completed
- Status: Complete
- Key outcome: Clawdbot uses fundamentally different architecture (messaging gateway vs. CLI wrapper) with similar skill format but different auth approach
