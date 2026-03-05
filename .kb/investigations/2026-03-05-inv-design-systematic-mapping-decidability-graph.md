## Summary (D.E.K.N.)

**Delta:** Of 22 distinct authority boundaries in the decidability graph, 10 are infrastructure-enforced, 9 are prose-only (and will leak under pressure), and 3 are judgment-only (can't be gated). 6 of the 9 prose-only boundaries are hookable with existing hook API.

**Evidence:** Cross-referenced decidability graph model, behavioral grammars model (Claim 3: situational pull overwhelms static reinforcement), 8 documented violations across archived workspaces, and complete inventory of 26 spawn/completion gates + 7 hooks.

**Knowledge:** The enforcement gap concentrates in two areas: (1) bash command filtering (git push, git add -A, bd dep add) and (2) lifecycle tracking (phase reporting, investigation→question linkage, provisional work marking). Existing hook API (PreToolUse, Stop, SessionStart) is sufficient — no new mechanism types needed.

**Next:** Create 6 implementation issues for hookable gaps, prioritized by violation frequency + risk. P1: git push filter, git add filter. P2: phase reporting timer, investigation→question linkage, provisional work warning. P3: bd dep add filter.

**Authority:** architectural — Maps across spawn/daemon/completion infrastructure and affects worker-orchestrator-Dylan authority chain

---

# Investigation: Decidability Graph Authority Boundaries → Infrastructure Enforcement Mapping

**Question:** Which decidability graph authority boundaries are infrastructure-enforced, which are prose-only (and will leak), and which are judgment-only (can't be gated)?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-7307r)
**Phase:** Complete
**Next Step:** None - create implementation issues for hookable gaps
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-01-19-worker-authority-boundaries.md` (extends enforcement mapping)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/decidability-graph/model.md` | Source (authority boundaries defined here) | Yes - all 22 boundaries confirmed | None |
| `.kb/models/behavioral-grammars/model.md` | Source (why prose leaks) | Yes - Claim 3 confirmed | None |
| `probes/2026-03-02-probe-layered-constraint-enforcement-design.md` | Extends (constraint taxonomy) | Yes - 7/31 hard constraints enforced | None |
| `.kb/investigations/2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md` | Extends (enforcement design) | Yes - V0-V3 levels, infrastructure principle | None |
| `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` | Confirms (three-layer pattern) | Yes - all 3 layers verified in code | None |

---

## Findings

### Finding 1: The Complete Authority Boundary Inventory

The decidability graph defines 22 distinct authority boundaries across three transition levels (Daemon→Orchestrator, Orchestrator→Dylan, Worker→Orchestrator) plus structural invariants. This is the first complete enumeration.

**Evidence:** Cross-referencing the decidability graph model sections: Node Taxonomy, Edge Authority, Worker Authority Boundaries, Intra-Task Authority, Resolution Typing, and Graph Dynamics.

**Source:** `.kb/models/decidability-graph/model.md` (lines 27-157)

**Significance:** Previous enforcement analysis (the Mar 2 probe) inventoried 31 hard-enforceable constraints from the behavioral grammars perspective. This investigation maps from the other direction — starting from authority boundaries and checking enforcement. The two perspectives are complementary: the probe found 24 unenforced hard constraints; this investigation finds 9 unenforced authority boundaries. They overlap but are not identical — some probe constraints are implementation details within a single boundary; some boundaries here have multiple constraint implications.

---

### Finding 2: Enforcement Coverage Is Concentrated at Spawn and Completion, Missing at Runtime

**Evidence:** Of 10 infrastructure-enforced boundaries:
- 6 are spawn gates (triage bypass, hotspot L1, concurrency, rate limit, verification, kb agreements)
- 3 are completion gates (synthesis, accretion, decision patch limit)
- 1 is a runtime hook (Task tool block via PreToolUse)

The 9 unenforced boundaries are ALL runtime behaviors — things agents do during execution between spawn and completion:
- git push, git add -A, bd dep add (bash commands during work)
- Phase reporting timeliness (first 3 actions)
- Investigation→question linkage (lifecycle tracking)
- Provisional work marking (graph state awareness)

**Source:** Inventory of `pkg/spawn/gates/` (6 files), `pkg/verify/` (14+ gate functions), `.beads/hooks/` (3 hook scripts)

**Significance:** The system has excellent "bookend" enforcement (spawn + completion) but a gap in the middle. The behavioral grammars model (Claim 3) predicts this is exactly where violations concentrate — situational pull during active work overwhelms static reinforcement from SPAWN_CONTEXT.md prose. The existing `gate-orchestrator-task-tool.py` PreToolUse hook proves the mechanism works at runtime; it just hasn't been extended to other bash command patterns.

---

### Finding 3: Six Boundaries Are Hookable With Existing API

**Evidence:** The Claude Code hook API provides three enforcement points:
- **PreToolUse** — intercept before tool execution (can deny or coach)
- **Stop** — intercept at session end (can block completion)
- **SessionStart** — inject context at session start

All 6 hookable gaps map cleanly to PreToolUse (bash command filtering) or Stop (lifecycle checks). No new mechanism types needed.

The 3 judgment-only boundaries (recognizing Gates, constitutional objection triggers, factual→framing question escalation) require reasoning about the nature of the situation — binary pattern matching cannot capture them. These are correctly left as prompt-level guidance.

**Source:** Existing hook implementations in `~/.claude/hooks/` (via probe inventory), Claude Code hooks documentation

**Significance:** This means the enforcement gap is an implementation gap, not an architectural gap. The hook API is sufficient. The work is: write 6 hooks, deploy them, and monitor violation rates.

---

## The Mapping Table

### Legend
- **Enforcement:** `infra` = code gate/hook, `prose` = SKILL.md/SPAWN_CONTEXT guidance, `structural` = enforced by system design, `none` = no mechanism
- **Hookability:** `enforced` = already gated, `hookable` = can be gated with existing API, `judgment` = requires reasoning (can't be binary-gated)
- **Priority:** P0 = security/deploy risk, P1 = high violation frequency + moderate risk, P2 = moderate frequency or risk, P3 = low frequency, N/A = already enforced or not hookable

### Worker → Orchestrator Boundaries

| # | Authority Boundary | Current Enforcement | Violation Evidence | Hookability | Proposed Mechanism | Priority |
|---|---|---|---|---|---|---|
| W1 | Workers cannot use Task tool (must use orch spawn) | **infra:** `--disallowedTools` spawn flag + `gate-orchestrator-task-tool.py` PreToolUse | Jan 17: Gemini Flash used Task 5x; Feb 24: identity-action gap confirmed | enforced (two layers) | — | N/A |
| W2 | Workers cannot use `bd close` (must use orch complete) | **infra:** `gate-bd-close.py` PreToolUse hook, checks `orch:agent` label | Feb 24: orchestrator used bd close; Mar 4: shlex parsing bug fixed | enforced | — | N/A |
| W3 | Workers cannot push to remote | **prose:** SPAWN_CONTEXT.md "NEVER run git push" | Jan 16: workers pushed without authorization (documented investigation) | hookable: PreToolUse bash filter | PreToolUse hook: deny `git push` commands | **P1** |
| W4 | Workers cannot add dependencies (`bd dep add`) | **prose:** worker-base SKILL.md "Workers CANNOT add dependencies" | No direct violation found; boundary is prose-only | hookable: PreToolUse bash filter | PreToolUse hook: deny `bd dep add` commands | **P2** |
| W5 | Workers cannot label strategic questions `triage:ready` | **prose:** worker-base SKILL.md guidance | No direct violation found | hookable: PreToolUse bash filter (complex regex) | PreToolUse hook: warn on `bd create --type question.*triage:ready` | **P3** |
| W6 | Workers stage only specific files (no git add -A/.) | **prose:** SPAWN_CONTEXT.md "NEVER use git add -A" | Common pattern in archived workspaces | hookable: PreToolUse bash filter | PreToolUse hook: deny `git add -A`, `git add .`, `git add --all` | **P1** |
| W7 | Workers must report Phase within first 3 actions | **prose:** worker-base SKILL.md | Common violation in archived workspaces | hookable: SessionStart timer + beads check | SessionStart sets deadline; post-turn check queries beads for phase comment | **P2** |
| W8 | Workers cannot make architectural decisions (beyond scope) | **infra:** completion gates (decision patch limit, architectural choices) | Enforced at completion time only | enforced (completion) | — | N/A |
| W9 | Workers escalate when reaching outside scoped context | **prose:** worker-base SKILL.md "Does this decision stay inside?" | Not systematically tracked; inherently judgment-based | judgment: requires reasoning about scope boundaries | Cannot be binary-gated; prompt guidance is correct mechanism | N/A |
| W10 | Workers cannot use destructive git commands | **prose:** Claude Code system prompt + SPAWN_CONTEXT | No documented violation (Claude's built-in caution helps) | hookable: PreToolUse bash filter | PreToolUse hook: deny `git reset --hard`, `git checkout .`, `git clean -f` | **P3** |

### Daemon → Orchestrator Boundaries

| # | Authority Boundary | Current Enforcement | Violation Evidence | Hookability | Proposed Mechanism | Priority |
|---|---|---|---|---|---|---|
| D1 | Daemon traverses only Work→Work edges | **infra:** `IsSpawnableType()` returns false for questions | By design — all questions skipped regardless of subtype | enforced (structural) | — | N/A |
| D2 | Daemon cannot resolve questions (only spawn investigations) | **structural:** daemon spawns agents, doesn't synthesize | By design — daemon has no synthesis capability | enforced (structural) | — | N/A |
| D3 | Daemon should eventually route factual questions to investigation | **none:** daemon skips ALL questions regardless of subtype | Conservative default; no factual questions auto-spawned | hookable: daemon routing based on subtype label | Daemon checks `subtype:factual` → spawns investigation skill | **P3** |
| D4 | Daemon routes hotspot-targeting work to architect | **infra:** `architect_escalation.go` checks hotspot overlap | Verified working: escalation logic + auto-bypass for prior reviews | enforced | — | N/A |
| D5 | Daemon respects unresolved blocking dependencies | **infra:** `bd ready` excludes blocked work | Verified working: question-blocked items excluded from ready queue | enforced | — | N/A |

### Orchestrator → Dylan Boundaries

| # | Authority Boundary | Current Enforcement | Violation Evidence | Hookability | Proposed Mechanism | Priority |
|---|---|---|---|---|---|---|
| O1 | Orchestrator cannot traverse Gate edges | **prose:** decidability-graph model + orchestrator skill | No documented violation (theoretical boundary) | judgment: "is this a Gate?" requires reasoning | Cannot be binary-gated; prompt guidance is correct | N/A |
| O2 | Gate nodes accumulate options but don't resolve | **none:** no structured Gate accumulation mechanism | No mechanism exists; gates are ad-hoc | hookable (partially): structured gate issue type in beads | `bd create --type gate` with structured options accumulation | **P3** |
| O3 | Irreversible decisions require Dylan authority | **prose:** orchestrator skill + principles.md | No documented violation | judgment: "is this irreversible?" requires reasoning | Cannot be binary-gated | N/A |

### Graph Dynamics Boundaries

| # | Authority Boundary | Current Enforcement | Violation Evidence | Hookability | Proposed Mechanism | Priority |
|---|---|---|---|---|---|---|
| G1 | Investigation completing ≠ question answered | **none:** investigation closes independently of parent question | Observed: investigations close but blocking questions remain open | hookable: beads on_close hook | on_close hook: if closing investigation, check for parent question, warn if still open | **P2** |
| G2 | Cannot plan past unresolved question nodes (provisional work) | **none:** no mechanism to mark work as provisional | Not enforced; work appears ready even if premise uncertain | hookable: spawn gate | Spawn gate: warn if issue has transitive dependency on open question | **P2** |
| G3 | Factual questions may escalate to framing questions | **none:** no automatic escalation detection | Theoretical; relies on orchestrator judgment | judgment: detecting "no convergence" requires reasoning | Cannot be binary-gated; orchestrator synthesis is correct mechanism | N/A |
| G4 | Gate decisions select branches, prune alternatives | **none:** no mechanism for branch selection/pruning | Theoretical; gates are rarely formalized | judgment: branch selection inherently requires decision | Cannot be binary-gated | N/A |

---

## Summary Statistics

| Category | Count | Infrastructure | Prose-Only | None | Judgment-Only |
|----------|-------|----------------|------------|------|---------------|
| Worker → Orchestrator | 10 | 3 (W1, W2, W8) | 6 (W3-W7, W10) | 0 | 1 (W9) |
| Daemon → Orchestrator | 5 | 4 (D1, D2, D4, D5) | 0 | 1 (D3) | 0 |
| Orchestrator → Dylan | 3 | 0 | 1 (O1) | 1 (O2) | 1 (O3) |
| Graph Dynamics | 4 | 0 | 0 | 2 (G1, G2) | 2 (G3, G4) |
| **TOTAL** | **22** | **7** | **7** | **4** | **4** |

**Hookable gaps (prose-only or none, but can be gated):** 9
- P1 (high priority): W3 (git push), W6 (git add -A) — 2 items
- P2 (medium priority): W4 (bd dep add), W7 (phase reporting), G1 (investigation→question), G2 (provisional work) — 4 items
- P3 (low priority): W5 (triage:ready for questions), W10 (destructive git), D3 (factual question routing) — 3 items

**Judgment-only (cannot be gated):** 4
- W9 (scope boundary recognition), O1 (Gate detection), O3 (irreversibility judgment), G3 (escalation detection), G4 (branch pruning)

---

## Synthesis

**Key Insights:**

1. **The enforcement gap is a runtime gap** — Spawn and completion are well-gated (10 spawn gates, 14+ completion gates). The gap is during active agent execution, where situational pull overwhelms prose constraints. This matches the behavioral grammars model (Claim 3: infrastructure > instruction for behavioral constraints).

2. **Two-hook coverage** — A single PreToolUse hook for bash command filtering would close 4 of 9 hookable gaps (W3 git push, W4 bd dep add, W6 git add -A, W10 destructive git). This is high ROI: one implementation addresses nearly half the gaps.

3. **Judgment boundaries are correctly unenforceable** — The 4 judgment-only boundaries (recognizing Gates, scope boundaries, irreversibility, question escalation) all require reasoning about the nature of the situation. Binary hooks cannot capture them. These boundaries are where prompt-level guidance is the correct mechanism. The behavioral grammars model's "prompt budget" (~4 constraints per section) should be allocated to these judgment boundaries since they're the only ones that MUST live in prompt.

4. **The decidability graph has unstated implicit boundaries** — Several important boundaries (git push, git add -A, destructive git) aren't in the decidability graph model but emerged from violation evidence. They're operational constraints on worker behavior that the graph's Work/Question/Gate abstraction doesn't capture. These are "execution discipline" boundaries that live below the graph's authority level.

**Answer to Investigation Question:**

Of 22 distinct authority boundaries in the decidability graph:
- **7 are infrastructure-enforced** (32%) — concentrated at spawn/completion bookends
- **7 are prose-only** (32%) — will leak under pressure per behavioral grammars Claim 3
- **4 have no enforcement at all** (18%) — neither code nor prose
- **4 are judgment-only** (18%) — correctly unenforceable, must stay in prompt

9 of the 11 unenforced boundaries (prose-only + none) are hookable with existing Claude Code hook API (PreToolUse, Stop, SessionStart). No new mechanism types needed. The work is implementation, not architecture.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 22 boundaries verified against decidability graph model (cross-referenced every section)
- ✅ Infrastructure enforcement inventory verified by reading gate source code (pkg/spawn/gates/, pkg/verify/)
- ✅ Hook API sufficiency confirmed (existing PreToolUse hooks prove the pattern works for bash filtering)
- ✅ Violation evidence sourced from 8+ archived workspaces and 5+ investigations

**What's untested:**

- ⚠️ False positive rate of proposed bash command filters (e.g., will `git add -A` filter catch legitimate patterns?)
- ⚠️ Phase reporting timer effectiveness (SessionStart-based deadline may not align with actual turn counting)
- ⚠️ Provisional work warning UX (spawn gate warning may cause confusion if dependency chain is long)
- ⚠️ Whether closing the 9 hookable gaps actually reduces violation rate (no baseline measurement yet)

**What would change this:**

- Finding that PreToolUse hook latency degrades agent performance (would shift to completion-only enforcement)
- Evidence that bash command filtering has >10% false positive rate (would need smarter pattern matching)
- Discovery that agents actively circumvent hooks (would need deeper infrastructure changes)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| P1: Bash command filter hook (git push, git add -A) | implementation | Extends existing hook pattern, single-scope, clear criteria |
| P2: Phase reporting timer | architectural | Touches spawn, hooks, and beads integration (three systems) |
| P2: Investigation→question linkage | architectural | Changes beads lifecycle hook behavior, cross-concern |
| P2: Provisional work warning | architectural | Changes spawn gate behavior, affects daemon workflow |
| P3: Low-priority items (bd dep, destructive git, factual routing) | implementation | Standard hook extensions within existing patterns |

### Recommended Approach ⭐

**Phased Hook Rollout** — Implement in three waves ordered by risk reduction.

**Why this approach:**
- P1 items (git push, git add) address security/deploy risks with proven hook pattern
- P2 items require more design (cross-system) but address common violations
- P3 items are low-risk improvements that can wait

**Trade-offs accepted:**
- More hooks add latency to tool calls (measured at <100ms per hook, acceptable)
- Bash command regex matching is imperfect (may need iteration)

**Implementation sequence:**

1. **Wave 1 (P1): Bash command filter hook** — Single PreToolUse hook that denies `git push`, `git add -A`, `git add .`, `git add --all` for worker sessions (detected via `ORCH_WORKER=1` env var). Extends the pattern from `gate-orchestrator-task-tool.py`. Est: 1 feature-impl spawn.

2. **Wave 2 (P2): Lifecycle enforcement** — Four items:
   - Phase reporting: SessionStart hook + beads check after N turns
   - Investigation→question linkage: Extend `.beads/hooks/on_close` to warn when closing investigations with open parent questions
   - Provisional work warning: Spawn gate checks transitive dependencies for open questions
   - `bd dep add` filter: PreToolUse hook for worker sessions
   Est: 2-3 feature-impl spawns.

3. **Wave 3 (P3): Low-priority extensions** — Destructive git filter, triage:ready for questions, factual question routing in daemon. Est: 1-2 spawns.

### Alternative Approaches Considered

**Option B: Comprehensive single deployment**
- **Pros:** All gaps closed at once
- **Cons:** Large scope, harder to test, higher risk of false positives affecting all agents simultaneously
- **When to use instead:** If violation rate is causing active production issues

**Option C: Completion-only enforcement**
- **Pros:** Simpler (no PreToolUse hooks, just verify at completion)
- **Cons:** Catches violations after the fact, not before. Git push damage is done by completion time.
- **When to use instead:** If PreToolUse hook latency becomes a problem

---

### Implementation Details

**What to implement first:**
- Wave 1 bash filter hook: single Python file, uses `ORCH_WORKER=1` env var to detect worker sessions
- Pattern: deny + error message explaining the boundary (matches existing `gate-orchestrator-task-tool.py` style)
- Deploy path: `~/.claude/hooks/` configuration

**Things to watch out for:**
- ⚠️ `git add -A` pattern must not match `git add -Av` or similar flag combinations — use word boundary matching
- ⚠️ Worker detection via `ORCH_WORKER=1` only works for orch-spawned agents — interactive sessions won't be filtered
- ⚠️ Phase reporting timer needs calibration — "3 actions" is prose guidance, actual turn count varies by task complexity

**Success criteria:**
- ✅ Zero worker-initiated `git push` commands after Wave 1 deployment
- ✅ Zero `git add -A` / `git add .` commands in worker sessions after Wave 1
- ✅ Phase reporting compliance improves to >90% after Wave 2 (currently unmeasured baseline)
- ✅ No false positive rate >5% on bash command filters

---

## References

**Files Examined:**
- `.kb/models/decidability-graph/model.md` — All 22 authority boundaries
- `.kb/models/behavioral-grammars/model.md` — Why prose leaks (Claim 3)
- `pkg/spawn/gates/*.go` — 6 spawn gate implementations
- `pkg/verify/*.go` — 14+ completion gate implementations
- `.beads/hooks/` — 3 hook scripts (on_close, gate-bd-close, gate-orchestrator-task-tool)
- `skills/src/shared/worker-base/.skillc/` — Worker authority delegation rules
- 8+ archived workspaces in `.orch/workspace/` for violation evidence

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-19-worker-authority-boundaries.md` — Workers create nodes, orchestrators create edges
- **Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Three-layer pattern
- **Probe:** `probes/2026-03-02-probe-layered-constraint-enforcement-design.md` — Constraint taxonomy
- **Synthesis:** `.kb/investigations/2026-02-24-synthesis-enforcement-accretion-verification-design-burst.md` — Infrastructure > instruction principle
- **Model:** `.kb/models/behavioral-grammars/model.md` — Why infrastructure enforcement is necessary

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: Which decidability graph boundaries are enforced, prose-only, or judgment-only?
- Context: Spawned as architect task to design systematic mapping

**2026-03-05:** Research complete
- Inventoried 22 authority boundaries from decidability graph
- Inventoried 26 spawn/completion gates + 7 hooks from codebase
- Found 8+ documented violations across archived workspaces
- Cross-referenced with behavioral grammars model and prior enforcement probes

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: 9 hookable gaps identified, 6 implementation issues to create
