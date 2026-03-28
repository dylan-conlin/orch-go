---
title: "Cross-domain harness portability — can structural enforcement transfer to music production and 3D modeling?"
status: resolved
created: 2026-03-16
updated: 2026-03-28
resolved_to: "culled: training-confound thread covers cross-domain better"
---

# Cross-domain harness portability — can structural enforcement transfer to music production and 3D modeling?

## 2026-03-16

Two candidate domains for harness portability: electronic music production and 3D model design (OpenSCAD). The question isn't 'can agents do these tasks' — it's whether the orchestration layer (gates, lifecycle, knowledge accretion, measurement) transfers to non-code domains.

Key tension: our gates are currently mechanical (file size, AST fingerprinting, build passes). Music and 3D modeling have different quality signals — aesthetic coherence, structural integrity, intent alignment. These aren't grep-able. This opens the question: can LLMs themselves serve as gates? Not generating the work, but judging whether work aligns with declared intent and stance.

Three layers to explore:
1. DOMAIN FIT — Do music/OpenSCAD have the properties that make orchestration valuable? (decomposable into parallel subtasks, measurable quality signals, accretion risk)
2. LLM-AS-GATE — Can an LLM evaluate 'does this synth patch serve the stated musical intent' or 'does this OpenSCAD module match the design stance' the way AST analysis evaluates code duplication? What's the false positive profile?
3. META — We built a system that measures its own enforcement. If we port it to music/3D, we're testing whether the meta-pattern (structural enforcement + self-measurement) is domain-general or code-specific. The self-measurement report becomes the methodology, not just the artifact.

PARAMETER SPACE MAPPING: Electronic music production has 600-1000+ parameters across a full session. Tools like SuperCollider (250+ UGens, headless, WAV export), Csound (batch processing, 22-param opcodes), and TidalCycles (declarative pattern language) are code-first and CLI-operable — they fit the spawn/monitor/complete lifecycle naturally.

The parameter count isn't the interesting question. The interesting question is WHICH parameters are gateable. Mechanical params (tempo, sample rate, buffer size) gate like code. Aesthetic params (filter sweep tension, reverb spatial feel, arrangement density) are judgment calls — the LLM-as-gate territory.

STRUCTURAL PARALLELS:
- Decomposition: track → parallel subtasks (drums, bass, lead, pad, FX) like feature → components
- Accretion risk: layers accumulate → frequency masking and arrangement clutter = audio file bloat
- Intent drift: agent optimizing a synth patch locally drifts from track intent, like code agent gold-plating

STRUCTURAL GAPS:
- No 'does it compile' — audio always renders, even when bad
- Quality measurement needs listening or spectral analysis, not grep
- 'Merge conflicts' are frequency collisions requiring spectral analysis, not diff

NEXT QUESTION: What would an LLM-as-gate actually look like for music? The gate receives: declared intent ('dark ambient tension building to release'), current state (spectral analysis + arrangement map), proposed change (new synth layer). It judges: does this change serve the declared intent? What's the false positive profile of that judgment?

LLM-AS-GATE DESIGN FOR MUSIC:

Three inputs: (1) intent declaration at session start ('dark ambient, tension building through first 3min, release via rhythmic shift, no clean tones'), (2) current state as machine-readable snapshot (spectral density per band, arrangement timeline, active voices, param values), (3) proposed change (new voice/effect/modification).

LLM judges: does this change serve the declared intent?

This is structurally identical to how the orchestrator skill already works — skill document declares stance, agent works, completion gates check alignment. The domain of judgment changes, not the pattern.

GATE TAXONOMY TRANSFERS:
- File size > 1500 lines → Spectral density > threshold per band (mechanical)
- Duplication detector → Timbral similarity detector, two voices in same freq/envelope space (mechanical + judgment)
- Build passes → Renders without clipping/artifacts (mechanical)
- Intent alignment → 'Does this serve the stated mood/arc?' (pure LLM judgment)
- Accretion pre-commit → Arrangement density check before adding voice (mechanical)

