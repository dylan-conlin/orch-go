# Session Synthesis

**Agent:** og-inv-audit-38-stale-22mar-d21f
**Issue:** orch-go-yuod9
**Outcome:** success

---

## Plain-Language Summary

Audited all 62 non-archived project decisions against enforcement evidence in code (`pkg/`, `cmd/`), skills (`skills/src/`), hooks (`.claude/`), and CLAUDE.md. Of the 38 decisions with 0 citations in `kb reflect`, **17 are actually still enforced in code or skills** (they were implemented but the decision document was never cited by the implementing code). 14 are genuinely aspirational — decided but never enforced. 3 are superseded by later decisions. 4 govern things that no longer exist or were one-time actions.

The most actionable finding: the 14 aspirational decisions represent governance debt — decisions that were made but never followed through on. Three clusters stand out: (1) publication gates (3 decisions, zero code), (2) philosophical principles that were never operationalized (5 decisions), and (3) half-built infrastructure where the signal is produced but never consumed (role-aware hook filtering).

---

## TLDR

38/62 decisions flagged stale by kb reflect. 17 are actually ACTIVE (implemented in code but decision doc not cited). 14 are ASPIRATIONAL (never enforced). 3 SUPERSEDED. 4 REMOVED. Recommend archiving 7 (3 superseded + 4 removed), flagging 14 aspirational for decide-or-delete review.

---

## Classification Table

### ACTIVE — Implemented but decision document not cited (17)

These decisions ARE enforced in code, skills, or configuration. The decision document simply isn't referenced by the enforcement mechanism. **No action needed** — they're working silently.

| # | Decision | Date | Enforcement Location | Notes |
|---|----------|------|---------------------|-------|
| 1 | `minimal-artifact-taxonomy` | 2025-12-21 | `.kb/models/`, `.kb/investigations/`, `.kb/decisions/` directory structure | Taxonomy is the active artifact system |
| 2 | `template-ownership-model` | 2025-12-22 | `skills/src/*/.skillc/skill.yaml` → SKILL.md.template pattern | skillc deploy enforces this |
| 3 | `strategic-orchestrator-model` | 2026-01-07 | `skills/src/meta/orchestrator/SKILL.md` embodies comprehension focus | Guides cite it |
| 4 | `synthesis-is-strategic-orchestrator-work` | 2026-01-07 | Worker skills require SYNTHESIS.md; orchestrator does synthesis | Active protocol |
| 5 | `dual-spawn-mode-architecture` | 2026-01-09 | `pkg/orch/spawn_backend.go`, `pkg/spawn/backends/*.go` | Fully implemented |
| 6 | `launchd-supervision-architecture` | 2026-01-10 | `cmd/orch/daemon_launchd.go`, `pkg/daemonconfig/plist.go` | Active infrastructure |
| 7 | `two-tier-cleanup-pattern` | 2026-01-14 | `pkg/agent/lifecycle_impl.go` (event) + `pkg/daemon/cleanup.go` (periodic) | Both tiers active |
| 8 | `ghost-visibility-over-cleanup` | 2026-01-15 | `pkg/agent/lifecycle_impl.go:124`, `pkg/agent/filters.go`, `pkg/daemon/active_count.go` | Pervasive ghost prevention |
| 9 | `event-sourced-monitoring-architecture` | 2026-01-17 | `pkg/opencode/sse.go`, `pkg/opencode/monitor.go`, `pkg/activity/export.go` | SSE infra actively used |
| 10 | `file-based-workspace-state-detection` | 2026-01-17 | `pkg/workspace/workspace.go` uses `.spawn_time`, `SYNTHESIS.md`, etc. | File metadata drives state |
| 11 | `questions-as-first-class-entities` | 2026-01-18 | `pkg/spawn/gates/question.go`, `pkg/daemon/question_detector.go`, `compliance.go` | Full code implementation |
| 12 | `recommendation-authority-classification` | 2026-01-30 | `skills/src/worker/architect/SKILL.md` (authority classification taxonomy) | Skill text enforcement |
| 13 | `investigation-lineage-enforcement` | 2026-01-31 | `pkg/kbgate/publish.go` `checkLineage()` — Gate 3 of publication pipeline | Code gate, blocks endogenous-only evidence |
| 14 | `model-centric-probes-replace-investigations` | 2026-02-08 | Investigation skill "Artifact Mode Selection (Probe Default)" in worker-base | Skill text redirects to probes |
| 15 | `orchestrator-skill-orientation-redesign` | 2026-02-16 | `skills/src/meta/orchestrator/SKILL.md` — orientation moments, `orch orient` | Materially implemented |
| 16 | `no-code-review-gate-expand-execution-verification` | 2026-02-25 | No code review gate exists (enforced by absence); `go vet` in V2+ build gate | Active constraint |
| 17 | `atc-not-conductor-orchestrator-reframe` | 2026-02-28 | Influenced orchestrator skill structure; cited by `2026-03-12-atc-instruments` | Active mental model |

