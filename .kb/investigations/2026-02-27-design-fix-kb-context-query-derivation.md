## Summary (D.E.K.N.)

**Delta:** OR-based matching in kb-cli scores a title-only single-keyword match (e.g., "dashboard") at 10+ points — indistinguishable from a 5/5 keyword match — because scoring has no keyword coverage dimension. Prior patches (MinStemmedScore, stop words, frame enrichment) are Coherence Over Patches signals; the missing coherent model is that relevance requires keyword coverage, not just keyword presence.

**Evidence:** Traced full pipeline across orch-go (kbcontext.go, extraction.go) and kb-cli (matcher.go, search.go, context.go). Score for 1/5 keywords matching in title = 10.0; score for 5/5 keywords matching in title = 10.0. MinStemmedScore only filters stemmed-only matches (score < 2.0), but exact substring hits bypass this entirely.

**Knowledge:** Four patches have been applied to the same search quality problem (stop words, frame enrichment, MinStemmedScore, raised local threshold). Each is locally correct but none addresses the root cause: scoring ignores how many query keywords matched. A keyword coverage multiplier on the score resolves all four symptoms at once.

**Next:** Implement keyword coverage multiplier in kb-cli `searchFileContentsWithScore` (cross-repo issue). No further orch-go changes needed — existing `ExtractKeywordsWithContext` and `MinMatchesForLocalSearch=5` are sufficient once kb-cli scoring is fixed.

**Authority:** architectural — Cross-repo change affecting how all `kb context` consumers receive results. Changes the scoring contract between orch-go and kb-cli.

---

# Investigation: Design Fix for KB Context Query Derivation

**Question:** Why does kb context produce 66KB of irrelevant infrastructure knowledge for cross-domain spawns, and what is the minimal coherent fix?

**Defect-Class:** integration-mismatch

**Started:** 2026-02-27
**Updated:** 2026-02-27
**Owner:** og-arch-fix-kb-context-27feb-70f7
**Phase:** Complete
**Next Step:** None — implement keyword coverage multiplier in kb-cli
**Status:** Complete

