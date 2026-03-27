# Brief: orch-go-iwp5d

## Frame

The daemon spawned led-totem-toppers-916 twice, both times logging "Spawned" and marking the issue in_progress. But no tmux window or workspace was ever created. The issue appeared as a phantom agent in orch status — stuck in_progress with nothing actually running, invisible to cleanup, blocking future spawn attempts.

## Resolution

The daemon trusted `orch work`'s exit code as proof of success. If the subprocess exited 0 — even without creating a workspace — the daemon logged "Spawned" and moved on. The fix adds a simple check after `SpawnWork` returns: scan `.orch/workspace/` for a workspace whose SPAWN_CONTEXT.md references the beads ID. If none is found, treat it as a spawn failure — roll the issue back to open, unmark the spawn tracker, release the pool slot. The daemon now distinguishes between "the command ran" and "the agent exists."

The surprise was that workspace names don't contain the beads ID at all — they use a random 4-char hex suffix. So the verification had to check SPAWN_CONTEXT.md file content, not directory names. I added a `WorkspaceVerifier` interface (following the existing daemon pattern of Spawner, IssueUpdater, etc.) so the check is mockable in tests.

## Tension

The original led-totem-toppers incident also showed a `bd show` failure during dependency checking. Whether that failure caused the phantom spawn or was a separate symptom is unclear. The workspace verification catches the phantom regardless of root cause, but the dependency-check-failure-then-spawn-anyway path might warrant its own investigation.
