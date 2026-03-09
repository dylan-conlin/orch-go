## Summary (D.E.K.N.)

**Delta:** Implement the harness engineering model's 5-layer enforcement system. Layer 0 shipped. Layer 1 partially shipped (4 tests, not in CI). Layers 2-4 not started. March 7-8 implementation work lost in ~/deletion incident — rebuilding from the model text.

**Evidence:** Harness engineering model (synthesized from 5 sub-models, 265 contrastive trials, 3 entropy spirals). daemon.go at 1,559 lines (grew +892 post-extraction). 6 cross-cutting concerns duplicated across 4-9 files (~2,100 lines). spawn_cmd.go shrank -840 lines after pkg/spawn/backends/ attractor created — proving attractors work. 162 hotspots detected by `orch hotspot`. Pre-commit growth gate shipped but warning-only.

**Knowledge:** Hard harness > soft harness. Attractors + gates together (neither alone works). Extraction without routing is a pump. Agent failure is harness failure. Every convention without a gate will eventually be violated.

**Next:** Layer 1 completion → Layer 2 (duplication detector) → Layer 3 (entropy agent) → Layer 4 (self-extending gates).

---

# Plan: Harness Engineering Implementation

**Date:** 2026-03-08
**Status:** Phase 1 ready, Phases 2-5 designed
**Owner:** Dylan
**Model:** `.kb/models/harness-engineering/model.md`
**Thread:** `.kb/threads/2026-03-07-harness-engineering-structural-enforcement-agent.md`

---

## Objective

Implement the harness engineering model's enforcement layers, converting soft conventions into hard gates. Success = daemon.go-scale re-accretion becomes structurally impossible (not just detected, but prevented).

---

## Substrate Consulted

- **Model:** Harness engineering (unifying frame), architectural-enforcement, entropy-spiral, skill-content-transfer, extract-patterns, completion-verification
- **Existing code:** `architecture_lint_test.go` (4 tests, lifecycle-state only), `pkg/verify/accretion.go`, `pkg/verify/precommit.go` (missing — may have been lost), `pkg/spawn/gates/hotspot.go`
- **Evidence:** daemon.go 1,559 lines, spawn_cmd.go 1,165 lines, 162 hotspots, 57 bloated files (>800 lines)
- **Decisions:** Two-lane agent discovery (no lifecycle state), three-layer hotspot enforcement

---

## Decision Points

### Decision 1: Structural test scope for Layer 1

**Context:** Current architecture lint tests cover only the lifecycle-state constraint (4 tests). The harness model identifies 3 missing test categories: function size limits, package boundary enforcement, cross-cutting duplication detection.

**Options:**
- **A: All three categories** — Function size, package boundaries, duplication. Comprehensive but large scope.
- **B: Function size + package boundaries only** — Duplication detection is complex enough to be its own layer (Layer 2).
- **C: Function size only** — Most immediate value (daemon.go runDaemonLoop at 702 lines is the gravitational center).

**Recommendation:** B. Function size limits catch the accretion symptom. Package boundary tests prevent the structural cause (wrong imports, cross-layer dependencies). Duplication detection is Layer 2's job — putting it in tests conflates detection with enforcement.

### Decision 2: Pre-commit gate severity

**Context:** Pre-commit growth gate is currently warning-only (exits 0 always). The model notes "mutable hard harness is soft harness with extra steps." Making it blocking would be true hard harness.

**Options:**
- **A: Keep warning-only** — Low friction, agents see it but aren't blocked.
- **B: Blocking at CRITICAL (>1,500)** — Hard gate for extreme violations, warning below.
- **C: Blocking at both tiers** — Full hard harness.

**Recommendation:** B. Warning at 800, blocking at 1,500. The CRITICAL threshold is already established in spawn gates — aligning pre-commit with it creates consistent enforcement. Warning tier stays advisory to avoid gate calibration death spiral.

### Decision 3: Entropy agent architecture

**Context:** Layer 3 is a periodic agent that reviews growth trends and duplication. Could be a daemon cron job, a standalone script, or an `orch` subcommand.

**Options:**
- **A: Daemon cron job** — Runs as part of daemon loop on a schedule (weekly). Pros: integrated, automatic. Cons: daemon.go is already 1,559 lines.
- **B: `orch entropy` subcommand** — Standalone command, can be cron'd externally. Pros: isolated, testable. Cons: needs external scheduling.
- **C: `orch entropy` subcommand + launchd plist** — Standalone command with system-level scheduling. Pros: independent of daemon, crash-resistant. Cons: more infrastructure to maintain.

**Recommendation:** C. The daemon is already a hotspot. The harness model's own "backend independence" principle says critical paths need independent mechanisms. A launchd plist scheduled weekly is the simplest reliable approach.

---

## Phases