**Patches-Decision:** N/A (no existing decision to patch; this produces a new recommendation)
**Extracted-From:** `.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Probe: KB Context Query Derivation (2026-02-27) | extends | Yes - code paths confirmed against current source | None |
| orch-go-amty (MinStemmedScore implementation) | extends | Yes - MinStemmedScore=2.0 confirmed in search.go:326 | Reveals insufficiency of stemmed-only filtering |

---

## Findings

### Finding 1: Four patches applied to same search quality area (Coherence Over Patches signal)

**Evidence:** The kb context search quality problem has received four incremental patches:

1. **Stop word expansion** (kbcontext.go:114-133): Added skill names ("architect", "investigation"), action verbs ("redesign", "refactor"), and common prefixes to stop words. Still OR-matches on remaining keywords.

2. **Frame enrichment** (kbcontext.go:155-199, extraction.go:1280-1283): `ExtractKeywordsWithContext()` combines title + orientation frame keywords. Produces richer queries (5 keywords vs 3). But more keywords with OR-matching means more irrelevant matches, not fewer.

3. **MinStemmedScore threshold** (kb-cli search.go:288-296, 326): Filters stemmed-only results scoring below 2.0. Only applies when ALL matches are stemmed — a single exact substring hit on any keyword bypasses this filter entirely.

4. **Raised local threshold** (kbcontext.go:35): `MinMatchesForLocalSearch` raised from 3 to 5. Delays global expansion but doesn't improve local result quality.

**Source:** `pkg/spawn/kbcontext.go:30-199`, `pkg/orch/extraction.go:1272-1315`, `kb-cli/cmd/kb/search.go:288-326`, `kb-cli/internal/search/matcher.go:59-106`

**Significance:** Per Coherence Over Patches principle, 4+ fixes in the same area signals a missing coherent model. Each patch is locally correct but addresses symptoms, not the root cause. The root cause is that scoring has no keyword coverage dimension.

---

### Finding 2: Score is blind to keyword coverage

**Evidence:** Scoring in `searchFileContentsWithScore` (kb-cli search.go:345-421):
- Title match: +10.0 (regardless of how many query keywords matched the title)
- Filename match: +3.0
- Content line match: +1/(n+1) per line (diminishing returns)

For a 5-keyword query like `"pricing kpi toolshed dashboard metrics"`:
- File A: title contains "dashboard" only → score 10.0 + content matches → ~11.5
- File B: title contains "pricing KPI dashboard metrics" → score 10.0 + content matches → ~11.5

Both files score identically despite File B matching 4/5 keywords and File A matching 1/5. The scoring model treats these as equally relevant.

The `MatchWithStemming` function (matcher.go:64-106) checks if ANY query keyword matches, returning a boolean. There's no per-keyword tracking. The `MatchWithStemmingResult` variant returns match type (exact/stemmed) but still uses OR semantics.

**Source:** `kb-cli/cmd/kb/search.go:345-421`, `kb-cli/internal/search/matcher.go:64-106`

**Significance:** This is the root cause. Without keyword coverage in scoring, all upstream improvements (richer queries, stop words, stemmed filtering) are partially neutralized. A single common word matching anywhere produces high scores that crowd out genuinely relevant results.

---

### Finding 3: projectDir already resolves correctly for cross-repo spawns

**Evidence:** `ResolveProjectDirectory` (extraction.go:549-573) sets `projectDir` to `--workdir` target when specified. `runKBContextQuery` (kbcontext.go:305-350) sets `cmd.Dir = projectDir`, so local `kb context` search runs in the target project. Global fallback uses `resolveProjectAllowlistForDir(projectDir)` for group-based filtering.

For cross-domain spawns like `orch spawn --workdir ~/toolshed architect "pricing KPI redesign"`, the local search correctly runs in `~/toolshed/.kb/` first, then falls back to global with the toolshed project group.

**Source:** `pkg/orch/extraction.go:549-573`, `pkg/spawn/kbcontext.go:253-298`, `pkg/spawn/kbcontext.go:305-350`

**Significance:** Cross-project query routing is NOT broken — `projectDir` correctly directs search to the target project. The problem is that when spawning FROM orch-go (without `--workdir`), the local search hits orch-go's .kb/ first, and OR-matching on common words floods results before global search triggers. Frame enrichment helps but doesn't solve the scoring flaw.

---

## Synthesis

**Key Insights:**

1. **Missing coherent model: keyword coverage** — The scoring system measures WHERE keywords match (title, filename, body) but not HOW MANY keywords match. This single gap makes all four prior patches insufficient — they reduce noise at the edges but can't distinguish "1 of 5 keywords matched in title" (irrelevant) from "5 of 5 keywords matched in title" (highly relevant).

2. **Frame enrichment is necessary but compounds the OR problem** — `ExtractKeywordsWithContext` correctly enriches queries with domain terms. But with OR-matching, 5 keywords means 5 chances for a false positive, not 5 requirements for a true positive. The fix must work WITH frame enrichment, not against it.

3. **The fix belongs in kb-cli, not orch-go** — The scoring model is in `searchFileContentsWithScore` (kb-cli). Orch-go's job is to produce good queries (already done via frame enrichment) and route results (already correct via projectDir). Adding scoring logic to orch-go would violate separation of concerns and duplicate what kb-cli should own.

**Answer to Investigation Question:**

KB context produces 66KB of irrelevant results because the scoring model in kb-cli's `searchFileContentsWithScore` has no keyword coverage dimension. A file matching 1 of 5 query keywords in its title scores 10.0+ — the same as a file matching all 5. Four prior patches (stop words, frame enrichment, MinStemmedScore, raised threshold) address symptoms but not the root cause. The minimal coherent fix is a keyword coverage multiplier on the score in kb-cli, which resolves all symptoms at once and works synergistically with the existing orch-go improvements.

---

## Structured Uncertainty

**What's tested:**

- ✅ OR-matching confirmed: `MatchWithStemming` returns true on ANY single keyword hit (verified: read matcher.go:99-101)
- ✅ Score is coverage-blind: title match = +10 regardless of keyword count (verified: read search.go:376-381)
- ✅ MinStemmedScore only filters stemmed-only matches (verified: search.go:288-296, StemmedOnly requires ALL matches to be stemmed)
- ✅ projectDir correctly resolves for --workdir spawns (verified: extraction.go:549-573, kbcontext.go:318-320)
- ✅ Frame enrichment is wired in (verified: extraction.go:1280-1283 calling ExtractKeywordsWithContext)

**What's untested:**

- ⚠️ Exact impact of keyword coverage multiplier on real queries (not benchmarked — need to test with actual kb context queries)
- ⚠️ Whether coverage multiplier would break existing non-cross-domain queries (single-keyword queries unaffected since coverage = 1.0, but 2-keyword queries might shift)
- ⚠️ Performance impact of per-keyword matching in searchFileContentsWithScore (should be negligible — tokenize + set lookup)

**What would change this:**

- If MinStemmedScore alone is sufficient for real-world cross-domain queries (unlikely given Finding 2, but testable)
- If single-keyword queries are the dominant use case (coverage multiplier has no effect on them, so no harm either)
- If kb-cli's searchFileContentsWithScore is already being refactored (avoid conflicting changes)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add keyword coverage multiplier to kb-cli scoring | architectural | Cross-repo change affecting all kb context consumers; changes scoring contract |
| Keep orch-go changes as-is (no further modifications) | implementation | Existing frame enrichment and threshold are sufficient once scoring is fixed |

### Recommended Approach ⭐

**Keyword Coverage Multiplier in kb-cli** — Add a keyword coverage factor to `searchFileContentsWithScore` that scales the score by the proportion of query keywords that matched anywhere in the file.

**Why this approach:**
- Resolves all four symptom patches at their root cause (one fix vs four patches)
- Backward-compatible: single-keyword queries get coverage = 1.0 (no change)
- Works synergistically with frame enrichment: more keywords = more discrimination power (not more noise)
- Simple implementation: count matched keywords, divide by total, multiply score

**Trade-offs accepted:**
- 2-keyword queries now penalize single-keyword matches (coverage = 0.5 → score halved). Acceptable because single-keyword matches on 2-keyword queries are genuinely less relevant.
- Requires cross-repo change in kb-cli. But the fix belongs there (scoring is kb-cli's responsibility).

**Implementation sequence:**

1. **Add keyword coverage counting to `searchFileContentsWithScore`** (kb-cli search.go)
   - After computing raw score, tokenize query and count distinct keywords that appeared in any matched line
   - Coverage = matchedKeywords / totalKeywords
   - Final score = rawScore * max(coverage, 0.3) (floor of 0.3 prevents zeroing out)

2. **Add `CountMatchedKeywords` helper to matcher.go** (kb-cli)
   - Takes content tokens and query tokens, returns count of distinct query tokens present
   - Uses stemmed matching for consistency with existing `MatchWithStemming`

3. **Add tests for keyword coverage scoring** (kb-cli)
   - Test: 1/5 keywords matching → score reduced by ~80%
   - Test: 5/5 keywords matching → full score
   - Test: 1/1 keyword matching → full score (backward compat)

### Alternative Approaches Considered

**Option B: Change MatchWithStemming to AND-matching (require all keywords)**
- **Pros:** Strict relevance — only files matching all keywords included
- **Cons:** Too strict for natural language queries; "pricing kpi toolshed" might not match a file about "toolshed pricing strategy" that uses "cost" instead of "kpi"
- **When to use instead:** Never — AND-matching breaks legitimate partial matches

**Option C: Add minimum coverage threshold (filter out < 40% coverage)**
- **Pros:** Explicit cutoff, easy to reason about
- **Cons:** Abrupt boundary (39% filtered, 41% included); harder to tune; loses the nuance of "2/5 keywords is somewhat relevant but less so than 5/5"
- **When to use instead:** If the multiplier approach produces confusing score distributions in practice

**Option D: Per-keyword scoring (track each keyword's contribution separately)**
- **Pros:** Most accurate relevance model
- **Cons:** Requires significant refactoring of `MatchWithStemming` and all callers; overkill for current needs
- **When to use instead:** If keyword coverage multiplier proves insufficient after testing

**Rationale for recommendation:** Option A (coverage multiplier) is the minimal coherent fix. It preserves backward compatibility, works with existing infrastructure, and directly addresses the root cause identified in Finding 2. Per Coherence Over Patches principle, one structural fix is better than another incremental patch.

---

### Implementation Details

**What to implement first:**
- Keyword coverage multiplier in `searchFileContentsWithScore` (kb-cli search.go:345-421)
- This is the single change with highest impact; all other pieces are already in place

**File targets (kb-cli):**
- `internal/search/matcher.go` — Add `CountMatchedKeywords(contentTokens map[string]bool, queryTokens []string) int`
- `cmd/kb/search.go:345-421` — Add coverage calculation after raw score, apply multiplier
- `cmd/kb/search_test.go` — Add coverage scoring tests

**Acceptance criteria:**
- ✅ Query "pricing kpi toolshed dashboard metrics" against file with only "dashboard" in title: score < 3.0 (was 10+)
- ✅ Same query against file with all 5 keywords: score ~10.0 (unchanged)
- ✅ Single-keyword query "dashboard": score unchanged (coverage = 1.0)
- ✅ Existing kb-cli tests pass without modification

**Things to watch out for:**
- ⚠️ The `searchFileContentsWithScore` function is called by `searchGuidesDir`, `searchModelsDir`, and `getGlobalStoreContext` — all will benefit from the fix
- ⚠️ Quick entries (`.kb/quick/entries.jsonl`) use `search.MatchWithStemming` directly in `searchKnEntries` — these don't go through `searchFileContentsWithScore`, so coverage multiplier won't apply. Quick entries are short enough that OR-matching is reasonable.
- ⚠️ The coverage floor (0.3) prevents zeroing out results that match legitimately on 1 keyword but happen to have a multi-keyword query. Tune if needed.

**Areas needing further investigation:**
- Optimal coverage floor value (0.3 proposed, may need empirical tuning)
- Whether quick entry matching should also get coverage scoring (lower priority — quick entries are brief)

**Success criteria:**
- ✅ Cross-domain spawns (e.g., toolshed work from orch-go) produce domain-relevant kb context, not infrastructure flood
- ✅ Same-domain spawns (e.g., orch-go work from orch-go) maintain or improve result quality
- ✅ kb context response time stays under 5 seconds (no performance regression)

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` — KB context query pipeline, ExtractKeywords, ExtractKeywordsWithContext, RunKBContextCheckForDir
- `pkg/orch/extraction.go` — GatherSpawnContext, runPreSpawnKBCheckFull, ResolveProjectDirectory
- `cmd/orch/spawn_cmd.go` — Spawn command flow, projectDir resolution, orientation frame extraction
- `kb-cli/internal/search/matcher.go` — MatchWithStemming, MatchWithStemmingResult, tokenize, stemWord
- `kb-cli/cmd/kb/search.go` — SearchArtifacts, searchFileContentsWithScore, MinStemmedScore
- `kb-cli/cmd/kb/context.go` — GetContext, GetContextGlobal, context command, searchKnEntries
- `.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md` — Probe findings

