## Summary (D.E.K.N.)

**Delta:** Case Files should be artifact types, not views. The chronicle precedent doesn't apply because case files are operational coordination artifacts with agent-readable structure needs, not historical narratives. The "views" approach creates a catch-22: agents can't read HTML to know when contradictions exist.

**Evidence:** Architect cited chronicle precedent (kn-160dc9: minimal taxonomy, synthesized narratives). But case files have: (1) diagnosis-first structure fundamentally different from investigations, (2) lifecycle states (Active/Resolved) unlike static views, (3) agent consumption requirement (kb context must surface contradictions), (4) operational purpose (coordinate ongoing work) vs chronicle's historical purpose.

**Knowledge:** The chronicle precedent conflates "synthesized narrative" (structure) with "view-not-type" (implementation). Case files share structure similarity but have different operational requirements. The coaching plugin failure mode was agents couldn't see prior contradictory conclusions - HTML views perpetuate this problem. Artifact status enables kb context integration, lifecycle management, and contradiction detection.

**Next:** Recommend artifact type adoption with phased implementation: (1) Add CaseFile to ArtifactType enum, (2) Create markdown template (not HTML), (3) Add kb create/update commands, (4) Integrate with kb context for agent consumption, (5) Add contradiction detection later.

**Authority:** architectural - Knowledge taxonomy decision, affects agent coordination patterns

---

# Investigation: Second Opinion on Case File Artifact Type

**Question:** Should case files be views (per chronicle precedent) or artifact types (new taxonomy addition)? Provide contrarian perspective to architect's views-based recommendation.

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** og-feat-second-opinion-case-31jan-7f1e
**Phase:** Complete
**Next Step:** Dylan decides between competing architectural perspectives
**Status:** Complete

**Challenges-Decision:** kn-160dc9 (chronicle precedent) - Argues precedent doesn't apply to case files

---

## Findings

### Finding 1: Chronicle Precedent Conflates Structure with Implementation

**Evidence:** Chronicle decision (kn-160dc9) states:
> "Chronicle should be a view over existing artifacts, not new artifact type. Reason: Minimal taxonomy principle; source data already exists; value is in narrative synthesis."

Architect applied this to case files: "Case files share the same DNA as chronicles - views over existing artifacts, not new artifacts themselves."