### Phase 1: Structural Attractors (Unblock Everything)

**Goal:** Create the destination packages that agents need before gates can block them from the wrong path. Gates without attractors block agents with nowhere to go.

**Rationale:** The harness model's critical invariant: "Cannot ship a gate without ensuring the alternative path exists." The 6 cross-cutting concerns need homes before we can enforce boundaries.

**Deliverables:**
- `pkg/workspace/` — workspace scanning, manifest operations (currently reimplemented in 4-9 files)
- `pkg/display/` — output formatting, table rendering (currently in every cmd file)
- `pkg/beadsutil/` — shared beads querying, filtering, ID extraction (currently in 6+ files)

**Exit criteria:**
- Each package has at least one function extracted from an existing cmd file
- Existing cmd files import from new packages (not copy-paste)
- `go build` passes
- Pre-commit accretion gate shows net reduction in at least one hotspot file

**Beads:** orch-go-qkcvr (workspace), orch-go-9m9gn (display), orch-go-6smy7 (beadsutil) — all parallel, no dependencies

### Phase 2: Structural Tests — Layer 1 Completion

**Goal:** Extend `architecture_lint_test.go` with function size limits and package boundary enforcement. Wire into CI (or pre-commit).

**Depends on:** Phase 1 (need attractors to exist before testing boundaries)

