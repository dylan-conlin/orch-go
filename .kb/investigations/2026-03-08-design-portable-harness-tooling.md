# Design: Portable Harness Tooling Extraction

**Date:** 2026-03-08
**Status:** Complete
**Beads:** orch-go-f85z0
**Depends on:** Cross-language evidence (orch-go-xi1tk), publication draft (orch-go-ap2jw)

---

## Problem

The harness engineering framework (init, lock, verify, entropy, accretion gates) currently lives inside orch-go — a 47K-line orchestration system. Any team wanting governance for their Claude Code agents must adopt orch-go wholesale, which is impractical. The publication promises a "minimum viable harness" that any team can deploy in 30 minutes.

**Goal:** Extract harness tooling into a standalone CLI+library so any team with Claude Code agents can add governance without adopting orch-go.

---

## Dependency Analysis: What's Portable Today

### Zero Internal Dependencies (Clean Extraction)

| Package | Lines | External Deps | What It Does |
|---------|-------|---------------|-------------|
| `pkg/control/` | 267 | stdlib + os/exec (`chflags`, `ls`) | Control plane lock/unlock/verify/discover |
| `pkg/verify/accretion_precommit.go` | 113 | stdlib + os/exec (`git`) | Pre-commit accretion gate |
| `pkg/verify/accretion.go` | 298 | stdlib + os/exec (`git`, `wc`) | Completion accretion verification |

**Total: ~678 lines with zero orch-go internal imports.** These are already self-contained.

### One Internal Dependency (Light Adaptation)

| Package | Lines | Internal Dep | Adaptation |
|---------|-------|-------------|------------|
| `pkg/entropy/entropy.go` | 619 | `pkg/events` (event log parsing) | Inline the events.jsonl parsing (~50 lines) or make events path a parameter |

### Orch-Specific (Do Not Extract)

| Component | Why It's Not Portable |
|-----------|----------------------|
| Beads close hook creation | Requires `bd` CLI (beads issue tracking) |
| `orch emit` integration | Orch-specific event system |
| Spawn hotspot gate | Depends on spawn system, skill routing |
| Daemon escalation | Depends on daemon autonomous loop |
| Dupdetect integration | Useful standalone but separate concern |

---

## Design: `claude-harness` CLI

### Naming

`claude-harness` — scoped to Claude Code, clear purpose. Not `harness` (too generic, likely conflicts).

Alternative: `agent-harness` (broader, not Claude-specific). The cross-language probe shows the framework is tool-agnostic at the design level, but the implementation is Claude Code-specific (settings.json, hooks format).

**Recommendation:** `claude-harness` for v1. Rename if it generalizes beyond Claude Code.

### Package Structure

```
claude-harness/
├── cmd/claude-harness/
│   └── main.go              # CLI entry point (Cobra)
├── pkg/
│   ├── control/             # Control plane immutability (from orch-go pkg/control)
│   │   └── control.go       # Lock, Unlock, Verify, Discover, DenyRules
│   ├── accretion/           # Accretion gates (from orch-go pkg/verify/accretion*)
│   │   ├── precommit.go     # CheckStagedAccretion — pre-commit gate
│   │   └── completion.go    # VerifyAccretionForCompletion — agent completion gate
│   ├── entropy/             # Health analysis (from orch-go pkg/entropy)
│   │   └── entropy.go       # Analyze, FormatText, SaveReport
│   └── settings/            # NEW: Claude Code settings.json manipulation
│       └── settings.go      # Read/write deny rules, hook registration
├── hooks/                   # Bundled hook scripts
│   ├── gate-bd-close.py     # Optional: bd close prevention
│   └── gate-git-add-all.py  # Prevent git add -A
├── go.mod
├── go.sum
└── README.md
```

### Commands

```
claude-harness init [--dry-run]     # Day 1 governance (MVH Tier 1)
claude-harness lock                  # Lock control plane (chflags uchg)
claude-harness unlock                # Unlock for modifications
claude-harness status                # Show lock state
claude-harness verify                # Verify all locked (pre-commit)
claude-harness accretion             # Pre-commit accretion gate
claude-harness entropy [--days N]    # Health analysis
claude-harness hotspot               # File size analysis
```

### Init Steps (Simplified from orch-go)

The `orch harness init` has 5 steps. For the portable version:

