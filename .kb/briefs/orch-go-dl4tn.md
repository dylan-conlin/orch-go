# Brief: orch-go-dl4tn

## Frame

Every daemon log line was appearing twice in `~/.orch/daemon.log`. The daemon has been running this way since launchd was set up — two independent write paths both targeting the same file, each unaware of the other. Classic Defect Class 5: contradictory authority signals where neither writer is wrong in isolation.

## Resolution

The DaemonLogger was designed for two audiences: stdout for whoever launched the process (terminal, tmux), and a direct file write for persistence. When launchd entered the picture, it redirected stdout to daemon.log via `StandardOutPath` — which meant the file was now receiving every line twice: once through the stdout capture and once through the direct `os.OpenFile` write.

The fix is a 12-line function that checks whether stdout's file descriptor and the log file resolve to the same inode (`os.SameFile`). If they match — meaning something upstream (launchd) already routes stdout to the log — the logger skips opening the file directly and just writes to stdout. This also fixes the same doubling in error output, since `Errorf` wrote to both stderr and the direct file handle. With the detection in place, `l.file` is nil under launchd, so stderr (also captured by launchd) is the single path.

## Tension

Log rotation has a subtle interaction with launchd: when `rotateIfNeeded` renames daemon.log to daemon.log.1, launchd's stdout fd follows the inode to the renamed file. New entries land in daemon.log.1 until launchd restarts the process. The detection check runs before rotation to avoid this race, but the rotation-under-launchd story itself is unresolved — worth thinking about whether rotation should be launchd's job (via newsyslog) rather than ours.