But this conflates TWO different characteristics:
1. **Structure:** Both synthesize existing data into narrative form (TRUE - they do share this)
2. **Implementation:** Both should be non-queryable HTML views (DOESN'T FOLLOW)

**Source:** 
- `.kb/investigations/2026-01-31-inv-review-case-file-artifact-type.md:3-9`
- `.kb/investigations/2026-01-31-inv-architect-synthesis-case-files.md:104-118`

**Significance:** The architect committed a category error. "Synthesized narrative" describes WHAT case files are (structure), not HOW they should be implemented (artifact vs view). Chronicles are historical snapshots; case files are operational coordination tools. Different purposes warrant different implementations even if structures are similar.

---

### Finding 2: Agent Consumption Creates Catch-22 for Views Approach

**Evidence:** The coaching plugin failure mode was agents couldn't see contradictions:
> "On Jan 28 alone, 5 investigations produced contradictory conclusions... No mechanism flagged these contradictions. They sat side-by-side in `.kb/investigations/`."

The proposed solution (case files as HTML) creates same problem:
- Agent investigates topic X
- Prior investigation on X exists with contradictions
- kb context can't surface HTML files (not in artifact taxonomy)
- Agent starts fresh, unaware of contradictions
- **Same failure mode repeats**

**Source:**
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:47-60`
- `.kb/investigations/2026-01-31-inv-architect-synthesis-case-files.md:125-136`

**Significance:** The entire point of case files is to prevent agents from repeating failed approaches. If agents can't discover case files exist via kb context, the problem isn't solved. Views-not-types perpetuates the exact problem case files were designed to fix.

---

### Finding 3: Lifecycle States Are Incompatible with Static Views

**Evidence:** Proposed case file states from design session:
- **Active:** Investigation ongoing, contradictions unresolved
- **Stalled:** Blocked, needs arbitration or human decision
- **Resolved:** Root cause identified, fix validated
- **Abandoned:** Gave up, documented why

Static HTML views can't represent state transitions. Questions:
- How does agent know if case is Active vs Resolved?
- Who updates the HTML when state changes?
- How to query "show active cases needing arbitration"?

Chronicles don't have this problem - they're point-in-time snapshots, not living documents.

**Source:**
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:203-205`
- `.kb/investigations/2026-01-31-inv-architect-synthesis-case-files.md:238-239`

**Significance:** Lifecycle management requires artifact infrastructure (metadata, queries, transitions). Views-not-types approach defers this to "manual curation" which won't scale if case files become common.

---

### Finding 4: Operational vs Historical Purpose Distinction

**Evidence:** Chronicles answer: "How did decision X evolve over time?" (historical, retrospective)

Case files answer: "What approaches failed for problem X?" (operational, prescriptive)

Usage patterns differ:
- **Chronicles:** Humans read to understand decision lineage (meta-learning)
- **Case files:** Agents read before investigating topic to avoid repeating failures (active coordination)

Frequency differs:
- **Chronicles:** Rarely created (major decisions only)
- **Case files:** Created when investigation churn detected (operational trigger)

**Source:**
- `.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md:7` (from kb context output)
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:110-119`

**Significance:** Operational artifacts need queryability, lifecycle, and agent integration. Historical artifacts can be static views. Different purposes warrant different implementations.

---

### Finding 5: Diagnosis-First Structure Doesn't Fit Investigation Template

**Evidence:** Investigation template:
```
## Findings
### Finding 1: [title]
**Evidence:** [what you observed]
**Source:** [where you found it]
**Significance:** [why it matters]

## Synthesis
[connect findings, answer question]
```

Case file structure (from spike learnings):
```
THE VERDICT: outcome, root cause, pattern (diagnosis first)
THE CONTRADICTION: side-by-side conflicting conclusions
THE GROUND TRUTH: human observations (screenshots)
THE TIMELINE: compressed history
THE FAILURE MODE: named pattern
WHAT SHOULD HAVE HAPPENED: intervention points
LESSONS: actionable takeaways
```

These are fundamentally incompatible. Forcing case files into investigation structure loses diagnosis-first value.

**Source:**
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:305-315`
- `~/.claude/skills/worker/feature-impl/reference/phase-investigation.md` (investigation template)

**Significance:** Structure incompatibility argues FOR artifact type (new template) not AGAINST it. The architect noted this but concluded "keep as views" rather than "template difference justifies new type."

---

### Finding 6: Minimal Taxonomy Argument Is Circular

**Evidence:** Architect argues: "Artifact taxonomy is intentionally minimal (only 2 types). Adding case files violates this principle."

But minimal taxonomy is a means, not an end. The principle exists to:
- Reduce maintenance overhead
- Prevent taxonomy bloat
- Keep system simple

Counter-evidence: Post-mortems exist as separate artifacts (`.kb/post-mortems/`) even though they're "narratives about what went wrong" (similar to case files). Why? Because operational need warranted it.

**Source:**
- `.kb/investigations/2026-01-31-inv-review-case-file-artifact-type.md:73-98`
- `.kb/post-mortems/` directory existence
- `.kb/investigations/2026-01-31-inv-architect-synthesis-case-files.md:53-60`

**Significance:** "Minimal taxonomy" prevents frivolous additions. But if case files have: (1) distinct structure, (2) operational need, (3) agent consumption requirement, (4) lifecycle needs - then they're not frivolous. The architect inverted the logic: treating "minimal" as sacred rather than asking "does this need justify expansion?"

---

### Finding 7: HTML Format Prevents Agent Integration

**Evidence:** Current spike: 991-line HTML file in `.kb/case-files/coaching-plugin-worker-detection.html`

HTML problems for agent consumption:
- kb context can't parse HTML structure (needs markdown frontmatter)
- Can't extract contradiction patterns programmatically
- No way to query "active cases on topic X"
- Git diffs are unreadable for 991-line HTML
- Agents would need to parse HTML to extract findings

Architect acknowledged this: "HTML is less git-diff friendly" but dismissed as acceptable trade-off.

**Source:**
- `.kb/case-files/coaching-plugin-worker-detection.html` (991 lines)
- `.kb/investigations/2026-01-31-inv-review-case-file-artifact-type.md:188`
- `pkg/kb/artifacts.go` (artifact scanning logic requires markdown)

**Significance:** HTML was right format for manual spike (visual presentation). Wrong format for operational artifact consumed by agents. Markdown with artifact structure enables both human readability AND agent consumption.

---

## Synthesis

**Challenging the Architect's Conclusion:**

The architect (Claude) applied the chronicle precedent (kn-160dc9) to conclude case files should be views, not types. This recommendation has five critical flaws:

### 1. Chronicle Precedent Doesn't Apply (Operational vs Historical)

**Why it doesn't apply:**
- Chronicles: Historical narratives, human-consumed, point-in-time
- Case Files: Operational coordination, agent-consumed, lifecycle-managed

The "synthesized narrative" similarity is superficial. Purpose and usage patterns are fundamentally different. Applying the precedent creates a false equivalence.

### 2. Views Create Catch-22 for Agent Discovery

**The problem:**
- Coaching plugin failed because agents couldn't see prior contradictions
- Case files (as views) remain invisible to kb context
- Agents investigating topic X won't see existing case file on X
- **Same problem persists**

The views approach solves presentation (humans can read HTML) but not the core problem (agents discovering contradictions).

### 3. Lifecycle Needs Require Artifact Infrastructure

**Required capabilities:**
- Query: "Show active cases needing arbitration"
- Update: Transition from Active → Resolved with evidence
- Filter: "Show cases on topic X"
- Integration: Surface in kb context for spawned agents

HTML views can't provide these without building custom tooling (which recreates artifact infrastructure anyway).

### 4. HTML Format Blocks Agent Consumption

**Technical incompatibility:**
- kb context scans markdown frontmatter (artifacts.go)
- HTML requires custom parsing
- Git diffs unreadable
- Can't extract structured data (contradictions, state, topic)

Markdown-based artifacts enable both human readability (rendered) AND agent consumption (structured).

### 5. Minimal Taxonomy Inverted from Means to End

**Logical error:**
- Architect treated "minimal" as inviolable principle
- Correct logic: Minimal until operational need justifies expansion
- Case files have: distinct structure, agent needs, lifecycle, operational purpose
- This justifies taxonomy expansion

Post-mortems exist despite being "narratives about failures" - operational need warranted it. Same logic applies to case files.

---

**Counter-Recommendation: Case Files as Artifact Type**

### Why Artifact Type is Correct Approach:

**1. Agent Integration (Primary Need)**
- kb context can surface case files when spawning agents on topic X
- "Prior investigations on X resulted in contradictions [link]"
- Agents inherit accumulated understanding, not just raw context

**2. Lifecycle Management**
- States: Active, Stalled, Resolved, Abandoned
- Transitions tracked in artifact history
- Query: "What cases need arbitration?" → actionable dashboard view

**3. Structured Data for Detection**
- Markdown frontmatter: topic, status, contradictions list
- Enables automated contradiction detection (future kb reflect)
- Supports metrics: "How many cases resolved vs abandoned?"

**4. Template Differentiation**
- Investigation template: Findings → Synthesis (question-driven)
- Case file template: Verdict → Contradiction → Diagnosis (diagnosis-driven)
- Structure difference warrants separate template, not forced fit

**5. Evolutionary Path**
- Phase 1: Add artifact type, markdown template, manual creation
- Phase 2: kb create case-file command
- Phase 3: Automated detection (kb reflect --type contradiction)
- Phase 4: Dashboard integration, arbitration triggers

HTML views lock us into Phase 0 (manual spike) with no evolution path.

---

## Structured Uncertainty

**What's tested:**

- ✅ Chronicle precedent exists (kn-160dc9 verified)
- ✅ Case files have different purpose than chronicles (verified: operational vs historical)
- ✅ HTML blocks agent consumption (verified: kb context scans markdown)
- ✅ Lifecycle states proposed in design session (verified: Active/Resolved/Abandoned)
- ✅ Post-mortems exist as separate artifact type (verified: `.kb/post-mortems/` directory)

**What's untested:**

- ⚠️ kb context integration effort for new artifact type (assumed feasible, not prototyped)
- ⚠️ Contradiction detection automation (proposed, not built)
- ⚠️ Whether agents would actually read case files before investigating (behavioral assumption)
- ⚠️ Markdown template can support diagnosis-first structure (assumed yes, not tested)

**What would change this:**

- If kb context integration is prohibitively complex → views approach more viable
- If agents ignore case files even when surfaced → value proposition weakens
- If case files rarely created (coaching plugin was one-off) → defer decision per Option C
- If markdown can't support diagnosis-first presentation → might need hybrid (markdown source + HTML render)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add CaseFile artifact type | architectural | Taxonomy expansion, affects agent patterns |
| Markdown template for case files | implementation | Technical format choice |
| kb create/update commands | implementation | Standard tooling |
| kb context integration | implementation | Existing integration pattern |

### Recommended Approach ⭐

**Artifact Type with Phased Implementation** - Add CaseFile to taxonomy, use markdown format, build tooling incrementally.

**Why this approach:**
- Solves agent discovery problem (kb context integration)
- Enables lifecycle management (states, transitions)
- Supports evolution (manual → automated detection)
- Markdown format: human-readable + agent-consumable
- Lower risk than views approach (which perpetuates problem)

**Trade-offs accepted:**
- Expands artifact taxonomy (justified by operational need)
- Requires kb tooling work (create/update/query commands)
- More complex than HTML views (but solves actual problem)

**Implementation sequence:**

**Phase 1: Minimal Viable Artifact (Week 1)**
1. Add `ArtifactCaseFile` to `pkg/kb/artifacts.go`
2. Create markdown template: `.kb/templates/case-file.md`
3. Manual creation: Copy template, fill in, commit
4. Test with kb context (does it surface case files?)

**Phase 2: Creation Tooling (Week 2)**
5. Add `kb create case-file <topic>` command
6. Populates template with topic, date, initial status
7. Opens in editor for completion

**Phase 3: Integration (Week 3)**
8. Update kb context to mention case files when topic matches
9. Add to knowledge placement table
10. Document when to create vs investigate

**Phase 4: Lifecycle (Future)**
11. Add `kb update case-file <id> --status resolved`
12. Dashboard view: Active cases needing attention
13. Metrics: Resolution rate, time-to-resolve

**Phase 5: Automation (Future)**
14. kb reflect --type contradiction (detects investigation contradictions)
15. Suggest case file creation when threshold met
16. Auto-populate timeline from investigation history

**Success criteria:**
- ✅ Agent spawned on topic X sees existing case file via kb context
- ✅ Case file lifecycle trackable (Active → Resolved)
- ✅ Markdown format: readable + parseable
- ✅ kb create case-file works without manual template copying

---

### Alternative Approaches Considered

**Option A: Views per Chronicle Precedent**
- **Pros:** Aligns with existing decision, lower implementation lift
- **Cons:** Doesn't solve agent discovery, no lifecycle, HTML blocks automation
- **When to use instead:** If agent consumption not critical, or if kb context can't easily add case file support

**Option B: Hybrid (Markdown source + HTML render)**
- **Pros:** Git-friendly source, beautiful presentation
- **Cons:** Dual-format complexity, build step required
- **When to use instead:** If presentation quality is critical (stakeholder communication)

**Option C: Defer per Synthesis Recommendation**
- **Pros:** Wait for second data point, avoid premature investment
- **Cons:** Next investigation churn will hit same problem
- **When to use instead:** If coaching plugin was truly one-off (unlikely in complex system)

**Rationale for recommendation:** Option "Artifact Type" (this recommendation) solves the core problem: agent discovery of contradictions. Views approach perpetuates the problem. The precedent doesn't apply because operational needs differ from historical narratives.

---

## Point-by-Point Rebuttal of Architect's Arguments

**Architect Argument 1:** "Prior decision kn-160dc9 applies: minimal taxonomy, synthesized narratives"

**Rebuttal:** Chronicle precedent conflates structure (synthesized narrative) with implementation (view-not-type). Case files have operational purpose (coordinate ongoing work) unlike chronicles (historical snapshot). Different purposes warrant different implementations. See Finding 1, Finding 4.

---

**Architect Argument 2:** "Current implementation works as HTML"

**Rebuttal:** Spike validated presentation, not agent consumption. HTML works for humans reading retrospectively. Doesn't work for agents discovering contradictions before investigating. Views approach perpetuates coaching plugin failure mode. See Finding 2, Finding 7.

---

**Architect Argument 3:** "Adding types has overhead (templates, lifecycle, scanning)"

**Rebuttal:** Overhead is justified when operational need exists. Post-mortems exist despite being "narratives about failures" - why? Operational need warranted it. Case files have same justification: agent coordination, contradiction detection, lifecycle management. See Finding 6.

---

**Architect Argument 4:** "Case files are output artifacts (generated), not source artifacts (created)"

**Rebuttal:** This is definitional, not logical. Post-mortems are "generated" from incident data. Investigations "generate" findings from exploration. All artifacts synthesize existing information. The distinction is: do agents need to read this? For case files: yes (to avoid repeating failures). See Finding 2.

---

**Architect Argument 5:** "Minimal taxonomy is intentional design"

**Rebuttal:** Minimal taxonomy prevents frivolous additions. Case files aren't frivolous if they: (1) solve agent coordination problem, (2) have distinct structure, (3) require lifecycle, (4) enable contradiction detection. "Minimal" is a guideline, not an absolute. The architect inverted means and ends. See Finding 6.

---

## Direct Challenge: Where Architect Went Wrong

**Category Error:** Conflated "synthesized narrative" (what it is) with "view-not-type" (how to implement). Chronicles and case files share structure similarity but have different operational needs.

**False Equivalence:** Treated historical artifacts (chronicles) as equivalent to operational artifacts (case files). Different purposes → different requirements.

**Catch-22 Ignored:** Acknowledged agent discovery problem ("No first-class status, not queryable") but dismissed as acceptable. This perpetuates the exact problem case files were designed to solve.

**Circular Logic:** "Taxonomy is minimal" → "Don't add types" → "Taxonomy stays minimal". Never asked: "Does operational need justify expansion?" Post-mortems prove expansion can be justified.

**Format Confusion:** Spike used HTML for presentation. Architect concluded HTML is the format, rather than recognizing spike was testing structure, not format. Markdown enables both presentation AND agent consumption.

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-31-inv-review-case-file-artifact-type.md` - Architect's views recommendation
- `.kb/investigations/2026-01-31-inv-architect-synthesis-case-files.md` - Three-option synthesis
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md` - Original design session
- `.kb/case-files/coaching-plugin-worker-detection.html` - Spike implementation
- `pkg/kb/artifacts.go` - Artifact type definitions
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Taxonomy decision
- Decision kn-160dc9 - Chronicle precedent

**Commands Run:**
```bash
# Verify artifact types
grep "ArtifactType" pkg/kb/artifacts.go

# Check post-mortems exist as artifact type
ls .kb/post-mortems/

# Get chronicle decision context
kb context "chronicle" --format json

# Verify kb context scans markdown
grep -n "\.md" pkg/kb/artifacts.go
```

**Related Artifacts:**
- **Decision:** kn-160dc9 (chronicle as view) - This investigation challenges its applicability
- **Investigation:** `.kb/investigations/2026-01-31-inv-review-case-file-artifact-type.md` - Competing perspective
- **Investigation:** `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md` - Original case file design
- **Constraint:** kb-b59d62 (added after synthesis) - May need revision if this perspective accepted

---

## Investigation History

**2026-01-31 ~11:00:** Investigation started
- Context: Spawned for second opinion on architect's views recommendation
- Task: Challenge views-not-types conclusion with contrarian perspective

**2026-01-31 ~11:15:** Read all three prior investigations
- Review case file artifact type (views recommendation)
- Design case files and arbitration (original design)
- Architect synthesis (three options)

**2026-01-31 ~11:30:** Identified core disagreement
- Chronicle precedent application is category error
- Operational vs historical purpose distinction is key
- Agent consumption requirement not adequately addressed

**2026-01-31 ~11:45:** Investigation completed
- Status: Complete
- Key outcome: Recommend artifact type, not views
- Rationale: Agent discovery, lifecycle needs, operational purpose justify taxonomy expansion
- Direct challenge: Architect conflated structure with implementation, applied wrong precedent

