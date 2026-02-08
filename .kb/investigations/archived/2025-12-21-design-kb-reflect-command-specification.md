<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `kb reflect` is a single command with `--type` flag for four reflection modes: synthesis (investigations needing consolidation), stale (decisions with no recent citations), drift (practice diverged from decision), promote (kn entries ready for kb promotion). Each mode uses grep-based detection with action-oriented output.

**Evidence:** Synthesized 5 prior investigations (.7 citations, .8 temporal signals, .9 chronicle, .10 constraint validation, 4kwt.8 unexplored questions). All signals are detectable via content parsing (grep/rg); no new data structures needed.

**Knowledge:** Reflection should be triggered by density patterns not time intervals. The four types map directly to discovered signals: repeated constraints → promote, investigation clusters → synthesis, implementation contradiction → drift, low citation → stale.

**Next:** Implement `kb reflect` as thin wrapper around rg queries. Close this issue and create feature-impl issue for implementation.

**Confidence:** High (85%) - Design synthesizes validated investigation findings; command interface untested.

---

# Design: kb reflect Command Specification

**Question:** What command interface and implementation approach should `kb reflect` use to surface synthesis opportunities, stale decisions, practice drift, and promotion candidates?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** og-arch-design-kb-reflect-21dec
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete
**Confidence:** High (85%)
**kn externalized:** kn-db08a3

---

## Problem Framing

### Design Question

How should `kb reflect` enable the system to surface patterns across artifacts that require human attention, without adding maintenance overhead or new data structures?

### Success Criteria

1. **Actionable output** - Each reflection type produces specific recommendations (not just raw data)
2. **Low false positive rate** - <20% noise in suggestions to avoid training users to ignore
3. **Fast execution** - <5 seconds for all reflection types
4. **Zero maintenance** - No new databases, indexes, or tracking to maintain
5. **Discoverable** - Users can run `kb reflect` to see what needs attention

### Constraints

- Must use existing data (kn entries, kb artifacts, git history)
- No new frontmatter requirements for artifacts
- Content parsing (grep) is the detection mechanism (per .7 citation investigation)
- Daemon integration optional (per .8 temporal signals investigation)

### Scope

**In scope:**
- Command interface design (`kb reflect --type <type>`)
- Detection algorithms for each reflection type
- Output format specification
- Implementation approach (thin wrapper around rg)

**Out of scope:**
- Daemon integration (follow-up work)
- SessionStart hook integration (follow-up work)
- Automated remediation (human-in-loop is intentional)

---

## Findings

### Finding 1: Four Reflection Types Map to Discovered Signals

**Evidence:** The five prior investigations discovered specific patterns that require human attention:

| Investigation | Signal Discovered | Maps to Reflection Type |
|---------------|-------------------|------------------------|
| ws4z.8 Temporal Signals | Investigation clustering | `--type synthesis` |
| ws4z.8 Temporal Signals | Repeated constraints | `--type promote` |
| ws4z.10 Constraint Validation | Implementation contradicts constraint | `--type drift` |
| ws4z.7 Citations | Low/zero citation count | `--type stale` |

Each signal has a tested detection mechanism:
- Investigation clustering: `ls .kb/investigations/*{topic}*.md | wc -l`
- Repeated constraints: `jq '[.[] | select(.content | test("topic"))]' .kn/entries.jsonl`
- Implementation contradiction: `rg "constraint-pattern" pkg/` + compare to constraint
- Low citation: `rg -c "artifact-name" .kb/` → count

**Source:** 
- `.kb/investigations/2025-12-21-inv-temporal-signals-autonomous-reflection.md` lines 35-85
- `.kb/investigations/2025-12-21-inv-questioning-inherited-constraints-when-how.md` lines 39-100
- `.kb/investigations/2025-12-21-inv-citation-mechanisms-how-artifacts-track.md` (SYNTHESIS.md)

**Significance:** The four reflection types are not arbitrary—they map directly to validated signals with tested detection methods. This ensures the command will surface genuinely actionable items.

---

### Finding 2: Content Parsing is Sufficient (No Index Needed)

