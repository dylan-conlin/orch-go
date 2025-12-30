<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented semantic parsing layer that classifies changelog commits by type (documentation/behavioral/structural), blast radius (local/cross-skill/infrastructure), and prominently flags BREAKING changes.

**Evidence:** All 40+ unit tests pass covering conventional commit parsing, change type inference, blast radius detection, semantic category mapping, and badge generation.

**Knowledge:** Commits can be semantically classified using two orthogonal axes from skill change taxonomy: change type (what kind of change) and blast radius (how widely it affects the system).

**Next:** Close - implementation complete, tests passing, integrated into parseGitLog pipeline.

---

# Investigation: Add Semantic Parsing Layer Changelog

**Question:** How should changelog entries be semantically parsed and classified to surface behavioral meaning?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-feat-add-semantic-parsing-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** Part of Epic orch-go-v7qs (Cross-Project Change Visibility)
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Conventional commit parsing enables semantic classification

**Evidence:** Implemented `parseConventionalCommit()` that extracts commit type (feat/fix/docs/refactor/etc.) and BREAKING status from commit subjects. Handles standard formats: `type: msg`, `type(scope): msg`, `type!: msg`, and `BREAKING:` prefix.

**Source:** `cmd/orch/changelog.go:306-344`

**Significance:** Provides primary signal for change type classification, enabling automated categorization based on developer intent expressed in commit messages.

---

### Finding 2: File path patterns reveal semantic meaning

**Evidence:** Implemented `inferSemanticCategory()` and `inferChangeTypeFromFiles()` that analyze file paths to determine:
- `.kb/decisions/` → decision-record category
- `.kb/investigations/` → investigation category  
- `skills/` with only .md files → skill-docs
- `skills/` with code files → skill-behavioral

**Source:** `cmd/orch/changelog.go:500-533`

**Significance:** Enables fallback classification when conventional commits not used, and provides more specific semantic categories beyond basic file location.

---

### Finding 3: Blast radius derivable from file patterns

**Evidence:** Implemented `inferBlastRadius()` that detects:
- Infrastructure changes: `pkg/spawn/`, `pkg/verify/`, `skill.yaml`, `SPAWN_CONTEXT`
- Cross-skill: changes to 2+ distinct skills
- Local: single file or single component

**Source:** `cmd/orch/changelog.go:411-464`

**Significance:** Enables orchestrators to quickly identify high-impact changes that may require design-session vs simple review.

---

## Synthesis

**Key Insights:**

1. **Two orthogonal axes** - Skill change taxonomy's axes (change type + blast radius) apply directly to commit classification, providing meaningful semantic grouping.

2. **Layered parsing strategy** - Conventional commits provide explicit intent; file patterns provide fallback and additional context; combination gives robust classification.

3. **Visual prominence matters** - BREAKING changes get dedicated 🚨 icon and bypass normal category icon, ensuring they're immediately visible in changelog output.

**Answer to Investigation Question:**

Changelog entries should be semantically parsed using: (1) conventional commit prefix parsing for change type and breaking status, (2) file path analysis for semantic category inference, and (3) file pattern matching for blast radius detection. Results are displayed as human-readable badges like `[BREAKING | behavioral | infrastructure]` alongside each commit entry.

---

## Structured Uncertainty

**What's tested:**

- ✅ Conventional commit parsing handles feat/fix/docs/refactor/chore/perf/test/build/ci (verified: 11 test cases pass)
- ✅ BREAKING detection via prefix, `!`, and message content (verified: 3 test cases pass)
- ✅ Blast radius detection for local/cross-skill/infrastructure (verified: 7 test cases pass)
- ✅ Semantic category inference for decision-record/investigation/skill-behavioral/skill-docs (verified: 5 test cases pass)

**What's untested:**

- ⚠️ Real-world changelog output appearance (not run against live repos during implementation)
- ⚠️ Edge cases with unusual commit message formats
- ⚠️ Performance with large numbers of commits

**What would change this:**

- Finding would be incomplete if conventional commit adoption is very low (fallback would dominate)
- Blast radius detection might need tuning if file patterns evolve

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-27-inv-skill-change-taxonomy.md` - Referenced taxonomy for change type and blast radius axes
- `cmd/orch/changelog.go` - Existing changelog implementation to extend

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-skill-change-taxonomy.md` - Source of taxonomy categories
- **Epic:** orch-go-v7qs (Cross-Project Change Visibility)

---

## Investigation History

**2025-12-30 15:30:** Investigation started
- Initial question: How should changelog entries be semantically parsed?
- Context: Epic orch-go-v7qs.2 requires semantic parsing layer for changelog

**2025-12-30 15:45:** Implementation complete
- Status: Complete
- Key outcome: Semantic parsing layer added with conventional commit parsing, file path inference, blast radius detection, and human-readable badges