**Deliverables:**
- Function size lint: warn >200 lines, fail >400 lines (calibrated to current codebase)
- Package boundary tests: cmd/orch/ must not import from other cmd/ packages; pkg/ packages must not import from cmd/
- Import direction enforcement: Types→Config→Pkg→Cmd (inspired by OpenAI's layered deps)
- Wire architecture lint into pre-commit hook (not just `go test`)

**Exit criteria:**
- `go test ./cmd/orch/ -run TestArchitectureLint` catches all current violations as warnings
- No false positives on compliant code
- Pre-commit runs lint automatically

**Beads:** orch-go-av0l5 (function size) → orch-go-vwxp4 (package boundaries) → orch-go-4bwm3 (CI wiring), sequential

### Phase 3: Pre-commit Hardening

**Goal:** Upgrade pre-commit growth gate from warning-only to blocking at CRITICAL threshold. Align with spawn gate thresholds.

**Depends on:** Phase 1 (agents need somewhere to put code when blocked)

**Deliverables:**
- Pre-commit blocks commits that push files past 1,500 lines
- Warning at 800 lines with ≥30 net additions (unchanged)
- `--force-accretion` escape hatch with reason logging
- Accretion events logged to `~/.orch/events.jsonl`

**Exit criteria:**
- Attempting to commit a file >1,500 lines fails pre-commit
- `--force-accretion` bypasses with logged reason
- Normal commits (<800 lines) are unaffected

**Beads:** orch-go-bpdpq

### Phase 4: Duplication Detector — Layer 2

**Goal:** Static analysis tool that finds pattern similarity across files, converting "agent failure = harness bug" into automation.

**Depends on:** Phase 1 (need package structure to detect violations against), Phase 2 (need structural tests as enforcement target)

**Deliverables:**
- `orch detect duplication` command
- AST-based function similarity detection across cmd/orch/ files
- Output: list of function clusters with similarity scores
- Integration: when similarity > threshold, create beads issue automatically

**Exit criteria:**
- Detects the known 6 cross-cutting concerns (workspace scanning, beads querying, output formatting, project resolution, filtering, ID extraction)
- False positive rate < 20%
- Can run as standalone command

**Beads:** orch-go-dtafv (detector core) → orch-go-a4ecd (beads integration), sequential

### Phase 5: Entropy Agent — Layer 3

**Goal:** Periodic agent that reviews growth trends, duplication detector output, and structural test results. Produces actionable recommendations.

**Depends on:** Phase 4 (needs duplication detector output), Phase 2 (needs structural test results)

**Deliverables:**
- `orch entropy` subcommand that runs analysis
- Inputs: `orch hotspot` output, duplication detector results, git log growth trends, structural test results
- Output: prioritized recommendations (e.g., "pkg/workspace/ needed", "daemon.go periodic tasks should extract")
- `com.orch.entropy.plist` for weekly launchd scheduling
- Results written to `.kb/entropy/YYYY-MM-DD-report.md`

**Exit criteria:**
- Running `orch entropy` produces a report with specific, actionable recommendations
- Recommendations reference specific files and suggest specific package destinations
- Report includes growth velocity (lines/week) per hotspot file
- launchd scheduling works (runs weekly, output captured)

**Beads:** orch-go-7im2f (orch entropy command) → orch-go-7h8d6 (launchd scheduling), sequential

### Phase 6: Control Plane Immutability

**Goal:** Protect harness infrastructure from agent modification. Address the "mutable hard harness is soft harness" invariant.

**Depends on:** Phases 1-5 (protect what's been built)

**Deliverables:**
- `chflags uchg` on gate hook files (`~/.orch/hooks/gate-*.py`)
- `chflags uchg` on `~/.claude/settings.json`
- Pre-commit hook to verify immutability flags haven't been removed
- Agent tool deny rules for editing control plane files

**Exit criteria:**
- Agent attempts to `Edit` gate hooks are blocked
- Agent attempts to modify settings.json are blocked
- Flags survive across sessions
- Escape hatch: `orch harness unlock` for intentional modifications

**Beads:** orch-go-192s8 (chflags + deny rules) → orch-go-bcaaf (verification hook), sequential

---

## Dependency Graph

```
Phase 1: Structural Attractors
├── pkg/workspace/          (orch-go-qkcvr)  ─┐
├── pkg/display/            (orch-go-9m9gn)  ─┼── all parallel
└── pkg/beadsutil/          (orch-go-6smy7)  ─┘
         │
         ▼
Phase 2: Structural Tests ──────────────────── Phase 3: Pre-commit Hardening
├── Function size lint      (orch-go-av0l5)     └── Blocking gate        (orch-go-bpdpq)
│        │
│        ▼
├── Package boundaries      (orch-go-vwxp4)
│        │
│        ▼
└── CI wiring               (orch-go-4bwm3)
         │
         ▼
Phase 4: Duplication Detector
├── Detector core           (orch-go-dtafv)
│        │
│        ▼
└── Beads integration       (orch-go-a4ecd)
         │
         ▼
Phase 5: Entropy Agent
├── orch entropy command    (orch-go-7im2f)
│        │
│        ▼
└── launchd scheduling      (orch-go-7h8d6)
         │
         ▼
Phase 6: Control Plane Immutability
├── chflags + deny rules    (orch-go-192s8)
│        │
│        ▼
└── Verification hook       (orch-go-bcaaf)
```

**Critical path:** qkcvr/9m9gn/6smy7 → av0l5 → vwxp4 → 4bwm3 → dtafv → a4ecd → 7im2f → 7h8d6

**Parallel opportunities:**
- HE01, HE02, HE03 are fully independent
- Phase 3 (HE07) can run parallel with Phase 2 (after Phase 1)
- Phase 6 can start after any phase (protects what exists so far)

---

## Readiness Assessment

| Phase | Substrate Available | Navigable? |
|-------|---------------------|------------|
| 1: Attractors | Hotspot analysis identifies the 6 concerns, harness model names the packages | Yes — well-scoped extraction |
| 2: Structural tests | architecture_lint_test.go exists, OpenAI's layered deps pattern documented | Yes — extend existing |
| 3: Pre-commit | pkg/verify/accretion.go exists (297 lines), thresholds calibrated | Yes — upgrade severity |
| 4: Duplication | Go AST stdlib available, 6 known duplications to validate against | Yes — clear test cases |
| 5: Entropy agent | Hotspot command exists, duplication detector (Phase 4) provides input | Yes — after Phase 4 |
| 6: Immutability | chflags available on macOS, hook deny pattern exists | Yes — straightforward |

---

## Structured Uncertainty

**What's tested:**
- Attractors work (spawn_cmd.go -840 lines after pkg/spawn/backends/)
- Gates work (hotspot spawn gate blocks CRITICAL files)
- Pre-commit detection works (accretion gate shipped, warning-only)
- Structural tests work (4 lifecycle-state tests catch violations)

**What's untested:**
- Whether function size limits cause gate calibration death spiral (too strict → --force reflex)
- Whether AST-based duplication detection has acceptable false positive rate
- Whether entropy agent recommendations are specific enough to be actionable
- Whether control plane immutability via chflags survives macOS updates

**What would change this plan:**
- If Phase 1 extraction reveals tighter coupling than expected → may need architect session before extracting
- If duplication detector false positive rate >20% → needs ML/heuristic tuning before beads integration
- If entropy agent recommendations are too generic → may need to be interactive (spawn architect instead of report)
- If chflags doesn't survive → need alternative immutability mechanism

---

## Success Criteria

- [ ] 3 structural attractor packages created with real extractions (Phase 1)
- [ ] Architecture lint tests cover function size + package boundaries (Phase 2)
- [ ] Structural tests wired into pre-commit (Phase 2)
- [ ] Pre-commit blocks >1,500 line files (Phase 3)
- [ ] Duplication detector finds the 6 known cross-cutting concerns (Phase 4)
- [ ] Entropy agent produces specific, actionable recommendations (Phase 5)
- [ ] Entropy agent runs weekly via launchd (Phase 5)
- [ ] Control plane files are immutable to agents (Phase 6)
- [ ] daemon.go line count trending down, not up (aggregate measure)
