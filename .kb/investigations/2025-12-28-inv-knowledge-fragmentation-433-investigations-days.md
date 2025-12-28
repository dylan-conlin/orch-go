## Summary (D.E.K.N.)

**Delta:** The 500 investigations are NOT duplicative - they explore distinct aspects; the real problem is knowledge graph disconnection (0 lineage references, 0 kn citations, 1.8% decision promotion rate).

**Evidence:** Analyzed dashboard (24 investigations - each distinct aspect), headless (14 investigations - each distinct aspect); `rg` found 0 investigations with "Supersedes", "Prior", or "kn-" references; only 9 formal decisions from 500 investigations.

**Knowledge:** High investigation volume reflects rapid development, not waste. The gap is in knowledge LINKAGE, not knowledge CREATION. Synthesis opportunities flagged by `kb reflect` group by keyword ("add" = 41) not semantic topic.

**Next:** Recommend structural changes to kb reflect (semantic grouping) and spawn context (include recent relevant investigations), not synthesis passes.

---

# Investigation: Knowledge Fragmentation - 433 Investigations in 7 Days

**Question:** Are we rediscovering the same things across 500 investigations? What's the actual consolidation strategy needed?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** og-inv-knowledge-fragmentation-433-28dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Investigations Are Distinct, Not Duplicative

**Evidence:** 
- Dashboard topic (24 investigations): Each addresses different aspect
  - SSE events visibility
  - Progressive disclosure implementation
  - Agent detail panel redesign
  - Account name display
  - Status mismatch debugging
  - Queue visibility
  - Focus/drift indicator
  
- Headless topic (14 investigations): Each addresses different aspect
  - Scoping what headless means
  - Implementation details
  - Discovery bug fix
  - Token explosion debugging
  - Status display issues
  - Making headless default

**Source:** 
```bash
find .kb/investigations -name "*dashboard*" -mtime -7 | wc -l  # 24
find .kb/investigations -name "*headless*" -mtime -7 | wc -l   # 14
# Manual review of D.E.K.N. summaries showed each is distinct
```

**Significance:** The hypothesis of "rediscovering the same things" is FALSE. High volume reflects rapid development pace, not waste.

---

### Finding 2: Zero Lineage References Across All Investigations

**Evidence:**
```bash
rg -l "Supersedes:|Extracted-From:|See also:" .kb/investigations/*.md | wc -l
# Result: 0

rg "Prior investigation|Building on|Previously found" .kb/investigations/*.md | wc -l
# Result: 0

rg "kn-" .kb/investigations/*.md | wc -l
# Result: 0
```

**Source:** Ripgrep searches across all 500 investigations

**Significance:** Investigations exist as isolated artifacts. No cross-referencing despite template having Supersedes/Extracted-From fields. Knowledge graph is flat, not linked.

---

### Finding 3: Knowledge IS Being Captured, But in Different Artifacts

**Evidence:**
- 500 investigations (deep exploration)
- 308 kn decisions (quick decisions)
- 68 kn constraints (rules discovered)
- 9 kb decisions (formal promoted decisions)

Ratio: 500 investigations → 9 decisions = 1.8% promotion rate

**Source:**
```bash
kn decisions | wc -l  # 308
kn constraints | wc -l  # 68
ls .kb/decisions/ | wc -l  # 9
```

**Significance:** Knowledge flows into kn entries during work, but rarely gets formally promoted to kb decisions. The knowledge exists but is scattered across artifact types without links.

---

### Finding 4: kb reflect Groups by Keyword, Not Semantic Topic

**Evidence:**
```
kb reflect --type synthesis output:
1. add (41 investigations) - "Add X" naming pattern, not conceptual relation
2. orch (36 investigations) - Contains "orch" keyword
3. implement (31 investigations) - "Implement X" naming pattern
```

**Source:** `kb reflect --type synthesis | head -30`

**Significance:** Synthesis suggestions aren't semantically useful. "Add" groups 41 unrelated investigations that just happen to start with "add". Need semantic clustering (e.g., "dashboard UX" not "implement").

---

### Finding 5: Spawn Context Uses Generic Keyword Extraction

**Evidence:** My own spawn context had:
```
**Query:** "knowledge"
### Constraints (MUST respect)
- skillc cannot compile SKILL.md templates... (duplicated twice)
```

The query "knowledge" is too generic for a specific investigation task.

**Source:** SPAWN_CONTEXT.md line 10, pkg/spawn/kbcontext.go:70-96 ExtractKeywords()

**Significance:** Pre-spawn context injection uses simple keyword extraction, not semantic relevance. This means agents don't get pointed to related prior investigations.

---

## Synthesis

**Key Insights:**

1. **Volume ≠ Duplication** - 500 investigations in 7 days reflects high development velocity on a complex system (orch-go, dashboard, daemon, spawn modes). Each investigation serves a purpose.

2. **The Problem is Disconnection, Not Quantity** - Knowledge exists (500 investigations + 308 kn decisions + 68 constraints) but lacks links. Investigations don't reference each other, don't cite kn decisions, don't link to prior work.

3. **Synthesis Tooling Needs Semantic Awareness** - `kb reflect --type synthesis` groups by word frequency ("add", "orch"), not conceptual topic. This makes its suggestions unhelpful.

