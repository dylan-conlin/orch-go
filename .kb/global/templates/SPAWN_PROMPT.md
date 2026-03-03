# Spawn Prompt Template

## Basic Structure

```
TASK: [One sentence description]

CONTEXT: [Minimal background needed]

ARCHITECTURE CONTEXT:
- **Orchestration Pattern:** Per-project orchestrators (Architecture B)
  - Multiple `.orch/` directories across projects (meta-orchestration, price-watch, context-driven-dev, etc.)
  - Each project has independent orchestration context
  - Dylan switches contexts via `cd` - not managing all projects from one instance
  - When in `/project-name/`, you ARE that project's orchestrator
- **Key Architectural Constraints:**
  - Projects are architecturally independent (loose coupling)
  - Cross-project dependencies = exception, not rule
  - Shared concerns extracted to libraries, not coordinated via meta-orchestrator

⚠️ **META-ORCHESTRATION TEMPLATE SYSTEM** (Critical if working on meta-orchestration):

**IF task involves these files/patterns:**
- .orch/CLAUDE.md updates
- Orchestrator guidance changes
- Pattern/workflow documentation
- Any file with <!-- ORCH-TEMPLATE: ... --> markers

**THEN you MUST understand the template build system:**

**Template Architecture (3 layers):**
1. **Source:** templates-src/orchestrator/*.md ← EDIT HERE
2. **Distribution:** ~/.orch/templates/orchestrator/*.md (synced via `orch build-global`)
3. **Consumption:** .orch/CLAUDE.md (rebuilt via `orch build --orchestrator`)

**Critical Rules:**
- ❌ NEVER edit .orch/CLAUDE.md sections between `<!-- ORCH-TEMPLATE: ... -->` markers
- ✅ ALWAYS edit source in templates-src/orchestrator/
- ✅ ALWAYS rebuild: `orch build-global && orch build --orchestrator`

**Before editing ANY file:**
```bash
grep "ORCH-TEMPLATE\|Auto-generated" <file>
```

**If file has template markers:**
1. Find source template path in the Auto-generated comment
2. Edit templates-src/orchestrator/[template-name].md
3. Run: `orch build-global` (sync source → distribution)
4. Run: `orch build --orchestrator` (regenerate .orch/CLAUDE.md)
5. Verify changes appear in .orch/CLAUDE.md

**Files that are NOT templates (safe to edit directly):**
- docs/*.md
- tools/orch/*.py
- templates-src/ files (these ARE the source)

**Why this matters:**
- Changes to template-generated sections get SILENTLY OVERWRITTEN on next build
- This is a recurring amnesia bug (see post-mortem: .orch/knowledge/spawning-lessons/2025-11-20-forgot-template-system-context-recurring.md)

**Reference:** .orch/CLAUDE.md lines 77-125 for template system documentation

[OPTIONAL] Context from Prior Work:
[Include if artifact check found relevant work - see Pre-Spawn Artifact Check section]
- Prior work: Read workspace at PROJECT_DIR/.orch/workspace/previous-agent/WORKSPACE.md
- Investigation: See findings at PROJECT_DIR/.orch/investigations/{type}/YYYY-MM-DD-topic.md (where {type} is systems, feasibility, audits, etc.)

PROJECT_DIR: [Absolute path to project]

SESSION SCOPE: [Small/Medium/Large] (estimated [1-2h / 2-4h / 4-6h+])
- [Brief justification: task count, complexity, unknowns]
- Recommend checkpoint after [specific phase/task] if session exceeds [X] hours

SCOPE:
- IN: [What's in scope]
- OUT: [What's explicitly out of scope]

AUTHORITY:
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

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be PROJECT_DIR)
2. [COORDINATION_CHECK]
3. [COORDINATION_UPDATE]
4. [COORDINATION_PHASE]
5. [Task-specific deliverables]

[STATUS_UPDATES]
```

> **Maintainer note:** The Basic Structure block above is the *only* text agents receive.
> The sections below explain when and how to edit each portion of that block—do not
> duplicate or drift from the template text itself.

---

## Architecture Context Guidelines

**Canonical text lives in the ARCHITECTURE CONTEXT block inside Basic Structure.**
Update the template there when architectural guidance changes. Use this section only to note:

- When to extend/reduce the default block (e.g., add project-specific constraints from `.orch/CLAUDE.md`).
- Historical reasons for the current default (Architecture B per project, etc.).
- References for future maintainers (e.g., decision docs).

---

## Session Scope Guidelines

The SESSION SCOPE paragraph in Basic Structure is filled with defaults. Use these notes when
editing the template to change default scope sizing or checkpoint cadence.

**Small (1-2h):**
- Bug fixes, typos, documentation updates
- Single-file changes
- Clear, well-defined requirements
- **Checkpoint strategy:** No planned checkpoints (single session expected)

**Example:**
```
SESSION SCOPE: Small (estimated 1-2h)
- Single file change (docs/README.md)
- Clear requirements
- No planned checkpoints
```

