# Leave it Better (Mandatory)

**Purpose:** Every session should leave the codebase, documentation, or knowledge base better than you found it.

**When you're in this phase:** Self-review has passed. Before marking complete, externalize what you learned.

---

## Why This Matters

Knowledge bases grow in two ways: **accretion** (adding new knowledge) and **entropy** (existing knowledge going stale). Both need attention. Every session should deposit something into the knowledge base AND clean up stale artifacts you encountered.

**Common examples of lost knowledge:**
- "We tried X but it didn't work because Y" (others will try X again)
- "This works but only if Z is configured this way" (constraint not documented)
- "We chose A over B because..." (decision not recorded)

**Common examples of stale knowledge:**
- Investigation superseded by a newer investigation or guide
- Decision that no longer reflects current architecture
- Constraint that was true but is no longer relevant

---

## What to Externalize

**Before marking complete, you MUST externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kb quick decide` | `kb quick decide "Use Redis for sessions" --reason "Need distributed state for horizontal scaling"` |
| Tried something that failed | `kb quick tried` | `kb quick tried "SQLite for sessions" --failed "Race conditions with multiple workers"` |
| Discovered a constraint | `kb quick constrain` | `kb quick constrain "All endpoints must be idempotent" --reason "Retry logic requires safe replay"` |
| Found an open question | `kb quick question` | `kb quick question "Should we rate-limit per-user or per-IP?"` |

---

## What to Prune

**If you encountered stale or outdated artifacts during your session, clean them up:**

| What You Found | Command | Example |
|----------------|---------|---------|
| Artifact replaced by your work or a newer artifact | `kb supersede <old> --by <new>` | `kb supersede .kb/investigations/2025-auth-v1.md --by .kb/investigations/2026-auth-v2.md` |
| Investigations already covered by a guide | `kb archive --synthesized-into <guide>` | `kb archive --synthesized-into spawn --dry-run` (preview first) |

**Quick check:** Did you read any `.kb/` artifacts during your session that were clearly wrong or outdated?
- If yes → supersede or archive them
- If no → Note "No stale artifacts encountered"

**This is NOT a full audit.** Only prune what you encountered naturally during your work.

---

## Quick Checklist

- [ ] **Reflected on session:** What did I learn that the next agent should know?
- [ ] **Externalized at least one item:** Ran `kb quick decide/tried/constrain/question`
- [ ] **Pruned stale artifacts:** Superseded or archived outdated `.kb/` files encountered during work (or noted none found)
- [ ] **Improved something:** Fixed a typo, clarified docs, added a missing comment, or updated stale info (optional but encouraged)

---

## If Nothing to Externalize

If the work was straightforward implementation with no new learnings:

1. Note in your completion comment: "Leave it Better: No new knowledge to externalize - straightforward implementation"
2. This is acceptable but should be rare

**Common case:** Even "straightforward" work often reveals something worth capturing (edge case, gotcha, or clarification).

---

## Examples

**Good externalization after feature work:**
```bash
kb quick decide "Use optimistic locking for updates" --reason "Prevents lost updates without blocking reads"
kb quick tried "Pessimistic locking" --failed "Caused deadlocks under high concurrency"
```

**Good externalization after debugging:**
```bash
kb quick constrain "Cache invalidation requires explicit call" --reason "TTL alone causes stale reads"
```

**Good externalization after investigation:**
```bash
kb quick question "Is the legacy API still used? Found no callers but unclear if external consumers exist"
```

**Good pruning after encountering stale artifacts:**
```bash
# Found an investigation that's now covered by the spawn guide
kb archive --synthesized-into spawn --dry-run
kb archive --synthesized-into spawn

# Found an old investigation replaced by a newer one
kb supersede .kb/investigations/2025-12-20-inv-old-topic.md --by .kb/investigations/2026-02-15-inv-new-topic.md
```

---

## Completion Criteria (Leave it Better)

- [ ] Reflected on what was learned during the session
- [ ] Ran at least one `kb quick` command OR documented why nothing to externalize
- [ ] Pruned stale artifacts OR noted "No stale artifacts encountered"
- [ ] Included "Leave it Better" status in completion comment

**Only proceed to final completion after Leave it Better is done.**
