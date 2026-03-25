# Brief: orch-go-p1v54

## Frame

Dead Claude agents were invisible. An agent that crashed after reporting "Phase: Planning" forty-five minutes ago looked the same as one actively writing code — permanently "processing" in the dashboard, counted as "active" in the swarm, never flagged for human attention. Three prior agents had patched the symptom (consumer-side overrides for IsProcessing), but the root classification in discovery still said "active" based on a historical comment that would never update.

## Resolution

The fix was a signal priority reorder in discovery.go — one structural change rather than patches at each consumer. Previously, phase comments were the strongest signal for Claude agent liveness: if an agent had ever reported a phase, it was "active." Now, tmux pane state (is the claude process actually running?) is the primary signal. Phase comments remain as metadata — they tell you WHAT the agent was doing, not WHETHER it's alive.

This also exposed why the prior consumer overrides were insufficient. They corrected `IsProcessing` (the binary "generating output right now?" flag) but left `Status` unchanged. A dead Claude agent would show IsProcessing=false but Status="active", getting counted as "idle" in the swarm rather than "phantom." With the discovery-level fix, dead agents get Status="dead" from the source, and both IsProcessing and Status are correct without consumer intervention.

Added `TmuxWindowID` and `IsProcessing` directly to the `AgentStatus` struct so discovery is the single source of truth. Removed the redundant tmux check loops from both status_cmd.go and serve_agents_handlers.go — they were compensating for a bug that no longer exists.

## Tension

IsPaneActive checks whether any non-shell process is running in the pane. This works for agents that exit when done. But if a Claude agent stalls at an idle prompt (process alive, not doing anything), IsPaneActive will say "processing" and the UNRESPONSIVE timer will never fire. The jhluq investigation proposed a pane-content-delta approach (hashing output between polls) that would catch this, but it adds StallTracker-like state machinery. Worth watching whether idle-prompt stalls actually happen before building for it.
