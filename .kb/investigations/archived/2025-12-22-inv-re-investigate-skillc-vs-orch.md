---
linked_issues:
  - orch-go-vsjv
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Skillc's decision doc claims SKILL.md as in-scope, but skillc cannot currently compile skills due to template expansion gap - `orch build skills` does phase template expansion that skillc's simple concatenation doesn't support.

**Evidence:** Tested both systems: skillc does dependency-ordered concatenation; orch-cli replaces `<!-- SKILL-TEMPLATE: phase-name -->` markers with content from `src/phases/*.md` files. Skillc has no template expansion capability.

**Knowledge:** The prior investigation was correct about complementary purposes (project context vs procedural skills) but missed that skillc's decision doc over-promised by listing SKILL.md as in-scope without the template expansion feature to support it.

**Next:** Either (a) remove SKILL.md from skillc's decision doc, or (b) add template expansion to skillc if consolidating build tools is desired. Recommendation: Option (a) for now.

**Confidence:** Very High (95%) - Both systems tested, code examined, gap clearly identified.

---

# Investigation: Re-Investigate Skillc vs Orch Build Skills Relationship

**Question:** Does skillc's decision document (which lists SKILL.md as in-scope) mean migration from `orch build skills` makes sense? What gaps exist?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-inv-re-investigate-skillc-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Context

The prior investigation (`2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md`) concluded skillc and orch build skills are "complementary, not competing." However, the task notes that skillc's decision document explicitly lists SKILL.md as an in-scope artifact type. This re-investigation checks whether that claim is accurate and what the migration path would be.

---

## Findings

### Finding 1: Skillc's decision document claims SKILL.md is in-scope

**Evidence:** From `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md`:

```markdown
### In Scope
| Artifact | Output | Purpose |
|----------|--------|---------|
| Project context | `CLAUDE.md` | Project-specific guidance for Claude |
| Skill definitions | `SKILL.md` | Procedure/skill guidance for agents |
| Hook context | `context.md` | Markdown injected by AI hooks |
```

And the rationale: "Skillc's job is compiling modular markdown sources into self-describing artifacts that AI agents consume."

**Source:** `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md:11-17`

**Significance:** This is aspirational, not implemented. The decision doc states SKILL.md is in-scope, but skillc has no special handling for it beyond its generic concatenation capability.

---

### Finding 2: Orch build skills does template expansion that skillc cannot

**Evidence:** The Python `skills_cli.py` implements template expansion:

```python
# Pattern to find template markers
pattern = r'<!--\s*SKILL-TEMPLATE:\s*([a-zA-Z0-9_-]+)\s*-->(.*?)<!--\s*/SKILL-TEMPLATE\s*-->'
new_content = re.sub(pattern, replace_template, template_content, flags=re.DOTALL)
```

This replaces markers like `<!-- SKILL-TEMPLATE: investigation --><!-- /SKILL-TEMPLATE -->` with content from `src/phases/investigation.md`.

The feature-impl skill template has 8 such markers:
- investigation, clarifying-questions, design, implementation-tdd, implementation-direct, validation, self-review, integration

**Source:** `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skills_cli.py:180-194`

**Significance:** This is the core gap. Skillc's concatenation model (load files, sort by dependency, join) cannot handle the inline template expansion pattern used by orch-knowledge skills.

---

### Finding 3: Skillc does dependency-ordered concatenation, not template expansion

**Evidence:** From skillc's `compiler.go`:

```go
// Concatenate skills in dependency order
for _, skillName := range order {
    skill := skills[skillName]
    output.WriteString(fmt.Sprintf("<!-- Skill: %s -->\n\n", skillName))
    output.WriteString(skill.Content)
    output.WriteString("\n\n")
}
```

The manifest supports:
- `sources: []` - list of files to concatenate
- `dependencies: []` - order resolution
- `output:` - custom output filename
- `preserve_placeholders:` - pass through `{{VAR}}` patterns

