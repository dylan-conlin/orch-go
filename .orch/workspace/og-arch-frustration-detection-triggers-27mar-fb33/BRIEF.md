# Brief: orch-go-u5x4g

## Frame

There was a session where the orchestrator caught itself spiraling — said "You're right. Let me stop" — and then kept doing the exact same thing. Dylan did what the system couldn't: he exported the session, closed it, and started a new one with just the question. That manual action is the design pattern. The system should do it automatically.

## Resolution

The reason "let me stop" doesn't work is the same reason the model already documents for frame collapse: framing is stronger than instructions. Once a conversation has been going in circles for 20 minutes, the attention patterns are baked into the conversation history. A prompt that says "stop" is processed WITHIN the degraded frame, not from outside it. The only genuine reset is a new session — a new frame.

The design splits into two tracks because interactive and headless sessions have fundamentally different control planes. For interactive sessions (Dylan typing in Claude Code), a UserPromptSubmit hook watches for frustration signals in what Dylan types — "this isn't working", "we keep going in circles", repeated corrections — and proposes a boundary. Not enforces — proposes. "This conversation may be fighting you. Want to save the question and start fresh?" If yes, the system writes a small artifact (the question + what was tried + why it failed) and the new session picks it up on start, getting a fresh cognitive frame with just the question.

For headless workers, the coaching plugin already tracks behavioral signals (thrashing, stuck-in-phase, contradicting prior work). The trick is that any single signal has a ~72% noise rate — that's why action_ratio and analysis_paralysis were removed. But when TWO signals fire together, that's a real signal. A compound frustration detector triggers when 2+ existing signals co-occur, reports `Phase: Boundary` via beads, and the daemon respawns with only the original question plus a "do not repeat" list.

## Tension

The design assumes frustration is always a signal to stop. But the resistance thread distinguishes "bolted-on" resistance (seams, each fix reveals more fixes, backlog grows) from "just-hard" resistance (interior complexity, each fix makes remaining problems fewer). A productive debugging session can FEEL frustrating without being the wrong conversation. Can the detection distinguish? And should it try, or is the cost of a false positive boundary (one respawn) low enough that it doesn't matter?
