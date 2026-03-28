# Brief: orch-go-lno6l

## Frame

The sketchybar widget was doing two separate things to figure out if the daemon was alive: parsing the PID out of daemon-status.json and sending it a signal (`kill -0`), then also parsing the `last_poll` ISO timestamp to compute how stale the data was. Both required the JSON to be intact and parseable. If the daemon died mid-write or left corrupt JSON, the widget had no way to detect it — the very mechanism for detecting death depended on the dead daemon's last output being readable.

## Resolution

The daemon writes daemon-status.json atomically on every poll cycle. That means the file's modification time *is* the liveness signal — it's set by the OS when the write completes, outside the file's content. One `stat -f %m` call replaces both the PID extraction + kill signal and the ISO timestamp parsing. Same thresholds as before: file untouched for >2 minutes → yellow (stalling), >10 minutes → red and marked dead. The comprehension fallback to `bd search` still triggers at the yellow threshold.

The parity test between Go health computation and bash was already in place. All 8 existing scenarios still pass because freshly-written test files have recent mtime (green), which matches the Go path computing green from `last_poll: now - 30s`. Added 3 new test cases using `os.Chtimes()` to backdate files and verify yellow/red/dead transitions.

## Tension

The Go server path (`ComputeDaemonHealth`) still uses the `last_poll` JSON field for liveness, while the bash widget now uses mtime. In practice they're equivalent — the daemon sets `last_poll = time.Now()` at the same moment it writes the file. But it's a conceptual split: two systems detecting the same thing differently. Worth deciding whether the Go path should also move to mtime, or whether the PID-based validation in `ReadValidatedStatusFile()` already covers that need well enough.
