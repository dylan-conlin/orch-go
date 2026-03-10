# Design: Adversarial Gate for the Knowledge Pipeline

**Date:** 2026-03-10
**Status:** Proposed
**Owner:** Orchestrator System
**Context:** Prevent investigate -> model -> probe -> publish from escalating internally coherent observations into overclaimed public theory.

## Problem Statement

The current pipeline has verification and advisory steps, but no independent challenge step. In practice, investigations produce real observations, models elevate them into generalized claims, probes test those claims from inside the same framing, and publication treats the resulting coherence as readiness. The result is a closed loop:

1. Observations become theory without a subtraction step.
2. Probes inherit the model vocabulary and confirm inside that frame.
3. Publications rename familiar governance/software concepts as novel theory.
4. Evidence becomes self-referential: models cite probes that cite the model.
5. External review arrives only after publication, when the cost of overclaim is already paid.

The gate must be mechanical, must require external perspective, must catch vocabulary inflation and endogenous evidence loops, and must not depend on the same kind of internal coherence it is policing.

## Design Goals

1. Block publication mechanically, not by reminder.
2. Require at least one independent external challenge before public-theory claims.
3. Detect vocabulary inflation by forcing canonical restatement and prior-art mapping.
4. Detect self-referencing evidence by validating lineage, not trusting prose.
5. Keep the gate asymmetric so it cannot approve itself through stronger rhetoric.

## Non-Goals

- Proving a model true.
- Automating deep theory judgment.
- Replacing human editorial taste.
- Preventing internal models from existing. The goal is to restrict what can be published as novel, general, or externally validated.

## Architecture Overview

Add a separate **challenge lane** between `probe` and `publish`.

New pipeline:

`investigate -> model -> probe -> challenge -> publish`

The key difference is that `challenge` is not another synthesis step. It is a blocking layer composed of fixed checks:

1. **Lineage Gate** checks whether publication claims resolve to raw or external evidence instead of model/probe recursion.
2. **Vocabulary Gate** checks whether coined language collapses to existing concepts without predictive residue.
3. **External Challenge Gate** requires an independent reviewer outside the originating loop.
4. **Publication Policy Gate** downgrades or blocks unsupported novelty/generality language.

Each gate emits machine-readable verdicts. Publication is allowed only when all required verdicts are `pass`.

## Artifact Additions

All new artifacts remain markdown in `.kb/`.

### 1. Claim Ledger

Add a required claim table to models and publications:

```md
| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Unenforced conventions decay under agent throughput | empirical | code workflows | restatement | inv:... |
| C2 | This pattern is substrate-independent physics | generalization | cross-domain | novel | model:... probe:... |
```

Rules:

- `claim_id` is stable and unique within the artifact.
- `claim_type` is one of `observation`, `mechanism`, `generalization`, `recommendation`.
- `scope` states the domain the claim actually covers.
- `novelty_level` is one of `restatement`, `synthesis`, `novel`.
- `evidence_refs` must point to investigations, external sources, or challenge artifacts. References to only models/probes are insufficient for publication.

### 2. Challenge Artifact

Add `.kb/challenges/YYYY-MM-DD-<slug>.md`.

Required sections:

- `Target Artifact`
- `Reviewer Independence`
- `Blind Canonicalization`
- `Prior-Art Mapping`
- `Evidence Loop Findings`
- `Severity Codes`
- `Publication Verdict`

The challenge artifact is created from a fixed template and must end in a structured table:

```md
| code | status | applies_to | note |
|------|--------|------------|------|
| ENDOGENOUS_EVIDENCE | fail | C2 | Claim cites only model+probe descendants |
| VOCABULARY_INFLATION | fail | C2 | "substrate-independent physics" collapses to governance/incentive framing |
| EXTERNAL_NOVELTY_DELTA | fail | C2 | No predictive content beyond known concepts |
| PUBLICATION_LANGUAGE | fail | publication | Cannot use "new framework/physics" wording |
```

