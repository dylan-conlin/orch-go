## Summary (D.E.K.N.)

**Delta:** Case files should remain as generated views (HTML reports), not become a formal artifact type. The prior decision for chronicles (kn-160dc9) applies: "minimal taxonomy principle; source data already exists; value is in narrative synthesis."

**Evidence:** Prior decision kn-160dc9 on chronicles establishes the pattern. Case files already work as HTML in `.kb/case-files/`. The current artifact taxonomy (investigation, decision) is intentionally minimal. Case files are synthesized from investigations, not new source data.

**Knowledge:** Case files are diagnosis-first narratives that answer "what went wrong?" across multiple investigations. They share the same DNA as chronicles - views over existing artifacts, not new artifacts themselves. The value is in the synthesis, not in storing another artifact type.

**Next:** Keep case files as generated HTML reports. Add `kb case-file generate <topic>` command if automation is needed. Close this review.

**Authority:** architectural - Cross-boundary decision affecting knowledge taxonomy, but aligned with existing decision

---

# Investigation: Review Case File Artifact Type Proposal

**Question:** Should case files become a formal artifact type in the kb system (like investigations and decisions), or should they remain as generated views/reports?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** og-arch-review-case-file-31jan-97c8
**Phase:** Complete
**Next Step:** None - recommendation clear
**Status:** Complete

---

## Findings

### Finding 1: Prior Decision Establishes "View Over Type" Pattern

**Evidence:** Decision kn-160dc9 from 2025-12-21:
> "Chronicle should be a view over existing artifacts, not new artifact type"
> Reason: "Minimal taxonomy principle; source data already exists in git/kn/kb; value is in narrative synthesis not data capture"

The chronicle investigation (`.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md`) analyzed this extensively:
- Chronicles capture decision evolution narratives spanning multiple artifacts
- All source data already exists (git, kn, investigations, decisions)
- The value is in the narrative synthesis, not new data capture
- Orchestrator creates narratives; tooling assists by gathering sources

**Source:** `kb context "chronicle"` output, `.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md`

**Significance:** Case files are conceptually identical to chronicles - both are synthesized narratives over existing artifacts. The prior decision applies directly.

---

### Finding 2: Case Files Are Already Working As Generated Views

**Evidence:** Current implementation in `.kb/case-files/`:
- `README.md` - Explains purpose and when to create case files
- `coaching-plugin-worker-detection.html` - 44KB standalone HTML file

The README states:
> "Case files are diagnosis-first narratives that explain complex, multi-agent investigation failures."
> "This is a manual spike to test whether case files are useful. If successful, could build automated case file generation."