**Medium (2-4h):**
- Feature additions with 3-6 tasks
- Multi-file changes with integration
- Requires testing/verification
- **Checkpoint strategy:** Plan 1 checkpoint after Phase 1 if >3h elapsed

**Example:**
```
SESSION SCOPE: Medium (estimated 3-4h)
- 5 tasks total across 3 files
- Multi-file integration required
- Recommend checkpoint after Task 3 (Phase 1 complete) if session exceeds 3h
```

**Large (4-6h+):**
- Major features with 7+ tasks
- Architecture changes across many files
- Research/investigation components
- **Checkpoint strategy:** Plan checkpoints every 2-3 tasks OR every 3-4 hours

**Example:**
```
SESSION SCOPE: Large (estimated 5-6h)
- 8 tasks total, includes research phase
- Architecture changes across 5+ files
- Recommend checkpoints:
  - After Task 3 (research complete)
  - After Task 6 (before integration testing)
  - Every 3-4 hours regardless of task progress
```

---

## Estimation Guidelines

These guidelines support maintainers when updating the SESSION SCOPE defaults above. If scope
estimation heuristics change, update both this explanation *and* the text inside Basic Structure:
- When uncertain, default to the larger scope and let the agent adjust downward.
- Capture actual effort in workspaces to refine these heuristics over time.

---

## Authority Guidelines

The AUTHORITY block inside Basic Structure should always be present. Use this section to guide
maintainers when customizing escalation rules:

- High-risk work (auth, data migrations, prod systems): add explicit “escalate when…” bullets.
- Low-risk work (docs, tests): you can expand the “you have authority” list to encourage autonomy.
- When in doubt, keep defaults and reference relevant decision docs to explain why.

Document rationale for any changes here so future editors know when/why the block deviated.

---

## Complete Example (Medium Scope)

The full example previously duplicated the Basic Structure content. Refer to the artifacts in
`.orch/workspace/2025-11-22-feat-impl-sessionstart-context-loading-hook-auto-load/` for a concrete
spawn output if you need a model. Keep this section as links or references instead of embedding a
copy of the template.

---

## Skill Maintenance Spawn Template (for meta-level skill changes)

**Purpose:** Provide a dedicated workflow for maintaining complex skills (templates, phases, and build artifacts) so that regular feature/debugging workers can treat skills as read-only process guides.

**When to use:** Any time you want to change multi-phase skills like `feature-impl`, `systematic-debugging`, `codebase-audit`, or other skills built from `src/SKILL.md.template` + `src/phases/*.md`.

**Spawn template:**
```
TASK: Maintain and update the "[skill-name]" skill (templates, phases, and build artifacts).

CONTEXT:
- Complex skills live under ~/.claude/skills/{worker,orchestrator,shared}/{skill}/
- SKILL.md for complex skills is built from src/SKILL.md.template + src/phases/*.md via `orch build --skills`
- Simple one-file skills may still be hand-authored SKILL.md files.

PROJECT_DIR: /Users/dylanconlin/meta-orchestration

SESSION SCOPE: Small/Medium (estimated 1-3h)
- Depends on size of changes; default to Medium if refactoring multiple phases

SCOPE:
- IN: Editing src/SKILL.md.template and src/phases/*.md for "[skill-name]"
- IN: Running `orch build --skills` and verifying the generated SKILL.md
- IN: Updating any README/docs that describe the skill
- OUT: Implementing project-specific feature work using the skill

DELIVERABLES (REQUIRED):
- Updated templates: ~/.claude/skills/{category}/[skill-name]/src/SKILL.md.template (and/or src/phases/*.md)
- Rebuilt skill: ~/.claude/skills/{category}/[skill-name]/SKILL.md (generated by `orch build --skills`)
- Knowledge note: .orch/knowledge/agent-lessons/YYYY-MM-DD-skill-maintenance-[skill-name].md (brief summary of changes and rationale)

WORKFLOW:
1. Locate the skill sources for "[skill-name]" under ~/.claude/skills/{category}/[skill-name]/src/
2. Make changes in src/SKILL.md.template and src/phases/*.md (do NOT edit SKILL.md directly for complex skills).
3. Run `orch build --skills` to regenerate SKILL.md and verify the auto-generated header.
4. If applicable, ensure any top-level aliases under ~/.claude/skills/[skill-name] still point to the correct hierarchical directory.
5. Sanity-check the rendered SKILL.md for clarity and consistency (phases, deliverables, verification).
6. Capture a short knowledge note with what changed, why, and any follow-ups.
```

**Note:** For simple skills that are just a single SKILL.md without phases, continue to edit the file directly and skip the build step. This template is optimized for complex, multi-phase skills managed by the build system.

---

## Notes

- **SESSION SCOPE** is about session size estimation (for checkpoint planning)
- **SCOPE (IN/OUT)** is about task boundaries (what's included/excluded)
- Both are important but serve different purposes
- Orchestrator sets SESSION SCOPE, agent validates during planning
- Agent can announce if scope estimate seems wrong early in session
