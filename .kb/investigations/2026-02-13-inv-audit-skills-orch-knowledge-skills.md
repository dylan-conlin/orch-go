<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** 6 orch commands referenced in skills don't exist at Jan 18 baseline (0bca3dec): `orch frontier`, `orch friction`, `orch health`, `orch stability`, `orch reap`, and `orch lint`.

**Evidence:** Cross-referenced `git ls-tree 0bca3dec cmd/orch/` against grep of all skill files - these 6 commands have no corresponding source files at baseline.

**Knowledge:** The diagnostic and meta-orchestrator skills are heavily affected; pre-commit script in orch-knowledge references non-existent `orch lint --skills`.

**Next:** Remove/update stale references in affected skills: diagnostic (5 commands), meta-orchestrator (1 command), orchestrator (1 command), and fix pre-commit script.

**Authority:** architectural - Affects multiple skill files across different skill categories, requires cross-boundary coordination.

---

# Investigation: Audit Skills Against Jan 18 Baseline

**Question:** Which skills in ~/orch-knowledge/skills/src/ contain references to orch features that don't exist in the Jan 18 baseline (0bca3dec)?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker-agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Six orch commands referenced in skills don't exist at baseline

**Evidence:** Grep of all skill files found references to these commands, but `git ls-tree 0bca3dec cmd/orch/` shows no corresponding source files:

| Missing Command | Exists at Baseline? | Files Referencing |
|----------------|---------------------|-------------------|
| `orch frontier` | NO | meta-orchestrator, orchestrator |
| `orch friction` | NO | diagnostic |
| `orch health` | NO | diagnostic |
| `orch stability` | NO | diagnostic |
| `orch reap` | NO | diagnostic |
| `orch lint` | NO | pre-commit script (not a skill, but related) |

**Source:** 
- `git ls-tree 0bca3dec cmd/orch/` - lists all files at baseline
- `grep -rn "orch frontier\|orch friction\|orch health\|orch stability\|orch reap" ~/orch-knowledge/skills/src/`

**Significance:** Agents following these skills will fail when trying to run these commands.

---

### Finding 2: Diagnostic skill is most affected (5 missing commands)

**Evidence:** The diagnostic skill (meta/diagnostic/SKILL.md) references:
- `orch health` (lines 67, 83, 231, 268, 296, 322)
- `orch stability` (lines 66, 160, 297)
- `orch reap` (lines 157, 216, 300)
- `orch friction` (lines 175, 308)
- `orch reconcile --fix` flag (need to verify this flag exists)

**Source:** `~/orch-knowledge/skills/src/meta/diagnostic/SKILL.md`

**Significance:** The entire diagnostic skill workflow depends on commands that don't exist, making it non-functional at the Jan 18 baseline.

---

### Finding 3: Pre-commit script references non-existent `orch lint --skills`

**Evidence:** The script at `~/orch-knowledge/scripts/pre-commit` contains:
```bash
lint_output=$(orch lint --skills 2>&1)
```

No `lint.go` or `lint_cmd.go` exists at baseline (or in current codebase):
```
$ ls cmd/orch/lint*.go
zsh: no matches found: cmd/orch/lint*.go
```

**Source:** `~/orch-knowledge/scripts/pre-commit`

**Significance:** The pre-commit hook will fail silently (exits 0 with warning) or cause commit failures when skill files are changed.

---

### Finding 4: Commands that DO exist at baseline

**Evidence:** The following commands exist at baseline and are safe to reference:
- `orch spawn`, `orch complete`, `orch status`, `orch review`
- `orch doctor`, `orch reconcile`, `orch stats`, `orch patterns`
- `orch daemon` (with subcommands: `run`, `once`, `preview`, `reflect`)
- `orch abandon`, `orch clean`, `orch wait`, `orch focus`, `orch drift`
- `orch serve`, `orch servers`, `orch send`, `orch resume`

**Source:** `git ls-tree 0bca3dec cmd/orch/` and `git show 0bca3dec:cmd/orch/main.go`

**Significance:** These commands can safely be referenced in skills.

---

### Finding 5: SQLite/state.db references are examples only

**Evidence:** Grep for `sqlite\|state\.db` found only documentation examples, not actual system references:
- `kb quick tried "SQLite for sessions" --failed "Race conditions"` (example in multiple skills)

**Source:** `grep -r "sqlite\|state\.db" ~/orch-knowledge/skills/src/`

**Significance:** No action needed - these are illustrative examples, not actual references to the system.

---

### Finding 6: No `orch phase` references remain

**Evidence:** `grep -r "orch phase" ~/orch-knowledge/skills/src/` returns no matches.

**Source:** grep output (empty)

**Significance:** Previous fix was successful - no remaining `orch phase` references.

---

## Synthesis

**Key Insights:**

1. **Diagnostic skill is broken** - It relies heavily on health/stability/reap/friction commands that don't exist, making the entire skill non-functional for the Jan 18 baseline.

2. **Meta-orchestrator and orchestrator skills use `orch frontier`** - This command doesn't exist at baseline but is referenced for cross-project backlog management.

3. **Pre-commit protection doesn't work** - The `orch lint --skills` command was designed to catch exactly these kinds of stale references, but the command itself doesn't exist.

**Answer to Investigation Question:**

Three skills contain references to orch commands that don't exist at the Jan 18 baseline:

| Skill | Missing Commands | Severity |
|-------|------------------|----------|
| meta/diagnostic | `orch health`, `orch stability`, `orch reap`, `orch friction` | HIGH - skill non-functional |
| meta/meta-orchestrator | `orch frontier` | MEDIUM - affects backlog visibility |
| meta/orchestrator | `orch frontier` | MEDIUM - affects backlog visibility |

Additionally, the pre-commit script references `orch lint --skills` which doesn't exist.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch frontier` doesn't exist at baseline (verified: `git ls-tree 0bca3dec cmd/orch/` has no frontier.go)
- ✅ `orch health`, `orch stability`, `orch reap`, `orch friction` don't exist (verified: same method)
- ✅ `orch lint` doesn't exist (verified: `ls cmd/orch/lint*.go` - no matches)
- ✅ `orch phase` references removed (verified: grep returns empty)
- ✅ SQLite references are examples only (verified: grep context shows `kb quick tried` examples)

**What's untested:**

- `orch reconcile --fix` flag existence at baseline (not verified)
- Whether the missing commands exist as subcommands of other commands (checked daemon, but not exhaustively)

**What would change this:**

- If any of these commands are implemented as subcommands (e.g., `orch daemon health`), the finding would need updating
- If the baseline commit hash is wrong, all findings would need re-verification

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Remove/stub stale command references in skills | architectural | Affects multiple skill files across categories |
| Fix pre-commit script | implementation | Single file, clear fix |
| Consider alternative approaches in skills | architectural | Requires rethinking diagnostic workflow |

### Recommended Approach: Stub Commands with "Not Available"

**Stub Commands with Clear Error Messages** - Update skills to note commands are unavailable at baseline, or provide alternative approaches.

**Why this approach:**
- Doesn't require implementing the commands
- Makes skills honest about current limitations
- Allows gradual implementation as commands are added

**Trade-offs accepted:**
- Skills lose some functionality
- Agents may need manual intervention where commands were used

**Implementation sequence:**
1. Fix pre-commit script first (remove/stub `orch lint --skills` check)
2. Update diagnostic skill to note unavailable commands and provide alternatives
3. Update meta-orchestrator/orchestrator to replace `orch frontier` with `bd ready` or similar

### Alternative Approaches Considered

**Option B: Implement missing commands**
- **Pros:** Skills work as documented
- **Cons:** Significant development effort, may not be needed
- **When to use instead:** If commands provide critical value

**Option C: Remove diagnostic skill entirely**
- **Pros:** No stale references
- **Cons:** Loses potentially useful skill when commands are implemented
- **When to use instead:** If skill is fundamentally incompatible with baseline

---

### Implementation Details

**What to implement first:**
- Fix pre-commit script (highest friction - blocks commits)
- Document missing commands in affected skill files

**Things to watch out for:**
- The `.skillc/` directories contain templates that also need updating
- Both SKILL.md and SKILL.md.template may need changes

**Affected files requiring changes:**

| File | Changes Needed |
|------|----------------|
| `~/orch-knowledge/scripts/pre-commit` | Remove/stub `orch lint --skills` |
| `meta/diagnostic/SKILL.md` | Note `orch health/stability/reap/friction` unavailable |
| `meta/meta-orchestrator/SKILL.md` | Replace `orch frontier` references |
| `meta/meta-orchestrator/.skillc/SKILL.md` | Same |
| `meta/meta-orchestrator/.skillc/SKILL.md.template` | Same |
| `meta/orchestrator/SKILL.md` | Replace `orch frontier` references |
| `meta/orchestrator/.skillc/SKILL.md` | Same |
| `meta/orchestrator/.skillc/SKILL.md.template` | Same |

**Success criteria:**
- ✅ `grep -r "orch frontier\|orch friction\|orch health\|orch stability\|orch reap\|orch lint" ~/orch-knowledge/skills/src/` returns no actionable references
- ✅ Pre-commit hook passes when skill files change

---

## References

**Files Examined:**
- `cmd/orch/*.go` via `git ls-tree 0bca3dec` - Jan 18 baseline commands
- `~/orch-knowledge/skills/src/**/*.md` - All skill markdown files
- `~/orch-knowledge/scripts/pre-commit` - Pre-commit hook

**Commands Run:**
```bash
# List all baseline command files
git ls-tree 0bca3dec cmd/orch/

# Search for stale command references
grep -rn "orch frontier\|orch friction\|orch health\|orch stability\|orch reap" ~/orch-knowledge/skills/src/

# Verify no orch phase references
grep -r "orch phase" ~/orch-knowledge/skills/src/

# Check pre-commit script
cat ~/orch-knowledge/scripts/pre-commit
```

---

## Investigation History

**2026-02-13 [start]:** Investigation started
- Initial question: Which skills reference orch features that don't exist at Jan 18 baseline?
- Context: Codebase reverted to Jan 18, skills may reference non-existent commands

**2026-02-13 [findings]:** Identified 6 missing commands
- `orch frontier`, `orch friction`, `orch health`, `orch stability`, `orch reap`, `orch lint`
- Diagnostic skill most affected (5 commands)
- Pre-commit script broken

**2026-02-13 [complete]:** Investigation completed
- Status: Complete
- Key outcome: 3 skills and 1 script need updates to remove references to 6 non-existent commands
