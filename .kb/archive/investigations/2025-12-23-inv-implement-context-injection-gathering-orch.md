<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented skill-driven context injection in orch spawn - skills can now declare required context via SKILL-REQUIRES block that orch-go parses and gathers at spawn time.

**Evidence:** All 26 new tests pass; build succeeds; integration with main.go spawn flow complete.

**Knowledge:** Context injection follows the Layer 1 pattern (outputs) - skillc embeds as HTML comments, orch-go extracts and gathers. Backwards compatible with existing kb-context behavior.

**Next:** Close - implementation complete. Skillc side (SKILL-REQUIRES embedding) can be implemented separately.

**Confidence:** High (85%) - Unit tests pass, but integration testing with real skills pending (requires skillc embedding).

---

# Investigation: Implement Context Injection Gathering in orch spawn

**Question:** How should orch spawn parse SKILL-REQUIRES from skill content and gather the declared context at spawn time?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** og-feat-implement-context-injection-23dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: SKILL-REQUIRES Block Format Mirrors SKILL-CONSTRAINTS

**Evidence:** The design investigation established that SKILL-REQUIRES should use the same HTML comment embedding pattern as SKILL-CONSTRAINTS (Layer 1):
```markdown
<!-- SKILL-REQUIRES -->
<!-- kb-context: true -->
<!-- beads-issue: true -->
<!-- prior-work: .kb/investigations/* -->
<!-- /SKILL-REQUIRES -->
```

**Source:** `/Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-23-design-context-injection-skillc-layer.md:127-132`

**Significance:** Following the established pattern ensures consistency and allows similar extraction logic in orch-go.

---

### Finding 2: Current kb-context Gathering is Implicit

**Evidence:** The `runPreSpawnKBCheck()` function in main.go (line 3406-3452) runs kb context for ALL spawns regardless of skill type. This is not skill-driven.

**Source:** `cmd/orch/main.go:3406-3452`

**Significance:** Making this explicit via skill requirements gives skills control over their context needs and prevents unnecessary context gathering for skills that don't need it.

---

### Finding 3: Import Cycle Avoided with Local Types

**Evidence:** Initial implementation tried to import `verify.GetIssue()` from spawn package, causing an import cycle. Solved by duplicating minimal beads types locally:
- `beadsIssue` struct with ID, Title, Description, IssueType, Status
- `getBeadsIssue()` function calling `bd show --json`

**Source:** `pkg/spawn/skill_requires.go:192-223`

**Significance:** Local types avoid import cycles while providing the same functionality. This is a common Go pattern for avoiding circular dependencies.

---

## Synthesis

**Key Insights:**

1. **Skill-driven context injection follows Layer 1 pattern** - Just as outputs are declared in skill.yaml and embedded by skillc, context requirements use the same mechanism.

2. **Backwards compatible by default** - When no SKILL-REQUIRES block is found, falls back to existing implicit kb-context behavior.

3. **Three context types supported** - kb-context (prior knowledge), beads-issue (issue details), and prior-work (file patterns) cover the main use cases identified in the design.

**Answer to Investigation Question:**

orch spawn should:
1. Load skill content (already done)
2. Call `ParseSkillRequires(skillContent)` to extract requirements
3. If requirements found, call `GatherRequiredContext()` with task, beadsID, projectDir
4. Otherwise, fall back to `runPreSpawnKBCheck(task)` for backwards compatibility
5. Pass gathered context to Config.KBContext for inclusion in SPAWN_CONTEXT.md

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Unit tests cover all parsing and gathering logic. The integration point in main.go is straightforward. However, end-to-end testing with real skills requires skillc to embed SKILL-REQUIRES blocks.

**What's certain:**

- Parsing of SKILL-REQUIRES HTML comment blocks works correctly (26 tests pass)
- TLDR extraction handles D.E.K.N., Summary sections, and fallback to first paragraph
- Integration into spawn flow is backwards compatible

**What's uncertain:**

- Real skill content from skillc (depends on skillc implementing embedding)
- Performance with large .kb/ directories (timeout protection in place but untested at scale)
- Beads JSON format assumptions (based on existing verify package patterns)

**What would increase confidence to Very High (95%+):**

- End-to-end test with real skill that has SKILL-REQUIRES block
- Performance testing with large prior-work pattern matches
- Integration testing with actual beads issues

---

## Implementation Details

**Files Created:**
- `pkg/spawn/skill_requires.go` - RequiresContext struct, parsing, and gathering logic
- `pkg/spawn/skill_requires_test.go` - 26 test cases

**Files Modified:**
- `cmd/orch/main.go` - Integration into `runSpawnWithSkill()` function

**Key Functions:**
- `ParseSkillRequires(content string)` - Extract RequiresContext from skill content
- `GatherRequiredContext(requires, task, beadsID, projectDir)` - Gather all declared context
- `gatherKBContext(task)` - Run kb context with task keywords
- `gatherBeadsIssueContext(beadsID)` - Fetch and format issue details
- `gatherPriorWorkContext(patterns, projectDir)` - Glob match and extract TLDRs

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-23-design-context-injection-skillc-layer.md` - Design investigation
- `pkg/spawn/context.go` - Existing SPAWN_CONTEXT.md generation
- `pkg/spawn/kbcontext.go` - Existing kb context gathering
- `cmd/orch/main.go` - Spawn command integration point

**Commands Run:**
```bash
# Build all packages
go build ./...

# Run skill requires tests
go test ./pkg/spawn/... -v -run 'Test(ParseSkillRequires|RequiresContext|ExtractTLDR|GatherPriorWork|ParseBool)'

# Run full test suite
go test ./...
```

**Related Artifacts:**
- **Design:** `/Users/dylanconlin/Documents/personal/skillc/.kb/investigations/2025-12-23-design-context-injection-skillc-layer.md` - Layer 3 design

---

## Investigation History

**2025-12-23:** Investigation started
- Initial question: How should orch spawn gather context based on skill requirements?
- Context: Layer 3 of Executable Skill Constraints epic

**2025-12-23:** Implementation completed
- Created skill_requires.go with RequiresContext parsing
- Added 26 test cases
- Integrated into main.go spawn flow
- All tests passing

**2025-12-23:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Skill-driven context injection implemented in orch-go, ready for skillc to embed SKILL-REQUIRES blocks
