# Session Synthesis

**Agent:** og-arch-design-knowledge-system-22dec
**Issue:** orch-go-8sa5
**Duration:** 2025-12-22 ~08:30 → 2025-12-22 ~09:45
**Outcome:** success

---

## TLDR

Designed knowledge system support for project extraction/refactoring scenarios. Recommended inline lineage metadata approach (headers in artifacts + project manifest) over centralized registry, aligned with Session Amnesia principle.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-design-knowledge-system-project-extraction.md` - Design investigation with 5 findings, 4 approaches evaluated, recommendation

### Files Modified
- None (design-only session)

### Commits
- Pending commit with investigation artifact

---

## Evidence (What Was Observed)

- kb-cli has migrate (.orch→.kb) and publish (local→global) but no cross-project migration
- `~/.kb/projects.json` tracks 17 projects but no relationships between them
- Session Amnesia principle (`~/.kb/principles.md:14-68`) dictates self-describing artifacts
- Git's distributed lineage model (commits carry parent refs, no central registry) proves pattern at scale
- Five extraction scenarios identified: component extraction, lineage tracking, cross-refs, supersedes, deprecated projects

### Tests Run
```bash
# Verified kb CLI capabilities
kb --help  # Confirmed migrate, publish, projects commands
kb projects list  # Confirmed 17 registered projects
cat ~/.kb/projects.json  # Verified registry structure (name + path only)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-design-knowledge-system-project-extraction.md` - Complete design investigation

### Decisions Made
- Inline lineage metadata > centralized registry because Session Amnesia requires self-describing artifacts
- Copy not move during extraction - originals stay for historical reference
- Two-way supersedes linking (old points forward, new points back)
- DEPRECATED.md pattern for project-level deprecation marker

### Constraints Discovered
- Cross-project knowledge migration is rare (2-3x/year) but high-impact
- Automated cross-reference updating is over-engineering - manual discovery acceptable

### Externalized via `kn`
- `kn decide "Use inline lineage metadata for project extraction, not centralized registry" --reason "Session Amnesia principle requires self-describing artifacts; centralized registry creates fragile external dependency"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation artifact produced)
- [x] Tests passing (N/A - design session)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-8sa5`

---

## Implementation Roadmap (for future work)

1. **Template changes** (low-effort, immediate value)
   - Add `extracted-from:`, `supersedes:`, `superseded-by:` to investigation/decision templates
   
2. **`kb extract` command** (enables first migration)
   - `kb extract <artifact> --to <project>` 
   - Copies artifact with inline lineage header
   
3. **`kb supersede` command** (marks replacements)
   - `kb supersede <old> --by <new>`
   - Updates both artifacts with two-way links

4. **`.kb/manifest.yaml`** (project-level metadata)
   - Record lineage at project level
   - `lineage: { extracted-from: X, extraction-date: Y, reason: Z }`

5. **DEPRECATED.md pattern** (project deprecation)
   - Template for marking projects as deprecated
   - Clear successor project reference

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `kb context` search across lineage relationships? (Would help find superseded decisions)
- Should `kb reflect --type lineage` surface broken cross-references?
- How to handle orch-cli → orch-go migration in practice? (200+ investigations)

**Areas worth exploring further:**
- Testing the extract workflow on skillc as retrospective exercise
- Whether manifest.yaml is sufficient or artifact-level lineage is always needed

**What remains unclear:**
- Exact ergonomics of `kb extract` command (needs implementation to validate)
- How often cross-references actually break in practice

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-knowledge-system-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-design-knowledge-system-project-extraction.md`
**Beads:** `bd show orch-go-8sa5`