The HTML file is a rendered view containing:
- Statistics (19 investigations, 34 commits, 3 weeks)
- Verdict section (diagnosis first)
- Contradiction highlights
- Ground truth (Dylan's observations)
- Timeline (compressed, grouped)
- Failure mode analysis
- Lessons learned

**Source:** `ls -la .kb/case-files/`, `.kb/case-files/README.md`

**Significance:** The spike validated that case files work. They're output artifacts (generated reports), not source artifacts (things agents create). Adding them to the artifact taxonomy would conflate these two concepts.

---

### Finding 3: Artifact Taxonomy Is Intentionally Minimal

**Evidence:** `pkg/kb/artifacts.go` defines only two artifact types:
```go
const (
    ArtifactInvestigation ArtifactType = "investigation"
    ArtifactDecision      ArtifactType = "decision"
)
```

The kb system has been intentionally kept to essential types:
- **Investigations:** Point-in-time findings with evidence
- **Decisions:** Architectural choices with rationale

Related: Guides and models exist but are not managed as first-class artifact types in code.

**Source:** `pkg/kb/artifacts.go:16-19`

**Significance:** Adding a third artifact type (case files) would require:
- Updating artifact scanning logic
- Adding case file templates
- Building case file parsing
- Managing case file lifecycle

This is significant overhead for what is essentially a view/report.

---

### Finding 4: Case Files Are Synthesized From Investigations, Not Source Data

**Evidence:** The case file design investigation (`.kb/investigations/2026-01-31-design-case-files-and-arbitration.md`) shows that case files aggregate:
- Investigation conclusions (primary source)
- Contradiction patterns (derived from investigations)
- Human observations (from chat, pasted into case file)
- Beads issue history (from beads, linked in case file)

The case file itself doesn't introduce new source data - it synthesizes and visualizes existing data. Compare to:
- **Investigation:** Creates new source data (findings, evidence, conclusions)
- **Decision:** Creates new source data (choice, rationale, constraints)
- **Case file:** Synthesizes existing data into diagnostic view

**Source:** `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md`

**Significance:** Source artifacts vs output artifacts is a useful distinction. Case files are output artifacts - they should be generated, not edited. This is why HTML works well (static output) rather than markdown (editable source).

---

### Finding 5: Case File Structure Is Diagnosis-First, Not Investigation-Compatible

**Evidence:** The revised case file structure (from orch-go-21125) has 8 sections:
1. THE VERDICT (outcome, root cause)
2. THE CONTRADICTION (conflicting conclusions)
3. THE GROUND TRUTH (human observations)
4. THE TIMELINE (compressed history)
5. THE FAILURE MODE (named pattern)
6. THE EVIDENCE TRAIL (links)
7. WHAT SHOULD HAVE HAPPENED (intervention points)
8. LESSONS FOR NEXT TIME (takeaways)

This structure is fundamentally different from investigations:
- Investigations: Question → Findings → Synthesis
- Case files: Verdict → Contradiction → Diagnosis

Trying to force case files into investigation templates would lose the diagnosis-first value.

**Source:** `bd show orch-go-21125`, `.kb/case-files/README.md`

**Significance:** Case files have different goals than investigations. They answer "what went wrong over many attempts?" rather than "what did we discover?" This structural difference supports keeping them as a separate view type, not forcing them into the artifact taxonomy.

---

## Synthesis

**Key Insights:**

1. **Prior precedent applies** - The chronicle decision (kn-160dc9) established "view over type" as the pattern for synthesized narratives. Case files fit this pattern exactly.

2. **Output vs source distinction** - Case files are generated outputs from existing data, not new source data. The artifact taxonomy is for source artifacts (things created, then referenced). Case files are reports (things generated from sources).

3. **Structure incompatibility** - Case files have a diagnosis-first structure that doesn't fit investigation templates. Forcing compatibility would lose the value.

4. **Minimal taxonomy is intentional** - Adding types has overhead. The spike proved case files work as HTML. No compelling reason to formalize.

**Answer to Investigation Question:**

Case files should **NOT** become a formal artifact type. They should remain as generated views/reports (currently HTML in `.kb/case-files/`).

This recommendation aligns with:
- Prior decision kn-160dc9 (chronicle = view)
- Minimal taxonomy principle
- Output vs source artifact distinction
- Current working implementation

If automation is desired, the path is `kb case-file generate <topic>` command that:
1. Gathers investigations matching topic
2. Detects contradictions
3. Renders diagnosis-first HTML
4. Saves to `.kb/case-files/`

Not: adding CaseFile to ArtifactType enum.

---

## Structured Uncertainty

**What's tested:**

- ✅ Prior chronicle decision exists and applies (verified: `kb context "chronicle"`)
- ✅ Case files work as HTML (verified: `.kb/case-files/coaching-plugin-worker-detection.html` exists)
- ✅ Artifact taxonomy is minimal (verified: `pkg/kb/artifacts.go` has only 2 types)
- ✅ Case files are synthesized from investigations (verified: design investigation analysis)

**What's untested:**

- ⚠️ Automated case file generation hasn't been built (current HTML is manual)
- ⚠️ "Same topic" detection for grouping investigations (proposed, not implemented)
- ⚠️ Whether HTML is the right format long-term (works for now, may want markdown)

**What would change this:**

- If case files need to be editable/versioned (then markdown source + HTML render)
- If case files need cross-referencing from other artifacts (then need some form of ID)
- If case files become so common that automation is insufficient (>10 case files)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Keep case files as views, not types | architectural | Aligns with prior decision, affects knowledge taxonomy |
| Add `kb case-file generate` command | implementation | Standard feature implementation |
| Keep HTML format for now | implementation | Working, can change later |

### Recommended Approach ⭐

**Views with Optional Automation** - Keep case files as generated HTML reports. Optionally add `kb case-file generate <topic>` to automate gathering and rendering.

**Why this approach:**
- Aligns with prior decision (chronicle = view)
- No artifact taxonomy changes needed
- Current implementation works
- Lower maintenance overhead

**Trade-offs accepted:**
- No first-class artifact status (can't query case files like investigations)
- Manual curation unless automation is built
- HTML format is less git-diff friendly

**Implementation sequence (if automation desired):**
1. Add `kb case-file generate <topic>` command
2. Implement topic matching (filename patterns, explicit tagging)
3. Implement contradiction detection (compare conclusions)
4. Render to diagnosis-first HTML template
5. Save to `.kb/case-files/`

### Alternative Approaches Considered

**Option B: Add CaseFile artifact type**
- **Pros:** First-class status, queryable, standard lifecycle
- **Cons:** Violates minimal taxonomy, conflates source/output, template incompatibility
- **When to use instead:** If case files become primary source artifacts (unlikely)

**Option C: Case files as special investigation subtype**
- **Pros:** Reuses investigation infrastructure
- **Cons:** Structure doesn't fit, loses diagnosis-first value
- **When to use instead:** If investigation template can accommodate diagnosis structure

**Option D: Case files as markdown with HTML generation**
- **Pros:** Git-friendly source, rendered output
- **Cons:** Dual format overhead, complexity
- **When to use instead:** If case files need to be collaboratively edited

**Rationale for recommendation:** Option A (views with automation) matches prior decision, requires no taxonomy changes, and leverages working implementation. The spike validated the concept; now it's about optional tooling, not architecture.

---

### Implementation Details

**What to implement first (if any):**
- Nothing required - current implementation works
- If automation desired: start with topic matching logic

**Things to watch out for:**
- ⚠️ Don't add CaseFile to ArtifactType enum without revisiting chronicle decision
- ⚠️ HTML size can grow large (44KB for coaching plugin case)
- ⚠️ Manual curation is feature, not bug (diagnosis requires judgment)

**Areas needing further investigation:**
- "Same topic" detection algorithm (if automation pursued)
- Contradiction detection heuristics
- Dashboard integration (showing case files in strategic center)

**Success criteria:**
- ✅ Case files remain working as HTML views
- ✅ No changes needed to artifact taxonomy
- ✅ If automation built, generates useful case files from existing investigations

---

## References

**Files Examined:**
- `pkg/kb/artifacts.go` - Artifact type definitions
- `cmd/orch/kb.go` - KB command implementations
- `.kb/case-files/README.md` - Case file documentation
- `.kb/case-files/coaching-plugin-worker-detection.html` - Working case file example
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md` - Design investigation
- `.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md` - Chronicle precedent

**Commands Run:**
```bash
# Check artifact types
grep -n "ArtifactType" pkg/kb/artifacts.go

# Check case files directory
ls -la .kb/case-files/

# Get chronicle decision context
kb context "chronicle" --format json

# Check related issues
bd show orch-go-21124
bd show orch-go-21125
```

**Related Artifacts:**
- **Decision:** kn-160dc9 - "Chronicle should be a view over existing artifacts, not new artifact type"
- **Investigation:** `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md` - Case file design
- **Investigation:** `.kb/investigations/2025-12-21-inv-chronicle-artifact-type-design.md` - Chronicle precedent

---

## Investigation History

**2026-01-31 ~10:45:** Investigation started
- Initial question: Should case files become formal artifact type?
- Context: Architect review spawned from orch-go-21126

**2026-01-31 ~10:50:** Found prior chronicle decision
- Discovery: kn-160dc9 establishes "view over type" pattern
- Significance: Directly applicable precedent

**2026-01-31 ~11:00:** Analyzed current implementation
- Case files work as HTML in `.kb/case-files/`
- Artifact taxonomy is intentionally minimal
- Case files are output artifacts, not source artifacts

**2026-01-31 ~11:15:** Investigation completed
- Status: Complete
- Key outcome: Keep case files as views, not types. Prior decision applies.
