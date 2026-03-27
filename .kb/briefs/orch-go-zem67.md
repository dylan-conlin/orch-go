# Brief: Daemon Verification Pause Stale State

## Frame

The daemon has a safety valve that pauses spawning after 3 agents complete without human review. Good design — it prevents work from piling up unreviewed. But the counter tracking those completions was set once at startup and never re-checked. So when the actual work got closed (through headless completions, automated paths), the daemon was still holding its breath waiting for someone to review work that no longer existed.

The symptom was confusing: `orch daemon status` said "PAUSED (3/3 unverified)" while `orch review` said "No pending completions." It looked like the system was disagreeing with itself, and since the daemon trusts the pause, it was stuck — wouldn't spawn new work, even with 5 empty slots.

## Resolution

The in-memory tracker had no refresh mechanism. It was seeded from `ListUnverifiedWork()` at startup, but the only way to reduce the counter was through a signal file written by interactive `orch complete`. Non-interactive closures (headless, bd close) correctly *don't* write that signal — that's intentional, to preserve the "human must review" invariant.

The fix: when the daemon is paused, re-check actual reality before staying paused. `ResyncWithBacklog()` takes the current `ListUnverifiedWork()` result, prunes stale entries from the tracker's seen set, and auto-unpauses if the real count drops below threshold. This runs each pause cycle, so the daemon unsticks within one poll interval.

## Tension

The 206 stale checkpoints in the checkpoint file (most with gate2=false for long-closed issues) are a separate concern. `ListUnverifiedWork()` correctly filters them out by checking issue status, so they don't cause bugs. But they accumulate forever — the checkpoint file is append-only with no compaction. If this file grows large enough to matter, or if the open-issue filter ever breaks, those 206 phantom entries become 206 phantom blockers. Worth considering a checkpoint compaction mechanism, but that's a separate decision.
