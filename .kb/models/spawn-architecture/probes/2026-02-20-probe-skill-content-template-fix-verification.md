# Probe: Skill Content Template Fix Verification

**Date:** 2026-02-20
**Model:** spawn-architecture
**Status:** Complete
**Claim Being Tested:** ProcessSkillContentTemplate correctly processes all template variables in real-world skill content before injection into SPAWN_CONTEXT.md

---

## Question

Does the fix in commits aae850fe7 and 143258fc0 correctly resolve the bug where skill content was injected as raw markdown with literal `{{.BeadsID}}` and unresolved tier conditionals?

---

## What I Tested

### Test 1: Bug reproduction in existing workspaces

Counted literal `{{.BeadsID}}` in SPAWN_CONTEXT.md files spawned BEFORE the fix:

```bash
grep -c '{{\.BeadsID}}' .orch/workspace/og-arch-process-skill-content-20feb-0721/SPAWN_CONTEXT.md
# Result: 15

grep -c '{{\.BeadsID}}' .orch/workspace/og-arch-process-skill-content-20feb-16f8/SPAWN_CONTEXT.md
# Result: 15
```

### Test 2: ProcessSkillContentTemplate on real worker-base SKILL.md

Loaded actual `~/.claude/skills/shared/worker-base/SKILL.md` (14,600 bytes, 13 `{{.BeadsID}}` references) and processed it:

```bash
go run /tmp/test_real_skill.go
# Loaded 14600 bytes of skill content
# {{.BeadsID}} count before: 13
# Template parsed successfully!
# {{.BeadsID}} count after: 0
# SUCCESS: All template variables processed correctly!
# SUCCESS: Tier conditionals processed
```

### Test 3: Full test suite

```bash
go test ./pkg/spawn/ -count=1
# PASS - all tests pass including:
# - TestProcessSkillContentTemplate (6 subtests)
# - TestGenerateContext_ProcessesSkillContentTemplates (2 subtests)
# - TestGenerateContext_RealWorldSkillContentTemplateProcessing (2 subtests - new)
```

### Test 4: Build verification

```bash
go build ./cmd/orch/  # Success
go vet ./cmd/orch/    # Clean
go vet ./pkg/spawn/   # Clean
```

---

## What I Observed

1. **Bug confirmed in pre-fix workspaces**: 15 literal `{{.BeadsID}}` per SPAWN_CONTEXT.md
2. **Fix works correctly**: ProcessSkillContentTemplate processes ALL template variables in real-world skill content (14,600 bytes, 13 references → 0 remaining)
3. **All three spawn paths covered**: context.go (workers), orchestrator_context.go, meta_orchestrator_context.go all call ProcessSkillContentTemplate
4. **Fail-open behavior verified**: Malformed templates and undefined fields return original content (tested)
5. **No regressions**: Full test suite passes

---

## Model Impact

**Confirms model's new invariant (added by prior probe):**

> **Invariant: Skill content must be processed through template engine**
> Skill content containing Go template variables must be processed through `ProcessSkillContentTemplate` before injection into SPAWN_CONTEXT.md.

**Extends with verification detail:**

- The fix correctly handles: `{{.BeadsID}}` substitution, `{{if eq .Tier "light"}}` conditionals, nested template variables within conditionals
- Processing order is critical: strip beads instructions (if NoTrack) → process templates → inject into outer template
- The `skillContentData` struct intentionally exposes only `BeadsID` and `Tier` (minimal surface area). Adding new template variables requires updating this struct.
