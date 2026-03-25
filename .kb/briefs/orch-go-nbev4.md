# Brief: orch-go-nbev4

## Frame

When you're orchestrating from scs-special-projects, the briefs page still shows orch-go briefs. Every other dashboard view follows your active tmux window — the work graph, beads stats, attention items — but briefs was the holdout, hardcoded to wherever `orch serve` started.

## Resolution

The fix was mechanical once the pattern was clear: `serve_beads.go` and `serve_attention.go` already accept a `project_dir` query parameter and fall back to `sourceDir`. The briefs endpoints were just never wired the same way. Three backend functions needed the parameter threaded through, plus the frontend store methods. The only non-obvious decision was keying read state by `projectDir:beadsID` instead of bare `beadsID` — two projects could theoretically share a beads ID, and marking one read shouldn't mark the other. The frontend subscribes to `orchestratorContext` (same pattern as work-graph) and re-fetches when the project changes, clearing the brief content cache so stale content from the previous project doesn't bleed through.

## Tension

The server needs a rebuild to pick up these changes. I couldn't do a live reproduction test against the running server — the test suite covers it, but the real proof is switching projects in the dashboard after `orch-dashboard restart`.
