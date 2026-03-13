### Understanding Through Engagement

Model-building requires direct engagement with findings. You can spawn work to gather facts, but synthesis into coherent models requires the cross-agent context and judgment that only the integrating level has.

**The test:** "Am I trying to spawn someone to understand this for me?"

**What this means:**

- Workers gather facts through investigations (spawnable)
- Orchestrators synthesize models through direct engagement (not spawnable)
- Models are understanding artifacts (`.kb/models/`), distinct from work (epics) and procedures (guides)
- Epic readiness = model completeness, not task enumeration
- Synthesis produces queryable understanding (constraints become visible, discussable)

**What this rejects:**

- Spawning architects to "think for me"
- Treating synthesis as spawnable task work
- Epic = task list (epic should reference model, not contain it)
- Understanding = reading reports (understanding requires engagement with findings)
- Synthesis without externalization (model must be written, not just held in head)

**Why distinct from Pressure Over Compensation:**

- Pressure Over Compensation: Don't fill gaps manually (create pressure for system to fix)
- Understanding Through Engagement: Don't delegate model-building (requires cross-agent context orchestrator has)

You could follow Pressure Over Compensation but still try to spawn architects to think. Understanding Through Engagement says that won't work - synthesis requires the vantage point orchestrator uniquely has.

**The value of models:**

Models create **surface area for questions** by making implicit constraints explicit.

Example: Model states "OpenCode doesn't expose session state via HTTP API" as constraint. This enables strategic question: "Should we add that endpoint?" Without the model, constraint stays buried in code, discovered during debugging, forgotten after.
