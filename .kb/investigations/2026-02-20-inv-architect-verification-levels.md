# Design: Verification Levels for orch Completion

**Date:** 2026-02-20
**Phase:** Complete
**Status:** Complete
**Beads:** orch-go-1157
**Skill:** architect
**Type:** design investigation

---

## Design Question

How should the orchestrator and Dylan communicate about verification? "Verify this works" means 5 different things depending on context. The current system has 14 gates that all fire by default (with auto-skip heuristics), requiring `--skip-*` flags or `--force` to bypass. We need a shared vocabulary — verification *levels* — where the level declared per-spawn determines which gates fire.

## Problem Framing

### Success Criteria

A good answer:
1. Provides a vocabulary both human and orchestrator understand for "how much verification does this need?"
2. Maps existing 14 gates to levels (not adding new gates)
3. Levels are declared at spawn time and flow through to completion — the common case requires zero flags
4. Subsumes both verification gates AND tradeoff visibility (from orch-go-1158)
5. `--force` becomes unnecessary for well-configured spawns
6. `--explain` and `--verified` remain orchestrator judgment calls, not mechanical requirements

### Constraints

- **Dylan doesn't read code** — verification must be behavioral, not comprehension-based
- **Orchestrator is AI** — flags are programmatic, not human-typed
- **14 gates exist** — we're reorganizing, not adding
- **Historical failure: gate proliferation → `--force` as default** — any design that recreates this has failed
- **Tradeoff visibility belongs upstream** (orch-go-1158 conclusion) — completion gates are safety net, not primary mechanism
- **Two existing implicit level systems:** spawn tier (light/full) and checkpoint tier (1/2/3 by issue type)

### Scope

**In scope:** Level taxonomy, gate mapping, declaration mechanism, replacement of SkipConfig, integration with tradeoff visibility
**Out of scope:** Implementation code, new gates, changes to beads/OpenCode

---

## Exploration (Fork Navigation)

### The Current State: Three Implicit Level Systems

The audit (orch-go-1165) and codebase exploration reveal that verification levels *already exist* — they're just encoded in three separate, uncoordinated systems:

**System 1: Spawn Tier** (`pkg/spawn/config.go`)
- `light` — skips SYNTHESIS.md requirement
- `full` — requires SYNTHESIS.md
- Declared per-skill in `SkillTierDefaults` map

**System 2: Checkpoint Tier** (`pkg/checkpoint/checkpoint.go`)
- Tier 1 (feature/bug/decision) — requires gate1 (explain-back) AND gate2 (behavioral)
- Tier 2 (investigation/probe) — requires gate1 only
- Tier 3 (task/question/other) — no checkpoint required
- Derived from beads issue type at completion time

**System 3: Skill-Based Auto-Skips** (scattered across `pkg/verify/`)
- `IsKnowledgeProducingSkill()` — auto-skips synthesis gate
- `IsSkillRequiringTestEvidence()` — determines test evidence gate
- `skillsExcludedFromTestEvidence` — explicit exclusion map
- Various per-gate heuristics (no web changes → skip visual, no Go changes → skip build)

**The vocabulary gap:** These three systems encode the same underlying concept — "how much verification does this change need?" — but they use different vocabularies (tier names, issue types, skill names) and are evaluated at different times (spawn, completion).

### Fork 1: What taxonomy?

**Options:**
- A: Numeric levels (L0-L4) based on risk/impact
- B: Named levels based on work type (knowledge, config, implementation, behavioral, architectural)
- C: Unify existing systems — a single "verification level" that subsumes spawn tier + checkpoint tier + skill-based skips

**Substrate says:**
- Principle "Evolve by Distinction": When problems recur, ask "what are we conflating?" — we're conflating three things that should be one
- Principle "Coherence Over Patches": If 5+ fixes hit the same area, redesign — we have 3 implicit systems and 12 skip flags
- Model "Orchestrator Session Lifecycle": Orchestrators need shared vocabulary with Dylan
- Decision from orch-go-1158: Tradeoff surfacing belongs upstream, not in completion gates

**Recommendation:** Option C (unify existing systems) with named levels for human legibility.

The key insight is that we don't need to *invent* levels — we need to *name* the levels that already exist implicitly. The spawn tier, checkpoint tier, and skill-based auto-skips all converge on a roughly 4-level spectrum:

