# Session Synthesis

**Agent:** og-arch-design-accretion-gravity-14feb-e5c9
**Issue:** orch-go-4mu
**Duration:** 2026-02-14
**Outcome:** success

---

## TLDR

Designed four-layer accretion gravity enforcement architecture — spawn gates (prevention), completion gates (rejection), coaching plugin (real-time correction), CLAUDE.md boundaries (declaration) — with 5 navigated decision forks, concrete file targets, and phased implementation plan. All layers extend existing infrastructure (hotspot analysis, completion verification, coaching plugin) rather than creating new systems.

---

## Delta (What Changed)

### Files Modified
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` — Completed investigation: added fork navigation (5 forks with substrate traces), concrete implementation details (file targets, acceptance criteria, success metrics), prior work references, investigation history

### Commits
- Investigation complete with four-layer enforcement architecture design

---

## Evidence (What Was Observed)

- Spawn hotspot check at `spawn_cmd.go:834-850` prints warning but always proceeds — confirmed warning-only behavior by reading code
- `RunHotspotCheckForSpawn()` returns `SpawnHotspotResult{HasHotspots, MatchedHotspots, MaxScore, Warning}` — all data needed for enforcement is already computed
- 12 completion gates exist in `pkg/verify/check.go` with consistent pattern: gate function returns result type, merged into VerificationResult — accretion gate follows same pattern
- Coaching plugin `tool.execute.after` hook at `coaching.ts:1543-1829` already implements tiered detection (frame collapse pattern) — accretion detection is a natural extension
- `git_diff.go` already has `GetGitDiffFiles()` returning changed file list and `projectDir` parameter — extension point exists for line count checking
- Hotspot thresholds calibrated: 800 lines moderate, 1,500 lines CRITICAL (hotspot.go:477-486) — reuse these, don't invent new ones
- Friction gate probe (2026-02-13) found skill-class blindness causes 31.7% of completion bypasses — accretion gate must be skill-class-aware to avoid same noise
- Friction gate probe found build gate has 0.7:1 bypass:fail ratio (most valuable gate) — accretion gate should target similar ratio by being selective about when it fires

### Tests Run
```bash
# No code changes — architecture design only
# Verification was code reading and substrate consultation
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` — Complete four-layer enforcement architecture with 5 decision forks navigated

### Decisions Made (recommendations, pending orchestrator acceptance)
- Fork 1: Tiered thresholds (warn at 800, error at 1,500) because hard error at 800 would create noise similar to test_evidence gate (5.5:1 bypass ratio)
- Fork 2: Exempt knowledge-producing skills (architect, investigation, capture-knowledge, audit) because Gate Over Remind caveat says "gates must be passable by the gated party"
- Fork 3: Net-negative delta passes completion gate because extraction IS the structural fix for accretion
- Fork 4: Escalating coaching warnings (matching existing frame collapse pattern) because plugin can't block but Pain as Signal says repeated friction changes behavior
- Fork 5: Use existing 800/1,500/+50 thresholds because they're already calibrated by hotspot analysis

### Constraints Discovered
- Coaching plugin cannot block tool execution — can only inject messages between turns via `noReply: true`
- Spawn gate must exempt extraction tasks (keyword: "extract", "decompose", "refactor") or the gate blocks its own fix
- Completion gate needs net-delta awareness — blocking extraction work defeats the purpose of accretion enforcement

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up (4 implementation tasks)

### Implementation Sequence

**Phase 1 — Implement first (highest ROI + zero cost):**

**Task 1: CLAUDE.md Accretion Boundaries**
- **Skill:** feature-impl
- **Context:** Add section to CLAUDE.md documenting CRITICAL hotspot files and the rule "Files >1,500 lines require extraction before feature addition." Link to `orch hotspot` and `.kb/guides/code-extraction-patterns.md`. Zero implementation cost, immediate effect.

**Task 2: Spawn-Time Gate**
- **Skill:** feature-impl
- **Context:** Modify `spawn_cmd.go:834-853` to BLOCK (not warn) when feature-impl/systematic-debugging skills target CRITICAL hotspots (>1,500 lines). Exempt: architect, investigation, capture-knowledge, audit skills. Add `--force-hotspot` override flag. Emit `spawn.accretion.blocked` event.

**Phase 2 — Catch escapes:**

**Task 3: Completion Accretion Gate**
- **Skill:** feature-impl
- **Context:** Add `GateAccretion` to `pkg/verify/check.go`. Create `pkg/verify/accretion.go` with `VerifyAccretionForCompletion()`. Check git diff for +50 lines to files >800 lines (warn) or >1,500 lines (error). Net-negative delta passes. Insert after git_diff gate in VerifyCompletionFull().

**Phase 3 — Real-time correction:**

**Task 4: Coaching Plugin Accretion Detection**
- **Skill:** feature-impl
- **Context:** Extend `coaching.ts` tool.execute.after hook. When worker edits file >800 lines, inject accretion warning. Escalating urgency: 1st edit warns, 3+ edits strong warning. Match existing frame collapse pattern. Add `accretion_warning` patternType to `injectCoachingMessage()`. Write metrics to coaching-metrics.jsonl.

---

## Unexplored Questions

**Questions that emerged during this session:**
- Cross-file accretion: agent creates 5 small files instead of one coherent package. Total complexity increases but no single file triggers thresholds. Out of scope — address if pattern emerges.
- Dynamic threshold loading: could `orch hotspot --json` feed thresholds to gates at runtime instead of hardcoding? Enhancement for later.
- Dashboard accretion panel: coaching metrics already flow to dashboard — adding an accretion view is straightforward but out of scope.

**What remains unclear:**
- Whether 800-line warning threshold will produce acceptable false positive rate. Plan: monitor for 2 weeks, adjust if >50% are false positives.
- Whether coaching plugin warnings actually change agent behavior for accretion (they work for frame collapse, but accretion is a different behavioral pattern).

---

## Verification Contract

**Verification Specification:** Not applicable (design-only — no code changes to verify)
**Key Outcomes:**
- Investigation file complete with Phase: Complete
- 5 decision forks navigated with substrate traces
- Concrete implementation plan with file targets and acceptance criteria
- 4 implementation tasks defined with phased sequencing

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-accretion-gravity-14feb-e5c9/`
**Investigation:** `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md`
**Beads:** `bd show orch-go-4mu`
