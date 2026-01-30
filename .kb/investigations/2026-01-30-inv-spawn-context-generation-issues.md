# Investigation: SPAWN_CONTEXT Generation Issues

**Date:** 2026-01-30
**Status:** Complete
**Issue:** orch-go-21072

## TLDR

Identified 4 systemic issues in SPAWN_CONTEXT generation affecting specs-platform worker spawns: (1) kb context runs from CWD instead of projectDir, pulling wrong project's context, (2) feature-impl SKILL.md contains duplicate content due to skillc append bug, (3) TEMPLATE.md included in kb context with unrendered placeholders, (4) contradictory bd close guidance between skill and template.

---

## Problem Statement

SPAWN_CONTEXT.md for specs-platform workers contained multiple issues that confused agents:
- Wrong project kb context (orch-go paths instead of specs-platform)
- Contradictory bd close guidance
- Duplicate skill content (feature-impl twice)
- Template placeholder ({Title}) not rendered

Source: `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/specs-platform/.orch/workspace/sp-feat-sheet-export-one-30jan-d11f/SPAWN_CONTEXT.md`

---

## Evidence Gathering

### Issue 1: Wrong kb Context Project

**Finding:** kb context queries run from current working directory (CWD), not from projectDir.

**Evidence:**
- `pkg/spawn/kbcontext.go:160-165`: `exec.CommandContext` creates command without setting `cmd.Dir`
- When spawning from orch-go with `--workdir /path/to/specs-platform`, kb context queries orch-go's .kb
- Specs-platform SPAWN_CONTEXT shows only orch-go models/guides/investigations

**Test Performed:**
```bash
# Verified kb context behavior
cd /Users/dylanconlin/Documents/personal/orch-go
kb context "spawn"  # Returns orch-go results

cd /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/specs-platform  
kb context "spawn"  # Returns specs-platform results
```

**Root Cause:** `runKBContextQuery()` doesn't set working directory on exec.Command. kb CLI defaults to CWD for local search.

---

### Issue 2: Duplicate Skill Content

**Finding:** feature-impl SKILL.md contains two complete copies of the skill with different checksums.

**Evidence:**
- `~/.claude/skills/worker/feature-impl/SKILL.md` is 497 lines
- Contains TWO frontmatter blocks:
  - Lines 1-7: Checksum `08ba58a7f0da` (compiled 2026-01-30 11:27:22)
  - Lines 23-29: Checksum `047ddb2689b3` (compiled 2026-01-07 14:41:54)
- Each frontmatter followed by complete skill body
- Result: ~500 lines of duplicate content injected into SPAWN_CONTEXT

**Test Performed:**
```bash
head -50 ~/.claude/skills/worker/feature-impl/SKILL.md
# Shows two distinct AUTO-GENERATED headers with different checksums
```

**Root Cause:** skillc deployment is appending content instead of replacing. Multiple `skillc deploy` runs accumulated duplicate content.

---

### Issue 3: Template Placeholder Not Rendered

**Finding:** `.kb/models/TEMPLATE.md` file included in kb context with `{Title}` placeholder unrendered.

**Evidence:**
- File exists: `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/TEMPLATE.md` (1516 bytes, created Jan 15)
- Content starts with: `# Model: {Title}`
- kb context includes it in Models section: `- {Title}\n  - See: .../TEMPLATE.md`
- This is a template for creating new models, not an actual model

**Test Performed:**
```bash
ls -la /Users/dylanconlin/Documents/personal/orch-go/.kb/models/TEMPLATE.md
head -20 /Users/dylanconlin/Documents/personal/orch-go/.kb/models/TEMPLATE.md
# Confirmed template file exists with placeholder
```

**Root Cause:** kb context scans all files in `.kb/models/` without filtering template files. Template placeholders aren't rendered because kb doesn't distinguish templates from content.

---

### Issue 4: Contradictory bd close Guidance

**Finding:** feature-impl skill says "run bd close" but SPAWN_CONTEXT template says "NEVER run bd close".

**Evidence:**
- `~/.claude/skills/worker/feature-impl/SKILL.md:472`: `1. bd close <beads-id> --reason "summary"`
- `pkg/spawn/context.go:285-287`: `⛔ **NEVER run bd close** - Only the orchestrator closes issues`
- Worker sees both instructions, conflicting guidance

**Test Performed:**
```bash
grep -n "bd close" ~/.claude/skills/worker/feature-impl/SKILL.md
# Line 472: Shows bd close in completion criteria
```

**Root Cause:** Skill content is stale. The "NEVER run bd close" guidance was added to worker-base and SPAWN_CONTEXT template, but old feature-impl skill still has legacy "bd close" instruction in completion criteria.

---

## Root Cause Analysis

### Issue 1: kb Context Working Directory
- **Immediate cause:** `exec.Command` doesn't set `cmd.Dir = projectDir`
- **Contributing factor:** kb context local search relies on CWD
- **Impact:** Cross-project spawns get wrong context, confuse agents

