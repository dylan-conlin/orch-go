<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Defined a 3-tier Minimum Viable Harness (MVH) checklist for new agent-heavy projects — Day 0 (structural scaffold, 30 min), Day 1 (behavioral enforcement, 2-4h), and Week 1 (verification + observability, 4-8h) — derived from reverse-engineering orch-go's 16-component governance stack into what's essential vs what evolved through pain.

**Evidence:** Cataloged orch-go's full harness (16 components across 3 layers: hard/soft/behavioral), cross-referenced with harness-engineering model invariants, control-plane bootstrap model, and `orch init` implementation. Validated that `orch init` already scaffolds Day 0 items. Identified 4 governance components that have no scaffold/template support today.

**Knowledge:** The minimum viable harness is NOT a subset of orch-go's current harness — it's a different framing. MVH asks "what must exist on day one to prevent the first entropy spiral?" rather than "what does orch-go have?" The answer: structural scaffold + control plane immutability + one hard gate. Everything else can accrete safely after these three.

**Next:** Create `orch init --harness` flag or standalone `orch harness init` command that scaffolds Day 1 items (deny rules, on_close hook, pre-commit gate). This is an architectural recommendation — requires orchestrator review before implementation.

**Authority:** architectural - Cross-component (init command + harness command + settings management), multiple valid approaches (flag vs subcommand vs checklist-only)

---

# Investigation: Define Minimum Viable Harness for Agent-Heavy Projects

**Question:** What is the minimum set of governance infrastructure a new agent-heavy project needs on day one to prevent entropy spirals, and can it be expressed as a checklist or scaffold?

**Started:** 2026-03-08
**Updated:** 2026-03-08
**Owner:** Investigation agent (orch-go-xbqnk)
**Phase:** Complete
**Next Step:** None — ready for orchestrator review
**Status:** Complete

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/harness-engineering/model.md | deepens | yes — cataloged all hard/soft harness components listed | - |
| .kb/global/models/control-plane-bootstrap.md | extends | yes — bootstrap sequence informs Day 1 ordering | - |
| .kb/investigations/2026-03-01-design-control-plane-data-plane-separation.md | extends | yes — 6-file control plane matches harness lock/unlock scope | - |
| .kb/investigations/archived/2026-01-06-inv-define-workspace-cleanup-strategy-context.md | tangential | pending | - |

---

## Findings

### Finding 1: orch-go's Full Harness is 16 Components Across 3 Layers

**Evidence:** Comprehensive codebase exploration identified the following governance components:

**Hard Harness (mechanically enforced):**
1. Control plane immutability (`orch harness lock/unlock/status/verify` + `chflags uchg`)
2. Spawn hotspot gate (`pkg/spawn/gates/hotspot.go` — blocks feature-impl on >1500-line files)
3. Build gate (`go build` — compilation as enforcement)
4. Completion accretion gate (`pkg/verify/accretion.go` — 800/1500 thresholds)
5. Pre-commit growth gate (`pkg/verify/precommit.go` — warning-only)
6. Architecture lint tests (`cmd/orch/architecture_lint_test.go` — 4 tests, not in CI)
7. Spawn rate limiter (`pkg/spawn/gates/ratelimit.go`)
8. Spawn concurrency gate (`pkg/spawn/gates/concurrency.go`)

**Soft Harness (behavioral guidance):**
9. CLAUDE.md (project charter, conventions, constraints)
10. Skill system (`skills/src/` — worker-base, investigation, feature-impl, etc.)
11. Knowledge base (`.kb/` — 70+ guides, 145+ decisions, models)
12. SPAWN_CONTEXT.md (per-agent context injection)

**Enforcement Infrastructure:**
13. Claude Code hooks (`~/.orch/hooks/*.py` — 12 scripts covering PreToolUse, Stop events)
14. Deny rules in settings.json (prevent agents from modifying control plane files)
15. Beads hooks (`.beads/hooks/on_close` — event emission on issue close)
16. Completion verification pipeline (9 gates across 4 verification levels)

**Source:** cmd/orch/harness_cmd.go, pkg/control/control.go, pkg/spawn/gates/, pkg/verify/, .beads/hooks/on_close, ~/.orch/hooks/