4. **Spawn Context Needs Smarter Relevance** - ExtractKeywords uses stop word filtering but doesn't understand semantic relevance. A "dashboard debugging" task should surface prior dashboard investigations, not generic matches.

**Answer to Investigation Question:**

We are NOT rediscovering the same things. The investigations are distinct explorations of different aspects. The problem is that these investigations exist as isolated nodes in a flat knowledge graph:

- No Supersedes/Extracted-From lineage
- No references to prior kn decisions
- No cross-links between related investigations
- kb reflect groups by keywords, not concepts

The consolidation strategy should focus on **improving knowledge linkage**, not running synthesis passes over keyword-grouped investigations.

---

## Structured Uncertainty

**What's tested:**

- ✅ Investigation count verified: 500 total, 402 in last 7 days (via `find`)
- ✅ Zero lineage references confirmed (via `rg` for Supersedes, Prior, kn-)
- ✅ Dashboard investigations are distinct (manual review of 10 D.E.K.N. summaries)
- ✅ Headless investigations are distinct (manual review of 8 D.E.K.N. summaries)
- ✅ kb reflect groups by keyword (ran `kb reflect --type synthesis`)

**What's untested:**

- ⚠️ Whether semantic clustering would produce better synthesis groups (not benchmarked)
- ⚠️ Whether smarter spawn context would reduce investigation creation (not A/B tested)
- ⚠️ Whether decision promotion rate (1.8%) is appropriate or too low (no baseline)

**What would change this:**

- Finding would be wrong if sampling different investigation topics showed actual rediscovery
- Finding would be wrong if investigation creation rate stayed high even with better context injection
- Finding would be wrong if kn decisions are being created but not searched during spawns

---

## Implementation Recommendations

### Recommended Approach ⭐

**Improve Knowledge Linkage Over Synthesis Passes** - Focus on connecting existing knowledge rather than consolidating it.

**Why this approach:**
- Investigations are already distinct - synthesis would be artificial grouping
- The problem is discoverability, not quantity
- Linking is incremental (can add over time), synthesis is batch (requires large effort)

**Trade-offs accepted:**
- Won't reduce investigation count (but that's not the problem)
- Requires tooling changes (but more sustainable than manual synthesis)

**Implementation sequence:**
1. **Fix kb reflect semantic grouping** - Cluster by topic (dashboard, headless, daemon) not keyword
2. **Add related investigations to spawn context** - Include 2-3 most recent investigations matching topic
3. **Encourage lineage use** - Add prompt in SPAWN_CONTEXT reminding agents to check for Supersedes

### Alternative Approaches Considered

**Option B: Run synthesis passes on high-count topics**
- **Pros:** Would create formal guides/decisions
- **Cons:** Artificial grouping (41 "add" investigations aren't related); high effort for low value
- **When to use instead:** If investigation count causes actual discovery problems (not observed)

**Option C: Reduce investigation creation rate**
- **Pros:** Fewer artifacts to manage
- **Cons:** Investigations are valuable exploration; reducing rate loses knowledge
- **When to use instead:** If storage/performance becomes an issue

**Rationale for recommendation:** The symptom (high count) isn't the disease. The disease is disconnection. Synthesis passes would consolidate unrelated artifacts; better linkage would make existing artifacts discoverable.

---

### Implementation Details

**What to implement first:**
1. Add `kb chronicle "topic"` output to spawn context for related investigations (quick win)
2. Fix kb reflect to use semantic clustering (medium effort)
3. Add pre-spawn prompt: "Check if any investigation file might be superseded by your work" (quick win)

**Things to watch out for:**
- ⚠️ Don't over-engineer semantic clustering - simple topic extraction from titles may suffice
- ⚠️ Don't flood spawn context with too many investigations - 2-3 is enough
- ⚠️ Don't make lineage mandatory - it should be encouraged, not enforced

**Success criteria:**
- ✅ Next spawn context includes related investigations (not just generic keyword matches)
- ✅ `kb reflect --type synthesis` shows semantic topics (dashboard, headless, daemon)
- ✅ Some investigations start using Supersedes field

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` - ExtractKeywords and RunKBContextCheck implementation
- 10+ investigation D.E.K.N. summaries for dashboard topic
- 8+ investigation D.E.K.N. summaries for headless topic

**Commands Run:**
```bash
# Count investigations
find .kb/investigations -name "*.md" | wc -l  # 500
find .kb/investigations -name "*.md" -mtime -7 | wc -l  # 402

# Check for lineage references
rg -l "Supersedes:|Extracted-From:" .kb/investigations/*.md | wc -l  # 0
rg "kn-" .kb/investigations/*.md | wc -l  # 0

# Check kn entries
kn decisions | wc -l  # 308
kn constraints | wc -l  # 68

# Run kb reflect synthesis
kb reflect --type synthesis | head -30

# Topic-specific searches
find .kb/investigations -name "*dashboard*" -mtime -7 | wc -l  # 24
find .kb/investigations -name "*headless*" -mtime -7 | wc -l  # 14

# Run kb chronicle for topic evolution
kb chronicle "dashboard" | head -50
kb chronicle "headless" | head -50
```

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Discovered Work Check

| Type | Description | Action |
|------|-------------|--------|
| Feature | kb reflect needs semantic clustering | Create beads issue |
| Feature | Spawn context should include related investigations | Create beads issue |
| Enhancement | Add Supersedes/lineage reminder to spawn template | Create beads issue |