| Step | orch-go | claude-harness | Change |
|------|---------|----------------|--------|
| 1. Deny rules | Add to settings.json | Same | None |
| 2. Hook registration | Register gate scripts | Register bundled hooks | Hooks bundled in binary, not external .py files |
| 3. Beads close hook | Create .beads/hooks/on_close | **SKIP** | Beads is orch-specific |
| 4. Pre-commit gate | Append to .git/hooks/pre-commit | Same, use `claude-harness accretion` | Different binary name |
| 5. Control plane lock | chflags uchg | Same | None |

**Key difference:** Step 3 (beads hook) is replaced with a generic "event emission" concept. Teams using their own issue tracking can wire their own close hooks. The portable tool provides the gates; the team provides the lifecycle integration.

### Hook Bundling Strategy

orch-go's hooks are external Python scripts (`~/.orch/hooks/gate-*.py`). For portability:

**Option A: Embed hooks in Go binary** (recommended)
- Use `//go:embed` to bundle hook scripts
- `claude-harness init` writes them to `~/.claude-harness/hooks/`
- Self-contained: no Python dependency for simple hooks
- Rewrite simple hooks (git add -A detection) as shell scripts

**Option B: Keep Python hooks**
- Requires Python 3 on target system
- More flexible but less portable
- Python is common on dev machines but adds a dependency

**Decision: Option A.** The two hooks are simple pattern matchers — they can be shell scripts. No reason to require Python for `grep -q "git add -A"`.

### Platform Portability

The current `chflags uchg` mechanism is **macOS-only**. For Linux:

| Mechanism | macOS | Linux |
|-----------|-------|-------|
| File immutability | `chflags uchg` | `chattr +i` (requires root or `CAP_LINUX_IMMUTABLE`) |
| Check immutability | `ls -lO` + grep "uchg" | `lsattr` + grep "i" |
| Unlock | `chflags nouchg` | `chattr -i` |

**Decision:** Abstract behind `control.Lock()`/`control.Unlock()` with platform detection via build tags:

```go
// control_darwin.go
func platformLock(path string) error {
    return exec.Command("chflags", "uchg", path).Run()
}

// control_linux.go
func platformLock(path string) error {
    return exec.Command("chattr", "+i", path).Run()
}
```

This is ~20 lines of platform-specific code. The rest of `pkg/control` is already platform-agnostic.

### Library vs CLI

Both. The `pkg/` packages are the library; `cmd/claude-harness/` is the CLI wrapper. Teams that want to integrate harness checks into their own tooling import the packages:

```go
import "github.com/dylan-conlin/claude-harness/pkg/accretion"
import "github.com/dylan-conlin/claude-harness/pkg/control"

// In their pre-commit hook:
result := accretion.CheckStaged(projectDir)
if !result.Passed {
    fmt.Fprintln(os.Stderr, accretion.FormatError(result))
    os.Exit(1)
}
```

---

## Extraction Plan

### Phase 1: Create `claude-harness` repo (~2-4h)

1. Create `github.com/dylan-conlin/claude-harness` repository
2. Copy packages with zero internal deps:
   - `pkg/control/control.go` → `pkg/control/control.go`
   - `pkg/verify/accretion_precommit.go` → `pkg/accretion/precommit.go`
   - `pkg/verify/accretion.go` → `pkg/accretion/completion.go`
3. Copy + adapt (inline events dep):
   - `pkg/entropy/entropy.go` → `pkg/entropy/entropy.go`
4. Add platform abstraction for chflags/chattr
5. Write CLI entry point (`cmd/claude-harness/main.go`)
6. Port tests from orch-go

### Phase 2: Self-contained init (~2-4h)

1. Rewrite hook scripts as embedded shell (no Python dep)
2. Implement `claude-harness init` with steps 1,2,4,5 (skip beads)
3. Add `claude-harness hotspot` (file size analysis)
4. Write integration tests

### Phase 3: Cross-reference with orch-go (~1-2h)

1. Add inline lineage metadata per prior decision:
   ```
   // Lineage: extracted from github.com/dylan-conlin/orch-go/pkg/control
   // Date: 2026-03-08
   ```
2. Consider: should orch-go import claude-harness, or maintain a copy?
   - **Recommendation:** orch-go imports claude-harness. Single source of truth.
   - Risk: orch-go takes a dependency on a new module
   - Mitigation: claude-harness is maintained by same author, can vendor if needed

### Phase 4: Documentation & README (~1-2h)

1. Write README with 30-minute quickstart
2. Document the "why" from the publication
3. Add `.github/` with CI (test on macOS + Linux)

---

## Thresholds to Carry Over

These are battle-tested values from 12 weeks of orch-go operation:

