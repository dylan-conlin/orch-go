# Brief: orch-go-jhluq

## Frame

Yesterday's fix (scs-sp-vzm) saved an agent from being wrongly abandoned — but it only works for agents that run through OpenCode. The fix asks "is the agent actively generating a response?" and suppresses the UNRESPONSIVE alarm if yes. Claude Code agents can't answer that question. They have no API. So the 30-minute phase timeout fires indiscriminately, and any Claude Code agent doing deep investigation work looks dead to the system.

## Resolution

I expected to find a gap — "Claude Code has no processing signal, so we need a different approach." What I actually found is that the signal exists, just not where I was looking. Claude Code writes session files to `~/.claude/sessions/{pid}.json` with the PID of every running instance. A `kill -0` check tells you if the process is alive in 1 millisecond. And tmux — which we already use for window management — can capture the pane content and hash it. If the hash changes between polls, the agent is actively generating output. That's the `IsProcessing` equivalent.

The surprise was how cleanly this composes. The existing code already has a `!IsProcessing` guard that suppresses UNRESPONSIVE flags — it just never gets set for Claude Code agents. Wire PID liveness + pane content delta into that same boolean, and the existing guard handles the rest. Three layers: PID alive (is the process running?), pane content changing (is it doing work?), phase timeout (is it reporting progress?). Each layer narrows the diagnosis. The first two are cheap and fast; the third remains as backstop for agents that are truly stuck.

## Tension

The pane content hashing approach is sound in theory but I haven't tested it on an actively-generating agent at two different time points to confirm the hash actually changes. ANSI escape codes in the terminal output could cause false positives (hash changes when nothing meaningful changed) or the opposite problem if tmux caches pane content. The implementation needs to strip ANSI before hashing — which is easy — but the approach should be validated empirically before building the full tracker. Worth a quick 5-minute test during the architect session.