**Significance:** This is a mature system evolved over 3+ months through 3 entropy spirals and 1,625 lost commits. A new project doesn't need all 16 components — but which ones are essential vs which evolved through pain?

---

### Finding 2: `orch init` Already Scaffolds Structural Items But Not Governance

**Evidence:** Reading `cmd/orch/init.go` (415 lines) shows `orch init` creates:

- `.orch/workspace/` and `.orch/templates/` directories
- `.orch/config.yaml` (ports, spawn mode)
- `.kb/` via `kb init`
- `.beads/` via `bd init`
- `CLAUDE.md` (auto-detected project type: go-cli, svelte-app, python-cli, minimal)
- Tmuxinator config for workers session

What `orch init` does NOT create:
- **No deny rules** in `.claude/settings.json` or `.claude/settings.local.json`
- **No hooks** (neither Claude Code hooks nor beads hooks)
- **No pre-commit gate** integration
- **No control plane lock** (`orch harness lock` is separate, manual)
- **No `.beads/hooks/on_close`** (event emission hook)

**Source:** cmd/orch/init.go:128-261

**Significance:** The structural scaffold exists, but the governance scaffold is entirely manual. A new project gets directories and templates but zero enforcement. The gap between `orch init` and "safe for autonomous agents" is the Day 1 checklist.

---

### Finding 3: The Bootstrap Paradox Constrains Day-One Setup Order

**Evidence:** The control-plane bootstrap model (`.kb/global/models/control-plane-bootstrap.md`) establishes that enforcement mechanisms cannot be deployed through the system they govern. This means:

1. Control plane files must be created BEFORE agents operate autonomously
2. Deny rules must exist BEFORE the first `orch spawn`
3. The daemon must NOT run until enforcement is verified behaviorally

The orch-go-ydzu incident (Feb 15, 2026) proved this: the daemon auto-closed its own pause mechanism. Three debugging agents claimed success while production was unchanged.

**Source:** .kb/global/models/control-plane-bootstrap.md (Claims 1-4)

**Significance:** The MVH checklist must be ordered — structural scaffold first, then enforcement, then verification, then daemon activation. Out-of-order setup risks the bootstrap paradox.

---

### Finding 4: Three Invariants Define "Minimum Viable"

**Evidence:** The harness-engineering model establishes critical invariants that define what's non-negotiable:

1. **"Every convention without a gate will eventually be violated"** — daemon.go grew past the stated 1,500-line CLAUDE.md convention. This means at least one hard gate is required from day one.

2. **"Agent failure is harness failure"** — The minimum harness must prevent the most common agent failures: modifying control plane files, closing their own issues, and unconstrained accretion.

3. **"Mutable hard harness is soft harness with extra steps"** — If agents can modify the enforcement files, the enforcement is illusory. Control plane immutability is required, not optional.

Combined with the constraint from SPAWN_CONTEXT: "Behavioral enforcement requires dynamic mechanisms (hooks, frame guards) not static skill text — static reinforcement fails under situational pull."

**Source:** .kb/models/harness-engineering/model.md (Critical Invariants 1-3, 6)

