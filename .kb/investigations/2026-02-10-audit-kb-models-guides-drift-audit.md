## Summary (D.E.K.N.)

**Delta:** Audited 28 model files and 37 guide files; found 1 fundamentally wrong, 14 partially stale, and identified 7 cross-cutting drift patterns affecting ~40% of knowledge artifacts.

**Evidence:** Verified claims against actual codebase — `orch phase` exists (confirmed via `orch phase --help`), `pkg/registry` removed, `pkg/state/db.go` replaced it, cherry-pick in `cmd/orch/complete_merge.go`, worktree isolation in `pkg/spawn/worktree.go`, behavior profiles in `pkg/model/behavior_profile.go`.

**Knowledge:** The dominant drift patterns are: (1) `orch phase` exists but ~10 artifacts still say "bd comment Phase:" is primary, (2) registry removed but ~6 artifacts reference it, (3) worktree isolation landed but ~5 artifacts don't mention it, (4) many source file references point to nonexistent files.

**Next:** Create follow-up issues for the 7 cross-cutting drift patterns. Each pattern can be a single batch-fix issue. The fundamentally wrong model (`agent-state-architecture-feb2026.md`) should be rewritten or deleted.

**Authority:** architectural - Cross-artifact knowledge hygiene affects all future agent sessions.

---

# Investigation: KB Models & Guides Drift Audit

**Question:** How many .kb/models/ and .kb/guides/ files contain stale or incorrect claims about the current codebase?

