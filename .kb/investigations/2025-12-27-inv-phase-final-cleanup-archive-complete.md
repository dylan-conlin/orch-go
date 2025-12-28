## Summary (D.E.K.N.)

**Delta:** orch-cli (Python) archived; agentlog should be archived (not installed/unused); skillc should stay separate (actively used for skill compilation).

**Evidence:** agentlog not in PATH despite 30 skill refs; skillc binary exists and .skillc/ structure used in skill authoring.

**Knowledge:** Tool consolidation should focus on actual usage, not just references. Dead references in skills need cleanup.

**Next:** Archive agentlog, install skillc to PATH, update skills to remove agentlog references, update documentation.

---

# Investigation: Phase 4 Final Cleanup

**Question:** Which repos should be archived vs kept for the orch-go consolidation?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Python orch-cli is fully superseded

**Evidence:** orch-go implements all orch-cli functionality (spawn, status, complete, daemon, etc.)

**Source:** /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md:119-150 (Commands section)

**Significance:** Safe to archive. Added deprecation notice pointing to orch-go.

---

### Finding 2: agentlog is not installed or used

**Evidence:** 
- `which agentlog` returns nothing
- 30 skill references exist but are aspirational
- Binary exists in repo at `/Users/dylanconlin/Documents/personal/agentlog/agentlog` but not installed

**Source:** grep of ~/.claude/skills/, which agentlog

**Significance:** Should archive agentlog and clean up skill references. The concept (AI-native error visibility) is good but not implemented.

---

### Finding 3: skillc is actively used

**Evidence:**
- `.skillc/` directories exist in multiple skills (feature-impl, systematic-debugging, writing-skills, orchestrator)
- writing-skills skill documents skillc workflow
- Binary exists but not installed to PATH

**Source:** grep for skillc in skills, ls of skillc/bin/

**Significance:** Keep separate. Install to PATH. Used for skill authoring workflow.

---

## Synthesis

**Key Insights:**

1. **Usage > References** - agentlog has many references but zero actual usage. References need cleanup.

2. **Clear separation** - skillc (skill compilation) and orch (agent orchestration) serve different purposes. Merging would complicate both.

3. **Installation gap** - Both agentlog and skillc binaries exist but aren't in PATH. Need better installation story.

**Answer to Investigation Question:**

Archive: orch-cli (Python), agentlog
Keep: orch-go, orch-knowledge, skillc, beads (external)

This leaves 4 functional repos per success criteria.

---

## Implementation Recommendations

### Recommended Approach ⭐

1. Archive orch-cli ✅ Done
2. Archive agentlog (add deprecation notice)
3. Remove agentlog references from skills
4. Install skillc to ~/bin
5. Update orchestrator skill Tool Ecosystem section
6. Update global CLAUDE.md if needed

---

## References

**Files Examined:**
- /Users/dylanconlin/Documents/personal/orch-cli/README.md
- /Users/dylanconlin/Documents/personal/agentlog/README.md
- /Users/dylanconlin/Documents/personal/skillc/README.md
- ~/.claude/skills/meta/orchestrator/SKILL.md

**Commands Run:**
```bash
which agentlog  # Not found
grep -rn "agentlog" ~/.claude/skills/  # 30 references
grep -rn "skillc" ~/.claude/skills/  # Active usage in writing-skills
```

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: What to archive vs keep for consolidation?
- Context: Phase 4 of orch-go epic

**2025-12-27:** Investigation completed
- Status: Complete
- Key outcome: Archive orch-cli and agentlog; keep skillc separate
