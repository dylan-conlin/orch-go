# Probe: Cross-Language Harness Portability (Go → TypeScript)

**Model:** harness-engineering
**Date:** 2026-03-08
**Status:** Complete

---

## Question

The harness engineering model claims to be a general framework for agent governance. Is the framework language-independent, or is it structurally dependent on Go-specific mechanisms (compiler, package structure, `go build` as "the only unfakeable gate")?

Tested by running `orch harness init --dry-run` and `orch harness verify` against the OpenCode TypeScript fork (~/Documents/personal/opencode), then comparing structural enforcement patterns.

---

## What I Tested

```bash
# 1. Dry-run harness init on TypeScript project
cd ~/Documents/personal/opencode && orch harness init --dry-run

# 2. Verify control plane (global, already locked from orch-go)
cd ~/Documents/personal/opencode && orch harness verify

# 3. Hotspot analysis on TypeScript project
cd ~/Documents/personal/opencode && orch hotspot

# 4. Pre-commit accretion check on TypeScript project
cd ~/Documents/personal/opencode && orch precommit accretion

# 5. Inspected existing enforcement mechanisms:
#    - .git/hooks/pre-commit (Drizzle migration gate + beads flush)
#    - .husky/pre-push (bun version check + typecheck)
#    - .claude/settings.local.json (MCP config only, no deny rules)
#    - No .beads/hooks/ directory (no close hook)
#    - No ESLint or architectural linting

# 6. Compared hand-written vs generated file sizes
find . \( -name "*.ts" -o -name "*.tsx" \) -not -name "*.gen.*" \
  -not -path "*/node_modules/*" | xargs wc -l | sort -rn | head -20
```

---

## What I Observed

### Harness Init Dry-Run Results (all 5 steps would apply)

| Step | What | Would Apply? | Notes |
|------|------|-------------|-------|
| 1. Deny rules | 6 rules to settings.json | Yes — 0 deny rules currently exist | Global settings.json has no deny rules despite hooks being registered |
| 2. Hook registration | gate-bd-close, gate-worker-git-add-all | Yes (format mismatch) | Hooks exist with `~/.orch/hooks/X.py` format, harness init expects `python3 ~/.orch/hooks/X.py` |
| 3. Beads close hook | .beads/hooks/on_close | Yes — directory doesn't exist | No beads hooks in opencode project |
| 4. Pre-commit accretion gate | Append to .git/hooks/pre-commit | Yes — not wired | Existing pre-commit has Drizzle gate + beads flush, but no accretion check |
| 5. Control plane lock | chflags uchg | Already locked (global) | 12/12 files locked from orch-go's harness init |

### Harness Verify Result
```
harness verify: OK (all control plane files locked)
```
Passes because control plane is global (shared settings.json + hooks). Not project-specific.

### Hotspot Analysis Comparison

| Metric | orch-go (Go) | opencode (TypeScript) |
|--------|-------------|----------------------|
| Bloated files (>800 lines) | 12 | **48** |
| Hotspots detected | — | **155** |
| Largest hand-written file | daemon.go (1,559) | icons/index.tsx (4,454) |
| Generated files in top 10 | 0 | **4** (*.gen.ts at 3,318-5,070 lines) |

**Critical finding:** Hotspot analysis has no generated-file exclusion. 4 of 10 top "hotspots" in opencode are code-generated SDK types/clients that no agent authored. These are false positives that would trigger incorrect architect routing.

### TypeScript-Specific Enforcement Already Present

| Mechanism | Type | Equivalent in orch-go? |
|-----------|------|----------------------|
| Drizzle migration gate (pre-commit) | **Hard** — blocks commit without migration | No equivalent (Go has no ORM schema drift concept) |
| Bun version pinning (pre-push) | **Hard** — blocks push with wrong runtime | No equivalent (Go version in go.mod is softer) |
| `bun typecheck` (pre-push) | **Soft-hard** — enforced but bypassable with `any` | `go build` is stronger (no escape hatch) |
| Prettier (formatting) | **Hard** — deterministic output | `gofmt` is equivalent |

