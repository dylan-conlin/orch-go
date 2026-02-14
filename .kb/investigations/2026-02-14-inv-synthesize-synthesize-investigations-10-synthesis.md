<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** 10 synthesis investigations reveal a consistent workflow pattern: read investigations → identify themes → update existing guides → archive originals; discovered kb reflect scans archived/synthesized directories causing false positives.

**Evidence:** All 10 investigations (status, daemon, dashboard, orchestrator, spawn, verification, completion, extract, serve) followed identical synthesis workflow; created or updated guides (status.md, daemon.md, dashboard.md, etc.); 3 investigations reported kb reflect bugs.

**Knowledge:** Synthesis is an iterative process happening in waves (Jan 6-7-8-14-17); guide-first approach established (single authoritative reference per topic); kb reflect needs to exclude archived/synthesized directories from synthesis detection.

**Next:** Document synthesis workflow pattern in `.kb/guides/synthesis-workflow.md`; file issue for kb reflect to exclude archived/synthesized directories; close this investigation.

**Authority:** implementation - Documenting established pattern within existing synthesis context, no architectural changes needed

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Synthesize Synthesize Investigations 10 Synthesis

**Question:** What common patterns emerge from 10 synthesis investigations, and how should the synthesis workflow itself be documented?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** og-feat-synthesize-synthesize-investigations-14feb-264b
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: All 10 Investigations Follow Identical Synthesis Workflow

**Evidence:** Every synthesis investigation analyzed followed this exact 5-step pattern:
1. Run `kb chronicle "topic"` to understand evolution
2. Read all related investigations (10-60 files per synthesis)
3. Identify patterns and themes (3-9 major themes per topic)
4. Update existing guide OR create new guide (never left as scattered investigations)
5. Document "Last verified" date and investigation count

Examples:
- Status synthesis (Jan 6): 10 investigations → updated `.kb/guides/status.md`
- Daemon synthesis (Jan 7): 33 investigations → updated `.kb/guides/daemon.md`
- Dashboard synthesis (Jan 7): 58 investigations → updated `.kb/guides/dashboard.md`
- Verification synthesis (Jan 14): 25 investigations → created `.kb/guides/verification.md`

**Source:** All 10 investigation files in SPAWN_CONTEXT.md

**Significance:** The synthesis workflow is well-established and consistent across all topics. This pattern should be documented as the authoritative synthesis process.

---

### Finding 2: Guide-First Approach is Established Pattern

**Evidence:** All 10 synthesis investigations prioritized updating existing guides over creating new ones:

| Synthesis Topic | Action Taken | Rationale |
|----------------|--------------|-----------|
| Status | Updated existing guide | "Guide structure already good, just needs updates" |
| Daemon | Updated existing guide | "Single authoritative reference prevents re-investigation" |
| Dashboard | Updated existing guide | "Consolidates knowledge from 14 new investigations" |
| Orchestrator | Updated existing guide | "Guide-first maintenance with periodic synthesis" |
| Spawn | Updated existing guide | "Existing guide is 85% complete, updating is ~15 min vs new doc" |
| Verification | Created NEW guide | Only when no existing guide covered the patterns |
| Completion (2x) | Updated existing guides | Both completion syntheses updated `.kb/guides/completion.md` |
| Extract | Verified existing guide | "Guide already complete and up-to-date" |
| Serve | Created NEW guide | New topic not covered elsewhere |

**Pattern:** Update existing guides first; only create new guides when no authoritative reference exists.

**Source:** Synthesis sections from all 10 investigation files

**Significance:** This prevents knowledge fragmentation and maintains single sources of truth per topic.

---

### Finding 3: kb reflect Has Systematic Issues with Synthesis Detection

**Evidence:** Three separate investigations discovered kb reflect bugs:

1. **Scans archived/synthesized directories** (Extract synthesis, Jan 17):
   - After archiving 14 investigations to `synthesized/code-extraction-patterns/`, kb reflect still reported "13 investigations need synthesis"
   - Quote: "This is a bug - investigations in archived/ and synthesized/ directories should be excluded from synthesis detection"

2. **Lexical clustering ≠ semantic clustering** (Verification synthesis, Jan 14):
   - "Extract" keyword matched: code extraction, knowledge extraction, constraint extraction - all unrelated topics
   - Quote: "Lexical cluster != conceptual model"

