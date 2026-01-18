# Session Synthesis

**Agent:** og-research-investigate-openai-chatgpt-18jan-861a
**Issue:** orch-go-a6jcx
**Duration:** 2026-01-18 → 2026-01-18
**Outcome:** success

---

## TLDR

Investigated whether ChatGPT Pro ($200/mo) provides API access and agent orchestration capabilities. Found that API is billed separately (no subscription API access), but Codex CLI authenticates via ChatGPT OAuth similar to Claude Code with Max—making it a viable backup for agent orchestration if needed.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md` - Comprehensive comparison of ChatGPT Pro vs Anthropic Max for agent orchestration

### Files Modified
- None

### Commits
- (pending) research investigation file

---

## Evidence (What Was Observed)

- ChatGPT Pro costs $200/mo and provides unlimited access to GPT-5.2, o3, o3-pro, o4-mini
- OpenAI API and ChatGPT are completely separate billing systems (verified: official docs)
- Codex CLI supports ChatGPT subscription OAuth authentication (verified: developer docs)
- Codex CLI has reported OAuth bugs where API keys are unexpectedly generated (GitHub issues)
- Anthropic blocked third-party tools (OpenCode, Cursor) from Max subscription on Jan 9, 2026
- Both platforms have $200/mo subscription-authenticated CLI agents for coding

### Research Sources
- OpenAI official: chatgpt.com/pricing, openai.com/codex
- Anthropic official: anthropic.com/pricing, support.claude.com
- GitHub issues documenting OAuth problems
- Builder.io comparison article

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md` - Full comparison of subscription tiers, models, CLI agents, and API access

### Decisions Made
- Recommendation: Monitor Codex CLI as backup, don't migrate (because existing Claude Code integration works and OAuth bugs reported)

### Constraints Discovered
- Neither ChatGPT Pro nor Anthropic Max includes API access with subscription
- Both platforms require separate API billing for programmatic access
- Anthropic has tightened third-party tool restrictions (Jan 2026 block)

### Key Models Comparison
| Platform | $200/mo Tier | Top Model | CLI Agent |
|----------|--------------|-----------|-----------|
| OpenAI | ChatGPT Pro | o3-pro, GPT-5.2 | Codex CLI |
| Anthropic | Max ($200) | Opus 4.5 | Claude Code |

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Research complete with structured findings
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for orchestrator review

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does Codex CLI performance compare to Claude Code in practice for multi-file refactoring?
- Could dual-vendor orchestration (orch-go supporting both backends) provide meaningful resilience?
- How does o3-pro reasoning compare to Claude Opus 4.5 extended thinking on hard problems?

**Areas worth exploring further:**
- Codex Cloud parallel task execution (could enable higher agent concurrency than current orch-go)
- OpenAI MCP (Model Context Protocol) integration for tool access

**What remains unclear:**
- Whether OpenAI will fix the Codex CLI OAuth bugs that cause unexpected API billing
- Whether Anthropic will further restrict Claude Code usage

---

## Session Metadata

**Skill:** research
**Model:** Claude Opus 4.5
**Workspace:** `.orch/workspace/og-research-investigate-openai-chatgpt-18jan-861a/`
**Investigation:** `.kb/investigations/2026-01-18-inv-research-compare-openai-chatgpt-pro-anthropic-max.md`
**Beads:** `bd show orch-go-a6jcx`