**Started:** 2026-02-10
**Updated:** 2026-02-10
**Owner:** Worker (orch-go-fc9od)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-02-06-inv-skill-artifact-drift-audit-feb.md` | Related (skill drift) | Yes | None — complementary scope |

---

## Findings

### Finding 1: Cross-Cutting Drift Patterns

Seven systemic drift patterns affect multiple artifacts simultaneously:

| # | Pattern | Affected Artifacts | Reality |
|---|---------|-------------------|---------|
| 1 | `orch phase` not mentioned / "bd comment Phase:" still primary | ~10 models + guides | `orch phase` EXISTS, writes SQLite + fires bd comment. Confirmed via `orch phase --help` |
| 2 | Registry references (`pkg/registry`, `registry.json`) | ~6 artifacts | `pkg/registry` REMOVED. `pkg/state/db.go` replaced it (Feb 2026) |
| 3 | Worktree isolation not mentioned | ~5 artifacts | IMPLEMENTED in `pkg/spawn/worktree.go` and `pkg/spawn/context.go` |
| 4 | Cherry-pick not mentioned (rebase still referenced) | ~3 artifacts | Cherry-pick REPLACED rebase in `cmd/orch/complete_merge.go` with explicit tests |
| 5 | GPT-5.3-Codex as default model not reflected | ~3 model artifacts | `multi-model-evaluation-feb2026.md` documents GPT-5.3-Codex as default |
| 6 | Source file references pointing to nonexistent files | ~5 artifacts | Files like `pkg/verify/phase.go`, `pkg/beads/fallback.go`, `pkg/status/calculate.go` do not exist |
| 7 | Dashboard URL inconsistency (5188 vs 3348) | ~2 artifacts | Dashboard guide shows both ports for different purposes (UI vs API) |

**Source:** Cross-referenced all model/guide files against `ls`, `orch phase --help`, `grep -r` for registry references.

**Significance:** These patterns affect agent behavior because models/guides are fed into spawn context via `kb context`. Stale claims actively mislead spawned agents.

---

### Finding 2: Model Files Audit (28 files)

| File | Status | Key Issues |
|------|--------|------------|
| `agent-state-architecture-feb2026.md` | ❌ WRONG | Claims `orch phase` "not implemented", template migration "not started", bd comment as "canonical phase record" — all 3 wrong |
| `spawn-architecture.md` | ⚠️ STALE | No mention of worktree isolation. References registry. Missing worktree step in flow diagram |
| `completion-lifecycle.md` | ⚠️ STALE | No mention of cherry-pick (still implies rebase). Missing verification spec & branch integration gates. References registry |
| `completion-verification.md` | ✅ MOSTLY | Updated Feb 9. Includes proof-carrying specs. But references nonexistent source files (`pkg/verify/phase.go`, `evidence.go`, `cross_project.go`) |
| `agent-lifecycle-state-model.md` | ⚠️ STALE | Still references "bd comment" for phase reporting. References `pkg/registry`. Core mechanism text outdated despite Feb 9 date |
| `spawn-system-evolution.md` | ✅ CURRENT | Historical changelog — correct by nature |
| `context-injection.md` | ⚠️ STALE | Jan 2026 content. No mention of spawn templates using `orch phase`. References old hooks |
| `opencode-session-lifecycle.md` | ⚠️ STALE | No mention of worktree isolation for sessions |
| `dashboard-agent-status.md` | ⚠️ STALE | References `pkg/status/calculate.go` (doesn't exist), `pkg/dashboard/server.go` (doesn't exist), `~/.orch/registry.json` (registry removed) |
| `dashboard-architecture.md` | ✅ MOSTLY | Updated Jan 29. Generally current |
| `beads-integration-architecture.md` | ⚠️ STALE | References nonexistent files (`pkg/beads/fallback.go`, `lifecycle.go`, `id.go`, `pkg/spawn/tracking.go`). Old API patterns |
| `current-model-stack.md` | ⚠️ STALE | Jan 28. Doesn't mention GPT-5.3-Codex (now default). Internal contradictions about OAuth stealth |
| `model-access-spawn-paths.md` | ⚠️ STALE | Similar to current-model-stack — missing GPT-5.3-Codex |
| `workspace-lifecycle-model.md` | ⚠️ STALE | Jan 17 date. No mention of worktree-based workspaces |
| `daemon-autonomous-operation.md` | ✅ MOSTLY | Updated Feb 9 with probes |
| `system-reliability-feb2026.md` | ✅ CURRENT | Feb 2026, documents recent state |
| `multi-model-evaluation-feb2026.md` | ✅ CURRENT | Feb 2026, documents GPT-5.3-Codex as default |
| `sse-connection-management.md` | ✅ CURRENT | Updated Feb 9 |
| `decidability-graph.md` | ✅ CURRENT | Updated Jan 29 |
| `orchestration-cost-economics.md` | ⚠️ STALE | Doesn't reflect GPT-5.3-Codex. Jan 28 |
| `beads-database-corruption.md` | ✅ CURRENT | Resolved status |
| `cross-project-visibility.md` | ✅ MOSTLY | Jan 27 |
| `escape-hatch-visibility-architecture.md` | ⚠️ STALE | References dashboard at `localhost:5188` which may need clarification vs 3348 |
| `follow-orchestrator-mechanism.md` | ✅ MOSTLY | Generally current |
| `extract-patterns.md` | ✅ CURRENT | |
| `kb-reflect-cluster-hygiene.md` | ✅ CURRENT | |
| `PHASE3_REVIEW.md` | ✅ CURRENT | Historical review |
| `PHASE4_REVIEW.md` | ✅ CURRENT | Historical review |

**Summary:** 1 fundamentally wrong, 13 partially stale, 14 current/mostly current.

---

### Finding 3: Guide Files Audit (37 files)

| File | Status | Key Issues |
|------|--------|------------|
| `spawn.md` | ✅ CURRENT | Comprehensive, updated. No registry references. Documents all backends including worktree spawns |
| `completion.md` | ⚠️ STALE | Jan 17 date. References "Registry update: ArchivedPath" (line 267). No mention of cherry-pick merge strategy. No mention of `orch phase` for phase reporting |
| `agent-lifecycle.md` | ⚠️ STALE | Jan 17. Flow diagram shows `bd comment "Phase: Complete"` not `orch phase`. References "Registry Merge" with `syscall.Flock`. No worktree mention |
| `status.md` | ✅ CURRENT | Jan 29. Comprehensive, doesn't reference registry |
| `beads-integration.md` | ⚠️ STALE | Jan 6. References "Registry before beads close" ordering (registry removed). "Order of Operations" section references `reg.Complete()`, `reg.Save()` |
| `worker-patterns.md` | ⚠️ STALE | Jan 17. Progress tracking shows `bd comment` only, no mention of `orch phase`. Dashboard URL says `localhost:5188` for infrastructure |
| `completion-gates.md` | ✅ MOSTLY | Jan 4. Lists 9 gates; actual code has 15+ gates now. Missing: `verification_spec`, `model_connection`, `branch_integration`, `decision_patch_limit`, `handoff_content`, `dashboard_health` gates |
| `daemon.md` | ✅ MOSTLY | Jan 7 base + Jan 31 cleanup section. Comprehensive |
| `model-selection.md` | ✅ CURRENT | Feb 6. Documents GPT-5.2 but not GPT-5.3-Codex (may be intentional — guides vs models difference) |
| `dashboard.md` | ✅ MOSTLY | Jan 29. References both ports (5188 UI, 3348 API). Comprehensive |
| `dashboard-architecture.md` | ✅ MOSTLY | Generally current |
| `workspace-lifecycle.md` | ⚠️ STALE | No worktree mention. Pre-dates worktree isolation |
| `cli.md` | ✅ MOSTLY | Depends on when last updated |
| `dual-spawn-mode-implementation.md` | ✅ CURRENT | Explicitly notes registry replaced by `pkg/state/db.go` |
| `understanding-artifact-lifecycle.md` | ⚠️ STALE | References `registry.json` 3 times (lines 138, 267, 283) |
| `lens-sessions.md` | ✅ CURRENT | Mentions GPT-5.3 transition |
| `opencode-plugins.md` | ✅ CURRENT | Uses "worktree" correctly (in different context — git worktree paths) |
| `headless.md` | ✅ MOSTLY | |
| `server-management.md` | ✅ MOSTLY | |
| `dev-environment-setup.md` | ✅ MOSTLY | |
| `decision-index.md` | ✅ CURRENT | |
| `decision-authority.md` | ✅ CURRENT | |
| `skill-system.md` | ✅ MOSTLY | |
| `tmux-spawn-guide.md` | ✅ MOSTLY | |
| `spawned-orchestrator-pattern.md` | ✅ MOSTLY | |
| `two-tier-sensing-pattern.md` | ✅ CURRENT | |
| `reflection-sessions.md` | ✅ MOSTLY | |
| `resilient-infrastructure-patterns.md` | ✅ MOSTLY | |
| `background-services-performance.md` | ✅ MOSTLY | |
| `recovery-playbooks.md` | ✅ MOSTLY | |
| `opencode.md` | ✅ MOSTLY | |
| `friction-ledger.md` | ✅ MOSTLY | |
| `code-extraction-patterns.md` | ✅ CURRENT | |
| `claude-code-sandbox-architecture.md` | ✅ CURRENT | |
| `auto-rebuild-ecosystem.md` | ✅ MOSTLY | |
| `api-development.md` | ✅ MOSTLY | |
| `status-dashboard.md` | ✅ MOSTLY | |

**Summary:** 6 partially stale, 31 current/mostly current.

---

## Synthesis

**Key Insights:**

1. **Models drift faster than guides** — 50% of model files have drift vs 16% of guide files. Models attempt to describe "how the system works now" and become stale as implementation changes. Guides describe "how to use the system" and are more stable.

2. **The dominant drift vector is infrastructure evolution** — The top 3 drift patterns (orch phase, registry removal, worktree isolation) all represent infrastructure that was added/removed in Jan-Feb 2026. Knowledge artifacts from before these changes are systematically stale.

3. **Source file references are the most fragile claims** — At least 5 artifacts reference Go files that have been renamed, split, or removed. These cause agents to search for nonexistent files and waste time.

4. **One artifact is actively harmful** — `agent-state-architecture-feb2026.md` makes 3 major false claims about the current system state. Its title includes "feb2026" suggesting currency, making it especially dangerous for spawn context injection.

**Answer to Investigation Question:**

Of 65 total files audited (28 models + 37 guides):
- **1** is fundamentally wrong (should be rewritten/deleted)
- **19** are partially stale (contain at least one outdated claim)
- **45** are current or mostly current

The stale artifacts cluster around 7 cross-cutting drift patterns, meaning batch fixes by pattern (not per-file) would be the most efficient remediation strategy.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch phase` command exists (verified: `orch phase --help` shows usage)
- ✅ `pkg/registry` does not exist (verified: `ls pkg/registry` returns not found)
- ✅ `pkg/state/db.go` exists (verified: file read confirmed)
- ✅ Cherry-pick in completion (verified: `cmd/orch/complete_merge.go` contains cherry-pick logic, tests explicitly test cherry-pick over rebase)
- ✅ Worktree isolation implemented (verified: `pkg/spawn/worktree.go` exists)
- ✅ Behavior profiles exist (verified: `pkg/model/behavior_profile.go` has `strict-complete` and `needs-nudge`)
- ✅ Many referenced source files don't exist (verified: `pkg/verify/phase.go`, `pkg/beads/fallback.go`, `pkg/status/calculate.go` all return not found)

