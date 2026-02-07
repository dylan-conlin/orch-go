## Summary (D.E.K.N.)

**Delta:** Added `outputs.required` to 4 artifact-producing skills: codebase-audit, architect, research, reliability-testing.

**Evidence:** All verify tests pass; grep confirms outputs.required patterns now exist in all 4 skill.yaml files.

**Knowledge:** Only skills with `required: true` deliverables need `outputs.required`; skills with optional deliverables (feature-impl, systematic-debugging) don't.

**Next:** Close - implementation complete, skills now have executable verification constraints.

---

# Investigation: Add Outputs Required to Artifact-Producing Skills

**Question:** Which skills produce artifacts and need `outputs.required` for orch complete verification?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Only investigation skill had outputs.required

**Evidence:** Grep for `outputs:` in ~/.claude/skills found only one match: worker/investigation/.skillc/skill.yaml

**Source:** `grep -r "outputs:" ~/.claude/skills --include="*.yaml"`

**Significance:** Other artifact-producing skills were missing this verification mechanism.

---

### Finding 2: Four skills have required file deliverables

**Evidence:**
- `codebase-audit`: `path: "{project}/.kb/investigations/{date}-audit-{slug}.md"` with `required: true`
- `architect`: `path: ".kb/investigations/{date}-design-{slug}.md"` with `required: true`  
- `research`: `path: "{project}/.kb/investigations/{date}-research-{slug}.md"` with `required: true`
- `reliability-testing`: `path: "{project}/.kb/investigations/{date}-reliability-*.md"` with `required: true`

**Source:** skill.yaml files in ~/.claude/skills/worker/{skill}/.skillc/

**Significance:** These skills need `outputs.required` for orch complete verification.

---

### Finding 3: Skills with optional deliverables don't need outputs.required

**Evidence:**
- `systematic-debugging`: investigation file has `required: false`
- `feature-impl`: all deliverables have `required: false` (phase-dependent)
- `design-session`: produces either epic OR investigation OR decision (all optional)
- `issue-creation`: produces beads issues, not files

**Source:** skill.yaml deliverables sections

**Significance:** These skills should NOT have outputs.required since their file outputs are conditional.

---

## Implementation

Added `outputs.required` to 4 skills with patterns matching their deliverables:

1. **codebase-audit**: `.kb/investigations/{date}-audit-*.md`
2. **architect**: `.kb/investigations/{date}-design-*.md`
3. **research**: `.kb/investigations/{date}-research-*.md`
4. **reliability-testing**: `.kb/investigations/{date}-reliability-*.md`

All orch-go verify tests pass after the changes.

---

## References

**Files Modified:**
- `~/.claude/skills/worker/codebase-audit/.skillc/skill.yaml`
- `~/.claude/skills/worker/architect/.skillc/skill.yaml`
- `~/.claude/skills/worker/research/.skillc/skill.yaml`
- `~/.claude/skills/worker/reliability-testing/.skillc/skill.yaml`

**Commands Run:**
```bash
# Verify outputs.required patterns
grep -A3 "outputs:" ~/.claude/skills/worker/{codebase-audit,architect,research,reliability-testing}/.skillc/skill.yaml

# Verify tests still pass
go test -v ./pkg/verify/...
```
