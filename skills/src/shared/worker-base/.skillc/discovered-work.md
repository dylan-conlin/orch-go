## Discovered Work (Mandatory)

**Before marking your session complete, review for discovered work.**

During any session, you may encounter:
- **Bugs** - Broken behavior not related to your current task
- **Tech debt** - Code that should be refactored but is out of scope
- **Enhancements** - Ideas for improvements noticed while working
- **Questions** - Strategic unknowns needing orchestrator input

### Checklist

Before completing your session:

- [ ] Reviewed for discovered work (bugs, tech debt, enhancements, questions)
- [ ] Created issues via `bd create` OR noted "No discovered work" in completion comment

**Friction vs. Discovered Work:**
- **Discovered work** = "I found a bug/debt in the code" → Create issue via `bd create`
- **Friction** = "The system made my work harder" → Report via `Friction:` comment (see Session Complete Protocol)
- Overlap is fine — report both if applicable

**Do NOT create empty investigation files as discovered work.** Empty investigation templates accumulate rapidly (~13/week) and create noise in the knowledge base. For discovered work, create beads issues — only create investigation files when you are actively investigating.

### Creating Issues

When creating follow-up issues, include 1-2 sentences of context with `-d` so the next agent knows what you saw and why it matters. Title-only issues create blind agents.

```bash
# For bugs found
bd create "description of bug" --type bug -l triage:review \
  -d "What you observed, where it showed up, and the concrete reason it needs follow-up."

# For tech debt or refactoring needs
bd create "description" --type task -l triage:review \
  -d "What is out of scope now, plus the context another agent would need to resume cleanly."

# For feature ideas or enhancements
bd create "description" --type feature -l triage:review \
  -d "What opportunity you noticed, what triggered it, and any constraint already visible."

# For strategic questions needing decision
bd create "description" --type question -l triage:review \
  -d "What decision is unclear, what evidence or tension surfaced it, and why it blocks clean progress."
```

### Reporting

In your `Phase: Complete` comment, include either:
- List of issues created: `Created: orch-go-XXXXX, orch-go-YYYYY`
- Or: `No discovered work`

**Why this matters:** Discovered work that isn't tracked gets lost. The next session has no visibility into bugs or opportunities you found. Creating issues ensures nothing falls through the cracks.

### Migration Status (Design/Architect Sessions Only)

**If your session produced a design or architectural recommendation that requires implementation:**

Include a `MIGRATION_STATUS` block in your Phase: Complete comment:

```
MIGRATION_STATUS:
  designed: [what was designed]
  implemented: [what code was written, or "none"]
  deployed: [what config/hooks/skills were wired, or "none"]
  remaining: [what's still incomplete, or "none"]
```

If `remaining` is not `"none"`, create a follow-up issue:
```bash
bd create "[remaining work description]" --type task -l triage:ready
```

**Why:** Configuration drift's #1 root cause is incomplete migrations — designs that are partially implemented but never fully wired. This gate ensures the gap between "designed" and "deployed" is tracked as an issue, not forgotten.

**Skip this if:** Your session was pure implementation (no design recommendations) or pure investigation (no implementation recommendations).

---

### Cross-Repo Issue Handoff

**When you discover an issue that belongs to a different repo**, you cannot create it directly — `bd create` only works in the current project directory, and shell sandboxing prevents `cd` to other repos.

**Instead, output a structured `CROSS_REPO_ISSUE` block** in your beads completion comment or SYNTHESIS.md. The orchestrator will pick this up during completion review and create the issue in the target repo.

**Format:**
```
CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/<target-repo>
  title: "<concise issue title>"
  type: bug|task|feature|question
  priority: 0-4
  description: "<1-3 sentences with context, evidence, and why it matters>"
```

**Rules:**
- Use absolute or `~`-relative paths for `repo`
- Include enough context in `description` for the issue to stand alone (the orchestrator in the other repo won't have your session context)
- One block per issue — multiple issues get multiple blocks
- Report blocks in your `Phase: Complete` comment: `Cross-repo: 1 CROSS_REPO_ISSUE block for price-watch`

**Example:**
```bash
bd comments add <beads-id> "Phase: Complete - Implemented token refresh. Cross-repo: 1 CROSS_REPO_ISSUE block below.

CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/price-watch
  title: Fix ScsOauthClient concurrent token refresh
  type: bug
  priority: 2
  description: During orch-go token handling work, discovered price-watch ScsOauthClient has a race condition when multiple goroutines call RefreshToken simultaneously. No mutex protects the shared token state."
```

---
