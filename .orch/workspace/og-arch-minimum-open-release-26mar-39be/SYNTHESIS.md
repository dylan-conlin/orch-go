# Session Synthesis

**Agent:** og-arch-minimum-open-release-26mar-39be
**Issue:** orch-go-ehper
**Duration:** 2026-03-26
**Outcome:** success

---

## Plain-Language Summary

The openness boundary matrix recommended publishing artifact formats as Wave 1, claiming they're "self-documenting and independently adoptable." I tested that claim by scoring all 7 artifact types for standalone comprehensibility — a new user reading raw artifacts without the CLI. Average score: 2.6 out of 5. Threads have 7 unexplained IDs in their frontmatter. The KB README documents 4 of 7+ artifact types, missing threads and briefs entirely. Nowhere in the system is the composition model — how artifacts relate to each other (thread → investigation → probe → model → brief) — documented for a new user.

This means Wave 1 alone doesn't teach the method. The minimum open release needs four things shipped together: (1) artifact formats with expanded documentation, (2) a method composition guide explaining the cycle, (3) thread commands as the first-contact entry point, and (4) curated real examples with IDs cleaned. The init flow also needs to lead with "start a thread" instead of "spawn an agent" — currently the first thing a user is told to do contradicts the product decision.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — investigation produced, probe tested and merged, 3 implementation issues created.

---

## TLDR

Defined the minimum open release as: artifact formats + composition guide + thread CLI + curated examples, shipped with a comprehension-first init. Contradicted the matrix's assumption that formats alone are self-documenting (scored 2.6/5). Created 3 follow-up implementation issues.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-minimum-open-release-bundle.md` — Full architect investigation with 4-layer bundle definition
- `.kb/models/knowledge-accretion/probes/2026-03-26-probe-minimum-open-release-bundle-definition.md` — Probe testing artifact self-documentation claim

### Files Modified
- `.kb/models/knowledge-accretion/model.md` — Merged probe finding about composition model invisibility as contributing factor to orphan rates; updated Last Updated date; added probe reference

### Commits
- (pending)

---

## Evidence (What Was Observed)

- Artifact comprehensibility scores: thread 2/5, brief 3.5/5, model 2/5, investigation 3/5, probe 2/5, decision 4/5, KB README 2/5 (avg 2.6/5)
- 29+ orch-specific references across artifacts that are noise without system context
- Init "Next steps" leads with "Spawn an agent" while README leads with "Threads: the organizing spine"
- KB README documents 4 of 7+ artifact types; threads and briefs missing
- No document in the system explains the composition cycle

### Tests Run
```bash
# Design investigation — no code changes, no tests applicable
# Validated by: reading real artifacts as a new user would, auditing init flow against README
```

---

## Architectural Choices

### Full binary with front-door change vs. separate `kb` binary
- **What I chose:** Ship the full orch binary, change the init flow and documentation to lead with comprehension
- **What I rejected:** Extracting a separate `kb` binary for the knowledge layer
- **Why:** A separate binary fragments the CLI surface and adds packaging complexity. The front-door change (init output, help ordering, method guide) achieves the same perception shift with less work. Confirmed by orch-go-wgkj4 brief: "The front door change is a weekend. The ratio change is a quarter."
- **Risk accepted:** Users who browse the CLI help will still see spawn/daemon commands. Mitigation: method guide and README frame these as "substrate."

### Cleaned real examples vs. synthetic examples
- **What I chose:** Curate real artifacts from orch-go, clean orch-specific IDs, add inline explanations
- **What I rejected:** Fully synthetic examples
- **Why:** Real examples are evidence the method works. Synthetic examples feel like marketing.
- **Risk accepted:** Cleaned examples may lose authenticity. Mitigation: keep the content real, only replace IDs with descriptive labels.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-design-minimum-open-release-bundle.md` — Minimum release bundle definition
- `.kb/models/knowledge-accretion/probes/2026-03-26-probe-minimum-open-release-bundle-definition.md` — Artifact self-documentation probe

### Decisions Made
- Minimum release = artifact formats + composition guide + thread CLI + curated examples (not formats alone)
- Entry point: full binary with comprehension-first init (not docs-only or separate kb binary)
- Front-door change, not ratio change (implementation scope: weekend, not quarter)

### Constraints Discovered
- Artifact formats are not self-documenting (2.6/5 avg); Wave 1 alone insufficient
- Init flow contradicts product decision (leads with spawn, not threads)
- Composition model is the critical missing link for first-contact comprehension

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation, probe, model merge, issues created)
- [x] Design investigation with recommendations produced
- [x] Probe findings merged into parent model
- [x] Ready for `orch complete orch-go-ehper`

### Implementation Issues Created
- `orch-go-36uco` — Write method composition guide
- `orch-go-9gtzr` — Rewrite init "Next steps" + expand KB README
- `orch-go-3jzpz` — Curate example artifacts

---

## Unexplored Questions

- **Name tension:** Does "orch" (short for orchestrator) create a permanent perception problem? The name implies execution. This is strategic, beyond this investigation's scope.
- **Thread-without-agents value:** Do threads feel valuable on day 1 before any agent has produced evidence? If not, the method guide needs to address this directly.
- **Packaging/distribution:** How to distribute the binary (Homebrew, go install, release binaries). Technical question deferred.

---

## Friction

Friction: none — smooth session. Exploration agents returned useful structured assessments.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-minimum-open-release-26mar-39be/`
**Investigation:** `.kb/investigations/2026-03-26-design-minimum-open-release-bundle.md`
**Beads:** `bd show orch-go-ehper`
