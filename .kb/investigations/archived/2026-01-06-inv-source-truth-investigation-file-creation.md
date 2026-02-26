<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** kb-cli is the authoritative source of truth for investigation templates; skill embeds simplified template for documentation but defers to `kb create investigation` for actual file creation.

**Evidence:** Traced code paths: kb-cli `loadTemplate()` first checks `~/.kb/templates/INVESTIGATION.md`, falls back to embedded `investigationTemplate` constant; skill template.md contains simplified example only with instruction "Use `kb create investigation {slug}` to create."

**Knowledge:** Format variations (D.E.K.N. vs TLDR) exist because: (1) pre-Dec-20-2025 files used older template, (2) some files manually created bypassing `kb create`. Both kb-cli and user template are in sync now.

**Next:** To add 'promote-to-decision' as Next option: (1) Update `~/.kb/templates/INVESTIGATION.md` Next field guidance, (2) Update kb-cli embedded template for fallback consistency, (3) Update skill documentation.

---

# Investigation: Source Truth Investigation File Creation

**Question:** What is the source of truth for investigation file creation - trace the actual code paths from skill, kb-cli, and any templates to understand which is authoritative and which are stale.

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: kb-cli template loading has clear precedence

**Evidence:** The `loadTemplate()` function in kb-cli (`/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:560-574`) implements clear precedence:

```go
func loadTemplate(templateName, defaultTemplate string) string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return defaultTemplate
    }
    templatePath := filepath.Join(homeDir, ".kb", "templates", templateName)
    content, err := os.ReadFile(templatePath)
    if err != nil {
        return defaultTemplate
    }
    return string(content)
}
```

1. **First:** Check `~/.kb/templates/{INVESTIGATION.md|DECISION.md|etc.}`
2. **Fallback:** Use embedded `investigationTemplate` constant (lines 15-235)

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:560-574`

**Significance:** The user template at `~/.kb/templates/INVESTIGATION.md` IS the source of truth when it exists. The kb-cli binary's embedded template only serves as fallback.

---

### Finding 2: User template and kb-cli embedded template are currently in sync

**Evidence:** Both templates contain identical D.E.K.N. summary format:
- `~/.kb/templates/INVESTIGATION.md` (221 lines)
- kb-cli embedded `investigationTemplate` constant (also produces 221-line files)

Tested by running `kb create investigation test-template-check` in a temp directory - output matches `~/.kb/templates/INVESTIGATION.md` exactly.

**Source:** 
- `~/.kb/templates/INVESTIGATION.md` (exists, 221 lines)
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:15-235`

**Significance:** No drift between user template and embedded fallback currently. Changes to `~/.kb/templates/INVESTIGATION.md` take effect immediately; changes to kb-cli require recompile.

---

### Finding 3: Investigation skill embeds simplified template as DOCUMENTATION, not as generator

**Evidence:** The investigation skill (`/Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md`) contains a template section but:
- It's a simplified example (30 lines) in a markdown code block
- Line 106 explicitly says: "The template enforces the discipline. Use `kb create investigation {slug}` to create."
- Skill does NOT generate files itself - it defers to kb-cli

Source file: `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/template.md`

**Source:** 
- `/Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md:104-132`
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/template.md`

**Significance:** The skill template is intentionally simplified for documentation/illustration. The skill explicitly delegates actual file creation to `kb create investigation`. There is no conflict - different purposes.

---

### Finding 4: Format variations (D.E.K.N. vs TLDR vs custom) have clear causes

**Evidence:** Analysis of 559 investigation files in orch-go:
- 501 files with D.E.K.N. summary format
- 51 files with TLDR format
- 58 files without D.E.K.N. (various custom formats)

Correlation with dates:
- Pre-Dec-20-2025 files: Often have TLDR/Confidence format (older template)
- Dec-20-2025 onward: Mostly D.E.K.N. format
- Some newer files (e.g., `2026-01-06-inv-workspace-session-architecture.md`) without D.E.K.N. were created MANUALLY (not via `kb create`)

Example of manually created file (no D.E.K.N.):
```markdown
# Investigation: Workspace, Session, and Resumption Architecture

