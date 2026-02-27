# Session Synthesis

**Agent:** og-inv-investigate-spawn-context-27feb-2fa5
**Issue:** orch-go-gd6r
**Duration:** 2026-02-27 → 2026-02-27
**Outcome:** success

---

## Plain-Language Summary

When orch spawns an agent, it builds a "SPAWN_CONTEXT.md" file containing task instructions, skill guidance, and prior knowledge from the kb system. The prior knowledge is retrieved by extracting keywords from the task title and searching the knowledge base. This investigation traced the full pipeline and found three compounding problems: (1) the search query only uses the short task title, ignoring the richer issue description that contains domain-specific terms, (2) the search engine uses OR-matching where any single keyword hit counts as a match, so a word like "architect" floods results with orch-go infrastructure knowledge, and (3) local project search finding 3+ matches prevents cross-project search from ever running. Together, these caused a pricing/KPI architect spawn to receive 66KB of irrelevant orch-go knowledge instead of pricing-relevant context. The investigation also confirmed that beads issue comments (which contain orchestrator FRAME annotations and prior work context) are not included in the default spawn path.

---

## TLDR

Traced the spawn context assembly pipeline end-to-end across orch-go and kb-cli codebases. The kb context query is derived exclusively from the task title (not the orientation frame or beads comments), uses OR-based stemmed matching that floods on generic keywords, and a 3-match local threshold prevents cross-project search. Five intervention points identified for architect follow-up.

---

## Delta (What Changed)

### Files Created
- `.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md` - Full pipeline trace with diagram and 5 intervention points
- `.orch/workspace/og-inv-investigate-spawn-context-27feb-2fa5/VERIFICATION_SPEC.yaml` - Verification spec
- `.orch/workspace/og-inv-investigate-spawn-context-27feb-2fa5/SYNTHESIS.md` - This file

### Files Modified
- None (investigation only)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- `ExtractKeywords("Architect pricing KPI redesign for toolshed", 3)` → `"architect pricing kpi"` — "architect" is not a stop word and dominates matching (kbcontext.go:97-122)
- `kb context "architect pricing kpi"` returns 10+ constraints, 10+ decisions, 10+ models, 10+ guides, 10+ investigations — all orch-go infrastructure, zero pricing content
- `kb context "pricing kpi"` returns 1 match — "[global] Focus on Domain Translation, Not AI Infrastructure" — domain keywords work when they reach kb
- `MatchWithStemming` in kb-cli uses ANY-keyword-match logic (matcher.go:49-53) — single stemmed token match = included
- `MinMatchesForLocalSearch = 3` (kbcontext.go:32) — trivially met when "architect" matches across 280 investigations
- `OrientationFrame` flows to SPAWN_CONTEXT.md template (context.go:92-93) but is never passed to `ExtractKeywords` or `RunKBContextCheckForDir`
- `gatherBeadsIssueContext` (skill_requires.go:224) correctly includes last 5 comments but is gated behind `ParseSkillRequires` — not in default path
- FRAME annotation is recorded at spawn time (extraction.go:1117) but only consumed during completion (completion.go:201)

### Tests Run
```bash
go run /tmp/test_keywords.go
# Task: "Redesign toolshed pricing KPI dashboard" → Keywords(3): "redesign toolshed pricing"
# Task: "Architect pricing KPI redesign for toolshed" → Keywords(3): "architect pricing kpi"

kb context "architect pricing kpi" 2>&1 | head -5
# Context for "architect pricing kpi": ## CONSTRAINTS (from kn) ...

kb context "pricing kpi" 2>&1 | head -5
# Context for "pricing kpi": ## DECISIONS (from kb) ...
```

---

## Architectural Choices

No architectural choices — investigation only, no code changes made.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The kb context query pipeline has no mechanism to incorporate domain context from beads issue descriptions or comments
- OR-based matching is fundamentally unsuited for multi-keyword queries where one keyword is generic and others are domain-specific
- The 3-match local threshold is too low for projects with rich knowledge bases

### Key Findings
1. **Query derivation gap**: `extraction.go:1258` calls `ExtractKeywords(task, 3)` where `task` is only the issue title. The orientation frame (issue description) contains domain-specific terms that would produce dramatically better kb context matches.
2. **Matching precision gap**: kb-cli's `MatchWithStemming` is OR-based — any single keyword stem matching = inclusion. No minimum score threshold filters low-relevance results.
3. **Cross-project gap**: Local-first search with a 3-match threshold means cross-project knowledge is never searched for projects with extensive local knowledge bases.
4. **Beads context gap**: The `requires: beads-issue` skill mechanism exists and works (includes last 5 comments), but no standard skills declare it.
5. **FRAME write-only in spawn**: FRAME is written to beads comments at spawn time but only read during completion — not available to the spawned agent via default path.

### Externalized via `kb`
- (probe file serves as externalization)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix kb context query derivation for cross-domain spawns
**Skill:** architect
**Context:**
```
Investigation orch-go-gd6r traced the full kb context query pipeline and identified 5 intervention
points. The highest-impact fix is query enrichment from OrientationFrame (extraction.go:1258) —
using both task title AND orientation frame for keyword extraction. Secondary fixes: minimum score
threshold in kb-cli matching, raising MinMatchesForLocalSearch, and defaulting beads comment
inclusion. See probe at .kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md
```

---

## Unexplored Questions

- How often do cross-domain spawns happen? If most spawns are within-project, the OR-matching problem may be less severe than this case suggests.
- Would a TF-IDF or BM25 scoring model be feasible for kb context? The current scoring (title=10, filename=3, content=1/n) doesn't consider term specificity.
- Should the `requires: beads-issue` be the default for all tracked spawns rather than opt-in?

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for gate 1 (explain-back) and gate 2 (behavioral) evidence.

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-spawn-context-27feb-2fa5/`
**Probe:** `.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md`
**Beads:** `bd show orch-go-gd6r`
