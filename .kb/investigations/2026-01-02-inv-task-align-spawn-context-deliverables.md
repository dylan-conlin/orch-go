<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** SPAWN_CONTEXT now conditionally includes investigation file instructions based on skill/phase requirements.

**Evidence:** All 17 new tests pass; skills like investigation/research/architect require investigation files, while feature-impl only requires them when investigation phase is included.

**Knowledge:** Investigation file instructions should only appear for skills that produce knowledge artifacts as deliverables; feature-impl is conditional on phase configuration.

**Next:** Close - implementation complete with tests passing.

---

# Investigation: Task Align Spawn Context Deliverables

**Question:** How should SPAWN_CONTEXT deliverables be aligned with skill/phase requirements so investigation file instructions are conditional?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** spawned agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Skills have distinct investigation file requirements

**Evidence:** Skills like investigation, research, architect, codebase-audit, and systematic-debugging always produce investigation files. feature-impl only requires investigation files when the investigation phase is included in phases configuration.

**Source:** pkg/spawn/config.go:SkillTierDefaults mapping, SPAWN_CONTEXT.md analysis

**Significance:** Investigation file instructions should not be shown for skills/phases that don't produce investigation files.

---

### Finding 2: Template uses conditional rendering

**Evidence:** SpawnContextTemplate already uses Go template conditionals like `{{if .NoTrack}}` and `{{if ne .Tier "light"}}` for conditional sections.

**Source:** pkg/spawn/context.go:18-335

**Significance:** The same pattern can be used for conditional investigation file instructions.

---

## Implementation

Added:
1. `SkillRequiresInvestigationFile` map in config.go
2. `RequiresInvestigationFile(skillName, phases string) bool` helper function
3. `RequiresInvestigationFile` field in `contextData` struct
4. Conditional template logic wrapping investigation file instructions

Tests added:
- TestRequiresInvestigationFile (17 cases)
- TestGenerateContext_WithInvestigationFile (3 cases)
- TestGenerateContext_WithoutInvestigationFile (4 cases)

---

## Structured Uncertainty

**What's tested:**

- ✅ investigation, research, architect, codebase-audit, systematic-debugging return true (verified: unit tests)
- ✅ feature-impl returns true only when phases include "investigation" (verified: unit tests)
- ✅ issue-creation, reliability-testing, design-session return false (verified: unit tests)
- ✅ Generated SPAWN_CONTEXT includes/excludes investigation file instructions correctly (verified: unit tests)

**What's untested:**

- ⚠️ End-to-end spawn with new template (not run, would require actual spawn)

---

## References

**Files Modified:**
- pkg/spawn/config.go - Added SkillRequiresInvestigationFile map and RequiresInvestigationFile function
- pkg/spawn/context.go - Added RequiresInvestigationFile field to contextData, updated GenerateContext, conditional template

**Files Examined:**
- pkg/spawn/context_test.go - Added 24 new test cases

**Commands Run:**
```bash
# Run all spawn package tests
go test ./pkg/spawn/... - PASS (0.317s)
```
