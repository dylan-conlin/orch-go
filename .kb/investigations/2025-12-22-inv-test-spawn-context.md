<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawn context generation works correctly, but untracked agents (--no-track) receive fake beads IDs that cause `bd comment` commands to fail.

**Evidence:** Template substituted 11 beads ID occurrences correctly. However, `bd comment orch-go-untracked-1766417753` failed with "issue not found" because the ID is a placeholder, not a real issue.

**Knowledge:** When `--no-track` is used, `determineBeadsID()` in `cmd/orch/main.go:1212-1213` generates `{project}-untracked-{timestamp}` without creating an actual beads issue. The spawn context still instructs agents to use `bd comment` in "FIRST 3 ACTIONS".

**Next:** Consider updating spawn context template to conditionally omit beads instructions for untracked agents, or document the expected failure.

**Confidence:** High (90%) - tested against real spawn, verified template source.

---

# Investigation: Spawn Context Generation

**Question:** Does the spawn context generation work correctly for spawned agents?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: SPAWN_CONTEXT.md file is correctly generated

**Evidence:** 
- File exists at `.orch/workspace/og-inv-test-spawn-context-22dec/SPAWN_CONTEXT.md`
- Contains 419 lines, 15,830 bytes
- Workspace also contains `.session_id` file with OpenCode session ID

**Source:** 
```bash
$ wc -l SPAWN_CONTEXT.md
419

$ ls -la .orch/workspace/og-inv-test-spawn-context-22dec/
.session_id
SPAWN_CONTEXT.md
```

**Significance:** Confirms spawn infrastructure is creating the expected workspace structure.

---

### Finding 2: All critical sections are present

**Evidence:** Key sections verified present:
- Line 1: `TASK:` - task description
- Line 24: `PROJECT_DIR:` - absolute project path
- Line 26: `SESSION SCOPE:` - Medium estimation
- Line 31: `AUTHORITY:` - delegation rules
- Line 49: `DELIVERABLES (REQUIRED):` - required outputs
- Line 77: `BEADS PROGRESS TRACKING` - bd comment workflow
- Line 109: `SKILL GUIDANCE (investigation)` - embedded skill content

**Source:** `rg -n "TASK:|PROJECT_DIR:|AUTHORITY:|DELIVERABLES|SKILL GUIDANCE|SESSION SCOPE:|BEADS PROGRESS" SPAWN_CONTEXT.md`

**Significance:** Template is complete - no missing sections that would confuse agents.

---

### Finding 3: Beads ID is correctly templated throughout

**Evidence:** Found 11 occurrences of `orch-go-untracked-*` beads ID:
- Line 5: First 3 actions comment instruction
- Line 15: Session complete protocol
- Line 56: Investigation path reporting
- Line 79: "You were spawned from beads issue" statement
- Lines 85-87: Example bd comment commands
- Lines 90, 93: Blocker/question examples
- Line 102: Why beads comments explanation
- Line 416: Final step reminder

**Source:** `rg "orch-go-untracked" SPAWN_CONTEXT.md` - 11 matches

**Significance:** BeadsID variable is properly substituted throughout the template, ensuring agents can track progress correctly.

---

### Finding 4: Template source code is well-structured

**Evidence:** Template defined in `pkg/spawn/context.go:14-163` using Go's text/template:
- `contextData` struct holds all template variables (lines 166-179)
- `GenerateContext()` parses and executes template (lines 182-212)
- `WriteContext()` creates workspace and writes file (lines 215-239)
- Additional templates for SYNTHESIS.md and FAILURE_REPORT.md

**Source:** `pkg/spawn/context.go`

**Significance:** Template is maintainable - changes to spawn context just require editing the template string.

---

## Test performed

**Test:** Verified that I (a spawned agent) could read and follow the spawn context by:
1. Reading the SPAWN_CONTEXT.md file from the specified path
2. Verifying all critical sections were present and parseable
3. Following the workflow instructions (bd comment, kb create, investigation file)
4. Cross-referencing with the source template to confirm correctness

**Result:** 
- ✅ Spawn context was readable and complete
- ✅ All template variables were substituted (no `{{.Variable}}` literals)
- ✅ Skill content was embedded (investigation skill, 285 lines)
- ⚠️ Beads issue didn't exist (expected for test spawn) - `bd comment` failed gracefully

---

## Conclusion

The spawn context generation is working correctly. The template produces a complete, well-structured SPAWN_CONTEXT.md that provides agents with:
1. Clear task description
2. Project location verification
3. Authority boundaries (what to decide vs escalate)
4. Required deliverables with paths
5. Beads progress tracking workflow
6. Embedded skill guidance
7. Session completion protocol

**Issue Found:** The beads issue does not exist (ID `orch-go-untracked-1766417753`). This is expected behavior for `--no-track` spawns, but the spawn context still instructs agents to run `bd comment` as their "FIRST 3 ACTIONS", causing a guaranteed failure. This is a UX issue - agents receive conflicting instructions.

**Recommendation:** Update spawn context template to conditionally omit beads tracking sections when `IsTracked=false`, or add a note explaining that untracked agents should skip beads commands.

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled
- [x] NOT DONE claims verified - N/A

**Self-Review Status:** PASSED

---

## References

**Files Examined:**
- `.orch/workspace/og-inv-test-spawn-context-22dec/SPAWN_CONTEXT.md` - Generated spawn context
- `pkg/spawn/context.go` - Template source and generation logic

**Commands Run:**
```bash
# Count lines in spawn context
wc -l SPAWN_CONTEXT.md
# 419

# List workspace contents
ls -la .orch/workspace/og-inv-test-spawn-context-22dec/

# Check key sections present
rg -n "TASK:|PROJECT_DIR:|AUTHORITY:|DELIVERABLES|SKILL GUIDANCE" SPAWN_CONTEXT.md

# Count beads ID occurrences  
rg -c "orch-go-untracked" SPAWN_CONTEXT.md
# 11

# Verify bd comment behavior
bd comment orch-go-untracked-1766417740 "Phase: Planning - test"
# Error: issue not found (expected for test)
```

---

## Investigation History

**2025-12-22 07:36:** Investigation started
- Initial question: Does spawn context generation work correctly?
- Context: Test spawn to validate infrastructure

**2025-12-22 07:38:** Verified spawn context file structure and content

**2025-12-22 07:40:** Examined source template in pkg/spawn/context.go

**2025-12-22 07:42:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Spawn context generation is working as designed
