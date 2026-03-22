# Probe: Knowledge Decay Verification — Domain Harness Architecture

**Date:** 2026-03-21
**Type:** Verification (knowledge decay check)
**Model:** domain-harness-architecture
**Trigger:** 999d since last probe (first probe for this model)

---

## Method

Verified the model's six critical invariants and four resolved architecture decisions against current filesystem state across orch-go, openscad-harness, led-magnetic-letters, and led-totem-toppers.

---

## Findings by Claim

### Claim 1: Three independent enforcement concerns (structural, domain, agent)
**Status: CONFIRMED (conceptually) — partially implemented**

The separation is sound as a mental model. In practice:
- **Structural governance** (harness CLI): exists and is mature (32+ commits). `orch harness` delegates to standalone `harness` binary.
- **Domain validation** (openscad-harness): exists at `~/Documents/personal/openscad-harness/` with gates (`geometry-check.sh`, `printability-check.sh`, `intent-check.sh`) and `lib/validate.scad`.
- **Agent restrictions**: hooks exist but are **project-local, not global**. Both led-magnetic-letters and led-totem-toppers have `.claude/hooks/` with `gate-openscad-stl-cgal.py`, `gate-openscad-post-render.py`, `gate-git-add-all.py`. `~/.orch/hooks/` contains only orchestration-level hooks (governance protection, bd-close gate, spawn context validation) — zero OpenSCAD-specific hooks.

**Gap:** The model says agent restrictions should live in global hooks (`~/.orch/hooks/`), but the migration from project-local to global has NOT happened.

### Claim 2: `harness init --openscad` is the installation mechanism
**Status: CONTRADICTED — not implemented**

- `cmd/orch/harness_cmd.go` exists but contains no `--openscad` flag or OpenSCAD-specific logic.
- `harness init` exists (scaffolds governance artifacts) but has no domain flag beyond what was committed in `63b82c881` ("harness init --openscad domain flag") — grepping the Go source for "openscad" returns zero matches in harness_cmd.go.
- No `.harness/config.yaml` exists in either led-magnetic-letters or led-totem-toppers (model claims this would contain `domain: openscad`).
- Projects still have manually-copied gate scripts, not installed ones.

**The model describes `harness init --openscad` as resolved and decided, but the implementation does not exist in the CLI.**

### Claim 3: Two-file validator split (validate.scad + validate-project.scad)
**Status: CONTRADICTED — not adopted**

- Both led-magnetic-letters and led-totem-toppers have a single `lib/validate.scad` with diverged content.
- Neither project has `lib/validate-project.scad`.
- The divergence the model predicted would be solved by this split is actively present: led-magnetic-letters has `validate_magnet_recess` in its validate.scad; led-totem-toppers has `validate_wire_channel` in the same file. Different default `min_wall` values (0.8 vs 1.2).
- openscad-harness has `lib/validate.scad` (base, 7 generic modules) but no project has adopted the two-file split.

### Claim 4: Global hooks with domain config detection (Q2)
**Status: CONTRADICTED — not implemented**

- No OpenSCAD-specific hooks exist in `~/.orch/hooks/`. All hooks there are orchestration-level (governance, spawn, bd-close).
- led-totem-toppers now has `gate-architect-production-files.py` in `.claude/hooks/` (added 2026-03-21, the day the model was written) — but it's project-local, not global with config detection.
- The config detection mechanism (read `.harness/config.yaml` for `domain: openscad`) cannot work because no project has this config file.

### Claim 5: Gate scripts divergence
**Status: CONFIRMED — divergence is real and ongoing**

- led-magnetic-letters `gates/geometry-check.sh`: 11 lines, adapted from openscad-harness
- led-totem-toppers `gates/geometry-check.sh`: different file (4.9KB), clearly diverged
- openscad-harness `gates/geometry-check.sh`: 2.3KB, the original
- led-magnetic-letters has 8 gate scripts; led-totem-toppers has 1; openscad-harness has 3. The duplication/drift pattern the model describes is actively worsening.

### Claim 6: "Described enforcement is not enforcement" (Invariant 6)
**Status: CONFIRMED — still true**

- led-totem-toppers now has hooks (added after the architect failure that motivated this model), but enforcement gaps remain.
- The model's diagnosis of the architect failure is accurate: enforcement was described in CLAUDE.md but not mechanically present.

---

## Overall Verdict

**Model is architecturally sound but describes a future state as resolved.**

The three-concern separation, the `harness init --openscad` mechanism, the two-file validator split, and global hook migration are all **designed but not implemented**. The model's "Resolved Architecture Decisions" section and "Remaining Open Questions: None" create a false impression of completion.

**What's accurate:**
- The problem diagnosis (duplication, drift, gaps) — confirmed and worsening
- The architect failure analysis — accurate
- The conceptual architecture (three concerns, layered diagram) — sound
- Invariant 6 ("described enforcement is not enforcement") — ironically applies to this model itself

**What's stale/wrong:**
- "Resolved Architecture Decisions" implies implementation exists — it doesn't
- `harness init --openscad` described as decided mechanism — not in codebase
- Two-file validator split described as decided — not adopted by any project
- Global hooks migration described as decided — not started
- "Remaining Open Questions: None" — misleading; implementation is the open question

---

## Recommended Model Updates

1. Change validation status from "WORKING HYPOTHESIS" to something that distinguishes "architecture decided" from "architecture implemented" — e.g., "DESIGNED — not yet implemented"
2. Add implementation status to each resolved decision (Q1-Q4): "Designed, not implemented as of 2026-03-21"
3. Add a "Current State" section documenting what actually exists (project-local hooks, single validate.scad, manual copies)
4. Update "Remaining Open Questions" to reflect implementation gaps
