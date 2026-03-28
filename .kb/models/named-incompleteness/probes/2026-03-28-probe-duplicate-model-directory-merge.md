# Probe: Duplicate Model Directory — Thread Promote Naming Bug Creates Split Model State

**Model:** named-incompleteness
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-02 (unnamed gaps)
**verdict:** confirms

---

## Question

NI-02 predicts that unnamed incompleteness — gaps that exist but aren't named or can't be found — breaks composition. The thread promote naming bug (orch-go-la6xo) created a duplicate model directory with a truncated name (`generative-systems-are-organized-around/`), splitting the model into two locations. Does this split demonstrate Failure Mode 2 (unnamed gaps) in practice?

---

## What I Tested

Compared the two model directories:

```bash
ls .kb/models/generative-systems-are-organized-around/probes/
# 1 probe: subsumption analysis (2026-03-27)

ls .kb/models/named-incompleteness/probes/
# 5 probes: subsumption analysis + 4 newer probes (2026-03-28)

diff .kb/models/generative-systems-are-organized-around/probes/2026-03-27-probe-subsumption-analysis-child-models.md \
     .kb/models/named-incompleteness/probes/2026-03-27-probe-subsumption-analysis-child-models.md
# Byte-identical — the subsumption probe was duplicated into both directories
```

Checked model.md currency:
- `generative-systems-are-organized-around/model.md`: Last Updated 2026-03-27 (stale)
- `named-incompleteness/model.md`: Last Updated 2026-03-28 (current, with quantitative confirmation)

Checked probe Model: headers in named-incompleteness/:
- `2026-03-27-probe-subsumption-analysis-child-models.md` — referenced old name `generative-systems-are-organized-around`
- `2026-03-28-probe-tension-clustering-spatial-prediction.md` — referenced both names `named-incompleteness (generative-systems-are-organized-around)`
- 3 other probes — correctly referenced `named-incompleteness`

---

## What I Observed

The split demonstrates Failure Mode 2 precisely: the model's identity was unnamed (or rather, named incorrectly). The truncated slug `generative-systems-are-organized-around` lost the concept ("named incompleteness") and kept only the setup phrase. This caused:

1. **Probe duplication:** The subsumption probe was written to both directories because agents couldn't tell which was authoritative.
2. **Divergent evolution:** The named-incompleteness model.md received 4 updates (spatial structure, bibliometrics pilot, full study, tension clustering) while the truncated copy stayed frozen at 2026-03-27.
3. **Reference confusion:** Even probes in the correct directory carried stale model references, indicating agents were uncertain about the canonical name.

The 4 newer probes correctly targeted `named-incompleteness/`, suggesting the system self-corrected after the initial confusion — agents learned to use the canonical name. But the stale directory and duplicate probe persisted as noise.

**Fix applied:**
- Deleted `generative-systems-are-organized-around/` (stale duplicate, no unique content)
- Fixed probe Model: headers to reference `named-incompleteness`
- Thread file references (`2026-03-27-generative-systems-are-organized-around.md`) left intact as historical records

---

## Model Impact

- [x] **Confirms** NI-02 / Failure Mode 2: The truncated model name created an unnamed gap — the model existed in two locations, neither clearly authoritative. Probes were duplicated rather than composed. The model's evolution split: 4 probes went to the right location, the wrong location stayed frozen. This is exactly the "two agents investigating the same unnamed gap can't cluster because neither knows it's the same gap" mechanism from Failure Mode 2.

- [x] **Extends** with a new instance of Failure Mode 2: **naming truncation** as a source of unnamed gaps. The thread promote bug didn't create a missing gap — it created a gap with two names, which is functionally equivalent to unnamed. When a model's identity is split across two slugs, composition degrades the same way as when models lack names entirely.

---

## Notes

- The test file `cmd/orch/thread_cmd_test.go` still references `generative-systems-are-organized-around` as a test case for the truncation fix — this is correct and should stay.
- The thread promote naming bug (orch-go-la6xo) was the root cause. This probe documents the downstream effect on model composition.
