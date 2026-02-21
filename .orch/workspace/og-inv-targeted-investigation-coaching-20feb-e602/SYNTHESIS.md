# Targeted Investigation: Coaching Plugin Injection

## Findings

1. **Injection mechanism**
   - `injectCoachingMessage` and `injectHealthSignal` use `client.session.prompt({ sessionID, prompt, noReply: true })` (and `promptAsync` for coach streaming). This inserts a new message/parts into the session rather than modifying the system prompt. No `experimental.chat.system.transform` is used in the coaching plugin.

2. **Where injected content is stored**
   - The injected content is part of the OpenCode session message stream. If using the OpenCode storage schema, this would appear as a new message + part record(s) in OpenCode’s session storage (not in any orch-go DB tables).

3. **Runtime logging**
   - Debug logging is gated by `ORCH_PLUGIN_DEBUG=1` (`log()` calls). Metrics are written to `~/.orch/coaching-metrics.jsonl` via `writeMetric()`.

4. **Verify `tool.execute.after` firing**
   - The `event-test` plugin logs `tool.executed.*` entries to `~/.orch/event-test.jsonl`. Recent log entries show `tool.executed.bash` for the current gpt-5.2-codex worker session, which confirms the `tool.execute.after` hook fires for this model/agent type.

## Evidence Pointers

- Coaching plugin source: `.opencode/plugin/coaching.ts`
- Plugin guide: `.kb/guides/opencode-plugins.md`
- Event test log: `~/.orch/event-test.jsonl` (recent entries show tool hook events with modelID `gpt-5.2-codex`)
