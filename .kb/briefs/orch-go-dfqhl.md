# Brief: orch-go-dfqhl

## Frame

The `/api/agentlog` endpoint was silently dropping events. Any event line in `events.jsonl` exceeding 64KB — and some reach 239KB with full agent output payloads — caused Go's `bufio.Scanner` to error out and stop reading entirely. Everything after the first oversized line disappeared from the dashboard.

## Resolution

This was a one-line root cause: `bufio.Scanner` defaults to a 64KB max token (line) size, and `readLastNEvents` never overrode it. The fix is `scanner.Buffer(make([]byte, 64*1024), 1024*1024)` — keeps the initial allocation small (64KB) but allows lines up to 1MB. The SSE streaming paths were not affected because they use `bufio.NewReader` + `ReadString('\n')`, which has no line length cap. Only the batch JSON endpoint hit this.

The interesting detail: this is the kind of bug that gets worse over time. As agents produce longer output, more events cross the 64KB threshold, and the dashboard shows fewer and fewer events. It degrades gradually rather than failing loudly.

## Tension

The 1MB cap is generous but still a cap. If event lines ever exceed 1MB, this breaks again with the same silent failure mode. The real question is whether `events.jsonl` lines should have unbounded growth, or whether there should be a truncation policy at write time rather than a read-time buffer gamble.
