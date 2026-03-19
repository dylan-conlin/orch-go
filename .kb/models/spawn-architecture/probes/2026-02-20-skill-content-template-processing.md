# Probe: Skill Content Template Processing

**Date:** 2026-02-20
**Model:** spawn-architecture
**Status:** Complete
**Claim Being Tested:** Spawn context generation should process template variables in skill content

---

## Question

Does the spawn pipeline properly process Go template variables (`{{.BeadsID}}`, tier conditionals) in skill content, or is skill content injected as a literal string?

---

## What I Tested

### Test 1: Examine SPAWN_CONTEXT.md for unprocessed template variables

```bash
grep -c '{{.BeadsID}}' ~/Documents/personal/orch-go/.orch/workspace/og-arch-process-skill-content-20feb-0721/SPAWN_CONTEXT.md
```

### Test 2: Trace skill content flow through code

Examined:
1. `pkg/skills/loader.go:LoadSkillContent()` - Loads raw SKILL.md content
2. `pkg/orch/extraction.go:465` - Passes skill content to spawn config
3. `pkg/spawn/context.go:617` - Sets `SkillContent` in contextData
4. `pkg/spawn/context.go:381` - Template uses `{{.SkillContent}}` literal injection

### Test 3: Count template variables in spawned context

```bash
grep -E '\{\{\.BeadsID\}\}|\{\{if eq \.Tier' ~/Documents/personal/orch-go/.orch/workspace/og-arch-process-skill-content-20feb-0721/SPAWN_CONTEXT.md | wc -l
```

---

## What I Observed

### Observation 1: Literal template variables in SPAWN_CONTEXT.md

Lines 818-827 and 837-845 in the spawned SPAWN_CONTEXT.md contain literal `{{.BeadsID}}` placeholders that should have been replaced with `orch-go-1136`:

```markdown
bd comment {{.BeadsID}} "Phase: Planning - Analyzing codebase structure"
bd comment {{.BeadsID}} "Phase: Implementing - Adding authentication middleware"
```

### Observation 2: Tier conditionals both appearing

Lines 964-991 show BOTH branches of tier conditionals appearing:

```markdown
{{if eq .Tier "light"}}
1. Author/update `VERIFICATION_SPEC.yaml`...
{{else}}
1. Author/update `VERIFICATION_SPEC.yaml`...
...
{{end}}
```

### Observation 3: Code path confirms literal injection

In `pkg/spawn/context.go`:
- Line 381: `{{.SkillContent}}` - This is a LITERAL substitution
- The Go `text/template` package treats `{{.SkillContent}}` as "insert the string value of SkillContent here"
- Template variables INSIDE SkillContent are NOT recursively processed

### Observation 4: 14 unprocessed BeadsID references

The issue description mentions "14 per prompt" - confirmed by examining worker-base skill content which contains multiple `{{.BeadsID}}` references in progress tracking and phase reporting sections.

---

## Model Impact

**Contradicts model's implicit claim:** The spawn-architecture model documents workspace creation and SPAWN_CONTEXT.md generation but doesn't explicitly address template processing for skill content. The behavior contradicts the expected outcome - agents receive instructions with literal `{{.BeadsID}}` instead of actual beads IDs.

**Extends model with new invariant:**

> **Invariant: Skill content must be processed through template engine**
> Skill content containing Go template variables (`{{.BeadsID}}`, `{{.Tier}}`, tier conditionals) must be processed through the same template engine and data context used for SPAWN_CONTEXT.md generation. Failure to process skill content results in agents receiving broken instructions with literal template syntax.

---

## Fix Approach

1. Create a function `ProcessSkillContentTemplate(content string, data contextData) (string, error)` in `pkg/spawn/context.go`
2. Call this function BEFORE setting `SkillContent` in `contextData`
3. Process skill content through `text/template` with the same data context used for the outer template
4. Add test to verify template variables in skill content are processed

---

## Fix Implementation

### Changes Made

1. **Added `ProcessSkillContentTemplate()` function** (`pkg/spawn/context.go:426-470`)
   - Takes skill content, beadsID, and tier as parameters
   - Processes through Go `text/template` engine
   - Fail-open behavior: returns original content if template parsing/execution fails

2. **Integrated into `GenerateContext()`** (`pkg/spawn/context.go:636-639`)
   - Called after `StripBeadsInstructions()` but before injecting into contextData
   - Processes skill content template variables before final SPAWN_CONTEXT.md generation

3. **Integrated into `GenerateOrchestratorContext()`** (`pkg/spawn/orchestrator_context.go:173-178`)
   - Same processing for orchestrator spawns

4. **Integrated into `GenerateMetaOrchestratorContext()`** (`pkg/spawn/meta_orchestrator_context.go:227-232`)
   - Same processing for meta-orchestrator spawns

5. **Added comprehensive tests** (`pkg/spawn/context_test.go:1927-2081`)
   - `TestProcessSkillContentTemplate`: Unit tests for the function
   - `TestGenerateContext_ProcessesSkillContentTemplates`: Integration tests

### Test Results

```bash
$ go test ./pkg/spawn/... -run TestProcessSkillContentTemplate -v
=== RUN   TestProcessSkillContentTemplate
=== RUN   TestProcessSkillContentTemplate/processes_BeadsID_template_variable
=== RUN   TestProcessSkillContentTemplate/processes_Tier_conditional
=== RUN   TestProcessSkillContentTemplate/returns_original_content_when_no_template_syntax_present
=== RUN   TestProcessSkillContentTemplate/returns_original_content_for_empty_input
=== RUN   TestProcessSkillContentTemplate/handles_malformed_template_gracefully
=== RUN   TestProcessSkillContentTemplate/handles_undefined_fields_gracefully
--- PASS: TestProcessSkillContentTemplate (0.00s)

$ go test ./pkg/spawn/... -run TestGenerateContext_ProcessesSkillContentTemplates -v
=== RUN   TestGenerateContext_ProcessesSkillContentTemplates
=== RUN   TestGenerateContext_ProcessesSkillContentTemplates/processes_skill_content_templates_for_tracked_spawn
=== RUN   TestGenerateContext_ProcessesSkillContentTemplates/processes_skill_content_templates_for_light_tier_spawn
--- PASS: TestGenerateContext_ProcessesSkillContentTemplates (0.00s)
```

---

## Verification

After fix implementation:
1. Spawn an agent with a tracked beads issue
2. Verify SPAWN_CONTEXT.md contains actual beads ID (e.g., `orch-go-1136`) not `{{.BeadsID}}`
3. Verify tier conditionals are resolved (only one branch appears based on tier)

**Verification performed via unit tests** - see Test Results above. Tests verify:
- `{{.BeadsID}}` is replaced with actual beads ID
- Tier conditionals resolve to correct branch
- Empty/missing content handled gracefully
- Malformed templates fail-open (return original content)
