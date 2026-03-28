### Absent Signal Trap

Systems convert their own ignorance into positive signal. When a feedback channel is empty, a search returns nothing, or a metric can't go red, the default interpretation is "everything is fine" rather than "we don't know."

**The test:** "Is this a confirmed negative, or did nobody check?"

**What this means:**

- Empty feedback channels (zero reworks, zero failures, zero complaints) indicate unused channels, not perfection
- A metric that structurally cannot produce negative signal is worse than no metric — it displaces the uncertainty it should represent
- "No prior art found" from an agent that didn't search is not the same as "searched thoroughly, nothing exists"
- Formulas and systems must distinguish "checked and found nothing" from "didn't check"

**What this rejects:**

- "Zero reworks means high quality" (maybe nobody has a mechanism to request rework)
- "No bugs reported" (maybe reporting is harder than working around the bug)
- "The metric is green" (maybe the metric can't go red)
- "No prior art found" (maybe the agent didn't look)
- "100% success rate" (maybe success is defined as "completed" and completion is guaranteed by exhaustive fallback)

**The failure mode:** orch-go's orient display showed three mutually-reinforcing green metrics — 75.7% success rate, 83.0% ground-truth-adjusted rate, 100% merge rate. The ground-truth adjustment blended self-reported success with rework rate at 70/30 weight. With 0 reworks across 817 completions, the formula computed `0.7 × 0.757 + 0.3 × 1.0 = 0.830` — a +7.3 percentage point inflation. The absence of rework data was converted into evidence of quality. The merge rate was tautological in a single-branch workflow. All three metrics were structurally incapable of going red. Together they produced a coherent picture of health while the system had a 91.8% investigation orphan rate and 54.4% decision audit false positives.

Separately: five agent briefs in a single session shared a pattern none of them named — the system routinely treats "didn't check" as "nothing there." Agents reporting "no prior art" without searching. Investigations concluding "no related work" without querying the knowledge base. The system's default is optimism about its own completeness.

**Why this is distinct from Provenance:** Provenance says "trace conclusions to external evidence." This principle addresses the specific case where there IS no evidence — and the system fills that void with a positive signal rather than marking uncertainty. Provenance tells you what to do with evidence. This tells you what to do without it.

**Why this is distinct from Evidence Hierarchy:** Evidence Hierarchy ranks the quality of sources that exist. This principle is about what happens when no source is consulted at all — the system treats the gap as if it consulted a source and got a positive result.

**Why this is distinct from AI Validation Loops:** AI Validation Loops addresses trust in AI-generated explanations (evidence exists but is unreliable). This addresses the absence of any evidence being treated as positive evidence.

**The broader pattern:** This isn't just about metrics. It's about how any system — measurement, knowledge, search, feedback — interprets silence. The structural fix is the same everywhere: when a channel returns nothing, label it "no data" not "all clear." Make the system's ignorance visible rather than converting it into confidence.

**Evidence:**
- `GroundTruthAdjustedRate()` in `pkg/daemon/allocation.go` — 0 reworks × 0.3 weight = +7.3pp false inflation (deleted per trust audit, Mar 2026)
- Skill inference "100% success rate" — exhaustive fallback chain (label → title → description → type) guarantees assignment; 69% fall through to coarsest signal (type-based); 0 mechanism to measure correctness (Mar 2026)
- Decision audit v1 — 54.4% false positives from checking file-existence for architectural principles that don't reference files (deleted and rebuilt, Mar 2026)
- Five agent briefs with "no prior art" claims where no search was performed (epistemic dishonesty thread, Mar 2026)

**Provenance:** Measurement honesty model invariant #2 (Mar 2026), epistemic dishonesty thread (Mar 2026), decision audit template case (Mar 2026). Pattern observed independently in metrics (false confidence), knowledge queries (no-search-no-results), and agent self-reports (claiming thoroughness without evidence).
