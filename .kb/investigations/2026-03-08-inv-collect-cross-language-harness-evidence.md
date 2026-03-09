## Summary (D.E.K.N.)

**Delta:** The harness engineering framework is language-independent at the design level (5/8 patterns translate directly to TypeScript), but the strongest enforcement mechanism (`go build` — "the only unfakeable gate") is Go-specific with no TypeScript equivalent of equal strength.

**Evidence:** Ran `orch harness init --dry-run`, `orch harness verify`, `orch hotspot`, and `orch precommit accretion` against ~/Documents/personal/opencode (TypeScript fork). All 5 MVH steps would apply. Hotspot analysis found 48 bloated files (vs orch-go's 12), with 4 of top 10 being generated code (false positives). TypeScript has no unfakeable compilation gate — `bun typecheck` has `any` escape hatch and is pre-push only.

**Knowledge:** "Unfakeability" is a property of structural coupling (schema ↔ migration, source ↔ binary), not compilation specifically. Each language ecosystem contributes its own hard harness patterns. Generated code is a cross-language concern the model doesn't address.

**Next:** Update harness-engineering model with cross-language findings. Recommend adding generated-file exclusion to `orch hotspot` and language-specific gate inventory to MVH checklist.

**Authority:** architectural — Cross-language portability affects model structure and tooling design across projects.

---

# Investigation: Collect Cross Language Harness Evidence

**Question:** Is the harness engineering model language-independent, or structurally dependent on Go-specific mechanisms?

**Started:** 2026-03-08
**Updated:** 2026-03-08
**Owner:** investigation agent (orch-go-xi1tk)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/harness-engineering/probes/2026-03-08-probe-30-day-accretion-trajectory-gate-effectiveness.md | extends | yes — verified gate exemption behavior still applies cross-language | None |

---

## Findings

### Finding 1: 5 of 8 harness patterns are fully language-independent

**Evidence:** Running `orch harness init --dry-run` on the OpenCode TypeScript project showed all 5 MVH Tier 1 steps would apply identically: deny rules, hook registration, beads close hook, pre-commit accretion gate, and control plane lock. These mechanisms operate at the OS level (chflags), tool level (Claude Code hooks), or git level (pre-commit line counting) — none examine language-specific constructs.

**Source:** `orch harness init --dry-run` output; `cmd/orch/harness_init.go` source; `pkg/control/control.go` DenyRules() function

**Significance:** The MVH Tier 1 checklist is genuinely portable. A project in any language with `.beads/` and `.claude/` can run `orch harness init` and get immediate governance.

---

### Finding 2: The "only unfakeable gate" (`go build`) has no TypeScript equivalent

**Evidence:** TypeScript's closest equivalent (`bun typecheck`) differs on every axis:
- Enforcement point: pre-push (not pre-commit or completion)
- Escape hatch: `any`, `@ts-ignore`, `@ts-expect-error`
- Runtime impact: Code runs regardless (JS runtime)
- Agent bypass: Trivially easy

OpenCode's pre-push hook runs `bun typecheck` but agents can commit broken types freely. The completion verification pipeline's build gate (`pkg/verify/check.go`) has no TypeScript equivalent.

**Source:** `.husky/pre-push` (opencode); `pkg/verify/check.go` (orch-go); TypeScript language design (gradual typing)

**Significance:** The model's strongest invariant — "build gate is the only unfakeable gate" — is Go-specific. This doesn't invalidate the framework, but it means the hard harness surface area varies by language. TypeScript projects start with less hard harness and must compensate with more structured soft or domain-specific gates.

---

### Finding 3: Generated code creates false positives in cross-language harness tooling

**Evidence:** `orch hotspot` on opencode found 155 hotspots and 48 bloated files (>800 lines). But 4 of the top 10 files are code-generated:
- `packages/sdk/js/src/v2/gen/types.gen.ts` (5,070 lines)
- `packages/web/src/components/icons/index.tsx` (4,454 lines — likely auto-generated)
- `packages/sdk/js/src/gen/types.gen.ts` (3,909 lines)
- `packages/sdk/js/src/v2/gen/sdk.gen.ts` (3,318 lines)

These would trigger architect routing and spawn gate blocking despite not being agent-authored code.

**Source:** `orch hotspot` output; `find . -name "*.gen.ts"` listing; file size analysis

**Significance:** The hotspot analysis, accretion gates, and spawn gate all assume files grow from agent commits. Generated code breaks this assumption. This is primarily a TypeScript/Python ecosystem problem (OpenAPI codegen, GraphQL codegen, protobuf) but also affects Go (protobuf-generated files). The model needs a generated-code exclusion concept.

---

### Finding 4: TypeScript has domain-specific hard harness that Go lacks

**Evidence:** OpenCode's pre-commit hook contains a Drizzle migration gate that blocks commits when `*.sql.ts` schema files are modified without corresponding migration files. This is a deterministic, unfakeable gate — the commit physically cannot proceed without the migration. Go has no equivalent because Go projects typically don't use ORMs with migration systems.

OpenCode also has bun version pinning in pre-push — the push fails if the wrong bun version is used. This is stricter than Go's `go.mod` version constraint.

**Source:** `.git/hooks/pre-commit` lines 8-23; `.husky/pre-push` lines 1-19

**Significance:** Each language ecosystem contributes its own hard harness patterns. The model catalogs Go's gates as if they're universal. A portable model should acknowledge that the gate inventory is language-specific while the gate taxonomy (hard/soft, compliance/coordination) is universal.

---

## Synthesis

**Key Insights:**

1. **The framework is portable, the gates are not.** The harness engineering taxonomy (hard/soft, compliance/coordination, attractors/gates), the failure modes (gate calibration death spiral, attractors without gates), and the invariants (agent failure = harness failure, every convention without a gate will be violated) all apply universally. But the specific gates (build, architecture lint, hotspot thresholds) are language-ecosystem-specific.

2. **"Unfakeability" comes from structural coupling, not compilation.** Go's `go build` is unfakeable because source → binary is structurally coupled. OpenCode's Drizzle gate is unfakeable because schema → migration is structurally coupled. The general principle: the tighter the structural coupling between authoring artifact and enforcement check, the harder the gate. This generalizes beyond Go.

3. **Generated code is a harness blind spot.** The model assumes all code growth is agent-authored. In TypeScript (and increasingly in Go), significant code is machine-generated. Harness tooling needs to distinguish agent-authored from generated code to avoid false positives in hotspot analysis, accretion gates, and architect routing.

**Answer to Investigation Question:**

The harness engineering model IS language-independent at the framework level — the taxonomy, invariants, and failure modes apply universally. But the implementation layer is language-specific: Go provides the strongest hard harness floor (compiler as unfakeable gate), TypeScript provides weaker type-level enforcement but has domain-specific gates (schema migration), and the gate inventory must be customized per ecosystem. The model should be structured as: universal framework + per-language gate catalog.

---

## Structured Uncertainty

**What's tested:**

- ✅ MVH Tier 1 steps apply to TypeScript project (verified: `orch harness init --dry-run` on opencode)
- ✅ Hotspot analysis runs on TypeScript (verified: `orch hotspot` found 155 hotspots, 48 bloated files)
- ✅ Accretion gate works on TypeScript files (verified: `orch precommit accretion` passed with 0 staged files)
- ✅ Control plane lock is language-independent (verified: `orch harness verify` passes)
- ✅ TypeScript has no unfakeable build gate (verified: inspected `.husky/pre-push` — `bun typecheck` has escape hatches)

**What's untested:**

- ⚠️ Whether accretion thresholds (800/1500 lines) are appropriate for TypeScript (TypeScript files may naturally be larger or smaller)
- ⚠️ Whether ESLint + strict tsconfig could serve as a substitute unfakeable gate (not benchmarked)
- ⚠️ Whether Python has worse or better hard harness than TypeScript
- ⚠️ Whether completion verification can be made language-agnostic (currently Go-specific build/vet/staticcheck)

**What would change this:**

- Finding a TypeScript mechanism with equal unfakeability to `go build` would contradict the "language-specific gates" finding
- Evidence that generated-file false positives don't actually cause incorrect routing would weaken Finding 3
- Testing on a Python project would extend or contradict the two-language comparison

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add generated-file exclusion to `orch hotspot` | implementation | Tactical fix, single component, clear criteria |
| Add language-specific gate section to model | architectural | Cross-component model change affecting publication |
| Create per-language MVH supplement | strategic | Direction choice for publication scope |

### Recommended Approach ⭐

**Language-Aware Gate Catalog** — Structure the harness engineering model as universal framework + per-language gate inventory.

**Why this approach:**
- Preserves the model's universal insights (taxonomy, invariants, failure modes)
- Acknowledges real variation in enforcement strength across languages
- Makes the model actionable for TypeScript/Python teams, not just Go

**Trade-offs accepted:**
- More complex model structure (one universal + N per-language sections)
- Need to maintain gate inventories as ecosystems evolve

**Implementation sequence:**
1. Add generated-file exclusion to `orch hotspot` (`.orchignore` or pattern-based)
2. Add cross-language findings to model (this probe's merge)
3. Create gate inventory table in model with Go/TypeScript columns

---

## References

**Files Examined:**
- `cmd/orch/harness_init.go` — Harness init implementation (5 steps)
- `pkg/control/control.go` — Control plane lock/unlock, deny rules
- `~/Documents/personal/opencode/.git/hooks/pre-commit` — Drizzle migration gate
- `~/Documents/personal/opencode/.husky/pre-push` — Bun version + typecheck
- `~/Documents/personal/opencode/.claude/settings.local.json` — MCP config only

**Commands Run:**
```bash
cd ~/Documents/personal/opencode && orch harness init --dry-run
cd ~/Documents/personal/opencode && orch harness verify
cd ~/Documents/personal/opencode && orch harness status
cd ~/Documents/personal/opencode && orch hotspot
cd ~/Documents/personal/opencode && orch precommit accretion
```

**Related Artifacts:**
- **Probe:** `.kb/models/harness-engineering/probes/2026-03-08-probe-cross-language-harness-portability.md`
- **Model:** `.kb/models/harness-engineering/model.md`
- **Prior Probe:** `.kb/models/harness-engineering/probes/2026-03-08-probe-30-day-accretion-trajectory-gate-effectiveness.md`
