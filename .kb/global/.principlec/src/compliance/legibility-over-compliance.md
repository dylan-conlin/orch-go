### Legibility Over Compliance

A skill should make agent behavior readable to the human supervisor, not try to make the agent always correct. The human is the failure detection mechanism.

**The test:** "Does this gate/rule make the agent's behavior more visible to the human, or just more likely to be correct?"

**What this means:**

- Compliance-optimized skills add gates. Each gate adds ceremony. Ceremony obscures signal.
- Legibility-optimized skills strip gates and add transparency — mode declarations, intent naming, explicit uncertainty.
- The agent operating under a grammar can't detect its own grammar failures (it experiences constraint as judgment). The human observing from outside the grammar is the failure detector.
- A heavy compliance process that hides failures from the human is worse than a light process that makes failures visible.

**What this rejects:**

- "More gates = safer" (gates add noise that hides the signal)
- "The skill should prevent wrong behavior" (it can't — probabilistic constraints have structural limits)
- "If we add enough rules, the agent will always behave correctly" (see Identity is Not Behavior)

**The failure mode:** The orchestrator skill accumulated compliance gates: delegation checklists, pre-response verification, anti-pattern tables, self-check protocols. Each gate was locally justified. Together, they produced so much process ceremony that the human couldn't tell whether the agent was following the process or hiding behind it. The agent doubled down on visible compliance (announcing what it would do) while violating constraints in the gaps between announcements.

**Why distinct from Gate Over Remind:** GoR says gates block progress, reminders don't. This principle says *even gates* can degrade the system if they optimize for compliance over legibility. A gate that makes agent behavior opaque is worse than a reminder that keeps behavior visible.

**Why distinct from Identity is Not Behavior:** IiNB explains why instructions fail (identity is additive, constraints are subtractive). This principle prescribes what to optimize for instead: make the agent's behavior readable so the human can intervene, rather than trying to make the agent always correct.

**Evidence:** Feb 2026 skill iterations added 14 commits of compliance gates in 18 days. Behavioral testing showed negligible improvement (39% vs 38% vs 30% bare). Meanwhile, the heavy process made it harder for the human to detect actual failures — the agent's responses were full of compliance theater (pre-response checklists, mode declarations) that obscured whether it was actually delegating or just announcing that it would.

**Provenance:** Grammar Design Discipline synthesis (Mar 1 2026), corroborated by revert spiral investigation and behavioral testing baseline.
