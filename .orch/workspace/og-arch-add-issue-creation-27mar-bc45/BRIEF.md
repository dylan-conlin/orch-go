# Brief: orch-go-aqq5a

## Frame

Agents keep creating follow-up issues for work that was already committed by other agents. The daemon catches these duplicates at spawn-time — it refuses to launch agents for already-done work — but by then the issue is already sitting in beads as a zombie. Created, never spawned, never closed. The question was: why are we filtering downstream when we could prevent upstream?

## Resolution

The gap was comically simple once I found it: the architect skill template had four `bd create` call sites with zero dedup guidance. Every follow-up issue was created blind. I added a "Prior Art Check" — before every `bd create`, the agent now runs `git log --grep` and `bd list --status open` to see if the work exists. If it does, skip creation and log the hit instead.

This is the same layered-dedup pattern the daemon already uses (7 spawn gates), just extended one level upstream. The daemon's CommitDedupGate becomes the backstop, not the primary defense. The interesting design choice was making this instructional (skill text) rather than mechanical (code gate). Agents follow procedures in their loaded context reliably enough that a soft gate here, backed by a hard gate at spawn-time, gives two layers of defense with minimal ceremony.

Worker-base — where ALL skills get their `bd create` templates — has the same gap, but it's governance-protected. Created orch-go-95x1b for that.

## Tension

This is a soft gate. Agents *can* skip the Prior Art Check because it's behavioral guidance, not enforced code. The CommitDedupGate backstop means skipping the check doesn't create wasted spawns, but it does leave zombie issues in beads. The question is whether the observation of agent compliance (or non-compliance) over the next few weeks should trigger a harder enforcement mechanism — a pre-commit hook on `bd create`, or a beads-level dedup gate.
