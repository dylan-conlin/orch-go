# Brief: orch-go-k8sle

## Frame

The orchestrator needs to know whether a Claude Code agent is alive and working. Yesterday's investigation (jhluq) found three liveness signals — PID alive, tmux pane content changes, phase timeout — and said the problem was that no IsProcessing signal exists for Claude agents. I was spawned to design how to compose those signals into one.

## Resolution

I started by reading the code to understand what jhluq found, and immediately hit a turn: IsProcessing IS set for Claude agents. It has been all along. When discovery classifies a Claude agent as "active" (because a phase comment exists), the conversion function maps active → IsProcessing=true. The guard in the UNRESPONSIVE detection then skips them. This is the OPPOSITE of what jhluq concluded.

But the signal is wrong. It's static. A Claude agent that reported "Phase: Planning" forty-five minutes ago and then crashed still looks "processing" to the system forever, because the phase comment that sets its status persists after death. OpenCode agents don't have this problem — their session API gives a live "is this agent generating right now?" signal that overrides the static mapping. Claude agents have no equivalent override.

The fix mirrors what already works for OpenCode: add a live signal that overrides the static one. `IsPaneActive()` — an existing, tested function in pkg/tmux that checks whether a non-shell process is running in the pane — is exactly that signal. It answers "is the claude process actively running right now?" in ~10ms. Wire it into discovery (which already queries tmux but throws away the window ID), then let both consumers use it as an override. Three files, one pattern.

The deeper fix is reordering discovery's signal priority for Claude agents. Currently, a historical phase comment takes priority over the live tmux window check. A dead agent with a phase comment is classified "active" because the phase check comes first and the tmux check is only a fallback. Flipping the order — check if the window is alive first, then use phase for categorization — prevents dead agents from being masked by their own history.

## Tension

I chose IsPaneActive over the more precise pane-content-delta approach (hashing output between polls) that jhluq proposed. IsPaneActive is simpler and sufficient for autonomous agents that don't idle at prompts. But if agents start sitting at idle prompts without exiting — claude process alive, not actually working — IsPaneActive will say "processing" and the UNRESPONSIVE timer will never fire. The pane delta approach would catch that, but adds a StallTracker-like state machine. Worth monitoring whether the idle-prompt scenario is real before adding that complexity.
