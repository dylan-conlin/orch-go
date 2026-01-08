<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Templates are split by ownership domain (kb-cli owns knowledge artifact templates, orch-go owns agent lifecycle templates), with canonical sources in Go constants, embeddable files, and .orch/templates/ directory, documented in existing decision `.kb/decisions/2025-12-22-template-ownership-model.md`.

**Evidence:** Found 14+ distinct templates across 4 locations: pkg/spawn/context.go (5 Go constants), .orch/templates/ (3 files), pkg/claudemd/templates/ (4 CLAUDE.md types), kb-cli/cmd/kb/create.go (4 embedded templates), plus ~90 skill component files in orch-knowledge that compile to deployed skills.

**Knowledge:** Template ownership follows the principle "the tool that creates the artifact owns its template" - kb-cli creates .kb/ artifacts, orch-go creates .orch/workspace/ artifacts. Skills are compiled by skillc and deployed to ~/.claude/skills/.

**Next:** Close this investigation - template ownership is well-documented and the system is coherent. Reference this audit for template inventory.

**Promote to Decision:** recommend-no (confirms existing decision, no new architectural choice needed)

---

# Investigation: Comprehensive Template Audit Canonical Sources

**Question:** What templates exist in the orchestration system, where are their canonical sources, what tools/skills use them, and what alternative versions exist?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: orch-go Spawn Templates (Canonical in pkg/spawn/)

**Evidence:** Five main template constants in `pkg/spawn/context.go`:

| Template | Constant Name | Line | Purpose |
|----------|---------------|------|---------|
| SPAWN_CONTEXT.md | `SpawnContextTemplate` | :30 | Worker agent task context |
| SYNTHESIS.md | `DefaultSynthesisTemplate` | :544 | Agent completion summary |
| FAILURE_REPORT.md | `DefaultFailureReportTemplate` | :816 | Agent failure documentation |
| SESSION_HANDOFF.md | `DefaultSessionHandoffTemplate` | :287 (orchestrator_context.go) | Orchestrator session transitions |
| Pre-filled SESSION_HANDOFF | `PreFilledSessionHandoffTemplate` | :355 (orchestrator_context.go) | Pre-populated handoff for progressive fill |

Additional context templates:
- `OrchestratorContextTemplate` in `pkg/spawn/orchestrator_context.go:19` 
- `MetaOrchestratorContextTemplate` in `pkg/spawn/meta_orchestrator_context.go:21`

**Source:** 
- `pkg/spawn/context.go:30-898`
- `pkg/spawn/orchestrator_context.go:19-583`
- `pkg/spawn/meta_orchestrator_context.go:21-410`

**Significance:** These are the canonical sources for spawn-time templates. The Go constants are embedded at compile time. Override mechanism: `.orch/templates/` in project directory.

---

### Finding 2: .orch/templates/ Directory (Project-level Overrides)

**Evidence:** Three template files in `.orch/templates/`:

| File | Size | Last Modified | Source |
|------|------|---------------|--------|
| FAILURE_REPORT.md | 83 lines | Reference template | Matches `DefaultFailureReportTemplate` |
| SESSION_HANDOFF.md | 211 lines | Reference template | Matches `DefaultSessionHandoffTemplate` with expanded sections |
| SYNTHESIS.md | 153 lines | Reference template | Matches `DefaultSynthesisTemplate` with progressive fill guidance |

**Source:** `ls -la .orch/templates/` and file reads

**Significance:** These files serve as:
1. Override templates (project-specific customization)
2. Reference documentation (users can see template structure)
3. Fallback source if Go constants need updating

The `EnsureSynthesisTemplate()` function in context.go:520 copies default template to project if missing.

---

### Finding 3: CLAUDE.md Templates (pkg/claudemd/)

**Evidence:** Four project-type templates in `pkg/claudemd/templates/`:

| Template | Purpose | Key Variables |
|----------|---------|---------------|
| `minimal.md` | Bare-bones CLAUDE.md | `{{.ProjectName}}` |
| `go-cli.md` | Go CLI projects | `{{.ProjectName}}`, Makefile targets, pkg structure |
| `python-cli.md` | Python CLI projects | `{{.ProjectName}}`, pyproject.toml patterns |
| `svelte-app.md` | SvelteKit applications | `{{.ProjectName}}`, `{{.PortWeb}}`, `{{.PortAPI}}` |

**Source:** 
- `pkg/claudemd/claudemd.go:13` (embed directive)
- `pkg/claudemd/templates/*.md`

**Significance:** These templates are embedded via Go's `//go:embed` directive and used by `orch init` to generate project CLAUDE.md files. User override path: `~/.orch/templates/claude/`

---

