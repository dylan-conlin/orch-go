# Session Synthesis

**Agent:** og-arch-evaluate-whether-claude-24feb-22f4
**Issue:** orch-go-1192
**Outcome:** success

---

## Plain-Language Summary

Claude Code has a built-in "plan mode" where the agent writes a plan, a human approves it, and then the agent executes. An architect agent spontaneously used this. We evaluated whether this should become the standard planning step for all feature-impl workers.

**The answer is no.** Plan mode requires a human sitting at the keyboard to approve the plan — but most agents are spawned headlessly by the daemon with no human present. They would hang forever waiting for approval. On top of that, plan mode's default behavior clears the agent's conversation context on approval, which would erase all the SPAWN_CONTEXT instructions, skill guidance, and beads tracking setup. And during plan mode, agents can't run bash commands, which means they can't report phases via `bd comment` — making them invisible to the orchestrator.

Feature-impl's existing Planning → Investigation → Design → Implementation flow is architecturally better for orchestrated agents because it works headlessly, keeps context, produces durable artifacts in known locations, and stays visible to the orchestrator throughout.

If premature implementation (agents jumping to coding before planning) becomes a recurring problem, the fix is to strengthen the prompt-level instructions in the skill, not to bolt on plan mode.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for acceptance criteria and evidence.

---

## Delta (What Changed)

### Files Created
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md` — Probe confirming lifecycle model's continuous observability assumption
- `.kb/investigations/2026-02-24-design-evaluate-plan-mode-feature-impl-integration.md` — Design investigation with recommendation

### Commits
- (pending — will commit all files together)

---

## Evidence (What Was Observed)

- Plan mode mechanics researched via web (Armin Ronacher analysis, Claude Code GitHub issues, system prompt extraction repos)
- Feature-impl skill read (574 lines, 6 configured phases, prompt-driven planning)
- Daemon spawn path traced: `daemon.go → SpawnWork() → orch work <beadsID>` (fully headless)
- SPAWN_CONTEXT template analyzed (context.go, 1505 lines) — no plan mode references
- Principles consulted: Gate Over Remind, Surfacing Over Browsing, Session Amnesia

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md` — Documents a new class of incompatibility: Claude Code features designed for interactive single-user operation that break in orchestrated headless environments

### Decisions Made
- Do not integrate Claude Code plan mode into feature-impl (recommendation, pending promotion to decision if accepted)

### Constraints Discovered
- Plan mode's ExitPlanMode has no programmatic bypass — requires interactive human approval
- Plan mode's DEFAULT approval option clears conversation context (open issue #18599 on Claude Code)
- Plan mode blocks ALL bash execution, not just writes — prevents bd comment phase reporting

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (probe, investigation, SYNTHESIS.md, VERIFICATION_SPEC.yaml)
- [x] Recommendation is clear and actionable
- [x] No implementation changes needed (this is a design evaluation)
- [x] Ready for `orch complete orch-go-1192`

---

## Discovered Work

No discovered work — this was a self-contained evaluation with no implementation needed.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-evaluate-whether-claude-24feb-22f4/`
**Beads:** `bd show orch-go-1192`