**Significance:** These invariants define the boundary between "nice to have" and "must have." A project without these three properties (one hard gate, control plane immutability, agent-can't-self-close) will experience its first entropy spiral within weeks.

---

## Synthesis

**Key Insights:**

1. **The MVH is 3 tiers, not a flat checklist** — Day 0 (scaffold) takes 30 minutes and is already automated. Day 1 (enforcement) takes 2-4 hours and is entirely manual today. Week 1 (verification + observability) takes 4-8 hours and requires behavioral testing. Each tier builds on the previous.

2. **The gap between `orch init` and "safe for agents" is exactly 7 items** — deny rules, on_close hook, pre-commit gate, CLAUDE.md governance section, control plane lock, one completion gate, and human behavioral verification. These 7 items are the Day 1 checklist.

3. **Order matters due to bootstrap paradox** — Enforcement files must exist before agents operate. The checklist is sequential, not parallel. `orch init` → deny rules → hooks → lock → verify → daemon.

**Answer to Investigation Question:**

The minimum viable harness for a new agent-heavy project consists of 3 tiers:

### Tier 0: Structural Scaffold (Day 0 — 30 min, already automated)

```
orch init
```

This creates: `.orch/`, `.kb/`, `.beads/`, `CLAUDE.md`, tmuxinator config, port allocations.

**Verification:** `ls .orch/ .kb/ .beads/ CLAUDE.md` — all exist.

### Tier 1: Behavioral Enforcement (Day 1 — 2-4h, currently manual)

| # | Item | Why Essential | How |
|---|------|---------------|-----|
| 1 | **Deny rules** in settings.json | Prevents agents from modifying their own constraints (the recursive vulnerability) | Add `Edit(~/.claude/settings.json)`, `Write(~/.claude/settings.json)`, `Edit(~/.orch/hooks/**)`, `Write(~/.orch/hooks/**)` to `permissions.deny` |
| 2 | **`gate-bd-close.py` hook** | Prevents agents from self-closing issues (bypasses verification) | Copy from orch-go's `~/.orch/hooks/gate-bd-close.py`, register in settings.json PreToolUse |
| 3 | **`gate-worker-git-add-all.py` hook** | Prevents careless `git add -A` that stages secrets/unrelated files | Copy from orch-go's hooks, register in settings.json PreToolUse |
| 4 | **`.beads/hooks/on_close`** | Emits completion events so work isn't silently lost | Copy from orch-go's `.beads/hooks/on_close`, make executable |
| 5 | **Pre-commit growth gate** | The one hard gate — warns when files grow past thresholds | Wire `orch precommit accretion` into `.git/hooks/pre-commit` |
| 6 | **Control plane lock** | Makes enforcement files immutable at OS level | `orch harness lock` after all hooks are placed |
| 7 | **CLAUDE.md governance section** | Documents authority boundaries, accretion limits, key gotchas | Add sections: Authority Delegation, Accretion Boundaries, Spawn Flow |

**Verification:** `orch harness status` shows all files LOCKED. `orch harness verify` exits 0.

### Tier 2: Verification & Observability (Week 1 — 4-8h)

| # | Item | Why Essential | How |
|---|------|---------------|-----|
| 8 | **Completion verification** | Without this, agents claim "done" without evidence | Configure `orch complete` gates (at minimum: Phase: Complete + test evidence) |
| 9 | **Event logging** | Without events, you can't measure agent success/failure rates | Verify `~/.orch/events.jsonl` receives `session.spawned` and `agent.completed` |
| 10 | **Human behavioral verification** | The bootstrap model requires observing gates firing against real system | Run one full agent cycle: spawn → work → complete → verify gates fire |
| 11 | **Hotspot gate** (if codebase >10K lines) | Prevents feature work on already-large files without architect review | Configure thresholds in spawn gates |
| 12 | **Circuit breaker** (if daemon enabled) | Halts autonomous operation when velocity exceeds verification bandwidth | Configure rolling average + hard cap thresholds |

**Verification:** Complete one full spawn→work→complete cycle with human observation at each gate.

### What's NOT in MVH (Can Accrete Later)

- Architecture lint tests (valuable but not day-one essential)
- Spawn rate limiter (only needed at scale, 5+ concurrent agents)
- Coaching plugin (only works in OpenCode runtime)
- Entropy agent (weekly maintenance, not day-one)
- Knowledge base guides (emerge from operations, not designed upfront)
- Skill system customization (worker-base provides defaults)
- Dashboard/web UI (monitoring is nice, not essential)

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch init` creates structural scaffold (verified: read cmd/orch/init.go, confirmed 6 creation steps)
- ✅ 16 governance components cataloged from codebase exploration (verified: read source files for each)
- ✅ Bootstrap paradox is a real constraint (verified: .kb/global/models/control-plane-bootstrap.md documents orch-go-ydzu incident)
- ✅ Deny rules are required for control plane protection (verified: .kb/models/harness-engineering/model.md invariant 6)

**What's untested:**

- ⚠️ Whether 7 Day-1 items are truly the minimum (could be fewer with smarter defaults)
- ⚠️ Whether the pre-commit growth gate alone is sufficient as "one hard gate" (might need build gate too)
- ⚠️ Time estimates for each tier (extrapolated from orch-go experience, not measured on a fresh project)
- ⚠️ Whether the checklist works for non-Go projects (orch-go's build gate is Go-specific)

**What would change this:**

- If `orch init` were extended to scaffold Day 1 items, the manual checklist shrinks significantly
- If a new project experiences an entropy spiral despite following the checklist, the "minimum" was wrong
- If deny rules alone (without hooks) prevent the most common agent failures, hooks could move to Tier 2

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Publish checklist as `.kb/guides/minimum-viable-harness.md` | implementation | Within knowledge base scope, no cross-boundary impact |
| Extend `orch init` with `--harness` flag for Day 1 automation | architectural | Touches init command + harness command + settings management across boundaries |
| Create `orch harness init` subcommand | architectural | New command surface, policy decisions about which hooks to install |
| Make the MVH checklist a verification step in `orch init` output | implementation | Extends existing "Next steps" output |

### Recommended Approach: Guide + Init Enhancement

**Phased approach** - Ship the guide immediately, automate incrementally.

**Why this approach:**
- The guide (`.kb/guides/minimum-viable-harness.md`) is immediately useful for any new project
- Automating Day 1 items requires design decisions about portability (which hooks are universal vs orch-go-specific)
- The checklist is the design artifact — automation follows

**Trade-offs accepted:**
- Day 1 items remain manual until automation is built
- Manual setup means some projects will skip steps

**Implementation sequence:**
1. Ship guide as `.kb/guides/minimum-viable-harness.md` (this session)
2. Add "Governance Setup" to `orch init` "Next steps" output (architect follow-up)
3. Implement `orch harness init` to scaffold deny rules + hooks + pre-commit gate (architect follow-up)

### Alternative Approaches Considered

**Option B: Full automation in `orch init`**
- **Pros:** Zero manual steps, consistent across projects
- **Cons:** Hooks are global (`~/.orch/hooks/`), not per-project — init can't safely modify global settings without risk of breaking existing projects
- **When to use instead:** If orch is deployed to a clean machine (no existing projects)

**Option C: Checklist-only (no code changes)**
- **Pros:** Simplest, no code to maintain
- **Cons:** Checklists without automation drift — the harness-engineering model's core insight
- **When to use instead:** For one-off projects that won't have sustained agent operations

---

## References

**Files Examined:**
- `cmd/orch/init.go` - Existing scaffold implementation (415 lines, 6 creation steps)
- `cmd/orch/harness_cmd.go` - Harness lock/unlock/status/verify commands
- `pkg/control/control.go` - Control plane file discovery and chflags management
- `.kb/models/harness-engineering/model.md` - Harness engineering framework (hard vs soft)
- `.kb/global/models/control-plane-bootstrap.md` - Bootstrap paradox and sequence
- `.kb/investigations/2026-03-01-design-control-plane-data-plane-separation.md` - 6-file control plane
- `.orch/templates/SYNTHESIS.md` - Template structure
- `~/.orch/hooks/` - 12 enforcement hook scripts
- `.beads/hooks/on_close` - Event emission hook
- `pkg/spawn/gates/hotspot.go` - Hotspot spawn gate
- `pkg/verify/precommit.go` - Pre-commit growth gate
- `pkg/verify/accretion.go` - Completion accretion gate

**Related Artifacts:**
- **Model:** `.kb/models/harness-engineering/model.md` - Framework for hard vs soft harness
- **Model:** `.kb/global/models/control-plane-bootstrap.md` - Bootstrap sequence for enforcement deployment
- **Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` - Hotspot gate design
- **Investigation:** `.kb/investigations/2026-03-01-design-control-plane-data-plane-separation.md` - Control plane architecture

---

## Investigation History

**2026-03-08:** Investigation started
- Initial question: What governance infrastructure does a new agent-heavy project need on day one?
- Context: Spawned by orchestrator to define reusable harness checklist/scaffold

**2026-03-08:** Comprehensive codebase exploration completed
- Cataloged 16 governance components across 3 layers (hard/soft/behavioral)
- Identified gap between `orch init` (structural) and full governance (manual)

**2026-03-08:** Investigation completed
- Status: Complete
- Key outcome: 3-tier MVH checklist (Day 0 / Day 1 / Week 1) with 12 items total, 7 of which are the critical Day 1 manual steps not covered by existing automation
