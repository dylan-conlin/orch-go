# Brief: orch-go-ev88j

## Frame

The question was: if you want a GPT-5.4 orchestrator running alongside Claude, which frontend gives it enough context to actually orchestrate? The orchestrator's value is entirely in its context — the skill, the orientation, the governance hooks. A blind orchestrator, regardless of model quality, is just an expensive chatbot. Two candidates: Codex CLI (OpenAI's native tool) and OpenCode TUI (the fork we already maintain).

## Resolution

I expected this to be closer. It wasn't. Codex CLI loads AGENTS.md reliably and accepts piped context via stdin — both work. But it has no hooks at all. No SessionStart, no PreToolUse, nothing. That means no runtime governance: a Codex orchestrator can't be prevented from running `bd close` or `git push --force`. The orchestrator skill could be injected by expanding AGENTS.md or piping it as the initial prompt, but that's a one-shot injection with no dynamic re-injection on resume.

OpenCode, by contrast, is already 80% wired. GPT-5.4 is in its model snapshot. The backend routing in orch-go already sends OpenAI models through OpenCode. The plugin hook system (`experimental.chat.system.transform`) fires for every model and can inject the orchestrator skill at system prompt level — functional equivalent of Claude Code's SessionStart hook. The project context loading (CLAUDE.md, AGENTS.md) works identically regardless of which model is selected.

The surprise was how clean the separation already is in the codebase. Everything in `pkg/spawn/` — the context generation, config struct, backend interface, KB injection — is already frontend-agnostic. Only the hooks in `~/.claude/` are Claude-locked. The code was ready for multi-frontend before anyone asked for it.

## Tension

All of this is infrastructure. The question it doesn't answer: can GPT-5.4 actually follow a 37k-token orchestrator protocol without stalling? Previous GPT models (4o, 5.2-codex) hit 67-87% stall rates on protocol-heavy skills. If 5.4 repeats that pattern, the frontend choice is irrelevant — the model itself is the bottleneck. That empirical test is the real gate, and this investigation deliberately didn't run it.