**Evidence:** From ws4z.7 citation investigation:
- Inbound link discovery via `rg "artifact-name" .kb/` takes <100ms on 138 files
- 37% of artifacts (51/138) contain explicit references to other artifacts
- `ref_count` field in kn entries is unused (always 0)
- Current scale (172 investigations, 30 kn entries) doesn't justify index

Performance validated:
```bash
# Citation count for single artifact: ~50ms
time rg -l "2025-12-21-design-minimal-artifact-taxonomy" .kb/

# All cross-references: ~200ms
time grep -roh "2025-12-[0-9][0-9]-[a-z0-9-]*\.md" .kb/ | sort | uniq -c | sort -rn
```

**Source:** `.orch/workspace/og-inv-citation-mechanisms-how-21dec/SYNTHESIS.md` lines 30-48

**Significance:** No new data structures needed. `kb reflect` can be implemented as thin wrappers around rg/grep/jq queries. This eliminates maintenance overhead and keeps the system simple.

---

### Finding 3: Density Thresholds Over Time Intervals

**Evidence:** From ws4z.8 temporal signals investigation:
- 37 kn entries on 2025-12-21 vs 7 on 2025-12-20 (density spike, not time-based)
- 4 investigation iterations on "tmux fallback" in single day (clustering)
- 5 duplicate constraint entries on "tmux fallback" in 3 minutes (repeated constraints)

The pattern: Reflection should trigger on **quantity thresholds** not **time elapsed**:
- 3+ investigations on same topic → synthesis needed
- 2+ kn entries with similar content → consolidation needed
- 0 citations in 7+ days → potentially stale (but age alone is weak signal)

**Source:** `.kb/investigations/2025-12-21-inv-temporal-signals-autonomous-reflection.md` lines 85-110

**Significance:** `kb reflect --type stale` should NOT use pure time-based detection. Combine with citation count: "decisions with zero citations AND >7 days old" reduces false positives.

---

### Finding 4: Constraint Validation Requires Code Testing

**Evidence:** From ws4z.10 constraint validation investigation:

"Wrong constraint" vs "misapplication" can only be distinguished by testing:
1. Search codebase for constraint's domain (`rg "session.*id" pkg/`)
2. If code contradicts constraint → constraint is outdated
3. If code matches constraint → constraint is valid (problem is misapplication)

Example: "fire-and-forget - no session ID capture" constraint (Dec 19) was valid until session.go implementation (Dec 21) superseded it.

**Source:** `.kb/investigations/2025-12-21-inv-questioning-inherited-constraints-when-how.md` lines 39-57, 298-309

**Significance:** `kb reflect --type drift` cannot fully automate contradiction detection. It should:
1. Find constraints mentioning code patterns
2. Check if code matching pattern exists
3. Surface potential contradictions for human review

---

### Finding 5: Chronicle Pattern Informs Synthesis Type

**Evidence:** From ws4z.9 chronicle investigation:

The registry evolution chronicle was created by:
1. Gathering sources across time (git, kn, investigations)
2. Orchestrator synthesizing narrative (not automated)
3. Producing investigation artifact with "synthesis-" prefix

This maps to `kb reflect --type synthesis`:
- Detect: 3+ investigations on same topic
- Output: "Topic X has N investigations, no synthesis"
- Action: Orchestrator runs `kb chronicle "topic"` to gather sources, then synthesizes

**Source:** `.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md` lines 1-12, 270-292

**Significance:** The synthesis reflection type surfaces opportunities; the chronicle command gathers sources; the orchestrator produces narrative. These are three distinct steps.

---

## Exploration: Command Interface Options

### Option A: Single Command with --type Flag ⭐

```bash
# Run all reflection types
kb reflect

# Run specific type
kb reflect --type synthesis
kb reflect --type stale
kb reflect --type drift
kb reflect --type promote
```

**Mechanism:** Single binary/script with type dispatch. Default runs all types.

**Pros:**
- Discoverable: `kb reflect` shows everything
- Consistent: One command to learn
- Extensible: Add new types easily

