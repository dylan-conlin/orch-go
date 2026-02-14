## Summary (D.E.K.N.)

**Delta:** Added model/probe awareness to 4 understanding-producing skills (codebase-audit, systematic-debugging, research, reliability-testing) that previously only produced standalone investigations.

**Evidence:** Before: only investigation skill had probe routing. After: grep for "model-claim markers" shows 7 skills with probe awareness (investigation, orchestrator, codebase-audit, systematic-debugging, research, reliability-testing).

**Knowledge:** The probe routing pattern is simple and portable: check SPAWN_CONTEXT for `### Models (synthesized understanding)` markers → if present, route to `.kb/models/{model-name}/probes/` instead of standalone investigation.

**Next:** Close. All 4 skills updated, built, and deployed via skillc.

**Authority:** implementation - Skill content updates within established patterns, no architectural changes.

---

# Investigation: Expand Model Probe Awareness Beyond Investigation Skill

**Question:** Which skills produce understanding artifacts and how should they route to probes when a model exists?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-02-13-inv-disambiguate-probe-terminology-across-skills.md | extends | yes | None |
| .kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md | extends | yes | None |

---

## Findings

### Finding 1: Only 3 of 15+ skills had probe awareness

**Evidence:** Before this change, only `investigation`, `orchestrator`, and `decision-navigation` (now renamed to spikes) had model/probe routing logic.

**Source:** Grep for "model-claim markers" across `~/.claude/skills/` before changes showed hits only in investigation and orchestrator SKILL.md files.

**Significance:** Understanding-producing skills like codebase-audit, systematic-debugging, research, and reliability-testing were silently producing standalone investigations even when a model existed for the domain, missing opportunities to confirm/contradict/extend model claims.

---

### Finding 2: The investigation skill's pattern is simple and portable

**Evidence:** The reference implementation in `investigation/.skillc/intro.md` and `workflow.md` uses a 3-step detection:
1. Find `### Models (synthesized understanding)` section in SPAWN_CONTEXT
2. Check for markers: `- Summary:`, `- Critical Invariants:`, `- Why This Fails:`
3. If found → Probe Mode, if absent → Investigation Mode

**Source:** `~/orch-knowledge/skills/src/worker/investigation/.skillc/intro.md:6-29`, `workflow.md:19-27`

**Significance:** The pattern is purely documentation-based (check SPAWN_CONTEXT markers, route output) and can be added to any understanding-producing skill without code changes.

---

### Finding 3: Each skill needed contextually appropriate probe guidance

**Evidence:** While the detection logic is identical across skills, the "how to use probe mode" guidance needed skill-specific framing:
- **codebase-audit**: audit findings confirm/contradict model invariants
- **systematic-debugging**: debugging findings relate to model failure modes
- **research**: external research extends model understanding of alternatives
- **reliability-testing**: reliability data confirms/contradicts system behavior claims

**Source:** Each skill's updated source file includes a tailored example.

**Significance:** Copy-pasting the investigation skill's probe section verbatim would miss the contextual connection between each skill's natural output and model claims.

---

## Synthesis

**Key Insights:**

1. **Portable pattern** - The model-claim detection is a simple marker check that can be added to any skill producing understanding artifacts.

2. **4 skills updated** - codebase-audit, systematic-debugging, research, reliability-testing now all check for model context before creating artifacts.

3. **Consistent terminology** - All updated skills use "probe" exclusively for model-scoped confirmatory tests, consistent with the orch-go-wh6 disambiguation.

**Answer to Investigation Question:**

Four understanding-producing skills needed model awareness. Each was updated with a "Model Awareness (Probe vs Investigation Routing)" section following the investigation skill's reference pattern. The changes are source-level (.skillc files in orch-knowledge) and were deployed via `skillc deploy`.

---

## Structured Uncertainty

**What's tested:**

- All 4 skill source files updated and compile successfully via `skillc build`
- All 20 skills deployed successfully via `skillc deploy`
- Grep confirms "model-claim markers" appears in all 4 updated deployed skills

**What's untested:**

- Actual agent behavior when spawned with these skills + model context (would need a spawn test)
- Whether the systematic-debugging token budget warning (125% of 5000) causes issues at spawn time

**What would change this:**

- If the SPAWN_CONTEXT marker format changes, all skills would need updating
- If new understanding-producing skills are added, they'd need the same pattern

---

## References

**Files Modified:**
- `~/orch-knowledge/skills/src/worker/systematic-debugging/.skillc/investigation-file.md` - Added Model Awareness section
- `~/orch-knowledge/skills/src/worker/reliability-testing/.skillc/phases.md` - Added probe routing to Phase 3
- `~/orch-knowledge/skills/src/worker/research/.skillc/SKILL.md.template` - Added Model Awareness section
- `~/orch-knowledge/skills/src/worker/codebase-audit/.skillc/phases/common-overview.md` - Added Model Awareness section
