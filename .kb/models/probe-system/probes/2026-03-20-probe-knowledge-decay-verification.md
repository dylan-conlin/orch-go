# Probe: Knowledge Decay Verification — Probe System

**Model:** probe-system
**Date:** 2026-03-20
**Status:** Complete
**claim:** infrastructure-accuracy, verdict-formats, metrics-currency
**verdict:** extends

---

## Question

Are the probe-system model's claims about infrastructure (files, fields, parsing), verdict formats, and metrics still accurate?

---

## What I Tested

Verified 7 core claims against current codebase:

1. **Key files exist:** `pkg/verify/probe_verdict.go`, `pkg/spawn/kbcontext.go`, `pkg/spawn/context.go`, `pkg/spawn/config.go`, `.orch/templates/PROBE.md`
2. **HasInjectedModels routing:** Grepped for `HasInjectedModels` across `pkg/spawn/` — checked field definition, population, and template usage
3. **PrimaryModelPath field:** Grepped across `pkg/spawn/` — confirmed in kbcontext.go, kbcontext_format.go, config.go
4. **Two verdict formats parsed:** Read `probe_verdict.go` — confirmed structured (`**Verdict:** extends — desc`) and checkbox (`- [x] **Confirms**`) formats
5. **PROBE.md template structure:** Read template — confirmed 4 mandatory sections (Question, What I Tested, What I Observed, Model Impact)
6. **"weakens" verdict support:** Grepped `probe_verdict.go` for "weakens" — no matches
7. **Current metrics:** Counted 47 models, 292 investigations, 42 model directories with probes/ subdirectories

---

## What I Observed

**Confirmed (5/7 claims):**
- All 5 key infrastructure files exist at documented paths
- `HasInjectedModels` routing is fully wired: field defined in `kbcontext.go:61`, populated in `kbcontext_format.go:70,168`, used in template conditional `worker_template.go:187,213,239`
- `PrimaryModelPath` field exists and is populated alongside `HasInjectedModels`
- Both verdict formats (structured + checkbox) are parsed with pre-compiled regex patterns (lines 29-36 of probe_verdict.go)
- PROBE.md template has 4 mandatory sections plus frontmatter with `claim:` and `verdict:` fields

**Extended (1 finding):**
- PROBE.md template now includes frontmatter fields (`claim:`, `verdict:`) not documented in the model. These are machine-readable metadata beyond the 4 mandatory sections.

**Contradicted (1 claim):**
- Model summary says probes produce verdicts including "weakens" — but `probe_verdict.go` only parses 3 verdict types: confirms, contradicts, extends. The PROBE.md template also only lists these 3. "Weakens" is not a supported verdict and would be silently ignored by the parser.

**Metrics update:**
- Model says "414 investigations : 29 models" — current count is 292 investigations : 47 models (ratio improved from 14.3:1 to 6.2:1)
- 42 model directories have probes/ subdirectories; top models: daemon-autonomous-operation (38 probes), completion-verification (26), orchestrator-session-lifecycle (25), spawn-architecture (24)

---

## Model Impact

- [x] **Extends** model with: (1) "weakens" verdict is not implemented — model incorrectly lists it as a supported verdict type. Only confirms/contradicts/extends are parsed. (2) PROBE.md template has gained frontmatter fields (claim, verdict) not documented in model. (3) Investigation:model ratio has improved from 414:29 to 292:47, suggesting probe system is working as intended — models growing while investigation count stabilized or decreased.

---

## Notes

- The "weakens" claim appears only in the model summary (line 10) — it was likely aspirational or copied from an early design. Low impact since no probe has ever used it (no grep matches).
- The investigation count decrease (414 → 292) may reflect cleanup/archival rather than fewer investigations created. The model count increase (29 → 47) is unambiguous growth.
- Open question 3 ("What's the actual probe-to-model merge compliance rate?") remains unanswered — no metrics infrastructure exists to measure this.