3. **Time-drifted conclusions** (Verification synthesis, Jan 14):
   - Investigations can become stale after code changes
   - Quote: "Investigation findings can become stale after code changes. Re-validation required during consolidation."

**Source:**
- `2026-01-17-inv-synthesize-extract-investigation-cluster-13.md:59-69`
- `2026-01-14-inv-synthesize-verification-investigations-consolidate-findings.md:160-184`

**Significance:** kb reflect needs fixes to:
- Exclude archived/ and synthesized/ from synthesis scanning
- Use semantic clustering not just keyword matching
- Surface stale investigations (code changed since investigation)

---

### Finding 4: Synthesis Happens in Iterative Waves

**Evidence:** Timeline analysis shows synthesis occurring in concentrated bursts:

| Date | Synthesis Count | Topics |
|------|----------------|--------|
| Jan 6 | 3 syntheses | Status (10), Spawn (36), Session (10) |
| Jan 7 | 5 syntheses | Daemon (33), Dashboard (58), Orchestrator (47), more |
| Jan 8 | 3 syntheses | Completion (10), CLI (18), Synthesis (26) |
| Jan 14 | 1 synthesis | Verification (25) |
| Jan 17 | 4 syntheses | Completion-2 (28), Extract (13), Serve (9), Design (26) |

**Pattern:** Synthesis work happens in focused sessions, typically triggered by `kb reflect --type synthesis` output showing 10+ investigation clusters.

**Source:** Investigation dates from all 10 files

**Significance:** Synthesis is a recurring maintenance task, not one-time cleanup. The rhythm appears to be: accumulate investigations over 1-2 weeks → synthesis session → repeat.

---

### Finding 5: Investigation Counts Provide System Health Metric

**Evidence:** Synthesis investigations track total investigation counts per topic:

| Topic | Count at Synthesis | Trend |
|-------|-------------------|-------|
| Dashboard | 44 → 58 (Jan 6-7) | +14 in 1 day (active development) |
| Spawn | 36 → 60 (Jan 6-7) | +24 in 1 day (mature but evolving) |
| Daemon | 31 → 33 (Jan 6-7) | +2 (stabilizing) |
| Completion | 10 → 28 (Jan 8-17) | +18 (major churn area) |
| Verification | 25 (Jan 14) | Standalone cluster |

**Pattern:** High investigation velocity indicates:
- Active feature development (dashboard: 14/day)
- System maturity issues (completion: 18 in 9 days)
- Friction points needing architectural attention

Low investigation velocity indicates:
- System stabilization (daemon: 2 additions)
- Effective guide coverage

**Source:** Investigation counts from synthesis summaries

**Significance:** Investigation accumulation rate is a leading indicator of system friction. Topics with 10+ investigations in a week may need architectural review, not just synthesis.

---

## Synthesis

**Key Insights:**

1. **Synthesis is a Well-Defined Workflow, Not Ad-Hoc Cleanup** - All 10 investigations followed the identical 5-step pattern (chronicle → read → themes → guide → verify). This consistency reveals synthesis as a repeatable process that should be documented, not rediscovered each time. (Finding 1)

2. **Guide-First Prevents Knowledge Fragmentation** - The pattern "update existing guide > create new guide > leave as scattered investigations" appears in 8 of 10 syntheses. This establishes guides as the single source of truth, with investigations serving as historical context. New guides are only created when no authoritative reference exists. (Finding 2)

3. **kb reflect Is a Discovery Tool, Not a Decision Tool** - Three investigations found kb reflect's synthesis recommendations require human validation. Lexical clustering (keyword matching) creates false positives; archived/synthesized directories shouldn't be scanned; time-drift makes old conclusions unreliable. kb reflect surfaces signals; humans must triage. (Finding 3)

4. **Synthesis Rhythm Follows Investigation Accumulation** - Synthesis happens in waves (Jan 6: 3 syntheses, Jan 7: 5 syntheses), triggered when `kb reflect` shows 10+ investigation clusters. This establishes a natural rhythm: accumulate over 1-2 weeks → synthesis session → repeat. (Finding 4)

5. **Investigation Velocity Indicates System Health** - Topics with high investigation counts (dashboard: 58, spawn: 60, completion: 28) signal either active development or system friction. Low counts (daemon: 33 stable) indicate maturity. Synthesis consolidates knowledge; it doesn't address root friction. (Finding 5)

**Answer to Investigation Question:**

