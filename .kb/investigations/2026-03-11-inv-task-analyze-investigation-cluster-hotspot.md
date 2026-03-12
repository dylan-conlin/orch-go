## Summary (D.E.K.N.)

**Delta:** Investigation-cluster detection has a ~56% false-positive rate (48/85 keywords are noise) due to single-word keyword matching on filenames with an insufficient stop word list.

**Evidence:** Ran `orch hotspot --json` against 228 investigation files. Classified all 85 flagged keywords: 37 map to real architectural components/topics, 48 are generic words ("system", "code", "investigation", "md", "10"). Verified by tracing the algorithm in `hotspot_analysis.go:193-247` — it splits filenames on hyphens and counts single-token frequency.

**Knowledge:** The stop word list has 33 entries but is missing ~40+ words that should be excluded ("investigation", "code", "system", "analysis", "complete", "md", "10"). Raising threshold from 3→6 reduces hotspots from 85→23, eliminating most noise but keeping legitimate signals. However, some "noise" keywords (like "clean", "complete", "status") actually DO represent real orch-go commands/features — classification requires domain awareness.

**Next:** Architectural recommendation: expand stop word list with ~20 high-confidence generic terms, raise default threshold to 5, and consider bigram matching (e.g., "work-graph" instead of separate "work" + "graph"). Route to architect for implementation.

**Authority:** architectural - Cross-component change affecting hotspot detection, spawn gates, and completion gates

---

# Investigation: Investigation-Cluster Hotspot False-Positive Rate

**Question:** Are the 84+ investigation-cluster hotspots real architectural signals or noise from broad keyword matching?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Algorithm is single-word frequency on filenames

**Evidence:** `analyzeInvestigationClusters` (hotspot_analysis.go:193-247) reads `.kb/investigations/*.md`, strips date prefix, strips type prefix (inv-, design-, audit-, etc.), strips priority prefix (p0-p4-), splits remaining slug on hyphens, filters against a stop word list, then counts how many files share each remaining token. Any token appearing in >= threshold (default 3) files becomes a hotspot.

This is pure filename-based keyword frequency. No content analysis, no bigram/phrase matching, no TF-IDF weighting.

**Source:** `cmd/orch/hotspot_analysis.go:193-290`

**Significance:** The algorithm fundamentally cannot distinguish between "skill" as a topic cluster and "code" as a common word. Both are single tokens extracted from filenames.

---

### Finding 2: Stop word list misses 40+ generic terms

**Evidence:** The stop word list (`hotspot_analysis.go:303-333`) has 33 entries covering:
- Common English articles/prepositions/conjunctions
- Generic investigation verbs (add, fix, implement, investigate, review, etc.)
- Generic descriptors (new, old, current, existing)
- Project-specific ("orch", "go")

Missing from stop words (verified by running the algorithm and checking output):
- **Meta/process words:** investigation, investigations, analysis, code, system, plan, testing, experiment, health, theory, comprehension, detection
- **Generic nouns:** mode, project, architecture, path, layer, language, view, status, work, gap, failure, noise
- **Artifacts/numbers:** md, 10, eval, evaluate, cross, default, demo, stale, dilution, spiral, harness, probes, skills
- **Plurals of tracked words:** investigations, skills, probes

**Source:** `cmd/orch/hotspot_analysis.go:300-333`, `orch hotspot --json` output

**Significance:** The stop word gap is the primary source of false positives. Adding ~20 high-confidence generic terms would eliminate roughly half the noise keywords.

---

### Finding 3: 48/85 keywords are noise, but boundary cases are ambiguous

**Evidence:** Classification of all 85 keywords flagged at threshold >= 3:

| Category | Count | Examples | Total Score |
|----------|-------|---------|-------------|
| Real components | ~22 | skill(18), agent(14), orchestrator(14), daemon(10), session(9), dashboard(8), spawn(8), gate(6) | ~155 |
| Real topics | ~15 | probe(12), model(10), knowledge(8), extraction(6), pipeline(6), verification(5) | ~70 |
| Noise | ~48 | system(8), investigation(8), code(7), plan(6), mode(4), md(3), 10(3) | ~197 |

**However**, some "noise" classifications are debatable:
- "clean" (5) — 4/5 files relate to `orch clean` command (legitimate cluster)
- "complete" (5) — all 5 relate to `orch complete` command (legitimate cluster)
- "status" (3) — all 3 relate to `orch status` command (legitimate cluster)
- "graph" (6) — all 6 relate to "work graph" feature (legitimate cluster)
- "claude" (8) — 7/8 relate to Claude CLI integration (borderline legitimate)

