# Brief: orch-go-s2sjc

## Frame

598 agents have logged Friction: comments over the past two weeks, but nobody reads them in aggregate. Each comment is fire-and-forget into a JSONL file. The same irritants — stale build files, governance hooks blocking valid edits, gitignored workspace paths — keep showing up because they're invisible at the system level.

## Resolution

`orch friction` reads `.beads/issues.jsonl` in a single pass and surfaces what was always there but never aggregated. The headline: 11.5% of agent sessions report friction, dominated by tooling (51%) and ceremony (22%). The top 5 recurring sources are concrete and fixable — build breaks from stale experiment files (9x), broad test failures from unrelated code (7x), governance hooks blocking valid actions (5x), missing `kb create --orphan` flag (5x), and gitignored workspace paths requiring force-adds (4x). Investigations and debugging skills hit friction at 16-17%, double the rate of feature implementations (8%). The weekly trend is just two bars right now but will become the first time-series view of system friction.

The clustering is keyword-based, not statistical — 69 entries is too small for anything fancier, and the patterns are obvious enough that string matching catches them cleanly. The command supports `--json` for scripting, `--days N` for windowing, and `--detail` for the full message list.

## Tension

The top sources are all known irritants that keep getting reported but haven't been fixed. The question isn't "what friction exists" — this command answers that. The question is: does making friction visible actually change the repair rate, or does it just add another dashboard nobody acts on? The first test is whether any of the top 5 sources get addressed in the next two weeks now that they're surfaced as a ranked list.