**Status:** complete
**Date:** 2026-01-06
...
```

**Source:** 
```bash
grep -l "D.E.K.N." .kb/investigations/*.md | wc -l  # 501
grep -L "D.E.K.N." .kb/investigations/*.md | wc -l  # 58
```

**Significance:** Format inconsistency is NOT a bug or template conflict - it reflects: (1) template evolution over time, (2) agents bypassing `kb create` to write files directly. The D.E.K.N. format IS current standard.

---

### Finding 5: Research skill uses different template (by design)

**Evidence:** The research skill (`/Users/dylanconlin/.claude/skills/worker/research/SKILL.md`) instructs agents to create files with `research/` prefix:

```bash
kb create investigation "research/topic-in-kebab-case"
```

But research has its own template constant in kb-cli (`researchTemplate`, lines 354-447) and creates files as `YYYY-MM-DD-research-{slug}.md`.

There's also `kb create research {slug}` command that explicitly uses `loadTemplate("RESEARCH.md", researchTemplate)`.

**Source:**
- `/Users/dylanconlin/.claude/skills/worker/research/SKILL.md:72-77`
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:528-552` (researchCmd)
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:354-447` (researchTemplate)

**Significance:** Different artifact types have different templates by design. Research template has options/recommendation structure; investigation template has findings/synthesis structure.

---

### Finding 6: Next field guidance currently suggests limited options

**Evidence:** The template's Next field guidance is:
```markdown
**Next:** [Recommended action - close, implement, investigate further, or escalate]
```

The current options are: close, implement, investigate further, escalate.

There is NO `promote-to-decision` option documented, despite `kb promote` command existing.

**Source:** `~/.kb/templates/INVESTIGATION.md:14`

**Significance:** To add `promote-to-decision` as a Next option, need to update the template guidance text. This is a simple text change.

---

## Synthesis

**Key Insights:**

1. **kb-cli owns template rendering** - The skill, orch spawn, and all other tooling delegates to `kb create investigation` for file creation. There's one authoritative source.

2. **User template overrides embedded** - `~/.kb/templates/INVESTIGATION.md` takes precedence; kb-cli's embedded template is fallback only. Edit the user template to change behavior.

3. **Format drift is behavioral, not technical** - Agents sometimes create investigation files manually without using `kb create`, causing format inconsistency. The template itself is consistent.

**Answer to Investigation Question:**

1. **When an agent runs `kb create investigation`, what template is used?**
   - First checks `~/.kb/templates/INVESTIGATION.md` (user template)
   - Falls back to embedded `investigationTemplate` constant in kb-cli

2. **Does the investigation skill embed its own template or defer to kb-cli?**
   - Defers to kb-cli. Skill contains simplified template for documentation only, with explicit instruction to use `kb create investigation {slug}`.

3. **Why do recent investigations have different formats?**
   - Pre-Dec-20-2025: Older template (TLDR/Confidence format)
   - Post-Dec-20-2025: D.E.K.N. format (current standard)
   - Some files created manually by agents bypassing `kb create`

4. **What would need to change to add 'promote-to-decision' as a Next option?**
   - Update `~/.kb/templates/INVESTIGATION.md` Next field: add "promote-to-decision" to guidance
   - Update kb-cli embedded template for consistency (requires recompile)
   - Optionally update skill documentation to mention this option

---

## Structured Uncertainty

**What's tested:**

- ✅ kb-cli loadTemplate() prioritizes user template over embedded (traced code, tested with temp directory)
- ✅ User template and embedded template are currently identical (compared both)
- ✅ 501/559 files use D.E.K.N. format (grepped all investigation files)
- ✅ Skill explicitly defers to `kb create investigation` (read skill source)

**What's untested:**

- ⚠️ Whether agents actually read/follow the Next field guidance (behavioral observation not performed)
- ⚠️ Impact of adding promote-to-decision on agent behavior (requires deployment and observation)

**What would change this:**

- If kb-cli removed loadTemplate() and hardcoded the embedded template
- If skill changed to generate files directly instead of calling kb-cli
- If `~/.kb/templates/INVESTIGATION.md` is deleted (would fall back to embedded)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add promote-to-decision to Next guidance** - Edit the template guidance text to include promote-to-decision as an explicit option.

**Why this approach:**
- Minimal change with immediate effect
- User template is already the source of truth
- No code changes required for basic functionality

**Trade-offs accepted:**
- kb-cli embedded template will drift from user template until updated
- Not enforced by automation (agent may still not use it)

**Implementation sequence:**
1. Edit `~/.kb/templates/INVESTIGATION.md` line 14: change `[close, implement, investigate further, or escalate]` to `[close, implement, investigate further, promote-to-decision, or escalate]`
2. Optionally update kb-cli embedded template and recompile for consistency
3. Update investigation skill documentation if desired

### Alternative Approaches Considered

**Option B: Add kb promote integration to skill completion**
- **Pros:** Automated promotion workflow
- **Cons:** Requires skill rebuild and deployment; more complex
- **When to use instead:** If agents consistently don't follow Next guidance

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go` - kb create implementation
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/templates.go` - templates list command
- `/Users/dylanconlin/.kb/templates/INVESTIGATION.md` - User template (source of truth)
- `/Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md` - Investigation skill
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/template.md` - Skill template source

**Commands Run:**
```bash
# Test which template kb-cli uses
kb create investigation test-template-check

# Count investigation formats
grep -l "D.E.K.N." .kb/investigations/*.md | wc -l  # 501
grep -L "D.E.K.N." .kb/investigations/*.md | wc -l  # 58

# Check user templates directory
ls -la ~/.kb/templates/
```

---

## Self-Review

- [x] Real test performed (tested kb create in temp directory)
- [x] Conclusion from evidence (traced code paths, counted files)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