**Cons:**
- All types have same output format (may not fit all)
- Running all types might be slow

**Complexity:** Low - single entry point with flag routing

---

### Option B: Separate Commands

```bash
kb synthesis-check     # investigations needing consolidation
kb stale-check         # decisions with no recent citations
kb drift-check         # practice diverged from decision
kb promote-check       # kn entries ready for promotion
```

**Mechanism:** Four separate commands, each with focused logic.

**Pros:**
- Each command can have tailored output format
- Clear separation of concerns

**Cons:**
- Four commands to learn
- No unified "show me everything" view
- Inconsistent naming pattern

**Complexity:** Low per command, but higher total surface area

---

### Option C: Query-Style Interface

```bash
kb query "synthesis candidates"
kb query "stale decisions"
kb query "drift detected"
kb query "promote candidates"
```

**Mechanism:** Natural language-ish query interface.

**Pros:**
- Flexible, extensible
- Could support ad-hoc queries

**Cons:**
- Harder to document
- Requires query parsing
- Less discoverable (what queries are valid?)

**Complexity:** Medium-high - query parsing adds complexity

---

## Synthesis

### Command Interface Recommendation

**⭐ RECOMMENDED: Option A - Single Command with --type Flag**

```bash
kb reflect                     # Run all reflection types, show summary
kb reflect --type synthesis    # Investigations needing consolidation
kb reflect --type stale        # Decisions with low/no citations
kb reflect --type drift        # Constraints contradicted by code
kb reflect --type promote      # kn entries worth promoting to kb
```

**Rationale:**
- Matches existing pattern: `kb context`, `kb search`, `kb chronicle` 
- Single command is most discoverable
- Flag provides filtering without separate commands
- Aligns with daemon integration (daemon runs `kb reflect`, surfaces summary)

### Detection Algorithms

#### synthesis: Investigations Needing Consolidation

**Detection:**
```bash
# Extract topic from investigation filenames
# Group by normalized topic (remove date, prefix)
# Filter to groups with 3+ members
ls .kb/investigations/*.md | 
  sed 's/.*2025-[0-9-]*-//' | 
  sed 's/\.md$//' |
  sort | uniq -c | sort -rn |
  awk '$1 >= 3 {print $0}'
```

**Output:**
```
SYNTHESIS OPPORTUNITIES
━━━━━━━━━━━━━━━━━━━━━━

1. tmux-fallback (4 investigations)
   └─ Consider: kb chronicle "tmux fallback" to synthesize findings

2. model-handling (3 investigations)
   └─ Consider: kb chronicle "model handling" to synthesize findings
```

#### stale: Decisions with Low/No Citations

**Detection:**
```bash
# For each decision, count citations across kb/
for decision in .kb/decisions/*.md; do
  name=$(basename "$decision" .md)
  count=$(rg -c "$name" .kb/investigations/ 2>/dev/null | wc -l)
  age=$(( ($(date +%s) - $(stat -f %m "$decision")) / 86400 ))
  if [ $count -eq 0 ] && [ $age -gt 7 ]; then
    echo "$decision (0 citations, ${age}d old)"
  fi
done
```

**Output:**
```
POTENTIALLY STALE DECISIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━

1. 2025-12-15-legacy-api-retirement.md
   └─ 0 citations, 6 days old
   └─ Consider: Validate still relevant or mark superseded

2. 2025-12-10-python-orch-cli-patterns.md
   └─ 0 citations, 11 days old
   └─ Consider: Context shift (Go rewrite) may have invalidated
```

#### drift: Constraints Contradicted by Code

**Detection:**
```bash
# For each constraint, extract code pattern mentioned
# Check if codebase contradicts the constraint
cat .kn/entries.jsonl | jq -r 'select(.type == "constraint") | 
  "\(.id)|\(.content)"' | while IFS='|' read -r id content; do
  # Extract code-related keywords
  patterns=$(echo "$content" | grep -oE '(func|pkg|cmd|\.go|session|registry)' | head -3)
  if [ -n "$patterns" ]; then
    # Check if code contradicts constraint (heuristic)
    # This is simplified - real impl needs semantic matching
    echo "Potential drift: $id"
  fi
done
```

