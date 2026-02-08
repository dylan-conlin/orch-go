## Summary (D.E.K.N.)

**Delta:** Pre-spawn `kb context` noise has two root causes: (1) `--global` flag searches ALL 17+ projects when most work is orch-ecosystem-specific, (2) keyword extraction generates single-word queries like "pre" that match 1,200+ entries across unrelated repos.

**Evidence:** "pre" query returned 1,210 entries, with 33% (400) from irrelevant repos (price-watch, dotfiles, scs-slack). Current project-only search returns 15 targeted kn entries but floods with 100+ investigations.

**Knowledge:** The noise problem has a layered solution: (1) Add `--project` flag to `kb context` (kb-cli enhancement), (2) Use tiered search: current project first + orch ecosystem repos, (3) Apply per-category limits to prevent investigation flood.

**Next:** Implement tiered filtering in orch-go: current project + explicit orch ecosystem allowlist + `--limit 20` per category.

**Confidence:** High (85%) - tested multiple filtering strategies with concrete measurements.

---

# Investigation: Pre-Spawn KB Context Noise Filtering

**Question:** Why does pre-spawn `kb context` check surface too much irrelevant cross-repo content (price-watch, dotfiles, etc), and what filtering strategies would preserve cross-repo signal while reducing noise?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None - ready for implementation decision
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Query generates 1,210 matches across 17 repos

**Evidence:**
```
$ kb context "pre" --global 2>&1 | grep -E '^\- \[' | wc -l
1210

$ kb context "pre" --global 2>&1 | grep -E '^\- \[' | sed 's/^\- \[\([^]]*\)\].*/\1/' | sort | uniq -c | sort -rn | head -10
 475 orch-knowledge
 306 price-watch
 174 orch-cli
 128 orch-go
  45 beads-ui-svelte
  20 agentlog
  15 kb-cli
   9 dotfiles
   9 .doom.d
   7 skillc
```

**Source:** kb context command output, manual testing

**Significance:** The generic keyword "pre" matches content in work-related repos (price-watch) and system configs (dotfiles) that have nothing to do with orchestration tasks.

---

### Finding 2: 33% of results are from irrelevant repos

**Evidence:**
```
# Relevant (orch ecosystem): 792 entries
$ kb context "pre" --global 2>&1 | grep -E '^\- \[(orch-go|orch-cli|kb-cli|orch-knowledge)\]' | wc -l
792

# Irrelevant (other projects): 400 entries  
$ kb context "pre" --global 2>&1 | grep -E '^\- \[(price-watch|dotfiles|.doom.d|beads-ui-svelte|agentlog|scs-slack|blog|snap)\]' | wc -l
400
```

**Source:** Filtered grep analysis of kb context output

**Significance:** About 1/3 of the context is noise for orch-ecosystem work. This consumes tokens and cognitive load without providing value.

---

### Finding 3: `kb search` has `--project` flag but `kb context` doesn't

**Evidence:**
```
$ kb search --help | grep -i project
  -p, --project string   Filter to specific project (use with -g)

$ kb context --help | grep -i project
  -g, --global          Search across all known projects
```

**Source:** CLI help output from kb-cli

**Significance:** The filtering capability exists in `kb search` but was not ported to `kb context`. This is the simplest fix point for kb-cli.

---

### Finding 4: Current project search is highly targeted but misses cross-repo knowledge

**Evidence:**
```
$ cd /Users/dylanconlin/Documents/personal/orch-go && kb context "pre" 2>&1 | grep -E '^\- \[' | wc -l
0  # (no project prefix because local search)

# Actual result: 4 constraints, 5 decisions, 1 kb decision, 1 attempt, ~100 investigations
```

**Source:** Local project kb context test

**Significance:** Current project-only search is too narrow - it misses critical knowledge from orch-cli, kb-cli, and orch-knowledge that applies to orchestration work.

---

### Finding 5: More specific queries dramatically reduce noise

**Evidence:**
```
$ kb context "spawn kb context" --global 2>&1 | grep -E '^\- \[' | wc -l
2  # Only 2 highly relevant results (both orch-go investigations)

vs.

$ kb context "pre" --global 2>&1 | grep -E '^\- \[' | wc -l
1210  # 1,210 results with single-word query
```

**Source:** Query comparison test

**Significance:** The keyword extraction in orch-go (ExtractKeywords) creates too-generic queries. Better keyword extraction or multi-word queries would help.

---

### Finding 6: --limit flag helps but still includes noise

**Evidence:**
```
$ kb context "pre" --global --limit 20 2>&1 | grep -E '^\- \[' | wc -l
65  # Much better than 1,210

# But still includes noise:
$ kb context "pre" --global --limit 20 2>&1 | grep -E '^\- \[' | sed 's/^\- \[\([^]]*\)\].*/\1/' | sort | uniq -c
  22 orch-knowledge
  15 kb-cli
   8 orch-go
   4 price-watch    # noise
   4 orch-cli
   4 dotfiles       # noise
   1 skillc
   1 scs-slack      # noise
   1 kn
   1 beads-ui-svelte  # borderline
```

**Source:** Limit flag testing

**Significance:** `--limit` reduces volume but doesn't filter by project relevance. Need both: limit AND project filtering.

---

## Synthesis

**Key Insights:**

1. **Two-factor noise problem** - The noise comes from both (a) generic keyword extraction and (b) searching all known projects indiscriminately.

2. **Orch ecosystem should be explicit** - Work in orch-go, orch-cli, kb-cli typically needs knowledge from all three repos plus orch-knowledge. This is a stable, identifiable set that should be searchable as a group.

3. **Layered filtering is the right model** - Start narrow (current project), then widen (related repos), then global (only if needed). The current binary (local vs global) is too coarse.

