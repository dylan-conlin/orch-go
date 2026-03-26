# Brief: orch-go-y4k6w

## Frame

We decided to move worker execution from OpenCode to OpenClaw. The decision was made, but no one had looked at the actual wiring to figure out what "move" means concretely. The OpenCode fork is 975 commits behind upstream, maintaining it costs real hours, and the claude CLI backend already proves the orchestration layer doesn't need OpenCode. But 48 files import `pkg/opencode/` — so how do you take it out without breaking everything that grew around it?

## Resolution

I expected the hard part to be the spawn boundary — swapping one session-creation API for another. That turned out to be the easy part. The `backends.Backend` interface is already clean: `Spawn(ctx, req) → Result`. Adding an OpenClaw implementation there is ~300 lines and a new file.

The surprise was the type leakage. Twenty-six files reference `opencode.Session`, `opencode.Message`, and `opencode.TokenStats` not because they call the API, but because those types leaked into status display, token counting, activity export, and the dashboard server. The spawn boundary is a door; the type boundary is kudzu.

So the design is four phases. Phase 1 extracts a `SessionClient` interface with backend-agnostic types — pure refactoring, no behavior change, but it's the phase that actually unlocks everything. Phase 2 implements that interface for OpenClaw's WebSocket API. Phase 3 adds `--backend openclaw` to the spawn command. Phase 4 deletes `pkg/opencode/` and the fork. Each phase has its own rollback point. The claude backend stays as stable ground throughout — it doesn't touch OpenCode, doesn't touch OpenClaw, and already handles all production spawns.

I also recommended converging completion detection on beads comments instead of adding a third transcript-access path. Today verification checks OpenCode transcripts, tmux pane captures, and beads comments depending on the backend. Beads is already the authority. The other two are supplementary checks that sometimes contradict each other. Adding a fourth path (OpenClaw transcripts) to an already-fragmented system felt wrong.

## Tension

Phase 1 (interface extraction) is the critical path, and it's 100% refactoring — no new capability, no visible improvement, just creating the seam that makes everything else possible. If that's hard to justify as a standalone piece of work, the alternative is coupling directly to OpenClaw's types and repeating exactly the mistake that makes OpenCode hard to remove now. The question is whether the discipline of "extract interface first" is worth the upfront cost when the codebase is a personal project with one user.
