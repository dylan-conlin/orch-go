# Brief: Why the Daemon Kept Dying

## Frame

The daemon had been killed and restarted 23 times. The symptom was exit code -9 (SIGKILL), which pointed at either the OOM killer or launchd's watchdog. I started by checking system logs for memory pressure events — nothing. Then `launchctl print` gave it away in one line: `exit timeout = 5`. launchd sends SIGTERM, waits 5 seconds, and if the process is still running, sends SIGKILL. The daemon was taking too long to shut down.

The blocking came from two places I didn't expect to find together. First, there's a `defer runReflectionAnalysis()` that runs `kb reflect --global` on every exit — an external process with no timeout. Second, the main loop only checks for the stop signal at the edges of its cycle (top and bottom), not between the dozen operations it runs in the middle. If SIGTERM lands while the loop is processing periodic tasks, completions, and listing issues, it has to wait for all of them to finish before it even notices it should stop. Together: easily 10+ seconds of exit delay against a 5-second budget.

## Resolution

Three changes that compose: (1) wrapped the exit-time reflection in a 3-second timeout via `exec.CommandContext`, (2) added `shutdownRequested()` gates between major operations in the main loop so the daemon notices SIGTERM within one operation's latency rather than a full cycle's, and (3) set `ExitTimeOut=15` in the plist as a safety net. After the fix, shutdown takes 23-25 milliseconds.

## Tension

The context gates are coarse-grained — they check between operations, not within them. If a single operation (say, `ListReadyIssuesMultiProject` hitting a slow beads RPC) takes longer than 15 seconds, we're back to SIGKILL. The proper fix would be threading context through every operation in the main loop. I chose not to do that because (a) none of the current operations individually take that long, (b) the code change would be much larger, and (c) the 15-second ExitTimeOut provides margin. But if the daemon starts polling more projects or beads RPC gets slower, this will need revisiting.
