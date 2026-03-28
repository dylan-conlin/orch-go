# Brief: orch-go-vn7cf

## Frame

Two GPT-5.4 agents were reported as consistently stalling in Planning phase — one "UNRESPONSIVE BLOCKED" for hours, the other a phantom with a vanished tmux window. The bug report raised the question: is GPT-5.4 fundamentally unreliable for protocol-heavy skills, or is there something wrong with the infrastructure?

## Resolution

I started by assuming the bug report was accurate and went to verify the stalls. The turn came immediately: the first agent (Prior Art Check) had 11 messages in its OpenCode session. It worked fine — read its task, identified it was blocked by governance-protected files, reported BLOCKED within 33 seconds. The "stall" in orch status was just the display showing a correctly-blocked agent that had been sitting there for hours. For the second agent (Kenning trademark search), there was no OpenCode session at all. The `skill:research` label routes through the daemon's capability routing to `opus`, not GPT-5.4. This agent was probably never GPT-5.4 to begin with.

But there is a real GPT-5.4 problem hiding behind the misdiagnosis. Across 24 production sessions, 29% die silently — the prompt arrives, the assistant "responds" with zero text and zero tools in under 10 milliseconds, and the session looks alive but produced nothing. This clusters dramatically: Mar 24 was 100% failures (a 4-session batch), Mar 26 was 18%, and the last two days are clean. The clustering says infrastructure, not model. Meanwhile the controlled benchmark (N=21, 95% success) used direct API calls without the full spawn pipeline — it was testing model capability, not production reliability.

## Tension

The CLAUDE.md stat says "GPT-5.4 is significantly better (95% completion, N=21)" and that's true in a controlled setting. But production is 71%, and the difference might be entirely about OpenCode server stability during concurrent spawns — we don't actually know. The retry strategy investigation designed a fix (fingerprinted one-shot retry) but it's not implemented. Until it is, GPT-5.4 is quietly eating ~29% of the tasks it gets routed. Whether that matters depends on how much GPT-5.4 routing you actually want to do.
