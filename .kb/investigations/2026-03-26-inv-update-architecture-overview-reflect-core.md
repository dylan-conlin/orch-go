## Summary (D.E.K.N.)

**Delta:** Rewrote architecture-overview.md to lead with orch-go as a coordination/comprehension layer, organizing the directory structure and system diagram around core/substrate/adjacent boundaries.

**Evidence:** Aligned with accepted decision (2026-03-26-thread-comprehension-layer-is-primary-product.md) and consolidation plan Phase 1 deliverable. All existing operational content preserved, repositioned under "Execution Substrate Detail."

**Knowledge:** The guide was entirely execution-focused (spawn backends, flow, directory flat-list). Reframing required adding a product boundary table, a layered system diagram, and reorganizing the directory listing by layer — not deleting content.

**Next:** Close. README rewrite is the remaining Phase 1 deliverable (separate issue).

**Authority:** implementation — Applies accepted strategic decision to a documentation artifact within scope.

---

# Investigation: Update Architecture Overview Reflect Core

**Question:** How should the architecture overview be restructured to reflect the core/substrate/adjacent product boundary?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-bw0y6
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation (implementing accepted decision) | - | - | - |

---

## Findings

### Finding 1: Current guide is entirely execution-focused

**Evidence:** The existing architecture-overview.md has four sections: System Diagram (spawn-centric), Directory Structure (flat listing), Spawn Backends (detailed execution plumbing), and Spawn Flow. No mention of threads, comprehension, knowledge composition, or what the product actually is.

**Source:** `.kb/guides/architecture-overview.md` (pre-revision)

**Significance:** Confirms the decision document's observation that "README and architecture docs still describe orch-go primarily as an orchestration CLI." The guide was functional for execution orientation but actively reproduced the older identity.

### Finding 2: Directory structure maps cleanly onto product boundary

**Evidence:** Packages like `pkg/thread/`, `pkg/claims/`, `pkg/completion/`, `pkg/verify/`, `pkg/digest/`, `pkg/kbmetrics/` are clearly core. Packages like `pkg/spawn/`, `pkg/tmux/`, `pkg/opencode/`, `pkg/daemon/` are clearly substrate. The mapping was unambiguous for most packages.

**Source:** `pkg/` directory listing in architecture-overview.md

**Significance:** The boundary classification from the decision document is not forced — it corresponds to actual package structure. This made reorganizing the directory listing natural rather than artificial.

### Finding 3: Execution content should be preserved, not removed

**Evidence:** The spawn backend details, architectural principles (backend independence, pain as signal), and spawn flow are operationally important. The consolidation plan explicitly says "execution plumbing remains usable" and "this does not mean execution plumbing can be deleted immediately."

**Source:** `.kb/plans/2026-03-26-thread-comprehension-consolidation.md`, `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md:144-150`

**Significance:** The revision preserves all execution detail under a clearly labeled "Execution Substrate Detail" section, with a framing sentence that positions it as important-but-not-central.

---

## Synthesis

**Key Insights:**

1. **Reframing is structural, not subtractive** — The guide needed new sections (product boundary, layered diagram, organized directory) more than it needed deletions. All operational content survived.

2. **Package structure already reflects the boundary** — The codebase naturally splits into comprehension-oriented and execution-oriented packages. The guide was just describing them in a flat list that obscured this.

3. **One connecting sentence bridges layers** — In the Backend Independence section, added a sentence connecting the substrate principle to the product boundary: "the core layer should not depend on any single execution backend." This ties the operational wisdom to the strategic frame without being heavy-handed.

**Answer to Investigation Question:**

The architecture overview was restructured into: (1) a "What orch-go Is" opening that names the product center, (2) a product boundary table mapping concerns to core/substrate/adjacent with key packages, (3) a layered system diagram, (4) a directory listing organized by layer, and (5) all execution detail preserved under "Execution Substrate Detail." This satisfies the consolidation plan's Phase 1 deliverable for the architecture guide.

---

## Structured Uncertainty

**What's tested:**

- ✅ All packages from original listing are present in revised listing (manually verified)
- ✅ Product boundary categories align with decision document (cross-referenced)
- ✅ Execution substrate content preserved verbatim where appropriate

**What's untested:**

- ⚠️ Whether a new reader finds the layered guide more orienting than the flat one (no user testing)
- ⚠️ Whether the core/substrate package classification is useful for actual prioritization decisions

**What would change this:**

- Evidence that the classification creates confusion for engineers debugging execution issues (they need to find spawn details quickly)
- Feedback that the product boundary tables are too abstract for engineering orientation

---

## References

**Files Examined:**
- `.kb/guides/architecture-overview.md` — Original and revised architecture guide
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Accepted decision establishing the boundary
- `.kb/plans/2026-03-26-thread-comprehension-consolidation.md` — Consolidation plan (Phase 1 deliverable)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Source of the product boundary
- **Plan:** `.kb/plans/2026-03-26-thread-comprehension-consolidation.md` — This work is Phase 1 item 2

---

## Investigation History

**2026-03-26:** Investigation started
- Initial question: How to restructure architecture overview for core/substrate/adjacent boundary
- Context: Phase 1 deliverable from thread/comprehension consolidation plan

**2026-03-26:** Investigation completed
- Status: Complete
- Key outcome: Architecture overview rewritten with layered structure, product boundary tables, and execution detail preserved as substrate
