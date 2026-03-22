# Model: Domain Harness Architecture

**Domain:** Cross-Project Enforcement Layering
**Last Updated:** 2026-03-21
**Validation Status:** DESIGNED, NOT IMPLEMENTED — architecture decided but no implementation exists. Duplication/drift/gap diagnosis confirmed by 2026-03-21 probe; all four "resolved" decisions remain unimplemented.
**Synthesized From:**
- Led-totem-toppers architect failure (2026-03-21): architect edited production .scad files, carved notch through ACME threads. No deny hook existed. Soft skill constraint failed.
- Led-magnetic-letters harness audit: 3 Claude Code hooks, 5-layer gate stack, pre-commit gates — most complete enforcement, but adapted (diverged) copies of openscad-harness gates.
- openscad-harness audit: standalone framework, 4-layer gate stack (L1-L4), tested but never used in production (2 commits).
- harness CLI audit: structural governance (accretion, file growth, control plane lock), extracted from orch-go, mature (32 commits).
- Harness engineering model (orch-go): "agent failure = harness bug", hard vs soft enforcement, compositional correctness gap.

---

## Summary (30 seconds)

Enforcement for domain-specific projects (OpenSCAD, etc.) splits into three independent concerns that were being conflated: **structural governance** (file growth, accretion — language-agnostic), **domain validation** (geometry, printability, intent — OpenSCAD-specific), and **agent restrictions** (deny hooks, spawn gates — orchestration-specific). Conflating them caused duplication across projects (geometry-check.sh copied and diverged), gaps (led-totem-toppers described gates in CLAUDE.md but had zero enforcement), and confusion about where new enforcement should live. The architecture: harness CLI owns structural governance, openscad-harness owns domain gates (consumed via `harness init --openscad`), global Claude Code hooks own cross-project agent restrictions, and each project owns only its project-specific validators and specs.

---

## Core Architecture

### Three Independent Enforcement Concerns

| Concern | What it prevents | Where it lives | Why there |
|---------|-----------------|----------------|-----------|
| **Structural governance** | File growth, accretion, control plane mutation | harness CLI (Go binary) | Language-agnostic. Same gates for Go, OpenSCAD, Python. Single binary, single install. |
| **Domain validation** | Bad geometry, non-manifold mesh, unprintable parts, intent drift | openscad-harness (shell scripts + validate.scad) | Domain-specific. Different domains need different gates. OpenSCAD gates don't apply to Go. |
| **Agent restrictions** | Wrong skill editing wrong files, blanket git add, bypassing CGAL | Global Claude Code hooks (~/.orch/hooks/) + orch-go spawn gates | Orchestration-level. Same restrictions needed by every OpenSCAD project — not project-specific. |

### Layer Diagram

```
~/.orch/hooks/                         GLOBAL — agent restrictions
    gate-openscad-stl-cgal.py             require CGAL for STL export
    gate-openscad-post-render.py          auto-run geometry check after render
    gate-git-add-all.py                   block blanket git add
    gate-scad-architect-protection.py     block architect skill from editing .scad

harness CLI                            STRUCTURAL — any project
    harness init                          scaffold config + pre-commit hook
    harness init --openscad               also install domain gates from openscad-harness
    harness check                         accretion detection
    harness precommit accretion           pre-commit gate (advisory)

openscad-harness/                      DOMAIN — source of truth for OpenSCAD gates
    lib/validate.scad                     base validators (validate_range, validate_positive, etc.)
    gates/geometry-check.sh               L2: CGAL manifold, polygon budget, bounding box
    gates/printability-check.sh           L3: PrusaSlicer CLI
    gates/intent-check.sh                 L4: LLM alignment (measured: 77% precision, 96% recall)
    skills/design-part/SKILL.md           domain context for spawned agents
    skills/iterate-design/SKILL.md        parameter exploration workflow

project/                               PROJECT-SPECIFIC — unique to each project
    lib/validate.scad                     extends base with project validators
    specs/                                functional requirements + verification viewpoints
    gates/                                installed by harness init, not hand-copied
    .harness/config.yaml                  structural thresholds (from harness CLI)
```

### What Does NOT Need Its Own Skill