| Level | Name | Current Implicit Equivalent |
|-------|------|-----------------------------|
| V0 | **Acknowledge** | Tier 3 + light tier + knowledge skill — "did agent finish?" |
| V1 | **Artifacts** | Tier 2 + full tier — "are deliverables present and coherent?" |
| V2 | **Evidence** | Tier 1 (feature-impl/debugging) — "is there evidence of testing?" |
| V3 | **Behavioral** | Tier 1 + web/UI changes — "did a human observe it working?" |

**Trade-off accepted:** Reduces granularity vs current 14-gate system. Acceptable because the 14 gates still exist — levels just determine which subset fires.

### Fork 2: How are levels declared?

**Options:**
- A: Orchestrator declares level at spawn time via `--verify-level V2`
- B: Level inferred from skill + issue type (what we have today, unified)
- C: Default inferred, orchestrator can override

**Substrate says:**
- Constraint "Judgment Over Mechanics" from task: `--explain` and `--verify` should be orchestrator judgment calls
- Principle "Gate Over Remind": Defaults must work without flags
- Evidence: Current system already infers per-skill defaults (`SkillTierDefaults`, `skillsRequiringTestEvidence`)
- Decision from orch-go-1158: Orchestrator declares verification level at spawn time based on judgment

**Recommendation:** Option C (infer + override).

The default verification level is deterministic from (skill, issue type):

```
V_default = max(skill_level, issue_type_level)
```

Where:
- investigation/architect/research/codebase-audit → V1 (artifacts)
- feature-impl/systematic-debugging/reliability-testing → V2 (evidence)
- issue-creation → V0 (acknowledge)
- feature/bug/decision issue type → min V2 (evidence)
- investigation/probe issue type → min V1 (artifacts)
- task/question → no minimum

The orchestrator can override upward or downward:
```
orch spawn feature-impl "update README" --verify-level V0  # config change, no tests needed
orch spawn investigation "security audit" --verify-level V3 # elevated: want behavioral confirmation
```

**Trade-off accepted:** Adding one more flag to spawn. Acceptable because: (a) it's optional — defaults work without it, (b) it replaces 12 flags at completion time, (c) it's the *one* flag that captures the judgment call.

### Fork 3: How do levels map to the 14 gates?

**Recommendation:** Each level is a strict superset of the level below. All 14 gates are assigned to exactly one level.

| Level | Gates That Fire | What Gets Checked |
|-------|-----------------|-------------------|
| **V0: Acknowledge** | Phase Complete | Agent reported "Phase: Complete" |
| **V1: Artifacts** | V0 + Synthesis, Handoff Content, Skill Output, Phase Gates, Constraint, Decision Patch Limit | Deliverables exist, constraints met, required phases reported |
| **V2: Evidence** | V1 + Test Evidence, Git Diff, Build, Accretion | Evidence of testing, code matches claims, project compiles, file sizes checked |
| **V3: Behavioral** | V2 + Visual Verification, Explain-Back (gate1), Behavioral (gate2) | Human observed behavior, orchestrator explains what was built, confirms behavior verified |

**Key design property:** The common case for each work type requires zero flags:

| Work Type | Default Level | Why |
|-----------|--------------|-----|
| Config change (update README, edit YAML) | V0 | No code, no artifacts needed |
| Investigation / architect | V1 | Produces knowledge artifacts, not tested code |
| Feature implementation | V2 | Code changes need evidence |
| Bug fix (Tier 1) | V2 | Must show the fix works |
| UI/behavioral feature | V3 | Human must observe behavior |

### Fork 4: How does this replace SkipConfig?

**Options:**
- A: Remove SkipConfig entirely, replace with level
- B: Keep SkipConfig as escape hatch, level as primary
- C: SkipConfig becomes "override within level" mechanism

**Substrate says:**
- Constraint "Friction Ceiling": Any design where `--force` is the happy path has failed
- Evidence: SkipConfig was built to replace `--force` — it's already one iteration better
- Principle "Gate Over Remind": The level IS the gate; skips should be rare exceptions

**Recommendation:** Option B (keep SkipConfig as escape hatch).

The verification level eliminates the *need* for most skip flags. But edge cases exist:

```bash
# Normal case: level handles everything, no flags needed
orch complete proj-123 --explain "Built JWT auth" --verified

# Edge case: V2 agent but build server down → skip one gate
orch complete proj-123 --skip-build --skip-reason "CI will catch build"
```