### Finding 4: kb-cli Knowledge Artifact Templates

**Evidence:** Four templates embedded in `kb-cli/cmd/kb/create.go`:

| Template | Constant | Output Location | Created By |
|----------|----------|-----------------|------------|
| `investigationTemplate` | Line ~17 | `.kb/investigations/` | `kb create investigation` |
| `decisionTemplate` | Line ~230 | `.kb/decisions/` | `kb create decision` |
| `guideTemplate` | (exists) | `.kb/guides/` | `kb create guide` |
| `researchTemplate` | (exists) | `.kb/investigations/` | `kb create research` |

**Source:** `~/Documents/personal/kb-cli/cmd/kb/create.go`

**Significance:** kb-cli owns templates for knowledge artifacts. Override mechanism: `~/.kb/templates/`. This separation ensures orch-go doesn't depend on kb-cli for spawn functionality.

---

### Finding 5: Skill Templates (orch-knowledge)

**Evidence:** ~90 skill component files across skill sources:

```
orch-knowledge/skills/src/
├── meta/
│   ├── meta-orchestrator/.skillc/ (8 files)
│   ├── orchestrator/.skillc/ (reference files)
│   └── writing-skills/.skillc/ (7 files)
├── shared/
│   ├── design-principles/.skillc/ (8 files)
│   ├── delegating-to-team/.skillc/ (7 files)
│   ├── issue-quality/.skillc/ (7 files)
│   └── worker-base/.skillc/ (5 files)
└── worker/
    ├── codebase-audit/.skillc/ (10 files)
    ├── feature-impl/.skillc/phases/ (8 files)
    ├── investigation/.skillc/ (6 files)
    ├── issue-creation/.skillc/ (3 files)
    ├── kb-reflect/.skillc/ (10 files)
    ├── reliability-testing/.skillc/ (4 files)
    └── systematic-debugging/.skillc/ (6 files)
```

**Source:** `glob **/*.skillc/**/*.md` in orch-knowledge

**Significance:** Skills are composed from modular `.skillc/` directories containing:
- `skill.yaml` (metadata, composition order)
- `intro.md`, `workflow.md`, `completion.md` (phase-specific content)
- Phase-specific files (e.g., `phases/implementation-tdd.md`)

Compiled by `skillc deploy` → output to `~/.claude/skills/{category}/{skill}/SKILL.md`

---

### Finding 6: Skill Embedding in Spawn Context

**Evidence:** The `SpawnContextTemplate` includes skill content dynamically:

```go
{{if .SkillContent}}
## SKILL GUIDANCE ({{.SkillName}})

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

{{.SkillContent}}

---
{{end}}
```

**Source:** `pkg/spawn/context.go:216-228`