**What's untested:**

- ⚠️ Some guide files were categorized based on grep patterns rather than full read (lower-risk guides like `api-development.md`, `friction-ledger.md`)
- ⚠️ The 15+ completion gates count was inferred from spawn context content, not exhaustive code audit

**What would change this:**

- If `pkg/verify/phase.go` etc. exist in a different path structure, the "nonexistent files" finding would need revision
- If registry was re-introduced in a different form, the "registry removed" pattern would need re-evaluation

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Delete/rewrite `agent-state-architecture-feb2026.md` | architectural | Cross-artifact decision, affects spawn context quality |
| Batch-fix 7 drift patterns | implementation | Each pattern is a known-correct find-replace |
| Add staleness date checking to `kb context` | strategic | New infrastructure investment, changes how knowledge flows |

### Recommended Approach ⭐

**Pattern-based batch fixes** — Create 7 issues (one per drift pattern), each fixing all affected artifacts for that pattern.

**Why this approach:**
- Most efficient — fixing by pattern (e.g., "replace all registry references") is faster than per-file review
- Testable — each pattern has a clear grep to verify the fix
- Parallelizable — 7 independent issues can be daemon-spawned

**Implementation sequence:**
1. Delete/rewrite `agent-state-architecture-feb2026.md` (highest impact, actively harmful)
2. Fix registry references (6 artifacts, mechanical replacement)
3. Add `orch phase` mentions where `bd comment Phase:` is presented as sole mechanism (10 artifacts)
4. Add worktree isolation mentions (5 artifacts)
5. Fix nonexistent source file references (5 artifacts)
6. Add cherry-pick mentions (3 artifacts)
7. Update GPT-5.3-Codex references (3 model artifacts)

### Alternative: Do Nothing

- **Pros:** No effort
- **Cons:** Agents continue being misled by stale context. The fundamentally wrong model will actively confuse every agent that receives it.

---

## References

**Files Examined:**
- All 28 files in `.kb/models/*.md`
- All 37 files in `.kb/guides/*.md`
- `pkg/spawn/worktree.go`, `pkg/spawn/context.go`, `cmd/orch/complete_merge.go`, `pkg/model/behavior_profile.go`, `pkg/state/db.go`, `pkg/verify/escalation.go`, `pkg/verify/visual.go`

**Commands Run:**
```bash
orch phase --help  # Confirmed orch phase exists
ls pkg/registry    # Confirmed registry removed
ls pkg/verify/phase.go  # Confirmed file doesn't exist
grep -r "registry.json\|pkg/registry\|orch phase\|worktree\|cherry-pick\|GPT-5.3" .kb/guides/
```

---

## Investigation History

**2026-02-10:** Investigation started — spawned as orch-go-fc9od codebase-audit worker
- Read all 28 model files, spot-checked claims against codebase
- Read 6 high-risk guide files in full, grepped remaining 31 for drift patterns
- Identified 7 cross-cutting drift patterns
- Categorized all 65 files