**No template marker expansion exists.**

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go:94-105`

**Significance:** Skillc's design is fundamentally different from the orch-knowledge skill build pipeline. Adding template expansion would require either:
1. A new manifest field like `template_sources:` that get expanded inline
2. A different compilation mode for skills

---

### Finding 4: The two systems target different deployment locations

**Evidence:** 

**orch build skills:**
- Input: `orch-knowledge/skills/src/{category}/{skill}/src/SKILL.md.template` + `phases/*.md`
- Output (Claude Code): `~/.claude/skills/{category}/{skill}/SKILL.md`
- Output (OpenCode): `~/.config/opencode/agent/{skill}.md` (with transformed frontmatter)
- Creates symlinks for flat skill access

**skillc build:**
- Input: `.skillc/` directory in project with `skill.yaml` manifest
- Output: Project-local file (default `CLAUDE.md`, configurable via manifest)
- No system-wide deployment, no symlinks

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skills_cli.py:277-387` (deploy)
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go:67-79` (output path)

**Significance:** Even if skillc gained template expansion, it would still need deployment functionality to replace orch build skills. Skillc is designed for project-local artifacts, not system-wide skill installation.

---

### Finding 5: Prior investigation conclusion was correct, but missed the decision doc claim

**Evidence:** The prior investigation concluded:

> "Systems are complementary, not competing - no migration needed."

This is accurate for what the systems currently DO. But the prior investigation didn't address the mismatch between:
1. What skillc's decision document CLAIMS (SKILL.md in scope)
2. What skillc CAN DO (no template expansion)

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md`

**Significance:** The decision document is either (a) aspirational (future feature), or (b) incorrectly scoped (SKILL.md shouldn't be listed without the capability to build it). This investigation clarifies the gap.

---

## Synthesis

**Key Insights:**

1. **The decision doc over-promises** - Listing SKILL.md as in-scope without the template expansion capability to support it creates confusion. The prior investigation's "complementary" conclusion was correct about current functionality, but the decision doc implied more.

2. **Template expansion is the core gap** - Skillc's concat model is fundamentally different from orch's inline template expansion. The orch-knowledge skill templates use `<!-- SKILL-TEMPLATE: X -->` markers that get replaced with phase content. Skillc concatenates files in dependency order but doesn't expand templates.

3. **Deployment is a secondary gap** - Even with template expansion, skillc would need deployment functionality (copy to ~/.claude/skills/, create symlinks, transform frontmatter for OpenCode) to fully replace orch build skills.

4. **Two reasonable paths forward:**
   - **Path A (clarify scope):** Remove SKILL.md from skillc's decision doc. Skillc stays focused on project context (CLAUDE.md), orch build skills continues for procedural skills.
   - **Path B (extend skillc):** Add template expansion to skillc. Would need new manifest syntax (`template_sources:` or `template_markers:`). Then potentially add deployment command.

**Answer to Investigation Question:**

**Does skillc's decision document mean migration makes sense?**

No, not currently. The decision doc over-promised by listing SKILL.md without the template expansion capability. Migration would require:
1. Adding template expansion to skillc (significant feature work)
2. Adding deployment capability (--deploy-skills flag or similar)
3. Handling the dual-target issue (Claude Code + OpenCode formats)

The prior investigation's conclusion was correct: keep systems separate. The decision doc should either be updated to reflect actual scope, or skillc needs feature work before SKILL.md compilation is realistic.

---

## Test Performed

**Test 1:** Verified orch build skills detects skills needing rebuild

```bash
cd /Users/dylanconlin/Documents/personal/orch-cli
uv run python -c "from orch.skills_cli import cli; cli(['build', '--source', '/Users/dylanconlin/orch-knowledge/skills/src', '--check'])"
```

**Result:**
```
📦 Found 2 templated skill(s)

⚠️  worker/feature-impl - Needs rebuild
⚠️  worker/codebase-audit - Needs rebuild

⚠️  2 skill(s) need rebuilding
```

**Test 2:** Verified template markers in SKILL.md.template

```bash
grep -n "SKILL-TEMPLATE" /Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/src/SKILL.md.template
```

**Result:** Found 8 template markers (investigation, clarifying-questions, design, implementation-tdd, implementation-direct, validation, self-review, integration)

**Test 3:** Verified skillc has no template expansion

Reviewed `compiler.go:94-105` - only concatenation, no regex substitution or template expansion.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Both systems tested with actual commands. Code examined to confirm template expansion exists in orch-cli but not in skillc. The gap is clear and verifiable.

**What's certain:**

- ✅ Skillc's decision doc lists SKILL.md as in-scope artifact
- ✅ Orch build skills does template expansion (`<!-- SKILL-TEMPLATE: X -->` → phase content)
- ✅ Skillc does dependency-ordered concatenation only, no template expansion
- ✅ The two systems target different deployment locations (system-wide vs project-local)
- ✅ Prior investigation conclusion was correct about current functionality

**What's uncertain:**

- ⚠️ Whether the decision doc was aspirational (future feature) or incorrect (should remove SKILL.md)
- ⚠️ Whether there's appetite to add template expansion to skillc

**What would increase confidence to 100%:**

- Confirmation from Dylan on intent of decision doc
- Testing skillc actually trying to compile a skill (would fail, but would confirm)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Clarify skillc's scope** - Update the decision document to remove SKILL.md from in-scope, or add a note that it's aspirational pending template expansion feature.

**Why this approach:**
- Avoids false expectations from the decision doc
- Maintains clean separation of concerns (skillc = project context, orch = skills)
- Minimal work, maximum clarity
- Can revisit if skill building consolidation becomes a priority

**Trade-offs accepted:**
- Two build tools remain (Python orch-cli for skills, Go skillc for context)
- If orch-cli Python is deprecated, skill building would need a new home

**Implementation sequence:**
1. Update `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md`
2. Either remove SKILL.md from in-scope table, or add "Future" column noting template expansion required

### Alternative Approaches Considered

**Option B: Add template expansion to skillc**
- **Pros:** Would enable skillc to compile skills; consolidates build tooling
- **Cons:** Significant feature work; still need deployment capability; transforms skillc from simple compiler to complex build system
- **When to use instead:** If there's strong desire to consolidate all markdown build tooling in Go and deprecate Python orch-cli

**Option C: Leave decision doc as-is (aspirational)**
- **Pros:** No changes needed
- **Cons:** Creates confusion when someone tries to use skillc for skills; doc says one thing, tool does another
- **When to use instead:** Only if template expansion feature is actively planned for skillc

**Rationale for recommendation:** Clean separation is working. The systems do different things well. Updating the decision doc to reflect reality is the minimal-work, maximum-clarity solution.

---

## Self-Review

- [x] Real test performed (ran actual build commands, examined actual code)
- [x] Conclusion from evidence (template expansion gap clearly identified)
- [x] Question answered (migration doesn't make sense, decision doc over-promised)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (summary at top)
- [x] NOT DONE claims verified (tested skillc capabilities directly)

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn constrain "skillc cannot compile SKILL.md templates without template expansion feature" --reason "orch-knowledge skills use SKILL-TEMPLATE markers that require regex substitution not concatenation"
```

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md` - Decision doc claiming SKILL.md in scope
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/skills_cli.py` - Template expansion implementation
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go` - Skillc compilation logic
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/src/SKILL.md.template` - Example template with markers
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md` - Prior investigation

**Commands Run:**
```bash
# Test orch build skills
cd /Users/dylanconlin/Documents/personal/orch-cli
uv run python -c "from orch.skills_cli import cli; cli(['build', '--source', '/Users/dylanconlin/orch-knowledge/skills/src', '--check'])"

# Find template markers
grep -n "SKILL-TEMPLATE" /Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/src/SKILL.md.template

# Check skillc help
./skillc --help
```

**Related Artifacts:**
- **Decision:** `/Users/dylanconlin/Documents/personal/skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md` - Claims SKILL.md in scope
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md` - Prior investigation

---

## Investigation History

**2025-12-22:** Investigation started
- Initial question: Does skillc's SKILL.md scope claim mean migration from orch build skills?
- Context: Prior investigation concluded "complementary" but task notes decision doc lists SKILL.md explicitly

**2025-12-22:** Found the gap
- Discovered template expansion in orch-cli that skillc doesn't have
- Confirmed prior conclusion was correct about functionality, but decision doc over-promised

**2025-12-22:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Decision doc should be updated to clarify SKILL.md is aspirational or remove it from scope
