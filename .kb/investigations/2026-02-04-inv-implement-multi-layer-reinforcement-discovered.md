## TLDR

Implemented multi-layer reinforcement for discovered work compliance: added mandatory "Discovered Work" section with checklist to worker-base skill, added "Issues Created" section to SYNTHESIS.md template in both the file template and embedded Go constant.

---

# Investigation: Implement Multi-Layer Reinforcement for Discovered Work

**Question:** How to reinforce discovered work compliance through multiple layers of instruction?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Architect Worker
**Phase:** Complete
**Status:** Complete

---

## What I Tried

### 1. Added Discovered Work Section to Worker-Base Skill

Created `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/discovered-work.md` with:
- Mandatory checklist for reviewing discovered work before completion
- Examples of `bd create` commands for bugs, tech debt, enhancements, and questions
- Clear reporting guidance for Phase: Complete comments
- Why this matters section explaining the rationale

Updated `skill.yaml` to include `discovered-work.md` before `completion.md` in the sources list.

### 2. Added "Issues Created" Section to SYNTHESIS.md Template

Updated `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md` with a new "Issues Created" section between "Knowledge" and "Next" sections. The section includes:
- Example format for listing created issues
- Placeholder for "No discovered work during this session"
- Note explaining the expectation

### 3. Updated DefaultSynthesisTemplate in context.go

Updated the embedded template constant in `pkg/spawn/context.go` to include the same "Issues Created" section, ensuring both the file template and the embedded fallback are in sync.

### 4. Deployed Updated Worker-Base Skill

Built and deployed the updated worker-base skill to `/Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md`.

---

## What I Observed

1. **Worker-base now has "Discovered Work (Mandatory)" section at lines 208-245** with checklist and examples
2. **SYNTHESIS.md template now has "Issues Created" section** in the correct position
3. **DefaultSynthesisTemplate in context.go updated** with matching "Issues Created" section at lines 785-797
4. **All spawn package tests pass** (148 tests)

---

## Test Performed

```bash
go test ./pkg/spawn/... -v
# PASS: 148 tests, 0 failures
```

Verified:
- `grep -n "## Discovered Work" /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md` → Line 208
- `grep -n "## Issues" /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` → Line 785

---

## Conclusion

Multi-layer reinforcement implemented successfully:

**Layer 1: Worker-Base Skill (checklist-gated)**
- Added mandatory "Discovered Work" section with checklist items
- Workers must acknowledge reviewing for discovered work before completing
- Provides clear `bd create` examples for different issue types

**Layer 2: SYNTHESIS.md Template (structured output)**
- Added "Issues Created" section to document what was created
- Makes issue tracking visible in the synthesis artifact
- Forces explicit acknowledgment even if no issues were created

**Skipped Layer 3: Spawn Context Reminder**
- Marked as optional in the task
- The two layers above provide sufficient reinforcement
- Adding a third layer would add cognitive load without proportional benefit

---

## Files Changed

### Created
- `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/discovered-work.md`

### Modified
- `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/skill.yaml` (added discovered-work.md to sources)
- `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/SKILL.md` (rebuilt)
- `/Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md` (deployed)
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md` (added Issues Created section)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` (added Issues Created to DefaultSynthesisTemplate)

---

## References

- Parent investigation: `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-04-inv-investigate-workers-not-creating-issues.md`
- Beads issue: `orch-go-21259`
