<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The `orch patterns` command was undocumented in orchestrator skill and reference docs.

**Evidence:** Searched skill template and orch-commands-reference.md - no mention of patterns command; code exists at cmd/orch/patterns.go:22-43.

**Knowledge:** Command surfaces behavioral patterns (retry, persistent_failure, empty_context, recurring_gap, futile_action) with severity levels (critical/warning/info).

**Next:** Close - documentation added to both orchestrator skill and orch-commands-reference.md.

---

# Investigation: Document Orch Patterns Command Orchestrator

**Question:** Where should `orch patterns` command be documented in orchestrator skill and reference docs?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: orch patterns command exists with rich functionality

**Evidence:** The command at cmd/orch/patterns.go:22-43 provides:
- Pattern detection for: retry, persistent_failure, empty_context, recurring_gap, context_drift, futile_action
- Severity levels: critical, warning, info
- Flags: --json for scripting, --verbose for info-level patterns
- Integration with verify, spawn, and action packages for pattern collection

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/patterns.go:22-48

**Significance:** Command provides orchestrator awareness of systemic issues that prevent blind respawning

---

### Finding 2: Documentation was missing from orchestrator skill

**Evidence:** Searched SKILL.md.template - no mention of `orch patterns` in "Orch Commands (Quick Reference)" section

**Source:** /Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template:1554-1610

**Significance:** Orchestrators wouldn't know about this awareness tool without documentation

---

### Finding 3: Documentation was missing from orch-commands-reference.md

**Evidence:** Searched orch-commands-reference.md - no section for patterns or behavioral awareness

**Source:** /Users/dylanconlin/orch-knowledge/docs/orch-commands-reference.md

**Significance:** Reference doc is the authoritative source for all orch commands

---

## Synthesis

**Key Insights:**

1. **Command provides orchestrator value** - Surfaces patterns that help avoid retry loops and blind respawning

2. **Logical placement found** - Added after "Learning Loop" section since patterns and learn are complementary tools

3. **Two documentation targets** - Both skill (for orchestrator context) and reference doc (for complete reference)

**Answer to Investigation Question:**

Documentation was added to:
1. Orchestrator skill template (~line 1598) - after Learning Loop section
2. orch-commands-reference.md - new "Behavioral Patterns" section after Strategic Alignment

---

## Structured Uncertainty

**What's tested:**

- ✅ Documentation added to SKILL.md.template (verified: file edited successfully)
- ✅ Documentation added to orch-commands-reference.md (verified: file edited successfully)
- ✅ Pattern types match source code (verified: cmd/orch/patterns.go:54-72)

**What's untested:**

- ⚠️ Skill rebuild (`orch build skills` not run - requires skillc)
- ⚠️ Command output matches documentation (not run interactively)

**What would change this:**

- Pattern types in source code change
- New patterns added
- Flag options change

---

## References

**Files Examined:**
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/patterns.go - Source implementation
- /Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template - Skill template
- /Users/dylanconlin/orch-knowledge/docs/orch-commands-reference.md - Reference docs

**Commands Run:**
```bash
# Check patterns command exists
glob **/patterns*.go
```

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: Where to document orch patterns command
- Context: New command existed but was undocumented

**2025-12-27:** Documentation added
- Added to orchestrator skill template after Learning Loop section
- Added new Behavioral Patterns section to orch-commands-reference.md

**2025-12-27:** Investigation completed
- Status: Complete
- Key outcome: orch patterns documented in both locations
