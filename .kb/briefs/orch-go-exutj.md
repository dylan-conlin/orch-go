# Brief: orch-go-exutj

## Frame

OpenCode kept going dark mid-session — no agents could spawn, no error surfaced, and the only fix was a manual dashboard restart. This happened repeatedly during long sessions, always the same: dashboard fine, daemon fine, just OpenCode silently unreachable on port 4096.

## Resolution

I expected to find crashes — OOM kills, Bun segfaults, something dramatic in the system logs. There was nothing. Zero crash reports. Zero jetsam events. The process was alive the entire time, same PID across every occurrence. That was the turn: the process wasn't dying, it was hanging.

The supervision layer was completely blind to this. Overmind's auto-restart and the ServiceMonitor both work by detecting PID changes — if the process stays alive, they report everything's fine. Meanwhile, OpenCode's event loop was blocked and couldn't serve HTTP requests. The root cause is in OpenCode's SSE event bus: Bus.publish() awaits every subscriber, and dead SSE connections accumulate over long sessions without cleanup. One blocked subscriber blocks all event processing, which blocks the HTTP server.

The fix adds HTTP liveness probing to the ServiceMonitor. Every 10 seconds, it makes a real HTTP request to OpenCode's /session endpoint. Three consecutive failures with an unchanged PID triggers a force-restart via overmind. This turns the manual orch-dashboard restart into automatic recovery in about 30 seconds.

## Tension

This is a band-aid on the orch-go side — it restarts the corpse reliably, but doesn't stop the killing. The actual fix is in the OpenCode fork: the SSE subscription model needs subscriber timeouts, dead connection detection, and probably a max listener cap. Every restart loses in-flight agent state. The question is whether the automatic recovery is good enough to live with while the fork fix gets prioritized, or whether the silent data loss from mid-session restarts is worse than the current manual restart workflow.