The 10 synthesis investigations reveal a **mature, repeatable synthesis workflow** with these characteristics:

1. **Standard Process:** Chronicle → Read investigations → Identify themes → Update/create guides → Archive originals
2. **Guide-First Pattern:** Update existing authoritative references rather than creating parallel documentation
3. **Iterative Rhythm:** Synthesis occurs in waves when kb reflect shows 10+ clusters
4. **Quality Gates:** Verify guide completeness, document "last verified" dates, cross-reference between guides
5. **Known Issues:** kb reflect scans archived/synthesized directories (false positives) and uses lexical not semantic clustering

**Recommended consolidation:**
- Create `.kb/guides/synthesis-workflow.md` documenting the 5-step synthesis process
- File kb reflect issues: exclude archived/synthesized directories, improve clustering
- Update `.kb/models/kb-reflect-cluster-hygiene.md` with findings from these 10 syntheses
- No additional synthesis needed - the pattern is already well-established through practice

---

## Structured Uncertainty

**What's tested:**

- ✅ All 10 investigations followed 5-step synthesis workflow (verified: read all 10 investigations, pattern consistent)
- ✅ Guide-first approach used in 8/10 syntheses (verified: counted guide updates vs new creations)
- ✅ kb reflect scans archived/synthesized directories (verified: extract synthesis explicitly tested this)
- ✅ Synthesis happens in waves Jan 6-7-8-14-17 (verified: dates from investigation files)
- ✅ Investigation velocity varies by topic (verified: counts from synthesis summaries)

**What's untested:**

- ⚠️ Whether documenting synthesis workflow will reduce future investigation of "how to synthesize"
- ⚠️ Whether kb reflect fixes would eliminate false positives (issue needs to be filed and fixed)
- ⚠️ Whether synthesis rhythm will continue (only observed Jan 2026, might be seasonal)
- ⚠️ Whether high investigation counts actually correlate with system friction vs feature development

**What would change this:**

- Finding would be wrong if additional synthesis investigations used different workflows (spot-checked 10, all identical)
- Finding would be wrong if kb reflect was already fixed to exclude archived/ (tested in Jan 17, still broken)
- Pattern would change if future syntheses skip guide updates (would indicate pattern regression)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Create Synthesis Workflow Guide** - Document the established 5-step synthesis process in `.kb/guides/synthesis-workflow.md` and update kb reflect model with discovered issues.

**Why this approach:**
- All 10 syntheses used identical workflow (Finding 1) - pattern is proven, not theoretical
- Guide-first approach is established (Finding 2) - synthesis itself deserves a guide
- kb reflect issues are systematic (Finding 3) - need model update to document workarounds
- Synthesis is recurring maintenance (Finding 4) - future agents benefit from documented process

**Trade-offs accepted:**
- Another guide to maintain (but synthesis is well-established, unlikely to change)
- Doesn't fix kb reflect bugs (requires kb-cli code changes, separate issue)
- Assumes current synthesis pattern is optimal (but 10 successful iterations suggest it is)

**Implementation sequence:**
1. Create `.kb/guides/synthesis-workflow.md` with 5-step process, guide-first pattern, archival workflow
2. Update `.kb/models/kb-reflect-cluster-hygiene.md` with findings about archived/synthesized scanning and lexical clustering
3. Create beads issue for kb-cli to exclude archived/synthesized directories from synthesis detection
4. Document investigation velocity as system health metric in synthesis guide

### Alternative Approaches Considered

**Option B: Leave synthesis undocumented (implicit knowledge)**
- **Pros:** No documentation overhead; pattern already established through practice
- **Cons:** New agents rediscover the pattern each time; kb reflect issues get re-investigated
- **When to use instead:** If synthesis is infrequent enough that documentation overhead exceeds rediscovery cost

**Option C: Create kb quick constraint instead of guide**
- **Pros:** Lighter weight than full guide; discoverable via kb context
- **Cons:** 5-step process with examples doesn't fit constraint format; no room for kb reflect troubleshooting
- **When to use instead:** If synthesis was a simple rule ("always update guides") instead of multi-step workflow

**Rationale for recommendation:** The synthesis workflow is complex enough (5 steps, guide-first pattern, archival decisions, kb reflect issues) to warrant a full guide. 10 successful syntheses prove the pattern works. Future synthesis work will benefit from documented process, especially kb reflect troubleshooting.

---

### Implementation Details

