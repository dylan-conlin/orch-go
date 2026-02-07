## Summary (D.E.K.N.)

**Delta:** Made investigation file mandate conditional on skill type - only investigation, architect, and research skills now mandate investigation files.

**Evidence:** Verified with test: investigation skill context contains "SET UP investigation file", feature-impl skill context does not.

**Knowledge:** Skills fall into two categories: investigation-type (produce knowledge artifacts, need investigation files) and implementation-type (produce code, don't need investigation files).

**Next:** Close - implementation complete, tests pass.

**Promote to Decision:** recommend-no (tactical fix, follows existing pattern from SkillTierDefaults)

---

# Investigation: Investigation File Mandate Conditional Skill

**Question:** How to make investigation file mandate conditional on skill type in SPAWN_CONTEXT template?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SPAWN_CONTEXT.md template mandates investigation files for ALL spawns

**Evidence:** Lines 220-233 in pkg/spawn/context.go unconditionally instruct all agents to create investigation files.

**Source:** `pkg/spawn/context.go:220-233`

**Significance:** This causes feature-impl and systematic-debugging agents to create investigation files they don't need, contributing to investigation over-production.

---

### Finding 2: SkillTierDefaults already categorizes skills by type

**Evidence:** `pkg/spawn/config.go` contains `SkillTierDefaults` map that separates:
- Full tier: investigation, architect, research, codebase-audit, design-session
- Light tier: feature-impl, systematic-debugging, reliability-testing, issue-creation

**Source:** `pkg/spawn/config.go:26-39`

**Significance:** The pattern for categorizing skills already exists; I followed the same approach for investigation file requirements.

---

## Implementation

### Changes Made

1. **Added `SkillRequiresInvestigation` map and `IsInvestigationSkill()` function** to `pkg/spawn/config.go`:
   - Skills requiring investigation files: investigation, architect, research, codebase-audit
   - Skills NOT requiring investigation files: feature-impl, systematic-debugging, reliability-testing, issue-creation, design-session
   - Unknown skills default to false (conservative - don't mandate investigation files)

2. **Added `IsInvestigationSkill` field** to `contextData` struct in `pkg/spawn/context.go`

3. **Modified template** to conditionally show investigation file deliverables:
   - Investigation skills: Show full investigation file setup (items 2-4)
   - Non-investigation skills: Show simplified "Task-specific deliverables from skill guidance" (item 2)
   - STATUS UPDATES section only shows for investigation skills

---

## Structured Uncertainty

**What's tested:**

- ✅ `IsInvestigationSkill("investigation")` returns true (verified: ran test script)
- ✅ `IsInvestigationSkill("feature-impl")` returns false (verified: ran test script)
- ✅ Investigation skill context contains "SET UP investigation file" (verified: GenerateContext test)
- ✅ Feature-impl skill context does NOT contain "SET UP investigation file" (verified: GenerateContext test)
- ✅ Project builds successfully (verified: `go build ./...`)

**What's untested:**

- ⚠️ End-to-end spawn behavior (not tested with actual orch spawn)

---

## References

**Files Modified:**
- `pkg/spawn/config.go` - Added `SkillRequiresInvestigation` map and `IsInvestigationSkill()` function
- `pkg/spawn/context.go` - Added `IsInvestigationSkill` field to contextData, modified template

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification
go run /tmp/test_spawn.go
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md` - Parent decision
- **Investigation:** `.kb/investigations/2026-01-23-inv-so-many-investigations-created-root.md` - Root cause analysis

---

## Investigation History

**2026-01-23:** Investigation started
- Initial question: How to make investigation file mandate conditional on skill type?
- Context: Investigation orch-go-fkn0p found template mandates investigation files for ALL spawns

**2026-01-23:** Implementation complete
- Status: Complete
- Key outcome: Investigation file mandate now conditional on skill type via IsInvestigationSkill()