| Threshold | Value | Evidence |
|-----------|-------|---------|
| CRITICAL file size | 1,500 lines | daemon.go trajectory — files above this always degrade further |
| WARNING file size | 800 lines | Entropy analysis signal — early intervention point |
| Accretion delta trigger | 50 net lines | Filters noise (renaming, imports) from real growth |
| Entropy fix:feat healthy | < 0.5 | Steady state ratio from orch-go operations |
| Entropy fix:feat spiral | > 0.9 | Agents fixing agents' work |
| Velocity red line | 45 commits/day | Exceeds human verification bandwidth |
| Entropy window | 28 days | Monthly cycle provides meaningful trend |

These should be configurable via a `.harness.yaml` or similar config file, with these as defaults.

---

## Configuration File: `.harness.yaml`

```yaml
# .harness.yaml — claude-harness configuration
# Place in project root. All values have sensible defaults.

thresholds:
  critical: 1500      # Block commits adding to files above this
  warning: 800        # Warn on growth above this
  delta: 50           # Minimum net line change to trigger checks

exclude:
  - "*.gen.ts"        # Generated code (cross-language probe finding)
  - "*.gen.go"
  - "vendor/**"
  - "node_modules/**"
  - "*.pb.go"         # Protobuf generated
  - "*.graphql.ts"    # GraphQL codegen

# Language-specific gates (from cross-language portability probe)
# These are NOT enforced by claude-harness — they're documented
# for teams to wire into their own CI.
language_gates:
  go:
    - "go build ./..."
    - "go vet ./..."
  typescript:
    - "bun typecheck"
    - "bun run lint"
  python:
    - "mypy ."
    - "ruff check ."
```

The `exclude` patterns directly address the cross-language probe's finding about generated code creating hotspot false positives.

---

## What NOT to Extract

| Component | Reason |
|-----------|--------|
| Spawn gates (`pkg/spawn/gates/hotspot.go`) | Tightly coupled to spawn system (skill routing, architect bypass, event logging) |
| Daemon escalation | Requires daemon autonomous loop |
| Beads integration | Issue tracking is a separate concern |
| Dupdetect (`pkg/dupdetect/`) | Useful standalone but separate tool, not core harness |
| Coaching plugins | OpenCode-specific |
| Event system (`pkg/events/`) | orch-specific event schema |
| Completion verification (`pkg/verify/check.go`) | Depends on beads, phases, skill system |

**Principle:** Extract the gates (mechanical enforcement), not the orchestration (lifecycle management). The publication's "minimum viable harness" is about gates. Orchestration is what orch-go provides on top.

---

## Risk Assessment

### Risk 1: Dual maintenance (orch-go + claude-harness)
- **Mitigation:** orch-go imports claude-harness. No copy.
- **Fallback:** If import creates friction, vendor claude-harness in orch-go with a sync script.

### Risk 2: macOS-only first release
- **Mitigation:** Platform abstraction is ~20 lines. Add Linux support in Phase 1.
- **Acceptable:** Most Claude Code users are on macOS. Linux support is important but not blocking.

### Risk 3: Naming collision with future Anthropic tooling
- **Mitigation:** "claude-harness" is descriptive but could conflict if Anthropic ships governance tooling.
- **Fallback:** Rename to `agent-harness` or `codebase-harness` if collision occurs.

### Risk 4: 30-minute promise is unrealistic
- **Mitigation:** `claude-harness init` automates 4 of 5 MVH steps. Manual step is adding governance sections to CLAUDE.md — provide a template.
- **Evidence:** orch harness init already completes in <1 minute. The 30 minutes accounts for reading docs + understanding why.

---

## Success Criteria

1. `go install github.com/dylan-conlin/claude-harness@latest` works
2. `claude-harness init` in a fresh repo sets up MVH Tier 1 in <5 minutes
3. `claude-harness verify` exits 0/1 correctly (pre-commit integration)
4. `claude-harness accretion` blocks commits to files >1500 lines
5. `claude-harness entropy` produces health report without orch-go installed
6. All tests pass on macOS and Linux
7. README enables a team to go from zero to governance in 30 minutes

---

## Follow-Up Work

- **Architect review** before implementation: This design makes assumptions about package boundaries and orch-go import direction that need validation.
- **Cross-language thresholds:** The 800/1500 line thresholds may need adjustment for TypeScript (files are naturally larger due to type definitions). Needs evidence from TypeScript agent operations.
- **Generated code exclusion:** Implement `.harness.yaml` exclude patterns in hotspot analysis before the TypeScript story is usable.
- **Python story:** Test the framework against a Python project to validate the third language.