### SUPERSEDED — Replaced by later decision (3)

These were valid at the time but a subsequent decision explicitly replaced them. **Recommend: archive with supersession note.**

| # | Decision | Date | Superseded By | Reason |
|---|----------|------|--------------|--------|
| 1 | `health-score-floor-gate-downgraded-from-blocking-t` | 2026-03-10 | `2026-03-11-remove-health-score-spawn-gate` | Gate removed entirely 1 day later |
| 2 | `health-score-targets-65-floor-gate-80-target` | 2026-03-10 | `2026-03-11-remove-health-score-spawn-gate` | Health score still computed for observability, but targets are moot since gate was removed |
| 3 | `registry-contract-spawn-cache-only` | 2026-01-14 | `pkg/registry` eliminated; CLAUDE.md says "No Local Agent State" | Registry package deleted, contract governs nothing |

### REMOVED — Thing it governed no longer exists (4)

These are historical records of completed actions or decisions about things that were never built. **Recommend: archive as historical record.**

| # | Decision | Date | Why Removed |
|---|----------|------|------------|
| 1 | `beads-oss-relationship-clean-slate` | 2025-12-21 | One-time migration decision ("drop local features, use upstream"). Action was executed. |
| 2 | `orchestrator-system-resource-visibility` | 2025-12-25 | Decided "no" — orchestrator should NOT monitor system resources. The thing was never built. Decision is a negative record. |
| 3 | `remove-health-score-spawn-gate` | 2026-03-11 | Removal was executed. Health score gate is gone from spawn path. Tombstone only. |
| 4 | `remove-self-review-completion-gate` | 2026-03-13 | Removal was executed. Self-review gate gone (tombstone in `check.go:29`). Historical record. |

### ASPIRATIONAL — Decided but never enforced (14)

These are the interesting ones. Decisions were made, often with specific implementation plans, but no enforcement exists in code, skills, or hooks. **These represent governance debt or dead intent.**

| # | Decision | Date | What Was Decided | Why It's Aspirational |
|---|----------|------|-----------------|----------------------|
| 1 | `capture-at-context` | 2026-01-14 | Forcing functions fire at context time (spawn/completion), not after | Referenced in `principles.md` but no code enforces capture timing |
| 2 | `schema-migration-pattern` | 2026-01-14 | Backward-compatible discovery + optional migration tooling for schema changes | Never implemented. Only appears as test fixture title. |
| 3 | `separate-observation-from-intervention` | 2026-01-14 | Decouple passive observation from active intervention logic | Philosophical. No code separates these concerns architecturally. |
| 4 | `trust-calibration-assert-knowledge` | 2026-01-14 | Surface user expertise level to AI decision-making | No trust calibration mechanism in code. Found only in evidence/ behavioral baselines. |
| 5 | `role-aware-hook-filtering` | 2026-01-17 | Claude Code hooks check CLAUDE_CONTEXT to filter by role | **Half-built:** `CLAUDE_CONTEXT` IS set by spawn (`claude.go:146`) but NO hook reads it. Signal produced, never consumed. |
| 6 | `three-tier-workspace-hierarchy` | 2026-01-17 | Three distinct workspace types with naming conventions | No code implements workspace "tiers." Workspace package has no tier concept. |
| 7 | `auto-memory-kb-cli-reconciliation` | 2026-02-25 | Define boundary between auto-memory (tactical) and kb-cli (durable) | Boundary definition only. No code enforces the boundary. |
| 8 | `plan-mode-incompatible-with-daemon-spawned-agents` | 2026-02-26 | Don't use plan mode in daemon-spawned agents | Documented but no gate blocks it. No skill text warns against it. |
| 9 | `remediate-configuration-drift-defect-class` | 2026-03-05 | Address configuration-drift as a defect pattern | Philosophical. No drift-detection mechanism or prevention code. |
| 10 | `five-design-principles-for-automation-legibility` | 2026-03-08 | Five principles for human-automation interaction design | Design principles with no code enforcement or skill text embedding. |
| 11 | `grammar-first-architecture-4-behavioral-slots-are` | 2026-03-08 | Behavioral skill slots as paired items (delegation, undefined-behavior, etc.) | Partially reflected in skill structure but no enforcement mechanism. |
| 12 | `adopt-uncontaminated-codex-gate-design-claim-ledg` | 2026-03-10 | Claim ledger + red-team memo + claim-label pass for publication | **Publication gates were never built.** Zero code in `pkg/` or `cmd/`. |
| 13 | `codex-cli-as-external-adversarial-reviewer-for-pub` | 2026-03-10 | Use Codex CLI as adversarial reviewer for publication claims | Never built. No Codex CLI integration. |
| 14 | `publication-gate-requires-three-artifacts-before-p` | 2026-03-10 | Publication requires claim ledger, red-team memo, claim-label pass | Never built. All three publication gate decisions are aspirational. |