**Output:**
```
POTENTIAL CONSTRAINT DRIFT
━━━━━━━━━━━━━━━━━━━━━━━━━

1. kn-34d52f: "orch-go tmux spawn is fire-and-forget - no session ID capture"
   └─ But: pkg/spawn/session.go provides WriteSessionID/ReadSessionID
   └─ Consider: Validate and supersede if outdated

2. kn-abc123: "Registry is single source of truth for agent state"
   └─ But: OpenCode sessions now provide authoritative state
   └─ Consider: Review architectural alignment
```

#### promote: kn Entries Ready for kb Promotion

**Detection:**
```bash
# Find kn entries with high citation count or duplicates
# Group similar constraints and count
cat .kn/entries.jsonl | jq -r '.content' | 
  sort | uniq -c | sort -rn |
  awk '$1 >= 2 {print "Duplicate:", $0}'

# Find constraints referenced in investigations
for entry in $(cat .kn/entries.jsonl | jq -r '.id'); do
  count=$(rg -c "$entry" .kb/investigations/ 2>/dev/null | wc -l)
  if [ $count -ge 3 ]; then
    echo "High citation: $entry ($count refs)"
  fi
done
```

**Output:**
```
PROMOTION CANDIDATES
━━━━━━━━━━━━━━━━━━━━

1. DUPLICATES NEEDING CONSOLIDATION:
   └─ 4 entries about "tmux fallback"
   └─ Consider: Consolidate to single authoritative kn entry

2. HIGH-CITATION ENTRIES:
   └─ kn-de6832 cited in 5 investigations
   └─ Consider: Promote to .kb/decisions/ or .kb/principles.md
```

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Design synthesizes five validated investigations with tested detection mechanisms. Command interface follows established patterns. Uncertainty is in implementation details and threshold tuning.

**What's certain:**

- ✅ Four reflection types map to discovered signals (validated by investigations)
- ✅ Content parsing (grep/rg) is sufficient for detection (tested in .7)
- ✅ Density thresholds beat time intervals (evidence from .8)
- ✅ Drift detection requires human validation (constraint testing from .10)

**What's uncertain:**

- ⚠️ Exact thresholds (3+ investigations? 2+ duplicates?) need tuning
- ⚠️ Drift detection heuristics may have high false positive rate
- ⚠️ Performance at scale (1000+ artifacts) untested

**What would increase confidence to Very High (95%):**

- Implement MVP and test against real usage patterns
- Tune thresholds based on false positive/negative rates
- Validate drift detection on known outdated constraints

---

## Implementation Recommendations

### Recommended Approach ⭐

**Shell Script MVP with Go Migration Path** - Implement `kb reflect` as shell script wrapping rg/jq queries, with plan to migrate to Go if performance or complexity requires.

**Why this approach:**
- Fast to implement (hours, not days)
- Easy to tune detection heuristics
- Zero dependencies (uses existing tools)
- Matches `kb` command pattern (kb is already shell-based)

**Trade-offs accepted:**
- Shell has limits for complex parsing
- Migration to Go may be needed later
- Cross-platform considerations (macOS vs Linux)

**Implementation sequence:**
1. Create `kb reflect` shell script with --type flag routing
2. Implement `--type synthesis` first (simplest detection)
3. Add `--type promote` (builds on synthesis logic)
4. Add `--type stale` (citation counting)
5. Add `--type drift` (most complex, last)

### Alternative Approaches Considered

**Option B: Go Implementation from Start**
- **Pros:** Better performance, easier testing, type safety
- **Cons:** Higher upfront investment, kb is currently shell-based
- **When to use instead:** If kb migrates to Go entirely

**Option C: Daemon-Only (No Command)**
- **Pros:** Fully autonomous, no user action needed
- **Cons:** Loses on-demand capability, harder to debug
- **When to use instead:** After command proves valuable, add daemon integration