**Related Artifacts:**
- **Probe:** `.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md` — Full pipeline trace
- **Issue:** `orch-go-amty` — MinStemmedScore implementation (in progress)
- **Workspace:** `.orch/workspace/og-feat-architect-fix-kb-27feb-fbde/` — Prior agent's frame enrichment work
- **Principle:** Coherence Over Patches (`~/.kb/principles.md`) — 4+ patches in same area = missing model

---

## Investigation History

**2026-02-27 21:30:** Investigation started
- Initial question: Why does kb context produce irrelevant results for cross-domain spawns?
- Context: Probe orch-go-gd6r identified 3 compounding problems and 5 intervention points

**2026-02-27 21:45:** Traced code and found prior agent work
- Prior agent (og-feat-architect-fix-kb-27feb-fbde) already implemented frame enrichment + raised threshold
- Agent orch-go-amty already implementing MinStemmedScore in kb-cli
- Identified that 4 patches = Coherence Over Patches signal

**2026-02-27 22:00:** Root cause confirmed
- Score is blind to keyword coverage: 1/5 match = 5/5 match in scoring
- MinStemmedScore only filters stemmed-only matches; exact substring matches bypass it
- Keyword coverage multiplier identified as the minimal coherent fix

**2026-02-27 22:15:** Investigation completed
- Status: Complete
- Key outcome: Single change in kb-cli (keyword coverage multiplier) resolves root cause; no further orch-go changes needed
