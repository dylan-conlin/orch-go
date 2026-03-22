## Summary (D.E.K.N.)

**Delta:** Of 38 stale decisions (0 citations), 17 are actually enforced in code/skills, 14 are genuinely aspirational (never enforced), 3 superseded, 4 removed.

**Evidence:** Grep of 40+ patterns across pkg/, cmd/, skills/src/, .claude/, CLAUDE.md; verified key implementations (escalation levels, workspace state detection, lineage enforcement, ghost filtering).

**Knowledge:** "Stale" per kb reflect (0 citations) does NOT mean "unenforced" — 45% of stale decisions are silently active in code. The real governance debt is 14 aspirational decisions, clustered around publication gates (3), philosophical principles (5), and half-built infrastructure (6).

**Next:** Archive 7 (3 superseded + 4 removed). Orchestrator reviews 14 aspirational: implement, shelve, or delete.

**Authority:** architectural — Cross-cuts knowledge system, governance, and code infrastructure

---

# Investigation: Audit 38 Stale Decisions — Which Are Still Enforced?

**Question:** Of the 38 decisions with 0 citations in kb reflect, which are still enforced in code, and which are orphaned governance?

**Started:** 2026-03-22
**Updated:** 2026-03-22
**Owner:** og-inv-audit-38-stale-22mar-d21f
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: 17 stale decisions are actually enforced in code

**Evidence:** Decisions like `questions-as-first-class-entities` (implemented in `pkg/spawn/gates/question.go`, `pkg/daemon/question_detector.go`), `investigation-lineage-enforcement` (code gate in `pkg/kbgate/publish.go`), and `event-sourced-monitoring-architecture` (SSE in `pkg/opencode/sse.go`) are fully implemented. They're "stale" only because the implementing code doesn't cite the decision document.

**Source:** Grep patterns for QuestionDetectionResult, checkLineage, SSEClient, EscalationLevel, ServiceMonitor, etc. across pkg/ and cmd/

**Significance:** The "staleness" metric conflates "uncited" with "unenforced." Nearly half the stale decisions are working silently. Adding `// Decision: .kb/decisions/...` comments to implementing code would resolve this.

---

### Finding 2: Publication gate cluster is pure aspiration (3 decisions, 0 code)

**Evidence:** Three decisions from 2026-03-10 describe a publication verification system (claim ledger, red-team memo, claim-label pass, Codex CLI review). Zero lines of code exist for any of this in pkg/ or cmd/. No configuration, no hooks, no skill text references.

**Source:** Grep for `publication.gate`, `claim.ledger`, `red.team`, `codex.cli` across entire codebase — all zero matches in Go files.

**Significance:** These represent the largest cluster of governance debt. Three decisions were made in a single day about a system that was never built.

---

### Finding 3: Role-aware hook filtering is half-built

**Evidence:** `CLAUDE_CONTEXT` env var IS set by spawn (`pkg/spawn/claude.go:146` exports `worker`/`orchestrator`/`meta-orchestrator`). Tests verify it. But the ONLY hook (`.claude/hooks/gate-git-add-all.py`) doesn't read `CLAUDE_CONTEXT` — it uses `SKIP_GIT_ADD_ALL_GATE=1` instead. The signal is produced but never consumed.

**Source:** `pkg/spawn/claude.go:146`, `pkg/spawn/claude_test.go`, `.claude/hooks/gate-git-add-all.py`, `.claude/settings.local.json`

**Significance:** This is a concrete example of configuration drift: infrastructure was built (signal production) but the consuming side (hook filtering) was never wired up.

---

## Synthesis

**Key Insights:**

1. **Staleness ≠ unenforcement** — kb reflect's citation graph misses enforcement-by-implementation. 17/38 stale decisions are silently active. The metric needs a secondary check: "does the concept this decision describes exist in code?"

2. **Governance-stripping era left clean tombstones** — The decisions that REMOVED gates (health score, self-review, accretion blocking) correctly documented the removal. But the earlier decisions they superseded (health score targets, floor gate downgrade) weren't archived, creating a chain of superseded-but-uncleaned decisions.

3. **Aspirational decisions cluster temporally** — 8 of the 14 aspirational decisions are from Jan 14, 2026 or Mar 8-10, 2026, suggesting bursts of decision-making without implementation follow-through.

**Answer to Investigation Question:**

Of 38 stale decisions: 17 ACTIVE (enforced in code/skills despite 0 citations), 3 SUPERSEDED (replaced by later decisions), 4 REMOVED (historical records of completed actions), 14 ASPIRATIONAL (decided but never enforced). Full classification table in SYNTHESIS.md.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code enforcement of 17 "active" decisions (verified via grep: question gates, escalation levels, SSE, ghost filtering, lineage enforcement, etc.)
- ✅ Absence of publication gate code (grep for claim_ledger, red_team, publication_gate — zero Go matches)
- ✅ CLAUDE_CONTEXT production but non-consumption (spawn sets it, no hook reads it)

**What's untested:**

- ⚠️ Whether kb reflect actually reports exactly these 38 as stale (I inferred from the "0 citations" criterion)
- ⚠️ Whether some "active" decisions have drifted from their original intent (I checked concept existence, not full specification compliance)
- ⚠️ Global decisions staleness (15 global decisions were outside the 62 scope)

**What would change this:**

- Running `kb reflect --format json` and comparing its stale list to my classification
- If any "aspirational" decision has enforcement in a hook or script I didn't search

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Archive 7 superseded/removed decisions | implementation | Mechanical cleanup, no judgment needed |
| Review 14 aspirational decisions | strategic | Involves resource commitment and value judgment on what to build |
| Add citation comments to 17 active decisions | implementation | Improves kb reflect accuracy, no behavior change |

### Recommended Approach ⭐

**Three-phase cleanup** — Archive dead decisions, cite living ones, decide on aspirational.

**Implementation sequence:**
1. Archive 3 superseded + 4 removed decisions to `.kb/decisions/archived/`
2. Add `// Decision: .kb/decisions/YYYY-MM-DD-name.md` comments to 17 implementing files
3. Orchestrator reviews 14 aspirational: for each, decide implement/shelve/delete

---

## References

**Files Examined:**
- `pkg/spawn/gates/question.go` — question-as-first-class enforcement
- `pkg/kbgate/publish.go` — lineage enforcement gate
- `pkg/verify/escalation.go` — five-tier escalation levels
- `pkg/workspace/workspace.go` — file-based workspace state
- `pkg/opencode/sse.go` — event-sourced monitoring
- `pkg/agent/lifecycle_impl.go` — ghost visibility filtering
- `pkg/daemon/cleanup.go` — two-tier cleanup (periodic)
- `pkg/spawn/claude.go` — CLAUDE_CONTEXT production
- `pkg/decisions/lifecycle.go` — staleness detection infrastructure
- `.claude/hooks/gate-git-add-all.py` — only hook (doesn't read CLAUDE_CONTEXT)
- `skills/src/meta/orchestrator/SKILL.md` — orchestrator skill orientation
- `skills/src/worker/architect/SKILL.md` — authority classification
- All 62 decision files in `.kb/decisions/` (non-archived)
