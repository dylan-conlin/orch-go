# Brief: orch-go-y012i

## Frame

The dirty-worktree problem keeps coming back, and every time it does, the response is the same: add another regex hook to block `git add -A`, then watch agents find a way around it. Seven thousand lines of "deleted" workspace artifacts drown out the two actual source files that matter. Meanwhile, a closed issue left uncommitted code that breaks the build, and nobody noticed because the build gate was checking the wrong thing.

## Resolution

I expected to design a 3-layer enforcement stack — scope manifests at spawn, ownership checks at commit, reconciliation at close. What I actually found is that two of those three layers repeat the exact mistake that made accretion gates fail: they block agents at points where agents control the bypass. The accretion-gate data was the turn — 100% bypass rate over two weeks. That killed the idea of spawn-time or commit-time blocking.

What survived is a single enforcement layer at the one point agents can't bypass: issue closure. When `orch complete` runs, it checks whether tracked dirty files from this agent's work are committed, owned by another issue, or classified as allowed residue. The invariant isn't "clean tree" — it's "owned tree." Dirty is fine as long as someone's responsible for it.

The surprising part was how much of the problem was the harness fighting itself. Feature-impl skill text says `git add -A` four times. Worker-base says NEVER. The hook blocks the command that the skill told the agent to use. That's textbook Class 5 — two authority sources disagreeing — and the fix isn't a better regex. The fix is making worker-base canonical and deleting the contradictions.

## Tension

The design assumes file-to-issue ownership can be determined at close time from SYNTHESIS.md Delta claims and git baseline diffs. But this only works if agents actually declare their files in SYNTHESIS — and the current Delta section is parsed with four regex patterns that handle backticks, bold, bullets, and parens. If agents use a fifth format, files slip through. The question is whether the existing Delta parser is robust enough to be a Gate 15 input, or whether ownership needs a more structured declaration mechanism.