**The difference:** Today, skip flags are routine (every completion of an investigation needs `--skip-test-evidence` or the gate auto-skips). With levels, auto-skips are encoded in the level itself. Skip flags are only for "something unexpected happened."

**Migration path:**
1. Add `--verify-level` to `orch spawn` (stored in AGENT_MANIFEST.json)
2. `orch complete` reads level from manifest, selects gate set
3. Existing auto-skip logic becomes redundant (level already excludes those gates)
4. SkipConfig remains for override, but usage should drop to near-zero
5. Deprecate `--force` (already deprecated, keep the warning)

### Fork 5: How does tradeoff visibility fit within levels?

**Integrating with orch-go-1158:** The tradeoff visibility design recommended:
1. Model "Pressure Points" sections (prevent upstream)
2. SYNTHESIS.md "Architectural Choices" section (capture at decision time)
3. Completion pipeline surfaces tradeoff content (bring to orchestrator)

**How this maps to verification levels:**

- **Pressure Points** → Injected at spawn time via `kb context`. This is upstream of verification levels entirely — it's context, not a gate.
- **Architectural Choices in SYNTHESIS.md** → Part of V1 (Artifacts). At V1+, if the skill is architect/feature-impl/systematic-debugging and SYNTHESIS.md has an "Architectural Choices" section, surface it during completion. This is NOT a separate gate — it's content the existing synthesis gate extracts.
- **Tradeoff content in completion output** → Part of V3 (Behavioral). When the orchestrator writes the explain-back text, the completion pipeline includes any tradeoff content from SYNTHESIS.md. The orchestrator incorporates this into its explanation.