CAD modeling does not warrant new orch-go skills. The work types (architect, feature-impl, systematic-debugging, investigation) are domain-independent. The domain knowledge enters via:
- CLAUDE.md (project constraints, physical dimensions, material properties)
- openscad-harness skill templates (render/validate/export workflow) injected via SPAWN_CONTEXT
- Project specs (functional requirements per part)

Creating CAD-specific skills would duplicate the existing skill taxonomy. The failure that motivated this model (architect editing .scad files) was a harness gap, not a skill gap.

---

## Evidence: The Duplication/Drift/Gap Pattern

### What happened (observed across 4 projects)

| Artifact | openscad-harness | led-magnetic-letters | led-totem-toppers |
|----------|-----------------|---------------------|------------------|
| geometry-check.sh | Original (tested) | Adapted copy (diverged) | Referenced in CLAUDE.md, not present |
| lib/validate.scad | Base (7 generic modules) | Extended copy (project-specific) | Extended copy (project-specific) |
| Claude Code hooks | None | 3 hooks (working) | None |
| Pre-commit gates | None | $fn budget + accretion (reimplements harness CLI) | None |
| Intent gate (L4) | Implemented + measured | Adapted copy | Not started |

### Why it happened

1. **openscad-harness was template, not dependency.** Projects copied what they needed and diverged. No mechanism to pull upstream improvements.
2. **harness CLI and openscad-harness weren't connected.** Led-magnetic-letters reimplemented accretion gates in its pre-commit hook because it didn't use harness CLI.
3. **Claude Code hooks were project-local.** Led-magnetic-letters had them; led-totem-toppers didn't. Same hooks needed by both — should have been global.
4. **Domain validation and structural governance were conflated.** Unclear whether geometry-check belongs in harness CLI or openscad-harness, so it ended up copied into each project.

### The architect failure (2026-03-21)

