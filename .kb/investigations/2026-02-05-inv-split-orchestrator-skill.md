# Investigation: Split Orchestrator Skill into Core + Reference Files

**Date:** 2026-02-05  
**Status:** Active  
**Beads:** orch-go-21304

## Problem

The orchestrator skill at `~/.opencode/skill/orchestrator/SKILL.md` is ~24K tokens with only 5 load-bearing patterns. This creates:

- Context overhead (agents load everything even when only core workflow needed)
- Maintenance burden (hard to find and update specific guidance)
- Cognitive load (mixing essential patterns with reference material)

**Goal:** Core skill under 10K tokens, with reference material moved to separate files that can be loaded on demand.

## Constraints

From spawn context:

- Auto-generated skills require template edits (but orchestrator skill has NO `.skillc/` directory - manually maintained)
- Orchestrator skill file is at `~/.opencode/skill/orchestrator/SKILL.md`

## Current State

**File metrics:**

- Location: `/Users/dylanconlin/.opencode/skill/orchestrator/SKILL.md`
- Lines: 1,111
- Words: 5,889
- Estimated tokens: Unknown (tiktoken not available, but task says ~24K)
- Build system: None (no `.skillc/` directory, manually maintained)
- Permissions: Read-only (`-r--r--r--`)

## Analysis: Section Breakdown

### Major Sections (from reading SKILL.md):

1. **Meta/Context** (Lines 1-61)
   - Name, description, metadata
   - Context Detection: Am I Orchestrator or Worker?
   - Skill System Architecture (Hybrid by Design)

2. **Tool Ecosystem** (Lines 62-85)
   - beads, kn, kb, orch commands

3. **Work Pipeline** (Lines 86-122)
   - How work flows through the system
   - Issue creation → Daemon → Completion Review

4. **Strategic Alignment** (Lines 123-150)
   - orch focus, drift, next commands

5. **Window Layout** (Lines 151-167)
   - Dylan's specific environment setup

6. **⛔ ABSOLUTE DELEGATION RULE** (Lines 168-275)
   - **LOAD-BEARING PATTERN #1**
   - Core principle: Orchestrators NEVER do spawnable work
   - Delegation paths and decision trees

7. **Strategic Dogfooding** (Lines 276-297)
   - Applying delegation even to meta-orchestration work

8. **Orchestrator Autonomy** (Lines 298-378)
   - **LOAD-BEARING PATTERN #2**
   - When to act vs ask: Always Act, Propose-and-Act, Actually Ask
   - Anti-patterns and mind-reading test

9. **Orchestrator Core Responsibilities** (Lines 379-405)
   - What orchestrators do themselves (never delegate)

10. **Backlog Ownership** (Lines 406-419)
    - Strategic partner, not just dispatcher

11. **Skill Selection Guide** (Lines 420-798)
    - **LOAD-BEARING PATTERN #3**
    - Massive section with decision trees for:
      - Worker skills (feature-impl, debugging, investigation, etc.)
      - Design triage (feature requests)
      - Bug triage (broken things)
      - Issue creation (two-tier quality model)
      - Beads tracking requirements
      - Beads labels and multi-issue patterns
      - Coordination skills
      - Meta skills
      - Investigation outcomes
      - Knowledge capture (kn vs kb)
      - Knowledge placement guide
      - Auto-capture user corrections
      - Reference material

12. **Spawning Checklist** (Lines 799-878)
    - **LOAD-BEARING PATTERN #4**
    - Pre-spawn knowledge check (REQUIRED)
    - Critical path context checklist
    - Spawning methods
    - Task description best practices

13. **Post-Completion Verification** (Lines 879-921)
    - Single agent: orch complete
    - Batch review: orch review

14. **Integration Audit** (Lines 922-932)
    - Epic/phase completion verification

15. **Amnesia-Resilient Artifact Design** (Lines 933-1003)
    - Foundational principles
    - The Four Questions
    - Standards and anti-patterns

16. **Pre-Response Protocol** (Lines 1004-1009)
    - Delegation reminder, tmux window naming

17. **Artifact Organization** (Lines 1010-1018)
    - Quick reference for paths and naming

18. **Error Recovery Patterns** (Lines 1019-1025)
    - Quick reference for error handling

19. **System Maintenance** (Lines 1026-1030)
    - Skill editing instructions

20. **Orch Commands** (Lines 1031-1037)
    - Quick reference for commands

21. **Checkpoint Management** (Lines 1038-1044)
    - Session scope and monitoring

22. **Common Red Flags & Quick Decisions** (Lines 1045-1111)
    - **LOAD-BEARING PATTERN #5**
    - Warning signs and decision trees
    - Red flags for orchestrator, agent work, verification
    - Quick decision trees for common scenarios

## Identifying Load-Bearing Patterns

Based on analysis, the **5 essential patterns orchestrators need constantly** are:

1. **ABSOLUTE DELEGATION RULE** - Core identity: orchestrators delegate, never implement
2. **Orchestrator Autonomy** - How to interact with Dylan: act vs ask patterns
3. **Skill Selection Guide** - What skill to spawn for each situation
4. **Spawning Checklist** - Pre-spawn requirements (knowledge check, context)
5. **Common Red Flags & Quick Decisions** - Warning signs and fast decision-making

**Everything else is reference material** that can be moved to external files.

## Proposed Split Strategy

### Core SKILL.md (Target: <10K tokens)

Keep only:

- Meta/context detection
- ABSOLUTE DELEGATION RULE (condensed version with pointer to reference)
- Orchestrator Autonomy (condensed version with pointer to reference)
- Skill Selection Guide (decision trees only, details in reference)
- Spawning Checklist (requirements only, examples in reference)
- Common Red Flags (top 5-10 only, rest in reference)
- Quick links to all reference files

### Reference Files (New: reference/ directory)

Create `~/.opencode/skill/orchestrator/reference/` with:

1. **tool-ecosystem.md** - Tool commands (beads, kn, kb, orch)
2. **work-pipeline.md** - How work flows, daemon usage
3. **strategic-alignment.md** - Focus, drift, next commands
4. **window-layout.md** - Dylan's environment setup
5. **dogfooding.md** - Meta-orchestration delegation
6. **core-responsibilities.md** - What orchestrators do themselves
7. **backlog-ownership.md** - Strategic backlog management
8. **skill-selection-details.md** - Full details for each skill, examples
9. **spawning-best-practices.md** - Task naming, phase selection, MCP servers
10. **completion-verification.md** - Post-completion workflows
11. **integration-audit.md** - Epic/phase completion verification
12. **amnesia-resilient-design.md** - Artifact design principles
13. **artifact-organization.md** - Paths, naming, search
14. **error-recovery.md** - Error handling patterns
15. **checkpoint-management.md** - Session scope management
16. **red-flags-comprehensive.md** - Full list of warning signs

## Implementation Plan

**Phase 1: Create reference files**

- Extract sections to individual reference files
- Ensure each file is self-contained with context

**Phase 2: Update core SKILL.md**

- Condense load-bearing patterns (keep decision trees, move examples)
- Add "See reference/X.md for details" pointers
- Verify token count <10K

**Phase 3: Validation**

- Check token count
- Verify all content preserved (no information loss)
- Ensure reference files are discoverable and loadable

## Next Steps

1. Create reference/ directory structure
2. Extract and write each reference file
3. Update core SKILL.md with condensed version + pointers
4. Validate token count and content preservation

## Questions

- How should reference files be loaded? Via Skill tool, or just as documentation?
- Should reference file paths be relative or absolute?
- Should we preserve the read-only permissions on SKILL.md?