**The key integration insight:** Verification levels subsume tradeoff visibility because:
- V0-V1: No tradeoff gate (config changes and pure knowledge don't have implementation tradeoffs)
- V2: Tradeoff *capture* happens (Architectural Choices section in SYNTHESIS.md, if present)
- V3: Tradeoff *comprehension* is gated (orchestrator must explain, which surfaces any tradeoffs)

This avoids the anti-pattern from orch-go-1158: "Adding more required sections to SYNTHESIS.md and more completion gates is EXACTLY the anti-pattern Dylan warned about." Instead, tradeoff visibility is a natural *property* of higher verification levels, not an additional gate.

### Fork 6: How do --explain and --verify become judgment-driven?

**Current state:** `--explain` is required for Tier 1/2 work. `--verified` is required for Tier 1. These are mechanical requirements derived from issue type.

**Recommendation:** Keep the mechanical defaults but make them level-driven, not issue-type-driven.

- V0-V1: Neither required (acknowledge/artifacts don't need human explanation)
- V2: `--explain` required by default (orchestrator must show comprehension of evidence)
- V3: Both `--explain` and `--verified` required by default (behavioral confirmation)

**The judgment shift:** The orchestrator chooses the verification level at spawn time based on its understanding of the work. By choosing V2, it's saying "I expect to need to explain this." By choosing V3, it's saying "I expect to need to observe behavior." The flags don't *trigger* different behavior — they *record* the orchestrator's judgment.

When the orchestrator overrides downward (`--verify-level V0` for a config change), it's making a judgment: "this doesn't need explanation." When it overrides upward, it's making a judgment: "this needs more scrutiny than usual."

---

## Blocking Questions

### Q1: Should `--verify-level` be stored in AGENT_MANIFEST.json or in a separate workspace file?

- **Authority:** implementation
- **Subtype:** factual
- **What changes based on answer:** If AGENT_MANIFEST.json, it's alongside other spawn metadata (tier, skill, beads-id). If separate `.verify-level` file, it follows the existing dotfile pattern (`.tier`, `.spawn_mode`). AGENT_MANIFEST.json is preferred (modern, structured), but `.verify-level` would be simpler for shell scripts.

### Q2: Should the default verification level be overridable at spawn time only, or also at completion time?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If spawn-only, the level is set once and cannot be changed. This is simpler but inflexible (what if work turns out simpler than expected?). If also at completion, the orchestrator can say `orch complete --verify-level V0` to downgrade. This adds a flag to completion but provides flexibility. Recommendation: spawn-only for now, with skip flags as the override mechanism at completion time.

### Q3: Should V3 (Behavioral) be auto-assigned when web/ files are modified, or only when explicitly declared?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If auto-assigned, the system detects web changes and elevates to V3 (current behavior via visual verification gate). If explicit-only, the orchestrator must declare V3 at spawn time. Auto-assignment is safer but reduces orchestrator judgment. Recommendation: auto-elevate to V3 when web changes detected, but allow `--verify-level V2` to override downward.

---

## Synthesis: Recommended Approach

### The Core Design

**One concept — verification level — declared at spawn time, determines everything at completion time.**

```
Spawn Time:
  orch spawn feature-impl "add JWT auth" --issue proj-123
  → Level inferred: V2 (feature-impl + feature issue type)
  → Stored in AGENT_MANIFEST.json: {"verify_level": "V2"}

Completion Time:
  orch complete proj-123 --explain "Built JWT..." --verified
  → Level read from manifest: V2
  → Gates fired: Phase Complete, Synthesis, Skill Output, Phase Gates,
                  Constraint, Decision Patch, Test Evidence, Git Diff, Build, Accretion
  → Gates NOT fired: Visual Verification, Explain-Back (promoted by --explain),
                      Behavioral (promoted by --verified)
  → Result: PASS (no flags needed beyond --explain --verified which are Tier 1 requirements)
```

### The Four Levels

**V0: Acknowledge** — "Did the agent finish?"
- Gates: Phase Complete
- Typical work: Config changes, README updates, issue creation
- Default for: issue-creation skill, task/question issue types
- Human effort: None (daemon can auto-complete)

**V1: Artifacts** — "Are the deliverables present and coherent?"
- Gates: V0 + Synthesis, Handoff Content, Skill Output, Phase Gates, Constraint, Decision Patch Limit
- Typical work: Investigations, architect designs, research, codebase audits
- Default for: investigation/architect/research/codebase-audit skills
- Human effort: Review synthesis (orchestrator reads SYNTHESIS.md)

**V2: Evidence** — "Is there evidence the code works?"
- Gates: V1 + Test Evidence, Git Diff, Build, Accretion
- Typical work: Feature implementation, bug fixes, debugging
- Default for: feature-impl/systematic-debugging/reliability-testing skills, feature/bug/decision issue types
- Human effort: Explain what was built (--explain gate)

**V3: Behavioral** — "Did a human observe it working?"
- Gates: V2 + Visual Verification, Explain-Back (gate1), Behavioral (gate2)
- Typical work: UI features, user-facing changes, critical behavioral modifications
- Default for: Spawns with `--verify-level V3` or auto-elevated when web/ changes detected
- Human effort: Explain + confirm behavior observed (--explain + --verified)

### How This Solves Each Constraint

**1. LEVELS OVER GATES:** Level declared per-spawn → only relevant gates fire. A config change at V0 never triggers test evidence checks. A daemon bug fix at V2 does.

**2. JUDGMENT OVER MECHANICS:** `--verify-level` at spawn time IS the orchestrator's judgment call. `--explain` and `--verified` at completion time record the orchestrator's review, but the *level* determines what's required.

**3. FRICTION CEILING:** Common case requires zero skip flags. The level pre-selects the right gates. `--force` and `--skip-*` become rare exception handling, not routine.

### Integration with Tradeoff Visibility (orch-go-1158)

- **Model Pressure Points** → Upstream of levels (injected in SPAWN_CONTEXT via kb context). Not a gate.
- **Architectural Choices section** → Captured in SYNTHESIS.md at V1+. Not a separate gate — just content the existing synthesis check surfaces.
- **Completion-time tradeoff surfacing** → At V3, explain-back includes tradeoff content. At V2, tradeoff content is surfaced in completion summary but not gated.

This means tradeoff visibility is *naturally embedded* in the verification level hierarchy rather than being Yet Another Gate.

---

## Recommendations

⭐ **RECOMMENDED:** Implement the four-level verification system (V0-V3) with infer + override

- **Why:** Unifies three implicit systems (spawn tier, checkpoint tier, skill-based auto-skips) into one shared vocabulary. Eliminates routine skip flag usage. Preserves orchestrator judgment via override.
- **Trade-off:** Requires migration of existing auto-skip logic. Acceptable because the auto-skip logic already encodes the same information — we're making it explicit.
- **Expected outcome:** Orchestrator declares V0-V3 at spawn time. Completion fires only the relevant gates. `--force` usage drops to near-zero. "Verify this works" has exactly one meaning per level.

**Alternative: Keep current system, just document the implicit levels**
- **Pros:** Zero implementation work
- **Cons:** Doesn't solve the vocabulary gap — orchestrator still needs to reason about which skip flags to use
- **When to choose:** If the friction of the current system is tolerable

**Alternative: Replace 14 gates with 4 composite checks (one per level)**
- **Pros:** Dramatically simpler code
- **Cons:** Loses granularity of individual gate bypass
- **When to choose:** If SkipConfig turns out to be unused after levels are implemented (check after 4 weeks)

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the verification vocabulary gap between orchestrator and human
- Future changes to completion verification should respect the level hierarchy

**Suggested blocks keywords:**
- "verification level", "completion gates", "verify"
- "orch complete", "skip flags", "force"
- "V0", "V1", "V2", "V3"

---

## Implementation-Ready Output

### File Targets

1. **`pkg/spawn/config.go`** — Add `VerifyLevel string` to `Config` struct. Add `VerifyLevelDefaults` map (skill → level). Add `DefaultVerifyLevelForSkill(skill, issueType) string`.
2. **`pkg/spawn/config.go`** — Add constants: `VerifyV0 = "V0"`, `VerifyV1 = "V1"`, `VerifyV2 = "V2"`, `VerifyV3 = "V3"`.
3. **`pkg/spawn/manifest.go`** — Add `VerifyLevel` field to `AgentManifest` struct. Written at spawn, read at completion.
4. **`cmd/orch/spawn_cmd.go`** — Add `--verify-level` flag (optional, default inferred).
5. **`pkg/verify/check.go`** — Add `GatesForLevel(level string) []string` function. Modify `VerifyCompletionFullWithComments` to accept level and only run gates for that level.
6. **`cmd/orch/complete_cmd.go`** — Read verify level from manifest. Pass to verification. Remove need for most auto-skip conditionals.
7. **`pkg/checkpoint/checkpoint.go`** — `RequiresCheckpoint` and `RequiresGate2` can delegate to level (V2+ requires checkpoint, V3 requires gate2).

### Acceptance Criteria

1. `orch spawn investigation "explore X"` → AGENT_MANIFEST.json has `verify_level: V1`
2. `orch complete` on that agent runs only V0+V1 gates (no test evidence, no build, no visual)
3. `orch spawn feature-impl "add Y" --issue proj-123` → V2 (from skill + issue type)
4. `orch spawn feature-impl "update README" --verify-level V0` → V0 (explicit override)
5. `orch complete` on V0 agent requires only Phase: Complete
6. `--force` is never needed for well-configured spawns
7. Existing `--skip-*` flags still work for edge cases

### Out of Scope

- Changes to beads/bd
- New verification gates
- Removal of SkipConfig (keep as escape hatch)
- Automated tradeoff detection at spawn time (Layer 4 from orch-go-1158)

### Phasing

**Phase 1:** Add `VerifyLevel` to spawn config and manifest. Infer from skill+issue type. Store in AGENT_MANIFEST.json.

**Phase 2:** Add `GatesForLevel()` to `pkg/verify/check.go`. Modify completion to select gates based on level.

**Phase 3:** Add `--verify-level` flag to `orch spawn`. Surface level in `orch status` output.

**Phase 4:** Remove redundant auto-skip logic (now handled by levels). Monitor `--skip-*` usage to see if it drops.

---

## References

- `.kb/investigations/2026-02-20-audit-verification-infrastructure-end-to-end.md` — 14-gate inventory
- `.kb/investigations/2026-02-20-design-tradeoff-visibility-for-non-code-reading-orchestrator.md` — Tradeoff visibility design
- `~/orch-knowledge/kb/models/verifiability-first-development.md` — Verification paradigm
- `~/orch-knowledge/kb/models/control-plane-bootstrap.md` — Enforcement theater
- `cmd/orch/complete_cmd.go` — Current completion flow
- `pkg/verify/check.go` — Current 14-gate implementation
- `pkg/spawn/config.go` — Spawn tier defaults
- `pkg/checkpoint/checkpoint.go` — Checkpoint tier system
- `pkg/verify/escalation.go` — EscalationLevel (related but different concept: post-verification escalation)