---

## Summary by Classification

| Classification | Count | Recommendation |
|---------------|-------|----------------|
| **ACTIVE** (enforced, just not cited) | 17 | Add code comments citing decision files, or accept silent enforcement |
| **SUPERSEDED** | 3 | Archive to `.kb/decisions/archived/` with supersession note |
| **REMOVED** | 4 | Archive to `.kb/decisions/archived/` as historical record |
| **ASPIRATIONAL** | 14 | Review: implement, delete, or explicitly shelve with rationale |

## Aspirational Decision Clusters

The 14 aspirational decisions cluster into three groups:

**Cluster 1: Publication gates (3 decisions, zero code)**
- `adopt-uncontaminated-codex-gate-design-claim-ledg`
- `codex-cli-as-external-adversarial-reviewer-for-pub`
- `publication-gate-requires-three-artifacts-before-p`

All from 2026-03-10. These were designed as a publication verification system that was never built. **Decision needed:** Build it, shelve it, or delete the decisions.

**Cluster 2: Philosophical principles never operationalized (5 decisions)**
- `capture-at-context`
- `separate-observation-from-intervention`
- `trust-calibration-assert-knowledge`
- `five-design-principles-for-automation-legibility`
- `remediate-configuration-drift-defect-class`

These are design principles or architectural aspirations that informed thinking but were never translated into enforcement mechanisms. **Decision needed:** Accept as principles (move to `.kb/global/principles.md`) or delete.

**Cluster 3: Half-built or convention-only (6 decisions)**
- `role-aware-hook-filtering` — signal produced, never consumed
- `three-tier-workspace-hierarchy` — no tier concept in workspace code
- `schema-migration-pattern` — never implemented
- `auto-memory-kb-cli-reconciliation` — boundary defined, never enforced
- `plan-mode-incompatible-with-daemon-spawned-agents` — no gate
- `grammar-first-architecture-4-behavioral-slots-are` — partial

These represent incomplete implementations or conventions that were never reinforced. **Decision needed:** Complete the implementation, add enforcement, or acknowledge as convention-only and possibly archive.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

**Key evidence:**
- Searched `pkg/`, `cmd/`, `skills/src/`, `.claude/`, `CLAUDE.md` for enforcement patterns
- Used ripgrep for 40+ pattern searches across codebase
- Cross-referenced decision file citations against code, skill text, and configuration
- Validated key findings: escalation levels in `pkg/verify/escalation.go`, workspace state in `pkg/workspace/workspace.go`, ghost filtering in `pkg/agent/lifecycle_impl.go`, lineage enforcement in `pkg/kbgate/publish.go`

---

## Next

**Recommendation:** close

### If Close
- [x] Classification table complete for all 38 stale decisions
- [x] Each classified as ACTIVE/SUPERSEDED/REMOVED/ASPIRATIONAL
- [x] Archival recommendations provided
- [x] Aspirational clusters identified for follow-up decision
- [ ] Orchestrator makes final calls on archival and aspirational decisions

---

## Unexplored Questions

- **BudgetCap = 30 in `pkg/decisions/lifecycle.go`**: The decisions package has a budget cap of 30 active decisions. With 62 non-archived, we're 2x over budget. Is this cap enforced anywhere?
- **Global decisions stale status**: This audit focused on 62 project decisions. The 15 global decisions weren't included in the kb reflect "38/62" count — should they be audited separately?
- **Citation bootstrapping**: 17 decisions are active but uncited. Would adding `// Decision: .kb/decisions/YYYY-MM-DD-name.md` comments to implementing code solve the staleness signal?

---

## Friction

No friction — smooth session. Parallel agent searches covered the codebase efficiently.

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-audit-38-stale-22mar-d21f/`
**Investigation:** `.kb/investigations/2026-03-22-inv-audit-38-stale-decisions-still.md`
**Beads:** `bd show orch-go-yuod9`