An architect agent spawned into led-totem-toppers edited `parts/universal-sled.scad` directly, carving a cable channel through ACME threads. Diagnosis:
- **Soft harness masquerading as hard** (harness engineering model §Why This Fails #1): architect skill says "don't implement" but nothing mechanically prevented it.
- **No deny hook existed** for this project, despite led-magnetic-letters having similar hooks.
- **No geometry gate ran** to catch the broken thread, because geometry-check.sh wasn't wired despite being described in CLAUDE.md.
- Root cause: enforcement infrastructure was described but not installed. Three separate gaps (deny hook, geometry gate, post-render check) that all trace to the same architectural problem — no mechanism to install shared enforcement into new projects.

---

## Critical Invariants

1. **Structural governance is language-agnostic.** File growth, accretion, and control plane protection apply identically to Go, OpenSCAD, Python. These live in harness CLI, never in domain-specific tools.

2. **Domain validation is domain-specific.** Geometry checks, printability, intent alignment are OpenSCAD concerns. They live in openscad-harness, consumed by projects, not reimplemented per-project.

3. **Agent restrictions are orchestration-level.** Deny hooks and spawn gates apply across all projects of a domain. They live in global hooks (~/.orch/hooks/), not per-project settings.

4. **Projects own only what's unique.** Project-specific validators (validate_wire_channel, validate_bore_fit), specs, and calibration data. Everything else is inherited.

5. **`harness init --openscad` is the installation mechanism.** Single command installs both structural governance (config, pre-commit) and domain gates (from openscad-harness). Projects don't manually copy scripts.

6. **Described enforcement is not enforcement.** CLAUDE.md listing a 5-layer gate stack with "Status: Not yet" is soft harness. The gate must be mechanically present and wired to count as enforcement.

---

## Resolved Architecture Decisions (2026-03-21)

### Q1: How `harness init --openscad` consumes openscad-harness

**Decision: Local path copy with configurable source.**

`harness init --openscad` copies files from a local openscad-harness checkout. Source path is configurable via `.harness/global.yaml` or env var `OPENSCAD_HARNESS_PATH`, defaulting to `~/Documents/personal/openscad-harness`.

**What gets installed:**
- `gates/geometry-check.sh`, `gates/printability-check.sh`, `gates/intent-check.sh` → `project/gates/`
- `lib/validate.scad` → `project/lib/validate.scad` (base validators, refreshable)
- `.claude/hooks/gate-openscad-stl-cgal.py`, `gate-openscad-post-render.py` → project hooks
- `.harness/config.yaml` gains `domain: openscad` field
- Starter `lib/validate-project.scad` (if not present) for project-specific validators

**Refresh:** `harness init --openscad --refresh` re-copies base files (gates, validate.scad) without touching project-owned files (validate-project.scad, specs, config overrides).

**Why not alternatives:**
- Git submodule: Too complex for single-user setup. OpenSCAD projects aren't Go modules.
- Go embed: Couples harness binary version to openscad-harness version. Requires rebuild on gate changes.
- Network download: Unnecessary indirection for local repos.
- Symlinks: Break when moving projects, invisible coupling.

**Future:** If distribution to other users matters, switch to Go embed. For now, local path copy is the simplest thing that works.

### Q2: Global vs project-local hook detection

**Decision: Command-pattern hooks go global. File-pattern hooks use domain config detection.**

Two categories of hooks, two strategies:

| Hook type | Detection | Scope | Rationale |
|-----------|-----------|-------|-----------|
| Command-pattern (CGAL gate, post-render) | Match `openscad` in command string | Global (`~/.orch/hooks/`) | Harmless in non-OpenSCAD projects — the command simply never fires |
| File-pattern (architect-deny) | Check `.harness/config.yaml` for `domain: openscad` | Global (`~/.orch/hooks/`) with config guard | Needs to know project type before blocking edits to `parts/*.scad` |

**Detection mechanism for file-pattern hooks:**
1. Hook reads `.harness/config.yaml` from project root (found via git rev-parse or cwd)
2. If `domain: openscad` is present → apply OpenSCAD-specific restrictions
3. If config missing or domain differs → pass through (no enforcement)
4. Cost: one file read per hook invocation (< 1ms, cached by OS)

**Migration from project-local:**
- `harness init --openscad` no longer installs hooks into `.claude/hooks/`
- Instead, hooks live permanently in `~/.orch/hooks/` and self-activate based on domain detection
- Existing project-local hooks in led-magnetic-letters and led-totem-toppers become redundant after global hooks are deployed

**Specific hooks moving global:**
- `gate-openscad-stl-cgal.py` → global, command-pattern (already fires only on `openscad -o *.stl`)
- `gate-openscad-post-render.py` → global, command-pattern (fires on `openscad -o` success)
- `gate-architect-production-files.py` → global, config-guarded (checks domain before blocking `parts/*.scad` edits)
- `gate-git-add-all.py` → already global (not domain-specific, lives in harness CLI scaffold)

### Q3: Base validator extension pattern

**Decision: Two-file split — base (refreshable) + project (owned).**

- `lib/validate.scad` — installed from openscad-harness by `harness init --openscad`. Contains 7 generic validators. Can be refreshed via `--refresh` without data loss.
- `lib/validate-project.scad` — project-owned. Contains project-specific validators (validate_wire_channel, validate_bore_fit, etc.). Never touched by harness init.

**Material/printer overrides:** Handled at call sites, not in the validator file. The base validators already accept parameters with defaults:
```openscad
validate_wall_thickness("wall", wall, min_wall=1.2);  // PETG override
validate_wall_thickness("wall", wall);                  // uses default 0.8
```

Projects pass their material-specific thresholds when calling validators. This requires no wrapper modules, no config-to-OpenSCAD translation, no file generation. The parameterization is already built into the base.

**For gate scripts (geometry-check.sh):** Same pattern. Environment variables already configure thresholds:
```bash
MAX_FACETS=200000 MAX_X=256 MAX_Y=256 ./gates/geometry-check.sh parts/file.scad
```
Project-specific thresholds live in `.harness/config.yaml` under `domain_config.openscad`, and `harness init --openscad` generates a thin wrapper (`gates/run-geometry-check.sh`) that reads config and passes as env vars. Or projects set these in their CLAUDE.md / pre-commit scripts.

**Why not symlinks:** Break on project relocation, invisible dependency, git tracks symlink target not content.
**Why not single merged file:** Updates to base clobber project customizations, which is exactly the current divergence problem.

### Q4: Where Layer 5 (vision verification) lives

**Decision: Gate script in openscad-harness, viewpoint config + calibration in project.**

| Component | Owner | Location | Why |
|-----------|-------|----------|-----|
| Gate script (`vision-check.sh`) | openscad-harness | `gates/vision-check.sh` | Generic workflow: render viewpoints → send to Claude vision → compare against checklist. Same for any OpenSCAD project. |
| Viewpoint definitions | Project spec | `specs/*-spec.md` (verification viewpoints section) | Camera angles and what-to-check are per-part, per-project. Led-magnetic-letters checks magnet placement; led-totem-toppers checks thread integrity. |
| Calibration data | Project | `.harness/vision-calibration/` | Known-good/bad renders for threshold tuning. Project-specific because failure modes differ per design. |
| Functional checklist | Project spec | Derived from spec's FR (functional requirements) | What constitutes "correct" is defined by the spec, not the gate. |

**Gate interface:**
```bash
./gates/vision-check.sh \
  parts/file.scad \
  specs/file-spec.md \
  exports/file-summary.json \
  --viewpoints "top-down,side-cross-section,bottom-back" \
  --model sonnet
```

The gate script:
1. Reads viewpoint definitions from the spec file (camera angles, verification criteria)
2. Renders each viewpoint via OpenSCAD (`-o exports/viewpoint-N.png --camera ...`)
3. Sends renders + functional checklist to Claude vision
4. Returns structured verdict (pass/fail per viewpoint with confidence)

**Relationship to Layer 4 (intent-check):** Layer 4 analyzes source code for intent alignment. Layer 5 analyzes rendered output for functional correctness. Different abstraction levels, different failure modes. Layer 5 catches things Layer 4 cannot (e.g., geometrically valid but functionally broken designs where the code "looks right" but the render reveals disconnected channels).

**Implementation sequence:** Layer 5 depends on the instrumented 26-letter sweep (led-magnetic-letters) for calibration data. Build the gate script first, calibrate against known-good/bad renders, measure precision/recall before promoting beyond advisory.

---

## Implementation Status (as of 2026-03-21 probe)

| Decision | Design Status | Implementation Status |
|----------|--------------|----------------------|
| Q1: `harness init --openscad` | Decided | NOT IMPLEMENTED — `harness_cmd.go` has no `--openscad` flag, no OpenSCAD logic |
| Q2: Global hooks with domain detection | Decided | NOT IMPLEMENTED — `~/.orch/hooks/` has zero OpenSCAD hooks; all OpenSCAD hooks remain project-local |
| Q3: Two-file validator split | Decided | NOT ADOPTED — both projects have single diverged `validate.scad`; no `validate-project.scad` exists |
| Q4: Vision check (L5) | Decided | PARTIALLY IMPLEMENTED — `vision-check.sh` exists in led-magnetic-letters gates but not via openscad-harness |

**Current actual state:**
- Hooks: project-local in `.claude/hooks/` (led-magnetic-letters: 3 hooks, led-totem-toppers: 4 hooks)
- Validators: single `lib/validate.scad` per project, diverged (different defaults, different domain modules)
- Gates: manually copied, diverging (led-magnetic-letters has 8 scripts, led-totem-toppers has 1, openscad-harness has 3)
- No `.harness/config.yaml` with `domain: openscad` in any project

## Remaining Open Questions

All four architecture questions are decided in design but none are implemented. The open question is implementation prioritization — the duplication/drift problem the architecture was designed to solve is actively worsening.

## Evidence

- 2026-03-21: Knowledge Decay Verification — first probe. Problem diagnosis confirmed, all four resolved decisions found unimplemented. Divergence actively worsening.

---

## Falsification Criteria

| Criterion | What would disprove it | How to test |
|-----------|----------------------|-------------|
| Layering reduces duplication | After implementing, check if domain gates still get copied into projects | Count diverged copies after 3+ months |
| Global hooks prevent gaps | New OpenSCAD project gets enforcement without manual setup | Create a new project, check what's enforced out of the box |
| Skills don't need domain variants | Feature-impl agents produce correct .scad geometry with domain context injection alone | Track agent success rate on OpenSCAD tasks vs hypothetical CAD-specific skill |
| `harness init` is sufficient | Projects don't need manual gate wiring after init | Check if any project modifies installed gate scripts (divergence signal) |