### Translation Scorecard

| Harness Pattern | Translates Directly? | Adaptation Needed? |
|----------------|---------------------|-------------------|
| Deny rules (Edit/Write settings.json, hooks) | ✅ 100% | None — language-independent |
| Control plane lock (chflags uchg) | ✅ 100% | None — OS-level |
| Claude Code hook registration | ✅ 100% | None — tool-level |
| Beads close hook | ✅ 100% | None — shell script |
| Pre-commit accretion gate | ✅ 100% | None — uses git diff line counts |
| Build gate (`go build`) | ❌ 0% | **No equivalent enforcement strength** |
| Architecture lint tests | ⚠️ ~50% | Needs ts-morph or eslint (different tooling) |
| Hotspot analysis | ⚠️ ~70% | Needs generated-file exclusion |

**5 of 8 patterns translate directly.** 3 need adaptation. But the pattern that the model identifies as "the only unfakeable gate" (`go build`) has NO TypeScript equivalent with equal enforcement strength.

### The Build Gate Gap

The model states (line 46-47): "Build gate (`go build`) — Broken compilation reaching completion — **the only unfakeable gate**"

TypeScript's closest equivalent (`bun typecheck`) differs structurally:

| Property | `go build` | `bun typecheck` |
|----------|-----------|-----------------|
| Enforcement point | Pre-commit + completion | Pre-push only |
| Escape hatch | None — won't compile | `any` type, `@ts-ignore`, `@ts-expect-error` |
| Runtime impact | Binary won't exist | Code still runs (JS runtime) |
| Agent bypass | Impossible | Trivially easy |

---

## Model Impact

- [x] **Extends** model with: The framework is **structurally language-independent** (5/8 patterns translate directly, the taxonomy applies universally), but the **strongest enforcement mechanism is language-specific**. Go's compiler provides an unfakeable gate that TypeScript cannot replicate. This means the harness engineering model is portable as a *design framework* but the hard harness surface area varies significantly by language ecosystem.

- [x] **Extends** model with: Generated code is a cross-language harness concern not addressed by the model. TypeScript ecosystems (and others with code generation — protobuf, GraphQL, OpenAPI) produce large files that are not agent-authored. Hotspot analysis, accretion gates, and architect routing all produce false positives on generated code. The model needs a "generated code exclusion" concept.

- [x] **Extends** model with: TypeScript ecosystems have **domain-specific hard harness** that Go lacks (Drizzle migration gate, runtime version pinning). The model should acknowledge that each language ecosystem contributes its own hard harness patterns, and the framework should catalog these rather than assuming Go's gate inventory is universal.

- [x] **Contradicts** invariant: "Build gate is the only unfakeable gate" — this is Go-specific, not universal. In TypeScript, the closest unfakeable gate is the Drizzle migration check (schema change without migration = blocked commit). The "unfakeable" property comes from domain coupling (schema ↔ migration), not from compilation. This suggests "unfakeability" is a property of **structural coupling** rather than of compilation specifically.

---

## Notes

**What this means for publication:**
The harness engineering model can be presented as language-independent at the framework level (taxonomy, invariants, failure modes), but implementation guides need per-language gate inventories. The Go story has the cleanest hard harness (compiler provides the floor), but TypeScript/Python projects need substitute gates.

**Possible substitute gates for TypeScript:**
1. ESLint with `--max-warnings 0` in pre-commit (catches type-level issues)
2. Strict tsconfig (`noImplicitAny: true`, `strict: true`) removing the `any` escape hatch
3. Integration test gates in completion verification (behavioral correctness where type correctness is insufficient)
4. Bundle size gates (Vite/esbuild output size as a proxy for accretion)

**Follow-up questions:**
1. Does Python have worse or better hard harness than TypeScript? (No compiler at all, but pytest + mypy could substitute)
2. Should `orch hotspot` have a `.orchignore` or pattern-based exclusion for generated files?
3. Should the MVH checklist have a "language-specific gates" section?
