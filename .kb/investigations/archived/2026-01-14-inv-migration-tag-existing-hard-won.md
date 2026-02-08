<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully registered 5 hard-won patterns in orchestrator skill.yaml using load_bearing array format per the 2026-01-08 data model decision.

**Evidence:** `skillc check` confirms "All 5 load-bearing patterns present" - patterns exist in SKILL.md.template and are now protected from refactor erosion.

**Knowledge:** The load_bearing data model works as designed: pattern strings are searched in compiled output, provenance captures friction story, severity controls blocking behavior.

**Next:** Commit changes to orch-knowledge, then run `skillc deploy` to propagate to ~/.claude/skills.

**Promote to Decision:** recommend-no - Implementation of existing decision (2026-01-08-load-bearing-guidance-data-model.md), no new architectural choice.

---

# Investigation: Migration - Tag Existing Hard-Won Patterns

**Question:** How do we register the 5 identified hard-won patterns in the orchestrator skill with provenance, so they're protected from refactor erosion?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Agent (spawned feature-impl)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Data Model Already Designed and Implemented

**Evidence:** Decision document at `.kb/decisions/2026-01-08-load-bearing-guidance-data-model.md` specifies the exact format:
- `load_bearing[]` array in skill.yaml
- Each entry has: pattern, provenance, evidence (optional), severity (error|warn)
- skillc checks for pattern presence in compiled output

**Source:** `.kb/decisions/2026-01-08-load-bearing-guidance-data-model.md:47-58`

**Significance:** No design work needed - followed existing decision. The data model was already validated and implemented in skillc.

---

### Finding 2: All 5 Patterns Exist in SKILL.md.template

**Evidence:** Grep search confirmed all patterns present:
- "ABSOLUTE DELEGATION RULE" - line 472 (section header) + multiple references
- "Filter before presenting" - line 621
- "Surface decision prerequisites" - line 623
- "Pressure Over Compensation" - line 907 (section header) + multiple references
- "Mode Declaration Protocol" - line 1046 (subsection header)

**Source:** `grep` searches on `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`

**Significance:** Patterns are currently present - this migration ensures they're PROTECTED from future removal.

---

### Finding 3: skillc check Validates Load-Bearing Patterns

**Evidence:** Running `skillc check` on orchestrator skill produces:
```
✓ All 5 load-bearing patterns present
```

The check also flagged token budget exceeded (139.6%) and checksum mismatch, but load-bearing validation passed.

**Source:** `skillc check skills/src/meta/orchestrator/.skillc/` output

**Significance:** The protection mechanism works. If any pattern is removed in a future refactor, `skillc check` will fail.

---

## Synthesis

**Key Insights:**

1. **Migration is straightforward when data model exists** - The hard work was the data model design (orch-go-lv3yx.4). Migration is just YAML editing.

2. **Provenance captures the WHY** - Each pattern now carries its friction story. Future agents can understand why "ABSOLUTE DELEGATION RULE" matters (3-day derailment) without having to re-learn from experience.

3. **Severity distinguishes load-bearing levels** - Two patterns are severity: error (ABSOLUTE DELEGATION RULE, Pressure Over Compensation) because removing them causes system failure. Three are severity: warn (Filter before presenting, Surface decision prerequisites, Mode Declaration Protocol) because they improve quality but aren't existential.

**Answer to Investigation Question:**

Register hard-won patterns by adding a `load_bearing[]` array to skill.yaml with pattern strings, provenance stories, evidence paths, and severity levels. The patterns are:

1. **ABSOLUTE DELEGATION RULE** (error) - Core orchestrator boundary
2. **Filter before presenting** (warn) - Anti-option-theater
3. **Surface decision prerequisites** (warn) - Context before choice
4. **Pressure Over Compensation** (error) - System learning over human memory
5. **Mode Declaration Protocol** (warn) - Frame collapse visibility

---

## Structured Uncertainty

**What's tested:**

- ✅ All 5 patterns present in SKILL.md.template (verified: grep search)
- ✅ skillc check validates load-bearing patterns (verified: ran check command)
- ✅ YAML syntax valid (verified: skillc parsed without error)

**What's untested:**

- ⚠️ skillc deploy propagation (not run - requires separate step)
- ⚠️ Pattern detection is case-sensitive (assumed based on skillc implementation)
- ⚠️ Behavior when patterns are actually removed (would need to test by removing)

**What would change this:**

- If skillc check doesn't block deploy for severity:error patterns, protection is incomplete
- If patterns are commonly reworded during edits, string matching becomes fragile

---

## References

**Files Modified:**
- `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/skill.yaml` - Added load_bearing array

**Files Examined:**
- `.kb/decisions/2026-01-08-load-bearing-guidance-data-model.md` - Data model specification
- `~/.kb/principles.md` - Provenance principle context
- `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md` - Mode Declaration Protocol evidence

**Commands Run:**
```bash
# Verify patterns exist in template
grep "ABSOLUTE DELEGATION RULE" SKILL.md.template
grep "Filter before presenting" SKILL.md.template
grep "Surface decision prerequisites" SKILL.md.template
grep "Pressure Over Compensation" SKILL.md.template
grep "Mode Declaration Protocol" SKILL.md.template

# Validate load-bearing check
skillc check skills/src/meta/orchestrator/.skillc/
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-08-load-bearing-guidance-data-model.md` - Defines the data model used
- **Epic:** orch-go-lv3yx - Parent epic for load-bearing guidance protection
- **Investigation:** `.kb/investigations/2026-01-14-inv-feature-register-friction-guidance-links.md` - skillc implementation

---

## Investigation History

**2026-01-14 21:17:** Investigation started
- Initial question: How to tag 5 hard-won patterns with provenance
- Context: Part of orch-go-lv3yx epic to protect load-bearing guidance

**2026-01-14 21:25:** Data model confirmed
- Found existing decision specifying load_bearing array format
- No design needed - implementation only

**2026-01-14 21:30:** Investigation completed
- Status: Complete
- Key outcome: All 5 patterns registered in skill.yaml with provenance