### 3. Publication Contract

Publications that make generalized or novelty-bearing claims must include:

- `challenge_refs`
- `claim_refs`
- `allowed_language`
- `disallowed_language`

If absent, `orch publish` fails.

## Gate Mechanics

### Gate 1: Lineage Gate

Purpose: catch self-referencing evidence.

Mechanism:

1. Build a DAG from artifact references.
2. For each publication/model claim, walk `evidence_refs`.
3. Fail if the transitive closure contains only:
   - the target model,
   - probes derived from that model,
   - publications derived from those probes.
4. Pass only if the claim is grounded by at least one of:
   - raw investigation evidence,
   - external primary source,
   - external observation not produced by the originating agent loop,
   - challenge artifact that explicitly cites such evidence.

Blocking rule:

- Any `generalization` or `novel` claim with zero exogenous evidence fails publication.

Why this works:

- It does not ask whether the prose sounds rigorous.
- It only checks graph structure and evidence class.

### Gate 2: Vocabulary Gate

Purpose: catch renaming familiar concepts as discoveries.

Mechanism:

Every term in a model/publication that looks coined, disciplinary, or framework-defining must appear in a canonicalization table:

```md
| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| accretion dynamics | things grow because addition is cheaper than cleanup | tech debt, institutional drift | predicts where decay appears | review |
| knowledge physics | governance constraints on shared systems | Ostrom, coordination cost, institutional memory | cross-domain mechanism | fail |
```

Rules:

1. If `plain_language` preserves the claim and `claimed_delta` is empty or rhetorical, mark `VOCABULARY_INFLATION`.
2. If `nearest_existing_concepts` is blank, fail schema validation.
3. If `novelty_level=novel`, at least one challenge artifact must confirm a non-empty predictive delta after canonicalization.
4. If the publication still uses the coined term while the verdict is `fail`, publication blocks.

Why this works:

- It forces the system to translate itself back into ordinary language before publishing.
- A term survives only if something remains after the translation.

### Gate 3: External Challenge Gate

Purpose: require perspective outside the generating loop.

Mechanism:

The target artifact must receive at least one challenge from a reviewer that is independent on all three axes:

1. **Model independence:** different provider/model family than the one used for the model/probe artifacts, or a human external to the project.
2. **Context independence:** receives a fixed challenge packet, not the full internal thread history.
3. **Authority independence:** can block language/novelty claims but cannot bless them with freeform approval.

The challenge packet is split into two passes:

1. **Blind pass:** reviewer sees only canonicalized observations and asks:
   - What existing concepts explain this?
   - What, if anything, is surprising?
   - What would be overclaim?
2. **Framed pass:** reviewer sees the actual model/publication and asks:
   - Which claims are renamed familiar concepts?
   - Which claims depend on endogenous evidence?
   - Which publication terms should be banned?

Blocking rules:

- If blind pass names a known concept family and framed pass maps the claimed novelty to the same family, mark `EXTERNAL_NOVELTY_DELTA=fail`.
- If the reviewer emits any severity code in `fail`, publication blocks until the claim is rewritten or downgraded.

Why this works:

- The external reviewer is not asked to generate a better theory.
- The reviewer only returns structured objections against a fixed rubric.

### Gate 4: Publication Policy Gate

Purpose: prevent internal operating models from being published as validated general theory.

Mechanism:

Publication language is computed from the strongest surviving claim class:

- If only `observation` and `recommendation` claims pass, publication may describe practice/tooling/case-study language only.
- If `mechanism` claims pass with exogenous evidence in one domain, publication may use `working model` language.
- If `generalization` claims pass across multiple independent domains, publication may use broader language, but still cannot claim a new discipline unless `novel` claims also pass.

Disallowed until explicitly earned:

- `physics`
- `new framework`
- `general law`
- `substrate-independent`
- `proves`
- `validated theory`

