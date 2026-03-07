## Prior Work (Template Independence)

**Applies to Investigation Mode only.** In Probe Mode, use the probe template and skip the Prior-Work table.

701 existing investigations lack the Prior-Work table. The skill handles both old and new investigations gracefully.

### Rules

| Situation                       | Action                                                    |
| ------------------------------- | --------------------------------------------------------- |
| New investigation (no priors)   | Add Prior-Work table with "N/A - novel investigation"     |
| New investigation (has priors)  | Add Prior-Work table, cite prior investigations           |
| Extending old investigation     | Create NEW file with Prior-Work table referencing the old |
| Old investigation without table | Leave it alone, don't modify                              |

Verification happens naturally: verify cited claims against primary sources (code, test output) as you encounter them during investigation — not all upfront.

**Reference:** See `~/.claude/skills/worker/investigation/reference/prior-work-examples.md` for relationship type guidance.
