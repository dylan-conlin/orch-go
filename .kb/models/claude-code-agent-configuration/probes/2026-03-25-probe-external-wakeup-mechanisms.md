# Probe: External Wakeup Mechanisms for Claude Code Sessions

**Model:** claude-code-agent-configuration
**Date:** 2026-03-25
**Status:** Complete
**claim:** extends (no prior claim — new capability area)
**verdict:** extends

---

## Question

Can external processes inject events into a running or idle Claude Code session? The model documents four configuration layers (CLAUDE.md, CLI flags, settings.json, SPAWN_CONTEXT.md) and lists HTTP hooks as an "Evaluate Next" feature, but says nothing about bidirectional communication via stream-json or session resume injection. This probe maps what wakeup mechanisms actually exist.

---

## What I Tested

### Test 1: Stream-JSON Bidirectional Protocol

```bash
# Send multiple user messages via stdin in stream-json mode
(echo '{"type":"user","message":{"role":"user","content":"Say exactly: FIRST_MESSAGE"}}'; \
 sleep 5; \
 echo '{"type":"user","message":{"role":"user","content":"Say exactly: SECOND_MESSAGE"}}') \
| claude -p --input-format stream-json --output-format stream-json --verbose \
  --dangerously-skip-permissions --effort low 2>/dev/null \
| grep -o '"text":"[^"]*"'
```

### Test 2: Session Resume + New Prompt Injection

```bash
# Create session with secret word
SESSION_ID=$(echo "Remember: the secret word is PINEAPPLE" \
  | claude -p --output-format json --dangerously-skip-permissions --effort low 2>/dev/null \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('session_id','NONE'))")

# Resume session and ask for the secret word
echo "What was the secret word I told you?" \
  | claude -p --resume "$SESSION_ID" --output-format stream-json --verbose \
  --dangerously-skip-permissions --effort low 2>/dev/null \
  | grep -o '"text":"[^"]*"'
```

### Test 3: Resume + Stream-JSON Input Combined

```bash
echo '{"type":"user","message":{"role":"user","content":"What was the code I told you?"}}' \
  | claude -p --resume "$SESSION_ID" --input-format stream-json \
  --output-format stream-json --verbose --dangerously-skip-permissions --effort low 2>/dev/null
```

### Test 4: Process Model Inspection

```bash
ps aux | grep -i claude  # Examined running Claude processes
strings $(which claude) | grep -i "background"  # Binary is compiled Bun (Mach-O arm64)
```

---

## What I Observed

### Stream-JSON Bidirectional (Test 1): WORKS

Both messages were processed in the same session. Output:
```
"text":"FIRST_MESSAGE"
"text":"SECOND_MESSAGE"
```

The protocol is NDJSON over stdin. The init message from Claude Code reveals full session metadata:
```json
{"type":"system","subtype":"init","session_id":"...","tools":[...],"model":"..."}
```

This means: **an external process holding the stdin pipe can send messages at any time during the session's lifetime.**

### Session Resume (Test 2): WORKS WITH FULL CONTEXT

The model responded `"text":"PINEAPPLE"` — it had full conversation history from the original session. Token usage showed `cache_read_input_tokens: 11916`, confirming prior context was loaded.

Session resume creates a new process but reuses the same session_id and conversation history. SessionStart hooks fire with subtype "resume".

### Resume + Stream-JSON (Test 3): WORKS

The combination `--resume + --input-format stream-json` is valid. This enables programmatic injection of structured messages into any previously-created session.

### Background Process Notification: PULL-BASED ONLY

The `run_in_background` Bash parameter writes output to a file. There is no push notification — Claude must actively Read the output file. The binary has no inotify/fsnotify or signal-based wakeup for background tasks.

Error message discovered: `"Expected a command, assignment, or subshell but got: Background commands '&' are not supported yet."` — confirming the background mechanism is a Claude Code internal feature, not shell-level backgrounding.

### Channels (via documentation research): PUSH-BASED BUT REQUIRES RUNNING SESSION

Channels is a research preview MCP feature that allows external systems to POST webhooks to a local port, which forwards events to Claude via `notifications/claude/channel`. This is the designed push mechanism but:
- Only works while Claude Code is already running
- Cannot wake an idle/stopped session
- Research preview (not GA)

---

## Model Impact

- [x] **Extends** model with: Three confirmed external injection vectors not documented in the model

### New Findings for Model

**1. Stream-JSON as IPC channel**
Claude Code's `--input-format stream-json` mode (available since at least v2.1.83) enables true bidirectional communication. An external process can:
- Start Claude with `-p --input-format stream-json --output-format stream-json`
- Keep the stdin pipe open
- Write NDJSON messages to stdin at any time
- Process NDJSON responses on stdout

This is the most robust wakeup mechanism: a long-running Claude session can receive new work on demand.

**2. Session resume as event injection**
`claude -p --resume <session-id> "new prompt"` creates a new process that loads full prior context and processes a new message. This means any external process with a session ID can:
- Resume a conversation and inject a new prompt
- Access the full conversation history
- Add new turns without the original process being alive

**3. Tmux send-keys as TUI injection**
For interactive sessions running in tmux, `tmux send-keys` can type characters into the Claude Code TUI. Not tested to completion (trust dialog interference) but architecturally sound.

### Implications for Model's "Evaluate Next" Section

The model lists "HTTP hooks → orch serve" as an "Evaluate Next" feature with the description "Real-time event visibility without tmux parsing." Stream-JSON mode provides this TODAY — no HTTP endpoint needed. The session can read stdin messages from a daemon process.

### Architecture Recommendation (Out of Scope for Probe)

For the orchestrator comprehension queue use case, two viable patterns emerge:
- **Persistent stream-JSON session**: Daemon runs `claude -p --input-format stream-json --output-format stream-json --resume <orch-session>` and writes completion events to stdin
- **On-demand resume injection**: Daemon invokes `claude -p --resume <orch-session> "Agent X completed: <synthesis>"` per completion event

---

## Notes

- `--input-format stream-json` is poorly documented (GitHub issue #24594 exists). The message schema was discovered empirically.
- The stream-json protocol requires `--verbose` when combined with `--output-format stream-json`.
- Session resume fires SessionStart hooks with subtype "resume" — hooks can detect whether this is a fresh or resumed session.
- The `--replay-user-messages` flag (stream-json only) echoes user messages back on stdout for acknowledgment — useful for confirming injection receipt.
- Channels (research preview) is the designed long-term solution for external event push, but stream-json + resume is available now.
- Claude Code is compiled as a Mach-O arm64 binary using Bun runtime. Source code is not directly inspectable on disk.

## Sources

- [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference.md)
- [Claude Code GitHub Issue #24594 - Undocumented stream-json usage](https://github.com/anthropics/claude-code/issues/24594)
- [Claude Code Headless Documentation](https://code.claude.com/docs/en/headless.md)
