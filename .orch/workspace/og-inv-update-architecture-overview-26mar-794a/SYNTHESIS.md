# Session Synthesis

**Agent:** og-inv-update-architecture-overview-26mar-794a
**Issue:** orch-go-bw0y6
**Outcome:** success

---

## Plain-Language Summary

The architecture guide described orch-go purely through execution mechanics — spawn backends, daemon plumbing, directory flat-lists. It didn't mention threads, comprehension, or knowledge composition at all. After the product-boundary decision, a new reader would get a misleading picture of what the project actually is. The rewrite opens with orch-go as a coordination/comprehension layer, adds a product boundary map (core/substrate/adjacent with key packages for each), replaces the flat system diagram with a layered one, and organizes the directory listing by layer — while preserving all execution detail under a clearly labeled substrate section.

## TLDR

Rewrote `.kb/guides/architecture-overview.md` to lead with the product identity (coordination + comprehension layer), add core/substrate/adjacent boundary tables, and reorganize the directory structure by layer — preserving all execution substrate detail.

---

## Delta (What Changed)

### Files Modified
- `.kb/guides/architecture-overview.md` - Complete rewrite: added product framing, boundary tables, layered diagram, organized directory listing, repositioned execution detail as substrate

### Files Created
- `.kb/investigations/2026-03-26-inv-update-architecture-overview-reflect-core.md` - Investigation documenting the rewrite rationale and findings

---

## Evidence (What Was Observed)

- Original guide had 4 sections, all execution-focused: System Diagram, Directory Structure, Spawn Backends, Spawn Flow
- No mention of threads, comprehension, knowledge, claims, models, or what the product is for
- Package structure naturally maps to the boundary: `pkg/thread/`, `pkg/claims/`, `pkg/completion/` are clearly core; `pkg/spawn/`, `pkg/tmux/`, `pkg/opencode/` are clearly substrate
- Decision document explicitly calls out architecture overview update as Phase 1 deliverable

---

## Architectural Choices

### Reorganized directory listing by layer instead of keeping flat list
- **What I chose:** Annotate directory listing with `── Core ──` and `── Substrate ──` section markers
- **What I rejected:** Keeping the flat alphabetical listing with a separate mapping table
- **Why:** Engineers looking for a package benefit from seeing it in context of its layer; a separate table duplicates without adding navigation value
- **Risk accepted:** Some packages could be reclassified as the boundary matures

### Preserved all execution detail verbatim
- **What I chose:** Keep spawn backends, architectural principles, and spawn flow as-is under "Execution Substrate Detail"
- **What I rejected:** Trimming or moving execution content to a separate guide
- **Why:** Decision doc says "execution plumbing remains usable" — splitting would break the single-guide orientation property
- **Risk accepted:** Guide is longer than before (~200 lines vs ~197 lines)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-update-architecture-overview-reflect-core.md` - Rewrite rationale

### Decisions Made
- Decision: Organize by layer within the same file rather than splitting into separate guides, because engineering orientation works best with one authoritative file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] Architecture overview rewritten with core/substrate/adjacent boundary
- [x] Investigation file complete with D.E.K.N.
- [x] VERIFICATION_SPEC.yaml authored
- [x] Ready for `orch complete orch-go-bw0y6`

---

## Unexplored Questions

- Whether the README should mirror this exact structure or take a more narrative approach (separate Phase 1 deliverable)
- Whether `pkg/attention/` and `pkg/orient/` belong in core (they route toward comprehension) or substrate (they are daemon mechanics)

---

## Friction

No friction — smooth session

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 7 checks, all passing. Key outcomes: guide opens with product identity, boundary tables present, execution detail preserved, investigation complete.

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-update-architecture-overview-26mar-794a/`
**Investigation:** `.kb/investigations/2026-03-26-inv-update-architecture-overview-reflect-core.md`
**Beads:** `bd show orch-go-bw0y6`
