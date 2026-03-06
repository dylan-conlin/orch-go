# Leave it Better Guide

**When to use:** Consult when externalizing knowledge before completing investigation.

## kb quick Commands

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kb quick decide` | `kb quick decide "Use Redis for sessions" --reason "Need distributed state"` |
| Tried something that failed | `kb quick tried` | `kb quick tried "SQLite for sessions" --failed "Race conditions"` |
| Discovered a constraint | `kb quick constrain` | `kb quick constrain "API requires idempotency" --reason "Retry logic"` |
| Found an open question | `kb quick question` | `kb quick question "Should we rate-limit per-user or per-IP?"` |

## Pruning Commands

| What You Found | Command | Example |
|----------------|---------|---------|
| Artifact replaced by newer work | `kb supersede <old> --by <new>` | `kb supersede .kb/investigations/old.md --by .kb/investigations/new.md` |
| Investigations covered by a guide | `kb archive --synthesized-into <guide>` | `kb archive --synthesized-into spawn --dry-run` |

## When to Use Each Command

**`kb quick decide`** - You made a choice between options and want to preserve the reasoning.
- Example: "Chose Go over Python for CLI because single binary distribution"

**`kb quick tried`** - You tried an approach and it didn't work; save others from repeating.
- Example: "Tried websockets for status updates; too complex for polling use case"

**`kb quick constrain`** - You discovered a rule that must be followed.
- Example: "API rate limited to 100 req/min; must cache or batch requests"

**`kb quick question`** - You found a question that needs future resolution.
- Example: "What happens when user deletes account with active subscriptions?"

**`kb supersede`** - You found an artifact that's been replaced by newer work.
- Example: "Old spawn investigation now covered by the spawn guide"

**`kb archive`** - Multiple investigations on a topic have been synthesized into a guide.
- Example: "3 dashboard investigations now covered by dashboard.md guide"

## Reflection Prompt

Before completing, ask yourself:
- What did I learn that the next agent should know?
- Did I try something that failed?
- Did I make a decision with reasoning that should be preserved?
- Is there a constraint others should respect?
- Did I read any `.kb/` artifacts that were clearly outdated or wrong?

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."

**If no stale artifacts found:** Note "No stale artifacts encountered" in completion comment.