This gate does not infer intent. It simply scans for banned terms against the current verdict state.

## Why the Gate Is Not Vulnerable to the Same Failure

The failure mode was “internal coherence masquerading as validation.” This design avoids repeating it by limiting what the gate can do:

1. The gate never proves a claim true. It only checks eligibility for stronger language.
2. The hardest checks are structural:
   - evidence graph closure,
   - reviewer independence metadata,
   - required canonicalization tables,
   - banned-language enforcement.
3. The external reviewer has negative authority only. It can block, downgrade, or map to prior art; it does not certify novelty by eloquence.
4. The blind pass strips internal vocabulary before review, reducing anchoring.
5. Novelty must survive translation into plain language plus prior-art comparison. Most inflation dies there.

## CLI / Workflow Changes

### New commands

```bash
orch kb challenge create <artifact>
orch kb gate model <model-path>
orch kb gate publish <publication-path>
orch publish <publication-path>
```

### Gate behavior

- `orch kb gate model`:
  - validates claim ledger schema,
  - validates vocabulary canonicalization coverage,
  - emits missing-exogenous-evidence warnings.

- `orch kb challenge create`:
  - builds the fixed challenge packet,
  - records reviewer/provider independence metadata,
  - stores structured severity codes.

- `orch kb gate publish`:
  - computes lineage,
  - checks challenge freshness,
  - enforces allowed/disallowed language,
  - exits non-zero on any blocker.

- `orch publish`:
  - wraps `orch kb gate publish`,
  - refuses to emit or sync the publication artifact unless the gate passes.

## Example: How This Would Catch the Failure

### Vocabulary inflation

`file growth -> accretion dynamics -> substrate-independent physics`

The vocabulary gate forces:

- plain language: “files and docs grow because adding is cheaper than cleanup”
- nearest concepts: `tech debt`, `institutional drift`, `governance cost`
- claimed delta: must state a predictive difference

If no predictive residue remains after canonicalization, `VOCABULARY_INFLATION` fails.

### Self-referencing evidence

`publication -> model -> probe -> model`

The lineage gate sees that the general claim is supported only by descendants of the same model. Without external evidence or raw investigation support, `ENDOGENOUS_EVIDENCE` fails.

### External puncture

An outside reviewer says “this is repackaged governance concepts with agent vocabulary.”

That maps directly to:

- `VOCABULARY_INFLATION`
- `EXTERNAL_NOVELTY_DELTA`
- `PUBLICATION_LANGUAGE`

The result is not “consider revising.” The publication command exits non-zero.

## Rollout Plan

### Phase 1: Block publication claims

Implement for publications first.

- Claim ledger on publication artifacts
- Challenge artifact template
- Lineage gate
- Publication language gate

This delivers the main protection quickly.

### Phase 2: Push requirements upstream into models

- Claim ledger on models
- Canonicalization table on models
- Early warnings during `orch kb gate model`

This shifts failures left.

### Phase 3: Separate confirmatory probes from adversarial probes

- Mark probe intent as `confirm`, `boundary`, or `adversarial`
- Prevent confirmatory probes from counting as novelty evidence by themselves

## Success Criteria

1. A publication that cites only internal model/probe loops fails mechanically.
2. A coined term without prior-art mapping fails mechanically.
3. A challenge artifact from the same model family or full internal context does not satisfy the external-review requirement.
4. A publication cannot use strong novelty language unless the evidence class supports it.
5. Useful internal models can still ship as “working models” or “practice notes” without pretending to be new theory.

## Open Questions

1. Whether reviewer independence should require a different provider, or merely a different model family plus blind packet.
2. Whether external human review should satisfy the gate for high-stakes publications better than external-model review.
3. How strict to make prior-art mapping for internal-only model artifacts versus public publications.

## Recommendation

Implement the publication gate first. The failure happened at publish time, and publication is where overclaim becomes externally costly. Everything else should be designed to feed that blocker.
