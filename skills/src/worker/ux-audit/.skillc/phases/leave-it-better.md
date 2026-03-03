# Leave it Better (Mandatory)

**Purpose:** Every session should leave the codebase, documentation, or knowledge base better than you found it.

**When you're in this phase:** Self-review has passed. Before marking complete, externalize what you learned.

---

## Why This Matters

UX audits surface patterns that recur across pages and projects. Externalizing what you learned prevents re-discovery and builds institutional knowledge about UX quality.

**Common examples of lost UX knowledge:**
- "This breakpoint causes layout issues because..." (constraint not documented)
- "axe-core flags this but it's a false positive because..." (gotcha not recorded)
- "The design system doesn't specify X, so pages handle it inconsistently" (gap not surfaced)

---

## What to Externalize

**Before marking complete, you MUST externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Design system gap | `kb quick constrain` | `kb quick constrain "No design token for table row height" --reason "Pages use 36-48px inconsistently"` |
| axe-core finding pattern | `kb quick tried` | `kb quick tried "axe-core color-contrast on SCS blue" --failed "4.2:1 ratio — fails AA for normal text"` |
| Breakpoint behavior | `kb quick constrain` | `kb quick constrain "Sidebar must collapse below 640px" --reason "Content unreadable at 640px with sidebar visible"` |
| Open design question | `kb quick question` | `kb quick question "Should tables scroll horizontally or stack vertically on mobile?"` |

---

## What to Prune

**If you encountered stale or outdated artifacts during your session, clean them up:**

| What You Found | Command |
|----------------|---------|
| Prior audit with fixed findings | `kb supersede <old-audit> --by <new-audit>` |
| Design decision no longer accurate | Note in completion comment |

**Quick check:** Did you compare with a prior audit that had findings now fixed?
- If yes → Consider superseding the old audit
- If findings partially fixed → Note in comparison section (don't supersede)

---

## Quick Checklist

- [ ] **Reflected on session:** What UX patterns did I discover that recur?
- [ ] **Externalized at least one item:** Ran `kb quick decide/tried/constrain/question`
- [ ] **Pruned stale artifacts:** Superseded outdated audits (or noted none found)

---

## If Nothing to Externalize

If the audit revealed nothing surprising:

1. Note: "Leave it Better: No new patterns discovered — findings were expected"
2. This should be rare — audits almost always reveal something

---

## Completion Criteria (Leave it Better)

- [ ] Reflected on what was learned during the session
- [ ] Ran at least one `kb quick` command OR documented why nothing to externalize
- [ ] Pruned stale artifacts OR noted "No stale artifacts encountered"
- [ ] Included "Leave it Better" status in completion comment

**Only proceed to final completion after Leave it Better is done.**
