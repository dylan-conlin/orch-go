## Summary (D.E.K.N.)

**Delta:** 'Probe' was used for 3 distinct concepts across skills; renamed decision-navigation probes to 'spike' and epic model probes to 'scouting', keeping 'probe' exclusively for model-scoped confirmatory tests.

**Evidence:** grep -r 'probe' across 9 skill files confirmed 3 distinct meanings; post-edit grep shows consistent usage — 'probe' only in investigation/orchestrator skills (model-scoped), 'spike' in decision-navigation/architect/design-session, 'scouting' in meta-orchestrator-interface.

**Knowledge:** When the same term means different things across skills loaded together, workers receive conflicting definitions. Each concept needs its own term.

**Next:** Close. No further work needed — terminology is now unambiguous across all skill files.

**Authority:** implementation - Documentation-only terminology change within existing patterns.

---

# Investigation: Disambiguate Probe Terminology Across Skills

**Question:** The term 'probe' is used for 3 distinct concepts across skills — how do we disambiguate them so workers don't receive conflicting definitions?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Three distinct meanings of 'probe' across skills

**Evidence:** grep across all 9 skill files containing 'probe' revealed:

1. **Model-scoped probes** (investigation, orchestrator skills): File artifacts in `.kb/models/{name}/probes/`, confirmatory tests of model claims using `.orch/templates/PROBE.md`. Has established directory structure, templates, and workflow.

2. **Decision-navigation probes** (decision-navigation, architect, design-session skills): Ad-hoc experiments to resolve unknown decision forks. Small, time-boxed experiments to surface constraints when substrate consultation returns nothing useful.

3. **Epic model probes** (meta-orchestrator-interface reference): Exploratory investigations during the "Probing" phase of the Epic Model (Probing → Forming → Ready). Used to build understanding of complex problems.

**Source:**
- `/Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md` (meaning 1)
- `/Users/dylanconlin/.claude/skills/shared/decision-navigation/SKILL.md` (meaning 2)
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/reference/meta-orchestrator-interface.md` (meaning 3)

**Significance:** Workers spawned with both investigation and architect skills receive both meanings of 'probe' simultaneously, creating ambiguity about what artifact to produce.

---

### Finding 2: Rename mapping preserves semantics

**Evidence:** Applied terminology changes:

| Original Term | New Term | Rationale |
|--------------|----------|-----------|
| Probe (model-scoped) | **probe** (kept) | Most established usage — has directory structure, templates, file naming convention |
| Probe (decision-navigation) | **spike** | Industry-standard term for time-boxed experiments to resolve uncertainty |
| Probing phase (epic model) | **scouting** | Conveys exploration/reconnaissance, which matches the purpose of sending investigations to build understanding |

**Source:** All edited files listed below.

**Significance:** Each term now has exactly one meaning across all skill files.

---

### Finding 3: Files edited across both source and deployed locations

**Evidence:** Edited 12 files total:

**Decision-navigation (deployed only — no source exists):**
- `~/.claude/skills/shared/decision-navigation/SKILL.md`

**Architect (source + deployed):**
- `orch-knowledge/skills/src/worker/architect/.skillc/SKILL.md`
- `orch-knowledge/skills/src/worker/architect/.skillc/SKILL.md.template`
- `~/.claude/skills/worker/architect/SKILL.md`
- `~/.claude/skills/src/worker/architect/SKILL.md`

**Design-session (source + deployed):**
- `orch-knowledge/skills/src/worker/design-session/.skillc/SKILL.md`
- `orch-knowledge/skills/src/worker/design-session/.skillc/SKILL.md.template`
- `~/.claude/skills/worker/design-session/SKILL.md`
- `~/.claude/skills/src/worker/design-session/SKILL.md`

**Meta-orchestrator-interface (deployed only — source removed):**
- `~/.claude/skills/meta/orchestrator/reference/meta-orchestrator-interface.md`
- `~/.claude/skills/reference/meta-orchestrator-interface.md`

**Source:** git diff of all edited files.

**Significance:** Both source (.skillc) and deployed files updated to prevent drift on next `skillc deploy`.

---

## Synthesis

**Key Insights:**

1. **Model-scoped probes are the most established concept** — They have directory structure, templates, a full workflow, and are core to the investigation/orchestrator skills. Keeping "probe" for this meaning was the right default.

2. **"Spike" is industry-standard** — The decision-navigation concept of "small experiment to resolve uncertainty" maps exactly to the Agile concept of a spike. Workers will immediately understand what it means.

3. **"Scouting" captures the epic model's exploration phase** — The Probing→Forming→Ready phases describe a reconnaissance pattern, where "scouting" accurately conveys sending investigations to gather intel before forming a mental model.

**Answer to Investigation Question:**

The disambiguation was achieved by keeping "probe" exclusively for model-scoped confirmatory tests (investigation/orchestrator skills), renaming decision-navigation probes to "spike" (decision-navigation/architect/design-session skills), and renaming epic model probes to "scouting" (meta-orchestrator-interface). Post-edit verification shows no skill file uses "probe" for more than one meaning.

---

## Structured Uncertainty

**What's tested:**

- ✅ Post-edit grep shows 'probe' only in 3 files: decision-navigation (disambiguation note), investigation (model-scoped), orchestrator (model-scoped)
- ✅ Architect and design-session files have zero 'probe' references remaining
- ✅ Meta-orchestrator-interface files have zero 'probe' references remaining

**What's untested:**

- ⚠️ Whether `skillc deploy` will correctly pick up source changes on next run (source files edited but not compiled)
- ⚠️ Whether the epic-model.md template at `~/.orch/templates/epic-model.md` also uses "probing" terminology (not checked)

**What would change this:**

- If additional skill files or templates reference "probe" in the decision-navigation or epic model sense
- If `skillc deploy` overwrites deployed changes before source changes are compiled

---

## References

**Files Examined:**
- `~/.claude/skills/shared/decision-navigation/SKILL.md` - Decision-navigation probe usage
- `~/.claude/skills/worker/investigation/SKILL.md` - Model-scoped probe usage
- `~/.claude/skills/worker/architect/SKILL.md` - Cross-reference to decision-navigation probes
- `~/.claude/skills/worker/design-session/SKILL.md` - Cross-reference to decision-navigation probes
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Model-scoped probe usage
- `~/.claude/skills/meta/orchestrator/reference/meta-orchestrator-interface.md` - Epic model probe usage

**Commands Run:**
```bash
# Find all skill files mentioning 'probe'
grep -ri 'probe' ~/.claude/skills/ --include='*.md' -l

# Verify post-edit: only model-scoped probe files remain
grep -ri 'probe' ~/.claude/skills/ --include='*.md' -l
```
