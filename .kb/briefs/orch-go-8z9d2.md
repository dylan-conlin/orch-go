# Brief: orch-go-8z9d2

## Frame

The cmd/orch test suite was failing with two tests down. The ticket pointed at a SessionStatusInfo type mismatch from the opencode-to-execution migration, but the actual situation was messier: one test had the wrong type, another was reading the real events log and choking on a 239KB line.

## Resolution

The type fix was one line -- `opencode.SessionStatusInfo` to `execution.SessionStatusInfo` in the mock server response. The mock simulates the OpenCode HTTP API, so using `opencode.SessionStatusInfo` was technically correct for the wire format (both serialize to identical JSON), but it violated the migration direction where all cmd/orch code should reference the execution abstraction layer. The second fix was more interesting: `TestHandleAgentlogJSONResponse` had zero test isolation -- it read `~/.orch/events.jsonl` directly, and the file's longest line (239KB, from a session.spawned event with gap analysis data) exceeds `bufio.Scanner`'s 64KB default buffer. Added `ORCH_EVENTS_PATH` env var pointing to a temp file.

## Tension

The `readLastNEvents` production code has the same 64KB buffer problem -- any event line over 64KB will cause the agentlog API endpoint to return 500. Filed as orch-go-dfqhl, but worth noting: the events.jsonl format has no line-length contract, and enrichment fields (gap analysis, spawn metadata) are growing. This will break for users too, not just tests.