FALSIFIABILITY: Measure the LLM intent gate the same way we measured dup detector precision. 100 changes (50 aligned, 50 deliberately drifting), measure precision/recall. Below 65% = not worth deploying. Above 80% = viable gate. The self-measurement methodology IS the transferable artifact, not the specific gates.

KEY INSIGHT: Two layers of LLM involvement with different trust profiles. LLM-as-worker (generating music) has unbounded output. LLM-as-gate (judging alignment) has binary output with measurable precision. The gate is falsifiable in ways the generation isn't. This separation — generation vs judgment — may be the core architectural pattern that transfers.

OPENSCAD DOMAIN ANALYSIS:

OpenSCAD is a script-based 3D compiler — CSG + functional + parametric. It reads .scad scripts and deterministically renders geometry. Fully headless (openscad -o output.stl input.scad -D param=value). This is closer to code than music is.

PARAMETER SCALE:
- Simple part (bracket): 5-10 params
- Multi-part assembly (enclosure): 15-30 params
- Complex mechanical (gears, threading): 30-100+ params
- Full project with multiple assemblies: 100-300+

WHAT'S MECHANICALLY GATEABLE (direct transfer from code gates):
- Syntax correctness (like 'does it compile')
- Type validity via assert() (like type checking)
- Manifold geometry at F6 render (like 'does it build')
- Zero-area triangles on export (like lint)
- Parameter bounds (wall_thickness >= 0.8mm via assert)

WHAT REQUIRES JUDGMENT (LLM-as-gate territory):
- Wall thickness adequacy for target printer/material
- Overhang feasibility (>45 degrees needs support, printer-specific)
- Part interference in assemblies (collision detection)
- Snap-fit feasibility (material properties not modeled)
- Design intent alignment ('is this bracket actually mountable in the enclosure?')

PARTIAL AUTOMATION POSSIBLE:
- Wall thickness: custom DRC scripts using offset() to probe geometry
- Overhang detection: measure Z-component of face normals
- These are the INTERESTING middle ground — mechanical analysis that could be gate-ified

KEY INSIGHT: OpenSCAD has a CLEARER gate boundary than music. The lower tier (syntax, manifold, render success) maps directly to code gates. The upper tier (printability, assembly fit, design intent) maps to LLM judgment. And there's a middle tier (wall thickness, overhang analysis) that's mechanical but requires domain-specific tooling — analogous to building the duplication detector for code.

COMPARISON TO MUSIC:
- OpenSCAD has 'does it compile' (F6 render). Music doesn't — audio always renders.
- OpenSCAD accretion = part count and geometric complexity. Measurable via polygon count, boolean operation depth.
- OpenSCAD 'merge conflicts' = part interference. Mechanically detectable unlike frequency collisions.
- OpenSCAD intent drift is MORE checkable — 'does this part fit the assembly' is geometric, not aesthetic.

VERDICT: OpenSCAD is the easier first domain to prototype harness portability. It has the compile/render gate that music lacks, the parameter space is smaller and more structured, and the judgment boundary is cleaner. Music is the harder test but the more impressive demonstration.

META SYNTHESIS — What we're actually testing:

Three hypotheses, ordered by falsifiability:

H1: The gate taxonomy (mechanical → mechanical+judgment → pure judgment) is domain-general.
Test: Implement gates for OpenSCAD. If the same three tiers emerge naturally, the taxonomy transfers. If we need a fourth tier or the boundaries are different, the taxonomy is code-specific.

H2: Self-measurement methodology transfers across domains.
Test: Measure OpenSCAD gate precision the same way we measured dup detector precision. If the methodology produces actionable numbers (ceremony vs enforcement, false positive rates, coverage gaps), it works outside code. This is the strongest claim — we're not saying 'our gates are good,' we're saying 'our way of measuring gates is good.'

