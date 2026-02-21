# Probe: Coaching plugin injection + tool.execute.after firing

**Model:** opencode-session-lifecycle
**Date:** 2026-02-20
**Status:** Complete

---

## Question

Does OpenCode plugin context injection use session.prompt (message/part insertion) rather than system prompt mutation, and does `tool.execute.after` fire for gpt-5.2-codex worker sessions?

---

## What I Tested

Reviewed project plugin implementation and plugin system guide, then checked the event-test log near the end for tool hook entries from a gpt-5.2-codex session.

```bash
pwd
wc -l "/Users/dylanconlin/.orch/event-test.jsonl"
# Read recent lines near end of event-test.jsonl
```

---

## What I Observed

- `coaching.ts` injects coaching/health signals via `client.session.prompt({ sessionID, prompt, noReply: true })` (and `promptAsync` for coach streaming), which indicates a new message with parts is inserted into the session rather than modifying the system prompt.
- The OpenCode plugin guide describes context injection via `client.session.prompt` with `noReply: true`, not via `experimental.chat.system.transform`.
- `event-test.jsonl` shows `tool.executed.bash` entries for the current worker session with `modelID: "gpt-5.2-codex"` (session metadata + tool hook event), indicating `tool.execute.after` fires for gpt-5.2-codex sessions.

---

## Model Impact

- [ ] **Confirms** invariant: 
- [ ] **Contradicts** invariant: 
- [x] **Extends** model with: Plugin context injection uses `client.session.prompt` (message/part insertion) and `tool.execute.after` fires for gpt-5.2-codex worker sessions (evidenced by event-test tool hook entries).

---

## Notes

- Coaching plugin also writes metrics to `~/.orch/coaching-metrics.jsonl` and only logs to console when `ORCH_PLUGIN_DEBUG=1`.
