# Brief: orch-go-zxe2j

## Frame

The orchestrator can only think about agent completions when Dylan starts a conversation. If three agents finish at 2am, their work sits in a queue until the next morning. The question was: does Claude Code have any mechanism that lets an external process — a daemon, a completing agent — poke a session and say "hey, there's new work"?

## Resolution

I expected to find nothing — or at most a research preview that wouldn't be usable yet. The turn: Claude Code already has two working injection mechanisms that nobody's documented.

The first is `--input-format stream-json`, an undocumented CLI flag that turns stdin into a structured message pipe. You start Claude with `-p --input-format stream-json`, and anything you write to stdin as NDJSON gets processed as a new user turn. I tested this by sending two messages through a single pipe — both processed, same session, full context continuity. A daemon could hold this pipe open and write messages whenever agents complete.

The second is session resume injection: `claude -p --resume <session-id> "new prompt"` loads the entire conversation history into a fresh process and adds your message as a new turn. I proved this works by creating a session with a secret word, killing the process, resuming in a new process, and asking for the word back. Full context recall. This means a daemon doesn't even need a running session — it can spin one up, inject a completion event, and let it process.

There's also Channels (an MCP-based push mechanism), but that's still in research preview. The stream-json and resume approaches work today, in v2.1.83.

## Tension

Both mechanisms work in `-p` (print/headless) mode, not in the interactive TUI that Dylan actually uses for orchestrator sessions. The gap between "this works programmatically" and "this can wake up Dylan's running orchestrator" still needs bridging — either by running the orchestrator in stream-json mode (losing TUI), or by the cruder `tmux send-keys` approach for interactive sessions (untested). The architectural decision about which mode to commit to is the real next step.
