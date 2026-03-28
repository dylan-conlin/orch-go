### Compose Over Monolith

Small, focused tools that combine. Each command does one thing well.

**Why:** Composable tools are testable, replaceable, understandable. Monoliths accumulate complexity and become fragile.

**Inspiration:** Unix philosophy, Git's porcelain/plumbing split.

**Examples:**

- `bd` (work tracking) + `kn` (knowledge) + `kb` (artifacts) + `orch` (coordination)
- Each tool focused, combined via workflows
- Skills own domain behavior, spawn owns orchestration infrastructure — skills containing beads/phase logic would reduce portability