**Rationale for recommendation:** Shell script matches existing kb pattern, allows rapid iteration on detection heuristics, and can be migrated to Go if needed.

---

### Implementation Details

**What to implement first:**
1. `kb reflect` entry point with --type dispatch
2. `--type synthesis` detection (filename clustering)
3. Human-readable output format (not JSON by default)
4. `--json` flag for machine-readable output

**File targets:**
- Create: `~/.local/bin/kb-reflect` (or wherever kb lives)
- Modify: None (standalone script)

**Things to watch out for:**
- ⚠️ macOS vs Linux sed/awk differences
- ⚠️ Path handling for cross-project usage
- ⚠️ Performance on large artifact sets (add timing)

**Success criteria:**
- ✅ `kb reflect` runs in <5 seconds on 200+ artifacts
- ✅ Each type produces actionable recommendations
- ✅ False positive rate <20% (measure via user feedback)

---

## Acceptance Criteria (for Implementation)

**Required for MVP:**
- [ ] `kb reflect` runs all types, shows summary
- [ ] `--type synthesis` detects investigation clusters (3+)
- [ ] `--type promote` detects duplicate kn entries
- [ ] `--type stale` detects zero-citation decisions >7d
- [ ] `--type drift` surfaces potential constraint contradictions
- [ ] Human-readable output with actionable suggestions
- [ ] `--json` flag for machine-readable output

**Optional enhancements:**
- [ ] `--project <path>` for cross-project reflection
- [ ] Daemon integration (surface in `orch daemon` output)
- [ ] SessionStart hook (surface if suggestions exist)

---

## References

**Investigations Synthesized:**
- `.kb/investigations/2025-12-21-inv-temporal-signals-autonomous-reflection.md` - Signal ranking, daemon recommendation
- `.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md` - Chronicle as view pattern
- `.kb/investigations/2025-12-21-inv-questioning-inherited-constraints-when-how.md` - Constraint validation signals
- `.kb/investigations/2025-12-21-inv-citation-mechanisms-how-artifacts-track.md` (via SYNTHESIS.md) - Grep-based citations
- `.kb/investigations/2025-12-21-inv-multi-agent-synthesis-when-multiple.md` - Multi-agent patterns

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - 5+3 artifact types
- **Epic:** `orch-go-ws4z` - System Self-Reflection - Temporal Pattern Awareness
- **Investigation:** `.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md`

**Commands Tested:**
```bash
# Investigation clustering
ls .kb/investigations/*.md | sed 's/.*2025-[0-9-]*-//' | sort | uniq -c | sort -rn

# Citation counting
rg -l "artifact-name" .kb/

# kn duplicate detection  
cat .kn/entries.jsonl | jq -s '[.[] | select(.content | test("pattern"))]'
```

---

## Self-Review

- [x] All 4 architect phases completed (Problem Framing, Exploration, Synthesis, Externalization)
- [x] Recommendation made with trade-off analysis
- [x] Implementation-ready output (file targets, acceptance criteria, out of scope)
- [x] Investigation artifact produced
- [ ] Feature list reviewed (N/A - no .orch/features.json exists)

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-21 17:00:** Investigation started
- Initial question: Design kb reflect command specification
- Context: Synthesize findings from ws4z epic investigations

**2025-12-21 17:15:** Read 5 prior investigations
- ws4z.7: Citations via grep (sufficient, no index)
- ws4z.8: Temporal signals (density > time, daemon recommended)
- ws4z.9: Chronicle as view (not new artifact type)
- ws4z.10: Constraint validation (test against code)
- 4kwt.8: Unexplored questions (reflection checkpoint pattern)

**2025-12-21 17:30:** Mapped signals to reflection types
- synthesis ← investigation clustering
- promote ← repeated constraints
- drift ← implementation contradiction
- stale ← low citation + age

**2025-12-21 17:45:** Explored command interface options
- Option A: Single command with --type flag (recommended)
- Option B: Separate commands (rejected - less discoverable)
- Option C: Query-style (rejected - complexity)

**2025-12-21 18:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: `kb reflect --type <type>` with four types, shell script MVP, grep-based detection
