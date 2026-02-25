# Session Synthesis

**Agent:** og-inv-orientation-frame-verify-21feb-02c1
**Issue:** orch-go-1170
**Outcome:** success

---

## Plain-Language Summary

Tested GPT model spawning end-to-end through the orch spawn pipeline. Found that **headless mode works** for properly-configured models (codex/gpt-5.2-codex, gpt5-latest/gpt-5.2) but **tmux mode is completely broken** for non-default models because `opencode attach` doesn't support a `--model` flag. Also found that the `gpt-5` alias maps to an unconfigured model in OpenCode, creating silent zombie sessions. Two of four tested combinations work; two fail silently.

## Verification Contract

See probe: `.kb/models/model-access-spawn-paths/probes/2026-02-21-probe-gpt-model-spawn-e2e-verification.md`

Key outcomes:
- codex headless: PASS (11 messages processed)
- gpt-5.2 headless: PASS (5 messages processed)
- codex tmux: FAIL (`opencode attach` lacks `--model`)
- gpt-5 headless: FAIL (model not configured in OpenCode)

---

## Delta (What Changed)

### Files Created
- `.kb/models/model-access-spawn-paths/probes/2026-02-21-probe-gpt-model-spawn-e2e-verification.md` - End-to-end probe documenting 4 test combinations

### Files Modified
- None (investigation only, no code changes per scope)

---

## Evidence (What Was Observed)

- `opencode attach` CLI help shows no `--model` flag (only `--dir`, `--continue`, `--session`, `--fork`, `--password`)
- `BuildOpencodeAttachCommand()` in `pkg/tmux/tmux.go:276` adds `--model` that doesn't exist
- OpenCode config (`~/.config/opencode/opencode.jsonc`) has: gpt-5.1, gpt-5.1-codex, gpt-5.1-codex-mini, gpt-5.1-codex-max, gpt-5.2, gpt-5.2-codex — but NOT `gpt-5`
- Session `ses_37f404988ffeNwkYJHQRvSOYUk` (codex headless) had 11 messages and was actively processing
- Session `ses_37f4011c5ffeS2oEt3vT75DonA` (gpt-5 headless) stalled at 1 message for 30+ seconds
- Session `ses_37f3f0abbffeiiU4xsBnaKJRs1` (gpt-5.2 headless) had 5 messages and was actively processing

### Tests Run
```bash
# Test 1: tmux + codex
./orch spawn --bypass-triage --backend opencode --model codex --tmux hello 'say hello and exit'
# FAIL: timeout waiting for OpenCode TUI to be ready after 15s

# Test 2: headless + codex
./orch spawn --bypass-triage --backend opencode --model codex --headless --no-track hello 'say hello'
# PASS: Session created, 11 messages

# Test 3: headless + gpt-5
./orch spawn --bypass-triage --backend opencode --model gpt-5 --headless --no-track hello 'say hello'
# FAIL: Session created but 0 responses after 30s

# Test 4: headless + gpt-5.2
./orch spawn --bypass-triage --backend opencode --model gpt5-latest --headless --no-track hello 'say hello'
# PASS: Session created, 5 messages
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `opencode attach` does not support `--model` flag — tmux+opencode model selection is broken
- `gpt-5` is not a valid model in OpenCode's provider config — only versioned variants (5.1, 5.2) exist
- Session creation via HTTP API does not validate model existence — allows zombie sessions

### Externalized via `kb`
- `kb quick tried "tmux spawn with opencode backend and --model flag" --failed "opencode attach does not support --model"` → kb-20b075
- `kb quick tried "gpt-5 alias for headless opencode spawn" --failed "gpt-5 not configured in OpenCode"` → kb-4ed2cc

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Issues to Create

1. **Fix `opencode attach` to support `--model`** — or change tmux spawn path to pre-create session via API then attach with `--session`
2. **Remove or remap `gpt-5` alias** — map to `gpt-5.2` or remove entirely to prevent silent failures

---

## Unexplored Questions

- Does `opencode attach` with `--session` work if the session was pre-created with a model via API? This would be a workaround for the tmux model selection bug.
- How do GPT models handle large spawn contexts (60-76KB)? Prior probe (2026-02-19) found they hallucinate and exhaust context. This test used the minimal `hello` skill.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5 (investigation agent)
**Workspace:** `.orch/workspace/og-inv-orientation-frame-verify-21feb-02c1/`
**Probe:** `.kb/models/model-access-spawn-paths/probes/2026-02-21-probe-gpt-model-spawn-e2e-verification.md`
**Beads:** `bd show orch-go-1170`
