<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully migrated feature-impl skill to .skillc/ structure with working SKILL-TEMPLATE expansion.

**Evidence:** skillc build produces 1757-line SKILL.md with frontmatter at line 1. Template expansion works (grep shows "Auto-generated from phases/investigation.md"). orch-go ParseSkillMetadata tests pass.

**Knowledge:** SKILL-TEMPLATE markers must be on same line (e.g., `<!-- SKILL-TEMPLATE: name --><!-- /SKILL-TEMPLATE -->`) for Go regex to match them - newlines between markers break the pattern.

**Next:** None - migration complete. Old src/ directory can be removed after verification.

**Confidence:** High (95%) - build successful, tests pass, frontmatter correct.

---

# Investigation: Migrate feature-impl Skill to .skillc Structure

**Question:** How to migrate the feature-impl skill from SKILL-TEMPLATE markers to the .skillc/ modular structure?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)

---

## Findings

### Finding 1: Existing feature-impl has src/ structure with SKILL-TEMPLATE markers

**Evidence:** The skill already has a modular structure:
- `~/.claude/skills/worker/feature-impl/src/SKILL.md.template` - main template (323 lines)
- `~/.claude/skills/worker/feature-impl/src/phases/*.md` - individual phase files

Template markers present in SKILL.md.template:
```
<!-- SKILL-TEMPLATE: investigation -->
<!-- SKILL-TEMPLATE: clarifying-questions -->
<!-- SKILL-TEMPLATE: design -->
<!-- SKILL-TEMPLATE: implementation-tdd -->
<!-- SKILL-TEMPLATE: implementation-direct -->
<!-- SKILL-TEMPLATE: validation -->
<!-- SKILL-TEMPLATE: self-review -->
<!-- SKILL-TEMPLATE: leave-it-better -->
<!-- SKILL-TEMPLATE: integration -->
```

**Source:** `~/.claude/skills/worker/feature-impl/src/SKILL.md.template:187-228`

**Significance:** The skill is already structured for modular build. Migration to .skillc/ is straightforward - we need to create skill.yaml with template_sources mapping.

### Finding 2: skillc already supports template_sources for SKILL-TEMPLATE expansion

**Evidence:** From `skillc/pkg/compiler/manifest.go:20`:
```go
TemplateSources map[string]string `yaml:"template_sources"` // Map of template name to source file path for SKILL-TEMPLATE expansion
```

And `compiler.go:241-318` implements `loadSourceFilesWithTemplates()` and `expandTemplates()` functions that:
1. Load template sources from the specified paths
2. Find `<!-- SKILL-TEMPLATE: name -->` markers in source files
3. Replace content between markers with content from template source files

**Source:** `skillc/pkg/compiler/compiler.go:241-371`

**Significance:** No changes to skillc needed. Just need to create proper skill.yaml manifest.

### Finding 3: Phases have significant content that should remain as separate files

**Evidence:** Phase file sizes:
- investigation.md: 187 lines
- clarifying-questions.md: 169 lines
- design.md: 150 lines
- implementation-tdd.md: 140 lines
- implementation-direct.md: 113 lines
- validation.md: 142 lines
- self-review.md: 306 lines
- leave-it-better.md: 79 lines
- integration.md: 121 lines

**Source:** Direct file inspection of `~/.claude/skills/worker/feature-impl/src/phases/`

**Significance:** These files are substantial and well-organized. Should keep them as separate source files in .skillc/ structure.

### Finding 4: Template markers must be on same line for Go regex match

**Evidence:** Initial build produced empty template sections. The regex:
```go
var templateMarkerRegex = regexp.MustCompile(`<!--\s*SKILL-TEMPLATE:\s*([a-zA-Z0-9_-]+)\s*-->(.*?)<!--\s*/SKILL-TEMPLATE\s*-->`)
```

The `(.*?)` non-greedy match doesn't cross newlines in Go by default. Changed from:
```
<!-- SKILL-TEMPLATE: name -->
<!-- /SKILL-TEMPLATE -->
```
To:
```
<!-- SKILL-TEMPLATE: name --><!-- /SKILL-TEMPLATE -->
```

**Source:** Testing during implementation

**Significance:** Key gotcha for future migrations - markers must be on same line.

### Finding 5: Migration produces working SKILL.md