**What to implement first:**
- Create `.kb/guides/synthesis-workflow.md` documenting the 5-step process (highest impact, prevents rediscovery)
- Update `.kb/models/kb-reflect-cluster-hygiene.md` with archived/synthesized directory issue
- File kb-cli issue for synthesis detection improvements

**Things to watch out for:**
- ⚠️ Synthesis workflow guide could become prescriptive instead of descriptive (document what works, don't mandate it)
- ⚠️ kb reflect model update needs to distinguish "bugs to fix" from "known limitations to work around"
- ⚠️ Investigation velocity metric could be misused (high counts aren't always bad - could be active development)

**Areas needing further investigation:**
- Whether semantic clustering would improve kb reflect (current lexical clustering has false positives)
- Whether investigation velocity correlates with code churn or bug reports (health metric validation)
- Whether synthesis rhythm is seasonal (only observed Jan 2026) or will continue monthly

**Success criteria:**
- ✅ Next synthesis investigation references the workflow guide instead of rediscovering the pattern
- ✅ kb reflect model documents archived/synthesized directory issue with workaround
- ✅ kb-cli issue filed and acknowledged by maintainers
- ✅ Future agents can answer "how do I synthesize investigations?" by reading the guide

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-06-inv-synthesize-status-investigations.md` - Status synthesis (10 investigations)
- `.kb/investigations/2026-01-07-inv-synthesize-daemon-investigations.md` - Daemon synthesis (33 investigations)
- `.kb/investigations/2026-01-07-inv-synthesize-dashboard-investigations.md` - Dashboard synthesis (58 investigations)
- `.kb/investigations/2026-01-07-inv-synthesize-orchestrator-investigations.md` - Orchestrator synthesis (47 investigations)
- `.kb/investigations/2026-01-07-inv-synthesize-spawn-investigations.md` - Spawn synthesis (60 investigations)
- `.kb/investigations/2026-01-14-inv-synthesize-verification-investigations-consolidate-findings.md` - Verification synthesis (25 investigations)
- `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md` - Completion architectural analysis (26 investigations)
- `.kb/investigations/2026-01-17-inv-synthesize-28-completed-investigations-complete.md` - Completion synthesis (28 investigations)
- `.kb/investigations/2026-01-17-inv-synthesize-extract-investigation-cluster-13.md` - Extract synthesis (13 investigations, kb reflect bug discovery)
- `.kb/investigations/2026-01-17-inv-synthesize-serve-investigation-cluster-investigations.md` - Serve synthesis (9 investigations)

**Commands Run:**
```bash
# Review chronicle of synthesis work
kb chronicle "synthesize" | head -100

# Report phase to beads
bd comment orch-go-v3d "Phase: Planning - Analyzing 10 synthesize investigations to identify patterns and create consolidated guide"
bd comment orch-go-v3d "investigation_path: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-synthesize-synthesize-investigations-10-synthesis.md"
```

**Related Artifacts:**
- **Model:** `.kb/models/kb-reflect-cluster-hygiene.md` - Should be updated with findings about archived/synthesized scanning
- **Guides Created by Syntheses:** `.kb/guides/status.md`, `.kb/guides/daemon.md`, `.kb/guides/dashboard.md`, `.kb/guides/orchestrator-session-management.md`, `.kb/guides/spawn.md`, `.kb/guides/verification.md`, `.kb/guides/completion.md`, `.kb/guides/background-services-performance.md`, `.kb/guides/code-extraction-patterns.md`

---

## Investigation History

**2026-02-14 [start]:** Investigation started
- Initial question: What patterns emerge from 10 synthesis investigations?
- Context: kb reflect flagged synthesis cluster; spawned to consolidate synthesis knowledge

**2026-02-14 [analysis]:** Read all 10 synthesis investigations
- Identified 5-step synthesis workflow used consistently across all 10
- Found guide-first approach in 8/10 syntheses
- Discovered kb reflect bugs (archived/synthesized scanning, lexical clustering)

**2026-02-14 [synthesis]:** Patterns documented
- Synthesis is repeatable process, not ad-hoc cleanup
- Investigation velocity indicates system health
- kb reflect issues need model update and kb-cli fix

**2026-02-14 [complete]:** Investigation completed
- Status: Complete
- Key outcome: Synthesis workflow pattern documented; recommend creating `.kb/guides/synthesis-workflow.md` and updating kb reflect model with discovered issues
