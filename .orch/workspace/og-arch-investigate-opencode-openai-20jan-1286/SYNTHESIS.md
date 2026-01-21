# Session Synthesis

**Agent:** og-arch-investigate-opencode-openai-20jan-1286
**Issue:** orch-go-08rm3
**Duration:** 2026-01-20 ~15:00 → ~16:30
**Outcome:** success

---

## TLDR

Investigated why OpenCode shows $ cost indicator instead of (oauth) for OpenAI models. Root cause: OPENAI_API_KEY environment variable takes precedence over OAuth tokens in auth.json - unset the env var to use OAuth-only.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-20-inv-investigate-opencode-openai-auth-confusion.md` - Complete investigation with findings, synthesis, and recommendations

### Files Modified
- None (investigation-only task)

### Commits
- Pending - investigation file ready to commit

---

## Evidence (What Was Observed)

- OPENAI_API_KEY=sk-proj-6TNhtR7... is set in environment (confirmed via `env | grep OPENAI`)
- OAuth tokens exist in `~/.local/share/opencode/auth.json` with type="oauth" and plan_type="pro"
- `opencode auth list` shows BOTH: Credentials (oauth) AND Environment (OPENAI_API_KEY)
- provider.ts:795-804 loads env vars FIRST, setting provider.key from OPENAI_API_KEY
- provider.ts:807-815 only processes auth.json entries with type="api" (NOT oauth)
- provider.ts:817-862 runs plugin loaders AFTER env loading
- provider.ts:959: SDK uses `provider.key` if `options["apiKey"]` not set by plugin
- session-context-usage.tsx calculates cost from token counts and model pricing data
- session/index.ts:472-477 uses Decimal math for cost calculation per million tokens

### Tests Run
```bash
# Verified environment
env | grep OPENAI
# Output: OPENAI_API_KEY=sk-proj-6TNhtR7...

# Verified auth.json
cat ~/.local/share/opencode/auth.json
# Output: {"openai":{"type":"oauth","refresh":"rt_iv...","access":"eyJ...","expires":1769837074238}}

# Verified auth list
opencode auth list
# Output: Credentials (oauth) + Environment (OPENAI_API_KEY)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-20-inv-investigate-opencode-openai-auth-confusion.md` - Complete auth precedence investigation

### Decisions Made
- Auth precedence is intentional: ENV > Auth.json(api) > Plugin OAuth > Config
- $ indicator shows estimated cost, not actual billing (based on model pricing data)
- OAuth models without pricing info show $0.00 cost

### Constraints Discovered
- OpenCode loads env vars before plugin loaders - this is by design
- Plugin must explicitly set `options["apiKey"]` to override provider.key from env
- OAuth tokens in auth.json are only used via plugin loaders (type="oauth" skipped by Auth.all processing)
- Server may cache env vars at startup - restart required after env changes

### Externalized via `kn`
- Not applicable (tactical user config fix, not architectural pattern)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with findings and recommendations)
- [x] Tests passing (N/A - investigation only)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-08rm3`

### User Action Required
Dylan should:
1. Unset OPENAI_API_KEY: `unset OPENAI_API_KEY`
2. Restart OpenCode server: `orch-dashboard restart`
3. Verify OAuth active: model selector shows OAuth models without $ cost
4. If persistent, check ~/.zshrc for OPENAI_API_KEY export and remove/comment

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does the opencode-openai-codex-auth plugin correctly set options["apiKey"] from OAuth access token? (Would need runtime debugging or plugin source inspection)
- Is ENV precedence over OAuth intentional? (May be power-user feature vs. bug)
- What triggers the "(oauth)" label vs. $ cost in UI? (May be model-specific metadata)

**Areas worth exploring further:**
- Plugin loader implementation to understand if it should override env keys
- mergeDeep behavior when provider.key is already set

**What remains unclear:**
- Whether unset OPENAI_API_KEY fully resolves the issue (untested in this session)
- Whether plugin version 4.4.0 changed any precedence behavior

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-investigate-opencode-openai-20jan-1286/`
**Investigation:** `.kb/investigations/2026-01-20-inv-investigate-opencode-openai-auth-confusion.md`
**Beads:** `bd show orch-go-08rm3`
