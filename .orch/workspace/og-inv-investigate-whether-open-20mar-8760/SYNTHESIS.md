# Session Synthesis

**Agent:** og-inv-investigate-whether-open-20mar-8760
**Issue:** orch-go-bhm8g
**Outcome:** success

---

## Plain-Language Summary

Scanned all 41 KB models and 20+ decisions to determine whether "open-loop systems" (action disconnected from observation) is a unifying model that replaces measurement-honesty, knowledge-accretion, agent-trust-enforcement, and architectural-enforcement. **The answer is no — it's a genuine cross-cutting pattern that appears in ~75% of models, but it covers only ~35% of what any individual model explains.** Each target model contains substantial domain-specific content (signal quality taxonomies, coordination theory, security architecture, enforcement design) that open-loop cannot express. The pattern is best used as a diagnostic lens ("is the loop between action and observation closed?") rather than a standalone model.

The most actionable finding: 14 of 16 identified open loops are missing **sensors** specifically. Orch-go has strong actuators (spawn, complete, gates, hooks) and clear reference signals (CLAUDE.md, thresholds, skills). The systematic gap is observing consequences of actions — which maps directly to harness-engineering's "enforcement without measurement is theological."

---

## TLDR

Open-loop systems is a real cross-cutting pattern (75% of KB models) but doesn't subsume existing models — each model has 60-75% domain-specific content that open-loop can't express. Best as a diagnostic lens added to measurement-honesty, not a standalone model. Most open loops are missing sensors (14/16), not actuators or reference signals.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-20-inv-investigate-whether-open-loop-systems.md` — Full investigation with model scan, control-theory mapping, subsumption analysis, and 3 novel instances

### Commits
- (pending)

---

## Evidence (What Was Observed)

- **41 models scanned**: 12 YES (core is open-loop), 18 PARTIALLY, 5 NO, 4 INVERSE (describe closing loops)
- **~21 of 30 open-loop models already name the pattern** using domain-specific vocabulary (drift, convention decay, TOCTOU, self-referential reporting, absent-signal trap, constraint dilution)
- **4 target models assessed for subsumption**: measurement-honesty (~60% overlap), architectural-enforcement (~40%), knowledge-accretion (~30%), agent-trust-enforcement (~25%)
- **Control theory mapping**: Most open loops are missing sensors (14/16). System has strong actuators and reference signals.
- **3 novel instances found**: deployment migration gaps, verification level coordination, recommendation stranding
- **Decisions scan**: 8 additional open-loop instances found in decisions, but ~5 of those have already been fixed by subsequent decisions (showing Dylan has been empirically closing loops)

### Tests Run
```
N/A — pure investigation, no code changes
```

---

## Architectural Choices

### Chose "diagnostic lens" over "standalone model"
- **What I chose:** Recommend open-loop as a diagnostic addition to measurement-honesty rather than a new model
- **What I rejected:** Creating `.kb/models/open-loop-systems/model.md`
- **Why:** Only 3 novel instances (barely meets threshold), ~70% of instances already named in existing models, subsumption of target models is 25-60% (not sufficient). A standalone model would duplicate most of measurement-honesty.
- **Risk accepted:** The 3 novel instances (deployment migration, verification coordination, recommendation stranding) may not get addressed without their own model attractor.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-20-inv-investigate-whether-open-loop-systems.md` — Full investigation

### Constraints Discovered
- Open-loop is tautological at the limit: "failures persist because consequences aren't observed" is close to circular. The useful version is the control-theory mapping: WHICH component is missing (sensor, comparator, actuator, reference)?
- The sensor gap finding is the most actionable insight: orch-go's architecture is actuator-rich but sensor-poor.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file, SYNTHESIS.md)
- [x] Investigation file has `Status: Complete`
- [x] D.E.K.N. filled
- [ ] Ready for `orch complete orch-go-bhm8g`

### Follow-up recommendations (discovered work):
1. Add "Is the Loop Closed?" diagnostic section to measurement-honesty model
2. Create beads issues for the 3 novel open-loop instances (deployment migration, verification coordination, recommendation stranding)
3. Update the thread with control-theory mapping

---

## Unexplored Questions

- Does the "14/16 missing sensors" finding suggest a specific infrastructure investment (a generic "consequence sensor" pattern)?
- How does the sensor gap relate to the daemon's periodic tasks? Many periodic tasks ARE sensors (health checks, staleness detection) — is the daemon already the sensor layer, just incompletely?
- The entropy-spiral model's "self-referential reporting" is the deepest treatment of closed-loop failure — should it be the home for the control-theory mapping instead of measurement-honesty?

---

## Friction

Friction: none — models were well-organized and scannable.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-investigate-whether-open-20mar-8760/`
**Investigation:** `.kb/investigations/2026-03-20-inv-investigate-whether-open-loop-systems.md`
**Beads:** `bd show orch-go-bhm8g`
