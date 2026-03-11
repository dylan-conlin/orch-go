# Probe: Empty Model Stub Creation Vectors

**Model:** knowledge-physics
**Date:** 2026-03-11
**Status:** Complete

---

## Question

What creates empty model stubs in .kb/models/? The knowledge-physics model claims an 85.5% orphan rate and identifies "attractor dynamics" as a core mechanism — does model creation follow the same accretion-without-synthesis pattern seen in investigations?

---

## What I Tested

### 1. Current empty model inventory

```bash
# Scanned all 35 model.md files in .kb/models/ and 2 in .kb/global/models/
# Checked each for template placeholder patterns: [What phenomenon...], [Concise claim...], {Title}, {Domain}
```

**Result:** Zero current empty stubs. All 37 model files contain substantial content (3K-26K words each).

### 2. kb create model template output (without --from)

```bash
kb create model test-empty-stub --project /tmp/kb-test-dir
cat /tmp/kb-test-dir/.kb/models/test-empty-stub/model.md
```

**Result:** Creates scaffold with bracket-enclosed placeholders:
- `[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]`
- `[Concise claim statement]`
- `[How to test this claim]`
- `[Scope item 1]`
- `[Question that further investigation could answer]`

These are the **detectable signatures** for unfilled model stubs.

### 3. Git history of model.md creation

```bash
git log --all --diff-filter=A --name-only -- '.kb/models/*/model.md'
```

**Result:** 11 commits created 41 model.md files across Feb 25 - Mar 10, 2026. Creation vectors:

| Vector | Commits | Models | Pattern |
|--------|---------|--------|---------|
| Batch migration (Feb 25) | 1 | 22 | Systematic knowledge system refactor |
| Investigation promotion | 2 | 2 | Direct `kb create model --from` with sources |
| Agent synthesis sessions | 5 | 10+ | Agents creating models during work |
| File recovery | 1 | 1 | Manual reconstruction from untracked file |
| Multi-model probe sessions | 1 | 5 | Probe-driven updates to multiple models |

### 4. Existing prevention gates

```bash
cat scripts/pre-commit-exec-start-cleanup.sh
cat .git/hooks/pre-commit
```

**Result:** Pre-commit hook runs `orch precommit accretion` (blocks >1500 lines) and `orch precommit knowledge` (blocks orphan investigations). No gate exists for model stub detection.

### 5. Agent creation paths (non-kb-create)

Searched commit messages for model creation patterns. Found agents creating models via:
- Direct file writes during synthesis sessions (agent opens Write tool, writes model.md)
- `kb create model --from` invocations with investigation references
- Batch refactoring sessions that touch multiple models

The **risk vector** is `kb create model` without `--from` followed by an agent dying or abandoning the session before filling the template. The scaffold file gets committed with placeholders intact.

---

## What I Observed

### Finding 1: No current empty stubs exist
All 37 models have been filled with real content. The Feb 25 batch migration created 22 models in one commit — all were filled during that same session.

### Finding 2: kb create model produces detectable stub signatures
Without `--from`, the output contains 7 distinct bracket-enclosed placeholders that are unique to the template and never appear in real model content:
1. `[What phenomenon or pattern does this model describe?`
2. `[Concise claim statement]`
3. `[Explanation of the claim.`
4. `[How to test this claim]`
5. `[Scope item 1]`
6. `[Exclusion 1]`
7. `[Question that further investigation could answer]`

### Finding 3: Three creation vectors for potential empty stubs
1. **kb create model (no --from):** Creates full scaffold with all placeholders. If agent dies before filling → committed stub.
2. **kb create model --from:** Fills some placeholders (Core Claims, What This Is, Evidence) but leaves others (Implications, Boundaries, Open Questions) with modified placeholder text.
3. **Agent direct-write:** Agent creates model.md by writing content directly. Could theoretically write template text if copying from TEMPLATE.md without filling it.

### Finding 4: Pre-commit gate infrastructure exists and is extensible
The `orch precommit` command already has `accretion` and `knowledge` subcommands. Adding a `model-stub` subcommand follows the exact same pattern: check staged files, detect placeholders, block commit.

---

## Model Impact

- [x] **Extends** model with: Model creation follows different accretion dynamics than investigations — models are created less frequently (41 total vs 700+ investigations) but with higher quality per-create. The orphan problem that afflicts investigations (85.5% rate) has not manifested in models, likely because models are created deliberately (kb create model) rather than as side-effects (empty investigation templates from dying agents). However, no structural gate prevents it — the prevention is currently behavioral, not architectural.

---

## Notes

The knowledge-physics model's accretion-without-synthesis pattern applies to investigations but not (yet) to models. This is likely because:
1. Models are created less frequently (41 vs 700+)
2. Models require explicit invocation (`kb create model`) vs investigations which are created by skill templates
3. The Feb 25 batch migration established a baseline of filled models

The recommended gate (pre-commit check for model.md placeholder text) closes this gap architecturally rather than relying on behavioral compliance.
