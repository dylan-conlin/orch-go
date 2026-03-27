# Brief: orch-go-pifxj

## Frame

The dashboard was pinning `orch serve` at 100% CPU every 10 minutes. The culprit was `CalculateOverrideTrend` — a function that counts verification bypasses over the last 14 days. To find those few dozen events, it was reading all 184,000 lines of the 64MB legacy `events.jsonl`, JSON-parsing each one, then throwing away 99.9% of them. Meanwhile, the rotation system that was supposed to solve this was already in place — `ScanEventsFromPath`, monthly file splitting, time-bounded reading — just never wired into this function.

## Resolution

The fix was wiring, not invention. `CalculateOverrideTrend` now calls `ScanEventsFromPath` with proper time bounds instead of doing its own `bufio.Scanner` loop on the monolith. That was the obvious part. The less obvious part: the legacy file still gets included because `EventFiles` always opens it regardless of the query window. Added an mtime check — if the legacy file's last-modified time is older than your `after` bound, it can't contain any relevant events, so skip it entirely. One `os.Stat` call saves 64MB of I/O. The legacy file was last written March 22, so this kicks in automatically around April 5 without anyone having to remember.

Also added stampede protection to the cache. The old code used a `sync.RWMutex` that let N concurrent requests past the read lock simultaneously, all discovering the cache was stale, all firing independent full-file scans. Now a `computing` flag gates recomputation — one goroutine refreshes, the rest get the previous value.

## Tension

The current month's rotated file is already 49MB. Monthly rotation prevents the legacy file from growing, but a busy month can still produce a file that takes hundreds of milliseconds to scan. The existing `SeekToTimestamp` function could skip directly to the relevant byte range, but it's not wired into `ScanEvents`. That's a separate piece of work — and it only matters if a single month's file becomes a bottleneck, which hasn't happened yet with the 10-minute cache TTL.
