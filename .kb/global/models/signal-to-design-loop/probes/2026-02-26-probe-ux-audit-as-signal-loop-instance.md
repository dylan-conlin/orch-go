# Probe: UX Audit Skill as Signal-to-Design Loop Instance

**Date:** 2026-02-26
**Status:** Complete
**Model:** signal-to-design-loop
**Triggered by:** toolshed-8sj — Design ux-audit skill for structured UI/UX auditing

---

## Question

Does the signal-to-design-loop model predict the design constraints and failure modes of a UX audit skill? Specifically: do the model's five stages (Capture → Accumulation → Clustering → Synthesis → Design Response) map cleanly to UI/UX auditing, and do the model's four failure modes predict real design pitfalls?

**Model claims tested:**
- Claim: "Individual signals are noise. Clustered signals are design pressure. The difference is metadata that makes signals groupable."
- Claim (Failure Mode 1): Capture friction kills the loop — separate reflection steps get skipped
- Claim (Failure Mode 2): Clustering resolution too low — natural language similarity fails; need explicit metadata keys
- Claim (Failure Mode 3): Synthesis without authority produces reports nobody acts on
- Claim (Stage requirements): Capture must be embedded in workflow; clustering needs constrained vocabulary

---

## What I Tested

### 1. Prior UX Audit Analysis (toolshed-88)

Analyzed the ad-hoc UX audit from 2026-02-22 (`.kb/investigations/2026-02-22-inv-usability-audit-toolshed-expedite-dashboard.md`) to see how signals were captured, clustered, and synthesized without a formal skill.

The prior audit produced 13 findings using Playwright MCP. Method was: navigate → screenshot → interact → resize → document. Findings were grouped by severity (Critical/Important/Suggestion) but NOT by dimension (layout, responsive, data, interaction).

### 2. Codebase Audit Skill Pattern Analysis

Analyzed the codebase-audit skill (closest structural model) to understand how the dimension-based pattern implements the signal-to-design loop. The skill uses:
- **Dimensions** as clustering keys (security, performance, tests, architecture, organizational)
- **Pattern searches** as structured signal capture (grep commands, file analysis)
- **ROI-prioritized recommendations** as synthesis output
- **Beads issues** as design response tracking

### 3. Mapping the Five Stages to UX Auditing

| Stage | Codebase Audit | UX Audit (proposed) |
|-------|---------------|---------------------|
| **Signal Capture** | grep/glob pattern search → findings | Playwright snapshot/screenshot → findings |
| **Accumulation** | Investigation file in .kb/ | Investigation file in .kb/ (same) |
| **Clustering** | Explicit dimension tag (security, performance, etc.) | Explicit dimension tag (visual, responsive, a11y, etc.) |
| **Synthesis** | ROI-prioritized recommendations | Severity-prioritized recommendations |
| **Design Response** | Beads issues created → implementation | Beads issues created → implementation |

---

## What I Observed

### The model maps cleanly with one extension needed

**Confirms: Five stages apply directly.** The prior ad-hoc audit (toolshed-88) naturally followed all five stages, even without a skill structuring the work. The auditor captured signals (13 findings with screenshots), accumulated them in an investigation file, clustered by severity, synthesized recommendations, and the orchestrator created beads issues for design response. The loop worked.

**Confirms: Failure Mode 1 predicts the "quick mode" need.** The prior audit took an unstructured, comprehensive approach. If every UX audit requires 2-4 hours of methodical dimension-by-dimension analysis, capture friction will kill adoption. The model correctly predicts that capture must be "embedded in existing workflow" — arguing for a lightweight quick mode (30 min surface scan) alongside full audits.

**Confirms: Failure Mode 2 predicts the dimension taxonomy need.** The prior audit clustered by severity but NOT by dimension. This made it impossible to compare "responsive behavior" across audits or track whether responsive issues decreased over time. The model's prediction — "natural language similarity fails without explicit metadata keys" — is exactly the design pressure for explicit UX dimensions (visual, responsive, a11y, data, navigation, interactive-states).

**Extends: UX auditing adds a temporal dimension the model doesn't address.** The signal-to-design-loop model describes clustering across concurrent signals. UX auditing also needs clustering across TIME — comparing the same page's audit results from February vs. March. This requires:
- Consistent dimension taxonomy (same categories across audits)
- Baseline metrics (quantitative measures, not just qualitative findings)
- Page identity (auditing the same page, same viewport, same auth state)

This temporal clustering is a new instance type the model should document.

**Extends: Visual evidence changes the accumulation stage.** The model's accumulation examples (`.kb/` artifacts, gap-tracker.json, beads issues) are all text-based. UX auditing accumulates screenshots alongside text, creating a dual-medium archive. Screenshots provide evidence that text cannot — but they also don't cluster well (you can't grep screenshots). This argues for text metadata AS the clustering layer, with screenshots AS supplementary evidence.

---

## Model Impact

**Confirms:**
- Five-stage loop applies to UI/UX auditing (new instance type)
- Failure Mode 1 correctly predicts need for low-friction quick mode
- Failure Mode 2 correctly predicts need for explicit dimension taxonomy as clustering keys
- "Constrained vocabulary" principle applies: dimensions must be enumerated, not free-text

**Extends:**
- **Temporal clustering**: Model should document that some signal-to-design loops need to track signal frequency OVER TIME for the same target (page, component, feature). Current model only describes clustering across concurrent signals.
- **Dual-medium accumulation**: When signals include visual evidence (screenshots, recordings), accumulation needs both text metadata (for clustering) and visual artifacts (for evidence). The model's accumulation stage should note that clustering happens on metadata, not on the raw signal medium.

**Recommendation for model update:**
- Add "UX Audit Skill" as Known Instance 4 in the model
- Add temporal clustering to the Clustering stage description
- Add dual-medium accumulation note
