# Session Synthesis

**Agent:** og-inv-probe-understand-openai-17feb-46b1
**Issue:** orch-go-1018
**Duration:** 2026-02-17T ~18:25 → ~18:45 UTC
**Outcome:** success

---

## TLDR

Orch's `gpt4o` alias resolves to `openai/gpt-4o` which will NOT work with OpenCode's ChatGPT Pro OAuth path — the Codex auth plugin explicitly whitelist-filters to 6 GPT-5.x models and deletes `gpt-4o`. To use OpenAI via OAuth, orch needs Codex-model aliases (`codex` → `gpt-5.2-codex`) and the user must complete a one-time OAuth login via the OpenCode TUI.

---

## Plain-Language Summary

Dylan added GPT model aliases to orch (`gpt4o` → `openai/gpt-4o`) and a `default_model` config key. Before flipping the switch, we investigated whether these model strings would actually work through OpenCode to the OpenAI provider. They won't — not because of format issues (the `provider/model` format flows correctly end-to-end), but because of model availability. OpenCode's Codex OAuth plugin (which enables free GPT usage via ChatGPT Pro subscription) explicitly removes all non-Codex models, leaving only 6 GPT-5.x models. Additionally, no OpenAI auth exists in auth.json yet — a one-time setup is needed. The fix is to add Codex-specific aliases in orch and authenticate via the OpenCode TUI.

---

## Verification Contract

See `.kb/investigations/2026-02-17-inv-probe-understand-openai-gpt-plugin.md` for full investigation with:
- 7 findings with file path + line number evidence
- Structured uncertainty (what was tested vs untested)
- Implementation recommendations with authority classification

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-17-inv-probe-understand-openai-gpt-plugin.md` - Full investigation answering 7 questions about OpenAI plugin compatibility

### Files Modified
- None (investigation only, no code changes)

### Commits
- (pending - investigation artifact commit)

---

## Evidence (What Was Observed)

- Codex plugin at `codex.ts:360-372` whitelists exactly: `gpt-5.1-codex-max`, `gpt-5.1-codex-mini`, `gpt-5.2`, `gpt-5.2-codex`, `gpt-5.3-codex`, `gpt-5.1-codex`
- `~/.local/share/opencode/auth.json` contains only Anthropic OAuth — no OpenAI entry
- `opencode.jsonc` already has 7 OpenAI model configs with reasoning effort variants
- `models-snapshot.ts` includes `gpt-4o` among 40+ OpenAI models (available with API key, not OAuth)
- OpenAI custom loader at `provider.ts:126-134` uses `sdk.responses(modelID)` — Responses API, not Chat
- Provider init order: env → stored auth → plugins → custom loaders → config merge (custom loader `autoload: false` means OpenAI only loads if auth already exists)

### Tests Run
```bash
# Checked auth.json for OpenAI credentials
cat ~/.local/share/opencode/auth.json | python3 -m json.tool
# Result: Only Anthropic OAuth tokens present

# Extracted OpenAI models from models-snapshot.ts
bun -e '<extraction script>'
# Result: gpt-4o confirmed present in snapshot (40+ models)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-17-inv-probe-understand-openai-gpt-plugin.md` - End-to-end model string compatibility analysis

### Decisions Made
- N/A (investigation only, recommendations for orchestrator)

### Constraints Discovered
- Codex OAuth path limits models to 6 GPT-5.x variants — this is by design, not a bug
- Provider custom loader registration depends on auth timing — `autoload: false` + no auth = no model loader
- OAuth tokens auto-refresh via refresh_token in codex plugin

### Externalized via `kb`
- (will run kb quick commands before commit)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Update orch GPT aliases for Codex models + authenticate OpenAI
**Skill:** feature-impl
**Context:**
```
Investigation orch-go-1018 found orch's GPT aliases don't match Codex-available models.
Need to: (1) add codex/codex-mini/codex-max aliases in model.go, (2) authenticate
OpenAI via OpenCode TUI (one-time manual setup), (3) update default_model recommendation.
See .kb/investigations/2026-02-17-inv-probe-understand-openai-gpt-plugin.md for full analysis.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What are the rate limits for Codex OAuth path? (not documented in source code)
- Does `gpt-5.2` (non-codex) have different capabilities than `gpt-5.2-codex` via OAuth?
- Can the Codex Responses API endpoint be used headlessly without issues?

**What remains unclear:**
- Whether OpenAI OAuth tokens survive OpenCode server restarts (likely yes since stored in auth.json)
- The exact user experience of the OAuth flow via TUI

---

## Session Metadata

**Skill:** investigation
**Model:** opus-4.6
**Workspace:** `.orch/workspace/og-inv-probe-understand-openai-17feb-46b1/`
**Investigation:** `.kb/investigations/2026-02-17-inv-probe-understand-openai-gpt-plugin.md`
**Beads:** `bd show orch-go-1018`