H3: LLM-as-gate is viable for judgment-layer enforcement.
Test: Build an intent gate for both domains. Measure precision against human judgment. Music is the hard test (aesthetic judgment). OpenSCAD is the easy test (geometric + intent judgment). If LLM-as-gate works for OpenSCAD but not music, we've found the boundary. If it works for both, the pattern is general.

THE PORTFOLIO APPROACH:
- OpenSCAD first: lower risk, clearer gate boundary, faster to prototype. Proves the methodology transfers.
- Music second: higher risk, more impressive, tests LLM judgment at the aesthetic boundary. Proves the pattern is general.
- Both together: demonstrates that the self-measurement report isn't about code — it's about any domain where autonomous agents produce artifacts under structural constraints.

WHAT THE SYSTEM UNIQUELY PRODUCES IN THIS FRAMING:
Not music. Not 3D models. The system produces MEASURED CONFIDENCE in autonomous creative output. 'This synth patch aligns with your declared intent at 82% precision.' 'This bracket meets printability constraints with 0 gate violations.' The artifact is secondary to the measured trust layer around it.

This reframes the self-measurement report: it's not a one-time artifact about orch-go. It's the first instance of a methodology that applies wherever agents produce things and humans need calibrated trust in the output.

---

## 2026-03-16 — MEASUREMENT: Layer 4 Intent Gate Precision (100-pair dataset)

H3 now has data. The LLM intent gate was measured against a 100-pair dataset (50 aligned, 50 misaligned across 5 drift types) using Haiku.

**Results:**
- Precision: 0.7742 — ADVISORY eligible (65-80% band), NOT blocking eligible (<80%)
- Recall: 0.96 — catches nearly all real misalignment
- F1: 0.8571
- Accuracy: 0.84
- Errors: 0, Needs Review: 3

**Per-drift-type recall (all misaligned categories):**
- dimensional: 10/10 (1.00) — perfect
- missing_features: 10/10 (1.00) — perfect
- constraint_violation: 10/10 (1.00) — perfect
- purpose_mismatch: 9/10 (0.90) — 1 FN (wall-hook)
- subtle: 9/10 (0.90) — 1 FN (pcb-standoff)

**False positive pattern:** 14 FPs, concentrated in 3 part types:
- cable-clip: 5/5 aligned pairs flagged as MISALIGNED (100% FP rate)
- spur-gear: 4/5 FP
- storage-box: 4/5 FP
- Remaining 7 part types: 1/35 FP (97% specificity)

**Dataset quality concern:** cable-clip "aligned" pairs have geometry metadata (bbox x=7,y=4,z=23) that doesn't match spec dimensions (15mm wide, 12mm tall). Some FPs may be correct gate verdicts on mislabeled data. If 10 of 14 FPs are confirmed mislabeled, corrected precision would be ~92% (blocking-eligible).

**What this means for the hypotheses:**
- H1 (gate taxonomy transfers): Confirmed — same 3 tiers emerged naturally in OpenSCAD
- H2 (measurement methodology transfers): Confirmed — same precision/recall framework produced actionable numbers
- H3 (LLM-as-gate viable): PARTIALLY CONFIRMED — precision 77% is above noise (65%), below blocking (80%). Gate is useful as advisory, not yet as hard gate. Path to blocking: fix dataset quality, re-measure.

**Infrastructure note:** Had to fix intent-check.sh to use `--output-format stream-json` and extract first assistant turn, because Claude CLI Stop hooks corrupt `--print` text output. The hook injects a synthetic user message that causes a second model turn, polluting the structured output the gate expects. This is a general issue for any tool using `claude --print` in environments with hooks.

**Raw data:** `~/Documents/personal/openscad-harness/test/results/2026-03-16-213035/`

## Auto-Linked Investigations

- .kb/investigations/2026-02-28-inv-cross-project-interface-agreement-coverage-gaps.md