### Issue 2: Duplicate Skill Content
- **Immediate cause:** skillc deployment appending instead of replacing
- **Contributing factor:** No atomic replace or content deduplication
- **Impact:** ~500 lines of bloat per spawn, duplicate/contradictory instructions

### Issue 3: Template Files in kb Context
- **Immediate cause:** kb context scans all .kb files without filtering
- **Contributing factor:** No naming convention to exclude templates (e.g., `*.template.md`)
- **Impact:** Unrendered placeholders confuse agents, look like bugs

### Issue 4: Stale Skill Content
- **Immediate cause:** feature-impl skill not updated when worker-base changed
- **Contributing factor:** No dependency staleness detection
- **Impact:** Contradictory guidance, agents unsure which to follow

---

## Recommendations

### Fix 1: Set kb Context Working Directory
**File:** `pkg/spawn/kbcontext.go:160-165`

```go
func runKBContextQuery(query string, global bool, workdir string) (*KBContextResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if global {
		cmd = exec.CommandContext(ctx, "kb", "context", "--global", query)
	} else {
		cmd = exec.CommandContext(ctx, "kb", "context", query)
	}
	
	// Set working directory for local kb context queries
	if workdir != "" {
		cmd.Dir = workdir
	}

	output, err := cmd.Output()
	// ... rest unchanged
}
```

**Update callers to pass projectDir through the chain:**
- `RunKBContextCheckWithDomain(query, domain, projectDir)`
- `runKBContextQuery(query, global, projectDir)`

---

### Fix 2: Atomic skillc Deployment
**Issue:** skillc should replace, not append

**Option A:** Fix in skillc (preferred if source available)
- Atomic write: Write to temp file, rename over target
- Verify no duplicate frontmatter before write

**Option B:** Workaround in orch-go
- Pre-deployment validation: Check for duplicate checksums
- Post-load deduplication: Strip duplicate frontmatter blocks

**Immediate action:**
```bash
# Manual fix for current duplication
cd ~/.claude/skills/worker/feature-impl
# Extract only first occurrence (up to second frontmatter)
head -496 SKILL.md > SKILL.md.tmp && mv SKILL.md.tmp SKILL.md
```

---

### Fix 3: Exclude Template Files from kb Context
**Option A:** Naming convention (preferred)
- Rename: `TEMPLATE.md` → `_TEMPLATE.md` or `TEMPLATE.md.template`
- Update kb context to skip files matching `_*.md` or `*.template.md`

**Option B:** Filter in kb context
- Add `--exclude-pattern` flag to kb context
- Default: exclude `TEMPLATE.md`, `*.template.md`, `_*.md`

**Immediate action:**
```bash
mv ~/.kb/models/TEMPLATE.md ~/.kb/models/_MODEL_TEMPLATE.md
```

---

### Fix 4: Update Stale Skill Content
**Immediate fix:**
```bash
# Remove legacy bd close instruction from feature-impl
# Line 472: "1. `bd close <beads-id>...`"
# Replace with: "1. Call /exit to close agent session"
```

**Systemic fix:**
- Add skill dependency staleness check
- When worker-base changes, flag dependent skills for review
- Or: Auto-propagate completion criteria from worker-base

---

## Validation

After implementing fixes, verify:

1. **kb context working directory:**
   ```bash
   cd /some/other/project
   orch spawn --workdir /path/to/specs-platform feature-impl "test"
   # Check SPAWN_CONTEXT.md has specs-platform kb context, not current dir
   ```

2. **No duplicate skills:**
   ```bash
   grep -c "^---$" ~/.claude/skills/worker/feature-impl/SKILL.md
   # Should show 2 (one frontmatter block), not 4+
   ```

3. **No template placeholders:**
   ```bash
   orch spawn feature-impl "test task" --skip-artifact-check
   grep "{Title}" .orch/workspace/*/SPAWN_CONTEXT.md
   # Should return no matches
   ```

4. **No bd close contradiction:**
   ```bash
   grep "bd close" .orch/workspace/*/SPAWN_CONTEXT.md
   # Should only appear in "NEVER run" warnings, not in instructions
   ```

---

## Related Issues

- orch-go-21061: features.json reference (deprecated pattern) - separate issue
- Beads ID from window name matching - related to cross-project spawn tracking

---

## Session Metadata

**Files Analyzed:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/specs-platform/.orch/workspace/sp-feat-sheet-export-one-30jan-d11f/SPAWN_CONTEXT.md`
- `pkg/spawn/kbcontext.go`
- `pkg/spawn/context.go`
- `~/.claude/skills/worker/feature-impl/SKILL.md`
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/TEMPLATE.md`

**Investigation Duration:** ~1 hour
**Testing Approach:** Code inspection + kb context behavior verification + file analysis