True noise is more like 30-35 keywords, not 48. The boundary between "common word" and "domain concept that happens to be a common word" requires domain awareness that a stop word list can't provide.

**Source:** Cross-referencing `orch hotspot --json` output with actual investigation filenames

**Significance:** The false-positive rate is between 35-56% depending on classification criteria. Even at the conservative end, over a third of flagged keywords are noise.

---

### Finding 4: Threshold sensitivity analysis

**Evidence:** Hotspot counts at different thresholds:

| Threshold | Hotspot Count | Reduction from default |
|-----------|--------------|----------------------|
| 3 (default) | 85 | — |
| 4 | 52 | -39% |
| 5 | 34 | -60% |
| 6 | 23 | -73% |
| 7 | 14 | -84% |
| 8 | 13 | -85% |
| 10 | 6 | -93% |

At threshold=5, the remaining 34 keywords include all high-signal items (skill, agent, orchestrator, daemon, session, dashboard, model, probe, spawn) but still include noise (system, investigation, code, analysis, experiment).

At threshold=6, only 23 keywords remain. This eliminates most single-word noise while retaining meaningful clusters.

**Source:** Running `orch hotspot --inv-threshold N --json` for N in {3,4,5,6,7,8,10}

**Significance:** Raising the default threshold from 3 to 5 or 6 would be the simplest improvement. Combined with stop word expansion, threshold=5 would produce clean results.

---

### Finding 5: Downstream consumers handle noise differently

**Evidence:** Investigation-cluster hotspots feed into two downstream systems:

1. **Spawn gate** (`hotspot_spawn.go:113-120`): Matches hotspot `Path` (the keyword) via `strings.Contains` against extracted file paths. E.g., investigation-cluster keyword "daemon" matches file path "cmd/orch/daemon.go". This means noise keywords like "code" could match any file path containing "code". However, lines 159-163 note that investigation-cluster keywords are NOT matched against raw task text (only extracted paths), which limits false positives.

2. **Completion gate** (`complete_hotspot.go:30`): Similar path-matching logic.

3. **Spawn context injection**: Matched hotspot `Path` values are injected into SPAWN_CONTEXT.md as "Hotspot files" — visible in this very task's context. Keywords like "investigation", "kb", "hotspot", "md", "orch" appear in the HOTSPOT AREA WARNING of the current spawn because the task description mentions these terms as paths.

**Source:** `cmd/orch/hotspot_spawn.go:100-149`, `cmd/orch/complete_hotspot.go:30-35`

**Significance:** Noise keywords in investigation-clusters cascade into spawn gate false positives. This is the known constraint documented in CLAUDE.md: "Hotspot gate keyword matching produces false positives on non-code tasks."

---

## Synthesis

**Key Insights:**

1. **The algorithm is fundamentally a bag-of-words approach** — it extracts single tokens from filenames and counts frequency. This is the simplest possible text analysis and it shows. Single-word tokens like "system" or "code" carry almost no specificity in this project's domain.

2. **The stop word list is necessary but insufficient** — you can't enumerate all generic words. Even with a perfect stop word list, words like "clean" and "complete" are simultaneously generic English words and real orch-go command names. A stop word approach hits a ceiling where domain ambiguity takes over.

3. **Threshold tuning provides the best immediate ROI** — raising the default from 3 to 5 eliminates 60% of noise with minimal signal loss. All genuinely important architectural areas (skill, agent, orchestrator, daemon, probe, model) have scores >= 8.

**Answer to Investigation Question:**

The 85 investigation-cluster hotspots are a mix of real signal and noise. Approximately 37 keywords (44%) represent genuine architectural topics or components, while ~48 (56%) are generic words that happen to appear in multiple investigation filenames. The noise comes from two sources: (1) an incomplete stop word list missing ~40 common terms, and (2) the fundamental limitation of single-word matching on filenames. The most impactful improvements would be: expand the stop word list, raise the default threshold to 5, and consider bigram matching for compound concepts like "work-graph" or "skill-system".

---

## Structured Uncertainty

**What's tested:**

- ✅ 85 investigation-cluster hotspots at threshold=3, 228 total investigations (verified: ran `orch hotspot --json`)
- ✅ Algorithm traces to single-word frequency on filenames (verified: read `hotspot_analysis.go:193-290`)
- ✅ Stop word list has 33 entries, missing ~40+ generic terms (verified: cross-referenced output with stop word map)
- ✅ Threshold=5 produces 34 hotspots, threshold=6 produces 23 (verified: ran at each threshold)
- ✅ "clean"(5), "complete"(5), "status"(3), "graph"(6) are legitimate clusters despite appearing generic (verified: checked actual investigation filenames)

