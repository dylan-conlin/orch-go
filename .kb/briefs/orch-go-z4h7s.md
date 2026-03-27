# Brief: orch-go-z4h7s

## Frame

The README still introduced orch-go as "Go CLI for OpenCode orchestration." Every section was spawn commands, SSE events, session IDs. A new reader's first impression of the project was that it was an OpenCode wrapper — the thread/comprehension layer that the product boundary decision just named as the center was invisible.

## Resolution

The rewrite opens with what the system is actually for: making agent output compound into understanding. Four sections cover the core concepts — threads as the organizing spine, synthesis/briefs for async comprehension, knowledge composition via claims and models, and verification as the trust layer. Each section includes real CLI commands so it stays concrete. Execution (spawn, daemon, backends) moves to a clearly labeled "Substrate" section. The architecture diagram from the recently-updated overview reinforces the layering visually: core on top, substrate below, external services at the bottom.

The surprising thing was how little the old README overlapped with the new one. Almost nothing survived from the original — not because the execution content was wrong, but because the project's identity had moved so far that the old framing was describing a different product. The rewrite isn't an edit; it's a replacement.

## Tension

Two other files still carry the old identity: CLAUDE.md opens with "Go rewrite of orch-cli - AI agent orchestration via OpenCode API" and the CLI reference guide leads with "kubectl for AI agents." The README now says something different than these companions. Whether to update them in this pass or let them evolve separately is a judgment call — updating creates consistency, but the CLI guide's "kubectl" framing might still be the right *operational* metaphor even if it's the wrong *product* metaphor.