**Significance:** Skills are embedded into SPAWN_CONTEXT.md at spawn time, not loaded dynamically. This means:
1. Skill content is available immediately (no tool invocation needed)
2. Skill is frozen at spawn time (won't change mid-session)
3. Beads-specific instructions stripped when `--no-track` via `StripBeadsInstructions()`

---

### Finding 7: Template Ownership Decision (Prior Decision)

**Evidence:** Existing decision document `.kb/decisions/2025-12-22-template-ownership-model.md` documents:

| Owner | Templates | Location |
|-------|-----------|----------|
| kb-cli | Investigation, Decision, Guide, Research | `kb-cli/cmd/kb/create.go` |
| orch-go | SPAWN_CONTEXT, SYNTHESIS, FAILURE_REPORT, SESSION_HANDOFF | `pkg/spawn/` |

**Source:** `.kb/decisions/2025-12-22-template-ownership-model.md`

**Significance:** Ownership principle: "The tool that creates the artifact owns its template." This investigation confirms the decision is accurately reflected in the codebase.

---

## Synthesis

**Key Insights:**

1. **Domain-Split Ownership Works** - The split between kb-cli (knowledge artifacts) and orch-go (lifecycle artifacts) is cleanly implemented. No circular dependencies, each tool is self-contained.

2. **Multiple Override Mechanisms** - Each template type has a user-customizable path:
   - orch-go spawn templates: `.orch/templates/` in project
   - orch-go CLAUDE.md templates: `~/.orch/templates/claude/`
   - kb-cli templates: `~/.kb/templates/`
   - Skills: Edit source in orch-knowledge, run `skillc deploy`

3. **Skills are Embedded, Not Referenced** - When an agent is spawned, the skill content is copied into SPAWN_CONTEXT.md. This is intentional (see `.kb/decisions/2025-11-22-skill-system-hybrid-architecture.md`).

**Answer to Investigation Question:**

**Canonical Sources:**
- **orch-go spawn templates**: Go constants in `pkg/spawn/context.go` and `pkg/spawn/orchestrator_context.go`
- **orch-go CLAUDE.md templates**: Embedded files in `pkg/claudemd/templates/`
- **kb-cli artifact templates**: Go constants in `kb-cli/cmd/kb/create.go`
- **Skill templates**: Modular files in `orch-knowledge/skills/src/{category}/{skill}/.skillc/`

**Associated Tools:**
- `orch spawn` → uses SpawnContextTemplate + skill embedding
- `orch init` → uses claudemd templates
- `kb create` → uses kb-cli templates
- `skillc deploy` → compiles skill components to SKILL.md

**Alternative Versions:**
- Project-level overrides in `.orch/templates/`
- User-level overrides in `~/.orch/templates/claude/` and `~/.kb/templates/`
- Pre-filled vs reference templates (SESSION_HANDOFF)

**Relationships:**
- Skills are embedded INTO spawn templates at spawn time
- Spawn templates are generated from Go constants WITH skill content injected
- Knowledge artifact templates are independent of spawn templates

---

## Structured Uncertainty

**What's tested:**

- ✅ Template files exist at documented locations (verified: glob and file reads)
- ✅ Go constants match .orch/templates/ files (verified: compared content)
- ✅ Skill embedding works via {{.SkillContent}} placeholder (verified: read context.go:216-228)
- ✅ Override mechanism exists for claudemd templates (verified: LoadTemplate() checks user path first)

**What's untested:**

- ⚠️ Whether all skill components successfully compile (not run `skillc deploy`)
- ⚠️ Whether user overrides in ~/.kb/templates/ work (kb-cli not deeply audited)
- ⚠️ Whether .orch/templates/ overrides are actually used at runtime (would need to modify and test)

**What would change this:**

- Finding additional template locations not discovered
- Discovering templates that bypass the ownership model
- Finding templates that are duplicated across tools

---

## Implementation Recommendations

**Purpose:** This is an audit/inventory investigation - no implementation needed.

### Recommended Approach ⭐

**Document and Close** - This investigation confirms the existing template ownership model is accurate and well-implemented.

**Why this approach:**
- Prior decision (2025-12-22) correctly documents the architecture
- No inconsistencies found between documentation and implementation
- Template locations are predictable and follow established patterns

**Trade-offs accepted:**
- Not testing every override mechanism
- Not auditing kb-cli templates deeply (separate codebase)

### Alternative Approaches Considered

**Option B: Consolidate template documentation**
- **Pros:** Single reference for all templates
- **Cons:** Creates maintenance burden, prior decision is sufficient
- **When to use instead:** If templates change frequently or cause confusion

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - Main spawn template and synthesis template
- `pkg/spawn/orchestrator_context.go` - Orchestrator spawn templates
- `pkg/spawn/meta_orchestrator_context.go` - Meta-orchestrator templates
- `pkg/claudemd/claudemd.go` - CLAUDE.md template loading
- `pkg/claudemd/templates/*.md` - CLAUDE.md templates by project type
- `.orch/templates/*.md` - Project-level template overrides
- `.kb/decisions/2025-12-22-template-ownership-model.md` - Prior decision
- `~/Documents/personal/kb-cli/cmd/kb/create.go` - kb-cli templates (partial)
- `~/orch-knowledge/skills/src/**/.skillc/*.md` - Skill source components

**Commands Run:**
```bash
# Find all template-related files
glob **/*template* in orch-go

# List .orch/templates directory
ls -la .orch/templates/

# List claudemd templates
ls pkg/claudemd/templates/

# Find skill components
glob **/*.skillc/**/*.md in orch-knowledge

# Check kb-cli templates
cat ~/Documents/personal/kb-cli/cmd/kb/create.go | head -300
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-22-template-ownership-model.md` - Establishes ownership principle
- **Decision:** `.kb/decisions/2025-11-22-skill-system-hybrid-architecture.md` - Why skills are embedded
- **Investigation:** `.kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md` - Prior template audit

---

## Investigation History

**2026-01-07 10:00:** Investigation started
- Initial question: What templates exist, their canonical sources, and relationships
- Context: Need comprehensive inventory for template system understanding

**2026-01-07 10:30:** Completed template mapping
- Found 4 major template categories across 3 repositories
- Confirmed prior decision document accuracy
- Identified override mechanisms for each category

**2026-01-07 11:00:** Investigation completed
- Status: Complete
- Key outcome: Template system is well-organized with clear ownership boundaries documented in 2025-12-22 decision

---

## Self-Review

- [x] Real test performed (glob/grep/file reads, not just code review)
- [x] Conclusion from evidence (based on observed file contents)
- [x] Question answered (comprehensive template inventory provided)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED
