<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch build skills` and `skillc` are complementary tools for different purposes - orch-cli builds templated Claude Code skills from orch-knowledge sources to ~/.claude/skills/, while skillc compiles project-local .skillc/ directories to CLAUDE.md files.

**Evidence:** Examined both codebases; orch-cli skills_cli.py handles `SKILL.md.template` + phases expansion and dual-target deployment (Claude Code + OpenCode); skillc compiler.go handles .skillc/ directory with skill.yaml manifests and dependency graph resolution to CLAUDE.md.

**Knowledge:** These tools solve different problems: orch-cli builds procedural skills for the Skill tool system, while skillc builds project-specific context artifacts. They can coexist - no replacement needed.

**Next:** Consider whether orch-go should port `orch build skills` functionality (if Python orch-cli is being deprecated) OR leave it in orch-cli. Skillc remains independent for project context compilation.

**Confidence:** High (90%) - Both systems examined with code reading; working deployed artifacts verified at ~/.claude/skills/ and ~/.config/opencode/agent/.

---

# Investigation: Skillc vs Orch-Knowledge Skill Build Pipeline

**Question:** What does `orch build skills` actually do? How does it differ from skillc's approach? Should skillc replace it, or should orch-go port it? What's the migration path if any?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-inv-investigate-skillc-vs-22dec
**Phase:** Complete
**Next Step:** None (investigation complete)
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: `orch build skills` builds Claude Code skills from templated sources

**Evidence:** The Python orch-cli at `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/cli.py:2069-2268` and `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skills_cli.py:102-271` implements skill building that:

1. Scans `orch-knowledge/skills/src/{category}/{skill}/src/SKILL.md.template` files
2. Loads phase templates from `src/phases/*.md` directories
3. Expands `<!-- SKILL-TEMPLATE: phase-name -->` markers with phase content
4. Adds AUTO-GENERATED headers warning not to edit directly
5. Deploys to TWO targets:
   - Claude Code: `~/.claude/skills/{category}/{skill}/SKILL.md`
   - OpenCode: `~/.config/opencode/agent/{skill}.md` (flat structure, transformed frontmatter)

**Source:** `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skills_cli.py:102-271`, `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skill_build.py:1-118`

**Significance:** This is a BUILD + DEPLOY pipeline for procedural skills (feature-impl, investigation, systematic-debugging, etc.) that live in orch-knowledge and get installed system-wide for Claude Code's Skill tool to discover.

---

### Finding 2: Skillc compiles project-local .skillc/ to CLAUDE.md with dependency resolution

**Evidence:** The Go skillc at `/Users/dylanconlin/Documents/personal/skillc/` implements:

1. Reads `.skillc/` directories containing `skill.yaml` manifests
2. Parses manifests with name, version, dependencies[], sources[] fields
3. Builds dependency graph and resolves in topological order
4. Concatenates sources in dependency order
5. Generates CLAUDE.md with self-describing headers ("Source: .skillc/", "Build command: skillc build")
6. Supports `--global` for `~/.claude/.skillc/` → `~/.claude/CLAUDE.md`
7. Supports `--recursive` to build all .skillc/ dirs in tree

**Source:** `/Users/dylanconlin/Documents/personal/skillc/cmd/skillc/main.go:1-323`, `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go:1-240`

**Significance:** This is a CONTEXT COMPILER for project-specific guidance. It solves the problem of large CLAUDE.md files becoming monolithic and hard to maintain by allowing modular source files with dependency ordering.

---

### Finding 3: The two systems serve different purposes and artifact types

**Evidence:** Examining the artifact structure:

| System | Input | Output | Purpose |
|--------|-------|--------|---------|
| `orch build skills` | `orch-knowledge/skills/src/{category}/{skill}/` with SKILL.md.template + phases/ | `~/.claude/skills/{category}/{skill}/SKILL.md` + `~/.config/opencode/agent/{skill}.md` | Procedural skills for Claude's Skill tool |
| `skillc build` | Any project's `.skillc/` with skill.yaml + *.md sources | CLAUDE.md (or custom via manifest output field) | Project-specific context |

Key differences:
- **Template expansion vs concatenation**: orch-cli replaces SKILL-TEMPLATE markers; skillc concatenates files in dependency order
- **System-wide vs project-local**: orch-cli deploys to ~/.claude/skills/; skillc produces project-local artifacts
- **Frontmatter transformation**: orch-cli transforms skill-type→mode for OpenCode; skillc preserves content as-is
- **Dependency resolution**: skillc has full DAG resolution; orch-cli has phase ordering only

**Source:** Verified deployed artifacts at `~/.claude/skills/` (32 items, symlinks + category dirs) and `~/.config/opencode/agent/` (17 .md files)

**Significance:** These are not competing solutions but complementary tools for different artifact types. Neither should replace the other.

---

### Finding 4: orch-go already has skill loading but not building

**Evidence:** The orch-go pkg/skills/loader.go at `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go:1-126` implements:

1. `Loader` that finds skills in `~/.claude/skills/`
2. `FindSkillPath()` to locate SKILL.md files
3. `LoadSkillContent()` to read skill content
4. `ParseSkillMetadata()` to extract YAML frontmatter

This is a CONSUMER of skills, not a BUILDER. It reads the deployed skills that `orch build skills` produces.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go:1-126`

**Significance:** If orch-go needs to fully replace Python orch-cli, it would need to port the skill building functionality (template expansion + dual-target deployment). Currently it only loads pre-built skills.

---

### Finding 5: Decision documents clarify skillc's intentional scope

**Evidence:** The skillc decision document at `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md` explicitly defines:

**In Scope:**
- Project context (CLAUDE.md) - project-specific guidance
- Skill definitions (SKILL.md) - procedure/skill guidance
- Hook context (context.md) - markdown injected by AI hooks

**Out of Scope:**
- Git hooks (shell scripts, not AI-consumed)
- Hook scripts (.sh) (shell logic, not markdown)
- Config files (machine-readable, no compilation needed)

**Source:** `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md`

**Significance:** Skillc was intentionally designed to compile "markdown that AI agents read for guidance." The orch-knowledge skill templates are a different artifact type (system-wide procedural skills vs project context).

---

## Synthesis

**Key Insights:**

1. **Different artifact types, different tools** - `orch build skills` produces procedural skills for Claude's Skill tool system (invoked with `@skill-name`), while skillc produces project-specific context files (read automatically from CLAUDE.md). These are orthogonal concerns.

2. **Different deployment targets** - orch-cli skills go to `~/.claude/skills/` (system-wide, category-organized) and `~/.config/opencode/agent/` (flat, transformed). Skillc outputs go to the project's own directory (CLAUDE.md or custom path from manifest).

3. **Different complexity handling** - orch-cli handles template expansion (SKILL-TEMPLATE markers replaced with phase content). Skillc handles dependency graphs (topological sort of .skillc/ modules). Neither has the other's complexity.

4. **orch-go is a consumer, not builder** - orch-go's pkg/skills/loader.go reads deployed skills but doesn't build them. This is correct for its role as an orchestration tool that spawns agents with skills.

**Answer to Investigation Question:**

**Q: What does `orch build skills` actually do?**
A: Compiles templated skills from orch-knowledge/skills/src/ by expanding SKILL-TEMPLATE markers with phase content, then deploys to ~/.claude/skills/ (Claude Code) and ~/.config/opencode/agent/ (OpenCode with transformed frontmatter).

**Q: How does it differ from skillc's approach?**
A: Skillc compiles project-local .skillc/ directories to CLAUDE.md via dependency graph resolution. Different input format (skill.yaml + sources vs SKILL.md.template + phases), different output location (project-local vs system-wide), different purpose (project context vs procedural skills).

**Q: Should skillc replace it?**
A: No. They solve different problems. Skillc could theoretically compile skills (it handles SKILL.md as an artifact type per the decision doc), but the orch-knowledge template expansion pattern is specific to the skill system and well-served by the existing Python tooling.

**Q: Should orch-go port it?**
A: Only if Python orch-cli is being deprecated and orch-go needs to replace all its functionality. Currently not required - skill building is separate from agent orchestration. The existing Python `orch build skills` works fine.

**Q: What's the migration path if any?**
A: None needed. These are complementary tools:
- Continue using `orch build skills` (Python) for procedural skills
- Use `skillc build` for project-specific context (CLAUDE.md)
- orch-go consumes skills via pkg/skills/loader.go (already implemented)

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Both codebases were directly examined. Working artifacts exist in their expected locations. The purposes are clearly distinct based on code analysis.

**What's certain:**

- ✅ `orch build skills` handles template expansion from orch-knowledge to ~/.claude/skills/
- ✅ skillc handles .skillc/ compilation with dependency resolution to CLAUDE.md
- ✅ These are different artifact types for different purposes
- ✅ orch-go's skill loader is a consumer, not a builder
- ✅ Both systems have working deployed artifacts

**What's uncertain:**

- ⚠️ Whether Python orch-cli will be deprecated (would affect whether orch-go needs to port skill building)
- ⚠️ Whether there's value in skillc also supporting the orch-knowledge skill format (probably not)

**What would increase confidence to Very High (95%+):**

- Confirmation from Dylan on whether orch-cli Python is being deprecated
- Running `orch build skills` and `skillc build` end-to-end to verify current state

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Keep systems separate** - Continue using Python orch-cli for skill building and skillc for project context. No migration or replacement needed.

**Why this approach:**
- Systems serve different purposes (procedural skills vs project context)
- Both work and are deployed
- No overlap in functionality that would cause confusion
- orch-go's role is orchestration, not skill authoring

**Trade-offs accepted:**
- Maintaining two build tools (Python skills CLI + Go skillc)
- Python dependency for skill building if orch-go is meant to fully replace orch-cli

**Implementation sequence:**
1. No implementation needed - current state is correct
2. Document the distinction clearly (this investigation)
3. If orch-cli deprecation happens later, revisit whether orch-go needs skill building

### Alternative Approaches Considered

**Option B: Port skill building to orch-go**
- **Pros:** Single Go binary for all orchestration tooling
- **Cons:** Duplicates working Python code; skill building is separate concern from orchestration
- **When to use instead:** Only if Python orch-cli is fully deprecated

**Option C: Make skillc handle orch-knowledge templates**
- **Pros:** Single context compiler for all markdown artifacts
- **Cons:** SKILL-TEMPLATE expansion is fundamentally different from .skillc dependency resolution; would require skillc to understand Claude Code skill system
- **When to use instead:** Only if there's strong demand to consolidate all markdown compilation

**Rationale for recommendation:** The systems are working, clearly scoped, and solve different problems. Adding complexity to merge them would be premature optimization with no clear benefit.

---

### Implementation Details

**What to implement first:**
- Nothing - document findings and close investigation

**Things to watch out for:**
- ⚠️ If orch-cli Python is deprecated, skill building would need a new home (orch-go or standalone)
- ⚠️ New developers might confuse the two systems - documentation helps

**Areas needing further investigation:**
- Whether orch-go should have a `orch build` command group at all
- Whether skillc should have CLI parity with skill building (`skillc build --template` mode?)

**Success criteria:**
- ✅ Investigation documents the distinction clearly
- ✅ Decision is recorded for future reference
- ✅ No action items blocking other work

---

## Self-Review

- [x] Real test performed (examined actual code, not just documentation)
- [x] Conclusion from evidence (based on code analysis and deployed artifacts)
- [x] Question answered (all 4 sub-questions addressed)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn decide "skillc and orch build skills are complementary, not competing" --reason "skillc compiles project-local .skillc/ to CLAUDE.md; orch build skills compiles templated skills to ~/.claude/skills/. Different purposes, both needed."
```

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skills_cli.py:102-271` - Python skill building implementation
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skill_build.py:1-118` - OpenCode frontmatter transformation
- `/Users/dylanconlin/Documents/personal/skillc/cmd/skillc/main.go:1-323` - Skillc CLI entry point
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go:1-240` - Skillc compilation engine
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go:1-126` - orch-go skill loader (consumer)
- `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md` - Skillc scope decision
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/src/SKILL.md.template` - Example templated skill

**Commands Run:**
```bash
# Check deployed skills structure
ls -la ~/.claude/skills/
# 32 items: symlinks + category directories

# Check OpenCode agent directory  
ls -la ~/.config/opencode/agent/
# 17 .md files with transformed frontmatter

# Test orch build skills help
cd /Users/dylanconlin/Documents/personal/orch-cli && uv run orch build skills --help
```

**Related Artifacts:**
- **Decision:** `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md` - Defines skillc's intentional scope
- **Investigation:** `/Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-21-inv-implement-skillc-go-as-standalone.md` - Prior investigation on skillc design

---

## Investigation History

**2025-12-22 ~09:45:** Investigation started
- Initial question: What does orch build skills do vs skillc?
- Context: Spawned to understand relationship between two build systems

**2025-12-22 ~09:50:** Found Python skill building implementation
- Discovered template expansion and dual-target deployment pattern
- Located OpenCode frontmatter transformation logic

**2025-12-22 ~10:00:** Found skillc compilation approach
- Discovered .skillc/ directory pattern with skill.yaml manifests
- Found dependency graph resolution and topological sort

**2025-12-22 ~10:10:** Verified deployed artifacts
- Confirmed ~/.claude/skills/ has category structure with symlinks
- Confirmed ~/.config/opencode/agent/ has flat transformed skills

**2025-12-22 ~10:15:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Systems are complementary, not competing - no migration needed
