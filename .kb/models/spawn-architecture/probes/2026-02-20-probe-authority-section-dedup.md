# Probe: Authority Section Duplication Between Spawn Template and Worker-Base

**Model:** spawn-architecture
**Date:** 2026-02-20
**Status:** Complete

---

## Question

Is the AUTHORITY section duplicated between the spawn template (`pkg/spawn/context.go`) and the worker-base skill, causing ~200 tokens of redundant content per prompt?

---

## What I Tested

1. Searched for AUTHORITY content in spawn template:
```bash
grep -n "AUTHORITY" pkg/spawn/context.go
# Line 233: AUTHORITY:
```

2. Compared spawn template authority section (lines 233-270) with worker-base skill authority section:
- Spawn template: `pkg/spawn/context.go` lines 233-270
- Worker-base source: `~/orch-knowledge/skills/src/shared/worker-base/.skillc/authority.md`
- Deployed skill: `~/.claude/skills/shared/worker-base/SKILL.md` lines 37-55

3. Verified "Surface Before Circumvent" section is unique to spawn template:
```bash
grep -r "Surface Before Circumvent" ~/orch-knowledge/skills/src/shared/worker-base
# No matches
grep -r "Surface Before Circumvent" ~/.claude/skills/shared/worker-base  
# No matches
```

---

## What I Observed

**Duplication confirmed.** The following content appears in BOTH locations:

### Spawn Template (`pkg/spawn/context.go:233-249`)
```
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation...
```

### Worker-Base Skill (`authority.md:3-18`)
```
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation...
```

**Content that is UNIQUE to spawn template (lines 251-270):**
1. `**Full criteria:** See .kb/guides/decision-authority.md...` (pointer to guide)
2. **Surface Before Circumvent** section (~18 lines)

**Token estimate:** ~175 tokens duplicated (authority core), ~200 tokens total for AUTHORITY section.

---

## Model Impact

- [x] **Confirms** invariant: The spawn architecture has accumulated content that should live in one canonical location (this confirms the "structural drift" finding from prior probes)
- [ ] **Contradicts** invariant: N/A
- [x] **Extends** model with: The fix is not simple removal — the "Surface Before Circumvent" content is spawn-template-specific and must be preserved. Fix options:
  1. Move "Surface Before Circumvent" to worker-base, then remove entire AUTHORITY section from spawn template
  2. Keep only unique content (Full criteria pointer + Surface Before Circumvent) in spawn template, remove duplicated core authority
  3. Create new spawn template conditional that skips AUTHORITY when skill has worker-base dependency

---

## Notes

**Recommended fix (Option 2):** Remove the duplicated core authority delegation from spawn template, keep only:
- "Full criteria" pointer to `.kb/guides/decision-authority.md`  
- "Surface Before Circumvent" section (unique to spawn context)

This preserves unique spawn-template content while eliminating duplication. The worker-base skill is already injected for all worker skills via dependency inheritance.

**Risk:** Non-worker skills (orchestrator, design-session) don't depend on worker-base. Need to verify they still get authority guidance through their own skill paths or through explicit AUTHORITY section. However, these are policy/meta skills that may not need the same worker authority rules.

**Token savings:** ~175 tokens per spawn (the duplicated core authority portion).

---

## Fix Applied

Removed duplicated authority delegation content from spawn template (`pkg/spawn/context.go`). The AUTHORITY section now reads:

```
AUTHORITY:
Authority delegation rules are provided via skill guidance (worker-base skill).
**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
[... rest of unique content preserved ...]
```

**Verification:**
- `go test ./pkg/spawn/... -run "Context"` — All 60+ context-related tests pass
- Failing tests in resolve_test.go are pre-existing and unrelated to this change