**Evidence:**
```bash
$ cd ~/.claude/skills/worker/feature-impl && skillc build
✓ Compiled .skillc to SKILL.md

$ wc -l SKILL.md
    1757 SKILL.md

$ head -5 SKILL.md
---
name: feature-impl
skill-type: procedure
audience: worker
spawnable: true

$ grep -n "Auto-generated from" SKILL.md | head -3
194:<!-- Auto-generated from phases/investigation.md -->
367:<!-- Auto-generated from phases/clarifying-questions.md -->
538:<!-- Auto-generated from phases/design.md -->
```

**Source:** Build output

**Significance:** Migration complete and working.

---

## Synthesis

**Key Insights:**

1. **skillc supports SKILL-TEMPLATE expansion out of the box** - The `template_sources` manifest field maps template names to source files, and the compiler expands markers automatically.

2. **Template marker format is critical** - Opening and closing markers must be on the same line for the Go regex to match. This is a potential gotcha for future migrations.

3. **Migration path is straightforward** - Create `.skillc/` with skill.yaml (including `type: skill`, `frontmatter`, and `template_sources`), copy template file without frontmatter, copy phase files to phases/ subdirectory, run `skillc build`.

**Answer to Question:**

Migration requires:
1. Create `.skillc/skill.yaml` with:
   - `type: skill` (for frontmatter output)
   - `frontmatter: frontmatter.yaml` (skill metadata)
   - `sources: [SKILL.md.template]` (main template)
   - `template_sources:` map (phase name → file path)
2. Create `frontmatter.yaml` with skill metadata (extracted from original SKILL.md frontmatter)
3. Create `SKILL.md.template` WITHOUT frontmatter, WITH template markers on same line
4. Copy phase files to `phases/` subdirectory
5. Run `skillc build`

---

## Confidence Assessment

**Current Confidence:** High (95%)

**Why this level?**
Build successful, output verified, orch-go tests pass. Only uncertainty is full end-to-end spawn test.

**What's tested:**
- skillc build produces valid SKILL.md
- Frontmatter at line 1 (YAML parseable)
- Template expansion works (9 phases expanded)
- orch-go ParseSkillMetadata tests pass

**What's untested:**
- Full `orch spawn feature-impl "test"` integration test

**What would change this:**
- orch spawn failing to load the skill content correctly

---

## References

**Files Created:**
- `~/.claude/skills/worker/feature-impl/.skillc/skill.yaml`
- `~/.claude/skills/worker/feature-impl/.skillc/frontmatter.yaml`
- `~/.claude/skills/worker/feature-impl/.skillc/SKILL.md.template`
- `~/.claude/skills/worker/feature-impl/.skillc/phases/*.md` (9 files)

**Files Examined:**
- `~/.claude/skills/worker/feature-impl/src/SKILL.md.template`
- `~/.claude/skills/worker/feature-impl/src/phases/*.md`
- `skillc/pkg/compiler/compiler.go`
- `skillc/pkg/compiler/manifest.go`

**Related Artifacts:**
- .kb/investigations/2025-12-22-inv-pilot-migration-convert-investigation-skill.md
- .kb/investigations/2025-12-22-inv-epic-replace-orch-knowledge-skillc.md

---

## Investigation History

**2025-12-22 10:00:** Investigation started
- Read spawn context and prior investigations
- Examined current feature-impl SKILL.md structure (1675 lines with SKILL-TEMPLATE markers)

**2025-12-22 10:05:** Context gathering
- Found existing src/ structure with modular phases
- Identified 9 phase files to migrate

**2025-12-22 10:10:** Implementation
- Created .skillc/ directory structure
- Created skill.yaml with template_sources mapping
- Created frontmatter.yaml with skill metadata
- Copied template and phase files

**2025-12-22 10:14:** First build attempt
- Build succeeded but template expansion didn't work
- Output was 10161 bytes (too small)

**2025-12-22 10:15:** Debugging
- Found markers on separate lines didn't match Go regex
- Fixed by putting markers on same line
- Rebuild produced 1757-line output with expanded phases

**2025-12-22 10:16:** Verification
- Confirmed frontmatter at line 1
- Confirmed template expansion working
- orch-go tests pass

---

## Self-Review

- [x] Real test performed (skillc build, test parsing)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
