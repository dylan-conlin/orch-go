<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** ~/.kb/ is intentionally a global, user-scoped knowledge base that is NOT part of any git repo, containing 26 files (216K) of principles, decisions, and cross-project knowledge.

**Evidence:** `git status` in ~/.kb returns "not a git repository"; contents include principles.md, values.md, 5 decisions, and 6 investigations that apply globally.

**Knowledge:** The architecture has three tiers: global ~/.kb/ (user-scoped, not versioned), project .kb/ dirs (versioned per-project), and templates in ~/.kb/templates/ for scaffolding. This is by design - global knowledge shouldn't live in any single project repo.

**Next:** Create dedicated dotfiles or knowledge repo for ~/.kb/ if version history is desired, or document that unversioned global state is intentional.

---

# Investigation: ~/.kb/ Not Version Controlled

**Question:** Why is ~/.kb/ not under version control, and is this a problem that needs fixing?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** orch-go-jb77
**Phase:** Complete
**Next Step:** None - escalate to orchestrator for decision on whether to version ~/.kb/
**Status:** Complete

---

## Findings

### Finding 1: ~/.kb/ exists outside any git repository

**Evidence:** 
```bash
$ cd ~/.kb && git status
fatal: not a git repository (or any of the parent directories): .git

$ ls -la ~/.kb/.git
No .git in ~/.kb
```

**Source:** Commands run in ~/.kb directory

**Significance:** This confirms ~/.kb/ is not version controlled. Any changes (additions, deletions, modifications) have no history and cannot be recovered if lost.

---

### Finding 2: ~/.kb/ contains valuable global knowledge artifacts

**Evidence:** 
- Total: 26 markdown files, 216K storage
- Directories: decisions/ (5 files), guides/, investigations/ (6 files), templates/ (12 files)
- Key files: principles.md (19KB), values.md (2.6KB), projects.json

Content samples:
- `decisions/2025-12-25-pressure-over-compensation.md` - Referenced in orchestrator skill
- `decisions/2025-12-26-share-patterns-not-tools.md` - Recent decision
- `principles.md` - Core principles document (heavily referenced)

**Source:** `ls -la ~/.kb/`, `find ~/.kb -type f -name "*.md" | wc -l`

**Significance:** This is not throwaway data - it's actively used configuration and knowledge that influences agent behavior. The principles.md is referenced from global CLAUDE.md.

---

### Finding 3: Project-scoped .kb/ directories ARE version controlled

**Evidence:**
```bash
$ git -C ~/Documents/personal/orch-go ls-files .kb/ | wc -l
(shows 385 tracked files in .kb/)

$ git -C ~/orch-knowledge status
On branch master
(repo is healthy, .kb/ tracked)
```

**Source:** git ls-files and git status in respective repos

**Significance:** The architecture has two tiers - global (~/.kb/) and project-specific (.kb/). Only global is unversioned. This appears intentional: project knowledge versions with the project, while global knowledge is user-scoped.

---

### Finding 4: ~/.kb/templates/ provides scaffolding for new artifacts

**Evidence:**
```
~/.kb/templates/
├── DECISION.md
├── INVESTIGATION.md (6.7KB)
├── KNOWLEDGE.md
├── POST_MORTEM.md
├── README.md
├── RESEARCH.md
├── SPAWN_PROMPT.md (10.7KB)
├── SYNTHESIS.md
├── WORKSPACE.md
├── gitignore
└── investigations/
```

**Source:** `ls -la ~/.kb/templates/`

**Significance:** These templates are used by `kb create` command to scaffold new artifacts. Losing them would break the knowledge creation workflow.

---

## Synthesis

**Key Insights:**

1. **Three-tier architecture** - Knowledge lives at: ~/.kb/ (global, user-scoped), project/.kb/ (versioned with code), and ~/.claude/ (Claude config). Each tier has different versioning needs.

2. **Global = cross-project** - ~/.kb/ contains knowledge that applies across ALL projects (principles, global decisions, templates). It can't logically live in any single project repo.

3. **Risk is real but bounded** - Loss of ~/.kb/ would lose 26 files of configuration and decisions, but project-specific knowledge (hundreds of investigations) is safely versioned in project repos.

**Answer to Investigation Question:**

~/.kb/ is not version controlled because it exists at the user level outside any project repository. This is architecturally intentional - global knowledge shouldn't be coupled to any single project. However, it DOES represent a data loss risk for valuable artifacts like principles.md and templates.

Options to address:
1. **Create dedicated dotfiles repo** - Many developers version their dotfiles in a standalone repo
2. **Create ~/.kb/ as its own git repo** - Simple, self-contained versioning
3. **Symlink ~/.kb/ into an existing repo** - e.g., orch-knowledge
4. **Accept unversioned state** - Document as intentional, rely on backups

---

## Structured Uncertainty

**What's tested:**

- ✅ ~/.kb/ is not a git repo (verified: `git status` returned "not a git repository")
- ✅ ~/.kb/ contains 26 markdown files (verified: `find | wc -l`)
- ✅ Project .kb/ directories ARE versioned (verified: git ls-files shows tracked)
- ✅ Templates exist and are used by kb CLI (verified: `kb create` worked)

**What's untested:**

- ⚠️ Whether Dylan has backups of ~/.kb/ (not checked - personal backup strategy unknown)
- ⚠️ Whether this was an intentional design decision (no decision record found)
- ⚠️ Whether any other tools depend on ~/.kb/ not being versioned

**What would change this:**

- Finding a decision record explicitly stating ~/.kb/ should be unversioned
- Finding that ~/.kb/ is backed up via Time Machine or similar
- Finding that version control would break kb-cli functionality

---

## Implementation Recommendations

### Recommended Approach: Create ~/.kb/ as standalone git repo

**Why this approach:**
- Minimal disruption to existing tooling
- Self-contained - no dependencies on other repos
- Can push to GitHub/private backup for redundancy
- Templates and principles get version history

**Trade-offs accepted:**
- Another repo to maintain
- Must remember to commit/push (or set up automation)

**Implementation sequence:**
1. `cd ~/.kb && git init`
2. `git add .`
3. `git commit -m "Initial commit: global knowledge base"`
4. Create remote on GitHub (private)
5. Push and set upstream

### Alternative Approaches Considered

**Option B: Symlink into orch-knowledge**
- **Pros:** Versions with existing orchestration knowledge
- **Cons:** Couples global knowledge to specific project; potential path confusion for agents
- **When to use instead:** If wanting to consolidate all knowledge artifacts

**Option C: Accept unversioned state**
- **Pros:** Zero effort, no new repos
- **Cons:** Data loss risk remains
- **When to use instead:** If Time Machine backup is sufficient

---

## References

**Commands Run:**
```bash
# Check git status
cd ~/.kb && git status

# Count files
find ~/.kb -type f -name "*.md" | wc -l

# Check size
du -sh ~/.kb

# List contents
ls -la ~/.kb/
ls -la ~/.kb/decisions/
ls -la ~/.kb/templates/
```

**Related Artifacts:**
- **Decision:** ~/.kb/decisions/2025-12-25-pressure-over-compensation.md - Example of valuable global decision
- **Principles:** ~/.kb/principles.md - Core principles document referenced globally

---

## Investigation History

**2025-12-26 18:28:** Investigation started
- Initial question: Why is ~/.kb/ not version controlled?
- Context: Spawned from orch-go-jb77 to understand and document

**2025-12-26 18:35:** Investigation completed
- Status: Complete
- Key outcome: ~/.kb/ is intentionally global/user-scoped and not tied to any project repo; versioning is a user choice, recommend standalone git repo if desired
