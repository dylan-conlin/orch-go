# Session Synthesis

**Agent:** og-inv-investigate-revisit-opencode-23mar-fc58
**Issue:** orch-go-uzx15
**Duration:** 2026-03-23
**Outcome:** success

---

## TLDR

GPT-5.4 support in orch-go requires only ~5 lines of code changes — the entire OpenCode provider layer and orch-go routing pipeline already support OpenAI. The strategic finding is that OpenAI allows subscription OAuth in third-party tools (ChatGPT Pro, $200/mo, $0/token) while Anthropic bans it, meaning GPT-5.4 unlocks headless agent spawning at flat rate — something impossible with Claude Max. The only real blocker is whether GPT-5.4 can follow the worker-base protocol without stalling (GPT-5.2 had 67.5% stall rates, but GPT-5.4 is a generation ahead).

---

## Plain-Language Summary

I investigated whether orch-go can route agent work to OpenAI's GPT-5.4 model (released March 5, 2026) instead of being locked into Claude. The answer is yes, and the plumbing is almost entirely done already. OpenCode (Dylan's fork) already has full OpenAI provider support with 26 providers, Codex OAuth authentication, and the URL rewriting fix that tripped up other tools. On the orch-go side, model routing auto-selects the OpenCode backend for any OpenAI model. The total code changes needed: add a "gpt-5.4" alias to model.go (2 lines) and add "gpt-5.4" to the allowed models whitelist in OpenCode's codex.ts (1 line).

The bigger finding is economic: Anthropic banned subscription OAuth in third-party tools in February 2026, so Claude Max ($200/mo) only works through the Claude Code CLI. But OpenAI explicitly allows ChatGPT Pro ($200/mo) to be used in third-party tools via OAuth — meaning GPT-5.4 can run headlessly through OpenCode at zero per-token cost. Per-token, GPT-5.4 is also 50% cheaper ($2.50/MTok input vs $5.00 for Opus). The catch: past GPT models (5.2-codex) had 67.5% stall rates on orch-go's multi-step agent protocol. GPT-5.4 needs empirical stall rate testing before it can be trusted for production routing.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` - Full investigation with findings, cost analysis, and implementation roadmap

### Files Modified
- None (investigation only — no code changes)

### Commits
- Investigation file + SYNTHESIS.md

---

## Evidence (What Was Observed)

- OpenCode provider.ts: 26 bundled providers including `@ai-sdk/openai` (line 93), with custom Responses API loader (lines 152-159)
- OpenCode codex.ts: Full Codex OAuth with PKCE, hardcoded `allowedModels` set (lines 360-367) that includes up to gpt-5.3-codex but NOT gpt-5.4
- OpenCode codex.ts: URL rewriting from `/v1/responses` → `https://chatgpt.com/backend-api/codex/responses` already implemented (lines 486-488)
- models-snapshot.ts: gpt-5.4 present with $2.50/$15 per MTok pricing, 1.05M context window, reasoning + tool_call capable
- orch-go model.go: OpenAI aliases exist through gpt-5.2 (lines 46-68), gpt-5.4 missing
- orch-go resolve.go: `modelBackendRequirement()` routes OpenAI → BackendOpenCode automatically (lines 604-616)
- orch-go skill_inference.go: Implementation skills return empty string for model, allowing override (lines 279-283)
- Agent lifecycle model: GPT-5.2-codex had 67.5% stall rate, GPT-4o had 87.5% (Feb 2026 audit)
- OpenAI allows ChatGPT Pro OAuth in third-party tools; Anthropic banned it Feb 19, 2026

### Tests Run
```bash
# No code changes made — this was a research investigation
# Verified architecture by reading source code and external documentation
```

---

## Architectural Choices

No architectural choices — investigation only. Architecture already supports the desired outcome.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Codex plugin `allowedModels` whitelist filters GPT-5.4 when OAuth is active — must be updated
- GPT-5.4 stall rate on worker-base protocol is unknown — only GPT-5.2 data exists
- ChatGPT Pro ($200/mo) vs Plus ($20/mo) — unclear which tier covers Codex GPT-5.4 access

### Key Insights
- The Anthropic/OpenAI subscription access asymmetry is strategically significant for orch-go's execution layer
- GPT-5.4's 1.05M context window vs Opus's 200K may reduce context-exhaustion stalls (DAO-13 cited 63-76KB SPAWN_CONTEXT consuming 40-50K GPT tokens)
- The falsification criteria for DAO-13 is clear: "Non-Anthropic models achieve >80% completion rate on protocol-heavy daemon spawns (N>30)"

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

**Issue 1:** "Add GPT-5.4 alias to model.go and Codex allowedModels whitelist"
**Skill:** feature-impl
**Context:**
```
Add "gpt-5.4" alias to pkg/model/model.go Aliases map and "gpt-5.4" to allowedModels in
~/Documents/personal/opencode/packages/opencode/src/plugin/codex.ts line 360. Rebuild OpenCode.
```

**Issue 2:** "Stall rate test: spawn 5 feature-impl tasks on GPT-5.4 via OpenCode"
**Skill:** investigation
**Context:**
```
After GPT-5.4 alias is added, spawn 5 feature-impl tasks with --model gpt-5.4. Measure completion
rate, phase reporting compliance, SYNTHESIS creation. Compare to Opus baseline (~96% completion).
Threshold: < 30% stall rate = viable for implementation routing.
```

---

## Unexplored Questions

- **ChatGPT Pro vs Plus for Codex access:** Does Plus ($20/mo) cover GPT-5.4 via Codex OAuth, or is Pro ($200/mo) required?
- **Codex rate limits:** What are the per-model rate limits via OAuth vs API key?
- **GPT-5.4-mini:** Also released — could be even better for simple implementation tasks at lower token usage
- **Hybrid routing economics:** What's the optimal cost split between Claude (reasoning) and GPT-5.4 (implementation)?
- **Protocol simplification:** Could a lighter-weight protocol (fewer bd comment requirements) improve GPT-5.4 completion rates?

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-investigate-revisit-opencode-23mar-fc58/`
**Investigation:** `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md`
**Beads:** `bd show orch-go-uzx15`