**Answer to Investigation Question:**

The noise comes from two root causes that require separate solutions:

1. **kb-cli enhancement needed:** Add `--project` flag to `kb context` (already exists in `kb search`). This enables filtering by project name when using `--global`.

2. **orch-go implementation change:** Instead of always using `--global`, implement tiered search:
   - First: Current project (no flag)
   - Then: Add related repos via multiple `--project` filters OR post-process results
   - Apply `--limit 20` per category to prevent flood

The recommended approach preserves cross-repo signal (orch ecosystem is still searched) while eliminating noise from unrelated projects (price-watch, dotfiles, etc.).

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Evidence is based on concrete measurements from real `kb context` output. The root causes are clearly identified and solutions are technically straightforward (the `--project` flag already exists in `kb search`).

**What's certain:**

- ✅ Noise ratio is ~33% from irrelevant repos
- ✅ `--limit` reduces volume but not noise source
- ✅ `--project` flag exists in kb search, not kb context
- ✅ Current project-only search works but is too narrow

**What's uncertain:**

- ⚠️ What the "orch ecosystem" allowlist should contain (orch-go, orch-cli, kb-cli, orch-knowledge, beads, kn?)
- ⚠️ Whether kb-cli team will accept the `--project` flag addition
- ⚠️ Whether tiered search adds meaningful latency (running multiple searches)

**What would increase confidence to Very High:**

- Test implementation of tiered filtering in orch-go
- Get confirmation from kb-cli that --project flag is acceptable
- Measure latency impact of multiple searches

---

## Implementation Recommendations

### Recommended Approach ⭐

**Tiered Filtering in orch-go (immediate) + kb-cli --project flag (follow-up)**

**Why this approach:**
- Immediate value: orch-go can implement filtering now using post-processing
- Layered solution: Fix both the query quality AND project scope issues
- Preserves signal: Still finds cross-repo knowledge within orch ecosystem

**Trade-offs accepted:**
- Post-processing in orch-go is less efficient than server-side filtering
- Hardcoding "orch ecosystem" repos requires maintenance
- Why acceptable: orch ecosystem is stable, performance is acceptable for small result sets

**Implementation sequence:**
1. **Phase 1 (orch-go):** Modify `runPreSpawnKBCheck()` to post-filter results to orch ecosystem repos
2. **Phase 2 (orch-go):** Apply per-category limits (20 per type) 
3. **Phase 3 (kb-cli):** Add `--project` flag to `kb context` (follow-up issue)
4. **Phase 4 (orch-go):** Refactor to use `--project` flags instead of post-processing

### Alternative Approaches Considered

**Option B: Current project only (no --global)**
- **Pros:** Zero noise from other projects
- **Cons:** Misses critical cross-repo knowledge (orch-cli decisions, kb-cli constraints)
- **When to use instead:** For tasks that are genuinely project-scoped

**Option C: Better keyword extraction**
- **Pros:** More specific queries reduce noise at source
- **Cons:** Risk of missing relevant content with too-specific queries; diminishing returns
- **When to use instead:** As additional improvement after filtering is in place

**Rationale for recommendation:** The orch ecosystem is a stable, known set of repos. Filtering to this set eliminates 33% noise immediately while preserving the cross-repo knowledge that makes `kb context` valuable.

---

### Implementation Details

**What to implement first:**
- Modify orch-go `kbcontext.go` to filter results by project name
- Add configurable allowlist: `["orch-go", "orch-cli", "kb-cli", "orch-knowledge", "beads", "kn"]`
- Apply `--limit 20` per category

**Things to watch out for:**
- ⚠️ Hardcoded allowlist needs maintenance when new repos are added
- ⚠️ JSON format from `kb context --format json` is easier to filter than text
- ⚠️ Need to handle case where current project isn't in allowlist

**Areas needing further investigation:**
- Should beads-ui-svelte be in the allowlist? (it has useful UI patterns)
- Should the allowlist be configurable via `.orch/config.yaml`?
- Should there be a `--no-filter` escape hatch?

**Success criteria:**
- ✅ Pre-spawn context check returns <100 entries (down from 1,200+)
- ✅ All returned entries are from orch ecosystem repos
- ✅ Critical cross-repo knowledge (constraints, decisions) still appears
- ✅ No price-watch, dotfiles, scs-slack entries in spawn context

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/kbcontext.go` - Current kb context integration
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go` - kb context implementation
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/search.go` - kb search with --project flag

**Commands Run:**
```bash
# Measure noise
kb context "pre" --global 2>&1 | grep -E '^\- \[' | wc -l
kb context "pre" --global 2>&1 | grep -E '^\- \[' | sed 's/^\- \[\([^]]*\)\].*/\1/' | sort | uniq -c | sort -rn

# Test filtering strategies
kb context "pre" --global --limit 20 2>&1 | grep -E '^\- \[' | wc -l
kb context "spawn kb context" --global 2>&1 | grep -E '^\- \[' | wc -l
cd /Users/dylanconlin/Documents/personal/orch-go && kb context "pre" 2>&1 | wc -l
```

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-fix-pre-spawn-kb-context.md` - Prior investigation on this issue

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn decide "Pre-spawn kb context should filter to orch ecosystem repos" --reason "33% of global results are noise from unrelated repos (price-watch, dotfiles). Filtering preserves cross-repo signal while eliminating noise."
```

---

## Investigation History

**2025-12-22 08:30:** Investigation started
- Initial question: Why does pre-spawn kb context surface irrelevant content?
- Context: SPAWN_CONTEXT.md had 2,400+ lines of prior knowledge

**2025-12-22 09:00:** Root cause identified
- Found two factors: generic queries + global search
- Measured noise ratio at 33%

**2025-12-22 09:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend tiered filtering with orch ecosystem allowlist
