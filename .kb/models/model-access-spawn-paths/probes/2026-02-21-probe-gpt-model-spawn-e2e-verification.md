# Probe: GPT Model Spawning End-to-End Verification

**Model:** model-access-spawn-paths
**Date:** 2026-02-21
**Status:** Complete

---

## Question

Do GPT model aliases (codex, gpt-5, gpt-5.2) resolve correctly and successfully spawn agents end-to-end via both tmux and headless paths on the opencode backend?

---

## What I Tested

### 1. Model Resolution (Code Review + Runtime)

Verified alias mappings in `pkg/model/model.go` against OpenCode's provider config (`~/.config/opencode/opencode.jsonc`).

| Alias | Resolves to | OpenCode Configured? |
|-------|------------|---------------------|
| `codex` | `openai/gpt-5.2-codex` | Yes |
| `gpt-5` | `openai/gpt-5` | **No** (only 5.1, 5.2 exist) |
| `gpt5-latest` | `openai/gpt-5.2` | Yes |
| `codex-5.2` | `openai/gpt-5.2` | Yes |
| `codex-mini` | `openai/gpt-5.1-codex-mini` | Yes |
| `codex-max` | `openai/gpt-5.1-codex-max` | Yes |

### 2. Tmux + Opencode + Codex

```bash
./orch spawn --bypass-triage --backend opencode --model codex --tmux hello 'say hello and exit'
```

### 3. Headless + Opencode + Codex

```bash
./orch spawn --bypass-triage --backend opencode --model codex --headless --no-track hello 'say hello'
```

### 4. Headless + Opencode + GPT-5

```bash
./orch spawn --bypass-triage --backend opencode --model gpt-5 --headless --no-track hello 'say hello'
```

### 5. Headless + Opencode + GPT-5.2 (gpt5-latest)

```bash
./orch spawn --bypass-triage --backend opencode --model gpt5-latest --headless --no-track hello 'say hello'
```

---

## What I Observed

### Test Results Summary

| Test | Model | Mode | Result | Evidence |
|------|-------|------|--------|----------|
| 1 | codex (gpt-5.2-codex) | tmux | **FAIL** | `opencode attach` has no `--model` flag; command exits with usage |
| 2 | codex (gpt-5.2-codex) | headless | **PASS** | Session created, 11 messages processed |
| 3 | gpt-5 | headless | **FAIL** | Session created but 0 responses (model not in OpenCode config) |
| 4 | gpt5-latest (gpt-5.2) | headless | **PASS** | Session created, 5 messages processed |

### Bug 1: `opencode attach` lacks `--model` flag

`runSpawnTmux()` calls `BuildOpencodeAttachCommand()` (`pkg/tmux/tmux.go:268`) which adds `--model "openai/gpt-5.2-codex"` to the command. But `opencode attach` does not support `--model`:

```
Options for opencode attach:
  -h, --help        show help
  -v, --version     show version number
      --print-logs  print logs to stderr
      --log-level   log level
      --dir         directory to run in
  -c, --continue    continue the last session
  -s, --session     session id to continue
      --fork        fork the session when continuing
  -p, --password    basic auth password
```

The command fails silently (shows usage and exits). The 15-second TUI ready timeout then fires:
```
Error: failed to start opencode: timeout waiting for OpenCode TUI to be ready after 15s
```

**Impact:** ALL tmux+opencode spawns with non-default model are broken. The model flag is silently ignored.

### Bug 2: `gpt-5` alias maps to unconfigured model

The alias `gpt-5` resolves to `openai/gpt-5`, but OpenCode's provider section only has:
- `gpt-5.1`, `gpt-5.1-codex`, `gpt-5.1-codex-mini`, `gpt-5.1-codex-max`
- `gpt-5.2`, `gpt-5.2-codex`

Session creation succeeds (API doesn't validate model existence), but no response is ever generated. The session stalls silently with 1 message (the prompt).

**Impact:** `gpt-5` alias is a silent failure path — appears to succeed but produces no work.

### Bug 3: No model existence validation

The spawn pipeline validates:
- Model-provider compatibility (`validateModelCompatibility`)
- Flash model blocking (`validateModel`)

But it does NOT validate whether the specific model ID exists in OpenCode's provider config. This means any typo or unconfigured model ID creates a silent zombie session.

### Headless Codex Agent Behavior

The codex session (gpt-5.2-codex) with 11 messages showed the agent reading SPAWN_CONTEXT.md, attempting to follow phase reporting protocol, and making tool calls. This confirms GPT-5.2-codex CAN process spawn contexts and follow protocol when the model is properly configured.

However, the prior probe (2026-02-19) found GPT-5.2-codex unreliable with large spawn contexts (63-76KB), hallucinating constraints and exhausting context windows. The "hello" skill with minimal context (small spawn context) is a much easier case.

---

## Model Impact

- [x] **Confirms** invariant: "Non-Anthropic providers require OpenCode backend" — openai models correctly auto-route to opencode backend via `modelBackendRequirement()`.
- [x] **Contradicts** invariant: "Escape hatch provides true independence" — tmux mode with opencode backend is broken for model selection because `opencode attach` doesn't support `--model`. The tmux "escape hatch" only works for the claude backend path.
- [x] **Extends** model with: **Tmux+opencode model selection is broken** — `BuildOpencodeAttachCommand` generates a `--model` flag that `opencode attach` doesn't support, causing silent failure for all non-default models. Only headless mode correctly passes models via HTTP API.
- [x] **Extends** model with: **Silent zombie sessions from unconfigured models** — `gpt-5` alias resolves to `openai/gpt-5` which isn't configured in OpenCode, creating sessions that never process. No validation catches this.

---

## Notes

### Recommendations

1. **Fix `opencode attach` to support `--model`** — this is the OpenCode fork; we can add the flag. Alternatively, the tmux spawn path could pre-create the session via HTTP API (like headless does) and then attach to it with `--session <id>`.

2. **Remove or remap `gpt-5` alias** — either remove the alias entirely or map it to `gpt-5.2` (latest available). Current mapping to `openai/gpt-5` is a guaranteed failure.

3. **Add model existence validation** — before spawning, query OpenCode's provider config to verify the model ID exists. This prevents zombie sessions.

### Working Combinations

For GPT model spawning today, only these paths work:
- `--model codex --headless` (or `--model codex` without flags, since headless is default)
- `--model gpt5-latest --headless`
- `--model codex-5.2 --headless`
- Any codex variant with `--headless`

**Do NOT use `--tmux` with opencode backend and a model flag** — it silently fails.