**What's untested:**

- ⚠️ Bigram matching effectiveness (not implemented, theoretical improvement)
- ⚠️ Impact of stop word expansion on spawn gate false positive rate (would need to rebuild and re-run spawn gate tests)
- ⚠️ Whether TF-IDF or other weighting would outperform simple threshold tuning (not benchmarked)

**What would change this:**

- If investigation filenames change to be more structured (e.g., always `inv-{component}-{action}`), keyword extraction could be smarter
- If investigation count grows past 500+, even threshold=6 may produce too many clusters

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Expand stop word list | implementation | Additive change, clear criteria, no cross-boundary impact |
| Raise default threshold to 5 | architectural | Affects spawn gates, completion gates, and dashboard behavior across components |
| Add bigram/phrase matching | architectural | Significant algorithm change affecting all downstream consumers |

### Recommended Approach ⭐

**Three-phase improvement: stop words → threshold → bigrams**

**Why this approach:**
- Phase 1 (stop words) is safe and immediately reduces noise by ~15-20 keywords
- Phase 2 (threshold 3→5) further reduces noise while keeping all score>=8 real signals
- Phase 3 (bigrams) is a bigger change that addresses the structural limitation

**Trade-offs accepted:**
- Threshold increase may suppress early signals for emerging investigation clusters (mitigated: emerging clusters should be monitored via `orch hotspot --inv-threshold 3` explicitly)
- Stop word expansion requires judgment calls on ambiguous words

**Implementation sequence:**
1. Add ~20 high-confidence generic terms to stop word list: `investigation`, `investigations`, `code`, `system`, `analysis`, `plan`, `testing`, `mode`, `project`, `architecture`, `path`, `view`, `md`, `10`, `eval`, `evaluate`, `cross`, `default`, `theory`, `language`
2. Raise `--inv-threshold` default from 3 to 5
3. (Future) Implement optional bigram matching for compound concepts

### Alternative Approaches Considered

**Option B: Content-based analysis instead of filename-based**
- **Pros:** Far higher precision — could extract actual topics from investigation content
- **Cons:** Much more complex, slower (reads 228+ files), investigation content structure varies widely
- **When to use instead:** If investigation count exceeds 1000+ and filename-based approach is fundamentally inadequate

**Option C: File-path matching only (drop keyword matching)**
- **Pros:** Eliminates all false positives from generic keywords
- **Cons:** Loses valuable signal — investigation clustering IS useful for identifying areas needing design synthesis
- **When to use instead:** Never — keyword clustering is the right idea, just needs better execution

---

### Implementation Details

**What to implement first:**
- Expand stop word list (immediate, safe, no downstream impact)
- Update tests to reflect new stop word additions

**Things to watch out for:**
- ⚠️ Words like "clean", "complete", "status" should NOT be added to stop words — they correspond to real orch-go commands despite appearing generic
- ⚠️ "claude" is borderline — 7/8 matches are about Claude CLI integration, which IS a real topic
- ⚠️ Plurals ("skills", "probes", "investigations") should be stopped even when singulars are kept as valid keywords

**Success criteria:**
- ✅ Investigation-cluster hotspot count at threshold=3 drops from 85 to <60
- ✅ At threshold=5 (new default), count drops from 34 to <25
- ✅ All top-10 real architectural signals (skill, agent, orchestrator, daemon, probe, model, session, dashboard, spawn, knowledge) retained
- ✅ No spawn gate regressions (existing tests pass)

---

## References

**Files Examined:**
- `cmd/orch/hotspot.go` — Main hotspot command, report structure, output formatting
- `cmd/orch/hotspot_analysis.go:193-333` — `analyzeInvestigationClusters`, `extractInvestigationKeywords`, `isInvestigationStopWord`, stop word list
- `cmd/orch/hotspot_spawn.go:100-193` — `matchPathToHotspots`, `checkSpawnHotspots` — downstream consumer of investigation-clusters
- `cmd/orch/complete_hotspot.go:30` — Completion gate consumer

**Commands Run:**
```bash
# Count total investigations
ls .kb/investigations/*.md | wc -l  # → 228

# Run hotspot analysis
go run ./cmd/orch hotspot --json

# Threshold sensitivity
for thresh in 3 4 5 6 7 8 10; do go run ./cmd/orch hotspot --inv-threshold $thresh --json; done

# Verify keyword matches against actual filenames
ls .kb/investigations/*.md | xargs -I{} basename {} | grep -E '\b<keyword>\b'
```
